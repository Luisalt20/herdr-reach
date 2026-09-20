package transport

// This file is the ordered registry of the V1 transports and the pure evaluation entry point of
// design §4's data flow (`transport.Evaluate(transport.Registry(), diagnosis)`).
//
// The order is declaration order and it is deterministic: PRD §5.2 lists direct-ssh, reverse-ssh,
// cloudflare-tunnel then tailscale, this slice registers the two SSH adapters in that order, and
// PR 14 appends the other two in the same list, closing the set at exactly four (R-HR-06). The
// order never depends on map iteration or on anything a run measured, so two runs over the same
// registry produce the same rows in the same positions.
//
// The set is typed as the registration contract, not as the full one (design D7). Every entry this
// slice registers is a full Transport, but the registry is stored as []Candidate because Candidate
// is the contract a transport registers under: a detect-only adapter — or the spec's test-only
// transport in design §6.4 — needs nothing but Name, Requires and Feasible, with no plan or
// verification stub to carry for members the product has not defined. A caller that needs the plan
// members of a registered adapter asserts it to Transport, and the contract suite asserts every
// shipped R1a entry satisfies that assertion, so a detect-only entry joining the shipped set is a
// deliberate decision rather than a silent append.
//
// Evaluate measures nothing. It calls Feasible once per registered candidate and preserves the
// registry order; every decision, reason, requirement and note belongs to the adapters.

import "github.com/Luisalt20/herdr-reach/internal/diagnosis"

// registry is the ordered transport registry. It is stored as []Candidate because Candidate is the
// registration contract (design D7): the shipped entries are full Transports, and the element type
// is what lets a detect-only or test-only adapter be registered and evaluated without three failing
// stubs.
var registry = []Candidate{
	directSSH{},
	reverseSSH{},
}

// Registry returns the registered candidates in deterministic order. The returned slice is a copy,
// so a caller cannot reorder or shrink the contract it read; callers must not assume otherwise.
//
// The element type is the registration contract on purpose (design D7): a detect-only or test-only
// adapter is a Candidate and nothing more, while every R1a entry is a full Transport whose plan and
// verification members are part of the shipped adapters. PR 14 appends cloudflare-tunnel and
// tailscale to the same list, in that order.
func Registry() []Candidate {
	registered := make([]Candidate, len(registry))
	copy(registered, registry)
	return registered
}

// Evaluate runs Feasible for every candidate of the registry it is handed, preserving that order
// one row per candidate. Its parameter is the registration contract, so the caller's registry type
// is the same type the shipped one has: design §4's call
// `transport.Evaluate(transport.Registry(), diagnosis)` compiles literally, and a test's []Candidate
// registry flows through the same code path. It is pure: it reads the diagnosis, measures nothing,
// and touches no machine or remote system (R-HR-05).
func Evaluate(registry []Candidate, d diagnosis.Diagnosis) []Feasibility {
	feasibilities := make([]Feasibility, 0, len(registry))
	for _, candidate := range registry {
		feasibilities = append(feasibilities, candidate.Feasible(d))
	}
	return feasibilities
}
