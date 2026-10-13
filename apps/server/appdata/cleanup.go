package appdata

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CleanupApp is repeatable after partial deletion. The control database must
// mark the app deleted before this is called.
func CleanupApp(ctx context.Context, pool *pgxpool.Pool, appID string) error {
	names, err := AppNames(appID)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 784233))`, appID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DROP SCHEMA IF EXISTS `+quoted(names.Schema)+` CASCADE`); err != nil {
		return fmt.Errorf("drop application schema: %w", err)
	}
	for _, role := range []string{names.APIRole, names.DDLRole} {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$1)`, role).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			continue
		}
		if _, err := tx.Exec(ctx, `REVOKE USAGE ON SCHEMA cellapp_internal FROM `+quoted(role)); err != nil {
			return err
		}
		function := "pre_request"
		if role == names.DDLRole {
			function = "account_storage"
		}
		if _, err := tx.Exec(ctx, `REVOKE EXECUTE ON FUNCTION cellapp_internal.`+function+`() FROM `+quoted(role)); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DROP ROLE `+quoted(role)); err != nil {
			return fmt.Errorf("drop application role: %w", err)
		}
	}
	for _, table := range []string{"app_versions", "app_storage_tables", "app_storage"} {
		if _, err := tx.Exec(ctx, `DELETE FROM cellapp_internal.`+table+` WHERE app_id=$1`, appID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `NOTIFY pgrst, 'reload config'`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `NOTIFY pgrst, 'reload schema'`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func CleanupPending(ctx context.Context, control, application *pgxpool.Pool) error {
	rows, err := control.Query(ctx, `SELECT id FROM apps WHERE deleted AND data_state='cleanup_pending' ORDER BY id LIMIT 100`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := CleanupApp(ctx, application, id); err != nil {
			return err
		}
		if _, err := control.Exec(ctx, `UPDATE apps SET data_state='cleaned',data_error_code=NULL,data_updated_at=now() WHERE id=$1 AND deleted AND data_state='cleanup_pending'`, id); err != nil {
			return err
		}
	}
	return nil
}
