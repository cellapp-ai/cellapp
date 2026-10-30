package hosting

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestModelCatalogUnionAndPerProviderAuthorization(t *testing.T) {
	f := setup(t)
	f.server.Config.ModelEncryptionKey = []byte(strings.Repeat("m", 32))
	app, _ := f.create(t)
	otherApp, _ := f.create(t)
	host, _ := url.Parse(app.URL)
	otherHost, _ := url.Parse(otherApp.URL)
	session := f.webSessionCookie(t)
	cookie := signedInApp(t, f, app, session)
	otherCookie := signedInApp(t, f, otherApp, session)
	makeConnection := func(name, base, key string) modelConnectionView {
		connection := response[modelConnectionView](t, f.webRequest(t, "POST", "/model-connections", map[string]string{
			"name": name, "apiBase": base, "modelId": "manual", "apiKey": key,
		}, session, f.server.Config.ControlOrigin), 201)
		if _, err := f.db.Exec(context.Background(), `UPDATE model_connections SET capabilities='{"text":{"supported":true},"stream":{"supported":true},"tools":{"supported":true}}' WHERE id=$1`, connection.ID); err != nil {
			t.Fatal(err)
		}
		return connection
	}
	first := makeConnection("First provider", "https://first.example.com/v1", "first-secret")
	second := makeConnection("Second provider", "https://second.example.com/v1", "second-secret")
	for _, connection := range []modelConnectionView{first, second} {
		if _, err := f.db.Exec(context.Background(), `INSERT INTO model_connection_grants(owner_id,app_id,connection_id) VALUES($1,$2,$3)`, f.owner, app.ID, connection.ID); err != nil {
			t.Fatal(err)
		}
	}
	catalogFailure := false
	f.server.ModelTestHTTPClient = &http.Client{Transport: modelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body string
		if r.Method == "GET" {
			if catalogFailure {
				return &http.Response{StatusCode: 503, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("unavailable"))}, nil
			}
			if r.URL.Path != "/v1/models" {
				t.Fatalf("wrong model catalog path: %s", r.URL.Path)
			}
			if r.URL.Host == "first.example.com" {
				body = `{"data":[{"id":"shared"},{"id":"first-only"}]}`
			} else {
				body = `{"data":[{"id":"shared"},{"id":"second-only"}]}`
			}
		} else {
			if r.URL.Host != "second.example.com" || r.Header.Get("Authorization") != "Bearer second-secret" {
				t.Fatal("model request reached wrong provider")
			}
			data, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(data), `"model":"shared"`) {
				t.Fatalf("provider model not unwrapped: %s", data)
			}
			body = `{"choices":[{"message":{"content":"ok"}}]}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	list := modelChatRequest(f, "GET", "/_cellapp/v1/models", host.Host, cookie, "", nil)
	if list.Code != 200 {
		t.Fatalf("model list: %d %s", list.Code, list.Body.String())
	}
	var catalog struct {
		Object string          `json:"object"`
		Data   []modelListItem `json:"data"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.Object != "list" || len(catalog.Data) != 6 {
		t.Fatalf("expected two manual and four discovered models: %+v", catalog)
	}
	ownerModels := f.webRequest(t, "GET", "/models", nil, session, "")
	if ownerModels.Code != 200 || json.Unmarshal(ownerModels.Body.Bytes(), &catalog) != nil || len(catalog.Data) != 6 {
		t.Fatalf("owner model list: %d %s", ownerModels.Code, ownerModels.Body.String())
	}
	for _, connection := range []modelConnectionView{first, second} {
		found := false
		for _, model := range catalog.Data {
			if model.ID == cellappModelID(connection.ID, "shared") {
				found = true
			}
		}
		if !found {
			t.Fatalf("shared model lost connection namespace: %s", connection.ID)
		}
	}
	catalogFailure = true
	fallback := modelChatRequest(f, "GET", "/_cellapp/v1/models", host.Host, cookie, "", nil)
	var manual struct {
		Data []modelListItem `json:"data"`
	}
	if fallback.Code != 200 || json.Unmarshal(fallback.Body.Bytes(), &manual) != nil || len(manual.Data) != 2 {
		t.Fatalf("manual catalog fallback failed: %d %s", fallback.Code, fallback.Body.String())
	}
	catalogFailure = false
	if got := modelChatRequest(f, "GET", "/_cellapp/v1/models", otherHost.Host, otherCookie, "", nil); got.Code != 403 || !strings.Contains(got.Body.String(), `"code":"model_authorization_required"`) {
		t.Fatalf("ungranted application saw model list: %d %s", got.Code, got.Body.String())
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE model_connections SET capabilities='{"text":{"supported":false},"stream":{"supported":false},"tools":{"supported":false}}' WHERE id=$1`, second.ID); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"model": cellappModelID(second.ID, "shared"), "messages": []map[string]string{{"role": "user", "content": "ping"}}}
	if got := modelChatRequest(f, "POST", "/_cellapp/v1/chat/completions", host.Host, cookie, "https://"+host.Host, input); got.Code != 200 {
		t.Fatalf("other model inherited test-model capability result: %d %s", got.Code, got.Body.String())
	}
	input["model"] = cellappModelID(second.ID, "manual")
	if got := modelChatRequest(f, "POST", "/_cellapp/v1/chat/completions", host.Host, cookie, "https://"+host.Host, input); got.Code != 409 {
		t.Fatalf("untested capability on test model was accepted: %d %s", got.Code, got.Body.String())
	}
	input["model"] = cellappModelID(second.ID, "shared")
	if got := modelChatRequest(f, "POST", "/_cellapp/v1/chat/completions", otherHost.Host, otherCookie, "https://"+otherHost.Host, input); got.Code != 403 {
		t.Fatalf("another application reused provider grant: %d", got.Code)
	}
	visitor, visitorSession := id(), token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$1)`, visitor); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, visitorSession), visitor); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET visibility='public' WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	visitorCookie := signedInApp(t, f, app, visitorSession)
	if got := modelChatRequest(f, "POST", "/_cellapp/v1/chat/completions", host.Host, visitorCookie, "https://"+host.Host, input); got.Code != 403 {
		t.Fatalf("another visitor reused provider grant: %d", got.Code)
	}
	response[any](t, f.webRequest(t, "POST", "/model-grants/"+app.ID+"/"+second.ID+"/revoke", nil, session, f.server.Config.ControlOrigin), 200)
	if got := modelChatRequest(f, "POST", "/_cellapp/v1/chat/completions", host.Host, cookie, "https://"+host.Host, input); got.Code != 403 {
		t.Fatalf("revoked provider remained callable: %d", got.Code)
	}
	if active, err := f.server.modelGrantActive(context.Background(), f.owner, app.ID, first.ID); err != nil || !active {
		t.Fatalf("other provider affected by revocation: %t %v", active, err)
	}
}
