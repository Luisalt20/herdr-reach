package probe

// This file pins the production verifier's safety refusals from inside the
// package, the issuer fold's agreement with the declared expected set, and the
// connected-socket adapter's WriteTo: R-HR-04's "verification is never weakened"
// property, the two halves of the issuer contract, and the datagram write path
// the kernel refuses on a connected socket. None of them is reachable from the
// external suite because productionTLSVerifier, chainIssuer,
// declaredExpectedIssuers and connectedPacketConn are unexported.
//
// It touches no socket and never calls ProductionSeams: both refusal paths
// return before any dial, so the zero-value verifier is enough to exercise them,
// and the packet adapter is driven through a scripted fake. The chain's
// certificates are scripted for the same reason — the fold's format is a
// decision this file pins, not a claim about a live chain.

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
