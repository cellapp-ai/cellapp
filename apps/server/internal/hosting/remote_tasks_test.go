package hosting

import (
	"context"
	"strings"
	"testing"
)

func TestRemoteDevelopmentUnavailableInFirstRelease(t *testing.T) {
	f := setup(t)
	session := f.webSessionCookie(t)
	deviceID, projectID, credential := id(), id(), token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO personal_devices(id,owner_id,name,credential_hash,capabilities,last_seen_at)
	VALUES($1,$2,'Laptop',$3,'{"development":true}',now())`, deviceID, f.owner, keyed(f.server.Config.Secret, credential)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO device_projects(id,owner_id,device_id,name,permissions)
	VALUES($1,$2,$3,'Project','{"read":true,"write":true}')`, projectID, f.owner, deviceID); err != nil {
		t.Fatal(err)
	}
	input := map[string]string{"deviceId": deviceID, "projectId": projectID, "model": "unused", "instruction": "Change the heading"}
	got := f.webRequest(t, "POST", "/remote-tasks", input, session, f.server.Config.ControlOrigin)
	if got.Code != 409 || !strings.Contains(got.Body.String(), "remote_development_unavailable") {
		t.Fatalf("remote development submission: %d %s", got.Code, got.Body.String())
	}
	var count int
	if err := f.db.QueryRow(context.Background(), `SELECT count(*) FROM remote_agent_tasks`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("remote task was queued: %d %v", count, err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO remote_agent_tasks(id,owner_id,device_id,project_id,kind,request)
	VALUES($1,$2,$3,$4,'develop','{}')`, id(), f.owner, deviceID, projectID); err != nil {
		t.Fatal(err)
	}
	if claim := f.request(t, "GET", "/api/personal/v1/tasks/claim?wait=25", nil, credential, "control.localhost:8443", ""); claim.Code != 404 {
		t.Fatalf("dormant task was claimed: %d %s", claim.Code, claim.Body.String())
	}
	if err := f.db.QueryRow(context.Background(), `SELECT count(*) FROM remote_agent_tasks WHERE state='queued'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("dormant task state changed: %d %v", count, err)
	}
}
