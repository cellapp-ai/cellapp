package hosting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Server) createApp(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	var in struct {
		Name      string `json:"name"`
		Key       string `json:"key"`
		RequestID string `json:"requestId"`
	}
	if e = decode(w, r, &in); e != nil {
		return e
	}
	if len(in.Name) < 1 || len(in.Name) > 100 || !keyValid(in.Key) || !idPattern.MatchString(in.RequestID) {
		return fail(400, "invalid_app", "Name, random key and requestId are required")
	}
	fingerprint := keyed(s.Config.Secret, string(encoded(in)))
	var app App
	e = s.transaction(r.Context(), func(tx pgx.Tx) error {
		if e := lockOwner(r.Context(), tx, owner); e != nil {
			return e
		}
		var existing, fp string
		e := tx.QueryRow(r.Context(), `SELECT id,create_digest FROM apps WHERE owner_id=$1 AND create_key=$2`, owner, in.RequestID).Scan(&existing, &fp)
		if e == nil {
			if fp != fingerprint {
				return fail(409, "idempotency_conflict", "Request ID already used with different content")
			}
			app, e = s.app(r.Context(), tx, existing, owner)
			return e
		}
		if e != pgx.ErrNoRows {
			return e
		}
		var count int64
		if e = tx.QueryRow(r.Context(), `SELECT count(*) FROM apps WHERE owner_id=$1 AND NOT deleted`, owner).Scan(&count); e != nil {
			return e
		}
		if count >= s.Config.Limits.Apps {
			return fail(429, "app_limit", "Delete an unused application before creating another")
		}
		app = App{ID: id(), Name: in.Name}
		app.URL = s.Config.AppURL(app.ID)
		_, e = tx.Exec(r.Context(), `INSERT INTO apps(id,owner_id,name,key_hash,create_key,create_digest) VALUES($1,$2,$3,$4,$5,$6)`, app.ID, owner, in.Name, keyed(s.Config.Secret, in.Key), in.RequestID, fingerprint)
		return e
	})
	if e != nil {
		return e
	}
	jsonResponse(w, 201, app)
	return nil
}
func (s *Server) listApps(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	apps, e := s.ownerApps(r.Context(), owner)
	if e != nil {
		return e
	}
	jsonResponse(w, 200, apps)
	return nil
}
func (s *Server) ownerApps(ctx context.Context, owner string) ([]App, error) {
	rows, e := s.DB.Query(ctx, `SELECT id,name,active_deployment,suspended FROM apps WHERE owner_id=$1 AND NOT deleted ORDER BY id`, owner)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	apps := []App{}
	for rows.Next() {
		var a App
		if e = rows.Scan(&a.ID, &a.Name, &a.Active, &a.Suspended); e != nil {
			return nil, e
		}
		a.URL = s.Config.AppURL(a.ID)
		apps = append(apps, a)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	return apps, nil
}
func (s *Server) deleteApp(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	if e = s.deleteOwnerApp(r.Context(), owner, r.PathValue("app")); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"deleted": true})
	return nil
}
func (s *Server) deleteOwnerApp(ctx context.Context, owner, appID string) error {
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if e := lockOwner(ctx, tx, owner); e != nil {
			return e
		}
		if _, e := s.app(ctx, tx, appID, owner); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE apps SET deleted=true,generation=generation+1 WHERE id=$1`, appID)
		return e
	})
}
func (s *Server) resetKey(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	var in struct {
		Key string `json:"key"`
	}
	if e = decode(w, r, &in); e != nil {
		return e
	}
	if !keyValid(in.Key) {
		return fail(400, "invalid_key", "A random 256-bit hex key is required")
	}
	if e = s.resetOwnerKey(r.Context(), owner, r.PathValue("app"), in.Key); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"reset": true})
	return nil
}
func (s *Server) resetOwnerKey(ctx context.Context, owner, appID, key string) error {
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if _, e := s.app(ctx, tx, appID, owner); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE apps SET key_hash=$2,generation=generation+1 WHERE id=$1`, appID, keyed(s.Config.Secret, key))
		return e
	})
}

type Deployment struct {
	ID       string    `json:"id"`
	Status   string    `json:"status"`
	Manifest []Entry   `json:"manifest"`
	Uploaded []string  `json:"uploaded"`
	Base     *string   `json:"baseVersion"`
	SPA      bool      `json:"spa"`
	Created  time.Time `json:"createdAt"`
}

func getDeployment(r *http.Request, tx pgx.Tx) (d Deployment, e error) {
	var raw, uploaded []byte
	e = tx.QueryRow(r.Context(), `SELECT id,status,manifest,uploaded,base_version,spa,created_at FROM deployments WHERE id=$1 AND app_id=$2 FOR UPDATE`, r.PathValue("deployment"), r.PathValue("app")).Scan(&d.ID, &d.Status, &raw, &uploaded, &d.Base, &d.SPA, &d.Created)
	if e != nil {
		return d, fail(404, "deployment_not_found", "Deployment not found")
	}
	if e = json.Unmarshal(raw, &d.Manifest); e != nil {
		return d, e
	}
	e = json.Unmarshal(uploaded, &d.Uploaded)
	return
}
func (s *Server) createDeployment(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	var in struct {
		Manifest  []Entry `json:"manifest"`
		RequestID string  `json:"requestId"`
		SPA       bool    `json:"spa"`
		Base      *string `json:"baseVersion"`
	}
	if e = decode(w, r, &in); e != nil {
		return e
	}
	size, e := validateManifest(in.Manifest, s.Config.Limits)
	if e != nil {
		return e
	}
	if !idPattern.MatchString(in.RequestID) {
		return fail(400, "invalid_request_id", "Invalid request ID")
	}
	fp := digest(encoded(in))
	deploymentID := ""
	e = s.transaction(r.Context(), func(tx pgx.Tx) error {
		if e := lockOwner(r.Context(), tx, owner); e != nil {
			return e
		}
		a, e := s.app(r.Context(), tx, r.PathValue("app"), owner)
		if e != nil {
			return e
		}
		if a.Suspended {
			return fail(403, "app_suspended", "Application suspended")
		}
		var existingFP string
		e = tx.QueryRow(r.Context(), `SELECT id,digest FROM deployments WHERE app_id=$1 AND idempotency_key=$2`, a.ID, in.RequestID).Scan(&deploymentID, &existingFP)
		if e == nil {
			if fp != existingFP {
				return fail(409, "idempotency_conflict", "Request ID content differs")
			}
			return nil
		}
		if e != pgx.ErrNoRows {
			return e
		}
		if !sameVersion(a.Active, in.Base) {
			return fail(409, "version_conflict", "Refresh current application version")
		}
		var used int64
		if e = tx.QueryRow(r.Context(), `SELECT COALESCE(sum(d.bytes),0) FROM deployments d JOIN apps a ON a.id=d.app_id WHERE a.owner_id=$1 AND d.status<>'cleaned'`, owner).Scan(&used); e != nil {
			return e
		}
		if size > s.Config.Limits.StorageBytes-used {
			return fail(429, "storage_limit", "Storage quota exceeded; delete unused applications or wait for cleanup")
		}
		deploymentID = id()
		_, e = tx.Exec(r.Context(), `INSERT INTO deployments(id,app_id,idempotency_key,digest,manifest,base_version,spa,bytes) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, deploymentID, a.ID, in.RequestID, fp, encoded(in.Manifest), in.Base, in.SPA, size)
		return e
	})
	if e != nil {
		return e
	}
	jsonResponse(w, 201, map[string]string{"id": deploymentID})
	return nil
}
func sameVersion(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
func (s *Server) deploymentStatus(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	var d Deployment
	e = s.transaction(r.Context(), func(tx pgx.Tx) error {
		if _, e := s.app(r.Context(), tx, r.PathValue("app"), owner); e != nil {
			return e
		}
		var e error
		d, e = getDeployment(r, tx)
		return e
	})
	if e != nil {
		return e
	}
	jsonResponse(w, 200, d)
	return nil
}
func (s *Server) upload(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	index, e := strconv.Atoi(r.PathValue("index"))
	if e != nil || index < 0 {
		return fail(400, "invalid_index", "Invalid file index")
	}
	e = s.transaction(r.Context(), func(tx pgx.Tx) error {
		a, e := s.app(r.Context(), tx, r.PathValue("app"), owner)
		if e != nil {
			return e
		}
		if a.Suspended {
			return fail(403, "app_suspended", "Application suspended")
		}
		d, e := getDeployment(r, tx)
		if e != nil {
			return e
		}
		if d.Status != "uploading" || time.Since(d.Created) > s.Config.UploadTTL {
			return fail(409, "deployment_closed", "Start a new deployment")
		}
		if index >= len(d.Manifest) {
			return fail(400, "invalid_index", "Invalid file index")
		}
		f := d.Manifest[index]
		// Strictly bounded body: neither chunked encoding nor Content-Length can bypass the manifest.
		temp, e := os.CreateTemp("", "cellapp-upload-*")
		if e != nil {
			return e
		}
		defer os.Remove(temp.Name())
		defer temp.Close()
		hash := sha256.New()
		size, e := io.Copy(io.MultiWriter(temp, hash), io.LimitReader(r.Body, f.Size+1))
		if e != nil {
			return fail(400, "upload_interrupted", "Retry file upload")
		}
		if size != f.Size || hex.EncodeToString(hash.Sum(nil)) != f.SHA256 {
			return fail(400, "digest_mismatch", "File does not match manifest")
		}
		if _, e = temp.Seek(0, 0); e != nil {
			return e
		}
		if e = s.Storage.Put(r.Context(), objectKey(d.ID, f.Path), temp, f.Size); e != nil {
			return e
		}
		found := false
		for _, p := range d.Uploaded {
			if p == f.Path {
				found = true
			}
		}
		if !found {
			d.Uploaded = append(d.Uploaded, f.Path)
		}
		_, e = tx.Exec(r.Context(), `UPDATE deployments SET uploaded=$2 WHERE id=$1`, d.ID, encoded(d.Uploaded))
		return e
	})
	if e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"uploaded": true})
	return nil
}
func (s *Server) publish(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	var result map[string]any
	e = s.transaction(r.Context(), func(tx pgx.Tx) error {
		a, e := s.app(r.Context(), tx, r.PathValue("app"), owner)
		if e != nil {
			return e
		}
		if a.Suspended {
			return fail(403, "app_suspended", "Application suspended")
		}
		d, e := getDeployment(r, tx)
		if e != nil {
			return e
		}
		result = map[string]any{"id": d.ID, "url": a.URL, "status": "published"}
		if d.Status == "published" {
			return nil
		}
		if d.Status != "uploading" || time.Since(d.Created) > s.Config.UploadTTL {
			return fail(409, "deployment_closed", "Start a new deployment")
		}
		if !sameVersion(a.Active, d.Base) {
			return fail(409, "version_conflict", "Another version was published first")
		}
		if len(d.Uploaded) != len(d.Manifest) {
			return fail(409, "incomplete_upload", "Upload all manifest files before publishing")
		}
		if _, e = tx.Exec(r.Context(), `UPDATE deployments SET status='published',published_at=now() WHERE id=$1`, d.ID); e != nil {
			return e
		}
		_, e = tx.Exec(r.Context(), `UPDATE apps SET active_deployment=$2 WHERE id=$1`, a.ID, d.ID)
		return e
	})
	if e != nil {
		return e
	}
	jsonResponse(w, 200, result)
	return nil
}
