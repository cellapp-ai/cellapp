package hosting

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpTestTransport struct {
	base  http.RoundTripper
	token string
}

func (t mcpTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Host = "control.localhost:8443"
	request.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(request)
}

func TestMCPAuthenticatedTools(t *testing.T) {
	f := setup(t)
	for _, auth := range []string{"", f.token} {
		unauthorized := f.request(t, "POST", "/mcp", []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`), auth, "control.localhost:8443", "")
		if unauthorized.Code != 401 || unauthorized.Header().Get("WWW-Authenticate") == "" {
			t.Fatalf("MCP auth challenge: %d %#v", unauthorized.Code, unauthorized.Header())
		}
	}
	if appDomain := f.request(t, "GET", "/mcp", nil, "", strings.Repeat("a", 32)+".apps.localhost:8443", ""); appDomain.Code != 404 {
		t.Fatalf("MCP reached application domain: %d", appDomain.Code)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientID := id()
	access := token()
	if _, err := f.db.Exec(ctx, `INSERT INTO mcp_oauth_clients(id,name,redirect_uris) VALUES($1,'test','[]')`, clientID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO mcp_oauth_consents(owner_id,client_id,scopes) VALUES($1,$2,ARRAY['mcp:read','mcp:deploy'])`, f.owner, clientID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO mcp_oauth_tokens(hash,owner_id,client_id,resource,scopes,expires_at) VALUES($1,$2,$3,$4,ARRAY['mcp:read','mcp:deploy'],now()+interval '1 hour')`, keyed(f.server.Config.Secret, access), f.owner, clientID, f.server.Config.ControlOrigin+"/mcp"); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(f.handler)
	defer httpServer.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "cellapp-integration", Version: "1.0.0"}, nil)
	connect := func(value string) *mcp.ClientSession {
		t.Helper()
		session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL + "/mcp", DisableStandaloneSSE: true, HTTPClient: &http.Client{Transport: mcpTestTransport{http.DefaultTransport, value}}}, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = session.Close() })
		return session
	}
	session := connect(access)
	listed, err := session.ListTools(ctx, nil)
	if err != nil || len(listed.Tools) < 2 {
		t.Fatalf("tools: %#v %v", listed, err)
	}
	created, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "create_app", Arguments: map[string]any{"name": "Agent app", "requestId": id()}})
	if err != nil || created.IsError {
		t.Fatalf("create app: %#v %v", created, err)
	}
	app := created.StructuredContent.(map[string]any)["app"].(map[string]any)
	if app["id"] == "" || app["url"] == "" {
		t.Fatalf("created app: %#v", app)
	}
	if _, exposed := app["accessKey"]; exposed {
		t.Fatal("MCP app result exposed its access key")
	}
	manifest := []map[string]any{{"path": "index.html", "size": 5, "sha256": digest([]byte("hello"))}, {"path": "z.js", "size": 8, "sha256": digest([]byte("hello-js"))}}
	prepareArgs := map[string]any{"appId": app["id"], "requestId": id(), "manifest": manifest}
	prepared, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "prepare_deployment", Arguments: prepareArgs})
	if err != nil || prepared.IsError {
		t.Fatalf("prepare: %#v %v", prepared, err)
	}
	deploymentID := prepared.StructuredContent.(map[string]any)["deploymentId"].(string)
	repeated, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "prepare_deployment", Arguments: prepareArgs})
	if err != nil || repeated.IsError || repeated.StructuredContent.(map[string]any)["deploymentId"] != deploymentID {
		t.Fatalf("prepare retry: %#v %v", repeated, err)
	}
	conflictArgs := map[string]any{"appId": app["id"], "requestId": prepareArgs["requestId"], "manifest": []map[string]any{{"path": "index.html", "size": 3, "sha256": digest([]byte("new"))}}}
	conflict, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "prepare_deployment", Arguments: conflictArgs})
	if err != nil || !conflict.IsError || conflict.StructuredContent.(map[string]any)["error"] != "idempotency_conflict" {
		t.Fatalf("idempotency conflict missing: %#v %v", conflict, err)
	}
	deploymentArgs := map[string]any{"appId": app["id"], "deploymentId": deploymentID}
	status, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_deployment", Arguments: deploymentArgs})
	if err != nil || status.IsError {
		t.Fatalf("status: %#v %v", status, err)
	}
	incomplete, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "publish_deployment", Arguments: deploymentArgs})
	if err != nil || !incomplete.IsError {
		t.Fatalf("incomplete publish accepted: %#v %v", incomplete, err)
	}
	f.server.Config.PublicationPaused = true
	paused, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "publish_deployment", Arguments: deploymentArgs})
	if err != nil || !paused.IsError {
		t.Fatalf("paused publish accepted: %#v %v", paused, err)
	}
	f.server.Config.PublicationPaused = false
	chunkArgs := map[string]any{"appId": app["id"], "deploymentId": deploymentID, "fileIndex": 0, "offsetBytes": 0, "sizeBytes": 5, "sha256": digest([]byte("hello")), "dataBase64": base64.StdEncoding.EncodeToString([]byte("hello"))}
	uploaded, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "upload_file_chunk", Arguments: chunkArgs})
	if err != nil || uploaded.IsError || uploaded.StructuredContent.(map[string]any)["nextOffsetBytes"] != float64(5) {
		t.Fatalf("upload: %#v %v", uploaded, err)
	}
	uploaded, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "upload_file_chunk", Arguments: chunkArgs})
	if err != nil || uploaded.IsError {
		t.Fatalf("chunk retry: %#v %v", uploaded, err)
	}
	conflictingChunk := map[string]any{}
	for key, value := range chunkArgs {
		conflictingChunk[key] = value
	}
	conflictingChunk["sha256"] = digest([]byte("world"))
	conflictingChunk["dataBase64"] = base64.StdEncoding.EncodeToString([]byte("world"))
	chunkConflict, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "upload_file_chunk", Arguments: conflictingChunk})
	if err != nil || !chunkConflict.IsError {
		t.Fatalf("chunk conflict accepted: %#v %v", chunkConflict, err)
	}
	progress, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_deployment", Arguments: deploymentArgs})
	if err != nil || progress.IsError || progress.StructuredContent.(map[string]any)["nextOffsetBytes"].(map[string]any)["0"] != float64(5) {
		t.Fatalf("progress: %#v %v", progress, err)
	}
	completed, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "complete_file", Arguments: map[string]any{"appId": app["id"], "deploymentId": deploymentID, "fileIndex": 0}})
	if err != nil || completed.IsError {
		t.Fatalf("complete: %#v %v", completed, err)
	}
	stillIncomplete, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "publish_deployment", Arguments: deploymentArgs})
	if err != nil || !stillIncomplete.IsError {
		t.Fatalf("multi-file incomplete publish accepted: %#v %v", stillIncomplete, err)
	}
	secondChunk := map[string]any{"appId": app["id"], "deploymentId": deploymentID, "fileIndex": 1, "offsetBytes": 0, "sizeBytes": 8, "sha256": digest([]byte("hello-js")), "dataBase64": base64.StdEncoding.EncodeToString([]byte("hello-js"))}
	if result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "upload_file_chunk", Arguments: secondChunk}); err != nil || result.IsError {
		t.Fatalf("second file upload: %#v %v", result, err)
	}
	if result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "complete_file", Arguments: map[string]any{"appId": app["id"], "deploymentId": deploymentID, "fileIndex": 1}}); err != nil || result.IsError {
		t.Fatalf("second file complete: %#v %v", result, err)
	}
	published, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "publish_deployment", Arguments: deploymentArgs})
	if err != nil || published.IsError || published.StructuredContent.(map[string]any)["status"] != "published" {
		t.Fatalf("publish: %#v %v", published, err)
	}
	if published.StructuredContent.(map[string]any)["url"] != app["url"] {
		t.Fatal("MCP publication changed the stable application URL")
	}
	for path, expected := range map[string]string{"index.html": "hello", "z.js": "hello-js"} {
		object, err := f.storage.Get(ctx, objectKey(deploymentID, path))
		if err != nil {
			t.Fatal(err)
		}
		content, err := io.ReadAll(object)
		object.Close()
		if err != nil || string(content) != expected {
			t.Fatalf("published %s: %q %v", path, content, err)
		}
	}
	apps, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_apps"})
	if err != nil || apps.IsError || len(apps.StructuredContent.(map[string]any)["apps"].([]any)) != 1 {
		t.Fatalf("list apps: %#v %v", apps, err)
	}
	otherOwner, otherAccess := id(), token()
	if _, err := f.db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$2)`, otherOwner, otherOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO mcp_oauth_consents(owner_id,client_id,scopes) VALUES($1,$2,ARRAY['mcp:read'])`, otherOwner, clientID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO mcp_oauth_tokens(hash,owner_id,client_id,resource,scopes,expires_at) VALUES($1,$2,$3,$4,ARRAY['mcp:read'],now()+interval '1 hour')`, keyed(f.server.Config.Secret, otherAccess), otherOwner, clientID, f.server.Config.ControlOrigin+"/mcp"); err != nil {
		t.Fatal(err)
	}
	otherSession := connect(otherAccess)
	otherApps, err := otherSession.CallTool(ctx, &mcp.CallToolParams{Name: "list_apps"})
	if err != nil || otherApps.IsError || len(otherApps.StructuredContent.(map[string]any)["apps"].([]any)) != 0 {
		t.Fatalf("cross-owner list: %#v %v", otherApps, err)
	}
	if _, err := f.db.Exec(ctx, `UPDATE mcp_oauth_tokens SET scopes=ARRAY['mcp:read'] WHERE hash=$1`, keyed(f.server.Config.Secret, access)); err != nil {
		t.Fatal(err)
	}
	denied, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "create_app", Arguments: map[string]any{"name": "Denied", "requestId": id()}})
	if err != nil || !denied.IsError {
		t.Fatalf("read-only create was accepted: %#v %v", denied, err)
	}
	browser := token()
	if _, err := f.db.Exec(ctx, `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 hour')`, keyed(f.server.Config.Secret, browser), f.owner); err != nil {
		t.Fatal(err)
	}
	connections := response[[]mcpConnection](t, f.request(t, "GET", "/api/console/mcp/connections", nil, "", "control.localhost:8443", "__Host-owner="+browser), 200)
	if len(connections) != 1 || connections[0].ClientID != clientID {
		t.Fatalf("connections: %#v", connections)
	}
	if w := f.request(t, "POST", "/api/console/mcp/connections/"+clientID+"/revoke", nil, "", "control.localhost:8443", "__Host-owner="+browser); w.Code != 403 {
		t.Fatal("missing Origin accepted for revocation")
	}
	revoke := httptest.NewRequest("POST", "https://control.localhost:8443/api/console/mcp/connections/"+clientID+"/revoke", nil)
	revoke.Header.Set("Cookie", "__Host-owner="+browser)
	revoke.Header.Set("Origin", f.server.Config.ControlOrigin)
	result := httptest.NewRecorder()
	f.handler.ServeHTTP(result, revoke)
	if result.Code != 200 {
		t.Fatalf("revoke: %d %s", result.Code, result.Body.String())
	}
	if _, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_apps"}); err == nil {
		t.Fatal("revoked token was accepted")
	}
}
