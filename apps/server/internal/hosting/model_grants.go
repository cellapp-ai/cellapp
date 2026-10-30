package hosting

import (
	"context"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type modelGrantView struct {
	AppID          string    `json:"appId"`
	AppName        string    `json:"appName"`
	ConnectionID   string    `json:"connectionId"`
	ConnectionName string    `json:"connectionName"`
	GrantedAt      time.Time `json:"grantedAt"`
}

type modelUsageView struct {
	AppID        string `json:"appId"`
	ConnectionID string `json:"connectionId"`
	Requests     int64  `json:"requests"`
	Succeeded    int64  `json:"succeeded"`
	Failed       int64  `json:"failed"`
	Cancelled    int64  `json:"cancelled"`
	UnknownUsage int64  `json:"unknownUsage"`
	InputTokens  int64  `json:"inputTokens"`
	OutputTokens int64  `json:"outputTokens"`
}

func (s *Server) webModelUsage(w http.ResponseWriter, r *http.Request, owner string) error {
	rows, err := s.DB.Query(r.Context(), `SELECT app_id,connection_id,count(*),
	count(*) FILTER (WHERE status='succeeded'),count(*) FILTER (WHERE status='failed'),
	count(*) FILTER (WHERE status='cancelled'),
	count(*) FILTER (WHERE input_tokens IS NULL OR output_tokens IS NULL),
	COALESCE(sum(input_tokens),0),COALESCE(sum(output_tokens),0)
	FROM model_request_usage WHERE owner_id=$1 AND started_at>=now()-interval '30 days'
	GROUP BY app_id,connection_id ORDER BY count(*) DESC,app_id,connection_id`, owner)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []modelUsageView{}
	for rows.Next() {
		var item modelUsageView
		if err := rows.Scan(&item.AppID, &item.ConnectionID, &item.Requests, &item.Succeeded, &item.Failed,
			&item.Cancelled, &item.UnknownUsage, &item.InputTokens, &item.OutputTokens); err != nil {
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

func (s *Server) webModelGrants(w http.ResponseWriter, r *http.Request, owner string) error {
	rows, err := s.DB.Query(r.Context(), `SELECT g.app_id,a.name,g.connection_id,c.name,g.granted_at
	FROM model_connection_grants g JOIN apps a ON a.id=g.app_id
	JOIN model_connections c ON c.id=g.connection_id AND c.owner_id=g.owner_id
	WHERE g.owner_id=$1 AND g.revoked_at IS NULL ORDER BY g.granted_at DESC,g.app_id`, owner)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []modelGrantView{}
	for rows.Next() {
		var item modelGrantView
		if err := rows.Scan(&item.AppID, &item.AppName, &item.ConnectionID, &item.ConnectionName, &item.GrantedAt); err != nil {
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

func (s *Server) webRevokeModelGrant(w http.ResponseWriter, r *http.Request, owner string) error {
	appID, connection := r.PathValue("app"), r.PathValue("connection")
	if !idPattern.MatchString(appID) || !idPattern.MatchString(connection) {
		return fail(404, "model_grant_not_found", "Model authorization not found")
	}
	result, err := s.DB.Exec(r.Context(), `UPDATE model_connection_grants SET revoked_at=now()
	WHERE owner_id=$1 AND app_id=$2 AND connection_id=$3 AND revoked_at IS NULL`, owner, appID, connection)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fail(404, "model_grant_not_found", "Model authorization not found")
	}
	jsonResponse(w, 200, map[string]bool{"revoked": true})
	return nil
}

type consentApp struct {
	ID, Name, Author, Host string
}

func (s *Server) modelConsentApp(r *http.Request, owner, appID string) (consentApp, error) {
	var app consentApp
	if !idPattern.MatchString(appID) {
		return app, fail(404, "app_not_found", "Application not found")
	}
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		access, err := appAccess(r, tx, appID, owner)
		if err != nil {
			return err
		}
		if !access.allowed() {
			return fail(403, "access_required", "You need access to this application")
		}
		return tx.QueryRow(r.Context(), `SELECT a.id,a.name,COALESCE(NULLIF(o.display_name,''),NULLIF(o.github_login,''),o.id),COALESCE(a.host_label,a.id)
		FROM apps a JOIN owners o ON o.id=a.owner_id WHERE a.id=$1 AND NOT a.deleted AND NOT a.suspended`, appID).Scan(&app.ID, &app.Name, &app.Author, &app.Host)
	})
	return app, err
}

func (s *Server) modelConsentPage(w http.ResponseWriter, r *http.Request) error {
	owner, err := s.browserOwner(r)
	if err != nil {
		http.Redirect(w, r, "/auth/login?"+url.Values{"return": {"/dashboard/model-consent?app=" + url.QueryEscape(r.URL.Query().Get("app"))}}.Encode(), http.StatusSeeOther)
		return nil
	}
	app, err := s.modelConsentApp(r, owner, r.URL.Query().Get("app"))
	if err != nil {
		return err
	}
	rows, err := s.DB.Query(r.Context(), `SELECT c.id,c.name,c.model_id,(g.connection_id IS NOT NULL AND g.revoked_at IS NULL) FROM model_connections c
	LEFT JOIN model_connection_grants g ON g.owner_id=c.owner_id AND g.app_id=$2 AND g.connection_id=c.id
	WHERE c.owner_id=$1 ORDER BY c.created_at DESC`, owner, app.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var options strings.Builder
	for rows.Next() {
		var connection, name, model string
		var selected bool
		if err := rows.Scan(&connection, &name, &model, &selected); err != nil {
			return err
		}
		checked := ""
		if selected {
			checked = " checked"
		}
		options.WriteString(`<label><input type="radio" name="connection" value="` + html.EscapeString(connection) + `" required` + checked + `> ` + html.EscapeString(name) + ` (` + html.EscapeString(model) + `)</label>`)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	appURL := s.Config.AppHostURL(app.Host)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	body := `<p>Application: <strong>` + html.EscapeString(app.Name) + `</strong></p><p>Author: ` + html.EscapeString(app.Author) + `</p><p>If you approve, this application's code can send your messages and data it can read to Cellapp, which forwards them to this provider. This application can call any model on an authorized connection. You may authorize more than one provider and revoke each separately. The author never receives your provider API key.</p>`
	if options.Len() == 0 {
		body += `<p>No available connection. Add one in <a href="/dashboard/model-connections">Model connections</a>, or return to the application.</p>`
	} else {
		body += `<form method="post" action="/dashboard/model-consent"><input type="hidden" name="app" value="` + html.EscapeString(app.ID) + `"><fieldset><legend>Choose your model connection</legend>` + options.String() + `</fieldset><button type="submit">Authorize this application</button></form>`
	}
	body += `<p><a href="` + html.EscapeString(appURL) + `">Cancel and return to application</a></p>`
	page(w, "Authorize model access", body)
	return nil
}

func (s *Server) modelConsentDecision(w http.ResponseWriter, r *http.Request) error {
	if err := s.browserPost(r); err != nil {
		return err
	}
	owner, err := s.browserOwner(r)
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		return fail(400, "invalid_form", "Invalid authorization form")
	}
	appID, connection := r.PostForm.Get("app"), r.PostForm.Get("connection")
	if !idPattern.MatchString(connection) {
		return fail(400, "invalid_connection", "Choose a model connection")
	}
	app, err := s.modelConsentApp(r, owner, appID)
	if err != nil {
		return err
	}
	err = s.transaction(r.Context(), func(tx pgx.Tx) error {
		var found bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM model_connections WHERE id=$1 AND owner_id=$2)`, connection, owner).Scan(&found); err != nil {
			return err
		}
		if !found {
			return fail(404, "model_connection_not_found", "Model connection not found")
		}
		_, err := tx.Exec(r.Context(), `INSERT INTO model_connection_grants(owner_id,app_id,connection_id) VALUES($1,$2,$3)
		ON CONFLICT(owner_id,app_id,connection_id) DO UPDATE SET revoked_at=NULL,granted_at=now()`, owner, app.ID, connection)
		return err
	})
	if err != nil {
		return err
	}
	http.Redirect(w, r, s.Config.AppHostURL(app.Host), http.StatusSeeOther)
	return nil
}

func (s *Server) modelGrantActive(ctx context.Context, owner, appID, connection string) (bool, error) {
	var active bool
	err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM model_connection_grants
	WHERE owner_id=$1 AND app_id=$2 AND connection_id=$3 AND revoked_at IS NULL)`, owner, appID, connection).Scan(&active)
	return active, err
}
