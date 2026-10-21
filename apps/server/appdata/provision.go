package appdata

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var appIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

const provisionerRole = "cellapp_appdata_provisioner"
const authenticatorRole = "cellapp_appdata_authenticator"

type Names struct {
	Schema        string
	APIRole       string
	AnonymousRole string
	DDLRole       string
}

func AppNames(appID string) (Names, error) {
	if !appIDPattern.MatchString(appID) {
		return Names{}, fmt.Errorf("invalid application ID")
	}
	return Names{Schema: "app_" + appID, APIRole: "app_api_" + appID, AnonymousRole: "app_anon_" + appID, DDLRole: "app_ddl_" + appID}, nil
}

func quoted(name string) string { return pgx.Identifier{name}.Sanitize() }

// Provision is safe to retry after an interrupted app creation. The caller
// must not mark the app ready until the API schema cache has also been probed.
func Provision(ctx context.Context, pool *pgxpool.Pool, appID string) error {
	names, err := AppNames(appID)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 784233))`, appID); err != nil {
		return err
	}
	for _, name := range []string{names.DDLRole, names.APIRole, names.AnonymousRole} {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=$1)`, name).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			if _, err = tx.Exec(ctx, `CREATE ROLE `+quoted(name)+` NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`); err != nil {
				return fmt.Errorf("create app role: %w", err)
			}
		}
	}
	if _, err = tx.Exec(ctx, `GRANT `+quoted(names.DDLRole)+` TO `+quoted(provisionerRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT `+quoted(names.APIRole)+` TO `+quoted(authenticatorRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT `+quoted(names.AnonymousRole)+` TO `+quoted(authenticatorRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+quoted(names.Schema)+` AUTHORIZATION `+quoted(names.DDLRole)); err != nil {
		return err
	}
	var owner string
	if err = tx.QueryRow(ctx, `SELECT r.rolname FROM pg_namespace n JOIN pg_roles r ON r.oid=n.nspowner WHERE n.nspname=$1`, names.Schema).Scan(&owner); err != nil {
		return err
	}
	if owner != names.DDLRole {
		return fmt.Errorf("application schema has unexpected owner")
	}
	if _, err = tx.Exec(ctx, `REVOKE ALL ON SCHEMA `+quoted(names.Schema)+` FROM PUBLIC`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT USAGE ON SCHEMA `+quoted(names.Schema)+` TO `+quoted(names.APIRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT USAGE ON SCHEMA `+quoted(names.Schema)+` TO `+quoted(names.AnonymousRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT USAGE ON SCHEMA cellapp_internal TO `+quoted(names.APIRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT EXECUTE ON FUNCTION cellapp_internal.pre_request() TO `+quoted(names.APIRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT USAGE ON SCHEMA cellapp_internal TO `+quoted(names.AnonymousRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT EXECUTE ON FUNCTION cellapp_internal.pre_request() TO `+quoted(names.AnonymousRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT USAGE ON SCHEMA cellapp_internal TO `+quoted(names.DDLRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT EXECUTE ON FUNCTION cellapp_internal.account_storage() TO `+quoted(names.DDLRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT USAGE ON SCHEMA `+quoted(names.Schema)+` TO `+quoted(authenticatorRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+quoted(names.DDLRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+quoted(names.Schema)+`._cellapp_ready (version integer PRIMARY KEY)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `RESET ROLE`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `GRANT SELECT ON `+quoted(names.Schema)+`._cellapp_ready TO `+quoted(names.APIRole)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO cellapp_internal.app_storage(app_id) VALUES($1) ON CONFLICT DO NOTHING`, appID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `NOTIFY pgrst, 'reload config'`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `NOTIFY pgrst, 'reload schema'`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func VerifyProvisioned(ctx context.Context, pool *pgxpool.Pool, appID string) error {
	names, err := AppNames(appID)
	if err != nil {
		return err
	}
	var owner string
	err = pool.QueryRow(ctx, `SELECT r.rolname FROM pg_namespace n JOIN pg_roles r ON r.oid=n.nspowner WHERE n.nspname=$1`, names.Schema).Scan(&owner)
	if err != nil {
		return fmt.Errorf("application schema missing: %w", err)
	}
	if owner != names.DDLRole {
		return fmt.Errorf("application schema owner mismatch")
	}
	var allowed bool
	if err = pool.QueryRow(ctx, `SELECT has_schema_privilege($1,$2,'USAGE')`, names.APIRole, names.Schema).Scan(&allowed); err != nil || !allowed {
		return fmt.Errorf("application API role lacks schema access: %w", err)
	}
	if err = pool.QueryRow(ctx, `SELECT has_schema_privilege($1,$2,'USAGE')`, names.AnonymousRole, names.Schema).Scan(&allowed); err != nil || !allowed {
		return fmt.Errorf("application anonymous role lacks schema access: %w", err)
	}
	return nil
}
