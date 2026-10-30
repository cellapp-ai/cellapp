package hosting

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModelConnectionSecretsAndOwnership(t *testing.T) {
	f := setup(t)
	f.server.Config.ModelEncryptionKey = bytes.Repeat([]byte("m"), 32)
	session := f.webSessionCookie(t)
	origin := f.server.Config.ControlOrigin
	oldKey, newKey := "test-provider-secret-old", "test-provider-secret-new"
	input := map[string]string{"name": "Personal", "apiBase": "https://models.example.com/v1", "modelId": "model-one", "apiKey": oldKey}
	response[any](t, f.webRequest(t, "POST", "/model-connections", input, "", origin), 401)
	response[any](t, f.webRequest(t, "POST", "/model-connections", input, session, ""), 403)
	response[any](t, f.webRequest(t, "POST", "/model-connections", input, session, "https://outside.example"), 403)
	created := response[modelConnectionView](t, f.webRequest(t, "POST", "/model-connections", input, session, origin), 201)
	if created.Name != "Personal" || created.APIBase != input["apiBase"] || created.ModelID != input["modelId"] || created.ID == "" {
		t.Fatal(created)
	}
	listed := f.webRequest(t, "GET", "/model-connections", nil, session, "")
	response[[]modelConnectionView](t, listed, 200)
	if strings.Contains(listed.Body.String(), oldKey) || strings.Contains(f.logs.String(), oldKey) {
		t.Fatal("model key leaked in list or logs")
	}
	var ciphertext []byte
	if err := f.db.QueryRow(context.Background(), `SELECT encrypted_key FROM model_connections WHERE id=$1`, created.ID).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ciphertext), oldKey) {
		t.Fatal("model key stored as plaintext")
	}
	if key, err := f.server.modelKeyForOwner(httpRequestForModel(), f.owner, created.ID); err != nil || key != oldKey {
		t.Fatalf("stored key unavailable: %v", err)
	}
	other, otherSession := id(), token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO owners(id,issuer,subject) VALUES($1,'test',$1)`, other); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($2,$1,now()+interval '1 day')`, other, keyed(f.server.Config.Secret, otherSession)); err != nil {
		t.Fatal(err)
	}
	if items := response[[]modelConnectionView](t, f.webRequest(t, "GET", "/model-connections", nil, otherSession, ""), 200); len(items) != 0 {
		t.Fatal(items)
	}
	response[any](t, f.webRequest(t, "POST", "/model-connections/"+created.ID+"/key", map[string]string{"apiKey": newKey}, otherSession, origin), 404)
	response[any](t, f.webRequest(t, "DELETE", "/model-connections/"+created.ID, nil, otherSession, origin), 404)
	response[any](t, f.webRequest(t, "POST", "/model-connections/"+created.ID+"/key", map[string]string{"apiKey": newKey}, session, origin), 200)
	if key, err := f.server.modelKeyForOwner(httpRequestForModel(), f.owner, created.ID); err != nil || key != newKey {
		t.Fatalf("rotation did not replace prior key: %v", err)
	}
	if strings.Contains(f.logs.String(), newKey) {
		t.Fatal("rotated key leaked to logs")
	}
	response[any](t, f.webRequest(t, "DELETE", "/model-connections/"+created.ID, nil, session, origin), 200)
	if _, err := f.server.modelKeyForOwner(httpRequestForModel(), f.owner, created.ID); err == nil {
		t.Fatal("deleted connection still usable")
	}
}

func httpRequestForModel() *http.Request {
	return httptest.NewRequest(http.MethodGet, "/", nil)
}

func TestModelConnectionDiscoveryAndManualFallback(t *testing.T) {
	f := setup(t)
	f.server.Config.ModelEncryptionKey = bytes.Repeat([]byte("m"), 32)
	session := f.webSessionCookie(t)
	f.server.ModelTestHTTPClient = &http.Client{Transport: modelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer provider-secret" {
			t.Fatal("unexpected discovery request")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"discovered-model"}]}`))}, nil
	})}
	input := map[string]string{"name": "Discovered", "apiBase": "https://models.example.com/v1", "apiKey": "provider-secret"}
	discovered := response[modelConnectionView](t, f.webRequest(t, "POST", "/model-connections", input, session, f.server.Config.ControlOrigin), 201)
	if discovered.ModelID != "discovered-model" {
		t.Fatalf("model ID not discovered: %+v", discovered)
	}
	f.server.ModelTestHTTPClient = &http.Client{Transport: modelRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("unavailable"))}, nil
	})}
	if got := f.webRequest(t, "POST", "/model-connections", input, session, f.server.Config.ControlOrigin); got.Code != 400 || !strings.Contains(got.Body.String(), "model_discovery_failed") {
		t.Fatalf("failed discovery did not request manual fallback: %d %s", got.Code, got.Body.String())
	}
	input["modelId"] = "manual-model"
	manual := response[modelConnectionView](t, f.webRequest(t, "POST", "/model-connections", input, session, f.server.Config.ControlOrigin), 201)
	if manual.ModelID != "manual-model" {
		t.Fatalf("manual fallback unavailable: %+v", manual)
	}
}

func TestModelEncryptionConfiguration(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte("m"), 32))
	for _, input := range []string{"bad", base64.RawURLEncoding.EncodeToString([]byte("short"))} {
		if _, err := LoadConfig(func(name string) string {
			if name == "MODEL_ENCRYPTION_KEY" {
				return input
			}
			return ""
		}); err == nil {
			t.Fatal("invalid model encryption key accepted")
		}
	}
	config, err := LoadConfig(func(name string) string {
		if name == "MODEL_ENCRYPTION_KEY" {
			return key
		}
		return ""
	})
	if err != nil || string(config.ModelEncryptionKey) != strings.Repeat("m", 32) {
		t.Fatalf("valid model encryption key rejected: %v", err)
	}
}
