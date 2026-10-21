package hosting

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEdgeGrantRouteAdmissionAndState(t *testing.T) {
	f := setup(t)
	app, key := f.create(t)
	deployment, _ := f.stage(t, app, "ready")
	f.uploadAndPublish(t, app, deployment, "ready")
	appHost, _ := url.Parse(app.URL)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	f.server.Config.EdgeOriginHost = "cellapp-example.us-west1.run.app"
	f.server.Config.EdgeOriginSecret = strings.Repeat("e", 32)
	f.server.Config.EdgeGrantKeyID = "test-key"
	f.server.Config.EdgeGrantSigningKey = private
	request := func(cookie, proof string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "https://"+f.server.Config.EdgeOriginHost+"/_cellapp/edge-grant", nil)
		r.Header.Set("X-Cellapp-Original-Host", appHost.Host)
		r.Header.Set("X-Cellapp-Edge-Secret", proof)
		r.Header.Set("Cookie", cookie)
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		return w
	}
	if w := request("", f.server.Config.EdgeOriginSecret); w.Code != 401 {
		t.Fatalf("anonymous grant: %d", w.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET visibility='public',anonymous_pages=true WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	anonymousGrant := request("", f.server.Config.EdgeOriginSecret)
	if anonymousGrant.Code != 200 {
		t.Fatalf("enabled anonymous grant: %d %s", anonymousGrant.Code, anonymousGrant.Body.String())
	}
	var anonymousIssued struct {
		Grant string `json:"grant"`
	}
	if err := json.Unmarshal(anonymousGrant.Body.Bytes(), &anonymousIssued); err != nil {
		t.Fatal(err)
	}
	if claims, err := verifyEdgeGrant(anonymousIssued.Grant, appHost.Host, "", map[string]ed25519.PublicKey{"test-key": public}, time.Now()); err != nil || claims.Mode != "anonymous" || claims.Deployment != deployment {
		t.Fatalf("anonymous grant claims: %+v %v", claims, err)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET anonymous_pages=false,visibility='restricted' WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := request("", f.server.Config.EdgeOriginSecret); w.Code != 401 {
		t.Fatalf("revoked anonymous grant: %d", w.Code)
	}
	if w := request("", "forged"); w.Code != 404 {
		t.Fatalf("forged edge proof: %d", w.Code)
	}
	cookie := f.unlock(t, app, key)
	w := request(cookie, f.server.Config.EdgeOriginSecret)
	if w.Code != 200 {
		t.Fatalf("grant issue: %d %s", w.Code, w.Body.String())
	}
	var issued struct {
		Grant string `json:"grant"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	claims, err := verifyEdgeGrant(issued.Grant, appHost.Host, strings.TrimPrefix(cookie, "__Host-cellapp="), map[string]ed25519.PublicKey{"test-key": public}, time.Now())
	if err != nil || claims.App != app.ID || claims.Deployment != deployment {
		t.Fatalf("grant claims: %+v %v", claims, err)
	}
	if w := request(cookie, f.server.Config.EdgeOriginSecret); w.Code != 200 {
		t.Fatalf("renewal: %d", w.Code)
	}
	var wg sync.WaitGroup
	statuses := make(chan int, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); statuses <- request(cookie, f.server.Config.EdgeOriginSecret).Code }()
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status != 200 {
			t.Fatalf("concurrent renewal=%d", status)
		}
	}
	var count int
	if err := f.db.QueryRow(context.Background(), `SELECT admitted FROM app_monthly_visitors WHERE app_id=$1`, app.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("renewal counted visitor again: %d %v", count, err)
	}
	f.server.Config.Limits.MonthlyVisitors = 1
	second := f.unlock(t, app, key)
	if w := request(second, f.server.Config.EdgeOriginSecret); w.Code != 429 {
		t.Fatalf("new visitor exceeded cap: %d %s", w.Code, w.Body.String())
	}
	ownerCookie := signedInApp(t, f, app, f.webSessionCookie(t))
	if w := request(ownerCookie, f.server.Config.EdgeOriginSecret); w.Code != 200 {
		t.Fatalf("owner grant: %d %s", w.Code, w.Body.String())
	}
	hash := keyed(f.server.Config.Secret, strings.TrimPrefix(cookie, "__Host-cellapp="))
	if _, err := f.db.Exec(context.Background(), `UPDATE app_browser_sessions SET expires_at=now()+interval '30 seconds' WHERE hash=$1`, hash); err != nil {
		t.Fatal(err)
	}
	short := request(cookie, f.server.Config.EdgeOriginSecret)
	if short.Code != 200 {
		t.Fatalf("short-session grant: %d", short.Code)
	}
	if err := json.Unmarshal(short.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	shortClaims, err := verifyEdgeGrant(issued.Grant, appHost.Host, strings.TrimPrefix(cookie, "__Host-cellapp="), map[string]ed25519.PublicKey{"test-key": public}, time.Now())
	if err != nil || shortClaims.ExpiresAt-time.Now().Unix() > 30 {
		t.Fatalf("grant outlived app session: %+v %v", shortClaims, err)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE app_browser_sessions SET expires_at=now()-interval '1 second' WHERE hash=$1`, hash); err != nil {
		t.Fatal(err)
	}
	if w := request(cookie, f.server.Config.EdgeOriginSecret); w.Code != 401 {
		t.Fatalf("expired app session renewed: %d", w.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE app_browser_sessions SET expires_at=now()+interval '1 day' WHERE hash=$1`, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE app_grants SET revoked_at=now() WHERE app_id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO app_access_requests(app_id,platform_user_id,status)
		SELECT $1,platform_user_id,'denied' FROM app_browser_sessions WHERE hash=$2
		ON CONFLICT(app_id,platform_user_id) DO UPDATE SET status='denied'`, app.ID, hash); err != nil {
		t.Fatal(err)
	}
	if w := request(cookie, f.server.Config.EdgeOriginSecret); w.Code != 403 {
		t.Fatalf("revoked visitor renewed: %d", w.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET visibility='public' WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := request(cookie, f.server.Config.EdgeOriginSecret); w.Code != 200 {
		t.Fatalf("public visitor grant: %d", w.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET visibility='restricted' WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := request(cookie, f.server.Config.EdgeOriginSecret); w.Code != 403 {
		t.Fatalf("restricted visitor renewed: %d", w.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET suspended=true WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := request(cookie, f.server.Config.EdgeOriginSecret); w.Code != 403 {
		t.Fatalf("suspended app renewed: %d", w.Code)
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET suspended=false,deleted=true WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	if w := request(cookie, f.server.Config.EdgeOriginSecret); w.Code != 404 {
		t.Fatalf("deleted app renewed: %d", w.Code)
	}
}
