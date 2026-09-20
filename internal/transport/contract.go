// Package transport owns the transport layer of herdr-reach: the candidate ways of getting bytes
// to the node, and the decision of which of them can work on the network the diagnosis measured.
//
// The layer is split in two contracts. Candidate is the registration contract — the stable name,
// the static prerequisite set and the feasibility decision — and Transport adds the plan and
// verification members of PRD §5.2, which later slices implement. A detect-only transport, or a
// test-only one, therefore needs nothing but Candidate to be registered and evaluated, and no
// adapter has to carry a plausible-looking stub for a member the product has not defined yet.
//
// This slice only decides. Feasible reads the diagnosis it is handed and measures nothing: no
// dial, no command, no file access and no configuration change happens here. The answer is always
// explained — a non-viable transport carries a reason, and a viable one names the measured
// observation it rests on — because a verdict without an explanation is unusable to both a reader
// and a script.
package transport

import (
	"context"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// RequirementKind names one prerequisite a transport may declare. It is the closed vocabulary of
// design D6, and it is short on purpose: a prerequisite is a fact a run reports or a user supplies,
// so a condition that requires changing something is a plan step (a later slice) rather than a
// requirement, and nothing here may encode "partially viable".
type RequirementKind string

const (
	// KindHostname is a hostname that exists in the user's Cloudflare account, so the edge has
	// something to route to. The user supplies it; no measurement of this machine can observe it.
	KindHostname RequirementKind = "hostname"
	// KindZoneMembership is the user's zone membership for the hostname's domain: an account fact
	// the user supplies and no run can measure.
	KindZoneMembership RequirementKind = "zone_membership"
	// KindThirdPartyPermission is a third-party tunnel adapter being permitted on this machine — a
	// policy fact the user knows and the tool only detects, never grants.
	KindThirdPartyPermission RequirementKind = "third_party_permission"
	// KindHubAddress is the hub's address for this run. It is the one kind that is both user input
	// (supplied with --hub) and an observation (the hub-directed probe), which is why a measured
	// block leaves the row Satisfied while Viable is false.
	KindHubAddress RequirementKind = "hub_address"
	// KindSSHDEffectiveConfig is an sshd whose configuration in force is the written one: the
	// forward half of a reverse path needs an sshd the hub can reach, and a divergent configuration
	// is never presented as satisfied (R-HR-18).
	KindSSHDEffectiveConfig RequirementKind = "sshd_effective_config"
)

// allRequirementKinds is the declaration order, which is also the order AllRequirementKinds
// reports.
var allRequirementKinds = []RequirementKind{
	KindHostname,
	KindZoneMembership,
	KindThirdPartyPermission,
	KindHubAddress,
	KindSSHDEffectiveConfig,
}

// AllRequirementKinds returns the closed requirement-kind set in declaration order. The returned
// slice is a copy, so a caller cannot reorder or shrink the contract; callers must not assume
// otherwise.
//
// It exists so the vocabulary is enumerable the way the reason-code set is
// (`probe.AllReasonCodes`) and the state set is (`diagnosis.AllStates`): a consumer that must
// cover every kind — the payload, the documentation, a later adapter's completeness case — reads
// the set instead of restating it.
func AllRequirementKinds() []RequirementKind {
	kinds := make([]RequirementKind, len(allRequirementKinds))
	copy(kinds, allRequirementKinds)
	return kinds
}

// Requirement is one prerequisite row, in both its static and its evaluated form: `Requires()`
// declares the set, and `Feasibility.Requires` carries the evaluated rows.
type Requirement struct {
	// Kind is which prerequisite this row is about.
	Kind RequirementKind
	// Satisfied reports whether the run supplied or measured the prerequisite. A measured block
	// leaves it true while Viable is false, because the block is a network fact rather than a
	// missing input; a reader must be able to tell the two apart. A prerequisite the run neither
	// supplied nor measured leaves it false, and Viable is false with it.
	Satisfied bool
	// Detail states what this row is: for a satisfied row, how it was supplied or measured; for an
	// unsatisfied one, what the user must supply or fix. Without a Detail an unsatisfied row would
	// read as "the network blocks this", which is exactly the fabricated reason D6 forbids.
	Detail string
}

// Feasibility is one transport's evaluated answer.
type Feasibility struct {
	// Viable is the strict decision: exactly true or false, never a third "sort of viable" value
	// and never a promised success (PRD §14.5). A transport whose prerequisite the user has not
	// supplied is not viable, because "not ready to use" must be unambiguously true to a script.
	Viable bool
	// Reason is the explanation, non-empty in both cases (R-HR-06). Precedence when several facts
	// could explain the verdict, in the order design §3.2 fixes: a measured blocking observation,
	// then an unresolved dependency of the decision, then an unmet requirement.
	Reason string
	// Notes carry the measured observations the verdict rests on — a viable transport names at
	// least one — and, for an unsatisfied requirement, what the user must supply so the row is not
	// read as a network block.
	Notes []string
	// Requires carries the evaluated prerequisite rows: one per kind Requires() declares, in
	// declaration order, each with its own satisfaction state.
	Requires []Requirement
}

// Candidate is the registration contract: everything a transport is, minus the plan and
// verification members of PRD §5.2. It is what the registry stores and evaluates, so a later
// detect-only adapter — or a test-only transport in a suite — implements it and nothing else.
type Candidate interface {
	// Name is the transport's stable identifier. It is what the payload, the human projection and
	// the registry order quote, so it is a contract rather than a display label.
	Name() string
	// Requires declares the transport's prerequisite set statically. The declaration carries each
	// row's Kind and Detail; satisfaction is decided per run by Feasible and reported in
	// Feasibility.Requires. Declaring a kind does not make it satisfied, and a prerequisite that
	// cannot be expressed as one of the closed kinds is not a requirement this contract can carry.
	Requires() []Requirement
	// Feasible reads a diagnosis and decides whether this transport can work on the network the
	// run measured. It performs no measurement of its own: every target, label and reason code it
	// names comes from the findings it was handed, and a measurement the run never made is never
	// turned into a network fact.
	Feasible(d diagnosis.Diagnosis) Feasibility
}

// Transport is the full contract of PRD §5.2's four V1 adapters. The three members beyond
// Candidate are the plan and verification halves of the product; in this slice every adapter fails
// them with ErrNotImplementedInThisPhase, and no adapter returns an empty plan or evidence set
// that a caller could mistake for success (RG-9).
type Transport interface {
	Candidate
	// PlanHub and PlanNode return the steps for the hub's and the node's side of a pairing. R3
	// defines the real plan types; until then both fail with ErrNotImplementedInThisPhase naming
	// the member and the owning slice.
	PlanHub(PairingBundle) ([]Step, error)
	PlanNode(PairingBundle) ([]Step, error)
	// Verify produces the evidence that a transport works. R5/R6 define the real handle and
	// verification semantics; until then it fails with ErrNotImplementedInThisPhase naming the
	// member and the owning slices.
	Verify(ctx context.Context, h Handle) ([]probe.Result, error)
}
