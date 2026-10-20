package hosting

import (
	"cellapp/apps/server/appdata"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) webDataStatus(w http.ResponseWriter, r *http.Request, owner string) error {
	var app App
	if err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		var err error
		app, err = s.app(r.Context(), tx, r.PathValue("app"), owner)
		return err
	}); err != nil {
		return err
	}
	if s.AppDataDB == nil {
		return fail(503, "app_data_unavailable", "Application data status unavailable")
	}
	manifest, digest, err := appdata.Latest(r.Context(), s.AppDataDB, app.ID)
	if err != nil {
		return fail(503, "app_data_unavailable", "Application data status unavailable")
	}
	version := 0
	if manifest != nil {
		version = manifest.Version
	}
	jsonResponse(w, 200, map[string]any{"state": app.DataState, "version": version, "digest": digest})
	return nil
}

type webTableStructure struct {
	Name       string           `json:"name"`
	Columns    []appdata.Column `json:"columns"`
	PrimaryKey []string         `json:"primaryKey"`
	Indexes    []appdata.Index  `json:"indexes"`
	Policies   appdata.Policies `json:"policies"`
}

func (s *Server) webDataStructure(w http.ResponseWriter, r *http.Request, owner string) error {
	var app App
	if err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		var err error
		app, err = s.app(r.Context(), tx, r.PathValue("app"), owner)
		return err
	}); err != nil {
		return err
	}
	if s.AppDataDB == nil {
		return fail(503, "app_data_unavailable", "Application table structure unavailable")
	}
	manifest, _, err := appdata.Latest(r.Context(), s.AppDataDB, app.ID)
	if err != nil {
		return fail(503, "app_data_unavailable", "Application table structure unavailable")
	}
	tables := []webTableStructure{}
	version := 0
	if manifest != nil {
		version = manifest.Version
		for _, table := range manifest.Tables {
			indexes := table.Indexes
			if indexes == nil {
				indexes = []appdata.Index{}
			}
			tables = append(tables, webTableStructure{
				Name: table.Name, Columns: table.Columns, PrimaryKey: table.PrimaryKey,
				Indexes: indexes, Policies: table.Policies,
			})
		}
	}
	jsonResponse(w, 200, struct {
		State   string              `json:"state"`
		Version int                 `json:"version"`
		Tables  []webTableStructure `json:"tables"`
	}{app.DataState, version, tables})
	return nil
}

type webAccessRequest struct {
	UserID      string    `json:"userId"`
	Status      string    `json:"status"`
	RequestedAt time.Time `json:"requestedAt"`
}

type webAccessGrant struct {
	UserID    string    `json:"userId"`
	Source    string    `json:"source"`
	GrantedAt time.Time `json:"grantedAt"`
}

func (s *Server) webAccess(w http.ResponseWriter, r *http.Request, owner string) error {
	requests := []webAccessRequest{}
	grants := []webAccessGrant{}
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		if _, err := s.app(r.Context(), tx, r.PathValue("app"), owner); err != nil {
			return err
		}
		rows, err := tx.Query(r.Context(), `SELECT u.id,q.status,q.requested_at FROM app_access_requests q
			JOIN app_users u ON u.app_id=q.app_id AND u.platform_user_id=q.platform_user_id
			WHERE q.app_id=$1 ORDER BY q.requested_at,u.id`, r.PathValue("app"))
		if err != nil {
			return err
		}
		for rows.Next() {
			var item webAccessRequest
			if err := rows.Scan(&item.UserID, &item.Status, &item.RequestedAt); err != nil {
				rows.Close()
				return err
			}
			requests = append(requests, item)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		rows, err = tx.Query(r.Context(), `SELECT u.id,g.source,g.granted_at FROM app_grants g
			JOIN app_users u ON u.app_id=g.app_id AND u.platform_user_id=g.platform_user_id
			WHERE g.app_id=$1 AND g.revoked_at IS NULL ORDER BY g.granted_at,u.id`, r.PathValue("app"))
		if err != nil {
			return err
		}
		for rows.Next() {
			var item webAccessGrant
			if err := rows.Scan(&item.UserID, &item.Source, &item.GrantedAt); err != nil {
				rows.Close()
				return err
			}
			grants = append(grants, item)
		}
		err = rows.Err()
		rows.Close()
		return err
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]any{"requests": requests, "grants": grants})
	return nil
}

func (s *Server) webVisibility(w http.ResponseWriter, r *http.Request, owner string) error {
	visibility := r.PathValue("visibility")
	if visibility != "public" && visibility != "restricted" {
		return fail(400, "invalid_visibility", "Choose public or restricted")
	}
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		if _, err := s.app(r.Context(), tx, r.PathValue("app"), owner); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `UPDATE apps SET visibility=$2 WHERE id=$1`, r.PathValue("app"), visibility)
		return err
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]string{"visibility": visibility})
	return nil
}

func (s *Server) webAccessDecision(w http.ResponseWriter, r *http.Request, owner string) error {
	decision := r.PathValue("decision")
	if decision != "approve" && decision != "deny" {
		return fail(400, "invalid_decision", "Choose approve or deny")
	}
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		if _, err := s.app(r.Context(), tx, r.PathValue("app"), owner); err != nil {
			return err
		}
		var platformUser, status string
		err := tx.QueryRow(r.Context(), `SELECT q.platform_user_id,q.status FROM app_access_requests q
			JOIN app_users u ON u.app_id=q.app_id AND u.platform_user_id=q.platform_user_id
			WHERE q.app_id=$1 AND u.id=$2 FOR UPDATE OF q`, r.PathValue("app"), r.PathValue("user")).Scan(&platformUser, &status)
		if err == pgx.ErrNoRows {
			return fail(404, "request_not_found", "Access request not found")
		}
		if err != nil {
			return err
		}
		if status == "approved" && decision == "approve" || status == "denied" && decision == "deny" {
			return nil
		}
		if status != "pending" {
			return fail(409, "request_decided", "Access request has already been decided")
		}
		newStatus := "approved"
		if decision == "deny" {
			newStatus = "denied"
		}
		if _, err = tx.Exec(r.Context(), `UPDATE app_access_requests SET status=$3,decided_at=now() WHERE app_id=$1 AND platform_user_id=$2`, r.PathValue("app"), platformUser, newStatus); err != nil {
			return err
		}
		if decision == "approve" {
			_, err = tx.Exec(r.Context(), `INSERT INTO app_grants(app_id,platform_user_id,source) VALUES($1,$2,'request')
				ON CONFLICT(app_id,platform_user_id) DO UPDATE SET source='request',revoked_at=NULL,granted_at=now()`, r.PathValue("app"), platformUser)
		}
		return err
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]string{"status": decision})
	return nil
}

func (s *Server) webGrantRevoke(w http.ResponseWriter, r *http.Request, owner string) error {
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		if _, err := s.app(r.Context(), tx, r.PathValue("app"), owner); err != nil {
			return err
		}
		result, err := tx.Exec(r.Context(), `UPDATE app_grants SET revoked_at=COALESCE(revoked_at,now())
			WHERE app_id=$1 AND platform_user_id=(SELECT platform_user_id FROM app_users WHERE app_id=$1 AND id=$2)`, r.PathValue("app"), r.PathValue("user"))
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return fail(404, "grant_not_found", "Grant not found")
		}
		if _, err := tx.Exec(r.Context(), `UPDATE app_access_requests SET status='denied',decided_at=now()
			WHERE app_id=$1 AND platform_user_id=(SELECT platform_user_id FROM app_users WHERE app_id=$1 AND id=$2)`, r.PathValue("app"), r.PathValue("user")); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]bool{"revoked": true})
	return nil
}
