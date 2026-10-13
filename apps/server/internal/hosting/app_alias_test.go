package hosting

import (
	"context"
	"net/http"
	"testing"
)

func TestApplicationNamespaceAliasAndTombstone(t *testing.T) {
	f := setup(t)
	host := "control.localhost:8443"
	key := token()
	requestID := id()
	firstRequest := map[string]any{"name": "Notes", "namespace": "my-space", "slug": "notes", "key": key, "requestId": requestID}
	first := response[App](t, f.request(t, "POST", "/apps", firstRequest, f.token, host, ""), 201)
	if first.Namespace != "my-space" || first.Slug != "notes" || first.AliasURL != f.server.Config.ControlOrigin+"/a/my-space/notes" {
		t.Fatalf("wrong alias: %+v", first)
	}
	repeated := response[App](t, f.request(t, "POST", "/apps", firstRequest, f.token, host, ""), 201)
	if repeated.ID != first.ID || repeated.AliasURL != first.AliasURL {
		t.Fatal("idempotent create changed the alias")
	}
	redirect := f.request(t, "GET", "/a/my-space/notes", nil, "", host, "")
	if redirect.Code != http.StatusTemporaryRedirect || redirect.Header().Get("Location") != first.URL {
		t.Fatalf("alias did not resolve to the ID host: %d %q", redirect.Code, redirect.Header().Get("Location"))
	}
	conflict := map[string]any{"name": "Other Notes", "slug": "notes", "key": token(), "requestId": id()}
	if response[map[string]any](t, f.request(t, "POST", "/apps", conflict, f.token, host, ""), 409)["error"] != "slug_taken" {
		t.Fatal("duplicate slug accepted")
	}
	conflict["name"], conflict["slug"], conflict["requestId"] = "Notes", "", id()
	second := response[App](t, f.request(t, "POST", "/apps", conflict, f.token, host, ""), 201)
	if second.Namespace != first.Namespace || second.Slug != "notes-2" {
		t.Fatalf("automatic collision resolution: %+v", second)
	}
	conflict["namespace"], conflict["requestId"] = "another-space", id()
	if response[map[string]any](t, f.request(t, "POST", "/apps", conflict, f.token, host, ""), 409)["error"] != "namespace_mismatch" {
		t.Fatal("owner created a second personal namespace")
	}
	otherOwner, otherCredential := id(), token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$2)`, otherOwner, otherOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO credentials(id,hash,owner_id,expires_at) VALUES($1,$2,$3,now()+interval '1 day')`, id(), keyed(f.server.Config.Secret, otherCredential), otherOwner); err != nil {
		t.Fatal(err)
	}
	otherRequest := map[string]any{"name": "Notes", "namespace": "my-space", "slug": "notes", "key": token(), "requestId": id()}
	if response[map[string]any](t, f.request(t, "POST", "/apps", otherRequest, otherCredential, host, ""), 409)["error"] != "namespace_taken" {
		t.Fatal("second owner claimed an occupied namespace")
	}
	otherRequest["namespace"], otherRequest["requestId"] = "other-space", id()
	other := response[App](t, f.request(t, "POST", "/apps", otherRequest, otherCredential, host, ""), 201)
	if other.Slug != first.Slug || other.Namespace == first.Namespace || other.ID == first.ID {
		t.Fatal("cross-owner namespace collision")
	}
	otherRequest["namespace"], otherRequest["requestId"] = "my-space", id()
	if response[map[string]any](t, f.request(t, "POST", "/apps", otherRequest, otherCredential, host, ""), 409)["error"] != "namespace_mismatch" {
		t.Fatal("existing namespace was changed")
	}
	shortOwner, shortCredential := id(), token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$2)`, shortOwner, shortOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO credentials(id,hash,owner_id,expires_at) VALUES($1,$2,$3,now()+interval '1 day')`, id(), keyed(f.server.Config.Secret, shortCredential), shortOwner); err != nil {
		t.Fatal(err)
	}
	short := response[App](t, f.request(t, "POST", "/apps", map[string]any{"name": "One", "namespace": "x", "slug": "y", "key": token(), "requestId": id()}, shortCredential, host, ""), 201)
	if short.Namespace != "x" || short.Slug != "y" {
		t.Fatalf("single-letter alias failed: %+v", short)
	}
	invalid := map[string]any{"name": "Invalid", "slug": "Bad_Alias", "key": token(), "requestId": id()}
	if response[map[string]any](t, f.request(t, "POST", "/apps", invalid, f.token, host, ""), 400)["error"] != "invalid_slug" {
		t.Fatal("invalid alias accepted")
	}
	response[any](t, f.request(t, "DELETE", "/apps/"+first.ID, nil, f.token, host, ""), 200)
	if f.request(t, "GET", "/a/my-space/notes", nil, "", host, "").Code != 404 {
		t.Fatal("deleted alias still resolves")
	}
	conflict["namespace"], conflict["slug"], conflict["requestId"] = "", "notes", id()
	if response[map[string]any](t, f.request(t, "POST", "/apps", conflict, f.token, host, ""), 409)["error"] != "slug_taken" {
		t.Fatal("deleted alias was rebound")
	}
}
