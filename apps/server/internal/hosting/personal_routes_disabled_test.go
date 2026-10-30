package hosting

import (
	"context"
	"strings"
	"testing"
)

func TestPersonalDeviceAndBootRoutesRemainClosed(t *testing.T) {
	f := setup(t)
	session := f.webSessionCookie(t)
	oldCredential := token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO personal_devices(id,owner_id,name,credential_hash) VALUES($1,$2,'Old device',$3)`, id(), f.owner, keyed(f.server.Config.Secret, oldCredential)); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/personal-devices", "/personal-devices/00000000000000000000000000000000/projects", "/boot-model-sessions"} {
		if got := f.webRequest(t, "GET", path, nil, session, ""); got.Code != 404 {
			t.Errorf("GET %s = %d; want 404", path, got.Code)
		}
	}
	for _, path := range []string{"/api/personal/v1/devices", "/api/personal/v1/projects", "/api/personal/v1/tasks/claim", "/api/personal/v1/boot-model-sessions"} {
		if got := f.request(t, "GET", path, nil, oldCredential, "control.localhost:8443", ""); got.Code != 404 {
			t.Errorf("GET %s = %d; want 404", path, got.Code)
		}
	}
	got := f.webRequest(t, "POST", "/remote-tasks", map[string]string{"instruction": "modify"}, session, f.server.Config.ControlOrigin)
	if got.Code != 409 || !strings.Contains(got.Body.String(), "remote_development_unavailable") {
		t.Fatalf("remote task = %d %s", got.Code, got.Body.String())
	}
}
