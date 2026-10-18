package internal

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"slices"
)

const (
	pinSectigoDVE36   = "ZSagvDzjltLkewXEBuDxIzpW/dpVw1Juvvmd0hhkzdY="
	pinSectigoRootE46 = "sLVjNUaFYfW7n6EtgBeEpjOlcnBdNPMrZDRF36iwBdE="
	pinLetsEncryptYR1 = "LoMHBotttiDko50Gi13uXW71eIy7LAttI+rYT8wXF4w="
	pinISRGRootYR     = "fk6IOKit1ild5647BH06ujSIq5XbCgqlbYl6ANhhi88="
	pinISRGRootYE     = "sCkq5UWXjg+7mKu9lMhhYF5bGLsy7VI/UNW3tccdR7w="
)

const maxRedirects = 10

var githubAssetHostPins = []string{pinLetsEncryptYR1, pinISRGRootYR, pinISRGRootYE}

var jarHostPins = map[string][]string{
	"github.com":                           {pinSectigoDVE36, pinSectigoRootE46},
	"release-assets.githubusercontent.com": githubAssetHostPins,
	"objects.githubusercontent.com":        githubAssetHostPins,
}

func spkiPin(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func pinVerifier(pins map[string][]string) func(tls.ConnectionState) error {
	return func(state tls.ConnectionState) error {
		allowed, ok := pins[state.ServerName]
		if !ok {
			return fmt.Errorf("no certificate pin configured for host %q", state.ServerName)
		}
		for _, chain := range state.VerifiedChains {
			for _, cert := range chain {
				if slices.Contains(allowed, spkiPin(cert)) {
					return nil
				}
			}
		}
		return fmt.Errorf("certificate pin mismatch for host %q", state.ServerName)
	}
}

func requireHTTPSRedirect(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" {
		return errors.New("refusing redirect to a non-https URL")
	}
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}
	return nil
}

func newPinnedClient(pins map[string][]string, roots *x509.CertPool) *http.Client {
	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		ForceAttemptHTTP2: true,
		TLSClientConfig: &tls.Config{
			MinVersion:       tls.VersionTLS12,
			RootCAs:          roots,
			VerifyConnection: pinVerifier(pins),
		},
	}
	return &http.Client{
		Transport:     transport,
		CheckRedirect: requireHTTPSRedirect,
	}
}
