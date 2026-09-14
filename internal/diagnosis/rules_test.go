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
