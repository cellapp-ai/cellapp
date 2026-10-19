package hosting

import (
	"context"
	"encoding/base64"
	"io"
	"net/url"
	"sync"
	"testing"
)

func TestMCPConcurrentPublishKeepsOneVersion(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	app, _ := f.create(t)
	deployments := []string{}
	for _, content := range []string{"first", "other"} {
		deploymentID, err := f.server.prepareOwnerDeployment(ctx, f.owner, app.ID, createDeploymentInput{Manifest: []Entry{{Path: "index.html", Size: int64(len(content)), SHA256: digest([]byte(content))}}, RequestID: id()})
		if err != nil {
			t.Fatal(err)
		}
		chunk := mcpChunkInput{AppID: app.ID, DeploymentID: deploymentID, FileIndex: 0, SizeBytes: len(content), SHA256: digest([]byte(content)), DataBase64: base64.StdEncoding.EncodeToString([]byte(content))}
		if _, err := f.server.uploadMCPChunk(ctx, f.owner, chunk); err != nil {
			t.Fatal(err)
		}
		if err := f.server.completeMCPFile(ctx, f.owner, app.ID, deploymentID, 0); err != nil {
			t.Fatal(err)
		}
		deployments = append(deployments, deploymentID)
	}
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, deploymentID := range deployments {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, err := f.server.publishOwnerDeployment(ctx, f.owner, app.ID, deploymentID)
			results <- err
		}()
	}
	workers.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if api, ok := err.(*APIError); ok && api.Code == "version_conflict" {
			conflict++
		} else {
			t.Fatalf("unexpected publish result: %v", err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("published=%d conflicted=%d", success, conflict)
	}
	var active string
	if err := f.db.QueryRow(ctx, `SELECT active_deployment FROM apps WHERE id=$1`, app.ID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != deployments[0] && active != deployments[1] {
		t.Fatalf("unexpected active deployment: %s", active)
	}
}

func TestMCPChunkUploadResumesAcrossInstances(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	app, _ := f.create(t)
	content := []byte("first-second")
	deploymentID, err := f.server.prepareOwnerDeployment(ctx, f.owner, app.ID, createDeploymentInput{Manifest: []Entry{{Path: "index.html", Size: int64(len(content)), SHA256: digest(content)}}, RequestID: id()})
	if err != nil {
		t.Fatal(err)
	}
	secondInstance := *f.server
	first := mcpChunkInput{AppID: app.ID, DeploymentID: deploymentID, FileIndex: 0, SizeBytes: 6, SHA256: digest(content[:6]), DataBase64: base64.StdEncoding.EncodeToString(content[:6])}
	if next, err := f.server.uploadMCPChunk(ctx, f.owner, first); err != nil || next != 6 {
		t.Fatalf("first instance upload: %d %v", next, err)
	}
	if progress, err := secondInstance.mcpChunkProgress(ctx, deploymentID); err != nil || progress[0] != 6 {
		t.Fatalf("second instance progress: %#v %v", progress, err)
	}
	second := mcpChunkInput{AppID: app.ID, DeploymentID: deploymentID, FileIndex: 0, OffsetBytes: 6, SizeBytes: len(content) - 6, SHA256: digest(content[6:]), DataBase64: base64.StdEncoding.EncodeToString(content[6:])}
	if _, err := secondInstance.uploadMCPChunk(ctx, id(), second); err == nil {
		t.Fatal("another owner resumed upload")
	}
	if next, err := secondInstance.uploadMCPChunk(ctx, f.owner, second); err != nil || next != int64(len(content)) {
		t.Fatalf("second instance upload: %d %v", next, err)
	}
	if err := secondInstance.completeMCPFile(ctx, f.owner, app.ID, deploymentID, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := f.server.publishOwnerDeployment(ctx, f.owner, app.ID, deploymentID); err != nil {
		t.Fatal(err)
	}
}

func TestMCPConcurrentChunkConflictKeepsStoredDigest(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	app, _ := f.create(t)
	deploymentID, err := f.server.prepareOwnerDeployment(ctx, f.owner, app.ID, createDeploymentInput{Manifest: []Entry{{Path: "index.html", Size: 5, SHA256: digest([]byte("hello"))}}, RequestID: id()})
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, value := range []string{"hello", "world"} {
		workers.Add(1)
		go func() {
			defer workers.Done()
			_, err := f.server.uploadMCPChunk(ctx, f.owner, mcpChunkInput{AppID: app.ID, DeploymentID: deploymentID, FileIndex: 0, SizeBytes: 5, SHA256: digest([]byte(value)), DataBase64: base64.StdEncoding.EncodeToString([]byte(value))})
			results <- err
		}()
	}
	workers.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if api, ok := err.(*APIError); ok && api.Code == "chunk_conflict" {
			conflict++
		} else {
			t.Fatalf("unexpected chunk result: %v", err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("uploaded=%d conflicted=%d", success, conflict)
	}
	var recorded string
	if err := f.db.QueryRow(ctx, `SELECT sha256 FROM mcp_upload_chunks WHERE deployment_id=$1 AND file_index=0 AND offset_bytes=0`, deploymentID).Scan(&recorded); err != nil {
		t.Fatal(err)
	}
	stored, err := f.storage.Get(ctx, mcpChunkKey(deploymentID, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	bytes, err := io.ReadAll(stored)
	stored.Close()
	if err != nil || digest(bytes) != recorded {
		t.Fatalf("chunk object differs from committed metadata: %q %q %v", bytes, recorded, err)
	}
}

func TestMCPInvalidChunksCannotReplacePublishedRelease(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	app, accessKey := f.create(t)
	previous, _ := f.stage(t, app, "old")
	f.uploadAndPublish(t, app, previous, "old")
	app.Active = &previous
	visitor := f.unlock(t, app, accessKey)
	deploymentID, err := f.server.prepareOwnerDeployment(ctx, f.owner, app.ID, createDeploymentInput{Manifest: []Entry{{Path: "index.html", Size: 6, SHA256: digest([]byte("abcdef"))}}, RequestID: id(), Base: &previous})
	if err != nil {
		t.Fatal(err)
	}
	chunk := mcpChunkInput{AppID: app.ID, DeploymentID: deploymentID, FileIndex: 0, OffsetBytes: 0, SizeBytes: 3, SHA256: digest([]byte("abc")), DataBase64: base64.StdEncoding.EncodeToString([]byte("abc"))}
	tooLarge := chunk
	tooLarge.SizeBytes = mcpChunkBytes + 1
	if _, err := f.server.uploadMCPChunk(ctx, f.owner, tooLarge); err == nil {
		t.Fatal("oversized chunk accepted")
	}
	badOffset := chunk
	badOffset.OffsetBytes = 1
	if _, err := f.server.uploadMCPChunk(ctx, f.owner, badOffset); err == nil {
		t.Fatal("noncontiguous chunk accepted")
	}
	badDigest := chunk
	badDigest.SHA256 = digest([]byte("xyz"))
	if _, err := f.server.uploadMCPChunk(ctx, f.owner, badDigest); err == nil {
		t.Fatal("invalid chunk digest accepted")
	}
	if next, err := f.server.uploadMCPChunk(ctx, f.owner, chunk); err != nil || next != 3 {
		t.Fatalf("first chunk: %d %v", next, err)
	}
	if err := f.server.completeMCPFile(ctx, f.owner, app.ID, deploymentID, 0); err == nil {
		t.Fatal("incomplete file accepted")
	}
	chunk.OffsetBytes = 3
	chunk.SHA256 = digest([]byte("xyz"))
	chunk.DataBase64 = base64.StdEncoding.EncodeToString([]byte("xyz"))
	if _, err := f.server.uploadMCPChunk(ctx, f.owner, chunk); err != nil {
		t.Fatal(err)
	}
	if err := f.server.completeMCPFile(ctx, f.owner, app.ID, deploymentID, 0); err == nil {
		t.Fatal("file digest mismatch accepted")
	}
	if _, err := f.server.publishOwnerDeployment(ctx, f.owner, app.ID, deploymentID); err == nil {
		t.Fatal("invalid file published")
	}
	host, _ := url.Parse(app.URL)
	if page := f.request(t, "GET", "/", nil, "", host.Host, visitor); page.Code != 200 || page.Body.String() != "old" {
		t.Fatalf("old release unavailable: %d %q", page.Code, page.Body.String())
	}
}
