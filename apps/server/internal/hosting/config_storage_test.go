package hosting

import (
	"strings"
	"testing"
)

func TestS3CredentialGroups(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		vars                    map[string]string
		access, secret, missing string
	}{
		{"generic", map[string]string{"S3_ACCESS_KEY_ID": "new-ak", "S3_SECRET_ACCESS_KEY": "new-sk"}, "new-ak", "new-sk", ""},
		{"access only", map[string]string{"S3_ACCESS_KEY_ID": "new-ak"}, "", "", "S3_SECRET_ACCESS_KEY"},
		{"secret only", map[string]string{"S3_SECRET_ACCESS_KEY": "new-sk"}, "", "", "S3_ACCESS_KEY_ID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := LoadConfig(func(k string) string { return tc.vars[k] })
			if tc.missing != "" {
				if err == nil || err.Error() != "missing configuration: "+tc.missing {
					t.Fatalf("expected missing variable %s, got %v", tc.missing, err)
				}
				return
			}
			if err != nil || c.S3AccessKeyID != tc.access || c.S3SecretAccessKey != tc.secret {
				t.Fatal("incorrect credential selection", err)
			}
		})
	}
}

func TestLegacyCredentialsIgnored(t *testing.T) {
	legacy := map[string]string{"QINIU_ACCESS_KEY": "old-ak", "QINIU_SECRET_KEY": "old-sk"}
	c, err := LoadConfig(func(k string) string { return legacy[k] })
	if err != nil || c.S3AccessKeyID != "cellapp-local" || c.S3SecretAccessKey != "cellapp-local-development-only" {
		t.Fatal("legacy credentials changed development defaults", err)
	}
}

func TestProductionS3RequiresExplicitConfiguration(t *testing.T) {
	vars := map[string]string{
		"NODE_ENV": "production", "CONTROL_ORIGIN": "https://control.example.com", "APPS_DOMAIN": "apps.example.net",
		"SECRET": strings.Repeat("s", 32), "DATABASE_URL": "postgres://test", "GITHUB_CLIENT_ID": "test", "GITHUB_CLIENT_SECRET": "test",
		"S3_ENDPOINT": "https://storage.example.com", "S3_REGION": "us-east-1", "S3_BUCKET": "private-test",
		"S3_ACCESS_KEY_ID": "test-ak", "S3_SECRET_ACCESS_KEY": "test-sk",
		"MAX_APPS": "10", "MAX_FILES": "1000", "MAX_FILE_BYTES": "100", "MAX_DEPLOYMENT_BYTES": "1000", "MAX_STORAGE_BYTES": "10000",
		"MAX_MONTHLY_TRAFFIC_BYTES": "10000", "MAX_REQUESTS_PER_MINUTE": "120", "UPLOAD_TTL_SECONDS": "3600", "RETENTION_SECONDS": "86400",
	}
	for _, missing := range []string{"S3_ENDPOINT", "S3_REGION", "S3_BUCKET", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY"} {
		t.Run(missing, func(t *testing.T) {
			_, err := LoadConfig(func(k string) string {
				if k == missing {
					return ""
				}
				return vars[k]
			})
			if err == nil || err.Error() != "missing configuration: "+missing {
				t.Fatalf("expected explicit %s, got %v", missing, err)
			}
		})
	}
	delete(vars, "S3_ACCESS_KEY_ID")
	delete(vars, "S3_SECRET_ACCESS_KEY")
	if _, err := LoadConfig(func(k string) string { return vars[k] }); err == nil {
		t.Fatal("production accepted default credentials")
	}
	vars["QINIU_ACCESS_KEY"], vars["QINIU_SECRET_KEY"] = "legacy-ak", "legacy-sk"
	if _, err := LoadConfig(func(k string) string { return vars[k] }); err == nil {
		t.Fatal("production accepted legacy credentials")
	}
}
