package appdata

import (
	"context"
	"strings"
	"sync"
	"testing"
)

func notesManifest(version int) Manifest {
	return Manifest{Version: version, Tables: []Table{{
		Name: "notes",
		Columns: []Column{
			{Name: "id", Type: "uuid"},
			{Name: "owner_id", Type: "text"},
			{Name: "body", Type: "text"},
		},
		PrimaryKey: []string{"id"},
		Indexes:    []Index{{Columns: []string{"owner_id"}}},
		Policies: Policies{
			Select: "own-rows", Insert: "own-rows", Update: "own-rows", Delete: "own-rows", OwnerColumn: "owner_id",
		},
	}}}
}

func TestManifestValidationAndPlan(t *testing.T) {
	valid := notesManifest(1)
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Manifest){
		func(m *Manifest) { m.Tables[0].Name = "notes;drop" },
		func(m *Manifest) { m.Tables[0].Columns[0].Type = "serial" },
		func(m *Manifest) { m.Tables[0].Policies.OwnerColumn = "body2" },
		func(m *Manifest) { m.Tables[0].Indexes[0].Columns = []string{"missing"} },
		func(m *Manifest) { m.Tables[0].Name = "rpc" },
		func(m *Manifest) { m.Tables[0].Columns[0].Name = "xmin" },
		func(m *Manifest) {
			m.Tables = append(m.Tables, Table{Name: indexName("notes", 0), Columns: []Column{{Name: "id", Type: "uuid"}}, PrimaryKey: []string{"id"}})
		},
	} {
		candidate := notesManifest(1)
		change(&candidate)
		if err := candidate.Validate(); err == nil {
			t.Fatal("invalid manifest accepted")
		}
	}
	for _, tableName := range []string{"users", "applications", "resources"} {
		candidate := notesManifest(1)
		candidate.Tables[0].Name = tableName
		if err := candidate.Validate(); err != nil {
			t.Fatalf("application table %q should not collide with control data: %v", tableName, err)
		}
	}
	if _, err := PlanManifest(nil, notesManifest(2)); err == nil {
		t.Fatal("initial version gap accepted")
	}
	if plan, err := PlanManifest(nil, valid); err != nil || len(plan.Operations) != 1 {
		t.Fatalf("initial plan: %+v %v", plan, err)
	}
	if plan, err := PlanManifest(&valid, valid); err != nil || !plan.AlreadyApplied {
		t.Fatalf("idempotent plan: %+v %v", plan, err)
	}
	changed := notesManifest(1)
	changed.Tables[0].Policies.Select = "none"
	if _, err := PlanManifest(&valid, changed); err == nil {
		t.Fatal("changed same-version manifest accepted")
	}
	dropped := Manifest{Version: 2}
	if _, err := PlanManifest(&valid, dropped); err == nil || !strings.Contains(err.Error(), "destructive") {
		t.Fatalf("destructive plan: %v", err)
	}
	dropped.AllowDestructive = true
	if plan, err := PlanManifest(&valid, dropped); err != nil || !plan.RequiresDestructive {
		t.Fatalf("explicit drop plan: %+v %v", plan, err)
	}
}

func TestManifestApplyLedgerAndOwnRows(t *testing.T) {
	provisioner, admin := provisionPools(t)
	appID := randomAppID(t)
	cleanupApp(t, admin, appID)
	ctx := context.Background()
	if err := Provision(ctx, provisioner, appID); err != nil {
		t.Fatal(err)
	}
	names, _ := AppNames(appID)
	first := notesManifest(1)
	plan, err := Apply(ctx, provisioner, appID, first)
	if err != nil || plan.AlreadyApplied {
		t.Fatalf("apply: %+v %v", plan, err)
	}
	if plan, err = Apply(ctx, provisioner, appID, first); err != nil || !plan.AlreadyApplied {
		t.Fatalf("retry: %+v %v", plan, err)
	}
	latest, digest, err := Latest(ctx, provisioner, appID)
	if err != nil || latest == nil || latest.Version != 1 || digest != plan.Digest {
		t.Fatalf("ledger: %+v %q %v", latest, digest, err)
	}
	changed := notesManifest(1)
	changed.Tables[0].Policies.Select = "none"
	if _, err := Apply(ctx, provisioner, appID, changed); err == nil {
		t.Fatal("changed same-version apply accepted")
	}
	if _, err := admin.Exec(ctx, `INSERT INTO `+quoted(names.Schema)+`.notes(id,owner_id,body) VALUES
		('00000000-0000-0000-0000-000000000001','u1','one'),
		('00000000-0000-0000-0000-000000000002','u2','two')`); err != nil {
		t.Fatal(err)
	}
	user, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer user.Rollback(ctx)
	if _, err := user.Exec(ctx, `SET LOCAL ROLE `+quoted(names.APIRole)); err != nil {
		t.Fatal(err)
	}
	if _, err := user.Exec(ctx, `SELECT set_config('request.jwt.claims','{"app_user_id":"u1"}',true)`); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := user.QueryRow(ctx, `SELECT count(*) FROM `+quoted(names.Schema)+`.notes`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("row policy: %d %v", count, err)
	}
	if result, err := user.Exec(ctx, `UPDATE `+quoted(names.Schema)+`.notes SET body='stolen' WHERE owner_id='u2'`); err != nil || result.RowsAffected() != 0 {
		t.Fatalf("cross-user update: %v %v", result, err)
	}
	if _, err := user.Exec(ctx, `INSERT INTO `+quoted(names.Schema)+`.notes(id,owner_id,body) VALUES('00000000-0000-0000-0000-000000000003','u2','forged')`); err == nil {
		t.Fatal("forged owner insert accepted")
	}
	if err := user.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `DELETE FROM `+quoted(names.Schema)+`.notes WHERE owner_id='u1'`); err != nil {
		t.Fatalf("storage accounting delete: %v", err)
	}
	var used int64
	if err := admin.QueryRow(ctx, `SELECT bytes FROM cellapp_internal.app_storage WHERE app_id=$1`, appID).Scan(&used); err != nil {
		t.Fatal(err)
	}
	limit := used + 400
	if _, err := admin.Exec(ctx, `UPDATE cellapp_internal.app_storage SET limit_bytes=$2 WHERE app_id=$1`, appID, limit); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	var writes sync.WaitGroup
	for _, rowID := range []string{"00000000-0000-0000-0000-000000000004", "00000000-0000-0000-0000-000000000005"} {
		writes.Add(1)
		go func(rowID string) {
			defer writes.Done()
			_, err := admin.Exec(ctx, `INSERT INTO `+quoted(names.Schema)+`.notes(id,owner_id,body) VALUES($1,'u2',$2)`, rowID, strings.Repeat("x", 100))
			results <- err
		}(rowID)
	}
	writes.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("storage limit allowed %d concurrent writes", successes)
	}
	if err := admin.QueryRow(ctx, `SELECT bytes FROM cellapp_internal.app_storage WHERE app_id=$1`, appID).Scan(&used); err != nil || used > limit {
		t.Fatalf("storage accounting: %d %v", used, err)
	}
	second := notesManifest(2)
	second.Tables[0].Columns = append(second.Tables[0].Columns, Column{Name: "label", Type: "text", Nullable: true})
	if _, err := Apply(ctx, provisioner, appID, second); err != nil {
		t.Fatal(err)
	}
	third := notesManifest(3)
	third.Tables[0].Columns = append(third.Tables[0].Columns, Column{Name: "label", Type: "text", Nullable: true}, Column{Name: "required", Type: "text"})
	if _, err := Apply(ctx, provisioner, appID, third); err == nil {
		t.Fatal("non-null column on populated table succeeded")
	}
	latest, _, err = Latest(ctx, provisioner, appID)
	if err != nil || latest.Version != 2 {
		t.Fatalf("failed migration advanced ledger: %+v %v", latest, err)
	}
}

func TestDroppedColumnReconcilesStorageLedger(t *testing.T) {
	provisioner, admin := provisionPools(t)
	appID := randomAppID(t)
	cleanupApp(t, admin, appID)
	ctx := context.Background()
	if err := Provision(ctx, provisioner, appID); err != nil {
		t.Fatal(err)
	}
	first := notesManifest(1)
	if _, err := Apply(ctx, provisioner, appID, first); err != nil {
		t.Fatal(err)
	}
	names, _ := AppNames(appID)
	if _, err := admin.Exec(ctx, `INSERT INTO `+quoted(names.Schema)+`.notes(id,owner_id,body) VALUES('00000000-0000-0000-0000-000000000001','u1',$1)`, strings.Repeat("x", 1000)); err != nil {
		t.Fatal(err)
	}
	var before, after int64
	if err := admin.QueryRow(ctx, `SELECT bytes FROM cellapp_internal.app_storage WHERE app_id=$1`, appID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	second := notesManifest(2)
	second.AllowDestructive = true
	second.Tables[0].Columns = second.Tables[0].Columns[:2]
	if _, err := Apply(ctx, provisioner, appID, second); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `SELECT bytes FROM cellapp_internal.app_storage WHERE app_id=$1`, appID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after >= before || after <= 0 {
		t.Fatalf("column drop did not reconcile storage: before=%d after=%d", before, after)
	}
	if _, err := admin.Exec(ctx, `DELETE FROM `+quoted(names.Schema)+`.notes`); err != nil {
		t.Fatal(err)
	}
	if err := admin.QueryRow(ctx, `SELECT bytes FROM cellapp_internal.app_storage WHERE app_id=$1`, appID).Scan(&after); err != nil || after != 0 {
		t.Fatalf("storage after delete: %d %v", after, err)
	}
}

func TestConcurrentManifestApplyIsIdempotent(t *testing.T) {
	provisioner, admin := provisionPools(t)
	appID := randomAppID(t)
	cleanupApp(t, admin, appID)
	ctx := context.Background()
	if err := Provision(ctx, provisioner, appID); err != nil {
		t.Fatal(err)
	}
	manifest := notesManifest(1)
	type result struct {
		plan Plan
		err  error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() { plan, err := Apply(ctx, provisioner, appID, manifest); results <- result{plan, err} }()
	}
	first, second := <-results, <-results
	if first.err != nil || second.err != nil || first.plan.AlreadyApplied == second.plan.AlreadyApplied {
		t.Fatalf("concurrent apply: %+v %+v", first, second)
	}
	latest, _, err := Latest(ctx, provisioner, appID)
	if err != nil || latest == nil || latest.Version != 1 {
		t.Fatalf("ledger after concurrent apply: %+v %v", latest, err)
	}
}

func TestManifestDefaultDeny(t *testing.T) {
	provisioner, admin := provisionPools(t)
	appID := randomAppID(t)
	cleanupApp(t, admin, appID)
	ctx := context.Background()
	if err := Provision(ctx, provisioner, appID); err != nil {
		t.Fatal(err)
	}
	manifest := notesManifest(1)
	manifest.Tables[0].Policies = Policies{}
	if _, err := Apply(ctx, provisioner, appID, manifest); err != nil {
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
	if _, err := tx.Exec(ctx, `SELECT * FROM `+quoted(names.Schema)+`.notes`); err == nil {
		t.Fatal("default-denied table was readable")
	}
}

func TestSameUserTableNameInDifferentApplications(t *testing.T) {
	provisioner, admin := provisionPools(t)
	first, second := randomAppID(t), randomAppID(t)
	cleanupApp(t, admin, first)
	cleanupApp(t, admin, second)
	ctx := context.Background()
	manifest := Manifest{Version: 1, Tables: []Table{{
		Name: "users", Columns: []Column{{Name: "id", Type: "text"}},
		PrimaryKey: []string{"id"}, Policies: Policies{},
	}}}
	for _, appID := range []string{first, second} {
		if err := Provision(ctx, provisioner, appID); err != nil {
			t.Fatal(err)
		}
		if _, err := Apply(ctx, provisioner, appID, manifest); err != nil {
			t.Fatalf("apply users table in %s: %v", appID, err)
		}
		names, _ := AppNames(appID)
		if _, err := admin.Exec(ctx, `INSERT INTO `+quoted(names.Schema)+`.users(id) VALUES($1)`, appID); err != nil {
			t.Fatal(err)
		}
	}
	for _, appID := range []string{first, second} {
		names, _ := AppNames(appID)
		var id string
		if err := admin.QueryRow(ctx, `SELECT id FROM `+quoted(names.Schema)+`.users`).Scan(&id); err != nil || id != appID {
			t.Fatalf("table name leaked across applications: got %q, %v", id, err)
		}
	}
}
