package hosting

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestModelStreamingAndUsageStates(t *testing.T) {
	f := setup(t)
	f.server.Config.ModelEncryptionKey = []byte(strings.Repeat("m", 32))
	app, _ := f.create(t)
	hostURL, _ := url.Parse(app.URL)
	session := f.webSessionCookie(t)
	cookie := signedInApp(t, f, app, session)
	connection := response[modelConnectionView](t, f.webRequest(t, "POST", "/model-connections", map[string]string{
		"name": "Personal", "apiBase": "https://models.example.com/v1", "modelId": "model-one", "apiKey": "provider-secret",
	}, session, f.server.Config.ControlOrigin), 201)
	if _, err := f.db.Exec(context.Background(), `UPDATE model_connections SET capabilities='{"text":{"supported":true},"stream":{"supported":true},"tools":{"supported":true}}' WHERE id=$1`, connection.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO model_connection_grants(owner_id,app_id,connection_id) VALUES($1,$2,$3)`, f.owner, app.ID, connection.ID); err != nil {
		t.Fatal(err)
	}
	mode := "full"
	f.server.ModelTestHTTPClient = &http.Client{Transport: modelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		status, contentType, body := 200, "text/event-stream", "data: {\"choices\":[{\"delta\":{\"content\":\"pong\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":1}}\n\ndata: [DONE]\n\n"
		switch mode {
		case "broken":
			body = "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"
		case "no-usage":
			body = "data: {\"choices\":[{\"delta\":{\"content\":\"pong\"}}]}\n\ndata: [DONE]\n\n"
		case "provider-error":
			status = 503
		case "empty":
			body = "data: [DONE]\n\n"
		case "tools":
			body = "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call-1\",\"type\":\"function\",\"function\":{\"name\":\"ping\",\"arguments\":\"{}\"}}]}}]}\n\ndata: [DONE]\n\n"
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	input := map[string]any{"model": cellappModelID(connection.ID, "model-one"), "messages": []map[string]string{{"role": "user", "content": "ping"}}, "stream": true}
	checkUsage := func(want string, inputTokens *int64) {
		t.Helper()
		var got string
		var tokens *int64
		if err := f.db.QueryRow(context.Background(), `SELECT status,input_tokens FROM model_request_usage WHERE app_id=$1 ORDER BY started_at DESC,id DESC LIMIT 1`, app.ID).Scan(&got, &tokens); err != nil {
			t.Fatal(err)
		}
		if got != want || (inputTokens == nil) != (tokens == nil) || inputTokens != nil && *inputTokens != *tokens {
			t.Fatalf("usage status=%s tokens=%v, want %s %v", got, tokens, want, inputTokens)
		}
	}
	tokens := int64(3)
	for _, tc := range []struct {
		mode, expectedStatus string
		usage                *int64
		contains             string
	}{
		{"full", "succeeded", &tokens, "data: [DONE]"},
		{"broken", "failed", nil, "model_stream_interrupted"},
		{"no-usage", "succeeded", nil, "data: [DONE]"},
		{"provider-error", "failed", nil, "model_provider_unavailable"},
		{"tools", "succeeded", nil, "tool_calls"},
		{"empty", "failed", nil, "model_stream_interrupted"},
	} {
		mode = tc.mode
		if mode == "tools" {
			input["tools"] = []any{map[string]any{"type": "function", "function": map[string]any{"name": "ping", "parameters": map[string]string{"type": "object"}}}}
		}
		got := modelChatRequest(f, "POST", "/_cellapp/v1/chat/completions", hostURL.Host, cookie, "https://"+hostURL.Host, input)
		if !strings.Contains(got.Body.String(), tc.contains) {
			t.Fatalf("%s output: %d %s", mode, got.Code, got.Body.String())
		}
		checkUsage(tc.expectedStatus, tc.usage)
	}
	started := make(chan struct{})
	f.server.ModelTestHTTPClient = &http.Client{Transport: modelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	data, _ := json.Marshal(input)
	requestContext, cancel := context.WithCancel(context.Background())
	r := httptest.NewRequest("POST", "https://"+hostURL.Host+"/_cellapp/v1/chat/completions", bytes.NewReader(data)).WithContext(requestContext)
	r.Host = hostURL.Host
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Cookie", cookie)
	r.Header.Set("Origin", "https://"+hostURL.Host)
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() { f.handler.ServeHTTP(w, r); close(done) }()
	<-started
	cancel()
	<-done
	checkUsage("cancelled", nil)
}
