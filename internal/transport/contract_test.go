package transport_test

// This file is the transport layer's contract suite: the type-level shape of the registration and
// feasibility contracts, the loud failure of the members this slice does not implement, and the
// behaviour of the two SSH adapters against real `probe.Result` fixtures.
//
// Every transport is driven through `Feasible`, never through its internals: a test-only
// transport proves that the registration contract needs nothing but `Candidate`, and the loop over
// `Registry()` proves that every registered row explains its verdict. The fixtures are built from
// the measurement layer's exported types and reduced through `probe.Aggregate`, so they describe
// what a probe reported rather than a second opinion about verdicts.

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/transport"
)

// The stable names and targets the fixtures use. They are the run's own vocabulary: the hub
// address a diagnosed run would have been given, the public endpoint the port-versus-protocol
// disambiguator measures, and the sshd paths the local probe reports.
const (
	hubProbe        = "egress.hub.direct"
	hubTarget       = "203.0.113.10:22"
	sshdProbe       = "local.sshd"
	publicSSHProbe  = "egress.ssh.known"
	publicSSHTarget = "github.com:22"
	sshdBinaryPath  = "/usr/sbin/sshd"
	sshdConfigPath  = "/etc/ssh/sshd_config"
)

// TestContractRequirementKindsAreTheClosedSetInDeclarationOrder pins D6's vocabulary: exactly the
// five kinds, in declaration order, each spelled as design D6 spells it, and returned as a copy so
// a caller cannot shrink or reorder the closed set.
func TestContractRequirementKindsAreTheClosedSetInDeclarationOrder(t *testing.T) {
	want := []transport.RequirementKind{
		transport.KindHostname,
		transport.KindZoneMembership,
		transport.KindThirdPartyPermission,
		transport.KindHubAddress,
		transport.KindSSHDEffectiveConfig,
	}
	got := transport.AllRequirementKinds()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllRequirementKinds() = %v, want the closed set in declaration order %v", got, want)
	}
	wantStrings := []string{"hostname", "zone_membership", "third_party_permission", "hub_address", "sshd_effective_config"}
	for i, kind := range got {
		if string(kind) != wantStrings[i] {
			t.Errorf("kind %d = %q, want the D6 spelling %q", i, kind, wantStrings[i])
		}
	}

	// The returned slice is a copy: mutating it cannot change what a later caller reads.
	got[0] = "mutated"
	if again := transport.AllRequirementKinds(); again[0] != transport.KindHostname {
		t.Errorf("AllRequirementKinds() returned an aliased slice: after mutating a copy, index 0 = %q, want %q", again[0], transport.KindHostname)
	}
}

// TestContractViabilityIsAStrictBooleanWithNoThirdValue is PRD §14.5's type-level case: `Viable`
// is a plain bool, `Feasibility` carries exactly the four contract fields, and no second
// viability-like field exists that could encode "sort of viable".
func TestContractViabilityIsAStrictBooleanWithNoThirdValue(t *testing.T) {
	typ := reflect.TypeOf(transport.Feasibility{})

	viable, ok := typ.FieldByName("Viable")
	if !ok {
		t.Fatal("Feasibility has no Viable field")
	}
	if viable.Type.Kind() != reflect.Bool {
		t.Fatalf("Feasibility.Viable is a %s, want a strict bool: PRD §14.5 forbids a third viability value", viable.Type)
	}

	var exported []string
	for i := 0; i < typ.NumField(); i++ {
		if field := typ.Field(i); field.IsExported() {
			exported = append(exported, field.Name)
		}
	}
	wantFields := []string{"Viable", "Reason", "Notes", "Requires"}
	if !slices.Equal(exported, wantFields) {
		t.Fatalf("Feasibility's exported fields are %v, want exactly %v", exported, wantFields)
	}
	for _, name := range exported {
		if name == "Viable" {
			continue
		}
		lower := strings.ToLower(name)
		for _, forbidden := range []string{"viab", "state", "partial", "mode", "confidence"} {
			if strings.Contains(lower, forbidden) {
				t.Errorf("Feasibility carries the viability-like field %q: no second value may encode a partially viable state", name)
			}
		}
	}

	requires, ok := typ.FieldByName("Requires")
	if !ok {
		t.Fatal("Feasibility has no Requires field")
	}
	if want := reflect.TypeOf([]transport.Requirement(nil)); requires.Type != want {
		t.Errorf("Feasibility.Requires is a %s, want %s: the evaluated prerequisite rows travel with the verdict", requires.Type, want)
	}
}

// TestNotImplementedMembersFailLoudlyWithoutAValue is RG-9's case for every registered adapter:
// each plan- and verification-producing member fails with the typed not-implemented error, the
// error names both the member and the owning slice, `errors.Is` matches the sentinel, and no value
// is returned beside it.
func TestNotImplementedMembersFailLoudlyWithoutAValue(t *testing.T) {
	registry := transport.Registry()
	if len(registry) == 0 {
		t.Fatal("the registry is empty, so the case proves nothing")
	}
	for _, candidate := range registry {
		// Candidate is the registration contract, and every R1a entry is a full Transport: the type
		// assertion below is the honesty property this loop now carries, so a future detect-only
		// entry cannot silently join the shipped set without a deliberate decision. PR 14 owns the
		// per-member proof for the adapters it appends; here every shipped entry's plan and
		// verification members fail loudly.
		tr, ok := candidate.(transport.Transport)
		if !ok {
			t.Fatalf("the registered candidate %q does not implement Transport: the registration contract is Candidate and every R1a entry is a full Transport, so a detect-only entry joining the shipped set must be a deliberate decision rather than a silent registry append", candidate.Name())
		}
		t.Run(tr.Name()+"/PlanHub", func(t *testing.T) {
			steps, err := tr.PlanHub(transport.PairingBundle{})
			assertNotImplemented(t, err, "PlanHub", "R3")
			if steps != nil {
				t.Errorf("PlanHub returned %v beside the error; no plan may be returned alongside the not-implemented error", steps)
			}
		})
		t.Run(tr.Name()+"/PlanNode", func(t *testing.T) {
			steps, err := tr.PlanNode(transport.PairingBundle{})
			assertNotImplemented(t, err, "PlanNode", "R3")
			if steps != nil {
				t.Errorf("PlanNode returned %v beside the error; no plan may be returned alongside the not-implemented error", steps)
			}
		})
		t.Run(tr.Name()+"/Verify", func(t *testing.T) {
			results, err := tr.Verify(context.Background(), transport.Handle{})
			assertNotImplemented(t, err, "Verify", "R5/R6")
			if results != nil {
				t.Errorf("Verify returned %v beside the error; no evidence may be returned alongside the not-implemented error", results)
			}
		})
	}
}

// assertNotImplemented asserts the loud-failure contract of one member: a non-nil typed error that
// names the member and its owner slice, unwraps to the sentinel, and matches it under errors.Is.
func assertNotImplemented(t *testing.T, err error, member, owner string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s returned a nil error: a caller could mistake that for a successful result", member)
	}
	if !errors.Is(err, transport.ErrNotImplementedInThisPhase) {
		t.Errorf("errors.Is(err, ErrNotImplementedInThisPhase) = false for %v", err)
	}
	if got := errors.Unwrap(err); got != transport.ErrNotImplementedInThisPhase {
		t.Errorf("errors.Unwrap(%v) = %v, want the sentinel", err, got)
	}
	var typed *transport.NotImplementedError
	if !errors.As(err, &typed) {
		t.Fatalf("the error %v is not the typed NotImplementedError", err)
	}
	if typed.Member != member {
		t.Errorf("the error names member %q, want %q", typed.Member, member)
	}
	if typed.Owner != owner {
		t.Errorf("the error names owner %q, want %q", typed.Owner, owner)
	}
	for _, want := range []string{member, owner} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error text %q does not name %q", err.Error(), want)
		}
	}
}

// candidateOnly implements only the registration contract. It is the proof that a detect-only
// transport — or a test-only one — can be evaluated without carrying the plan and verification
// stubs a full Transport needs, which is why Candidate is the registration contract.
type candidateOnly struct {
	name        string
	requires    []transport.Requirement
	feasibility transport.Feasibility
	// handed records the diagnosis the fixture was given, so a case can prove the evaluation did
	// not mutate what it read.
	handed *diagnosis.Diagnosis
}

func (c *candidateOnly) Name() string { return c.name }

func (c *candidateOnly) Requires() []transport.Requirement {
	return append([]transport.Requirement(nil), c.requires...)
}

func (c *candidateOnly) Feasible(d diagnosis.Diagnosis) transport.Feasibility {
	handed := d
	c.handed = &handed
	return c.feasibility
}

// TestFixtureCandidateOnlyTransportIsEvaluatedWithoutTouchingTheDiagnosis is R-HR-NF-04's
// test-only-transport case at the contract level: a fixture that implements only Candidate appears
// in the produced feasibility list with its stated viability and reason, carries its declared
// requirement list, and changes no diagnosis finding — the diagnosis it was handed is still the
// diagnosis the run produced.
func TestFixtureCandidateOnlyTransportIsEvaluatedWithoutTouchingTheDiagnosis(t *testing.T) {
	d := diagnosis.Diagnose([]probe.Result{hubPassed(), sshdConfigured()})
	before := cloneDiagnosis(d)

	fixture := &candidateOnly{
		name: "candidate-only",
		requires: []transport.Requirement{{
			Kind:   transport.KindHostname,
			Detail: "a hostname in the fixture's account",
		}},
		feasibility: transport.Feasibility{
			Viable: true,
			Reason: "the fixture declares its own viability",
			Notes:  []string{"fixture: declared, not measured"},
			Requires: []transport.Requirement{{
				Kind:      transport.KindHostname,
				Satisfied: true,
				Detail:    "a hostname in the fixture's account",
			}},
		},
	}

	// Evaluate twice through the shipped entry point: once without the fixture (the registered
	// candidates) and once with it. The diagnosis must come out of both evaluations unchanged, and
	// the fixture's row must be the one Evaluate produced — a Candidate-only adapter needs no
	// conversion and nothing but the registration contract to flow through the shipped code path.
	transport.Evaluate(transport.Registry(), d)
	rows := transport.Evaluate([]transport.Candidate{fixture}, d)

	if len(rows) != 1 {
		t.Fatalf("evaluating the fixture produced %d rows, want one", len(rows))
	}
	if !rows[0].Viable || rows[0].Reason != fixture.feasibility.Reason {
		t.Errorf("the fixture's row is %+v, want its stated viability and reason", rows[0])
	}
	for _, declared := range fixture.Requires() {
		found := false
		for _, evaluated := range rows[0].Requires {
			if evaluated.Kind == declared.Kind && evaluated.Detail == declared.Detail {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("the fixture declares requirement %+v, which does not appear in its evaluated rows %+v", declared, rows[0].Requires)
		}
	}

	if !reflect.DeepEqual(d, before) {
		t.Errorf("evaluating the transports changed the diagnosis:\n got %+v\nwant %+v", d, before)
	}
	if fixture.handed == nil {
		t.Fatal("the fixture never received the diagnosis")
	}
	if !reflect.DeepEqual(*fixture.handed, before) {
		t.Errorf("the fixture was handed %+v, want the run's own diagnosis %+v", *fixture.handed, before)
	}
}

// TestRegistryIsDeterministicInDeclarationOrder pins the registration order and the copy: PR 13
// registers exactly the two SSH adapters, direct-ssh before reverse-ssh, PR 14 appends the other
// two, and a caller who mutates the returned slice cannot reorder the registry.
func TestRegistryIsDeterministicInDeclarationOrder(t *testing.T) {
	first := transport.Registry()
	var names []string
	for _, tr := range first {
		names = append(names, tr.Name())
	}
	want := []string{"direct-ssh", "reverse-ssh"}
	if !slices.Equal(names, want) {
		t.Fatalf("Registry() enumerates %v, want PRD §5.2's declaration order for this slice %v", names, want)
	}
	if again := transport.Registry(); !slices.Equal(transportNames(again), want) {
		t.Errorf("a second Registry() enumerates %v, want the same stable order %v", transportNames(again), want)
	}

	first[0] = nil
	if third := transport.Registry(); third[0] == nil || third[0].Name() != "direct-ssh" {
		t.Errorf("mutating the slice Registry() returned changed a later call: index 0 is %v, want direct-ssh", third[0])
	}
}

// TestRegistryRowsExplainEveryVerdict is the loop design §3.2 requires over the four adapters'
// shared contract: every row carries a non-empty reason, a non-viable row never explains nothing,
// a viable row names at least one measured observation in Notes, and every kind Requires() declares
// appears among the evaluated Requires rows.
func TestRegistryRowsExplainEveryVerdict(t *testing.T) {
	cases := []struct {
		name string
		run  []probe.Result
	}{
		{"hub passed and sshd configured", []probe.Result{hubPassed(), sshdConfigured()}},
		{"hub rejected", []probe.Result{hubRejected(), sshdConfigured()}},
		{"hub unanswered", []probe.Result{hubUnanswered(), sshdConfigured()}},
		{"hub not measured", []probe.Result{hubNotMeasured(), sshdConfigured()}},
		{"sshd divergent", []probe.Result{hubPassed(), sshdDivergent()}},
		{"sshd absent", []probe.Result{hubPassed(), sshdAbsent()}},
		{"sshd not measured", []probe.Result{hubPassed(), sshdNotMeasured()}},
		{"nothing measured", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := diagnosis.Diagnose(tc.run)
			registry, rows := registryRows(t, d)
			for i, tr := range registry {
				row := rows[i]
				if row.Reason == "" {
					t.Errorf("%s carries no reason (viable=%t): every transport explains its verdict in both cases", tr.Name(), row.Viable)
				}
				if row.Viable && !namesAMeasuredObservation(row.Notes, d) {
					t.Errorf("%s is viable with notes %v, which name no measured observation", tr.Name(), row.Notes)
				}
				for _, declared := range tr.Requires() {
					if !hasRequirementKind(row.Requires, declared.Kind) {
						t.Errorf("%s declares requirement kind %q, which does not appear among the evaluated rows %+v", tr.Name(), declared.Kind, row.Requires)
					}
				}
			}
		})
	}
}

// TestDirectSSHDecidesOnTheHubMeasurement is the first adapter's decision table: a pass is viable
// and names the accepted address, a fail and an unresolved attempt are not viable and name what was
// measured, and a missing measurement is not viable and claims no block.
func TestDirectSSHDecidesOnTheHubMeasurement(t *testing.T) {
	t.Run("a finding without an observation is not read as a measurement", func(t *testing.T) {
		// A caller can hand Feasible any Diagnosis value. A finding that fired but carries no
		// observation must not be read as a measured target: the adapter names the rule id it saw
		// and invents no target, no port and no block.
		d := diagnosis.Diagnosis{Findings: []diagnosis.Finding{{
			Question:   "hub.reachability",
			Rule:       "EGRESS_HUB_DIRECT_FAIL",
			Conclusion: "a conclusion whose evidence a caller stripped",
		}}}
		registry, rows := registryRows(t, d)
		for _, name := range []string{"direct-ssh", "reverse-ssh"} {
			row := rowFor(t, registry, rows, name)
			if row.Viable {
				t.Errorf("%s is viable on a finding that carries no observation", name)
			}
			if !strings.Contains(row.Reason, "EGRESS_HUB_DIRECT_FAIL") {
				t.Errorf("%s reason %q does not name the rule it could not read", name, row.Reason)
			}
			for _, invented := range []string{"203.0.113.10", "(port "} {
				if strings.Contains(row.Reason, invented) {
					t.Errorf("%s reason %q invents %q for an observation the diagnosis did not carry", name, row.Reason, invented)
				}
			}
		}
	})
	// The two defensive absence branches take no observation at all, and they read differently on
	// purpose: a measurement that was never made leaves hub_address unsatisfied and names the input
	// the user must supply, while an attempt that was made keeps the row satisfied because the
	// address was supplied — the absence is the answer, not a missing input. The rule-naming
	// fallback differs too: the unresolved branch renders the rule it could not read, and the
	// not-measured fallback deliberately does not, because with no evidence there is nothing to key
	// its reason on. Each case asserts its own branch rather than a shared shape, so a mutation that
	// turns either branch into a block claim fails here.
	evidenceLessAbsences := []struct {
		name string
		rule string
		// wantSaid lists what the reason must state about the absence.
		wantSaid []string
		// wantRuleNamed reports whether this branch's defensive reason names the rule it could not
		// read. The not-measured fallback deliberately does not, so no rule-id assertion runs for it.
		wantRuleNamed bool
		// wantHubSatisfied is the hub_address row of this branch: an attempt that was made keeps the
		// address supplied, a measurement that was never made does not.
		wantHubSatisfied bool
	}{
		{
			name:             "the not-measured rule without evidence claims no block and no supplied hub",
			rule:             "EGRESS_HUB_DIRECT_NOT_MEASURED",
			wantSaid:         []string{"was not made", "--hub"},
			wantRuleNamed:    false,
			wantHubSatisfied: false,
		},
		{
			name:             "the unresolved rule without evidence claims no block and keeps the supplied hub",
			rule:             "EGRESS_HUB_DIRECT_UNRESOLVED",
			wantSaid:         []string{"produced no answer"},
			wantRuleNamed:    true,
			wantHubSatisfied: true,
		},
	}
	for _, tc := range evidenceLessAbsences {
		t.Run(tc.name, func(t *testing.T) {
			d := diagnosis.Diagnosis{Findings: []diagnosis.Finding{{
				Question:   "hub.reachability",
				Rule:       tc.rule,
				Conclusion: "a conclusion whose evidence a caller stripped",
			}}}
			registry, rows := registryRows(t, d)
			for _, name := range []string{"direct-ssh", "reverse-ssh"} {
				row := rowFor(t, registry, rows, name)
				if row.Viable {
					t.Errorf("%s is viable on the %s finding that carries no observation", name, tc.rule)
				}
				for _, want := range tc.wantSaid {
					if !strings.Contains(row.Reason, want) {
						t.Errorf("%s reason %q does not state %q for the %s absence", name, row.Reason, want, tc.rule)
					}
				}
				if tc.wantRuleNamed && !strings.Contains(row.Reason, tc.rule) {
					t.Errorf("%s reason %q does not name the rule it could not read", name, row.Reason)
				}
				if strings.Contains(row.Reason, "blocked") {
					t.Errorf("%s reason %q claims a block for an absence", name, row.Reason)
				}
				for _, invented := range []string{"203.0.113.10", "(port "} {
					if strings.Contains(row.Reason, invented) {
						t.Errorf("%s reason %q invents %q for an observation the diagnosis did not carry", name, row.Reason, invented)
					}
				}
				req := requirementFor(t, row.Requires, transport.KindHubAddress)
				if req.Satisfied != tc.wantHubSatisfied {
					t.Errorf("%s hub_address Satisfied = %t, want %t for the %s absence: an attempted address is supplied while a measurement that was never made is not", name, req.Satisfied, tc.wantHubSatisfied, tc.rule)
				}
				if req.Detail == "" {
					t.Errorf("%s hub_address carries an empty Detail, so the row cannot explain its satisfaction state", name)
				}
			}
		})
	}
	t.Run("a hub pass is viable and names the measured observation", func(t *testing.T) {
		d := diagnosis.Diagnose([]probe.Result{hubPassed()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "direct-ssh")
		if !row.Viable {
			t.Fatalf("direct-ssh is not viable on a hub pass: %s", row.Reason)
		}
		for _, want := range []string{"203.0.113.10:22", "accepted a connection"} {
			if !strings.Contains(row.Reason, want) {
				t.Errorf("the viable reason %q does not state %q", row.Reason, want)
			}
		}
		if !noteNames(row.Notes, hubTarget) {
			t.Errorf("the notes %v name no measured observation with its target", row.Notes)
		}
		if req := requirementFor(t, row.Requires, transport.KindHubAddress); !req.Satisfied {
			t.Errorf("hub_address is unsatisfied on a measured pass: %+v", req)
		}
	})

	t.Run("a hub fail is not viable and names the target and port", func(t *testing.T) {
		d := diagnosis.Diagnose([]probe.Result{hubRejected()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "direct-ssh")
		if row.Viable {
			t.Fatal("direct-ssh is viable on a measured hub failure")
		}
		for _, want := range []string{"203.0.113.10", "22", "EGRESS_HUB_DIRECT_FAIL"} {
			if !strings.Contains(row.Reason, want) {
				t.Errorf("the rejection %q does not name %q", row.Reason, want)
			}
		}
		if req := requirementFor(t, row.Requires, transport.KindHubAddress); !req.Satisfied {
			t.Errorf("hub_address is unsatisfied on a measured block: a block is a network fact, not a missing input (%+v)", req)
		}
	})

	t.Run("a hub unresolved attempt is not viable and claims no block", func(t *testing.T) {
		d := diagnosis.Diagnose([]probe.Result{hubUnanswered()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "direct-ssh")
		if row.Viable {
			t.Fatal("direct-ssh is viable on an unanswered hub attempt")
		}
		if !strings.Contains(row.Reason, "no answer") {
			t.Errorf("the reason %q does not state that the attempt produced no answer", row.Reason)
		}
		if strings.Contains(row.Reason, "blocked") {
			t.Errorf("the reason %q asserts a block the unanswered attempt did not measure", row.Reason)
		}
		if req := requirementFor(t, row.Requires, transport.KindHubAddress); !req.Satisfied {
			t.Errorf("hub_address is unsatisfied after an attempted measurement: %+v", req)
		}
	})
}

// TestRegistryUnmeasuredHubIsHonestAboutTheGap is R-HR-06's unmeasured scenario for both SSH
// adapters, in the two forms a run can produce: a not-measured hub observation carrying its own
// reason code, and a partial run that reported nothing for the hub probe at all.
func TestRegistryUnmeasuredHubIsHonestAboutTheGap(t *testing.T) {
	cases := []struct {
		name string
		run  []probe.Result
	}{
		{"a not-measured hub observation", []probe.Result{hubNotMeasured()}},
		{"no hub result at all", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := diagnosis.Diagnose(tc.run)
			registry, rows := registryRows(t, d)
			for _, name := range []string{"direct-ssh", "reverse-ssh"} {
				row := rowFor(t, registry, rows, name)
				if row.Viable {
					t.Errorf("%s is viable although the hub was never measured", name)
				}
				if row.Reason == "" {
					t.Errorf("%s carries no reason for an unmeasured dependency", name)
				}
				if strings.Contains(row.Reason, "blocked") {
					t.Errorf("%s reason %q claims the transport is blocked; an unmeasured hub supports no such claim", name, row.Reason)
				}
				if strings.Contains(row.Reason, "203.0.113.10") {
					t.Errorf("%s reason %q names a hub target no measurement carried", name, row.Reason)
				}
				req := requirementFor(t, row.Requires, transport.KindHubAddress)
				if req.Satisfied {
					t.Errorf("%s reports hub_address satisfied although the hub was never measured: %+v", name, req)
				}
				if req.Detail == "" {
					t.Errorf("%s reports an unsatisfied hub_address without stating what the user must supply", name)
				}
			}

			// The not-measured observation's own reason code and the input it names reach the
			// reason structurally, from the evidence.
			if len(tc.run) > 0 {
				direct := rowFor(t, registry, rows, "direct-ssh")
				for _, want := range []string{"input_missing_hub", "--hub"} {
					if !strings.Contains(direct.Reason, want) {
						t.Errorf("the reason %q does not name %q, which the diagnosis's evidence carries", direct.Reason, want)
					}
				}
			}
		})
	}
}

// TestReverseSSHNeedsTheHubPassAndAConfiguredSSHD is the second adapter's decision table, including
// the reason precedence design §3.2 fixes: a measured hub block, then a measured sshd rejection,
// then an unanswered hub attempt, then an unmet requirement.
func TestReverseSSHNeedsTheHubPassAndAConfiguredSSHD(t *testing.T) {
	t.Run("a hub pass with a configured sshd is viable", func(t *testing.T) {
		d := diagnosis.Diagnose([]probe.Result{hubPassed(), sshdConfigured()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "reverse-ssh")
		if !row.Viable {
			t.Fatalf("reverse-ssh is not viable with a hub pass and a configured sshd: %s", row.Reason)
		}
		for _, want := range []string{hubTarget} {
			if !strings.Contains(strings.Join(row.Notes, "\n"), want) {
				t.Errorf("the notes %v do not name the measured hub observation %q", row.Notes, want)
			}
		}
		for _, kind := range []transport.RequirementKind{transport.KindHubAddress, transport.KindSSHDEffectiveConfig} {
			if req := requirementFor(t, row.Requires, kind); !req.Satisfied {
				t.Errorf("the %s requirement is unsatisfied on a viable reverse-ssh: %+v", kind, req)
			}
		}
	})

	t.Run("a hub pass with an unmeasured sshd is not viable", func(t *testing.T) {
		d := diagnosis.Diagnose([]probe.Result{hubPassed(), sshdNotMeasured()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "reverse-ssh")
		if row.Viable {
			t.Fatal("reverse-ssh is viable without a measured sshd")
		}
		if !strings.Contains(row.Reason, "capability_excluded") {
			t.Errorf("the reason %q does not name the excluded capability's own reason code", row.Reason)
		}
		req := requirementFor(t, row.Requires, transport.KindSSHDEffectiveConfig)
		if req.Satisfied {
			t.Errorf("sshd_effective_config is satisfied although the measurement was excluded: %+v", req)
		}
		if !noteStates(row.Notes, transport.KindSSHDEffectiveConfig) {
			t.Errorf("the notes %v do not state the unmet prerequisite, so the row reads as a network block", row.Notes)
		}
	})

	t.Run("a hub pass with no sshd result at all is not viable and claims no block", func(t *testing.T) {
		// A partial run that reported nothing for the sshd probe: the missing measurement is an
		// absence, never an absent sshd and never a network block.
		d := diagnosis.Diagnose([]probe.Result{hubPassed()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "reverse-ssh")
		if row.Viable {
			t.Fatal("reverse-ssh is viable without any sshd measurement")
		}
		if !strings.Contains(row.Reason, "never measured") {
			t.Errorf("the reason %q does not state that the effective configuration was never measured", row.Reason)
		}
		if strings.Contains(row.Reason, "blocked") {
			t.Errorf("the reason %q asserts a block for a measurement the run never made", row.Reason)
		}
		if req := requirementFor(t, row.Requires, transport.KindSSHDEffectiveConfig); req.Satisfied {
			t.Errorf("sshd_effective_config is satisfied without a measurement: %+v", req)
		}
	})

	t.Run("a divergent sshd is a measured rejection, never success", func(t *testing.T) {
		d := diagnosis.Diagnose([]probe.Result{hubPassed(), sshdDivergent()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "reverse-ssh")
		if row.Viable {
			t.Fatal("reverse-ssh is viable with a divergent sshd configuration")
		}
		for _, want := range []string{"SSHD_PRESENT_CONFIG_DIVERGENT", "not the written configuration"} {
			if !strings.Contains(row.Reason, want) {
				t.Errorf("the reason %q does not state %q", row.Reason, want)
			}
		}
		if strings.Contains(row.Reason, "SSHD_PRESENT_CONFIGURED") {
			t.Errorf("the reason %q presents a divergent configuration as the configured conclusion", row.Reason)
		}
		if req := requirementFor(t, row.Requires, transport.KindSSHDEffectiveConfig); req.Satisfied {
			t.Errorf("sshd_effective_config is satisfied although the configuration diverges: %+v", req)
		}
	})

	t.Run("a measured sshd rejection outranks an unanswered hub attempt", func(t *testing.T) {
		d := diagnosis.Diagnose([]probe.Result{hubUnanswered(), sshdAbsent()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "reverse-ssh")
		for _, want := range []string{"SSHD_ABSENT", "sshd_absent"} {
			if !strings.Contains(row.Reason, want) {
				t.Errorf("the reason %q does not name the measured sshd rejection %q", row.Reason, want)
			}
		}
		if strings.Contains(row.Reason, "EGRESS_HUB_DIRECT_UNRESOLVED") {
			t.Errorf("the reason %q rests on the unanswered hub attempt; a measured rejection is the definite answer (design §3.2 precedence 1)", row.Reason)
		}
	})

	t.Run("a measured hub block outranks a measured sshd rejection", func(t *testing.T) {
		d := diagnosis.Diagnose([]probe.Result{hubRejected(), sshdDivergent()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "reverse-ssh")
		if !strings.Contains(row.Reason, "EGRESS_HUB_DIRECT_FAIL") {
			t.Errorf("the reason %q does not name the hub block, the first measured rejection in precedence order", row.Reason)
		}
	})

	t.Run("an unanswered hub attempt outranks an unmet sshd requirement", func(t *testing.T) {
		d := diagnosis.Diagnose([]probe.Result{hubUnanswered(), sshdNotMeasured()})
		registry, rows := registryRows(t, d)
		row := rowFor(t, registry, rows, "reverse-ssh")
		if !strings.Contains(row.Reason, "EGRESS_HUB_DIRECT_UNRESOLVED") {
			t.Errorf("the reason %q does not name the unresolved dependency, which outranks an unmet requirement (design §3.2 precedence 2)", row.Reason)
		}
	})
}

// TestRegistryRejectionNamesTheBlockedHubTargetAndPort is R-HR-06's matrix replay: the public SSH
// signal passes and the hub address is blocked, so both SSH transports must reject naming the
// measured hub target and its port rather than a network fact the diagnosis did not carry.
func TestRegistryRejectionNamesTheBlockedHubTargetAndPort(t *testing.T) {
	d := diagnosis.Diagnose([]probe.Result{publicSSHPassed(), hubRejected()})
	registry, rows := registryRows(t, d)
	for _, name := range []string{"direct-ssh", "reverse-ssh"} {
		row := rowFor(t, registry, rows, name)
		if row.Viable {
			t.Errorf("%s is viable on a run whose hub address was rejected", name)
		}
		for _, want := range []string{"203.0.113.10", "22", "EGRESS_HUB_DIRECT_FAIL"} {
			if !strings.Contains(row.Reason, want) {
				t.Errorf("%s reason %q does not name the measured observation (%q): a rejection must name the blocked hub target and its port", name, row.Reason, want)
			}
		}
		if req := requirementFor(t, row.Requires, transport.KindHubAddress); !req.Satisfied {
			t.Errorf("%s reports hub_address unsatisfied on a measured block: the block is a network fact, not a missing input", name)
		}
	}
}

// TestReverseSSHReadsTheDeclaredSSHDIDs guards the one duplication the vocabulary boundary forces:
// the reasoning layer's hand-named `local.sshd` ids are unexported constants, so the reverse
// adapter spells them literally. A rename in the table would silently strand the adapter on its
// unrecognised-rule path, so the literals are asserted to stay declared ids.
func TestReverseSSHReadsTheDeclaredSSHDIDs(t *testing.T) {
	declared := diagnosis.AllRuleIDs()
	for _, id := range []string{
		"SSHD_PRESENT_CONFIGURED",
		"SSHD_PRESENT_CONFIG_DIVERGENT",
		"SSHD_ABSENT",
		"SSHD_EFFECTIVE_CONFIG_NOT_MEASURED",
	} {
		if !slices.Contains(declared, id) {
			t.Errorf("the reverse adapter switches on %q, which diagnosis.AllRuleIDs() no longer declares", id)
		}
	}
}

// registryRows evaluates the registered candidates and returns the registry beside its rows, so a
// case never looks a row up by guessing an index.
func registryRows(t *testing.T, d diagnosis.Diagnosis) ([]transport.Candidate, []transport.Feasibility) {
	t.Helper()
	registry := transport.Registry()
	rows := transport.Evaluate(registry, d)
	if len(rows) != len(registry) {
		t.Fatalf("Evaluate returned %d rows for %d registered candidates: one row per candidate, in registry order", len(rows), len(registry))
	}
	return registry, rows
}

// rowFor returns the evaluated row of one candidate by name, failing the case when the candidate is
// not registered.
func rowFor(t *testing.T, registry []transport.Candidate, rows []transport.Feasibility, name string) transport.Feasibility {
	t.Helper()
	for i, tr := range registry {
		if tr.Name() == name {
			return rows[i]
		}
	}
	t.Fatalf("no transport named %q is registered", name)
	return transport.Feasibility{}
}

// requirementFor returns the evaluated requirement row of one kind, failing the case when the
// adapter declared the kind but returned no row for it.
func requirementFor(t *testing.T, rows []transport.Requirement, kind transport.RequirementKind) transport.Requirement {
	t.Helper()
	for _, row := range rows {
		if row.Kind == kind {
			return row
		}
	}
	t.Fatalf("no evaluated requirement row of kind %q among %+v", kind, rows)
	return transport.Requirement{}
}

// hasRequirementKind reports whether the evaluated rows carry one declared kind.
func hasRequirementKind(rows []transport.Requirement, kind transport.RequirementKind) bool {
	for _, row := range rows {
		if row.Kind == kind {
			return true
		}
	}
	return false
}

// noteNames reports whether at least one note quotes one measured observation, identified by the
// observation's own target.
func noteNames(notes []string, target string) bool {
	for _, note := range notes {
		if strings.Contains(note, target) {
			return true
		}
	}
	return false
}

// noteStates reports whether at least one note states one requirement kind, so an unsatisfied row
// is not left to read as "the network blocks this".
func noteStates(notes []string, kind transport.RequirementKind) bool {
	for _, note := range notes {
		if strings.Contains(note, string(kind)) {
			return true
		}
	}
	return false
}

// namesAMeasuredObservation reports whether at least one note quotes an observation the diagnosis
// actually measured: the measured fact's target, or its label when it named no target. The check
// reads the diagnosis rather than the adapter's wording, so a viable transport cannot pass by
// inventing an observation.
func namesAMeasuredObservation(notes []string, d diagnosis.Diagnosis) bool {
	for _, finding := range d.Findings {
		for _, fact := range finding.Evidence {
			if fact.Resolution != probe.Measured {
				continue
			}
			name := fact.Observation.Target
			if name == "" {
				name = fact.Observation.Label
			}
			if name == "" {
				continue
			}
			if noteNames(notes, name) {
				return true
			}
		}
	}
	return false
}

// transportNames names the candidates of a registry slice, for order assertions.
func transportNames(registry []transport.Candidate) []string {
	names := make([]string, 0, len(registry))
	for _, tr := range registry {
		names = append(names, tr.Name())
	}
	return names
}

// cloneDiagnosis deep-copies a diagnosis so a case can compare what a transport read against what
// the run produced, slices included.
func cloneDiagnosis(d diagnosis.Diagnosis) diagnosis.Diagnosis {
	var out diagnosis.Diagnosis
	for _, finding := range d.Findings {
		copied := finding
		copied.DependsOn = append([]string(nil), finding.DependsOn...)
		copied.Evidence = append([]diagnosis.Fact(nil), finding.Evidence...)
		out.Findings = append(out.Findings, copied)
	}
	for _, open := range d.OpenQuestions {
		copied := open
		copied.NeededStates = append([]string(nil), open.NeededStates...)
		out.OpenQuestions = append(out.OpenQuestions, copied)
	}
	return out
}

// measured builds one measured observation a fixture reports.
func measured(label, target string, verdict probe.Verdict, reason probe.ReasonCode, detail string) probe.Observation {
	return probe.Observation{
		Label:      label,
		Target:     target,
		Resolution: probe.Measured,
		Verdict:    verdict,
		Reason:     reason,
		Detail:     detail,
	}
}

// unresolved builds one attempted-but-unanswered observation a fixture reports.
func unresolved(label, target string, reason probe.ReasonCode, detail string) probe.Observation {
	return probe.Observation{
		Label:      label,
		Target:     target,
		Resolution: probe.Unresolved,
		Verdict:    probe.Indeterminate,
		Reason:     reason,
		Detail:     detail,
	}
}

// notMeasured builds one never-attempted observation a fixture reports. A not-measured observation
// names no target, because nothing was attempted.
func notMeasured(label string, reason probe.ReasonCode, detail string) probe.Observation {
	return probe.Observation{
		Label:      label,
		Resolution: probe.NotMeasured,
		Verdict:    probe.Indeterminate,
		Reason:     reason,
		Detail:     detail,
	}
}

// result builds one probe result from its observations, reducing the verdict and reason through
// the measurement layer's own Aggregate, so a fixture is a run in miniature rather than a second
// opinion about verdicts.
func result(name string, kind probe.ProbeKind, observations ...probe.Observation) probe.Result {
	verdict, reason := probe.Aggregate(observations)
	return probe.Result{
		Probe:        name,
		Kind:         kind,
		Verdict:      verdict,
		Reason:       reason,
		Observations: observations,
	}
}

// hubPassed is a hub-directed measurement whose declared address accepted a connection.
func hubPassed() probe.Result {
	return result(hubProbe, probe.ProbeEgress,
		measured("tcp 22", hubTarget, probe.Pass, probe.ReasonOK,
			"dial tcp 203.0.113.10:22: the connection was established and closed without reading or writing"))
}

// hubRejected is a hub-directed measurement whose declared address refused the connection.
func hubRejected() probe.Result {
	return result(hubProbe, probe.ProbeEgress,
		measured("tcp 22", hubTarget, probe.Fail, probe.ReasonConnRefused,
			"dial tcp 203.0.113.10:22: connect: connection refused"))
}

// hubUnanswered is a hub-directed attempt that produced no answer: the address was supplied and
// attempted, and the result is an absence, never a block.
func hubUnanswered() probe.Result {
	return result(hubProbe, probe.ProbeEgress,
		unresolved("tcp 22", hubTarget, probe.ReasonDNSUnresolved,
			"dial tcp 203.0.113.10:22: the resolver produced no answer inside the budget"))
}

// hubNotMeasured is the default live case: no hub address was supplied, so the measurement was
// never attempted.
func hubNotMeasured() probe.Result {
	return result(hubProbe, probe.ProbeEgress,
		notMeasured("hub target", probe.ReasonInputMissingHub,
			"no hub address was supplied for this run, so the hub measurement was not made"))
}

// publicSSHPassed is PRD §1.1's disambiguating signal: public SSH answers, so the SSH protocol is
// not blocked and a destination-level rejection is separable from a protocol-level one.
func publicSSHPassed() probe.Result {
	return result(publicSSHProbe, probe.ProbeEgress,
		measured("tcp 22 ssh banner", publicSSHTarget, probe.Pass, probe.ReasonOK,
			"tcp github.com:22: an SSH identification string answered"))
}

// sshdConfigured is an sshd binary present beside an effective configuration that agrees with the
// written one.
func sshdConfigured() probe.Result {
	return result(sshdProbe, probe.ProbeLocal,
		measured("binary present", sshdBinaryPath, probe.Pass, probe.ReasonOK,
			"the sshd binary is present at /usr/sbin/sshd"),
		measured("effective config", sshdConfigPath, probe.Pass, probe.ReasonOK,
			"the written configuration agrees with the configuration in force"),
	)
}

// sshdDivergent is a written configuration that is not the configuration in force.
func sshdDivergent() probe.Result {
	return result(sshdProbe, probe.ProbeLocal,
		measured("binary present", sshdBinaryPath, probe.Pass, probe.ReasonOK,
			"the sshd binary is present at /usr/sbin/sshd"),
		measured("effective config", sshdConfigPath, probe.Fail, probe.ReasonSSHDConfigDivergence,
			"the written configuration at /etc/ssh/sshd_config is not the configuration in force"),
	)
}

// sshdAbsent is an sshd binary absent at the documented path.
func sshdAbsent() probe.Result {
	return result(sshdProbe, probe.ProbeLocal,
		measured("binary present", sshdBinaryPath, probe.Fail, probe.ReasonSSHDAbsent,
			"the sshd binary is not present at /usr/sbin/sshd"),
	)
}

// sshdNotMeasured is the zero-execution boundary's default: the effective configuration was
// excluded by this slice, so nothing is claimed about the configuration in force.
func sshdNotMeasured() probe.Result {
	return result(sshdProbe, probe.ProbeLocal,
		notMeasured("effective config", probe.ReasonCapabilityExcluded,
			"sshd -T: no command runner is injected for this run"),
	)
}
