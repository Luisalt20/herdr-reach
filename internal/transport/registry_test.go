package transport_test

// This file is the registry closure proof of R-HR-06 and the feasibility cases of the two adapters
// this slice appends to it: exactly four V1 transports in declaration order, a copy returned to
// every caller, and no adapter researched for a later slice present. The feasibility cases drive
// cloudflare-tunnel and tailscale through the shipped Registry() and Evaluate, so what they assert
// is what a run would report.
//
// The cloudflare cases pin design D6's concrete case — a reachable edge with no hostname is not
// viable, names the unmet hostname, and states that the prerequisite is an account fact rather
// than a network block — and design §3.2's reason precedence over the remaining edge states. The
// tailscale case pins the unmeasured-transport answer: never viable in R1a, no measurement of the
// adapter exists, and no rejection is claimed.

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/transport"
)

// v1TransportNames is PRD §5.2's V1 adapter list in declaration order. It is shared with
// notimplemented_test.go, so a registry that shrinks, grows or reorders fails the closure case and
// the not-implemented proof together.
var v1TransportNames = []string{"direct-ssh", "reverse-ssh", "cloudflare-tunnel", "tailscale"}

// The cloudflare edge fixtures use the probe name the reasoning layer derives the edge group from
// and documentation-range targets (RFC 5737), so a case names a real measurement without dialing
// anything.
const (
	cfEdgeProbe     = "egress.cf.443"
	cfEdgeRegionOne = "198.51.100.10:443"
	cfEdgeRegionTwo = "198.51.100.11:443"
)

// TestRegistryIsExactlyFourTransportsInDeclarationOrder is R-HR-06's registry scenario: the exact
// name set in declaration order, stable across calls, with no adapter outside the set.
func TestRegistryIsExactlyFourTransportsInDeclarationOrder(t *testing.T) {
	got := transportNames(transport.Registry())
	if !slices.Equal(got, v1TransportNames) {
		t.Fatalf("Registry() enumerates %v, want PRD §5.2's four V1 transports in declaration order %v", got, v1TransportNames)
	}
	// The comparison above is equality over the exact name set, not containment: an adapter
	// researched for a later slice (Microsoft Dev Tunnels, Herdr Cloud, any other) makes it fail
	// instead of being tolerated as an extra row.
	if again := transportNames(transport.Registry()); !slices.Equal(again, v1TransportNames) {
		t.Errorf("a second Registry() enumerates %v, want the same stable order %v", again, v1TransportNames)
	}
}

// TestRegistryReturnsACopy mutates a returned slice and reads the registry again, so an aliased
// registry would be observed through the second call.
func TestRegistryReturnsACopy(t *testing.T) {
	first := transport.Registry()
	if len(first) != len(v1TransportNames) {
		t.Fatalf("Registry() holds %d entries, want %d", len(first), len(v1TransportNames))
	}
	first[0] = nil
	first[1], first[3] = first[3], first[1]
	again := transport.Registry()
	if !slices.Equal(transportNames(again), v1TransportNames) {
		t.Errorf("mutating the slice Registry() returned changed a later call: got %v, want %v", transportNames(again), v1TransportNames)
	}
	for i, candidate := range again {
		if candidate == nil {
			t.Errorf("entry %d is nil after a caller mutated a previous copy: Registry() must return a copy", i)
		}
	}
}

// candidateFor returns one registered adapter by name, failing the case when it is not registered.
func candidateFor(t *testing.T, name string) transport.Candidate {
	t.Helper()
	for _, candidate := range transport.Registry() {
		if candidate.Name() == name {
			return candidate
		}
	}
	t.Fatalf("no transport named %q is registered", name)
	return nil
}

// TestCloudflareTunnelRequiresAHostnameAndZoneMembership pins the declared prerequisite set — the
// hostname first, then the zone membership, each with a Detail naming what the user must supply —
// and the R1a evaluated rows: no input of this slice supplies a Cloudflare account fact, so both
// rows are unsatisfied and their Details are preserved verbatim.
func TestCloudflareTunnelRequiresAHostnameAndZoneMembership(t *testing.T) {
	declared := candidateFor(t, "cloudflare-tunnel").Requires()
	wantKinds := []transport.RequirementKind{transport.KindHostname, transport.KindZoneMembership}
	if len(declared) != len(wantKinds) {
		t.Fatalf("cloudflare-tunnel declares %d prerequisites, want exactly %v in declaration order: %+v", len(declared), wantKinds, declared)
	}
	for i, kind := range wantKinds {
		if declared[i].Kind != kind {
			t.Errorf("declared row %d is %q, want %q: the hostname comes before the zone membership", i, declared[i].Kind, kind)
		}
		if declared[i].Detail == "" {
			t.Errorf("the %s declaration carries no Detail naming what the user must supply", kind)
		}
		if declared[i].Satisfied {
			t.Errorf("the static %s declaration claims satisfaction; satisfaction is decided per run by Feasible", kind)
		}
	}

	d := diagnosis.Diagnose([]probe.Result{cfEdgeReachable()})
	registry, rows := registryRows(t, d)
	row := rowFor(t, registry, rows, "cloudflare-tunnel")
	if len(row.Requires) != len(wantKinds) {
		t.Fatalf("cloudflare-tunnel evaluates %d requirement rows, want exactly %d: %+v", len(row.Requires), len(wantKinds), row.Requires)
	}
	for i, kind := range wantKinds {
		evaluated := row.Requires[i]
		if evaluated.Kind != kind {
			t.Errorf("evaluated row %d is %q, want %q: declaration order is evaluation order", i, evaluated.Kind, kind)
		}
		if evaluated.Satisfied {
			t.Errorf("R1a supplies no Cloudflare account fact, so %s must be evaluated unsatisfied: %+v", kind, evaluated)
		}
		if evaluated.Detail != declared[i].Detail {
			t.Errorf("the evaluated %s Detail %q drifted from the declared one %q", kind, evaluated.Detail, declared[i].Detail)
		}
	}
}

// TestCloudflareTunnelIsNeverViableInThisSlice is design D6's concrete case. A reachable edge with
// no hostname must be not viable, must name the hostname prerequisite in its reason, and must
// state in its notes that the edge is reachable and that the prerequisite is an account fact
// rather than a network block — the most attractive transport reads not viable on a healthy
// network, and the explanation is what keeps that honest.
func TestCloudflareTunnelIsNeverViableInThisSlice(t *testing.T) {
	d := diagnosis.Diagnose([]probe.Result{cfEdgeReachable()})
	registry, rows := registryRows(t, d)
	row := rowFor(t, registry, rows, "cloudflare-tunnel")
	if row.Viable {
		t.Fatal("cloudflare-tunnel is viable with no hostname: R1a supplies no account fact, so it can never be viable in this slice")
	}
	if row.Reason == "" {
		t.Fatal("cloudflare-tunnel carries no reason for its verdict")
	}
	for _, want := range []string{"reachable", string(transport.KindHostname), "not satisfied"} {
		if !strings.Contains(row.Reason, want) {
			t.Errorf("the reason %q does not name %q: the reason must name the missing hostname", row.Reason, want)
		}
	}
	if strings.Contains(row.Reason, "blocked") {
		t.Errorf("the reason %q claims the transport is blocked; a reachable edge measured no block", row.Reason)
	}
	notes := strings.Join(row.Notes, "\n")
	for _, want := range []string{cfEdgeRegionOne, "account prerequisite", "not a network block"} {
		if !strings.Contains(notes, want) {
			t.Errorf("the notes %q do not state %q: the prerequisite must not read as a network fact", notes, want)
		}
	}
	if !noteStates(row.Notes, transport.KindHostname) || !noteStates(row.Notes, transport.KindZoneMembership) {
		t.Errorf("the notes %v do not state both unsatisfied prerequisites", row.Notes)
	}
	if strings.Contains(row.Reason+notes, "ready") {
		t.Errorf("the answer describes the transport as ready to use: %q", row.Reason+notes)
	}
}

// TestTailscaleIsNeverViableInThisSlice pins the unmeasured-transport answer R-HR-06 requires: the
// adapter is never viable in R1a, its reason states that the relevant measurement is missing and
// claims no block, its one prerequisite is the unsatisfied third_party_permission row, and its
// notes carry both the requirement note and the slice's measurement gap. It reads no finding, so
// the same answer must come back whatever diagnosis it is handed.
func TestTailscaleIsNeverViableInThisSlice(t *testing.T) {
	declared := candidateFor(t, "tailscale").Requires()
	if len(declared) != 1 {
		t.Fatalf("tailscale declares %d prerequisites, want exactly the one third_party_permission row: %+v", len(declared), declared)
	}
	if declared[0].Kind != transport.KindThirdPartyPermission {
		t.Errorf("tailscale declares requirement kind %q, want %q", declared[0].Kind, transport.KindThirdPartyPermission)
	}
	if declared[0].Detail == "" {
		t.Error("the third_party_permission declaration carries no Detail naming what the user must check")
	}

	runs := []struct {
		name string
		run  []probe.Result
	}{
		{"an empty run", nil},
		{"a run with a rejected hub", []probe.Result{hubRejected()}},
		{"a run with a reachable cloudflare edge", []probe.Result{cfEdgeReachable()}},
	}
	var first transport.Feasibility
	for i, tc := range runs {
		t.Run(tc.name, func(t *testing.T) {
			registry, rows := registryRows(t, diagnosis.Diagnose(tc.run))
			row := rowFor(t, registry, rows, "tailscale")
			if row.Viable {
				t.Fatal("tailscale is viable in R1a: no probe measures whether a tunnel adapter is permitted or installed")
			}
			if row.Reason == "" {
				t.Fatal("tailscale carries no reason for its verdict")
			}
			for _, want := range []string{"no measurement", "not reported viable"} {
				if !strings.Contains(row.Reason, want) {
					t.Errorf("the reason %q does not state %q: the unmeasured-transport answer must say the relevant measurement is missing", row.Reason, want)
				}
			}
			if strings.Contains(row.Reason, "blocked") {
				t.Errorf("the reason %q claims the transport is blocked; an unmeasured adapter supports no such claim", row.Reason)
			}
			if req := requirementFor(t, row.Requires, transport.KindThirdPartyPermission); req.Satisfied {
				t.Errorf("third_party_permission is satisfied although no run measured or supplied it: %+v", req)
			}
			if len(row.Requires) != 1 {
				t.Errorf("the evaluated prerequisite rows are %+v, want exactly the one declared kind", row.Requires)
			} else if row.Requires[0].Detail != declared[0].Detail {
				t.Errorf("the evaluated Detail %q drifted from the declared one %q", row.Requires[0].Detail, declared[0].Detail)
			}
			if !noteStates(row.Notes, transport.KindThirdPartyPermission) {
				t.Errorf("the notes %v do not state the unsatisfied prerequisite, so the row could read as a network fact", row.Notes)
			}
			if !strings.Contains(strings.Join(row.Notes, "\n"), "no measurement") {
				t.Errorf("the notes %v do not state that no measurement of this adapter exists in this slice", row.Notes)
			}
			if i == 0 {
				first = row
			} else if !reflect.DeepEqual(row, first) {
				t.Errorf("the adapter's answer changed with the diagnosis (%+v vs %+v): it must read no finding", row, first)
			}
			for _, target := range []string{hubTarget, cfEdgeRegionOne, publicSSHTarget} {
				if strings.Contains(row.Reason+strings.Join(row.Notes, "\n"), target) {
					t.Errorf("the answer names the target %q, which no measurement of this adapter carried", target)
				}
			}
		})
	}
}

// cfEdgeReachable is a run in which one declared Cloudflare edge endpoint answered on TCP: the
// measured pass CF_EDGE_REACHABLE rests on.
func cfEdgeReachable() probe.Result {
	return result(cfEdgeProbe, probe.ProbeEgress,
		measured("tcp 443 region1", cfEdgeRegionOne, probe.Pass, probe.ReasonOK,
			"dial tcp 198.51.100.10:443: the connection was established"),
	)
}

// cfEdgePartial is a run in which one declared edge endpoint answered and another did not: the
// split CF_EDGE_PARTIAL states.
func cfEdgePartial() probe.Result {
	return result(cfEdgeProbe, probe.ProbeEgress,
		measured("tcp 443 region1", cfEdgeRegionOne, probe.Pass, probe.ReasonOK,
			"dial tcp 198.51.100.10:443: the connection was established"),
		measured("tcp 443 region2", cfEdgeRegionTwo, probe.Fail, probe.ReasonConnRefused,
			"dial tcp 198.51.100.11:443: connect: connection refused"),
	)
}

// cfEdgeUnreachable is a run in which every measured edge endpoint failed: the measured block
// CF_EDGE_UNREACHABLE states.
func cfEdgeUnreachable() probe.Result {
	return result(cfEdgeProbe, probe.ProbeEgress,
		measured("tcp 443 region1", cfEdgeRegionOne, probe.Fail, probe.ReasonConnRefused,
			"dial tcp 198.51.100.10:443: connect: connection refused"),
	)
}

// cfEdgeUnresolved is a run in which the edge attempt produced no answer: the absence
// CF_EDGE_UNRESOLVED states, never a failure.
func cfEdgeUnresolved() probe.Result {
	return result(cfEdgeProbe, probe.ProbeEgress,
		unresolved("tcp 443 region1", cfEdgeRegionOne, probe.ReasonProbeTimeout,
			"dial tcp 198.51.100.10:443: the attempt produced no answer inside the probe budget"),
	)
}

// cfEdgeNotMeasured is a run in which the edge measurement was never made, so the cloudflare.edge
// question matches no clause and the adapter must report an absence.
func cfEdgeNotMeasured() probe.Result {
	return result(cfEdgeProbe, probe.ProbeEgress,
		notMeasured("tcp 443 region1", probe.ReasonCapabilityExcluded,
			"the edge measurement was excluded for this run"),
	)
}

// TestCloudflareTunnelFeasibilityDecisionTable is the adapter's decision table over the four edge
// states and the three absences a diagnosis can carry. Every row asserts the same three
// properties: the transport is never viable in R1a (no input supplies the account prerequisite),
// its reason carries the highest-precedence fact design §3.2 allows, and the answer never states a
// network fact the diagnosis did not carry nor describes the transport as ready to use.
func TestCloudflareTunnelFeasibilityDecisionTable(t *testing.T) {
	cases := []struct {
		name string
		run  []probe.Result
		// finding, when set, is handed to Feasible as a literal diagnosis instead of a run reduced
		// through the reasoning layer, so a caller-constructed absence can be exercised.
		finding *diagnosis.Finding
		// wantReason lists what the reason must state.
		wantReason []string
		// wantNotes lists what the joined notes must state.
		wantNotes []string
		// wantNotSaid lists what neither the reason nor the notes may state.
		wantNotSaid []string
	}{
		{
			name:       "a reachable edge leaves the unmet hostname prerequisite as the reason",
			run:        []probe.Result{cfEdgeReachable()},
			wantReason: []string{"is reachable", cfEdgeRegionOne, "hostname", "not satisfied"},
			wantNotes:  []string{cfEdgeRegionOne, "account prerequisite", "not a network block"},
			wantNotSaid: []string{
				"blocked", "unreachable",
			},
		},
		{
			name: "a partly reachable edge is qualified by the split and decided by the hostname prerequisite",
			run:  []probe.Result{cfEdgePartial()},
			// The split is named from the evidence and is never the rejecting fact: at least one
			// endpoint answered, so the edge half is satisfied and the unmet account prerequisite is
			// what makes the transport not viable.
			wantReason: []string{
				"partly reachable",
				cfEdgeRegionOne,
				cfEdgeRegionTwo,
				"at least one declared endpoint answered",
				", but the transport is not viable: prerequisite hostname is not satisfied",
			},
			wantNotes: []string{cfEdgeRegionOne, cfEdgeRegionTwo, "prerequisite hostname", "account prerequisite", "not a network block"},
			wantNotSaid: []string{
				"blocked", "rejects",
			},
		},
		{
			name:       "an unreachable edge is the measured block that rejects the transport",
			run:        []probe.Result{cfEdgeUnreachable()},
			wantReason: []string{"unreachable", cfEdgeRegionOne, "failed", "hostname"},
			wantNotes:  []string{cfEdgeRegionOne, "prerequisite hostname"},
			wantNotSaid: []string{
				"blocked",
			},
		},
		{
			name:       "an unresolved edge claims neither reachability nor a block",
			run:        []probe.Result{cfEdgeUnresolved()},
			wantReason: []string{"not established", cfEdgeRegionOne, "no answer", "hostname"},
			wantNotes:  []string{cfEdgeRegionOne, "prerequisite hostname"},
			wantNotSaid: []string{
				"unreachable", "blocked",
			},
		},
		{
			name:       "a run that measured no edge is an absence, not a block",
			run:        nil,
			wantReason: []string{"was not made", "cloudflare.edge", "hostname"},
			wantNotes:  []string{"not measured", "prerequisite hostname"},
			wantNotSaid: []string{
				"unreachable", "blocked", cfEdgeRegionOne,
			},
		},
		{
			name:       "a run whose findings are unrelated to the edge is the same absence",
			run:        []probe.Result{hubPassed()},
			wantReason: []string{"was not made", "cloudflare.edge", "hostname"},
			wantNotes:  []string{"not measured", "prerequisite hostname"},
			wantNotSaid: []string{
				"unreachable", "blocked", cfEdgeRegionOne, hubTarget,
			},
		},
		{
			name:       "a not-measured edge result is an absence, not a block",
			run:        []probe.Result{cfEdgeNotMeasured()},
			wantReason: []string{"was not made", "cloudflare.edge", "hostname"},
			wantNotes:  []string{"not measured", "prerequisite hostname"},
			wantNotSaid: []string{
				"unreachable", "blocked", cfEdgeRegionOne,
			},
		},
		{
			name: "an unrecognised rule is an absence, never one of the four states",
			finding: &diagnosis.Finding{
				Question:   "cloudflare.edge",
				Rule:       "CF_EDGE_SOMETHING_NEW",
				Conclusion: "a conclusion this adapter does not know",
				Evidence: []diagnosis.Fact{{
					Probe:       cfEdgeProbe,
					Resolution:  probe.Measured,
					Verdict:     probe.Pass,
					State:       diagnosis.StatePass,
					Observation: measured("tcp 443 region1", cfEdgeRegionOne, probe.Pass, probe.ReasonOK, "dial tcp 198.51.100.10:443: the connection was established"),
				}},
			},
			wantReason: []string{"CF_EDGE_SOMETHING_NEW", "does not recognise", "no edge state"},
			wantNotes:  []string{"prerequisite hostname"},
			wantNotSaid: []string{
				"reachable", "unreachable", "not established", "blocked",
			},
		},
		{
			name: "a recognised rule without evidence names no endpoint and claims no state",
			finding: &diagnosis.Finding{
				Question:   "cloudflare.edge",
				Rule:       "CF_EDGE_REACHABLE",
				Conclusion: "a conclusion whose evidence a caller stripped",
			},
			wantReason: []string{"CF_EDGE_REACHABLE", "no observation"},
			wantNotes:  []string{"prerequisite hostname"},
			wantNotSaid: []string{
				"is reachable from this network", "unreachable", "not established", "blocked", cfEdgeRegionOne,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var d diagnosis.Diagnosis
			if tc.finding != nil {
				d = diagnosis.Diagnosis{Findings: []diagnosis.Finding{*tc.finding}}
			} else {
				d = diagnosis.Diagnose(tc.run)
			}
			registry, rows := registryRows(t, d)
			row := rowFor(t, registry, rows, "cloudflare-tunnel")
			if row.Viable {
				t.Fatal("cloudflare-tunnel is viable: R1a supplies no Cloudflare account fact, so it can never be viable in this slice")
			}
			if row.Reason == "" {
				t.Fatal("cloudflare-tunnel carries no reason for its verdict")
			}
			for _, want := range tc.wantReason {
				if !strings.Contains(row.Reason, want) {
					t.Errorf("the reason %q does not state %q", row.Reason, want)
				}
			}
			notes := strings.Join(row.Notes, "\n")
			for _, want := range tc.wantNotes {
				if !strings.Contains(notes, want) {
					t.Errorf("the notes %q do not state %q", notes, want)
				}
			}
			for _, notSaid := range tc.wantNotSaid {
				if strings.Contains(row.Reason+notes, notSaid) {
					t.Errorf("the answer states %q, which this diagnosis did not carry", notSaid)
				}
			}
			if len(row.Requires) != 2 {
				t.Fatalf("the evaluated requirement rows are %+v, want the two declared rows", row.Requires)
			}
			for _, kind := range []transport.RequirementKind{transport.KindHostname, transport.KindZoneMembership} {
				req := requirementFor(t, row.Requires, kind)
				if req.Satisfied {
					t.Errorf("R1a supplies no account fact, so %s must be unsatisfied: %+v", kind, req)
				}
				if req.Detail == "" {
					t.Errorf("the unsatisfied %s row carries no Detail naming what the user must supply", kind)
				}
				if !noteStates(row.Notes, kind) {
					t.Errorf("the notes %v do not state the unsatisfied %s prerequisite", row.Notes, kind)
				}
			}
			if strings.Contains(row.Reason+notes, "ready") {
				t.Errorf("the answer describes the transport as ready to use: %q", row.Reason+notes)
			}
		})
	}
}

// TestCloudflareReadsTheDeclaredEdgeIDs guards the duplication the vocabulary boundary forces: the
// reasoning layer's hand-named `cloudflare.edge` ids are unexported constants, so the cloudflare
// adapter spells them literally. A rename in the table would silently strand the adapter on its
// unrecognised-rule path, so the literals are asserted to stay declared ids.
func TestCloudflareReadsTheDeclaredEdgeIDs(t *testing.T) {
	declared := diagnosis.AllRuleIDs()
	for _, id := range []string{
		"CF_EDGE_REACHABLE",
		"CF_EDGE_PARTIAL",
		"CF_EDGE_UNREACHABLE",
		"CF_EDGE_UNRESOLVED",
	} {
		if !slices.Contains(declared, id) {
			t.Errorf("the cloudflare adapter switches on %q, which diagnosis.AllRuleIDs() no longer declares", id)
		}
	}
}
