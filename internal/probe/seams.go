package probe

// This file is the seam set of design §6.1: the eight interfaces through which
// every measurement reaches the outside world, and the deny-all default of
// design §6.2 that every probe unit test starts from.
//
// The seams are injected, never global. The alternative — a package-level
// variable such as `var dial = net.Dial` that a test swaps — cannot run under
// t.Parallel(), hides which capability a measurement actually needed, and lets a
// sub-test that forgets to swap reach the real network. Instead the whole set
// travels as one explicit Seams value: the zero value of Seams carries nothing,
// DenyAllSeams() carries every capability switched off, and the production set is
// built in exactly one place (internal/probe/real.go, called from main.go).
//
// Nothing here dials, resolves, reads or executes. This file declares the
// contracts and the denying implementations; the measuring happens in the probes.

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"sync"
	"time"
)

// ErrSeamDenied is the sentinel every capability of DenyAllSeams() returns. It is
// exported because the seam set is used by tests in other packages (the doctor
// and command-surface suites), which must be able to assert that a seam denied
// rather than answered.
var ErrSeamDenied = errors.New("probe: denied by the deny-all test seams")

// Dialer opens TCP connections. It is the seam every reachability probe and both
// TLS probes dial through, so a test can script a connection without a socket.
type Dialer interface {
	// DialContext opens a connection to addr. It must honour ctx's cancellation
	// and deadline, because the probe's own budget is expressed through them.
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// Resolver turns a declared host name into addresses. It is separate from Dialer
// so name resolution can be scripted independently: "the name does not exist"
// and "the port did not answer" are different measurements (design §5.1).
type Resolver interface {
	// LookupHost returns the addresses host resolves to, in the resolver's own
	// order, or an error when the name did not resolve.
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// TLSVerification is what a verifier observed on the wire, before the
// classification table attaches any judgement to it.
//
// It deliberately carries no verdict and no reason code: the table in classify.go
// is the only place that chooses a code, so the verifier reports facts and the
// probe classifies them (design §5.1, R-HR-07).
type TLSVerification struct {
	// Issuer is the chain's issuer as the peer presented it, for example
	// "Let's Encrypt/ISRG". It is verbatim and stays verbatim in the output; it
	// is empty when no chain was fetched at all.
	Issuer string
	// VerificationCode is the verifier's own code for the verification attempt,
	// for example "0". It is carried verbatim into the observation's detail and
	// is never parsed to choose a reason code.
	VerificationCode string
}

// ErrTLSVerification is what a TLSVerifier wraps when the chain did not verify.
// The distinction is load-bearing: a rejected chain is a measurement of the
// target (fail), while a handshake that produced no answer at all is an
// unresolved attempt, and collapsing the two would report an interception-free
// network whenever a handshake failed for an unrelated reason.
var ErrTLSVerification = errors.New("probe: TLS chain did not verify")

// TLSVerifier performs a TLS handshake with caller-supplied configuration and
// reports what the chain looked like. It exists so the chain, the root pool and
// the verification code are injectable: the probes must never be the place where
// a certificate is trusted.
type TLSVerifier interface {
	// Verify handshakes with target using cfg and reports the observed chain.
	//
	// A chain the verifier rejects returns an error wrapping ErrTLSVerification.
	// Any other error means the attempt produced no answer. A nil error means the
	// chain verified and verification.Issuer is the issuer to compare against the
	// declared expected set. Verify must never disable verification, retry
	// unverified, or modify cfg (R-HR-04).
	Verify(ctx context.Context, target string, cfg *tls.Config) (TLSVerification, error)
}

// PacketConn is the datagram side of the seam set: the four operations D8's
// narrow UDP question needs, and nothing else, so a scripted connection is small
// enough to read in one screen.
type PacketConn interface {
	// WriteTo sends one datagram to addr.
	WriteTo(p []byte, addr net.Addr) (int, error)
	// ReadFrom reads one datagram, returning the sender's address. It honours the
	// deadline set by SetDeadline, which is how "silence" is observed rather than
	// waited for forever.
	ReadFrom(p []byte) (int, net.Addr, error)
	// SetDeadline bounds the reads and writes that follow.
	SetDeadline(t time.Time) error
	// Close releases the socket.
	Close() error
}

// PacketDialer opens packet sockets. It is a separate seam from Dialer because a
// UDP probe's honest outcomes are different: silence is an unresolved attempt,
// never a blocked port (design D8, RG-5).
type PacketDialer interface {
	// DialPacket opens a packet socket to addr, honouring ctx's cancellation.
	DialPacket(ctx context.Context, network, addr string) (PacketConn, error)
}

// CommandRunner runs one external command and captures its streams.
//
// This is the only seam that can execute a third-party binary, which is why R1a
// leaves its production implementation nil: no code path in this slice can run
// `sshd`, `cloudflared`, `herdr` or anything else, and a runner that is absent
// reads as an excluded capability rather than as a failure (design §6.2,
// R-HR-02). Probes that need it must tolerate nil.
type CommandRunner interface {
	// Run executes name with args and returns its captured stdout and stderr.
	Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error)
}

// Clock is the single clock a run reads. Every timestamp and every elapsed value
// in the output comes from it, so a scripted clock makes a whole run
// deterministic (design §3.3, "only the injected clock moves the timestamps").
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// FS is the read-only local input a probe is allowed. Reads and environment
// lookups go through it so "reads nothing outside the seams" is checkable by
// passing a seam that reads nothing.
type FS interface {
	// ReadFile returns the whole contents of path.
	ReadFile(path string) ([]byte, error)
	// Stat reports path's metadata, or an error when it does not exist.
	Stat(path string) (os.FileInfo, error)
	// Getenv returns the value of the named environment variable, or "".
	Getenv(name string) string
}

// Platform reports what kind of machine the run is happening on.
type Platform interface {
	// GOOS is the operating system name, in Go's own vocabulary ("linux",
	// "darwin", "windows"). A system that cannot be identified reports
	// "unknown", which is a value the classification recognises rather than a
	// default guess (R-HR-29).
	GOOS() string
	// Arch is the machine architecture, for example "amd64" or "arm64".
	Arch() string
	// WSL2 reports whether this Linux system is a WSL2 instance, detected from
	// the kernel's own version string.
	WSL2() bool
	// Systemd reports whether systemd is the running service manager. It is a
	// signal, not a requirement: WSL2 without systemd is a documented state this
	// slice detects and does not repair (R-HR-30).
	Systemd() bool
}

// Seams is the injectable capability set of one run. Every field is a capability
// the run was given; a nil field is a capability the run was not given, and
// nothing here supplies a default, because a silent default would be a
// measurement performed by accident.
//
// The zero Seams value is therefore a seam set with no capabilities at all, and
// DenyAllSeams() is the test default: every capability present, every capability
// refusing.
type Seams struct {
	Dialer        Dialer
	Resolver      Resolver
	TLSVerifier   TLSVerifier
	PacketDialer  PacketDialer
	CommandRunner CommandRunner
	Clock         Clock
	FS            FS
	Platform      Platform
}

// CommandFact returns the raw fact that explains why one command-seam invocation
// did not run, or the zero RawObservation when the command ran and the caller
// must judge its output itself.
//
// A Seams value whose CommandRunner is nil was never given the capability, so the
// fact is ObsCapabilityExcluded whatever the error argument says. An injected
// runner that returned an error denied the command, so the fact is
// ObsCommandDenied. Those are two different observables with two different reason
// codes (design §5.1 obligation 2), and neither classifies as a pass: a run that
// could not execute the command must never be read as a healthy one.
//
// command is quoted verbatim into the detail so the output names the capability
// that was missing instead of leaving the reader to guess it.
func (s Seams) CommandFact(command string, err error) RawObservation {
	if s.CommandRunner == nil {
		return RawObservation{
			Kind:    ObsCapabilityExcluded,
			Wording: command + ": no command runner is injected for this run",
		}
	}
	if err != nil {
		return RawObservation{
			Kind:    ObsCommandDenied,
			Wording: command + ": " + err.Error(),
		}
	}
	return RawObservation{}
}

// DenyAllSeams returns the test default of design §6.2: a complete seam set in
// which every capability refuses.
//
// Every dial, lookup, TLS verification and packet operation fails with
// ErrSeamDenied; the command runner denies everything; FS reports fs.ErrNotExist
// for every path and an empty environment; Platform reports "unknown"; and Clock
// is a scripted stepper starting at 2000-01-01T00:00:00Z and advancing 1ms per
// reading, so an elapsed value computed from it is reproducible.
//
// A fresh value is returned on every call, so a test may override one field
// without any other test observing the change.
func DenyAllSeams() Seams {
	return Seams{
		Dialer:        denyAllDialer{},
		Resolver:      denyAllResolver{},
		TLSVerifier:   denyAllTLSVerifier{},
		PacketDialer:  denyAllPacketDialer{},
		CommandRunner: denyAllCommandRunner{},
		Clock:         NewStepperClock(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC), denyAllClockStep),
		FS:            denyAllFS{},
		Platform:      denyAllPlatform{},
	}
}

// denyAllClockStep is the deny-all clock's increment: one millisecond per
// reading, small enough that a scripted run reads as immediate and exact enough
// that golden elapsed values are stable.
const denyAllClockStep = time.Millisecond

// NewStepperClock returns a Clock that starts at origin and advances by step on
// every reading. It is the scripted clock of design §6.2 and the clock any test
// needing deterministic timestamps injects.
//
// The stepper holds its position behind a mutex because the runner starts probes
// concurrently and they share one clock; without it the race detector would flag
// a scripted clock that four in-flight probes read at once. Callers pass a
// positive step: a zero step is legal but does not advance, which is not the
// scripted stepper the deny-all set uses.
func NewStepperClock(origin time.Time, step time.Duration) Clock {
	return &stepperClock{next: origin, step: step}
}

// stepperClock is the scripted clock behind NewStepperClock and DenyAllSeams.
type stepperClock struct {
	mu   sync.Mutex
	next time.Time
	step time.Duration
}

// Now returns the scripted instant and advances the script. Each reading is a
// distinct instant, so an elapsed measurement is never zero and never depends on
// how fast the machine is.
func (c *stepperClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.next
	c.next = c.next.Add(c.step)
	return now
}

// denyAllDialer refuses every connection.
type denyAllDialer struct{}

// DialContext denies the connection and returns no conn, so a caller cannot use a
// half-returned value beside the error.
func (denyAllDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return nil, ErrSeamDenied
}

// denyAllResolver refuses every lookup: no name resolves in a seam set that was
// never given addresses.
type denyAllResolver struct{}

// LookupHost denies the lookup and returns no addresses.
func (denyAllResolver) LookupHost(context.Context, string) ([]string, error) {
	return nil, ErrSeamDenied
}

// denyAllTLSVerifier refuses every handshake. Its error is ErrSeamDenied and not
// ErrTLSVerification, so a denied seam can never be misread as a rejected chain.
type denyAllTLSVerifier struct{}

// Verify denies the handshake and returns the zero verification.
func (denyAllTLSVerifier) Verify(context.Context, string, *tls.Config) (TLSVerification, error) {
	return TLSVerification{}, ErrSeamDenied
}

// denyAllPacketDialer refuses every packet socket.
type denyAllPacketDialer struct{}

// DialPacket denies the socket and returns no connection.
func (denyAllPacketDialer) DialPacket(context.Context, string, string) (PacketConn, error) {
	return nil, ErrSeamDenied
}

// denyAllCommandRunner denies every command.
type denyAllCommandRunner struct{}

// Run denies the command and captures nothing.
func (denyAllCommandRunner) Run(context.Context, string, ...string) ([]byte, []byte, error) {
	return nil, nil, ErrSeamDenied
}

// denyAllFS reports that nothing exists and that the environment is empty.
// fs.ErrNotExist is the honest answer for a seam set with no filesystem behind
// it: "this path is not there", not "this path exists and could not be read".
type denyAllFS struct{}

// ReadFile reports the path as not existing.
func (denyAllFS) ReadFile(path string) ([]byte, error) {
	return nil, fmt.Errorf("%s: %w", path, fs.ErrNotExist)
}

// Stat reports the path as not existing and returns no metadata.
func (denyAllFS) Stat(path string) (os.FileInfo, error) {
	return nil, fmt.Errorf("%s: %w", path, fs.ErrNotExist)
}

// Getenv reports an empty environment.
func (denyAllFS) Getenv(string) string {
	return ""
}

// denyAllPlatform reports a machine that could not be classified: the literal
// "unknown" for both the system and the architecture, and no positive signal for
// WSL2 or systemd. The classification turns "unknown" into an explicit unresolved
// outcome rather than into a default guess (R-HR-29).
type denyAllPlatform struct{}

// GOOS reports the unknown platform.
func (denyAllPlatform) GOOS() string { return "unknown" }

// Arch reports the unknown architecture.
func (denyAllPlatform) Arch() string { return "unknown" }

// WSL2 reports no WSL2 signal: a platform that was never given signals must not
// claim one.
func (denyAllPlatform) WSL2() bool { return false }

// Systemd reports no service-manager signal, for the same reason.
func (denyAllPlatform) Systemd() bool { return false }
