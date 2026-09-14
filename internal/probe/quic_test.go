package probe_test

// This file is the D8 datagram suite: the `egress.quic` probe (PR 8), which asks one
// narrow question of each declared Cloudflare edge region over UDP —
//
//	does one UDP datagram to this edge and port draw any reply inside the probe's own budget?
//
// — and reports exactly one of four outcomes per region: a reply (a measured pass,
// claiming only that the datagram was not silently dropped), silence (unresolved, because
// a datagram that drew no answer measured no answer), an ICMP port-unreachable (a
// measured failure), and any other socket error (unresolved, carried verbatim and never
// dressed up as an answer). A datagram question can also be unable to attempt its
// exchange at all — no packet dialer injected, or a packet seam that refused — and those
// are two distinguishable not-measured facts, never a claimed absence of a reply.
//
// The cases script the packet seam rather than a real socket, exactly as the
// reachability suite scripts the dial seam: a `scriptedPacketConn` serves one scripted
// exchange and records the deadline and the recipient of the datagram, and a
// `scriptedPacketDialer` records the addresses it was asked for. Every case asserts the
// declared address was dialed over exactly the `udp` network, and the module-level case
// asserts no non-stdlib dependency was added (D8's rejected alternative was a
// third-party QUIC stack).
//
// The property the cases exist for is RG-5: a silent socket is never a failure, and no
// wording in this probe's output states or implies that anything is blocked. Silence here
// means "no answer on UDP", so the declared narrow question is stated in the detail text
// itself and is asserted per outcome.

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
	// testQuicProbe is D8's probe, as the registry declares it.
	testQuicProbe = "egress.quic"
	// testQuicDeclaredQuestion is the question the probe states in its own detail
	// text, transcribed here from design D8. A wording change that drops the narrow
	// question breaks this suite rather than silently widening the probe's claim.
	testQuicDeclaredQuestion = "does one UDP datagram to this edge and port draw any reply inside the probe's own budget?"
	// testQuicReply is the datagram the scripted edge answers with. Its content is
	// irrelevant to the declared question, which is only whether any reply arrives.
	testQuicReply = "any reply at all"
)

// scriptedPacketConn is the PacketConn seam one datagram case injects. It serves the
// case's scripted exchange — a reply payload, or the error that ends the read — records
// the deadline the probe set, records the recipients of every datagram the probe sent,
// and records that the probe closed the socket.
type scriptedPacketConn struct {
	// reply is the datagram the far end sends. It is served by ReadFrom when set.
	reply []byte
	// readErr is what ReadFrom reports in place of a reply. A deadline expiry here is
	// how "silence" is scripted.
	readErr error
	// writeErr is what WriteTo reports. It is unset in every case where the datagram
	// is sent.
	writeErr error
	// deadline records the instant the probe asked the socket to stop by.
	deadline time.Time
	// hasDeadline records that the probe bounded the exchange at all.
	hasDeadline bool
	// recipients records the addresses the probe sent a datagram to, in order.
	recipients []string
	// closed records the probe's close.
	closed bool
}

// WriteTo records the recipient and reports the bytes sent, or the scripted write error.
func (c *scriptedPacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	c.recipients = append(c.recipients, addr.String())
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	return len(p), nil
}

// ReadFrom serves the scripted reply, then the scripted read error.
func (c *scriptedPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	if c.reply != nil {
		return copy(p, c.reply), scriptedAddr("edge"), nil
	}
	if c.readErr != nil {
		return 0, scriptedAddr("edge"), c.readErr
	}
	return 0, scriptedAddr("edge"), nil
}

// SetDeadline records the bound the probe put on the datagram exchange: without one, a
// socket that draws neither a reply nor an error would hold the probe open for as long
// as the runner allows.
func (c *scriptedPacketConn) SetDeadline(t time.Time) error {
	c.deadline = t
	c.hasDeadline = true
	return nil
}

// Close records the close.
func (c *scriptedPacketConn) Close() error {
	c.closed = true
	return nil
}

// scriptedPacketDialer is the PacketDialer seam a case injects. It records every address
// it was asked to open, and the deadline the probe put on the dial context, and answers
// with a fresh copy of its template connection per address — so each declared region's
// socket is observable on its own.
type scriptedPacketDialer struct {
	// template is copied for every dial. A case scripts the whole exchange here.
	template *scriptedPacketConn
	// perAddress scripts one connection per declared address, for the cases that
	// need the two regions to answer differently. An address with no entry falls
	// back to template.
	perAddress map[string]*scriptedPacketConn
	// err is what the dial returns when it is not nil.
	err error
	// calls records the dials, as "network address".
	calls []string
	// hasDeadline records, per dial, whether the probe bounded the attempt.
	hasDeadline []bool
	// deadlines records, per dial, how much of the probe's budget remained.
	deadlines []time.Duration
	// conns records the connection handed to each address, so a case can assert the
	// socket of every region was closed.
	conns map[string]*scriptedPacketConn
}

// DialPacket records the attempt and returns a fresh copy of the scripted connection.
func (d *scriptedPacketDialer) DialPacket(ctx context.Context, network, addr string) (probe.PacketConn, error) {
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
	template := d.template
	if scripted, ok := d.perAddress[addr]; ok {
		template = scripted
	}
	if template == nil {
		// A dialer that reports neither a socket nor an error leaves nothing to
		// measure; the probe must report that rather than panicking.
		return nil, nil
	}
	conn := *template
	if d.conns == nil {
		d.conns = map[string]*scriptedPacketConn{}
	}
	d.conns[addr] = &conn
	return &conn, nil
}

// packetSeams builds the seam set one datagram case describes: the deny-all set of
// design §6.2 with the packet dialer replaced. A nil dialer is the zero capability —
// the run was never given one — which is a different fact from a seam that refuses, and
// both are cases of their own.
func packetSeams(dialer probe.PacketDialer) probe.Seams {
	seams := probe.DenyAllSeams()
	seams.PacketDialer = dialer
	return seams
}

// runQuic runs `egress.quic` over one scripted run, the way a run reaches it: through
// the registry, with the run's seams and its declared target input.
func runQuic(t *testing.T, seams probe.Seams, targets probe.TargetInput) probe.Result {
	t.Helper()
	return egressBuild(t, testQuicProbe, seams, targets).Run(context.Background())
}

// TestEgressQuicReportsTheFourDatagramOutcomes is D8's outcome table: a reply is a
// measured pass carrying only the narrow claim, silence is unresolved, an ICMP
// port-unreachable is a measured failure, and any other socket error is unresolved. Each
// case runs over both declared regions, asserts the observation triple and the aggregate,
// asserts the declared address was dialed over the `udp` network with the probe's own
// deadline, asserts every socket was closed, and asserts the declared narrow question is
// stated in the detail.
func TestEgressQuicReportsTheFourDatagramOutcomes(t *testing.T) {
	addresses := declaredAddresses(t, testQuicProbe)
	if len(addresses) != 2 {
		t.Fatalf("%q declares %d endpoints, want the two documented regions", testQuicProbe, len(addresses))
	}
	if got := len(probe.Registry()); got != 10 {
		t.Fatalf("the registry holds %d probes, want the ten of PRD §5.1", got)
	}

	timeout := func(addr string) error {
		return fmt.Errorf("read udp %s: i/o timeout: %w", addr, os.ErrDeadlineExceeded)
	}
	unreachable := func(addr string) error {
		return fmt.Errorf("read udp %s: %w", addr, syscall.ECONNREFUSED)
	}
	other := func(addr string) error {
		return fmt.Errorf("read udp %s: socket is not connected", addr)
	}

	cases := []struct {
		name           string
		conn           *scriptedPacketConn
		wantResolution probe.Resolution
		wantVerdict    probe.Verdict
		wantReason     probe.ReasonCode
		wantDetail     []string
	}{
		{
			name:           "any udp reply is a measured pass",
			conn:           &scriptedPacketConn{reply: []byte(testQuicReply)},
			wantResolution: probe.Measured,
			wantVerdict:    probe.Pass,
			wantReason:     probe.ReasonUDPResponseReceived,
			wantDetail:     []string{"not silently dropped"},
		},
		{
			name:           "silence is unresolved",
			conn:           &scriptedPacketConn{readErr: timeout(addresses[0])},
			wantResolution: probe.Unresolved,
			wantVerdict:    probe.Indeterminate,
			wantReason:     probe.ReasonUDPSilence,
			wantDetail:     []string{"no answer on UDP", probe.DefaultDialBudget.String()},
		},
		{
			name:           "an icmp port-unreachable is a measured failure",
			conn:           &scriptedPacketConn{readErr: unreachable(addresses[0])},
			wantResolution: probe.Measured,
			wantVerdict:    probe.Fail,
			wantReason:     probe.ReasonUDPUnreachable,
			wantDetail:     []string{"ICMP port-unreachable"},
		},
		{
			name:           "any other socket error is unresolved and carried verbatim",
			conn:           &scriptedPacketConn{readErr: other(addresses[0])},
			wantResolution: probe.Unresolved,
			wantVerdict:    probe.Indeterminate,
			wantReason:     probe.ReasonUDPErrorUnclassified,
			wantDetail:     []string{"socket is not connected"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dialer := &scriptedPacketDialer{template: tc.conn}
			result := runQuic(t, packetSeams(dialer), probe.TargetInput{})

			if result.Probe != testQuicProbe {
				t.Fatalf("result names the probe %q, want %q", result.Probe, testQuicProbe)
			}
			if result.Kind != probe.ProbeProto {
				t.Fatalf("result kind = %q, want %q", result.Kind, probe.ProbeProto)
			}
			if len(result.Observations) != 2 {
				t.Fatalf("the probe reported %d observations, want one per declared region", len(result.Observations))
			}
			for i, observation := range result.Observations {
				if !holds(observation) {
					t.Fatalf("observation %d does not satisfy the measurement vocabulary's invariant: %+v", i, observation)
				}
				if observation.Resolution != tc.wantResolution || observation.Verdict != tc.wantVerdict || observation.Reason != tc.wantReason {
					t.Errorf("observation %d = (%q, %q, %q), want (%q, %q, %q)", i,
						observation.Resolution, observation.Verdict, observation.Reason,
						tc.wantResolution, tc.wantVerdict, tc.wantReason)
				}
				if observation.Target != addresses[i] {
					t.Errorf("observation %d target = %q, want the declared %q", i, observation.Target, addresses[i])
				}
				if wantLabel := fmt.Sprintf("udp 7844 region%d", i+1); observation.Label != wantLabel {
					t.Errorf("observation %d label = %q, want %q", i, observation.Label, wantLabel)
				}
				if !strings.Contains(observation.Detail, testQuicDeclaredQuestion) {
					t.Errorf("observation %d does not state the declared narrow question: %q", i, observation.Detail)
				}
				for _, want := range tc.wantDetail {
					if !strings.Contains(observation.Detail, want) {
						t.Errorf("observation %d detail does not carry %q: %q", i, want, observation.Detail)
					}
				}
			}

			wantCalls := []string{"udp " + addresses[0], "udp " + addresses[1]}
			if !reflect.DeepEqual(dialer.calls, wantCalls) {
				t.Errorf("the dialed addresses = %v, want exactly %v over the udp network", dialer.calls, wantCalls)
			}
			if len(dialer.hasDeadline) != 2 {
				t.Fatalf("the probe dialed %d addresses, want 2", len(dialer.hasDeadline))
			}
			for i, bounded := range dialer.hasDeadline {
				if !bounded {
					t.Errorf("dial %d had no deadline, so a silent socket could hold the probe open", i)
				}
				if dialer.deadlines[i] <= 0 || dialer.deadlines[i] > probe.DefaultDialBudget {
					t.Errorf("dial %d had %s of the probe's %s budget left", i, dialer.deadlines[i], probe.DefaultDialBudget)
				}
			}
			for _, address := range addresses {
				conn, ok := dialer.conns[address]
				if !ok {
					t.Errorf("the declared address %q was never dialed", address)
					continue
				}
				if !conn.hasDeadline {
					t.Errorf("the socket for %q carried no deadline, so the probe bounded nothing", address)
				}
				if !conn.closed {
					t.Errorf("the probe left the socket for %q open", address)
				}
				if want := []string{address}; !reflect.DeepEqual(conn.recipients, want) {
					t.Errorf("the datagrams for %q went to %v, want exactly %v", address, conn.recipients, want)
				}
			}

			verdict, reason := probe.Aggregate(result.Observations)
			if result.Verdict != verdict || result.Reason != reason {
				t.Fatalf("the result's reduction %s/%s disagrees with Aggregate over its observations %s/%s", result.Verdict, result.Reason, verdict, reason)
			}
			if result.Verdict != tc.wantVerdict || result.Reason != tc.wantReason {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, tc.wantVerdict, tc.wantReason)
			}
		})
	}
}

// TestEgressQuicReportsTheIndeterminateControlCase covers every way the datagram
// question can end without an answer: no capability at all, a capability that refused, a
// socket that drew neither a reply nor an error, an unclassified socket error, and a
// dialer that reported a socket it did not return. None of them is a pass, none is a
// failure, and none of them states or implies a block.
func TestEgressQuicReportsTheIndeterminateControlCase(t *testing.T) {
	cases := []struct {
		name       string
		dialer     probe.PacketDialer
		wantRes    probe.Resolution
		wantReason probe.ReasonCode
	}{
		{
			"no capability at all",
			nil,
			probe.NotMeasured, probe.ReasonCapabilityExcluded,
		},
		{
			"a packet seam that denies everything",
			probe.DenyAllSeams().PacketDialer,
			probe.NotMeasured, probe.ReasonCommandDenied,
		},
		{
			"a packet dialer that refuses to open a socket",
			&scriptedPacketDialer{err: fmt.Errorf("dial udp: %w", probe.ErrSeamDenied)},
			probe.NotMeasured, probe.ReasonCommandDenied,
		},
		{
			"a socket that drew neither a reply nor an error",
			&scriptedPacketDialer{template: &scriptedPacketConn{readErr: fmt.Errorf("read udp: i/o timeout: %w", os.ErrDeadlineExceeded)}},
			probe.Unresolved, probe.ReasonUDPSilence,
		},
		{
			"an unclassified socket error",
			&scriptedPacketDialer{template: &scriptedPacketConn{readErr: errors.New("udp: some other failure")}},
			probe.Unresolved, probe.ReasonUDPErrorUnclassified,
		},
		{
			"a dialer that reported a socket it did not return",
			&scriptedPacketDialer{},
			probe.Unresolved, probe.ReasonInternalError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runQuic(t, packetSeams(tc.dialer), probe.TargetInput{})

			if len(result.Observations) != 2 {
				t.Fatalf("the probe reported %d observations, want one per declared region", len(result.Observations))
			}
			for i, observation := range result.Observations {
				if observation.Resolution != tc.wantRes || observation.Verdict != probe.Indeterminate || observation.Reason != tc.wantReason {
					t.Errorf("observation %d = (%q, %q, %q), want (%q, %q, %q)", i,
						observation.Resolution, observation.Verdict, observation.Reason,
						tc.wantRes, probe.Indeterminate, tc.wantReason)
				}
				if observation.Verdict == probe.Pass || observation.Verdict == probe.Fail {
					t.Errorf("observation %d reported the definite verdict %q for a question that was not answered", i, observation.Verdict)
				}
			}
			if result.Verdict != probe.Indeterminate || result.Reason != tc.wantReason {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Indeterminate, tc.wantReason)
			}
			if strings.Contains(strings.ToLower(resultText(result)), "block") {
				t.Errorf("an unanswered datagram question states or implies a block: %q", resultText(result))
			}
		})
	}
}

// TestEgressQuicSilenceIsNeverAFailureAndClaimsOnlyTheQuestion is RG-5's property: a
// silent socket is reported as unresolved, never as a failure; the aggregate never
// promotes a silent region to a pass; and no outcome's wording states or implies that
// anything is blocked, viable, recommended or a transport. It also scripts a split
// result — one region answering, one silent — because the aggregate must not hide the
// region that measured nothing.
func TestEgressQuicSilenceIsNeverAFailureAndClaimsOnlyTheQuestion(t *testing.T) {
	addresses := declaredAddresses(t, testQuicProbe)
	silentRead := fmt.Errorf("read udp %s: i/o timeout: %w", addresses[0], os.ErrDeadlineExceeded)

	forbidden := []string{"block", "viable", "viability", "transport", "recommend"}
	scripts := []struct {
		name   string
		dialer probe.PacketDialer
	}{
		{"a reply", &scriptedPacketDialer{template: &scriptedPacketConn{reply: []byte(testQuicReply)}}},
		{"silence", &scriptedPacketDialer{template: &scriptedPacketConn{readErr: silentRead}}},
		{"an icmp port-unreachable", &scriptedPacketDialer{template: &scriptedPacketConn{readErr: fmt.Errorf("read udp: %w", syscall.ECONNREFUSED)}}},
		{"any other socket error", &scriptedPacketDialer{template: &scriptedPacketConn{readErr: errors.New("udp: some other failure")}}},
		{"a dial that failed", &scriptedPacketDialer{err: errors.New("dial udp: network is down")}},
		{"no capability at all", nil},
		{"a packet seam that denies everything", probe.DenyAllSeams().PacketDialer},
	}

	for _, script := range scripts {
		t.Run(script.name, func(t *testing.T) {
			result := runQuic(t, packetSeams(script.dialer), probe.TargetInput{})
			text := strings.ToLower(resultText(result))
			for _, token := range forbidden {
				if strings.Contains(text, token) {
					t.Errorf("the datagram probe's output contains %q, which is a claim beyond its declared question: %q", token, resultText(result))
				}
			}
			for i, observation := range result.Observations {
				if observation.Verdict == probe.Fail && observation.Reason == probe.ReasonUDPSilence {
					t.Errorf("observation %d reported silence as a failure: %+v", i, observation)
				}
			}
		})
	}

	// Silence itself: unresolved, never a failure, and the aggregate keeps it that way.
	silent := runQuic(t, packetSeams(&scriptedPacketDialer{template: &scriptedPacketConn{readErr: silentRead}}), probe.TargetInput{})
	if silent.Verdict != probe.Indeterminate || silent.Reason != probe.ReasonUDPSilence {
		t.Errorf("a silent socket aggregated to (%q, %q), want (%q, %q)", silent.Verdict, silent.Reason, probe.Indeterminate, probe.ReasonUDPSilence)
	}
	if silent.Verdict == probe.Fail {
		t.Fatal("a silent socket was reported as a failure")
	}
	for i, observation := range silent.Observations {
		if observation.Resolution != probe.Unresolved || observation.Verdict != probe.Indeterminate || observation.Reason != probe.ReasonUDPSilence {
			t.Errorf("silent region %d = (%q, %q, %q), want (%q, %q, %q)", i,
				observation.Resolution, observation.Verdict, observation.Reason,
				probe.Unresolved, probe.Indeterminate, probe.ReasonUDPSilence)
		}
		if !strings.Contains(observation.Detail, testQuicDeclaredQuestion) {
			t.Errorf("the silence detail does not state the narrow question it answered nothing about: %q", observation.Detail)
		}
	}

	// A split result: region1 answers, region2 is silent. The aggregate must be the
	// unanswered region's, not the answering region's pass, and each observation keeps
	// its own outcome.
	split := runQuic(t, packetSeams(&scriptedPacketDialer{
		template: &scriptedPacketConn{reply: []byte(testQuicReply)},
		perAddress: map[string]*scriptedPacketConn{
			addresses[1]: {readErr: fmt.Errorf("read udp %s: i/o timeout: %w", addresses[1], os.ErrDeadlineExceeded)},
		},
	}), probe.TargetInput{})
	if len(split.Observations) != 2 {
		t.Fatalf("the split run reported %d observations, want one per declared region", len(split.Observations))
	}
	if first := split.Observations[0]; first.Resolution != probe.Measured || first.Verdict != probe.Pass {
		t.Errorf("region1 = (%q, %q), want a measured pass", first.Resolution, first.Verdict)
	}
	if second := split.Observations[1]; second.Resolution != probe.Unresolved || second.Verdict != probe.Indeterminate || second.Reason != probe.ReasonUDPSilence {
		t.Errorf("region2 = (%q, %q, %q), want the silent (%q, %q, %q)", second.Resolution, second.Verdict, second.Reason, probe.Unresolved, probe.Indeterminate, probe.ReasonUDPSilence)
	}
	if split.Verdict != probe.Indeterminate || split.Reason != probe.ReasonUDPSilence {
		t.Errorf("the split aggregate = (%q, %q), want the unanswered region's (%q, %q)", split.Verdict, split.Reason, probe.Indeterminate, probe.ReasonUDPSilence)
	}
}

// TestEgressQuicDeclaredRegionsSurviveAnOverride is the triangulation case for the
// declared target set: the probe declares both regions on the datagram protocol, an
// override *replaces* them rather than appending, and the replacement is reflected in
// both the observation and the address the datagram is sent to.
func TestEgressQuicDeclaredRegionsSurviveAnOverride(t *testing.T) {
	declaration := declaredTargetsFor(t, testQuicProbe)
	if len(declaration) != 2 {
		t.Fatalf("%q declares %d endpoints, want both regions", testQuicProbe, len(declaration))
	}
	if declaration[0].Label != "region1" || declaration[1].Label != "region2" {
		t.Errorf("%q declares labels (%q, %q), want region1 and region2", testQuicProbe, declaration[0].Label, declaration[1].Label)
	}
	if declaration[0].Port != 7844 || declaration[1].Port != 7844 {
		t.Errorf("%q declares ports (%d, %d), want the tunnel port 7844", testQuicProbe, declaration[0].Port, declaration[1].Port)
	}

	for _, entry := range probe.DeclaredTargets() {
		if entry.Probe != testQuicProbe {
			continue
		}
		if entry.Protocol != probe.ProtocolUDP {
			t.Errorf("%q is declared over %q, want the datagram protocol %q", testQuicProbe, entry.Protocol, probe.ProtocolUDP)
		}
	}

	override := probe.TargetOverride{Probe: testQuicProbe, Host: "198.51.100.7", Port: 7844}
	dialer := &scriptedPacketDialer{template: &scriptedPacketConn{reply: []byte(testQuicReply)}}
	result := runQuic(t, packetSeams(dialer), probe.TargetInput{Overrides: []probe.TargetOverride{override}})

	if len(result.Observations) != 1 {
		t.Fatalf("the overridden probe reported %d observations, want the one replaced endpoint", len(result.Observations))
	}
	if result.Observations[0].Target != "198.51.100.7:7844" {
		t.Errorf("observation target = %q, want the override %q", result.Observations[0].Target, "198.51.100.7:7844")
	}
	if result.Target != "198.51.100.7:7844" {
		t.Errorf("result target = %q, want %q", result.Target, "198.51.100.7:7844")
	}
	if want := []string{"udp 198.51.100.7:7844"}; !reflect.DeepEqual(dialer.calls, want) {
		t.Errorf("the dialed addresses = %v, want exactly %v", dialer.calls, want)
	}
	if conn, ok := dialer.conns["198.51.100.7:7844"]; !ok {
		t.Error("the override address was never dialed")
	} else if want := []string{"198.51.100.7:7844"}; !reflect.DeepEqual(conn.recipients, want) {
		t.Errorf("the datagrams went to %v, want exactly %v", conn.recipients, want)
	}
}

// TestQuicProbeAddsNoQuicDependency asserts D8's decision from the outside: this slice
// answers the datagram question with a raw datagram over the injected packet seam rather
// than with a third-party QUIC stack, so the module still carries no non-stdlib
// dependency. The rejected alternative was a full QUIC handshake through such a stack.
func TestQuicProbeAddsNoQuicDependency(t *testing.T) {
	contents, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("the module file could not be read, so the no-dependency property cannot be checked: %v", err)
	}
	if strings.Contains(string(contents), "require") {
		t.Errorf("go.mod declares a module requirement, so this slice added a dependency it was assigned not to add:\n%s", contents)
	}
}
