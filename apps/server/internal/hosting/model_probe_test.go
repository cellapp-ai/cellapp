package hosting

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type modelRoundTripFunc func(*http.Request) (*http.Response, error)

func (f modelRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestModelCapabilityProbe(t *testing.T) {
	f := setup(t)
	f.server.Config.ModelEncryptionKey = []byte(strings.Repeat("m", 32))
	session := f.webSessionCookie(t)
	origin := f.server.Config.ControlOrigin
	created := response[modelConnectionView](t, f.webRequest(t, "POST", "/model-connections", map[string]string{
		"name": "Personal", "apiBase": "https://models.example.com/v1", "modelId": "model-one", "apiKey": "provider-secret",
	}, session, origin), 201)
	compatible := false
	calls := 0
	f.server.ModelTestHTTPClient = &http.Client{Transport: modelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer provider-secret" {
			t.Fatal("probe request was not fixed to the selected connection")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if json.Unmarshal(body, &payload) != nil || payload["model"] != "model-one" {
			t.Fatal("invalid probe body")
		}
		status, contentType, result := 200, "application/json", `{"choices":[{"message":{"content":"pong"}}]}`
		if payload["stream"] == true {
			contentType = "text/event-stream"
			if compatible {
				result = "data: {\"choices\":[{\"delta\":{\"content\":\"pong\"}}]}\n\ndata: [DONE]\n\n"
			} else {
				result = "data: malformed\n\n"
			}
		} else if payload["tools"] != nil {
			if compatible {
				result = `{"choices":[{"message":{"tool_calls":[{"id":"call-1","type":"function","function":{"name":"ping","arguments":"{}"}}]}}]}`
			} else {
				status = 400
			}
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(result))}, nil
	})}
	partial := response[modelCapabilities](t, f.webRequest(t, "POST", "/model-connections/"+created.ID+"/test", nil, session, origin), 200)
	if !partial.Text.Supported || partial.Stream.Supported || partial.Stream.Error != "invalid_stream" || partial.Tools.Supported || partial.Tools.Error != "provider_rejected" {
		t.Fatal(partial)
	}
	if calls != 3 {
		t.Fatalf("expected three independent capability probes, got %d", calls)
	}
	var stored []byte
	if err := f.db.QueryRow(context.Background(), `SELECT capabilities FROM model_connections WHERE id=$1`, created.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stored), `"invalid_stream"`) || strings.Contains(string(stored), "provider-secret") {
		t.Fatal("capability status was not safely stored")
	}
	compatible = true
	full := response[modelCapabilities](t, f.webRequest(t, "POST", "/model-connections/"+created.ID+"/test", nil, session, origin), 200)
	if !full.Text.Supported || !full.Stream.Supported || !full.Tools.Supported {
		t.Fatal(full)
	}
	if strings.Contains(f.logs.String(), "provider-secret") {
		t.Fatal("probe leaked provider secret to logs")
	}
}
