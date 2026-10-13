package hosting

import (
	"cellapp/apps/server/appdata"
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (s *Server) dataOwner(r *http.Request) (App, error) {
	owner, err := s.owner(r)
	if err != nil {
		return App{}, err
	}
	var app App
	err = s.transaction(r.Context(), func(tx pgx.Tx) error {
		var err error
		app, err = s.app(r.Context(), tx, r.PathValue("app"), owner)
		return err
	})
	return app, err
}

func (s *Server) dataStatus(w http.ResponseWriter, r *http.Request) error {
	app, err := s.dataOwner(r)
	if err != nil {
		return err
	}
	if s.AppDataDB == nil {
		return fail(503, "app_data_not_ready", "Application data database unavailable")
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

func (s *Server) dataPlan(w http.ResponseWriter, r *http.Request) error {
	app, err := s.dataOwner(r)
	if err != nil {
		return err
	}
	if s.AppDataDB == nil || app.DataState != "ready" {
		return fail(503, "app_data_not_ready", "Application data is not ready")
	}
	var manifest appdata.Manifest
	if err := decode(w, r, &manifest); err != nil {
		return err
	}
	previous, _, err := appdata.Latest(r.Context(), s.AppDataDB, app.ID)
	if err != nil {
		return fail(503, "app_data_unavailable", "Application data status unavailable")
	}
	plan, err := appdata.PlanManifest(previous, manifest)
	if err != nil {
		return fail(400, "invalid_manifest", err.Error())
	}
	jsonResponse(w, 200, plan)
	return nil
}

func (s *Server) dataApply(w http.ResponseWriter, r *http.Request) error {
	app, err := s.dataOwner(r)
	if err != nil {
		return err
	}
	if s.AppDataDB == nil || app.DataState != "ready" || s.AppDataProbe == nil {
		return fail(503, "app_data_not_ready", "Application data is not ready")
	}
	var manifest appdata.Manifest
	if err := decode(w, r, &manifest); err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return fail(400, "invalid_manifest", err.Error())
	}
	plan, err := appdata.Apply(r.Context(), s.AppDataDB, app.ID, manifest)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fail(504, "data_apply_timeout", "Table change timed out; inspect status and retry the same version")
		}
		return fail(409, "data_apply_failed", "Table change failed or version conflicted; inspect status and retry safely")
	}
	names, _ := appdata.AppNames(app.ID)
	if err := s.AppDataProbe(r.Context(), names); err != nil {
		return fail(503, "data_api_not_ready", "Table change applied but data API is not ready; verify or retry")
	}
	if err := s.dataTableProbe(r.Context(), names, manifest); err != nil {
		return fail(503, "data_api_not_ready", "Table change applied but schema cache is not ready; verify or retry")
	}
	jsonResponse(w, 200, plan)
	return nil
}

func (s *Server) dataVerify(w http.ResponseWriter, r *http.Request) error {
	app, err := s.dataOwner(r)
	if err != nil {
		return err
	}
	if s.AppDataDB == nil || app.DataState != "ready" || s.AppDataProbe == nil {
		return fail(503, "app_data_not_ready", "Application data is not ready")
	}
	names, _ := appdata.AppNames(app.ID)
	if err := s.AppDataProbe(r.Context(), names); err != nil {
		return fail(503, "data_api_not_ready", "Application data API is not ready")
	}
	manifest, digest, err := appdata.Latest(r.Context(), s.AppDataDB, app.ID)
	if err != nil {
		return fail(503, "app_data_unavailable", "Application data status unavailable")
	}
	version := 0
	if manifest != nil {
		if err := s.dataTableProbe(r.Context(), names, *manifest); err != nil {
			return fail(503, "data_api_not_ready", "Application table API is not ready")
		}
		version = manifest.Version
	}
	jsonResponse(w, 200, map[string]any{"ready": true, "version": version, "digest": digest})
	return nil
}
