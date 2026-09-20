package transport

// This file is the direct-ssh adapter — PRD §5.2's cheapest path, which works when the hub is
// reachable on an allowed port — and the two shared readings both SSH adapters need: locating the
// finding of a question, and reading the hub-directed measurement as an outcome, a requirement row
// and an explanation. Both adapters decide on that same measurement, so its reading lives once
// here instead of drifting between them.
//
// Feasible performs no measurement. Every target, port, label and reason code it names is taken
// from the finding's evidence, and a hub question the run never answered is reported as an
// absence, never as a block.

import (
	"context"
	"fmt"
	"net"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// The reasoning-layer vocabulary this file reads: the probe whose observation answers the hub
// question, and the question names the adapters locate findings by. A finding is read by question
// and rule id; its prose conclusion is never parsed.
const (
	probeEgressHub          = "egress.hub.direct"
	questionHubReachability = "hub.reachability"
	questionLocalSSHD       = "local.sshd"
)

// directSSH is the direct-ssh adapter. It carries no state, so it is registered as a value and two
// registries cannot observe each other through it.
type directSSH struct{}

// Name is PRD §5.2's stable identifier for this adapter.
func (directSSH) Name() string { return "direct-ssh" }

// Requires declares direct-ssh's one prerequisite: the hub address the run dials. It is a static
// declaration; satisfaction is decided per run in Feasible, where a measured block leaves the row
// satisfied while the transport is not viable.
func (directSSH) Requires() []Requirement {
	return []Requirement{{
		Kind:   KindHubAddress,
		Detail: "a hub address supplied with --hub host[:port], so the hub-directed measurement can decide this transport",
	}}
}

// Feasible decides direct-ssh from the hub-directed measurement alone. A pass is the "works when"
// condition of PRD §5.2's direct-ssh row; a failure rejects the transport with the measured target
// and port; an unanswered attempt and a missing measurement are absences and are worded as such.
func (directSSH) Feasible(d diagnosis.Diagnosis) Feasibility {
	finding, found := findingByQuestion(d, questionHubReachability)
	hub := assessHub(finding, found)
	return Feasibility{
		Viable:   hub.outcome == hubPassed,
		Reason:   hub.reason,
		Notes:    hub.notes,
		Requires: []Requirement{hub.requirement},
	}
}

// PlanHub, PlanNode and Verify fail loudly until their owning slices define the real plan and
// verification shapes (RG-9). Each error names the member and the owner; no value is returned
// beside it.
func (directSSH) PlanHub(PairingBundle) ([]Step, error) {
	return nil, notImplemented("PlanHub", "R3")
}

func (directSSH) PlanNode(PairingBundle) ([]Step, error) {
	return nil, notImplemented("PlanNode", "R3")
}

func (directSSH) Verify(context.Context, Handle) ([]probe.Result, error) {
	return nil, notImplemented("Verify", "R5/R6")
}

// findingByQuestion returns the finding of one question. The adapters read the diagnosis by
// question and rule id, which are the reasoning layer's public vocabulary; they never parse a
// conclusion, because a conclusion is prose for a human and could be reworded without warning.
func findingByQuestion(d diagnosis.Diagnosis, question string) (diagnosis.Finding, bool) {
	for _, finding := range d.Findings {
		if finding.Question == question {
			return finding, true
		}
	}
	return diagnosis.Finding{}, false
}

// hubOutcome is the four-way reading of the hub question that both SSH adapters decide on.
type hubOutcome int

const (
	// hubPassed is a measured connection: the declared address accepted one.
	hubPassed hubOutcome = iota
	// hubRejected is a measured failure: the declared address was dialed and did not accept.
	hubRejected
	// hubUnanswered is an attempt that produced no answer: an absence, never a block.
	hubUnanswered
	// hubUnmeasured is a measurement that was never made, either because the run supplied no hub
	// address or because no result for the hub probe exists in the diagnosis at all.
	hubUnmeasured
)

// hubAssessment is the hub question read the way both adapters need it: the outcome, the
// requirement row the outcome produces, the explanation for the outcome, and the notes that name
// the observations it rests on.
type hubAssessment struct {
	outcome     hubOutcome
	requirement Requirement
	reason      string
	notes       []string
}

// assessHub reads the hub question's finding — or the absence of one — into the shared assessment.
// The states are exactly the reasoning layer's four derived id states; any other rule is reported
// as an unreadable absence rather than folded into one of them, because a state the vocabulary
// does not define is not a measurement.
func assessHub(finding diagnosis.Finding, found bool) hubAssessment {
	if !found {
		return hubAssessment{
			outcome: hubUnmeasured,
			requirement: Requirement{
				Kind:      KindHubAddress,
				Satisfied: false,
				Detail:    "no hub measurement is present in this run; supply a hub address with --hub host[:port] and make sure the hub-directed measurement runs",
			},
			reason: "the hub measurement was not made: the diagnosis carries no " + questionHubReachability +
				" finding, so nothing is claimed about the hub; supply --hub host[:port] to measure it",
			notes: []string{"hub reachability was not measured: the diagnosis carries no " + questionHubReachability + " finding"},
		}
	}

	fact, hasFact := firstEvidence(finding)
	switch finding.Rule {
	case diagnosis.RuleID(probeEgressHub, diagnosis.StatePass):
		return hubAssessment{
			outcome: hubPassed,
			requirement: Requirement{
				Kind:      KindHubAddress,
				Satisfied: true,
				Detail:    "supplied and measured: " + hubSubject(fact, hasFact) + " accepted a TCP connection",
			},
			reason: hubMeasuredReason(finding, fact, hasFact, "accepted a connection") + ", so this transport can work on that address",
			notes:  evidenceNotes(finding),
		}
	case diagnosis.RuleID(probeEgressHub, diagnosis.StateFail):
		return hubAssessment{
			outcome: hubRejected,
			requirement: Requirement{
				Kind:      KindHubAddress,
				Satisfied: true,
				Detail:    "supplied and measured: " + hubSubject(fact, hasFact) + " did not accept a TCP connection",
			},
			reason: hubMeasuredReason(finding, fact, hasFact, "did not accept a connection") + ", so this transport is rejected by that measurement",
			notes:  evidenceNotes(finding),
		}
	case diagnosis.RuleID(probeEgressHub, diagnosis.StateUnresolved):
		return hubAssessment{
			outcome: hubUnanswered,
			requirement: Requirement{
				Kind:      KindHubAddress,
				Satisfied: true,
				Detail:    "supplied and attempted: " + hubSubject(fact, hasFact) + "; the attempt produced no answer",
			},
			reason: hubMeasuredReason(finding, fact, hasFact, "produced no answer") + ": the attempt produced no answer, so no rejection is claimed and this transport is not decided",
			notes:  evidenceNotes(finding),
		}
	case diagnosis.RuleID(probeEgressHub, diagnosis.StateNotMeasured):
		reason := "the hub measurement was not made: no hub observation exists for this run, so nothing is claimed about the hub"
		if hasFact {
			reason = fmt.Sprintf("the hub measurement was not made: the diagnosis reports %s (rule %s), so nothing is claimed about the hub",
				fact.Observation.Reason, finding.Rule)
		}
		if !hasFact || fact.Observation.Reason == probe.ReasonInputMissingHub {
			reason += "; supply --hub host[:port] to measure it"
		} else {
			reason += "; the hub measurement must be enabled before this transport can be decided"
		}
		return hubAssessment{
			outcome: hubUnmeasured,
			requirement: Requirement{
				Kind:      KindHubAddress,
				Satisfied: false,
				Detail:    "no hub address was measured: supply a hub address with --hub host[:port] so the hub-directed measurement can decide this transport",
			},
			reason: reason,
			notes:  evidenceNotes(finding),
		}
	default:
		return hubAssessment{
			outcome: hubUnmeasured,
			requirement: Requirement{
				Kind:      KindHubAddress,
				Satisfied: false,
				Detail:    "the hub question was answered by a rule this adapter does not recognise, so no hub measurement is claimed; supply a hub address with --hub host[:port] and re-measure",
			},
			reason: fmt.Sprintf("the hub measurement cannot be read: the diagnosis fired rule %s, which this adapter does not recognise as one of the hub measurement's states", finding.Rule),
			notes:  evidenceNotes(finding),
		}
	}
}

// hubMeasuredReason names the measured hub observation exactly: the target verbatim, its port when
// the address carries one, and the reason code the measurement layer attached. Building it from
// the finding's evidence is what makes "a rejection names the measured observation" structural
// instead of a promise — there is no other source for the target or the port. A finding that
// somehow carries no observation yields the rule id alone, because no measurement may be invented.
func hubMeasuredReason(finding diagnosis.Finding, fact diagnosis.Fact, hasFact bool, verb string) string {
	if !hasFact {
		return fmt.Sprintf("the hub measurement fired rule %s, but its finding carries no observation to name", finding.Rule)
	}
	where := fact.Observation.Target
	if _, port, err := net.SplitHostPort(fact.Observation.Target); err == nil && port != "" {
		where = fmt.Sprintf("%s (port %s)", where, port)
	}
	return fmt.Sprintf("hub target %s %s (reason %s, rule %s)", where, verb, fact.Observation.Reason, finding.Rule)
}

// hubSubject names the hub observation for a requirement row: the target verbatim when the
// observation carried one, otherwise the probe's own label.
func hubSubject(fact diagnosis.Fact, hasFact bool) string {
	if !hasFact {
		return "the hub measurement"
	}
	if fact.Observation.Target != "" {
		return fact.Observation.Target
	}
	return fact.Observation.Label
}

// firstEvidence returns the first observation a finding rests on. A fired rule has at least one
// satisfied need, so this normally succeeds; when it does not, the adapters report that no
// observation is available instead of naming a target.
func firstEvidence(finding diagnosis.Finding) (diagnosis.Fact, bool) {
	if len(finding.Evidence) == 0 {
		return diagnosis.Fact{}, false
	}
	return finding.Evidence[0], true
}

// evidenceNotes names every observation a finding rests on, one note per fact, in Match order. A
// note is only ever built from a fact, so it cannot quote an observation the diagnosis did not
// carry; it names the probe's own label, target and reason code, which is what a reader needs to
// trace the verdict back to the run.
func evidenceNotes(finding diagnosis.Finding) []string {
	notes := make([]string, 0, len(finding.Evidence))
	for _, fact := range finding.Evidence {
		observation := fact.Observation
		if observation.Target == "" {
			notes = append(notes, fmt.Sprintf("%s: %s; reason %s", fact.Probe, observation.Label, observation.Reason))
			continue
		}
		notes = append(notes, fmt.Sprintf("%s: %s at %s; reason %s", fact.Probe, observation.Label, observation.Target, observation.Reason))
	}
	return notes
}

// requirementNote states an unsatisfied requirement the way D6 requires: it names what the user
// must supply or fix, so the row cannot be read as "the network blocks this".
func requirementNote(requirement Requirement) string {
	return fmt.Sprintf("prerequisite %s is not satisfied: %s", requirement.Kind, requirement.Detail)
}
