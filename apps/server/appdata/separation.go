package appdata

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureSeparate checks connected database names, including connections through
// different proxies or network addresses. Deployments use distinct names even
// when the databases reside on different PostgreSQL clusters.
func EnsureSeparate(ctx context.Context, control, application *pgxpool.Pool) error {
	read := func(pool *pgxpool.Pool) (string, error) {
		var database string
		err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&database)
		return database, err
	}
	a, err := read(control)
	if err != nil {
		return fmt.Errorf("control database identity: %w", err)
	}
	b, err := read(application)
	if err != nil {
		return fmt.Errorf("application database identity: %w", err)
	}
	if a == b {
		return fmt.Errorf("application data must use a database distinct from control data")
	}
	return nil
}
