package hosting

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type catalogConnection struct {
	id, name, base, manualModel, key string
	created                          time.Time
}

type modelListItem struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

func cellappModelID(connectionID, providerModelID string) string {
	return "cellapp:" + connectionID + ":" + providerModelID
}

func parseCellappModelID(value string) (string, string, bool) {
	parts := strings.SplitN(value, ":", 3)
	if len(parts) != 3 || parts[0] != "cellapp" || !idPattern.MatchString(parts[1]) || parts[2] == "" || len(parts[2]) > 256 || strings.ContainsAny(parts[2], "\x00\r\n") {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func (s *Server) modelList(w http.ResponseWriter, r *http.Request, appID string) error {
	if origin := r.Header.Get("Origin"); origin != "" && origin != "https://"+s.requestHost(r) {
		return fail(403, "origin_rejected", "Invalid application origin")
	}
	user, err := s.appSession(r, appID)
	if err != nil {
		return err
	}
	if err := s.rate(r.Context(), "model-list:"+user+":"+appID, 10); err != nil {
		return err
	}
	connections := []catalogConnection{}
	err = s.transaction(r.Context(), func(tx pgx.Tx) error {
		access, err := appAccess(r, tx, appID, user)
		if err != nil {
			return err
		}
		if !access.allowed() {
			return fail(403, "access_required", "Application access is required")
		}
		rows, err := tx.Query(r.Context(), `SELECT c.id,c.name,c.api_base,c.model_id,c.encrypted_key,c.key_version,c.created_at
		FROM model_connection_grants g JOIN model_connections c ON c.id=g.connection_id AND c.owner_id=g.owner_id
		WHERE g.owner_id=$1 AND g.app_id=$2 AND g.revoked_at IS NULL ORDER BY c.id LIMIT 21`, user, appID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var connection catalogConnection
			var encrypted []byte
			var version int
			if err := rows.Scan(&connection.id, &connection.name, &connection.base, &connection.manualModel, &encrypted, &version, &connection.created); err != nil {
				return err
			}
			key, err := s.decryptModelKey(user, connection.id, encrypted, version)
			if err != nil {
				return fail(503, "model_secret_unavailable", "Model connection is unavailable")
			}
			connection.key = key
			connections = append(connections, connection)
		}
		return rows.Err()
	})
	if err != nil {
		return err
	}
	if len(connections) == 0 {
		return &APIError{Status: 403, Code: "model_authorization_required", Message: "Authorize a model provider for this application", Details: map[string]string{"authorizeUrl": s.modelAuthorizeURL(appID)}}
	}
	if len(connections) > 20 {
		return fail(429, "model_connection_limit", "Too many authorized model providers")
	}
	w.Header().Set("Link", "<"+s.modelAuthorizeURL(appID)+">; rel=\"cellapp-authorize\"")
	items := []modelListItem{}
	lookupContext, stopLookup := context.WithTimeout(r.Context(), 8*time.Second)
	defer stopLookup()
	for _, connection := range connections {
		seen := map[string]bool{}
		for _, name := range append([]string{connection.manualModel}, s.providerModels(lookupContext, connection)...) {
			if name == "" || len(name) > 256 || strings.ContainsAny(name, "\x00\r\n") || seen[name] {
				continue
			}
			seen[name] = true
			items = append(items, modelListItem{ID: cellappModelID(connection.id, name), Object: "model", Created: connection.created.Unix(), OwnedBy: connection.name})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	jsonResponse(w, 200, map[string]any{"object": "list", "data": items})
	return nil
}

func (s *Server) modelAuthorizeURL(appID string) string {
	return s.Config.ControlOrigin + "/dashboard/model-consent?" + url.Values{"app": {appID}}.Encode()
}

func (s *Server) webPersonalModels(w http.ResponseWriter, r *http.Request, owner string) error {
	rows, err := s.DB.Query(r.Context(), `SELECT id,name,api_base,model_id,encrypted_key,key_version,created_at
	FROM model_connections WHERE owner_id=$1 ORDER BY id LIMIT 21`, owner)
	if err != nil {
		return err
	}
	connections := []catalogConnection{}
	for rows.Next() {
		var connection catalogConnection
		var encrypted []byte
		var version int
		if err := rows.Scan(&connection.id, &connection.name, &connection.base, &connection.manualModel, &encrypted, &version, &connection.created); err != nil {
			rows.Close()
			return err
		}
		connection.key, err = s.decryptModelKey(owner, connection.id, encrypted, version)
		if err != nil {
			rows.Close()
			return fail(503, "model_secret_unavailable", "Model connection is unavailable")
		}
		connections = append(connections, connection)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(connections) > 20 {
		return fail(429, "model_connection_limit", "Too many model connections")
	}
	items := []modelListItem{}
	lookupContext, stopLookup := context.WithTimeout(r.Context(), 8*time.Second)
	defer stopLookup()
	for _, connection := range connections {
		seen := map[string]bool{}
		for _, name := range append([]string{connection.manualModel}, s.providerModels(lookupContext, connection)...) {
			if name == "" || len(name) > 256 || strings.ContainsAny(name, "\x00\r\n") || seen[name] {
				continue
			}
			seen[name] = true
			items = append(items, modelListItem{ID: cellappModelID(connection.id, name), Object: "model", Created: connection.created.Unix(), OwnedBy: connection.name})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	jsonResponse(w, 200, map[string]any{"object": "list", "data": items})
	return nil
}

func (s *Server) providerModels(parent context.Context, connection catalogConnection) []string {
	endpoint, err := modelChatURL(connection.base)
	if err != nil {
		return nil
	}
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/chat/completions") + "/models"
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil
	}
	request.Header.Set("Authorization", "Bearer "+connection.key)
	request.Header.Set("Accept", "application/json")
	client := s.ModelTestHTTPClient
	if client == nil {
		client = modelHTTPClient(net.DefaultResolver)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 256<<10+1))
	if err != nil || len(data) > 256<<10 {
		return nil
	}
	var catalog struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(data, &catalog) != nil || len(catalog.Data) > 200 {
		return nil
	}
	result := make([]string, 0, len(catalog.Data))
	for _, model := range catalog.Data {
		result = append(result, model.ID)
	}
	return result
}
