// Package diagnosis owns the reasoning layer of herdr-reach: it turns the probe results
// of one run into the matchable facts a rule table can reason over, and (in later slices
// of the same package) into findings and open questions.
//
// It is deliberately pure. Nothing in this package dials, resolves, reads a file,
// executes a command or touches the measured machine: the measurement vocabulary lives in
// `internal/probe`, and this package only reads what that vocabulary reported. It never
// re-measures to fill a gap and never upgrades an absence into an answer: an observation
// that was not made stays not measured and an attempt that produced nothing stays
// unresolved, all the way into the conclusions that depend on them.
//
// The bridge between the two layers is this file. Everything downstream of it reasons
// over `Fact` values, which carry one observation's own state, so the reasoning layer can
// never match on a probe's aggregate verdict where the probe reported several
// observations (DEV-2, design §7).
package diagnosis

import "github.com/Luisalt20/herdr-reach/internal/probe"

// State is one observation's matchable state: the four values design §3.6 derives a
// probe's rule ids from. It is a separate vocabulary from `probe.Resolution` on purpose —
// a resolution says how far a measurement got, and a state says what the reasoning layer
// may match on, which is the resolution and the verdict together — and the mapping
// between the two lives in exactly one place (StateOf).
//
// The four states are the whole set. A state is never chosen from wording, from a reason
// code, or from a probe's aggregate verdict: it is derived from the observation the probe
// reported.
type State string

const (
	// StatePass is a measured observation whose declared question was answered
	// positively. It is the only state a rule may read as success.
	StatePass State = "PASS"
	// StateFail is a measured observation whose declared question was answered
	// negatively. It is a definite answer, not an absence of one.
	StateFail State = "FAIL"
	// StateUnresolved is an attempt that was made and produced nothing classifiable.
	// It is the absence of an answer, and it can never contribute to a confident
	// conclusion (R-HR-NF-03).
	StateUnresolved State = "UNRESOLVED"
	// StateNotMeasured is an attempt that was never made, because a required input or
	// capability is outside this slice's boundary. Like StateUnresolved it is the
	// absence of an answer, and it is distinguishable from it by a consumer.
	StateNotMeasured State = "NOT_MEASURED"
)

// Fact is one matchable observation: which probe reported it, the exact resolution and
// verdict the measurement layer attached to it, the state those two values map to, and
// the observation itself, so a conclusion can quote the label, target, reason code and
// verbatim detail without re-reading the run's results.
//
// A Fact is built only by the accessors in this file. Its Resolution, Verdict and
// Observation fields are therefore copies of one observation's own values and cannot
// disagree with each other, which is the property the accessor cases assert: there is no
// second opinion about a measurement anywhere in the reasoning layer.
type Fact struct {
	// Probe is the stable probe name the observation belongs to, e.g.
	// "egress.cf.443". Several facts can share one probe — one per observation — and
	// that is the point of DEV-2.
	Probe string
	// Resolution is the observation's own resolution: measured, unresolved or not
	// measured.
	Resolution probe.Resolution
	// Verdict is the observation's own verdict: pass, fail or indeterminate.
	Verdict probe.Verdict
	// State is the matchable state Resolution and Verdict map to.
	State State
	// Observation is the observation this fact is a view of, carried whole so a
	// conclusion can name the label, target, reason code and verbatim detail.
	Observation probe.Observation
}

// allStates is the declaration order, which is also the order design §3.6 derives rule
// ids in and the order AllStates returns.
var allStates = []State{StatePass, StateFail, StateUnresolved, StateNotMeasured}

// AllStates returns the closed set of matchable states, in declaration order. The
// returned slice is a copy, so a caller cannot reorder or shrink the contract; callers
// must not assume otherwise.
//
// It exists so the state vocabulary is enumerable the way the reason-code set is
// (`probe.AllReasonCodes`): a consumer that must cover every state — the rule-id coverage
// case of the later reasoning slices, the payload, the documentation — reads the set
// rather than restating it.
func AllStates() []State {
	states := make([]State, len(allStates))
	copy(states, allStates)
	return states
}

// StateOf maps one observation to its matchable state.
//
// The four cells the measurement vocabulary allows are the four states: a measured
// observation is a pass or a failure, an unresolved observation and a not-measured
// observation are both indeterminate and are told apart by their resolution, which is
// exactly the distinction DEV-2 exists for. A not-measured observation is never a failure
// and never a pass: a measurement that was not made is not evidence for anything.
//
// Any combination the vocabulary does not allow — a definite verdict on a non-measured
// observation, or a measured observation with no answer — is reported as unresolved
// rather than as a pass or a failure. Such a fact cannot have come from the classification
// table of design §5.1, and the honest reading of "this observation cannot be classified"
// is the absence of an answer, never a confident one.
func StateOf(observation probe.Observation) State {
	switch {
	case observation.Resolution == probe.Measured && observation.Verdict == probe.Pass:
		return StatePass
	case observation.Resolution == probe.Measured && observation.Verdict == probe.Fail:
		return StateFail
	case observation.Resolution == probe.Unresolved && observation.Verdict == probe.Indeterminate:
		return StateUnresolved
	case observation.Resolution == probe.NotMeasured && observation.Verdict == probe.Indeterminate:
		return StateNotMeasured
	default:
		return StateUnresolved
	}
}

// Facts returns one fact per observation of every result, in the order the run reported
// them: the results in the order they were handed over, and each result's observations in
// the order the probe reported them.
//
// A result's aggregate verdict is deliberately not consulted. A probe that reported
// several observations measured several things, and its Result.Verdict is the worst of
// them: reading that verdict here would fold a passing region into a sibling's failure or
// a not-measured capability into a neighbour's pass, which is the wrong-classification
// risk DEV-2 removes. A result that carries no observation contributes no fact: the
// accessors report measurements and never invent one.
func Facts(results []probe.Result) []Fact {
	var facts []Fact
	for _, result := range results {
		for _, observation := range result.Observations {
			facts = append(facts, newFact(result.Probe, observation))
		}
	}
	return facts
}

// FactsFor returns the facts of one probe, in the order the probe reported them. It is
// Facts restricted to the observations whose result named that probe, which is how a rule
// reads the states of the probe it needs without scanning unrelated measurements.
//
// A probe the run did not report yields no facts. That is different from a probe that
// reported a not-measured observation, and the two must stay distinguishable: the first is
// a measurement that is missing from the run, the second is a measurement the run made of
// its own inability to measure. Callers that need the first case read the probe registry
// and the run's coverage, not this accessor.
func FactsFor(results []probe.Result, probeName string) []Fact {
	var facts []Fact
	for _, result := range results {
		if result.Probe != probeName {
			continue
		}
		for _, observation := range result.Observations {
			facts = append(facts, newFact(result.Probe, observation))
		}
	}
	return facts
}

// newFact builds the fact for one observation of one probe. It is the only construction
// path, so a fact's resolution, verdict and state cannot drift from the observation it
// describes.
func newFact(probeName string, observation probe.Observation) Fact {
	return Fact{
		Probe:       probeName,
		Resolution:  observation.Resolution,
		Verdict:     observation.Verdict,
		State:       StateOf(observation),
		Observation: observation,
	}
}
