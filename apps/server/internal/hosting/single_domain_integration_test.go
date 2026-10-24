package hosting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOwnerCookieHasNoDomain(t *testing.T) {
	w := httptest.NewRecorder()
	cookie(w, "__Host-owner", "test", 60)
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Domain != "" || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].Path != "/" {
		t.Fatalf("owner cookie is not host-only: %+v", cookies)
	}
}

func TestIntegrationSingleDomainHostIsolation(t *testing.T) {
	f := setup(t)
	f.server.Config.ControlOrigin = "https://example.com"
	f.server.Config.AppsDomain = "example.com"
	f.server.Config.AppPort = ""
	f.handler = f.server.Handler()
	const control, appHost = "example.com", "visitor.example.com"
	browser := token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, browser), f.owner); err != nil {
		t.Fatal(err)
	}
	ownerCookie := "__Host-owner=" + browser
	if got := f.request(t, "GET", "/api/console/session", nil, "", control, ownerCookie); got.Code != http.StatusOK {
		t.Fatalf("control session: %d %s", got.Code, got.Body.String())
	}
	if got := f.request(t, "GET", "/api/console/session", nil, "", appHost, ownerCookie); got.Code != http.StatusNotFound {
		t.Fatalf("application host served control session: %d", got.Code)
	}
	if got := f.request(t, "GET", "/api/console/session", nil, "", "unknown.example.net", ownerCookie); got.Code != http.StatusNotFound {
		t.Fatalf("unknown host served control session: %d", got.Code)
	}
	for _, origin := range []string{"", "https://visitor.example.com"} {
		req := httptest.NewRequest(http.MethodPut, "https://example.com/api/console/profile", strings.NewReader(`{"displayName":"Injected"}`))
		req.Header.Set("Cookie", ownerCookie)
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("origin %q changed owner profile: %d %s", origin, w.Code, w.Body.String())
		}
	}
	var displayName string
	if err := f.db.QueryRow(context.Background(), `SELECT display_name FROM owners WHERE id=$1`, f.owner).Scan(&displayName); err != nil || displayName != "" {
		t.Fatalf("rejected write changed profile: %q %v", displayName, err)
	}
	if got := f.request(t, "GET", "/api/control/v1/apps", nil, "", control, ownerCookie); got.Code != http.StatusUnauthorized {
		t.Fatalf("owner cookie authenticated CLI API: %d", got.Code)
	}
	if got := f.request(t, "GET", "/api/control/v1/apps", nil, f.token, control, ""); got.Code != http.StatusOK {
		t.Fatalf("Bearer CLI request failed: %d %s", got.Code, got.Body.String())
	}
}
