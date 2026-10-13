package migrations

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func migrationDatabase(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required for isolated PostgreSQL migration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	schema := "migration_test_" + hex.EncodeToString(random[:])
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+pgx.Identifier{schema}.Sanitize()); err != nil {
		admin.Close()
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
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if _, err := admin.Exec(cleanup, `DROP SCHEMA `+pgx.Identifier{schema}.Sanitize()+` CASCADE`); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	return pool, admin
}

func TestRunFreshAndRepeat(t *testing.T) {
	pool, _ := migrationDatabase(t)
	ctx := context.Background()
	if err := Run(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&count); err != nil || count != len(controlMigrations) {
		t.Fatalf("migration count=%d, err=%v", count, err)
	}
	for _, table := range []string{"owners", "apps", "deployments", "access_sessions"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil || !exists {
			t.Fatalf("missing %s: %v", table, err)
		}
	}
}

func TestRunFromLegacySchema(t *testing.T) {
	pool, _ := migrationDatabase(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, initial); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES('existing','test','existing')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO apps(id,owner_id,name,key_hash,create_key,create_digest) VALUES('old-app','existing','Old','hash','request','digest')`); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var visibility, dataState string
	if err := pool.QueryRow(ctx, `SELECT visibility,data_state FROM apps WHERE id='old-app'`).Scan(&visibility, &dataState); err != nil || visibility != "restricted" || dataState != "pending" {
		t.Fatalf("legacy app not upgraded: %q %q %v", visibility, dataState, err)
	}
	var id string
	if err := pool.QueryRow(ctx, `SELECT id FROM owners WHERE subject='existing'`).Scan(&id); err != nil || id != "existing" {
		t.Fatalf("legacy owner lost: %q, %v", id, err)
	}
	var handle, slug string
	if err := pool.QueryRow(ctx, `SELECT n.handle,a.slug FROM apps a JOIN app_namespaces n ON n.id=a.namespace_id WHERE a.id='old-app'`).Scan(&handle, &slug); err != nil || handle == "" || slug != "app-old-app" {
		t.Fatalf("legacy alias not backfilled: %q/%q %v", handle, slug, err)
	}
}

func TestRunFromAppSessionsSchema(t *testing.T) {
	pool, _ := migrationDatabase(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `CREATE TABLE schema_migrations(version integer PRIMARY KEY, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		t.Fatal(err)
	}
	for _, item := range controlMigrations[:3] {
		if _, err := pool.Exec(ctx, item.sql); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256([]byte(item.sql))
		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)`, item.version, hex.EncodeToString(sum[:])); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES('existing','test','old')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO apps(id,owner_id,name,key_hash,create_key,create_digest) VALUES('old-app','existing','Old','hash','request','digest')`); err != nil {
		t.Fatal(err)
	}
	if err := Run(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var handle, slug string
	if err := pool.QueryRow(ctx, `SELECT n.handle,a.slug FROM apps a JOIN app_namespaces n ON n.id=a.namespace_id WHERE a.id='old-app'`).Scan(&handle, &slug); err != nil || handle == "" || slug != "app-old-app" {
		t.Fatalf("v3 upgrade lost application alias: %q/%q %v", handle, slug, err)
	}
	if err := Run(ctx, pool); err != nil {
		t.Fatalf("v4 upgrade is not repeatable: %v", err)
	}
}

func TestRunConcurrent(t *testing.T) {
	pool, _ := migrationDatabase(t)
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errors <- Run(context.Background(), pool)
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM schema_migrations`).Scan(&count); err != nil || count != len(controlMigrations) {
		t.Fatalf("migration count=%d, err=%v", count, err)
	}
}

func TestAppAccessConstraints(t *testing.T) {
	pool, _ := migrationDatabase(t)
	ctx := context.Background()
	if err := Run(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES('one','test','one'),('two','test','two')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_namespaces(id,owner_id,handle) VALUES
		('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','one','one'),
		('bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','two','two')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO apps(id,owner_id,namespace_id,slug,name,key_hash,create_key,create_digest) VALUES
		('app-one','one','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','one','One','hash','request-one','digest'),
		('app-two','two','bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','two','Two','hash','request-two','digest')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_users(id,app_id,platform_user_id) VALUES('scoped-one','app-one','one'),('scoped-two','app-two','one')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_users(id,app_id,platform_user_id) VALUES('duplicate','app-one','one')`); err == nil {
		t.Fatal("duplicate application identity accepted")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_grants(app_id,platform_user_id,source) VALUES('app-one','one','key')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_grants(app_id,platform_user_id,source) VALUES('app-one','one','request')`); err == nil {
		t.Fatal("duplicate grant accepted")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_access_requests(app_id,platform_user_id,status) VALUES('app-one','two','pending')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_access_requests(app_id,platform_user_id,status) VALUES('app-one','two','invalid')`); err == nil {
		t.Fatal("invalid access request state accepted")
	}
	if _, err := pool.Exec(ctx, `UPDATE apps SET visibility='unlisted' WHERE id='app-one'`); err == nil {
		t.Fatal("invalid visibility accepted")
	}
}
