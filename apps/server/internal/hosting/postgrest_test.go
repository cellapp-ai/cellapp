package hosting

import (
	"cellapp/apps/server/appdata"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRealPostgRESTProvisioning(t *testing.T) {
	provisionerDSN := os.Getenv("TEST_APP_DATA_PROVISIONER_URL")
	apiURL := os.Getenv("TEST_POSTGREST_URL")
	adminDSN := os.Getenv("TEST_APP_DATA_DATABASE_URL")
	if provisionerDSN == "" || apiURL == "" || adminDSN == "" {
		t.Skip("real PostgREST and isolated app-data DSNs required")
	}
	f := setup(t)
	ctx := context.Background()
	provisioner, err := pgxpool.New(ctx, provisionerDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer provisioner.Close()
	admin, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	f.server.AppDataDB = provisioner
	f.server.AppDataProbe = func(context.Context, appdata.Names) error { return errors.New("simulated API outage") }
	f.server.Config.PostgRESTURL = apiURL
	requestID := id()
	accessKey := token()
	created := f.request(t, "POST", "/apps", map[string]string{"name": "PostgREST probe", "key": accessKey, "requestId": requestID}, f.token, "control.localhost:8443", "")
	failed := response[map[string]any](t, created, 503)
	if failed["error"] != "app_data_not_ready" {
		t.Fatalf("unexpected create failure: %v", failed)
	}
	var appID string
	if err := f.db.QueryRow(ctx, `SELECT id FROM apps WHERE create_key=$1`, requestID).Scan(&appID); err != nil {
		t.Fatalf("app creation failed before provisioning: %d %s: %v", created.Code, created.Body.String(), err)
	}
	if failed["details"].(map[string]any)["appId"] != appID {
		t.Fatalf("create failure omitted committed app ID: %v", failed)
	}
	names, err := appdata.AppNames(appID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		defer admin.Close()
		_, _ = admin.Exec(ctx, `DROP OWNED BY `+names.APIRole+`,`+names.DDLRole+` CASCADE`)
		_, _ = admin.Exec(ctx, `DROP ROLE IF EXISTS `+names.APIRole+`,`+names.DDLRole)
		_, _ = admin.Exec(ctx, `DELETE FROM cellapp_internal.app_storage_tables WHERE app_id=$1`, appID)
		_, _ = admin.Exec(ctx, `DELETE FROM cellapp_internal.app_storage WHERE app_id=$1`, appID)
		_, _ = admin.Exec(ctx, `DELETE FROM cellapp_internal.app_versions WHERE app_id=$1`, appID)
		_, _ = admin.Exec(ctx, `NOTIFY pgrst, 'reload config'`)
		_, _ = admin.Exec(ctx, `NOTIFY pgrst, 'reload schema'`)
	})
	status := response[map[string]any](t, f.request(t, "GET", "/apps/"+appID+"/data", nil, f.token, "control.localhost:8443", ""), 200)
	if status["state"] != "error" || status["errorCode"] != "api_not_ready" {
		t.Fatalf("wrong failed provisioning status: %v", status)
	}
	f.server.AppDataProbe = f.server.DataProbe
	response[map[string]any](t, f.request(t, "POST", "/apps/"+appID+"/data/reconcile", map[string]any{}, f.token, "control.localhost:8443", ""), 200)
	if err := f.server.DataProbe(ctx, names); err != nil {
		t.Fatalf("real PostgREST probe failed: %v", err)
	}
	manifest := appdata.Manifest{Version: 1, Tables: []appdata.Table{{
		Name: "notes", Columns: []appdata.Column{{Name: "id", Type: "uuid"}, {Name: "body", Type: "text"}},
		PrimaryKey: []string{"id"}, Policies: appdata.Policies{Select: "all-admitted-users", Insert: "all-admitted-users", Update: "all-admitted-users", Delete: "all-admitted-users"},
	}}}
	if w := f.request(t, "POST", "/apps/"+appID+"/data/apply", manifest, f.token, "control.localhost:8443", ""); w.Code != 200 {
		t.Fatalf("table apply: %d %s", w.Code, w.Body.String())
	}
	appHost, _ := url.Parse(f.server.Config.AppURL(appID))
	appCookie := signedInApp(t, f, App{ID: appID, URL: appHost.String()}, f.webSessionCookie(t))
	rowID := "00000000-0000-0000-0000-000000000010"
	if w := f.request(t, "POST", "/rest/v1/notes", map[string]string{"id": rowID, "body": "hello"}, "", appHost.Host, appCookie); w.Code != 201 {
		t.Fatalf("data insert: %d %s", w.Code, w.Body.String())
	}
	if w := f.request(t, "GET", "/rest/v1/notes?select=id,body&body=eq.hello", nil, "", appHost.Host, appCookie); w.Code != 200 || !strings.Contains(w.Body.String(), "hello") {
		t.Fatalf("data select: %d %s", w.Code, w.Body.String())
	}
	if w := f.request(t, "PATCH", "/rest/v1/notes?id=eq."+rowID, map[string]string{"body": "updated"}, "", appHost.Host, appCookie); w.Code != 204 {
		t.Fatalf("data update: %d %s", w.Code, w.Body.String())
	}
	if w := f.request(t, "GET", "/rest/v1/notes?select=id,body&order=body.desc&limit=1&offset=0", nil, "", appHost.Host, appCookie); w.Code != 200 || !strings.Contains(w.Body.String(), "updated") {
		t.Fatalf("data order/page: %d %s", w.Code, w.Body.String())
	}
	if w := f.request(t, "POST", "/rest/v1/notes", map[string]string{"id": id(), "body": strings.Repeat("x", 1<<20)}, "", appHost.Host, appCookie); w.Code != 413 {
		t.Fatalf("oversized data request accepted: %d", w.Code)
	}
	for range 3 {
		if w := f.request(t, "POST", "/rest/v1/notes", map[string]string{"id": id(), "body": strings.Repeat("x", 780000)}, "", appHost.Host, appCookie); w.Code != 201 {
			t.Fatalf("bounded data insert failed: %d %s", w.Code, w.Body.String()[:min(w.Body.Len(), 200)])
		}
	}
	if w := f.request(t, "GET", "/rest/v1/notes?select=id,body", nil, "", appHost.Host, appCookie); w.Code != 413 {
		t.Fatalf("oversized data response accepted: %d", w.Code)
	}
	badOrigin := httptest.NewRequest(http.MethodDelete, "https://"+appHost.Host+"/rest/v1/notes?id=eq."+rowID, nil)
	badOrigin.Host = appHost.Host
	badOrigin.Header.Set("Cookie", appCookie)
	denied := httptest.NewRecorder()
	f.handler.ServeHTTP(denied, badOrigin)
	if denied.Code != 403 {
		t.Fatalf("cross-origin write accepted: %d", denied.Code)
	}
	forged := httptest.NewRequest(http.MethodGet, "https://"+appHost.Host+"/rest/v1/notes?select=id", nil)
	forged.Host = appHost.Host
	forged.Header.Set("Cookie", appCookie)
	forged.Header.Set("Authorization", "Bearer forged")
	forged.Header.Set("Accept-Profile", "cellapp_internal")
	forged.Header.Set("Accept", "application/json")
	returned := httptest.NewRecorder()
	f.handler.ServeHTTP(returned, forged)
	if returned.Code != 200 || !strings.Contains(returned.Body.String(), rowID) {
		t.Fatalf("client headers influenced identity or profile: %d %s", returned.Code, returned.Body.String())
	}
	visitorCookie := f.unlock(t, App{ID: appID, URL: appHost.String()}, accessKey)
	visitor := response[map[string]string](t, f.request(t, "GET", "/_cellapp/me", nil, "", appHost.Host, visitorCookie), 200)
	if w := f.request(t, "GET", "/rest/v1/notes?select=id", nil, "", appHost.Host, visitorCookie); w.Code != 200 {
		t.Fatalf("granted visitor denied: %d %s", w.Code, w.Body.String())
	}
	ownerSession := f.webSessionCookie(t)
	response[any](t, f.webRequest(t, "POST", "/apps/"+appID+"/grants/"+visitor["id"]+"/revoke", nil, ownerSession, f.server.Config.ControlOrigin), 200)
	if w := f.request(t, "GET", "/rest/v1/notes?select=id", nil, "", appHost.Host, visitorCookie); w.Code != 403 {
		t.Fatalf("revoked visitor data access survived: %d", w.Code)
	}
	if w := f.request(t, "DELETE", "/rest/v1/notes?id=eq."+rowID, nil, "", appHost.Host, appCookie); w.Code != 204 {
		t.Fatalf("data delete: %d %s", w.Code, w.Body.String())
	}
	if w := f.request(t, "GET", "/rest/v1/notes", nil, "", strings.Repeat("f", 32)+".apps.localhost:8443", appCookie); w.Code != 401 {
		t.Fatalf("app cookie crossed host: %d", w.Code)
	}
	direct, err := http.Get(apiURL + "/notes?select=id")
	if err != nil {
		t.Fatal(err)
	}
	direct.Body.Close()
	if direct.StatusCode == 200 {
		t.Fatal("direct PostgREST request bypassed gateway")
	}
}
