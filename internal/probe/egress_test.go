package probe_test

// This file is the egress measurement suite of PRD §5.1: the `egress.hub.direct`
// probe (PR 6) and, in the PR 7 block at the end, the two public-SSH destination
// probes (`egress.ssh.known`, `egress.ssh.443`) and the two per-region Cloudflare
// edge probes (`egress.cf.7844`, `egress.cf.443`).
//
// The hub probe's declared target is run input, not a constant: with `--hub` the
// target is exactly the supplied `host[:port]`, and with no hub there is no target
// and the observation says so. Every case therefore states two things — the run
// input and the seam the dial would travel through — and the dialer records both
// the address it was asked for and the deadline the probe set on the dial. The
// deadline is recorded because "the probe's own budget expired" is only true if
// the probe set one; a refusal needs no deadline at all.
//
// The four probes the PR 7 block covers differ from the hub in one respect that
// shapes their cases: their targets are declared constants (targets.go), and the
// two SSH probes additionally read the identification string the far end sends,
// which is what separates "SSH is reachable" from "some port answered".

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

const (
	// testHubProbeName is the probe's stable identifier, as the registry declares
	// it.
	testHubProbeName = "egress.hub.direct"
	// testHubAddress is the hub address the cases supply: a documented-example
	// address with an explicit port, so "the target is exactly what was supplied"
	// is a statement about a value a reader can compare.
	testHubAddress = "203.0.113.10:2222"
)

// errNoScriptedPayload is what the scripted connection reports for any read. The
// hub probe measures whether a TCP connection completes; it never reads or writes
// a payload, and a connection that carried one would not make the case stronger.
var errNoScriptedPayload = errors.New("the scripted connection carries no payload")

// scriptedAddr is the net.Addr of the scripted connection.
type scriptedAddr string

// Network reports the scripted network.
func (scriptedAddr) Network() string { return "scripted" }

// String reports the scripted address.
func (a scriptedAddr) String() string { return string(a) }

// scriptedConn is the net.Conn a successful scripted dial returns. It records that
// it was closed, because a probe that establishes a connection owns closing it.
type scriptedConn struct {
	closed bool
}

// Read reports that the scripted connection carries no payload.
func (c *scriptedConn) Read([]byte) (int, error) { return 0, errNoScriptedPayload }

// Write accepts the bytes and reports them written.
func (c *scriptedConn) Write(p []byte) (int, error) { return len(p), nil }

// Close records the close.
func (c *scriptedConn) Close() error {
	c.closed = true
	return nil
}

// LocalAddr reports the scripted local address.
func (c *scriptedConn) LocalAddr() net.Addr { return scriptedAddr("local") }

// RemoteAddr reports the scripted remote address.
func (c *scriptedConn) RemoteAddr() net.Addr { return scriptedAddr("remote") }

// SetDeadline accepts a deadline without acting on it.
func (c *scriptedConn) SetDeadline(time.Time) error { return nil }

// SetReadDeadline accepts a read deadline without acting on it.
func (c *scriptedConn) SetReadDeadline(time.Time) error { return nil }

// SetWriteDeadline accepts a write deadline without acting on it.
func (c *scriptedConn) SetWriteDeadline(time.Time) error { return nil }

// scriptedDialer is the Dialer seam an egress case injects. It records every
// address it was asked to dial and the deadline the probe put on the dial context,
// and it answers with the scripted connection or the scripted error.
type scriptedDialer struct {
	// conn is returned when err is nil. It may be nil: the probe must tolerate a
	// dialer that reports success without a connection rather than panicking.
	conn net.Conn
	// err is what the dial returns when it is not nil.
	err error
	// calls records the addresses dialed, as "network address".
	calls []string
	// hasDeadline records, per call, whether the probe bounded the dial.
	hasDeadline []bool
	// deadlines records, per call, how much of the probe's budget remained.
	deadlines []time.Duration
}

// DialContext records the attempt and returns the scripted answer.
func (d *scriptedDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d.calls = append(d.calls, network+" "+addr)
	if deadline, ok := ctx.Deadline(); ok {
		d.hasDeadline = append(d.hasDeadline, true)
		d.deadlines = append(d.deadlines, time.Until(deadline))
	} else {
		d.hasDeadline = append(d.hasDeadline, false)
		d.deadlines = append(d.deadlines, 0)
	}
	if d.err != nil {
		return nil, d.err
	}
	return d.conn, nil
}

// hubSeams builds the seam set one case describes: the deny-all set of design §6.2
// with the dialer replaced. A nil dialer is the zero capability — the run was
// never given one — which is a different fact from a dialer that refuses.
func hubSeams(dialer probe.Dialer) probe.Seams {
	seams := probe.DenyAllSeams()
	seams.Dialer = dialer
	return seams
}

// hubBuild builds `egress.hub.direct` the way a run reaches it: through the
// registry, with the run's seams and its declared target input.
func hubBuild(t *testing.T, seams probe.Seams, targets probe.TargetInput) probe.Probe {
	t.Helper()
	for _, built := range probe.ProbesFor(seams, targets) {
		if built.Name() == testHubProbeName {
			return built
		}
	}
	t.Fatalf("the registry built no %q probe, so no run could measure the hub", testHubProbeName)
	return nil
}

// runHub runs `egress.hub.direct` over one scripted run.
func runHub(t *testing.T, dialer probe.Dialer, targets probe.TargetInput) probe.Result {
	t.Helper()
	return hubBuild(t, hubSeams(dialer), targets).Run(context.Background())
}

// resultText is every string a result carries, so a wording guard reads the whole
// report and not only the field a case happened to think of.
func resultText(result probe.Result) string {
	parts := []string{result.Probe, result.Target, result.Detail}
	for _, observation := range result.Observations {
		parts = append(parts, observation.Label, observation.Target, observation.Detail)
	}
	return strings.Join(parts, "\n")
}

// TestEgressHubTargetIsExactlyTheSuppliedAddress is RG-13's first two scenarios:
// the supplied hub address becomes the measured target, and an address without a
// port resolves by the one documented rule. The target is asserted on the result
// and on the observation, and the dialer's own record proves the resolved address
// is the one that was dialed.
func TestEgressHubTargetIsExactlyTheSuppliedAddress(t *testing.T) {
	cases := []struct {
		name       string
		hub        string
		wantTarget string
		wantPort   int
	}{
		{
			name:       "a hub address including a port",
			hub:        testHubAddress,
			wantTarget: testHubAddress,
			wantPort:   2222,
		},
		{
			name:       "a hub address that omits the port",
			hub:        "hub.example.com",
			wantTarget: "hub.example.com:22",
			wantPort:   22,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn := &scriptedConn{}
			dialer := &scriptedDialer{conn: conn}
			result := runHub(t, dialer, probe.TargetInput{Hub: tc.hub})

			if result.Probe != testHubProbeName {
				t.Fatalf("result names the probe %q, want %q", result.Probe, testHubProbeName)
			}
			if result.Kind != probe.ProbeEgress {
				t.Fatalf("result kind = %q, want %q", result.Kind, probe.ProbeEgress)
			}
			if result.Target != tc.wantTarget {
				t.Errorf("result target = %q, want the supplied address %q", result.Target, tc.wantTarget)
			}
			if len(result.Observations) != 1 {
				t.Fatalf("the hub probe reported %d observations, want 1 TCP question", len(result.Observations))
			}
			observation := result.Observations[0]
			if !holds(observation) {
				t.Fatalf("observation does not satisfy the measurement vocabulary's invariant: %+v", observation)
			}
			if observation.Target != tc.wantTarget {
				t.Errorf("observation target = %q, want %q", observation.Target, tc.wantTarget)
			}
			wantLabel := fmt.Sprintf("tcp %d", tc.wantPort)
			if observation.Label != wantLabel {
				t.Errorf("observation label = %q, want %q", observation.Label, wantLabel)
			}
			if observation.Resolution != probe.Measured || observation.Verdict != probe.Pass || observation.Reason != probe.ReasonOK {
				t.Errorf("observation = (%q, %q, %q), want (%q, %q, %q)", observation.Resolution, observation.Verdict, observation.Reason, probe.Measured, probe.Pass, probe.ReasonOK)
			}
			if result.Verdict != probe.Pass || result.Reason != probe.ReasonOK {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Pass, probe.ReasonOK)
			}
			verdict, reason := probe.Aggregate(result.Observations)
			if result.Verdict != verdict || result.Reason != reason {
				t.Fatalf("the result's reduction %s/%s disagrees with Aggregate over its observations %s/%s", result.Verdict, result.Reason, verdict, reason)
			}

			wantCalls := []string{"tcp " + tc.wantTarget}
			if !reflect.DeepEqual(dialer.calls, wantCalls) {
				t.Errorf("the dialed addresses = %v, want exactly %v", dialer.calls, wantCalls)
			}
			if !conn.closed {
				t.Error("the probe did not close the connection it established")
			}
		})
	}
}

// TestEgressHubWithoutHubInputIsNotMeasuredAsBlocked is the diagnosis domain's
// "no hub supplied is not a blocked hub" scenario (R-HR-02) and RG-13's third
// scenario: the measurement was not made, no target was attempted, and no text
// states or implies a block.
func TestEgressHubWithoutHubInputIsNotMeasuredAsBlocked(t *testing.T) {
	dialer := &scriptedDialer{conn: &scriptedConn{}}
	result := runHub(t, dialer, probe.TargetInput{})

	if len(dialer.calls) != 0 {
		t.Fatalf("the probe dialed %v although no hub address was supplied, so an attempt was made", dialer.calls)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("the hub probe reported %d observations, want 1", len(result.Observations))
	}
	observation := result.Observations[0]
	if observation.Resolution != probe.NotMeasured {
		t.Errorf("resolution = %q, want %q: no attempt was made", observation.Resolution, probe.NotMeasured)
	}
	if observation.Verdict != probe.Indeterminate || result.Verdict != probe.Indeterminate {
		t.Errorf("verdict = (%q, %q), want %q: a measurement that was not made is not an answer", observation.Verdict, result.Verdict, probe.Indeterminate)
	}
	if observation.Reason != probe.ReasonInputMissingHub || result.Reason != probe.ReasonInputMissingHub {
		t.Errorf("reason = (%q, %q), want %q: the reason names the missing input", observation.Reason, result.Reason, probe.ReasonInputMissingHub)
	}
	if observation.Target != "" || result.Target != "" {
		t.Errorf("targets = (%q, %q), want empty: no target was attempted", observation.Target, result.Target)
	}
	if strings.Contains(strings.ToLower(resultText(result)), "block") {
		t.Errorf("the not-measured result states or implies a block: %q", resultText(result))
	}
	if !strings.Contains(observation.Detail, "not made") {
		t.Errorf("the not-measured detail does not state that the measurement was not made: %q", observation.Detail)
	}
	if result.Verdict == probe.Pass {
		t.Error("a hub that was never measured was reported as a pass")
	}
}

// TestEgressHubRefusedDialIsAMeasuredFailure is the diagnosis domain's "a refused
// hub dial is a measurement, not a skip" scenario: the attempt was made, the far
// end answered, and the answer is a definite negative that a consumer can tell
// apart from the not-measured outcome.
func TestEgressHubRefusedDialIsAMeasuredFailure(t *testing.T) {
	refused := fmt.Errorf("dial tcp %s: connect: %w", testHubAddress, syscall.ECONNREFUSED)
	result := runHub(t, &scriptedDialer{err: refused}, probe.TargetInput{Hub: testHubAddress})

	if len(result.Observations) != 1 {
		t.Fatalf("the hub probe reported %d observations, want 1", len(result.Observations))
	}
	observation := result.Observations[0]
	if observation.Resolution != probe.Measured {
		t.Errorf("resolution = %q, want %q: the dial was attempted and answered", observation.Resolution, probe.Measured)
	}
	if observation.Verdict != probe.Fail || observation.Reason != probe.ReasonConnRefused {
		t.Errorf("observation = (%q, %q), want (%q, %q)", observation.Verdict, observation.Reason, probe.Fail, probe.ReasonConnRefused)
	}
	if result.Verdict != probe.Fail || result.Reason != probe.ReasonConnRefused {
		t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Fail, probe.ReasonConnRefused)
	}
	if !strings.Contains(observation.Detail, refused.Error()) {
		t.Errorf("the refused detail does not carry the dialer's own error verbatim: %q", observation.Detail)
	}
	if observation.Target != testHubAddress {
		t.Errorf("observation target = %q, want %q: the refusal happened at the declared address", observation.Target, testHubAddress)
	}
	if result.Verdict == probe.Pass {
		t.Error("a refused dial was reported as a pass")
	}
}

// TestEgressHubBlackholedPortIsTheProbesOwnBudget is the third timeout path
// (design §5.1 obligation 3, RG-8): a declared port that never answers inside the
// probe's own dial budget is a *measured* failure carrying `budget_expired`, and
// the difference from the runner's per-probe bound is what keeps it distinct from
// an unresolved hanging probe.
//
// The case scripts the dial error a real socket returns when its deadline expires,
// which keeps the test fast, and asserts the deadline the probe set: that deadline
// is what makes the expiry the probe's own rather than the runner's.
func TestEgressHubBlackholedPortIsTheProbesOwnBudget(t *testing.T) {
	timeout := fmt.Errorf("dial tcp %s: i/o timeout: %w", testHubAddress, os.ErrDeadlineExceeded)
	dialer := &scriptedDialer{err: timeout}
	result := runHub(t, dialer, probe.TargetInput{Hub: testHubAddress})

	observation := result.Observations[0]
	if observation.Resolution != probe.Measured {
		t.Errorf("resolution = %q, want %q: this fact is the measurement for a reachability question", observation.Resolution, probe.Measured)
	}
	if observation.Verdict != probe.Fail || observation.Reason != probe.ReasonBudgetExpired {
		t.Errorf("observation = (%q, %q), want (%q, %q)", observation.Verdict, observation.Reason, probe.Fail, probe.ReasonBudgetExpired)
	}
	if result.Reason != probe.ReasonBudgetExpired {
		t.Errorf("result reason = %q, want %q", result.Reason, probe.ReasonBudgetExpired)
	}
	if result.Reason == probe.ReasonProbeTimeout {
		t.Fatal("a probe-owned dial expiry was reported with the runner's probe_timeout reason")
	}
	if !strings.Contains(observation.Detail, probe.DefaultDialBudget.String()) {
		t.Errorf("the detail does not name the probe's own %s dial budget: %q", probe.DefaultDialBudget, observation.Detail)
	}

	if len(dialer.hasDeadline) != 1 || !dialer.hasDeadline[0] {
		t.Fatalf("the probe dialed without a deadline, so the expiry cannot be its own: %v", dialer.hasDeadline)
	}
	if dialer.deadlines[0] <= 0 || dialer.deadlines[0] > probe.DefaultDialBudget {
		t.Errorf("the dial deadline was %s, want a positive value inside the probe's %s budget", dialer.deadlines[0], probe.DefaultDialBudget)
	}
	if probe.DefaultDialBudget >= probe.DefaultProbeTimeout {
		t.Errorf("the probe's dial budget %s is not shorter than the runner's per-probe bound %s, so a dial expiry and a hanging probe could collapse", probe.DefaultDialBudget, probe.DefaultProbeTimeout)
	}
}

// TestEgressHubDeniedOrMissingDialCapabilityIsNotMeasured is design §5.1
// obligation 2 for this probe: a capability the run was never given, and a
// capability that refused, are two different not-measured facts, and neither may
// be reported as a measurement or as a pass.
func TestEgressHubDeniedOrMissingDialCapabilityIsNotMeasured(t *testing.T) {
	cases := []struct {
		name       string
		dialer     probe.Dialer
		wantReason probe.ReasonCode
	}{
		{"the dial seam denies the dial", probe.DenyAllSeams().Dialer, probe.ReasonCommandDenied},
		{"no dialer is injected for the run", nil, probe.ReasonCapabilityExcluded},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runHub(t, tc.dialer, probe.TargetInput{Hub: testHubAddress})

			if len(result.Observations) != 1 {
				t.Fatalf("the hub probe reported %d observations, want 1", len(result.Observations))
			}
			observation := result.Observations[0]
			if observation.Resolution != probe.NotMeasured || observation.Verdict != probe.Indeterminate {
				t.Errorf("observation = (%q, %q), want (%q, %q): the attempt was not made", observation.Resolution, observation.Verdict, probe.NotMeasured, probe.Indeterminate)
			}
			if observation.Reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", observation.Reason, tc.wantReason)
			}
			if result.Reason != tc.wantReason {
				t.Errorf("result reason = %q, want %q", result.Reason, tc.wantReason)
			}
			if result.Verdict == probe.Pass {
				t.Fatal("a hub that was not measured was reported as a pass")
			}
			if strings.TrimSpace(observation.Detail) == "" {
				t.Error("the not-measured observation carries no verbatim detail")
			}
			if observation.Target != testHubAddress {
				t.Errorf("observation target = %q, want %q: the address the attempt was aimed at", observation.Target, testHubAddress)
			}
		})
	}

	denied := runHub(t, probe.DenyAllSeams().Dialer, probe.TargetInput{Hub: testHubAddress})
	missing := runHub(t, nil, probe.TargetInput{Hub: testHubAddress})
	if denied.Reason == missing.Reason {
		t.Fatalf("a denying seam and a missing capability report the same reason %q; design §5.1 obligation 2 requires two", denied.Reason)
	}
}

// TestEgressHubEstablishesNoTransportViability asserts the R-HR-02 and RG-13
// boundary this probe must not cross: it measures one address, and a reachable hub
// establishes nothing about a transport. The structural half of the claim is that
// transport viability lives in `internal/transport` and this package cannot
// express it; the wording half is asserted here, over every outcome the probe can
// produce, because a claim in the text is what a user would read.
func TestEgressHubEstablishesNoTransportViability(t *testing.T) {
	refused := fmt.Errorf("dial tcp %s: connect: %w", testHubAddress, syscall.ECONNREFUSED)
	blackholed := fmt.Errorf("dial tcp %s: i/o timeout: %w", testHubAddress, os.ErrDeadlineExceeded)

	cases := []struct {
		name    string
		dialer  probe.Dialer
		targets probe.TargetInput
	}{
		{"a hub that accepted the connection", &scriptedDialer{conn: &scriptedConn{}}, probe.TargetInput{Hub: testHubAddress}},
		{"a hub that refused the connection", &scriptedDialer{err: refused}, probe.TargetInput{Hub: testHubAddress}},
		{"a hub that never answered", &scriptedDialer{err: blackholed}, probe.TargetInput{Hub: testHubAddress}},
		{"a dial seam that denied the attempt", probe.DenyAllSeams().Dialer, probe.TargetInput{Hub: testHubAddress}},
		{"no hub address at all", &scriptedDialer{conn: &scriptedConn{}}, probe.TargetInput{}},
	}

	forbidden := []string{"viable", "viability", "transport", "recommend", "block"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runHub(t, tc.dialer, tc.targets)

			if len(result.Observations) != 1 {
				t.Fatalf("the hub probe reported %d observations, want 1", len(result.Observations))
			}
			text := strings.ToLower(resultText(result))
			for _, token := range forbidden {
				if strings.Contains(text, token) {
					t.Errorf("the hub probe's output contains %q, which is a claim about a transport rather than about this address: %q", token, resultText(result))
				}
			}
			if result.Verdict == probe.Pass && tc.targets.Hub == "" {
				t.Error("a hub that was never supplied was reported as a pass")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// PR 7 — the egress reachability probes: the two public-SSH destination probes
// (`egress.ssh.known`, `egress.ssh.443`) and the two per-region Cloudflare edge
// probes (`egress.cf.7844`, `egress.cf.443`).
//
// The probes share their shape with the hub probe above — resolve the declared
// name through the injected resolver, dial the declared address through the
// injected dialer, classify through the one table — and differ in what they read:
// the SSH probes read the far end's identification string, and the Cloudflare
// probes measure both declared regions and report one observation per region, so a
// split result survives into the output instead of being hidden behind a single
// aggregate (design D10).
// -----------------------------------------------------------------------------

const (
	// testSSHKnownProbe and testSSH443Probe are the two public-SSH probes, as the
	// registry declares them.
	testSSHKnownProbe = "egress.ssh.known"
	testSSH443Probe   = "egress.ssh.443"
	// testCF7844Probe and testCF443Probe are the two Cloudflare edge probes.
	testCF7844Probe = "egress.cf.7844"
	testCF443Probe  = "egress.cf.443"
	// testSSHBanner is a well-formed SSH identification string (RFC 4253 §4.2): the
	// positive signal the SSH probes exist to read.
	testSSHBanner = "SSH-2.0-OpenSSH_9.6\r\n"
	// testNonSSHBanner is what an intercepting proxy or a port some other service
	// took over answers with.
	testNonSSHBanner = "HTTP/1.1 403 Forbidden\r\n"
	// testRegion1Host and testRegion2Host are the two declared Cloudflare edge
	// regions, repeated here as literals from targets.go and PRD §1.1 so a rename in
	// the declaration breaks this suite instead of silently moving what is measured.
	testRegion1Host = "region1.v2.argotunnel.com"
	testRegion2Host = "region2.v2.argotunnel.com"
)

// sshProbes transcribes the two SSH destination probes and the declared target each
// measures: SSH to a well-known public host on :22 (the protocol-versus-port
// disambiguator of PRD §1.1) and the same question on the allowed port :443.
var sshProbes = []struct {
	name string
	host string
	port int
}{
	{testSSHKnownProbe, "github.com", 22},
	{testSSH443Probe, "ssh.github.com", 443},
}

// sshBannerLabel is the observation label both SSH probes report: the port, the
// protocol question and the fact that the identification string is what answers it.
func sshBannerLabel(port int) string { return fmt.Sprintf("tcp %d ssh banner", port) }

// cfProbes transcribes the two Cloudflare edge probes and the port each declares. Both
// regions are declared for each of them (design D10), which is why the declared target
// counts are asserted in the cases rather than assumed here.
var cfProbes = []struct {
	name string
	port int
}{
	{testCF7844Probe, 7844},
	{testCF443Probe, 443},
}

// scriptedResolver is the Resolver seam a reachability case injects: it records
// every host it was asked about and answers per host first, then with the case's own
// error, then with the case's own addresses. The per-host script is how a case tells
// one declared region's name apart from the other's.
type scriptedResolver struct {
	// addresses is what a lookup answers with when nothing else applies.
	addresses []string
	// err is what a lookup returns when nothing else applies.
	err error
	// perHost overrides the answer for specific hosts.
	perHost map[string]error
	// lookups records the hosts that were asked about, in order.
	lookups []string
}

// LookupHost records the lookup and answers it from the case's script.
func (r *scriptedResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	r.lookups = append(r.lookups, host)
	if err, ok := r.perHost[host]; ok {
		return nil, err
	}
	if r.err != nil {
		return nil, r.err
	}
	return r.addresses, nil
}

// scriptedBannerConn is the net.Conn a scripted successful dial returns to the SSH
// probes. It serves its payload in the chunks the case gives it — which is how a
// partial read is expressed — then reports the error that ended the read (end of
// stream unless the case scripts another one), records the read deadline the probe
// set, and records that the probe closed it.
type scriptedBannerConn struct {
	// chunks are the payload pieces, each delivered by one Read.
	chunks [][]byte
	// readErr ends the read once the chunks are exhausted. io.EOF is the default.
	readErr error
	// position is the next chunk to serve.
	position int
	// deadline records the instant the probe asked reads to stop by.
	deadline time.Time
	// hasDeadline records that the probe set one at all.
	hasDeadline bool
	// closed records the probe's close.
	closed bool
}

// newBannerConn returns a scripted connection that serves one banner in the single
// chunk a well-behaved server sends.
func newBannerConn(banner string) *scriptedBannerConn {
	return &scriptedBannerConn{chunks: [][]byte{[]byte(banner)}}
}

// Read serves the next scripted chunk, or the scripted ending.
func (c *scriptedBannerConn) Read(p []byte) (int, error) {
	if c.position < len(c.chunks) {
		chunk := c.chunks[c.position]
		c.position++
		return copy(p, chunk), nil
	}
	if c.readErr != nil {
		return 0, c.readErr
	}
	return 0, io.EOF
}

// Write accepts the bytes and reports them written. These probes send nothing; a
// case that wanted to prove otherwise would have to count the writes.
func (c *scriptedBannerConn) Write(p []byte) (int, error) { return len(p), nil }

// Close records the close.
func (c *scriptedBannerConn) Close() error {
	c.closed = true
	return nil
}

// LocalAddr reports the scripted local address.
func (c *scriptedBannerConn) LocalAddr() net.Addr { return scriptedAddr("local") }

// RemoteAddr reports the scripted remote address.
func (c *scriptedBannerConn) RemoteAddr() net.Addr { return scriptedAddr("remote") }

// SetDeadline accepts a deadline without acting on it.
func (c *scriptedBannerConn) SetDeadline(time.Time) error { return nil }

// SetReadDeadline records the deadline the probe bounded its read with: without one,
// a far end that accepts the connection and then says nothing would hold the probe
// open for as long as the runner allows.
func (c *scriptedBannerConn) SetReadDeadline(t time.Time) error {
	c.deadline = t
	c.hasDeadline = true
	return nil
}

// SetWriteDeadline accepts a write deadline without acting on it.
func (c *scriptedBannerConn) SetWriteDeadline(time.Time) error { return nil }

// scriptedDialFunc is the Dialer seam a case uses when the answer depends on the
// address it is asked for: a probe that measures two regions dials two addresses and
// must be able to see a different outcome for each. It records the addresses and, per
// dial, whether the probe bounded the attempt.
type scriptedDialFunc struct {
	// answer produces the scripted outcome for one address.
	answer func(addr string) (net.Conn, error)
	// calls records the addresses dialed, as "network address".
	calls []string
	// hasDeadline records, per call, whether the probe bounded the dial.
	hasDeadline []bool
	// deadlines records, per call, how much of the probe's budget remained.
	deadlines []time.Duration
}

// DialContext records the attempt and answers it through the case's own function.
func (d *scriptedDialFunc) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d.calls = append(d.calls, network+" "+addr)
	if deadline, ok := ctx.Deadline(); ok {
		d.hasDeadline = append(d.hasDeadline, true)
		d.deadlines = append(d.deadlines, time.Until(deadline))
	} else {
		d.hasDeadline = append(d.hasDeadline, false)
		d.deadlines = append(d.deadlines, 0)
	}
	return d.answer(addr)
}

// cfDialer scripts the outcome of each declared address one Cloudflare edge case
// measures: an error for the addresses that fail, and a scripted connection whose
// close is recorded for the ones that answer. An address with no entry answers.
func cfDialer(perAddress map[string]error) (*scriptedDialFunc, map[string]*scriptedConn) {
	conns := map[string]*scriptedConn{}
	dialer := &scriptedDialFunc{}
	dialer.answer = func(addr string) (net.Conn, error) {
		if err := perAddress[addr]; err != nil {
			return nil, err
		}
		conn := &scriptedConn{}
		conns[addr] = conn
		return conn, nil
	}
	return dialer, conns
}

// reachSeams builds the seam set one reachability case describes: the deny-all set
// of design §6.2 with the resolver and the dialer replaced. A nil value is the zero
// capability — the run was never given one — which is a different fact from a seam
// that refuses, and both are cases of their own.
func reachSeams(resolver probe.Resolver, dialer probe.Dialer) probe.Seams {
	seams := probe.DenyAllSeams()
	seams.Resolver = resolver
	seams.Dialer = dialer
	return seams
}

// egressBuild builds one egress probe the way a run reaches it: through the
// registry, with the run's seams and its declared target input.
func egressBuild(t *testing.T, name string, seams probe.Seams, targets probe.TargetInput) probe.Probe {
	t.Helper()
	for _, built := range probe.ProbesFor(seams, targets) {
		if built.Name() == name {
			return built
		}
	}
	t.Fatalf("the registry built no %q probe, so no run could measure it", name)
	return nil
}

// runSsh runs one public-SSH probe over one scripted run.
func runSsh(t *testing.T, name string, seams probe.Seams, targets probe.TargetInput) probe.Result {
	t.Helper()
	return egressBuild(t, name, seams, targets).Run(context.Background())
}

// runCF runs one Cloudflare edge probe over one scripted run.
func runCF(t *testing.T, name string, seams probe.Seams, targets probe.TargetInput) probe.Result {
	t.Helper()
	return egressBuild(t, name, seams, targets).Run(context.Background())
}

// declaredAddresses returns the addresses targets.go declares for one probe, read
// through EffectiveTargets rather than repeated from the code: the probe must measure
// the target the declaration names, and the two values must agree (R-HR-NF-10).
func declaredAddresses(t *testing.T, name string) []string {
	t.Helper()
	effective, err := probe.EffectiveTargets(probe.TargetInput{})
	if err != nil {
		t.Fatalf("the declared target set did not resolve: %v", err)
	}
	var addresses []string
	for _, target := range effective {
		if target.Probe == name {
			addresses = append(addresses, target.Address())
		}
	}
	if len(addresses) == 0 {
		t.Fatalf("targets.go declares no endpoint for %q, so nothing could be measured", name)
	}
	return addresses
}

// declaredAddress returns one declared address for a probe that declares exactly
// one endpoint.
func declaredAddress(t *testing.T, name string) string {
	t.Helper()
	addresses := declaredAddresses(t, name)
	if len(addresses) != 1 {
		t.Fatalf("%q declares %d endpoints, want exactly one: %v", name, len(addresses), addresses)
	}
	return addresses[0]
}

// TestEgressSshProbesIdentifyTheServiceOnTheDeclaredPort is the positive outcome of
// both SSH destination probes: the declared host answers on the declared port with
// an SSH identification string. The probe reports a measured pass naming its own
// declared target and carrying the banner verbatim, and the dialer's own record
// proves the declared address is the address that was dialed (R-HR-NF-10).
func TestEgressSshProbesIdentifyTheServiceOnTheDeclaredPort(t *testing.T) {
	for _, probeCase := range sshProbes {
		t.Run(probeCase.name, func(t *testing.T) {
			address := declaredAddress(t, probeCase.name)
			wantAddress := fmt.Sprintf("%s:%d", probeCase.host, probeCase.port)
			if address != wantAddress {
				t.Fatalf("targets.go declares %q for %q, want the documented %q", address, probeCase.name, wantAddress)
			}

			conn := newBannerConn(testSSHBanner)
			resolver := &scriptedResolver{addresses: []string{"203.0.113.7"}}
			dialer := &scriptedDialer{conn: conn}
			result := runSsh(t, probeCase.name, reachSeams(resolver, dialer), probe.TargetInput{})

			if result.Probe != probeCase.name {
				t.Fatalf("result names the probe %q, want %q", result.Probe, probeCase.name)
			}
			if result.Kind != probe.ProbeEgress {
				t.Fatalf("result kind = %q, want %q", result.Kind, probe.ProbeEgress)
			}
			if result.Target != wantAddress {
				t.Errorf("result target = %q, want the declared target %q", result.Target, wantAddress)
			}
			if len(result.Observations) != 1 {
				t.Fatalf("the probe reported %d observations, want 1 answer to its one question", len(result.Observations))
			}
			observation := result.Observations[0]
			if !holds(observation) {
				t.Fatalf("observation does not satisfy the measurement vocabulary's invariant: %+v", observation)
			}
			if observation.Label != sshBannerLabel(probeCase.port) {
				t.Errorf("observation label = %q, want %q", observation.Label, sshBannerLabel(probeCase.port))
			}
			if observation.Target != wantAddress {
				t.Errorf("observation target = %q, want %q", observation.Target, wantAddress)
			}
			if observation.Resolution != probe.Measured || observation.Verdict != probe.Pass || observation.Reason != probe.ReasonOK {
				t.Errorf("observation = (%q, %q, %q), want (%q, %q, %q)", observation.Resolution, observation.Verdict, observation.Reason, probe.Measured, probe.Pass, probe.ReasonOK)
			}
			if result.Verdict != probe.Pass || result.Reason != probe.ReasonOK {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Pass, probe.ReasonOK)
			}
			verdict, reason := probe.Aggregate(result.Observations)
			if result.Verdict != verdict || result.Reason != reason {
				t.Fatalf("the result's reduction %s/%s disagrees with Aggregate over its observations %s/%s", result.Verdict, result.Reason, verdict, reason)
			}
			if !strings.Contains(observation.Detail, "SSH-2.0-OpenSSH_9.6") {
				t.Errorf("the pass detail does not carry the identification string verbatim: %q", observation.Detail)
			}

			if want := []string{"tcp " + wantAddress}; !reflect.DeepEqual(dialer.calls, want) {
				t.Errorf("the dialed addresses = %v, want exactly %v", dialer.calls, want)
			}
			if want := []string{probeCase.host}; !reflect.DeepEqual(resolver.lookups, want) {
				t.Errorf("the resolved names = %v, want exactly %v", resolver.lookups, want)
			}
			if !conn.closed {
				t.Error("the probe did not close the connection it read from")
			}
		})
	}
}

// TestEgressSshProbesReportANonSSHBannerAsAMeasuredFailure is the interception case:
// the port answered, so an attempt was made and answered — but not with an SSH
// identification string. That is a definite negative (`banner_not_ssh`), never a
// pass, and the text the far end sent stays available verbatim.
func TestEgressSshProbesReportANonSSHBannerAsAMeasuredFailure(t *testing.T) {
	for _, probeCase := range sshProbes {
		t.Run(probeCase.name, func(t *testing.T) {
			address := declaredAddress(t, probeCase.name)
			conn := newBannerConn(testNonSSHBanner)
			result := runSsh(t, probeCase.name, reachSeams(&scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{conn: conn}), probe.TargetInput{})

			if len(result.Observations) != 1 {
				t.Fatalf("the probe reported %d observations, want 1", len(result.Observations))
			}
			observation := result.Observations[0]
			if observation.Resolution != probe.Measured {
				t.Errorf("resolution = %q, want %q: the port answered", observation.Resolution, probe.Measured)
			}
			if observation.Verdict != probe.Fail || observation.Reason != probe.ReasonBannerNotSSH {
				t.Errorf("observation = (%q, %q), want (%q, %q)", observation.Verdict, observation.Reason, probe.Fail, probe.ReasonBannerNotSSH)
			}
			if result.Verdict != probe.Fail || result.Reason != probe.ReasonBannerNotSSH {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Fail, probe.ReasonBannerNotSSH)
			}
			if observation.Target != address {
				t.Errorf("observation target = %q, want %q: the answer came from the declared port", observation.Target, address)
			}
			if !strings.Contains(observation.Detail, "HTTP/1.1 403 Forbidden") {
				t.Errorf("the failure detail does not carry what the far end sent verbatim: %q", observation.Detail)
			}
			if result.Verdict == probe.Pass {
				t.Error("a port that is not serving SSH was reported as a pass")
			}
			if !conn.closed {
				t.Error("the probe did not close the connection it read from")
			}
		})
	}
}

// TestEgressSshProbesReportRefusedAndResetConnections asserts the two definite
// negatives a TCP attempt can produce for these probes: the far end refused the
// connection, or it reset one it had accepted. Both are measurements (an attempt was
// made and answered), both are failures, and the two reason codes stay distinct.
func TestEgressSshProbesReportRefusedAndResetConnections(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantReason probe.ReasonCode
	}{
		{
			"the far end refused the connection",
			fmt.Errorf("dial tcp: connect: %w", syscall.ECONNREFUSED),
			probe.ReasonConnRefused,
		},
		{
			"the far end reset the connection after it was accepted",
			fmt.Errorf("read tcp: %w", syscall.ECONNRESET),
			probe.ReasonConnReset,
		},
	}

	for _, probeCase := range sshProbes {
		for _, tc := range cases {
			t.Run(probeCase.name+"/"+tc.name, func(t *testing.T) {
				result := runSsh(t, probeCase.name, reachSeams(&scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{err: tc.err}), probe.TargetInput{})
				observation := result.Observations[0]
				if observation.Resolution != probe.Measured || observation.Verdict != probe.Fail {
					t.Errorf("observation = (%q, %q), want (%q, %q)", observation.Resolution, observation.Verdict, probe.Measured, probe.Fail)
				}
				if observation.Reason != tc.wantReason {
					t.Errorf("reason = %q, want %q", observation.Reason, tc.wantReason)
				}
				if result.Verdict != probe.Fail || result.Reason != tc.wantReason {
					t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Fail, tc.wantReason)
				}
				if !strings.Contains(observation.Detail, tc.err.Error()) {
					t.Errorf("the failure detail does not carry the socket's own error verbatim: %q", observation.Detail)
				}
			})
		}
	}

	refused := runSsh(t, testSSHKnownProbe, reachSeams(&scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{err: fmt.Errorf("connect: %w", syscall.ECONNREFUSED)}), probe.TargetInput{})
	reset := runSsh(t, testSSHKnownProbe, reachSeams(&scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{err: fmt.Errorf("read: %w", syscall.ECONNRESET)}), probe.TargetInput{})
	if refused.Reason == reset.Reason {
		t.Fatalf("a refused and a reset connection report the same reason %q; the reason set holds one code for each", refused.Reason)
	}
}

// TestEgressSshProbesDistinguishANameThatDoesNotExistFromAResolverThatDidNotAnswer
// is the pair these probes exist for. A resolver answering authoritatively that the
// name does not exist is a measurement *of the name* (measured fail,
// `dns_no_such_host`); a resolver that timed out or returned SERVFAIL produced no
// answer at all (unresolved, `dns_unresolved`). Collapsing the two would turn "we
// could not find out" into "the destination is blocked", which is the mistake
// PRD §1.1 exists to prevent.
//
// The pair is asserted on both surfaces that can raise it: the resolver seam, and a
// dialer whose own name resolution failed. The fact is about the name either way, so
// the two surfaces must not produce two different classifications.
func TestEgressSshProbesDistinguishANameThatDoesNotExistFromAResolverThatDidNotAnswer(t *testing.T) {
	negatives := []struct {
		name        string
		err         error
		wantRes     probe.Resolution
		wantVerdict probe.Verdict
		wantReason  probe.ReasonCode
	}{
		{
			"the resolver answered that the name does not exist",
			&net.DNSError{Err: "no such host", Name: "github.com", IsNotFound: true},
			probe.Measured, probe.Fail, probe.ReasonDNSNoSuchHost,
		},
		{
			"the resolver did not answer at all",
			&net.DNSError{Err: "i/o timeout", Name: "github.com", IsTimeout: true},
			probe.Unresolved, probe.Indeterminate, probe.ReasonDNSUnresolved,
		},
	}

	surfaces := []struct {
		name        string
		script      func(err error) (*scriptedResolver, *scriptedDialer)
		mustNotDial bool
	}{
		{
			"a resolver seam that fails",
			func(err error) (*scriptedResolver, *scriptedDialer) {
				return &scriptedResolver{err: err}, &scriptedDialer{conn: newBannerConn(testSSHBanner)}
			},
			true,
		},
		{
			"a dialer whose own name resolution fails",
			func(err error) (*scriptedResolver, *scriptedDialer) {
				return &scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{err: err}
			},
			false,
		},
	}

	for _, probeCase := range sshProbes {
		for _, surface := range surfaces {
			for _, negative := range negatives {
				t.Run(probeCase.name+"/"+surface.name+"/"+negative.name, func(t *testing.T) {
					resolver, dialer := surface.script(negative.err)
					result := runSsh(t, probeCase.name, reachSeams(resolver, dialer), probe.TargetInput{})

					if len(result.Observations) != 1 {
						t.Fatalf("the probe reported %d observations, want 1", len(result.Observations))
					}
					observation := result.Observations[0]
					if observation.Resolution != negative.wantRes || observation.Verdict != negative.wantVerdict || observation.Reason != negative.wantReason {
						t.Errorf("observation = (%q, %q, %q), want (%q, %q, %q)",
							observation.Resolution, observation.Verdict, observation.Reason,
							negative.wantRes, negative.wantVerdict, negative.wantReason)
					}
					if result.Verdict != negative.wantVerdict || result.Reason != negative.wantReason {
						t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, negative.wantVerdict, negative.wantReason)
					}
					if !strings.Contains(observation.Detail, negative.err.Error()) {
						t.Errorf("the detail does not carry the resolver's own error verbatim: %q", observation.Detail)
					}
					if observation.Target != declaredAddress(t, probeCase.name) {
						t.Errorf("observation target = %q, want the declared name's address", observation.Target)
					}
					if want := []string{probeCase.host}; !reflect.DeepEqual(resolver.lookups, want) {
						t.Errorf("the resolved names = %v, want exactly %v", resolver.lookups, want)
					}
					if surface.mustNotDial && len(dialer.calls) != 0 {
						t.Errorf("a name that could not be resolved was dialed anyway: %v", dialer.calls)
					}
					if negative.wantRes == probe.Measured && result.Verdict == probe.Pass {
						t.Error("a name-level failure was reported as a pass")
					}
				})
			}
		}
	}

	// The two must not collapse: same probe, same surface, two different measurements.
	notFound := runSsh(t, testSSHKnownProbe, reachSeams(&scriptedResolver{err: &net.DNSError{Err: "no such host", Name: "github.com", IsNotFound: true}}, &scriptedDialer{}), probe.TargetInput{})
	noAnswer := runSsh(t, testSSHKnownProbe, reachSeams(&scriptedResolver{err: &net.DNSError{Err: "i/o timeout", Name: "github.com", IsTimeout: true}}, &scriptedDialer{}), probe.TargetInput{})
	if notFound.Observations[0].Resolution == noAnswer.Observations[0].Resolution {
		t.Fatalf("both name-level failures report resolution %q; a name that does not exist is measured and a resolver that did not answer is not", notFound.Observations[0].Resolution)
	}
	if notFound.Reason == noAnswer.Reason {
		t.Fatalf("both name-level failures report reason %q; the closed set holds one code for each", notFound.Reason)
	}
}

// TestEgressSshProbesReportTheNotMeasuredAndBudgetOutcomes covers the remaining
// outcomes of the SSH question: a capability the run was never given or one that
// refused (two not-measured facts with two different reasons, design §5.1 obligation
// 2), and a port that accepted nothing inside the probe's own budget — which for the
// question "does this port speak SSH?" is the measurement itself, not an ambiguity.
func TestEgressSshProbesReportTheNotMeasuredAndBudgetOutcomes(t *testing.T) {
	blackholed := fmt.Errorf("dial tcp: i/o timeout: %w", os.ErrDeadlineExceeded)

	cases := []struct {
		name         string
		resolver     func() probe.Resolver
		dialer       func() probe.Dialer
		wantRes      probe.Resolution
		wantVerdict  probe.Verdict
		wantReason   probe.ReasonCode
		wantContains string
	}{
		{
			"the resolver seam denies the lookup",
			func() probe.Resolver { return probe.DenyAllSeams().Resolver },
			func() probe.Dialer { return &scriptedDialer{conn: newBannerConn(testSSHBanner)} },
			probe.NotMeasured, probe.Indeterminate, probe.ReasonCommandDenied, "denied",
		},
		{
			"no resolver is injected for the run",
			func() probe.Resolver { return nil },
			func() probe.Dialer { return &scriptedDialer{conn: newBannerConn(testSSHBanner)} },
			probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded, "no resolver",
		},
		{
			"the dial seam denies the connection",
			func() probe.Resolver { return &scriptedResolver{addresses: []string{"203.0.113.7"}} },
			func() probe.Dialer { return probe.DenyAllSeams().Dialer },
			probe.NotMeasured, probe.Indeterminate, probe.ReasonCommandDenied, "denied",
		},
		{
			"no dialer is injected for the run",
			func() probe.Resolver { return &scriptedResolver{addresses: []string{"203.0.113.7"}} },
			func() probe.Dialer { return nil },
			probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded, "no dialer",
		},
		{
			"the declared port never answered inside the probe's own budget",
			func() probe.Resolver { return &scriptedResolver{addresses: []string{"203.0.113.7"}} },
			func() probe.Dialer { return &scriptedDialer{err: blackholed} },
			probe.Measured, probe.Fail, probe.ReasonBudgetExpired, "budget",
		},
	}

	for _, probeCase := range sshProbes {
		for _, tc := range cases {
			t.Run(probeCase.name+"/"+tc.name, func(t *testing.T) {
				dialer := tc.dialer()
				result := runSsh(t, probeCase.name, reachSeams(tc.resolver(), dialer), probe.TargetInput{})
				observation := result.Observations[0]
				if !holds(observation) {
					t.Fatalf("observation does not satisfy the measurement vocabulary's invariant: %+v", observation)
				}
				if observation.Resolution != tc.wantRes || observation.Verdict != tc.wantVerdict || observation.Reason != tc.wantReason {
					t.Errorf("observation = (%q, %q, %q), want (%q, %q, %q)",
						observation.Resolution, observation.Verdict, observation.Reason,
						tc.wantRes, tc.wantVerdict, tc.wantReason)
				}
				if result.Verdict == probe.Pass {
					t.Error("an outcome that is not a positive answer was reported as a pass")
				}
				if !strings.Contains(observation.Detail, tc.wantContains) {
					t.Errorf("the detail does not name what was missing (%q): %q", tc.wantContains, observation.Detail)
				}
				if observation.Target != declaredAddress(t, probeCase.name) {
					t.Errorf("observation target = %q, want the declared target", observation.Target)
				}
				if scripted, ok := dialer.(*scriptedDialer); ok && tc.wantRes == probe.Measured {
					if len(scripted.hasDeadline) != 1 || !scripted.hasDeadline[0] {
						t.Errorf("the probe dialed without a deadline, so the expiry cannot be its own: %v", scripted.hasDeadline)
					}
				}
			})
		}
	}

	// Design §5.1 obligation 2: the two not-measured facts stay distinguishable.
	denied := runSsh(t, testSSHKnownProbe, reachSeams(probe.DenyAllSeams().Resolver, &scriptedDialer{}), probe.TargetInput{})
	missing := runSsh(t, testSSHKnownProbe, reachSeams(nil, &scriptedDialer{}), probe.TargetInput{})
	if denied.Reason == missing.Reason {
		t.Fatalf("a denying seam and a missing capability report the same reason %q; the two must be distinguishable", denied.Reason)
	}

	// The SSH probe waits twice inside one probe budget: once for the dial, once for
	// the identification string. Both waits must fit inside the runner's per-probe
	// bound, or a blackholed target would be abandoned by the runner and reported as
	// `probe_timeout` instead of as the probe's own measured negative.
	if 2*probe.DefaultDialBudget > probe.DefaultProbeTimeout {
		t.Errorf("the probe's two %s waits do not fit inside the runner's %s per-probe bound", probe.DefaultDialBudget, probe.DefaultProbeTimeout)
	}
}

// TestEgressSshProbesPreserveAPartialIdentificationString is the triangulation case
// for the banner read: a peer that answers in more than one packet, or whose stream
// ends mid-identification, must still be classified from what arrived. A truncated
// read that carries the identification prefix IS the identification string the probe
// was asking for, and the verbatim bytes stay beside the stable reason code (R-HR-07)
// while `SSH` alone, which is not an identification string, does not.
func TestEgressSshProbesPreserveAPartialIdentificationString(t *testing.T) {
	cases := []struct {
		name        string
		chunks      [][]byte
		readErr     error
		wantVerdict probe.Verdict
		wantReason  probe.ReasonCode
		wantDetail  string
	}{
		{
			"a peer that answers in two packets, truncated in the middle of the version",
			[][]byte{[]byte("SSH"), []byte("-2.0-OpenS")},
			nil,
			probe.Pass, probe.ReasonOK, "\"SSH-2.0-OpenS\"",
		},
		{
			"a read truncated to the identification prefix itself",
			[][]byte{[]byte("SSH-")},
			nil,
			probe.Pass, probe.ReasonOK, "\"SSH-\"",
		},
		{
			"a peer that sends three bytes and stops",
			[][]byte{[]byte("SSH")},
			nil,
			probe.Fail, probe.ReasonBannerNotSSH, "\"SSH\"",
		},
		{
			"a connection reset before anything arrived",
			nil,
			syscall.ECONNRESET,
			probe.Fail, probe.ReasonConnReset, "connection reset",
		},
		{
			"a connection that said nothing inside the budget",
			nil,
			fmt.Errorf("read tcp: i/o timeout: %w", os.ErrDeadlineExceeded),
			probe.Fail, probe.ReasonBudgetExpired, "budget",
		},
	}

	for _, probeCase := range sshProbes {
		for _, tc := range cases {
			t.Run(probeCase.name+"/"+tc.name, func(t *testing.T) {
				conn := &scriptedBannerConn{chunks: tc.chunks, readErr: tc.readErr}
				result := runSsh(t, probeCase.name, reachSeams(&scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{conn: conn}), probe.TargetInput{})

				if len(result.Observations) != 1 {
					t.Fatalf("the probe reported %d observations, want 1", len(result.Observations))
				}
				observation := result.Observations[0]
				if observation.Verdict != tc.wantVerdict || observation.Reason != tc.wantReason {
					t.Errorf("observation = (%q, %q), want (%q, %q)", observation.Verdict, observation.Reason, tc.wantVerdict, tc.wantReason)
				}
				if !strings.Contains(observation.Detail, tc.wantDetail) {
					t.Errorf("the detail does not carry %s verbatim: %q", tc.wantDetail, observation.Detail)
				}
				if !conn.hasDeadline {
					t.Error("the probe read the identification string without a deadline, so a silent far end could hold it open")
				}
				if !conn.closed {
					t.Error("the probe did not close the connection it read from")
				}
			})
		}
	}
}

// TestEgressSshProbesEstablishNoTransportViability asserts the R-HR-02 boundary these
// probes must not cross: they measure two ports, and a reachable public host
// establishes nothing about a transport. The structural half is that transport
// viability lives in `internal/transport` and this package cannot express it; the
// wording half is asserted here, over every outcome the probes can produce, because a
// claim in the text is what a user would read.
func TestEgressSshProbesEstablishNoTransportViability(t *testing.T) {
	scripts := []struct {
		name     string
		resolver probe.Resolver
		dialer   probe.Dialer
	}{
		{"a port that speaks SSH", &scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{conn: newBannerConn(testSSHBanner)}},
		{"a port that speaks something else", &scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{conn: newBannerConn(testNonSSHBanner)}},
		{"a port that refused the connection", &scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{err: fmt.Errorf("connect: %w", syscall.ECONNREFUSED)}},
		{"a name that does not exist", &scriptedResolver{err: &net.DNSError{Err: "no such host", IsNotFound: true}}, &scriptedDialer{}},
		{"a resolver that did not answer", &scriptedResolver{err: &net.DNSError{Err: "i/o timeout", IsTimeout: true}}, &scriptedDialer{}},
		{"a port that never answered", &scriptedResolver{addresses: []string{"203.0.113.7"}}, &scriptedDialer{err: fmt.Errorf("i/o timeout: %w", os.ErrDeadlineExceeded)}},
		{"no capability at all", nil, nil},
	}

	forbidden := []string{"viable", "viability", "transport", "recommend", "block"}
	for _, probeCase := range sshProbes {
		for _, script := range scripts {
			t.Run(probeCase.name+"/"+script.name, func(t *testing.T) {
				result := runSsh(t, probeCase.name, reachSeams(script.resolver, script.dialer), probe.TargetInput{})
				text := strings.ToLower(resultText(result))
				for _, token := range forbidden {
					if strings.Contains(text, token) {
						t.Errorf("the SSH probe's output contains %q, which is a claim about a transport rather than about this port: %q", token, resultText(result))
					}
				}
			})
		}
	}
}

// declaredTargetsFor returns the endpoints one probe declares, read from the
// declaration itself: the region probe's cases must assert that the declaration
// carries both regions, not that the probe happens to report two observations.
func declaredTargetsFor(t *testing.T, name string) []probe.Target {
	t.Helper()
	for _, declaration := range probe.DeclaredTargets() {
		if declaration.Probe == name {
			return declaration.Targets
		}
	}
	t.Fatalf("targets.go declares no probe %q, so no run could measure it", name)
	return nil
}

// TestEgressCFProbesObserveEachRegionSeparately is design D10's positive case: each
// Cloudflare edge probe measures both declared regions and reports one observation per
// region, from the one registry entry the ten-probe enumeration fixes. The declared
// addresses are read from the declaration and compared with the literals PRD §1.1
// measured, the dialer's own record proves both declared addresses were dialed, and
// both connections are closed.
func TestEgressCFProbesObserveEachRegionSeparately(t *testing.T) {
	if got := len(probe.Registry()); got != 10 {
		t.Fatalf("the registry holds %d probes, want the ten of PRD §5.1: two regions are two observations, not two probes", got)
	}

	for _, probeCase := range cfProbes {
		t.Run(probeCase.name, func(t *testing.T) {
			entries := 0
			for _, entry := range probe.Registry() {
				if entry.Name == probeCase.name {
					entries++
				}
			}
			if entries != 1 {
				t.Fatalf("%q is declared %d times in the registry: the two regions belong to one probe (D10)", probeCase.name, entries)
			}

			declaration := declaredTargetsFor(t, probeCase.name)
			if len(declaration) != 2 {
				t.Fatalf("%q declares %d endpoints, want both edge regions", probeCase.name, len(declaration))
			}
			wantHosts := []string{testRegion1Host, testRegion2Host}
			wantLabels := []string{"region1", "region2"}
			for i, target := range declaration {
				if target.Host != wantHosts[i] || target.Port != probeCase.port || target.Label != wantLabels[i] {
					t.Errorf("declared target %d = (%q, %d, %q), want (%q, %d, %q)", i, target.Host, target.Port, target.Label, wantHosts[i], probeCase.port, wantLabels[i])
				}
			}

			addresses := declaredAddresses(t, probeCase.name)
			wantAddresses := []string{
				fmt.Sprintf("%s:%d", testRegion1Host, probeCase.port),
				fmt.Sprintf("%s:%d", testRegion2Host, probeCase.port),
			}
			if !reflect.DeepEqual(addresses, wantAddresses) {
				t.Fatalf("the effective declared addresses = %v, want %v", addresses, wantAddresses)
			}

			dialer, conns := cfDialer(nil)
			result := runCF(t, probeCase.name, reachSeams(&scriptedResolver{addresses: []string{"203.0.113.7"}}, dialer), probe.TargetInput{})

			if len(result.Observations) != 2 {
				t.Fatalf("the probe reported %d observations, want one per declared region", len(result.Observations))
			}
			for i, observation := range result.Observations {
				if !holds(observation) {
					t.Fatalf("observation %d does not satisfy the measurement vocabulary's invariant: %+v", i, observation)
				}
				if want := fmt.Sprintf("tcp %d region%d", probeCase.port, i+1); observation.Label != want {
					t.Errorf("observation %d label = %q, want %q", i, observation.Label, want)
				}
				if observation.Target != addresses[i] {
					t.Errorf("observation %d target = %q, want the declared %q", i, observation.Target, addresses[i])
				}
				if observation.Resolution != probe.Measured || observation.Verdict != probe.Pass || observation.Reason != probe.ReasonOK {
					t.Errorf("observation %d = (%q, %q, %q), want (%q, %q, %q)", i,
						observation.Resolution, observation.Verdict, observation.Reason, probe.Measured, probe.Pass, probe.ReasonOK)
				}
			}
			if result.Verdict != probe.Pass || result.Reason != probe.ReasonOK {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Pass, probe.ReasonOK)
			}
			if want := strings.Join(addresses, ", "); result.Target != want {
				t.Errorf("result target = %q, want the declared set %q: a single-endpoint string would hide the region that was measured", result.Target, want)
			}
			if want := []string{"tcp " + addresses[0], "tcp " + addresses[1]}; !reflect.DeepEqual(dialer.calls, want) {
				t.Errorf("the dialed addresses = %v, want exactly %v", dialer.calls, want)
			}
			for _, address := range addresses {
				conn, ok := conns[address]
				if !ok {
					t.Errorf("%s was never dialed", address)
					continue
				}
				if !conn.closed {
					t.Errorf("the connection to %s was left open", address)
				}
			}
		})
	}
}

// TestEgressCFProbesReportASplitResultPerObservation is the reason both regions are
// measured at all: "one region reachable" is materially different from "both
// blocked", so a split must survive into the observations while the aggregate reports
// the failure. The failing region is varied so the aggregate cannot depend on a
// position.
func TestEgressCFProbesReportASplitResultPerObservation(t *testing.T) {
	refused := fmt.Errorf("dial tcp: connect: %w", syscall.ECONNREFUSED)

	for _, probeCase := range cfProbes {
		addresses := declaredAddresses(t, probeCase.name)
		for failing := range addresses {
			t.Run(fmt.Sprintf("%s/region%d fails", probeCase.name, failing+1), func(t *testing.T) {
				dialer, _ := cfDialer(map[string]error{addresses[failing]: refused})
				result := runCF(t, probeCase.name, reachSeams(&scriptedResolver{addresses: []string{"203.0.113.7"}}, dialer), probe.TargetInput{})

				if len(result.Observations) != 2 {
					t.Fatalf("the probe reported %d observations, want 2", len(result.Observations))
				}
				failed := result.Observations[failing]
				passed := result.Observations[1-failing]
				if failed.Resolution != probe.Measured || failed.Verdict != probe.Fail || failed.Reason != probe.ReasonConnRefused {
					t.Errorf("the failing region reported (%q, %q, %q), want (%q, %q, %q)",
						failed.Resolution, failed.Verdict, failed.Reason, probe.Measured, probe.Fail, probe.ReasonConnRefused)
				}
				if passed.Resolution != probe.Measured || passed.Verdict != probe.Pass || passed.Reason != probe.ReasonOK {
					t.Errorf("the reachable region reported (%q, %q, %q), want (%q, %q, %q)",
						passed.Resolution, passed.Verdict, passed.Reason, probe.Measured, probe.Pass, probe.ReasonOK)
				}
				if failed.Verdict == passed.Verdict {
					t.Error("the split is invisible: both regions report the same verdict")
				}
				if result.Verdict != probe.Fail || result.Reason != probe.ReasonConnRefused {
					t.Errorf("the aggregate = (%q, %q), want the failing region's (%q, %q)", result.Verdict, result.Reason, probe.Fail, probe.ReasonConnRefused)
				}
				if !strings.Contains(failed.Detail, addresses[failing]) {
					t.Errorf("the failing region's detail does not name its own address: %q", failed.Detail)
				}
			})
		}
	}
}

// TestEgressCFBlackholedTargetsAreTheProbesOwnBudget is the third timeout path for
// these probes: a declared edge that never answers inside the probe's own dial budget
// is a *measured* failure carrying `budget_expired`, produced by the probe rather than
// by the runner abandoning a hung one (design §5.1 obligation 3, RG-8). That is right
// here because the declared question is whether the edge is reachable: no answer inside
// the budget is the answer.
func TestEgressCFBlackholedTargetsAreTheProbesOwnBudget(t *testing.T) {
	blackholed := fmt.Errorf("dial tcp: i/o timeout: %w", os.ErrDeadlineExceeded)

	for _, probeCase := range cfProbes {
		t.Run(probeCase.name, func(t *testing.T) {
			addresses := declaredAddresses(t, probeCase.name)
			dialer, _ := cfDialer(map[string]error{addresses[0]: blackholed, addresses[1]: blackholed})
			result := runCF(t, probeCase.name, reachSeams(&scriptedResolver{addresses: []string{"203.0.113.7"}}, dialer), probe.TargetInput{})

			if len(result.Observations) != 2 {
				t.Fatalf("the probe reported %d observations, want 2", len(result.Observations))
			}
			for i, observation := range result.Observations {
				if observation.Resolution != probe.Measured || observation.Verdict != probe.Fail || observation.Reason != probe.ReasonBudgetExpired {
					t.Errorf("observation %d = (%q, %q, %q), want (%q, %q, %q)", i,
						observation.Resolution, observation.Verdict, observation.Reason, probe.Measured, probe.Fail, probe.ReasonBudgetExpired)
				}
				if !strings.Contains(observation.Detail, probe.DefaultDialBudget.String()) {
					t.Errorf("observation %d does not name the probe's own %s budget: %q", i, probe.DefaultDialBudget, observation.Detail)
				}
			}
			if result.Verdict != probe.Fail || result.Reason != probe.ReasonBudgetExpired {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Fail, probe.ReasonBudgetExpired)
			}
			if result.Reason == probe.ReasonProbeTimeout {
				t.Fatal("a probe-owned dial expiry was reported with the runner's probe_timeout reason")
			}
			if len(dialer.hasDeadline) != 2 {
				t.Fatalf("the probe dialed %d addresses, want 2", len(dialer.hasDeadline))
			}
			for i, bounded := range dialer.hasDeadline {
				if !bounded {
					t.Errorf("dial %d had no deadline, so the expiry cannot be the probe's own", i)
				}
				if dialer.deadlines[i] <= 0 || dialer.deadlines[i] > probe.DefaultDialBudget {
					t.Errorf("dial %d had %s of the probe's %s budget left", i, dialer.deadlines[i], probe.DefaultDialBudget)
				}
			}
		})
	}
}

// TestEgressCFProbesReportTheIndeterminateControlCase covers every way the edge
// question can end without an answer: no capability at all, a capability that
// refused, and a name that could not be resolved. None of them is a pass, none is a
// failure, and none of them states or implies a block — "we could not find out" is
// the honest report.
func TestEgressCFProbesReportTheIndeterminateControlCase(t *testing.T) {
	resolveTimeout := &net.DNSError{Err: "i/o timeout", Name: testRegion1Host, IsTimeout: true}

	cases := []struct {
		name       string
		resolver   probe.Resolver
		dialer     probe.Dialer
		wantRes    probe.Resolution
		wantReason probe.ReasonCode
	}{
		{
			"no capability at all",
			nil, nil,
			probe.NotMeasured, probe.ReasonCapabilityExcluded,
		},
		{
			"seams that deny everything",
			probe.DenyAllSeams().Resolver, probe.DenyAllSeams().Dialer,
			probe.NotMeasured, probe.ReasonCommandDenied,
		},
		{
			"a resolver that answered but no dialer for the run",
			&scriptedResolver{addresses: []string{"203.0.113.7"}}, nil,
			probe.NotMeasured, probe.ReasonCapabilityExcluded,
		},
		{
			"a dial seam that denies both regions",
			&scriptedResolver{addresses: []string{"203.0.113.7"}}, probe.DenyAllSeams().Dialer,
			probe.NotMeasured, probe.ReasonCommandDenied,
		},
		{
			"a resolver that did not answer for either region",
			&scriptedResolver{err: resolveTimeout}, &scriptedDialer{},
			probe.Unresolved, probe.ReasonDNSUnresolved,
		},
	}

	for _, probeCase := range cfProbes {
		for _, tc := range cases {
			t.Run(probeCase.name+"/"+tc.name, func(t *testing.T) {
				result := runCF(t, probeCase.name, reachSeams(tc.resolver, tc.dialer), probe.TargetInput{})

				if len(result.Observations) != 2 {
					t.Fatalf("the probe reported %d observations, want one per declared region", len(result.Observations))
				}
				for i, observation := range result.Observations {
					if observation.Resolution != tc.wantRes || observation.Verdict != probe.Indeterminate || observation.Reason != tc.wantReason {
						t.Errorf("observation %d = (%q, %q, %q), want (%q, %q, %q)", i,
							observation.Resolution, observation.Verdict, observation.Reason, tc.wantRes, probe.Indeterminate, tc.wantReason)
					}
					if observation.Verdict == probe.Pass || observation.Verdict == probe.Fail {
						t.Errorf("observation %d reported the definite verdict %q for a question that was not answered", i, observation.Verdict)
					}
				}
				if result.Verdict != probe.Indeterminate || result.Reason != tc.wantReason {
					t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Indeterminate, tc.wantReason)
				}
				if strings.Contains(strings.ToLower(resultText(result)), "block") {
					t.Errorf("an unanswered edge question states or implies a block: %q", resultText(result))
				}
			})
		}
	}
}

// TestEgressCFDeclaredRegionsSurviveAnOverride is the triangulation case for the
// declared target set: both regions are declared for both probes, an override
// *replaces* a probe's declared targets rather than appending to them, the replacement
// is reflected per observation, and the closed-set assertion still holds. It also
// scripts one region whose name does not resolve beside one that answers, so the name
// question is visible per region too.
func TestEgressCFDeclaredRegionsSurviveAnOverride(t *testing.T) {
	for _, probeCase := range cfProbes {
		declaration := declaredTargetsFor(t, probeCase.name)
		if len(declaration) != 2 {
			t.Fatalf("%q declares %d endpoints, want both regions", probeCase.name, len(declaration))
		}
		if declaration[0].Label != "region1" || declaration[1].Label != "region2" {
			t.Errorf("%q declares labels (%q, %q), want region1 and region2", probeCase.name, declaration[0].Label, declaration[1].Label)
		}
	}

	override := probe.TargetOverride{Probe: testCF443Probe, Host: "198.51.100.7", Port: 443}
	dialer, conns := cfDialer(nil)
	result := runCF(t, testCF443Probe, reachSeams(&scriptedResolver{addresses: []string{"203.0.113.7"}}, dialer), probe.TargetInput{Overrides: []probe.TargetOverride{override}})

	if len(result.Observations) != 1 {
		t.Fatalf("the overridden probe reported %d observations, want the one replaced endpoint", len(result.Observations))
	}
	if result.Observations[0].Target != "198.51.100.7:443" {
		t.Errorf("observation target = %q, want the override %q", result.Observations[0].Target, "198.51.100.7:443")
	}
	if result.Target != "198.51.100.7:443" {
		t.Errorf("result target = %q, want %q", result.Target, "198.51.100.7:443")
	}
	if want := []string{"tcp 198.51.100.7:443"}; !reflect.DeepEqual(dialer.calls, want) {
		t.Errorf("the dialed addresses = %v, want exactly %v", dialer.calls, want)
	}
	if _, ok := conns["198.51.100.7:443"]; !ok {
		t.Error("the override address was never dialed")
	}

	// The closed-set assertion: the override replaces that probe's two declared
	// regions with one endpoint, and no other probe's effective set moves.
	effective, err := probe.EffectiveTargets(probe.TargetInput{Overrides: []probe.TargetOverride{override}})
	if err != nil {
		t.Fatalf("the effective declared set did not resolve: %v", err)
	}
	wantTotal := 1
	for _, declaration := range probe.DeclaredTargets() {
		if declaration.Protocol == probe.ProtocolLocal || declaration.Probe == testCF443Probe {
			continue
		}
		wantTotal += len(declaration.Targets)
	}
	if len(effective) != wantTotal {
		t.Fatalf("the effective declared set holds %d targets, want %d: an override never appends", len(effective), wantTotal)
	}
	for _, declaration := range probe.DeclaredTargets() {
		var got []probe.EffectiveTarget
		for _, target := range effective {
			if target.Probe == declaration.Probe {
				got = append(got, target)
			}
		}
		if declaration.Probe == testCF443Probe {
			if len(got) != 1 || got[0].Address() != "198.51.100.7:443" {
				t.Errorf("the overridden probe's effective set = %+v, want the one supplied endpoint", got)
			}
			continue
		}
		if len(got) != len(declaration.Targets) {
			t.Errorf("%q declares %d endpoints but the effective set holds %d: an override moved another probe's targets", declaration.Probe, len(declaration.Targets), len(got))
		}
	}

	// A name question asked per region: region1's name does not exist, region2's
	// resolves and answers, so the split is visible per observation.
	nameDialer, _ := cfDialer(nil)
	resolver := &scriptedResolver{
		addresses: []string{"203.0.113.7"},
		perHost: map[string]error{
			testRegion1Host: &net.DNSError{Err: "no such host", Name: testRegion1Host, IsNotFound: true},
		},
	}
	nameResult := runCF(t, testCF443Probe, reachSeams(resolver, nameDialer), probe.TargetInput{})
	if len(nameResult.Observations) != 2 {
		t.Fatalf("the probe reported %d observations, want 2", len(nameResult.Observations))
	}
	if first := nameResult.Observations[0]; first.Resolution != probe.Measured || first.Verdict != probe.Fail || first.Reason != probe.ReasonDNSNoSuchHost {
		t.Errorf("region1 = (%q, %q, %q), want (%q, %q, %q)", first.Resolution, first.Verdict, first.Reason, probe.Measured, probe.Fail, probe.ReasonDNSNoSuchHost)
	}
	if second := nameResult.Observations[1]; second.Resolution != probe.Measured || second.Verdict != probe.Pass {
		t.Errorf("region2 = (%q, %q), want a measured pass", second.Resolution, second.Verdict)
	}
	if want := []string{testRegion1Host, testRegion2Host}; !reflect.DeepEqual(resolver.lookups, want) {
		t.Errorf("the resolved names = %v, want exactly %v", resolver.lookups, want)
	}
	if nameResult.Verdict != probe.Fail || nameResult.Reason != probe.ReasonDNSNoSuchHost {
		t.Errorf("the aggregate = (%q, %q), want the failing region's (%q, %q)", nameResult.Verdict, nameResult.Reason, probe.Fail, probe.ReasonDNSNoSuchHost)
	}
}

// TestEgressCFProbesEstablishNoTransportViability asserts the same R-HR-02 boundary as
// the SSH guard does, for the edge question: an edge that answered is a measurement of
// that edge, not evidence that a tunnel is viable or that a transport should be
// recommended. The Cloudflare-specific wording lives in `internal/transport`; this
// package cannot express viability at all.
func TestEgressCFProbesEstablishNoTransportViability(t *testing.T) {
	refused := fmt.Errorf("dial tcp: connect: %w", syscall.ECONNREFUSED)
	blackholed := fmt.Errorf("dial tcp: i/o timeout: %w", os.ErrDeadlineExceeded)

	for _, probeCase := range cfProbes {
		addresses := declaredAddresses(t, probeCase.name)
		scripts := []struct {
			name     string
			resolver probe.Resolver
			dialer   func() probe.Dialer
		}{
			{"both regions answered", &scriptedResolver{addresses: []string{"203.0.113.7"}}, cfDialerFunc(nil)},
			{"one region refused", &scriptedResolver{addresses: []string{"203.0.113.7"}}, cfDialerFunc(map[string]error{addresses[0]: refused})},
			{"both regions blackholed", &scriptedResolver{addresses: []string{"203.0.113.7"}}, cfDialerFunc(map[string]error{addresses[0]: blackholed, addresses[1]: blackholed})},
			{"no region's name resolves", &scriptedResolver{err: &net.DNSError{Err: "no such host", IsNotFound: true}}, func() probe.Dialer { return &scriptedDialer{} }},
			{"no capability at all", nil, func() probe.Dialer { return nil }},
		}

		forbidden := []string{"viable", "viability", "transport", "recommend", "block", "tunnel is"}
		for _, script := range scripts {
			t.Run(probeCase.name+"/"+script.name, func(t *testing.T) {
				result := runCF(t, probeCase.name, reachSeams(script.resolver, script.dialer()), probe.TargetInput{})
				text := strings.ToLower(resultText(result))
				for _, token := range forbidden {
					if strings.Contains(text, token) {
						t.Errorf("the edge probe's output contains %q, which is a claim about a transport rather than about this edge: %q", token, resultText(result))
					}
				}
			})
		}
	}
}

// cfDialerFunc is cfDialer as a value a case table can hold: a factory that builds the
// address-scripted dialer when the case runs, since a table cannot call a two-result
// function in a composite literal.
func cfDialerFunc(perAddress map[string]error) func() probe.Dialer {
	return func() probe.Dialer {
		dialer, _ := cfDialer(perAddress)
		return dialer
	}
}
