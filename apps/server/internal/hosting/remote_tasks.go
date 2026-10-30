package hosting

import (
	"encoding/json"
	"net/http"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
)

type remoteTaskView struct {
	ID        string          `json:"id"`
	DeviceID  string          `json:"deviceId"`
	ProjectID string          `json:"projectId"`
	Kind      string          `json:"kind"`
	Request   json.RawMessage `json:"request"`
	State     string          `json:"state"`
	Sequence  int64           `json:"sequence"`
	Progress  json.RawMessage `json:"progress"`
	Result    json.RawMessage `json:"result,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
}

func taskFields() string {
	return `id,device_id,project_id,kind,request,state,sequence,progress,result,created_at`
}

func scanTask(row pgx.Row) (remoteTaskView, error) {
	var task remoteTaskView
	err := row.Scan(&task.ID, &task.DeviceID, &task.ProjectID, &task.Kind, &task.Request, &task.State, &task.Sequence, &task.Progress, &task.Result, &task.CreatedAt)
	return task, err
}

func (s *Server) webCreateRemoteTask(_ http.ResponseWriter, _ *http.Request, _ string) error {
	return fail(409, "remote_development_unavailable", "Remote development is not available in this release")
}

func (s *Server) webRemoteTask(w http.ResponseWriter, r *http.Request, owner string) error {
	task, err := scanTask(s.DB.QueryRow(r.Context(), `SELECT `+taskFields()+` FROM remote_agent_tasks WHERE id=$1 AND owner_id=$2`, r.PathValue("task"), owner))
	if err == pgx.ErrNoRows {
		return fail(404, "task_not_found", "Remote task not found")
	}
	if err != nil {
		return err
	}
	jsonResponse(w, 200, task)
	return nil
}

func (s *Server) webRemoteTasks(w http.ResponseWriter, r *http.Request, owner string) error {
	deviceID := r.URL.Query().Get("device")
	if !idPattern.MatchString(deviceID) {
		return fail(400, "invalid_device", "Choose a device")
	}
	var exists bool
	if err := s.DB.QueryRow(r.Context(), `SELECT true FROM personal_devices WHERE id=$1 AND owner_id=$2 AND revoked_at IS NULL`, deviceID, owner).Scan(&exists); err == pgx.ErrNoRows {
		return fail(404, "device_not_found", "Personal device not found")
	} else if err != nil {
		return err
	}
	rows, err := s.DB.Query(r.Context(), `SELECT `+taskFields()+` FROM remote_agent_tasks
	WHERE owner_id=$1 AND device_id=$2 ORDER BY created_at DESC,id DESC LIMIT 50`, owner, deviceID)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []remoteTaskView{}
	for rows.Next() {
		item, err := scanTask(rows)
		if err != nil {
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

func (s *Server) webCancelRemoteTask(w http.ResponseWriter, r *http.Request, owner string) error {
	var task remoteTaskView
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		var err error
		task, err = scanTask(tx.QueryRow(r.Context(), `SELECT `+taskFields()+` FROM remote_agent_tasks WHERE id=$1 AND owner_id=$2 FOR UPDATE`, r.PathValue("task"), owner))
		if err == pgx.ErrNoRows {
			return fail(404, "task_not_found", "Remote task not found")
		}
		if err != nil {
			return err
		}
		switch task.State {
		case "queued":
			task, err = scanTask(tx.QueryRow(r.Context(), `UPDATE remote_agent_tasks SET state='cancelled',sequence=sequence+1,finished_at=now() WHERE id=$1 RETURNING `+taskFields(), task.ID))
		case "claimed", "running":
			task, err = scanTask(tx.QueryRow(r.Context(), `UPDATE remote_agent_tasks SET state='cancel_requested',sequence=sequence+1 WHERE id=$1 RETURNING `+taskFields(), task.ID))
		}
		return err
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, task)
	return nil
}

func (s *Server) deviceClaimRemoteTask(w http.ResponseWriter, r *http.Request) error {
	if _, _, err := s.personalDeviceIdentity(r); err != nil {
		return err
	}
	if value := r.URL.Query().Get("wait"); value != "" && value != "0" && value != "25" {
		return fail(400, "invalid_wait", "Wait must be 0 or 25 seconds")
	}
	w.WriteHeader(204)
	return nil
}

func (s *Server) deviceUpdateRemoteTask(w http.ResponseWriter, r *http.Request) error {
	deviceID, owner, err := s.personalDeviceIdentity(r)
	if err != nil {
		return err
	}
	var input struct {
		Sequence int64           `json:"sequence"`
		State    string          `json:"state"`
		Progress json.RawMessage `json:"progress"`
		Result   json.RawMessage `json:"result"`
	}
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if input.Sequence < 1 || input.State != "running" && input.State != "succeeded" && input.State != "failed" && input.State != "cancelled" ||
		len(input.Progress) > 16<<10 || len(input.Result) > 64<<10 || !validTaskJSON(input.Progress) || !validTaskJSON(input.Result) {
		return fail(400, "invalid_task_update", "Invalid task status update")
	}
	var task remoteTaskView
	err = s.transaction(r.Context(), func(tx pgx.Tx) error {
		var revoked bool
		if err := tx.QueryRow(r.Context(), `SELECT revoked_at IS NOT NULL FROM personal_devices WHERE id=$1 AND owner_id=$2 FOR SHARE`, deviceID, owner).Scan(&revoked); err != nil || revoked {
			return fail(401, "device_auth_required", "Personal device was revoked")
		}
		var err error
		task, err = scanTask(tx.QueryRow(r.Context(), `SELECT `+taskFields()+` FROM remote_agent_tasks WHERE id=$1 AND device_id=$2 AND owner_id=$3 FOR UPDATE`, r.PathValue("task"), deviceID, owner))
		if err == pgx.ErrNoRows {
			return fail(404, "task_not_found", "Remote task not found")
		}
		if err != nil {
			return err
		}
		if input.Sequence == task.Sequence && input.State == task.State && sameTaskJSON(input.Progress, task.Progress, true) && sameTaskJSON(input.Result, task.Result, false) {
			return nil
		}
		if input.Sequence != task.Sequence+1 || !allowedTaskTransition(task.State, input.State) {
			return fail(409, "task_sequence_conflict", "Refresh task status before updating")
		}
		finished := input.State == "succeeded" || input.State == "failed" || input.State == "cancelled"
		task, err = scanTask(tx.QueryRow(r.Context(), `UPDATE remote_agent_tasks SET state=$1,sequence=$2,progress=COALESCE($3,'{}'::jsonb),result=$4,
		finished_at=CASE WHEN $5 THEN now() ELSE NULL END WHERE id=$6 RETURNING `+taskFields(), input.State, input.Sequence, input.Progress, input.Result, finished, task.ID))
		return err
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, task)
	return nil
}

func validTaskJSON(data json.RawMessage) bool {
	if len(data) == 0 {
		return true
	}
	var value map[string]any
	return json.Unmarshal(data, &value) == nil && value != nil
}

func sameTaskJSON(a, b json.RawMessage, emptyObject bool) bool {
	if emptyObject {
		if len(a) == 0 {
			a = json.RawMessage(`{}`)
		}
		if len(b) == 0 {
			b = json.RawMessage(`{}`)
		}
	}
	if len(a) == 0 || len(b) == 0 {
		return len(a) == len(b)
	}
	var left, right any
	return json.Unmarshal(a, &left) == nil && json.Unmarshal(b, &right) == nil && reflect.DeepEqual(left, right)
}

func allowedTaskTransition(from, to string) bool {
	switch from {
	case "claimed":
		return to == "running" || to == "succeeded" || to == "failed" || to == "cancelled"
	case "running":
		return to == "running" || to == "succeeded" || to == "failed" || to == "cancelled"
	case "cancel_requested":
		return to == "cancelled" || to == "succeeded" || to == "failed"
	default:
		return false
	}
}
