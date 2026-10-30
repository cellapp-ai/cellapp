package hosting

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

type modelResolverStub struct {
	ips []netip.Addr
}

func (r modelResolverStub) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return r.ips, nil
}

func TestModelPublicTargetsAndFixedPath(t *testing.T) {
	for _, base := range []string{
		"http://models.example.com/v1", "https://localhost/v1", "https://127.0.0.1/v1",
		"https://10.1.2.3/v1", "https://169.254.169.254/v1", "https://[::1]/v1",
		"https://203.0.113.1/v1", "https://models.example.com/v1?token=secret",
		"https://user:pass@models.example.com/v1", "https://models.example.com/v1/../admin",
	} {
		if _, err := modelChatURL(base); err == nil {
			t.Errorf("unsafe target accepted: %s", base)
		}
	}
	for _, base := range []string{"https://models.example.com/v1", "https://8.8.8.8:443/v1"} {
		u, err := modelChatURL(base)
		if err != nil || u.Path != "/v1/chat/completions" {
			t.Errorf("public candidate rejected: %s %v", base, err)
		}
	}
	request, err := newModelChatRequest(context.Background(), "https://models.example.com/v1", "provider-key", []byte(`{"model":"model-one","messages":[]}`))
	if err != nil || request.Method != http.MethodPost || request.URL.Path != "/v1/chat/completions" || request.Header.Get("Authorization") != "Bearer provider-key" {
		t.Fatalf("fixed model request invalid: %v", err)
	}
	if request.Header.Get("X-Extra") != "" {
		t.Fatal("custom header forwarded")
	}
}

func TestModelDNSRebindingAndRedirect(t *testing.T) {
	ctx := context.Background()
	if _, err := resolveModelHost(ctx, modelResolverStub{[]netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")}}, "models.example.com"); err == nil {
		t.Fatal("mixed public/private DNS answer accepted")
	}
	if _, err := resolveModelHost(ctx, modelResolverStub{[]netip.Addr{netip.MustParseAddr("8.8.8.8")}}, "models.example.com"); err != nil {
		t.Fatal("public DNS candidate rejected", err)
	}
	client := modelHTTPClient(modelResolverStub{[]netip.Addr{netip.MustParseAddr("127.0.0.1")}})
	if _, err := client.Get("https://models.example.com/v1/chat/completions"); err == nil {
		t.Fatal("private DNS target connected")
	}
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://other.example.com", http.StatusFound)
	}))
	defer redirect.Close()
	redirectClient := redirect.Client()
	redirectClient.CheckRedirect = client.CheckRedirect
	if _, err := redirectClient.Get(redirect.URL); err == nil {
		t.Fatal("model redirect followed")
	}
}

func TestModelResponseLimit(t *testing.T) {
	response := &http.Response{Body: io.NopCloser(strings.NewReader("12345"))}
	if _, err := modelResponseBody(response, 4); err == nil {
		t.Fatal("oversized model response accepted")
	}
}
