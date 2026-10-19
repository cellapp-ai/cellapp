package hosting

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var mcpPKCEPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43,128}$`)

const mcpScopes = "mcp:read mcp:deploy mcp:data"

type mcpClientRegistration struct {
	Name            string   `json:"client_name"`
	RedirectURIs    []string `json:"redirect_uris"`
	GrantTypes      []string `json:"grant_types,omitempty"`
	ResponseTypes   []string `json:"response_types,omitempty"`
	TokenAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
}

func (s *Server) mcpProtectedResourceMetadata(w http.ResponseWriter, _ *http.Request) error {
	jsonResponse(w, 200, map[string]any{
		"resource":                 s.Config.ControlOrigin + "/mcp",
		"authorization_servers":    []string{s.Config.ControlOrigin},
		"scopes_supported":         strings.Fields(mcpScopes),
		"bearer_methods_supported": []string{"header"},
	})
	return nil
}

func (s *Server) mcpAuthorizationServerMetadata(w http.ResponseWriter, _ *http.Request) error {
	base := s.Config.ControlOrigin
	jsonResponse(w, 200, map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"registration_endpoint":                 base + "/oauth/register",
		"revocation_endpoint":                   base + "/oauth/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"code_challenge_methods_supported":      []string{"S256"},
		"scopes_supported":                      strings.Fields(mcpScopes),
	})
	return nil
}

func validMCPRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Fragment != "" || u.Host == "" || u.Path == "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	return u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
}

func validMCPMetadataURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && len(raw) <= 2048
}

func publicMCPIP(ip net.IP) bool {
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	for _, raw := range []string{"100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4", "2001:db8::/32", "fc00::/7", "fe80::/10"} {
		if netip.MustParsePrefix(raw).Contains(address) {
			return false
		}
	}
	return true
}

func mcpMetadataClient() *http.Client {
	return &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			for _, ip := range ips {
				if !publicMCPIP(ip) {
					return nil, fail(400, "invalid_client_metadata", "Client metadata host is not public")
				}
			}
			if len(ips) == 0 {
				return nil, fail(400, "invalid_client_metadata", "Client metadata host has no address")
			}
			return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
	}}
}

func (s *Server) mcpMetadataRegistration(ctx context.Context, clientID string) error {
	if !validMCPMetadataURL(clientID) {
		return fail(400, "invalid_client", "Unknown MCP client")
	}
	client := s.MCPMetadataHTTPClient
	if client == nil {
		client = mcpMetadataClient()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, nil)
	if err != nil {
		return fail(400, "invalid_client_metadata", "Invalid client metadata URL")
	}
	req.Header.Set("Accept", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return fail(400, "invalid_client_metadata", "Client metadata unavailable")
	}
	defer response.Body.Close()
	if response.Request == nil || response.Request.URL.String() != clientID {
		return fail(400, "invalid_client_metadata", "Client metadata redirect is not allowed")
	}
	if response.StatusCode != 200 {
		return fail(400, "invalid_client_metadata", "Client metadata unavailable")
	}
	var document struct {
		ClientID string `json:"client_id"`
		mcpClientRegistration
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 32769))
	if err := decoder.Decode(&document); err != nil || document.ClientID != clientID || len(document.Name) == 0 || len(document.Name) > 100 || len(document.RedirectURIs) == 0 || len(document.RedirectURIs) > 5 || document.TokenAuthMethod != "none" {
		return fail(400, "invalid_client_metadata", "Invalid client metadata document")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fail(400, "invalid_client_metadata", "Client metadata must be one JSON document")
	}
	for _, grant := range document.GrantTypes {
		if grant != "authorization_code" {
			return fail(400, "invalid_client_metadata", "Unsupported client grant")
		}
	}
	for _, kind := range document.ResponseTypes {
		if kind != "code" {
			return fail(400, "invalid_client_metadata", "Unsupported client response")
		}
	}
	for _, redirect := range document.RedirectURIs {
		if !validMCPRedirect(redirect) {
			return fail(400, "invalid_client_metadata", "Invalid client redirect URI")
		}
	}
	_, err = s.DB.Exec(ctx, `INSERT INTO mcp_oauth_clients(id,name,redirect_uris,metadata_uri) VALUES($1,$2,$3,$1) ON CONFLICT(id) DO NOTHING`, clientID, document.Name, encoded(document.RedirectURIs))
	return err
}

func (s *Server) mcpRegisterClient(w http.ResponseWriter, r *http.Request) error {
	var in mcpClientRegistration
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	// RFC 7591 requires ignoring client metadata that this server does not understand.
	if err := decoder.Decode(&in); err != nil {
		return fail(400, "invalid_json", "Invalid request body")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fail(400, "invalid_json", "Expected one JSON value")
	}
	if len(in.Name) == 0 || len(in.Name) > 100 || len(in.RedirectURIs) == 0 || len(in.RedirectURIs) > 5 || (in.TokenAuthMethod != "" && in.TokenAuthMethod != "none") {
		return fail(400, "invalid_client_metadata", "Invalid public client registration")
	}
	for _, value := range in.GrantTypes {
		if value != "authorization_code" && value != "refresh_token" {
			return fail(400, "invalid_client_metadata", "Unsupported client grant")
		}
	}
	for _, value := range in.ResponseTypes {
		if value != "code" {
			return fail(400, "invalid_client_metadata", "Only code responses are supported")
		}
	}
	seen := map[string]bool{}
	for _, redirect := range in.RedirectURIs {
		if len(redirect) > 2048 || !validMCPRedirect(redirect) || seen[redirect] {
			return fail(400, "invalid_redirect_uri", "Redirect URI must be unique and use HTTPS or localhost HTTP")
		}
		seen[redirect] = true
	}
	clientID := id()
	if _, err := s.DB.Exec(r.Context(), `INSERT INTO mcp_oauth_clients(id,name,redirect_uris) VALUES($1,$2,$3)`, clientID, in.Name, encoded(in.RedirectURIs)); err != nil {
		return err
	}
	jsonResponse(w, 201, map[string]any{
		"client_id":                  clientID,
		"client_name":                in.Name,
		"redirect_uris":              in.RedirectURIs,
		"grant_types":                []string{"authorization_code"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	})
	return nil
}

type mcpAuthorization struct {
	ClientID, ClientName, RedirectURI, Resource, State, Challenge string
	Scopes                                                        []string
}

func (s *Server) parseMCPAuthorization(ctx context.Context, values url.Values) (mcpAuthorization, error) {
	a := mcpAuthorization{ClientID: values.Get("client_id"), RedirectURI: values.Get("redirect_uri"), Resource: values.Get("resource"), State: values.Get("state"), Challenge: values.Get("code_challenge")}
	if values.Get("response_type") != "code" || values.Get("code_challenge_method") != "S256" || !mcpPKCEPattern.MatchString(a.Challenge) || len(a.State) > 2048 || a.Resource != s.Config.ControlOrigin+"/mcp" {
		return a, fail(400, "invalid_request", "Invalid OAuth request, resource or PKCE challenge")
	}
	requested := strings.Fields(values.Get("scope"))
	if len(requested) == 0 || len(requested) > 3 {
		return a, fail(400, "invalid_scope", "Request an MCP scope")
	}
	seen := map[string]bool{}
	for _, scope := range requested {
		if !strings.Contains(" "+mcpScopes+" ", " "+scope+" ") || seen[scope] {
			return a, fail(400, "invalid_scope", "Unsupported or duplicate MCP scope")
		}
		seen[scope] = true
	}
	a.Scopes = requested
	var raw []byte
	err := s.DB.QueryRow(ctx, `SELECT name,redirect_uris FROM mcp_oauth_clients WHERE id=$1`, a.ClientID).Scan(&a.ClientName, &raw)
	if err == pgx.ErrNoRows && validMCPMetadataURL(a.ClientID) {
		if err = s.mcpMetadataRegistration(ctx, a.ClientID); err != nil {
			return a, err
		}
		err = s.DB.QueryRow(ctx, `SELECT name,redirect_uris FROM mcp_oauth_clients WHERE id=$1`, a.ClientID).Scan(&a.ClientName, &raw)
	}
	if err != nil {
		return a, fail(400, "invalid_client", "Unknown MCP client")
	}
	var redirects []string
	if json.Unmarshal(raw, &redirects) != nil {
		return a, fail(500, "internal_error", "Invalid client registration")
	}
	for _, redirect := range redirects {
		if redirect == a.RedirectURI {
			return a, nil
		}
	}
	return a, fail(400, "invalid_redirect_uri", "Callback does not match registered client")
}

func (s *Server) mcpAuthorizePage(w http.ResponseWriter, r *http.Request) error {
	a, err := s.parseMCPAuthorization(r.Context(), r.URL.Query())
	if err != nil {
		return err
	}
	if _, err := s.browserOwner(r); err != nil {
		http.Redirect(w, r, "/auth/login?return="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
		return nil
	}
	var fields strings.Builder
	for key, values := range r.URL.Query() {
		for _, value := range values {
			fields.WriteString(`<input type="hidden" name="` + html.EscapeString(key) + `" value="` + html.EscapeString(value) + `">`)
		}
	}
	page(w, "授权 Agent 连接", `<p>客户端：`+html.EscapeString(a.ClientName)+`</p><p>权限：`+html.EscapeString(strings.Join(a.Scopes, ", "))+`</p><form method="post" action="/oauth/authorize">`+fields.String()+`<button name="decision" value="approve">允许连接</button> <button name="decision" value="deny">拒绝</button></form>`)
	return nil
}

func (s *Server) mcpAuthorizeDecision(w http.ResponseWriter, r *http.Request) error {
	if err := s.browserPost(r); err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		return fail(400, "invalid_request", "Invalid authorization form")
	}
	a, err := s.parseMCPAuthorization(r.Context(), r.PostForm)
	if err != nil {
		return err
	}
	owner, err := s.browserOwner(r)
	if err != nil {
		return err
	}
	redirect, _ := url.Parse(a.RedirectURI)
	query := redirect.Query()
	query.Set("state", a.State)
	if r.PostForm.Get("decision") == "deny" {
		query.Set("error", "access_denied")
	} else if r.PostForm.Get("decision") == "approve" {
		code := token()
		err = s.transaction(r.Context(), func(tx pgx.Tx) error {
			if _, err := tx.Exec(r.Context(), `INSERT INTO mcp_oauth_consents(owner_id,client_id,scopes) VALUES($1,$2,$3) ON CONFLICT(owner_id,client_id) DO UPDATE SET scopes=EXCLUDED.scopes,granted_at=now()`, owner, a.ClientID, a.Scopes); err != nil {
				return err
			}
			_, err := tx.Exec(r.Context(), `INSERT INTO mcp_oauth_codes(hash,owner_id,client_id,redirect_uri,resource,scopes,code_challenge,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,now()+interval '5 minutes')`, keyed(s.Config.Secret, code), owner, a.ClientID, a.RedirectURI, a.Resource, a.Scopes, a.Challenge)
			return err
		})
		if err != nil {
			return err
		}
		query.Set("code", code)
	} else {
		return fail(400, "invalid_decision", "Choose allow or deny")
	}
	redirect.RawQuery = query.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusSeeOther)
	return nil
}

func mcpOAuthError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Cache-Control", "no-store")
	jsonResponse(w, status, map[string]string{"error": code})
}

func (s *Server) mcpExchangeCode(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		mcpOAuthError(w, 400, "invalid_request")
		return nil
	}
	if r.PostForm.Get("grant_type") != "authorization_code" || r.PostForm.Get("code") == "" || r.PostForm.Get("client_id") == "" || r.PostForm.Get("redirect_uri") == "" || r.PostForm.Get("resource") != s.Config.ControlOrigin+"/mcp" {
		mcpOAuthError(w, 400, "invalid_request")
		return nil
	}
	verifier := r.PostForm.Get("code_verifier")
	if !mcpPKCEPattern.MatchString(verifier) {
		mcpOAuthError(w, 400, "invalid_grant")
		return nil
	}
	proof := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(proof[:])
	access := token()
	var owner, client, redirect, resource, expected string
	var scopes []string
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		err := tx.QueryRow(r.Context(), `DELETE FROM mcp_oauth_codes WHERE hash=$1 AND expires_at>now() RETURNING owner_id,client_id,redirect_uri,resource,scopes,code_challenge`, keyed(s.Config.Secret, r.PostForm.Get("code"))).Scan(&owner, &client, &redirect, &resource, &scopes, &expected)
		if err != nil {
			return err
		}
		if client != r.PostForm.Get("client_id") || redirect != r.PostForm.Get("redirect_uri") || resource != r.PostForm.Get("resource") || !equal(expected, challenge) {
			return pgx.ErrNoRows
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO mcp_oauth_tokens(hash,owner_id,client_id,resource,scopes,expires_at) VALUES($1,$2,$3,$4,$5,now()+interval '1 hour')`, keyed(s.Config.Secret, access), owner, client, resource, scopes)
		return err
	})
	if err == pgx.ErrNoRows {
		mcpOAuthError(w, 400, "invalid_grant")
		return nil
	}
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "no-store")
	jsonResponse(w, 200, map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": 3600, "scope": strings.Join(scopes, " ")})
	return nil
}

func (s *Server) mcpRevokeToken(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		return fail(400, "invalid_request", "Invalid revocation request")
	}
	if r.PostForm.Get("token") == "" || r.PostForm.Get("client_id") == "" {
		return fail(400, "invalid_request", "Token and client are required")
	}
	_, err := s.DB.Exec(r.Context(), `UPDATE mcp_oauth_tokens SET revoked_at=now() WHERE hash=$1 AND client_id=$2 AND revoked_at IS NULL`, keyed(s.Config.Secret, r.PostForm.Get("token")), r.PostForm.Get("client_id"))
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func (s *Server) mcpPrincipal(ctx context.Context, bearer string) (string, []string, error) {
	if bearer == "" || len(bearer) > 256 {
		return "", nil, fail(401, "authorization_required", "MCP authorization required")
	}
	var owner string
	var scopes []string
	err := s.DB.QueryRow(ctx, `SELECT t.owner_id,t.scopes FROM mcp_oauth_tokens t JOIN mcp_oauth_consents c ON c.owner_id=t.owner_id AND c.client_id=t.client_id WHERE t.hash=$1 AND t.resource=$2 AND t.expires_at>now() AND t.revoked_at IS NULL AND t.scopes <@ c.scopes`, keyed(s.Config.Secret, bearer), s.Config.ControlOrigin+"/mcp").Scan(&owner, &scopes)
	if err == pgx.ErrNoRows {
		return "", nil, fail(401, "authorization_required", "MCP authorization expired or revoked")
	}
	return owner, scopes, err
}

func mcpHasScope(scopes []string, wanted string) bool {
	for _, value := range scopes {
		if value == wanted {
			return true
		}
	}
	return false
}
