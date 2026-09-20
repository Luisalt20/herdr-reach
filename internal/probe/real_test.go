package probe

// This file pins the production verifier's safety refusals from inside the
// package, and the issuer fold's agreement with the declared expected set:
// R-HR-04's "verification is never weakened" property and the two halves of the
// issuer contract, neither of which the external suite can reach because
// productionTLSVerifier, chainIssuer and declaredExpectedIssuers are unexported.
//
// It touches no socket and never calls ProductionSeams: both refusal paths
// return before any dial, so the zero-value verifier is enough to exercise them
// without a seam set or a network. The chain's certificates are scripted for the
// same reason — the fold's format is a decision this file pins, not a claim
// about a live chain.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"slices"
	"testing"
)

// TestProductionTLSVerifierRefusesNilConfiguration asserts the seam's first
// refusal: a handshake cannot run without the caller's configuration, and the
// refusal is an attempt that produced no answer rather than a rejected chain.
func TestProductionTLSVerifierRefusesNilConfiguration(t *testing.T) {
	_, err := productionTLSVerifier{}.Verify(context.Background(), "www.cloudflare.com:443", nil)
	if err == nil {
		t.Fatal("Verify accepted a nil configuration: the seam requires the caller's configuration")
	}
	if errors.Is(err, ErrTLSVerification) {
		t.Errorf("the refusal %v matches ErrTLSVerification: a missing configuration is an attempt that produced no answer, never a rejected chain", err)
	}
}

// TestProductionTLSVerifierRefusesInsecureSkipVerify pins R-HR-04's one absolute:
// a configuration that disables certificate verification is refused rather than
// honored, because honoring it would perform the unverified handshake the
// contract forbids. The refusal is not a rejected chain, and it leaves the
// caller's configuration exactly as it was.
func TestProductionTLSVerifierRefusesInsecureSkipVerify(t *testing.T) {
	roots := x509.NewCertPool()
	const serverName = "www.cloudflare.com"
	cfg := &tls.Config{
		InsecureSkipVerify: true,
		RootCAs:            roots,
		ServerName:         serverName,
		MinVersion:         tls.VersionTLS12,
	}
	before := struct {
		insecure   bool
		rootCAs    *x509.CertPool
		serverName string
		minVersion uint16
	}{cfg.InsecureSkipVerify, cfg.RootCAs, cfg.ServerName, cfg.MinVersion}

	_, err := productionTLSVerifier{}.Verify(context.Background(), "www.cloudflare.com:443", cfg)
	if err == nil {
		t.Fatal("Verify honored a configuration that disables verification: the unverified handshake is forbidden")
	}
	if errors.Is(err, ErrTLSVerification) {
		t.Errorf("the refusal %v matches ErrTLSVerification: a refused configuration is not a rejected chain", err)
	}

	// The refusal reads the caller's configuration and never rewrites it.
	if cfg.InsecureSkipVerify != before.insecure {
		t.Errorf("the refusal modified cfg.InsecureSkipVerify to %t, want the caller's %t", cfg.InsecureSkipVerify, before.insecure)
	}
	if cfg.RootCAs != before.rootCAs {
		t.Errorf("the refusal replaced cfg.RootCAs (%p), want the caller's non-nil pool %p", cfg.RootCAs, before.rootCAs)
	}
	if cfg.ServerName != before.serverName {
		t.Errorf("the refusal modified cfg.ServerName to %q, want the caller's %q", cfg.ServerName, before.serverName)
	}
	if cfg.MinVersion != before.minVersion {
		t.Errorf("the refusal modified cfg.MinVersion to %#x, want the caller's %#x", cfg.MinVersion, before.minVersion)
	}
}

// TestChainIssuerFormatAgreesWithDeclaredIssuers pins the fold's format against
// the value the declared expected set carries for the TLS probe's host: the
// recorded chain's leaf is issued by "Let's Encrypt" and its topmost certificate
// is "ISRG Root X1", so the fold must render "Let's Encrypt/ISRG" — the same
// string tls.go declares for that host. The certificates are scripted; the case
// checks the format and the agreement, never a live chain.
func TestChainIssuerFormatAgreesWithDeclaredIssuers(t *testing.T) {
	chain := []*x509.Certificate{
		{Issuer: pkix.Name{Organization: []string{"Let's Encrypt"}}},
		{Subject: pkix.Name{CommonName: "ISRG Root X1"}},
	}
	const want = "Let's Encrypt/ISRG"
	if got := chainIssuer(chain); got != want {
		t.Fatalf("chainIssuer(recorded-shaped chain) = %q, want the documented form %q", got, want)
	}
	if !slices.Contains(declaredExpectedIssuers[tlsProbeHost], want) {
		t.Errorf("the declared expected issuers for %q are %v, which do not carry the fold's %q: the two halves of the issuer contract have drifted", tlsProbeHost, declaredExpectedIssuers[tlsProbeHost], want)
	}
	if !expectedIssuerFor(tlsProbeHost, want) {
		t.Errorf("expectedIssuerFor(%q, %q) = false, want the folded issuer to be one of the declared values", tlsProbeHost, want)
	}
}

// TestChainIssuerEmptyAndNamelessRendersNothing pins the documented empty-input
// behaviour of the fold: no certificates at all renders the empty string, and a
// certificate whose organization and common name are both absent contributes
// nothing rather than a fabricated token.
func TestChainIssuerEmptyAndNamelessRendersNothing(t *testing.T) {
	if got := chainIssuer(nil); got != "" {
		t.Errorf("chainIssuer(nil) = %q, want the empty string: no certificates, no digest", got)
	}
	if got := chainIssuer([]*x509.Certificate{}); got != "" {
		t.Errorf("chainIssuer(empty) = %q, want the empty string", got)
	}
	if got := chainIssuer([]*x509.Certificate{{}}); got != "" {
		t.Errorf("chainIssuer(one certificate with no organization and no common name) = %q, want the empty string: an absent name contributes nothing rather than a fabricated token", got)
	}
}
