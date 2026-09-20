package diagnosis_test

// This file is the first suite of the reasoning layer: the measurement→reasoning bridge
// of design §7 (`facts.go`) and DEV-2.
//
// The cases here cover only the accessors — turning one run's `[]probe.Result` into
// matchable facts — and nothing about rules, conclusions or verdicts. Two properties are
// the reason the accessors exist at all:
//
//   - a probe that reports several observations exposes every observation's own state,
//     so a run that measured two Cloudflare regions, or an sshd binary beside its
//     effective configuration, does not lose one half behind the aggregate; and
//   - an observation that was never measured keeps its own not-measured state instead of
//     being folded into a sibling's failure or into a pass. A measurement that was not
//     made is not evidence for anything.
//
// The fixtures are hand-written results, which is exactly what the accessors are supposed
// to be testable against: no probe runs, no seam is injected, and no CLI exists. The
// results are reduced through `probe.Aggregate`, the measurement layer's own reduction,
// so a fixture is a run in miniature rather than a second opinion about verdicts.

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// observation builds one observation a case is about. Label, target, reason and detail
// are realistic on purpose: the fixtures should read like measurements rather than like
// anonymous structs, and the accessor cases assert that each of them survives the bridge
// unchanged.
func observation(label, target string, resolution probe.Resolution, verdict probe.Verdict, reason probe.ReasonCode, detail string) probe.Observation {
	return probe.Observation{
		Label:      label,
		Target:     target,
		Resolution: resolution,
		Verdict:    verdict,
		Reason:     reason,
		Detail:     detail,
	}
}

// result builds one probe's result from its observations, reducing the verdict and the
// reason through the measurement layer's own `Aggregate`, so a fixture describes what a
// probe would have reported without restating the reduction here.
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

// factsFixture is the run the accessor cases share: one probe that reports two regions
// with different outcomes (so the aggregate hides one of them), and one probe whose
// observations mix a measured failure with a capability this run was never given.
func factsFixture() []probe.Result {
	return []probe.Result{
		result("egress.cf.443", probe.ProbeEgress,
			observation("tcp 443 region1", "region1.v2.argotunnel.com:443", probe.Measured, probe.Pass, probe.ReasonOK,
				"tcp region1.v2.argotunnel.com:443: connected"),
			observation("tcp 443 region2", "region2.v2.argotunnel.com:443", probe.Measured, probe.Fail, probe.ReasonConnRefused,
				"dial tcp region2.v2.argotunnel.com:443: connect: connection refused"),
		),
		result("local.sshd", probe.ProbeLocal,
			observation("sshd binary", "/usr/sbin/sshd", probe.Measured, probe.Fail, probe.ReasonSSHDAbsent,
				"/usr/sbin/sshd: no such file; installing it is a later slice"),
			observation("effective sshd config", "", probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded,
				"sshd -T: no command runner is injected for this run"),
			observation("sshd service", "ssh.service", probe.Measured, probe.Pass, probe.ReasonOK,
				"systemctl is-active ssh.service: active"),
		),
	}
}

// TestFactsExposeEveryObservationOfAMultiObservationProbe is DEV-2's own case: every
// observation of every result becomes its own matchable fact, in the order the probe
// reported them, carrying the observation's own resolution, verdict, reason and detail —
// including the passing half of a probe whose aggregate is a failure, and the
// not-measured half of a probe whose aggregate is a failure. Folding either half into the
// aggregate would make the reasoning layer match on something the probe never measured.
func TestFactsExposeEveryObservationOfAMultiObservationProbe(t *testing.T) {
	results := factsFixture()

	facts := diagnosis.Facts(results)
	if len(facts) != 5 {
		t.Fatalf("Facts returned %d facts for 2 results holding 5 observations, want 5", len(facts))
	}

	// The first result aggregates to a failure, because one region was refused. The
	// other region was still measured, and it still passed: a fact per observation is
	// the only way that survives.
	if results[0].Verdict != probe.Fail {
		t.Fatalf("the fixture's first result aggregates to %q, want a failure", results[0].Verdict)
	}
	if facts[0].Probe != "egress.cf.443" || facts[1].Probe != "egress.cf.443" {
		t.Errorf("the first two facts name the probes %q and %q, want both from egress.cf.443", facts[0].Probe, facts[1].Probe)
	}
	if facts[0].State != diagnosis.StatePass {
		t.Errorf("the passing region's fact state = %q, want %q: the aggregate's failure must not swallow a measured pass", facts[0].State, diagnosis.StatePass)
	}
	if facts[1].State != diagnosis.StateFail {
		t.Errorf("the refused region's fact state = %q, want %q", facts[1].State, diagnosis.StateFail)
	}

	// The second result also aggregates to a failure — the binary is absent — and its
	// not-measured observation keeps its own state rather than borrowing that failure.
	if results[1].Verdict != probe.Fail {
		t.Fatalf("the fixture's second result aggregates to %q, want a failure", results[1].Verdict)
	}
	if facts[2].State != diagnosis.StateFail {
		t.Errorf("the absent-binary fact state = %q, want %q", facts[2].State, diagnosis.StateFail)
	}
	if facts[3].State != diagnosis.StateNotMeasured {
		t.Errorf("the excluded-capability fact state = %q, want %q", facts[3].State, diagnosis.StateNotMeasured)
	}
	if facts[4].State != diagnosis.StatePass {
		t.Errorf("the running-service fact state = %q, want %q", facts[4].State, diagnosis.StatePass)
	}

	// Every fact is a faithful view of one observation: the same triple, the same state
	// the map assigns, and the same wording, target, label and reason code, so a
	// conclusion can quote the measurement without re-reading the results.
	want := []struct {
		probe          string
		resolution     probe.Resolution
		verdict        probe.Verdict
		state          diagnosis.State
		label          string
		target         string
		reason         probe.ReasonCode
		detailContains string
	}{
		{"egress.cf.443", probe.Measured, probe.Pass, diagnosis.StatePass, "tcp 443 region1", "region1.v2.argotunnel.com:443", probe.ReasonOK, "connected"},
		{"egress.cf.443", probe.Measured, probe.Fail, diagnosis.StateFail, "tcp 443 region2", "region2.v2.argotunnel.com:443", probe.ReasonConnRefused, "connection refused"},
		{"local.sshd", probe.Measured, probe.Fail, diagnosis.StateFail, "sshd binary", "/usr/sbin/sshd", probe.ReasonSSHDAbsent, "later slice"},
		{"local.sshd", probe.NotMeasured, probe.Indeterminate, diagnosis.StateNotMeasured, "effective sshd config", "", probe.ReasonCapabilityExcluded, "no command runner"},
		{"local.sshd", probe.Measured, probe.Pass, diagnosis.StatePass, "sshd service", "ssh.service", probe.ReasonOK, "active"},
	}
	for i, tc := range want {
		fact := facts[i]
		if fact.Probe != tc.probe || fact.Resolution != tc.resolution || fact.Verdict != tc.verdict {
			t.Errorf("facts[%d] = (%q, %q, %q), want (%q, %q, %q)", i, fact.Probe, fact.Resolution, fact.Verdict, tc.probe, tc.resolution, tc.verdict)
		}
		if fact.State != tc.state {
			t.Errorf("facts[%d].State = %q, want %q", i, fact.State, tc.state)
		}
		if fact.State != diagnosis.StateOf(fact.Observation) {
			t.Errorf("facts[%d].State = %q, StateOf reports %q: the two must agree", i, fact.State, diagnosis.StateOf(fact.Observation))
		}
		if fact.Observation.Resolution != fact.Resolution || fact.Observation.Verdict != fact.Verdict {
			t.Errorf("facts[%d] does not carry its observation's own resolution and verdict: %+v", i, fact.Observation)
		}
		if fact.Observation.Label != tc.label || fact.Observation.Target != tc.target || fact.Observation.Reason != tc.reason {
			t.Errorf("facts[%d] observation = (%q, %q, %q), want (%q, %q, %q)", i,
				fact.Observation.Label, fact.Observation.Target, fact.Observation.Reason, tc.label, tc.target, tc.reason)
		}
		if !strings.Contains(fact.Observation.Detail, tc.detailContains) {
			t.Errorf("facts[%d] detail %q does not carry %q", i, fact.Observation.Detail, tc.detailContains)
		}
	}
}

// TestFactsForSelectsOneProbesOwnObservations covers the per-probe accessor the rule
// table reads: one probe's facts, in the order the probe reported them, and nothing from
// another probe. A probe the run did not report yields no facts rather than a fabricated
// not-measured one — the accessors report what a run measured, and never invent a
// measurement.
func TestFactsForSelectsOneProbesOwnObservations(t *testing.T) {
	results := factsFixture()

	sshd := diagnosis.FactsFor(results, "local.sshd")
	if len(sshd) != 3 {
		t.Fatalf("FactsFor(local.sshd) returned %d facts, want the probe's 3 observations", len(sshd))
	}
	for i, fact := range sshd {
		if fact.Probe != "local.sshd" {
			t.Errorf("local.sshd facts[%d] names the probe %q", i, fact.Probe)
		}
	}
	if sshd[0].Observation.Label != "sshd binary" || sshd[2].Observation.Label != "sshd service" {
		t.Errorf("FactsFor reordered or replaced the probe's observations: %+v", sshd)
	}

	if got := diagnosis.FactsFor(results, "egress.quic"); len(got) != 0 {
		t.Errorf("FactsFor(egress.quic) returned %d facts for a probe the run did not report, want none", len(got))
	}
}

// TestFactsMapEveryObservableStateAndNeverPromote pins the mapping from the measurement
// vocabulary to the four matchable states of design §3.6. The four legal cells are the
// whole vocabulary; a combination the vocabulary does not allow is reported as the
// absence of an answer rather than as a pass or a failure, because a fact the
// measurement layer could not have produced must never become a confident one.
func TestFactsMapEveryObservableStateAndNeverPromote(t *testing.T) {
	cases := []struct {
		name       string
		resolution probe.Resolution
		verdict    probe.Verdict
		want       diagnosis.State
	}{
		{"a measured pass", probe.Measured, probe.Pass, diagnosis.StatePass},
		{"a measured failure", probe.Measured, probe.Fail, diagnosis.StateFail},
		{"an attempted but unresolved observation", probe.Unresolved, probe.Indeterminate, diagnosis.StateUnresolved},
		{"an attempt that was never made", probe.NotMeasured, probe.Indeterminate, diagnosis.StateNotMeasured},

		// The cells the vocabulary forbids: a definite verdict is only ever carried by
		// a measured observation. None of them may become a pass or a failure.
		{"a measured observation with no answer", probe.Measured, probe.Indeterminate, diagnosis.StateUnresolved},
		{"an unresolved observation claiming a pass", probe.Unresolved, probe.Pass, diagnosis.StateUnresolved},
		{"an unresolved observation claiming a failure", probe.Unresolved, probe.Fail, diagnosis.StateUnresolved},
		{"a not-measured observation claiming a pass", probe.NotMeasured, probe.Pass, diagnosis.StateUnresolved},
		{"a not-measured observation claiming a failure", probe.NotMeasured, probe.Fail, diagnosis.StateUnresolved},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := observation("fact", "target", tc.resolution, tc.verdict, probe.ReasonInternalError, "detail")
			got := diagnosis.StateOf(o)
			if got != tc.want {
				t.Errorf("StateOf(%q, %q) = %q, want %q", tc.resolution, tc.verdict, got, tc.want)
			}
			if tc.resolution == probe.NotMeasured && (got == diagnosis.StatePass || got == diagnosis.StateFail) {
				t.Errorf("a not-measured observation was mapped to %q", got)
			}
			if tc.resolution != probe.Measured && (got == diagnosis.StatePass || got == diagnosis.StateFail) {
				t.Errorf("an observation that measured nothing was mapped to the definite state %q", got)
			}
		})
	}

	// The four states are a closed set in declaration order, and every state a fact can
	// carry is in it: the rule-id vocabulary of design §3.6 is derived from exactly
	// these four, so a fifth state would be a contract change.
	states := diagnosis.AllStates()
	wantStates := []diagnosis.State{diagnosis.StatePass, diagnosis.StateFail, diagnosis.StateUnresolved, diagnosis.StateNotMeasured}
	if len(states) != len(wantStates) {
		t.Fatalf("AllStates() returned %d states, want the 4 of design §3.6: %v", len(states), states)
	}
	seen := map[diagnosis.State]bool{}
	for i, state := range states {
		if state != wantStates[i] {
			t.Errorf("AllStates()[%d] = %q, want %q in declaration order", i, state, wantStates[i])
		}
		if seen[state] {
			t.Errorf("AllStates()[%d] = %q is a duplicate", i, state)
		}
		seen[state] = true
	}
	for _, fact := range diagnosis.Facts(factsFixture()) {
		if !seen[fact.State] {
			t.Errorf("the fact for %q carries the state %q, which is not in the closed set %v", fact.Probe, fact.State, states)
		}
	}
}

// TestFactsInventNoMeasurement asserts the accessors report only what a run measured: a
// result that carries no observation contributes no fact, and a run with no result
// contributes none either. Synthesising a fact here would let the reasoning layer match
// on a measurement that never happened.
func TestFactsInventNoMeasurement(t *testing.T) {
	empty := result("egress.hub.direct", probe.ProbeEgress)
	empty.Verdict = probe.Indeterminate
	empty.Reason = probe.ReasonInternalError

	if got := diagnosis.Facts(nil); len(got) != 0 {
		t.Errorf("Facts(nil) returned %d facts, want none", len(got))
	}
	if got := diagnosis.Facts([]probe.Result{empty}); len(got) != 0 {
		t.Errorf("Facts reported %d facts for a result with no observations, want none", len(got))
	}
	if got := diagnosis.FactsFor([]probe.Result{empty}, "egress.hub.direct"); len(got) != 0 {
		t.Errorf("FactsFor reported %d facts for a probe that reported no observation, want none", len(got))
	}
}

// TestRuleIDsCoverEveryProbeState is design §3.6's coverage duty for the reasoning layer: every
// registered probe × every matchable state has exactly one rule id, and the list the reasoning
// layer reports has no duplicate and no orphan.
//
// The case enumerates the probes through probe.Registry() and the states through AllStates()
// rather than restating either, so a probe or a state that arrives without its derived id fails
// here instead of leaving an observable unclassified.
func TestRuleIDsCoverEveryProbeState(t *testing.T) {
	// The derivation itself, pinned with the design's own spellings: the id base is the probe's
	// name upper-cased with underscores for its separators, the suffix is the matchable state.
	// Changing the case, the separator or the suffix is a contract change and has to be an explicit
	// one.
	anchors := []struct {
		probe string
		state diagnosis.State
		want  string
	}{
		{"egress.hub.direct", diagnosis.StateFail, "EGRESS_HUB_DIRECT_FAIL"},
		{"local.env", diagnosis.StatePass, "LOCAL_ENV_PASS"},
		{"egress.ssh.443", diagnosis.StateUnresolved, "EGRESS_SSH_443_UNRESOLVED"},
		{"tls.truststore", diagnosis.StateNotMeasured, "TLS_TRUSTSTORE_NOT_MEASURED"},
	}
	for _, anchor := range anchors {
		if got := diagnosis.RuleID(anchor.probe, anchor.state); got != anchor.want {
			t.Errorf("RuleID(%q, %q) = %q, want %q", anchor.probe, anchor.state, got, anchor.want)
		}
	}

	probes := probe.Registry()
	states := diagnosis.AllStates()
	ids := diagnosis.AllRuleIDs()

	// AllRuleIDs is reported in declaration order: the fact questions in registry order, and
	// inside a question the observable states in the order the rules are declared — which is the
	// order first-match evaluation reads, so the weaker observable is declared first and a pass
	// can never stand in for an observation that produced no answer (design §5.2, R-HR-NF-03).
	wantHead := []string{"LOCAL_ENV_FAIL", "LOCAL_ENV_UNRESOLVED", "LOCAL_ENV_NOT_MEASURED", "LOCAL_ENV_PASS"}
	if len(ids) < len(wantHead) {
		t.Fatalf("AllRuleIDs() reported %d rule ids, want at least the first question's %d", len(ids), len(wantHead))
	}
	for i, want := range wantHead {
		if ids[i] != want {
			t.Errorf("AllRuleIDs()[%d] = %q, want %q: the first registered probe's four states are declared in this order", i, ids[i], want)
		}
	}

	occurrences := map[string]int{}
	for _, id := range ids {
		occurrences[id]++
	}

	// Every registered probe × every state carries exactly one rule id, so no observable is left
	// unclassified by the reasoning layer.
	for _, entry := range probes {
		for _, state := range states {
			id := diagnosis.RuleID(entry.Name, state)
			switch occurrences[id] {
			case 1:
			case 0:
				t.Errorf("probe %q in state %q has no rule id: AllRuleIDs() does not report %q", entry.Name, state, id)
			default:
				t.Errorf("probe %q in state %q has %d rule ids, want exactly one: %q appears %d times", entry.Name, state, occurrences[id], id, occurrences[id])
			}
		}
	}

	// The vocabulary's other half is hand-named (design §3.6, §5.2): the ids of the derived question
	// groups, pinned literally here for the same reason the derivation's anchors are — a rule id is a
	// contract that reaches the payload and the documentation (§5.4), so changing one has to be an
	// explicit act. Each is carried by at least one rule of the table, and by more than one wherever
	// §5.2's `Requires` column is a disjunction: an id names a conclusion and each row is one clause
	// that establishes it (rules.go).
	handNamed := []string{
		"SSH_DEST_BLOCKED_BY_PUBLIC_SSH",
		"SSH_DEST_BLOCKED_BY_PUBLIC_SSH_443",
		"SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED",
		"SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_FAILED",
		"SSH_NO_DEST_BLOCK_OBSERVED",
		"SSH_DEST_BLOCK_NOT_ASSESSED",
		"CF_EDGE_REACHABLE",
		"CF_EDGE_PARTIAL",
		"CF_EDGE_UNREACHABLE",
		"CF_EDGE_UNRESOLVED",
		"CF_HTTP2_ADVISED_QUIC_FAILED",
		"CF_HTTP2_ADVISED_QUIC_UNCONFIRMED",
		"CF_NO_HTTP2_ADVICE_QUIC_USABLE",
		"CF_HTTP2_ADVISORY_NOT_ASSESSED",
		"SSHD_PRESENT_CONFIG_DIVERGENT",
		"SSHD_ABSENT",
		"SSHD_EFFECTIVE_CONFIG_NOT_MEASURED",
		"SSHD_PRESENT_CONFIGURED",
		"NODE_PLATFORM_REFUSED_NATIVE_WINDOWS",
		"NODE_PLATFORM_UNKNOWN",
		"NODE_PLATFORM_SUPPORTED",
	}

	// Both halves are checked the same way: every id AllRuleIDs reports has to be a derived id of a
	// registered probe in one of the four states or one of the hand-named ids above. Anything else is
	// a typo, a stale probe name or a state the vocabulary does not have, and it would reach the
	// payload and the documentation as a rule that can never fire.
	derived := map[string]bool{}
	for _, entry := range probes {
		for _, state := range states {
			derived[diagnosis.RuleID(entry.Name, state)] = true
		}
	}
	allowed := map[string]bool{}
	for id := range derived {
		allowed[id] = true
	}
	for _, id := range handNamed {
		allowed[id] = true
	}
	for _, id := range ids {
		if !allowed[id] {
			t.Errorf("AllRuleIDs() reports %q, which is neither the derived id of a registered probe in one of the four states nor one of the hand-named ids the vocabulary declares", id)
		}
	}
	if len(ids) != len(probes)*len(states)+len(handNamed) {
		t.Errorf("AllRuleIDs() reports %d rule ids for %d probes × %d states plus %d hand-named ids, want %d: the difference is a duplicate or an orphan",
			len(ids), len(probes), len(states), len(handNamed), len(probes)*len(states)+len(handNamed))
	}

	// A hand-named id appears exactly once — no duplicate — and is carried by the table, so the
	// vocabulary and the rules agree in both directions: an id without a rule would be documented and
	// unreachable, and the case that compares Rules() with AllRuleIDs refuses a rule without an id.
	carried := map[string]int{}
	for _, rule := range diagnosis.Rules() {
		carried[rule.ID]++
	}
	for _, id := range handNamed {
		if occurrences[id] != 1 {
			t.Errorf("AllRuleIDs() reports the hand-named id %q %d times, want exactly once", id, occurrences[id])
		}
		if carried[id] == 0 {
			t.Errorf("the vocabulary declares the hand-named id %q, but no rule of the table carries it", id)
		}
	}
}

// declaredQuestions returns the questions the table declares, in declaration order, each once. It is
// the list Diagnose walks, read from the table the engine evaluates rather than restated here, so a
// case that asserts what a run left open keeps covering the questions a later slice adds.
func declaredQuestions() []string {
	var questions []string
	for _, rule := range diagnosis.Rules() {
		if slices.Contains(questions, rule.Question) {
			continue
		}
		questions = append(questions, rule.Question)
	}
	return questions
}

// findingsFor returns every finding of one question, in the order Diagnose reported them. A
// question the diagnosis did not answer yields none, which is a result the fall-through cases
// assert rather than an error a caller has to interpret.
func findingsFor(got diagnosis.Diagnosis, question string) []diagnosis.Finding {
	var findings []diagnosis.Finding
	for _, finding := range got.Findings {
		if finding.Question == question {
			findings = append(findings, finding)
		}
	}
	return findings
}

// findingFor returns the one finding of a question, or fails the case: a question the table
// answered is answered exactly once, and a case that expected one finding must not silently accept
// none.
func findingFor(t *testing.T, got diagnosis.Diagnosis, question string) diagnosis.Finding {
	t.Helper()
	findings := findingsFor(got, question)
	if len(findings) != 1 {
		t.Fatalf("the diagnosis carries %d findings for question %q, want exactly one: %+v", len(findings), question, got.Findings)
	}
	return findings[0]
}

// openQuestionFor returns the one open question of one question, or fails the case. An open
// question is not a failure of the run: it is the table's explicit statement that nothing was
// concluded about that question, and the cases assert that the statement names what was needed.
func openQuestionFor(t *testing.T, got diagnosis.Diagnosis, question string) diagnosis.OpenQuestion {
	t.Helper()
	questions := openQuestionsFor(got, question)
	if len(questions) != 1 {
		t.Fatalf("the diagnosis carries %d open questions for question %q, want exactly one: %+v", len(questions), question, got.OpenQuestions)
	}
	return questions[0]
}

// openQuestionsFor returns every open question of one question, in the order Diagnose reported
// them. A question the table answered yields none.
func openQuestionsFor(got diagnosis.Diagnosis, question string) []diagnosis.OpenQuestion {
	var questions []diagnosis.OpenQuestion
	for _, open := range got.OpenQuestions {
		if open.Question == question {
			questions = append(questions, open)
		}
	}
	return questions
}

// TestNoFallThrough covers the mechanism's three refusals: the first matching rule wins and a
// later one is not consulted, a question no rule matched concludes nothing at all, and nothing
// falls through to a default conclusion.
//
// The cases are behavioural on purpose: they drive Diagnose with runs and assert what a caller
// sees, rather than reading the table's declaration and re-checking it against itself.
func TestNoFallThrough(t *testing.T) {
	t.Run("the first matching rule wins", func(t *testing.T) {
		// One probe reported a passing region beside a region it could not answer. Two of the
		// probe's derived rules can match this run, and exactly one of them may fire: the weaker
		// one, because design §5.2 makes an observation that produced no answer weaken the
		// conclusion instead of disappearing behind a pass (R-HR-NF-03).
		results := []probe.Result{
			result("egress.cf.443", probe.ProbeEgress,
				observation("tcp 443 region1", "region1.v2.argotunnel.com:443", probe.Measured, probe.Pass, probe.ReasonOK,
					"tcp region1.v2.argotunnel.com:443: connected"),
				observation("tcp 443 region2", "region2.v2.argotunnel.com:443", probe.Unresolved, probe.Indeterminate, probe.ReasonDNSUnresolved,
					"resolver timeout for region2.v2.argotunnel.com: no answer inside the budget"),
			),
		}

		got := diagnosis.Diagnose(results)
		// The case is about the probe's own derived fact rule. This run may also answer the hand-named
		// questions that read the same observations — the Cloudflare edge and HTTP/2 groups do, for
		// `egress.cf.443` — so the finding under test is selected by question rather than by position.
		finding := findingFor(t, got, "egress.cf.443")
		if want := diagnosis.RuleID("egress.cf.443", diagnosis.StateUnresolved); finding.Rule != want {
			t.Errorf("the split probe fired %q, want the weaker rule %q: a pass must not stand in for the observation that produced no answer", finding.Rule, want)
		}
		if !strings.Contains(finding.Conclusion, "resolver timeout") {
			t.Errorf("the conclusion %q does not name the observation that produced no answer", finding.Conclusion)
		}
		want := []string{"egress.cf.443"}
		if !reflect.DeepEqual(finding.DependsOn, want) {
			t.Errorf("the conclusion depends on %v, want %v", finding.DependsOn, want)
		}
	})

	t.Run("a question with no matching rule concludes nothing", func(t *testing.T) {
		// A run whose results carried no observation. Every question the table declares is
		// unanswered, and the honest output is the unanswered question: no finding, no default
		// conclusion, and no fall-through to the opposite answer. The expected set is read from the
		// table, so the derived fact questions and the hand-named groups of §5.2 are covered alike.
		got := diagnosis.Diagnose(nil)
		if len(got.Findings) != 0 {
			t.Errorf("a run with no result produced %d findings, want none: %+v", len(got.Findings), got.Findings)
		}
		want := declaredQuestions()
		if len(got.OpenQuestions) != len(want) {
			t.Fatalf("a run with no result produced %d open questions, want one per declared question (%d): %v", len(got.OpenQuestions), len(want), want)
		}
		for i, question := range want {
			if got.OpenQuestions[i].Question != question {
				t.Errorf("open questions[%d] = %q, want %q in declaration order", i, got.OpenQuestions[i].Question, question)
			}
		}
	})

	t.Run("the open question names the states the table needed", func(t *testing.T) {
		// The hub's fact question is the one design §5.2 names `hub.reachability`, and the states it
		// needs are the four observable states of `egress.hub.direct`, spelled "<probe> <STATE>",
		// declared weakest-observable first.
		got := diagnosis.Diagnose(nil)
		open := openQuestionFor(t, got, "hub.reachability")
		want := []string{
			"egress.hub.direct FAIL",
			"egress.hub.direct UNRESOLVED",
			"egress.hub.direct NOT_MEASURED",
			"egress.hub.direct PASS",
		}
		if !reflect.DeepEqual(open.NeededStates, want) {
			t.Errorf("the open question for %q needs %v, want %v", open.Question, open.NeededStates, want)
		}
	})

	t.Run("a probe the run did not report is an open question, not a conclusion", func(t *testing.T) {
		// A partial run: the results carry one probe only. Its question is answered and every other
		// question is reported as unanswered, rather than fitted with a not-measured fact the
		// reasoning layer would have had to invent (the accessors report what a run measured).
		results := []probe.Result{
			result("egress.quic", probe.ProbeProto,
				observation("cloudflare edge udp", "region1.v2.argotunnel.com:7844", probe.Measured, probe.Pass, probe.ReasonUDPResponseReceived,
					"udp region1.v2.argotunnel.com:7844: 1200 bytes received"),
			),
		}

		got := diagnosis.Diagnose(results)
		finding := findingFor(t, got, "egress.quic")
		if want := diagnosis.RuleID("egress.quic", diagnosis.StatePass); finding.Rule != want {
			t.Errorf("the run fired %q, want %q", finding.Rule, want)
		}

		// A question the table declares is either answered by a finding or left open: the run reported
		// one probe, so the questions its facts answer are answered and every other declared question is
		// open. The set is computed from the table and from the findings rather than counted, so the
		// hand-named groups are covered and a question a later slice adds cannot make this case silently
		// vacuous.
		answered := map[string]bool{}
		for _, gotFinding := range got.Findings {
			answered[gotFinding.Question] = true
		}
		var wantOpen []string
		for _, question := range declaredQuestions() {
			if !answered[question] {
				wantOpen = append(wantOpen, question)
			}
		}
		if len(got.OpenQuestions) != len(wantOpen) {
			t.Fatalf("the run left %d questions open, want the %d declared questions it did not answer: %v", len(got.OpenQuestions), len(wantOpen), wantOpen)
		}
		for i, question := range wantOpen {
			if got.OpenQuestions[i].Question != question {
				t.Errorf("open questions[%d] = %q, want %q in declaration order", i, got.OpenQuestions[i].Question, question)
			}
		}

		// The run reported one probe: a question whose measurement it never made stays open instead of
		// being fitted with a conclusion, which is the fall-through this mechanism refuses.
		if others := findingsFor(got, "ssh.public_22"); len(others) != 0 {
			t.Errorf("the run produced %d findings for a probe it did not report: %+v", len(others), others)
		}
	})

	t.Run("an observation the vocabulary forbids satisfies no need", func(t *testing.T) {
		// A not-measured observation claiming a definite verdict cannot come from the classification
		// table, and the mechanism must not read it as the state it claims nor as the state the
		// mapping would assign it: the triple satisfies no need, so the question stays open rather
		// than producing a conclusion no measurement supports.
		results := []probe.Result{
			result("egress.hub.direct", probe.ProbeEgress,
				observation("tcp 22", "203.0.113.10:22", probe.NotMeasured, probe.Pass, probe.ReasonInternalError,
					"a fact the classification table cannot produce"),
			),
		}

		got := diagnosis.Diagnose(results)
		if len(got.Findings) != 0 {
			t.Errorf("a forbidden observation produced %d findings, want none: %+v", len(got.Findings), got.Findings)
		}
		open := openQuestionFor(t, got, "hub.reachability")
		if len(open.NeededStates) != len(diagnosis.AllStates()) {
			t.Errorf("the open question needs %d states, want the four the table declares: %v", len(open.NeededStates), open.NeededStates)
		}
	})
}

// ruleFor returns the one table rule whose id is wanted, or fails the case. It reads the table the
// engine evaluates rather than a second declaration of it, so a case can never assert against a
// rule the engine does not have.
func ruleFor(t *testing.T, id string) diagnosis.Rule {
	t.Helper()
	var found []diagnosis.Rule
	for _, rule := range diagnosis.Rules() {
		if rule.ID == id {
			found = append(found, rule)
		}
	}
	if len(found) != 1 {
		t.Fatalf("the table declares %d rules with the id %q, want exactly one", len(found), id)
	}
	return found[0]
}

// observablesOf is the observable set a rule's Match rests on, built here from Match in Match
// order — the probes its needs name, each once. The cases compare Rule.DependsOn with this
// independently built set, so a rule whose DependsOn disagrees with its Match fails rather than
// being trusted.
func observablesOf(match []diagnosis.Need) []string {
	var probes []string
	for _, need := range match {
		if slices.Contains(probes, need.Probe) {
			continue
		}
		probes = append(probes, need.Probe)
	}
	return probes
}

// observationInState builds one realistic observation of one probe in one matchable state: a
// measured pass, a measured failure, an attempt that produced no answer, and an attempt that was
// never made. The per-id cases run these through the engine, so a rule that fired on the wrong
// observation is visible in the conclusion's own wording.
//
// No detail here repeats the target, which is deliberate: the cases assert the target reaches the
// conclusion through the observation's own target field, and a detail that spelled the address
// again would satisfy that assertion on the target's behalf.
func observationInState(t *testing.T, probeName string, state diagnosis.State) probe.Observation {
	t.Helper()
	const target = "region1.v2.argotunnel.com:443"
	label := probeName + " observation"
	switch state {
	case diagnosis.StatePass:
		return observation(label, target, probe.Measured, probe.Pass, probe.ReasonOK,
			"connection established inside the probe budget")
	case diagnosis.StateFail:
		return observation(label, target, probe.Measured, probe.Fail, probe.ReasonConnRefused,
			"connect: connection refused")
	case diagnosis.StateUnresolved:
		return observation(label, target, probe.Unresolved, probe.Indeterminate, probe.ReasonDNSUnresolved,
			"resolver timeout: no answer inside the probe budget")
	case diagnosis.StateNotMeasured:
		return observation(label, "", probe.NotMeasured, probe.Indeterminate, probe.ReasonInputMissingHub,
			"no --hub was supplied, so this measurement was not made")
	}
	t.Fatalf("the cases have no observation for the state %q", state)
	return probe.Observation{}
}

// concludedValues lists the values a plain-fact conclusion of one observation has to carry: the
// observation's label and its verbatim detail; the target the observation names and the port that
// target declares when the observation was measured; the probe's own reason code when the
// observation is an absence, which is where the missing input or capability is named; and the
// wording of the state, so a state's conclusion is its own and cannot be rendered as another
// state's. The phrases are pinned literally on purpose: the conclusion string is what PRD §5.3
// requires the report to quote, so rewording one is a contract change and has to be an explicit
// one. Anything a conclusion adds beyond these is interpretation, and interpretation is not a
// derived fact rule's business.
func concludedValues(state diagnosis.State, o probe.Observation) []string {
	values := []string{o.Label, o.Detail}
	switch state {
	case diagnosis.StatePass:
		values = append(values, o.Target, "measured and passed")
		if i := strings.LastIndex(o.Target, ":"); i >= 0 {
			values = append(values, o.Target[i:])
		}
	case diagnosis.StateFail:
		values = append(values, o.Target, "measured and failed")
		if i := strings.LastIndex(o.Target, ":"); i >= 0 {
			values = append(values, o.Target[i:])
		}
	case diagnosis.StateUnresolved:
		// An unresolved attempt was still made against a target, so its conclusion names the target
		// as well: an absence with no address would leave a reader unable to tell what was attempted.
		values = append(values, o.Target, string(o.Reason), "attempted and unresolved")
	case diagnosis.StateNotMeasured:
		values = append(values, string(o.Reason), "not measured")
	}
	return values
}

// TestRulesOneCasePerID is design §5.2's per-rule case: one case per derived fact rule id,
// asserting that the conclusion is the plain fact of that observation and that DependsOn is
// exactly the observable set the rule matched.
//
// The case walks the derived vocabulary the way the coverage case does — registered probes × the
// four states — and reads the rule the table declares for each id instead of restating the table,
// so it keeps covering the ids a later slice adds. It also closes the orphan check the vocabulary
// alone cannot make: an id without a rule can never fire, and a rule whose id the vocabulary does
// not report would be documented nowhere (design §5.4).
func TestRulesOneCasePerID(t *testing.T) {
	declared := map[string]int{}
	for _, id := range diagnosis.AllRuleIDs() {
		declared[id]++
	}
	for _, rule := range diagnosis.Rules() {
		if declared[rule.ID] != 1 {
			t.Errorf("the id vocabulary reports %d ids for the table's rule %q, want exactly one: an id the table declares without being reportable is an orphan", declared[rule.ID], rule.ID)
		}
	}

	// Design §5.2's DependsOn contract over every row of the table: DependsOn is exactly the set of
	// observables in Match, and no rule is declared without a need — a rule that needed nothing
	// could fire on a run that measured nothing, which is the fall-through this mechanism refuses.
	for _, rule := range diagnosis.Rules() {
		if rule.ID == "" || rule.Question == "" {
			t.Errorf("the table declares a row without an id or a question: %+v", rule)
		}
		if len(rule.Match) == 0 {
			t.Errorf("the rule %q declares no need, so it could match a run that measured nothing", rule.ID)
		}
		if want := observablesOf(rule.Match); !reflect.DeepEqual(rule.DependsOn, want) {
			t.Errorf("the rule %q depends on %v, want the observables of its Match %v", rule.ID, rule.DependsOn, want)
		}
	}

	// The fact questions design §5.2 names for the derived rules, pinned literally. `local.sshd`
	// answers its fact question under `local.sshd.fact` because §5.2 gives the probe's own name to
	// the hand-named group, and the suffix is a decision that has to be visible rather than an
	// accident of naming. The two `tls.*` questions keep their plain names because this slice
	// restated §5.2's hand-named tls ids to the derived ones, so no group shadows them.
	named := []struct {
		probe    string
		question string
	}{
		{"egress.ssh.known", "ssh.public_22"},
		{"egress.ssh.443", "ssh.public_443"},
		{"egress.hub.direct", "hub.reachability"},
		{"egress.quic", "egress.quic"},
		{"local.env", "local.env"},
		{"local.sshd", "local.sshd.fact"},
		{"tls.interception", "tls.interception"},
		{"tls.truststore", "tls.truststore"},
	}
	for _, want := range named {
		rule := ruleFor(t, diagnosis.RuleID(want.probe, diagnosis.StatePass))
		if rule.Question != want.question {
			t.Errorf("the fact question of %q is %q, want %q", want.probe, rule.Question, want.question)
		}
	}

	for _, entry := range probe.Registry() {
		for _, state := range diagnosis.AllStates() {
			t.Run(entry.Name+" in state "+string(state), func(t *testing.T) {
				wantID := diagnosis.RuleID(entry.Name, state)
				rule := ruleFor(t, wantID)

				// The rule rests on exactly this observable: one need, on this probe, whose triple maps
				// back through the measurement→state mapping to the state the id names.
				if len(rule.Match) != 1 || rule.Match[0].Probe != entry.Name {
					t.Fatalf("the rule %q rests on %+v, want exactly one need on %q", wantID, rule.Match, entry.Name)
				}
				need := rule.Match[0]
				if got := diagnosis.StateOf(probe.Observation{Resolution: need.Resolution, Verdict: need.Verdict}); got != state {
					t.Errorf("the need %+v maps to the state %q, want %q: a rule's observable and its id must be the same slot", need, got, state)
				}

				// One run that reported exactly this observation, and nothing else.
				measured := observationInState(t, entry.Name, state)
				got := diagnosis.Diagnose([]probe.Result{result(entry.Name, entry.Kind, measured)})

				finding := findingFor(t, got, rule.Question)
				if finding.Rule != wantID {
					t.Fatalf("the run in state %s fired %q, want %q", state, finding.Rule, wantID)
				}
				if finding.Conclusion == "" {
					t.Fatalf("the rule %q concluded nothing", wantID)
				}
				// The conclusion is the plain fact: it names the probe and carries the observation's own
				// label, target (or reason code), verbatim detail and state wording, and nothing else.
				if !strings.Contains(finding.Conclusion, entry.Name) {
					t.Errorf("the conclusion %q does not name the probe %q", finding.Conclusion, entry.Name)
				}
				for _, want := range concludedValues(state, measured) {
					if !strings.Contains(finding.Conclusion, want) {
						t.Errorf("the conclusion %q does not carry the observation's own %q", finding.Conclusion, want)
					}
				}

				// DependsOn is the matched observable set — here exactly the probe the rule needs — on the
				// finding and on the table row the finding came from.
				wantDepends := []string{entry.Name}
				if !reflect.DeepEqual(finding.DependsOn, wantDepends) {
					t.Errorf("the conclusion depends on %v, want %v", finding.DependsOn, wantDepends)
				}
				if !reflect.DeepEqual(rule.DependsOn, wantDepends) {
					t.Errorf("the rule %q depends on %v, want %v", wantID, rule.DependsOn, wantDepends)
				}

				// The question was answered, not left open: this run measured exactly the state the rule
				// needs, so no open question may remain for it and nothing else may have fired.
				if open := openQuestionsFor(got, rule.Question); len(open) != 0 {
					t.Errorf("the question %q stayed open for a run that measured its state: %+v", rule.Question, open)
				}
			})
		}
	}
}

// TestMixedObservationsFireTheWeakerRule is the consequence-order case: when one probe reported
// several observations, the derived rule that fires is the one for the weakest state present, in
// the declared order fail > unresolved > not measured > pass.
//
// The order is a decision, and it is the measurement layer's own: probe.Aggregate reduces a probe's
// observations with fail above indeterminate above pass so an absence can never outrank a definite
// answer, and the fact rules keep that order so a conclusion names the worst thing the probe
// reported rather than its best. Without it a pass would stand in for an observation that produced
// no answer, which is exactly the promotion R-HR-NF-03 refuses.
func TestMixedObservationsFireTheWeakerRule(t *testing.T) {
	cases := []struct {
		name   string
		states []diagnosis.State
		want   diagnosis.State
	}{
		{"a single measured state keeps its own rule", []diagnosis.State{diagnosis.StatePass}, diagnosis.StatePass},
		{"a failure beside a pass fires the failure", []diagnosis.State{diagnosis.StatePass, diagnosis.StateFail}, diagnosis.StateFail},
		{"a failure beside an absence fires the failure", []diagnosis.State{diagnosis.StateUnresolved, diagnosis.StateFail}, diagnosis.StateFail},
		{"an unresolved attempt beside a pass fires the unresolved attempt", []diagnosis.State{diagnosis.StatePass, diagnosis.StateUnresolved}, diagnosis.StateUnresolved},
		{"an unresolved attempt beside a not-measured one fires the unresolved attempt", []diagnosis.State{diagnosis.StateNotMeasured, diagnosis.StateUnresolved}, diagnosis.StateUnresolved},
		{"a not-measured capability beside a pass fires the not-measured capability", []diagnosis.State{diagnosis.StatePass, diagnosis.StateNotMeasured}, diagnosis.StateNotMeasured},
		{"absences alone never fire a definite rule", []diagnosis.State{diagnosis.StateUnresolved, diagnosis.StateNotMeasured}, diagnosis.StateUnresolved},
	}

	const probeName = "egress.cf.443"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observations := make([]probe.Observation, 0, len(tc.states))
			for i, state := range tc.states {
				o := observationInState(t, probeName, state)
				o.Label = fmt.Sprintf("tcp 443 region%d", i+1)
				observations = append(observations, o)
			}

			got := diagnosis.Diagnose([]probe.Result{result(probeName, probe.ProbeEgress, observations...)})
			// The case is about the probe's own derived fact rule. The run may also answer the hand-named
			// questions that read this probe's observations (the Cloudflare edge group reads
			// `egress.cf.443`), so the finding under test is selected by question rather than by position.
			finding := findingFor(t, got, probeName)
			if wantID := diagnosis.RuleID(probeName, tc.want); finding.Rule != wantID {
				t.Errorf("the probe reporting %v fired %q, want %q", tc.states, finding.Rule, wantID)
			}
			if tc.want != diagnosis.StatePass && finding.Rule == diagnosis.RuleID(probeName, diagnosis.StatePass) {
				t.Errorf("the pass rule won against %v", tc.states)
			}

			// The conclusion names the observation that decided it — the observation carrying the state
			// that fired — and not a sibling the run happened to report first.
			var deciding probe.Observation
			for _, o := range observations {
				if diagnosis.StateOf(o) == tc.want {
					deciding = o
					break
				}
			}
			if !strings.Contains(finding.Conclusion, deciding.Label) {
				t.Errorf("the conclusion %q does not name the observation that decided it (%q)", finding.Conclusion, deciding.Label)
			}
		})
	}
}

// TestDiagnoseReadsObservationsNotTheAggregate is DEV-2 at the reasoning layer: a result whose
// aggregate verdict contradicts its own observations must not move a single conclusion.
//
// The aggregate is the worst of a probe's observations, so reading it here would fold one region's
// measured pass into a sibling's failure, or a measured failure into a sibling's pass. The rules
// read facts — one per observation — and nothing else.
func TestDiagnoseReadsObservationsNotTheAggregate(t *testing.T) {
	t.Run("an aggregate that claims an absence cannot hide a measured pass", func(t *testing.T) {
		measured := observation("tcp 22", "ssh.github.com:22", probe.Measured, probe.Pass, probe.ReasonOK,
			"tcp ssh.github.com:22: connected")
		contradicting := probe.Result{
			Probe:        "egress.ssh.known",
			Kind:         probe.ProbeEgress,
			Target:       measured.Target,
			Verdict:      probe.Indeterminate,
			Reason:       probe.ReasonInternalError,
			Detail:       measured.Detail,
			Observations: []probe.Observation{measured},
		}

		got := diagnosis.Diagnose([]probe.Result{contradicting})
		if len(got.Findings) != 1 {
			t.Fatalf("the run reported one probe and the diagnosis carries %d findings, want one", len(got.Findings))
		}
		if want := diagnosis.RuleID("egress.ssh.known", diagnosis.StatePass); got.Findings[0].Rule != want {
			t.Errorf("the run fired %q, want %q: the aggregate verdict was read where the observations say otherwise", got.Findings[0].Rule, want)
		}
	})

	t.Run("an aggregate that claims a pass cannot hide a measured failure", func(t *testing.T) {
		measured := observation("tcp 22", "ssh.github.com:22", probe.Measured, probe.Fail, probe.ReasonBannerNotSSH,
			"tcp ssh.github.com:22: connected, banner is not SSH")
		contradicting := probe.Result{
			Probe:        "egress.ssh.known",
			Kind:         probe.ProbeEgress,
			Target:       measured.Target,
			Verdict:      probe.Pass,
			Reason:       probe.ReasonOK,
			Detail:       measured.Detail,
			Observations: []probe.Observation{measured},
		}

		got := diagnosis.Diagnose([]probe.Result{contradicting})
		if len(got.Findings) != 1 {
			t.Fatalf("the run reported one probe and the diagnosis carries %d findings, want one", len(got.Findings))
		}
		if want := diagnosis.RuleID("egress.ssh.known", diagnosis.StateFail); got.Findings[0].Rule != want {
			t.Errorf("the run fired %q, want %q: a fabricated aggregate pass must not survive the observations", got.Findings[0].Rule, want)
		}
	})
}

// sshDestinationFixture names one scripted observation the `ssh.destination` cases reason over: the
// probe that reported it and the state it reported.
type sshDestinationFixture struct {
	probe string
	state diagnosis.State
}

// sshDestinationObservation builds the observation one of the group's three probes reports in one
// state, with the endpoint that probe declares: the two public probes measure their declared public
// host, and the hub measures the address the run was given — the documented placeholder of PRD §1.1,
// never a real address.
func sshDestinationObservation(t *testing.T, probeName string, state diagnosis.State) probe.Observation {
	t.Helper()
	label, target := "tcp 22 ssh banner", "github.com:22"
	switch probeName {
	case "egress.ssh.443":
		label, target = "tcp 443 ssh banner", "ssh.github.com:443"
	case "egress.hub.direct":
		label, target = "tcp 22", "203.0.113.10:22"
	}
	switch state {
	case diagnosis.StatePass:
		return observation(label, target, probe.Measured, probe.Pass, probe.ReasonOK,
			"the connection was established inside the probe budget")
	case diagnosis.StateFail:
		return observation(label, target, probe.Measured, probe.Fail, probe.ReasonBudgetExpired,
			"i/o timeout after the probe's own dial budget, which for the question \"is this port reachable?\" is the measurement itself")
	case diagnosis.StateUnresolved:
		return observation(label, target, probe.Unresolved, probe.Indeterminate, probe.ReasonDNSUnresolved,
			"the resolver produced no answer inside the probe budget")
	case diagnosis.StateNotMeasured:
		return observation(label, "", probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded,
			"no dialer is injected for this run, so this measurement could not be attempted")
	}
	t.Fatalf("the cases have no observation for the state %q", state)
	return probe.Observation{}
}

// sshDestinationRun builds the run the fixtures describe: one result per probe, in the group's probe
// order, each carrying its own observations. A probe no fixture names contributes nothing, which is
// how a case expresses a measurement the run did not make at all — distinct from a not-measured
// observation, which is a probe that ran and reported it could not attempt something.
func sshDestinationRun(t *testing.T, fixtures ...sshDestinationFixture) []probe.Result {
	t.Helper()
	var results []probe.Result
	for _, probeName := range []string{"egress.ssh.known", "egress.ssh.443", "egress.hub.direct"} {
		var observations []probe.Observation
		for _, fixture := range fixtures {
			if fixture.probe != probeName {
				continue
			}
			observations = append(observations, sshDestinationObservation(t, probeName, fixture.state))
		}
		if len(observations) == 0 {
			continue
		}
		results = append(results, result(probeName, probe.ProbeEgress, observations...))
	}
	return results
}

// TestSSHDestinationOneCasePerID is design §5.2's per-rule case for the `ssh.destination` group: one
// case per id, each asserting that the declared row fired, that its conclusion carries the
// measurement the row rested on, and that DependsOn names exactly those observables and no others.
//
// A case states the run as observations of the group's three probes and nothing else, so what fires
// is decided by the group's own rows rather than by an unrelated measurement. The cases that reach
// the weaker ids also assert that the conclusion does not carry the confident half's wording:
// "outbound SSH is allowed" and "outbound SSH works" are claims a run may only make from a measured
// public pass (R-HR-NF-03).
func TestSSHDestinationOneCasePerID(t *testing.T) {
	cases := []struct {
		name string
		run  []sshDestinationFixture
		// want is the rule id the run must fire.
		want string
		// wantContains lists phrases the conclusion must carry: the row's own evidence.
		wantContains []string
		// wantAbsent lists phrases the conclusion must not carry.
		wantAbsent []string
		// wantDepends is the observable set the fired clause rests on, in Match order.
		wantDepends []string
	}{
		{
			name: "a passing public measurement beside a failing hub concludes the destination is blocked",
			run: []sshDestinationFixture{
				{"egress.ssh.known", diagnosis.StatePass},
				{"egress.hub.direct", diagnosis.StateFail},
			},
			want:         "SSH_DEST_BLOCKED_BY_PUBLIC_SSH",
			wantContains: []string{"outbound SSH is allowed; the hub address", "203.0.113.10:22 is blocked", "github.com:22"},
			wantDepends:  []string{"egress.ssh.known", "egress.hub.direct"},
		},
		{
			name: "port 443 carries the disambiguation when port 22 did not pass",
			run: []sshDestinationFixture{
				{"egress.ssh.443", diagnosis.StatePass},
				{"egress.hub.direct", diagnosis.StateFail},
			},
			want:         "SSH_DEST_BLOCKED_BY_PUBLIC_SSH_443",
			wantContains: []string{"outbound SSH is allowed; the hub address", "203.0.113.10:22 is blocked", "ssh.github.com:443"},
			wantAbsent:   []string{"github.com:22"},
			wantDepends:  []string{"egress.ssh.443", "egress.hub.direct"},
		},
		{
			name: "an unresolved public measurement beside a failing hub leaves the block unestablished",
			run: []sshDestinationFixture{
				{"egress.ssh.known", diagnosis.StateUnresolved},
				{"egress.hub.direct", diagnosis.StateFail},
			},
			want:         "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED",
			wantContains: []string{"not established", "egress.ssh.known", "dns_unresolved", "was attempted and produced no answer"},
			wantAbsent:   []string{"outbound SSH is allowed", "outbound SSH works"},
			wantDepends:  []string{"egress.ssh.known", "egress.hub.direct"},
		},
		{
			name: "a public measurement that was never made leaves the block unestablished",
			run: []sshDestinationFixture{
				{"egress.ssh.known", diagnosis.StateNotMeasured},
				{"egress.hub.direct", diagnosis.StateFail},
			},
			want:         "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED",
			wantContains: []string{"not established", "egress.ssh.known", "capability_excluded", "was not made"},
			wantAbsent:   []string{"outbound SSH is allowed", "outbound SSH works", "was attempted"},
			wantDepends:  []string{"egress.ssh.known", "egress.hub.direct"},
		},
		{
			name: "both public measurements and the hub failing leaves the block unestablished",
			run: []sshDestinationFixture{
				{"egress.ssh.known", diagnosis.StateFail},
				{"egress.ssh.443", diagnosis.StateFail},
				{"egress.hub.direct", diagnosis.StateFail},
			},
			want:         "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_FAILED",
			wantContains: []string{"not established", "github.com:22", "ssh.github.com:443", "203.0.113.10:22", "cannot be separated"},
			wantAbsent:   []string{"outbound SSH is allowed", "outbound SSH works"},
			wantDepends:  []string{"egress.ssh.known", "egress.ssh.443", "egress.hub.direct"},
		},
		{
			name: "a passing public measurement beside a reachable hub observes no block",
			run: []sshDestinationFixture{
				{"egress.ssh.known", diagnosis.StatePass},
				{"egress.hub.direct", diagnosis.StatePass},
			},
			want:         "SSH_NO_DEST_BLOCK_OBSERVED",
			wantContains: []string{"no destination block observed", "github.com:22", "203.0.113.10:22"},
			wantAbsent:   []string{"is blocked"},
			wantDepends:  []string{"egress.hub.direct", "egress.ssh.known"},
		},
		{
			name: "an unresolved hub is not assessed",
			run: []sshDestinationFixture{
				{"egress.ssh.known", diagnosis.StatePass},
				{"egress.hub.direct", diagnosis.StateUnresolved},
			},
			want:         "SSH_DEST_BLOCK_NOT_ASSESSED",
			wantContains: []string{"nothing is claimed about the hub", "egress.hub.direct", "dns_unresolved", "was attempted and produced no answer"},
			wantAbsent:   []string{"outbound SSH is allowed", "outbound SSH works", "reachable"},
			wantDepends:  []string{"egress.hub.direct"},
		},
		{
			name: "a hub that was never measured is not assessed",
			run: []sshDestinationFixture{
				{"egress.hub.direct", diagnosis.StateNotMeasured},
			},
			want:         "SSH_DEST_BLOCK_NOT_ASSESSED",
			wantContains: []string{"nothing is claimed about the hub", "egress.hub.direct", "capability_excluded", "was not made"},
			wantAbsent:   []string{"outbound SSH is allowed", "outbound SSH works", "reachable", "was attempted"},
			wantDepends:  []string{"egress.hub.direct"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := diagnosis.Diagnose(sshDestinationRun(t, tc.run...))
			finding := findingFor(t, got, "ssh.destination")
			if finding.Rule != tc.want {
				t.Errorf("the run fired %q, want %q", finding.Rule, tc.want)
			}
			if finding.Conclusion == "" {
				t.Fatalf("the rule %q concluded nothing", tc.want)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(finding.Conclusion, want) {
					t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(finding.Conclusion, absent) {
					t.Errorf("the conclusion %q carries %q, which this row must not claim", finding.Conclusion, absent)
				}
			}
			if !reflect.DeepEqual(finding.DependsOn, tc.wantDepends) {
				t.Errorf("the conclusion depends on %v, want %v: the fired clause's own observables", finding.DependsOn, tc.wantDepends)
			}
		})
	}
}

// TestSSHDestinationCounterfactuals is the group's triangulation: the cases the weaker rows exist
// for, and the controls that show the group does not conclude from an absence.
//
// Each case is a run that is one measurement away from the acceptance case, and each asserts the
// outcome that distance must produce. The controls matter as much as the conclusions: a group that
// reported a block whenever the hub did not answer would pass the acceptance case and be wrong on
// every network where the public measurement is blocked too.
func TestSSHDestinationCounterfactuals(t *testing.T) {
	t.Run("no public pass, no destination-block conclusion", func(t *testing.T) {
		// Both public measurements failed (measured, not absences) and the hub failed: the weaker row
		// fires, because a failed public measurement establishes no more than an absent one does about
		// what the hub failure means.
		run := sshDestinationRun(t,
			sshDestinationFixture{"egress.ssh.known", diagnosis.StateFail},
			sshDestinationFixture{"egress.ssh.443", diagnosis.StateFail},
			sshDestinationFixture{"egress.hub.direct", diagnosis.StateFail},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "ssh.destination")
		if finding.Rule != "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_FAILED" {
			t.Errorf("the run fired %q, want the weaker %q", finding.Rule, "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_FAILED")
		}
		for _, absent := range []string{"outbound SSH is allowed", "outbound SSH works"} {
			if strings.Contains(finding.Conclusion, absent) {
				t.Errorf("a failed public measurement produced the claim %q: %s", absent, finding.Conclusion)
			}
		}
	})

	t.Run("a public probe the run never measured leaves the comparison open", func(t *testing.T) {
		// The hub failed and port 22 failed, and this run did not include port 443 at all. A probe the
		// run did not report contributes no fact (facts.go), so no row matches: the design's
		// `ssh.public_443 ≠ pass` is not read as an absence the run never produced, and the honest
		// output is the open question that names the states the table needed. A complete run always
		// reports both public probes, so this is the partial-run boundary rather than a live path.
		run := sshDestinationRun(t,
			sshDestinationFixture{"egress.ssh.known", diagnosis.StateFail},
			sshDestinationFixture{"egress.hub.direct", diagnosis.StateFail},
		)
		got := diagnosis.Diagnose(run)
		if findings := findingsFor(got, "ssh.destination"); len(findings) != 0 {
			t.Errorf("a run that measured one public probe produced %d findings for the question: %+v", len(findings), findings)
		}
		open := openQuestionFor(t, got, "ssh.destination")
		if !slices.Contains(open.NeededStates, "egress.ssh.443 PASS") {
			t.Errorf("the open question needs %v, want the unreported probe's states among them", open.NeededStates)
		}
	})

	t.Run("a public measurement that was never made fires the weaker row, not the confident one", func(t *testing.T) {
		// The public probe ran and reported that it could not attempt anything (a capability this
		// run was not given), while the hub failed. Nothing here establishes what the hub failure
		// means, and the conclusion names the probe that did not answer.
		run := sshDestinationRun(t,
			sshDestinationFixture{"egress.ssh.known", diagnosis.StateNotMeasured},
			sshDestinationFixture{"egress.ssh.443", diagnosis.StateNotMeasured},
			sshDestinationFixture{"egress.hub.direct", diagnosis.StateFail},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "ssh.destination")
		if finding.Rule != "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED" {
			t.Errorf("the run fired %q, want the weaker %q", finding.Rule, "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED")
		}
		if !strings.Contains(finding.Conclusion, "egress.ssh.known") {
			t.Errorf("the conclusion %q does not name the public probe that did not answer", finding.Conclusion)
		}
		for _, absent := range []string{"outbound SSH is allowed", "outbound SSH works"} {
			if strings.Contains(finding.Conclusion, absent) {
				t.Errorf("an absence produced the claim %q: %s", absent, finding.Conclusion)
			}
		}
	})

	t.Run("an absence outranks a failure, because a failure is a definite answer", func(t *testing.T) {
		// Port 22 failed and port 443 produced no answer. Both rows could be read as applying; the
		// weaker reading is the one declared first, so the conclusion names the unanswered
		// measurement instead of concluding from two failed ones.
		run := sshDestinationRun(t,
			sshDestinationFixture{"egress.ssh.known", diagnosis.StateFail},
			sshDestinationFixture{"egress.ssh.443", diagnosis.StateUnresolved},
			sshDestinationFixture{"egress.hub.direct", diagnosis.StateFail},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "ssh.destination")
		if finding.Rule != "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED" {
			t.Errorf("the run fired %q, want the weaker %q", finding.Rule, "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED")
		}
		if !strings.Contains(finding.Conclusion, "ssh.github.com:443") {
			t.Errorf("the conclusion %q does not name the measurement that produced no answer", finding.Conclusion)
		}
	})

	t.Run("both signals passing yields no block conclusion", func(t *testing.T) {
		// PRD §1.1's control: SSH works and the hub is reachable, so the run must not report a
		// block. The row that fires says so, and neither destination-block id appears.
		run := sshDestinationRun(t,
			sshDestinationFixture{"egress.ssh.known", diagnosis.StatePass},
			sshDestinationFixture{"egress.hub.direct", diagnosis.StatePass},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "ssh.destination")
		if finding.Rule != "SSH_NO_DEST_BLOCK_OBSERVED" {
			t.Errorf("the run fired %q, want %q", finding.Rule, "SSH_NO_DEST_BLOCK_OBSERVED")
		}
		for _, blocked := range []string{"SSH_DEST_BLOCKED_BY_PUBLIC_SSH", "SSH_DEST_BLOCKED_BY_PUBLIC_SSH_443"} {
			if finding.Rule == blocked {
				t.Errorf("both signals passing produced the block conclusion %q", blocked)
			}
		}
	})

	t.Run("nothing is claimed about a hub that did not answer", func(t *testing.T) {
		// The hub was attempted and produced no answer. The conclusion says exactly that and claims
		// neither reachability nor a block, whichever way the public measurements came out.
		run := sshDestinationRun(t,
			sshDestinationFixture{"egress.ssh.known", diagnosis.StateFail},
			sshDestinationFixture{"egress.hub.direct", diagnosis.StateUnresolved},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "ssh.destination")
		if finding.Rule != "SSH_DEST_BLOCK_NOT_ASSESSED" {
			t.Errorf("the run fired %q, want %q", finding.Rule, "SSH_DEST_BLOCK_NOT_ASSESSED")
		}
		for _, absent := range []string{"outbound SSH is allowed", "outbound SSH works", "reachable"} {
			if strings.Contains(finding.Conclusion, absent) {
				t.Errorf("an unanswered hub produced the claim %q: %s", absent, finding.Conclusion)
			}
		}
	})

	t.Run("the source run makes no hub result at all and the question stays open", func(t *testing.T) {
		// A hub the run never measured contributes no fact, so no row of the group can match and the
		// mechanism reports the question as open rather than inventing a not-measured hub. This is
		// what separates a probe that ran and reported an absence from a probe the run never ran.
		run := sshDestinationRun(t,
			sshDestinationFixture{"egress.ssh.known", diagnosis.StatePass},
		)
		got := diagnosis.Diagnose(run)
		if findings := findingsFor(got, "ssh.destination"); len(findings) != 0 {
			t.Errorf("a run that measured no hub produced %d findings for the question: %+v", len(findings), findings)
		}
		open := openQuestionFor(t, got, "ssh.destination")
		if len(open.NeededStates) == 0 {
			t.Errorf("the open question %q names no needed state", open.Question)
		}
	})

	t.Run("the conclusion names the port the run declared", func(t *testing.T) {
		// PRD §1.1's hub row lists three ports measured by hand; a run declares one address (design
		// D3), and the conclusion must name that address with its port rather than assuming 22.
		run := []probe.Result{
			result("egress.ssh.known", probe.ProbeEgress, sshDestinationObservation(t, "egress.ssh.known", diagnosis.StatePass)),
			result("egress.hub.direct", probe.ProbeEgress, observation("tcp 2222", "203.0.113.10:2222", probe.Measured, probe.Fail,
				probe.ReasonBudgetExpired, "i/o timeout after the probe's own dial budget")),
		}
		finding := findingFor(t, diagnosis.Diagnose(run), "ssh.destination")
		if finding.Rule != "SSH_DEST_BLOCKED_BY_PUBLIC_SSH" {
			t.Errorf("the run fired %q, want %q", finding.Rule, "SSH_DEST_BLOCKED_BY_PUBLIC_SSH")
		}
		if !strings.Contains(finding.Conclusion, "203.0.113.10:2222 is blocked") {
			t.Errorf("the conclusion %q does not name the declared port", finding.Conclusion)
		}
	})
}

// cloudflareFixture names one scripted observation of one of the probes the Cloudflare groups read:
// the probe that reported it, the region it measured, and the state it reported.
type cloudflareFixture struct {
	probe  string
	region string
	state  diagnosis.State
}

// cloudflareObservation builds one per-region observation of one of the two edge probes or of the
// datagram probe, with the label, target, reason code and detail a real measurement of that probe in
// that region would carry. The two regions are measured separately on purpose: the edge groups reason
// over the set of per-region observations, so a case that could not express a split could not express
// what they are for (design D10).
func cloudflareObservation(t *testing.T, probeName, region string, state diagnosis.State) probe.Observation {
	t.Helper()
	label, target := "", ""
	switch probeName {
	case "egress.cf.443":
		label = fmt.Sprintf("tcp 443 %s", region)
		target = fmt.Sprintf("%s.v2.argotunnel.com:443", region)
	case "egress.cf.7844":
		label = fmt.Sprintf("tcp 7844 %s", region)
		target = fmt.Sprintf("%s.v2.argotunnel.com:7844", region)
	case "egress.quic":
		label = fmt.Sprintf("udp 7844 %s", region)
		target = fmt.Sprintf("%s.v2.argotunnel.com:7844", region)
	default:
		t.Fatalf("the Cloudflare cases have no fixture for the probe %q", probeName)
	}
	switch state {
	case diagnosis.StatePass:
		if probeName == "egress.quic" {
			return observation(label, target, probe.Measured, probe.Pass, probe.ReasonUDPResponseReceived,
				"a datagram arrived from the edge, so this datagram to this edge and port was not silently dropped")
		}
		return observation(label, target, probe.Measured, probe.Pass, probe.ReasonOK,
			"the connection was established and closed without reading or writing, which measures TCP reachability of the declared address and nothing else")
	case diagnosis.StateFail:
		if probeName == "egress.quic" {
			return observation(label, target, probe.Measured, probe.Fail, probe.ReasonUDPUnreachable,
				"the socket surfaced an ICMP port-unreachable, so the far end answered that nothing is listening on this UDP port")
		}
		return observation(label, target, probe.Measured, probe.Fail, probe.ReasonBudgetExpired,
			"i/o timeout after the probe's own dial budget, which for the question \"is this port reachable?\" is the measurement itself")
	case diagnosis.StateUnresolved:
		if probeName == "egress.quic" {
			return observation(label, target, probe.Unresolved, probe.Indeterminate, probe.ReasonUDPSilence,
				"no answer on UDP arrived inside the probe's own budget, and the socket reported no error at all")
		}
		return observation(label, target, probe.Unresolved, probe.Indeterminate, probe.ReasonDNSUnresolved,
			"the resolver produced no answer inside the probe budget")
	case diagnosis.StateNotMeasured:
		if probeName == "egress.quic" {
			return observation(label, "", probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded,
				"no packet dialer is injected for this run, so this measurement could not be attempted")
		}
		return observation(label, "", probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded,
			"no resolver or dialer is injected for this run, so this measurement could not be attempted")
	}
	t.Fatalf("the Cloudflare cases have no observation for the state %q", state)
	return probe.Observation{}
}

// cloudflareRun builds the run the fixtures describe: one result per probe, in registry order, each
// carrying its own per-region observations. A probe no fixture names contributes no fact, which is
// how a case expresses a measurement the run never made.
func cloudflareRun(t *testing.T, fixtures ...cloudflareFixture) []probe.Result {
	t.Helper()
	var results []probe.Result
	for _, probeName := range []string{"egress.cf.7844", "egress.cf.443", "egress.quic"} {
		var observations []probe.Observation
		for _, fixture := range fixtures {
			if fixture.probe != probeName {
				continue
			}
			observations = append(observations, cloudflareObservation(t, probeName, fixture.region, fixture.state))
		}
		if len(observations) == 0 {
			continue
		}
		kind := probe.ProbeEgress
		if probeName == "egress.quic" {
			kind = probe.ProbeProto
		}
		results = append(results, result(probeName, kind, observations...))
	}
	return results
}

// TestCloudflareEdgeOneCasePerID is design §5.2's per-rule case for the `cloudflare.edge` group: one
// case per id, each asserting what the set of per-region observations establishes and nothing more.
//
// Every case is expressed as per-region observations, so the states the group distinguishes are the
// states a reader of the run sees: every measured region reachable, a split between regions, every
// measured region failed, and a region that produced no answer beside no measured success. The last
// case is the one that keeps the group honest — an absence weakens the conclusion instead of being
// read as a failure or a pass (R-HR-NF-03).
func TestCloudflareEdgeOneCasePerID(t *testing.T) {
	cases := []struct {
		name string
		run  []cloudflareFixture
		// want is the rule id the run must fire.
		want string
		// wantContains lists phrases the conclusion must carry: the observations the row rested on.
		wantContains []string
		// wantDepends is the observable set the fired clause rests on, in Match order.
		wantDepends []string
	}{
		{
			name: "every measured edge region is reachable",
			run: []cloudflareFixture{
				{"egress.cf.443", "region1", diagnosis.StatePass},
				{"egress.cf.443", "region2", diagnosis.StatePass},
			},
			want:         "CF_EDGE_REACHABLE",
			wantContains: []string{"the Cloudflare edge is reachable", "region1.v2.argotunnel.com:443", "region2.v2.argotunnel.com:443", "not a statement about a tunnel"},
			wantDepends:  []string{"egress.cf.443"},
		},
		{
			name: "a split between the measured edge regions is partial reachability",
			run: []cloudflareFixture{
				{"egress.cf.443", "region1", diagnosis.StatePass},
				{"egress.cf.443", "region2", diagnosis.StateFail},
			},
			want:         "CF_EDGE_PARTIAL",
			wantContains: []string{"only partly reachable", "region1.v2.argotunnel.com:443", "region2.v2.argotunnel.com:443", "do not agree"},
			wantDepends:  []string{"egress.cf.443"},
		},
		{
			name: "every measured edge region failed",
			run: []cloudflareFixture{
				{"egress.cf.443", "region1", diagnosis.StateFail},
				{"egress.cf.443", "region2", diagnosis.StateFail},
				// A failure of the other edge probe as well. The clause that fires is the one for the probe
				// whose observations it quotes, and the universal in the wording ("every measured edge
				// endpoint failed") is guaranteed by the clauses declared above it: a pass anywhere would have
				// fired REACHABLE and an unresolved attempt anywhere would have fired UNRESOLVED, so this row
				// is reached only when the remaining observations are failures or unmeasured endpoints.
				{"egress.cf.7844", "region1", diagnosis.StateFail},
			},
			want:         "CF_EDGE_UNREACHABLE",
			wantContains: []string{"the Cloudflare edge is unreachable", "every measured edge endpoint failed", "region1.v2.argotunnel.com:443", "region2.v2.argotunnel.com:443"},
			wantDepends:  []string{"egress.cf.443"},
		},
		{
			name: "a region that produced no answer beside a failed one leaves reachability unestablished",
			run: []cloudflareFixture{
				{"egress.cf.443", "region1", diagnosis.StateUnresolved},
				{"egress.cf.443", "region2", diagnosis.StateFail},
			},
			want:         "CF_EDGE_UNRESOLVED",
			wantContains: []string{"is not established", "region1.v2.argotunnel.com:443", "dns_unresolved", "no edge endpoint was measured as reachable"},
			wantDepends:  []string{"egress.cf.443"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := diagnosis.Diagnose(cloudflareRun(t, tc.run...))
			finding := findingFor(t, got, "cloudflare.edge")
			if finding.Rule != tc.want {
				t.Errorf("the run fired %q, want %q", finding.Rule, tc.want)
			}
			if finding.Conclusion == "" {
				t.Fatalf("the rule %q concluded nothing", tc.want)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(finding.Conclusion, want) {
					t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
				}
			}
			if !reflect.DeepEqual(finding.DependsOn, tc.wantDepends) {
				t.Errorf("the conclusion depends on %v, want %v", finding.DependsOn, tc.wantDepends)
			}
		})
	}
}

// TestCloudflareHTTP2OneCasePerID is design §5.2's per-rule case for the `cloudflare.http2` group:
// one case per id, each asserting the advice and the measurements it rests on.
//
// The group is the one place in this slice that recommends a change, so every case checks the
// boundary R-HR-05 draws as well: the conclusion recommends and says so; it never claims that a
// fallback was applied, that a configuration was written, or that the HTTP/2 path is equivalent to
// QUIC. A passing datagram measurement produces no downgrade advice at all, which is the case that
// stops the group from recommending against a measurement that did not fail.
func TestCloudflareHTTP2OneCasePerID(t *testing.T) {
	// claimsEnforcement lists the wordings that would overstate what this slice does. None of them
	// may appear in any conclusion of the group: the recommendation is a recommendation.
	claimsEnforcement := []string{
		"has been applied",
		"was applied",
		"is applied",
		"enforced",
		"configuration was written",
		"equivalent to QUIC",
	}

	cases := []struct {
		name string
		run  []cloudflareFixture
		// want is the rule id the run must fire.
		want string
		// wantContains lists phrases the conclusion must carry: the advice and its evidence.
		wantContains []string
		// wantAbsent lists phrases the conclusion must not carry.
		wantAbsent []string
		// wantDepends is the observable set the fired clause rests on, in Match order.
		wantDepends []string
	}{
		{
			name: "a failed datagram measurement beside a reachable edge advises HTTP/2",
			run: []cloudflareFixture{
				{"egress.cf.443", "region1", diagnosis.StatePass},
				{"egress.quic", "region1", diagnosis.StateFail},
			},
			want:         "CF_HTTP2_ADVISED_QUIC_FAILED",
			wantContains: []string{"recommend forcing the Cloudflare tunnel transport to HTTP/2", "egress.quic", "port-unreachable", "region1.v2.argotunnel.com:7844", "recommendation only"},
			wantAbsent:   claimsEnforcement,
			wantDepends:  []string{"egress.quic", "egress.cf.443"},
		},
		{
			name: "an unconfirmed datagram measurement beside a reachable edge advises HTTP/2 conservatively",
			run: []cloudflareFixture{
				{"egress.cf.443", "region1", diagnosis.StatePass},
				{"egress.quic", "region1", diagnosis.StateUnresolved},
			},
			want:         "CF_HTTP2_ADVISED_QUIC_UNCONFIRMED",
			wantContains: []string{"recommend forcing the Cloudflare tunnel transport to HTTP/2", "egress.quic", "udp_silence", "region1.v2.argotunnel.com:7844", "was attempted and produced no answer", "never confirmed", "recommendation only"},
			wantAbsent:   append([]string{"downgrade"}, claimsEnforcement...),
			wantDepends:  []string{"egress.quic", "egress.cf.443"},
		},
		{
			name: "a confirmed datagram measurement produces no downgrade advice",
			run: []cloudflareFixture{
				{"egress.cf.443", "region1", diagnosis.StatePass},
				{"egress.quic", "region1", diagnosis.StatePass},
			},
			want:         "CF_NO_HTTP2_ADVICE_QUIC_USABLE",
			wantContains: []string{"no HTTP/2 downgrade is recommended", "egress.quic", "region1.v2.argotunnel.com:7844", "not silently dropped"},
			wantAbsent:   []string{"recommend forcing"},
			wantDepends:  []string{"egress.quic"},
		},
		{
			name: "an unreachable edge leaves the advice unassessed",
			run: []cloudflareFixture{
				{"egress.cf.443", "region1", diagnosis.StateFail},
				{"egress.quic", "region1", diagnosis.StateUnresolved},
			},
			want:         "CF_HTTP2_ADVISORY_NOT_ASSESSED",
			wantContains: []string{"no HTTP/2 advice is given", "region1.v2.argotunnel.com:443"},
			wantAbsent:   []string{"recommend forcing", "no HTTP/2 downgrade is recommended"},
			wantDepends:  []string{"egress.cf.443"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := diagnosis.Diagnose(cloudflareRun(t, tc.run...))
			finding := findingFor(t, got, "cloudflare.http2")
			if finding.Rule != tc.want {
				t.Errorf("the run fired %q, want %q", finding.Rule, tc.want)
			}
			if finding.Conclusion == "" {
				t.Fatalf("the rule %q concluded nothing", tc.want)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(finding.Conclusion, want) {
					t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(finding.Conclusion, absent) {
					t.Errorf("the conclusion %q carries %q, which this rule must not claim", finding.Conclusion, absent)
				}
			}
			if !reflect.DeepEqual(finding.DependsOn, tc.wantDepends) {
				t.Errorf("the conclusion depends on %v, want %v", finding.DependsOn, tc.wantDepends)
			}
		})
	}
}

// TestCloudflareCounterfactuals is the Cloudflare groups' triangulation: the runs that must not
// produce a recommendation, and the ones that must produce the weaker conclusion.
//
// The two properties the cases exist for are the honesty rules of the slice. First, the conservative
// advice must name the measurement it rests on: a run whose datagram measurement produced no answer
// learns which probe and which endpoint did not answer, and the conclusion says it rests on a
// missing confirmation rather than on a measured block (design D8, RG-5). Second, a datagram
// measurement that drew a reply must produce no downgrade advice at all, whatever the rest of the run
// looks like: a probe that cannot fail is not a probe, and a probe that cannot succeed without
// causing advice is not a measurement either.
func TestCloudflareCounterfactuals(t *testing.T) {
	t.Run("the unconfirmed advice names every unresolved datagram measurement it rests on", func(t *testing.T) {
		// Both regions were attempted and neither produced an answer, while the edge answered on TCP.
		// The conclusion must name the probe, both endpoints and the reason codes, and it must not
		// report a failure or a block: nothing was measured as failing.
		run := cloudflareRun(t,
			cloudflareFixture{"egress.cf.443", "region1", diagnosis.StatePass},
			cloudflareFixture{"egress.quic", "region1", diagnosis.StateUnresolved},
			cloudflareFixture{"egress.quic", "region2", diagnosis.StateUnresolved},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "cloudflare.http2")
		if finding.Rule != "CF_HTTP2_ADVISED_QUIC_UNCONFIRMED" {
			t.Fatalf("the run fired %q, want %q", finding.Rule, "CF_HTTP2_ADVISED_QUIC_UNCONFIRMED")
		}
		for _, want := range []string{
			"egress.quic",
			"region1.v2.argotunnel.com:7844",
			"region2.v2.argotunnel.com:7844",
			"udp_silence",
			"was attempted and produced no answer",
		} {
			if !strings.Contains(finding.Conclusion, want) {
				t.Errorf("the conclusion %q does not name %q, which is what it rests on", finding.Conclusion, want)
			}
		}
		for _, absent := range []string{"failed", "blocked"} {
			if strings.Contains(finding.Conclusion, absent) {
				t.Errorf("the conclusion %q claims %q from an absence", finding.Conclusion, absent)
			}
		}
		if !reflect.DeepEqual(finding.DependsOn, []string{"egress.quic", "egress.cf.443"}) {
			t.Errorf("the conclusion depends on %v, want the datagram probe and the edge it rests on", finding.DependsOn)
		}
	})

	t.Run("a working datagram measurement produces no downgrade advice", func(t *testing.T) {
		// Both regions drew a reply. No advice is given in either direction, and no id that advises
		// appears: the recommendation must not be produced by a measurement that did not fail.
		run := cloudflareRun(t,
			cloudflareFixture{"egress.cf.443", "region1", diagnosis.StatePass},
			cloudflareFixture{"egress.quic", "region1", diagnosis.StatePass},
			cloudflareFixture{"egress.quic", "region2", diagnosis.StatePass},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "cloudflare.http2")
		if finding.Rule != "CF_NO_HTTP2_ADVICE_QUIC_USABLE" {
			t.Fatalf("the run fired %q, want %q", finding.Rule, "CF_NO_HTTP2_ADVICE_QUIC_USABLE")
		}
		if !strings.Contains(finding.Conclusion, "no HTTP/2 downgrade is recommended") {
			t.Errorf("the conclusion %q does not state that no downgrade is recommended", finding.Conclusion)
		}
		if strings.Contains(finding.Conclusion, "recommend forcing") {
			t.Errorf("a passing datagram measurement produced the downgrade advice: %s", finding.Conclusion)
		}
	})

	t.Run("a split edge still carries the conservative advice", func(t *testing.T) {
		// Region1 on 443 answered and region1 on 7844 did not, while the datagram path was never
		// confirmed. The edge conclusion reports the split and the advice still fires: it rests on the
		// measured TCP path that did answer, which is the condition §5.2 states as `edge ∈ {reachable,
		// partial}`.
		run := cloudflareRun(t,
			cloudflareFixture{"egress.cf.443", "region1", diagnosis.StatePass},
			cloudflareFixture{"egress.cf.7844", "region1", diagnosis.StateFail},
			cloudflareFixture{"egress.quic", "region1", diagnosis.StateUnresolved},
		)
		got := diagnosis.Diagnose(run)
		if edge := findingFor(t, got, "cloudflare.edge"); edge.Rule != "CF_EDGE_PARTIAL" {
			t.Errorf("the edge question fired %q, want %q", edge.Rule, "CF_EDGE_PARTIAL")
		}
		if advice := findingFor(t, got, "cloudflare.http2"); advice.Rule != "CF_HTTP2_ADVISED_QUIC_UNCONFIRMED" {
			t.Errorf("the advice question fired %q, want %q", advice.Rule, "CF_HTTP2_ADVISED_QUIC_UNCONFIRMED")
		}
	})

	t.Run("a failed datagram measurement beside an unreachable edge gives no advice", func(t *testing.T) {
		// The datagram measurement failed, and the edge did not answer on TCP either. The advice
		// question reports that no advice is given rather than recommending a fallback over a path
		// that reaches nothing, and it is not left open: the absence of an advice is itself the
		// conclusion here.
		run := cloudflareRun(t,
			cloudflareFixture{"egress.cf.443", "region1", diagnosis.StateFail},
			cloudflareFixture{"egress.quic", "region1", diagnosis.StateFail},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "cloudflare.http2")
		if finding.Rule != "CF_HTTP2_ADVISORY_NOT_ASSESSED" {
			t.Fatalf("the run fired %q, want %q", finding.Rule, "CF_HTTP2_ADVISORY_NOT_ASSESSED")
		}
		if strings.Contains(finding.Conclusion, "recommend forcing") {
			t.Errorf("an unreachable edge produced the advice: %s", finding.Conclusion)
		}
	})

	t.Run("an unmeasured edge leaves the advice unassessed rather than open", func(t *testing.T) {
		// The edge probes reported that they could not attempt anything, so no edge measurement exists
		// and the advice question says so. The conclusion names the absence and the capability that
		// was missing, and it recommends nothing.
		run := cloudflareRun(t,
			cloudflareFixture{"egress.cf.443", "region1", diagnosis.StateNotMeasured},
			cloudflareFixture{"egress.quic", "region1", diagnosis.StateUnresolved},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "cloudflare.http2")
		if finding.Rule != "CF_HTTP2_ADVISORY_NOT_ASSESSED" {
			t.Fatalf("the run fired %q, want %q", finding.Rule, "CF_HTTP2_ADVISORY_NOT_ASSESSED")
		}
		if !strings.Contains(finding.Conclusion, "capability_excluded") {
			t.Errorf("the conclusion %q does not name why the edge was not measured", finding.Conclusion)
		}
	})
}

// sshdFixture names one scripted observation of the `local.sshd` probe: the observation's own label
// — one of the three the probe declares in local.go — and the state it reported.
type sshdFixture struct {
	label string
	state diagnosis.State
}

// The three observation labels the `local.sshd` probe declares, pinned literally because the
// probe's constants are unexported and the group's rows match on these exact strings: a rename has
// to be an explicit act on both sides.
const (
	testSSHDLabelBinary  = "binary present"
	testSSHDLabelService = "service state"
	testSSHDLabelConfig  = "effective config"
)

// sshdObservation builds the observation one `local.sshd` label reports in one state, with the
// target, reason code and verbatim detail the probe's own wording carries: the documented binary
// path for the binary observation, the unit list for the service observation, and the written
// configuration path for the effective-config observation. The two configurations of the diverging
// and agreeing cases are stated in the detail because the group conclusion quotes it.
func sshdObservation(t *testing.T, label string, state diagnosis.State) probe.Observation {
	t.Helper()
	switch label {
	case testSSHDLabelBinary:
		switch state {
		case diagnosis.StatePass:
			return observation(label, "/usr/sbin/sshd", probe.Measured, probe.Pass, probe.ReasonOK,
				"the sshd binary is present at /usr/sbin/sshd")
		case diagnosis.StateFail:
			return observation(label, "/usr/sbin/sshd", probe.Measured, probe.Fail, probe.ReasonSSHDAbsent,
				"the sshd binary is not present at /usr/sbin/sshd: no sshd is installed here, and installing it is not part of this run but work owned by a later slice; nothing was changed")
		case diagnosis.StateNotMeasured:
			return observation(label, "/usr/sbin/sshd", probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded,
				"/usr/sbin/sshd: no filesystem seam is injected for this run, so the sshd binary cannot be checked")
		case diagnosis.StateUnresolved:
			return observation(label, "/usr/sbin/sshd", probe.Unresolved, probe.Indeterminate, probe.ReasonInternalError,
				"/usr/sbin/sshd: the path could not be checked, so its absence is not claimed")
		}
	case testSSHDLabelService:
		switch state {
		case diagnosis.StatePass:
			return observation(label, "sshd.service,ssh.service", probe.Measured, probe.Pass, probe.ReasonOK,
				"the sshd service is running: systemctl is-active sshd.service ssh.service reported \"active\" for sshd.service,ssh.service")
		case diagnosis.StateFail:
			return observation(label, "sshd.service,ssh.service", probe.Measured, probe.Fail, probe.ReasonSSHDAbsent,
				"the sshd service is not running: systemctl is-active sshd.service ssh.service answered \"inactive\" for sshd.service,ssh.service, so no sshd unit is active on this machine; this run reports the state it read and changes nothing")
		case diagnosis.StateNotMeasured:
			return observation(label, "sshd.service,ssh.service", probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded,
				"systemctl is-active sshd.service ssh.service: no command runner is injected for this run")
		case diagnosis.StateUnresolved:
			return observation(label, "sshd.service,ssh.service", probe.Unresolved, probe.Indeterminate, probe.ReasonInternalError,
				"systemctl is-active sshd.service ssh.service: the command produced no answer to classify")
		}
	case testSSHDLabelConfig:
		switch state {
		case diagnosis.StatePass:
			return observation(label, "/etc/ssh/sshd_config", probe.Measured, probe.Pass, probe.ReasonOK,
				"/etc/ssh/sshd_config agrees with the configuration in force reported by sshd -T for every one of the 2 directive(s) the written file sets; written configuration of /etc/ssh/sshd_config: Port 22, PermitRootLogin no; effective configuration reported by sshd -T: port 22, permitrootlogin no")
		case diagnosis.StateFail:
			return observation(label, "/etc/ssh/sshd_config", probe.Measured, probe.Fail, probe.ReasonSSHDConfigDivergence,
				"the written configuration at /etc/ssh/sshd_config is not the configuration in force: sshd -T disagrees with it on 1 of the 2 directive(s) the written file sets (port); written configuration of /etc/ssh/sshd_config: Port 22; effective configuration reported by sshd -T: port 2222")
		case diagnosis.StateNotMeasured:
			return observation(label, "/etc/ssh/sshd_config", probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded,
				"sshd -T: no command runner is injected for this run, so the configuration in force cannot be measured")
		case diagnosis.StateUnresolved:
			return observation(label, "/etc/ssh/sshd_config", probe.Unresolved, probe.Indeterminate, probe.ReasonInternalError,
				"sshd -T: the command produced no answer to classify")
		}
	}
	t.Fatalf("the local.sshd cases have no fixture for the label %q in state %q", label, state)
	return probe.Observation{}
}

// sshdRun builds the run the fixtures describe: one `local.sshd` result carrying its observations in
// the order the probe reports them — the binary, the service state and the effective configuration.
// A label no fixture names is not reported at all, which is how a case expresses a measurement the
// run did not make, distinct from a not-measured observation the probe reports itself.
func sshdRun(t *testing.T, fixtures ...sshdFixture) []probe.Result {
	t.Helper()
	var observations []probe.Observation
	for _, label := range []string{testSSHDLabelBinary, testSSHDLabelService, testSSHDLabelConfig} {
		for _, fixture := range fixtures {
			if fixture.label != label {
				continue
			}
			observations = append(observations, sshdObservation(t, fixture.label, fixture.state))
		}
	}
	if len(observations) == 0 {
		return nil
	}
	return []probe.Result{result("local.sshd", probe.ProbeLocal, observations...)}
}

// TestSSHDGroupOneCasePerID is design §5.2's per-rule case for the `local.sshd` group: one case per
// id, each asserting that the declared row fired, that its conclusion carries the measurement the
// row rested on, and that DependsOn names exactly that probe and no other.
//
// The rows match on the probe's own observation labels, so every case is expressed as the
// observations the probe reported and the labels disappear behind the rule that fired. The two
// absences matter as much as the outcomes: a diverging configuration is a measured divergence and
// never a success, and a configuration that could not be measured is the weaker conclusion rather
// than a configured sshd.
//
// As in the node group, each `wantContains` list pins at least one phrase from the conclusion's own
// sentence: the absent-binary case asserts "Installing sshd is not part of this run", which its
// quoted detail ("installing it is not part of this run") does not carry, and the configured case
// asserts "its effective configuration was measured" and "reports the measurement".
func TestSSHDGroupOneCasePerID(t *testing.T) {
	cases := []struct {
		name string
		run  []sshdFixture
		// want is the rule id the run must fire.
		want string
		// wantContains lists phrases the conclusion must carry: the row's own evidence.
		wantContains []string
		// wantAbsent lists phrases the conclusion must not carry.
		wantAbsent []string
		// wantDepends is the observable set the fired clause rests on, in Match order.
		wantDepends []string
	}{
		{
			name: "a present binary beside a diverging effective configuration reports the divergence",
			run: []sshdFixture{
				{testSSHDLabelBinary, diagnosis.StatePass},
				{testSSHDLabelConfig, diagnosis.StateFail},
			},
			want:         "SSHD_PRESENT_CONFIG_DIVERGENT",
			wantContains: []string{"differs from the written one", "not a success", testSSHDLabelConfig, "/etc/ssh/sshd_config", "Port 22", "port 2222"},
			wantAbsent:   []string{"agrees with the written one"},
			wantDepends:  []string{"local.sshd"},
		},
		{
			name: "an absent binary reports the absence and the later slice",
			run: []sshdFixture{
				{testSSHDLabelBinary, diagnosis.StateFail},
				{testSSHDLabelConfig, diagnosis.StateNotMeasured},
			},
			want:         "SSHD_ABSENT",
			wantContains: []string{"no sshd binary is present", "/usr/sbin/sshd", "later slice", "nothing was changed", "Installing sshd is not part of this run"},
			wantAbsent:   []string{"differs from the written one", "agrees with the written one"},
			wantDepends:  []string{"local.sshd"},
		},
		{
			name: "an effective configuration that could not be measured is the weaker conclusion",
			run: []sshdFixture{
				{testSSHDLabelBinary, diagnosis.StatePass},
				{testSSHDLabelService, diagnosis.StateNotMeasured},
				{testSSHDLabelConfig, diagnosis.StateNotMeasured},
			},
			want:         "SSHD_EFFECTIVE_CONFIG_NOT_MEASURED",
			wantContains: []string{"was not measured", "capability_excluded", "no command runner", "No claim is made about the configuration in force"},
			wantAbsent:   []string{"agrees with the written one", "no sshd binary is present"},
			wantDepends:  []string{"local.sshd"},
		},
		{
			name: "a present binary with an agreeing effective configuration is the positive conclusion",
			run: []sshdFixture{
				{testSSHDLabelBinary, diagnosis.StatePass},
				{testSSHDLabelService, diagnosis.StatePass},
				{testSSHDLabelConfig, diagnosis.StatePass},
			},
			want:         "SSHD_PRESENT_CONFIGURED",
			wantContains: []string{testSSHDLabelBinary, testSSHDLabelConfig, "binary is present", "agrees with the written one", "/usr/sbin/sshd", "/etc/ssh/sshd_config", "its effective configuration was measured", "reports the measurement"},
			wantAbsent:   []string{"differs from the written one", "no sshd binary is present"},
			wantDepends:  []string{"local.sshd"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := diagnosis.Diagnose(sshdRun(t, tc.run...))
			finding := findingFor(t, got, "local.sshd")
			if finding.Rule != tc.want {
				t.Errorf("the run fired %q, want %q", finding.Rule, tc.want)
			}
			if finding.Conclusion == "" {
				t.Fatalf("the rule %q concluded nothing", tc.want)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(finding.Conclusion, want) {
					t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(finding.Conclusion, absent) {
					t.Errorf("the conclusion %q carries %q, which this row must not claim", finding.Conclusion, absent)
				}
			}
			if !reflect.DeepEqual(finding.DependsOn, tc.wantDepends) {
				t.Errorf("the conclusion depends on %v, want %v", finding.DependsOn, tc.wantDepends)
			}
		})
	}
}

// nodePlatformObservation builds the one observation `local.env` reports in one state, with the
// label, target, reason code and verbatim detail the probe's own wording carries: the classification
// and the architecture in the target, and the classification's own detail text.
func nodePlatformObservation(t *testing.T, state diagnosis.State) probe.Observation {
	t.Helper()
	switch state {
	case diagnosis.StateFail:
		return observation("platform", "windows-native/x86_64", probe.Measured, probe.Fail, probe.ReasonNodePlatformUnsupported,
			"native Windows node (GOOS \"windows\", architecture \"x86_64\"): this tool does not operate on Windows itself, and WSL2 is the supported path on a Windows machine; nothing was changed")
	case diagnosis.StateUnresolved:
		return observation("platform", "unknown/unknown", probe.Unresolved, probe.Indeterminate, probe.ReasonPlatformUnknown,
			"the platform signals (GOOS \"plan9\", architecture \"unknown\") match no supported classification (linux, macOS, wsl2 or native Windows); no platform is assumed and no default is guessed")
	case diagnosis.StatePass:
		return observation("platform", "linux/x86_64", probe.Measured, probe.Pass, probe.ReasonOK,
			"linux node (architecture \"x86_64\"): systemd is the running service manager; this run detects and reports the environment and changes nothing")
	}
	t.Fatalf("the node.platform cases have no observation for the state %q", state)
	return probe.Observation{}
}

// TestNodePlatformOneCasePerID is design §5.2's per-rule case for the `node.platform` group: one
// case per id, each asserting that the classification `local.env` measured decides the conclusion
// and that the conclusion quotes the classification it rests on.
//
// The refusal names WSL2 as the supported path and decides nothing about transport; the unknown
// case assumes no platform; the supported case quotes the classification and the architecture the
// observation's target carries.
//
// Each `wantContains` list also pins at least one phrase that appears only in the conclusion's own
// sentence. The fixture detail and the conclusion deliberately share wording (the conclusion quotes
// the observation), so a list built only from shared phrases cannot tell a dropped conclusion claim
// from a quoted one — the refusal case's own sentence, for instance, must be asserted through
// "cannot host a supported node" and "No transport is decided here", which the detail never carries.
func TestNodePlatformOneCasePerID(t *testing.T) {
	cases := []struct {
		name string
		// state is the state `local.env` reports, the group's whole evidence.
		state diagnosis.State
		// want is the rule id the run must fire.
		want string
		// wantContains lists phrases the conclusion must carry.
		wantContains []string
		// wantAbsent lists phrases the conclusion must not carry.
		wantAbsent []string
	}{
		{
			name:         "native Windows is a measured refusal that names WSL2",
			state:        diagnosis.StateFail,
			want:         "NODE_PLATFORM_REFUSED_NATIVE_WINDOWS",
			wantContains: []string{"measured refusal", "native Windows", "WSL2", "nothing was changed", "transport layer", "cannot host a supported node", "No transport is decided here"},
			wantAbsent:   []string{"node platform is supported"},
		},
		{
			name:         "signals matching no supported classification assume no platform",
			state:        diagnosis.StateUnresolved,
			want:         "NODE_PLATFORM_UNKNOWN",
			wantContains: []string{"platform is unknown", "platform_unknown", "No platform is assumed", "unknown/unknown"},
			wantAbsent:   []string{"node platform is supported"},
		},
		{
			name:         "a supported classification quotes the classification and the architecture",
			state:        diagnosis.StatePass,
			want:         "NODE_PLATFORM_SUPPORTED",
			wantContains: []string{"node platform is supported", "linux", "x86_64", "changes nothing", "it changes nothing"},
			wantAbsent:   []string{"measured refusal"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := []probe.Result{result("local.env", probe.ProbeLocal, nodePlatformObservation(t, tc.state))}
			got := diagnosis.Diagnose(run)
			finding := findingFor(t, got, "node.platform")
			if finding.Rule != tc.want {
				t.Errorf("the run fired %q, want %q", finding.Rule, tc.want)
			}
			if finding.Conclusion == "" {
				t.Fatalf("the rule %q concluded nothing", tc.want)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(finding.Conclusion, want) {
					t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(finding.Conclusion, absent) {
					t.Errorf("the conclusion %q carries %q, which this row must not claim", finding.Conclusion, absent)
				}
			}
			if !reflect.DeepEqual(finding.DependsOn, []string{"local.env"}) {
				t.Errorf("the conclusion depends on %v, want the classification probe", finding.DependsOn)
			}
		})
	}
}

// TestSSHDGroupCounterfactuals is the `local.sshd` group's triangulation: the runs the label and
// the row order exist for.
//
// Three properties are asserted rather than assumed. A diverging configuration beside a present
// binary fires the divergence conclusion and never the configured one (R-HR-18). The default live
// case — every one of the probe's three observations not measured, as the deny-all seams produce —
// fires the not-measured conclusion and makes no claim about the configuration in force. And a
// present binary with an agreeing configuration fires the positive conclusion even beside a stopped
// service: the group does not require the service, while the fact layer still reports the stopped
// service as its own finding and no conclusion claims the binary is absent.
func TestSSHDGroupCounterfactuals(t *testing.T) {
	t.Run("a diverging configuration never fires the configured conclusion", func(t *testing.T) {
		run := sshdRun(t,
			sshdFixture{testSSHDLabelBinary, diagnosis.StatePass},
			sshdFixture{testSSHDLabelConfig, diagnosis.StateFail},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "local.sshd")
		if finding.Rule != "SSHD_PRESENT_CONFIG_DIVERGENT" {
			t.Fatalf("the run fired %q, want %q", finding.Rule, "SSHD_PRESENT_CONFIG_DIVERGENT")
		}
		if finding.Rule == "SSHD_PRESENT_CONFIGURED" {
			t.Errorf("a diverging configuration fired the configured conclusion")
		}
		if strings.Contains(finding.Conclusion, "agrees with the written one") {
			t.Errorf("the divergence conclusion claims the configurations agree: %s", finding.Conclusion)
		}
	})

	t.Run("the default live case is the not-measured conclusion", func(t *testing.T) {
		// A live run under the zero-execution boundary: no filesystem seam and no command runner, so
		// every `local.sshd` observation reports the capability it was denied. The group must report
		// the weaker conclusion and neither a configured nor an absent sshd.
		run := sshdRun(t,
			sshdFixture{testSSHDLabelBinary, diagnosis.StateNotMeasured},
			sshdFixture{testSSHDLabelService, diagnosis.StateNotMeasured},
			sshdFixture{testSSHDLabelConfig, diagnosis.StateNotMeasured},
		)
		finding := findingFor(t, diagnosis.Diagnose(run), "local.sshd")
		if finding.Rule != "SSHD_EFFECTIVE_CONFIG_NOT_MEASURED" {
			t.Fatalf("the run fired %q, want %q", finding.Rule, "SSHD_EFFECTIVE_CONFIG_NOT_MEASURED")
		}
		for _, want := range []string{"capability_excluded", "No claim is made about the configuration in force"} {
			if !strings.Contains(finding.Conclusion, want) {
				t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
			}
		}
		for _, absent := range []string{"no sshd binary is present", "agrees with the written one"} {
			if strings.Contains(finding.Conclusion, absent) {
				t.Errorf("the default live case claims %q: %s", absent, finding.Conclusion)
			}
		}
	})

	t.Run("a stopped service beside a present binary keeps both facts apart", func(t *testing.T) {
		run := sshdRun(t,
			sshdFixture{testSSHDLabelBinary, diagnosis.StatePass},
			sshdFixture{testSSHDLabelService, diagnosis.StateFail},
			sshdFixture{testSSHDLabelConfig, diagnosis.StatePass},
		)
		got := diagnosis.Diagnose(run)

		// The group answers its own question — binary present, configuration in force agreeing — and
		// does not require the service state.
		group := findingFor(t, got, "local.sshd")
		if group.Rule != "SSHD_PRESENT_CONFIGURED" {
			t.Fatalf("the run fired %q, want %q", group.Rule, "SSHD_PRESENT_CONFIGURED")
		}

		// The stopped service is still its own finding: the fact layer reports it under
		// `local.sshd.fact`, and it is the measured failure of the service observation, not the
		// binary's absence.
		fact := findingFor(t, got, "local.sshd.fact")
		if fact.Rule != diagnosis.RuleID("local.sshd", diagnosis.StateFail) {
			t.Errorf("the fact layer fired %q, want the failure of the stopped service", fact.Rule)
		}
		for _, want := range []string{testSSHDLabelService, "not running"} {
			if !strings.Contains(fact.Conclusion, want) {
				t.Errorf("the fact conclusion %q does not name the stopped service via %q", fact.Conclusion, want)
			}
		}

		// And no conclusion of the run may claim the binary is absent.
		for _, finding := range got.Findings {
			for _, absent := range []string{"no sshd binary is present", "binary is not present"} {
				if strings.Contains(finding.Conclusion, absent) {
					t.Errorf("the finding %q claims %q: %s", finding.Rule, absent, finding.Conclusion)
				}
			}
		}
	})
}

// TestTLSFactQuestionsReachTrustStoreConclusions is the RG-3 case for the restated `tls.truststore`
// question: the two platform limitations §5.2 split into hand-named ids reach a conclusion through
// the derived `TLS_TRUSTSTORE_UNRESOLVED` id, each carrying its own reason code and verbatim detail.
//
// The macOS limitation is quoted from the probe's own documented sentence — the constant is
// unexported, so the case pins a distinctive substring — and the override case pins the wording that
// says the platform verifier is bypassed. Both are absences: the conclusion reports what was not
// measured, never a rejected or an accepted chain, and the platform/override split that §5.2's
// hand-named ids carried survives in exactly the reason code and detail this case asserts.
func TestTLSFactQuestionsReachTrustStoreConclusions(t *testing.T) {
	t.Run("the macOS trust-store limitation reaches a conclusion through the derived id", func(t *testing.T) {
		run := []probe.Result{
			result("tls.truststore", probe.ProbeTLS,
				observation("tls 443 truststore", "example.com:443", probe.Unresolved, probe.Indeterminate, probe.ReasonTrustStorePlatformUnavailable,
					"tls example.com:443: the local trust store's answer for the chain presented by the declared TLS target cannot be resolved on this platform, and the documented limitation is that Go cannot enumerate macOS system roots, and keychain trust is only visible through the platform verifier when no explicit root pool is supplied"),
			),
		}
		finding := findingFor(t, diagnosis.Diagnose(run), "tls.truststore")
		if want := diagnosis.RuleID("tls.truststore", diagnosis.StateUnresolved); finding.Rule != want {
			t.Fatalf("the run fired %q, want %q", finding.Rule, want)
		}
		for _, want := range []string{"truststore_platform_unavailable", "Go cannot enumerate macOS system roots", "attempted and unresolved"} {
			if !strings.Contains(finding.Conclusion, want) {
				t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
			}
		}
		if strings.Contains(finding.Conclusion, "rejected the chain") {
			t.Errorf("an absence was reported as a rejection: %s", finding.Conclusion)
		}
	})

	t.Run("the override case reaches the same id carrying the override wording", func(t *testing.T) {
		run := []probe.Result{
			result("tls.truststore", probe.ProbeTLS,
				observation("tls 443 truststore", "example.com:443", probe.Unresolved, probe.Indeterminate, probe.ReasonTrustStoreOverridePlatformBypass,
					"tls example.com:443: SSL_CERT_FILE=\"/tmp/roots.pem\" is set for this run, so Go loads that pool in place of the platform trust store and the platform verifier is bypassed; the chain is reported as unresolved rather than accepted or rejected, because an answer about that pool would not be an answer about the local trust store's answer for the chain presented by the declared TLS target"),
			),
		}
		finding := findingFor(t, diagnosis.Diagnose(run), "tls.truststore")
		if want := diagnosis.RuleID("tls.truststore", diagnosis.StateUnresolved); finding.Rule != want {
			t.Fatalf("the run fired %q, want %q", finding.Rule, want)
		}
		for _, want := range []string{"truststore_override_platform_bypass", "the platform verifier is bypassed", "SSL_CERT_FILE", "attempted and unresolved"} {
			if !strings.Contains(finding.Conclusion, want) {
				t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
			}
		}
	})
}

// TestDeferredHandNamedRuleIDs is this slice's deferral assertion, not an assumption: the hand-named
// tls spellings §5.2 prints are not declared, because the two `tls.*` groups were restated to the
// derived fact ids, and `NODE_WSL2_SYSTEMD_ABSENT` is not implemented because the service-manager
// signal has no structured home in this slice and matching a conclusion on wording is forbidden
// (R-HR-07).
//
// One collision is stated rather than hidden: `TLS_INTERCEPTION_UNRESOLVED` is both one of §5.2's
// hand-named spellings and the derived id of `tls.interception` in the unresolved state, so it
// necessarily stays in the vocabulary — carrying the derived fact conclusion, which is what the
// restatement decided. That is why the case asserts the exact `TLS_*`/`TRUSTSTORE_*` set instead of
// only listing forbidden spellings: the set is the eight derived ids of the two probes in the fact
// slots' declaration order (FAIL, UNRESOLVED, NOT_MEASURED, PASS), and an id this case's list did
// not foresee fails the equality too.
func TestDeferredHandNamedRuleIDs(t *testing.T) {
	notDeclared := []string{
		"TLS_INTERCEPTION_DETECTED",
		"TLS_CHAIN_AS_EXPECTED",
		"TRUSTSTORE_ACCEPTS_CHAIN",
		"TRUSTSTORE_REJECTS_CHAIN",
		"TRUSTSTORE_UNRESOLVED_PLATFORM",
		"TRUSTSTORE_UNRESOLVED_OVERRIDE",
		"NODE_WSL2_SYSTEMD_ABSENT",
	}
	ids := diagnosis.AllRuleIDs()
	for _, id := range notDeclared {
		if slices.Contains(ids, id) {
			t.Errorf("AllRuleIDs() declares %q, which this slice deliberately does not implement", id)
		}
	}

	wantTLS := []string{
		"TLS_INTERCEPTION_FAIL",
		"TLS_INTERCEPTION_UNRESOLVED",
		"TLS_INTERCEPTION_NOT_MEASURED",
		"TLS_INTERCEPTION_PASS",
		"TLS_TRUSTSTORE_FAIL",
		"TLS_TRUSTSTORE_UNRESOLVED",
		"TLS_TRUSTSTORE_NOT_MEASURED",
		"TLS_TRUSTSTORE_PASS",
	}
	var gotTLS []string
	for _, id := range ids {
		if strings.HasPrefix(id, "TLS_") || strings.HasPrefix(id, "TRUSTSTORE_") {
			gotTLS = append(gotTLS, id)
		}
	}
	if !slices.Equal(gotTLS, wantTLS) {
		t.Errorf("the tls vocabulary is %v, want exactly the restated derived ids %v", gotTLS, wantTLS)
	}
}

// TestRemainingGroupsOpenQuestionNeededStates pins what the open questions of the two new groups
// report when nothing matched: the labeled `local.sshd` states in declaration order, and the
// label-less `local.env` states with the original "<probe> <STATE>" spelling. The spelling reaches
// the payload and the human projection, so it is a contract, and the labeled form is the only way
// a reader can tell two needs of one probe apart.
func TestRemainingGroupsOpenQuestionNeededStates(t *testing.T) {
	got := diagnosis.Diagnose(nil)

	sshd := openQuestionFor(t, got, "local.sshd")
	wantSSHD := []string{
		"local.sshd (effective config) FAIL",
		"local.sshd (binary present) FAIL",
		"local.sshd (effective config) NOT_MEASURED",
		"local.sshd (binary present) PASS",
		"local.sshd (effective config) PASS",
	}
	if !reflect.DeepEqual(sshd.NeededStates, wantSSHD) {
		t.Errorf("the open question for %q needs %v, want %v", sshd.Question, sshd.NeededStates, wantSSHD)
	}

	platform := openQuestionFor(t, got, "node.platform")
	wantPlatform := []string{"local.env FAIL", "local.env UNRESOLVED", "local.env PASS"}
	if !reflect.DeepEqual(platform.NeededStates, wantPlatform) {
		t.Errorf("the open question for %q needs %v, want %v", platform.Question, platform.NeededStates, wantPlatform)
	}
}

// TestEvidenceFindingsCarryTheObservationsTheyRestedOn is the case for `Finding.Evidence` (design
// §3.2's dated PR 13 note): a finding hands a consumer the observations its conclusion rests on as
// structured facts — the probe, the matchable state, and the observation's own label, target,
// reason code and detail — so a consumer that must name the measured target it rejected does not
// parse the prose conclusion.
//
// The facts `matchRule` selected are exactly what the finding must carry, and the cases below pin
// the three properties that make the field usable rather than decorative: the evidence is the
// fired rule's own match (not the whole run, and for a hand-named id carried by several clauses
// not the sibling clause's observations), every finding of a run carries at least one observation
// because a fired rule has at least one satisfied need, and what a caller gets back is a copy —
// mutating it cannot reach a later `Diagnose`.
func TestEvidenceFindingsCarryTheObservationsTheyRestedOn(t *testing.T) {
	t.Run("a derived fact finding carries exactly the observation it states", func(t *testing.T) {
		run := []probe.Result{
			result("egress.cf.443", probe.ProbeEgress,
				observation("tcp 443 region1", "region1.v2.argotunnel.com:443", probe.Measured, probe.Fail, probe.ReasonConnRefused,
					"dial tcp region1.v2.argotunnel.com:443: connect: connection refused"),
			),
		}
		finding := findingFor(t, diagnosis.Diagnose(run), "egress.cf.443")
		want := diagnosis.Facts(run)
		if !reflect.DeepEqual(finding.Evidence, want) {
			t.Fatalf("the finding carries evidence %+v, want exactly the observation it states %+v", finding.Evidence, want)
		}
	})

	t.Run("a hand-named id carried by several clauses carries the fired clause's observations", func(t *testing.T) {
		// The `ssh.destination` weaker id has one clause per (public probe, absence) pair. This run's
		// port 22 measurement failed, so only the port 443 clause can fire: the evidence must be that
		// clause's own two observations — the port 443 absence and the hub failure — and must never
		// include the sibling public probe, which the clause never needed.
		run := []probe.Result{
			result("egress.ssh.known", probe.ProbeEgress,
				observation("tcp 22 ssh banner", "github.com:22", probe.Measured, probe.Fail, probe.ReasonBannerNotSSH,
					"tcp github.com:22: something answered and it is not an SSH server"),
			),
			result("egress.ssh.443", probe.ProbeEgress,
				observation("tcp 443 ssh banner", "ssh.github.com:443", probe.Unresolved, probe.Indeterminate, probe.ReasonTLSHandshakeUnresolved,
					"tls ssh.github.com:443: the handshake produced no classification"),
			),
			result("egress.hub.direct", probe.ProbeEgress,
				observation("tcp 22", "203.0.113.10:22", probe.Measured, probe.Fail, probe.ReasonConnRefused,
					"dial tcp 203.0.113.10:22: connect: connection refused"),
			),
		}
		finding := findingFor(t, diagnosis.Diagnose(run), "ssh.destination")
		if want := "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED"; finding.Rule != want {
			t.Fatalf("the run fired %q, want %q", finding.Rule, want)
		}
		if len(finding.Evidence) != 2 {
			t.Fatalf("the fired clause carries %d observations, want the two it matched: %+v", len(finding.Evidence), finding.Evidence)
		}
		wantProbes := []string{"egress.ssh.443", "egress.hub.direct"}
		for i, want := range wantProbes {
			if got := finding.Evidence[i].Probe; got != want {
				t.Errorf("evidence[%d] names probe %q, want %q: the evidence is in Match order", i, got, want)
			}
		}
		if got := finding.Evidence[0].State; got != diagnosis.StateUnresolved {
			t.Errorf("the public observation's evidence state = %q, want %q", got, diagnosis.StateUnresolved)
		}
		if got := finding.Evidence[1].State; got != diagnosis.StateFail {
			t.Errorf("the hub observation's evidence state = %q, want %q", got, diagnosis.StateFail)
		}
		for _, fact := range finding.Evidence {
			if fact.Probe == "egress.ssh.known" {
				t.Errorf("the fired clause's evidence names egress.ssh.known, which the clause never needed")
			}
		}
	})

	t.Run("every finding of a run carries at least one observation", func(t *testing.T) {
		run := append(factsFixture(),
			result("egress.hub.direct", probe.ProbeEgress,
				observation("tcp 22", "203.0.113.10:22", probe.Measured, probe.Fail, probe.ReasonConnRefused,
					"dial tcp 203.0.113.10:22: connect: connection refused")),
			result("egress.ssh.known", probe.ProbeEgress,
				observation("tcp 22 ssh banner", "github.com:22", probe.Measured, probe.Pass, probe.ReasonOK,
					"tcp github.com:22: an SSH identification string answered")),
		)
		got := diagnosis.Diagnose(run)
		if len(got.Findings) == 0 {
			t.Fatal("the run produced no findings, so the case proves nothing")
		}
		for _, finding := range got.Findings {
			if len(finding.Evidence) == 0 {
				t.Errorf("the finding %q (%s) carries no evidence: %s", finding.Question, finding.Rule, finding.Conclusion)
			}
		}
	})

	t.Run("the evidence is a copy of the matched observations", func(t *testing.T) {
		run := []probe.Result{
			result("egress.hub.direct", probe.ProbeEgress,
				observation("tcp 22", "203.0.113.10:22", probe.Measured, probe.Fail, probe.ReasonConnRefused,
					"dial tcp 203.0.113.10:22: connect: connection refused")),
		}
		baseline := diagnosis.Diagnose(run)
		mutated := diagnosis.Diagnose(run)
		if len(mutated.Findings) == 0 || len(mutated.Findings[0].Evidence) == 0 {
			t.Fatalf("the run produced no finding with evidence, so the case proves nothing: %+v", mutated.Findings)
		}
		for i := range mutated.Findings {
			for j := range mutated.Findings[i].Evidence {
				mutated.Findings[i].Evidence[j].Probe = "mutated"
				mutated.Findings[i].Evidence[j].State = diagnosis.StatePass
				mutated.Findings[i].Evidence[j].Observation.Target = "mutated.example:22"
			}
		}
		again := diagnosis.Diagnose(run)
		if !reflect.DeepEqual(again, baseline) {
			t.Errorf("mutating the evidence a caller got back changed a later Diagnose:\n got %+v\nwant %+v", again, baseline)
		}
	})
}
