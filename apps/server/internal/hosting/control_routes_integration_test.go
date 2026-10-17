package hosting

import (
	"context"
	"net/url"
	"testing"
)

func TestIntegrationCanonicalControlPathCompatibility(t *testing.T) {
	f := setup(t)
	const host = "control.localhost:8443"
	requestID := id()
	input := map[string]any{"name": "Canonical routes", "key": token(), "requestId": requestID}
	created := response[App](t, f.request(t, "POST", "/api/control/v1/apps", input, f.token, host, ""), 201)
	replayed := response[App](t, f.request(t, "POST", "/apps", input, f.token, host, ""), 201)
	if created.ID != replayed.ID {
		t.Fatal("legacy replay created another application")
	}
	input["name"] = "different"
	conflict := response[map[string]any](t, f.request(t, "POST", "/apps", input, f.token, host, ""), 409)
	if conflict["error"] != "idempotency_conflict" {
		t.Fatal(conflict)
	}
	for _, path := range []string{"/limits", "/api/control/v1/limits", "/apps", "/api/control/v1/apps"} {
		response[any](t, f.request(t, "GET", path, nil, f.token, host, ""), 200)
	}
	for _, path := range []string{"/apps/" + created.ID + "/data", "/api/control/v1/apps/" + created.ID + "/data"} {
		failure := response[map[string]any](t, f.request(t, "GET", path, nil, f.token, host, ""), 503)
		if failure["error"] != "app_data_not_ready" {
			t.Fatal(failure)
		}
	}
	files := []Entry{{Path: "index.html", Size: 1, SHA256: digest([]byte("x"))}}
	stageInput := map[string]any{"requestId": id(), "manifest": files, "baseVersion": nil, "spa": false}
	oldBase := "/apps/" + created.ID + "/deployments"
	newBase := "/api/control/v1" + oldBase
	first := response[map[string]string](t, f.request(t, "POST", oldBase, stageInput, f.token, host, ""), 201)
	second := response[map[string]string](t, f.request(t, "POST", newBase, stageInput, f.token, host, ""), 201)
	if first["id"] != second["id"] {
		t.Fatal("canonical replay created another deployment")
	}
	stageInput["spa"] = true
	failure := response[map[string]any](t, f.request(t, "POST", newBase, stageInput, f.token, host, ""), 409)
	if failure["error"] != "idempotency_conflict" {
		t.Fatal(failure)
	}
	response[any](t, f.request(t, "DELETE", "/api/control/v1/apps/"+created.ID, nil, f.token, host, ""), 200)
	response[any](t, f.request(t, "GET", "/apps/"+created.ID+"/data", nil, f.token, host, ""), 404)
}

func TestIntegrationCanonicalDeviceAndCredentialPaths(t *testing.T) {
	f := setup(t)
	const host = "control.localhost:8443"
	device := response[map[string]any](t, f.request(t, "POST", "/auth/device", map[string]any{}, "", host, ""), 201)
	pollBody := map[string]any{"deviceSecret": device["deviceSecret"]}
	pending := response[map[string]any](t, f.request(t, "POST", "/auth/device/poll", pollBody, "", host, ""), 400)
	if pending["error"] != "authorization_pending" {
		t.Fatal(pending)
	}
	slow := response[map[string]any](t, f.request(t, "POST", "/device-authorizations/poll", pollBody, "", host, ""), 429)
	if slow["error"] != "slow_down" {
		t.Fatal(slow)
	}
	browser := token()
	_, err := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, browser), f.owner)
	if err != nil {
		t.Fatal(err)
	}
	approval := url.Values{"code": {device["userCode"].(string)}, "decision": {"approved"}}.Encode()
	if got := f.request(t, "POST", "/dashboard/device", approval, "", host, "__Host-owner="+browser); got.Code != 200 {
		t.Fatal(got.Body.String())
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE device_authorizations SET last_poll=NULL`); err != nil {
		t.Fatal(err)
	}
	issued := response[map[string]any](t, f.request(t, "POST", "/auth/device/poll", pollBody, "", host, ""), 200)
	credential := issued["token"].(string)
	response[any](t, f.request(t, "GET", "/api/control/v1/apps", nil, credential, host, ""), 200)
	response[any](t, f.request(t, "DELETE", "/auth/credentials/current", nil, credential, host, ""), 200)
	response[any](t, f.request(t, "GET", "/apps", nil, credential, host, ""), 401)
	if _, err := f.db.Exec(context.Background(), `UPDATE device_authorizations SET last_poll=NULL`); err != nil {
		t.Fatal(err)
	}
	replayed := response[map[string]any](t, f.request(t, "POST", "/device-authorizations/poll", pollBody, "", host, ""), 400)
	if replayed["error"] != "already_claimed" {
		t.Fatal(replayed)
	}

	denied := response[map[string]any](t, f.request(t, "POST", "/device-authorizations", map[string]any{}, "", host, ""), 201)
	if _, err := f.db.Exec(context.Background(), `UPDATE device_authorizations SET status='denied' WHERE user_code=$1`, denied["userCode"]); err != nil {
		t.Fatal(err)
	}
	denial := response[map[string]any](t, f.request(t, "POST", "/auth/device/poll", map[string]any{"deviceSecret": denied["deviceSecret"]}, "", host, ""), 400)
	if denial["error"] != "access_denied" {
		t.Fatal(denial)
	}
	expired := response[map[string]any](t, f.request(t, "POST", "/auth/device", map[string]any{}, "", host, ""), 201)
	if _, err := f.db.Exec(context.Background(), `UPDATE device_authorizations SET expires_at=now()-interval '1 second' WHERE user_code=$1`, expired["userCode"]); err != nil {
		t.Fatal(err)
	}
	expiry := response[map[string]any](t, f.request(t, "POST", "/auth/device/poll", map[string]any{"deviceSecret": expired["deviceSecret"]}, "", host, ""), 400)
	if expiry["error"] != "expired_token" {
		t.Fatal(expiry)
	}
}

func TestIntegrationCanonicalControlAuthenticationIsolation(t *testing.T) {
	f := setup(t)
	const host = "control.localhost:8443"
	app, _ := f.create(t)
	browser := token()
	_, err := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, browser), f.owner)
	if err != nil {
		t.Fatal(err)
	}
	cookie := "__Host-owner=" + browser
	for _, path := range []string{"/apps", "/api/control/v1/apps", "/limits", "/api/control/v1/limits", "/credentials/current", "/auth/credentials/current"} {
		method := "GET"
		if path == "/credentials/current" || path == "/auth/credentials/current" {
			method = "DELETE"
		}
		response[any](t, f.request(t, method, path, nil, "", host, cookie), 401)
	}
	response[any](t, f.request(t, "GET", "/api/console/session", nil, f.token, host, ""), 401)
	response[any](t, f.request(t, "GET", "/api/control/v1/apps", nil, f.token, "unknown.ce11.app", ""), 404)
	appURL, err := url.Parse(app.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.request(t, "POST", "/api/control/v1/apps", map[string]any{"name": "wrong host"}, f.token, appURL.Host, ""); got.Code != 405 {
		t.Fatalf("application host accepted control API: %d", got.Code)
	}
	otherOwner, otherToken := id(), token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','other')`, otherOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO credentials(id,hash,owner_id,expires_at) VALUES($1,$2,$3,now()+interval '1 day')`, id(), keyed(f.server.Config.Secret, otherToken), otherOwner); err != nil {
		t.Fatal(err)
	}
	response[any](t, f.request(t, "GET", "/api/control/v1/apps/"+app.ID+"/data", nil, otherToken, host, ""), 404)
	response[any](t, f.request(t, "DELETE", "/api/control/v1/apps/"+app.ID, nil, otherToken, host, ""), 404)
	if got := f.request(t, "GET", "/api/control/v1/unknown", nil, f.token, host, ""); got.Code != 404 {
		t.Fatalf("unknown control route returned %d", got.Code)
	}
}
