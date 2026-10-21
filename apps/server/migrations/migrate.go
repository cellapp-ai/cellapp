package migrations

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed 001_initial.sql
var initial string

//go:embed 002_app_access.sql
var appAccess string

//go:embed 003_app_sessions.sql
var appSessions string

//go:embed 004_app_namespaces.sql
var appNamespaces string

//go:embed 006_owner_profile.sql
var ownerProfile string

//go:embed 007_app_slug_hosts.sql
var appSlugHosts string

//go:embed 008_monthly_visitors.sql
var monthlyVisitors string

//go:embed 009_mcp_agent_access.sql
var mcpAgentAccess string

//go:embed 010_github_profile.sql
var githubProfile string

//go:embed 011_anonymous_app_access.sql
var anonymousAppAccess string

type migration struct {
	version int
	sql     string
}

var controlMigrations = []migration{
	{version: 1, sql: initial},
	{version: 2, sql: appAccess},
	{version: 3, sql: appSessions},
	{version: 4, sql: appNamespaces},
	{version: 6, sql: ownerProfile},
	{version: 7, sql: appSlugHosts},
	{version: 8, sql: monthlyVisitors},
	{version: 9, sql: mcpAgentAccess},
	{version: 10, sql: githubProfile},
	{version: 11, sql: anonymousAppAccess},
}

// Run applies each control migration once, including the initial schema on
// databases created before migration versions were recorded.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(784231)"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version integer PRIMARY KEY,
		checksum text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}
	for _, item := range controlMigrations {
		sum := sha256.Sum256([]byte(item.sql))
		checksum := hex.EncodeToString(sum[:])
		var applied string
		err = tx.QueryRow(ctx, `SELECT checksum FROM schema_migrations WHERE version=$1`, item.version).Scan(&applied)
		switch err {
		case nil:
			if applied != checksum {
				return fmt.Errorf("control migration %d checksum mismatch", item.version)
			}
			continue
		case pgx.ErrNoRows:
		default:
			return err
		}
		if _, err = tx.Exec(ctx, item.sql); err != nil {
			return fmt.Errorf("control migration %d: %w", item.version, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)`, item.version, checksum); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
