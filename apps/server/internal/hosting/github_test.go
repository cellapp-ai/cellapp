package hosting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestGitHubOAuthPKCEAndUserIdentity(t *testing.T) {
	var userID int64 = 123456
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("code") != "valid" || r.Form.Get("code_verifier") != "verifier" || r.Form.Get("client_id") != "client" || r.Form.Get("client_secret") != "secret" {
			http.Error(w, "invalid exchange", http.StatusBadRequest)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"access_token": "access", "token_type": "bearer"})
	})
	mux.HandleFunc("GET /user", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access" || r.Header.Get("X-GitHub-Api-Version") == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]int64{"id": userID})
	})
	githubServer := httptest.NewServer(mux)
	defer githubServer.Close()

	c, _ := LoadConfig(func(string) string { return "" })
	c.GitHubClientID = "client"
	c.GitHubClientSecret = "secret"
	identity := NewIdentity(c)
	identity.config.Endpoint = oauth2.Endpoint{AuthURL: githubServer.URL + "/login/oauth/authorize", TokenURL: githubServer.URL + "/login/oauth/access_token"}
	identity.userURL = githubServer.URL + "/user"
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, githubServer.Client())

	auth, e := url.Parse(identity.Start("state", "challenge"))
	if e != nil {
		t.Fatal(e)
	}
	if auth.Query().Get("state") != "state" || auth.Query().Get("code_challenge") != "challenge" || auth.Query().Get("code_challenge_method") != "S256" {
		t.Fatal("missing OAuth state or PKCE binding")
	}
	issuer, subject, e := identity.Exchange(ctx, "valid", "verifier")
	if e != nil || issuer != githubIssuer || subject != "123456" {
		t.Fatal(issuer, subject, e)
	}
	userID = 0
	if _, _, e = identity.Exchange(ctx, "valid", "verifier"); e == nil {
		t.Fatal("invalid GitHub user ID accepted")
	}
}

func TestIntegrationLoginStateAndLogout(t *testing.T) {
	f := setup(t)
	login := f.request(t, "GET", "/auth/login?return=/device", nil, "", "control.localhost:8443", "")
	if login.Code != 302 {
		t.Fatal(login.Body.String())
	}
	auth, _ := url.Parse(login.Header().Get("Location"))
	cookies := login.Result().Cookies()
	loginCookie := cookies[0].Name + "=" + cookies[0].Value
	bad := f.request(t, "GET", "/auth/callback?code=valid&state=wrong", nil, "", "control.localhost:8443", loginCookie)
	if bad.Code != 401 {
		t.Fatal("wrong state accepted")
	}
	login = f.request(t, "GET", "/auth/login?return=/device", nil, "", "control.localhost:8443", "")
	auth, _ = url.Parse(login.Header().Get("Location"))
	cookies = login.Result().Cookies()
	loginCookie = cookies[0].Name + "=" + cookies[0].Value
	callback := f.request(t, "GET", "/auth/callback?code=valid&state="+auth.Query().Get("state"), nil, "", "control.localhost:8443", loginCookie)
	if callback.Code != 303 || callback.Header().Get("Location") != "/device" {
		t.Fatal("callback failed", callback.Body.String())
	}
	ownerCookie := ""
	for _, c := range callback.Result().Cookies() {
		if c.Name == "__Host-owner" {
			ownerCookie = c.Name + "=" + c.Value
		}
	}
	if ownerCookie == "" {
		t.Fatal("missing owner session")
	}
	if w := f.request(t, "GET", "/credentials", nil, "", "control.localhost:8443", ownerCookie); w.Code != 200 {
		t.Fatal("owner session failed")
	}
	if w := f.request(t, "POST", "/auth/logout", "", "", "control.localhost:8443", ownerCookie); w.Code != 303 {
		t.Fatal("logout failed")
	}
	if w := f.request(t, "GET", "/credentials", nil, "", "control.localhost:8443", ownerCookie); w.Code != 302 {
		t.Fatal("logout did not revoke session")
	}
}

func TestDevelopmentLoginUsesFixedLocalOwner(t *testing.T) {
	f := setup(t)
	f.server.Config.AuthMode = "dev"
	f.server.Identity = nil
	login := f.request(t, "GET", "/auth/login?return=/credentials", nil, "", "control.localhost:8443", "")
	if login.Code != http.StatusSeeOther || login.Header().Get("Location") != "/credentials" {
		t.Fatal("development login failed", login.Body.String())
	}
	var issuer, subject string
	if e := f.db.QueryRow(context.Background(), `SELECT issuer,subject FROM owners WHERE issuer='ohmyapp:development'`).Scan(&issuer, &subject); e != nil || subject != "local-owner" {
		t.Fatal("fixed development owner was not created", issuer, subject, e)
	}
	callback := f.request(t, "GET", "/auth/callback?code=ignored", nil, "", "control.localhost:8443", "")
	if callback.Code != http.StatusNotFound {
		t.Fatal("OAuth callback enabled in development authentication mode")
	}
}
