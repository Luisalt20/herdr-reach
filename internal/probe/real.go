package probe

// This file is the production seam set of design §6.1: the real capabilities a
// live run is given, built over the standard library's own primitives.
//
// It is the only file in the module that may construct a real network or process
// primitive. internal/probe/guard_test.go enforces that statically (design §6.2
// level 3): every construction of net.Dial*, net.Lookup*, net.ListenPacket,
// tls.Dial* and os/exec outside this file and cmd/herdr-reach/main.go fails the
// unit suite. The confinement is what makes "the deny-all set is the test seam of
// record" checkable rather than aspirational.
//
// ProductionSeams is the one constructor. It is called from
// cmd/herdr-reach/main.go and nowhere else, and tests never call it, because a
// test that built production seams would be measuring the machine the suite
// happens to run on.
//
// What this file deliberately does not do:
//
//   - no command execution. CommandRunner is left nil: it is the only seam that
//     can run a third-party binary, and R1a leaves it absent so no code path can
//     run `sshd`, `cloudflared`, `herdr` or anything else (R-HR-02, design §6.2).
//     A probe that needs the capability reports it as excluded, never as failed.
//   - no writes. Every capability here reads, or opens an outbound connection;
//     nothing creates, moves, deletes or changes a file, an environment entry or
//     a service. The FS seam exposes ReadFile, Stat and Getenv and nothing more,
//     so a write is not expressible through it.
//   - no verdicts. The TLS verifier reports what the handshake produced and never
//     a reason code: the classification table is the only place a code is chosen
//     (R-HR-07).

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

// The machine-evidence paths the Platform seam reads on Linux. They are named
// here, beside the seam that reads them, so a reviewer can see exactly which two
// facts a classification rests on.
const (
	// kernelReleasePath is the kernel's own version string. WSL2 kernels identify
	// themselves in it; the detection below reads the machine's answer rather
	// than asking WSL-specific tooling the run was not given.
	kernelReleasePath = "/proc/sys/kernel/osrelease"
	// systemdRuntimeDir is the service-manager runtime directory systemd creates
	// when it is the running service manager. Its presence is the signal; its
	// absence is reported as "no systemd signal", never as a different manager.
	systemdRuntimeDir = "/run/systemd/system"
)

// verificationOKCode is the verify code of an accepted chain. The field is a
// string because verifiers are not required to number their refusals; "0" is the
// conventional code for "ok" (PRD §1.1 records the same value for the measured
// chain).
const verificationOKCode = "0"

// ProductionSeams returns the seam set a live run is given: every capability of
// design §6.1 implemented over the standard library, except CommandRunner, which
// is deliberately left nil.
//
// This file is the only place real network primitives are constructed, and the
// constructor is called from cmd/herdr-reach/main.go and nowhere else. Tests
// never call it: a test that built production seams would measure the machine
// the suite happens to run on, and internal/probe/guard_test.go asserts the call
// confinement statically.
//
// The constructor takes no input. A configurable production set would let a
// caller hand a live run a partially real seam set, and a half-real measurement
// is worse than none: the run would answer some questions from the machine and
// some from nothing. A test that needs one scripted capability starts from
// DenyAllSeams and overrides exactly that field.
//
// The returned value is fresh on every call; nothing here holds package state, so
// two runs cannot share one connection, one resolver or one clock.
func ProductionSeams() Seams {
	dialer := &net.Dialer{}
	return Seams{
		Dialer:       productionDialer{dialer: dialer},
		Resolver:     productionResolver{resolver: &net.Resolver{}},
		TLSVerifier:  productionTLSVerifier{dialer: dialer},
		PacketDialer: productionPacketDialer{dialer: dialer},
		// CommandRunner is deliberately absent, not forgotten. It is the only
		// seam that can execute a third-party binary, and R1a leaves it nil so no
		// code path — including a probe this slice did not write — can run
		// `sshd`, `cloudflared`, `herdr` or anything else (R-HR-02, design §6.2).
		// A nil runner is read by every probe as an excluded capability, never as
		// a failure.
		CommandRunner: nil,
		Clock:         productionClock{},
		FS:            productionFS{},
		Platform:      productionPlatform{},
	}
}

// productionDialer is the Dialer seam over net.Dialer. net.Dialer.DialContext
// already treats the context as the attempt's deadline and cancellation, so the
// probes' budgets travel through it without this file re-implementing them; a
// separate timeout here would be a second budget beside the one the probe owns.
type productionDialer struct{ dialer *net.Dialer }

// DialContext opens the connection under the context's deadline and cancellation.
func (d productionDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.dialer.DialContext(ctx, network, addr)
}

// productionResolver is the Resolver seam over net.Resolver. The resolver is the
// system's own: the run must report what this machine's name resolution does, not
// what a bespoke resolver would do.
type productionResolver struct{ resolver *net.Resolver }

// LookupHost resolves the host under the context's deadline and cancellation.
func (r productionResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return r.resolver.LookupHost(ctx, host)
}

// productionTLSVerifier performs a real TLS handshake with the caller's
// configuration. It owns the socket for the duration of the handshake; the
// configuration stays the caller's.
//
// Verification is never weakened (R-HR-04). The caller's *tls.Config is handed to
// the handshake unchanged, verification is performed exactly once, and there is
// no unverified retry: a verifier that second-guessed crypto/tls would be a
// second trust decision, and the seam exists so the trust decision has exactly
// one owner. A configuration that already disables verification is refused
// rather than honored, because honoring it would perform the unverified
// handshake this contract forbids.
type productionTLSVerifier struct{ dialer *net.Dialer }

// Verify handshakes with target using cfg and reports the observed chain.
//
// A chain the verifier rejects returns a *tlsChainRejected, which wraps
// ErrTLSVerification while keeping crypto/tls's own error reachable; any other
// error is an attempt that produced no answer. On success the verification
// carries the observed publisher digest and the conventional ok code.
func (v productionTLSVerifier) Verify(ctx context.Context, target string, cfg *tls.Config) (TLSVerification, error) {
	if cfg == nil {
		// The seam's contract is a handshake with the caller's configuration; a
		// nil one cannot carry the declared server name. The refusal is an
		// attempt that produced no answer, never a rejected chain.
		return TLSVerification{}, errors.New("probe: TLS verification requires the caller's configuration, and none was given")
	}
	if cfg.InsecureSkipVerify {
		return TLSVerification{}, errors.New("probe: the caller's TLS configuration disables certificate verification, and this verifier will not perform an unverified handshake")
	}

	conn, err := v.dialer.DialContext(ctx, "tcp", target)
	if err != nil {
		return TLSVerification{}, err
	}
	defer func() { _ = conn.Close() }()

	tlsConn := tls.Client(conn, cfg)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		var rejected *tls.CertificateVerificationError
		if errors.As(err, &rejected) {
			// UnverifiedCertificates is the chain that failed verification, and
			// it is the only place the chain survives a rejected handshake:
			// crypto/tls records PeerCertificates only after verification
			// succeeds. The facts are reported beside the refusal so a reader
			// can see which chain was rejected.
			return TLSVerification{
				Issuer:           chainIssuer(rejected.UnverifiedCertificates),
				VerificationCode: verificationFailureCode(rejected.Err),
			}, &tlsChainRejected{err: err}
		}
		// A handshake error that is not a verification refusal produced no
		// answer at all. Collapsing it into a rejection would report an
		// interception-free network whenever a handshake failed for an unrelated
		// reason.
		return TLSVerification{}, err
	}

	state := tlsConn.ConnectionState()
	// The verified chain is the chain the platform's own verifier accepted, and
	// its last element is the trust anchor. The peer's presented chain is the
	// defensive fallback: with verification on a successful handshake has a
	// verified chain, but reporting an observed chain beats reporting nothing.
	chain := state.PeerCertificates
	if len(state.VerifiedChains) > 0 {
		chain = state.VerifiedChains[0]
	}
	return TLSVerification{
		Issuer:           chainIssuer(chain),
		VerificationCode: verificationOKCode,
	}, nil
}

// tlsChainRejected is the error of a chain the verifier rejected. It wraps the
// exported ErrTLSVerification — so the probe's classification can tell a rejected
// chain (a measurement of the target) from a handshake that produced no answer —
// while keeping crypto/tls's own error reachable with errors.As, so the
// verifier's fact is not lost to the distinction.
type tlsChainRejected struct{ err error }

// Error reports the refusal and the verifier's own error.
func (e *tlsChainRejected) Error() string {
	return "TLS chain did not verify: " + e.err.Error()
}

// Unwrap returns both errors, so errors.Is reaches ErrTLSVerification and
// errors.As reaches the verifier's own refusal.
func (e *tlsChainRejected) Unwrap() []error {
	return []error{ErrTLSVerification, e.err}
}

// chainIssuer renders the observed publisher of a chain as "<issuing CA
// organization>/<topmost certificate short name>", for example
// "Let's Encrypt/ISRG" for the chain PRD §1.1 recorded.
//
// The digest names two facts, never a judgement: the organization that issued the
// leaf, and the name the chain's topmost certificate is commonly known by. The
// seam carries one string while a real chain has one issuer per certificate, so
// the digest is a deliberate fold that keeps the two ends a reader needs to
// recognize a publisher; the classification table is the only place a judgement
// is attached to it, and this file never compares it to anything.
//
// The topmost certificate is the trust anchor on a verified chain and the
// topmost certificate the peer presented on a rejected one. Its short name is
// the first word of its common name — the convention publishers themselves use
// ("ISRG Root X1" is ISRG, "DigiCert Global Root G2" is DigiCert) — and a
// certificate whose organization and common name are both absent contributes
// nothing rather than a fabricated token.
func chainIssuer(chain []*x509.Certificate) string {
	if len(chain) == 0 {
		return ""
	}
	ca := organizationOrCommonName(chain[0].Issuer)
	top := commonNameOrOrganization(chain[len(chain)-1].Subject)
	switch {
	case ca == "":
		return top
	case top == "":
		return ca
	default:
		return ca + "/" + top
	}
}

// organizationOrCommonName reports a distinguished name's organization, or its
// common name when no organization was presented.
func organizationOrCommonName(name pkix.Name) string {
	if len(name.Organization) > 0 && strings.TrimSpace(name.Organization[0]) != "" {
		return strings.TrimSpace(name.Organization[0])
	}
	return strings.TrimSpace(name.CommonName)
}

// commonNameOrOrganization reports a distinguished name's short name: the first
// word of its common name, or its organization when no common name was
// presented. The first word is the publisher's own brand in the root names this
// tool meets ("ISRG Root X1", "DigiCert Global Root G2"), and keeping it makes
// the digest stable across the root's version suffixes.
func commonNameOrOrganization(name pkix.Name) string {
	commonName := strings.TrimSpace(name.CommonName)
	if commonName == "" {
		return organizationOrCommonName(name)
	}
	short, _, _ := strings.Cut(commonName, " ")
	return short
}

// verificationFailureCode names the verifier's own refusal in a short stable
// token. crypto/x509 does not number its refusals the way openssl's verify codes
// are numbered, so a refusal carries the verifier's own reason name and never a
// fabricated number; the token is printed verbatim by the probe and never parsed
// to choose a reason code.
func verificationFailureCode(err error) string {
	var invalid x509.CertificateInvalidError
	if errors.As(err, &invalid) {
		switch invalid.Reason {
		case x509.NotAuthorizedToSign:
			return "not_authorized_to_sign"
		case x509.Expired:
			return "expired"
		case x509.CANotAuthorizedForThisName:
			return "ca_not_authorized_for_name"
		case x509.TooManyIntermediates:
			return "too_many_intermediates"
		case x509.IncompatibleUsage:
			return "incompatible_usage"
		case x509.NameMismatch:
			return "name_mismatch"
		case x509.NameConstraintsWithoutSANs:
			return "name_constraints_without_sans"
		case x509.UnconstrainedName:
			return "unconstrained_name"
		case x509.TooManyConstraints:
			return "too_many_constraints"
		case x509.CANotAuthorizedForExtKeyUsage:
			return "ca_not_authorized_for_ext_key_usage"
		case x509.NoValidChains:
			return "no_valid_chains"
		default:
			return "certificate_invalid"
		}
	}
	var unknownAuthority x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthority) {
		return "unknown_authority"
	}
	var hostname x509.HostnameError
	if errors.As(err, &hostname) {
		return "hostname_mismatch"
	}
	var systemRoots x509.SystemRootsError
	if errors.As(err, &systemRoots) {
		return "system_roots_unavailable"
	}
	return "verification_failed"
}

// productionPacketDialer is the PacketDialer seam over net.Dialer's datagram
// path.
//
// The socket is connected rather than merely opened, and that is a measurement
// requirement, not a convenience: the ICMP port-unreachable the classification
// table records as a definite negative is surfaced by the kernel only on a
// connected UDP socket. A wildcard socket would hide it and turn a measured
// refusal into silence, which the table classifies as unresolved. The local side
// is left to the kernel — an ephemeral port on the requested family — so nothing
// here binds to the far end, and net.Dialer.DialContext is where the probe's
// deadline and cancellation reach the attempt.
type productionPacketDialer struct{ dialer *net.Dialer }

// connectedDatagram is the smallest surface the packet adapter needs from the
// connected socket net.Dialer returned. *net.UDPConn satisfies it, and naming it
// here is what lets the adapter be pinned by a scripted fake that never opens a
// socket.
type connectedDatagram interface {
	// Write sends one datagram to the socket's fixed peer.
	Write(p []byte) (int, error)
	// ReadFrom reads one datagram and its sender address.
	ReadFrom(p []byte) (int, net.Addr, error)
	// SetDeadline bounds the reads and writes that follow.
	SetDeadline(t time.Time) error
	// Close releases the socket.
	Close() error
}

// connectedPacketConn adapts the connected datagram socket to the PacketConn
// seam.
//
// A connected UDP socket fixes its peer at dial time, and the kernel refuses
// WriteTo on it ("use of WriteTo with pre-connected connection"), so the seam's
// WriteTo is served by Write: the address argument is the target the caller
// asked for, and the dial has already fixed that peer, so nothing is
// re-targeted here. ReadFrom, SetDeadline and Close pass through unchanged. The
// socket stays connected because connectedness is what makes the ICMP
// port-unreachable observable to the classification table.
type connectedPacketConn struct{ conn connectedDatagram }

// WriteTo sends one datagram through the connected socket's Write. The address
// argument is the target the caller asked for; the dial fixed that peer, so the
// kernel's WriteTo refusal on a connected socket is never reached.
func (c connectedPacketConn) WriteTo(p []byte, _ net.Addr) (int, error) {
	return c.conn.Write(p)
}

// ReadFrom reads one datagram and its sender address, unchanged.
func (c connectedPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	return c.conn.ReadFrom(p)
}

// SetDeadline bounds the reads and writes that follow, unchanged.
func (c connectedPacketConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// Close releases the socket, unchanged.
func (c connectedPacketConn) Close() error {
	return c.conn.Close()
}

// DialPacket opens the datagram socket under the context's cancellation and
// adapts it to the seam: the socket is connected, and the adapter serves the
// seam's WriteTo through Write.
func (d productionPacketDialer) DialPacket(ctx context.Context, network, addr string) (PacketConn, error) {
	conn, err := d.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	socket, ok := conn.(connectedDatagram)
	if !ok {
		// The system returned a connection that cannot carry the datagram
		// exchange the seam promises; it is closed and reported rather than used
		// as something it is not.
		_ = conn.Close()
		return nil, fmt.Errorf("packet dial %s %s: the system returned %T, which cannot carry datagrams", network, addr, conn)
	}
	return connectedPacketConn{conn: socket}, nil
}

// productionClock is the Clock seam over time.Now. Every timestamp a live run
// reports is a reading of this one clock (design §3.3).
type productionClock struct{}

// Now returns the current wall-clock instant.
func (productionClock) Now() time.Time { return time.Now() }

// productionFS is the FS seam over the process's own read-only inputs: the file
// reads and environment lookups a probe is allowed. The interface has no write
// method and none is implemented here, so the seam cannot change the machine even
// by mistake.
type productionFS struct{}

// ReadFile returns the whole contents of path.
func (productionFS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

// Stat reports path's metadata.
func (productionFS) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }

// Getenv returns the value of the named environment variable, or "".
func (productionFS) Getenv(name string) string { return os.Getenv(name) }

// productionPlatform reports the machine the run is happening on.
//
// GOOS and Arch are runtime's own values, never a mapping. WSL2 and Systemd are
// signals read from the machine's own evidence on Linux, and false everywhere
// else: on another operating system the question does not apply, and false is the
// honest answer to "is this Linux running under WSL2/systemd?", not a guess about
// the machine. A signal that cannot be read — the file is absent, or unreadable —
// is also reported as absent rather than assumed, because a system that cannot be
// identified reports what it observed (R-HR-29).
type productionPlatform struct{}

// GOOS reports the operating system in Go's own vocabulary.
func (productionPlatform) GOOS() string { return runtime.GOOS }

// Arch reports the machine architecture.
func (productionPlatform) Arch() string { return runtime.GOARCH }

// WSL2 reports whether the kernel's own version string identifies a WSL2 kernel.
//
// WSL2 kernels carry "microsoft-standard" in their release string ("5.15.90.1-
// microsoft-standard-WSL2", "4.19.128-microsoft-standard"), while WSL1 kernels
// carry "Microsoft" without the "-standard" marker. The detection reads the
// machine's answer and matches the WSL2 marker case-insensitively; anything else
// is no WSL2 signal.
func (productionPlatform) WSL2() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	release, err := os.ReadFile(kernelReleasePath)
	if err != nil {
		return false
	}
	version := strings.ToLower(strings.TrimSpace(string(release)))
	return strings.Contains(version, "microsoft-standard") || strings.Contains(version, "wsl2")
}

// Systemd reports whether systemd is the running service manager, detected by
// the service-manager runtime directory systemd creates when it is. The signal
// is a directory that exists or does not; an unreadable or absent path is no
// signal, and a path that is not a directory is not the runtime directory.
func (productionPlatform) Systemd() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	info, err := os.Stat(systemdRuntimeDir)
	return err == nil && info.IsDir()
}
