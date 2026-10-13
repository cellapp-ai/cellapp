package hosting

import (
	"net/http"

	"github.com/jackc/pgx/v5"
)

type appAdmission struct {
	Visibility string
	Owner      bool
	Granted    bool
	Request    string
}

func (a appAdmission) allowed() bool { return a.Owner || a.Visibility == "public" || a.Granted }

func appAccess(r *http.Request, tx pgx.Tx, appID, user string) (appAdmission, error) {
	var a appAdmission
	err := tx.QueryRow(r.Context(), `SELECT visibility, owner_id=$2,
		EXISTS(SELECT 1 FROM app_grants WHERE app_id=$1 AND platform_user_id=$2 AND revoked_at IS NULL),
		COALESCE((SELECT status FROM app_access_requests WHERE app_id=$1 AND platform_user_id=$2),'')
		FROM apps WHERE id=$1 AND NOT deleted AND NOT suspended`, appID, user).Scan(&a.Visibility, &a.Owner, &a.Granted, &a.Request)
	if err == pgx.ErrNoRows {
		return a, fail(404, "app_not_found", "Application not found")
	}
	return a, err
}

func (s *Server) appUser(r *http.Request, tx pgx.Tx, appID, platformUser string) (string, error) {
	var appUserID string
	err := tx.QueryRow(r.Context(), `INSERT INTO app_users(id,app_id,platform_user_id) VALUES($1,$2,$3)
		ON CONFLICT(app_id,platform_user_id) DO UPDATE SET platform_user_id=EXCLUDED.platform_user_id
		RETURNING id`, id(), appID, platformUser).Scan(&appUserID)
	return appUserID, err
}

func (s *Server) appMe(w http.ResponseWriter, r *http.Request, appID string) error {
	user, err := s.appSession(r, appID)
	if err != nil {
		return err
	}
	var appUserID string
	err = s.transaction(r.Context(), func(tx pgx.Tx) error {
		access, err := appAccess(r, tx, appID, user)
		if err != nil {
			return err
		}
		if !access.allowed() {
			switch access.Request {
			case "pending":
				return fail(403, "access_pending", "Application access request is pending")
			case "denied":
				return fail(403, "access_denied", "Application access request was denied")
			default:
				return fail(403, "access_required", "Request access or enter an access key")
			}
		}
		appUserID, err = s.appUser(r, tx, appID, user)
		return err
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]string{"id": appUserID, "access": "admitted"})
	return nil
}

func (s *Server) requestAppAccess(w http.ResponseWriter, r *http.Request, appID string) error {
	if r.Header.Get("Origin") != "https://"+r.Host {
		return fail(403, "origin_rejected", "Invalid form origin")
	}
	user, err := s.appSession(r, appID)
	if err != nil {
		return err
	}
	err = s.transaction(r.Context(), func(tx pgx.Tx) error {
		var active bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM apps WHERE id=$1 AND NOT deleted AND NOT suspended)`, appID).Scan(&active); err != nil {
			return err
		}
		if !active {
			return fail(404, "app_not_found", "Application not found")
		}
		if _, err := s.appUser(r, tx, appID, user); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `INSERT INTO app_access_requests(app_id,platform_user_id,status) VALUES($1,$2,'pending')
			ON CONFLICT(app_id,platform_user_id) DO UPDATE SET status=CASE WHEN app_access_requests.status='approved' THEN 'approved' ELSE 'pending' END,
			requested_at=CASE WHEN app_access_requests.status='denied' THEN now() ELSE app_access_requests.requested_at END,
			decided_at=CASE WHEN app_access_requests.status='denied' THEN NULL ELSE app_access_requests.decided_at END`, appID, user)
		return err
	})
	if err != nil {
		return err
	}
	page(w, "申请已提交", `<p>应用所有者审核后，你可以再次打开此应用。</p>`)
	return nil
}
