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
	if c.S3Endpoint != "http://127.0.0.1:9000" || c.S3Region != "us-east-1" || c.S3Bucket != "cellapp" || c.S3AccessKeyID != "cellapp-local" || c.S3SecretAccessKey != "cellapp-local-development-only" {
		t.Fatal("unexpected local S3 defaults")
	}
	c, e = load(map[string]string{"S3_ENDPOINT": "https://storage.example.com", "S3_REGION": "custom-region", "S3_BUCKET": "existing-bucket"})
	if e != nil || c.S3Endpoint != "https://storage.example.com" || c.S3Region != "custom-region" || c.S3Bucket != "existing-bucket" {
		t.Fatal("explicit S3 configuration was not preserved")
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
