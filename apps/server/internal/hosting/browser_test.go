package hosting

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

type browserIdentity struct {
	identityStub
	origin string
}

func (b browserIdentity) Start(state, challenge string) string {
	return b.origin + "/_test/identity?state=" + url.QueryEscape(state)
}

func TestBrowserEndToEnd(t *testing.T) {
	if os.Getenv("RUN_BROWSER_TESTS") != "1" {
		t.Skip("RUN_BROWSER_TESTS=1 and a local Chrome executable are required")
	}
	f := setup(t)
	server := httptest.NewUnstartedServer(nil)
	_, port, _ := net.SplitHostPort(server.Listener.Addr().String())
	f.server.Config.ControlOrigin = "https://control.localhost:" + port
	f.server.Config.AppPort = port
	f.server.Identity = browserIdentity{origin: f.server.Config.ControlOrigin}
	handler := f.server.Handler()
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only this test fixture exposes a simulated identity provider; production has no bypass.
		if r.URL.Path == "/_test/identity" {
			http.Redirect(w, r, "/auth/callback?code=valid&state="+url.QueryEscape(r.URL.Query().Get("state")), 302)
			return
		}
		handler.ServeHTTP(w, r)
	})
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CellApp isolated browser test"}, DNSNames: []string{"control.localhost", "*.apps.localhost"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	der, e := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if e != nil {
		t.Fatal(e)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pair, e := tls.X509KeyPair(certPEM, keyPEM)
	if e != nil {
		t.Fatal(e)
	}
	server.TLS = &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
	server.StartTLS()
	defer server.Close()
	root, e := filepath.Abs("../../../..")
	if e != nil {
		t.Fatal(e)
	}
	ca := filepath.Join(t.TempDir(), "root.pem")
	if e = os.WriteFile(ca, certPEM, 0600); e != nil {
		t.Fatal(e)
	}
	command := exec.Command("node", "--import", "tsx", "tests/browser.e2e.ts")
	command.Dir = root
	command.Env = append(os.Environ(), "TEST_CONTROL_ORIGIN="+f.server.Config.ControlOrigin, "NODE_EXTRA_CA_CERTS="+ca)
	output, e := command.CombinedOutput()
	if e != nil {
		t.Fatalf("browser test failed: %v\n%s", e, output)
	}
	t.Log(string(output))
}
