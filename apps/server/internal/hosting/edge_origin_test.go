package hosting

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestEdgeOriginConfiguration(t *testing.T) {
	vars := map[string]string{
		"EDGE_ORIGIN_HOST":   "edge-probe---cellapp-example.us-west1.run.app",
		"EDGE_ORIGIN_SECRET": strings.Repeat("s", 32),
	}
	load := func() error {
		_, err := LoadConfig(func(key string) string { return vars[key] })
		return err
	}
	if err := load(); err != nil {
		t.Fatal(err)
	}
	delete(vars, "EDGE_ORIGIN_SECRET")
	if err := load(); err == nil {
		t.Fatal("origin host without a secret was accepted")
	}
	vars["EDGE_ORIGIN_SECRET"] = "short"
	if err := load(); err == nil {
		t.Fatal("short origin secret was accepted")
	}
}

func TestAuthenticatedOriginalHost(t *testing.T) {
	const origin = "cellapp-example.us-west1.run.app"
	const secret = "a-dedicated-edge-secret-with-32-bytes"
	s := &Server{Config: Config{EdgeOriginHost: origin, EdgeOriginSecret: secret}}
	request := func(host, assertedHost, proof string) string {
		r := httptest.NewRequest("GET", "https://"+host+"/", nil)
		r.Host = host
		r.Header.Set("X-Cellapp-Original-Host", assertedHost)
		r.Header.Set("X-Cellapp-Edge-Secret", proof)
		return s.requestHost(r)
	}
	if got := request(origin, "console.cellapp.ai", secret); got != "console.cellapp.ai" {
		t.Fatalf("authenticated control host = %q", got)
	}
	if got := request(origin, strings.Repeat("a", 32)+".ce11.app", secret); got != strings.Repeat("a", 32)+".ce11.app" {
		t.Fatalf("authenticated visitor host = %q", got)
	}
	for _, proof := range []string{"", "invalid"} {
		if got := request(origin, "console.cellapp.ai", proof); got != origin {
			t.Fatalf("forged host accepted with proof %q", proof)
		}
	}
	if got := request("unknown.example", "console.cellapp.ai", secret); got != "unknown.example" {
		t.Fatalf("assertion from wrong origin accepted: %q", got)
	}
	if got := request("console.cellapp.ai", "unknown.example", ""); got != "console.cellapp.ai" {
		t.Fatalf("direct host changed: %q", got)
	}
}

func TestEdgeAppHandoffAndSameOriginRequests(t *testing.T) {
	f := setup(t)
	app, _ := f.create(t)
	deployment, _ := f.stage(t, app, "<h1>ready</h1>")
	f.uploadAndPublish(t, app, deployment, "<h1>ready</h1>")
	f.server.Config.EdgeOriginHost = "cellapp-example.us-west1.run.app"
	f.server.Config.EdgeOriginSecret = strings.Repeat("s", 32)
	appURL, _ := url.Parse(app.URL)
	request := func(method, path, browserHost, cookie, origin, proof string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "https://"+f.server.Config.EdgeOriginHost+path, nil)
		r.Header.Set("X-Cellapp-Original-Host", browserHost)
		r.Header.Set("X-Cellapp-Edge-Secret", proof)
		if cookie != "" {
			r.Header.Set("Cookie", cookie)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		r.Header.Set("Accept", "text/html")
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		return w
	}
	first := request("GET", "/", appURL.Host, "", "", f.server.Config.EdgeOriginSecret)
	if first.Code != http.StatusSeeOther {
		t.Fatalf("app navigation: %d %s", first.Code, first.Body.String())
	}
	redirect, err := url.Parse(first.Header().Get("Location"))
	if err != nil || redirect.Host != "control.localhost:8443" || redirect.Query().Get("host") != strings.TrimSuffix(appURL.Hostname(), "."+f.server.Config.AppsDomain) {
		t.Fatalf("handoff used origin instead of browser host: %s", first.Header().Get("Location"))
	}
	platform := f.webSessionCookie(t)
	issued := request("GET", redirect.RequestURI(), "control.localhost:8443", "__Host-owner="+platform, "", f.server.Config.EdgeOriginSecret)
	if issued.Code != http.StatusSeeOther || !strings.HasPrefix(issued.Header().Get("Location"), app.URL+"/_cellapp/handoff?") {
		t.Fatalf("handoff issue: %d %s", issued.Code, issued.Body.String())
	}
	target, _ := url.Parse(issued.Header().Get("Location"))
	consumed := request("GET", target.RequestURI(), appURL.Host, "", "", f.server.Config.EdgeOriginSecret)
	if consumed.Code != http.StatusSeeOther || consumed.Header().Get("Location") != "/" {
		t.Fatalf("handoff consume: %d %s", consumed.Code, consumed.Body.String())
	}
	appCookie := consumed.Result().Cookies()[0]
	cookie := appCookie.Name + "=" + appCookie.Value
	page := request("GET", "/", appURL.Host, cookie, "", f.server.Config.EdgeOriginSecret)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "ready") {
		t.Fatalf("app page: %d %s", page.Code, page.Body.String())
	}
	for path, expected := range map[string]int{"/_cellapp/request": http.StatusOK, "/_cellapp/redeem": http.StatusUnauthorized, "/rest/v1/notes": http.StatusServiceUnavailable} {
		allowed := request("POST", path, appURL.Host, cookie, app.URL, f.server.Config.EdgeOriginSecret)
		if allowed.Code != expected {
			t.Fatalf("same-origin request on %s: %d %s", path, allowed.Code, allowed.Body.String())
		}
		denied := request("POST", path, appURL.Host, cookie, "https://other.apps.localhost:8443", f.server.Config.EdgeOriginSecret)
		if denied.Code != http.StatusForbidden {
			t.Fatalf("cross-origin request accepted on %s: %d", path, denied.Code)
		}
	}
	if forged := request("GET", "/", appURL.Host, "", "", "wrong-secret"); forged.Code != http.StatusNotFound {
		t.Fatalf("unauthenticated edge host accepted: %d", forged.Code)
	}
}
