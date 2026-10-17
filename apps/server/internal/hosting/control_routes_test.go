package hosting

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Keep every published CLI operation reachable at both its legacy and
// canonical path while older clients remain supported.
func TestCanonicalControlRoutes(t *testing.T) {
	raw, err := os.ReadFile("../../../../skills/contracts/http-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Operations []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	if len(contract.Operations) != 17 {
		t.Fatalf("expected 17 legacy operations, got %d", len(contract.Operations))
	}
	source, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range contract.Operations {
		canonical := operation.Path
		switch operation.Path {
		case "/device-authorizations":
			canonical = "/auth/device"
		case "/device-authorizations/poll":
			canonical = "/auth/device/poll"
		case "/credentials/current":
			canonical = "/auth/credentials/current"
		default:
			canonical = "/api/control/v1" + operation.Path
		}
		for _, path := range []string{operation.Path, canonical} {
			registration := `mux.HandleFunc("` + operation.Method + " " + path + `"`
			if !strings.Contains(string(source), registration) {
				t.Errorf("missing route %s %s", operation.Method, path)
			}
		}
	}
}
