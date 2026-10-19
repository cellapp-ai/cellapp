package hosting

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMCPClientIDMetadataDocument(t *testing.T) {
	f := setup(t)
	var metadataURL string
	metadata := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{"client_id": metadataURL, "client_name": "Metadata client", "redirect_uris": []string{"http://127.0.0.1:49152/callback"}, "grant_types": []string{"authorization_code"}, "response_types": []string{"code"}, "token_endpoint_auth_method": "none"})
	}))
	defer metadata.Close()
	metadataURL = metadata.URL + "/client.json"
	f.server.MCPMetadataHTTPClient = metadata.Client()
	browser := token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 hour')`, keyed(f.server.Config.Secret, browser), f.owner); err != nil {
		t.Fatal(err)
	}
	verifier := strings.Repeat("c", 43)
	sum := sha256.Sum256([]byte(verifier))
	query := url.Values{"response_type": {"code"}, "client_id": {metadataURL}, "redirect_uri": {"http://127.0.0.1:49152/callback"}, "resource": {f.server.Config.ControlOrigin + "/mcp"}, "scope": {"mcp:read"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"}}
	w := f.request(t, "GET", "/oauth/authorize?"+query.Encode(), nil, "", "control.localhost:8443", "__Host-owner="+browser)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Metadata client") {
		t.Fatalf("metadata authorization: %d %s", w.Code, w.Body.String())
	}
	var stored string
	if err := f.db.QueryRow(context.Background(), `SELECT metadata_uri FROM mcp_oauth_clients WHERE id=$1`, metadataURL).Scan(&stored); err != nil || stored != metadataURL {
		t.Fatalf("metadata client not registered: %q %v", stored, err)
	}
}

func TestMCPOAuthAuthorizationCodeAndRevocation(t *testing.T) {
	f := setup(t)
	host := "control.localhost:8443"
	resource := f.server.Config.ControlOrigin + "/mcp"
	registered := response[map[string]any](t, f.request(t, "POST", "/oauth/register", map[string]any{
		"client_name": "Codex test", "redirect_uris": []string{"http://127.0.0.1:49152/callback"},
		"grant_types": []string{"authorization_code", "refresh_token"},
		"client_uri":  "https://example.com", "software_id": "codex", "scope": "mcp:read mcp:deploy",
	}, "", host, ""), 201)
	if grants := registered["grant_types"].([]any); len(grants) != 1 || grants[0] != "authorization_code" {
		t.Fatalf("registration advertised unsupported grant: %v", grants)
	}
	client := registered["client_id"].(string)
	browser := token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, browser), f.owner); err != nil {
		t.Fatal(err)
	}
	verifier := strings.Repeat("a", 43)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	values := url.Values{"response_type": {"code"}, "client_id": {client}, "redirect_uri": {"http://127.0.0.1:49152/callback"}, "resource": {resource}, "scope": {"mcp:read mcp:deploy"}, "state": {"state-1"}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}
	path := "/oauth/authorize?" + values.Encode()
	if w := f.request(t, "GET", path, nil, "", host, "__Host-owner="+browser); w.Code != 200 || !strings.Contains(w.Body.String(), "Codex test") {
		t.Fatalf("consent page: %d %s", w.Code, w.Body.String())
	}
	wrong := url.Values{}
	for key, entry := range values {
		wrong[key] = append([]string(nil), entry...)
	}
	wrong.Set("redirect_uri", "https://attacker.example/callback")
	response[any](t, f.request(t, "GET", "/oauth/authorize?"+wrong.Encode(), nil, "", host, "__Host-owner="+browser), 400)
	wrong.Set("redirect_uri", values.Get("redirect_uri"))
	wrong.Set("resource", "https://other.example/mcp")
	response[any](t, f.request(t, "GET", "/oauth/authorize?"+wrong.Encode(), nil, "", host, "__Host-owner="+browser), 400)
	values.Set("decision", "approve")
	w := f.request(t, "POST", "/oauth/authorize", values.Encode(), "", host, "__Host-owner="+browser)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("consent response: %d %s", w.Code, w.Body.String())
	}
	callback, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := callback.Query().Get("code")
	if code == "" || callback.Query().Get("state") != "state-1" {
		t.Fatalf("invalid callback: %s", callback.String())
	}
	exchange := url.Values{"grant_type": {"authorization_code"}, "client_id": {client}, "redirect_uri": {"http://127.0.0.1:49152/callback"}, "resource": {resource}, "code": {code}, "code_verifier": {verifier}}
	bad := url.Values{}
	for key, entry := range exchange {
		bad[key] = append([]string(nil), entry...)
	}
	bad.Set("code_verifier", strings.Repeat("b", 43))
	if got := response[map[string]any](t, f.request(t, "POST", "/oauth/token", bad.Encode(), "", host, ""), 400); got["error"] != "invalid_grant" {
		t.Fatal(got)
	}
	bad.Set("code_verifier", verifier)
	bad.Set("client_id", id())
	if got := response[map[string]any](t, f.request(t, "POST", "/oauth/token", bad.Encode(), "", host, ""), 400); got["error"] != "invalid_grant" {
		t.Fatal(got)
	}
	bad.Set("client_id", client)
	bad.Set("resource", "https://other.example/mcp")
	if got := response[map[string]any](t, f.request(t, "POST", "/oauth/token", bad.Encode(), "", host, ""), 400); got["error"] != "invalid_request" {
		t.Fatal(got)
	}
	issued := response[map[string]any](t, f.request(t, "POST", "/oauth/token", exchange.Encode(), "", host, ""), 200)
	access := issued["access_token"].(string)
	if owner, scopes, err := f.server.mcpPrincipal(context.Background(), access); err != nil || owner != f.owner || !mcpHasScope(scopes, "mcp:deploy") || mcpHasScope(scopes, "mcp:data") {
		t.Fatalf("principal: %q %v %v", owner, scopes, err)
	}
	if got := response[map[string]any](t, f.request(t, "POST", "/oauth/token", exchange.Encode(), "", host, ""), 400); got["error"] != "invalid_grant" {
		t.Fatal(got)
	}
	expired := token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO mcp_oauth_codes(hash,owner_id,client_id,redirect_uri,resource,scopes,code_challenge,expires_at) VALUES($1,$2,$3,$4,$5,ARRAY['mcp:read'],$6,now()-interval '1 second')`, keyed(f.server.Config.Secret, expired), f.owner, client, exchange.Get("redirect_uri"), resource, challenge); err != nil {
		t.Fatal(err)
	}
	exchange.Set("code", expired)
	if got := response[map[string]any](t, f.request(t, "POST", "/oauth/token", exchange.Encode(), "", host, ""), 400); got["error"] != "invalid_grant" {
		t.Fatal(got)
	}
	if _, _, err := f.server.mcpPrincipal(context.Background(), f.token); err == nil {
		t.Fatal("CLI credential accepted as MCP token")
	}
	revoke := url.Values{"client_id": {client}, "token": {access}}
	if w := f.request(t, "POST", "/oauth/revoke", revoke.Encode(), "", host, ""); w.Code != 200 {
		t.Fatalf("revoke: %d", w.Code)
	}
	if _, _, err := f.server.mcpPrincipal(context.Background(), access); err == nil {
		t.Fatal("revoked MCP token accepted")
	}
}
