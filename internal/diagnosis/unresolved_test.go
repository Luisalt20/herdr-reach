package diagnosis_test

// This file is R-HR-NF-03's engine-level evidence: an unresolved or not-measured observation never
// yields a confident conclusion, it weakens only the conclusions that depend on it, and it stays
// visible as a gap in the run.
//
// The mechanism is structural — a need states an exact resolution and verdict and matchNeeds
// compares that triple, so a rule that needs a measured pass or failure cannot be satisfied by an
// absence — and these cases are the evidence over the real table rather than over a restatement of
// it: they walk diagnosis.Rules() and drive the engine with runs whose observations are the
// absences. Nothing here adds production code; a row the property does not hold for is reported by
// the case that walks the table.
//
// The fixtures are the ones rules_test.go builds (result, observation, observationInState and the
// per-group builders), so these cases reason over the same observations the rule cases do and the
// two files cannot drift into two vocabularies.

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// confidentNeed reports whether a need demands a measured observation: a measured resolution with a
// definite verdict. Those are the needs a confident rule rests on, and the ones no absence can
// satisfy (R-HR-NF-03).
func confidentNeed(need diagnosis.Need) bool {
	return need.Resolution == probe.Measured && (need.Verdict == probe.Pass || need.Verdict == probe.Fail)
}

// needsMeasurement reports whether any need of a rule demands a measured observation.
func needsMeasurement(rule diagnosis.Rule) bool {
	for _, need := range rule.Match {
		if confidentNeed(need) {
			return true
		}
	}
	return false
}

// rowSatisfied reports whether every need of one row is satisfied by the run's facts, comparing the
// exported Fact fields the way matchNeeds compares them: the probe, resolution and verdict exactly,
// and the observation's label when the need declares one. It is a second, independent reading of
// the same contract, so a case can tell a clause the run's own facts support from one the engine
// must never have matched.
func rowSatisfied(rule diagnosis.Rule, facts []diagnosis.Fact) bool {
	if len(rule.Match) == 0 {
		return false
	}
	for _, need := range rule.Match {
		satisfied := false
		for _, fact := range facts {
			if fact.Probe != need.Probe || fact.Resolution != need.Resolution || fact.Verdict != need.Verdict {
				continue
			}
			if need.Label != "" && fact.Observation.Label != need.Label {
				continue
			}
			satisfied = true
			break
		}
		if !satisfied {
			return false
		}
	}
	return true
}

// absenceClauseSatisfied reports whether some row carrying one id needs no measured observation and
// is satisfied by the run's facts. An id may be carried by several rows (rules.go), so an id that
// appears on a run which measured nothing was produced by such a weaker clause — never by a
// confident one, whose needs the run's absences cannot satisfy.
func absenceClauseSatisfied(id string, facts []diagnosis.Fact) bool {
	for _, rule := range diagnosis.Rules() {
		if rule.ID != id || needsMeasurement(rule) {
			continue
		}
		if rowSatisfied(rule, facts) {
			return true
		}
	}
	return false
}

// rulesWithID returns every row of the table carrying one id. Derived fact ids are one row each,
// but a hand-named id may be a disjunction of clauses, so a case must not assume one row per id.
func rulesWithID(id string) []diagnosis.Rule {
	var found []diagnosis.Rule
	for _, rule := range diagnosis.Rules() {
		if rule.ID == id {
			found = append(found, rule)
		}
	}
	return found
}

// probeKindOf reads one registered probe's question family, so a fixture result declares the kind
// the registry does instead of a hardcoded one.
func probeKindOf(t *testing.T, probeName string) probe.ProbeKind {
	t.Helper()
	for _, entry := range probe.Registry() {
		if entry.Name == probeName {
			return entry.Kind
		}
	}
	t.Fatalf("no registered probe is named %q", probeName)
	return ""
}

// absenceRun reports the probe of every confident need of one row as a single absence of the given
// state, in Match order, each probe once. Reporting the probes rather than measured answers is the
// whole scenario: the run measured nothing, and the row under test must not be able to conclude
// anything from it.
func absenceRun(t *testing.T, rule diagnosis.Rule, state diagnosis.State) []probe.Result {
	t.Helper()
	var results []probe.Result
	for _, need := range rule.Match {
		if !confidentNeed(need) {
			continue
		}
		seen := false
		for _, existing := range results {
			if existing.Probe == need.Probe {
				seen = true
				break
			}
		}
		if seen {
			continue
		}
		results = append(results, result(need.Probe, probeKindOf(t, need.Probe), observationInState(t, need.Probe, state)))
	}
	return results
}

// TestUnresolvedConfidentClausesAreUnreachable walks every row of the table that needs a measured
// observation and drives the engine with a run that reports each of those probes as an absence:
// unresolved in one round, not measured in the other.
//
// The row under test must never reach the findings, and every finding the run does produce must be
// carried by a clause that needs no measurement and is satisfied by the run's own facts — so on a
// run that measured nothing, only an absence can conclude anything, and the question is answered by
// that weaker clause rather than left to the confident one.
//
// One nuance is asserted rather than assumed. A hand-named id may be carried by several rows
// (rules.go), so "the id never fires" is too strong for an id whose rows include both a confident
// clause and an absence clause: the absence round fires such an id through the weaker clause, with
// the weaker wording. What must hold — and what this case asserts — is that no finding is carried
// by a clause that needs a measurement, which for a confident-only id means the id never fires at
// all. `CF_HTTP2_ADVISORY_NOT_ASSESSED` is the current mixed id, and the wording it fired with was
// the "produced no answer" / "was not measured" branch, never the measured-failure branch.
func TestUnresolvedConfidentClausesAreUnreachable(t *testing.T) {
	absences := []struct {
		name  string
		state diagnosis.State
	}{
		{"unresolved", diagnosis.StateUnresolved},
		{"not measured", diagnosis.StateNotMeasured},
	}
	for _, absence := range absences {
		for i, rule := range diagnosis.Rules() {
			if !needsMeasurement(rule) {
				continue
			}
			t.Run(fmt.Sprintf("%02d %s on %s", i, rule.ID, absence.name), func(t *testing.T) {
				run := absenceRun(t, rule, absence.state)
				facts := diagnosis.Facts(run)
				got := diagnosis.Diagnose(run)

				// Every finding of the run must be carried by a clause that needs no measurement and
				// is satisfied by the run's own facts: on a run that measured nothing, only an absence
				// can conclude anything. The row under test demands a measured observation, so its own
				// clause can never be the one that carried its id — an id may be a disjunction of
				// clauses (rules.go), and then the weaker clause, never the confident one, is what
				// fired; the message names the row under test when its id is the one that failed, so a
				// failure points back at the row being walked.
				for _, finding := range got.Findings {
					if absenceClauseSatisfied(finding.Rule, facts) {
						continue
					}
					note := ""
					if finding.Rule == rule.ID {
						note = fmt.Sprintf(" (the fired id is the row under test, %q)", rule.ID)
					}
					t.Errorf("the finding %q%s was produced on a run that measured nothing, but no clause of its id needs no measurement and is satisfied by the run's facts: %s", finding.Rule, note, finding.Conclusion)
				}
			})
		}
	}
}

// TestUnresolvedWeakerRowNamesTheProbeThatProducedNoAnswer is R-HR-NF-03's own scenario for the
// `ssh.destination` group: a measured public-SSH failure and a hub failure beside a second public
// probe that did not answer. The group must fire the weaker row rather than either
// `SSH_DEST_BLOCKED_BY_PUBLIC_SSH*` id, and the conclusion must name the probe that produced no
// answer — the second public probe, never the failed one.
//
// The two absences are told apart by their own resolution, not by wording: an attempt that produced
// no answer and a measurement that was never made share one id by design (§5.2 gives the row one),
// and the conclusion says which one happened. This case asserts both directions.
func TestUnresolvedWeakerRowNamesTheProbeThatProducedNoAnswer(t *testing.T) {
	cases := []struct {
		name string
		// state is the state the second public probe reports.
		state diagnosis.State
		// want is the rule id the run must fire.
		want string
		// wantSaid lists the wording the conclusion must carry about the absence. The target is
		// part of it only when the observation carried one: a measurement that was never made has
		// no target, and the conclusion names the probe and the reason code instead.
		wantSaid []string
		// wantUnsaid is the wording the other absence would have produced.
		wantUnsaid string
	}{
		{
			name:       "an attempt that produced no answer",
			state:      diagnosis.StateUnresolved,
			want:       "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED",
			wantSaid:   []string{"egress.ssh.443", "ssh.github.com:443", "dns_unresolved", "was attempted and produced no answer"},
			wantUnsaid: "was not made",
		},
		{
			name:       "a measurement that was never made",
			state:      diagnosis.StateNotMeasured,
			want:       "SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED",
			wantSaid:   []string{"egress.ssh.443", "capability_excluded", "was not made"},
			wantUnsaid: "was attempted and produced no answer",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := sshDestinationRun(t,
				sshDestinationFixture{"egress.ssh.known", diagnosis.StateFail},
				sshDestinationFixture{"egress.ssh.443", tc.state},
				sshDestinationFixture{"egress.hub.direct", diagnosis.StateFail},
			)
			finding := findingFor(t, diagnosis.Diagnose(run), "ssh.destination")
			if finding.Rule != tc.want {
				t.Fatalf("the run fired %q, want %q", finding.Rule, tc.want)
			}
			if finding.Conclusion == "" {
				t.Fatalf("the rule %q concluded nothing", tc.want)
			}
			for _, want := range append([]string{"not established"}, tc.wantSaid...) {
				if !strings.Contains(finding.Conclusion, want) {
					t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
				}
			}
			if strings.Contains(finding.Conclusion, tc.wantUnsaid) {
				t.Errorf("the conclusion %q carries %q, which is the other absence's wording", finding.Conclusion, tc.wantUnsaid)
			}
			if strings.Contains(finding.Conclusion, "github.com:22") {
				t.Errorf("the conclusion %q names the failed public probe's target as the absence", finding.Conclusion)
			}
			for _, absent := range []string{"outbound SSH is allowed", "outbound SSH works"} {
				if strings.Contains(finding.Conclusion, absent) {
					t.Errorf("the weaker conclusion claims %q: %s", absent, finding.Conclusion)
				}
			}
			if want := []string{"egress.ssh.443", "egress.hub.direct"}; !reflect.DeepEqual(finding.DependsOn, want) {
				t.Errorf("the conclusion depends on %v, want the weaker clause's own observables %v", finding.DependsOn, want)
			}
		})
	}
}

// TestUnresolvedAbsencesKeepTheirOwnState is obligations 1 and 2 at the engine layer: a
// not-measured observation produces a not-measured conclusion naming the missing input or
// capability through its reason code and verbatim detail, and an attempted-but-unresolved
// observation produces an unresolved conclusion naming the probe — neither is rendered as a pass, a
// failure, or a silent omission.
//
// The probe is `egress.hub.direct`, whose fact question is answered by one observation, so the
// conclusion under test cannot borrow a sibling's state. The run leaves the question answered: an
// absent conclusion and an open question would be the silent omission this case refuses.
func TestUnresolvedAbsencesKeepTheirOwnState(t *testing.T) {
	cases := []struct {
		name string
		// state is the state of the probe's one observation.
		state diagnosis.State
		// want is the derived fact rule id of that state.
		want string
		// wantSaid lists the phrases the conclusion must carry: the reason code and the detail.
		wantSaid []string
		// wantUnsaid lists the phrases of every other rendering of the same observation.
		wantUnsaid []string
	}{
		{
			name:       "an attempt that produced no answer",
			state:      diagnosis.StateUnresolved,
			want:       "EGRESS_HUB_DIRECT_UNRESOLVED",
			wantSaid:   []string{"attempted and unresolved", "egress.hub.direct", string(probe.ReasonDNSUnresolved), "resolver timeout"},
			wantUnsaid: []string{"not measured", "measured and passed", "measured and failed"},
		},
		{
			name:       "a measurement that was never made",
			state:      diagnosis.StateNotMeasured,
			want:       "EGRESS_HUB_DIRECT_NOT_MEASURED",
			wantSaid:   []string{"not measured", "egress.hub.direct", string(probe.ReasonInputMissingHub), "no --hub was supplied"},
			wantUnsaid: []string{"attempted and unresolved", "measured and passed", "measured and failed"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := []probe.Result{
				result("egress.hub.direct", probe.ProbeEgress, observationInState(t, "egress.hub.direct", tc.state)),
			}
			got := diagnosis.Diagnose(run)

			findings := findingsFor(got, "hub.reachability")
			if len(findings) != 1 {
				t.Fatalf("the run produced %d findings for the question, want one: %+v", len(findings), findings)
			}
			finding := findings[0]
			if finding.Rule != tc.want {
				t.Fatalf("the run fired %q, want %q", finding.Rule, tc.want)
			}
			if finding.Conclusion == "" {
				t.Fatalf("the rule %q concluded nothing", tc.want)
			}
			for _, want := range tc.wantSaid {
				if !strings.Contains(finding.Conclusion, want) {
					t.Errorf("the conclusion %q does not carry %q", finding.Conclusion, want)
				}
			}
			for _, unsaid := range tc.wantUnsaid {
				if strings.Contains(finding.Conclusion, unsaid) {
					t.Errorf("the conclusion %q carries %q, which belongs to another rendering", finding.Conclusion, unsaid)
				}
			}
			if open := openQuestionsFor(got, "hub.reachability"); len(open) != 0 {
				t.Errorf("the question stayed open for a run that reported an absence: %+v", open)
			}
			if want := []string{"egress.hub.direct"}; !reflect.DeepEqual(finding.DependsOn, want) {
				t.Errorf("the conclusion depends on %v, want %v", finding.DependsOn, want)
			}
		})
	}
}

// TestUnresolvedProbeDoesNotPoisonUnrelatedConclusions is R-HR-NF-03's own scenario: an unresolved
// probe whose measurement no conclusion depends on leaves every established conclusion unchanged
// and stays visible as a gap.
//
// The candidate probe is derived, not hardcoded: the case collects the probes the base run's
// findings depend on and the probes the rules of its answered questions name, then takes the first
// registered probe that is neither. A table that grows a new dependency moves the candidate rather
// than silently weakening the case, and if no unrelated probe remains the case fails loudly.
func TestUnresolvedProbeDoesNotPoisonUnrelatedConclusions(t *testing.T) {
	baseRun := append(sshdRun(t,
		sshdFixture{testSSHDLabelBinary, diagnosis.StatePass},
		sshdFixture{testSSHDLabelConfig, diagnosis.StatePass},
	), []probe.Result{
		result("local.env", probe.ProbeLocal, nodePlatformObservation(t, diagnosis.StatePass)),
		result("egress.ssh.known", probe.ProbeEgress, sshDestinationObservation(t, "egress.ssh.known", diagnosis.StatePass)),
		result("egress.hub.direct", probe.ProbeEgress, sshDestinationObservation(t, "egress.hub.direct", diagnosis.StateFail)),
		result("egress.cf.443", probe.ProbeEgress, cloudflareObservation(t, "egress.cf.443", "region1", diagnosis.StatePass)),
		result("tls.truststore", probe.ProbeTLS, observation("tls 443 truststore", "example.com:443", probe.Measured, probe.Pass, probe.ReasonOK,
			"tls example.com:443: the local trust store accepted the chain (observed issuer \"Example Root\", verification code \"ok\")")),
	}...)
	base := diagnosis.Diagnose(baseRun)
	if len(base.Findings) < 2 {
		t.Fatalf("the base run established %d conclusions, want several for the comparison", len(base.Findings))
	}
	if len(base.OpenQuestions) == 0 {
		t.Fatalf("the base run left no question open, so the case could not show that an unrelated question is untouched")
	}

	// The related set is derived from the conclusions and the table: every probe a finding depends
	// on, and every probe any rule of an answered question names. A fact for a probe outside it
	// cannot satisfy any need of an answered question, so no established conclusion can move.
	related := map[string]bool{}
	for _, finding := range base.Findings {
		for _, probeName := range finding.DependsOn {
			related[probeName] = true
		}
	}
	answered := map[string]bool{}
	for _, finding := range base.Findings {
		answered[finding.Question] = true
	}
	for _, rule := range diagnosis.Rules() {
		if !answered[rule.Question] {
			continue
		}
		for _, need := range rule.Match {
			related[need.Probe] = true
		}
	}
	measured := map[string]bool{}
	for _, r := range baseRun {
		measured[r.Probe] = true
	}

	var candidate string
	var candidateKind probe.ProbeKind
	for _, entry := range probe.Registry() {
		if measured[entry.Name] || related[entry.Name] {
			continue
		}
		candidate, candidateKind = entry.Name, entry.Kind
		break
	}
	if candidate == "" {
		t.Fatalf("every registered probe is measured by the base run or named by a rule of an answered question, so the case needs a run whose conclusions leave a probe unrelated")
	}

	added := append(append([]probe.Result(nil), baseRun...),
		result(candidate, candidateKind, observationInState(t, candidate, diagnosis.StateUnresolved)))
	got := diagnosis.Diagnose(added)

	// Every conclusion of the base run is byte-identical, in the same relative order.
	baseQuestions := map[string]bool{}
	for _, finding := range base.Findings {
		baseQuestions[finding.Question] = true
	}
	var preserved []diagnosis.Finding
	for _, finding := range got.Findings {
		if baseQuestions[finding.Question] {
			preserved = append(preserved, finding)
		}
	}
	if !reflect.DeepEqual(preserved, base.Findings) {
		t.Errorf("the added unresolved probe changed an established conclusion:\n got %+v\nwant %+v", preserved, base.Findings)
	}

	// A question the base run left open either stays open with the same needed states, or is
	// answered — and then only by a clause that names the added probe, because no other new fact
	// exists. Nothing unrelated may change.
	for _, baseOpen := range base.OpenQuestions {
		open := openQuestionsFor(got, baseOpen.Question)
		if len(open) == 0 {
			continue
		}
		if !reflect.DeepEqual(open[0].NeededStates, baseOpen.NeededStates) {
			t.Errorf("the open question %q needs %v, want the base run's %v: the added probe changed a question it does not touch", baseOpen.Question, open[0].NeededStates, baseOpen.NeededStates)
		}
	}
	for _, finding := range got.Findings {
		if baseQuestions[finding.Question] {
			continue
		}
		usesCandidate := false
		for _, rule := range rulesWithID(finding.Rule) {
			for _, need := range rule.Match {
				if need.Probe == candidate {
					usesCandidate = true
				}
			}
		}
		if !usesCandidate {
			t.Errorf("the finding %q appeared although no clause of its id needs the added probe %q", finding.Rule, candidate)
		}
	}

	// The unresolved probe is never a silent omission: its derived fact conclusion is present, and
	// it names the probe, its reason code and its detail.
	passRules := rulesWithID(diagnosis.RuleID(candidate, diagnosis.StatePass))
	if len(passRules) != 1 {
		t.Fatalf("the derived rule of %q in the pass state has %d rows, want exactly one", candidate, len(passRules))
	}
	fact := findingFor(t, got, passRules[0].Question)
	if want := diagnosis.RuleID(candidate, diagnosis.StateUnresolved); fact.Rule != want {
		t.Errorf("the gap of the added probe fired %q, want %q", fact.Rule, want)
	}
	for _, want := range []string{candidate, string(probe.ReasonDNSUnresolved), "resolver timeout"} {
		if !strings.Contains(fact.Conclusion, want) {
			t.Errorf("the gap conclusion %q does not carry %q", fact.Conclusion, want)
		}
	}
}
