package hosting

import (
	"testing"
)

func TestPublicationPauseBlocksNewAndInFlightDeployments(t *testing.T) {
	f := setup(t)
	app, _ := f.create(t)
	deployment, _ := f.stage(t, app, "ready")
	f.server.Config.PublicationPaused = true
	if w := f.request(t, "POST", "/apps/"+app.ID+"/deployments", map[string]any{}, f.token, "control.localhost:8443", ""); w.Code != 503 {
		t.Fatalf("new deployment during pause=%d", w.Code)
	}
	path := "/apps/" + app.ID + "/deployments/" + deployment
	if w := f.request(t, "PUT", path+"/files/0", []byte("ready"), f.token, "control.localhost:8443", ""); w.Code != 503 {
		t.Fatalf("upload during pause=%d", w.Code)
	}
	if w := f.request(t, "POST", path+"/publish", map[string]any{}, f.token, "control.localhost:8443", ""); w.Code != 503 {
		t.Fatalf("publication during pause=%d", w.Code)
	}
	f.server.Config.PublicationPaused = false
	f.uploadAndPublish(t, app, deployment, "ready")
	if _, ok := f.storage.data[releaseManifestKey(deployment)]; !ok {
		t.Fatal("published release lacks private manifest")
	}
}
