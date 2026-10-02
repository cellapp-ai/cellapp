package hosting

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

type AppData struct {
	Provider string `json:"provider"`
	URL      string `json:"url"`
	AnonKey  string `json:"anonKey"`
	Dataset  string `json:"dataset"`
}

func parseAppData(provider, dataURL, anonKey *string) *AppData {
	if provider == nil || dataURL == nil || anonKey == nil {
		return nil
	}
	return &AppData{Provider: *provider, URL: *dataURL, AnonKey: *anonKey, Dataset: "shared"}
}

func publicDataURL(raw string) (string, error) {
	if len(raw) < 8 || len(raw) > 2048 {
		return "", fail(400, "invalid_data", "url must be an HTTPS origin")
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || strings.Contains(u.Host, "..") {
		return "", fail(400, "invalid_data", "url must be an HTTPS origin")
	}
	return u.Scheme + "://" + u.Host, nil
}

func publicAnonKey(key string) error {
	if key == "" || len(key) > 4096 || strings.ContainsAny(key, " \t\r\n") {
		return fail(400, "invalid_data", "anonKey must be the Supabase anon or publishable key")
	}
	if strings.HasPrefix(key, "sb_secret") {
		return fail(400, "invalid_data", "service role keys are not allowed")
	}
	if role, ok := jwtRole(key); ok && role == "service_role" {
		return fail(400, "invalid_data", "service role keys are not allowed")
	}
	return nil
}

func jwtRole(token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", false
	}
	payload, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		payload, e = base64.URLEncoding.DecodeString(parts[1])
	}
	if e != nil {
		return "", false
	}
	var claims struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return "", false
	}
	return claims.Role, true
}

func validatePublicData(provider, rawURL, anonKey string) (AppData, error) {
	if provider != "supabase" {
		return AppData{}, fail(400, "invalid_data", "Only provider supabase is supported")
	}
	origin, e := publicDataURL(rawURL)
	if e != nil {
		return AppData{}, e
	}
	if e = publicAnonKey(anonKey); e != nil {
		return AppData{}, e
	}
	return AppData{Provider: "supabase", URL: origin, AnonKey: anonKey, Dataset: "shared"}, nil
}

func (s *Server) bindAppData(ctx context.Context, owner, appID string, data AppData) error {
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if _, e := s.app(ctx, tx, appID, owner); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE apps SET data_provider=$2, data_url=$3, data_anon_key=$4 WHERE id=$1 AND owner_id=$5 AND NOT deleted`, appID, data.Provider, data.URL, data.AnonKey, owner)
		return e
	})
}

func (s *Server) clearAppData(ctx context.Context, owner, appID string) error {
	return s.transaction(ctx, func(tx pgx.Tx) error {
		if _, e := s.app(ctx, tx, appID, owner); e != nil {
			return e
		}
		_, e := tx.Exec(ctx, `UPDATE apps SET data_provider=NULL, data_url=NULL, data_anon_key=NULL WHERE id=$1 AND owner_id=$2 AND NOT deleted`, appID, owner)
		return e
	})
}

func (s *Server) readOwnerData(ctx context.Context, owner, appID string) (*AppData, error) {
	var data *AppData
	e := s.transaction(ctx, func(tx pgx.Tx) error {
		app, e := s.app(ctx, tx, appID, owner)
		if e != nil {
			return e
		}
		data = app.Data
		return nil
	})
	if e != nil {
		return nil, e
	}
	if data == nil {
		return nil, fail(404, "data_not_configured", "No data backend is bound")
	}
	return data, nil
}

func decodePublicData(w http.ResponseWriter, r *http.Request) (AppData, error) {
	var in struct {
		Provider string `json:"provider"`
		URL      string `json:"url"`
		AnonKey  string `json:"anonKey"`
	}
	if e := decode(w, r, &in); e != nil {
		return AppData{}, e
	}
	return validatePublicData(in.Provider, in.URL, in.AnonKey)
}

func (s *Server) getOwnerData(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	data, e := s.readOwnerData(r.Context(), owner, r.PathValue("app"))
	if e != nil {
		return e
	}
	jsonResponse(w, 200, data)
	return nil
}

func (s *Server) putOwnerData(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	data, e := decodePublicData(w, r)
	if e != nil {
		return e
	}
	if e = s.bindAppData(r.Context(), owner, r.PathValue("app"), data); e != nil {
		return e
	}
	jsonResponse(w, 200, data)
	return nil
}

func (s *Server) deleteOwnerData(w http.ResponseWriter, r *http.Request) error {
	owner, e := s.owner(r)
	if e != nil {
		return e
	}
	if e = s.clearAppData(r.Context(), owner, r.PathValue("app")); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"cleared": true})
	return nil
}

func (s *Server) webPutData(w http.ResponseWriter, r *http.Request, owner string) error {
	data, e := decodePublicData(w, r)
	if e != nil {
		return e
	}
	if e = s.bindAppData(r.Context(), owner, r.PathValue("app"), data); e != nil {
		return e
	}
	jsonResponse(w, 200, data)
	return nil
}

func (s *Server) webDeleteData(w http.ResponseWriter, r *http.Request, owner string) error {
	if e := s.clearAppData(r.Context(), owner, r.PathValue("app")); e != nil {
		return e
	}
	jsonResponse(w, 200, map[string]bool{"cleared": true})
	return nil
}
