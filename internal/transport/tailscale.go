package transport

// This file is the tailscale adapter — PRD §5.2's detect-and-report path, which works when a
// third-party tunnel adapter is permitted and installed on the node. It performs no measurement:
// R1a's probe registry observes this machine's sshd, egress and TLS state, and whether an adapter
// is installed or permitted is detected by Verify (R5/R6), not measured here. Feasible therefore
// reports the unmeasured-transport answer the spec requires — not viable, a reason stating the
// relevant measurement is missing, and an unsatisfied third_party_permission row whose Detail names
// what the user must check — and never claims the transport is blocked.
//
// It reads no finding and names no target, because no measurement of this adapter exists to read.

import (
	"context"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// tailscale is the tailscale adapter. It carries no state, so it is registered as a value and two
// registries cannot observe each other through it.
type tailscale struct{}

// Name is PRD §5.2's stable identifier for this adapter.
func (tailscale) Name() string { return "tailscale" }

// tailscaleRequirement is the adapter's one prerequisite in both its static and its evaluated
// form: the adapter being permitted on this machine. The Detail names what the user must check,
// because no run in this slice observes it.
func tailscaleRequirement() Requirement {
	return Requirement{
		Kind:   KindThirdPartyPermission,
		Detail: "the Tailscale adapter being permitted on this machine: check whether your endpoint agent or security policy allows it, because no measurement in this slice observes an adapter's installation or policy",
	}
}

// Requires declares tailscale's one prerequisite: a third-party tunnel adapter permitted on this
// machine. It is a static declaration; satisfaction is decided per run in Feasible.
func (tailscale) Requires() []Requirement { return []Requirement{tailscaleRequirement()} }

// Feasible reports the unmeasured-transport answer. It deliberately reads nothing from the
// diagnosis — no probe in this slice observes whether a tunnel adapter is permitted or installed —
// so the parameter is ignored, and no target is ever named.
//
// The verdict is the evaluated prerequisite row rather than a hardcoded false: R1a supplies no
// policy fact, so the row is unsatisfied by construction, and a later slice that detects the
// adapter changes the row instead of this function.
func (tailscale) Feasible(_ diagnosis.Diagnosis) Feasibility {
	requirement := tailscaleRequirement()
	requirement.Satisfied = false
	return Feasibility{
		Viable: requirement.Satisfied,
		Reason: "no measurement of a third-party tunnel adapter was made in this slice: no probe observes whether Tailscale is installed or permitted, because that detection belongs to Verify (R5/R6); the transport is not reported viable and no rejection of it is claimed. Check whether your endpoint agent or security policy allows the adapter.",
		Notes: []string{
			"no measurement of this adapter exists in this slice: the probe registry observes sshd, egress and TLS state, and whether an adapter is installed or permitted is detected by Verify (R5/R6)",
			requirementNote(requirement),
		},
		Requires: []Requirement{requirement},
	}
}

// PlanHub, PlanNode and Verify fail loudly until their owning slices define the real plan and
// verification shapes (RG-9). Each error names the member and the owner; no value is returned
// beside it.
func (tailscale) PlanHub(PairingBundle) ([]Step, error) {
	return nil, notImplemented("PlanHub", "R3")
}

func (tailscale) PlanNode(PairingBundle) ([]Step, error) {
	return nil, notImplemented("PlanNode", "R3")
}

func (tailscale) Verify(context.Context, Handle) ([]probe.Result, error) {
	return nil, notImplemented("Verify", "R5/R6")
}
