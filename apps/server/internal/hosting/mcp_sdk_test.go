package hosting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Keep this wire test while the server targets the Go 1.24-compatible SDK line.
func TestMCPGoSDKStreamableHTTP(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "cellapp-sdk-check", Version: "1.0.0"}, nil)
	type input struct {
		Name string `json:"name"`
	}
	type output struct {
		Greeting string `json:"greeting"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "Greet a named caller"},
		func(_ context.Context, _ *mcp.CallToolRequest, in input) (*mcp.CallToolResult, output, error) {
			return nil, output{Greeting: "Hello, " + in.Name}, nil
		})
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	))
	defer httpServer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "cellapp-sdk-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	listed, err := session.ListTools(ctx, nil)
	if err != nil || len(listed.Tools) != 1 || listed.Tools[0].Name != "greet" {
		t.Fatalf("tool discovery: %#v, %v", listed, err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "greet", Arguments: map[string]any{"name": "Agent"}})
	if err != nil || result.IsError {
		t.Fatalf("valid tool call: %#v, %v", result, err)
	}
	if result.StructuredContent.(map[string]any)["greeting"] != "Hello, Agent" {
		t.Fatalf("unexpected result: %#v", result.StructuredContent)
	}
	invalid, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "missing"})
	if err == nil && !invalid.IsError {
		t.Fatalf("unknown tool unexpectedly succeeded: %#v", invalid)
	}
}
