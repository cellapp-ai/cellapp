package hosting

import (
	"bytes"
	"cellapp/apps/server/appdata"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var tablePath = regexp.MustCompile(`^/_cellapp/db/([a-z][a-z0-9_]{0,62})$`)
var anonymousDataSlots = make(chan struct{}, 8)

var dataHTTP = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy:               nil,
		MaxConnsPerHost:     10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	},
}

func (s *Server) dataToken(role, appUserID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claimsMap := map[string]any{"role": role, "exp": time.Now().Add(30 * time.Second).Unix()}
	if appUserID != "" {
		claimsMap["app_user_id"] = appUserID
	}
	claims, _ := json.Marshal(claimsMap)
	payload := base64.RawURLEncoding.EncodeToString(claims)
	message := header + "." + payload
	mac := hmac.New(sha256.New, []byte(s.Config.AppDataJWTSecret))
	_, _ = mac.Write([]byte(message))
	return message + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) DataProbe(ctx context.Context, names appdata.Names) error {
	for attempt := 0; attempt < 20; attempt++ {
		target := s.Config.PostgRESTURL + "/_cellapp_ready?select=version&limit=0"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+s.dataToken(names.APIRole, id()))
		req.Header.Set("Accept-Profile", names.Schema)
		response, err := dataHTTP.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
			response.Body.Close()
			if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusPartialContent {
				return nil
			}
			err = fmt.Errorf("PostgREST returned %d", response.StatusCode)
		}
		if attempt == 19 {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("PostgREST probe exhausted")
}

func (s *Server) dataTableProbe(ctx context.Context, names appdata.Names, manifest appdata.Manifest) error {
	for _, table := range manifest.Tables {
		ready := false
		for attempt := 0; attempt < 30; attempt++ {
			target := s.Config.PostgRESTURL + "/" + table.Name + "?select=" + url.QueryEscape(table.PrimaryKey[0]) + "&limit=0"
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+s.dataToken(names.APIRole, id()))
			req.Header.Set("Accept-Profile", names.Schema)
			response, err := dataHTTP.Do(req)
			if err == nil {
				body, readError := io.ReadAll(io.LimitReader(response.Body, 4096))
				response.Body.Close()
				if readError != nil {
					return readError
				}
				if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusPartialContent {
					ready = true
					break
				}
				var detail struct {
					Code string `json:"code"`
				}
				if json.Unmarshal(body, &detail) == nil && detail.Code == "42501" {
					ready = true // The table is cached but this operation is intentionally ungranted.
					break
				}
				err = fmt.Errorf("PostgREST table probe returned %d", response.StatusCode)
			}
			if attempt == 29 {
				return fmt.Errorf("table %s: %w", table.Name, err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
		if !ready {
			return fmt.Errorf("table %s was not available", table.Name)
		}
	}
	return nil
}

func (s *Server) serveData(w http.ResponseWriter, r *http.Request, appID string) error {
	contract, err := selectAppAPIContract(r)
	if err != nil {
		return err
	}
	match := tablePath.FindStringSubmatch(r.URL.Path)
	if match == nil || strings.HasPrefix(match[1], "_cellapp_") {
		return fail(404, "not_found", "Data resource not found")
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPatch, http.MethodDelete:
	default:
		return fail(405, "method_not_allowed", "Unsupported data operation")
	}
	if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
		if r.Header.Get("Origin") != "https://"+s.requestHost(r) {
			return fail(403, "origin_rejected", "Invalid data request origin")
		}
	}
	if s.AppDataDB == nil {
		return fail(503, "app_data_not_ready", "Application data is not ready")
	}
	if r.ContentLength > 1<<20 {
		return fail(413, "request_too_large", "Data request is too large")
	}
	requestBody, err := io.ReadAll(io.LimitReader(r.Body, 1<<20+1))
	if err != nil {
		return fail(400, "invalid_body", "Could not read data request")
	}
	if len(requestBody) > 1<<20 {
		return fail(413, "request_too_large", "Data request is too large")
	}
	anonymous := cookieValue(r, "__Host-cellapp") == ""
	var platformUser string
	if !anonymous {
		platformUser, err = s.appSession(r, appID)
		if err != nil {
			return err
		}
	} else {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
			return fail(401, "login_required", "Sign in to Cellapp")
		}
		operation := "read"
		limit := int64(60)
		if r.Method == http.MethodPost {
			operation, limit = "write", 20
		}
		if err := s.rate(r.Context(), "anonymous-data:"+operation+":"+appID+":"+s.rateSource(r), limit); err != nil {
			return err
		}
		select {
		case anonymousDataSlots <- struct{}{}:
			defer func() { <-anonymousDataSlots }()
		default:
			return fail(429, "anonymous_concurrency_limit", "Too many anonymous data requests")
		}
	}
	names, err := appdata.AppNames(appID)
	if err != nil {
		return fail(404, "app_not_found", "Application not found")
	}
	var resultStatus int
	var resultHeader http.Header
	var resultBody []byte
	err = s.transaction(r.Context(), func(tx pgx.Tx) error {
		var dataState, visibility string
		var pages, read, write bool
		if err := tx.QueryRow(r.Context(), `SELECT data_state,visibility,anonymous_pages,anonymous_read,anonymous_write FROM apps WHERE id=$1 AND NOT deleted AND NOT suspended FOR SHARE`, appID).Scan(&dataState, &visibility, &pages, &read, &write); err != nil {
			return fail(404, "app_not_found", "Application not found")
		}
		if dataState != "ready" {
			return fail(503, "app_data_not_ready", "Application data is not ready")
		}
		role, appUserID := names.APIRole, ""
		minimalInsert := false
		if anonymous {
			if visibility != "public" || !pages {
				return fail(401, "login_required", "Sign in to Cellapp")
			}
			if r.Method == http.MethodPost && (!write || !s.Config.AllowAnonymousAppDataWrites) || r.Method != http.MethodPost && !read {
				return fail(403, "anonymous_data_disabled", "Anonymous data access is disabled")
			}
			manifest, _, err := appdata.Latest(r.Context(), s.AppDataDB, appID)
			if err != nil {
				return err
			}
			var policy appdata.Policies
			found := false
			if manifest != nil {
				for _, table := range manifest.Tables {
					if table.Name == match[1] {
						policy, found = table.Policies, true
						break
					}
				}
			}
			if !found || r.Method == http.MethodPost && policy.AnonymousInsert != "all" || r.Method != http.MethodPost && policy.AnonymousSelect != "all" {
				return fail(403, "anonymous_table_denied", "Table does not allow anonymous access")
			}
			minimalInsert = r.Method == http.MethodPost && (!read || policy.AnonymousSelect != "all")
			role = names.AnonymousRole
		} else {
			access, err := appAccess(r, tx, appID, platformUser)
			if err != nil {
				return err
			}
			if !access.allowed() {
				return fail(403, "access_required", "Application access required")
			}
			if err := admitVisitor(r.Context(), tx, appID, platformUser, access.Owner, s.Config.Limits.MonthlyVisitors, time.Now()); err != nil {
				return err
			}
			appUserID, err = s.appUser(r, tx, appID, platformUser)
			if err != nil {
				return err
			}
		}
		target, err := url.Parse(s.Config.PostgRESTURL)
		if err != nil {
			return err
		}
		target.Path = "/" + match[1]
		target.RawQuery, err = appAPIDataQuery(contract, r.URL.Query())
		if err != nil {
			return err
		}
		upstream, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), bytes.NewReader(requestBody))
		if err != nil {
			return err
		}
		for _, name := range []string{"Accept", "Content-Type", "Prefer", "Range", "If-Match", "If-None-Match"} {
			if value := r.Header.Get(name); value != "" {
				upstream.Header.Set(name, value)
			}
		}
		if minimalInsert {
			upstream.Header.Set("Prefer", "return=minimal")
		}
		upstream.Header.Set("Authorization", "Bearer "+s.dataToken(role, appUserID))
		if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			upstream.Header.Set("Content-Profile", names.Schema)
		} else {
			upstream.Header.Set("Accept-Profile", names.Schema)
		}
		response, err := dataHTTP.Do(upstream)
		if err != nil {
			return fail(503, "data_gateway_unavailable", "Application data service unavailable")
		}
		defer response.Body.Close()
		resultBody, err = io.ReadAll(io.LimitReader(response.Body, 2<<20+1))
		if err != nil {
			return err
		}
		if len(resultBody) > 2<<20 {
			return fail(413, "response_too_large", "Data response exceeded limit")
		}
		resultStatus = response.StatusCode
		resultHeader = response.Header.Clone()
		return nil
	})
	if err != nil {
		return err
	}
	for _, name := range []string{"Content-Type", "Content-Range", "Preference-Applied", "ETag"} {
		if value := resultHeader.Get(name); value != "" {
			w.Header().Set(name, value)
		}
	}
	w.WriteHeader(resultStatus)
	if r.Method != http.MethodHead {
		_, _ = w.Write(resultBody)
	}
	return nil
}
