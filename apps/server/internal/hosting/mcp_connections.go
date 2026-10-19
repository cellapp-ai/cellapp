package hosting

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type mcpConnection struct {
	ClientID   string    `json:"clientId"`
	ClientName string    `json:"clientName"`
	Scopes     []string  `json:"scopes"`
	GrantedAt  time.Time `json:"grantedAt"`
}

func (s *Server) webMCPConnections(w http.ResponseWriter, r *http.Request, owner string) error {
	rows, err := s.DB.Query(r.Context(), `SELECT c.client_id,k.name,c.scopes,c.granted_at FROM mcp_oauth_consents c JOIN mcp_oauth_clients k ON k.id=c.client_id WHERE c.owner_id=$1 ORDER BY c.granted_at DESC,c.client_id`, owner)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []mcpConnection{}
	for rows.Next() {
		var item mcpConnection
		if err := rows.Scan(&item.ClientID, &item.ClientName, &item.Scopes, &item.GrantedAt); err != nil {
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

func (s *Server) webMCPRevoke(w http.ResponseWriter, r *http.Request, owner string) error {
	client := r.PathValue("client")
	if !idPattern.MatchString(client) {
		return fail(404, "connection_not_found", "MCP connection not found")
	}
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		result, err := tx.Exec(r.Context(), `DELETE FROM mcp_oauth_consents WHERE owner_id=$1 AND client_id=$2`, owner, client)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return fail(404, "connection_not_found", "MCP connection not found")
		}
		if _, err := tx.Exec(r.Context(), `UPDATE mcp_oauth_tokens SET revoked_at=now() WHERE owner_id=$1 AND client_id=$2 AND revoked_at IS NULL`, owner, client); err != nil {
			return err
		}
		_, err = tx.Exec(r.Context(), `DELETE FROM mcp_oauth_codes WHERE owner_id=$1 AND client_id=$2`, owner, client)
		return err
	})
	if err != nil {
		return err
	}
	jsonResponse(w, 200, map[string]bool{"revoked": true})
	return nil
}
