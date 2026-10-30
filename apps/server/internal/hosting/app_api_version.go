package hosting

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

const appAPIVersionGuide = "https://cellapp.ai/deploy/api-versions.md"

type appAPIContract struct {
	Version string `json:"version"`
	Date    string `json:"date"`
	Status  string `json:"status"`
}

var appAPIContracts = []appAPIContract{{Version: "1.1", Date: "261022", Status: "supported"}, {Version: "1.2", Date: "261027", Status: "supported"}}
var appAPILatest = "1.2"
var appAPIVersionPattern = regexp.MustCompile(`^[1-9][0-9]*\.[0-9]+$`)
var appAPIDatePattern = regexp.MustCompile(`^[0-9]{6}$`)

func validAppAPIDate(value string) bool {
	if !appAPIDatePattern.MatchString(value) {
		return false
	}
	year, _ := strconv.Atoi(value[:2])
	month, _ := strconv.Atoi(value[2:4])
	day, _ := strconv.Atoi(value[4:])
	parsed := time.Date(2000+year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return parsed.Year() == 2000+year && int(parsed.Month()) == month && parsed.Day() == day
}

func validateAppAPIContracts(contracts []appAPIContract, latest, stable string) error {
	versions := map[string]bool{}
	dates := map[string]bool{}
	for _, contract := range contracts {
		if !appAPIVersionPattern.MatchString(contract.Version) || !validAppAPIDate(contract.Date) || !implementedAppAPIContract(contract.Version) ||
			contract.Status != "supported" && contract.Status != "retired" || versions[contract.Version] || dates[contract.Date] {
			return fmt.Errorf("invalid or duplicate application API contract: %s = %s", contract.Version, contract.Date)
		}
		versions[contract.Version] = true
		dates[contract.Date] = true
	}
	if !versions[latest] || !versions[stable] {
		return fmt.Errorf("latest or stable application API contract is unavailable")
	}
	for _, contract := range contracts {
		if (contract.Version == latest || contract.Version == stable) && contract.Status != "supported" {
			return fmt.Errorf("latest or stable application API contract is retired")
		}
	}
	return nil
}

func implementedAppAPIContract(version string) bool {
	switch version {
	case "1.1", "1.2":
		return true
	default:
		return false
	}
}

func appAPIIdentityResponse(contract appAPIContract, appUserID string) (any, error) {
	switch contract.Version {
	case "1.1", "1.2":
		return map[string]string{"id": appUserID, "access": "admitted"}, nil
	default:
		return nil, fmt.Errorf("unimplemented application API identity contract %s", contract.Version)
	}
}

func appAPIDataQuery(contract appAPIContract, values url.Values) (string, error) {
	switch contract.Version {
	case "1.1", "1.2":
		values.Del("v")
		return values.Encode(), nil
	default:
		return "", fmt.Errorf("unimplemented application API table contract %s", contract.Version)
	}
}

func appAPIContractFor(contracts []appAPIContract, alias string) (appAPIContract, bool) {
	for _, contract := range contracts {
		if alias == contract.Version || alias == contract.Date {
			return contract, true
		}
	}
	return appAPIContract{}, false
}

func selectAppAPIContract(r *http.Request) (appAPIContract, error) {
	return selectAppAPIContractFrom(r, appAPIContracts, appAPILatest)
}

func selectAppAPIContractFrom(r *http.Request, contracts []appAPIContract, latest string) (appAPIContract, error) {
	query := r.URL.Query()
	values, present := query["v"]
	if !present {
		contract, _ := appAPIContractFor(contracts, latest)
		return contract, nil
	}
	requested := ""
	if len(values) == 1 {
		requested = values[0]
	}
	contract, found := appAPIContractFor(contracts, requested)
	if len(values) != 1 || !found || contract.Status != "supported" {
		return appAPIContract{}, &APIError{Status: 400, Code: "unsupported_api_version", Message: "Unsupported application API version", Details: map[string]any{"requested": values, "guide": appAPIVersionGuide}}
	}
	return contract, nil
}

func (s *Server) appAPIVersions(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.owner(r); err != nil {
		return err
	}
	stable := s.Config.AppAPIStableVersion
	if stable == "" {
		stable = appAPILatest
	}
	jsonResponse(w, 200, map[string]any{"stable": stable, "latest": appAPILatest, "versions": appAPIContracts})
	return nil
}
