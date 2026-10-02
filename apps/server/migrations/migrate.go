package migrations

import (
	"context"
	"fmt"

	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 001_initial.sql
var migration001 string

//go:embed 002_app_data.sql
var migration002 string

type migration struct {
	version int
	sql     string
}

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(784231)"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version integer PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); e != nil {
		return e
	}
	var hasApps bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		WHERE table_schema = current_schema() AND table_name = 'apps'
	)`).Scan(&hasApps); e != nil {
		return e
	}
	if hasApps {
		if _, e = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES (1) ON CONFLICT DO NOTHING`); e != nil {
			return e
		}
	}
	for _, item := range []migration{{1, migration001}, {2, migration002}} {
		var applied bool
		if e = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version=$1)`, item.version).Scan(&applied); e != nil {
			return e
		}
		if applied {
			continue
		}
		if _, e = tx.Exec(ctx, item.sql); e != nil {
			return fmt.Errorf("migration %d: %w", item.version, e)
		}
		if _, e = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, item.version); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
