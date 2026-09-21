// Package report owns the machine-readable projection of one doctor run: the
// `schema_version: "1"` payload of design §3.3 and the single mapping that
// builds it from the run's own values.
//
// The package is a projection and nothing else. Build measures nothing, opens
// nothing, dials nothing and executes nothing: every value the payload states was
// already produced by the run — the effective bounds it ran under, the declared
// targets it was going to dial, the results its probes reported, the diagnosis
// the reasoning layer derived and the feasibility rows the transport layer
// evaluated. The human projection and the JSON writer are later slices; this
// package is the payload contract's one machine-side home.
package report

import (
	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/transport"
)

// SchemaVersion is the payload's schema version. This slice freezes no schema
// file, writes none and promises no cross-machine compatibility (R-HR-NF-06); it
// carries the version field so a later additive contract can recognise the shape
// it is reading.
const SchemaVersion = "1"

// Completeness is how much of the run's declared measurement set was actually
// answered. It is computed over observations by resolution, never over verdicts:
// an attempted measurement that resolved unresolved makes the run incomplete,
// while a measurement that was never attempted does not (design D5's
// reconciliation, R-HR-NF-02).
type Completeness string

const (
	// CompletenessComplete is a run in which no probe was attempted and left
	// unresolved. Not-measured observations do not change it: a coverage gap is
	// reported in run.not_measured, not by calling the run incomplete.
	CompletenessComplete Completeness = "complete"
	// CompletenessIncomplete is a run in which at least one probe was attempted
	// and resolved unresolved. The unresolved probes are named in run.unresolved.
	CompletenessIncomplete Completeness = "incomplete"
)

// Payload is the whole machine-readable document of design §3.3: the top level
// of the `schema_version: "1"` contract. Every field is always present; an empty
// collection serialises as `[]` rather than `null`, and the only nullable fields
// in the document are the target fields of ProbeRow and ObservationRow, plus
// Targets.Hub. The ProbeRow and ObservationRow target fields are null when no
// subject was declared and the verbatim subject otherwise; resolution plays no
// part, which is why local.sshd's not-measured observations can carry their
// targets while egress.quic's internal-failure observation carries none.
type Payload struct {
	SchemaVersion string            `json:"schema_version"`
	GeneratedAt   string            `json:"generated_at"`
	Tool          ToolInfo          `json:"tool"`
	Run           RunInfo           `json:"run"`
	Targets       TargetsInfo       `json:"targets"`
	Probes        []ProbeRow        `json:"probes"`
	Findings      []FindingRow      `json:"findings"`
	OpenQuestions []OpenQuestionRow `json:"open_questions"`
	Node          NodeInfo          `json:"node"`
	Transports    []TransportInfo   `json:"transports"`
}

// ToolInfo is the payload's `tool` level: which build produced the document.
// Name is the version package's constant; Version is the build's own version
// identity, so a release binary and a dev binary are told apart by the document
// itself.
type ToolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// RunInfo is the payload's `run` level: the run's completeness and coverage, and
// the effective bounds it ran under. Concurrency and RunBudgetMS are the
// effective values after probe.Options' documented defaults, so the document
// reports what the run actually used rather than what a caller happened to
// leave unset.
type RunInfo struct {
	Completeness Completeness `json:"completeness"`
	// Unresolved names the probes with at least one attempted observation that
	// produced no answer, one entry per probe, in registry order.
	Unresolved []string `json:"unresolved"`
	// NotMeasured names the probes with at least one observation that was never
	// attempted, one entry per probe, in registry order. A name here is a
	// coverage gap, never a blocked target.
	NotMeasured []string `json:"not_measured"`
	Concurrency int      `json:"concurrency"`
	// RunBudgetMS is the run's ceiling in whole milliseconds.
	RunBudgetMS int64 `json:"run_budget_ms"`
}

// TargetsInfo is the payload's `targets` level: the hub the run measured and the
// effective declared set it resolved, so "the dialed set equals the declared set"
// is checkable from the document alone (R-HR-NF-10).
type TargetsInfo struct {
	// Hub is the resolved hub address in "host:port" form, or nil when no hub
	// was supplied. It is JSON null rather than an empty string so a script
	// cannot confuse an absent hub with a hostname; a run with no hub declares no
	// hub entry at all rather than an empty one.
	Hub *string `json:"hub"`
	// Declared is the effective declared set, in declaration order, with each
	// probe's resolved target.
	Declared []DeclaredTarget `json:"declared"`
}

// DeclaredTarget is one entry of targets.declared: the probe that declares the
// measurement, the resolved "host:port" it will dial, and the wire protocol.
// Target is deliberately not nullable: this set is the effective declared targets
// the run resolved, so a run with no hub declares no hub entry rather than an
// empty one.
type DeclaredTarget struct {
	Probe string `json:"probe"`
	// Target is the resolved "host:port" the probe will dial, never null.
	Target   string         `json:"target"`
	Protocol probe.Protocol `json:"protocol"`
}

// ProbeRow is one entry of the payload's `probes` array: one registered probe's
// outcome, with its aggregate verdict, resolution and reason carried verbatim
// beside the probe's detail and its observations. The rows follow
// probe.Registry() order.
type ProbeRow struct {
	Name string          `json:"name"`
	Kind probe.ProbeKind `json:"kind"`
	// Target names the subject the probe is about — a dialed address, a local
	// path, a unit list, a service name — verbatim, or JSON null when no subject
	// was declared; resolution plays no part. local.sshd's row carries its binary
	// path while the probe is not_measured, and its not-measured observations
	// carry their targets; egress.quic's internal-failure observation carries
	// none and is unresolved. The key is always present: a script must not have
	// to tell "no target declared" from an empty string, which is design D3's
	// reason for the nullable field.
	Target     *string          `json:"target"`
	Verdict    probe.Verdict    `json:"verdict"`
	Resolution probe.Resolution `json:"resolution"`
	Reason     probe.ReasonCode `json:"reason"`
	Detail     string           `json:"detail"`
	// ElapsedMS is the probe's own elapsed time in whole milliseconds; a
	// sub-millisecond remainder is truncated, never rounded up.
	ElapsedMS    int64            `json:"elapsed_ms"`
	Observations []ObservationRow `json:"observations"`
}

// ObservationRow is one entry of a probe row's `observations` array: one fact the
// probe reported, in the order the probe reported it. Every value is copied
// verbatim; nothing here is derived from a verdict.
type ObservationRow struct {
	Label string `json:"label"`
	// Target names the subject the observation is about — a dialed address, a
	// local path, a unit list, a service name — verbatim, or JSON null when no
	// subject was declared; resolution plays no part. local.sshd's not-measured
	// service and config observations carry their targets, while egress.quic's
	// internal-failure observation carries none and is unresolved. The key is
	// always present: a script must not have to tell "no target declared" from
	// an empty string, which is design D3's reason for the nullable field.
	Target     *string          `json:"target"`
	Verdict    probe.Verdict    `json:"verdict"`
	Resolution probe.Resolution `json:"resolution"`
	Reason     probe.ReasonCode `json:"reason"`
	Detail     string           `json:"detail"`
}

// FindingRow is one entry of the payload's `findings` array: a question the rule
// table answered, with the rule id quoted, the conclusion text, and the probes
// the conclusion rests on (design §5.3).
type FindingRow struct {
	Question   string   `json:"question"`
	Rule       string   `json:"rule"`
	Conclusion string   `json:"conclusion"`
	DependsOn  []string `json:"depends_on"`
}

// OpenQuestionRow is one entry of the payload's `open_questions` array: a
// question no rule matched, with the observable states the table needed.
type OpenQuestionRow struct {
	Question     string   `json:"question"`
	NeededStates []string `json:"needed_states"`
}

// NodeInfo is the payload's `node` level: the classification this machine was
// placed in, its architecture, the refusal field the payload keeps for
// compatibility, and the classification's own verbatim detail. A run that never
// classified the machine reports the unknown platform rather than inventing one.
//
// Refused is always false: native Windows is a supported classification as of
// Herdr 0.9.1, so no platform is refused and no finding can set the field. It
// stays in the payload shape so a consumer written against `schema_version` "1"
// still finds the key.
type NodeInfo struct {
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
	Refused  bool   `json:"refused"`
	Note     string `json:"note"`
}

// TransportInfo is one entry of the payload's `transports` array: one transport's
// evaluated feasibility, sorted by name. Viable is the strict decision; Requires
// carries the evaluated prerequisite rows and Notes the measured observations the
// verdict rests on.
type TransportInfo struct {
	Name     string           `json:"name"`
	Viable   bool             `json:"viable"`
	Reason   string           `json:"reason"`
	Requires []RequirementRow `json:"requires"`
	Notes    []string         `json:"notes"`
}

// RequirementRow is one entry of a transport row's `requires` array: one
// prerequisite, its satisfaction state, and what remains for the user to supply
// when it is unsatisfied.
type RequirementRow struct {
	Kind      transport.RequirementKind `json:"kind"`
	Satisfied bool                      `json:"satisfied"`
	Detail    string                    `json:"detail"`
}

// Input is everything Build consumes. It is the run's own values, already
// collected: the mapping resolves nothing, measures nothing and invents nothing,
// so a field that is not here cannot appear in the payload, and a field here is
// the run's fact rather than a second construction of it.
type Input struct {
	// Run is the run's effective probe.Options — probe.NewRunner's Options, whose
	// documented defaults are already applied. Its Concurrency and RunBudget
	// become run.concurrency and run.run_budget_ms, and its Clock is the single
	// injected clock that stamps generated_at.
	Run probe.Options
	// ToolVersion is the build's version identity for tool.version. The tool name
	// is the version package's constant; only the version varies per build.
	ToolVersion string
	// Declared is the effective declared target set, resolved by
	// probe.EffectiveTargets. It is echoed in targets.declared, and its hub entry
	// — the one whose probe is egress.hub.direct — supplies targets.hub.
	Declared []probe.EffectiveTarget
	// Results is what the run's probes reported. Rows appear in probe.Registry()
	// order whatever order the results arrived in; a result the registry does not
	// declare is not part of the run's contract and is ignored.
	Results []probe.Result
	// Diagnosis is the reasoning layer's output for the same run: findings and
	// open questions are echoed in their own order. It no longer sets
	// node.refused: native Windows is a supported classification as of Herdr
	// 0.9.1, so no finding carries a refusal.
	Diagnosis diagnosis.Diagnosis
	// Transports pairs every evaluated feasibility with the name of the transport
	// it belongs to, so a name cannot silently misalign with its viability.
	Transports []TransportRow
}

// TransportRow pairs one transport's stable name with its evaluated
// feasibility. transport.Feasibility intentionally does not carry the name —
// the adapter knows it and the caller has it — so the pairing happens here,
// where the name travels beside the verdict instead of being matched to it
// afterwards by position.
type TransportRow struct {
	Name        string
	Feasibility transport.Feasibility
}
