package hosting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func modelConsentRequest(f *fixture, t *testing.T, method, app, connection, session, origin string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/dashboard/model-consent?app=" + url.QueryEscape(app)
	var body *strings.Reader
	if method == http.MethodPost {
		path = "/dashboard/model-consent"
		body = strings.NewReader(url.Values{"app": {app}, "connection": {connection}}.Encode())
	} else {
		body = strings.NewReader("")
	}
	r := httptest.NewRequest(method, "https://control.localhost:8443"+path, body)
	if session != "" {
		r.AddCookie(&http.Cookie{Name: "__Host-owner", Value: session})
	}
	if method == http.MethodPost {
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}

func TestModelGrantConsentAndRevocation(t *testing.T) {
	f := setup(t)
	f.server.Config.ModelEncryptionKey = []byte(strings.Repeat("m", 32))
	appA, _ := f.create(t)
	appB, _ := f.create(t)
	for _, app := range []App{appA, appB} {
		if _, err := f.db.Exec(context.Background(), `UPDATE apps SET visibility='public' WHERE id=$1`, app.ID); err != nil {
			t.Fatal(err)
		}
	}
	user, session := id(), token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$1)`, user); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, session), user); err != nil {
		t.Fatal(err)
	}
	connection := response[modelConnectionView](t, f.webRequest(t, "POST", "/model-connections", map[string]string{
		"name": "Visitor connection", "apiBase": "https://models.example.com/v1", "modelId": "model-one", "apiKey": "visitor-secret",
	}, session, f.server.Config.ControlOrigin), 201)
	if got := modelConsentRequest(f, t, "GET", appA.ID, "", "", ""); got.Code != 303 || !strings.HasPrefix(got.Header().Get("Location"), "/auth/login?") {
		t.Fatalf("unsigned consent page: %d %s", got.Code, got.Header().Get("Location"))
	}
	page := modelConsentRequest(f, t, "GET", appA.ID, "", session, "")
	if page.Code != 200 || !strings.Contains(page.Body.String(), appA.Name) || !strings.Contains(page.Body.String(), "messages and data") || strings.Contains(page.Body.String(), "visitor-secret") {
		t.Fatalf("consent page did not explain app and data flow: %d", page.Code)
	}
	if active, err := f.server.modelGrantActive(context.Background(), user, appA.ID, connection.ID); err != nil || active {
		t.Fatalf("viewing or cancelling consent granted access: %t %v", active, err)
	}
	if got := modelConsentRequest(f, t, "POST", appA.ID, connection.ID, session, ""); got.Code != 403 {
		t.Fatalf("consent without origin: %d", got.Code)
	}
	if got := modelConsentRequest(f, t, "POST", appA.ID, connection.ID, session, f.server.Config.ControlOrigin); got.Code != 303 {
		t.Fatalf("consent failed: %d %s", got.Code, got.Body.String())
	}
	if active, err := f.server.modelGrantActive(context.Background(), user, appA.ID, connection.ID); err != nil || !active {
		t.Fatalf("consent not recorded: %t %v", active, err)
	}
	if active, err := f.server.modelGrantActive(context.Background(), user, appB.ID, connection.ID); err != nil || active {
		t.Fatalf("grant inherited by another app: %t %v", active, err)
	}
	if grants := response[[]modelGrantView](t, f.webRequest(t, "GET", "/model-grants", nil, session, ""), 200); len(grants) != 1 || grants[0].AppID != appA.ID {
		t.Fatal(grants)
	}
	second := response[modelConnectionView](t, f.webRequest(t, "POST", "/model-connections", map[string]string{
		"name": "Other model", "apiBase": "https://other.example.com/v1", "modelId": "model-two", "apiKey": "other-secret",
	}, session, f.server.Config.ControlOrigin), 201)
	if got := modelConsentRequest(f, t, "POST", appA.ID, second.ID, session, f.server.Config.ControlOrigin); got.Code != 303 {
		t.Fatalf("connection switch failed: %d %s", got.Code, got.Body.String())
	}
	if active, err := f.server.modelGrantActive(context.Background(), user, appA.ID, connection.ID); err != nil || !active {
		t.Fatalf("first provider authorization disappeared: %t %v", active, err)
	}
	if active, err := f.server.modelGrantActive(context.Background(), user, appA.ID, second.ID); err != nil || !active {
		t.Fatalf("new connection not active: %t %v", active, err)
	}
	if grants := response[[]modelGrantView](t, f.webRequest(t, "GET", "/model-grants", nil, session, ""), 200); len(grants) != 2 {
		t.Fatalf("expected two provider authorizations: %+v", grants)
	}
	if page := modelConsentRequest(f, t, "GET", appA.ID, "", session, ""); page.Code != 200 || !strings.Contains(page.Body.String(), `value="`+second.ID+`" required checked`) {
		t.Fatalf("current connection not selected: %d", page.Code)
	}
	authorSession := f.webSessionCookie(t)
	if grants := response[[]modelGrantView](t, f.webRequest(t, "GET", "/model-grants", nil, authorSession, ""), 200); len(grants) != 0 {
		t.Fatal("application author saw visitor's model grant")
	}
	response[any](t, f.webRequest(t, "POST", "/model-grants/"+appA.ID+"/"+connection.ID+"/revoke", nil, authorSession, f.server.Config.ControlOrigin), 404)
	response[any](t, f.webRequest(t, "POST", "/model-grants/"+appA.ID+"/"+connection.ID+"/revoke", nil, session, f.server.Config.ControlOrigin), 200)
	if active, err := f.server.modelGrantActive(context.Background(), user, appA.ID, second.ID); err != nil || !active {
		t.Fatalf("revoking first provider affected second: %t %v", active, err)
	}
	response[any](t, f.webRequest(t, "POST", "/model-grants/"+appA.ID+"/"+second.ID+"/revoke", nil, session, f.server.Config.ControlOrigin), 200)
	if active, err := f.server.modelGrantActive(context.Background(), user, appA.ID, second.ID); err != nil || active {
		t.Fatalf("revoked grant still active: %t %v", active, err)
	}
}
