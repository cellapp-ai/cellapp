package hosting

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func modelChatRequest(f *fixture, method, path, host, cookie, origin string, body any) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	r := httptest.NewRequest(method, "https://"+host+path, bytes.NewReader(data))
	r.Host = host
	r.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		r.Header.Set("Cookie", cookie)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}

func TestModelChatAdmissionAndLimits(t *testing.T) {
	f := setup(t)
	f.server.Config.ModelEncryptionKey = []byte(strings.Repeat("m", 32))
	app, _ := f.create(t)
	appHost, _ := url.Parse(app.URL)
	ownerSession := f.webSessionCookie(t)
	appCookie := signedInApp(t, f, app, ownerSession)
	connection := response[modelConnectionView](t, f.webRequest(t, "POST", "/model-connections", map[string]string{
		"name": "Personal", "apiBase": "https://models.example.com/v1", "modelId": "model-one", "apiKey": "provider-secret",
	}, ownerSession, f.server.Config.ControlOrigin), 201)
	if _, err := f.db.Exec(context.Background(), `UPDATE model_connections SET capabilities='{"text":{"supported":true},"stream":{"supported":true},"tools":{"supported":true}}' WHERE id=$1`, connection.ID); err != nil {
		t.Fatal(err)
	}
	calls := 0
	f.server.ModelTestHTTPClient = &http.Client{Transport: modelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer provider-secret" {
			t.Fatal("unsafe upstream request")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || !bytes.Contains(body, []byte(`"model":"model-one"`)) {
			t.Fatal("selected model not fixed by server")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"pong"}}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`))}, nil
	})}
	model := cellappModelID(connection.ID, "model-one")
	input := map[string]any{"model": model, "messages": []map[string]string{{"role": "user", "content": "ping"}}}
	path := "/_cellapp/v1/chat/completions"
	for _, tc := range []struct {
		path, cookie, origin string
		status               int
	}{
		{path, "", appURLOrigin(appHost.Host), 401},
		{path, appCookie, "", 403},
		{path, appCookie, "https://outside.example", 403},
		{"/_cellapp/v1/chat/completions?v=1.1", appCookie, appURLOrigin(appHost.Host), 400},
		{"/_cellapp/v1/chat/completions?v=unregistered", appCookie, appURLOrigin(appHost.Host), 400},
		{"/_cellapp/ai/chat", appCookie, appURLOrigin(appHost.Host), 405},
		{path, appCookie, appURLOrigin(appHost.Host), 403},
	} {
		got := modelChatRequest(f, "POST", tc.path, appHost.Host, tc.cookie, tc.origin, input)
		if got.Code != tc.status {
			t.Fatalf("%s returned %d, expected %d: %s", tc.path, got.Code, tc.status, got.Body.String())
		}
	}
	if calls != 0 {
		t.Fatal("rejected requests reached provider")
	}
	unauthorized := modelChatRequest(f, "POST", path, appHost.Host, appCookie, appURLOrigin(appHost.Host), input)
	if unauthorized.Code != 403 || !strings.Contains(unauthorized.Body.String(), "authorizeUrl") {
		t.Fatalf("missing authorization link: %d %s", unauthorized.Code, unauthorized.Body.String())
	}
	withConnection := map[string]any{"connectionId": connection.ID, "model": model, "messages": input["messages"]}
	if got := modelChatRequest(f, "POST", path, appHost.Host, appCookie, appURLOrigin(appHost.Host), withConnection); got.Code != 400 || calls != 0 {
		t.Fatalf("application-selected connection accepted: %d", got.Code)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO model_connection_grants(owner_id,app_id,connection_id) VALUES($1,$2,$3)`, f.owner, app.ID, connection.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO model_request_usage(id,owner_id,app_id,connection_id,status) VALUES($1,$3,$4,$5,'started'),($2,$3,$4,$5,'started')`, id(), id(), f.owner, app.ID, connection.ID); err != nil {
		t.Fatal(err)
	}
	if limited := modelChatRequest(f, "POST", path, appHost.Host, appCookie, appURLOrigin(appHost.Host), input); limited.Code != 429 || calls != 0 {
		t.Fatalf("concurrency limit failed: %d", limited.Code)
	}
	if _, err := f.db.Exec(context.Background(), `DELETE FROM model_request_usage WHERE app_id=$1 AND status='started'`, app.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO rate_limits(key,count,expires_at) VALUES($1,20,now()+interval '1 minute')
	ON CONFLICT(key) DO UPDATE SET count=20,expires_at=now()+interval '1 minute'`, keyed(f.server.Config.Secret, "model:"+f.owner+":"+app.ID)); err != nil {
		t.Fatal(err)
	}
	if limited := modelChatRequest(f, "POST", path, appHost.Host, appCookie, appURLOrigin(appHost.Host), input); limited.Code != 429 || calls != 0 {
		t.Fatalf("rate limit failed: %d", limited.Code)
	}
	if _, err := f.db.Exec(context.Background(), `DELETE FROM rate_limits WHERE key=$1`, keyed(f.server.Config.Secret, "model:"+f.owner+":"+app.ID)); err != nil {
		t.Fatal(err)
	}
	got := modelChatRequest(f, "POST", path, appHost.Host, appCookie, appURLOrigin(appHost.Host), input)
	if got.Code != 200 || !strings.Contains(got.Body.String(), "pong") || calls != 1 {
		t.Fatalf("authorized request failed: %d %s", got.Code, got.Body.String())
	}
	var status string
	var tokens int64
	if err := f.db.QueryRow(context.Background(), `SELECT status,input_tokens FROM model_request_usage WHERE app_id=$1`, app.ID).Scan(&status, &tokens); err != nil || status != "succeeded" || tokens != 3 {
		t.Fatalf("usage not recorded: %s %d %v", status, tokens, err)
	}
	usage := response[[]modelUsageView](t, f.webRequest(t, "GET", "/model-usage", nil, ownerSession, ""), 200)
	if len(usage) != 1 || usage[0].AppID != app.ID || usage[0].ConnectionID != connection.ID || usage[0].Requests != 1 || usage[0].Succeeded != 1 || usage[0].InputTokens != 3 || usage[0].UnknownUsage != 0 {
		t.Fatalf("usage summary incorrect: %+v", usage)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE model_connection_grants SET revoked_at=now() WHERE owner_id=$1 AND app_id=$2`, f.owner, app.ID); err != nil {
		t.Fatal(err)
	}
	if got := modelChatRequest(f, "POST", path, appHost.Host, appCookie, appURLOrigin(appHost.Host), input); got.Code != 403 || calls != 1 {
		t.Fatalf("revocation did not stop new request: %d", got.Code)
	}
	large := map[string]any{"model": model, "messages": []map[string]string{{"role": "user", "content": strings.Repeat("x", 1<<20)}}}
	if got := modelChatRequest(f, "POST", path, appHost.Host, appCookie, appURLOrigin(appHost.Host), large); got.Code != 400 || calls != 1 {
		t.Fatalf("oversized request reached provider: %d", got.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE browser_sessions SET expires_at=now()-interval '1 second' WHERE hash=$1`, keyed(f.server.Config.Secret, ownerSession)); err != nil {
		t.Fatal(err)
	}
	if got := modelChatRequest(f, "POST", path, appHost.Host, appCookie, appURLOrigin(appHost.Host), input); got.Code != 401 || calls != 1 {
		t.Fatalf("expired session reached provider: %d", got.Code)
	}
}

func appURLOrigin(host string) string { return "https://" + host }
