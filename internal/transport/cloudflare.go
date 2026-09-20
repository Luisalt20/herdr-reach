package transport

// This file is the cloudflare-tunnel adapter — PRD §5.2's primary implementation, which works when
// a hostname exists in a Cloudflare account and the edge is reachable — and the reading of the
// `cloudflare.edge` and `cloudflare.http2` question groups it decides on.
//
// Feasible performs no measurement. Every endpoint, label and reason code it names comes from the
// findings' evidence, and a question the run never answered is reported as an absence — never
// folded into one of the groups' states, because a measurement that was not made supports no claim
// in either direction.
//
// Reason precedence follows design §3.2: a measured edge block first, then an unresolved edge, then
// the unmet account prerequisite. The HTTP/2 advice is informational: it recommends and never
// enforces, it changes no verdict, and the cloudflared pin note from pin_note.go joins the notes as
// report-only research knowledge.

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

// The hand-named rule ids of the `cloudflare.http2` group (design §5.2), declared and guarded the
// same way: the feasibility suite asserts these literals stay declared ids.
const (
	cloudflareRuleHTTP2AdvisedQUICFailed      = "CF_HTTP2_ADVISED_QUIC_FAILED"
	cloudflareRuleHTTP2AdvisedQUICUnconfirmed = "CF_HTTP2_ADVISED_QUIC_UNCONFIRMED"
	cloudflareRuleHTTP2NoAdviceQUICUsable     = "CF_NO_HTTP2_ADVICE_QUIC_USABLE"
	cloudflareRuleHTTP2AdvisoryNotAssessed    = "CF_HTTP2_ADVISORY_NOT_ASSESSED"
)

// questionCloudflareHTTP2 is the reasoning layer's question for the HTTP/2 advice group. Like the
// edge question, the adapter locates the finding by question and switches on its rule id; the
// prose conclusion is never parsed. The advice is informational and never changes Viable: R1a's
// account prerequisite decides this transport, and the recommendation concerns the client's tunnel
// transport protocol rather than whether a tunnel can work.
const questionCloudflareHTTP2 = "cloudflare.http2"

// probeEgressQUIC is the probe whose observation the unconfirmed-advice note names. The note filters
// the finding's evidence by this key, so the note quotes the measurement the recommendation rests
// on instead of restating one.
const probeEgressQUIC = "egress.quic"

// The two shared sentences of the advising paths. They are separate notes so a reader sees the
// advice, its cost and its boundary as distinct statements, and so a case can assert each without
// matching a larger sentence.
const (
	// cloudflareHTTP2TradeOff is the documented cost of forcing HTTP/2 on the tunnel transport.
	cloudflareHTTP2TradeOff = "the HTTP/2 path does not support post-quantum key agreement on the tunnel transport, so choosing it forfeits that there"
	// cloudflareHTTP2RecommendOnly is the enforcement boundary R-HR-05 requires on every advising
	// path: this slice recommends, and nothing was applied or written.
	cloudflareHTTP2RecommendOnly = "this slice recommends and does not enforce: no fallback was applied and no configuration was written"
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
// allows, and both evaluated requirement rows travel with the verdict. The HTTP/2 advice and the
// pin note are informational: they join the notes whatever the verdict, because the transport is
// never viable in R1a and the advice is about the client's tunnel transport protocol, not about
// whether the tunnel can work.
func (cloudflareTunnel) Feasible(d diagnosis.Diagnosis) Feasibility {
	finding, found := findingByQuestion(d, questionCloudflareEdge)
	edge := assessCloudflareEdge(finding, found)
	http2Finding, http2Found := findingByQuestion(d, questionCloudflareHTTP2)
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
	// The notes order is fixed and deterministic by construction: the measured edge observations
	// first, because the verdict is traced to them; then the HTTP/2 advice the measurements support,
	// because it is what a reader does with them; then the pin note, which is report-only research
	// knowledge about the client rather than a measurement of this run, and therefore travels with
	// every verdict; and last the prerequisite notes — the D6 account note when the edge half is
	// satisfied, then the unmet rows — because they are what remains to act on.
	notes = append(notes, cloudflareHTTP2AdviceNotes(http2Finding, http2Found)...)
	notes = append(notes, PinNote())
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

// cloudflareHTTP2AdviceNotes reads the `cloudflare.http2` finding and returns the notes the advice
// contributes, or nil when the run produced no advice.
//
// The advice is informational and never changes Viable. In R1a the account prerequisite is
// unsatisfied by construction, so the transport is never viable, and the recommendation concerns
// the tunnel transport protocol the client would use rather than whether a tunnel can work.
//
//   - A failed datagram measurement beside a measured TCP path advises HTTP/2, states the
//     post-quantum trade-off, and states the recommendation-only boundary.
//   - An unconfirmed datagram measurement advises the same way and names the measurement it rests
//     on, from this finding's own evidence, because the recommendation is made from an absence of
//     confirmation rather than from a measured block (design D8, RG-5).
//   - A datagram reply produces no downgrade advice: the probe claims only that the datagram was
//     not silently dropped, which justifies no protocol change.
//   - The not-assessed rule, a missing finding and a rule this adapter does not recognise produce
//     no advice and no invented measurement: an HTTP/2 recommendation needs a measured TCP path,
//     and a state this adapter cannot read is not a measurement.
func cloudflareHTTP2AdviceNotes(finding diagnosis.Finding, found bool) []string {
	if !found {
		return nil
	}
	switch finding.Rule {
	case cloudflareRuleHTTP2AdvisedQUICFailed:
		return []string{
			"HTTP/2 recommendation: use HTTP/2 for the cloudflared tunnel transport on this path, because the UDP datagram measurement failed",
			cloudflareHTTP2TradeOff,
			cloudflareHTTP2RecommendOnly,
		}
	case cloudflareRuleHTTP2AdvisedQUICUnconfirmed:
		return []string{
			"HTTP/2 recommendation: use HTTP/2 for the cloudflared tunnel transport on this path, because the datagram path was never confirmed",
			cloudflareHTTP2UnconfirmedMeasurement(finding),
			cloudflareHTTP2TradeOff,
			cloudflareHTTP2RecommendOnly,
		}
	case cloudflareRuleHTTP2NoAdviceQUICUsable:
		return []string{
			"no HTTP/2 downgrade is recommended: the UDP datagram measurement to the edge drew a reply, so the datagram path is usable",
		}
	default:
		// The not-assessed rule and every unrecognised rule: no advice, no measurement.
		return nil
	}
}

// cloudflareHTTP2UnconfirmedMeasurement names the unconfirmed measurement the conservative advice
// rests on, from the finding's evidence only. When the finding carries no datagram observation the
// note says so instead of quoting one: the recommendation's wording is the tool's own, but the
// measurement is not, and no target is invented for it.
func cloudflareHTTP2UnconfirmedMeasurement(finding diagnosis.Finding) string {
	var quic []diagnosis.Fact
	for _, fact := range finding.Evidence {
		if fact.Probe == probeEgressQUIC {
			quic = append(quic, fact)
		}
	}
	if len(quic) == 0 {
		return "the recommendation rests on an unconfirmed measurement, but the finding carries no datagram observation to name, so no measurement is quoted"
	}
	return "the recommendation rests on an unconfirmed measurement: " +
		strings.Join(evidenceNotes(diagnosis.Finding{Evidence: quic}), "; ")
}
