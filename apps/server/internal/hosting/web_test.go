package hosting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func (f *fixture) webRequest(t *testing.T, method, path string, body any, session, origin string) *httptest.ResponseRecorder {
	t.Helper()
	var content string
	if body != nil {
		content = string(encoded(body))
	}
	r := httptest.NewRequest(method, "https://control.localhost:8443/api/console"+path, strings.NewReader(content))
	if session != "" {
		r.AddCookie(&http.Cookie{Name: "__Host-owner", Value: session})
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}
func (f *fixture) webSessionCookie(t *testing.T) string {
	t.Helper()
	session := token()
	if _, e := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, session), f.owner); e != nil {
		t.Fatal(e)
	}
	return session
}
func TestWebOwnerManagement(t *testing.T) {
	f := setup(t)
	session := f.webSessionCookie(t)
	origin := f.server.Config.ControlOrigin
	response[any](t, f.webRequest(t, "GET", "/session", nil, session, ""), 200)
	response[any](t, f.webRequest(t, "GET", "/session", nil, "", ""), 401)
	apps := response[[]App](t, f.webRequest(t, "GET", "/apps", nil, session, ""), 200)
	if len(apps) != 0 {
		t.Fatal(apps)
	}
	a, oldKey := f.create(t)
	info := response[map[string]any](t, f.webRequest(t, "GET", "/apps/"+a.ID, nil, session, ""), 200)
	if info["release"] != nil {
		t.Fatal(info)
	}
	d, _ := f.stage(t, a, "test")
	f.uploadAndPublish(t, a, d, "test")
	info = response[map[string]any](t, f.webRequest(t, "GET", "/apps/"+a.ID, nil, session, ""), 200)
	if info["release"] == nil {
		t.Fatal(info)
	}
	visitor := f.unlock(t, a, oldKey)
	for _, badOrigin := range []string{"", "https://outside.example", "https://" + a.ID + ".apps.localhost:8443"} {
		response[any](t, f.webRequest(t, "POST", "/apps/"+a.ID+"/key", nil, session, badOrigin), 403)
	}
	result := response[map[string]string](t, f.webRequest(t, "POST", "/apps/"+a.ID+"/key", nil, session, origin), 200)
	if !keyValid(result["key"]) || result["key"] == oldKey {
		t.Fatal("invalid key")
	}
	u, _ := url.Parse(a.URL)
	if w := f.request(t, "HEAD", "/main.js", nil, "", u.Host, visitor); w.Code != 401 {
		t.Fatal("visitor session survived reset")
	}
	f.unlock(t, a, result["key"])
	if strings.Contains(f.logs.String(), result["key"]) {
		t.Fatal("key leaked to log")
	}
	if w := f.request(t, "GET", "/apps", nil, "", "control.localhost:8443", "__Host-owner="+session); w.Code != 401 {
		t.Fatal("cookie authenticated Bearer API")
	}
	response[any](t, f.webRequest(t, "DELETE", "/apps/"+a.ID, nil, session, origin), 200)
	response[any](t, f.webRequest(t, "GET", "/apps/"+a.ID, nil, session, ""), 404)
	response[any](t, f.webRequest(t, "POST", "/apps/"+a.ID+"/key", nil, session, origin), 404)
	response[any](t, f.webRequest(t, "POST", "/logout", nil, session, origin), 200)
	response[any](t, f.webRequest(t, "GET", "/session", nil, session, ""), 401)
}
func TestWebResourceScopeAndCredentials(t *testing.T) {
	f := setup(t)
	session := f.webSessionCookie(t)
	origin := f.server.Config.ControlOrigin
	a, _ := f.create(t)
	other := id()
	otherSession := token()
	otherCredential := id()
	ctx := context.Background()
	if _, e := f.db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','other')`, other); e != nil {
		t.Fatal(e)
	}
	if _, e := f.db.Exec(ctx, `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, otherSession), other); e != nil {
		t.Fatal(e)
	}
	if _, e := f.db.Exec(ctx, `INSERT INTO credentials(id,hash,owner_id,expires_at) VALUES($1,$2,$3,now()+interval '1 day')`, otherCredential, keyed(f.server.Config.Secret, token()), other); e != nil {
		t.Fatal(e)
	}
	response[any](t, f.webRequest(t, "GET", "/apps/"+a.ID, nil, otherSession, ""), 404)
	response[any](t, f.webRequest(t, "DELETE", "/apps/"+a.ID, nil, otherSession, origin), 404)
	response[any](t, f.webRequest(t, "POST", "/apps/"+a.ID+"/key", nil, otherSession, origin), 404)
	response[any](t, f.webRequest(t, "POST", "/credentials/"+otherCredential+"/revoke", nil, session, origin), 404)
	items := response[[]webCredential](t, f.webRequest(t, "GET", "/credentials", nil, session, ""), 200)
	if len(items) != 1 {
		t.Fatal(items)
	}
	for i := 0; i < 2; i++ {
		response[any](t, f.webRequest(t, "POST", "/credentials/"+items[0].ID+"/revoke", nil, session, origin), 200)
	}
	response[any](t, f.request(t, "GET", "/apps", nil, f.token, "control.localhost:8443", ""), 401)
	if _, e := f.db.Exec(ctx, `UPDATE browser_sessions SET expires_at=now()-interval '1 second' WHERE hash=$1`, keyed(f.server.Config.Secret, session)); e != nil {
		t.Fatal(e)
	}
	response[any](t, f.webRequest(t, "GET", "/session", nil, session, ""), 401)
}

func TestWebDeviceExpiryAndConcurrentDecision(t *testing.T) {
	f := setup(t)
	session := f.webSessionCookie(t)
	ctx := context.Background()
	expired := response[map[string]any](t, f.request(t, "POST", "/device-authorizations", map[string]any{}, "", "control.localhost:8443", ""), 201)
	if _, e := f.db.Exec(ctx, `UPDATE device_authorizations SET expires_at=now()-interval '1 second' WHERE user_code=$1`, expired["userCode"]); e != nil {
		t.Fatal(e)
	}
	response[any](t, f.webRequest(t, "POST", "/device/decision", map[string]any{"code": expired["userCode"], "decision": "approved"}, session, f.server.Config.ControlOrigin), 400)
	code := "CONCURRENT"
	if _, e := f.db.Exec(ctx, `INSERT INTO device_authorizations(hash,user_code,expires_at) VALUES($1,$2,now()+interval '1 hour')`, keyed(f.server.Config.Secret, token()), code); e != nil {
		t.Fatal(e)
	}
	results := make(chan error, 12)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			decision := "approved"
			if i%2 == 0 {
				decision = "denied"
			}
			results <- f.server.decideDevice(ctx, f.owner, code, decision)
		}(i)
	}
	wg.Wait()
	close(results)
	completed := 0
	for e := range results {
		if e == nil {
			completed++
		}
	}
	if completed != 1 {
		t.Fatalf("device decision completed %d times", completed)
	}
}

func TestWebCredentialFiltering(t *testing.T) {
	f := setup(t)
	session := f.webSessionCookie(t)
	for _, revoked := range []bool{false, true} {
		expiry := "now()-interval '1 second'"
		if revoked {
			expiry = "now()+interval '1 day'"
		}
		if _, e := f.db.Exec(context.Background(), `INSERT INTO credentials(id,hash,owner_id,expires_at,revoked) VALUES($1,$2,$3,`+expiry+`,$4)`, id(), keyed(f.server.Config.Secret, token()), f.owner, revoked); e != nil {
			t.Fatal(e)
		}
	}
	items := response[[]webCredential](t, f.webRequest(t, "GET", "/credentials", nil, session, ""), 200)
	if len(items) != 1 {
		t.Fatal("inactive credentials exposed", items)
	}
}

func TestControlReturnPaths(t *testing.T) {
	for _, value := range []string{"https://outside.example", "//outside.example", "/api/console/apps", "/applications/%2f%2foutside.example", "/notes", "/credentials\\outside"} {
		if safeControlReturn(value) != "/" {
			t.Fatal("unsafe return", value)
		}
	}
	for _, value := range []string{"/applications", "/applications/" + id(), "/device?code=ABC", "/credentials"} {
		if safeControlReturn(value) != value {
			t.Fatal("valid return lost", value)
		}
	}
}
func TestWebDeviceDecisionValidation(t *testing.T) {
	f := setup(t)
	session := f.webSessionCookie(t)
	origin := f.server.Config.ControlOrigin
	for _, decision := range []string{"approved", "denied"} {
		device := response[map[string]any](t, f.request(t, "POST", "/device-authorizations", map[string]any{}, "", "control.localhost:8443", ""), 201)
		input := map[string]any{"code": device["userCode"], "decision": decision}
		response[any](t, f.webRequest(t, "POST", "/device/decision", input, session, ""), 403)
		response[any](t, f.webRequest(t, "POST", "/device/decision", input, session, origin), 200)
		response[any](t, f.webRequest(t, "POST", "/device/decision", input, session, origin), 400)
	}
	response[any](t, f.webRequest(t, "POST", "/device/decision", map[string]string{"code": "bad", "decision": "approved"}, session, origin), 400)
	response[any](t, f.webRequest(t, "POST", "/device/decision", map[string]string{"code": "bad", "decision": "other"}, session, origin), 400)
	r := httptest.NewRequest("POST", origin+"/api/console/device/decision", strings.NewReader(`{"code":"bad","decision":"approved"}`))
	r.Header.Set("Origin", origin)
	r.AddCookie(&http.Cookie{Name: "__Host-owner", Value: session})
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatal(w.Code)
	}
	r = httptest.NewRequest("POST", origin+"/api/console/device/decision", strings.NewReader(strings.Repeat("x", (1<<20)+1)))
	r.Header.Set("Origin", origin)
	r.AddCookie(&http.Cookie{Name: "__Host-owner", Value: session})
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
}
func TestWebConcurrentDeleteReset(t *testing.T) {
	f := setup(t)
	a, _ := f.create(t)
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_ = f.server.resetOwnerKey(context.Background(), f.owner, a.ID, token())
			} else {
				_ = f.server.deleteOwnerApp(context.Background(), f.owner, a.ID)
			}
		}(i)
	}
	wg.Wait()
	var deleted bool
	if e := f.db.QueryRow(context.Background(), `SELECT deleted FROM apps WHERE id=$1`, a.ID).Scan(&deleted); e != nil || !deleted {
		t.Fatal("deleted application resurrected", e)
	}
}
func TestWebStaticDelivery(t *testing.T) {
	root := t.TempDir()
	if e := os.Mkdir(filepath.Join(root, "assets"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<html><title>Cellapp</title></html>`), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "assets", "test-abc.js"), []byte(`console.log('test')`), 0600); e != nil {
		t.Fatal(e)
	}
	web, e := LoadWebFS(root)
	if e != nil {
		t.Fatal(e)
	}
	s := Server{Web: web}
	for _, p := range []string{"/", "/login", "/applications", "/applications/" + id(), "/credentials", "/device?code=ABC"} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", p, nil)
		if e := s.webPage(w, r); e != nil || w.Code != 200 || !strings.Contains(w.Header().Get("Content-Security-Policy"), "script-src 'self'") {
			t.Fatal(p, e)
		}
	}
	for _, p := range []string{"/api/console/missing", "/missing.js", "/applications/bad"} {
		w := httptest.NewRecorder()
		if e := s.webPage(w, httptest.NewRequest("GET", p, nil)); e == nil {
			t.Fatal("unknown path accepted", p)
		}
	}
	outside := filepath.Join(t.TempDir(), "secret.js")
	os.WriteFile(outside, []byte("private"), 0600)
	if e := os.Symlink(outside, filepath.Join(root, "assets", "secret.js")); e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{"/assets/missing.js", "/assets/secret.js", "/assets/../index.html"} {
		w := httptest.NewRecorder()
		if e := s.webAsset(w, httptest.NewRequest("GET", p, nil)); e == nil {
			t.Fatal("unsafe resource accepted", p)
		}
	}
	w := httptest.NewRecorder()
	if e := s.webAsset(w, httptest.NewRequest("GET", "/assets/test-abc.js", nil)); e != nil || w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatal(e)
	}
	if _, e := LoadWebFS(t.TempDir()); e == nil {
		t.Fatal("missing index accepted")
	}
	data, _ := json.Marshal(App{})
	if strings.Contains(string(data), "KeyHash") {
		t.Fatal("internal field exposed")
	}
}
