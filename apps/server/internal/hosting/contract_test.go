package hosting

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type publicOperation struct {
	Method           string   `json:"method"`
	Path             string   `json:"path"`
	SuccessStatus    int      `json:"successStatus"`
	ResponseRequired []string `json:"responseRequired"`
	Errors           []string `json:"errors"`
}

func TestPinnedPublicContract(t *testing.T) {
	raw, err := os.ReadFile("../../../../contracts/http-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		ContractVersion string            `json:"contractVersion"`
		Operations      []publicOperation `json:"operations"`
		ErrorEnvelope   struct {
			Required []string `json:"required"`
		} `json:"errorEnvelope"`
	}
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	if contract.ContractVersion != "1.1.0" || len(contract.Operations) != 15 {
		t.Fatalf("unexpected contract version or operation count: %s, %d", contract.ContractVersion, len(contract.Operations))
	}
	serverSource, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	allSource := string(serverSource)
	for _, filename := range []string{"auth.go", "apps.go", "core.go", "data.go"} {
		file, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		allSource += string(file)
	}
	for _, operation := range contract.Operations {
		registration := `mux.HandleFunc("` + operation.Method + " " + operation.Path + `"`
		if !strings.Contains(string(serverSource), registration) {
			t.Errorf("route missing from server: %s %s", operation.Method, operation.Path)
		}
		if operation.SuccessStatus < 200 || operation.SuccessStatus >= 300 {
			t.Errorf("invalid success status: %s %s", operation.Method, operation.Path)
		}
		for _, code := range operation.Errors {
			if !strings.Contains(allSource, `"`+code+`"`) {
				t.Errorf("error code absent from server: %s", code)
			}
		}
	}
	response := httptest.NewRecorder()
	errorResponse(response, fail(401, "authorization_required", "Run login"))
	if response.Code != 401 {
		t.Fatalf("error status = %d", response.Code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	for _, field := range contract.ErrorEnvelope.Required {
		if _, ok := envelope[field]; !ok {
			t.Errorf("error envelope missing %q", field)
		}
	}
	if envelope["error"] != "authorization_required" {
		t.Errorf("error code = %v", envelope["error"])
	}
	responseShapes := map[string]any{
		"GET /limits": Limits{},
		"GET /apps":   []App{{}},
		"POST /apps":  App{},
		"GET /apps/{app}/deployments/{deployment}": Deployment{},
		"GET /apps/{app}/data":                     AppData{},
		"PUT /apps/{app}/data":                     AppData{},
	}
	for _, operation := range contract.Operations {
		fixture, ok := responseShapes[operation.Method+" "+operation.Path]
		if !ok {
			continue
		}
		encoded, err := json.Marshal(fixture)
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal(encoded, &value); err != nil {
			t.Fatal(err)
		}
		if entries, ok := value.([]any); ok {
			value = entries[0]
		}
		object := value.(map[string]any)
		for _, field := range operation.ResponseRequired {
			if _, ok := object[field]; !ok {
				t.Errorf("%s %s missing %s", operation.Method, operation.Path, field)
			}
		}
	}
}
