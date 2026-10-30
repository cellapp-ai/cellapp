package hosting

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type deviceProjectView struct {
	ID          string          `json:"id"`
	DeviceID    string          `json:"deviceId"`
	Name        string          `json:"name"`
	Permissions json.RawMessage `json:"permissions"`
	CreatedAt   time.Time       `json:"createdAt"`
}

func (s *Server) deviceRegisterProject(w http.ResponseWriter, r *http.Request) error {
	deviceID, owner, err := s.personalDeviceIdentity(r)
	if err != nil {
		return err
	}
	var input struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Permissions struct {
			Read    bool `json:"read"`
			Write   bool `json:"write"`
			Build   bool `json:"build"`
			Publish bool `json:"publish"`
		} `json:"permissions"`
	}
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if !idPattern.MatchString(input.ID) || input.Name == "" || len(input.Name) > 100 || strings.TrimSpace(input.Name) != input.Name || !input.Permissions.Read || (input.Permissions.Build || input.Permissions.Publish) && !input.Permissions.Write {
		return fail(400, "invalid_project", "Choose a project ID and name")
	}
	permissions, err := json.Marshal(input.Permissions)
	if err != nil {
		return err
	}
	var item deviceProjectView
	err = s.transaction(r.Context(), func(tx pgx.Tx) error {
		var revoked bool
		if err := tx.QueryRow(r.Context(), `SELECT revoked_at IS NOT NULL FROM personal_devices WHERE id=$1 AND owner_id=$2 FOR SHARE`, deviceID, owner).Scan(&revoked); err != nil || revoked {
			return fail(401, "device_auth_required", "Personal device was revoked")
		}
		err := tx.QueryRow(r.Context(), `INSERT INTO device_projects(id,owner_id,device_id,name,permissions)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT(id) DO UPDATE SET permissions=EXCLUDED.permissions
		WHERE device_projects.owner_id=EXCLUDED.owner_id AND device_projects.device_id=EXCLUDED.device_id
		AND device_projects.name=EXCLUDED.name AND device_projects.revoked_at IS NULL
		RETURNING id,device_id,name,permissions,created_at`, input.ID, owner, deviceID, input.Name, permissions).Scan(&item.ID, &item.DeviceID, &item.Name, &item.Permissions, &item.CreatedAt)
		if err == pgx.ErrNoRows {
			return fail(409, "project_conflict", "Project ID already belongs to another registration")
		}
		return err
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 201, item)
	return nil
}

func (s *Server) deviceProjects(w http.ResponseWriter, r *http.Request) error {
	deviceID, owner, err := s.personalDeviceIdentity(r)
	if err != nil {
		return err
	}
	return s.writeDeviceProjects(w, r, owner, deviceID)
}

func (s *Server) webDeviceProjects(w http.ResponseWriter, r *http.Request, owner string) error {
	return s.writeDeviceProjects(w, r, owner, r.PathValue("device"))
}

func (s *Server) writeDeviceProjects(w http.ResponseWriter, r *http.Request, owner, deviceID string) error {
	var valid bool
	if err := s.DB.QueryRow(r.Context(), `SELECT true FROM personal_devices WHERE id=$1 AND owner_id=$2 AND revoked_at IS NULL`, deviceID, owner).Scan(&valid); err == pgx.ErrNoRows {
		return fail(404, "device_not_found", "Personal device not found")
	} else if err != nil {
		return err
	}
	rows, err := s.DB.Query(r.Context(), `SELECT id,device_id,name,permissions,created_at FROM device_projects
	WHERE owner_id=$1 AND device_id=$2 AND revoked_at IS NULL ORDER BY created_at DESC,id`, owner, deviceID)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []deviceProjectView{}
	for rows.Next() {
		var item deviceProjectView
		if err := rows.Scan(&item.ID, &item.DeviceID, &item.Name, &item.Permissions, &item.CreatedAt); err != nil {
			return err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	jsonResponse(w, 200, items)
	return nil
}
