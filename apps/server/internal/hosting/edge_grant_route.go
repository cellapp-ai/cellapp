package hosting

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// issueEdgeGrant is callable only through the authenticated edge origin.
func (s *Server) issueEdgeGrant(w http.ResponseWriter, r *http.Request, appID string) error {
	if s.Config.EdgeOriginHost == "" || r.Host != s.Config.EdgeOriginHost || s.requestHost(r) == r.Host {
		return fail(404, "not_found", "Unknown platform path")
	}
	if len(s.Config.EdgeGrantSigningKey) == 0 {
		return fail(503, "edge_grant_unavailable", "Edge authorization unavailable")
	}
	secret := cookieValue(r, "__Host-cellapp")
	anonymous := secret == ""
	var user string
	var err error
	if !anonymous {
		user, err = s.appSession(r, appID)
		if err != nil {
			return err
		}
	}
	var grant string
	err = s.transaction(r.Context(), func(tx pgx.Tx) error {
		var deployment *string
		var deleted, suspended bool
		var sessionExpiry time.Time
		var anonymousPages bool
		if err := tx.QueryRow(r.Context(), `SELECT active_deployment,deleted,suspended,visibility='public' AND anonymous_pages FROM apps WHERE id=$1 FOR UPDATE`, appID).Scan(&deployment, &deleted, &suspended, &anonymousPages); err == pgx.ErrNoRows || deleted {
			return fail(404, "app_not_found", "Application not found")
		} else if err != nil {
			return err
		}
		if suspended {
			return fail(403, "app_suspended", "Application suspended")
		}
		var access appAdmission
		if anonymous {
			if !anonymousPages {
				return fail(401, "login_required", "Sign in to Cellapp")
			}
			if err := s.rate(r.Context(), "anonymous-edge:"+appID+":"+s.rateSource(r), 120); err != nil {
				return err
			}
		} else {
			if err := tx.QueryRow(r.Context(), `SELECT LEAST(s.expires_at,b.expires_at)
			FROM app_browser_sessions s JOIN browser_sessions b ON b.hash=s.browser_hash AND b.owner_id=$2
			WHERE s.app_id=$1 AND s.platform_user_id=$2 AND s.hash=$3 AND s.expires_at>now() AND b.expires_at>now()`,
				appID, user, keyed(s.Config.Secret, secret)).Scan(&sessionExpiry); err == pgx.ErrNoRows {
				return fail(401, "login_required", "Sign in to Cellapp")
			} else if err != nil {
				return err
			}
			access, err = appAccess(r, tx, appID, user)
			if err != nil {
				return err
			}
			if !access.allowed() {
				return fail(403, "access_required", "Request access or enter an access key")
			}
		}
		if deployment == nil {
			return fail(404, "not_published", "Application has not been published")
		}
		var spa bool
		if err := tx.QueryRow(r.Context(), `SELECT spa FROM deployments WHERE id=$1 AND app_id=$2 AND status='published'`, *deployment, appID).Scan(&spa); err == pgx.ErrNoRows {
			return fail(404, "not_published", "Application has not been published")
		} else if err != nil {
			return err
		}
		if !anonymous {
			if err := admitVisitor(r.Context(), tx, appID, user, access.Owner, s.Config.Limits.MonthlyVisitors, time.Now()); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		expires := now.Add(edgeGrantLifetime)
		if !anonymous && sessionExpiry.Before(expires) {
			expires = sessionExpiry
		}
		mode, digest := "session", appCookieDigest(secret)
		if anonymous {
			mode, digest = "anonymous", ""
		}
		claims := edgeGrant{Host: s.requestHost(r), App: appID, Deployment: *deployment, SPA: spa,
			Mode: mode, Session: digest, IssuedAt: now.Unix(), ExpiresAt: expires.Unix()}
		if claims.ExpiresAt <= claims.IssuedAt {
			return fail(401, "login_required", "Sign in to Cellapp")
		}
		grant, err = signEdgeGrant(s.Config.EdgeGrantKeyID, s.Config.EdgeGrantSigningKey, claims)
		return err
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]string{"grant": grant})
	return nil
}
