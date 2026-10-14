package hosting

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestMonthlyVisitorAdmissionAtCapAndRollover(t *testing.T) {
	f := setup(t)
	app, _ := f.create(t)
	ctx := context.Background()
	month := time.Date(2026, time.October, 31, 23, 59, 59, 0, time.UTC)
	for i := 0; i < 1002; i++ {
		user := fmt.Sprintf("visitor-%04d", i)
		if _, err := f.db.Exec(ctx, `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$1)`, user); err != nil {
			t.Fatal(err)
		}
	}
	admit := func(user string, owner bool, at time.Time) error {
		return f.server.transaction(ctx, func(tx pgx.Tx) error {
			return admitVisitor(ctx, tx, app.ID, user, owner, 1000, at)
		})
	}
	if err := admit(f.owner, true, month); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO app_monthly_visitors(app_id,period,admitted) VALUES($1,'2026-10',999)`, app.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO app_visitor_admissions(app_id,period,platform_user_id)
		SELECT $1,'2026-10',id FROM owners WHERE id LIKE 'visitor-%' AND id<'visitor-0999'`, app.ID); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 3)
	for i := 999; i < 1002; i++ {
		wg.Add(1)
		go func(user string) { defer wg.Done(); results <- admit(user, false, month) }(fmt.Sprintf("visitor-%04d", i))
	}
	wg.Wait()
	close(results)
	accepted, rejected := 0, 0
	for err := range results {
		if err == nil {
			accepted++
		} else if api, ok := err.(*APIError); ok && api.Status == 429 && api.Code == "visitor_limit" && api.Details.(map[string]string)["resetAt"] == "2026-11-01T00:00:00Z" {
			rejected++
		} else {
			t.Fatalf("unexpected admission result: %v", err)
		}
	}
	if accepted != 1 || rejected != 2 {
		t.Fatalf("admission race: accepted=%d rejected=%d", accepted, rejected)
	}
	var admitted int
	var admittedUser string
	if err := f.db.QueryRow(ctx, `SELECT admitted FROM app_monthly_visitors WHERE app_id=$1 AND period='2026-10'`, app.ID).Scan(&admitted); err != nil || admitted != 1000 {
		t.Fatalf("monthly count=%d: %v", admitted, err)
	}
	if err := f.db.QueryRow(ctx, `SELECT platform_user_id FROM app_visitor_admissions WHERE app_id=$1 AND period='2026-10' AND platform_user_id>='visitor-0999'`, app.ID).Scan(&admittedUser); err != nil {
		t.Fatal(err)
	}
	if err := admit(admittedUser, false, month); err != nil {
		t.Fatalf("existing visitor rejected: %v", err)
	}
	if err := admit(f.owner, true, month); err != nil {
		t.Fatalf("owner rejected: %v", err)
	}
	if err := admit("visitor-1000", false, month.Add(time.Second)); err != nil {
		t.Fatalf("new UTC month failed: %v", err)
	}
	if err := f.db.QueryRow(ctx, `SELECT admitted FROM app_monthly_visitors WHERE app_id=$1 AND period='2026-11'`, app.ID).Scan(&admitted); err != nil || admitted != 1 {
		t.Fatalf("new month count=%d: %v", admitted, err)
	}
}

func TestDataOnlyAdmissionAndUnauthorizedExclusion(t *testing.T) {
	f := setup(t)
	app, key := f.create(t)
	ctx := context.Background()
	if _, err := f.db.Exec(ctx, `UPDATE apps SET data_state='ready' WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	f.server.AppDataDB = f.db
	data := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer data.Close()
	f.server.Config.PostgRESTURL = data.URL
	host, _ := url.Parse(app.URL)
	if w := f.request(t, "GET", "/rest/v1/notes", nil, "", host.Host, ""); w.Code != 401 {
		t.Fatalf("anonymous data request=%d", w.Code)
	}
	var count int
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM app_visitor_admissions WHERE app_id=$1`, app.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("unauthorized request counted: %d %v", count, err)
	}
	cookie := f.unlock(t, app, key)
	for range 2 {
		if w := f.request(t, "GET", "/rest/v1/notes", nil, "", host.Host, cookie); w.Code != 200 {
			t.Fatalf("authorized data request=%d %s", w.Code, w.Body.String())
		}
	}
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM app_visitor_admissions WHERE app_id=$1`, app.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("data-only visitor counted %d times: %v", count, err)
	}
}
