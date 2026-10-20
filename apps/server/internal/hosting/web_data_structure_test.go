package hosting

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"cellapp/apps/server/appdata"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWebDataStructure(t *testing.T) {
	dsn := os.Getenv("TEST_APP_DATA_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_APP_DATA_DATABASE_URL required for isolated app-data integration")
	}
	f := setup(t)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	f.server.AppDataDB = pool
	app, _ := f.create(t)
	path := "/apps/" + app.ID + "/data/structure"
	session := f.webSessionCookie(t)
	if got := response[map[string]any](t, f.webRequest(t, http.MethodGet, path, nil, session, ""), 200); got["version"] != float64(0) || len(got["tables"].([]any)) != 0 {
		t.Fatalf("expected empty applied structure: %v", got)
	}
	manifest := appdata.Manifest{Version: 1, AllowDestructive: true, Tables: []appdata.Table{{
		Name: "notes", Columns: []appdata.Column{{Name: "id", Type: "uuid"}, {Name: "owner_id", Type: "text"}, {Name: "body", Type: "text", Nullable: true}},
		PrimaryKey: []string{"id"}, Indexes: []appdata.Index{{Columns: []string{"owner_id"}}},
		Policies: appdata.Policies{Select: "own-rows", Insert: "own-rows", OwnerColumn: "owner_id"},
	}}}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO cellapp_internal.app_versions(app_id,version,digest,manifest) VALUES($1,1,$2,$3)`, app.ID, strings.Repeat("f", 64), raw); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM cellapp_internal.app_versions WHERE app_id=$1`, app.ID)
	})
	if _, err := f.db.Exec(ctx, `UPDATE apps SET data_state='pending' WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	w := f.webRequest(t, http.MethodGet, path, nil, session, "")
	got := response[map[string]any](t, w, 200)
	if got["state"] != "pending" || got["version"] != float64(1) || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("wrong structure status: %v", got)
	}
	if strings.Contains(w.Body.String(), "allowDestructive") || strings.Contains(w.Body.String(), "digest") || strings.Contains(w.Body.String(), "app_") {
		t.Fatalf("internal metadata leaked: %s", w.Body.String())
	}
	tables := got["tables"].([]any)
	if len(tables) != 1 || tables[0].(map[string]any)["name"] != "notes" {
		t.Fatalf("wrong applied tables: %v", tables)
	}
	table := tables[0].(map[string]any)
	if len(table["columns"].([]any)) != 3 || table["primaryKey"].([]any)[0] != "id" || len(table["indexes"].([]any)) != 1 || table["policies"].(map[string]any)["select"] != "own-rows" {
		t.Fatalf("missing applied structure fields: %v", table)
	}
	if _, err := f.db.Exec(ctx, `UPDATE apps SET data_state='error' WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if got := response[map[string]any](t, f.webRequest(t, http.MethodGet, path, nil, session, ""), 200); got["state"] != "error" {
		t.Fatalf("error readiness hidden: %v", got)
	}
	if _, err := f.db.Exec(ctx, `UPDATE apps SET data_state='ready' WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if got := response[map[string]any](t, f.webRequest(t, http.MethodGet, path, nil, session, ""), 200); got["state"] != "ready" {
		t.Fatalf("ready status hidden: %v", got)
	}
	response[any](t, f.webRequest(t, http.MethodGet, path, nil, "", ""), 401)
	foreign := id()
	if _, err := f.db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','foreign')`, foreign); err != nil {
		t.Fatal(err)
	}
	foreignSession := token()
	if _, err := f.db.Exec(ctx, `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(f.server.Config.Secret, foreignSession), foreign); err != nil {
		t.Fatal(err)
	}
	response[any](t, f.webRequest(t, http.MethodGet, path, nil, foreignSession, ""), 404)
	visitorHost := app.ID + ".apps.localhost:8443"
	if w := f.request(t, http.MethodGet, "/api/console"+path, nil, "", visitorHost, "__Host-owner="+session); w.Code == 200 {
		t.Fatal("visitor host exposed owner table structure")
	}
	f.server.AppDataDB = nil
	response[any](t, f.webRequest(t, http.MethodGet, path, nil, session, ""), 503)
}
