package transport_test

// This file is NF-04's test-only-transport proof (design §6.4): a transport implementing only
// `Candidate` is registered, evaluated through the shipped `Evaluate`, and appears in the produced
// list with its stated viability and reason, with its declared requirement set, and with every
// diagnosis finding byte-identical to the run without it. The fixture and the case moved here from
// `contract_test.go`, where they lived beside the contract-shape cases; `contract_test.go` keeps
// its remaining assertions.

import (
	"reflect"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/transport"
)

// testTunnel implements only the registration contract. It is the proof that a detect-only
// transport — or a test-only one — can be evaluated without carrying the plan and verification
// stubs a full Transport needs, which is why Candidate is the registration contract.
type testTunnel struct {
	name        string
	requires    []transport.Requirement
	feasibility transport.Feasibility
	// handed records the diagnosis the fixture was given, so a case can prove the evaluation did
	// not mutate what it read.
	handed *diagnosis.Diagnosis
}

func (c *testTunnel) Name() string { return c.name }

func (c *testTunnel) Requires() []transport.Requirement {
	return append([]transport.Requirement(nil), c.requires...)
}

func (c *testTunnel) Feasible(d diagnosis.Diagnosis) transport.Feasibility {
	handed := d
	c.handed = &handed
	return c.feasibility
}

// TestTestTransportIsEvaluatedWithoutTouchingTheDiagnosis is R-HR-NF-04's test-only-transport case
// at full strength: the diagnosis is built once, the fixture is registered beside the shipped four
// and flows through the same shipped entry point, its row carries its stated viability and reason
// and its declared requirement set, every shipped row is unchanged by its presence, and every
// diagnosis finding is byte-identical to the run without it — both the diagnosis value itself and
// the value handed to the adapter.
func TestTestTransportIsEvaluatedWithoutTouchingTheDiagnosis(t *testing.T) {
	// The run is diagnosed once and compared against a deep copy, so every comparison reads the
	// same run instead of a re-diagnosis that could hide a mutation.
	d := diagnosis.Diagnose([]probe.Result{hubPassed(), sshdConfigured()})
	before := cloneDiagnosis(d)

	fixture := &testTunnel{
		name: "test-tunnel",
		requires: []transport.Requirement{{
			Kind:   transport.KindHostname,
			Detail: "a hostname the test-only transport declares",
		}},
		feasibility: transport.Feasibility{
			Viable: true,
			Reason: "the test-only transport states its own viability",
			Notes:  []string{"test-only: declared, not measured"},
			Requires: []transport.Requirement{{
				Kind:      transport.KindHostname,
				Satisfied: true,
				Detail:    "a hostname the test-only transport declares",
			}},
		},
	}

	// Evaluate the shipped registry without the fixture first: that is the run the diagnosis must
	// come out of unchanged.
	without := transport.Evaluate(transport.Registry(), d)
	if !reflect.DeepEqual(d, before) {
		t.Fatalf("evaluating the shipped registry changed the diagnosis:\n got %+v\nwant %+v", d, before)
	}

	// Then evaluate the same run with the fixture registered beside the shipped four, through the
	// same shipped entry point and with no conversion: a Candidate-only adapter needs nothing but
	// the registration contract to flow through the shipped code path.
	combined := append(append([]transport.Candidate(nil), transport.Registry()...), fixture)
	rows := transport.Evaluate(combined, d)
	if len(rows) != len(combined) {
		t.Fatalf("Evaluate returned %d rows for %d registered candidates: one row per candidate, in registry order", len(rows), len(combined))
	}
	if !reflect.DeepEqual(rows[:len(without)], without) {
		t.Errorf("adding the test-only transport changed the shipped rows:\n got %+v\nwant %+v", rows[:len(without)], without)
	}

	// The fixture's row is in the produced list with its stated viability and reason.
	fixtureRow := rows[len(rows)-1]
	if fixtureRow.Viable != fixture.feasibility.Viable || fixtureRow.Reason != fixture.feasibility.Reason {
		t.Errorf("the test-only transport's row is %+v, want its stated viability %t and reason %q", fixtureRow, fixture.feasibility.Viable, fixture.feasibility.Reason)
	}
	// Its declared requirement set survived evaluation.
	for _, declared := range fixture.Requires() {
		if !hasDeclaredRequirement(fixtureRow.Requires, declared) {
			t.Errorf("the fixture declares requirement %+v, which does not appear in its evaluated rows %+v", declared, fixtureRow.Requires)
		}
	}

	// Every diagnosis finding is byte-identical to the run without it, and the value handed to the
	// adapter is that same diagnosis.
	if !reflect.DeepEqual(d, before) {
		t.Errorf("evaluating the test-only transport changed the diagnosis:\n got %+v\nwant %+v", d, before)
	}
	if fixture.handed == nil {
		t.Fatal("the test-only transport never received the diagnosis")
	}
	if !reflect.DeepEqual(*fixture.handed, before) {
		t.Errorf("the test-only transport was handed %+v, want the run's own diagnosis %+v", *fixture.handed, before)
	}

	// A registry of the fixture alone still flows through the shipped entry point: Candidate is all
	// the registration contract requires.
	solo := transport.Evaluate([]transport.Candidate{fixture}, d)
	if len(solo) != 1 || solo[0].Reason != fixture.feasibility.Reason {
		t.Errorf("a Candidate-only registry produced %+v, want the fixture's one stated row", solo)
	}
}

// hasDeclaredRequirement reports whether one declared requirement row appears in the evaluated
// rows, by kind and by Detail: a declaration that loses its Detail is not the declared row.
func hasDeclaredRequirement(rows []transport.Requirement, declared transport.Requirement) bool {
	for _, row := range rows {
		if row.Kind == declared.Kind && row.Detail == declared.Detail {
			return true
		}
	}
	return false
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
