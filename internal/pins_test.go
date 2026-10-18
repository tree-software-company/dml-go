package internal

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type testPKI struct {
	ca       *x509.Certificate
	leaf     *x509.Certificate
	leafPair tls.Certificate
	roots    *x509.CertPool
}

func newTestPKI(t *testing.T) testPKI {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "dml-go test ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(ca)

	return testPKI{
		ca:       ca,
		leaf:     leaf,
		leafPair: tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: leafKey},
		roots:    roots,
	}
}

func startTLSServer(t *testing.T, pki testPKI, handler http.Handler) string {
	t.Helper()

	server := httptest.NewUnstartedServer(handler)
	server.TLS = &tls.Config{Certificates: []tls.Certificate{pki.leafPair}}
	server.StartTLS()
	t.Cleanup(server.Close)

	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	return "https://localhost:" + port
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
}

func TestPinnedClientAcceptsPinnedIssuer(t *testing.T) {
	pki := newTestPKI(t)
	url := startTLSServer(t, pki, okHandler())
	client := newPinnedClient(map[string][]string{"localhost": {spkiPin(pki.ca)}}, pki.roots)

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("expected pinned issuer to be accepted, got %v", err)
	}
	resp.Body.Close()
}

func TestPinnedClientAcceptsPinnedLeaf(t *testing.T) {
	pki := newTestPKI(t)
	url := startTLSServer(t, pki, okHandler())
	client := newPinnedClient(map[string][]string{"localhost": {spkiPin(pki.leaf)}}, pki.roots)

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("expected pinned leaf to be accepted, got %v", err)
	}
	resp.Body.Close()
}

func TestPinnedClientAcceptsAnyPinFromList(t *testing.T) {
	pki := newTestPKI(t)
	url := startTLSServer(t, pki, okHandler())
	client := newPinnedClient(map[string][]string{"localhost": {pinSectigoDVE36, spkiPin(pki.ca)}}, pki.roots)

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("expected a matching pin anywhere in the list to be accepted, got %v", err)
	}
	resp.Body.Close()
}

func TestPinnedClientRejects(t *testing.T) {
	pki := newTestPKI(t)
	url := startTLSServer(t, pki, okHandler())

	tests := []struct {
		name    string
		pins    map[string][]string
		roots   *x509.CertPool
		wantErr string
	}{
		{
			name:    "wrong pin",
			pins:    map[string][]string{"localhost": {pinSectigoDVE36}},
			roots:   pki.roots,
			wantErr: "certificate pin mismatch",
		},
		{
			name:    "host without pin entry",
			pins:    map[string][]string{"example.com": {spkiPin(pki.ca)}},
			roots:   pki.roots,
			wantErr: "no certificate pin configured",
		},
		{
			name:    "empty pin list",
			pins:    map[string][]string{"localhost": {}},
			roots:   pki.roots,
			wantErr: "certificate pin mismatch",
		},
		{
			name:    "matching pin but issuer not trusted",
			pins:    map[string][]string{"localhost": {spkiPin(pki.ca)}},
			roots:   nil,
			wantErr: "x509",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := newPinnedClient(tt.pins, tt.roots).Get(url)
			if err == nil {
				resp.Body.Close()
				t.Fatal("expected the connection to be rejected")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestPinnedClientRefusesRedirectToHTTP(t *testing.T) {
	pki := newTestPKI(t)
	plain := httptest.NewServer(okHandler())
	t.Cleanup(plain.Close)

	url := startTLSServer(t, pki, http.RedirectHandler(plain.URL, http.StatusFound))
	client := newPinnedClient(map[string][]string{"localhost": {spkiPin(pki.ca)}}, pki.roots)

	resp, err := client.Get(url)
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected redirect to http to be refused")
	}
	if !strings.Contains(err.Error(), "non-https") {
		t.Fatalf("expected non-https redirect error, got %v", err)
	}
}

func TestPinnedClientStopsRedirectLoop(t *testing.T) {
	pki := newTestPKI(t)
	var url string
	url = startTLSServer(t, pki, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, url+"/again", http.StatusFound)
	}))
	client := newPinnedClient(map[string][]string{"localhost": {spkiPin(pki.ca)}}, pki.roots)

	resp, err := client.Get(url)
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected redirect loop to be stopped")
	}
	if !strings.Contains(err.Error(), "stopped after") {
		t.Fatalf("expected redirect limit error, got %v", err)
	}
}

func skipUnlessLive(t *testing.T) {
	t.Helper()
	if os.Getenv("DML_LIVE_PIN_TEST") != "1" {
		t.Skip("set DML_LIVE_PIN_TEST=1 to run live network tests")
	}
}

func TestEnsureJarLive(t *testing.T) {
	skipUnlessLive(t)

	cacheRoot := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheRoot)
	t.Setenv("HOME", cacheRoot)

	jarPath, err := EnsureJar()
	if err != nil {
		t.Fatalf("expected the pinned download to succeed, got %v", err)
	}
	valid, err := verifySHA256(jarPath, jarSHA256)
	if err != nil || !valid {
		t.Fatalf("expected the downloaded jar to match the pinned SHA256, valid=%v err=%v", valid, err)
	}
}

func TestWrongPinRejectedLive(t *testing.T) {
	skipUnlessLive(t)

	client := newPinnedClient(map[string][]string{"github.com": {pinISRGRootYR}}, nil)

	resp, err := client.Head(jarURL)
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected a wrong pin to be rejected by github.com")
	}
	if !strings.Contains(err.Error(), "certificate pin mismatch") {
		t.Fatalf("expected pin mismatch error, got %v", err)
	}
}
