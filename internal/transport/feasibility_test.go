package transport_test

// This file is the feasibility evidence table of R-HR-06: a finding set crossed with every
// registered adapter, driven through the shipped `transport.Evaluate(transport.Registry(),
// diagnosis)` so each row is what a run would report. The cases cover a viable transport, the
// non-viable ones, transports whose relevant observation was measured and blocks them, transports
// whose relevant observation was never made, and the HTTP/2 advice paths — including the rule that
// no reason or note may state a fact the diagnosis did not carry.
//
// The PRD §1.1 replay is the table's named acceptance case: the public SSH signal passes and the
// hub address is blocked, so both SSH transports must reject naming the measured hub target and its
// port. The shared invariants run over every row of every case, so a row that stops explaining
// itself — or starts inventing a measurement — fails whichever case happens to produce it.

import (
	"net"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/transport"
)

// cfQUICRegion is the documentation-range endpoint the datagram fixtures measure (RFC 5737), beside
// the edge endpoints registry_test.go uses on port 443. It is a fixture target, never a dial.
const cfQUICRegion = "198.51.100.10:7844"

// quicFailed is a datagram measurement the socket reported as unreachable: the measured failure
// CF_HTTP2_ADVISED_QUIC_FAILED rests on.
func quicFailed() probe.Result {
	return result("egress.quic", probe.ProbeProto,
		measured("udp 7844 region1", cfQUICRegion, probe.Fail, probe.ReasonUDPUnreachable,
			"the socket surfaced an ICMP port-unreachable, so the far end answered that nothing is listening on this UDP port"),
	)
}

// quicUnanswered is a datagram attempt that produced no answer: the unconfirmed measurement
// CF_HTTP2_ADVISED_QUIC_UNCONFIRMED rests on, never a measured block.
func quicUnanswered() probe.Result {
	return result("egress.quic", probe.ProbeProto,
		unresolved("udp 7844 region1", cfQUICRegion, probe.ReasonUDPSilence,
			"no answer on UDP arrived inside the probe's own budget, and the socket reported no error at all"),
	)
}

// quicUsable is a datagram measurement that drew a reply: the narrow positive
// CF_NO_HTTP2_ADVICE_QUIC_USABLE rests on.
func quicUsable() probe.Result {
	return result("egress.quic", probe.ProbeProto,
		measured("udp 7844 region1", cfQUICRegion, probe.Pass, probe.ReasonUDPResponseReceived,
			"a datagram arrived from the edge, so this datagram to this edge and port was not silently dropped"),
	)
}

// quicNotMeasured is a datagram measurement the run could not attempt: the probe reports the
// excluded capability itself, and the conservative advice names that absence.
func quicNotMeasured() probe.Result {
	return result("egress.quic", probe.ProbeProto,
		notMeasured("udp 7844 region1", probe.ReasonCapabilityExcluded,
			"no packet dialer is injected for this run, so this measurement could not be attempted"),
	)
}

// adapterExpectation pins one adapter's row in one case. Every case also runs the shared invariants
// over every registered adapter, so this struct carries only the expectations that case exists for.
type adapterExpectation struct {
	name        string
	viable      bool
	reason      []string
	notes       []string
	notSaid     []string
	satisfied   []transport.RequirementKind
	unsatisfied []transport.RequirementKind
}

// feasibilityEvidenceCase is one finding set and the rows it must produce.
type feasibilityEvidenceCase struct {
	name string
	run  []probe.Result
	// want pins the adapters this case is about.
	want []adapterExpectation
	// blockedHub lists the adapters whose reason must name the measured hub target and its port.
	blockedHub []string
}

// feasibilityEvidenceCases is the table. The runs are built from the fixtures the contract suite
// already uses, so the table states finding sets rather than a second opinion about them.
func feasibilityEvidenceCases() []feasibilityEvidenceCase {
	return []feasibilityEvidenceCase{
		{
			name:       "the PRD §1.1 replay rejects both SSH transports naming the blocked hub target and port",
			run:        []probe.Result{publicSSHPassed(), hubRejected()},
			blockedHub: []string{"direct-ssh", "reverse-ssh"},
			want: []adapterExpectation{
				{name: "direct-ssh", viable: false, reason: []string{"EGRESS_HUB_DIRECT_FAIL", "did not accept a connection"}},
				{name: "reverse-ssh", viable: false, reason: []string{"EGRESS_HUB_DIRECT_FAIL"}},
				{name: "cloudflare-tunnel", viable: false, reason: []string{"cloudflare.edge", "was not made"}},
				{name: "tailscale", viable: false, reason: []string{"no measurement", "not reported viable"}},
			},
		},
		{
			name: "a measured hub pass makes direct-ssh viable and names the measured observation",
			run:  []probe.Result{hubPassed()},
			want: []adapterExpectation{
				{
					name:      "direct-ssh",
					viable:    true,
					reason:    []string{"accepted a connection", "can work on that address"},
					notes:     []string{hubTarget},
					satisfied: []transport.RequirementKind{transport.KindHubAddress},
				},
				{name: "cloudflare-tunnel", viable: false},
				{name: "tailscale", viable: false},
			},
		},
		{
			name: "a hub pass beside a configured sshd makes reverse-ssh viable",
			run:  []probe.Result{hubPassed(), sshdConfigured()},
			want: []adapterExpectation{
				{name: "direct-ssh", viable: true},
				{
					name:      "reverse-ssh",
					viable:    true,
					reason:    []string{"configuration in force is the written one", "can dial the hub and forward"},
					notes:     []string{hubTarget, sshdBinaryPath, sshdConfigPath},
					satisfied: []transport.RequirementKind{transport.KindHubAddress, transport.KindSSHDEffectiveConfig},
				},
			},
		},
		{
			name:       "a measured hub block keeps both SSH transports non-viable even with a configured sshd",
			run:        []probe.Result{hubRejected(), sshdConfigured()},
			blockedHub: []string{"direct-ssh", "reverse-ssh"},
			want: []adapterExpectation{
				{
					name:      "direct-ssh",
					viable:    false,
					reason:    []string{"did not accept a connection", "rejected by that measurement"},
					satisfied: []transport.RequirementKind{transport.KindHubAddress},
				},
				{
					name:      "reverse-ssh",
					viable:    false,
					reason:    []string{"EGRESS_HUB_DIRECT_FAIL", "rejected by that measurement"},
					satisfied: []transport.RequirementKind{transport.KindHubAddress, transport.KindSSHDEffectiveConfig},
				},
			},
		},
		{
			name: "an unreachable edge is a measured block for cloudflare-tunnel and produces no HTTP/2 advice",
			run:  []probe.Result{cfEdgeUnreachable(), quicFailed()},
			want: []adapterExpectation{
				{
					name:    "cloudflare-tunnel",
					viable:  false,
					reason:  []string{"unreachable", "every measured edge endpoint failed", cfEdgeRegionOne},
					notes:   []string{transport.PinNote()},
					notSaid: []string{"HTTP/2 recommendation", "post-quantum", "no HTTP/2 downgrade", cfQUICRegion},
				},
			},
		},
		{
			name: "a hub probe absent from the run leaves both SSH transports unmeasured and claims no block",
			run:  []probe.Result{sshdConfigured()},
			want: []adapterExpectation{
				{
					name:        "direct-ssh",
					viable:      false,
					reason:      []string{"was not made", "hub.reachability"},
					notSaid:     []string{"blocked", "203.0.113.10"},
					unsatisfied: []transport.RequirementKind{transport.KindHubAddress},
				},
				{
					name:        "reverse-ssh",
					viable:      false,
					reason:      []string{"was not made", "hub.reachability"},
					notSaid:     []string{"blocked", "203.0.113.10"},
					unsatisfied: []transport.RequirementKind{transport.KindHubAddress},
				},
			},
		},
		{
			name: "a not-measured hub observation names the missing input and claims no block",
			run:  []probe.Result{hubNotMeasured(), sshdConfigured()},
			want: []adapterExpectation{
				{
					name:        "direct-ssh",
					viable:      false,
					reason:      []string{"input_missing_hub", "--hub"},
					notSaid:     []string{"blocked", "203.0.113.10"},
					unsatisfied: []transport.RequirementKind{transport.KindHubAddress},
				},
				{
					name:        "reverse-ssh",
					viable:      false,
					reason:      []string{"input_missing_hub"},
					notSaid:     []string{"blocked", "203.0.113.10"},
					unsatisfied: []transport.RequirementKind{transport.KindHubAddress},
				},
			},
		},
		{
			name: "an excluded sshd measurement leaves reverse-ssh unmeasured and direct-ssh viable",
			run:  []probe.Result{hubPassed(), sshdNotMeasured()},
			want: []adapterExpectation{
				{name: "direct-ssh", viable: true},
				{
					name:        "reverse-ssh",
					viable:      false,
					reason:      []string{"capability_excluded", "was not measured"},
					notSaid:     []string{"blocked"},
					unsatisfied: []transport.RequirementKind{transport.KindSSHDEffectiveConfig},
				},
			},
		},
		{
			name: "tailscale carries no measurement at all and no rejection is claimed",
			run:  []probe.Result{hubPassed(), sshdConfigured()},
			want: []adapterExpectation{
				{
					name:    "tailscale",
					viable:  false,
					reason:  []string{"no measurement", "not reported viable"},
					notSaid: []string{"blocked", hubTarget, sshdConfigPath},
				},
			},
		},
		{
			name: "a cloudflare edge probe absent from the run is an absence, not a block",
			run:  []probe.Result{hubPassed()},
			want: []adapterExpectation{
				{
					name:    "cloudflare-tunnel",
					viable:  false,
					reason:  []string{"cloudflare.edge", "was not made"},
					notes:   []string{transport.PinNote(), "not measured"},
					notSaid: []string{"blocked", cfEdgeRegionOne, cfQUICRegion, "HTTP/2 recommendation"},
				},
			},
		},
		{
			name: "a failed datagram measurement beside a reachable edge advises HTTP/2 and states the trade-off",
			run:  []probe.Result{cfEdgeReachable(), quicFailed()},
			want: []adapterExpectation{
				{
					name:   "cloudflare-tunnel",
					viable: false,
					reason: []string{"is reachable", "hostname", "not satisfied"},
					notes: []string{
						"HTTP/2 recommendation", "post-quantum", "this slice recommends and does not enforce",
						"no fallback was applied", "no configuration was written", transport.PinNote(),
					},
					notSaid: []string{"equivalent", "has been applied", "was applied to", "a configuration was written"},
				},
			},
		},
		{
			name: "an unconfirmed datagram measurement names the measurement it rests on",
			run:  []probe.Result{cfEdgeReachable(), quicUnanswered()},
			want: []adapterExpectation{
				{
					name:   "cloudflare-tunnel",
					viable: false,
					notes: []string{
						"HTTP/2 recommendation", "unconfirmed measurement", "egress.quic", cfQUICRegion, "udp_silence",
						"post-quantum", "this slice recommends and does not enforce", transport.PinNote(),
					},
				},
			},
		},
		{
			name: "a not-measured datagram measurement names the excluded capability it rests on",
			run:  []probe.Result{cfEdgeReachable(), quicNotMeasured()},
			want: []adapterExpectation{
				{
					name:   "cloudflare-tunnel",
					viable: false,
					notes: []string{
						"HTTP/2 recommendation", "unconfirmed measurement", "egress.quic", "capability_excluded",
					},
					notSaid: []string{cfQUICRegion},
				},
			},
		},
		{
			name: "a datagram reply produces no downgrade advice",
			run:  []probe.Result{cfEdgeReachable(), quicUsable()},
			want: []adapterExpectation{
				{
					name:    "cloudflare-tunnel",
					viable:  false,
					notes:   []string{"no HTTP/2 downgrade is recommended", "drew a reply"},
					notSaid: []string{"HTTP/2 recommendation", "post-quantum"},
				},
			},
		},
		{
			name: "a missing cloudflare.http2 finding produces no advice",
			run:  []probe.Result{cfEdgeReachable()},
			want: []adapterExpectation{
				{
					name:    "cloudflare-tunnel",
					viable:  false,
					notes:   []string{transport.PinNote()},
					notSaid: []string{"HTTP/2 recommendation", "post-quantum", "recommend", cfQUICRegion},
				},
			},
		},
	}
}

// TestFeasibilityEvidenceTable is the table's body: every case is diagnosed and evaluated through
// the shipped entry point, the shared R-HR-06 invariants run over every registered adapter, and the
// case's own expectations pin the rows it exists for.
func TestFeasibilityEvidenceTable(t *testing.T) {
	for _, tc := range feasibilityEvidenceCases() {
		t.Run(tc.name, func(t *testing.T) {
			d := diagnosis.Diagnose(tc.run)
			registry := transport.Registry()
			rows := transport.Evaluate(registry, d)
			assertEveryRowExplainsItself(t, d, registry, rows)

			for _, want := range tc.want {
				row := rowFor(t, registry, rows, want.name)
				if row.Viable != want.viable {
					t.Errorf("%s viability = %t, want %t: reason %q", want.name, row.Viable, want.viable, row.Reason)
				}
				for _, phrase := range want.reason {
					if !strings.Contains(row.Reason, phrase) {
						t.Errorf("%s reason %q does not state %q", want.name, row.Reason, phrase)
					}
				}
				notes := strings.Join(row.Notes, "\n")
				for _, phrase := range want.notes {
					if !strings.Contains(notes, phrase) {
						t.Errorf("%s notes %q do not state %q", want.name, notes, phrase)
					}
				}
				for _, phrase := range want.notSaid {
					if strings.Contains(row.Reason+"\n"+notes, phrase) {
						t.Errorf("%s states %q, which this diagnosis did not carry", want.name, phrase)
					}
				}
				for _, kind := range want.satisfied {
					if req := requirementFor(t, row.Requires, kind); !req.Satisfied {
						t.Errorf("%s reports %s unsatisfied, want satisfied: %+v", want.name, kind, req)
					}
				}
				for _, kind := range want.unsatisfied {
					if req := requirementFor(t, row.Requires, kind); req.Satisfied {
						t.Errorf("%s reports %s satisfied, want unsatisfied: %+v", want.name, kind, req)
					}
				}
			}

			for _, name := range tc.blockedHub {
				row := rowFor(t, registry, rows, name)
				for _, phrase := range []string{"203.0.113.10", "port 22"} {
					if !strings.Contains(row.Reason, phrase) {
						t.Errorf("%s reason %q does not name the blocked hub target and its port (%q)", name, row.Reason, phrase)
					}
				}
			}
		})
	}
}

// TestCloudflareReadsTheDeclaredHTTP2IDs guards the duplication the vocabulary boundary forces: the
// reasoning layer's hand-named `cloudflare.http2` ids are unexported constants, so the cloudflare
// adapter spells them literally. A rename in the table would silently strand the adapter on its
// unrecognised-rule path, so the literals are asserted to stay declared ids.
func TestCloudflareReadsTheDeclaredHTTP2IDs(t *testing.T) {
	declared := diagnosis.AllRuleIDs()
	for _, id := range []string{
		"CF_HTTP2_ADVISED_QUIC_FAILED",
		"CF_HTTP2_ADVISED_QUIC_UNCONFIRMED",
		"CF_NO_HTTP2_ADVICE_QUIC_USABLE",
		"CF_HTTP2_ADVISORY_NOT_ASSESSED",
	} {
		if !slices.Contains(declared, id) {
			t.Errorf("the cloudflare adapter switches on %q, which diagnosis.AllRuleIDs() no longer declares", id)
		}
	}
}

// TestCloudflareHTTP2AdviceIgnoresRulesItCannotRead is the advice block's defensive case: a rule
// this adapter does not recognise and a missing finding produce no advice and no invented
// measurement, while a recognised unconfirmed rule whose finding carries no evidence still states
// the recommendation but says the measurement cannot be named instead of quoting or inventing one.
func TestCloudflareHTTP2AdviceIgnoresRulesItCannotRead(t *testing.T) {
	edge := diagnosis.Finding{
		Question: "cloudflare.edge",
		Rule:     "CF_EDGE_REACHABLE",
		Evidence: []diagnosis.Fact{{
			Probe:       cfEdgeProbe,
			Resolution:  probe.Measured,
			Verdict:     probe.Pass,
			State:       diagnosis.StatePass,
			Observation: measured("tcp 443 region1", cfEdgeRegionOne, probe.Pass, probe.ReasonOK, "the connection was established"),
		}},
	}
	quicFact := diagnosis.Fact{
		Probe:       "egress.quic",
		Resolution:  probe.Unresolved,
		Verdict:     probe.Indeterminate,
		State:       diagnosis.StateUnresolved,
		Observation: unresolved("udp 7844 region1", cfQUICRegion, probe.ReasonUDPSilence, "no answer arrived inside the budget"),
	}

	cases := []struct {
		name string
		// advice, when set, is added as a `cloudflare.http2` finding.
		advice *diagnosis.Finding
		// wantAdvice reports whether the advising wording is expected; the measurement it rests on
		// may never be invented either way.
		wantAdvice bool
	}{
		{name: "a rule this adapter does not recognise", advice: &diagnosis.Finding{
			Question: "cloudflare.http2",
			Rule:     "CF_HTTP2_SOMETHING_NEW",
			Evidence: []diagnosis.Fact{quicFact},
		}},
		{name: "a missing http2 finding"},
		{name: "a recognised unconfirmed rule without evidence", advice: &diagnosis.Finding{
			Question: "cloudflare.http2",
			Rule:     "CF_HTTP2_ADVISED_QUIC_UNCONFIRMED",
		}, wantAdvice: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := []diagnosis.Finding{edge}
			if tc.advice != nil {
				findings = append(findings, *tc.advice)
			}
			d := diagnosis.Diagnosis{Findings: findings}
			registry := transport.Registry()
			rows := transport.Evaluate(registry, d)
			row := rowFor(t, registry, rows, "cloudflare-tunnel")
			text := row.Reason + "\n" + strings.Join(row.Notes, "\n")
			hasAdvice := strings.Contains(text, "HTTP/2 recommendation")
			if hasAdvice != tc.wantAdvice {
				t.Errorf("the notes carry the recommendation = %t, want %t: %q", hasAdvice, tc.wantAdvice, text)
			}
			if strings.Contains(text, cfQUICRegion) {
				t.Errorf("the answer names %q, a measurement the finding did not carry: %q", cfQUICRegion, text)
			}
			if !tc.wantAdvice && strings.Contains(text, "post-quantum") {
				t.Errorf("the answer states the HTTP/2 trade-off without giving advice: %q", text)
			}
			if !slices.Contains(row.Notes, transport.PinNote()) {
				t.Errorf("the cloudflare notes %q do not carry the pin note", row.Notes)
			}
		})
	}
}

// TestFeasibilityAddressGuardIsLoadBearing guards the guard: the invented-target check is only
// meaningful if the token matcher actually sees measured addresses and the table actually names
// some. The controls assert the fixture-shaped forms match and non-address words do not, and that
// at least one row of the table names a measured address — a matcher broken into matching nothing,
// or a table whose rows stopped naming targets, cannot make every case pass vacuously.
func TestFeasibilityAddressGuardIsLoadBearing(t *testing.T) {
	for _, match := range []string{hubTarget, cfEdgeRegionOne, cfQUICRegion, "203.0.113.10 (port 22)"} {
		if !addressToken.MatchString(match) {
			t.Errorf("the address matcher does not match %q, so the invented-target check is vacuous", match)
		}
	}
	for _, noMatch := range []string{"host[:port]", "--hub", "cloudflared"} {
		if addressToken.MatchString(noMatch) {
			t.Errorf("the address matcher matches %q, which is not a measured address", noMatch)
		}
	}

	named := 0
	for _, tc := range feasibilityEvidenceCases() {
		d := diagnosis.Diagnose(tc.run)
		for _, row := range transport.Evaluate(transport.Registry(), d) {
			text := row.Reason + "\n" + strings.Join(row.Notes, "\n")
			named += len(addressToken.FindAllString(text, -1))
		}
	}
	if named == 0 {
		t.Fatal("no case in the table names a measured address, so the invented-target assertion never exercises its matcher")
	}
}

// assertEveryRowExplainsItself is the R-HR-06 loop design §3.2 requires over every row of every
// case: the reason is non-empty in both cases, a viable row names at least one measured observation
// in its notes, a non-viable row never reads as viable, and no row states an address the diagnosis
// did not carry.
func assertEveryRowExplainsItself(t *testing.T, d diagnosis.Diagnosis, registry []transport.Candidate, rows []transport.Feasibility) {
	t.Helper()
	if len(rows) != len(registry) {
		t.Fatalf("Evaluate returned %d rows for %d registered candidates: one row per candidate", len(rows), len(registry))
	}
	known := diagnosisTargets(d)
	for i, candidate := range registry {
		row := rows[i]
		if row.Reason == "" {
			t.Errorf("%s carries no reason (viable=%t): every transport explains its verdict in both cases", candidate.Name(), row.Viable)
		}
		if row.Viable && !namesAMeasuredObservation(row.Notes, d) {
			t.Errorf("%s is viable with notes %v, which name no measured observation", candidate.Name(), row.Notes)
		}
		text := row.Reason + "\n" + strings.Join(row.Notes, "\n")
		if !row.Viable {
			for _, phrase := range []string{"can work on that address", "is viable", "will work", "ready to use"} {
				if strings.Contains(text, phrase) {
					t.Errorf("%s is not viable but reads as viable (%q): %q", candidate.Name(), phrase, text)
				}
			}
		}
		assertTargetsWereMeasured(t, candidate.Name(), text, known)
	}
}

// addressToken matches an IPv4 address with an optional port, the shape a measured target is named
// with in a reason or a note.
var addressToken = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}(?::\d+)?\b`)

// assertTargetsWereMeasured fails the case when a reason or note names an address no observation in
// the diagnosis carried. The comparison is by host, because the measured-observation wording names
// the port beside the host while the evidence note carries host:port.
func assertTargetsWereMeasured(t *testing.T, label, text string, known []string) {
	t.Helper()
	for _, address := range addressToken.FindAllString(text, -1) {
		host := hostOf(address)
		found := false
		for _, target := range known {
			if hostOf(target) == host {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s states the address %q, which no measurement in the diagnosis carried; known targets: %v", label, address, known)
		}
	}
}

// hostOf returns the host half of a host[:port] string, or the string itself when it carries no
// port.
func hostOf(hostPort string) string {
	if host, _, err := net.SplitHostPort(hostPort); err == nil {
		return host
	}
	return hostPort
}

// diagnosisTargets collects every measured target the diagnosis carries, across every finding's
// evidence.
func diagnosisTargets(d diagnosis.Diagnosis) []string {
	var targets []string
	for _, finding := range d.Findings {
		for _, fact := range finding.Evidence {
			if fact.Observation.Target != "" {
				targets = append(targets, fact.Observation.Target)
			}
		}
	}
	return targets
}
