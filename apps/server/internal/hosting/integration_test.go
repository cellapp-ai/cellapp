package hosting

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"ohmyapp/apps/server/migrations"
)

type memoryStorage struct {
	sync.Mutex
	data       map[string][]byte
	failDelete bool
}

func (m *memoryStorage) Put(_ context.Context, k string, r io.Reader, n int64) error {
	b, e := io.ReadAll(r)
	if e != nil {
		return e
	}
	m.Lock()
	defer m.Unlock()
	m.data[k] = b
	return nil
}
func (m *memoryStorage) Get(_ context.Context, k string) (io.ReadCloser, error) {
	m.Lock()
	defer m.Unlock()
	b, ok := m.data[k]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}
func (m *memoryStorage) Delete(_ context.Context, k string) error {
	m.Lock()
	defer m.Unlock()
	if m.failDelete {
		return os.ErrPermission
	}
	delete(m.data, k)
	return nil
}
func (m *memoryStorage) Health(context.Context) error { return nil }

type identityStub struct{}

func (identityStub) Start(state, challenge string) string {
	return "https://identity.example/authorize?state=" + state + "&code_challenge=" + challenge
}
func (identityStub) Exchange(_ context.Context, code, verifier string) (string, string, error) {
	if code != "valid" || verifier == "" {
		return "", "", os.ErrPermission
	}
	return "https://identity.example", "owner", nil
}

type fixture struct {
	server  *Server
	handler http.Handler
	db      *pgxpool.Pool
	token   string
	owner   string
	logs    *bytes.Buffer
	storage *memoryStorage
}

func setup(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required for real PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	schema := "test_" + id()
	if _, e = admin.Exec(ctx, `CREATE SCHEMA `+schema); e != nil {
		t.Fatal(e)
	}
	pc, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	pc.ConnConfig.RuntimeParams["search_path"] = schema
	db, e := pgxpool.NewWithConfig(ctx, pc)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close(); _, _ = admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`); admin.Close() })
	if e = migrations.Run(ctx, db); e != nil {
		t.Fatal(e)
	}
	if e = migrations.Run(ctx, db); e != nil {
		t.Fatal("migration not repeatable", e)
	}
	c, e := LoadConfig(func(string) string { return "" })
	if e != nil {
		t.Fatal(e)
	}
	c.Limits.Requests = 10000
	c.AuthMode = "github"
	logs := new(bytes.Buffer)
	storage := &memoryStorage{data: map[string][]byte{}}
	s := &Server{Config: c, DB: db, Storage: storage, Identity: identityStub{}, Logger: slog.New(slog.NewJSONHandler(logs, nil))}
	owner, credential := id(), token()
	_, e = db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','owner')`, owner)
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec(ctx, `INSERT INTO credentials(id,hash,owner_id,expires_at) VALUES($1,$2,$3,now()+interval '1 day')`, id(), keyed(c.Secret, credential), owner)
	if e != nil {
		t.Fatal(e)
	}
	return &fixture{s, s.Handler(), db, credential, owner, logs, storage}
}
func (f *fixture) request(t *testing.T, method, path string, body any, credential, host, cookie string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	switch v := body.(type) {
	case []byte:
		reader = bytes.NewReader(v)
	case string:
		reader = strings.NewReader(v)
	case nil:
	default:
		reader = bytes.NewReader(encoded(v))
	}
	req := httptest.NewRequest(method, "https://"+host+path, reader)
	req.Host = host
	if credential != "" {
		req.Header.Set("Authorization", "Bearer "+credential)
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/html")
	if _, ok := body.(string); ok {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "https://"+host)
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	return w
}
func response[T any](t *testing.T, w *httptest.ResponseRecorder, expected int) T {
	t.Helper()
	if w.Code != expected {
		t.Fatalf("expected %d got %d: %s", expected, w.Code, w.Body.String())
	}
	var v T
	if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	return v
}
func (f *fixture) create(t *testing.T) (App, string) {
	key := token()
	w := f.request(t, "POST", "/apps", map[string]any{"name": "Notes", "key": key, "requestId": id()}, f.token, "control.localhost:8443", "")
	return response[App](t, w, 201), key
}
func (f *fixture) stage(t *testing.T, a App, content string) (string, []Entry) {
	files := []Entry{{"index.html", int64(len(content)), digest([]byte(content))}, {"main.js", 1, digest([]byte("x"))}}
	w := f.request(t, "POST", "/apps/"+a.ID+"/deployments", map[string]any{"requestId": id(), "manifest": files, "baseVersion": a.Active, "spa": true}, f.token, "control.localhost:8443", "")
	d := response[map[string]string](t, w, 201)["id"]
	return d, files
}
func (f *fixture) uploadAndPublish(t *testing.T, a App, d, content string) {
	base := "/apps/" + a.ID + "/deployments/" + d
	response[any](t, f.request(t, "PUT", base+"/files/0", []byte(content), f.token, "control.localhost:8443", ""), 200)
	response[any](t, f.request(t, "PUT", base+"/files/1", []byte("x"), f.token, "control.localhost:8443", ""), 200)
	response[any](t, f.request(t, "POST", base+"/publish", map[string]any{}, f.token, "control.localhost:8443", ""), 200)
}
func (f *fixture) unlock(t *testing.T, a App, key string) string {
	u, _ := url.Parse(a.URL)
	w := f.request(t, "POST", "/_hosting/unlock", url.Values{"key": {key}, "return": {"/notes"}}.Encode(), "", u.Host, "")
	if w.Code != 303 {
		t.Fatalf("unlock: %d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].Domain != "" {
		t.Fatal("unsafe cookie")
	}
	return cookies[0].Name + "=" + cookies[0].Value
}

func TestIntegrationLifecycleAndKeys(t *testing.T) {
	f := setup(t)
	a, key := f.create(t)
	host, _ := url.Parse(a.URL)
	d, _ := f.stage(t, a, "<h1>one</h1>")
	f.uploadAndPublish(t, a, d, "<h1>one</h1>")
	if w := f.request(t, "GET", "/", "", "", host.Host, ""); !strings.Contains(w.Body.String(), "输入分享密钥") {
		t.Fatal("missing key gate")
	}
	for _, method := range []string{"GET", "HEAD"} {
		w := f.request(t, method, "/main.js", nil, "", host.Host, "")
		if method == "GET" { // navigation gate still must never expose the requested script
			if strings.TrimSpace(w.Body.String()) == "x" {
				t.Fatal("asset leaked")
			}
		} else if w.Code != 401 {
			t.Fatal("HEAD bypass")
		}
	}
	browser1 := f.unlock(t, a, key)
	browser2 := f.unlock(t, a, key)
	for _, cookie := range []string{browser1, browser2} {
		w := f.request(t, "GET", "/notes", nil, "", host.Host, cookie)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "one") || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("deep link or cache protection failed")
		}
	}
	if w := f.request(t, "GET", "/missing.js", nil, "", host.Host, browser1); w.Code != 404 {
		t.Fatal("missing asset fell back")
	}
	a.Active = &d
	update, _ := f.stage(t, a, "<h1>two</h1>")
	f.uploadAndPublish(t, a, update, "<h1>two</h1>")
	if w := f.request(t, "GET", "/", nil, "", host.Host, browser1); !strings.Contains(w.Body.String(), "two") {
		t.Fatal("update lost session or address")
	}
	other, otherKey := f.create(t)
	otherHost, _ := url.Parse(other.URL)
	if w := f.request(t, "POST", "/_hosting/unlock", url.Values{"key": {key}}.Encode(), "", otherHost.Host, ""); w.Code != 401 {
		t.Fatal("cross-app key accepted")
	}
	_ = otherKey
	newKey := token()
	response[any](t, f.request(t, "POST", "/apps/"+a.ID+"/key", map[string]string{"key": newKey}, f.token, "control.localhost:8443", ""), 200)
	for _, cookie := range []string{browser1, browser2} {
		w := f.request(t, "GET", "/", nil, "", host.Host, cookie)
		if !strings.Contains(w.Body.String(), "输入分享密钥") {
			t.Fatal("old session survived reset")
		}
	}
	if w := f.request(t, "POST", "/_hosting/unlock", url.Values{"key": {key}}.Encode(), "", host.Host, ""); w.Code != 401 {
		t.Fatal("old key accepted")
	}
	fresh := f.unlock(t, a, newKey)
	response[any](t, f.request(t, "DELETE", "/apps/"+a.ID, nil, f.token, "control.localhost:8443", ""), 200)
	if w := f.request(t, "GET", "/", nil, "", host.Host, fresh); w.Code != 404 {
		t.Fatal("deleted app readable")
	}
	if e := f.server.Cleanup(context.Background()); e != nil {
		t.Fatal(e)
	}
	if len(f.storage.data) != 0 {
		t.Fatal("objects not reclaimed")
	}
	for _, secret := range []string{key, newKey, f.token, browser1} {
		if strings.Contains(f.logs.String(), secret) {
			t.Fatal("secret in logs")
		}
	}
}

func TestIntegrationConflictsQuotasAndOwnership(t *testing.T) {
	f := setup(t)
	a, _ := f.create(t)
	d, _ := f.stage(t, a, "old")
	second, _ := f.stage(t, a, "new")
	response[any](t, f.request(t, "POST", "/apps/"+a.ID+"/deployments/"+d+"/publish", map[string]any{}, f.token, "control.localhost:8443", ""), 409)
	f.uploadAndPublish(t, a, d, "old")
	base := "/apps/" + a.ID + "/deployments/" + second
	response[any](t, f.request(t, "PUT", base+"/files/0", []byte("new"), f.token, "control.localhost:8443", ""), 200)
	response[any](t, f.request(t, "PUT", base+"/files/1", []byte("x"), f.token, "control.localhost:8443", ""), 200)
	response[any](t, f.request(t, "POST", base+"/publish", map[string]any{}, f.token, "control.localhost:8443", ""), 409)
	if w := f.request(t, "DELETE", "/apps/"+a.ID, nil, token(), "control.localhost:8443", ""); w.Code != 401 {
		t.Fatal("invalid credential accepted")
	}
	owner2, credential := id(), token()
	_, e := f.db.Exec(context.Background(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','second')`, owner2)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.db.Exec(context.Background(), `INSERT INTO credentials(id,hash,owner_id,expires_at) VALUES($1,$2,$3,now()+interval '1 day')`, id(), keyed(f.server.Config.Secret, credential), owner2)
	if e != nil {
		t.Fatal(e)
	}
	response[any](t, f.request(t, "DELETE", "/apps/"+a.ID, nil, credential, "control.localhost:8443", ""), 404)
	f.server.Config.Limits.Apps = 1
	response[any](t, f.request(t, "POST", "/apps", map[string]any{"name": "overflow", "key": token(), "requestId": id()}, f.token, "control.localhost:8443", ""), 429)
	if _, e = f.db.Exec(context.Background(), `UPDATE credentials SET revoked=true WHERE owner_id=$1`, f.owner); e != nil {
		t.Fatal(e)
	}
	response[any](t, f.request(t, "GET", "/apps", nil, f.token, "control.localhost:8443", ""), 401)
}

func TestIntegrationDeviceAuthorization(t *testing.T) {
	f := setup(t)
	w := f.request(t, "POST", "/device-authorizations", map[string]any{}, "", "control.localhost:8443", "")
	auth := response[map[string]any](t, w, 201)
	poll := func() *httptest.ResponseRecorder {
		return f.request(t, "POST", "/device-authorizations/poll", map[string]any{"deviceSecret": auth["deviceSecret"]}, "", "control.localhost:8443", "")
	}
	if v := response[map[string]any](t, poll(), 400); v["error"] != "authorization_pending" {
		t.Fatal(v)
	}
	if v := response[map[string]any](t, poll(), 429); v["error"] != "slow_down" {
		t.Fatal(v)
	}
	browser := token()
	_, e := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, browser), f.owner)
	if e != nil {
		t.Fatal(e)
	}
	approve := f.request(t, "POST", "/device", url.Values{"code": {auth["userCode"].(string)}, "decision": {"approved"}}.Encode(), "", "control.localhost:8443", "__Host-owner="+browser)
	if approve.Code != 200 {
		t.Fatal(approve.Body.String())
	}
	_, _ = f.db.Exec(context.Background(), `UPDATE device_authorizations SET last_poll=NULL`)
	issued := response[map[string]any](t, poll(), 200)
	credential := issued["token"].(string)
	response[any](t, f.request(t, "GET", "/apps", nil, credential, "control.localhost:8443", ""), 200)
	_, _ = f.db.Exec(context.Background(), `UPDATE device_authorizations SET last_poll=NULL`)
	if v := response[map[string]any](t, poll(), 400); v["error"] != "already_claimed" {
		t.Fatal(v)
	}
	response[any](t, f.request(t, "DELETE", "/credentials/current", nil, credential, "control.localhost:8443", ""), 200)
	response[any](t, f.request(t, "GET", "/apps", nil, credential, "control.localhost:8443", ""), 401)
}

func TestIntegrationTrafficAndExpiry(t *testing.T) {
	f := setup(t)
	a, key := f.create(t)
	d, _ := f.stage(t, a, "ok")
	f.uploadAndPublish(t, a, d, "ok")
	cookie := f.unlock(t, a, key)
	host, _ := url.Parse(a.URL)
	f.server.Config.Limits.TrafficBytes = 2
	if w := f.request(t, "GET", "/", nil, "", host.Host, cookie); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	response[any](t, f.request(t, "GET", "/", nil, "", host.Host, cookie), 429)
	if w := f.request(t, "HEAD", "/", nil, "", host.Host, cookie); w.Code != 200 {
		t.Fatal("HEAD charged body bytes")
	}
	_, _ = f.db.Exec(context.Background(), `UPDATE usage SET period=$1`, time.Now().UTC().AddDate(0, -1, 0).Format("2006-01"))
	if w := f.request(t, "GET", "/", nil, "", host.Host, cookie); w.Code != 200 {
		t.Fatal("new period did not reset")
	}
	_, _ = f.db.Exec(context.Background(), `UPDATE access_sessions SET expires_at=now()-interval '1 second'`)
	if w := f.request(t, "GET", "/", nil, "", host.Host, cookie); !strings.Contains(w.Body.String(), "输入分享密钥") {
		t.Fatal("expired session accepted")
	}
}
