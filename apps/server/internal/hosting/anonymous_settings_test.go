package hosting

import (
	"context"
	"net/http"
	"testing"
)

func TestOwnerAnonymousSettings(t *testing.T) {
	f := setup(t)
	app, _ := f.create(t)
	session := f.webSessionCookie(t)
	path := "/apps/" + app.ID + "/anonymous"
	initial := response[anonymousSettings](t, f.webRequest(t, http.MethodGet, path, nil, session, ""), 200)
	if initial.Pages || initial.Read || initial.Write || initial.EffectiveWrite || initial.OperatorWritesEnabled {
		t.Fatalf("unexpected default settings: %+v", initial)
	}
	full := map[string]bool{"pages": true, "read": true, "write": true}
	response[any](t, f.webRequest(t, http.MethodPut, path, full, session, f.server.Config.ControlOrigin), 409)
	response[any](t, f.webRequest(t, http.MethodPut, path, full, session, ""), 403)
	response[any](t, f.webRequest(t, http.MethodPut, path, map[string]bool{"pages": true, "read": true}, session, f.server.Config.ControlOrigin), 400)
	response[any](t, f.webRequest(t, http.MethodPut, path, map[string]bool{"pages": false, "read": true, "write": false}, session, f.server.Config.ControlOrigin), 400)
	response[any](t, f.webRequest(t, http.MethodPost, "/apps/"+app.ID+"/visibility/public", nil, session, f.server.Config.ControlOrigin), 200)
	state := response[anonymousSettings](t, f.webRequest(t, http.MethodPut, path, full, session, f.server.Config.ControlOrigin), 200)
	if !state.Pages || !state.Read || !state.Write || state.EffectiveWrite {
		t.Fatalf("wrong owner settings: %+v", state)
	}
	f.server.Config.AllowAnonymousAppDataWrites = true
	state = response[anonymousSettings](t, f.request(t, http.MethodGet, "/api/control/v1"+path, nil, f.token, "control.localhost:8443", ""), 200)
	if !state.EffectiveWrite || !state.OperatorWritesEnabled {
		t.Fatalf("operator setting missing: %+v", state)
	}
	state = response[anonymousSettings](t, f.request(t, http.MethodPut, "/api/control/v1"+path,
		map[string]bool{"pages": true, "read": false, "write": true}, f.token, "control.localhost:8443", ""), 200)
	if !state.Pages || state.Read || !state.Write || !state.EffectiveWrite {
		t.Fatalf("Bearer update failed: %+v", state)
	}
	response[any](t, f.request(t, http.MethodGet, "/api/control/v1"+path, nil, "", "control.localhost:8443", ""), 401)
	otherOwner, otherToken := id(), token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$1)`, otherOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO credentials(id,hash,owner_id,expires_at) VALUES($1,$2,$3,now()+interval '1 day')`, id(), keyed(f.server.Config.Secret, otherToken), otherOwner); err != nil {
		t.Fatal(err)
	}
	response[any](t, f.request(t, http.MethodPut, "/api/control/v1"+path, full, otherToken, "control.localhost:8443", ""), 404)
	response[any](t, f.webRequest(t, http.MethodPost, "/apps/"+app.ID+"/visibility/restricted", nil, session, f.server.Config.ControlOrigin), 200)
	state = response[anonymousSettings](t, f.webRequest(t, http.MethodGet, path, nil, session, ""), 200)
	if state.Pages || state.Read || state.Write || state.EffectiveWrite {
		t.Fatalf("restricted app retained anonymous settings: %+v", state)
	}
}
