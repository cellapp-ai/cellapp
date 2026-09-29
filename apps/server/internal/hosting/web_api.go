package hosting

import (
	"context"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type webHandler func(http.ResponseWriter, *http.Request, string) error

func (s *Server) webAPI(h webHandler) http.HandlerFunc {
	return s.wrap(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if e := s.browserPost(r); e != nil {
				return e
			}
			if r.ContentLength != 0 {
				if r.URL.Path != "/api/console/device/decision" {
					return fail(400, "invalid_json", "This operation does not accept a request body")
				}
				kind, _, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
				if e != nil || kind != "application/json" {
					return fail(415, "json_required", "Use application/json")
				}
			}
		}
		owner, e := s.browserOwner(r)
		if e != nil {
			return e
		}
		return h(w, r, owner)
	})
}

func (s *Server) webSession(w http.ResponseWriter, r *http.Request, owner string) error {
	jsonResponse(w, 200, map[string]string{"ownerId": owner, "authMode": s.Config.AuthMode})
	return nil
}
func (s *Server) webLogout(w http.ResponseWriter, r *http.Request, owner string) error {
	if _, e := s.DB.Exec(r.Context(), `DELETE FROM browser_sessions WHERE hash=$1`, keyed(s.Config.Secret, cookieValue(r, "__Host-owner"))); e != nil {
		return e
	}
	cookie(w, "__Host-owner", "", -1)
	jsonResponse(w, 200, map[string]bool{"loggedOut": true})
	return nil
}
func (s *Server) webApps(w http.ResponseWriter, r *http.Request, owner string) error {
	apps, e := s.ownerApps(r.Context(), owner)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, apps)
	return nil
}

type webRelease struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	PublishedAt *time.Time `json:"publishedAt"`
	SPA         bool       `json:"spa"`
	FileCount   int        `json:"fileCount"`
	Bytes       int64      `json:"bytes"`
}

func (s *Server) webApp(w http.ResponseWriter, r *http.Request, owner string) error {
	var app App
	var release *webRelease
	e := s.transaction(r.Context(), func(tx pgx.Tx) error {
		var e error
		app, e = s.app(r.Context(), tx, r.PathValue("app"), owner)
		if e != nil {
			return e
		}
		if app.Active == nil {
			return nil
		}
		release = &webRelease{}
		return tx.QueryRow(r.Context(), `SELECT id,status,created_at,published_at,spa,jsonb_array_length(manifest),bytes FROM deployments WHERE id=$1 AND app_id=$2`, *app.Active, app.ID).Scan(&release.ID, &release.Status, &release.CreatedAt, &release.PublishedAt, &release.SPA, &release.FileCount, &release.Bytes)
	})
	if e != nil {
		return e
	}
	jsonResponse(w, 200, struct {
		App
		Release *webRelease `json:"release"`
	}{app, release})
	return nil
}
func (s *Server) webDeleteApp(w http.ResponseWriter, r *http.Request, owner string) error {
	if e := s.deleteOwnerApp(r.Context(), owner, r.PathValue("app")); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"deleted": true})
	return nil
}
func (s *Server) webResetKey(w http.ResponseWriter, r *http.Request, owner string) error {
	key := token()
	if e := s.resetOwnerKey(r.Context(), owner, r.PathValue("app"), key); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]string{"key": key})
	return nil
}

type webCredential struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (s *Server) webCredentials(w http.ResponseWriter, r *http.Request, owner string) error {
	rows, e := s.DB.Query(r.Context(), `SELECT id,created_at,expires_at FROM credentials WHERE owner_id=$1 AND NOT revoked AND expires_at>now() ORDER BY created_at DESC,id`, owner)
	if e != nil {
		return e
	}
	defer rows.Close()
	items := []webCredential{}
	for rows.Next() {
		var item webCredential
		if e = rows.Scan(&item.ID, &item.CreatedAt, &item.ExpiresAt); e != nil {
			return e
		}
		items = append(items, item)
	}
	if e = rows.Err(); e != nil {
		return e
	}
	jsonResponse(w, 200, items)
	return nil
}
func (s *Server) webRevoke(w http.ResponseWriter, r *http.Request, owner string) error {
	result, e := s.DB.Exec(r.Context(), `UPDATE credentials SET revoked=true WHERE id=$1 AND owner_id=$2`, r.PathValue("credential"), owner)
	if e != nil {
		return e
	}
	if result.RowsAffected() != 1 {
		return fail(404, "credential_not_found", "Credential not found")
	}
	jsonResponse(w, 200, map[string]bool{"revoked": true})
	return nil
}
func (s *Server) decideDevice(ctx context.Context, owner, code, decision string) error {
	if decision != "approved" && decision != "denied" {
		return fail(400, "invalid_decision", "Choose approve or deny")
	}
	if len(code) > 64 {
		return fail(400, "invalid_code", "Code expired or already used")
	}
	result, e := s.DB.Exec(ctx, `UPDATE device_authorizations SET owner_id=$1,status=$2 WHERE user_code=$3 AND status='pending' AND expires_at>now()`, owner, decision, strings.ToUpper(strings.TrimSpace(code)))
	if e != nil {
		return e
	}
	if result.RowsAffected() != 1 {
		return fail(400, "invalid_code", "Code expired or already used")
	}
	return nil
}
func (s *Server) webDevice(w http.ResponseWriter, r *http.Request, owner string) error {
	var in struct {
		Code     string `json:"code"`
		Decision string `json:"decision"`
	}
	if e := decode(w, r, &in); e != nil {
		return e
	}
	if e := s.decideDevice(r.Context(), owner, in.Code, in.Decision); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]string{"status": in.Decision})
	return nil
}
