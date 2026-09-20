package transport

// This file is the cloudflare-tunnel adapter — PRD §5.2's primary implementation, which works when
// a hostname exists in a Cloudflare account and the edge is reachable — and the reading of the
// `cloudflare.edge` question group it decides on.
//
// Feasible performs no measurement. Every endpoint, label and reason code it names comes from the
// edge finding's evidence, and a question the run never answered is reported as an absence — never
// folded into one of the edge group's states, because a measurement that was not made supports no
// claim in either direction.
//
// Reason precedence follows design §3.2: a measured edge block first, then an unresolved edge, then
// the unmet account prerequisite. The HTTP/2 advice and the cloudflared pin note belong to a later
// slice; this file decides feasibility and states nothing about a fallback.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// questionCloudflareEdge is the reasoning layer's question for the edge reachability group. The
// adapter locates the finding by question and switches on its rule id; the prose conclusion is
// never parsed.
const questionCloudflareEdge = "cloudflare.edge"

// The hand-named rule ids of the `cloudflare.edge` group (design §5.2). The reasoning layer
// declares them as unexported constants, so this adapter spells them; the registry suite asserts
// they are still among diagnosis.AllRuleIDs(), which is what stops a table rename from silently
// stranding this adapter on its unrecognised-rule path.
const (
	cloudflareRuleEdgeReachable   = "CF_EDGE_REACHABLE"
	cloudflareRuleEdgePartial     = "CF_EDGE_PARTIAL"
	cloudflareRuleEdgeUnreachable = "CF_EDGE_UNREACHABLE"
	cloudflareRuleEdgeUnresolved  = "CF_EDGE_UNRESOLVED"
)

// cloudflareTunnel is the cloudflare-tunnel adapter. It carries no state, so it is registered as a
// value and two registries cannot observe each other through it.
type cloudflareTunnel struct{}

// Name is PRD §5.2's stable identifier for this adapter.
func (cloudflareTunnel) Name() string { return "cloudflare-tunnel" }

// cloudflareRequires is the adapter's declared prerequisite set in declaration order: a hostname
// first, so the edge has something to route to, then the zone membership the account needs to
// publish it. Each Detail names what the user must supply, because no measurement of this machine
// can observe an account fact.
func cloudflareRequires() []Requirement {
	return []Requirement{
		{
			Kind:   KindHostname,
			Detail: "a hostname that exists in your Cloudflare account, so the edge has something to route to: supply the hostname the tunnel will publish, or ask the domain owner for one",
		},
		{
			Kind:   KindZoneMembership,
			Detail: "membership in the Cloudflare zone that owns that hostname's domain, so the account can create the tunnel's DNS record: supply it in the Cloudflare account, which no measurement of this machine can observe",
		},
	}
}

// Requires declares the two prerequisites statically; Feasible evaluates them per run.
func (cloudflareTunnel) Requires() []Requirement { return cloudflareRequires() }

// Feasible decides cloudflare-tunnel from the measured edge state and the two account
// prerequisites. It is viable only when the edge half is satisfied — the edge is reachable, or
// partly reachable with at least one declared endpoint answering — and both account rows are
// satisfied; every other combination is explained by the highest-precedence fact design §3.2
// allows, and both evaluated requirement rows travel with the verdict.
func (cloudflareTunnel) Feasible(d diagnosis.Diagnosis) Feasibility {
	finding, found := findingByQuestion(d, questionCloudflareEdge)
	edge := assessCloudflareEdge(finding, found)
	requires := evaluateCloudflareRequires()

	// The viability gate reads the evaluated rows and the measured edge state, not a hardcoded
	// constant. R1a supplies no Cloudflare account fact — no flag carries a hostname or a zone
	// membership, and no probe measures one — so both rows are unsatisfied by construction and this
	// transport is never viable in this slice, which is the honest answer rather than a
	// qualification a strict boolean cannot carry.
	//
	// The edge half is satisfied by a measured pass, whether every declared endpoint answered or
	// only some did: CF_EDGE_PARTIAL means at least one endpoint answered inside the budget beside
	// one that did not, so the edge is measured reachable and the split belongs in the reason as a
	// qualification, not in the gate as a rejection. Treating it as the rejecting fact would turn a
	// measured pass into a block claim the diagnosis does not support, and design §3.2's precedence
	// reserves "a measured blocking observation" for CF_EDGE_UNREACHABLE, where every measured
	// endpoint really failed.
	viable := cloudflareEdgeHalfSatisfied(edge.outcome) && requires[0].Satisfied && requires[1].Satisfied

	notes := append([]string(nil), edge.notes...)
	if cloudflareEdgeHalfSatisfied(edge.outcome) {
		// Design D6's account note: with the edge half measured satisfied — reachable, or partly
		// reachable with at least one endpoint answering — the unmet hostname must read as
		// something the user supplies, not as a network fact this run established.
		notes = append(notes, "the Cloudflare edge is reachable from this network (measured), so the unmet hostname is an account prerequisite to supply, not a network block")
	}
	for _, row := range requires {
		if !row.Satisfied {
			notes = append(notes, requirementNote(row))
		}
	}

	return Feasibility{
		Viable:   viable,
		Reason:   cloudflareReason(edge, requires),
		Notes:    notes,
		Requires: requires,
	}
}

// PlanHub, PlanNode and Verify fail loudly until their owning slices define the real plan and
// verification shapes (RG-9). Each error names the member and the owner; no value is returned
// beside it.
func (cloudflareTunnel) PlanHub(PairingBundle) ([]Step, error) {
	return nil, notImplemented("PlanHub", "R3")
}

func (cloudflareTunnel) PlanNode(PairingBundle) ([]Step, error) {
	return nil, notImplemented("PlanNode", "R3")
}

func (cloudflareTunnel) Verify(context.Context, Handle) ([]probe.Result, error) {
	return nil, notImplemented("Verify", "R5/R6")
}

// evaluateCloudflareRequires returns the declared prerequisite rows in their R1a evaluated form.
//
// R1a has no input that supplies a Cloudflare account fact: no flag carries a hostname or a zone
// membership, and no probe measures one — the account is outside every measurement seam — so both
// rows are unsatisfied by construction, and the function says so instead of merely happening to be
// false. The declared Detail is preserved verbatim, because it is what the user must supply. A
// later slice that gains the account input changes this function and the rows' satisfaction with
// it, not a hardcoded verdict beside them.
func evaluateCloudflareRequires() []Requirement {
	declared := cloudflareRequires()
	evaluated := make([]Requirement, 0, len(declared))
	for _, row := range declared {
		row.Satisfied = false
		evaluated = append(evaluated, row)
	}
	return evaluated
}

// cloudflareEdgeOutcome is the seven-way reading of the edge question the adapter switches on.
// Four outcomes are the group's own states; three are the absences a caller-constructed diagnosis
// or a partial run can produce, which are never folded into one of the four.
type cloudflareEdgeOutcome int

const (
	// cloudflareEdgeReachable is a measured pass with no measured failure beside it.
	cloudflareEdgeReachable cloudflareEdgeOutcome = iota
	// cloudflareEdgePartial is a measured pass beside a measured failure: the declared endpoints
	// disagree.
	cloudflareEdgePartial
	// cloudflareEdgeBlocked is every measured endpoint failing, with no measured success and no
	// unanswered attempt.
	cloudflareEdgeBlocked
	// cloudflareEdgeUnresolved is no measured success beside an attempt that produced no answer.
	cloudflareEdgeUnresolved
	// cloudflareEdgeUnmeasured is a question the diagnosis never answered: the measurement was not
	// made, which is an absence and never a block.
	cloudflareEdgeUnmeasured
	// cloudflareEdgeNoEvidence is a recognised rule whose finding carries no observation, as a
	// caller-constructed diagnosis can produce. No endpoint is named, because none was carried.
	cloudflareEdgeNoEvidence
	// cloudflareEdgeUnrecognized is a rule id this adapter does not know: the question was answered
	// by a table this adapter was not written for, so no edge state is claimed.
	cloudflareEdgeUnrecognized
)

// cloudflareEdgeHalfSatisfied reports whether the measured edge half of the decision is satisfied:
// the edge is reachable, or it is partly reachable with at least one declared endpoint answering.
// CF_EDGE_PARTIAL is a measured pass — at least one endpoint answered inside the budget — so it
// satisfies the edge half; what it does not satisfy is the account prerequisite, and the split
// itself is named in the reason rather than turned into a rejection. Every other outcome is an
// absence or a measured block, and none of them satisfies the gate.
func cloudflareEdgeHalfSatisfied(outcome cloudflareEdgeOutcome) bool {
	return outcome == cloudflareEdgeReachable || outcome == cloudflareEdgePartial
}

// cloudflareEdgeAssessment is the edge question read the way the adapter needs it: the outcome, the
// evidence naming every endpoint the finding carried, the notes the verdict rests on, and the rule
// id the absence wording needs.
type cloudflareEdgeAssessment struct {
	outcome cloudflareEdgeOutcome
	// evidence names the facts the finding carried, one entry per observation, in Match order. It
	// is what the reason quotes, so no target can be named that the diagnosis did not carry.
	evidence string
	// notes carry the same observations for the notes list, or the absence note when the question
	// was not answered.
	notes []string
	// rule is the fired rule id, empty when no finding answered the question.
	rule string
}

// assessCloudflareEdge reads the edge question's finding — or the absence of one — into the
// assessment. The four rule ids are exactly the group's declared states; any other rule and a
// missing finding are absences the adapter reports as themselves rather than folding into one of
// the four, because a state the vocabulary does not define is not a measurement.
func assessCloudflareEdge(finding diagnosis.Finding, found bool) cloudflareEdgeAssessment {
	if !found {
		return cloudflareEdgeAssessment{
			outcome: cloudflareEdgeUnmeasured,
			notes:   []string{"cloudflare.edge was not measured: the diagnosis carries no " + questionCloudflareEdge + " finding"},
		}
	}
	if _, hasFact := firstEvidence(finding); !hasFact {
		return cloudflareEdgeAssessment{
			outcome: cloudflareEdgeNoEvidence,
			rule:    finding.Rule,
			notes:   []string{"the cloudflare.edge finding fired rule " + finding.Rule + " without an observation; no endpoint is named"},
		}
	}
	assessment := cloudflareEdgeAssessment{
		evidence: strings.Join(evidenceNotes(finding), "; "),
		notes:    evidenceNotes(finding),
		rule:     finding.Rule,
	}
	switch finding.Rule {
	case cloudflareRuleEdgeReachable:
		assessment.outcome = cloudflareEdgeReachable
	case cloudflareRuleEdgePartial:
		assessment.outcome = cloudflareEdgePartial
	case cloudflareRuleEdgeUnreachable:
		assessment.outcome = cloudflareEdgeBlocked
	case cloudflareRuleEdgeUnresolved:
		assessment.outcome = cloudflareEdgeUnresolved
	default:
		assessment.outcome = cloudflareEdgeUnrecognized
	}
	return assessment
}

// cloudflareReason words the verdict with design §3.2's precedence: the measured edge state comes
// first when it leaves no success (a block, then an unresolved edge), the unmet account
// prerequisite named through the shared requirementNote comes last — and it is the cause of
// non-viability when the edge half is satisfied, reachable or partly reachable, which is why the
// PARTIAL clause carries the split as a qualification and never as a rejection. The reason never
// states a network fact the diagnosis did not carry.
func cloudflareReason(edge cloudflareEdgeAssessment, requires []Requirement) string {
	clause := ""
	switch edge.outcome {
	case cloudflareEdgeReachable:
		clause = "the Cloudflare edge is reachable from this network (" + edge.evidence + ")"
	case cloudflareEdgePartial:
		clause = "the Cloudflare edge is only partly reachable (" + edge.evidence + "): at least one declared endpoint answered while another did not"
	case cloudflareEdgeBlocked:
		clause = "the Cloudflare edge is unreachable: every measured edge endpoint failed (" + edge.evidence + "), so the measured edge state rejects this transport"
	case cloudflareEdgeUnresolved:
		clause = "whether the Cloudflare edge is reachable is not established: there was no measured success and at least one edge attempt was unresolved (" + edge.evidence + ")"
	case cloudflareEdgeUnmeasured:
		clause = "the Cloudflare edge measurement was not made: the diagnosis carries no " + questionCloudflareEdge + " finding, so nothing is claimed about the edge in either direction"
	case cloudflareEdgeNoEvidence:
		clause = fmt.Sprintf("the Cloudflare edge question fired rule %s, but its finding carries no observation to name, so no edge state is claimed and no endpoint is invented", edge.rule)
	default:
		clause = fmt.Sprintf("the Cloudflare edge question was answered by rule %s, which this adapter does not recognise as one of the edge measurement's states, so no edge state is claimed", edge.rule)
	}

	prerequisite := ""
	for _, row := range requires {
		if !row.Satisfied {
			prerequisite = requirementNote(row)
			break
		}
	}
	switch {
	case prerequisite == "":
		return clause
	case cloudflareEdgeHalfSatisfied(edge.outcome):
		return clause + ", but the transport is not viable: " + prerequisite
	default:
		return clause + "; " + prerequisite
	}
}
