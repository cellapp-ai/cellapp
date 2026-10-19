package hosting

import (
	"cellapp/apps/server/appdata"
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (s *Server) mcpDataApp(ctx context.Context, owner, appID string) (App, error) {
	var app App
	err := s.transaction(ctx, func(tx pgx.Tx) error {
		var err error
		app, err = s.app(ctx, tx, appID, owner)
		return err
	})
	return app, err
}

func (s *Server) mcpDataStatus(ctx context.Context, owner, appID string) (map[string]any, error) {
	app, err := s.mcpDataApp(ctx, owner, appID)
	if err != nil {
		return nil, err
	}
	if s.AppDataDB == nil {
		return nil, fail(503, "app_data_not_ready", "Application data database unavailable")
	}
	manifest, manifestDigest, err := appdata.Latest(ctx, s.AppDataDB, app.ID)
	if err != nil {
		return nil, fail(503, "app_data_unavailable", "Application data status unavailable")
	}
	version := 0
	if manifest != nil {
		version = manifest.Version
	}
	var errorCode *string
	if err := s.DB.QueryRow(ctx, `SELECT data_error_code FROM apps WHERE id=$1`, app.ID).Scan(&errorCode); err != nil {
		return nil, err
	}
	return map[string]any{"state": app.DataState, "errorCode": errorCode, "version": version, "digest": manifestDigest}, nil
}

func (s *Server) mcpDataReconcile(ctx context.Context, owner, appID string) (map[string]any, error) {
	app, err := s.mcpDataApp(ctx, owner, appID)
	if err != nil {
		return nil, err
	}
	if s.AppDataDB == nil || s.AppDataProbe == nil {
		return nil, fail(503, "app_data_not_ready", "Application data service unavailable")
	}
	if err := appdata.Reconcile(ctx, s.DB, s.AppDataDB, app.ID, s.AppDataProbe); err != nil {
		s.logDataReconcileFailure(ctx, app.ID, err)
		return nil, fail(503, "app_data_not_ready", "Application data is not ready; inspect data status and server logs")
	}
	return map[string]any{"ready": true}, nil
}

func (s *Server) mcpDataPlan(ctx context.Context, owner, appID string, manifest appdata.Manifest) (appdata.Plan, error) {
	app, err := s.mcpDataApp(ctx, owner, appID)
	if err != nil {
		return appdata.Plan{}, err
	}
	if s.AppDataDB == nil || app.DataState != "ready" {
		return appdata.Plan{}, fail(503, "app_data_not_ready", "Application data is not ready")
	}
	previous, _, err := appdata.Latest(ctx, s.AppDataDB, app.ID)
	if err != nil {
		return appdata.Plan{}, fail(503, "app_data_unavailable", "Application data status unavailable")
	}
	plan, err := appdata.PlanManifest(previous, manifest)
	if err != nil {
		return appdata.Plan{}, fail(400, "invalid_manifest", err.Error())
	}
	return plan, nil
}

func (s *Server) mcpDataApply(ctx context.Context, owner, appID string, manifest appdata.Manifest, reviewedDigest string, allowDestructive bool) (appdata.Plan, error) {
	if !digestPattern.MatchString(reviewedDigest) {
		return appdata.Plan{}, fail(400, "reviewed_digest_required", "A reviewed plan digest is required")
	}
	plan, err := s.mcpDataPlan(ctx, owner, appID, manifest)
	if err != nil {
		return appdata.Plan{}, err
	}
	if plan.Digest != reviewedDigest {
		return appdata.Plan{}, fail(409, "plan_changed", "Manifest differs from the reviewed plan")
	}
	if plan.RequiresDestructive && !allowDestructive {
		return appdata.Plan{}, fail(409, "destructive_confirmation_required", "Explicitly allow the reviewed destructive change")
	}
	if s.AppDataProbe == nil {
		return appdata.Plan{}, fail(503, "app_data_not_ready", "Application data API is not ready")
	}
	applied, err := appdata.Apply(ctx, s.AppDataDB, appID, manifest)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return appdata.Plan{}, fail(504, "data_apply_timeout", "Table change timed out; inspect status and retry the same version")
		}
		return appdata.Plan{}, fail(409, "data_apply_failed", "Table change failed or version conflicted; inspect status and retry safely")
	}
	names, _ := appdata.AppNames(appID)
	if err := s.AppDataProbe(ctx, names); err != nil {
		return appdata.Plan{}, fail(503, "data_api_not_ready", "Table change applied but data API is not ready; verify or retry")
	}
	if err := s.dataTableProbe(ctx, names, manifest); err != nil {
		return appdata.Plan{}, fail(503, "data_api_not_ready", "Table change applied but schema cache is not ready; verify or retry")
	}
	return applied, nil
}

func (s *Server) mcpDataVerify(ctx context.Context, owner, appID string) (map[string]any, error) {
	app, err := s.mcpDataApp(ctx, owner, appID)
	if err != nil {
		return nil, err
	}
	if s.AppDataDB == nil || app.DataState != "ready" || s.AppDataProbe == nil {
		return nil, fail(503, "app_data_not_ready", "Application data is not ready")
	}
	names, _ := appdata.AppNames(app.ID)
	if err := s.AppDataProbe(ctx, names); err != nil {
		return nil, fail(503, "data_api_not_ready", "Application data API is not ready")
	}
	manifest, manifestDigest, err := appdata.Latest(ctx, s.AppDataDB, app.ID)
	if err != nil {
		return nil, fail(503, "app_data_unavailable", "Application data status unavailable")
	}
	version := 0
	if manifest != nil {
		if err := s.dataTableProbe(ctx, names, *manifest); err != nil {
			return nil, fail(503, "data_api_not_ready", "Application table API is not ready")
		}
		version = manifest.Version
	}
	return map[string]any{"ready": true, "version": version, "digest": manifestDigest}, nil
}
