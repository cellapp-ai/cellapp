package hosting

import (
	"cellapp/apps/server/appdata"
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpIdentityKey struct{}

func mcpControlPath(path string) bool {
	switch path {
	case "/mcp", "/oauth/authorize", "/oauth/token", "/oauth/register", "/oauth/revoke", "/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp", "/.well-known/oauth-authorization-server":
		return true
	}
	return false
}

type mcpIdentity struct {
	owner  string
	scopes []string
}

func mcpBool(value bool) *bool { return &value }

var mcpReadOnly = &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: mcpBool(false)}
var mcpAdditive = &mcp.ToolAnnotations{DestructiveHint: mcpBool(false), IdempotentHint: true, OpenWorldHint: mcpBool(false)}
var mcpChange = &mcp.ToolAnnotations{IdempotentHint: true, OpenWorldHint: mcpBool(false)}

func mcpIdentityFor(ctx context.Context, scope string) (string, error) {
	identity, ok := ctx.Value(mcpIdentityKey{}).(mcpIdentity)
	if !ok || !mcpHasScope(identity.scopes, scope) {
		return "", fail(http.StatusForbidden, "insufficient_scope", "MCP scope is required")
	}
	return identity.owner, nil
}

func mcpToolFailure(err error) (*mcp.CallToolResult, any, error) {
	var api *APIError
	if !errors.As(err, &api) {
		api = &APIError{Status: 500, Code: "internal_error", Message: "Request could not be completed"}
	}
	return &mcp.CallToolResult{IsError: true, StructuredContent: map[string]any{"error": api.Code, "message": api.Message, "details": api.Details}}, nil, nil
}

func (s *Server) mcpHTTPHandler() http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "cellapp", Version: "1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "list_apps", Description: "List applications owned by the authorized Cellapp user. Requires mcp:read.", Annotations: mcpReadOnly},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:read")
			if err != nil {
				return mcpToolFailure(err)
			}
			apps, err := s.ownerApps(ctx, owner)
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, map[string]any{"apps": apps}, nil
		})
	type createInput struct {
		Name      string `json:"name" jsonschema:"Application display name"`
		RequestID string `json:"requestId" jsonschema:"Stable 32-character lowercase hexadecimal request identifier"`
		Namespace string `json:"namespace,omitempty"`
		Slug      string `json:"slug,omitempty"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "create_app", Description: "Create a Cellapp application with a stable requestId. Requires mcp:deploy. The initial access key is kept server-side.", Annotations: mcpAdditive},
		func(ctx context.Context, _ *mcp.CallToolRequest, input createInput) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:deploy")
			if err != nil {
				return mcpToolFailure(err)
			}
			app, err := s.createOwnerApp(ctx, owner, createAppInput{Name: input.Name, Key: keyed(s.Config.Secret, "mcp-app-key:"+owner+":"+input.RequestID), RequestID: input.RequestID, Namespace: input.Namespace, Slug: input.Slug})
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, map[string]any{"app": app}, nil
		})
	type prepareInput struct {
		AppID       string  `json:"appId"`
		RequestID   string  `json:"requestId"`
		Manifest    []Entry `json:"manifest"`
		BaseVersion *string `json:"baseVersion,omitempty"`
		SPA         bool    `json:"spa,omitempty"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "prepare_deployment", Description: "Validate a static artifact manifest and prepare an upload. Requires mcp:deploy.", Annotations: mcpAdditive},
		func(ctx context.Context, _ *mcp.CallToolRequest, input prepareInput) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:deploy")
			if err != nil {
				return mcpToolFailure(err)
			}
			deploymentID, err := s.prepareOwnerDeployment(ctx, owner, input.AppID, createDeploymentInput{Manifest: input.Manifest, RequestID: input.RequestID, SPA: input.SPA, Base: input.BaseVersion})
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, map[string]any{"deploymentId": deploymentID, "chunkBytes": 256 << 10}, nil
		})
	type deploymentInput struct {
		AppID        string `json:"appId"`
		DeploymentID string `json:"deploymentId"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "get_deployment", Description: "Get an owned deployment and upload progress. Requires mcp:read.", Annotations: mcpReadOnly},
		func(ctx context.Context, _ *mcp.CallToolRequest, input deploymentInput) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:read")
			if err != nil {
				return mcpToolFailure(err)
			}
			deployment, err := s.ownerDeploymentStatus(ctx, owner, input.AppID, input.DeploymentID)
			if err != nil {
				return mcpToolFailure(err)
			}
			progress, err := s.mcpChunkProgress(ctx, input.DeploymentID)
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, map[string]any{"deployment": deployment, "nextOffsetBytes": progress}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "upload_file_chunk", Description: "Upload one base64 encoded chunk of at most 256 KiB. Requires mcp:deploy.", Annotations: mcpAdditive},
		func(ctx context.Context, _ *mcp.CallToolRequest, input mcpChunkInput) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:deploy")
			if err != nil {
				return mcpToolFailure(err)
			}
			next, err := s.uploadMCPChunk(ctx, owner, input)
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, map[string]any{"nextOffsetBytes": next}, nil
		})
	type completeInput struct {
		AppID        string `json:"appId"`
		DeploymentID string `json:"deploymentId"`
		FileIndex    int    `json:"fileIndex"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "complete_file", Description: "Assemble and verify all chunks for one manifest file. Requires mcp:deploy.", Annotations: mcpAdditive},
		func(ctx context.Context, _ *mcp.CallToolRequest, input completeInput) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:deploy")
			if err != nil {
				return mcpToolFailure(err)
			}
			if err := s.completeMCPFile(ctx, owner, input.AppID, input.DeploymentID, input.FileIndex); err != nil {
				return mcpToolFailure(err)
			}
			return nil, map[string]any{"completed": true}, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "publish_deployment", Description: "Atomically publish a complete deployment. Requires mcp:deploy.", Annotations: mcpChange},
		func(ctx context.Context, _ *mcp.CallToolRequest, input deploymentInput) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:deploy")
			if err != nil {
				return mcpToolFailure(err)
			}
			result, err := s.publishOwnerDeployment(ctx, owner, input.AppID, input.DeploymentID)
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, result, nil
		})
	type dataTarget struct {
		AppID string `json:"appId"`
	}
	type dataManifestInput struct {
		AppID    string           `json:"appId"`
		Manifest appdata.Manifest `json:"manifest"`
	}
	type dataApplyInput struct {
		AppID            string           `json:"appId"`
		Manifest         appdata.Manifest `json:"manifest"`
		ReviewedDigest   string           `json:"reviewedDigest"`
		AllowDestructive bool             `json:"allowDestructive,omitempty"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "data_status", Description: "Read the owned application's data readiness and schema version. Requires mcp:read.", Annotations: mcpReadOnly},
		func(ctx context.Context, _ *mcp.CallToolRequest, input dataTarget) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:read")
			if err != nil {
				return mcpToolFailure(err)
			}
			result, err := s.mcpDataStatus(ctx, owner, input.AppID)
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, result, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "data_reconcile", Description: "Retry provisioning the owned application's data schema. Requires mcp:data.", Annotations: mcpAdditive},
		func(ctx context.Context, _ *mcp.CallToolRequest, input dataTarget) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:data")
			if err != nil {
				return mcpToolFailure(err)
			}
			result, err := s.mcpDataReconcile(ctx, owner, input.AppID)
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, result, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "data_plan", Description: "Plan a versioned table manifest for the owned application. Requires mcp:data.", Annotations: mcpReadOnly},
		func(ctx context.Context, _ *mcp.CallToolRequest, input dataManifestInput) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:data")
			if err != nil {
				return mcpToolFailure(err)
			}
			plan, err := s.mcpDataPlan(ctx, owner, input.AppID, input.Manifest)
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, plan, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "data_apply", Description: "Apply a reviewed table plan with matching digest. Destructive changes need explicit permission. Requires mcp:data.", Annotations: mcpChange},
		func(ctx context.Context, _ *mcp.CallToolRequest, input dataApplyInput) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:data")
			if err != nil {
				return mcpToolFailure(err)
			}
			plan, err := s.mcpDataApply(ctx, owner, input.AppID, input.Manifest, input.ReviewedDigest, input.AllowDestructive)
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, plan, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "data_verify", Description: "Probe the owned application's data API and report the applied version. Requires mcp:data.", Annotations: mcpReadOnly},
		func(ctx context.Context, _ *mcp.CallToolRequest, input dataTarget) (*mcp.CallToolResult, any, error) {
			owner, err := mcpIdentityFor(ctx, "mcp:data")
			if err != nil {
				return mcpToolFailure(err)
			}
			result, err := s.mcpDataVerify(ctx, owner, input.AppID)
			if err != nil {
				return mcpToolFailure(err)
			}
			return nil, result, nil
		})
	protocol := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+s.Config.ControlOrigin+`/.well-known/oauth-protected-resource/mcp"`)
			mcpOAuthError(w, http.StatusUnauthorized, "invalid_token")
			return
		}
		owner, scopes, err := s.mcpPrincipal(r.Context(), strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+s.Config.ControlOrigin+`/.well-known/oauth-protected-resource/mcp", error="invalid_token"`)
			mcpOAuthError(w, http.StatusUnauthorized, "invalid_token")
			return
		}
		protocol.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), mcpIdentityKey{}, mcpIdentity{owner, scopes})))
	})
}
