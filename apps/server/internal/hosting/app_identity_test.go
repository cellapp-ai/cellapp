package hosting

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAppHandoffSingleUseAndPlatformSessionBinding(t *testing.T) {
	f := setup(t)
	a, _ := f.create(t)
	b, _ := f.create(t)
	appURL, _ := url.Parse(a.URL)
	otherURL, _ := url.Parse(b.URL)
	returnPath := "/notes?view=mine"
	start := "/auth/app?" + url.Values{"app": {a.ID}, "return": {returnPath}}.Encode()
	login := f.request(t, "GET", start, nil, "", "control.localhost:8443", "")
	if login.Code != http.StatusSeeOther || !strings.HasPrefix(login.Header().Get("Location"), "/auth/login?") {
		t.Fatalf("expected login redirect: %d %q", login.Code, login.Header().Get("Location"))
	}
	platform := f.webSessionCookie(t)
	issued := f.request(t, "GET", start, nil, "", "control.localhost:8443", "__Host-owner="+platform)
	if issued.Code != http.StatusSeeOther || !strings.HasPrefix(issued.Header().Get("Location"), a.URL+"/_cellapp/handoff?") {
		t.Fatalf("handoff target: %d %q", issued.Code, issued.Header().Get("Location"))
	}
	target, _ := url.Parse(issued.Header().Get("Location"))
	if wrong := f.request(t, "GET", target.RequestURI(), nil, "", otherURL.Host, ""); wrong.Code != 401 {
		t.Fatalf("ticket accepted by other app: %d", wrong.Code)
	}
	consumed := f.request(t, "GET", target.RequestURI(), nil, "", appURL.Host, "")
	if consumed.Code != http.StatusSeeOther || consumed.Header().Get("Location") != returnPath {
		t.Fatalf("handoff return: %d %q", consumed.Code, consumed.Header().Get("Location"))
	}
	cookies := consumed.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "__Host-cellapp" || cookies[0].Domain != "" || !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatalf("unsafe app session cookie: %+v", cookies)
	}
	appCookie := cookies[0].Name + "=" + cookies[0].Value
	if user, err := f.server.appSession(newAppRequest(appURL.Host, appCookie), a.ID); err != nil || user != f.owner {
		t.Fatalf("app session: %q %v", user, err)
	}
	if _, err := f.server.appSession(newAppRequest(otherURL.Host, appCookie), b.ID); err == nil {
		t.Fatal("app session crossed app boundary")
	}
	if replay := f.request(t, "GET", target.RequestURI(), nil, "", appURL.Host, ""); replay.Code != 401 {
		t.Fatalf("ticket replay accepted: %d", replay.Code)
	}
	expiring := f.request(t, "GET", start, nil, "", "control.localhost:8443", "__Host-owner="+platform)
	expiringURL, _ := url.Parse(expiring.Header().Get("Location"))
	if _, err := f.db.Exec(newAppRequest(appURL.Host, "").Context(), `UPDATE app_handoff_tickets SET expires_at=now()-interval '1 second' WHERE hash=$1`, keyed(f.server.Config.Secret, expiringURL.Query().Get("ticket"))); err != nil {
		t.Fatal(err)
	}
	if expired := f.request(t, "GET", expiringURL.RequestURI(), nil, "", appURL.Host, ""); expired.Code != 401 {
		t.Fatalf("expired ticket accepted: %d", expired.Code)
	}
	if _, err := f.db.Exec(newAppRequest(appURL.Host, "").Context(), `DELETE FROM browser_sessions WHERE hash=$1`, keyed(f.server.Config.Secret, platform)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.server.appSession(newAppRequest(appURL.Host, appCookie), a.ID); err == nil {
		t.Fatal("app session survived platform logout")
	}
}

func newAppRequest(host, session string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "https://"+host+"/", nil)
	if session != "" {
		r.Header.Set("Cookie", session)
	}
	return r
}

func signedInApp(t *testing.T, f *fixture, a App, platformCookie string) string {
	t.Helper()
	host, _ := url.Parse(a.URL)
	issued := f.request(t, "GET", "/auth/app?app="+a.ID, nil, "", "control.localhost:8443", "__Host-owner="+platformCookie)
	if issued.Code != http.StatusSeeOther {
		t.Fatalf("ticket issue: %d %s", issued.Code, issued.Body.String())
	}
	target, _ := url.Parse(issued.Header().Get("Location"))
	consumed := f.request(t, "GET", target.RequestURI(), nil, "", host.Host, "")
	if consumed.Code != http.StatusSeeOther {
		t.Fatalf("ticket consume: %d %s", consumed.Code, consumed.Body.String())
	}
	c := consumed.Result().Cookies()[0]
	return c.Name + "=" + c.Value
}

func TestAppScopedUserAndSignedInAdmission(t *testing.T) {
	f := setup(t)
	a, key := f.create(t)
	b, _ := f.create(t)
	appHost, _ := url.Parse(a.URL)
	bHost, _ := url.Parse(b.URL)
	platform := f.webSessionCookie(t)
	aCookie := signedInApp(t, f, a, platform)
	bCookie := signedInApp(t, f, b, platform)
	first := response[map[string]string](t, f.request(t, "GET", "/_cellapp/me", nil, "", appHost.Host, aCookie), 200)
	again := response[map[string]string](t, f.request(t, "GET", "/_cellapp/me?v=1.1", nil, "", appHost.Host, aCookie), 200)
	other := response[map[string]string](t, f.request(t, "GET", "/_cellapp/me", nil, "", bHost.Host, bCookie), 200)
	if first["id"] == "" || first["id"] != again["id"] || first["id"] == other["id"] || first["id"] == f.owner || len(first) != 2 {
		t.Fatalf("app identity leaked or unstable: %v %v %v", first, again, other)
	}
	dateAlias := response[map[string]string](t, f.request(t, "GET", "/_cellapp/me?v=261022", nil, "", appHost.Host, aCookie), 200)
	if dateAlias["id"] != first["id"] {
		t.Fatal("date alias selected a different identity")
	}
	response[any](t, f.request(t, "GET", "/_cellapp/me?v=1.1", nil, "", appHost.Host, ""), 401)
	response[any](t, f.request(t, "GET", "/_cellapp/me?v=1.1", nil, "", bHost.Host, aCookie), 401)
	visitor := id()
	_, err := f.db.Exec(newAppRequest(appHost.Host, "").Context(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','visitor')`, visitor)
	if err != nil {
		t.Fatal(err)
	}
	visitorPlatform := token()
	_, err = f.db.Exec(newAppRequest(appHost.Host, "").Context(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, visitorPlatform), visitor)
	if err != nil {
		t.Fatal(err)
	}
	visitorCookie := signedInApp(t, f, a, visitorPlatform)
	response[any](t, f.request(t, "GET", "/_cellapp/me", nil, "", appHost.Host, visitorCookie), 403)
	response[any](t, f.request(t, "GET", "/_cellapp/me?v=1.1", nil, "", appHost.Host, visitorCookie), 403)
	if _, err := f.db.Exec(newAppRequest(appHost.Host, "").Context(), `UPDATE apps SET visibility='public' WHERE id=$1`, a.ID); err != nil {
		t.Fatal(err)
	}
	publicUser := response[map[string]string](t, f.request(t, "GET", "/_cellapp/me", nil, "", appHost.Host, visitorCookie), 200)
	if publicUser["id"] == "" || publicUser["id"] == first["id"] {
		t.Fatal("public app identity invalid")
	}
	if _, err := f.db.Exec(newAppRequest(appHost.Host, "").Context(), `UPDATE apps SET visibility='restricted' WHERE id=$1`, a.ID); err != nil {
		t.Fatal(err)
	}
	response[any](t, f.request(t, "GET", "/_cellapp/me", nil, "", appHost.Host, visitorCookie), 403)
	redeem := f.request(t, "POST", "/_cellapp/redeem", url.Values{"key": {key}}.Encode(), "", appHost.Host, visitorCookie)
	if redeem.Code != http.StatusSeeOther {
		t.Fatalf("redeem: %d %s", redeem.Code, redeem.Body.String())
	}
	response[any](t, f.request(t, "GET", "/_cellapp/me", nil, "", appHost.Host, visitorCookie), 200)
	if err := f.server.resetOwnerKey(newAppRequest(appHost.Host, "").Context(), f.owner, a.ID, token()); err != nil {
		t.Fatal(err)
	}
	response[any](t, f.request(t, "GET", "/_cellapp/me", nil, "", appHost.Host, visitorCookie), 200)
}

func TestAccessRequestApprovalRevokeAndOwnerScope(t *testing.T) {
	f := setup(t)
	a, _ := f.create(t)
	host, _ := url.Parse(a.URL)
	ownerSession := f.webSessionCookie(t)
	visitor := id()
	visitorSession := token()
	ctx := newAppRequest(host.Host, "").Context()
	if _, err := f.db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','requester')`, visitor); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, visitorSession), visitor); err != nil {
		t.Fatal(err)
	}
	appCookie := signedInApp(t, f, a, visitorSession)
	request := f.request(t, "POST", "/_cellapp/request", "", "", host.Host, appCookie)
	if request.Code != 200 {
		t.Fatalf("request: %d %s", request.Code, request.Body.String())
	}
	list := response[struct {
		Requests []webAccessRequest `json:"requests"`
		Grants   []webAccessGrant   `json:"grants"`
	}](t, f.webRequest(t, "GET", "/apps/"+a.ID+"/access", nil, ownerSession, ""), 200)
	if len(list.Requests) != 1 || list.Requests[0].Status != "pending" || len(list.Grants) != 0 {
		t.Fatalf("unexpected access list: %+v", list)
	}
	response[any](t, f.request(t, "GET", "/_cellapp/me?v=1.1", nil, "", host.Host, appCookie), 403)
	userID := list.Requests[0].UserID
	endpoint := "/apps/" + a.ID + "/requests/" + userID + "/approve"
	response[any](t, f.webRequest(t, "POST", endpoint, nil, ownerSession, f.server.Config.ControlOrigin), 200)
	response[any](t, f.webRequest(t, "POST", endpoint, nil, ownerSession, f.server.Config.ControlOrigin), 200)
	me := response[map[string]string](t, f.request(t, "GET", "/_cellapp/me", nil, "", host.Host, appCookie), 200)
	if me["id"] != userID {
		t.Fatalf("approved app user changed: %v", me)
	}
	response[any](t, f.webRequest(t, "POST", "/apps/"+a.ID+"/grants/"+userID+"/revoke", nil, ownerSession, f.server.Config.ControlOrigin), 200)
	response[any](t, f.request(t, "GET", "/_cellapp/me", nil, "", host.Host, appCookie), 403)
	response[any](t, f.request(t, "GET", "/_cellapp/me?v=261022", nil, "", host.Host, appCookie), 403)
	response[any](t, f.webRequest(t, "POST", endpoint, nil, ownerSession, f.server.Config.ControlOrigin), 409)
	response[any](t, f.request(t, "GET", "/_cellapp/me", nil, "", host.Host, appCookie), 403)
	response[any](t, f.webRequest(t, "POST", "/apps/"+a.ID+"/visibility/public", nil, ownerSession, ""), 403)
	response[any](t, f.webRequest(t, "POST", "/apps/"+a.ID+"/visibility/public", nil, ownerSession, f.server.Config.ControlOrigin), 200)
	response[any](t, f.request(t, "GET", "/_cellapp/me", nil, "", host.Host, appCookie), 200)
	response[any](t, f.webRequest(t, "POST", "/apps/"+a.ID+"/visibility/restricted", nil, ownerSession, f.server.Config.ControlOrigin), 200)
	response[any](t, f.request(t, "GET", "/_cellapp/me", nil, "", host.Host, appCookie), 403)
	otherSession := token()
	if _, err := f.db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','other-owner')`, id()); err != nil {
		t.Fatal(err)
	}
	var otherID string
	if err := f.db.QueryRow(ctx, `SELECT id FROM owners WHERE subject='other-owner'`).Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, otherSession), otherID); err != nil {
		t.Fatal(err)
	}
	response[any](t, f.webRequest(t, "GET", "/apps/"+a.ID+"/access", nil, otherSession, ""), 404)
}

func TestWebDataStatusDoesNotHideFailureOrOtherOwner(t *testing.T) {
	f := setup(t)
	a, _ := f.create(t)
	ownerSession := f.webSessionCookie(t)
	response[any](t, f.webRequest(t, "GET", "/apps/"+a.ID+"/data", nil, ownerSession, ""), 503)
	otherOwner, otherSession := id(), token()
	if _, err := f.db.Exec(newAppRequest("control.localhost:8443", "").Context(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','data-outsider')`, otherOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(newAppRequest("control.localhost:8443", "").Context(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, otherSession), otherOwner); err != nil {
		t.Fatal(err)
	}
	response[any](t, f.webRequest(t, "GET", "/apps/"+a.ID+"/data", nil, otherSession, ""), 404)
}

func TestConcurrentGrantRevokeAndAppDeletion(t *testing.T) {
	f := setup(t)
	a, key := f.create(t)
	visitorCookie := f.unlock(t, a, key)
	host, _ := url.Parse(a.URL)
	user := response[map[string]string](t, f.request(t, "GET", "/_cellapp/me", nil, "", host.Host, visitorCookie), 200)["id"]
	ownerCookie := f.webSessionCookie(t)
	statuses := make(chan int, 1)
	deletion := make(chan error, 1)
	go func() {
		statuses <- f.webRequest(t, "POST", "/apps/"+a.ID+"/grants/"+user+"/revoke", nil, ownerCookie, f.server.Config.ControlOrigin).Code
	}()
	go func() { deletion <- f.server.deleteOwnerApp(newAppRequest(host.Host, "").Context(), f.owner, a.ID) }()
	status := <-statuses
	if err := <-deletion; err != nil {
		t.Fatal(err)
	}
	if status != 200 && status != 404 {
		t.Fatalf("unexpected concurrent revoke result: %d", status)
	}
	response[any](t, f.request(t, "GET", "/_cellapp/me", nil, "", host.Host, visitorCookie), 404)
}
