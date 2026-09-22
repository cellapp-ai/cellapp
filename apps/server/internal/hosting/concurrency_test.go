package hosting

import (
	"context"
	"encoding/json"
	"net/url"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestIntegrationConcurrentQuotasAndIdempotency(t *testing.T) {
	f := setup(t)
	f.server.Config.Limits.Apps = 1
	type answer struct {
		status int
		body   []byte
	}
	results := make(chan answer, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := f.request(t, "POST", "/apps", map[string]any{"name": "race", "key": token(), "requestId": id()}, f.token, "control.localhost:8443", "")
			results <- answer{w.Code, w.Body.Bytes()}
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	var app App
	for r := range results {
		if r.status == 201 {
			successes++
			if e := json.Unmarshal(r.body, &app); e != nil {
				t.Fatal(e)
			}
		} else if r.status != 429 {
			t.Fatalf("unexpected race response %d: %s", r.status, r.body)
		}
	}
	if successes != 1 {
		t.Fatalf("created %d apps under quota 1", successes)
	}
	body := map[string]any{"requestId": id(), "baseVersion": nil, "spa": false, "manifest": []Entry{{"index.html", 2, digest([]byte("ok"))}}}
	path := "/apps/" + app.ID + "/deployments"
	first := response[map[string]string](t, f.request(t, "POST", path, body, f.token, "control.localhost:8443", ""), 201)
	again := response[map[string]string](t, f.request(t, "POST", path, body, f.token, "control.localhost:8443", ""), 201)
	if first["id"] != again["id"] {
		t.Fatal("idempotency did not reuse deployment")
	}
	body["spa"] = true
	response[any](t, f.request(t, "POST", path, body, f.token, "control.localhost:8443", ""), 409)
	response[any](t, f.request(t, "PUT", path+"/"+first["id"]+"/files/0", []byte("oversized"), f.token, "control.localhost:8443", ""), 400)
	response[any](t, f.request(t, "POST", path+"/"+first["id"]+"/publish", map[string]any{}, f.token, "control.localhost:8443", ""), 409)
	f.server.Config.Limits.StorageBytes = 2
	body["requestId"] = id()
	body["spa"] = false
	response[any](t, f.request(t, "POST", path, body, f.token, "control.localhost:8443", ""), 429)
	_, e := f.db.Exec(context.Background(), `UPDATE deployments SET created_at=now()-interval '2 hours'`)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.server.Cleanup(context.Background()); e != nil {
		t.Fatal(e)
	}
	response[any](t, f.request(t, "POST", path, body, f.token, "control.localhost:8443", ""), 201)
}

func TestIntegrationTrafficRaceAndCleanupRetry(t *testing.T) {
	f := setup(t)
	a, key := f.create(t)
	d, _ := f.stage(t, a, "ok")
	f.uploadAndPublish(t, a, d, "ok")
	session := f.unlock(t, a, key)
	u, _ := url.Parse(a.URL)
	f.server.Config.Limits.TrafficBytes = 2
	statuses := make(chan int, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); statuses <- f.request(t, "GET", "/", nil, "", u.Host, session).Code }()
	}
	wg.Wait()
	close(statuses)
	success := 0
	for status := range statuses {
		if status == 200 {
			success++
		} else if status != 429 {
			t.Fatal(status)
		}
	}
	if success != 1 {
		t.Fatal("traffic quota raced", success)
	}
	response[any](t, f.request(t, "DELETE", "/apps/"+a.ID, nil, f.token, "control.localhost:8443", ""), 200)
	f.storage.failDelete = true
	if e := f.server.Cleanup(context.Background()); e == nil {
		t.Fatal("storage failure ignored")
	}
	var size int64
	if e := f.db.QueryRow(context.Background(), `SELECT bytes FROM deployments WHERE id=$1`, d).Scan(&size); e != nil {
		t.Fatal(e)
	}
	if size == 0 {
		t.Fatal("quota released before object deletion")
	}
	f.storage.failDelete = false
	if e := f.server.Cleanup(context.Background()); e != nil {
		t.Fatal(e)
	}
	if e := f.db.QueryRow(context.Background(), `SELECT bytes FROM deployments WHERE id=$1`, d).Scan(&size); e != nil || size != 0 {
		t.Fatal("cleanup failed", size, e)
	}
}

func TestIntegrationDatabaseConstraintsAndRollback(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	e := f.server.transaction(ctx, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','owner')`, id())
		return e
	})
	if e == nil {
		t.Fatal("identity uniqueness missing")
	}
	rollbackID := id()
	e = f.server.transaction(ctx, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test','rollback')`, rollbackID); e != nil {
			return e
		}
		return context.Canceled
	})
	if e == nil {
		t.Fatal("expected rollback")
	}
	var count int
	if e = f.db.QueryRow(ctx, `SELECT count(*) FROM owners WHERE id=$1`, rollbackID).Scan(&count); e != nil || count != 0 {
		t.Fatal("transaction not rolled back")
	}
}

func TestIntegrationConcurrentDeviceClaim(t *testing.T) {
	f := setup(t)
	auth := response[map[string]any](t, f.request(t, "POST", "/device-authorizations", map[string]any{}, "", "control.localhost:8443", ""), 201)
	_, e := f.db.Exec(context.Background(), `UPDATE device_authorizations SET status='approved',owner_id=$1 WHERE user_code=$2`, f.owner, auth["userCode"])
	if e != nil {
		t.Fatal(e)
	}
	statuses := make(chan int, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- f.request(t, "POST", "/device-authorizations/poll", map[string]any{"deviceSecret": auth["deviceSecret"]}, "", "control.localhost:8443", "").Code
		}()
	}
	wg.Wait()
	close(statuses)
	issued := 0
	for status := range statuses {
		if status == 200 {
			issued++
		} else if status != 429 && status != 400 {
			t.Fatal("unexpected claim result", status)
		}
	}
	if issued != 1 {
		t.Fatalf("issued %d credentials for one authorization", issued)
	}
}
