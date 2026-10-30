package hosting

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type modelChatInput struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream              bool              `json:"stream"`
	Tools               []json.RawMessage `json:"tools,omitempty"`
	MaxTokens           *int              `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int              `json:"max_completion_tokens,omitempty"`
	Temperature         *float64          `json:"temperature,omitempty"`
	TopP                *float64          `json:"top_p,omitempty"`
}

func (input modelChatInput) valid() bool {
	if _, _, ok := parseCellappModelID(input.Model); !ok || len(input.Messages) == 0 || len(input.Messages) > 32 || len(input.Tools) > 8 {
		return false
	}
	if input.MaxTokens != nil && input.MaxCompletionTokens != nil ||
		input.MaxTokens != nil && (*input.MaxTokens < 1 || *input.MaxTokens > 1024) ||
		input.MaxCompletionTokens != nil && (*input.MaxCompletionTokens < 1 || *input.MaxCompletionTokens > 1024) ||
		input.Temperature != nil && (math.IsNaN(*input.Temperature) || math.IsInf(*input.Temperature, 0) || *input.Temperature < 0 || *input.Temperature > 2) ||
		input.TopP != nil && (math.IsNaN(*input.TopP) || math.IsInf(*input.TopP, 0) || *input.TopP < 0 || *input.TopP > 1) {
		return false
	}
	for _, message := range input.Messages {
		if message.Role != "system" && message.Role != "user" && message.Role != "assistant" || message.Content == "" || len(message.Content) > 32<<10 {
			return false
		}
	}
	for _, raw := range input.Tools {
		var tool struct {
			Type     string `json:"type"`
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		if len(raw) > 16<<10 || json.Unmarshal(raw, &tool) != nil || tool.Type != "function" || tool.Function.Name == "" || len(tool.Function.Name) > 64 {
			return false
		}
	}
	return true
}

func (s *Server) modelChat(w http.ResponseWriter, r *http.Request, appID string) error {
	if r.Header.Get("Origin") != "https://"+s.requestHost(r) {
		return fail(403, "origin_rejected", "Invalid application origin")
	}
	if _, present := r.URL.Query()["v"]; present {
		contract, err := selectAppAPIContract(r)
		if err != nil {
			return err
		}
		if contract.Version != "1.2" {
			return fail(400, "unsupported_api_version", "Model calls require application API version 1.2")
		}
	}
	user, err := s.appSession(r, appID)
	if err != nil {
		return err
	}
	var input modelChatInput
	if err := decode(w, r, &input); err != nil {
		return err
	}
	if !input.valid() {
		return fail(400, "invalid_model_request", "Invalid model request")
	}
	if err := s.rate(r.Context(), "model:"+user+":"+appID, 20); err != nil {
		return err
	}
	requestID := id()
	target, err := s.authorizeAndReserveModelRequest(r, user, appID, requestID, input)
	if err != nil {
		return err
	}
	status := "failed"
	var inputTokens, outputTokens *int64
	defer func() {
		finish, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if r.Context().Err() != nil {
			status = "cancelled"
		}
		if _, err := s.DB.Exec(finish, `UPDATE model_request_usage SET status=$1,input_tokens=$2,output_tokens=$3,finished_at=now() WHERE id=$4 AND status='started'`, status, inputTokens, outputTokens, requestID); err != nil {
			s.Logger.Error("model_usage_update_failed", "request_id", requestID)
		}
	}()
	upstream := map[string]any{"model": target.model, "messages": input.Messages, "max_tokens": 1024, "stream": input.Stream}
	if input.MaxTokens != nil {
		upstream["max_tokens"] = *input.MaxTokens
	}
	if input.MaxCompletionTokens != nil {
		delete(upstream, "max_tokens")
		upstream["max_completion_tokens"] = *input.MaxCompletionTokens
	}
	if input.Temperature != nil {
		upstream["temperature"] = *input.Temperature
	}
	if input.TopP != nil {
		upstream["top_p"] = *input.TopP
	}
	if len(input.Tools) > 0 {
		upstream["tools"] = input.Tools
	}
	if input.Stream {
		upstream["stream_options"] = map[string]bool{"include_usage": true}
	}
	body, err := json.Marshal(upstream)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	request, err := newModelChatRequest(ctx, target.base, target.key, body)
	if err != nil {
		return err
	}
	client := s.ModelTestHTTPClient
	if client == nil {
		client = modelHTTPClient(net.DefaultResolver)
	}
	response, err := client.Do(request)
	if err != nil {
		var networkError net.Error
		if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded || errors.As(err, &networkError) && networkError.Timeout() {
			return fail(504, "model_upstream_timeout", "Model provider timed out")
		}
		return fail(502, "model_upstream_unavailable", "Model provider could not be reached")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		switch response.StatusCode {
		case 401, 403:
			return fail(502, "model_provider_authentication_failed", "Model provider rejected the connection key")
		case 429:
			return fail(502, "model_provider_rate_limited", "Model provider rate limit reached")
		default:
			if response.StatusCode >= 500 {
				return fail(502, "model_provider_unavailable", "Model provider is unavailable")
			}
			return fail(502, "model_upstream_rejected", "Model provider rejected the request")
		}
	}
	if input.Stream {
		if !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
			return fail(502, "model_invalid_stream", "Model provider returned an invalid stream")
		}
		finished, in, out := forwardModelStream(w, response.Body)
		inputTokens, outputTokens = in, out
		if finished {
			status = "succeeded"
		}
		return nil
	}
	data, err := modelResponseBody(response, 4<<20)
	if err != nil || !probeJSONResponse(data, map[bool]string{true: "tools", false: "text"}[len(input.Tools) > 0]) {
		return fail(502, "model_invalid_response", "Model provider returned an invalid response")
	}
	var parsed struct {
		Usage struct {
			PromptTokens     *int64 `json:"prompt_tokens"`
			CompletionTokens *int64 `json:"completion_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(data, &parsed) == nil {
		inputTokens, outputTokens = safeModelTokens(parsed.Usage.PromptTokens), safeModelTokens(parsed.Usage.CompletionTokens)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	_, _ = w.Write(data)
	status = "succeeded"
	return nil
}

type modelRequestTarget struct {
	base, model, key, connectionID string
}

func (s *Server) authorizeAndReserveModelRequest(r *http.Request, user, appID, requestID string, input modelChatInput) (modelRequestTarget, error) {
	var target modelRequestTarget
	err := s.transaction(r.Context(), func(tx pgx.Tx) error {
		access, err := appAccess(r, tx, appID, user)
		if err != nil {
			return err
		}
		if !access.allowed() {
			return fail(403, "access_required", "Application access is required")
		}
		connectionID, upstreamModel, _ := parseCellappModelID(input.Model)
		var raw, encrypted []byte
		var keyVersion int
		err = tx.QueryRow(r.Context(), `SELECT c.id,c.api_base,c.model_id,c.encrypted_key,c.key_version,c.capabilities
		FROM model_connection_grants g JOIN model_connections c ON c.id=g.connection_id AND c.owner_id=g.owner_id
		WHERE g.owner_id=$1 AND g.app_id=$2 AND g.connection_id=$3 AND g.revoked_at IS NULL FOR SHARE OF g,c`, user, appID, connectionID).Scan(&target.connectionID, &target.base, &target.model, &encrypted, &keyVersion, &raw)
		if err == pgx.ErrNoRows {
			return &APIError{Status: 403, Code: "model_authorization_required", Message: "Authorize this application to use your model connection", Details: map[string]string{"authorizeUrl": s.Config.ControlOrigin + "/dashboard/model-consent?" + url.Values{"app": {appID}}.Encode()}}
		}
		if err != nil {
			return err
		}
		testedModel := target.model
		target.model = upstreamModel
		var capabilities modelCapabilities
		if err := json.Unmarshal(raw, &capabilities); err != nil {
			return fail(503, "model_connection_unavailable", "Model connection needs testing")
		}
		if upstreamModel == testedModel && (!capabilities.Text.Supported || input.Stream && !capabilities.Stream.Supported || len(input.Tools) > 0 && !capabilities.Tools.Supported) {
			return fail(409, "model_capability_unavailable", "Test and enable the required model capability")
		}
		key, err := s.decryptModelKey(user, target.connectionID, encrypted, keyVersion)
		if err != nil {
			return fail(503, "model_secret_unavailable", "Model connection is unavailable")
		}
		target.key = key
		if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtext($1)::bigint)`, user+":"+appID); err != nil {
			return err
		}
		var count int
		if err := tx.QueryRow(r.Context(), `SELECT count(*) FROM model_request_usage WHERE owner_id=$1 AND app_id=$2 AND status='started' AND started_at>now()-interval '2 minutes'`, user, appID).Scan(&count); err != nil {
			return err
		}
		if count >= 2 {
			return fail(429, "model_concurrency_limit", "Too many model requests are running")
		}
		_, err = tx.Exec(r.Context(), `INSERT INTO model_request_usage(id,owner_id,app_id,connection_id,status) VALUES($1,$2,$3,$4,'started')`, requestID, user, appID, target.connectionID)
		return err
	})
	return target, err
}

func forwardModelStream(w http.ResponseWriter, reader io.Reader) (bool, *int64, *int64) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return false, nil, nil
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(200)
	scanner := bufio.NewScanner(io.LimitReader(reader, 4<<20+1))
	scanner.Buffer(make([]byte, 4096), 256<<10)
	var in, out *int64
	complete := false
	seenOutput := false
	writeOK := true
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		value := strings.TrimPrefix(line, "data: ")
		if value == "[DONE]" {
			if !seenOutput {
				break
			}
			complete = true
			if _, err := io.WriteString(w, "data: [DONE]\n\n"); err != nil {
				writeOK = false
			}
			flusher.Flush()
			break
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content   string            `json:"content"`
					ToolCalls []json.RawMessage `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     *int64 `json:"prompt_tokens"`
				CompletionTokens *int64 `json:"completion_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal([]byte(value), &event) != nil || len(event.Choices) == 0 && event.Usage.PromptTokens == nil && event.Usage.CompletionTokens == nil {
			break
		}
		for _, choice := range event.Choices {
			if choice.Delta.Content != "" || len(choice.Delta.ToolCalls) > 0 {
				seenOutput = true
			}
		}
		if event.Usage.PromptTokens != nil {
			in = safeModelTokens(event.Usage.PromptTokens)
		}
		if event.Usage.CompletionTokens != nil {
			out = safeModelTokens(event.Usage.CompletionTokens)
		}
		if _, err := io.WriteString(w, "data: "+value+"\n\n"); err != nil {
			writeOK = false
			break
		}
		flusher.Flush()
	}
	if !complete {
		_, _ = io.WriteString(w, "event: error\ndata: {\"error\":\"model_stream_interrupted\"}\n\n")
		flusher.Flush()
	}
	return complete && scanner.Err() == nil && writeOK, in, out
}

func safeModelTokens(value *int64) *int64 {
	if value == nil || *value < 0 {
		return nil
	}
	return value
}
