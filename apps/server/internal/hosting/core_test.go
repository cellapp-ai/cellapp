package hosting

import (
	"strings"
	"testing"
)

func TestPathsAndManifest(t *testing.T) {
	for _, p := range []string{"../x", "/x", "x/../y", ".env", "x/.git/config", "_hosting/x", "credentials.json", "private.pem", "a\\b", "a%2fb", "a\x00b"} {
		if validatePath(p) == nil {
			t.Errorf("accepted unsafe path %q", p)
		}
	}
	l := Limits{Files: 2, FileBytes: 10, DeploymentBytes: 10}
	f := Entry{"index.html", 1, digest([]byte("x"))}
	if n, e := validateManifest([]Entry{f}, l); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if _, e := validateManifest([]Entry{f, f}, l); e == nil {
		t.Fatal("duplicate paths accepted")
	}
	f.Size = 11
	if _, e := validateManifest([]Entry{f}, l); e == nil {
		t.Fatal("oversized file accepted")
	}
	f.Size = 1
	f.Path = "server.js"
	if _, e := validateManifest([]Entry{f}, l); e == nil {
		t.Fatal("missing index accepted")
	}
}
func TestConfig(t *testing.T) {
	load := func(m map[string]string) (Config, error) { return LoadConfig(func(k string) string { return m[k] }) }
	if _, e := load(map[string]string{"NODE_ENV": "production"}); e == nil {
		t.Fatal("production started without config")
	}
	for _, m := range []map[string]string{{"MAX_APPS": "0"}, {"SECRET": "short"}, {"CONTROL_ORIGIN": "http://example.com"}, {"APPS_DOMAIN": "control.localhost"}, {"APP_PORT": "-1"}, {"AUTH_MODE": "unknown"}, {"AUTH_MODE": "dev", "CONTROL_ORIGIN": "https://example.com"}} {
		if _, e := load(m); e == nil {
			t.Fatalf("accepted invalid config: %v", m)
		}
	}
	c, e := load(nil)
	if e != nil {
		t.Fatal(e)
	}
	if c.Limits.Apps <= 0 || c.Limits.StorageBytes <= 0 {
		t.Fatal("unbounded default")
	}
	if c.AuthMode != "dev" {
		t.Fatal("development should default to local authentication")
	}
	c, e = load(map[string]string{"QINIU_ACCESS_KEY": "access-from-environment", "QINIU_SECRET_KEY": "secret-from-environment"})
	if e != nil || c.QiniuAccessKey != "access-from-environment" || c.QiniuSecretKey != "secret-from-environment" {
		t.Fatal("Qiniu credentials were not loaded from the process environment")
	}
}
func TestSecretsAndReturnPath(t *testing.T) {
	a, b := token(), token()
	if !keyValid(a) || a == b {
		t.Fatal("bad secret generation")
	}
	if strings.Contains(keyed("secret", a), a) {
		t.Fatal("raw token persisted")
	}
	for _, p := range []string{"//evil.com", "https://evil.com", "/\\evil.com", "/_hosting/unlock"} {
		if safeReturn(p) != "/" {
			t.Fatalf("unsafe redirect %s", p)
		}
	}
	if safeReturn("/notes?tab=one") != "/notes?tab=one" {
		t.Fatal("lost deep link")
	}
}
