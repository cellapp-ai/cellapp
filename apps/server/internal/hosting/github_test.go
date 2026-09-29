package hosting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
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
	if w := f.request(t, "GET", "/api/console/session", nil, "", "control.localhost:8443", ownerCookie); w.Code != 200 {
		t.Fatal("owner session failed")
	}
	if w := f.request(t, "POST", "/auth/logout", "", "", "control.localhost:8443", ownerCookie); w.Code != 303 {
		t.Fatal("logout failed")
	}
	if w := f.request(t, "GET", "/api/console/session", nil, "", "control.localhost:8443", ownerCookie); w.Code != 401 {
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
	if e := f.db.QueryRow(context.Background(), `SELECT issuer,subject FROM owners WHERE issuer='cellapp:development'`).Scan(&issuer, &subject); e != nil || subject != "local-owner" {
		t.Fatal("fixed development owner was not created", issuer, subject, e)
	}
	callback := f.request(t, "GET", "/auth/callback?code=ignored", nil, "", "control.localhost:8443", "")
	if callback.Code != http.StatusNotFound {
		t.Fatal("OAuth callback enabled in development authentication mode")
	}
}

func TestDevelopmentLoginMigratesLegacyOwnerConcurrently(t *testing.T) {
	f := setup(t)
	f.server.Config.AuthMode = "dev"
	f.server.Identity = nil
	ctx := context.Background()
	if _, e := f.db.Exec(ctx, `UPDATE owners SET issuer='ohmyapp:development',subject='local-owner' WHERE id=$1`, f.owner); e != nil {
		t.Fatal(e)
	}
	app, _ := f.create(t)
	existingSession := token()
	if _, e := f.db.Exec(ctx, `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, existingSession), f.owner); e != nil {
		t.Fatal(e)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			login := f.request(t, "GET", "/auth/login", nil, "", "control.localhost:8443", "")
			if login.Code != http.StatusSeeOther {
				t.Errorf("concurrent login failed: %d %s", login.Code, login.Body.String())
			}
			for _, cookie := range login.Result().Cookies() {
				if cookie.Name == "__Host-owner" {
					var owner string
					if e := f.db.QueryRow(ctx, `SELECT owner_id FROM browser_sessions WHERE hash=$1`, keyed(f.server.Config.Secret, cookie.Value)).Scan(&owner); e != nil || owner != f.owner {
						t.Errorf("login changed owner: %s %v", owner, e)
					}
				}
			}
		}()
	}
	workers.Wait()
	var owner string
	var count int
	if e := f.db.QueryRow(ctx, `SELECT id FROM owners WHERE issuer='cellapp:development' AND subject='local-owner'`).Scan(&owner); e != nil || owner != f.owner {
		t.Fatal("legacy owner ID was not preserved", owner, e)
	}
	if e := f.db.QueryRow(ctx, `SELECT count(*) FROM owners WHERE issuer IN ('ohmyapp:development','cellapp:development')`).Scan(&count); e != nil || count != 1 {
		t.Fatal("duplicate development owners", count, e)
	}
	apps := response[[]App](t, f.request(t, "GET", "/apps", nil, f.token, "control.localhost:8443", ""), 200)
	if len(apps) != 1 || apps[0].ID != app.ID {
		t.Fatal("existing credential or app association lost", apps)
	}
	if w := f.request(t, "GET", "/api/console/session", nil, "", "control.localhost:8443", "__Host-owner="+existingSession); w.Code != http.StatusOK {
		t.Fatal("existing browser session lost", w.Body.String())
	}
}

func TestDevelopmentLoginRejectsConflictingOwners(t *testing.T) {
	f := setup(t)
	f.server.Config.AuthMode = "dev"
	f.server.Identity = nil
	ctx := context.Background()
	if _, e := f.db.Exec(ctx, `UPDATE owners SET issuer='ohmyapp:development',subject='local-owner' WHERE id=$1`, f.owner); e != nil {
		t.Fatal(e)
	}
	current := id()
	if _, e := f.db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'cellapp:development','local-owner')`, current); e != nil {
		t.Fatal(e)
	}
	w := f.request(t, "GET", "/auth/login", nil, "", "control.localhost:8443", "")
	result := response[map[string]any](t, w, http.StatusConflict)
	if result["error"] != "development_identity_conflict" {
		t.Fatal("missing conflict explanation", result)
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "__Host-owner" {
			t.Fatal("conflicting login granted a session")
		}
	}
	var owner string
	if e := f.db.QueryRow(ctx, `SELECT id FROM owners WHERE issuer='ohmyapp:development' AND subject='local-owner'`).Scan(&owner); e != nil || owner != f.owner {
		t.Fatal("legacy identity changed on conflict", owner, e)
	}
	if e := f.db.QueryRow(ctx, `SELECT id FROM owners WHERE issuer='cellapp:development' AND subject='local-owner'`).Scan(&owner); e != nil || owner != current {
		t.Fatal("current identity changed on conflict", owner, e)
	}
}
