package hosting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAppAndUserSlugHosts(t *testing.T) {
	f := setup(t)
	control := "control.localhost:8443"
	session := f.webSessionCookie(t)
	created := response[App](t, f.request(t, "POST", "/apps", map[string]any{
		"name": "Notes", "namespace": "alice", "slug": "notes", "key": token(), "requestId": id(),
	}, f.token, control, ""), 201)
	if created.URL != "https://notes-alice.apps.localhost:8443" {
		t.Fatalf("unexpected composite address %q", created.URL)
	}
	for _, reserved := range []string{"accounts", "cloud", "login", "app"} {
		if got := response[map[string]any](t, f.webRequest(t, "PUT", "/profile/slug", map[string]string{"userSlug": reserved}, session, f.server.Config.ControlOrigin), 400); got["error"] != "invalid_user_slug" {
			t.Fatalf("reserved user slug %q accepted: %v", reserved, got)
		}
		if got := response[map[string]any](t, f.webRequest(t, "PUT", "/apps/"+created.ID+"/slug", map[string]string{"slug": reserved}, session, f.server.Config.ControlOrigin), 400); got["error"] != "invalid_slug" {
			t.Fatalf("reserved application slug %q accepted: %v", reserved, got)
		}
	}
	response[any](t, f.webRequest(t, "PUT", "/profile/slug", map[string]string{"userSlug": "ab"}, session, f.server.Config.ControlOrigin), 400)
	response[any](t, f.webRequest(t, "PUT", "/profile/slug", map[string]string{"userSlug": "alice-"}, session, f.server.Config.ControlOrigin), 400)
	response[any](t, f.webRequest(t, "PUT", "/apps/"+created.ID+"/slug", map[string]string{"slug": "notes-"}, session, f.server.Config.ControlOrigin), 400)
	response[any](t, f.webRequest(t, "PUT", "/profile/slug", map[string]string{"userSlug": "alice2"}, session, "https://outside.example"), 403)
	updated := response[webProfileData](t, f.webRequest(t, "PUT", "/profile/slug", map[string]string{"userSlug": "alice2"}, session, f.server.Config.ControlOrigin), 200)
	if updated.UserSlug != "alice2" {
		t.Fatal(updated)
	}
	app := response[App](t, f.webRequest(t, "PUT", "/apps/"+created.ID+"/slug", map[string]string{"slug": "journal"}, session, f.server.Config.ControlOrigin), 200)
	if app.URL != "https://journal-alice2.apps.localhost:8443" {
		t.Fatal(app.URL)
	}
	response[App](t, f.webRequest(t, "PUT", "/apps/"+created.ID+"/slug", map[string]string{"slug": "notes"}, session, f.server.Config.ControlOrigin), 200)
	response[App](t, f.webRequest(t, "PUT", "/apps/"+created.ID+"/slug", map[string]string{"slug": "journal"}, session, f.server.Config.ControlOrigin), 200)
	response[webProfileData](t, f.webRequest(t, "PUT", "/profile/slug", map[string]string{"userSlug": "alice"}, session, f.server.Config.ControlOrigin), 200)
	response[webProfileData](t, f.webRequest(t, "PUT", "/profile/slug", map[string]string{"userSlug": "alice2"}, session, f.server.Config.ControlOrigin), 200)
	otherApp := response[App](t, f.request(t, "POST", "/apps", map[string]any{
		"name": "Other", "slug": "my-app", "key": token(), "requestId": id(),
	}, f.token, control, ""), 201)
	if otherApp.URL != "https://my-app-alice2.apps.localhost:8443" {
		t.Fatal(otherApp.URL)
	}
	otherOwner, otherCredential := id(), token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$2)`, otherOwner, otherOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO credentials(id,hash,owner_id,expires_at) VALUES($1,$2,$3,now()+interval '1 day')`, id(), keyed(f.server.Config.Secret, otherCredential), otherOwner); err != nil {
		t.Fatal(err)
	}
	if result := response[map[string]any](t, f.request(t, "POST", "/apps", map[string]any{
		"name": "Collision", "namespace": "app-alice2", "slug": "my", "key": token(), "requestId": id(),
	}, otherCredential, control, ""), 409); result["error"] != "slug_taken" {
		t.Fatal(result)
	}
	for _, host := range []string{"notes-alice.apps.localhost:8443", "notes-alice2.apps.localhost:8443", "journal-alice2.apps.localhost:8443", created.ID + ".apps.localhost:8443"} {
		w := f.request(t, "GET", "/", nil, "", host, "")
		if w.Code == http.StatusNotFound {
			t.Fatalf("assigned host stopped resolving: %s: %s", host, w.Body.String())
		}
	}
	if w := f.request(t, "GET", "/a/alice/notes", nil, "", control, ""); w.Code != http.StatusTemporaryRedirect || w.Header().Get("Location") != app.URL {
		t.Fatalf("old alias did not follow current address: %d %s", w.Code, w.Header().Get("Location"))
	}
	if w := f.request(t, "GET", "/a/alice2/journal", nil, "", control, ""); w.Code != http.StatusTemporaryRedirect || w.Header().Get("Location") != app.URL {
		t.Fatalf("new alias did not resolve: %d %s", w.Code, w.Header().Get("Location"))
	}
	platform := f.webSessionCookie(t)
	appCookie := signedInApp(t, f, app, platform)
	if w := f.request(t, "POST", "/_cellapp/request", "", "", "journal-alice2.apps.localhost:8443", appCookie); w.Code != 200 {
		t.Fatalf("canonical host rejected same-origin form: %d %s", w.Code, w.Body.String())
	}
	crossOrigin := httptest.NewRequest("POST", "https://journal-alice2.apps.localhost:8443/_cellapp/request", strings.NewReader(""))
	crossOrigin.Header.Set("Cookie", appCookie)
	crossOrigin.Header.Set("Origin", "https://notes-alice.apps.localhost:8443")
	crossOrigin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	denied := httptest.NewRecorder()
	f.handler.ServeHTTP(denied, crossOrigin)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("other app host origin accepted: %d", denied.Code)
	}
	legacy := f.request(t, "GET", "/auth/app?app="+created.ID+"&host=notes-alice", nil, "", control, "__Host-owner="+platform)
	if legacy.Code != http.StatusSeeOther {
		t.Fatal(legacy.Code)
	}
	target, err := url.Parse(legacy.Header().Get("Location"))
	if err != nil || target.Host != "notes-alice.apps.localhost:8443" {
		t.Fatalf("handoff host changed: %v %v", target, err)
	}
	response[any](t, f.request(t, "GET", "/auth/app?app="+created.ID+"&host=attacker", nil, "", control, "__Host-owner="+platform), 404)
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET deleted=true WHERE id=$1`, created.ID); err != nil {
		t.Fatal(err)
	}
	response[any](t, f.request(t, "GET", "/", nil, "", "notes-alice.apps.localhost:8443", ""), 404)
}
