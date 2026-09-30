package hosting

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var domainPattern = regexp.MustCompile(`^[a-z0-9]+(?:[.-][a-z0-9]+)*$`)
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var idPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type APIError struct {
	Status        int
	Code, Message string
	Details       any
}

func (e *APIError) Error() string                 { return e.Code }
func fail(status int, code, message string) error { return &APIError{status, code, message, nil} }
func token() string {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func id() string             { return token()[:32] }
func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func keyed(secret, value string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}
func equal(a, b string) bool { return hmac.Equal([]byte(a), []byte(b)) }
func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(dst); e != nil {
		return fail(400, "invalid_json", "Invalid request body")
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return fail(400, "invalid_json", "Expected one JSON value")
	}
	return nil
}
func errorResponse(w http.ResponseWriter, e error) {
	var api *APIError
	if !errors.As(e, &api) {
		api = &APIError{500, "internal_error", "Request could not be completed", nil}
	}
	jsonResponse(w, api.Status, map[string]any{"error": api.Code, "message": api.Message, "details": api.Details})
}

type Entry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func validatePath(p string) error {
	if p == "" || len(p) > 512 || strings.ContainsAny(p, "\\%?#") || strings.HasPrefix(p, "/") {
		return fail(400, "invalid_path", "Unsafe artifact path")
	}
	for _, c := range p {
		if c < 32 || c == 127 {
			return fail(400, "invalid_path", "Unsafe artifact path")
		}
	}
	for _, v := range strings.Split(p, "/") {
		l := strings.ToLower(v)
		if v == "" || strings.HasPrefix(v, ".") || l == "_hosting" || l == "node_modules" || l == "credentials" || strings.HasPrefix(l, "credentials.") || l == "id_rsa" || l == "id_ed25519" {
			return fail(400, "sensitive_path", "Reserved or sensitive path")
		}
	}
	for _, suffix := range []string{".pem", ".key", ".p12", ".pfx"} {
		if strings.HasSuffix(strings.ToLower(p), suffix) {
			return fail(400, "sensitive_path", "Sensitive file")
		}
	}
	return nil
}
func validateManifest(files []Entry, l Limits) (int64, error) {
	if len(files) == 0 || int64(len(files)) > l.Files {
		return 0, fail(413, "file_limit", "Invalid file count")
	}
	seen := map[string]bool{}
	var total int64
	for _, f := range files {
		if e := validatePath(f.Path); e != nil {
			return 0, e
		}
		if seen[f.Path] || !digestPattern.MatchString(f.SHA256) {
			return 0, fail(400, "invalid_manifest", "Duplicate path or invalid digest")
		}
		if f.Size < 0 || f.Size > l.FileBytes || f.Size > l.DeploymentBytes-total {
			return 0, fail(413, "deployment_limit", "Artifact exceeds configured limits")
		}
		seen[f.Path] = true
		total += f.Size
	}
	if !seen["index.html"] {
		return 0, fail(400, "static_export_required", "A static index.html is required")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return total, nil
}
func safeReturn(path string) string {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "\\\r\n") || strings.HasPrefix(path, "/_hosting/") {
		return "/"
	}
	return path
}
func safeControlReturn(value string) string {
	value = safeReturn(value)
	u, e := url.ParseRequestURI(value)
	if e != nil || u.IsAbs() || u.Fragment != "" || !webUIPath(u.Path) {
		return "/"
	}
	return value
}
func objectKey(deployment, path string) string {
	return fmt.Sprintf("releases/%s/%s", deployment, digest([]byte(path)))
}
