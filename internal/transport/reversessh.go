package transport

// This file is the reverse-ssh adapter — PRD §5.2's zero-third-party path, which works when the
// node can dial the hub and the hub can accept and forward. Feasible decides the dial half from
// the hub-directed measurement and the forward half from the local sshd's configuration in force,
// and performs no measurement of its own.
//
// Reason precedence follows design §3.2 when several facts could explain the same verdict: a
// measured blocking observation first — the hub rejection, then a measured sshd rejection — then
// an unresolved dependency of the decision (the unanswered hub attempt), then an unmet requirement
// (the missing hub measurement or the excluded sshd capability). A measured rejection therefore
// always outranks an absence, and an absence is never worded as a block.

import (
	"context"
	"fmt"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// The hand-named rule ids of the `local.sshd` question group (design §5.2). The reasoning layer
// declares them as unexported constants, so this adapter spells them; the contract suite asserts
// they are still among diagnosis.AllRuleIDs(), which is what stops a table rename from silently
// stranding this adapter on its unrecognised-rule path.
const (
	ruleSSHDPresentConfigured          = "SSHD_PRESENT_CONFIGURED"
	ruleSSHDPresentConfigDivergent     = "SSHD_PRESENT_CONFIG_DIVERGENT"
	ruleSSHDAbsent                     = "SSHD_ABSENT"
	ruleSSHDEffectiveConfigNotMeasured = "SSHD_EFFECTIVE_CONFIG_NOT_MEASURED"
)

// reverseSSH is the reverse-ssh adapter. It carries no state, so it is registered as a value.
type reverseSSH struct{}

// Name is PRD §5.2's stable identifier for this adapter.
func (reverseSSH) Name() string { return "reverse-ssh" }

// Requires declares reverse-ssh's two prerequisites in declaration order: the hub address the node
// dials, and an sshd whose configuration in force is the written one. Satisfaction is decided per
// run in Feasible; a measured hub block leaves the hub row satisfied while the transport is not
// viable.
func (reverseSSH) Requires() []Requirement {
	return []Requirement{
		{
			Kind:   KindHubAddress,
			Detail: "a hub address supplied with --hub host[:port], so the node-directed dial can be measured",
		},
		{
			Kind:   KindSSHDEffectiveConfig,
			Detail: "an sshd whose configuration in force is the written one, so the hub has something to reach",
		},
	}
}

// Feasible decides reverse-ssh. It is viable only when the hub measurement passed and the sshd
// requirement is satisfied; every other combination is explained by the highest-precedence fact
// that supports it, and both evaluated requirement rows travel with the verdict.
func (reverseSSH) Feasible(d diagnosis.Diagnosis) Feasibility {
	hubFinding, hubFound := findingByQuestion(d, questionHubReachability)
	sshdFinding, sshdFound := findingByQuestion(d, questionLocalSSHD)
	hub := assessHub(hubFinding, hubFound)
	sshd := assessSSHD(sshdFinding, sshdFound)

	viable := hub.outcome == hubPassed && sshd.outcome == sshdConfigured
	reason := ""
	switch {
	case hub.outcome == hubRejected:
		// A measured block of the dial leg: it rejects the transport regardless of the sshd state.
		reason = hub.reason
	case sshd.outcome == sshdDivergent || sshd.outcome == sshdAbsent:
		// A measured rejection of the forward leg: it is a definite answer too, and it outranks an
		// unanswered hub attempt.
		reason = sshd.reason
	case hub.outcome == hubUnanswered:
		reason = hub.reason
	case viable:
		reason = "the hub measurement passed and the configuration in force is the written one, so the node can dial the hub and forward"
	default:
		// An unmet requirement. The first unsatisfied row in declaration order is named, so the
		// reason is the prerequisite a reader must act on rather than a network claim.
		if !hub.requirement.Satisfied {
			reason = hub.reason
		} else {
			reason = sshd.reason
		}
	}

	notes := append(append([]string(nil), hub.notes...), sshd.notes...)
	for _, requirement := range []Requirement{hub.requirement, sshd.requirement} {
		if !requirement.Satisfied {
			notes = append(notes, requirementNote(requirement))
		}
	}
	return Feasibility{
		Viable:   viable,
		Reason:   reason,
		Notes:    notes,
		Requires: []Requirement{hub.requirement, sshd.requirement},
	}
}

// PlanHub, PlanNode and Verify fail loudly until their owning slices define the real plan and
// verification shapes (RG-9). Each error names the member and the owner; no value is returned
// beside it.
func (reverseSSH) PlanHub(PairingBundle) ([]Step, error) {
	return nil, notImplemented("PlanHub", "R3")
}

func (reverseSSH) PlanNode(PairingBundle) ([]Step, error) {
	return nil, notImplemented("PlanNode", "R3")
}

func (reverseSSH) Verify(context.Context, Handle) ([]probe.Result, error) {
	return nil, notImplemented("Verify", "R5/R6")
}

// sshdOutcome is the reading of the `local.sshd` question group that reverse-ssh decides on.
type sshdOutcome int

const (
	// sshdConfigured is the positive conclusion: the binary is present and the configuration in
	// force agrees with the written one.
	sshdConfigured sshdOutcome = iota
	// sshdDivergent is a measured rejection: the configuration in force is not the written one and
	// is never presented as configured (R-HR-18).
	sshdDivergent
	// sshdAbsent is a measured rejection: no sshd is served to be reached.
	sshdAbsent
	// sshdUnmeasured is an absence: the effective configuration was not measured, either because
	// the run excluded the capability or because no result for the probe exists in the diagnosis.
	sshdUnmeasured
)

// sshdAssessment is the `local.sshd` question read the way reverse-ssh needs it: the outcome, the
// requirement row, the explanation and the notes naming the observations.
type sshdAssessment struct {
	outcome     sshdOutcome
	requirement Requirement
	reason      string
	notes       []string
}

// assessSSHD reads the `local.sshd` finding — or the absence of one — into the shared assessment.
// The four states are exactly the group's declared rules; a missing finding is reported as an
// unmeasured capability, never as an absent sshd, because a probe the run did not report measured
// nothing.
func assessSSHD(finding diagnosis.Finding, found bool) sshdAssessment {
	if !found {
		return sshdAssessment{
			outcome: sshdUnmeasured,
			requirement: Requirement{
				Kind:      KindSSHDEffectiveConfig,
				Satisfied: false,
				Detail:    "the effective sshd configuration was not measured; an sshd whose configuration in force can be compared with the written one is required",
			},
			reason: "the sshd requirement is unmet: the diagnosis carries no " + questionLocalSSHD + " finding, so the effective configuration was never measured",
			notes:  []string{"local.sshd was not measured: the diagnosis carries no " + questionLocalSSHD + " finding"},
		}
	}

	fact, hasFact := firstEvidence(finding)
	reasonCode := ""
	target := ""
	label := ""
	if hasFact {
		reasonCode = string(fact.Observation.Reason)
		target = fact.Observation.Target
		label = fact.Observation.Label
	}
	switch finding.Rule {
	case ruleSSHDPresentConfigured:
		return sshdAssessment{
			outcome: sshdConfigured,
			requirement: Requirement{
				Kind:      KindSSHDEffectiveConfig,
				Satisfied: true,
				Detail:    "measured: the sshd binary is present and the configuration in force agrees with the written one",
			},
			reason: "the sshd configuration in force is the written one",
			notes:  evidenceNotes(finding),
		}
	case ruleSSHDPresentConfigDivergent:
		return sshdAssessment{
			outcome: sshdDivergent,
			requirement: Requirement{
				Kind:      KindSSHDEffectiveConfig,
				Satisfied: false,
				Detail:    "the configuration in force is not the written one (reason " + reasonCode + "); correct the written configuration and measure again before this transport can be considered",
			},
			reason: "the sshd configuration in force is not the written configuration (reason " + reasonCode + ", rule " + finding.Rule + "): this transport is rejected, and no configuration was changed",
			notes:  evidenceNotes(finding),
		}
	case ruleSSHDAbsent:
		where := target
		if where == "" {
			where = label
		}
		return sshdAssessment{
			outcome: sshdAbsent,
			requirement: Requirement{
				Kind:      KindSSHDEffectiveConfig,
				Satisfied: false,
				Detail:    "no sshd is served to be reached at " + where + ", so the hub has nothing to forward to",
			},
			reason: "no sshd is served to be reached: the " + label + " observation at " + where + " reports reason " + reasonCode + " (rule " + finding.Rule + ")",
			notes:  evidenceNotes(finding),
		}
	case ruleSSHDEffectiveConfigNotMeasured:
		return sshdAssessment{
			outcome: sshdUnmeasured,
			requirement: Requirement{
				Kind:      KindSSHDEffectiveConfig,
				Satisfied: false,
				Detail:    "the effective sshd configuration was not measured (reason " + reasonCode + "); the excluded capability must be supplied before this transport can be considered",
			},
			reason: "the effective sshd configuration was not measured (reason " + reasonCode + ", rule " + finding.Rule + "), so nothing is claimed about the configuration in force; this transport is not decided",
			notes:  evidenceNotes(finding),
		}
	default:
		return sshdAssessment{
			outcome: sshdUnmeasured,
			requirement: Requirement{
				Kind:      KindSSHDEffectiveConfig,
				Satisfied: false,
				Detail:    "the local.sshd question was answered by a rule this adapter does not recognise, so no sshd measurement is claimed",
			},
			reason: fmt.Sprintf("the sshd measurement cannot be read: the diagnosis fired rule %s, which this adapter does not recognise as a local.sshd state", finding.Rule),
			notes:  evidenceNotes(finding),
		}
	}
}
