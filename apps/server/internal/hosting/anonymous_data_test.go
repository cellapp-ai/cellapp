package hosting

import (
	"cellapp/apps/server/appdata"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAnonymousDataWithRealPostgREST(t *testing.T) {
	provisionerDSN, apiURL := os.Getenv("TEST_APP_DATA_PROVISIONER_URL"), os.Getenv("TEST_POSTGREST_URL")
	if provisionerDSN == "" || apiURL == "" {
		t.Skip("isolated app-data PostgreSQL and PostgREST required")
	}
	f := setup(t)
	app, _ := f.create(t)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, provisionerDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		defer pool.Close()
		if err := appdata.CleanupApp(ctx, pool, app.ID); err != nil {
			t.Error(err)
		}
	})
	f.server.AppDataDB, f.server.Config.PostgRESTURL = pool, apiURL
	f.server.Config.AllowAnonymousAppDataWrites = true
	if err := appdata.Provision(ctx, pool, app.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `UPDATE apps SET data_state='ready',visibility='public',anonymous_pages=true,anonymous_read=false,anonymous_write=true WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	manifest := appdata.Manifest{Version: 1, Tables: []appdata.Table{{Name: "forms", Columns: []appdata.Column{{Name: "id", Type: "uuid"}, {Name: "body", Type: "text"}}, PrimaryKey: []string{"id"}, Policies: appdata.Policies{Select: "all-admitted-users", Insert: "all-admitted-users", AnonymousInsert: "all"}}}}
	if _, err := appdata.Apply(ctx, pool, app.ID, manifest); err != nil {
		t.Fatal(err)
	}
	names, _ := appdata.AppNames(app.ID)
	if err := f.server.DataProbe(ctx, names); err != nil {
		t.Fatal(err)
	}
	if err := f.server.dataTableProbe(ctx, names, manifest); err != nil {
		t.Fatal(err)
	}
	host, _ := url.Parse(app.URL)
	request := func(method, path, body, origin, cookie string, headers map[string]string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "https://"+host.Host+path, strings.NewReader(body))
		r.Host = host.Host
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if cookie != "" {
			r.Header.Set("Cookie", cookie)
		}
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		for key, value := range headers {
			r.Header.Set(key, value)
		}
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		return w
	}
	rowID := "00000000-0000-0000-0000-000000000123"
	insert := func() *httptest.ResponseRecorder {
		return request("POST", "/_cellapp/db/forms", `{"id":"`+rowID+`","body":"hello"}`, app.URL, "", map[string]string{"Prefer": "return=representation", "Content-Profile": "cellapp_internal", "Authorization": "Bearer forged"})
	}
	if w := request("GET", "/_cellapp/db/forms", "", "", "", nil); w.Code != 403 {
		t.Fatalf("anonymous read initially %d", w.Code)
	}
	if w := insert(); w.Code != 201 || strings.Contains(w.Body.String(), "hello") {
		t.Fatalf("insert-only response %d %s", w.Code, w.Body.String())
	}
	if w := request("PATCH", "/_cellapp/db/forms?id=eq."+rowID, `{"body":"changed"}`, app.URL, "", nil); w.Code != 401 {
		t.Fatalf("anonymous update %d", w.Code)
	}
	if w := request("POST", "/_cellapp/db/forms", `{}`, "https://other.example", "", nil); w.Code != 403 {
		t.Fatalf("foreign origin %d", w.Code)
	}
	if w := request("POST", "/_cellapp/db/forms", strings.Repeat("x", 1<<20+1), app.URL, "", nil); w.Code != 413 {
		t.Fatalf("oversized anonymous request %d", w.Code)
	}
	manifest.Version = 2
	manifest.Tables[0].Policies.AnonymousSelect = "all"
	if _, err := appdata.Apply(ctx, pool, app.ID, manifest); err != nil {
		t.Fatal(err)
	}
	if err := f.server.dataTableProbe(ctx, names, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `UPDATE apps SET anonymous_read=true WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", "/_cellapp/db/forms?select=id,body", "", "", "", map[string]string{"Accept-Profile": "cellapp_internal", "Authorization": "Bearer forged"}); w.Code != 200 || !strings.Contains(w.Body.String(), "hello") {
		t.Fatalf("anonymous read %d %s", w.Code, w.Body.String())
	}
	blocked := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-blocked:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case <-r.Context().Done():
		}
	}))
	f.server.Config.PostgRESTURL = backend.URL
	var group sync.WaitGroup
	results := make(chan int, cap(anonymousDataSlots))
	for range cap(anonymousDataSlots) {
		group.Add(1)
		go func() { defer group.Done(); results <- request("GET", "/_cellapp/db/forms", "", "", "", nil).Code }()
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(anonymousDataSlots) < cap(anonymousDataSlots) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(anonymousDataSlots) != cap(anonymousDataSlots) {
		close(blocked)
		group.Wait()
		backend.Close()
		close(results)
		var statuses []int
		for status := range results {
			statuses = append(statuses, status)
		}
		t.Fatalf("anonymous requests did not reach concurrency cap: %d, results %v", len(anonymousDataSlots), statuses)
	}
	if w := request("GET", "/_cellapp/db/forms", "", "", "", nil); w.Code != 429 {
		t.Errorf("concurrent anonymous request %d", w.Code)
	}
	close(blocked)
	group.Wait()
	backend.Close()
	f.server.Config.PostgRESTURL = "http://127.0.0.1:1"
	if w := request("GET", "/_cellapp/db/forms", "", "", "", nil); w.Code != 503 {
		t.Fatalf("anonymous backend failure %d", w.Code)
	}
	f.server.Config.PostgRESTURL = apiURL
	if _, err := f.db.Exec(ctx, `UPDATE apps SET anonymous_write=false WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := insert(); w.Code != 403 {
		t.Fatalf("owner write revocation %d", w.Code)
	}
	if _, err := f.db.Exec(ctx, `UPDATE apps SET anonymous_write=true WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	f.server.Config.AllowAnonymousAppDataWrites = false
	if w := insert(); w.Code != 403 {
		t.Fatalf("operator write revocation %d", w.Code)
	}
	limited := false
	for range 25 {
		if w := insert(); w.Code == 429 {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("anonymous write burst was not limited")
	}
	var admissions int
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM app_visitor_admissions WHERE app_id=$1`, app.ID).Scan(&admissions); err != nil {
		t.Fatal(err)
	}
	if admissions != 0 {
		t.Fatalf("anonymous traffic created %d signed-in admissions", admissions)
	}
}
