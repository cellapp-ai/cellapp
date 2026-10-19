package hosting

import (
	"cellapp/apps/server/appdata"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPDataToolsWithIsolatedPostgREST(t *testing.T) {
	appURL := os.Getenv("TEST_APP_DATA_PROVISIONER_URL")
	apiURL := os.Getenv("TEST_POSTGREST_URL")
	adminURL := os.Getenv("TEST_APP_DATA_DATABASE_URL")
	if appURL == "" || apiURL == "" || adminURL == "" {
		t.Skip("isolated application-data PostgreSQL and PostgREST required")
	}
	f := setup(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	appDB, err := pgxpool.New(ctx, appURL)
	if err != nil {
		t.Fatal(err)
	}
	defer appDB.Close()
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	f.server.AppDataDB = appDB
	f.server.Config.PostgRESTURL = apiURL
	f.server.AppDataProbe = f.server.DataProbe
	clientID, access := id(), token()
	if _, err := f.db.Exec(ctx, `INSERT INTO mcp_oauth_clients(id,name,redirect_uris) VALUES($1,'data-test','[]')`, clientID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO mcp_oauth_consents(owner_id,client_id,scopes) VALUES($1,$2,ARRAY['mcp:read','mcp:deploy','mcp:data'])`, f.owner, clientID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO mcp_oauth_tokens(hash,owner_id,client_id,resource,scopes,expires_at) VALUES($1,$2,$3,$4,ARRAY['mcp:read','mcp:deploy','mcp:data'],now()+interval '1 hour')`, keyed(f.server.Config.Secret, access), f.owner, clientID, f.server.Config.ControlOrigin+"/mcp"); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(f.handler)
	defer httpServer.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "cellapp-data-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL + "/mcp", DisableStandaloneSSE: true, HTTPClient: &http.Client{Transport: mcpTestTransport{http.DefaultTransport, access}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	call := func(name string, args map[string]any) *mcp.CallToolResult {
		t.Helper()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	created := call("create_app", map[string]any{"name": "MCP data test", "requestId": id()})
	if created.IsError {
		t.Fatalf("create: %#v", created)
	}
	appID := created.StructuredContent.(map[string]any)["app"].(map[string]any)["id"].(string)
	names, err := appdata.AppNames(appID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_, _ = admin.Exec(cleanup, `DROP OWNED BY `+names.APIRole+`,`+names.DDLRole+` CASCADE`)
		_, _ = admin.Exec(cleanup, `DROP ROLE IF EXISTS `+names.APIRole+`,`+names.DDLRole)
		_, _ = admin.Exec(cleanup, `DELETE FROM cellapp_internal.app_storage_tables WHERE app_id=$1`, appID)
		_, _ = admin.Exec(cleanup, `DELETE FROM cellapp_internal.app_storage WHERE app_id=$1`, appID)
		_, _ = admin.Exec(cleanup, `DELETE FROM cellapp_internal.app_versions WHERE app_id=$1`, appID)
		_, _ = admin.Exec(cleanup, `NOTIFY pgrst, 'reload config'`)
		_, _ = admin.Exec(cleanup, `NOTIFY pgrst, 'reload schema'`)
	})
	status := call("data_status", map[string]any{"appId": appID})
	if status.IsError || status.StructuredContent.(map[string]any)["state"] != "ready" {
		t.Fatalf("status: %#v", status)
	}
	foreignOwner := id()
	if _, err := f.db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$2)`, foreignOwner, foreignOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.server.mcpDataStatus(ctx, foreignOwner, appID); err == nil {
		t.Fatal("cross-owner data status accepted")
	}
	f.server.AppDataProbe = func(context.Context, appdata.Names) error { return errors.New("test API outage") }
	if result := call("data_reconcile", map[string]any{"appId": appID}); !result.IsError {
		t.Fatal("failed data reconcile reported ready")
	}
	if result := call("data_verify", map[string]any{"appId": appID}); !result.IsError {
		t.Fatal("failed data probe verified successfully")
	}
	failedStatus := call("data_status", map[string]any{"appId": appID})
	if failedStatus.IsError || failedStatus.StructuredContent.(map[string]any)["state"] != "error" {
		t.Fatalf("failed provisioning status: %#v", failedStatus)
	}
	f.server.AppDataProbe = f.server.DataProbe
	if result := call("data_reconcile", map[string]any{"appId": appID}); result.IsError {
		t.Fatalf("idempotent reconcile: %#v", result)
	}
	manifest := appdata.Manifest{Version: 1, Tables: []appdata.Table{{Name: "notes", Columns: []appdata.Column{{Name: "id", Type: "uuid"}, {Name: "body", Type: "text"}}, PrimaryKey: []string{"id"}, Policies: appdata.Policies{Select: "all-admitted-users", Insert: "all-admitted-users"}}}}
	input := map[string]any{"appId": appID, "manifest": manifest}
	invalid := appdata.Manifest{Version: 1, Tables: []appdata.Table{{Name: "rpc", Columns: []appdata.Column{{Name: "id", Type: "uuid"}}, PrimaryKey: []string{"id"}}}}
	if result := call("data_plan", map[string]any{"appId": appID, "manifest": invalid}); !result.IsError {
		t.Fatal("reserved table accepted")
	}
	invalid.Tables[0].Name = "notes"
	invalid.Tables[0].Columns[0].Type = "serial"
	if result := call("data_plan", map[string]any{"appId": appID, "manifest": invalid}); !result.IsError {
		t.Fatal("invalid column type accepted")
	}
	plan := call("data_plan", input)
	if plan.IsError {
		t.Fatalf("plan: %#v", plan)
	}
	planDigest := plan.StructuredContent.(map[string]any)["digest"].(string)
	if !digestPattern.MatchString(planDigest) {
		t.Fatalf("plan digest: %#v", plan)
	}
	if result := call("data_apply", map[string]any{"appId": appID, "manifest": manifest, "reviewedDigest": digest([]byte("wrong"))}); !result.IsError {
		t.Fatal("wrong reviewed digest accepted")
	}
	applied := call("data_apply", map[string]any{"appId": appID, "manifest": manifest, "reviewedDigest": planDigest})
	if applied.IsError {
		t.Fatalf("apply: %#v", applied)
	}
	verified := call("data_verify", map[string]any{"appId": appID})
	if verified.IsError || verified.StructuredContent.(map[string]any)["version"] != float64(1) {
		t.Fatalf("verify: %#v", verified)
	}
	if result := call("data_apply", map[string]any{"appId": appID, "manifest": manifest, "reviewedDigest": planDigest}); result.IsError {
		t.Fatalf("same-version retry: %#v", result)
	}
	changedVersion := manifest
	changedVersion.Tables = []appdata.Table{{Name: "other", Columns: []appdata.Column{{Name: "id", Type: "uuid"}}, PrimaryKey: []string{"id"}}}
	if result := call("data_plan", map[string]any{"appId": appID, "manifest": changedVersion}); !result.IsError {
		t.Fatal("same-version content conflict accepted")
	}
	var activeDeployment *string
	if err := f.db.QueryRow(ctx, `SELECT active_deployment FROM apps WHERE id=$1`, appID).Scan(&activeDeployment); err != nil || activeDeployment != nil {
		t.Fatalf("table change affected static publication: %v %v", activeDeployment, err)
	}
	if _, err := f.server.mcpDataPlan(ctx, foreignOwner, appID, manifest); err == nil {
		t.Fatal("cross-owner data plan accepted")
	}
	destructive := appdata.Manifest{Version: 2, AllowDestructive: true, Tables: []appdata.Table{}}
	deletePlan := call("data_plan", map[string]any{"appId": appID, "manifest": destructive})
	if deletePlan.IsError || deletePlan.StructuredContent.(map[string]any)["requiresDestructive"] != true {
		t.Fatalf("destructive plan: %#v", deletePlan)
	}
	deleteDigest := deletePlan.StructuredContent.(map[string]any)["digest"].(string)
	if result := call("data_apply", map[string]any{"appId": appID, "manifest": destructive, "reviewedDigest": deleteDigest}); !result.IsError {
		t.Fatal("destructive apply without explicit allow accepted")
	}
	if result := call("data_apply", map[string]any{"appId": appID, "manifest": destructive, "reviewedDigest": deleteDigest, "allowDestructive": true}); result.IsError {
		t.Fatalf("destructive apply: %#v", result)
	}
	if result := call("data_verify", map[string]any{"appId": appID}); result.IsError || result.StructuredContent.(map[string]any)["version"] != float64(2) {
		t.Fatalf("destructive verify: %#v", result)
	}
	if _, err := f.db.Exec(ctx, `UPDATE mcp_oauth_tokens SET scopes=ARRAY['mcp:read','mcp:deploy'] WHERE hash=$1`, keyed(f.server.Config.Secret, access)); err != nil {
		t.Fatal(err)
	}
	if result := call("data_plan", input); !result.IsError {
		t.Fatal("data plan accepted without mcp:data")
	}
}
