package hosting

import (
	"context"
	"strings"
	"testing"
	"time"
)

type timedMCPStorage struct {
	*memoryStorage
	modified map[string]time.Time
}

func (s *timedMCPStorage) WalkTemporary(ctx context.Context, visit func(string, time.Time) error) error {
	s.Lock()
	items := map[string]time.Time{}
	for key := range s.data {
		if strings.HasPrefix(key, "mcp-chunks/") {
			items[key] = s.modified[key]
		}
	}
	s.Unlock()
	for key, modified := range items {
		if err := visit(key, modified); err != nil {
			return err
		}
	}
	return nil
}

func TestMCPCleanupRemovesOnlyOrphanedChunks(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	app, _ := f.create(t)
	deploymentID, _ := f.stage(t, app, "hello")
	storage := &timedMCPStorage{memoryStorage: f.storage, modified: map[string]time.Time{}}
	f.server.Storage = storage
	known := mcpChunkKey(deploymentID, 0, 0)
	orphan := "mcp-chunks/" + id() + "/0/0"
	recent := "mcp-chunks/" + id() + "/0/0"
	old := time.Now().Add(-2 * f.server.Config.UploadTTL)
	for key, modified := range map[string]time.Time{known: old, orphan: old, recent: time.Now()} {
		f.storage.data[key] = []byte("h")
		storage.modified[key] = modified
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO mcp_upload_chunks(deployment_id,file_index,offset_bytes,size_bytes,sha256,object_key) VALUES($1,0,0,1,$2,$3)`, deploymentID, digest([]byte("h")), known); err != nil {
		t.Fatal(err)
	}
	f.storage.failDelete = true
	if err := f.server.Cleanup(ctx); err == nil {
		t.Fatal("temporary storage failure was hidden")
	}
	if _, ok := f.storage.data[orphan]; !ok {
		t.Fatal("failed orphan delete was reported as complete")
	}
	f.storage.failDelete = false
	if err := f.server.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.storage.data[orphan]; ok {
		t.Fatal("expired orphan retained")
	}
	if _, ok := f.storage.data[known]; !ok {
		t.Fatal("referenced upload chunk removed")
	}
	if _, ok := f.storage.data[recent]; !ok {
		t.Fatal("young orphan removed")
	}
}
