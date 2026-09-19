package diagnosis

// This file is the rule-id vocabulary of design §3.6: the ordered fact-rule slots the rule table
// is built over, the derivation that turns one slot into one rule id, and the ordered list of
// every rule id the table declares.
//
// The scheme is a derivation, not a catalogue. A fact rule's id is
// `<PROBE_NAME_UPPERCASED_WITH_UNDERSCORES>_<STATE>`: the probe whose observation the rule
// states, then the matchable state the rule matched — `EGRESS_HUB_DIRECT_FAIL` for the
// `egress.hub.direct` failure state, `LOCAL_ENV_PASS` for the `local.env` pass state. The id
// base is the probe's name and never the question's name, because §3.6 derives the id from the
// observable: several fact questions carry a name of their own (`hub.reachability` answers for
// `egress.hub.direct`, `ssh.public_22` for `egress.ssh.known`) while every state of one probe
// shares one id base.
//
// The declarations below are the single home of the derived vocabulary, and rules.go builds one
// rule per slot declared here: an id without a rule and a rule without an id are therefore
// impossible to declare by accident. The coverage case enumerates probe.Registry() and
// AllStates() and compares them with AllRuleIDs, so a probe or a state that arrives without its
// slot fails the suite instead of silently leaving an observable unclassified.

import (
	"strings"
	"unicode"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// factQuestion is one registered probe's fact question: the probe whose observation the derived
// rules state, and the question those rules answer.
//
// The question is the human-facing label of the rules, and it is not always the probe's name.
// Design §5.2 names the fact question of `egress.ssh.known` `ssh.public_22`, of `egress.ssh.443`
// `ssh.public_443` and of `egress.hub.direct` `hub.reachability`. Where §5.2 gives a probe's own
// name to a hand-named question group (`local.sshd`, `tls.interception`, `tls.truststore`), the
// fact question carries a `.fact` suffix instead: these rules are declared first and would
// otherwise win the first-match race for a question name their owning slice has yet to fill, and
// a name reserved for another slice is never consumed here.
type factQuestion struct {
	probe    string
	question string
}

// factQuestions is the ordered declaration of the fact questions: one row per registered probe,
// in probe.Registry() order.
//
// Every registered probe has exactly one row, and the coverage case refuses both directions of
// drift: a probe the registry declares without a row here has no derived rules, and a row whose
// probe is not registered is an orphan the table must not carry.
var factQuestions = []factQuestion{
	{"local.env", "local.env"},
	{"local.sshd", "local.sshd.fact"},
	{"egress.hub.direct", "hub.reachability"},
	{"egress.ssh.known", "ssh.public_22"},
	{"egress.ssh.443", "ssh.public_443"},
	{"egress.cf.7844", "egress.cf.7844"},
	{"egress.cf.443", "egress.cf.443"},
	{"egress.quic", "egress.quic"},
	{"tls.interception", "tls.interception.fact"},
	{"tls.truststore", "tls.truststore.fact"},
}

// factSlot is one derived fact rule's slot: the matchable state the rule states, and the exact
// resolution and verdict pair a fact carries in that state.
type factSlot struct {
	state      State
	resolution probe.Resolution
	verdict    probe.Verdict
}

// factSlots is the ordered declaration of the derived fact rules of one probe, and it is
// deliberately not AllStates() order.
//
// Design §5.2's propagation rule says the weaker rule is the one that matches when a measurement
// could not be read as a confident answer, and the same paragraph makes first-match evaluation
// the whole mechanism — so the order below keeps the measurement layer's own reduction, fail
// above indeterminate above pass, which is what stops an absence from ever outranking a definite
// answer. The two absences are ordered by consequence: an attempt that produced nothing
// (unresolved) makes the run incomplete, while an attempt that was never made (not measured)
// does not. A probe that reported a pass beside an observation it could not answer therefore
// fires the weaker rule and names the observation that produced no answer, instead of letting a
// pass rule stand in for a probe that was not entirely measured (DEV-2, R-HR-NF-03).
var factSlots = []factSlot{
	{StateFail, probe.Measured, probe.Fail},
	{StateUnresolved, probe.Unresolved, probe.Indeterminate},
	{StateNotMeasured, probe.NotMeasured, probe.Indeterminate},
	{StatePass, probe.Measured, probe.Pass},
}

// The hand-named rule ids of design §5.2's derived question groups.
//
// §3.6 derives a fact question's id from the probe and the state; a derived question's id is named
// by hand instead, because its conclusion is about several observables at once and no single
// (probe, state) pair derives it. The names below are therefore a declaration rather than a
// derivation, and they live here, beside the derivation, because this file is the vocabulary:
// rules.go builds its rows on these names and AllRuleIDs reports them, so one hand-named id cannot
// be spelled two ways.
//
// The table of §5.2 is longer than this slice lands — the `tls.*`, `local.sshd` and `node.platform`
// groups belong to the slice that owns their conclusions — and a later slice declares its ids here
// in the same change that declares their rules, which is what keeps AllRuleIDs and the table in
// step: an id without a rule and a rule without an id are the two drift directions the case that
// compares them refuses.
const (
	// ruleSSHDestBlockedByPublicSSH is the port-versus-protocol conclusion of design §5.2's
	// `ssh.destination` group: a public SSH measurement passed, so the protocol is not blocked, and
	// the hub-directed measurement failed, so the destination is.
	ruleSSHDestBlockedByPublicSSH = "SSH_DEST_BLOCKED_BY_PUBLIC_SSH"
	// ruleSSHDestBlockedByPublicSSH443 is the same conclusion reached from the public measurement on
	// port 443 instead of port 22.
	ruleSSHDestBlockedByPublicSSH443 = "SSH_DEST_BLOCKED_BY_PUBLIC_SSH_443"
	// ruleSSHDestBlockUnestablishedPublicUnresolved is the weaker conclusion for a hub failure whose
	// public counterpart did not answer: what the hub failure means for SSH egress is not
	// established. It covers an attempted public measurement that produced no answer and one that
	// was never made; the conclusion's own wording names which.
	ruleSSHDestBlockUnestablishedPublicUnresolved = "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED"
	// ruleSSHDestBlockUnestablishedPublicFailed is the weaker conclusion when the public measurements
	// failed as well: destination-level and protocol-level blocks are not separable from this run.
	ruleSSHDestBlockUnestablishedPublicFailed = "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_FAILED"
	// ruleSSHNoDestBlockObserved is the conclusion of a hub that accepted a connection beside a
	// public SSH measurement that passed: no destination block was observed.
	ruleSSHNoDestBlockObserved = "SSH_NO_DEST_BLOCK_OBSERVED"
	// ruleSSHDestBlockNotAssessed is the conclusion of a hub measurement that produced no answer or
	// was never made: nothing is claimed about the hub. It is the default live case, because a run
	// with no `--hub` measures no hub.
	ruleSSHDestBlockNotAssessed = "SSH_DEST_BLOCK_NOT_ASSESSED"

	// ruleCFEdgeReachable is the conclusion of at least one measured TCP pass on a declared
	// Cloudflare edge endpoint, with no measured failure beside it.
	ruleCFEdgeReachable = "CF_EDGE_REACHABLE"
	// ruleCFEdgePartial is the conclusion of a split: some declared edge endpoints answered and some
	// did not.
	ruleCFEdgePartial = "CF_EDGE_PARTIAL"
	// ruleCFEdgeUnreachable is the conclusion of a run in which every measured edge endpoint failed
	// and no endpoint produced an absence.
	ruleCFEdgeUnreachable = "CF_EDGE_UNREACHABLE"
	// ruleCFEdgeUnresolved is the conclusion of no measured edge success beside at least one attempt
	// that produced no answer: whether the edge is reachable is not established.
	ruleCFEdgeUnresolved = "CF_EDGE_UNRESOLVED"

	// ruleCFHTTP2AdvisedQUICFailed is the HTTP/2 recommendation that rests on a failed datagram
	// measurement. It recommends and states that it does not enforce.
	ruleCFHTTP2AdvisedQUICFailed = "CF_HTTP2_ADVISED_QUIC_FAILED"
	// ruleCFHTTP2AdvisedQUICUnconfirmed is the same recommendation resting on a datagram path that was
	// never confirmed — an attempt that produced no answer, or a measurement that was never made. The
	// conclusion names the measurement it rests on.
	ruleCFHTTP2AdvisedQUICUnconfirmed = "CF_HTTP2_ADVISED_QUIC_UNCONFIRMED"
	// ruleCFNoHTTP2AdviceQUICUsable is the conclusion of a datagram measurement that drew a reply: no
	// downgrade is recommended.
	ruleCFNoHTTP2AdviceQUICUsable = "CF_NO_HTTP2_ADVICE_QUIC_USABLE"
	// ruleCFHTTP2AdvisoryNotAssessed is the conclusion of a run whose edge did not answer on TCP: no
	// HTTP/2 advice is given, because no measured TCP path could carry one.
	ruleCFHTTP2AdvisoryNotAssessed = "CF_HTTP2_ADVISORY_NOT_ASSESSED"
)

// handNamedRuleIDs is the ordered declaration of the hand-named ids, in the order rules.go declares
// their rows. It is what AllRuleIDs reports after the derived vocabulary, and it is ordered so the
// documented list can be read beside the table.
var handNamedRuleIDs = []string{
	ruleSSHDestBlockedByPublicSSH,
	ruleSSHDestBlockedByPublicSSH443,
	ruleSSHDestBlockUnestablishedPublicUnresolved,
	ruleSSHDestBlockUnestablishedPublicFailed,
	ruleSSHNoDestBlockObserved,
	ruleSSHDestBlockNotAssessed,
	ruleCFEdgeReachable,
	ruleCFEdgePartial,
	ruleCFEdgeUnreachable,
	ruleCFEdgeUnresolved,
	ruleCFHTTP2AdvisedQUICFailed,
	ruleCFHTTP2AdvisedQUICUnconfirmed,
	ruleCFNoHTTP2AdviceQUICUsable,
	ruleCFHTTP2AdvisoryNotAssessed,
}

// RuleID derives the rule id of one matchable state of one probe: the probe's stable name
// upper-cased with every non-alphanumeric character replaced by an underscore, then the state's
// own name. RuleID("egress.hub.direct", StateFail) is "EGRESS_HUB_DIRECT_FAIL" (design §3.6).
//
// The probe name is the base rather than the question's name on purpose: several fact questions
// carry a human-facing name of their own while sharing one id base with every other state of the
// same probe. A caller that needs to know whether an id is a derived one builds the id the same
// way and compares, so the derivation has one home and no consumer restates it.
func RuleID(probeName string, state State) string {
	return upperUnderscores(probeName) + "_" + string(state)
}

// upperUnderscores is the "<PROBE_NAME_UPPERCASED_WITH_UNDERSCORES>" half of §3.6's derivation.
// Dots are the separator probe names actually use ("tls.truststore"); every other non-alphanumeric
// character is mapped the same way rather than left in place, so the derivation is total for any
// name the registry declares.
func upperUnderscores(name string) string {
	var id strings.Builder
	id.Grow(len(name))
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			id.WriteRune(unicode.ToUpper(r))
			continue
		}
		id.WriteByte('_')
	}
	return id.String()
}

// AllRuleIDs returns every rule id the table declares, in declaration order, as a copy: the
// derived ids of every fact question, each in the four observable states, then the hand-named ids
// of §5.2's derived question groups, in the order ids.go and rules.go declare them.
//
// Declaration order is the table's order — the fact questions in registry order, the states
// weakest-observable first, then the hand-named groups — so the list is deterministic across runs
// and independent of map iteration. The returned slice is a copy, so a caller cannot reorder or
// shrink the contract.
//
// The list is the complete rule-id set of design §5.4's documentation contract: the doc's
// rule-id table is compared with exactly this set, and a later slice that declares a hand-named
// rule for a derived question registers that id here in the same change (adding a rule id is a
// contract change), which the case that compares Rules() with AllRuleIDs refuses to let pass
// unnoticed.
func AllRuleIDs() []string {
	ids := make([]string, 0, len(factQuestions)*len(factSlots)+len(handNamedRuleIDs))
	for _, question := range factQuestions {
		for _, slot := range factSlots {
			ids = append(ids, RuleID(question.probe, slot.state))
		}
	}
	return append(ids, handNamedRuleIDs...)
}
