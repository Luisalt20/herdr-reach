package probe_test

// This file is the `egress.hub.direct` suite of PRD §5.1 and RG-13.
//
// The probe's declared target is run input, not a constant: with `--hub` the
// target is exactly the supplied `host[:port]`, and with no hub there is no target
// and the observation says so. Every case therefore states two things — the run
// input and the seam the dial would travel through — and the dialer records both
// the address it was asked for and the deadline the probe set on the dial. The
// deadline is recorded because "the probe's own budget expired" is only true if
// the probe set one; a refusal needs no deadline at all.

import (
	"context"
	"errors"
	"fmt"
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

// hubText is every string a result carries, so a wording guard reads the whole
// report and not only the field a case happened to think of.
func hubText(result probe.Result) string {
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
	if strings.Contains(strings.ToLower(hubText(result)), "block") {
		t.Errorf("the not-measured result states or implies a block: %q", hubText(result))
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
			text := strings.ToLower(hubText(result))
			for _, token := range forbidden {
				if strings.Contains(text, token) {
					t.Errorf("the hub probe's output contains %q, which is a claim about a transport rather than about this address: %q", token, hubText(result))
				}
			}
			if result.Verdict == probe.Pass && tc.targets.Hub == "" {
				t.Error("a hub that was never supplied was reported as a pass")
			}
		})
	}
}
