package hosting

import (
	"encoding/json"
	"html"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) serveApp(w http.ResponseWriter, r *http.Request, appID string) error {
	w.Header().Set("Content-Security-Policy", "worker-src 'none'; frame-ancestors 'none'; base-uri 'self'")
	if r.Method != "GET" && r.Method != "HEAD" && !(r.Method == "POST" && r.URL.Path == "/_hosting/unlock") {
		return fail(405, "method_not_allowed", "Static apps only support reading files")
	}
	if r.Method == "POST" {
		if e := s.rate(r.Context(), "unlock:"+appID+":"+remoteIP(r), 10); e != nil {
			return e
		}
	}
	// Keep the application row locked until the authorization decision and traffic reservation
	// commit, serializing resets/deletions with the decision to serve a response.
	var content io.ReadCloser
	var file Entry
	var showPage bool
	var redirect string
	var publicData *AppData
	e := s.transaction(r.Context(), func(tx pgx.Tx) error {
		var a App
		var provider, dataURL, anonKey *string
		e := tx.QueryRow(r.Context(), `SELECT id,active_deployment,key_hash,generation,deleted,suspended,data_provider,data_url,data_anon_key FROM apps WHERE id=$1 FOR UPDATE`, appID).Scan(&a.ID, &a.Active, &a.KeyHash, &a.Generation, &a.Deleted, &a.Suspended, &provider, &dataURL, &anonKey)
		if e != nil || a.Deleted {
			return fail(404, "app_not_found", "Application not found")
		}
		if a.Suspended {
			return fail(403, "app_suspended", "Application suspended")
		}
		a.Data = parseAppData(provider, dataURL, anonKey)
		if r.Method == "POST" {
			if r.Header.Get("Origin") != s.Config.AppURL(appID) {
				return fail(403, "origin_rejected", "Invalid form origin")
			}
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
			if e = r.ParseForm(); e != nil {
				return fail(400, "invalid_form", "Invalid form")
			}
			if !equal(a.KeyHash, keyed(s.Config.Secret, r.Form.Get("key"))) {
				return fail(401, "invalid_key", "分享密钥不正确，请重新输入")
			}
			secret := token()
			_, e = tx.Exec(r.Context(), `INSERT INTO access_sessions(hash,app_id,generation,expires_at) VALUES($1,$2,$3,now()+interval '7 days')`, keyed(s.Config.Secret, secret), appID, a.Generation)
			if e != nil {
				return e
			}
			cookie(w, "__Host-access", secret, 7*86400)
			redirect = safeReturn(r.Form.Get("return"))
			return nil
		}
		dataRequest := r.URL.Path == "/_hosting/data"
		if strings.HasPrefix(r.URL.Path, "/_hosting/") && !dataRequest {
			return fail(404, "not_found", "Unknown platform path")
		}
		var valid bool
		e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM access_sessions WHERE hash=$1 AND app_id=$2 AND generation=$3 AND expires_at>now())`, keyed(s.Config.Secret, cookieValue(r, "__Host-access")), appID, a.Generation).Scan(&valid)
		if e != nil {
			return e
		}
		navigation := strings.Contains(r.Header.Get("Accept"), "text/html") && (r.Header.Get("Sec-Fetch-Dest") == "" || r.Header.Get("Sec-Fetch-Dest") == "document")
		if !valid {
			if r.Method == "GET" && navigation {
				showPage = true
				return nil
			}
			return fail(401, "key_required", "Enter the application share key")
		}
		if dataRequest {
			if a.Data == nil {
				return fail(404, "data_not_configured", "No data backend is bound")
			}
			publicData = a.Data
			return nil
		}
		if a.Active == nil {
			return fail(404, "not_published", "Application has not been published")
		}
		var raw []byte
		var spa bool
		e = tx.QueryRow(r.Context(), `SELECT manifest,spa FROM deployments WHERE id=$1 AND status='published'`, a.Active).Scan(&raw, &spa)
		if e != nil {
			return e
		}
		var files []Entry
		if e = json.Unmarshal(raw, &files); e != nil {
			return e
		}
		requested := strings.TrimPrefix(r.URL.Path, "/")
		if requested == "" {
			requested = "index.html"
		}
		if e = validatePath(requested); e != nil {
			return fail(404, "not_found", "File not found")
		}
		found := false
		for _, f := range files {
			if f.Path == requested {
				file = f
				found = true
				break
			}
		}
		if !found && spa && navigation && path.Ext(requested) == "" {
			for _, f := range files {
				if f.Path == "index.html" {
					file = f
					found = true
				}
			}
		}
		if !found {
			return fail(404, "not_found", "File not found")
		}
		if r.Method != "HEAD" {
			now := time.Now().UTC()
			period := now.Format("2006-01")
			var used int64
			if _, e = tx.Exec(r.Context(), `INSERT INTO usage(app_id,period,bytes) VALUES($1,$2,0) ON CONFLICT DO NOTHING`, appID, period); e != nil {
				return e
			}
			if e = tx.QueryRow(r.Context(), `SELECT bytes FROM usage WHERE app_id=$1 AND period=$2 FOR UPDATE`, appID, period).Scan(&used); e != nil {
				return e
			}
			if file.Size > s.Config.Limits.TrafficBytes-used {
				return &APIError{429, "traffic_limit", "Monthly traffic exhausted; try after reset", map[string]string{"resetAt": time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)}}
			}
			content, e = s.Storage.Get(r.Context(), objectKey(*a.Active, file.Path))
			if e != nil {
				return e
			}
			_, e = tx.Exec(r.Context(), `UPDATE usage SET bytes=bytes+$3 WHERE app_id=$1 AND period=$2`, appID, period, file.Size)
			return e
		}
		return nil
	})
	if content != nil {
		defer content.Close()
	}
	if e != nil {
		return e
	}
	if redirect != "" {
		http.Redirect(w, r, redirect, 303)
		return nil
	}
	if showPage {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'; worker-src 'none'")
		page(w, "输入分享密钥", `<p>此应用由独立密钥保护。验证后可在当前浏览器访问。</p><form method="post" action="/_hosting/unlock"><input type="hidden" name="return" value="`+html.EscapeString(safeReturn(r.URL.RequestURI()))+`"><label>分享密钥<input type="password" name="key" autocomplete="current-password" required autofocus></label><button>打开应用</button></form>`)
		return nil
	}
	if publicData != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(200)
		if r.Method != "HEAD" {
			_ = json.NewEncoder(w).Encode(publicData)
		}
		return nil
	}
	kind := mime.TypeByExtension(path.Ext(file.Path))
	if kind == "" {
		kind = "application/octet-stream"
	}
	w.Header().Set("Content-Type", kind)
	w.WriteHeader(200)
	if content != nil {
		_, _ = io.Copy(w, content)
	}
	return nil
}
