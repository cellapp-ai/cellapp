package hosting

import (
	"context"
	"crypto"
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
	"runtime"
	"strings"
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
	// Browser acceptance uses real S3 so a storage substitute cannot mask release failures.
	storageConfig, e := LoadConfig(os.Getenv)
	if e != nil {
		t.Fatal(e)
	}
	storage, e := NewStorage(storageConfig)
	if e != nil {
		t.Fatal(e)
	}
	probeCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if e = storage.Health(probeCtx); e != nil {
		t.Fatalf("isolated S3 bucket is required for browser acceptance: %v", e)
	}
	f.server.Storage = storage
	server := httptest.NewUnstartedServer(nil)
	_, port, _ := net.SplitHostPort(server.Listener.Addr().String())
	f.server.Config.ControlOrigin = "https://control.localhost:" + port
	f.server.Config.AppPort = port
	f.server.Identity = browserIdentity{origin: f.server.Config.ControlOrigin}
	root, e := filepath.Abs("../../../..")
	if e != nil {
		t.Fatal(e)
	}
	f.server.Web, e = LoadWebFS(filepath.Join(root, "apps/web/dist"))
	if e != nil {
		t.Fatal(e)
	}
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
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Cellapp isolated browser test"}, DNSNames: []string{"control.localhost", "*.apps.localhost"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	parent := cert
	var signer crypto.Signer = key
	caPEM := []byte(nil)
	providedCA := os.Getenv("TEST_BROWSER_CA_CERT")
	providedKey := os.Getenv("TEST_BROWSER_CA_KEY")
	if (providedCA == "") != (providedKey == "") {
		t.Fatal("Set both TEST_BROWSER_CA_CERT and TEST_BROWSER_CA_KEY to an isolated, already trusted test CA")
	}
	if providedCA != "" {
		caPEM, e = os.ReadFile(providedCA)
		if e != nil {
			t.Fatal(e)
		}
		caKey, e := os.ReadFile(providedKey)
		if e != nil {
			t.Fatal(e)
		}
		caPair, e := tls.X509KeyPair(caPEM, caKey)
		if e != nil {
			t.Fatal("invalid test CA certificate/key pair")
		}
		parent, e = x509.ParseCertificate(caPair.Certificate[0])
		if e != nil || !parent.IsCA || time.Now().After(parent.NotAfter) {
			t.Fatal("test CA must be valid and unexpired")
		}
		var ok bool
		signer, ok = caPair.PrivateKey.(crypto.Signer)
		if !ok {
			t.Fatal("test CA key cannot sign")
		}
		cert.IsCA = false
		cert.KeyUsage = x509.KeyUsageDigitalSignature
	}
	der, e := x509.CreateCertificate(rand.Reader, cert, parent, &key.PublicKey, signer)
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
	ca := filepath.Join(t.TempDir(), "root.pem")
	if caPEM == nil {
		caPEM = certPEM
	}
	if e = os.WriteFile(ca, caPEM, 0600); e != nil {
		t.Fatal(e)
	}
	if providedCA == "" {
		trustBrowserCertificate(t, ca)
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

// Chromium on macOS uses the OS trust store. Trust only this disposable CA,
// and restore the keychain search list and trust settings after the test.
func trustBrowserCertificate(t *testing.T, ca string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Fatal("Browser CA trust setup currently requires macOS; TLS verification is never disabled")
	}
	run := func(args ...string) []byte {
		out, e := exec.Command("security", args...).CombinedOutput()
		if e != nil {
			t.Fatalf("temporary browser CA trust failed: %v: %s", e, out)
		}
		return out
	}
	out := string(run("list-keychains", "-d", "user"))
	var original []string
	for _, line := range strings.Split(out, "\n") {
		if value := strings.Trim(strings.TrimSpace(line), `"`); value != "" {
			original = append(original, value)
		}
	}
	keychain := filepath.Join(t.TempDir(), "browser.keychain-db")
	run("create-keychain", "-p", "isolated-browser-test", keychain)
	trusted := false
	t.Cleanup(func() {
		if trusted {
			if out, e := exec.Command("security", "remove-trusted-cert", ca).CombinedOutput(); e != nil {
				t.Errorf("remove temporary CA: %v: %s", e, out)
			}
		}
		args := append([]string{"list-keychains", "-d", "user", "-s"}, original...)
		if out, e := exec.Command("security", args...).CombinedOutput(); e != nil {
			t.Errorf("restore keychains: %v: %s", e, out)
		}
		if out, e := exec.Command("security", "delete-keychain", keychain).CombinedOutput(); e != nil {
			t.Errorf("delete temporary keychain: %v: %s", e, out)
		}
	})
	run("unlock-keychain", "-p", "isolated-browser-test", keychain)
	run(append([]string{"list-keychains", "-d", "user", "-s", keychain}, original...)...)
	run("add-trusted-cert", "-r", "trustRoot", "-k", keychain, ca)
	trusted = true
}
