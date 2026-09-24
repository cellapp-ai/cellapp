package hosting

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"time"
)

func (s *Server) Cleanup(ctx context.Context) error {
	rows, e := s.DB.Query(ctx, `SELECT d.id,a.owner_id,a.id FROM deployments d JOIN apps a ON a.id=d.app_id WHERE d.status<>'cleaned' AND (a.deleted OR (d.status='uploading' AND d.created_at < $1) OR (d.status='published' AND d.id IS DISTINCT FROM a.active_deployment AND d.published_at < $2))`, time.Now().Add(-s.Config.UploadTTL), time.Now().Add(-s.Config.Retention))
	if e != nil {
		return e
	}
	type item struct{ id, owner, app string }
	items := []item{}
	for rows.Next() {
		var v item
		if e = rows.Scan(&v.id, &v.owner, &v.app); e != nil {
			rows.Close()
			return e
		}
		items = append(items, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, v := range items {
		e = s.transaction(ctx, func(tx pgx.Tx) error {
			if e := lockOwner(ctx, tx, v.owner); e != nil {
				return e
			}
			var active *string
			var deleted bool
			if e := tx.QueryRow(ctx, `SELECT active_deployment,deleted FROM apps WHERE id=$1 FOR UPDATE`, v.app).Scan(&active, &deleted); e != nil {
				return e
			}
			if !deleted && active != nil && *active == v.id {
				return nil
			}
			var raw []byte
			var status string
			var created time.Time
			var published *time.Time
			if e := tx.QueryRow(ctx, `SELECT manifest,status,created_at,published_at FROM deployments WHERE id=$1 FOR UPDATE`, v.id).Scan(&raw, &status, &created, &published); e != nil {
				return e
			}
			if status == "cleaned" {
				return nil
			}
			if !deleted && ((status == "uploading" && time.Since(created) < s.Config.UploadTTL) || (status == "published" && published != nil && time.Since(*published) < s.Config.Retention)) {
				return nil
			}
			var files []Entry
			if e := json.Unmarshal(raw, &files); e != nil {
				return e
			}
			for _, f := range files {
				if e := s.Storage.Delete(ctx, objectKey(v.id, f.Path)); e != nil {
					return e
				}
			}
			_, e := tx.Exec(ctx, `UPDATE deployments SET status='cleaned',bytes=0,uploaded='[]' WHERE id=$1`, v.id)
			return e
		})
		if e != nil {
			return e
		}
	}
	_, e = s.DB.Exec(ctx, `DELETE FROM rate_limits WHERE expires_at<now(); DELETE FROM access_sessions WHERE expires_at<now(); DELETE FROM browser_sessions WHERE expires_at<now(); DELETE FROM device_authorizations WHERE expires_at<now()`)
	return e
}
func (s *Server) Suspend(ctx context.Context, app string) error {
	result, e := s.DB.Exec(ctx, `UPDATE apps SET suspended=true,generation=generation+1 WHERE id=$1 AND NOT deleted`, app)
	if e != nil {
		return e
	}
	if result.RowsAffected() != 1 {
		return fail(404, "app_not_found", "Application not found")
	}
	return nil
}
