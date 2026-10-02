package migrations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testDB(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL required for real PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	b := make([]byte, 8)
	if _, e = rand.Read(b); e != nil {
		t.Fatal(e)
	}
	schema := "mig_" + hex.EncodeToString(b)
	if _, e = admin.Exec(ctx, `CREATE SCHEMA `+schema); e != nil {
		t.Fatal(e)
	}
	pc, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	pc.ConnConfig.RuntimeParams["search_path"] = schema
	db, e := pgxpool.NewWithConfig(ctx, pc)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close(); _, _ = admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`); admin.Close() })
	return db, ctx
}

func columnExists(ctx context.Context, db *pgxpool.Pool, name string) bool {
	var exists bool
	if e := db.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'apps' AND column_name = $1
	)`, name).Scan(&exists); e != nil {
		return false
	}
	return exists
}

func TestMigrationFreshAndRepeatable(t *testing.T) {
	db, ctx := testDB(t)
	if e := Run(ctx, db); e != nil {
		t.Fatal(e)
	}
	if e := Run(ctx, db); e != nil {
		t.Fatal(e)
	}
	var versions int
	if e := db.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&versions); e != nil || versions != 2 {
		t.Fatal(versions, e)
	}
	if !columnExists(ctx, db, "data_provider") || !columnExists(ctx, db, "data_url") || !columnExists(ctx, db, "data_anon_key") {
		t.Fatal("data columns missing")
	}
}

func TestMigrationUpgradeFromInitial(t *testing.T) {
	db, ctx := testDB(t)
	if _, e := db.Exec(ctx, migration001); e != nil {
		t.Fatal(e)
	}
	if columnExists(ctx, db, "data_provider") {
		t.Fatal("initial schema already had data columns")
	}
	if e := Run(ctx, db); e != nil {
		t.Fatal(e)
	}
	if e := Run(ctx, db); e != nil {
		t.Fatal(e)
	}
	if !columnExists(ctx, db, "data_anon_key") {
		t.Fatal("upgrade did not add data columns")
	}
	if _, e := db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES('o','iss','sub')`); e != nil {
		t.Fatal(e)
	}
	if _, e := db.Exec(ctx, `INSERT INTO apps(id,owner_id,name,key_hash,create_key,create_digest) VALUES('a','o','n','h','k','d')`); e != nil {
		t.Fatal(e)
	}
	var provider *string
	if e := db.QueryRow(ctx, `SELECT data_provider FROM apps WHERE id='a'`).Scan(&provider); e != nil || provider != nil {
		t.Fatal(provider, e)
	}
}
