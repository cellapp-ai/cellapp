package hosting

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestEdgeGrantBindingsExpiryAndRotation(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	oldPublic, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	claims := edgeGrant{Host: "notes.ce11.app", App: strings.Repeat("a", 32), Deployment: strings.Repeat("b", 32), SPA: true,
		Session: appCookieDigest("opaque-session"), IssuedAt: now.Unix(), ExpiresAt: now.Add(edgeGrantLifetime).Unix()}
	value, err := signEdgeGrant("new", private, claims)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]ed25519.PublicKey{"old": oldPublic, "new": public}
	parts := strings.Split(value, ".")
	first := "A"
	if strings.HasPrefix(parts[3], "A") {
		first = "B"
	}
	tampered := strings.Join(parts[:3], ".") + "." + first + parts[3][1:]
	if got, err := verifyEdgeGrant(value, claims.Host, "opaque-session", keys, now); err != nil || got.Deployment != claims.Deployment {
		t.Fatalf("valid grant rejected: %v", err)
	}
	if _, err := verifyEdgeGrant(value, claims.Host, "opaque-session", keys, now.Add(59*time.Second)); err != nil {
		t.Fatalf("grant expired too early: %v", err)
	}
	for name, candidate := range map[string]struct {
		value, host, session string
		keys                 map[string]ed25519.PublicKey
		at                   time.Time
	}{
		"tamper":        {tampered, claims.Host, "opaque-session", keys, now},
		"wrong host":    {value, "other.ce11.app", "opaque-session", keys, now},
		"wrong session": {value, claims.Host, "another-session", keys, now},
		"expired":       {value, claims.Host, "opaque-session", keys, now.Add(60 * time.Second)},
		"future":        {value, claims.Host, "opaque-session", keys, now.Add(-time.Second)},
		"retired key":   {value, claims.Host, "opaque-session", map[string]ed25519.PublicKey{"old": oldPublic}, now},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifyEdgeGrant(candidate.value, candidate.host, candidate.session, candidate.keys, candidate.at); err == nil {
				t.Fatal("invalid grant accepted")
			}
		})
	}
	claims.ExpiresAt++
	if _, err := signEdgeGrant("new", private, claims); err == nil {
		t.Fatal("grant longer than one minute accepted")
	}
}

func TestEdgeSigningKeyConfiguration(t *testing.T) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key := base64.RawURLEncoding.EncodeToString(private)
	values := map[string]string{"EDGE_GRANT_KEY_ID": "key-1", "EDGE_GRANT_SIGNING_KEY": key}
	if c, err := LoadConfig(func(name string) string { return values[name] }); err != nil || c.EdgeGrantKeyID != "key-1" || len(c.EdgeGrantSigningKey) != ed25519.PrivateKeySize {
		t.Fatalf("valid signing key rejected: %v", err)
	}
	for name, bad := range map[string]string{"missing key": "", "short key": "abc"} {
		t.Run(name, func(t *testing.T) {
			values["EDGE_GRANT_SIGNING_KEY"] = bad
			if _, err := LoadConfig(func(key string) string { return values[key] }); err == nil {
				t.Fatal("invalid signing key accepted")
			}
		})
	}
	values["EDGE_GRANT_SIGNING_KEY"] = key
	values["RETENTION_SECONDS"] = "60"
	if _, err := LoadConfig(func(name string) string { return values[name] }); err == nil {
		t.Fatal("retention shorter than grant lifetime accepted")
	}
}
