package hosting

import (
	"encoding/json"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) serveApp(w http.ResponseWriter, r *http.Request, appID string) error {
	w.Header().Set("Content-Security-Policy", "worker-src 'none'; frame-ancestors 'none'; base-uri 'self'")
	for _, prefix := range []string{"/auth/v1", "/storage/v1", "/functions/v1", "/realtime/v1", "/rest/v1/rpc"} {
		if r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/") {
			return fail(501, "unsupported_feature", "This Supabase feature is not supported")
		}
	}
	if r.URL.Path == "/rest/v1" || strings.HasPrefix(r.URL.Path, "/rest/v1/") {
		return fail(404, "not_found", "Data resource not found")
	}
	if r.URL.Path == "/_cellapp/db" || strings.HasPrefix(r.URL.Path, "/_cellapp/db/") {
		return s.serveData(w, r, appID)
	}
	if r.Method == http.MethodGet && r.URL.Path == "/_cellapp/handoff" {
		return s.consumeAppHandoff(w, r, appID)
	}
	if r.Method == http.MethodGet && r.URL.Path == "/_cellapp/me" {
		return s.appMe(w, r, appID)
	}
	if r.Method == http.MethodGet && r.URL.Path == "/_cellapp/edge-grant" {
		return s.issueEdgeGrant(w, r, appID)
	}
	if r.Method == http.MethodPost && r.URL.Path == "/_cellapp/request" {
		return s.requestAppAccess(w, r, appID)
	}
	if r.Method != "GET" && r.Method != "HEAD" && !(r.Method == "POST" && r.URL.Path == "/_cellapp/redeem") {
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
	var requestState string
	var redirect string
	e := s.transaction(r.Context(), func(tx pgx.Tx) error {
		var a App
		var anonymousPages bool
		e := tx.QueryRow(r.Context(), `SELECT id,active_deployment,key_hash,generation,deleted,suspended,visibility='public' AND anonymous_pages FROM apps WHERE id=$1 FOR UPDATE`, appID).Scan(&a.ID, &a.Active, &a.KeyHash, &a.Generation, &a.Deleted, &a.Suspended, &anonymousPages)
		if e != nil || a.Deleted {
			return fail(404, "app_not_found", "Application not found")
		}
		if a.Suspended {
			return fail(403, "app_suspended", "Application suspended")
		}
		if r.Method == "POST" {
			if r.Header.Get("Origin") != "https://"+s.requestHost(r) {
				return fail(403, "origin_rejected", "Invalid form origin")
			}
			user, sessionError := s.appSession(r, appID)
			if sessionError != nil {
				return sessionError
			}
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
			if e = r.ParseForm(); e != nil {
				return fail(400, "invalid_form", "Invalid form")
			}
			if !equal(a.KeyHash, keyed(s.Config.Secret, r.Form.Get("key"))) {
				return fail(401, "invalid_key", "访问密钥不正确，请重新输入")
			}
			var revoked bool
			if e = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM app_grants WHERE app_id=$1 AND platform_user_id=$2 AND revoked_at IS NOT NULL)`, appID, user).Scan(&revoked); e != nil {
				return e
			}
			if revoked {
				return fail(403, "access_revoked", "访问已被所有者撤销，请申请访问")
			}
			_, e = tx.Exec(r.Context(), `INSERT INTO app_grants(app_id,platform_user_id,source) VALUES($1,$2,'key')
				ON CONFLICT(app_id,platform_user_id) DO UPDATE SET revoked_at=NULL,source='key',granted_at=now()`, appID, user)
			if e != nil {
				return e
			}
			redirect = appReturn(r.Form.Get("return"))
			return nil
		}
		if strings.HasPrefix(r.URL.Path, "/_hosting/") || strings.HasPrefix(r.URL.Path, "/_cellapp/") {
			return fail(404, "not_found", "Unknown platform path")
		}
		navigation := strings.Contains(r.Header.Get("Accept"), "text/html") && (r.Header.Get("Sec-Fetch-Dest") == "" || r.Header.Get("Sec-Fetch-Dest") == "document")
		user, sessionError := s.appSession(r, appID)
		anonymous := cookieValue(r, "__Host-cellapp") == "" && anonymousPages
		var access appAdmission
		if !anonymous {
			if sessionError != nil {
				if r.Method == "GET" && navigation {
					label := strings.TrimSuffix(strings.TrimSuffix(s.requestHost(r), ":"+s.Config.AppPort), "."+s.Config.AppsDomain)
					redirect = s.Config.ControlOrigin + "/auth/app?" + url.Values{"app": {appID}, "host": {label}, "return": {appReturn(r.URL.RequestURI())}}.Encode()
					return nil
				}
				return sessionError
			}
			access, e = appAccess(r, tx, appID, user)
			if e != nil {
				return e
			}
			if !access.allowed() {
				if r.Method == "GET" && navigation {
					showPage = true
					requestState = access.Request
					return nil
				}
				return fail(403, "access_required", "Request access or enter an access key")
			}
		} else if e := s.rate(r.Context(), "anonymous-page:"+appID+":"+s.rateSource(r), 120); e != nil {
			return e
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
		if !anonymous {
			if e = admitVisitor(r.Context(), tx, appID, user, access.Owner, s.Config.Limits.MonthlyVisitors, time.Now()); e != nil {
				return e
			}
		}
		if r.Method != "HEAD" {
			content, e = s.Storage.Get(r.Context(), objectKey(*a.Active, file.Path))
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
		title, explanation, requestButton := "申请访问", "此应用需要授权。你可以向应用所有者申请访问，或输入访问密钥。", "申请访问"
		if requestState == "pending" {
			title, explanation, requestButton = "申请审核中", "你的申请正在等待应用所有者审核。审核后重新打开此页面，也可以输入访问密钥。", "重新提交申请"
		} else if requestState == "denied" {
			title, explanation, requestButton = "申请未通过", "应用所有者未批准上次申请。你可以重新申请，或输入访问密钥。", "重新申请访问"
		}
		page(w, title, `<p>`+explanation+`</p><form method="post" action="/_cellapp/request"><button>`+requestButton+`</button></form><form method="post" action="/_cellapp/redeem"><input type="hidden" name="return" value="`+html.EscapeString(appReturn(r.URL.RequestURI()))+`"><label>访问密钥<input type="password" name="key" autocomplete="current-password" required autofocus></label><button>授权访问</button></form>`)
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
