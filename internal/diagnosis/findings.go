package diagnosis

// This file is what the reasoning layer says: one finding per question the table answered, one
// open question per question it could not answer, and the wording of every conclusion the table
// emits.
//
// The wording lives here and nowhere else. rules.go names which conclusion a rule emits and
// diagnose.go renders it against the facts that matched, so no conclusion string is spelled
// inline beside a rule, and the same conclusion cannot be worded two ways in two files.

import (
	"fmt"
	"strings"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// Finding is one question the table answered: the question, the rule that fired, the conclusion
// text, and the observables the conclusion rests on.
type Finding struct {
	// Question is the question this finding answers.
	Question string
	// Rule is the id of the rule that fired. PRD §5.3 requires the report to quote which rule
	// produced a conclusion, so the id travels beside the text instead of being recoverable only
	// from the table.
	Rule string
	// Conclusion is the rendered conclusion text.
	Conclusion string
	// DependsOn names the probes the conclusion rests on, in the rule's Match order: it is how a
	// reader tells what a conclusion was established from and what it would lose if one of those
	// measurements had been unresolved.
	DependsOn []string
	// Evidence carries the observations the fired rule matched, in Match order: the probe, the
	// matchable state, and the observation's own label, target, reason code and verbatim detail.
	//
	// It exists so a consumer that must name the measurement a conclusion was established from — a
	// rejection that has to quote the measured target and its port, for instance — reads structured
	// observation data instead of parsing the prose Conclusion, which no consumer may parse. A
	// finding that fired always carries at least one observation, because a rule with no satisfied
	// need cannot have matched.
	Evidence []Fact
}

// OpenQuestion is one question the table could not answer: no rule of the question matched the
// run's facts, so nothing was concluded about it.
//
// It is not an error and not a failure. Design §5.2 has no fall-through default: a question that
// matched nothing concludes nothing, and the honest output is the question together with the
// observable states the table needed and did not get, so a reader sees which measurement was
// missing instead of reading a conclusion the run cannot support.
type OpenQuestion struct {
	// Question is the question that stayed unanswered.
	Question string
	// NeededStates is the observable state of every need the question's rules declared, one entry
	// per (probe, state) pair, spelled "<probe> <STATE>", in declaration order and without
	// duplicates.
	NeededStates []string
}

// Diagnosis is the whole result of one reasoning pass: the questions the table answered, in
// declaration order, and the questions it could not.
//
// Neither list is a verdict on the run. A finding records what the measurements established and
// an open question records what they did not; neither is ever filled in from anything but the
// facts of the run that was reasoned over.
type Diagnosis struct {
	Findings      []Finding
	OpenQuestions []OpenQuestion
}

// Conclusion names one conclusion text.
//
// A derived fact conclusion carries the observation slot — the probe and the state — because the
// text has to name the observation's own target, reason and verbatim detail. A hand-named group
// conclusion carries the rule id instead: its text is about several observables at once, and the id
// is the name the table and the vocabulary already share, so the wording is keyed by the same
// string the finding reports and the two cannot drift apart. Choosing a conclusion is a table
// decision (rules.go); wording it is a decision of this file.
type Conclusion struct {
	probe string
	state State
	// named is the rule id of a hand-named group conclusion, empty for a derived fact conclusion.
	named string
}

// factConclusion names the plain-fact conclusion of one observable state of one probe: the
// conclusion of every derived fact rule.
func factConclusion(probeName string, state State) Conclusion {
	return Conclusion{probe: probeName, state: state}
}

// namedConclusion names one hand-named conclusion of §5.2's derived question groups by the rule id
// that emits it. The id is the name because it is the string the table, the id vocabulary and the
// payload already share: a group's conclusion and its rule id are one value here, so they cannot
// disagree about what fired.
func namedConclusion(id string) Conclusion {
	return Conclusion{named: id}
}

// render builds the conclusion text of a rule that matched, from the facts its Match selected.
//
// A derived fact conclusion states the first observation the rule matched — the observation the
// first-match evaluation selected — and the probe's other observations stay visible in the run's own
// observations and behind DependsOn: a derived fact rule states a measurement, it does not summarise
// a probe, and the hand-named questions of §5.2 are where a set of observations is interpreted. A
// hand-named group conclusion is rendered by namedConclusionText from the observables its own
// clauses matched, which is where a set of observations is interpreted.
func (c Conclusion) render(matched []Fact) string {
	if len(matched) == 0 {
		// Unreachable for a rule that matched: matchNeeds returns at least one fact per need and
		// no rule is declared without one. It returns the absence rather than a fallback sentence,
		// so a table that managed to match nothing would fail the case that asserts every
		// finding's conclusion carries its own measurement instead of quietly concluding something.
		return ""
	}
	if c.named != "" {
		return namedConclusionText(c.named, matched)
	}
	return factConclusionText(c.probe, c.state, matched[0].Observation)
}

// The wording of every conclusion a derived fact rule emits, one constant per observable state.
//
// They state the measurement and nothing more. A derived fact rule answers "what did this probe
// report", so its conclusion names the probe, the observation's own label, the target the
// observation names and the probe's verbatim detail; an absence names the probe's own reason
// code, which is where the missing input or capability is recorded. Interpreting a measurement —
// "outbound SSH is allowed and the hub address is blocked", "advise HTTP/2", "installing sshd is
// a later slice" — is the business of the hand-named questions of §5.2, and no such sentence is
// worded here.
const (
	// factPassText is the conclusion of a measured pass. Arguments: the observation's subject,
	// the target, and the probe's verbatim detail.
	factPassText = "%s: measured and passed on %s — %s"
	// factFailText is the conclusion of a measured failure. Arguments: the observation's subject,
	// the target, and the probe's verbatim detail. The target is carried verbatim, so the port a
	// dial target declares survives into the conclusion — design §5.2 requires the hub's failure
	// to name the target and its port, and a conclusion that dropped the port could not.
	factFailText = "%s: measured and failed on %s — %s"
	// factUnresolvedText is the conclusion of an attempt that produced no answer. Arguments: the
	// observation's subject, the target the attempt was made against, the probe's reason code and
	// its verbatim detail.
	factUnresolvedText = "%s: attempted and unresolved on %s (%s) — %s"
	// factNotMeasuredText is the conclusion of an attempt that was never made. Arguments: the
	// observation's subject, the probe's reason code — which names the missing input or capability
	// — and its verbatim detail. There is no target to name: a measurement that was not made has
	// none, and printing the absence as an address would invent the target the run never had.
	factNotMeasuredText = "%s: not measured (%s) — %s"
)

// factConclusionText renders the plain fact of one observation. The four states are the whole
// vocabulary of facts.go and every derived rule is built from one of them, so the four cases
// below are exhaustive for the table this file is written for.
func factConclusionText(probeName string, state State, observation probe.Observation) string {
	subject := factSubject(probeName, observation)
	switch state {
	case StatePass:
		return fmt.Sprintf(factPassText, subject, factTarget(observation), factDetail(observation))
	case StateFail:
		return fmt.Sprintf(factFailText, subject, factTarget(observation), factDetail(observation))
	case StateUnresolved:
		return fmt.Sprintf(factUnresolvedText, subject, factTarget(observation), string(observation.Reason), factDetail(observation))
	case StateNotMeasured:
		return fmt.Sprintf(factNotMeasuredText, subject, string(observation.Reason), factDetail(observation))
	}
	// Unreachable: State is the closed four-state set of facts.go and ids.go's factSlots declare
	// exactly those four. It renders nothing rather than a catch-all sentence, so a fifth state
	// that arrived without wording would fail the case asserting that every rule's conclusion
	// carries its own measurement, instead of quietly emitting a default.
	return ""
}

// factSubject names the observation a conclusion rests on: the probe's stable name, plus the
// observation's own label when it reported one ("egress.cf.443 (tcp 443 region2)"), so one
// conclusion can be told apart from its probe's other observations.
func factSubject(probeName string, observation probe.Observation) string {
	if observation.Label == "" {
		return probeName
	}
	return probeName + " (" + observation.Label + ")"
}

// factTarget is the target the observation names, carried verbatim. A not-measured observation
// has none, and the wording says so instead of printing an empty address.
func factTarget(observation probe.Observation) string {
	if observation.Target == "" {
		return "no target"
	}
	return observation.Target
}

// factDetail is the observation's own verbatim detail, which R-HR-07 requires to survive beside
// the reason code. When a probe reported none, the reason code stands in, so a conclusion never
// ends in silence.
func factDetail(observation probe.Observation) string {
	if observation.Detail == "" {
		return "reason " + string(observation.Reason)
	}
	return observation.Detail
}

// This section is the wording of §5.2's hand-named conclusions: what each of the group rules says
// when it fires, rendered from the observables its own row matched.
//
// Two disciplines shape every text below.
//
// First, a conclusion states what was measured and no more. It names the measurements that decided
// it, and it never fills a gap: where a rule's clause is an absence — a public SSH measurement that
// produced no answer, a hub measurement that was never made, a datagram path that was never
// confirmed — the absence is rendered as an absence, naming the probe, its reason code and its
// verbatim detail, and the sentence says which question that absence leaves open. No group text
// reports an absence as a block, as a pass, or as a fact about the far end (R-HR-NF-03).
//
// Second, no text claims an effect. The HTTP/2 conclusions recommend a transport setting and say so
// in those words; none of them claims that a fallback was applied, that a configuration was
// written, or that anything on the machine changed. The post-quantum trade-off the recommendation
// carries is worded where the recommendation is assembled (design §5.4 gives it one home in the
// transport layer), so it is stated once rather than twice with two chances to drift.
//
// The `local.sshd` and `node.platform` groups landed in this slice and keep the same boundary. The
// sshd wording never states what a configuration change would do: a divergence is quoted as the
// two configurations the probe measured, an absence as the absence the probe reported, and the
// positive conclusion only states what was found and measured. The node wording repeats the
// semantics the classification itself carries — upstream Herdr's Windows server as of 0.9.1, this
// tool's missing provisioning slices, detection and no change — and carries the version caveat in
// the supported conclusion's own sentence, because native Windows is classified supported while
// this run does not measure which Herdr version is installed. It decides no transport: whether any
// transport can reach the hub is the transport layer's decision, and no group text here claims
// otherwise. The two `tls.*`
// questions have no hand-named wording because this slice restated them to the derived fact ids,
// whose wording is the fact wording above, and `NODE_WSL2_SYSTEMD_ABSENT` has no wording here
// because this slice does not implement it.

// namedConclusionText renders one hand-named conclusion from the facts its rule matched. It is the
// single home of the group wording: rules.go names which conclusion it emits, exactly as for a
// derived fact rule.
//
// Every branch returns "" when a fact its wording needs is not among the matched ones — the same
// absence a derived conclusion reports — so a row whose conclusion cannot be rendered fails the
// cases that assert every finding carries its own measurement instead of quietly concluding
// something.
func namedConclusionText(id string, matched []Fact) string {
	switch id {
	case ruleSSHDestBlockedByPublicSSH, ruleSSHDestBlockedByPublicSSH443:
		return sshDestinationBlockedText(matched)
	case ruleSSHDestBlockUnestablishedPublicUnresolved:
		return sshDestinationUnestablishedText(matched)
	case ruleSSHDestBlockUnestablishedPublicFailed:
		return sshDestinationPublicFailedText(matched)
	case ruleSSHNoDestBlockObserved:
		return sshNoDestinationBlockText(matched)
	case ruleSSHDestBlockNotAssessed:
		return sshDestinationNotAssessedText(matched)
	case ruleCFEdgeReachable:
		return cloudflareEdgeReachableText(matched)
	case ruleCFEdgePartial:
		return cloudflareEdgePartialText(matched)
	case ruleCFEdgeUnreachable:
		return cloudflareEdgeUnreachableText(matched)
	case ruleCFEdgeUnresolved:
		return cloudflareEdgeUnresolvedText(matched)
	case ruleCFHTTP2AdvisedQUICFailed:
		return cloudflareHTTP2AdvisedQUICFailedText(matched)
	case ruleCFHTTP2AdvisedQUICUnconfirmed:
		return cloudflareHTTP2AdvisedQUICUnconfirmedText(matched)
	case ruleCFNoHTTP2AdviceQUICUsable:
		return cloudflareNoHTTP2AdviceQUICUsableText(matched)
	case ruleCFHTTP2AdvisoryNotAssessed:
		return cloudflareHTTP2AdvisoryNotAssessedText(matched)
	case ruleSSHDPresentConfigDivergent:
		return sshdPresentConfigDivergentText(matched)
	case ruleSSHDAbsent:
		return sshdAbsentText(matched)
	case ruleSSHDEffectiveConfigNotMeasured:
		return sshdEffectiveConfigNotMeasuredText(matched)
	case ruleSSHDPresentConfigured:
		return sshdPresentConfiguredText(matched)
	case ruleNodePlatformUnknown:
		return nodePlatformUnknownText(matched)
	case ruleNodePlatformSupported:
		return nodePlatformSupportedText(matched)
	}
	return ""
}

// publicSSHProbes are the two probes whose measured pass establishes that SSH egress is not blocked:
// PRD §1.1's rows 1 and 2. A conclusion of the `ssh.destination` group names which one carried the
// disambiguation and never assumes port 22.
var publicSSHProbes = []string{"egress.ssh.known", "egress.ssh.443"}

// factsInState returns the matched facts in one state, in the order first-match evaluation selected
// them. A group conclusion quotes the observables it matched through these accessors rather than by
// indexing the slice: a row may need two observables of one probe in different states, and a
// position in the slice is an artifact of Match order rather than a meaning.
func factsInState(matched []Fact, state State) []Fact {
	var facts []Fact
	for _, fact := range matched {
		if fact.State == state {
			facts = append(facts, fact)
		}
	}
	return facts
}

// firstFact returns the first matched fact of one probe in one state, and whether there is one.
func firstFact(matched []Fact, probeName string, state State) (Fact, bool) {
	for _, fact := range matched {
		if fact.Probe == probeName && fact.State == state {
			return fact, true
		}
	}
	return Fact{}, false
}

// firstFactInLabel returns the first matched fact of one probe, reported under one label, in one
// state. It is how a group conclusion quotes the exact observation it matched when one probe
// reported several: the label is the probe's own stable one, so the conclusion names the
// measurement rather than a position in the matched slice. The label is compared verbatim, exactly
// as matchNeeds compared it.
func firstFactInLabel(matched []Fact, probeName, label string, state State) (Fact, bool) {
	for _, fact := range matched {
		if fact.Probe == probeName && fact.Observation.Label == label && fact.State == state {
			return fact, true
		}
	}
	return Fact{}, false
}

// firstFactOfAny returns the first matched fact in one state whose probe is one of the named probes,
// in the order the facts were matched — which is the order the rule's Match declared them, and so
// the order its conclusion should quote them in.
func firstFactOfAny(matched []Fact, state State, probeNames ...string) (Fact, bool) {
	for _, fact := range matched {
		if fact.State != state {
			continue
		}
		for _, probeName := range probeNames {
			if fact.Probe == probeName {
				return fact, true
			}
		}
	}
	return Fact{}, false
}

// firstPublicFact returns the first matched public-SSH fact in one state: the public measurement that
// carried the disambiguation, whichever of the two probes reported it.
func firstPublicFact(matched []Fact, state State) (Fact, bool) {
	return firstFactOfAny(matched, state, publicSSHProbes...)
}

// firstAbsentFact returns the first matched absence of one probe: an attempted measurement that
// produced no answer, or a measurement that was never made. Unresolved is looked for before not
// measured, the declaration order of the fact slots (ids.go), so a rule that can match either
// reports the stronger absence.
func firstAbsentFact(matched []Fact, probeName string) (Fact, bool) {
	if fact, ok := firstFact(matched, probeName, StateUnresolved); ok {
		return fact, true
	}
	return firstFact(matched, probeName, StateNotMeasured)
}

// firstPublicAbsentFact returns the first matched public-SSH absence, unresolved before not measured
// for the same reason.
func firstPublicAbsentFact(matched []Fact) (Fact, bool) {
	if fact, ok := firstPublicFact(matched, StateUnresolved); ok {
		return fact, true
	}
	return firstPublicFact(matched, StateNotMeasured)
}

// describeFact renders one matched observation the way a group conclusion quotes it, by its own
// state. A measured observation is quoted by the probe, its label, the target it names and its
// verbatim detail; an absence is quoted by its stable reason code and its verbatim detail as well,
// because for an absence the code is where the missing input or capability is recorded (R-HR-07).
func describeFact(f Fact) string {
	if f.State == StateUnresolved || f.State == StateNotMeasured {
		if f.Observation.Target == "" {
			return fmt.Sprintf("%s (%s): %s — %s", f.Probe, f.Observation.Label, f.Observation.Reason, factDetail(f.Observation))
		}
		return fmt.Sprintf("%s (%s on %s): %s — %s", f.Probe, f.Observation.Label, f.Observation.Target, f.Observation.Reason, factDetail(f.Observation))
	}
	if f.Observation.Target == "" {
		return fmt.Sprintf("%s (%s): %s", f.Probe, f.Observation.Label, factDetail(f.Observation))
	}
	return fmt.Sprintf("%s (%s on %s): %s", f.Probe, f.Observation.Label, f.Observation.Target, factDetail(f.Observation))
}

// describeFacts joins several matched observations the way describeFact renders each, in the order
// first-match evaluation selected them.
func describeFacts(facts []Fact) string {
	descriptions := make([]string, 0, len(facts))
	for _, fact := range facts {
		descriptions = append(descriptions, describeFact(fact))
	}
	return strings.Join(descriptions, "; ")
}

// absenceExplanation states how a matched absence came about, in the words the conclusion uses: an
// attempt that produced no answer, or an attempt that was never made. The two are told apart by the
// observation's own resolution and never by its wording (DEV-2), and a conclusion that could not
// tell a reader which one happened would be reporting an absence as though it were the other.
func absenceExplanation(f Fact) string {
	if f.State == StateNotMeasured {
		return "was not made"
	}
	return "was attempted and produced no answer"
}

// sshDestinationBlockedText is the wording of `SSH_DEST_BLOCKED_BY_PUBLIC_SSH` and
// `SSH_DEST_BLOCKED_BY_PUBLIC_SSH_443`: PRD §1.1's conclusion, and the acceptance case of this
// change (R-HR-03, PRD §13).
//
// It names the public target that answered as an SSH server and the hub address that did not, with
// the port the hub measurement declared, and it states the block as a fact about the destination
// rather than about the protocol. The last clause draws the boundary the requirement draws: what the
// block means for any particular transport is the transport layer's decision, and the reasoning
// layer does not make it.
func sshDestinationBlockedText(matched []Fact) string {
	hub, ok := firstFact(matched, "egress.hub.direct", StateFail)
	if !ok {
		return ""
	}
	public, ok := firstPublicFact(matched, StatePass)
	if !ok {
		return ""
	}
	return fmt.Sprintf("outbound SSH is allowed; the hub address %s is blocked — %s, so SSH egress is not blocked on that path, while the hub-directed measurement failed: %s. The block is on the destination, not on the protocol; whether a transport can reach the hub through a different address is a transport decision and is not made here.",
		factTarget(hub.Observation), describeFact(public), describeFact(hub))
}

// sshDestinationUnestablishedText is the wording of
// `SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED`: the hub failed and the public measurement the
// disambiguation turns on produced no answer.
//
// The two absences are rendered differently on purpose. An attempt that produced no answer and an
// attempt that was never made are the same id here — §5.2 gives the row one id — and they are not
// the same fact, so the sentence says which one happened and names the probe, its reason code and
// its detail. The conclusion claims the hub failure and nothing about SSH egress: it never says
// "SSH is blocked", and it never says outbound SSH works.
func sshDestinationUnestablishedText(matched []Fact) string {
	hub, ok := firstFact(matched, "egress.hub.direct", StateFail)
	if !ok {
		return ""
	}
	absent, ok := firstPublicAbsentFact(matched)
	if !ok {
		return ""
	}
	return fmt.Sprintf("whether the hub address %s is blocked is not established: the hub-directed measurement failed — %s — and the public SSH measurement %s: %s. This run establishes the hub failure itself and nothing about SSH egress.",
		factTarget(hub.Observation), describeFact(hub), absenceExplanation(absent), describeFact(absent))
}

// sshDestinationPublicFailedText is the wording of `SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_FAILED`: the
// hub failed and both public measurements failed as well.
//
// Nothing was left unmeasured here, and the conclusion still does not claim a destination block: two
// failures on the public targets and one on the hub do not separate a block at the destination from
// one at the protocol or the port, and saying which it was would be a guess.
func sshDestinationPublicFailedText(matched []Fact) string {
	hub, ok := firstFact(matched, "egress.hub.direct", StateFail)
	if !ok {
		return ""
	}
	known, ok := firstFact(matched, "egress.ssh.known", StateFail)
	if !ok {
		return ""
	}
	secure, ok := firstFact(matched, "egress.ssh.443", StateFail)
	if !ok {
		return ""
	}
	return fmt.Sprintf("whether the hub address %s is blocked is not established: the public SSH measurements failed as well — %s; %s — and so did the hub-directed measurement — %s. With every SSH measurement of this run failing, a destination-level block cannot be separated from one at the protocol or the port level.",
		factTarget(hub.Observation), describeFact(known), describeFact(secure), describeFact(hub))
}

// sshNoDestinationBlockText is the wording of `SSH_NO_DEST_BLOCK_OBSERVED`: the hub answered and the
// public SSH measurement passed.
//
// It states the two measurements and the negative that follows from them. A block is never reported
// from this run because no measurement reported one.
func sshNoDestinationBlockText(matched []Fact) string {
	hub, ok := firstFact(matched, "egress.hub.direct", StatePass)
	if !ok {
		return ""
	}
	public, ok := firstFact(matched, "egress.ssh.known", StatePass)
	if !ok {
		return ""
	}
	return fmt.Sprintf("no destination block observed: the hub address %s accepted a TCP connection and the public SSH measurement answered — %s; %s. Outbound SSH works and the declared hub address is reachable on this run, so no measurement here reports a block on either.",
		factTarget(hub.Observation), describeFact(hub), describeFact(public))
}

// sshDestinationNotAssessedText is the wording of `SSH_DEST_BLOCK_NOT_ASSESSED`: the hub measurement
// produced no answer, or was never made.
//
// It is the default live case, because a run with no `--hub` measures no hub, and it is the
// conclusion that must claim the least: nothing is claimed about the hub at all, in either
// direction. The sentence says which absence happened, so a reader can tell "we tried and got
// nothing" from "we never asked".
func sshDestinationNotAssessedText(matched []Fact) string {
	hub, ok := firstAbsentFact(matched, "egress.hub.direct")
	if !ok {
		return ""
	}
	return fmt.Sprintf("nothing is claimed about the hub: the hub-directed measurement %s — %s. Whether the hub address is blocked is not assessed in this run, and no block is stated from an absence.",
		absenceExplanation(hub), describeFact(hub))
}

// cloudflareEdgeReachableText is the wording of `CF_EDGE_REACHABLE`: at least one declared edge
// endpoint answered, and none of the measured ones failed.
//
// The claim is exactly the one the measurement supports. It says the edge is reachable from the
// endpoint that answered and that this is a TCP reachability fact; both edge probes measure TCP
// reachability and nothing else (egress.go), so the conclusion does not report a tunnel, a TLS
// handshake or a Cloudflare account state.
func cloudflareEdgeReachableText(matched []Fact) string {
	passes := factsInState(matched, StatePass)
	if len(passes) == 0 {
		return ""
	}
	return fmt.Sprintf("the Cloudflare edge is reachable: %s. At least one declared edge endpoint answered on TCP, which is the whole of this conclusion: a measured reachability fact about that endpoint, not a statement about a tunnel.",
		describeFacts(passes))
}

// cloudflareEdgePartialText is the wording of `CF_EDGE_PARTIAL`: the declared edge endpoints
// disagree, some answering and some not.
//
// Both sides of the split are quoted, because the split is the conclusion: a reader must see which
// endpoints answered and which did not, and a text that named only one side would hide the fact the
// group exists to state (design D10, PRD §8.2).
func cloudflareEdgePartialText(matched []Fact) string {
	passes := factsInState(matched, StatePass)
	fails := factsInState(matched, StateFail)
	if len(passes) == 0 || len(fails) == 0 {
		return ""
	}
	return fmt.Sprintf("the Cloudflare edge is only partly reachable: %s answered on TCP while %s did not. The declared edge endpoints do not agree on reachability, so the edge is reachable from some of them and not from others.",
		describeFacts(passes), describeFacts(fails))
}

// cloudflareEdgeUnreachableText is the wording of `CF_EDGE_UNREACHABLE`: every edge endpoint this
// run measured failed, and none produced an absence.
//
// "Every measured endpoint" is stated in those words: an endpoint the run never measured is not a
// failure, and the conclusion does not claim one. The run's own coverage carries that gap.
func cloudflareEdgeUnreachableText(matched []Fact) string {
	fails := factsInState(matched, StateFail)
	if len(fails) == 0 {
		return ""
	}
	return fmt.Sprintf("the Cloudflare edge is unreachable: every measured edge endpoint failed — %s.",
		describeFacts(fails))
}

// cloudflareEdgeUnresolvedText is the wording of `CF_EDGE_UNRESOLVED`: no measured success, and at
// least one attempt that produced no answer.
//
// The conclusion is the absence of an answer and says so, naming the endpoint whose measurement
// produced nothing. It never reports the edge as unreachable — the attempt did not fail, it did not
// answer — and it never reports it as reachable (D8, R-HR-NF-03).
func cloudflareEdgeUnresolvedText(matched []Fact) string {
	unresolved := factsInState(matched, StateUnresolved)
	if len(unresolved) == 0 {
		return ""
	}
	return fmt.Sprintf("whether the Cloudflare edge is reachable is not established: %s produced no answer, and no edge endpoint was measured as reachable in this run.",
		describeFacts(unresolved))
}

// cloudflareHTTP2AdvisedQUICFailedText is the wording of `CF_HTTP2_ADVISED_QUIC_FAILED`: the datagram
// measurement failed and a TCP edge endpoint answered.
//
// It recommends the fallback and states that it recommends it. Nothing here claims that a fallback
// was applied or that a configuration was changed, because nothing in this slice applies one
// (R-HR-05); the post-quantum trade-off the recommendation carries is worded in the transport layer
// where the recommendation is assembled (design §5.4).
func cloudflareHTTP2AdvisedQUICFailedText(matched []Fact) string {
	quic := factsInState(matched, StateFail)
	edge := factsInState(matched, StatePass)
	if len(quic) == 0 || len(edge) == 0 {
		return ""
	}
	return fmt.Sprintf("recommend forcing the Cloudflare tunnel transport to HTTP/2: the UDP datagram measurement to the edge failed — %s — while the edge answered on TCP — %s. This is a recommendation only: this slice reports the measurement and changes nothing.",
		describeFacts(quic), describeFacts(edge))
}

// cloudflareHTTP2AdvisedQUICUnconfirmedText is the wording of `CF_HTTP2_ADVISED_QUIC_UNCONFIRMED`: the
// datagram measurement produced no answer, or was never made, and a TCP edge endpoint answered.
//
// The conclusion says which absence happened and names every measurement it rests on — the probe,
// each endpoint, the reason code and the verbatim detail — because the recommendation is made from
// the absence of a confirmation, not from a measured block. A silent UDP socket proves nothing about
// QUIC (design D8, RG-5), and a conclusion that reported the silence as a block would make exactly
// the wrong claim the product exists to avoid. The two absences are rendered differently for the same
// reason they are elsewhere in this file: an attempt that produced no answer and an attempt that was
// never made are not the same fact, and the sentence says which one happened.
func cloudflareHTTP2AdvisedQUICUnconfirmedText(matched []Fact) string {
	edge := factsInState(matched, StatePass)
	if len(edge) == 0 {
		return ""
	}
	if unresolved := factsInState(matched, StateUnresolved); len(unresolved) != 0 {
		return fmt.Sprintf("recommend forcing the Cloudflare tunnel transport to HTTP/2 conservatively: the UDP datagram measurement to the edge was attempted and produced no answer — %s — while the edge answered on TCP — %s. The recommendation rests on a datagram path that was never confirmed, not on a measured block, and it is a recommendation only: this slice reports the measurement and changes nothing.",
			describeFacts(unresolved), describeFacts(edge))
	}
	if notMeasured := factsInState(matched, StateNotMeasured); len(notMeasured) != 0 {
		return fmt.Sprintf("recommend forcing the Cloudflare tunnel transport to HTTP/2 conservatively: the UDP datagram measurement to the edge was not made — %s — while the edge answered on TCP — %s. The recommendation rests on a datagram path that was never confirmed, not on a measured block, and it is a recommendation only: this slice reports the measurement and changes nothing.",
			describeFacts(notMeasured), describeFacts(edge))
	}
	return ""
}

// cloudflareNoHTTP2AdviceQUICUsableText is the wording of `CF_NO_HTTP2_ADVICE_QUIC_USABLE`: the
// datagram measurement drew a reply.
//
// No downgrade is recommended, and the conclusion is careful about what the reply proves: the probe's
// declared question is narrow (does any datagram come back?), so the reply establishes that the
// datagram was not silently dropped and nothing more (design D8).
func cloudflareNoHTTP2AdviceQUICUsableText(matched []Fact) string {
	passes := factsInState(matched, StatePass)
	if len(passes) == 0 {
		return ""
	}
	return fmt.Sprintf("no HTTP/2 downgrade is recommended: the UDP datagram measurement to the edge drew a reply — %s — and that probe establishes only that the datagram was not silently dropped, so no measurement here justifies forcing HTTP/2.",
		describeFacts(passes))
}

// cloudflareHTTP2AdvisoryNotAssessedText is the wording of `CF_HTTP2_ADVISORY_NOT_ASSESSED`: the edge
// itself did not answer on TCP, was never measured, or produced no answer.
//
// No advice is given, and the conclusion says why rather than staying silent: an HTTP/2 recommendation
// is a recommendation about a path that has to reach the edge, and no measured TCP path did
// (R-HR-05). The absence is rendered as the absence it is — the three branches are the edge's own
// measured failure, its unanswered attempt, and its unattempted measurement.
func cloudflareHTTP2AdvisoryNotAssessedText(matched []Fact) string {
	if fails := factsInState(matched, StateFail); len(fails) != 0 {
		return fmt.Sprintf("no HTTP/2 advice is given: the Cloudflare edge did not answer on TCP — %s — so there is no measured TCP path an HTTP/2 recommendation could rest on.",
			describeFacts(fails))
	}
	if unresolved := factsInState(matched, StateUnresolved); len(unresolved) != 0 {
		return fmt.Sprintf("no HTTP/2 advice is given: the Cloudflare edge measurement produced no answer — %s — so whether an HTTP/2 path is needed is not established in this run.",
			describeFacts(unresolved))
	}
	if notMeasured := factsInState(matched, StateNotMeasured); len(notMeasured) != 0 {
		return fmt.Sprintf("no HTTP/2 advice is given: the Cloudflare edge was not measured — %s — so whether an HTTP/2 path is needed was not assessed in this run.",
			describeFacts(notMeasured))
	}
	return ""
}

// sshdPresentConfigDivergentText is the wording of `SSHD_PRESENT_CONFIG_DIVERGENT`: the written
// configuration and the configuration in force were both measured and they disagree (R-HR-18).
//
// The conclusion quotes the matched observation's verbatim detail, which carries both
// configurations, and states the divergence as the measurement it is. It is never worded as a
// success: the row exists precisely so a configuration that disagrees with the written file cannot
// reach the positive conclusion.
func sshdPresentConfigDivergentText(matched []Fact) string {
	config, ok := firstFactInLabel(matched, "local.sshd", "effective config", StateFail)
	if !ok {
		return ""
	}
	return fmt.Sprintf("the sshd effective configuration differs from the written one: %s. This is a measured divergence and not a success: the configuration in force is not the one the written file declares.",
		describeFact(config))
}

// sshdAbsentText is the wording of `SSHD_ABSENT`: no sshd binary is present at the documented path.
//
// The absence is the probe's own measured negative, quoted with its label, target and verbatim
// detail, and the sentence states the slice boundary the probe's detail already carries: installing
// sshd is work owned by a later slice and nothing was changed. The row matches the binary
// observation's own label, so this text is unreachable for a stopped service — an installed binary
// can never be reported as absent, and a stopped service is reported by its own observation and its
// own fact.
func sshdAbsentText(matched []Fact) string {
	binary, ok := firstFactInLabel(matched, "local.sshd", "binary present", StateFail)
	if !ok {
		return ""
	}
	return fmt.Sprintf("no sshd binary is present at the documented path: %s. Installing sshd is not part of this run but work owned by a later slice, and nothing was changed.",
		describeFact(binary))
}

// sshdEffectiveConfigNotMeasuredText is the wording of `SSHD_EFFECTIVE_CONFIG_NOT_MEASURED`: the
// configuration in force was not measured.
//
// The conclusion names the excluded capability through the observation's own reason code and
// verbatim detail and states the boundary: no claim is made about the configuration in force. It is
// the default live case, because the zero-execution boundary excludes `sshd -T`, and it must never
// be read as a configured or a divergent sshd.
func sshdEffectiveConfigNotMeasuredText(matched []Fact) string {
	config, ok := firstFactInLabel(matched, "local.sshd", "effective config", StateNotMeasured)
	if !ok {
		return ""
	}
	return fmt.Sprintf("the sshd effective configuration was not measured: %s. No claim is made about the configuration in force in this run.",
		describeFact(config))
}

// sshdPresentConfiguredText is the wording of `SSHD_PRESENT_CONFIGURED`: the binary is present and
// the configuration in force was measured and agrees with the written one.
//
// It is the group's positive conclusion, and it quotes both matched observations: the presence of
// the binary and the agreement the probe measured, which carries the two configurations verbatim.
// The service state is deliberately not part of this conclusion — a stopped service does not
// change either measurement, and the probe reports it as its own fact.
func sshdPresentConfiguredText(matched []Fact) string {
	binary, ok := firstFactInLabel(matched, "local.sshd", "binary present", StatePass)
	if !ok {
		return ""
	}
	config, ok := firstFactInLabel(matched, "local.sshd", "effective config", StatePass)
	if !ok {
		return ""
	}
	return fmt.Sprintf("the sshd binary is present and its effective configuration was measured and agrees with the written one: %s; %s. This run reports the measurement and changes nothing.",
		describeFact(binary), describeFact(config))
}

// nodePlatformUnknownText is the wording of `NODE_PLATFORM_UNKNOWN`: the signals matched no
// supported classification, so no platform is assumed.
//
// The conclusion names the missing classification through the observation's own reason code and
// verbatim detail, which quotes the signals the seam reported. It is the absence of a
// classification and never a default guess (R-HR-29).
func nodePlatformUnknownText(matched []Fact) string {
	classification, ok := firstFact(matched, "local.env", StateUnresolved)
	if !ok {
		return ""
	}
	return fmt.Sprintf("the node platform is unknown: %s. No platform is assumed and no default is guessed.",
		describeFact(classification))
}

// nodePlatformSupportedText is the wording of `NODE_PLATFORM_SUPPORTED`: `local.env` measured one of
// the supported classifications.
//
// The observation's target carries the classification and the architecture, and so does its own
// detail; the conclusion splits the target with `probe.SplitNodePlatformIdentity` — the inverse of
// the constructor the probe used — so the two halves are quoted as the values they are rather than
// as a string. A target the splitter does not recognise is quoted verbatim rather than guessed at.
// The sentence states the boundary: this is a detection, and nothing was changed.
//
// Native Windows is supported as of Herdr 0.9.1, and the probe does not measure the node's Herdr
// version, so the conclusion carries the caveat in its own sentence: the platform can host a
// supported server as of 0.9.1, and a node running an older server cannot host a saved-machine
// connection. The caveat is stated here as well as in the classification's detail because the
// conclusion is what a reader and an agent act on, and a pass must be as earned as a failure.
func nodePlatformSupportedText(matched []Fact) string {
	classification, ok := firstFact(matched, "local.env", StatePass)
	if !ok {
		return ""
	}
	if platform, arch, split := probe.SplitNodePlatformIdentity(classification.Observation.Target); split {
		if platform == probe.NodePlatformWindowsNative {
			return fmt.Sprintf("the node platform is supported: classification %q, architecture %q — %s. This classification measures the platform and not the installed Herdr version: a node running Herdr older than 0.9.1 cannot host a saved-machine connection. This run detects and reports the environment; it changes nothing.",
				platform, arch, describeFact(classification))
		}
		return fmt.Sprintf("the node platform is supported: classification %q, architecture %q — %s. This run detects and reports the environment; it changes nothing.",
			platform, arch, describeFact(classification))
	}
	return fmt.Sprintf("the node platform is supported: %s. This run detects and reports the environment; it changes nothing.",
		describeFact(classification))
}
