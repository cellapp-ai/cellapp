package hosting

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestSlugHostBrowser(t *testing.T) {
	if os.Getenv("RUN_BROWSER_TESTS") != "1" {
		t.Skip("RUN_BROWSER_TESTS=1 requires local Chrome")
	}
	f := setup(t)
	server := httptest.NewUnstartedServer(nil)
	_, port, _ := net.SplitHostPort(server.Listener.Addr().String())
	f.server.Config.ControlOrigin = "https://control.localhost:" + port
	f.server.Config.AppPort = port
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	f.server.Web, err = LoadWebFS(filepath.Join(root, "apps/web/dist"))
	if err != nil {
		t.Fatal(err)
	}
	f.handler = f.server.Handler()
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.handler.ServeHTTP(w, r) })
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Cellapp slug browser test"}, DNSNames: []string{"control.localhost", "*.apps.localhost"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	if err != nil {
		t.Fatal(err)
	}
	server.TLS = &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	defer server.Close()
	ownerCookie := f.webSessionCookie(t)
	app := response[App](t, f.request(t, "POST", "/apps", map[string]any{"name": "Notes", "namespace": "alice", "slug": "notes", "key": token(), "requestId": id()}, f.token, "control.localhost:"+port, ""), 201)
	spki, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(spki)
	command := exec.Command("node", "--import", "tsx", "tests/slug-host.browser.ts")
	command.Dir = root
	command.Env = append(os.Environ(), "TEST_CONTROL_ORIGIN="+f.server.Config.ControlOrigin, "TEST_APP_URL="+app.URL, "TEST_APP_ID="+app.ID, "TEST_OWNER_COOKIE="+ownerCookie, "TEST_BROWSER_SPKI="+base64.StdEncoding.EncodeToString(digest[:]))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("slug host browser failed: %v\n%s", err, output)
	}
	t.Log(string(output))
}
