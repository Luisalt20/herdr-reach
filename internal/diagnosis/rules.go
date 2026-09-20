package diagnosis

// This file is the ordered declarative rule table of design §5.2: the rule and need types, the
// table itself, and the first-match evaluation the table's questions are answered by.
//
// Mechanism. For each question the first matching rule wins: rules are read in declaration order
// and the first rule whose every need the run's facts satisfy is the one that fires — nothing is
// merged with a later rule and nothing is retried. There is no fall-through default and no
// catch-all conclusion: a question no rule matched concludes nothing at all, and diagnose.go
// turns it into an explicit open question naming the observable states the table needed. Every
// rule is a pure predicate over the facts one run reported, so the same facts always produce the
// same findings and the same open questions.
//
// Scope. This slice declares the derived fact rules — one per fact slot of ids.go, so one per
// registered probe and observable state — and every hand-named question group of §5.2: the
// `ssh.destination` group (PRD §1.1's port-versus-protocol disambiguation), the `cloudflare.edge`
// and `cloudflare.http2` groups, and the `local.sshd` and `node.platform` groups. The two `tls.*`
// groups are restated to the derived fact ids of `tls.interception` and `tls.truststore` rather
// than declared as hand-named groups: the fact conclusions already carry each observation's reason
// code and verbatim detail, which is where the platform/override split lives (RG-3). Hand-named
// ids are declared in ids.go beside the derivation, so no id is spelled twice.
//
// `Need` gained an optional observation label in this slice. `local.sshd` reports three
// observations and §5.2's rows must tell an absent binary from a stopped service, which share one
// (resolution, verdict) pair; a non-empty Label matches only the observation the probe reported
// under that label, while an empty one keeps the semantics every earlier row has.
//
// No transport viability is derived anywhere in this package. The `node.platform` refusal names
// WSL2 as the Windows path this tool handles today because the classification does, and every
// conclusion that borders on transport states that the transport decision belongs to the transport
// layer.
//
// A hand-named id may be carried by more than one row, and that is the one thing this file adds to
// the mechanism PR 10 landed. §5.2's `Requires` column contains disjunctions — a hub failure beside
// either public measurement having produced no answer, an edge that is reachable from some regions
// and not others — and a row's Match is a conjunction of exact observables. An id therefore names a
// conclusion and each row is one clause that establishes it, so first-match evaluation over the
// rows is exactly a disjunction of conjunctions: the first clause the run's facts satisfy fires,
// and its DependsOn names precisely the observables that clause rested on. Nothing else about
// evaluation changes, and the derived fact rules, which need no disjunction, are one row per id as
// before.
//
// DependsOn is computed from Match when a rule is built rather than declared beside it, because
// §5.2 defines it as exactly the set of observables in Match. Two data fields that must agree are
// a drift waiting to happen, and the payload's `depends_on` is how a reader sees what a
// conclusion rests on.

import (
	"slices"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// Need is one observable state a rule requires: the probe that reported it, the exact resolution
// and verdict the measurement layer attached to that observation, and — when the probe reported
// several observations — the label of the one observation the need is about.
//
// It is deliberately the triple rather than a State value. The table states what must have been
// measured, and the matchable state is derived from exactly this triple in one place (StateOf,
// facts.go); matching on the triple also means an observation the vocabulary forbids — a definite
// verdict on a not-measured observation, for instance — satisfies no need at all, instead of
// being quietly mapped onto the absence of an answer.
//
// Label is the optional fourth part, and it exists because one probe can report several
// observations whose triples are identical: `local.sshd`'s absent binary and stopped service are
// both measured failures with the vocabulary's shared `sshd_absent` reason code, and a row that
// must match one of them cannot say which with the triple alone. An empty Label means "any
// observation of the probe" — exactly the semantics every earlier row has — and a non-empty Label
// matches only the observation the probe reported under that label. Labels are the probe's own
// declared, stable ones: the reasoning layer and the human projection both quote them, so a rename
// is a contract change on both sides rather than an internal detail.
type Need struct {
	Probe      string
	Label      string
	Resolution probe.Resolution
	Verdict    probe.Verdict
}

// Rule is one row of the table: the question it answers, the observable states it needs, the
// conclusion it emits when it fires, and the observables that conclusion rests on.
type Rule struct {
	// ID is the rule's stable identifier: derived for a fact question (ids.go, design §3.6) and
	// hand-named for a derived question (§5.2). PRD §5.3 requires the report to quote the rule
	// that produced a conclusion, so the id is part of the conclusion's provenance.
	//
	// A hand-named id may be carried by more than one row, because §5.2's `Requires` column contains
	// disjunctions while a row's Match is a conjunction: the id names the conclusion and each row is
	// one clause that establishes it, so an id is a conclusion's name rather than a row key. Derived
	// fact ids are one row each; nothing else in the table shares an id.
	ID string
	// Question is the question this rule answers. Rules that share a question are evaluated in
	// declaration order against the same facts, and the first match wins.
	Question string
	// Match is every observable state the rule requires. All of them must be satisfied: a rule
	// that needs several observables does not fire when any one of them was not measured as
	// needed, which is what makes an unresolved observation weaken only the conclusions that
	// rest on it (R-HR-NF-03).
	Match []Need
	// Conclusion is the conclusion text the rule emits, named here and worded in findings.go:
	// the wording of every conclusion lives in exactly one file.
	Conclusion Conclusion
	// DependsOn names the probes Match rests on, in Match order, each once — §5.2's "exactly the
	// set of observables in Match", computed rather than declared twice.
	DependsOn []string
}

// rules is the ordered rule table: the derived fact rules of every fact question, in ids.go's
// declaration order, then the hand-named rows of the derived question groups, in the order
// groupRules declares them. Declaration order is evaluation order, and it is the order AllRuleIDs
// reports and questions() walks.
var rules = append(factRules(), groupRules()...)

// factRules builds the derived fact rules from the declarations in ids.go. It is the only
// construction path of the derived half of the table, so a rule cannot carry an id its slot does
// not derive, a need the slot does not declare, or a DependsOn set that disagrees with its Match.
func factRules() []Rule {
	var table []Rule
	for _, question := range factQuestions {
		for _, slot := range factSlots {
			table = append(table, newRule(
				RuleID(question.probe, slot.state),
				question.question,
				factConclusion(question.probe, slot.state),
				Need{Probe: question.probe, Resolution: slot.resolution, Verdict: slot.verdict},
			))
		}
	}
	return table
}

// newRule is the single construction path of one table row. The needs are variadic because most
// rules state one observable and a derived question states several; a rule with no need would be a
// rule that could match a run which measured nothing, and the case that walks every row of the
// table refuses one.
func newRule(id, question string, conclusion Conclusion, match ...Need) Rule {
	return Rule{
		ID:         id,
		Question:   question,
		Match:      match,
		Conclusion: conclusion,
		DependsOn:  dependsOn(match),
	}
}

// The question names of §5.2's hand-named groups this slice lands. A question name is the label a
// finding and an open question carry, and it is never the probe's own name where §5.2 gives that
// name to another group: `egress.ssh.known` answers to `ssh.public_22` and `egress.hub.direct` to
// `hub.reachability`, so each group below is a question of its own rather than a rename of either.
const (
	// questionSSHDestination is PRD §1.1's port-versus-protocol question: is the block on the SSH
	// protocol, or on the hub's address?
	questionSSHDestination = "ssh.destination"
	// questionCloudflareEdge is the Cloudflare edge reachability question: is the tunnel edge
	// reachable from this network, per measured edge endpoint?
	questionCloudflareEdge = "cloudflare.edge"
	// questionCloudflareHTTP2 is the HTTP/2 advice question: should this run recommend forcing the
	// tunnel transport to HTTP/2?
	questionCloudflareHTTP2 = "cloudflare.http2"
	// questionLocalSSHD is the sshd question: is an sshd present on this machine, and is the
	// configuration in force the one that was written? It is the group question, distinct from the
	// fact question `local.sshd.fact`, because §5.2 gives the probe's own name to the group.
	questionLocalSSHD = "local.sshd"
	// questionNodePlatform is the node-classification question: what kind of machine is this node,
	// and is it one this tool supports?
	questionNodePlatform = "node.platform"
)

// needInState builds the need for one probe in one matchable state, matching any observation the
// probe reported: the label-less form of needLabeled, and the semantics every derived fact rule
// and every earlier group row has.
//
// It reads the slot declaration the derived fact rules are built from (ids.go), so a hand-named row
// and a derived rule can never spell one observable two ways: the state names which of the four
// observable states is needed, and the resolution and verdict pair a fact carries in that state is
// the vocabulary's own rather than a second list here. A state outside the closed four-state set
// yields a need no observation can satisfy — the empty pair — instead of a need that would match
// anything.
func needInState(probeName string, state State) Need {
	return needLabeled(probeName, "", state)
}

// needLabeled builds the need for one labeled observation of one probe in one matchable state.
//
// An empty label keeps needInState's semantics exactly: any observation of the probe satisfies the
// need. A non-empty label narrows the need to the observation the probe reported under that label,
// which is what lets a row distinguish two observations that carry the same (resolution, verdict)
// pair. The label is compared verbatim and is never parsed: the probe's own stable label is the
// contract, and a conclusion that quotes the observation quotes the same string.
//
// The slot lookup is shared with needInState, so a hand-named row and a derived rule cannot spell
// one observable two ways, and an unknown state still yields a need no observation can satisfy.
func needLabeled(probeName, label string, state State) Need {
	for _, slot := range factSlots {
		if slot.state == state {
			return Need{Probe: probeName, Label: label, Resolution: slot.resolution, Verdict: slot.verdict}
		}
	}
	return Need{Probe: probeName, Label: label}
}

// groupRules is the hand-named question groups of §5.2 that this slice lands. Their rows are built
// through newRule like every other row, so the DependsOn contract is computed one way for the whole
// table; what is group-specific is the ordering, which is stated beside each group.
func groupRules() []Rule {
	var table []Rule
	table = append(table, sshDestinationRules()...)
	table = append(table, cloudflareEdgeRules()...)
	table = append(table, cloudflareHTTP2Rules()...)
	table = append(table, sshdRules()...)
	table = append(table, nodePlatformRules()...)
	return table
}

// sshDestinationRules is the `ssh.destination` group of design §5.2: PRD §1.1's port-versus-protocol
// disambiguation, and this change's acceptance case (R-HR-03, PRD §13).
//
// The order below is the propagation order. The two rows that conclude a destination block from a
// measured public pass come first, because nothing weaker may stand in front of a conclusion the
// run's own measurements support. The rows that state a hub failure without a public answer come
// next, in the order of what they leave established: an absence first (the public measurement that
// would establish what the hub failure means produced no answer, or was never made, and the
// conclusion names which), then a public measurement that failed as well (both sides failed, so a
// destination-level block cannot be separated from a protocol-level one), because a failure is a
// definite answer and the weaker reading of two failures is the one that makes no claim. The rows
// that conclude nothing about the hub come last, so a hub that answered can never be reported as
// unassessed.
//
// A run whose public measurement did not answer therefore cannot reach `SSH_DEST_BLOCKED_BY_PUBLIC_SSH`
// however its hub measurement came out: the needs are the exact triples of ids.go, no absence
// carries the pass triple, and first-match evaluation never consults a second clause for a question
// once one has fired. Every row that shares an id is one clause of §5.2's `Requires` disjunction,
// and the clause that fires is the one whose observables its conclusion rests on — which is how the
// weaker conclusion names the public probe that produced no answer instead of the two probes a
// reader might assume it consulted.
func sshDestinationRules() []Rule {
	return []Rule{
		// SSH is allowed and the destination is blocked: a public SSH measurement passed, so the
		// protocol is not blocked, and the declared hub address did not answer inside the probe's
		// budget. Port 22 first, then port 443 — the second row is reached only when port 22 did not
		// pass, so it names the port that carried the disambiguation rather than assuming one.
		newRule(ruleSSHDestBlockedByPublicSSH, questionSSHDestination, namedConclusion(ruleSSHDestBlockedByPublicSSH),
			needInState("egress.ssh.known", StatePass),
			needInState("egress.hub.direct", StateFail),
		),
		newRule(ruleSSHDestBlockedByPublicSSH443, questionSSHDestination, namedConclusion(ruleSSHDestBlockedByPublicSSH443),
			needInState("egress.ssh.443", StatePass),
			needInState("egress.hub.direct", StateFail),
		),

		// The hub failed and no public measurement answered: what the hub failure means for SSH egress
		// is not established. One clause per (public probe, absence) pair, so the conclusion names the
		// measurement that produced no answer.
		newRule(ruleSSHDestBlockUnestablishedPublicUnresolved, questionSSHDestination, namedConclusion(ruleSSHDestBlockUnestablishedPublicUnresolved),
			needInState("egress.ssh.known", StateUnresolved),
			needInState("egress.hub.direct", StateFail),
		),
		newRule(ruleSSHDestBlockUnestablishedPublicUnresolved, questionSSHDestination, namedConclusion(ruleSSHDestBlockUnestablishedPublicUnresolved),
			needInState("egress.ssh.known", StateNotMeasured),
			needInState("egress.hub.direct", StateFail),
		),
		newRule(ruleSSHDestBlockUnestablishedPublicUnresolved, questionSSHDestination, namedConclusion(ruleSSHDestBlockUnestablishedPublicUnresolved),
			needInState("egress.ssh.443", StateUnresolved),
			needInState("egress.hub.direct", StateFail),
		),
		newRule(ruleSSHDestBlockUnestablishedPublicUnresolved, questionSSHDestination, namedConclusion(ruleSSHDestBlockUnestablishedPublicUnresolved),
			needInState("egress.ssh.443", StateNotMeasured),
			needInState("egress.hub.direct", StateFail),
		),

		// The hub failed and both public measurements failed as well: destination-level and
		// protocol-level blocks are not separable from this run. It is declared after the absence
		// rows, so "both failed" is only reached when neither public measurement produced an absence —
		// any unresolved or not-measured public observation fires the weaker row above.
		newRule(ruleSSHDestBlockUnestablishedPublicFailed, questionSSHDestination, namedConclusion(ruleSSHDestBlockUnestablishedPublicFailed),
			needInState("egress.ssh.known", StateFail),
			needInState("egress.ssh.443", StateFail),
			needInState("egress.hub.direct", StateFail),
		),

		// No destination block was observed: the hub's declared address accepted a connection and the
		// public measurement passed. Nothing is said about a block, because no measurement reported one.
		newRule(ruleSSHNoDestBlockObserved, questionSSHDestination, namedConclusion(ruleSSHNoDestBlockObserved),
			needInState("egress.hub.direct", StatePass),
			needInState("egress.ssh.known", StatePass),
		),

		// Nothing is claimed about the hub. One clause per absence, because the conclusion says which
		// one happened: an attempt that produced no answer, or an attempt that was never made — the
		// default live case, where no hub address was supplied.
		newRule(ruleSSHDestBlockNotAssessed, questionSSHDestination, namedConclusion(ruleSSHDestBlockNotAssessed),
			needInState("egress.hub.direct", StateUnresolved),
		),
		newRule(ruleSSHDestBlockNotAssessed, questionSSHDestination, namedConclusion(ruleSSHDestBlockNotAssessed),
			needInState("egress.hub.direct", StateNotMeasured),
		),
	}
}

// cloudflareEdgeRules is the `cloudflare.edge` group of design §5.2: is the Cloudflare edge reachable
// from this network?
//
// The group reasons over the set of per-region observations of both TCP edge probes and never over an
// aggregate: `egress.cf.443` and `egress.cf.7844` each report one observation per region (design
// D10), so "region1 answered, region2 did not" is a difference the group states instead of hiding.
// Reading the observations directly is also what keeps a split observable at all: both probes'
// aggregate verdicts would be the worst of their regions, and a pass beside a failure would
// disappear.
//
// The order is the order of what each state claims, most specific first.
//
//  1. PARTIAL — a measured pass beside a measured failure: the declared endpoints disagree. One
//     clause per (pass probe, fail probe) pair, because a row's Match is a conjunction and the
//     conclusion quotes both sides of the split.
//  2. REACHABLE — a measured pass, and no failure (every pass-beside-failure pair was taken above):
//     at least one declared endpoint answered. The claim is existential, so an unresolved or
//     not-measured sibling does not weaken it — it is visible in the run's own coverage and in the
//     derived fact rules — and declaring the clause before the absence clauses keeps a measured
//     answer from being reported as an unestablished one.
//  3. UNRESOLVED — no measured pass, and at least one attempt that produced no answer: whether the
//     edge is reachable is not established. It is declared before the failure clause on purpose: an
//     attempt that produced nothing makes the run incomplete, and "all measured regions failed" is
//     not the same statement when another region was never answered (R-HR-NF-03).
//  4. UNREACHABLE — no measured pass and no unresolved attempt, so the failures stand alone: every
//     measured endpoint failed. It is reached only when every measured endpoint really did fail,
//     and a not-measured sibling stays visible in the run's coverage rather than being turned into a
//     failure.
//
// A run whose edge observations are all not measured matches no clause: no measurement of the edge
// was made, and the group reports the question as open instead of inventing a state for it.
//
// One reading of clause 4 deserves stating, because it is where the mechanism shows through. A clause
// quotes the observations its own Match selected — the failures of the probe it names — while the
// wording quantifies over every measured endpoint. The negative half of that universal is not matched
// evidence but the ordering's guarantee: a pass anywhere would have fired a PARTIAL or REACHABLE
// clause, and an unresolved attempt anywhere would have fired an UNRESOLVED clause, so this clause is
// reached only when the rest of the set is failures and unmeasured endpoints. That is also why the
// wording says "every measured edge endpoint" rather than "every declared edge endpoint": an endpoint
// this run never measured is not a failure, and the run's own coverage carries the gap.
func cloudflareEdgeRules() []Rule {
	return []Rule{
		// A split: a measured pass beside a measured failure.
		newRule(ruleCFEdgePartial, questionCloudflareEdge, namedConclusion(ruleCFEdgePartial),
			needInState("egress.cf.443", StatePass),
			needInState("egress.cf.443", StateFail),
		),
		newRule(ruleCFEdgePartial, questionCloudflareEdge, namedConclusion(ruleCFEdgePartial),
			needInState("egress.cf.443", StatePass),
			needInState("egress.cf.7844", StateFail),
		),
		newRule(ruleCFEdgePartial, questionCloudflareEdge, namedConclusion(ruleCFEdgePartial),
			needInState("egress.cf.7844", StatePass),
			needInState("egress.cf.443", StateFail),
		),
		newRule(ruleCFEdgePartial, questionCloudflareEdge, namedConclusion(ruleCFEdgePartial),
			needInState("egress.cf.7844", StatePass),
			needInState("egress.cf.7844", StateFail),
		),

		// At least one declared endpoint answered, and none failed.
		newRule(ruleCFEdgeReachable, questionCloudflareEdge, namedConclusion(ruleCFEdgeReachable),
			needInState("egress.cf.443", StatePass),
		),
		newRule(ruleCFEdgeReachable, questionCloudflareEdge, namedConclusion(ruleCFEdgeReachable),
			needInState("egress.cf.7844", StatePass),
		),

		// No measured success, and an attempt that produced no answer.
		newRule(ruleCFEdgeUnresolved, questionCloudflareEdge, namedConclusion(ruleCFEdgeUnresolved),
			needInState("egress.cf.443", StateUnresolved),
		),
		newRule(ruleCFEdgeUnresolved, questionCloudflareEdge, namedConclusion(ruleCFEdgeUnresolved),
			needInState("egress.cf.7844", StateUnresolved),
		),

		// No measured success and no unanswered attempt: the measured endpoints all failed.
		newRule(ruleCFEdgeUnreachable, questionCloudflareEdge, namedConclusion(ruleCFEdgeUnreachable),
			needInState("egress.cf.443", StateFail),
		),
		newRule(ruleCFEdgeUnreachable, questionCloudflareEdge, namedConclusion(ruleCFEdgeUnreachable),
			needInState("egress.cf.7844", StateFail),
		),
	}
}

// cloudflareHTTP2Rules is the `cloudflare.http2` group of design §5.2: should this run recommend
// forcing the tunnel transport to HTTP/2?
//
// The order is the order of the evidence, and it is the same order the derived fact rules read one
// probe's own observations in (ids.go): a measured failure first, then an absence, then a pass.
//
//  1. QUIC_FAILED — the datagram measurement failed. The conclusion recommends HTTP/2 and says that
//     it recommends.
//  2. QUIC_UNCONFIRMED — the datagram measurement produced no answer or was never made. The same
//     advice, conservatively, and the conclusion names the measurement that was never confirmed,
//     because that absence is what it rests on rather than a measured block (design D8, RG-5).
//  3. QUIC_USABLE — the datagram measurement drew a reply. No downgrade is recommended: nothing here
//     measured a block, and this probe claims only that a datagram was not silently dropped.
//  4. NOT_ASSESSED — the edge itself did not answer, was never measured, or produced no answer. No
//     advice is given, because no measured TCP path could carry one; every advising clause above
//     needs a measured edge pass, which is §5.2's `cloudflare.edge ∈ {reachable, partial}` condition
//     expressed over the observations themselves rather than over another rule's conclusion.
//
// The group recommends and never enforces: no clause would be reachable if a fallback had to have
// been applied, because nothing in this slice applies one (R-HR-05). The post-quantum trade-off the
// recommendation carries is worded where the recommendation is assembled (design §5.4).
func cloudflareHTTP2Rules() []Rule {
	return []Rule{
		newRule(ruleCFHTTP2AdvisedQUICFailed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisedQUICFailed),
			needInState("egress.quic", StateFail),
			needInState("egress.cf.443", StatePass),
		),
		newRule(ruleCFHTTP2AdvisedQUICFailed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisedQUICFailed),
			needInState("egress.quic", StateFail),
			needInState("egress.cf.7844", StatePass),
		),
		newRule(ruleCFHTTP2AdvisedQUICUnconfirmed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisedQUICUnconfirmed),
			needInState("egress.quic", StateUnresolved),
			needInState("egress.cf.443", StatePass),
		),
		newRule(ruleCFHTTP2AdvisedQUICUnconfirmed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisedQUICUnconfirmed),
			needInState("egress.quic", StateUnresolved),
			needInState("egress.cf.7844", StatePass),
		),
		newRule(ruleCFHTTP2AdvisedQUICUnconfirmed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisedQUICUnconfirmed),
			needInState("egress.quic", StateNotMeasured),
			needInState("egress.cf.443", StatePass),
		),
		newRule(ruleCFHTTP2AdvisedQUICUnconfirmed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisedQUICUnconfirmed),
			needInState("egress.quic", StateNotMeasured),
			needInState("egress.cf.7844", StatePass),
		),
		newRule(ruleCFNoHTTP2AdviceQUICUsable, questionCloudflareHTTP2, namedConclusion(ruleCFNoHTTP2AdviceQUICUsable),
			needInState("egress.quic", StatePass),
		),
		newRule(ruleCFHTTP2AdvisoryNotAssessed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisoryNotAssessed),
			needInState("egress.cf.443", StateFail),
		),
		newRule(ruleCFHTTP2AdvisoryNotAssessed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisoryNotAssessed),
			needInState("egress.cf.7844", StateFail),
		),
		newRule(ruleCFHTTP2AdvisoryNotAssessed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisoryNotAssessed),
			needInState("egress.cf.443", StateUnresolved),
		),
		newRule(ruleCFHTTP2AdvisoryNotAssessed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisoryNotAssessed),
			needInState("egress.cf.7844", StateUnresolved),
		),
		newRule(ruleCFHTTP2AdvisoryNotAssessed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisoryNotAssessed),
			needInState("egress.cf.443", StateNotMeasured),
		),
		newRule(ruleCFHTTP2AdvisoryNotAssessed, questionCloudflareHTTP2, namedConclusion(ruleCFHTTP2AdvisoryNotAssessed),
			needInState("egress.cf.7844", StateNotMeasured),
		),
	}
}

// sshdRules is the `local.sshd` group of design §5.2: is an sshd present on this machine, and is
// the configuration in force the one that was written (R-HR-18)?
//
// The group reads two of the probe's three observations by their labels. `binary present` and
// `service state` share the vocabulary's `sshd_absent` reason code and the measured-failure triple
// — the closed reason set has no code for "installed but not running" — so without the label an
// absent binary and a stopped service would match the same row, and the conclusion that says
// "installing sshd is a later slice" could be produced by a binary that is installed and merely
// stopped. The labels are the probe's own stable ones, declared in local.go, and the cases pin them
// literally.
//
// The order follows the table's FAIL → NOT_MEASURED → PASS discipline, which is what §5.2's
// suppression requires: a diverging configuration fires the divergence conclusion and an absent
// binary fires the absence conclusion, so neither can ever be reported as configured, and the
// not-measured row stays the default live case — the zero-execution boundary excludes `sshd -T`,
// so a live run reports the excluded capability rather than a configured sshd.
//
// The service state is deliberately not required. A stopped service is a fact the probe reports and
// the derived fact rule states; it does not change whether the binary is present and its
// configuration agrees, which is the question this group answers. Requiring the service would also
// mean a machine that has sshd installed and configured but not running could not receive the
// positive conclusion its measurements support.
func sshdRules() []Rule {
	return []Rule{
		// The written configuration and the configuration in force were both measured and they
		// disagree. This is the first row so a divergence can never fall through to the positive
		// conclusion below.
		newRule(ruleSSHDPresentConfigDivergent, questionLocalSSHD, namedConclusion(ruleSSHDPresentConfigDivergent),
			needLabeled("local.sshd", "effective config", StateFail),
		),
		// No sshd binary at the documented path. The label keeps a stopped service out of this row.
		newRule(ruleSSHDAbsent, questionLocalSSHD, namedConclusion(ruleSSHDAbsent),
			needLabeled("local.sshd", "binary present", StateFail),
		),
		// The configuration in force was not measured: the probe reports the excluded capability, and
		// no claim is made about what configuration is in force. This is the default live case.
		newRule(ruleSSHDEffectiveConfigNotMeasured, questionLocalSSHD, namedConclusion(ruleSSHDEffectiveConfigNotMeasured),
			needLabeled("local.sshd", "effective config", StateNotMeasured),
		),
		// The positive conclusion: the binary is present and the effective configuration agrees with
		// the written one. Both labels are required, so neither half can stand in for the other.
		newRule(ruleSSHDPresentConfigured, questionLocalSSHD, namedConclusion(ruleSSHDPresentConfigured),
			needLabeled("local.sshd", "binary present", StatePass),
			needLabeled("local.sshd", "effective config", StatePass),
		),
	}
}

// nodePlatformRules is the `node.platform` group of design §5.2: what kind of machine is this node,
// and is it one this tool supports?
//
// `local.env` reports one observation, so every row needs the probe without a label and the group
// answers the classification's own question over the probe's derived states. The order follows the
// same FAIL → UNRESOLVED → PASS discipline as the fact slots: a measured refusal is declared first
// so it can never be reported as supported, and an unclassifiable machine is the absence it is
// rather than a default.
//
// The refusal conclusion names WSL2 as the Windows path this tool handles today because the
// classification's own wording does (R-HR-30), and it decides no transport: whether any transport
// can reach the hub is the transport layer's decision, and this package derives no viability
// anywhere.
func nodePlatformRules() []Rule {
	return []Rule{
		// Native Windows: a measured refusal, declared before any weaker reading.
		newRule(ruleNodePlatformRefusedNativeWindows, questionNodePlatform, namedConclusion(ruleNodePlatformRefusedNativeWindows),
			needInState("local.env", StateFail),
		),
		// The signals matched no supported classification: no platform is assumed.
		newRule(ruleNodePlatformUnknown, questionNodePlatform, namedConclusion(ruleNodePlatformUnknown),
			needInState("local.env", StateUnresolved),
		),
		// One of the supported classifications was measured.
		newRule(ruleNodePlatformSupported, questionNodePlatform, namedConclusion(ruleNodePlatformSupported),
			needInState("local.env", StatePass),
		),
	}
}

// dependsOn is the observable set a rule rests on: the probes its Match names, in Match order,
// each once. It is §5.2's DependsOn contract, and it is derived here so a rule cannot declare a
// dependency its Match does not carry.
func dependsOn(match []Need) []string {
	var probes []string
	for _, need := range match {
		if slices.Contains(probes, need.Probe) {
			continue
		}
		probes = append(probes, need.Probe)
	}
	return probes
}

// Rules returns the ordered rule table as a copy: the rows in evaluation order, with their Match
// and DependsOn slices copied too, so a caller cannot reorder, shrink or rewrite the contract it
// read.
func Rules() []Rule {
	table := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		table = append(table, Rule{
			ID:         rule.ID,
			Question:   rule.Question,
			Match:      append([]Need(nil), rule.Match...),
			Conclusion: rule.Conclusion,
			DependsOn:  append([]string(nil), rule.DependsOn...),
		})
	}
	return table
}

// matchRule returns the first rule of one question whose needs the run's facts satisfy, together
// with the facts that satisfied them.
//
// First match wins: rules are read in declaration order and the first rule whose every need is
// satisfied fires, whether or not a later rule of the same question could also have matched. No
// rule is merged with another, no rule is consulted twice, and no default is consulted at all —
// when nothing matches, the caller reports the question as open.
func matchRule(question string, facts []Fact) (Rule, []Fact, bool) {
	for _, rule := range rules {
		if rule.Question != question {
			continue
		}
		matched, ok := matchNeeds(rule.Match, facts)
		if ok {
			return rule, matched, true
		}
	}
	return Rule{}, nil, false
}

// matchNeeds reports whether every need has at least one fact in the state it requires, and
// returns the facts that satisfied them in Match order, each need's own matches in the order the
// run reported its observations.
//
// The comparison is the exact triple — plus the need's label when it declares one — never the
// fact's derived state: a need states what the measurement layer had to report, and an observation
// whose triple the vocabulary forbids must satisfy no need rather than inherit the state the
// mapping assigns to it. A need with an empty label matches any observation of its probe, which is
// the semantics every earlier row has; a labeled need matches only the observation reported under
// that label, so two observations sharing one triple are still distinguishable.
func matchNeeds(needs []Need, facts []Fact) ([]Fact, bool) {
	var matched []Fact
	for _, need := range needs {
		satisfied := false
		for _, fact := range facts {
			if fact.Probe != need.Probe || fact.Resolution != need.Resolution || fact.Verdict != need.Verdict {
				continue
			}
			if need.Label != "" && fact.Observation.Label != need.Label {
				continue
			}
			matched = append(matched, fact)
			satisfied = true
		}
		if !satisfied {
			return nil, false
		}
	}
	return matched, true
}

// questions returns the questions the table declares, in declaration order, each once. It is the
// evaluation order Diagnose walks: every declared question is answered or reported open, and no
// question is answered twice.
func questions() []string {
	var declared []string
	for _, rule := range rules {
		if slices.Contains(declared, rule.Question) {
			continue
		}
		declared = append(declared, rule.Question)
	}
	return declared
}

// neededStates names the observable states the table declared for one question, in declaration
// order and without duplicates, one entry per observable: "<probe> <STATE>" for a need that matches
// any observation of its probe, and "<probe> (<label>) <STATE>" for a need that matches one labeled
// observation — the label is how a reader tells two states of one probe apart.
//
// It is what an open question reports when no rule of the question matched, so a reader can see
// which measurements the table looked for and did not get — design §5.2's no-fall-through rule
// made visible instead of silent.
func neededStates(question string) []string {
	var needed []string
	for _, rule := range rules {
		if rule.Question != question {
			continue
		}
		for _, need := range rule.Match {
			entry := needState(need)
			if slices.Contains(needed, entry) {
				continue
			}
			needed = append(needed, entry)
		}
	}
	return needed
}

// needState names one need the way an open question reports it: the probe, then the observation's
// label when the need carries one, then the matchable state the need's resolution and verdict pair
// maps to. A label-less need keeps the original "<probe> <STATE>" spelling, which the cases pin
// literally; a labeled need reports "<probe> (<label>) <STATE>", because two needs of one probe can
// map to the same state and only the label tells them apart. The mapping is StateOf — the same one
// facts.go uses — so a state the table needs and a state a fact carries are spelled identically.
func needState(need Need) string {
	state := StateOf(probe.Observation{Resolution: need.Resolution, Verdict: need.Verdict})
	if need.Label == "" {
		return need.Probe + " " + string(state)
	}
	return need.Probe + " (" + need.Label + ") " + string(state)
}
