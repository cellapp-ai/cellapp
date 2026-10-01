package hosting

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

type Limits struct {
	Apps            int64 `json:"apps"`
	Files           int64 `json:"files"`
	FileBytes       int64 `json:"fileBytes"`
	DeploymentBytes int64 `json:"deploymentBytes"`
	StorageBytes    int64 `json:"storageBytes"`
	TrafficBytes    int64 `json:"trafficBytes"`
	Requests        int64 `json:"requests"`
}
type Config struct {
	WebRoot                                                          string
	Production                                                       bool
	Address, ControlOrigin, AppsDomain, AppPort, Secret, DatabaseURL string
	AuthMode                                                         string
	GitHubClientID, GitHubClientSecret                               string
	S3Endpoint, S3Region, S3Bucket, S3AccessKeyID, S3SecretAccessKey string
	Limits                                                           Limits
	UploadTTL, Retention                                             time.Duration
}

func LoadConfig(get func(string) string) (c Config, err error) {
	c.Production = get("NODE_ENV") == "production"
	value := func(key, fallback string) string {
		v := get(key)
		if v == "" && !c.Production {
			v = fallback
		}
		if v == "" && err == nil {
			err = fmt.Errorf("missing configuration: %s", key)
		}
		return v
	}
	number := func(key string, fallback int64) int64 {
		n, e := strconv.ParseInt(value(key, strconv.FormatInt(fallback, 10)), 10, 64)
		if (e != nil || n <= 0 || n > 1<<50) && err == nil {
			err = fmt.Errorf("invalid positive integer: %s", key)
		}
		return n
	}
	c.Address = get("LISTEN_ADDRESS")
	c.WebRoot = get("WEB_ROOT")
	if c.WebRoot == "" {
		c.WebRoot = "apps/web/dist"
	}
	if c.Address == "" {
		c.Address = "127.0.0.1:3000"
	}
	c.ControlOrigin = value("CONTROL_ORIGIN", "https://control.localhost:8443")
	c.AppsDomain = value("APPS_DOMAIN", "apps.localhost")
	c.AppPort = get("APP_PORT")
	if !c.Production && c.AppPort == "" {
		c.AppPort = "8443"
	}
	c.Secret = value("SECRET", "local-development-secret-not-for-production")
	c.DatabaseURL = value("DATABASE_URL", "postgres://cellapp:development@localhost:5432/cellapp")
	c.AuthMode = get("AUTH_MODE")
	if c.AuthMode == "" {
		if c.Production {
			c.AuthMode = "github"
		} else {
			c.AuthMode = "dev"
		}
	}
	if c.AuthMode == "github" {
		c.GitHubClientID = value("GITHUB_CLIENT_ID", "development-only")
		c.GitHubClientSecret = value("GITHUB_CLIENT_SECRET", "development-only")
	} else if c.AuthMode != "dev" && err == nil {
		err = fmt.Errorf("AUTH_MODE must be github or dev")
	}
	c.S3Endpoint = value("S3_ENDPOINT", "http://127.0.0.1:9000")
	c.S3Region = value("S3_REGION", "us-east-1")
	c.S3Bucket = value("S3_BUCKET", "cellapp")
	access, secret := get("S3_ACCESS_KEY_ID"), get("S3_SECRET_ACCESS_KEY")
	if access == "" && secret == "" && !c.Production {
		access, secret = "cellapp-local", "cellapp-local-development-only"
	}
	if err == nil {
		if access == "" {
			err = fmt.Errorf("missing configuration: S3_ACCESS_KEY_ID")
		} else if secret == "" {
			err = fmt.Errorf("missing configuration: S3_SECRET_ACCESS_KEY")
		}
	}
	c.S3AccessKeyID, c.S3SecretAccessKey = access, secret
	c.Limits = Limits{number("MAX_APPS", 10), number("MAX_FILES", 1000), number("MAX_FILE_BYTES", 10_000_000), number("MAX_DEPLOYMENT_BYTES", 50_000_000), number("MAX_STORAGE_BYTES", 500_000_000), number("MAX_MONTHLY_TRAFFIC_BYTES", 1_000_000_000), number("MAX_REQUESTS_PER_MINUTE", 120)}
	c.UploadTTL = time.Duration(number("UPLOAD_TTL_SECONDS", 3600)) * time.Second
	c.Retention = time.Duration(number("RETENTION_SECONDS", 86400)) * time.Second
	if err != nil {
		return c, err
	}
	origin, e := url.Parse(c.ControlOrigin)
	if e != nil || origin.Scheme != "https" || origin.Host == "" || origin.Path != "" || origin.RawQuery != "" || origin.User != nil || origin.Fragment != "" {
		return c, fmt.Errorf("CONTROL_ORIGIN must be an HTTPS origin")
	}
	if c.AuthMode == "dev" {
		host := origin.Hostname()
		ip := net.ParseIP(host)
		local := host == "localhost" || strings.HasSuffix(host, ".localhost") || ip != nil && ip.IsLoopback()
		if c.Production || !local {
			return c, fmt.Errorf("AUTH_MODE=dev requires a non-production localhost control origin")
		}
	}
	if !domainPattern.MatchString(c.AppsDomain) || origin.Hostname() == c.AppsDomain || strings.HasSuffix(origin.Hostname(), "."+c.AppsDomain) {
		return c, fmt.Errorf("invalid or overlapping application domains")
	}
	if c.Production {
		a, ae := publicsuffix.EffectiveTLDPlusOne(origin.Hostname())
		b, be := publicsuffix.EffectiveTLDPlusOne(c.AppsDomain)
		if ae != nil || be != nil || a == b {
			return c, fmt.Errorf("control and app domains must have different registrable domains")
		}
	}
	if c.AppPort != "" {
		p, e := strconv.Atoi(c.AppPort)
		if e != nil || p < 1 || p > 65535 {
			return c, fmt.Errorf("invalid APP_PORT")
		}
	}
	if len(c.Secret) < 32 {
		return c, fmt.Errorf("SECRET must contain at least 32 bytes")
	}
	if c.Limits.FileBytes > c.Limits.DeploymentBytes || c.Limits.DeploymentBytes > c.Limits.StorageBytes {
		return c, fmt.Errorf("file, deployment and storage limits must be ascending")
	}
	return c, nil
}
func (c Config) AppURL(id string) string {
	host := id + "." + c.AppsDomain
	if c.AppPort != "" {
		host += ":" + c.AppPort
	}
	return "https://" + host
}
