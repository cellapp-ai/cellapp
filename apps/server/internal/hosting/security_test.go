package hosting

import (
	"context"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type interruptedReader struct{}

func (interruptedReader) Read(p []byte) (int, error) {
	if len(p) > 0 {
		p[0] = 'n'
		return 1, io.ErrUnexpectedEOF
	}
	return 0, io.ErrUnexpectedEOF
}
func TestIntegrationInterruptedUploadKeepsCurrentVersion(t *testing.T) {
	f := setup(t)
	a, key := f.create(t)
	first, _ := f.stage(t, a, "old")
	f.uploadAndPublish(t, a, first, "old")
	session := f.unlock(t, a, key)
	host, _ := url.Parse(a.URL)
	a.Active = &first
	next, _ := f.stage(t, a, "new")
	req := httptest.NewRequest("PUT", "https://control.localhost:8443/apps/"+a.ID+"/deployments/"+next+"/files/0", interruptedReader{})
	req.Header.Set("Authorization", "Bearer "+f.token)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	response[any](t, w, 400)
	live := f.request(t, "GET", "/", nil, "", host.Host, session)
	if live.Code != 200 || live.Body.String() != "old" {
		t.Fatal("interrupted upload replaced live version")
	}
	f.uploadAndPublish(t, a, next, "new")
	live = f.request(t, "GET", "/", nil, "", host.Host, session)
	if live.Code != 200 || live.Body.String() != "new" {
		t.Fatal("retry failed")
	}
}

func TestIntegrationDeviceDenialExpiryAndCSRF(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	for _, status := range []string{"denied", "expired"} {
		auth := response[map[string]any](t, f.request(t, "POST", "/device-authorizations", map[string]any{}, "", "control.localhost:8443", ""), 201)
		if status == "denied" {
			_, _ = f.db.Exec(ctx, `UPDATE device_authorizations SET status='denied' WHERE user_code=$1`, auth["userCode"])
		} else {
			_, _ = f.db.Exec(ctx, `UPDATE device_authorizations SET expires_at=now()-interval '1 second' WHERE user_code=$1`, auth["userCode"])
		}
		answer := response[map[string]any](t, f.request(t, "POST", "/device-authorizations/poll", map[string]any{"deviceSecret": auth["deviceSecret"]}, "", "control.localhost:8443", ""), 400)
		expected := "access_denied"
		if status == "expired" {
			expected = "expired_token"
		}
		if answer["error"] != expected {
			t.Fatal(answer)
		}
		response[any](t, f.request(t, "POST", "/device-authorizations/poll", map[string]any{"deviceSecret": auth["userCode"]}, "", "control.localhost:8443", ""), 400)
	}
	browser := token()
	_, e := f.db.Exec(ctx, `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, browser), f.owner)
	if e != nil {
		t.Fatal(e)
	}
	req := httptest.NewRequest("POST", "https://control.localhost:8443/credentials/revoke", strings.NewReader("id=anything"))
	req.Header.Set("Cookie", "__Host-owner="+browser)
	req.Header.Set("Origin", "https://attacker.example")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal("cross-origin mutation accepted")
	}
	_, e = f.db.Exec(ctx, `UPDATE credentials SET expires_at=now()-interval '1 second' WHERE owner_id=$1`, f.owner)
	if e != nil {
		t.Fatal(e)
	}
	response[any](t, f.request(t, "GET", "/apps", nil, f.token, "control.localhost:8443", ""), 401)
}

func TestIntegrationCreationReplayRateLimitAndSuspension(t *testing.T) {
	f := setup(t)
	key := token()
	input := map[string]string{"name": "Replay", "requestId": id(), "key": key}
	a := response[App](t, f.request(t, "POST", "/apps", input, f.token, "control.localhost:8443", ""), 201)
	same := response[App](t, f.request(t, "POST", "/apps", input, f.token, "control.localhost:8443", ""), 201)
	if same.ID != a.ID {
		t.Fatal("duplicate app created")
	}
	input["name"] = "different"
	response[any](t, f.request(t, "POST", "/apps", input, f.token, "control.localhost:8443", ""), 409)
	d, _ := f.stage(t, a, "ok")
	f.uploadAndPublish(t, a, d, "ok")
	session := f.unlock(t, a, key)
	host, _ := url.Parse(a.URL)
	for i := 0; i < 9; i++ {
		response[any](t, f.request(t, "POST", "/_hosting/unlock", url.Values{"key": {"bad"}}.Encode(), "", host.Host, ""), 401)
	}
	response[any](t, f.request(t, "POST", "/_hosting/unlock", url.Values{"key": {"bad"}}.Encode(), "", host.Host, ""), 429)
	if e := f.server.Suspend(context.Background(), a.ID); e != nil {
		t.Fatal(e)
	}
	response[any](t, f.request(t, "GET", "/", nil, "", host.Host, session), 403)
	a.Active = &d
	w := f.request(t, "POST", "/apps/"+a.ID+"/deployments", map[string]any{"manifest": []Entry{{"index.html", 1, digest([]byte("x"))}}, "requestId": id(), "baseVersion": d}, f.token, "control.localhost:8443", "")
	if w.Code != 403 {
		t.Fatal("suspended app accepted update")
	}
	if strings.Contains(f.logs.String(), key) || strings.Contains(f.logs.String(), f.token) {
		t.Fatal("secret logged")
	}
}
