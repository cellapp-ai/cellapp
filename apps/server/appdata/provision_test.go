package appdata

import (
	"cellapp/apps/server/migrations"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func provisionPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	provisionURL, adminURL := os.Getenv("TEST_APP_DATA_PROVISIONER_URL"), os.Getenv("TEST_APP_DATA_DATABASE_URL")
	if provisionURL == "" || adminURL == "" {
		t.Skip("TEST_APP_DATA_PROVISIONER_URL and TEST_APP_DATA_DATABASE_URL required")
	}
	ctx := context.Background()
	provisioner, err := pgxpool.New(ctx, provisionURL)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		provisioner.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { provisioner.Close(); admin.Close() })
	return provisioner, admin
}

func randomAppID(t *testing.T) string {
	t.Helper()
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(random[:])
}

func cleanupApp(t *testing.T, admin *pgxpool.Pool, appID string) {
	t.Helper()
	names, err := AppNames(appID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, statement := range []string{
			`DROP OWNED BY ` + quoted(names.APIRole) + `,` + quoted(names.AnonymousRole) + `,` + quoted(names.DDLRole) + ` CASCADE`,
			`DROP ROLE IF EXISTS ` + quoted(names.APIRole),
			`DROP ROLE IF EXISTS ` + quoted(names.AnonymousRole),
			`DROP ROLE IF EXISTS ` + quoted(names.DDLRole),
		} {
			if _, err := admin.Exec(ctx, statement); err != nil {
				t.Errorf("cleanup %s: %v", statement, err)
			}
		}
		_, _ = admin.Exec(ctx, `DELETE FROM cellapp_internal.app_storage_tables WHERE app_id=$1`, appID)
		_, _ = admin.Exec(ctx, `DELETE FROM cellapp_internal.app_storage WHERE app_id=$1`, appID)
		_, _ = admin.Exec(ctx, `DELETE FROM cellapp_internal.app_versions WHERE app_id=$1`, appID)
	})
}

func TestProvisionRetryConcurrentAndIsolated(t *testing.T) {
	provisioner, admin := provisionPools(t)
	first, second := randomAppID(t), randomAppID(t)
	cleanupApp(t, admin, first)
	cleanupApp(t, admin, second)
	ctx := context.Background()
	firstNames, _ := AppNames(first)
	if _, err := provisioner.Exec(ctx, `CREATE ROLE `+quoted(firstNames.DDLRole)+` NOLOGIN`); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errors <- Provision(ctx, provisioner, first)
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := Provision(ctx, provisioner, first); err != nil {
		t.Fatal(err)
	}
	if err := Provision(ctx, provisioner, second); err != nil {
		t.Fatal(err)
	}
	for _, appID := range []string{first, second} {
		if err := VerifyProvisioned(ctx, provisioner, appID); err != nil {
			t.Fatal(err)
		}
	}
	secondNames, _ := AppNames(second)
	var cross, own bool
	if err := admin.QueryRow(ctx, `SELECT has_schema_privilege($1,$2,'USAGE'),has_schema_privilege($1,$3,'USAGE')`, firstNames.APIRole, secondNames.Schema, firstNames.Schema).Scan(&cross, &own); err != nil {
		t.Fatal(err)
	}
	if cross || !own {
		t.Fatalf("API role isolation: own=%t cross=%t", own, cross)
	}
	allowed, err := provisioner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer allowed.Rollback(ctx)
	if _, err := allowed.Exec(ctx, `SET LOCAL ROLE `+quoted(firstNames.DDLRole)); err != nil {
		t.Fatal(err)
	}
	if _, err := allowed.Exec(ctx, `CREATE TABLE `+quoted(firstNames.Schema)+`.guard_check (id integer)`); err != nil {
		t.Fatalf("table DDL was blocked: %v", err)
	}
	denied, err := provisioner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer denied.Rollback(ctx)
	if _, err := denied.Exec(ctx, `SET LOCAL ROLE `+quoted(firstNames.DDLRole)); err != nil {
		t.Fatal(err)
	}
	if _, err := denied.Exec(ctx, `CREATE FUNCTION `+quoted(firstNames.Schema)+`.bad() RETURNS integer LANGUAGE sql AS 'SELECT 1'`); err == nil {
		t.Fatal("application DDL role created a function")
	}
	for _, statement := range []string{
		`CREATE TABLE ` + quoted(secondNames.Schema) + `.stolen (id integer)`,
		`CREATE EXTENSION IF NOT EXISTS hstore WITH SCHEMA ` + quoted(firstNames.Schema),
	} {
		attempt, err := provisioner.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := attempt.Exec(ctx, `SET LOCAL ROLE `+quoted(firstNames.DDLRole)); err != nil {
			t.Fatal(err)
		}
		if _, err := attempt.Exec(ctx, statement); err == nil {
			attempt.Rollback(ctx)
			t.Fatalf("application DDL role escaped scope: %s", statement)
		}
		attempt.Rollback(ctx)
	}
	if err := Provision(ctx, provisioner, "unsafe;drop"); err == nil {
		t.Fatal("unsafe application ID accepted")
	}
}

func TestCleanupAppIdempotentAndScoped(t *testing.T) {
	provisioner, admin := provisionPools(t)
	first, second := randomAppID(t), randomAppID(t)
	cleanupApp(t, admin, second)
	ctx := context.Background()
	if err := Provision(ctx, provisioner, first); err != nil {
		t.Fatal(err)
	}
	if err := Provision(ctx, provisioner, second); err != nil {
		t.Fatal(err)
	}
	if err := CleanupApp(ctx, provisioner, first); err != nil {
		t.Fatal(err)
	}
	if err := CleanupApp(ctx, provisioner, first); err != nil {
		t.Fatal("cleanup retry:", err)
	}
	if err := VerifyProvisioned(ctx, provisioner, second); err != nil {
		t.Fatal("other app damaged:", err)
	}
	var exists bool
	names, _ := AppNames(first)
	if err := admin.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1) OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$2)`, names.Schema, names.APIRole).Scan(&exists); err != nil || exists {
		t.Fatalf("cleanup incomplete: %t %v", exists, err)
	}
}

func TestAppStatementTimeoutDoesNotBlockControlDatabase(t *testing.T) {
	provisioner, admin := provisionPools(t)
	control := isolatedControl(t)
	appID := randomAppID(t)
	cleanupApp(t, admin, appID)
	ctx := context.Background()
	if err := Provision(ctx, provisioner, appID); err != nil {
		t.Fatal(err)
	}
	names, _ := AppNames(appID)
	tx, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+quoted(names.APIRole)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SELECT cellapp_internal.pre_request()`); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := tx.Exec(ctx, `SELECT pg_sleep(6)`); err == nil || time.Since(start) >= 6*time.Second {
		t.Fatalf("app statement limit was bypassed: %v", err)
	}
	var one int
	if err := control.QueryRow(ctx, `SELECT 1`).Scan(&one); err != nil || one != 1 {
		t.Fatalf("control query impaired: %d %v", one, err)
	}
}

func isolatedControl(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "app_control_test_" + randomAppID(t)
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+quoted(schema)); err != nil {
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := admin.Exec(cleanup, `DROP SCHEMA `+pgx.Identifier{schema}.Sanitize()+` CASCADE`); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	if err := migrations.Run(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestReconcileReadinessAndRetry(t *testing.T) {
	control := isolatedControl(t)
	provisioner, admin := provisionPools(t)
	appID := randomAppID(t)
	cleanupApp(t, admin, appID)
	ctx := context.Background()
	if _, err := control.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES('owner','test','owner')`); err != nil {
		t.Fatal(err)
	}
	if _, err := control.Exec(ctx, `INSERT INTO app_namespaces(id,owner_id,handle) VALUES($1,'owner',$2)`, randomAppID(t), "n-"+randomAppID(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := control.Exec(ctx, `INSERT INTO apps(id,owner_id,namespace_id,slug,name,key_hash,create_key,create_digest)
    VALUES($1,'owner',(SELECT id FROM app_namespaces WHERE owner_id='owner'),$2,'Test','hash','request','digest')`, appID, "app-"+appID); err != nil {
		t.Fatal(err)
	}
	if err := Reconcile(ctx, control, provisioner, appID, nil); !errors.Is(err, ErrAPIProbeRequired) {
		t.Fatalf("missing probe: %v", err)
	}
	var state string
	if err := control.QueryRow(ctx, `SELECT data_state FROM apps WHERE id=$1`, appID).Scan(&state); err != nil || state != "pending" {
		t.Fatalf("prematurely ready: %q %v", state, err)
	}
	if err := Reconcile(ctx, control, provisioner, appID, func(context.Context, Names) error { return errors.New("cache stale") }); err == nil {
		t.Fatal("failed API probe accepted")
	}
	if err := control.QueryRow(ctx, `SELECT data_state FROM apps WHERE id=$1`, appID).Scan(&state); err != nil || state != "error" {
		t.Fatalf("probe failure state: %q %v", state, err)
	}
	if err := ReconcilePending(ctx, control, provisioner, func(context.Context, Names) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := control.QueryRow(ctx, `SELECT data_state FROM apps WHERE id=$1`, appID).Scan(&state); err != nil || state != "ready" {
		t.Fatalf("ready state: %q %v", state, err)
	}
	if _, err := control.Exec(ctx, `UPDATE apps SET deleted=true WHERE id=$1`, appID); err != nil {
		t.Fatal(err)
	}
	if err := Reconcile(ctx, control, provisioner, appID, func(context.Context, Names) error { return nil }); !errors.Is(err, ErrAppUnavailable) {
		t.Fatalf("deleted application reconciled: %v", err)
	}
}
