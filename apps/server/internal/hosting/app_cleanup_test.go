package hosting

import (
	"cellapp/apps/server/appdata"
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDeletedAppDataCleanupRecoversAfterOutage(t *testing.T) {
	dsn := os.Getenv("TEST_APP_DATA_PROVISIONER_URL")
	if dsn == "" {
		t.Skip("TEST_APP_DATA_PROVISIONER_URL required")
	}
	f := setup(t)
	first, _ := f.create(t)
	second, _ := f.create(t)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx := context.Background()
	for _, app := range []App{first, second} {
		if err := appdata.Provision(ctx, pool, app.ID); err != nil {
			t.Fatal(err)
		}
	}
	defer appdata.CleanupApp(ctx, pool, second.ID)
	if err := f.server.deleteOwnerApp(ctx, f.owner, first.ID); err != nil {
		t.Fatal(err)
	}
	firstHost, _ := url.Parse(first.URL)
	if response := f.request(t, "GET", "/", nil, "", firstHost.Host, ""); response.Code != 404 {
		t.Fatalf("deleted app still available: %d", response.Code)
	}
	// The control tombstone survives an unavailable app database.
	if err := f.server.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := f.db.QueryRow(ctx, `SELECT data_state FROM apps WHERE id=$1`, first.ID).Scan(&state); err != nil || state != "cleanup_pending" {
		t.Fatalf("outage state: %q %v", state, err)
	}
	f.server.AppDataDB = pool
	if err := f.server.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.server.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(ctx, `SELECT data_state FROM apps WHERE id=$1`, first.ID).Scan(&state); err != nil || state != "cleaned" {
		t.Fatalf("recovery state: %q %v", state, err)
	}
	if err := appdata.VerifyProvisioned(ctx, pool, second.ID); err != nil {
		t.Fatalf("other app affected: %v", err)
	}
	if err := appdata.VerifyProvisioned(ctx, pool, first.ID); err == nil {
		t.Fatal("deleted app schema remained")
	}
}
