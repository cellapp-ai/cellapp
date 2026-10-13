package hosting

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

func appReturn(value string) string {
	u, err := url.ParseRequestURI(value)
	if err != nil || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") ||
		strings.ContainsAny(value, "\\\r\n") || u.IsAbs() || u.Host != "" || u.Fragment != "" ||
		strings.HasPrefix(u.Path, "/_cellapp/") || strings.HasPrefix(u.Path, "/_hosting/") {
		return "/"
	}
	return value
}

// The control cookie stays on the control host. This short-lived ticket is
// consumed by the app host before any uploaded content is served.
func (s *Server) startAppHandoff(w http.ResponseWriter, r *http.Request) error {
	appID := r.URL.Query().Get("app")
	if !idPattern.MatchString(appID) {
		return fail(404, "app_not_found", "Application not found")
	}
	returnTo := appReturn(r.URL.Query().Get("return"))
	var exists bool
	if err := s.DB.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM apps WHERE id=$1 AND NOT deleted AND NOT suspended)`, appID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fail(404, "app_not_found", "Application not found")
	}
	owner, err := s.browserOwner(r)
	if err != nil {
		var api *APIError
		if !errors.As(err, &api) || api.Status != http.StatusUnauthorized {
			return err
		}
		query := url.Values{"return": {"/auth/app?" + url.Values{"app": {appID}, "return": {returnTo}}.Encode()}}
		http.Redirect(w, r, "/auth/login?"+query.Encode(), http.StatusSeeOther)
		return nil
	}
	_ = owner // Identity is resolved again from the browser session when consuming the ticket.
	secret := token()
	_, err = s.DB.Exec(r.Context(), `INSERT INTO app_handoff_tickets(hash,app_id,browser_hash,return_path,expires_at)
		VALUES($1,$2,$3,$4,now()+interval '2 minutes')`, keyed(s.Config.Secret, secret), appID,
		keyed(s.Config.Secret, cookieValue(r, "__Host-owner")), returnTo)
	if err != nil {
		return err
	}
	http.Redirect(w, r, s.Config.AppURL(appID)+"/_cellapp/handoff?"+url.Values{"ticket": {secret}}.Encode(), http.StatusSeeOther)
	return nil
}

func (s *Server) consumeAppHandoff(w http.ResponseWriter, r *http.Request, appID string) error {
	secret := r.URL.Query().Get("ticket")
	if secret == "" {
		return fail(401, "invalid_ticket", "Login handoff expired")
	}
	var returnTo, session string
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		var browserHash, owner string
		var active bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM apps WHERE id=$1 AND NOT deleted AND NOT suspended)`, appID).Scan(&active); err != nil {
			return err
		}
		if !active {
			return fail(404, "app_not_found", "Application not found")
		}
		err := tx.QueryRow(r.Context(), `DELETE FROM app_handoff_tickets WHERE hash=$1 AND app_id=$2 AND expires_at>now()
			RETURNING browser_hash,return_path`, keyed(s.Config.Secret, secret), appID).Scan(&browserHash, &returnTo)
		if err == pgx.ErrNoRows {
			return fail(401, "invalid_ticket", "Login handoff expired")
		}
		if err != nil {
			return err
		}
		err = tx.QueryRow(r.Context(), `SELECT owner_id FROM browser_sessions WHERE hash=$1 AND owner_id IS NOT NULL AND expires_at>now()`, browserHash).Scan(&owner)
		if err != nil {
			return fail(401, "login_required", "Sign in again")
		}
		session = token()
		_, err = tx.Exec(r.Context(), `INSERT INTO app_browser_sessions(hash,app_id,browser_hash,platform_user_id,expires_at)
			VALUES($1,$2,$3,$4,now()+interval '1 day')`, keyed(s.Config.Secret, session), appID, browserHash, owner)
		return err
	})
	if err != nil {
		return err
	}
	cookie(w, "__Host-cellapp", session, 86400)
	http.Redirect(w, r, appReturn(returnTo), http.StatusSeeOther)
	return nil
}

func (s *Server) appSession(r *http.Request, appID string) (string, error) {
	secret := cookieValue(r, "__Host-cellapp")
	if secret == "" {
		return "", fail(401, "login_required", "Sign in to Cellapp")
	}
	var owner string
	err := s.DB.QueryRow(r.Context(), `SELECT a.platform_user_id FROM app_browser_sessions a
		JOIN browser_sessions b ON b.hash=a.browser_hash AND b.owner_id=a.platform_user_id AND b.expires_at>now()
		WHERE a.hash=$1 AND a.app_id=$2 AND a.expires_at>now()`, keyed(s.Config.Secret, secret), appID).Scan(&owner)
	if err == pgx.ErrNoRows {
		return "", fail(401, "login_required", "Sign in to Cellapp")
	}
	return owner, err
}
