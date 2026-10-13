package hosting

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEdgeOriginConfiguration(t *testing.T) {
	vars := map[string]string{
		"EDGE_ORIGIN_HOST":   "edge-probe---cellapp-example.us-west1.run.app",
		"EDGE_ORIGIN_SECRET": strings.Repeat("s", 32),
	}
	load := func() error {
		_, err := LoadConfig(func(key string) string { return vars[key] })
		return err
	}
	if err := load(); err != nil {
		t.Fatal(err)
	}
	delete(vars, "EDGE_ORIGIN_SECRET")
	if err := load(); err == nil {
		t.Fatal("origin host without a secret was accepted")
	}
	vars["EDGE_ORIGIN_SECRET"] = "short"
	if err := load(); err == nil {
		t.Fatal("short origin secret was accepted")
	}
}

func TestAuthenticatedOriginalHost(t *testing.T) {
	const origin = "cellapp-example.us-west1.run.app"
	const secret = "a-dedicated-edge-secret-with-32-bytes"
	s := &Server{Config: Config{EdgeOriginHost: origin, EdgeOriginSecret: secret}}
	request := func(host, assertedHost, proof string) string {
		r := httptest.NewRequest("GET", "https://"+host+"/", nil)
		r.Host = host
		r.Header.Set("X-Cellapp-Original-Host", assertedHost)
		r.Header.Set("X-Cellapp-Edge-Secret", proof)
		return s.requestHost(r)
	}
	if got := request(origin, "console.cellapp.ai", secret); got != "console.cellapp.ai" {
		t.Fatalf("authenticated control host = %q", got)
	}
	if got := request(origin, strings.Repeat("a", 32)+".ce11.app", secret); got != strings.Repeat("a", 32)+".ce11.app" {
		t.Fatalf("authenticated visitor host = %q", got)
	}
	for _, proof := range []string{"", "invalid"} {
		if got := request(origin, "console.cellapp.ai", proof); got != origin {
			t.Fatalf("forged host accepted with proof %q", proof)
		}
	}
	if got := request("unknown.example", "console.cellapp.ai", secret); got != "unknown.example" {
		t.Fatalf("assertion from wrong origin accepted: %q", got)
	}
	if got := request("console.cellapp.ai", "unknown.example", ""); got != "console.cellapp.ai" {
		t.Fatalf("direct host changed: %q", got)
	}
}
