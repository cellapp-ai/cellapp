package hosting

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func testJWT(role string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"role":"` + role + `"}`))
	return header + "." + payload + ".sig"
}

func TestValidatePublicData(t *testing.T) {
	ok, e := validatePublicData("supabase", "https://abcd.supabase.co", testJWT("anon"))
	if e != nil || ok.URL != "https://abcd.supabase.co" || ok.Dataset != "shared" || ok.Provider != "supabase" {
		t.Fatal(ok, e)
	}
	if _, e := validatePublicData("supabase", "https://abcd.supabase.co/", testJWT("anon")); e != nil {
		t.Fatal(e)
	}
	if _, e := validatePublicData("postgres", "https://abcd.supabase.co", testJWT("anon")); e == nil {
		t.Fatal("non-supabase provider accepted")
	}
	for _, raw := range []string{"http://abcd.supabase.co", "https://user:pass@abcd.supabase.co", "https://abcd.supabase.co/path", "https://abcd.supabase.co?x=1", "not-a-url"} {
		if _, e := validatePublicData("supabase", raw, testJWT("anon")); e == nil {
			t.Fatalf("accepted url %q", raw)
		}
	}
	if _, e := validatePublicData("supabase", "https://abcd.supabase.co", testJWT("service_role")); e == nil {
		t.Fatal("service role jwt accepted")
	}
	if _, e := validatePublicData("supabase", "https://abcd.supabase.co", "sb_secret_example"); e == nil {
		t.Fatal("secret key accepted")
	}
	if _, e := validatePublicData("supabase", "https://abcd.supabase.co", "sb_publishable_example"); e != nil {
		t.Fatal(e)
	}
}

func TestOwnerAndVisitorDataAccess(t *testing.T) {
	f := setup(t)
	a, key := f.create(t)
	host, _ := url.Parse(a.URL)
	anon := testJWT("anon")
	body := map[string]string{"provider": "supabase", "url": "https://demo.supabase.co", "anonKey": anon}
	response[any](t, f.request(t, "GET", "/apps/"+a.ID+"/data", nil, f.token, "control.localhost:8443", ""), 404)
	got := response[AppData](t, f.request(t, "PUT", "/apps/"+a.ID+"/data", body, f.token, "control.localhost:8443", ""), 200)
	if got.URL != "https://demo.supabase.co" || got.AnonKey != anon || got.Dataset != "shared" {
		t.Fatal(got)
	}
	got = response[AppData](t, f.request(t, "GET", "/apps/"+a.ID+"/data", nil, f.token, "control.localhost:8443", ""), 200)
	if got.AnonKey != anon {
		t.Fatal(got)
	}
	response[any](t, f.request(t, "PUT", "/apps/"+a.ID+"/data", map[string]string{"provider": "supabase", "url": "https://demo.supabase.co", "anonKey": testJWT("service_role")}, f.token, "control.localhost:8443", ""), 400)
	other, _ := f.create(t)
	response[any](t, f.request(t, "GET", "/apps/"+other.ID+"/data", nil, f.token, "control.localhost:8443", ""), 404)
	if w := f.request(t, "GET", "/_hosting/data", nil, "", host.Host, ""); !strings.Contains(w.Body.String(), "输入分享密钥") {
		t.Fatal("unauthenticated data request leaked config")
	}
	session := f.unlock(t, a, key)
	visitor := response[AppData](t, f.request(t, "GET", "/_hosting/data", nil, "", host.Host, session), 200)
	if visitor.AnonKey != anon || visitor.URL != "https://demo.supabase.co" {
		t.Fatal(visitor)
	}
	if w := f.request(t, "HEAD", "/_hosting/data", nil, "", host.Host, session); w.Code != 200 || w.Body.Len() != 0 {
		t.Fatal("HEAD wrote a body", w.Code, w.Body.String())
	}
	otherHost, _ := url.Parse(other.URL)
	if w := f.request(t, "GET", "/_hosting/data", nil, "", otherHost.Host, session); w.Code != 401 && !strings.Contains(w.Body.String(), "输入分享密钥") {
		t.Fatal("cross-app session read data", w.Code, w.Body.String())
	}
	d, _ := f.stage(t, a, "<h1>app</h1>")
	f.uploadAndPublish(t, a, d, "<h1>app</h1>")
	if w := f.request(t, "GET", "/", nil, "", host.Host, session); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	response[any](t, f.request(t, "DELETE", "/apps/"+a.ID+"/data", nil, f.token, "control.localhost:8443", ""), 200)
	response[any](t, f.request(t, "GET", "/_hosting/data", nil, "", host.Host, session), 404)
	if strings.Contains(f.logs.String(), anon) {
		t.Fatal("anon key leaked to log")
	}
}

func TestWebAppDataBinding(t *testing.T) {
	f := setup(t)
	session := f.webSessionCookie(t)
	origin := f.server.Config.ControlOrigin
	a, _ := f.create(t)
	anon := testJWT("anon")
	info := response[map[string]any](t, f.webRequest(t, "GET", "/apps/"+a.ID, nil, session, ""), 200)
	if _, ok := info["data"]; ok {
		t.Fatal(info)
	}
	for _, badOrigin := range []string{"", "https://outside.example"} {
		response[any](t, f.webRequest(t, "PUT", "/apps/"+a.ID+"/data", map[string]string{"provider": "supabase", "url": "https://demo.supabase.co", "anonKey": anon}, session, badOrigin), 403)
	}
	got := response[AppData](t, f.webRequest(t, "PUT", "/apps/"+a.ID+"/data", map[string]string{"provider": "supabase", "url": "https://demo.supabase.co", "anonKey": anon}, session, origin), 200)
	if got.Provider != "supabase" {
		t.Fatal(got)
	}
	info = response[map[string]any](t, f.webRequest(t, "GET", "/apps/"+a.ID, nil, session, ""), 200)
	data, _ := info["data"].(map[string]any)
	if data["url"] != "https://demo.supabase.co" {
		t.Fatal(info)
	}
	response[any](t, f.webRequest(t, "DELETE", "/apps/"+a.ID+"/data", nil, session, origin), 200)
	info = response[map[string]any](t, f.webRequest(t, "GET", "/apps/"+a.ID, nil, session, ""), 200)
	if _, ok := info["data"]; ok {
		t.Fatal(info)
	}
}
