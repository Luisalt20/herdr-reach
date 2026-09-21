package probe

// This file pins the production verifier's safety refusals from inside the
// package, the issuer/anchor split's agreement with the declared expected set,
// and the connected-socket adapter's WriteTo: R-HR-04's "verification is never
// weakened" property, the compared publisher and the anchor that must never
// decide the comparison (issue #78), and the datagram write path the kernel
// refuses on a connected socket. None of them is reachable from the external
// suite because productionTLSVerifier, chainIssuer, chainAnchor,
// declaredExpectedIssuers and connectedPacketConn are unexported.
//
// It touches no socket and never calls ProductionSeams: both refusal paths
// return before any dial, so the zero-value verifier is enough to exercise them,
// and the packet adapter is driven through a scripted fake. The chain's
// publishes one — the string's format is a decision this file pins, not a claim
// about a live chain.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"net"
	"slices"
	"testing"
	"time"
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

// TestChainIssuerAndAnchorAreSeparate pins the two halves of the chain contract
// at the point where they are produced: chainIssuer is the leaf's issuing
// organization only ("Let's Encrypt"), chainAnchor is the topmost certificate's
// short name ("ISRG" for "ISRG Root X1"), the declared expected set carries the
// former, and the anchor is not a comparison key. The certificates are scripted;
// no live chain is claimed.
//
// The portability control is the reason the split exists (issue #78): the same
// leaf issuer beside a differently anchored chain must produce the same compared
// value — the declaration has to hold on any machine's trust store — while the
// anchor still renders as evidence. A regression that folds the anchor back into
// chainIssuer fails the second chain's first assertion.
func TestChainIssuerAndAnchorAreSeparate(t *testing.T) {
	recorded := []*x509.Certificate{
		{Issuer: pkix.Name{Organization: []string{"Let's Encrypt"}}},
		{Subject: pkix.Name{CommonName: "ISRG Root X1"}},
	}
	if got, want := chainIssuer(recorded), "Let's Encrypt"; got != want {
		t.Fatalf("chainIssuer(recorded-shaped chain) = %q, want the issuing organization %q", got, want)
	}
	if got, want := chainAnchor(recorded), "ISRG"; got != want {
		t.Fatalf("chainAnchor(recorded-shaped chain) = %q, want the topmost short name %q", got, want)
	}
	if !slices.Contains(declaredExpectedIssuers[tlsProbeHost], chainIssuer(recorded)) {
		t.Errorf("the declared expected issuers for %q are %v, which do not carry the compared value %q: the two halves of the issuer contract have drifted", tlsProbeHost, declaredExpectedIssuers[tlsProbeHost], chainIssuer(recorded))
	}
	if !expectedIssuerFor(tlsProbeHost, chainIssuer(recorded)) {
		t.Errorf("expectedIssuerFor(%q, %q) = false, want the issuing organization to be one of the declared values", tlsProbeHost, chainIssuer(recorded))
	}
	if slices.Contains(declaredExpectedIssuers[tlsProbeHost], chainAnchor(recorded)) {
		t.Errorf("the declared expected issuers for %q carry the anchor %q, which is chosen by the local trust store and must never decide the comparison", tlsProbeHost, chainAnchor(recorded))
	}

	otherAnchor := []*x509.Certificate{
		{Issuer: pkix.Name{Organization: []string{"Let's Encrypt"}}},
		{Subject: pkix.Name{CommonName: "GlobalSign Root CA"}},
	}
	if got, want := chainIssuer(otherAnchor), "Let's Encrypt"; got != want {
		t.Errorf("chainIssuer(chain anchored elsewhere) = %q, want the unchanged issuing organization %q: the anchor is chosen by the local store and must not move the compared value", got, want)
	}
	if got, want := chainAnchor(otherAnchor), "GlobalSign"; got != want {
		t.Errorf("chainAnchor(chain anchored elsewhere) = %q, want %q", got, want)
	}
}

// TestChainIssuerAndAnchorFallBackToTheOtherName pins the documented fallbacks:
// an issuer with no organization is named by its common name, and an anchor
// certificate with no common name is named by its organization. Both keep a
// presented name rather than dropping it.
func TestChainIssuerAndAnchorFallBackToTheOtherName(t *testing.T) {
	commonNameOnly := []*x509.Certificate{{Issuer: pkix.Name{CommonName: "Acme Intermediate"}}}
	if got, want := chainIssuer(commonNameOnly), "Acme Intermediate"; got != want {
		t.Errorf("chainIssuer(issuer with no organization) = %q, want its common name %q", got, want)
	}
	organizationOnly := []*x509.Certificate{{}, {Subject: pkix.Name{Organization: []string{"ISRG"}}}}
	if got, want := chainAnchor(organizationOnly), "ISRG"; got != want {
		t.Errorf("chainAnchor(anchor with no common name) = %q, want its organization %q", got, want)
	}

	// The compared value never borrows the anchor. A leaf whose issuer presents no
	// name compares as nothing even when the chain is anchored at a named root: a
	// fallback to the anchor here would make the compared value depend on the local
	// trust store again, which is the defect issue #78 fixed.
	namelessIssuerNamedAnchor := []*x509.Certificate{{}, {Subject: pkix.Name{CommonName: "ISRG Root X1"}}}
	if got := chainIssuer(namelessIssuerNamedAnchor); got != "" {
		t.Errorf("chainIssuer(leaf issuer with no name, anchored at a named root) = %q, want the empty string: the compared value must not borrow the anchor", got)
	}
	if got, want := chainAnchor(namelessIssuerNamedAnchor), "ISRG"; got != want {
		t.Errorf("chainAnchor(leaf issuer with no name, anchored at a named root) = %q, want %q", got, want)
	}
}

// TestChainIssuerAndAnchorEmptyAndNamelessRendersNothing pins the documented
// empty-input behaviour of both halves: no certificates at all renders the empty
// string, and a certificate whose organization and common name are both absent
// contributes nothing rather than a fabricated token.
func TestChainIssuerAndAnchorEmptyAndNamelessRendersNothing(t *testing.T) {
	for _, chain := range [][]*x509.Certificate{nil, {}, {{}}} {
		if got := chainIssuer(chain); got != "" {
			t.Errorf("chainIssuer(%v) = %q, want the empty string: no issuer name, no compared value", chain, got)
		}
		if got := chainAnchor(chain); got != "" {
			t.Errorf("chainAnchor(%v) = %q, want the empty string: an absent name contributes nothing rather than a fabricated token", chain, got)
		}
	}
}

// fakeConnectedDatagram is a scripted connected datagram socket: it records what
// the adapter asks of it, so the adapter can be pinned without opening one. It
// implements WriteTo deliberately — a connected socket has one, and the kernel
// refuses it — so a call to it is observable; the adapter must never make one.
type fakeConnectedDatagram struct {
	written      []byte
	writeCalls   int
	writeErr     error
	writeToCalls int

	readPayload []byte
	readAddr    net.Addr
	readErr     error

	deadlineSet bool
	deadline    time.Time
	deadlineErr error

	closed   bool
	closeErr error
}

// Write records the datagram and answers with its length.
func (f *fakeConnectedDatagram) Write(p []byte) (int, error) {
	f.writeCalls++
	f.written = append(f.written, p...)
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return len(p), nil
}

// WriteTo is the call the adapter must never make on a connected socket.
func (f *fakeConnectedDatagram) WriteTo(_ []byte, _ net.Addr) (int, error) {
	f.writeToCalls++
	return 0, errors.New("a connected socket must not be written through WriteTo")
}

// ReadFrom returns the scripted datagram, its address and its error.
func (f *fakeConnectedDatagram) ReadFrom(p []byte) (int, net.Addr, error) {
	copy(p, f.readPayload)
	return len(f.readPayload), f.readAddr, f.readErr
}

// SetDeadline records the deadline it was given.
func (f *fakeConnectedDatagram) SetDeadline(t time.Time) error {
	f.deadlineSet, f.deadline = true, t
	return f.deadlineErr
}

// Close records that it was closed.
func (f *fakeConnectedDatagram) Close() error {
	f.closed = true
	return f.closeErr
}

// TestConnectedPacketConnServesWriteToThroughWrite pins the adapter's one
// non-pass-through method: a connected socket refuses WriteTo in the kernel, so
// the seam's WriteTo must write through Write and must never call the
// underlying WriteTo. A revert to the raw socket's WriteTo fails here, and the
// scripted fake keeps the case off every socket.
func TestConnectedPacketConnServesWriteToThroughWrite(t *testing.T) {
	target := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 7844}
	fake := &fakeConnectedDatagram{}
	conn := connectedPacketConn{conn: fake}

	n, err := conn.WriteTo([]byte("datagram"), target)
	if err != nil {
		t.Fatalf("WriteTo returned an error: %v", err)
	}
	if n != len("datagram") {
		t.Errorf("WriteTo reported %d bytes, want %d", n, len("datagram"))
	}
	if fake.writeCalls != 1 || string(fake.written) != "datagram" {
		t.Errorf("WriteTo wrote %q in %d Write call(s), want the datagram in one Write", fake.written, fake.writeCalls)
	}
	if fake.writeToCalls != 0 {
		t.Errorf("WriteTo called the connected socket's WriteTo %d time(s): the kernel refuses it, and that refusal is the production bug this adapter exists for", fake.writeToCalls)
	}

	// ReadFrom is a pass-through: the socket's own payload, address and error
	// reach the caller unchanged.
	from := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 46846}
	readErr := errors.New("the read failed")
	fake.readPayload, fake.readAddr, fake.readErr = []byte("reply"), from, readErr
	buffer := make([]byte, 16)
	read, addr, err := conn.ReadFrom(buffer)
	if read != len("reply") || string(buffer[:read]) != "reply" {
		t.Errorf("ReadFrom returned %d bytes %q, want the socket's own payload %q", read, buffer[:read], "reply")
	}
	if addr == nil || addr.String() != from.String() {
		t.Errorf("ReadFrom returned the address %v, want the socket's own %v", addr, from)
	}
	if !errors.Is(err, readErr) {
		t.Errorf("ReadFrom returned the error %v, want the socket's own %v", err, readErr)
	}

	// SetDeadline and Close are forwarded, including their errors.
	deadline := time.Unix(1700000000, 0)
	if err := conn.SetDeadline(deadline); err != nil {
		t.Errorf("SetDeadline returned an error: %v", err)
	}
	if !fake.deadlineSet || !fake.deadline.Equal(deadline) {
		t.Errorf("SetDeadline did not reach the socket: set=%t value=%v", fake.deadlineSet, fake.deadline)
	}
	if err := conn.Close(); err != nil {
		t.Errorf("Close returned an error: %v", err)
	}
	if !fake.closed {
		t.Errorf("Close did not reach the socket")
	}

	deadlineErr := errors.New("the deadline was refused")
	closeErr := errors.New("the close was refused")
	fake.deadlineErr, fake.closeErr = deadlineErr, closeErr
	if err := conn.SetDeadline(deadline); !errors.Is(err, deadlineErr) {
		t.Errorf("SetDeadline returned %v, want the socket's own %v", err, deadlineErr)
	}
	if err := conn.Close(); !errors.Is(err, closeErr) {
		t.Errorf("Close returned %v, want the socket's own %v", err, closeErr)
	}
}
