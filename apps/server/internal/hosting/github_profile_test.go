package hosting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func githubSession(t *testing.T, f *fixture, subject, login string) (string, string) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://control.localhost:8443/auth/callback", nil)
	if err := f.server.finishLogin(w, r, IdentityClaims{Issuer: githubIssuer, Subject: subject, GitHubLogin: login}, "/dashboard"); err != nil {
		t.Fatal(err)
	}
	var cookie string
	for _, item := range w.Result().Cookies() {
		if item.Name == "__Host-owner" {
			cookie = item.Value
		}
	}
	if cookie == "" {
		t.Fatal("missing owner cookie")
	}
	var owner string
	if err := f.db.QueryRow(context.Background(), `SELECT id FROM owners WHERE issuer=$1 AND subject=$2`, githubIssuer, subject).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	return owner, cookie
}

func TestGitHubProfileAndNamespaceDefaults(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	owner, session := githubSession(t, f, "123456", "Alice-Dev")
	identity := response[map[string]string](t, f.webRequest(t, "GET", "/session", nil, session, ""), http.StatusOK)
	if identity["ownerId"] != owner || identity["username"] != "Alice-Dev" || identity["avatarUrl"] != "https://avatars.githubusercontent.com/u/123456?s=96" {
		t.Fatal(identity)
	}
	app, err := f.server.createOwnerApp(ctx, owner, createAppInput{Name: "Notes", Key: token(), RequestID: id()})
	if err != nil || app.Namespace != "alice-dev" {
		t.Fatal("GitHub login was not used for first implicit namespace", app, err)
	}
	profile := response[webProfileData](t, f.webRequest(t, "GET", "/profile", nil, session, ""), http.StatusOK)
	if profile.DisplayName != "" || profile.GitHubUsername != "Alice-Dev" || profile.UserSlug != "alice-dev" {
		t.Fatal(profile)
	}
	updated := response[webProfileData](t, f.webRequest(t, "PUT", "/profile", map[string]string{"displayName": "Chosen name"}, session, f.server.Config.ControlOrigin), http.StatusOK)
	if updated.DisplayName != "Chosen name" {
		t.Fatal(updated)
	}
	_, secondSession := githubSession(t, f, "123456", "Renamed-Dev")
	identity = response[map[string]string](t, f.webRequest(t, "GET", "/session", nil, secondSession, ""), http.StatusOK)
	if identity["ownerId"] != owner || identity["username"] != "Chosen name" {
		t.Fatal("identity or chosen display name changed", identity)
	}
	profile = response[webProfileData](t, f.webRequest(t, "GET", "/profile", nil, secondSession, ""), http.StatusOK)
	if profile.GitHubUsername != "Renamed-Dev" || profile.UserSlug != "alice-dev" {
		t.Fatal("GitHub rename changed an existing address", profile)
	}
	response[webProfileData](t, f.webRequest(t, "PUT", "/profile", map[string]string{"displayName": ""}, secondSession, f.server.Config.ControlOrigin), http.StatusOK)
	identity = response[map[string]string](t, f.webRequest(t, "GET", "/session", nil, secondSession, ""), http.StatusOK)
	if identity["username"] != "Renamed-Dev" {
		t.Fatal("clearing chosen name did not restore GitHub default", identity)
	}
	response[any](t, f.webRequest(t, "GET", "/session", nil, "", ""), http.StatusUnauthorized)
}

func TestGitHubNamespaceConflictAndExplicitChoice(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if _, err := f.db.Exec(ctx, `INSERT INTO app_namespaces(id,owner_id,handle) VALUES($1,$2,'alice')`, id(), f.owner); err != nil {
		t.Fatal(err)
	}
	owner, _ := githubSession(t, f, "123456", "Alice")
	app, err := f.server.createOwnerApp(ctx, owner, createAppInput{Name: "Notes", Key: token(), RequestID: id()})
	if err != nil || app.Namespace != "u-123456" {
		t.Fatal("occupied GitHub login did not use stable fallback", app, err)
	}
	other, _ := githubSession(t, f, "777", "Fancy")
	app, err = f.server.createOwnerApp(ctx, other, createAppInput{Name: "Site", Namespace: "custom", Key: token(), RequestID: id()})
	if err != nil || app.Namespace != "custom" {
		t.Fatal("explicit first namespace changed", app, err)
	}
	short, _ := githubSession(t, f, "888", "2")
	app, err = f.server.createOwnerApp(ctx, short, createAppInput{Name: "Site", Key: token(), RequestID: id()})
	if err != nil || app.Namespace != "u-2" {
		t.Fatal("invalid GitHub login did not get a valid namespace", app, err)
	}
}
