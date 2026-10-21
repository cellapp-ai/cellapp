package hosting

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAnonymousPagesAndAssets(t *testing.T) {
	f := setup(t)
	app, _ := f.create(t)
	deployment, _ := f.stage(t, app, "<h1>public page</h1>")
	f.uploadAndPublish(t, app, deployment, "<h1>public page</h1>")
	host, _ := url.Parse(app.URL)
	request := func(method, path, cookie string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "https://"+host.Host+path, nil)
		r.Host = host.Host
		r.Header.Set("Accept", "text/html")
		if cookie != "" {
			r.Header.Set("Cookie", cookie)
		}
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/", ""); w.Code != 303 {
		t.Fatalf("default page %d", w.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET visibility='public',anonymous_pages=true WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/deep/link"} {
		if w := request("GET", path, ""); w.Code != 200 || !strings.Contains(w.Body.String(), "public page") {
			t.Fatalf("anonymous page %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	if w := request("GET", "/main.js", ""); w.Code != 200 {
		t.Fatalf("asset %d", w.Code)
	}
	if w := request("HEAD", "/main.js", ""); w.Code != 200 || w.Body.Len() != 0 {
		t.Fatalf("asset HEAD %d", w.Code)
	}
	if w := request("GET", "/_cellapp/me", ""); w.Code != 401 {
		t.Fatalf("identity endpoint %d", w.Code)
	}
	if w := request("GET", "/", "__Host-cellapp=forged"); w.Code != 303 {
		t.Fatalf("invalid session fell back %d", w.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET anonymous_pages=false WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", "/", ""); w.Code != 303 {
		t.Fatalf("revoked page %d", w.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET anonymous_pages=true,suspended=true WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := request("GET", "/", ""); w.Code != 403 {
		t.Fatalf("suspended app %d", w.Code)
	}
}
