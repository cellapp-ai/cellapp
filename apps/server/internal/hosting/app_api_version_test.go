package hosting

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestAppAPIVersionAliases(t *testing.T) {
	if err := validateAppAPIContracts(appAPIContracts, appAPILatest, "1.1"); err != nil {
		t.Fatal(err)
	}
	for _, alias := range []string{"1.1", "261022"} {
		r := httptest.NewRequest("GET", "/_cellapp/me?v="+alias, nil)
		got, err := selectAppAPIContract(r)
		if err != nil || got.Version != "1.1" || got.Date != "261022" {
			t.Fatalf("alias %q: %+v, %v", alias, got, err)
		}
	}
	got, err := selectAppAPIContract(httptest.NewRequest("GET", "/_cellapp/me", nil))
	if err != nil || got.Version != appAPILatest {
		t.Fatalf("latest: %+v, %v", got, err)
	}
	for _, query := range []string{"v=", "v=1.1&v=261022", "v=1.2", "v=261023", "v=261332", "v=1.1.261022"} {
		_, err := selectAppAPIContract(httptest.NewRequest("GET", "/_cellapp/me?"+query, nil))
		api, ok := err.(*APIError)
		if !ok || api.Code != "unsupported_api_version" {
			t.Fatalf("query %q: %v", query, err)
		}
	}
	retired := []appAPIContract{{Version: "1.1", Date: "261022", Status: "retired"}, {Version: "1.2", Date: "261023", Status: "supported"}}
	for _, alias := range []string{"1.1", "261022"} {
		_, err := selectAppAPIContractFrom(httptest.NewRequest("GET", "/_cellapp/me?v="+alias, nil), retired, "1.2")
		api, ok := err.(*APIError)
		if !ok || api.Code != "unsupported_api_version" {
			t.Fatalf("retired alias %q: %v", alias, err)
		}
	}
}

func TestAppAPIVersionRegistryValidation(t *testing.T) {
	base := appAPIContract{Version: "1.1", Date: "261022", Status: "supported"}
	for _, contracts := range [][]appAPIContract{
		{base, {Version: "1.2", Date: "261022", Status: "supported"}},
		{base, {Version: "1.2", Date: "261023", Status: "supported"}},
		{base, {Version: "1.1", Date: "261023", Status: "supported"}},
		{{Version: "1.1", Date: "261332", Status: "supported"}},
		{{Version: "1.1", Date: "261022", Status: "retired"}},
	} {
		if err := validateAppAPIContracts(contracts, "1.1", "1.1"); err == nil {
			t.Fatalf("accepted invalid contracts: %+v", contracts)
		}
	}
	if _, err := LoadConfig(func(key string) string {
		if key == "APP_API_STABLE_VERSION" {
			return "2.0"
		}
		return ""
	}); err == nil {
		t.Fatal("accepted unsupported stable version")
	}
}

func TestAppAPIContractAdapters(t *testing.T) {
	contract := appAPIContracts[0]
	payload, err := appAPIIdentityResponse(contract, "app-user")
	if err != nil || payload.(map[string]string)["id"] != "app-user" {
		t.Fatalf("identity adapter: %v, %v", payload, err)
	}
	query, err := appAPIDataQuery(contract, url.Values{"v": {"1.1"}, "select": {"id,body"}, "body": {"eq.hello"}})
	if err != nil {
		t.Fatal(err)
	}
	forwarded, err := url.ParseQuery(query)
	if err != nil || forwarded.Has("v") || forwarded.Get("select") != "id,body" || forwarded.Get("body") != "eq.hello" {
		t.Fatalf("table adapter: %q, %v", query, err)
	}
}
