package hosting

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const edgeGrantCookie = "__Host-cellapp-grant"
const edgeGrantLifetime = 60 * time.Second

// Short field names keep the signed value within cookie size limits.
type edgeGrant struct {
	Host       string `json:"h"`
	App        string `json:"a"`
	Deployment string `json:"d"`
	SPA        bool   `json:"s"`
	Session    string `json:"c"`
	IssuedAt   int64  `json:"i"`
	ExpiresAt  int64  `json:"e"`
}

func appCookieDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func signEdgeGrant(keyID string, private ed25519.PrivateKey, claims edgeGrant) (string, error) {
	if len(private) != ed25519.PrivateKeySize || keyID == "" || claims.Host == "" || claims.App == "" || claims.Deployment == "" || claims.Session == "" ||
		claims.ExpiresAt <= claims.IssuedAt || claims.ExpiresAt-claims.IssuedAt > int64(edgeGrantLifetime/time.Second) {
		return "", errors.New("invalid edge grant")
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	message := "v1." + keyID + "." + base64.RawURLEncoding.EncodeToString(payload)
	signature := ed25519.Sign(private, []byte(message))
	return message + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func verifyEdgeGrant(value, host, session string, keys map[string]ed25519.PublicKey, now time.Time) (edgeGrant, error) {
	var claims edgeGrant
	parts := strings.Split(value, ".")
	if len(parts) != 4 || parts[0] != "v1" || len(value) > 4096 {
		return claims, errors.New("invalid edge grant format")
	}
	key := keys[parts[1]]
	payload, payloadError := base64.RawURLEncoding.DecodeString(parts[2])
	signature, signatureError := base64.RawURLEncoding.DecodeString(parts[3])
	if payloadError != nil || signatureError != nil || len(key) != ed25519.PublicKeySize || !ed25519.Verify(key, []byte(strings.Join(parts[:3], ".")), signature) {
		return claims, errors.New("invalid edge grant signature")
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, errors.New("invalid edge grant payload")
	}
	current := now.UTC().Unix()
	if claims.Host != host || claims.Session != appCookieDigest(session) || session == "" || claims.App == "" || claims.Deployment == "" ||
		claims.IssuedAt > current || claims.ExpiresAt <= current || claims.ExpiresAt-claims.IssuedAt > int64(edgeGrantLifetime/time.Second) {
		return claims, errors.New("edge grant expired or bound to another request")
	}
	return claims, nil
}
