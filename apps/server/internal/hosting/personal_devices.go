package hosting

import (
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type personalDeviceView struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	PairedAt             time.Time  `json:"pairedAt"`
	LastSeenAt           *time.Time `json:"lastSeenAt"`
	Online               bool       `json:"online"`
	DevelopmentAvailable bool       `json:"developmentAvailable"`
}

func (s *Server) startPersonalDevicePairing(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		Name string `json:"name"`
	}
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if input.Name == "" || len(input.Name) > 80 || strings.TrimSpace(input.Name) != input.Name || strings.IndexFunc(input.Name, func(ch rune) bool { return ch < 32 || ch == 127 }) >= 0 {
		return fail(400, "invalid_device_name", "Choose a device name up to 80 characters")
	}
	if err := s.rate(r.Context(), "personal-device-pair:"+remoteIP(r), 5); err != nil {
		return err
	}
	secret := token()
	code := strings.ToUpper(id()[:10])
	_, err := s.DB.Exec(r.Context(), `INSERT INTO personal_device_authorizations(secret_hash,user_code,name,expires_at)
	VALUES($1,$2,$3,now()+interval '10 minutes')`, keyed(s.Config.Secret, secret), code, input.Name)
	if err != nil {
		return err
	}
	jsonResponse(w, 201, map[string]any{"deviceSecret": secret, "userCode": code, "verificationUri": s.Config.ControlOrigin + "/dashboard/personal-device?code=" + code, "interval": 5, "expiresIn": 600})
	return nil
}

func (s *Server) personalDeviceApprovalPage(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.browserOwner(r); err != nil {
		return err
	}
	code := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("code")))
	var name string
	err := s.DB.QueryRow(r.Context(), `SELECT name FROM personal_device_authorizations WHERE user_code=$1 AND status='pending' AND expires_at>now()`, code).Scan(&name)
	if err == pgx.ErrNoRows {
		return fail(404, "pairing_not_found", "Device pairing expired or unavailable")
	}
	if err != nil {
		return err
	}
	page(w, "确认个人设备", `<p>将设备 <strong>`+html.EscapeString(name)+`</strong> 绑定到你的 Cellapp 账户，用于本机引导与项目登记。首版尚不支持远程开发任务。</p><form method="post" action="/dashboard/personal-device"><input type="hidden" name="code" value="`+html.EscapeString(code)+`"><button type="submit" name="decision" value="approved">确认绑定</button><button type="submit" name="decision" value="denied">拒绝</button></form>`)
	return nil
}

func (s *Server) personalDeviceApproval(w http.ResponseWriter, r *http.Request) error {
	if err := s.browserPost(r); err != nil {
		return err
	}
	owner, err := s.browserOwner(r)
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		return fail(400, "invalid_form", "Invalid device approval")
	}
	code := strings.ToUpper(strings.TrimSpace(r.Form.Get("code")))
	decision := r.Form.Get("decision")
	if decision != "approved" && decision != "denied" {
		return fail(400, "invalid_decision", "Choose approve or deny")
	}
	result, err := s.DB.Exec(r.Context(), `UPDATE personal_device_authorizations SET status=$1,owner_id=$2
	WHERE user_code=$3 AND status='pending' AND expires_at>now()`, decision, map[bool]*string{true: &owner, false: nil}[decision == "approved"], code)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fail(400, "pairing_not_found", "Device pairing expired or unavailable")
	}
	page(w, "设备授权已处理", `<p>可以回到本地 Cellapp 客户端继续。</p>`)
	return nil
}

func (s *Server) pollPersonalDevicePairing(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		DeviceSecret string `json:"deviceSecret"`
	}
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if input.DeviceSecret == "" || len(input.DeviceSecret) > 256 {
		return fail(400, "invalid_device_secret", "Invalid device pairing secret")
	}
	var credential, deviceID string
	var outcome error
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		var status string
		var owner *string
		var name string
		var expiresAt time.Time
		var lastPoll *time.Time
		hash := keyed(s.Config.Secret, input.DeviceSecret)
		err := tx.QueryRow(r.Context(), `SELECT status,owner_id,name,expires_at,last_poll_at FROM personal_device_authorizations WHERE secret_hash=$1 FOR UPDATE`, hash).Scan(&status, &owner, &name, &expiresAt, &lastPoll)
		if err == pgx.ErrNoRows || time.Now().After(expiresAt) {
			outcome = fail(400, "expired_token", "Restart device pairing")
			return nil
		}
		if err != nil {
			return err
		}
		if lastPoll != nil && time.Since(*lastPoll) < 5*time.Second {
			outcome = fail(429, "slow_down", "Poll no faster than every five seconds")
			return nil
		}
		if _, err := tx.Exec(r.Context(), `UPDATE personal_device_authorizations SET last_poll_at=now() WHERE secret_hash=$1`, hash); err != nil {
			return err
		}
		if status != "approved" {
			outcome = fail(400, map[string]string{"pending": "authorization_pending", "denied": "access_denied", "claimed": "already_claimed"}[status], "Device pairing is not available")
			return nil
		}
		credential, deviceID = token(), id()
		if _, err := tx.Exec(r.Context(), `INSERT INTO personal_devices(id,owner_id,name,credential_hash) VALUES($1,$2,$3,$4)`, deviceID, *owner, name, keyed(s.Config.Secret, credential)); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `UPDATE personal_device_authorizations SET status='claimed' WHERE secret_hash=$1`, hash)
		return err
	})
	if err != nil {
		return err
	}
	if outcome != nil {
		return outcome
	}
	jsonResponse(w, 200, map[string]string{"deviceId": deviceID, "token": credential})
	return nil
}

func (s *Server) personalDeviceIdentity(r *http.Request) (string, string, error) {
	authorization := r.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") || len(authorization) > 300 {
		return "", "", fail(401, "device_auth_required", "Personal device token required")
	}
	var deviceID, owner string
	err := s.DB.QueryRow(r.Context(), `SELECT id,owner_id FROM personal_devices WHERE credential_hash=$1 AND revoked_at IS NULL`, keyed(s.Config.Secret, strings.TrimPrefix(authorization, "Bearer "))).Scan(&deviceID, &owner)
	if err == pgx.ErrNoRows {
		return "", "", fail(401, "device_auth_required", "Personal device token invalid or revoked")
	}
	return deviceID, owner, err
}

func (s *Server) personalDeviceHeartbeat(w http.ResponseWriter, r *http.Request) error {
	deviceID, owner, err := s.personalDeviceIdentity(r)
	if err != nil {
		return err
	}
	result, err := s.DB.Exec(r.Context(), `UPDATE personal_devices SET last_seen_at=now() WHERE id=$1 AND owner_id=$2 AND revoked_at IS NULL`, deviceID, owner)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fail(401, "device_auth_required", "Personal device was revoked")
	}
	jsonResponse(w, 200, map[string]bool{"online": true})
	return nil
}

func (s *Server) webPersonalDevices(w http.ResponseWriter, r *http.Request, owner string) error {
	rows, err := s.DB.Query(r.Context(), `SELECT id,name,paired_at,last_seen_at,COALESCE(last_seen_at>now()-interval '30 seconds',false) AS online,
	false AS development_available
	FROM personal_devices WHERE owner_id=$1 AND revoked_at IS NULL ORDER BY paired_at DESC,id`, owner)
	if err != nil {
		return err
	}
	defer rows.Close()
	devices := []personalDeviceView{}
	for rows.Next() {
		var device personalDeviceView
		if err := rows.Scan(&device.ID, &device.Name, &device.PairedAt, &device.LastSeenAt, &device.Online, &device.DevelopmentAvailable); err != nil {
			return err
		}
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	jsonResponse(w, 200, devices)
	return nil
}

func (s *Server) webRevokePersonalDevice(w http.ResponseWriter, r *http.Request, owner string) error {
	result, err := s.DB.Exec(r.Context(), `UPDATE personal_devices SET revoked_at=now() WHERE id=$1 AND owner_id=$2 AND revoked_at IS NULL`, r.PathValue("device"), owner)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fail(404, "device_not_found", "Personal device not found")
	}
	jsonResponse(w, 200, map[string]bool{"revoked": true})
	return nil
}
