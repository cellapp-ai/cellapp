package hosting

import (
	"net/http"

	"github.com/jackc/pgx/v5"
)

type anonymousSettings struct {
	Pages                 bool `json:"pages"`
	Read                  bool `json:"read"`
	Write                 bool `json:"write"`
	OperatorWritesEnabled bool `json:"operatorWritesEnabled"`
	EffectiveWrite        bool `json:"effectiveWrite"`
}

type anonymousSettingsInput struct {
	Pages *bool `json:"pages"`
	Read  *bool `json:"read"`
	Write *bool `json:"write"`
}

func (s *Server) ownerAnonymousSettings(w http.ResponseWriter, r *http.Request, owner string) error {
	var input anonymousSettingsInput
	if r.Method == http.MethodPut {
		if err := decode(w, r, &input); err != nil {
			return err
		}
		if input.Pages == nil || input.Read == nil || input.Write == nil {
			return fail(400, "invalid_anonymous_settings", "Specify pages, read and write")
		}
		if !*input.Pages && (*input.Read || *input.Write) {
			return fail(400, "invalid_anonymous_settings", "Anonymous data requires anonymous pages")
		}
	}
	var state anonymousSettings
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		app, err := s.app(r.Context(), tx, r.PathValue("app"), owner)
		if err != nil {
			return err
		}
		if r.Method == http.MethodPut {
			if app.Visibility != "public" && (*input.Pages || *input.Read || *input.Write) {
				return fail(409, "anonymous_requires_public", "Make the application public before enabling anonymous access")
			}
			if _, err := tx.Exec(r.Context(), `UPDATE apps SET anonymous_pages=$2,anonymous_read=$3,anonymous_write=$4 WHERE id=$1`, app.ID, *input.Pages, *input.Read, *input.Write); err != nil {
				return err
			}
		}
		if err := tx.QueryRow(r.Context(), `SELECT anonymous_pages,anonymous_read,anonymous_write FROM apps WHERE id=$1`, app.ID).Scan(&state.Pages, &state.Read, &state.Write); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	state.OperatorWritesEnabled = s.Config.AllowAnonymousAppDataWrites
	state.EffectiveWrite = state.Pages && state.Write && state.OperatorWritesEnabled
	jsonResponse(w, 200, state)
	return nil
}

func (s *Server) webAnonymousSettings(w http.ResponseWriter, r *http.Request, owner string) error {
	return s.ownerAnonymousSettings(w, r, owner)
}

func (s *Server) controlAnonymousSettings(w http.ResponseWriter, r *http.Request) error {
	owner, err := s.owner(r)
	if err != nil {
		return err
	}
	return s.ownerAnonymousSettings(w, r, owner)
}
