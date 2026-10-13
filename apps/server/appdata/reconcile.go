package appdata

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAppUnavailable = errors.New("application missing or deleted")
var ErrAPIProbeRequired = errors.New("application API readiness probe not configured")

type Probe func(context.Context, Names) error

// Reconcile prepares an application's data schema and records readiness only
// after the API layer can serve it. Cross-database steps are safe to retry.
func Reconcile(ctx context.Context, control, application *pgxpool.Pool, appID string, probe Probe) error {
	names, err := AppNames(appID)
	if err != nil {
		return err
	}
	var deleted bool
	if err := control.QueryRow(ctx, `SELECT deleted FROM apps WHERE id=$1`, appID).Scan(&deleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAppUnavailable
		}
		return err
	}
	if deleted {
		return ErrAppUnavailable
	}
	if err := Provision(ctx, application, appID); err != nil {
		return errors.Join(fmt.Errorf("provision application data: %w", err), markDataError(ctx, control, appID, "provision_failed"))
	}
	if err := VerifyProvisioned(ctx, application, appID); err != nil {
		return errors.Join(err, markDataError(ctx, control, appID, "provision_incomplete"))
	}
	if probe == nil {
		return ErrAPIProbeRequired
	}
	if err := probe(ctx, names); err != nil {
		return errors.Join(fmt.Errorf("application data API not ready: %w", err), markDataError(ctx, control, appID, "api_not_ready"))
	}
	result, err := control.Exec(ctx, `UPDATE apps SET data_state='ready',data_error_code=NULL,data_updated_at=now() WHERE id=$1 AND NOT deleted`, appID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrAppUnavailable
	}
	return nil
}

func markDataError(ctx context.Context, control *pgxpool.Pool, appID, code string) error {
	_, err := control.Exec(ctx, `UPDATE apps SET data_state='error',data_error_code=$2,data_updated_at=now() WHERE id=$1 AND NOT deleted`, appID, code)
	return err
}

// ReconcilePending retries interrupted provisioning in bounded batches.
func ReconcilePending(ctx context.Context, control, application *pgxpool.Pool, probe Probe) error {
	if probe == nil {
		return ErrAPIProbeRequired
	}
	rows, err := control.Query(ctx, `SELECT id FROM apps WHERE NOT deleted AND data_state IN ('pending','error') ORDER BY id LIMIT 100`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var appID string
		if err := rows.Scan(&appID); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, appID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var failures []error
	for _, appID := range ids {
		if err := Reconcile(ctx, control, application, appID, probe); err != nil && !errors.Is(err, ErrAppUnavailable) {
			failures = append(failures, fmt.Errorf("application %s: %w", appID, err))
		}
	}
	return errors.Join(failures...)
}
