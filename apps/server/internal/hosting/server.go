package hosting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Identity interface {
	Start(state, challenge string) string
	Exchange(context.Context, string, string) (string, string, error)
}
type Server struct {
	Web      fs.FS
	Config   Config
	DB       *pgxpool.Pool
	Storage  Storage
	Identity Identity
	Logger   *slog.Logger
}
type handler func(http.ResponseWriter, *http.Request) error

func (s *Server) wrap(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if e := h(w, r); e != nil {
			code := "internal_error"
			var api *APIError
			if errors.As(e, &api) {
				code = api.Code
			}
			s.Logger.Warn("request_failed", "request_id", w.Header().Get("X-Request-ID"), "code", code)
			errorResponse(w, e)
		}
	}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if e := s.DB.Ping(r.Context()); e != nil {
			return fail(503, "database_unavailable", "Database unavailable")
		}
		if e := s.Storage.Health(r.Context()); e != nil {
			return fail(503, "storage_unavailable", "Storage unavailable")
		}
		jsonResponse(w, 200, map[string]bool{"ok": true})
		return nil
	}))
	mux.HandleFunc("GET /auth/login", s.wrap(s.login))
	mux.HandleFunc("GET /auth/callback", s.wrap(s.callback))
	mux.HandleFunc("POST /auth/logout", s.wrap(s.logoutBrowser))
	mux.HandleFunc("GET /", s.wrap(s.webPage))
	mux.HandleFunc("GET /device", s.wrap(s.webPage))
	mux.HandleFunc("GET /credentials", s.wrap(s.webPage))
	mux.HandleFunc("GET /assets/", s.wrap(s.webAsset))
	mux.HandleFunc("GET /logos/", s.wrap(s.webAsset))
	mux.HandleFunc("GET /api/console/session", s.webAPI(s.webSession))
	mux.HandleFunc("POST /api/console/logout", s.webAPI(s.webLogout))
	mux.HandleFunc("GET /api/console/apps", s.webAPI(s.webApps))
	mux.HandleFunc("GET /api/console/apps/{app}", s.webAPI(s.webApp))
	mux.HandleFunc("DELETE /api/console/apps/{app}", s.webAPI(s.webDeleteApp))
	mux.HandleFunc("POST /api/console/apps/{app}/key", s.webAPI(s.webResetKey))
	mux.HandleFunc("PUT /api/console/apps/{app}/data", s.webAPI(s.webPutData))
	mux.HandleFunc("DELETE /api/console/apps/{app}/data", s.webAPI(s.webDeleteData))
	mux.HandleFunc("GET /api/console/credentials", s.webAPI(s.webCredentials))
	mux.HandleFunc("POST /api/console/credentials/{credential}/revoke", s.webAPI(s.webRevoke))
	mux.HandleFunc("POST /api/console/device/decision", s.webAPI(s.webDevice))
	mux.HandleFunc("POST /device", s.wrap(s.deviceDecision))
	mux.HandleFunc("POST /device-authorizations", s.wrap(s.createDevice))
	mux.HandleFunc("POST /device-authorizations/poll", s.wrap(s.pollDevice))
	mux.HandleFunc("POST /credentials/revoke", s.wrap(s.revokeBrowser))
	mux.HandleFunc("DELETE /credentials/current", s.wrap(s.revokeCurrent))
	mux.HandleFunc("GET /limits", s.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if _, e := s.owner(r); e != nil {
			return e
		}
		jsonResponse(w, 200, s.Config.Limits)
		return nil
	}))
	mux.HandleFunc("GET /apps", s.wrap(s.listApps))
	mux.HandleFunc("POST /apps", s.wrap(s.createApp))
	mux.HandleFunc("DELETE /apps/{app}", s.wrap(s.deleteApp))
	mux.HandleFunc("POST /apps/{app}/key", s.wrap(s.resetKey))
	mux.HandleFunc("GET /apps/{app}/data", s.wrap(s.getOwnerData))
	mux.HandleFunc("PUT /apps/{app}/data", s.wrap(s.putOwnerData))
	mux.HandleFunc("DELETE /apps/{app}/data", s.wrap(s.deleteOwnerData))
	mux.HandleFunc("POST /apps/{app}/deployments", s.wrap(s.createDeployment))
	mux.HandleFunc("GET /apps/{app}/deployments/{deployment}", s.wrap(s.deploymentStatus))
	mux.HandleFunc("PUT /apps/{app}/deployments/{deployment}/files/{index}", s.wrap(s.upload))
	mux.HandleFunc("POST /apps/{app}/deployments/{deployment}/publish", s.wrap(s.publish))
	control, _ := url.Parse(s.Config.ControlOrigin)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := id()
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		start := time.Now()
		defer func() {
			s.Logger.Info("request", "request_id", requestID, "method", r.Method, "elapsed_ms", time.Since(start).Milliseconds())
		}()
		// Host is validated directly; forwarding headers are never trusted for authorization.
		if r.Host == control.Host {
			w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
			if r.Method != "GET" && r.Method != "HEAD" && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != s.Config.ControlOrigin {
				errorResponse(w, fail(403, "origin_rejected", "Invalid origin"))
				return
			}
			if e := s.rate(r.Context(), "control:"+remoteIP(r), s.Config.Limits.Requests); e != nil {
				errorResponse(w, e)
				return
			}
			mux.ServeHTTP(w, r)
			return
		}
		suffix := "." + s.Config.AppsDomain
		if s.Config.AppPort != "" {
			suffix += ":" + s.Config.AppPort
		}
		app := strings.TrimSuffix(r.Host, suffix)
		if app == r.Host || !idPattern.MatchString(app) {
			errorResponse(w, fail(404, "not_found", "Unknown host"))
			return
		}
		if e := s.rate(r.Context(), "app:"+app+":"+remoteIP(r), s.Config.Limits.Requests); e != nil {
			errorResponse(w, e)
			return
		}
		if e := s.serveApp(w, r, app); e != nil {
			errorResponse(w, e)
		}
	})
}
func remoteIP(r *http.Request) string { // Reverse proxy is local; do not trust arbitrary X-Forwarded-For.
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host
}
func (s *Server) rate(ctx context.Context, key string, limit int64) error {
	var count int64
	e := s.DB.QueryRow(ctx, `INSERT INTO rate_limits(key,count,expires_at) VALUES($1,1,now()+interval '1 minute') ON CONFLICT(key) DO UPDATE SET count=CASE WHEN rate_limits.expires_at<now() THEN 1 ELSE rate_limits.count+1 END,expires_at=CASE WHEN rate_limits.expires_at<now() THEN now()+interval '1 minute' ELSE rate_limits.expires_at END RETURNING count`, keyed(s.Config.Secret, key)).Scan(&count)
	if e != nil {
		return e
	}
	if count > limit {
		return fail(429, "rate_limited", "Retry after one minute")
	}
	return nil
}
func (s *Server) owner(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", fail(401, "authorization_required", "Run login")
	}
	var owner string
	e := s.DB.QueryRow(r.Context(), `SELECT owner_id FROM credentials WHERE hash=$1 AND NOT revoked AND expires_at>now()`, keyed(s.Config.Secret, strings.TrimPrefix(auth, "Bearer "))).Scan(&owner)
	if e != nil {
		return "", fail(401, "authorization_required", "Credential expired or revoked; run login")
	}
	return owner, nil
}
func (s *Server) transaction(ctx context.Context, f func(pgx.Tx) error) error {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = f(tx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func lockOwner(ctx context.Context, tx pgx.Tx, owner string) error {
	var x string
	return tx.QueryRow(ctx, "SELECT id FROM owners WHERE id=$1 FOR UPDATE", owner).Scan(&x)
}

type App struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Active     *string  `json:"activeDeployment"`
	KeyHash    string   `json:"-"`
	Generation int      `json:"-"`
	Deleted    bool     `json:"-"`
	Suspended  bool     `json:"suspended"`
	URL        string   `json:"url"`
	Data       *AppData `json:"data,omitempty"`
}

func (s *Server) app(ctx context.Context, tx pgx.Tx, app, owner string) (a App, err error) {
	var provider, dataURL, anonKey *string
	err = tx.QueryRow(ctx, `SELECT id,name,active_deployment,key_hash,generation,deleted,suspended,data_provider,data_url,data_anon_key FROM apps WHERE id=$1 AND owner_id=$2 FOR UPDATE`, app, owner).Scan(&a.ID, &a.Name, &a.Active, &a.KeyHash, &a.Generation, &a.Deleted, &a.Suspended, &provider, &dataURL, &anonKey)
	if err != nil || a.Deleted {
		return a, fail(404, "app_not_found", "App not found")
	}
	a.URL = s.Config.AppURL(a.ID)
	a.Data = parseAppData(provider, dataURL, anonKey)
	return
}
func keyValid(k string) bool { return len(k) == 64 && digestPattern.MatchString(k) }
func encoded(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(fmt.Sprintf("internal encoding: %v", e))
	}
	return b
}
