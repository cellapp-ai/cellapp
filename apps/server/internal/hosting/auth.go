package hosting

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const githubIssuer = "https://github.com"

type GitHubIdentity struct {
	config  oauth2.Config
	userURL string
}

func NewIdentity(c Config) *GitHubIdentity {
	return &GitHubIdentity{
		config: oauth2.Config{
			ClientID: c.GitHubClientID, ClientSecret: c.GitHubClientSecret,
			Endpoint: github.Endpoint, RedirectURL: c.ControlOrigin + "/auth/callback",
			Scopes: []string{"read:user"},
		},
		userURL: "https://api.github.com/user",
	}
}
func (i *GitHubIdentity) Start(state, challenge string) string {
	return i.config.AuthCodeURL(state, oauth2.SetAuthURLParam("code_challenge", challenge), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}
func (i *GitHubIdentity) Exchange(ctx context.Context, code, verifier string) (string, string, error) {
	t, e := i.config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if e != nil {
		return "", "", e
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, i.userURL, nil)
	if e != nil {
		return "", "", e
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Cellapp")
	response, e := i.config.Client(ctx, t).Do(req)
	if e != nil {
		return "", "", e
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("github user API returned %s", response.Status)
	}
	var user struct {
		ID int64 `json:"id"`
	}
	if e = json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&user); e != nil || user.ID <= 0 {
		return "", "", fail(401, "invalid_identity", "GitHub did not return a valid user")
	}
	return githubIssuer, strconv.FormatInt(user.ID, 10), nil
}
func cookie(w http.ResponseWriter, name, value string, seconds int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: seconds})
}
func cookieValue(r *http.Request, name string) string {
	c, e := r.Cookie(name)
	if e != nil {
		return ""
	}
	return c.Value
}
func page(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + html.EscapeString(title) + ` · Cellapp</title><style>body{font:16px/1.6 system-ui;background:#f5f6fa;color:#182235;margin:0;padding:8vh 20px}main{max-width:520px;margin:auto;background:white;padding:36px;border:1px solid #e0e5ec;border-radius:18px}h1{font-size:26px}input,button{font:inherit;padding:12px;border-radius:8px;border:1px solid #ccd3df;box-sizing:border-box}input{width:100%;margin:12px 0}button{background:#295bdd;color:white;cursor:pointer}a{color:#295bdd}code{overflow-wrap:anywhere}</style><main><p>Cellapp</p><h1>` + html.EscapeString(title) + `</h1>` + body + `</main></html>`))
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	if s.Config.AuthMode == "dev" {
		if e := s.migrateDevelopmentOwner(r); e != nil {
			return e
		}
		return s.finishLogin(w, r, "cellapp:development", "local-owner", safeControlReturn(r.URL.Query().Get("return")))
	}
	state, verifier, browser := token(), token(), token()
	h := sha256.Sum256([]byte(verifier))
	data := map[string]string{"state": state, "verifier": verifier, "return": safeControlReturn(r.URL.Query().Get("return"))}
	_, e := s.DB.Exec(r.Context(), `INSERT INTO browser_sessions(hash,data,expires_at) VALUES($1,$2,now()+interval '10 minutes')`, keyed(s.Config.Secret, browser), encoded(data))
	if e != nil {
		return e
	}
	cookie(w, "__Host-login", browser, 600)
	http.Redirect(w, r, s.Identity.Start(state, base64.RawURLEncoding.EncodeToString(h[:])), 302)
	return nil
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) error {
	if s.Config.AuthMode != "github" || s.Identity == nil {
		return fail(404, "not_found", "OAuth callback is not enabled")
	}
	var raw []byte
	e := s.DB.QueryRow(r.Context(), `DELETE FROM browser_sessions WHERE hash=$1 AND owner_id IS NULL AND expires_at>now() RETURNING data`, keyed(s.Config.Secret, cookieValue(r, "__Host-login"))).Scan(&raw)
	if e != nil {
		return fail(401, "invalid_state", "Login expired")
	}
	var data map[string]string
	if json.Unmarshal(raw, &data) != nil || !equal(data["state"], r.URL.Query().Get("state")) {
		return fail(401, "invalid_state", "Login state mismatch")
	}
	issuer, subject, e := s.Identity.Exchange(r.Context(), r.URL.Query().Get("code"), data["verifier"])
	if e != nil {
		return fail(401, "invalid_identity", "Identity verification failed")
	}
	return s.finishLogin(w, r, issuer, subject, safeControlReturn(data["return"]))
}

// Preserve the owner ID so a brand rename cannot orphan existing apps or credentials.
func (s *Server) migrateDevelopmentOwner(r *http.Request) error {
	return s.transaction(r.Context(), func(tx pgx.Tx) error {
		if _, e := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(784232)`); e != nil {
			return e
		}
		var legacy, current bool
		if e := tx.QueryRow(r.Context(), `SELECT
			EXISTS(SELECT 1 FROM owners WHERE issuer='ohmyapp:development' AND subject='local-owner'),
			EXISTS(SELECT 1 FROM owners WHERE issuer='cellapp:development' AND subject='local-owner')`).Scan(&legacy, &current); e != nil {
			return e
		}
		if legacy && current {
			return fail(409, "development_identity_conflict", "Legacy and Cellapp development owners both exist; resolve their ownership before signing in")
		}
		if legacy {
			_, e := tx.Exec(r.Context(), `UPDATE owners SET issuer='cellapp:development' WHERE issuer='ohmyapp:development' AND subject='local-owner'`)
			return e
		}
		return nil
	})
}

func (s *Server) finishLogin(w http.ResponseWriter, r *http.Request, issuer, subject, returnTo string) error {
	var owner string
	e := s.DB.QueryRow(r.Context(), `INSERT INTO owners(id,issuer,subject) VALUES($1,$2,$3) ON CONFLICT(issuer,subject) DO UPDATE SET subject=EXCLUDED.subject RETURNING id`, id(), issuer, subject).Scan(&owner)
	if e != nil {
		return e
	}
	session := token()
	_, e = s.DB.Exec(r.Context(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 day')`, keyed(s.Config.Secret, session), owner)
	if e != nil {
		return e
	}
	cookie(w, "__Host-login", "", -1)
	cookie(w, "__Host-owner", session, 86400)
	http.Redirect(w, r, safeReturn(returnTo), 303)
	return nil
}
func (s *Server) browserOwner(r *http.Request) (string, error) {
	if cookieValue(r, "__Host-owner") == "" {
		return "", fail(401, "login_required", "Sign in first")
	}
	var owner string
	e := s.DB.QueryRow(r.Context(), `SELECT owner_id FROM browser_sessions WHERE hash=$1 AND owner_id IS NOT NULL AND expires_at>now()`, keyed(s.Config.Secret, cookieValue(r, "__Host-owner"))).Scan(&owner)
	if e == pgx.ErrNoRows {
		return "", fail(401, "login_required", "Sign in first")
	}
	if e != nil {
		return "", e
	}
	return owner, nil
}
func (s *Server) browserPost(r *http.Request) error {
	if r.Header.Get("Origin") != s.Config.ControlOrigin {
		return fail(403, "origin_rejected", "Invalid form origin")
	}
	return nil
}
func (s *Server) logoutBrowser(w http.ResponseWriter, r *http.Request) error {
	if e := s.browserPost(r); e != nil {
		return e
	}
	_, e := s.DB.Exec(r.Context(), "DELETE FROM browser_sessions WHERE hash=$1", keyed(s.Config.Secret, cookieValue(r, "__Host-owner")))
	cookie(w, "__Host-owner", "", -1)
	http.Redirect(w, r, "/", 303)
	return e
}
func (s *Server) createDevice(w http.ResponseWriter, r *http.Request) error {
	secret := token()
	code := strings.ToUpper(id()[:10])
	_, e := s.DB.Exec(r.Context(), `INSERT INTO device_authorizations(hash,user_code,expires_at) VALUES($1,$2,now()+interval '10 minutes')`, keyed(s.Config.Secret, secret), code)
	if e != nil {
		return e
	}
	jsonResponse(w, 201, map[string]any{"deviceSecret": secret, "userCode": code, "verificationUri": s.Config.ControlOrigin + "/device?code=" + code, "expiresIn": 600, "interval": 5})
	return nil
}
func (s *Server) deviceDecision(w http.ResponseWriter, r *http.Request) error {
	if e := s.browserPost(r); e != nil {
		return e
	}
	owner, e := s.browserOwner(r)
	if e != nil {
		return e
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if e = r.ParseForm(); e != nil {
		return fail(400, "invalid_form", "Invalid form")
	}
	if e = s.decideDevice(r.Context(), owner, r.Form.Get("code"), r.Form.Get("decision")); e != nil {
		return e
	}
	page(w, "已处理授权请求", `<p>回到 AI 对话或命令行继续。</p>`)
	return nil
}
func (s *Server) pollDevice(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		DeviceSecret string `json:"deviceSecret"`
	}
	if e := decode(w, r, &in); e != nil {
		return e
	}
	var result map[string]any
	var outcome error
	e := s.transaction(r.Context(), func(tx pgx.Tx) error {
		var status string
		var owner *string
		var expiry time.Time
		var last *time.Time
		h := keyed(s.Config.Secret, in.DeviceSecret)
		e := tx.QueryRow(r.Context(), `SELECT status,owner_id,expires_at,last_poll FROM device_authorizations WHERE hash=$1 FOR UPDATE`, h).Scan(&status, &owner, &expiry, &last)
		if e != nil || time.Now().After(expiry) {
			outcome = fail(400, "expired_token", "Restart device authorization")
			return nil
		}
		if last != nil && time.Since(*last) < 5*time.Second {
			outcome = fail(429, "slow_down", "Poll no faster than every five seconds")
			return nil
		}
		if _, e = tx.Exec(r.Context(), "UPDATE device_authorizations SET last_poll=now() WHERE hash=$1", h); e != nil {
			return e
		}
		if status != "approved" {
			code := map[string]string{"pending": "authorization_pending", "denied": "access_denied", "claimed": "already_claimed"}[status]
			outcome = fail(400, code, "Authorization is not available")
			return nil
		}
		credential := token()
		credentialID := id()
		_, e = tx.Exec(r.Context(), `INSERT INTO credentials(id,hash,owner_id,expires_at) VALUES($1,$2,$3,now()+interval '30 days')`, credentialID, keyed(s.Config.Secret, credential), owner)
		if e != nil {
			return e
		}
		if _, e = tx.Exec(r.Context(), "UPDATE device_authorizations SET status='claimed' WHERE hash=$1", h); e != nil {
			return e
		}
		result = map[string]any{"token": credential, "credentialId": credentialID, "expiresIn": 2592000}
		return nil
	})
	if e != nil {
		return e
	}
	if outcome != nil {
		return outcome
	}
	jsonResponse(w, 200, result)
	return nil
}
func (s *Server) revokeBrowser(w http.ResponseWriter, r *http.Request) error {
	if e := s.browserPost(r); e != nil {
		return e
	}
	owner, e := s.browserOwner(r)
	if e != nil {
		return e
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if e = r.ParseForm(); e != nil {
		return e
	}
	_, e = s.DB.Exec(r.Context(), `UPDATE credentials SET revoked=true WHERE id=$1 AND owner_id=$2`, r.Form.Get("id"), owner)
	if e != nil {
		return e
	}
	http.Redirect(w, r, "/credentials", 303)
	return nil
}
func (s *Server) revokeCurrent(w http.ResponseWriter, r *http.Request) error {
	if _, e := s.owner(r); e != nil {
		return e
	}
	_, e := s.DB.Exec(r.Context(), `UPDATE credentials SET revoked=true WHERE hash=$1`, keyed(s.Config.Secret, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")))
	if e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"revoked": true})
	return nil
}
