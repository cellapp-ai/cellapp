package hosting

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type modelCapability struct {
	Supported bool   `json:"supported"`
	Error     string `json:"error,omitempty"`
}

type modelCapabilities struct {
	Text   modelCapability `json:"text"`
	Stream modelCapability `json:"stream"`
	Tools  modelCapability `json:"tools"`
}

func modelProbeBody(model, kind string) []byte {
	payload := map[string]any{
		"model": model, "messages": []map[string]string{{"role": "user", "content": "Reply with pong."}}, "max_tokens": 32,
	}
	if kind == "stream" {
		payload["stream"] = true
	}
	if kind == "tools" {
		payload["tools"] = []any{map[string]any{"type": "function", "function": map[string]any{
			"name": "ping", "description": "Return pong", "parameters": map[string]any{"type": "object", "properties": map[string]any{}},
		}}}
		payload["tool_choice"] = "required"
	}
	data, _ := json.Marshal(payload)
	return data
}

func classifyModelProbe(err error, status int) string {
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() || errors.Is(err, context.DeadlineExceeded) {
			return "timeout"
		}
		if strings.Contains(err.Error(), "restricted address") || strings.Contains(err.Error(), "not public") {
			return "unsafe_target"
		}
		return "network_error"
	}
	if status == 401 || status == 403 {
		return "authentication_failed"
	}
	if status == 429 {
		return "rate_limited"
	}
	if status >= 500 {
		return "provider_unavailable"
	}
	if status < 200 || status >= 300 {
		return "provider_rejected"
	}
	return "invalid_response"
}

func probeJSONResponse(data []byte, kind string) bool {
	var response struct {
		Choices []struct {
			Message struct {
				Content   any               `json:"content"`
				ToolCalls []json.RawMessage `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &response) != nil || len(response.Choices) == 0 {
		return false
	}
	if kind == "tools" {
		return len(response.Choices[0].Message.ToolCalls) > 0
	}
	content, ok := response.Choices[0].Message.Content.(string)
	return ok && content != ""
}

func probeStreamResponse(reader io.Reader) bool {
	scanner := bufio.NewScanner(io.LimitReader(reader, 4<<20+1))
	scanner.Buffer(make([]byte, 4096), 256<<10)
	content := false
	done := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		value := strings.TrimPrefix(line, "data: ")
		if value == "[DONE]" {
			done = true
			continue
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(value), &event) != nil {
			return false
		}
		if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
			content = true
		}
	}
	return scanner.Err() == nil && content && done
}

func (s *Server) probeModelCapability(ctx context.Context, base, key, model, kind string) modelCapability {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := newModelChatRequest(ctx, base, key, modelProbeBody(model, kind))
	if err != nil {
		return modelCapability{Error: classifyModelProbe(err, 0)}
	}
	client := s.ModelTestHTTPClient
	if client == nil {
		client = modelHTTPClient(net.DefaultResolver)
	}
	response, err := client.Do(request)
	if err != nil {
		return modelCapability{Error: classifyModelProbe(err, 0)}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return modelCapability{Error: classifyModelProbe(nil, response.StatusCode)}
	}
	if kind == "stream" {
		if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") || !probeStreamResponse(response.Body) {
			return modelCapability{Error: "invalid_stream"}
		}
		return modelCapability{Supported: true}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20+1))
	if err != nil || len(data) > 4<<20 {
		return modelCapability{Error: "invalid_response"}
	}
	if !probeJSONResponse(data, kind) {
		if kind == "tools" {
			return modelCapability{Error: "tool_call_unsupported"}
		}
		return modelCapability{Error: "invalid_response"}
	}
	return modelCapability{Supported: true}
}

func (s *Server) webTestModelConnection(w http.ResponseWriter, r *http.Request, owner string) error {
	connection := r.PathValue("connection")
	if !idPattern.MatchString(connection) {
		return fail(404, "model_connection_not_found", "Model connection not found")
	}
	var base, model string
	var encrypted []byte
	var version int
	err := s.DB.QueryRow(r.Context(), `SELECT api_base,model_id,encrypted_key,key_version FROM model_connections WHERE id=$1 AND owner_id=$2`, connection, owner).Scan(&base, &model, &encrypted, &version)
	if err == pgx.ErrNoRows {
		return fail(404, "model_connection_not_found", "Model connection not found")
	}
	if err != nil {
		return err
	}
	key, err := s.decryptModelKey(owner, connection, encrypted, version)
	if err != nil {
		return fail(503, "model_secret_unavailable", "Model connection is unavailable")
	}
	capabilities := modelCapabilities{
		Text:   s.probeModelCapability(r.Context(), base, key, model, "text"),
		Stream: s.probeModelCapability(r.Context(), base, key, model, "stream"),
		Tools:  s.probeModelCapability(r.Context(), base, key, model, "tools"),
	}
	encoded, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("encode model capabilities: %w", err)
	}
	result, err := s.DB.Exec(r.Context(), `UPDATE model_connections SET capabilities=$1,updated_at=now()
	WHERE id=$2 AND owner_id=$3 AND encrypted_key=$4`, encoded, connection, owner, encrypted)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fail(409, "model_connection_changed", "Connection changed during test; test again")
	}
	jsonResponse(w, 200, capabilities)
	return nil
}
