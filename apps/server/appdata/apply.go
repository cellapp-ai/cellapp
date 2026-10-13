package appdata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Latest(ctx context.Context, pool *pgxpool.Pool, appID string) (*Manifest, string, error) {
	if _, err := AppNames(appID); err != nil {
		return nil, "", err
	}
	var raw []byte
	var digest string
	err := pool.QueryRow(ctx, `SELECT manifest,digest FROM cellapp_internal.app_versions WHERE app_id=$1 ORDER BY version DESC LIMIT 1`, appID).Scan(&raw, &digest)
	if err == pgx.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, "", err
	}
	return &manifest, digest, nil
}

func Apply(ctx context.Context, pool *pgxpool.Pool, appID string, next Manifest) (Plan, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	names, err := AppNames(appID)
	if err != nil {
		return Plan{}, err
	}
	if err := next.Validate(); err != nil {
		return Plan{}, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Plan{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SET LOCAL statement_timeout = '10000'`); err != nil {
		return Plan{}, err
	}
	if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '2000'`); err != nil {
		return Plan{}, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 784234))`, appID); err != nil {
		return Plan{}, err
	}
	var old Manifest
	var raw []byte
	var oldDigest string
	err = tx.QueryRow(ctx, `SELECT manifest,digest FROM cellapp_internal.app_versions WHERE app_id=$1 ORDER BY version DESC LIMIT 1`, appID).Scan(&raw, &oldDigest)
	var previous *Manifest
	switch err {
	case nil:
		if err := json.Unmarshal(raw, &old); err != nil {
			return Plan{}, err
		}
		previous = &old
	case pgx.ErrNoRows:
	default:
		return Plan{}, err
	}
	plan, err := PlanManifest(previous, next)
	if err != nil {
		return Plan{}, err
	}
	if plan.AlreadyApplied {
		if oldDigest != plan.Digest {
			return Plan{}, fmt.Errorf("stored manifest digest mismatch")
		}
		return plan, nil
	}
	if err := VerifyProvisioned(ctx, pool, appID); err != nil {
		return Plan{}, err
	}
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+quoted(names.DDLRole)); err != nil {
		return Plan{}, err
	}
	if err := applyDDL(ctx, tx, names, previous, next); err != nil {
		return Plan{}, err
	}
	if _, err := tx.Exec(ctx, `RESET ROLE`); err != nil {
		return Plan{}, err
	}
	newTables := map[string]bool{}
	for _, table := range next.Tables {
		newTables[table.Name] = true
		qualified := quoted(names.Schema) + "." + quoted(table.Name)
		if _, err := tx.Exec(ctx, `INSERT INTO cellapp_internal.app_storage_tables(app_id,table_name) VALUES($1,$2) ON CONFLICT DO NOTHING`, appID, table.Name); err != nil {
			return Plan{}, err
		}
		if _, err := tx.Exec(ctx, `CREATE OR REPLACE TRIGGER _cellapp_storage BEFORE INSERT OR UPDATE OR DELETE ON `+qualified+
			` FOR EACH ROW EXECUTE FUNCTION cellapp_internal.account_storage()`); err != nil {
			return Plan{}, err
		}
		if previous != nil {
			var old *Table
			for i := range previous.Tables {
				if previous.Tables[i].Name == table.Name {
					old = &previous.Tables[i]
					break
				}
			}
			if old != nil && len(old.Columns) > len(table.Columns) {
				var used, actual int64
				if err := tx.QueryRow(ctx, `SELECT bytes FROM cellapp_internal.app_storage_tables WHERE app_id=$1 AND table_name=$2 FOR UPDATE`, appID, table.Name).Scan(&used); err != nil {
					return Plan{}, err
				}
				// ALTER TABLE already holds an exclusive lock. Temporarily disable RLS
				// only in this transaction so the accounting scan sees every row.
				if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+quoted(names.DDLRole)); err != nil {
					return Plan{}, err
				}
				if _, err := tx.Exec(ctx, `ALTER TABLE `+qualified+` DISABLE ROW LEVEL SECURITY`); err != nil {
					return Plan{}, err
				}
				if _, err := tx.Exec(ctx, `RESET ROLE`); err != nil {
					return Plan{}, err
				}
				if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(octet_length(to_jsonb(t)::text)+128),0) FROM `+qualified+` AS t`).Scan(&actual); err != nil {
					return Plan{}, err
				}
				if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+quoted(names.DDLRole)); err != nil {
					return Plan{}, err
				}
				if _, err := tx.Exec(ctx, `ALTER TABLE `+qualified+` ENABLE ROW LEVEL SECURITY`); err != nil {
					return Plan{}, err
				}
				if _, err := tx.Exec(ctx, `ALTER TABLE `+qualified+` FORCE ROW LEVEL SECURITY`); err != nil {
					return Plan{}, err
				}
				if _, err := tx.Exec(ctx, `RESET ROLE`); err != nil {
					return Plan{}, err
				}
				if _, err := tx.Exec(ctx, `UPDATE cellapp_internal.app_storage_tables SET bytes=$3 WHERE app_id=$1 AND table_name=$2`, appID, table.Name, actual); err != nil {
					return Plan{}, err
				}
				if _, err := tx.Exec(ctx, `UPDATE cellapp_internal.app_storage SET bytes=bytes+$2 WHERE app_id=$1`, appID, actual-used); err != nil {
					return Plan{}, err
				}
			}
		}
		if _, err := tx.Exec(ctx, `REVOKE ALL ON TABLE `+qualified+` FROM `+quoted(names.APIRole)); err != nil {
			return Plan{}, err
		}
		for _, operation := range []struct{ mode, privilege string }{
			{table.Policies.Select, "SELECT"}, {table.Policies.Insert, "INSERT"},
			{table.Policies.Update, "UPDATE"}, {table.Policies.Delete, "DELETE"},
		} {
			if operation.mode != "" && operation.mode != "none" {
				if _, err := tx.Exec(ctx, `GRANT `+operation.privilege+` ON TABLE `+qualified+` TO `+quoted(names.APIRole)); err != nil {
					return Plan{}, err
				}
			}
		}
	}
	if previous != nil {
		for _, oldTable := range previous.Tables {
			if newTables[oldTable.Name] {
				continue
			}
			if _, err := tx.Exec(ctx, `WITH removed AS (
				DELETE FROM cellapp_internal.app_storage_tables WHERE app_id=$1 AND table_name=$2 RETURNING bytes
			) UPDATE cellapp_internal.app_storage SET bytes=bytes-COALESCE((SELECT sum(bytes) FROM removed),0) WHERE app_id=$1`, appID, oldTable.Name); err != nil {
				return Plan{}, err
			}
		}
	}
	content, err := json.Marshal(next)
	if err != nil {
		return Plan{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO cellapp_internal.app_versions(app_id,version,digest,manifest) VALUES($1,$2,$3,$4)`, appID, next.Version, plan.Digest, content); err != nil {
		return Plan{}, err
	}
	if _, err := tx.Exec(ctx, `NOTIFY pgrst, 'reload schema'`); err != nil {
		return Plan{}, err
	}
	return plan, tx.Commit(ctx)
}

func applyDDL(ctx context.Context, tx pgx.Tx, names Names, previous *Manifest, next Manifest) error {
	oldTables := map[string]Table{}
	newTables := map[string]Table{}
	if previous != nil {
		for _, table := range previous.Tables {
			oldTables[table.Name] = table
		}
	}
	for _, table := range next.Tables {
		newTables[table.Name] = table
		qualified := quoted(names.Schema) + "." + quoted(table.Name)
		old, exists := oldTables[table.Name]
		if !exists {
			parts := make([]string, 0, len(table.Columns)+1)
			for _, column := range table.Columns {
				parts = append(parts, columnDefinition(column))
			}
			keys := make([]string, len(table.PrimaryKey))
			for i, key := range table.PrimaryKey {
				keys[i] = quoted(key)
			}
			parts = append(parts, "PRIMARY KEY ("+strings.Join(keys, ",")+")")
			if _, err := tx.Exec(ctx, `CREATE TABLE `+qualified+` (`+strings.Join(parts, ",")+`)`); err != nil {
				return err
			}
		} else {
			oldColumns := map[string]bool{}
			newColumns := map[string]bool{}
			for _, column := range old.Columns {
				oldColumns[column.Name] = true
			}
			for _, column := range table.Columns {
				newColumns[column.Name] = true
				if !oldColumns[column.Name] {
					if _, err := tx.Exec(ctx, `ALTER TABLE `+qualified+` ADD COLUMN `+columnDefinition(column)); err != nil {
						return err
					}
				}
			}
			for _, column := range old.Columns {
				if !newColumns[column.Name] {
					if _, err := tx.Exec(ctx, `ALTER TABLE `+qualified+` DROP COLUMN `+quoted(column.Name)); err != nil {
						return err
					}
				}
			}
			for i := range old.Indexes {
				if _, err := tx.Exec(ctx, `DROP INDEX IF EXISTS `+quoted(names.Schema)+`.`+quoted(indexName(table.Name, i))); err != nil {
					return err
				}
			}
		}
		for i, index := range table.Indexes {
			columns := make([]string, len(index.Columns))
			for j, column := range index.Columns {
				columns[j] = quoted(column)
			}
			unique := ""
			if index.Unique {
				unique = "UNIQUE "
			}
			if _, err := tx.Exec(ctx, `CREATE `+unique+`INDEX `+quoted(indexName(table.Name, i))+` ON `+qualified+` (`+strings.Join(columns, ",")+`)`); err != nil {
				return err
			}
		}
		if err := setPolicies(ctx, tx, qualified, table.Policies); err != nil {
			return err
		}
	}
	for _, old := range oldTables {
		if _, exists := newTables[old.Name]; !exists {
			if _, err := tx.Exec(ctx, `DROP TABLE `+quoted(names.Schema)+`.`+quoted(old.Name)); err != nil {
				return err
			}
		}
	}
	return nil
}

func columnDefinition(column Column) string {
	definition := quoted(column.Name) + " " + column.Type
	if !column.Nullable {
		definition += " NOT NULL"
	}
	return definition
}

func indexName(table string, position int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", table, position)))
	return "i_" + table[:min(len(table), 40)] + "_" + hex.EncodeToString(sum[:4])
}

func setPolicies(ctx context.Context, tx pgx.Tx, qualified string, p Policies) error {
	if _, err := tx.Exec(ctx, `ALTER TABLE `+qualified+` ENABLE ROW LEVEL SECURITY`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `ALTER TABLE `+qualified+` FORCE ROW LEVEL SECURITY`); err != nil {
		return err
	}
	for _, item := range []struct{ name, operation, mode string }{
		{"_cellapp_select", "SELECT", p.Select}, {"_cellapp_insert", "INSERT", p.Insert},
		{"_cellapp_update", "UPDATE", p.Update}, {"_cellapp_delete", "DELETE", p.Delete},
	} {
		if _, err := tx.Exec(ctx, `DROP POLICY IF EXISTS `+quoted(item.name)+` ON `+qualified); err != nil {
			return err
		}
		if item.mode == "" || item.mode == "none" {
			continue
		}
		expression := "true"
		if item.mode == "own-rows" {
			expression = quoted(p.OwnerColumn) + ` = (NULLIF(current_setting('request.jwt.claims',true),'')::jsonb ->> 'app_user_id')`
		}
		statement := `CREATE POLICY ` + quoted(item.name) + ` ON ` + qualified + ` FOR ` + item.operation
		if item.operation != "INSERT" {
			statement += ` USING (` + expression + `)`
		}
		if item.operation == "INSERT" || item.operation == "UPDATE" {
			statement += ` WITH CHECK (` + expression + `)`
		}
		if _, err := tx.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
