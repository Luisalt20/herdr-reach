package report

// This file is the single mapping of design §3.3: the one function that turns a
// run's own values into the payload.
//
// It reads, derives, sorts and emits. It performs no I/O, measures nothing and
// touches no machine: every value it emits is either copied verbatim from the
// input or derived from values the run already produced, and no code path here
// can bring a measurement into being that did not happen.

import (
	"cmp"
	"slices"
	"time"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/transport"
	"github.com/Luisalt20/herdr-reach/internal/version"
)

// The names this mapping must know by name rather than by value.
//
// The measurement layer declares the hub probe's name and the local environment
// probe's name as unexported constants, so the mapping spells them here, exactly
// as the transport adapters spell the rule ids they switch on. The suite asserts
// each is still declared — both among probe.Registry()'s names — so a rename
// cannot silently strand the mapping on its absence path.
const (
	hubProbeName      = "egress.hub.direct"
	localEnvProbeName = "local.env"
)

// nodeAbsentIdentity is the classification identity the payload reports when the
// run never classified the machine, or when its classification target is not an
// identity the probe's own format recognises. It is built by the probe's own
// constructor, so a machine that was never classified reports the probe's own
// unknown vocabulary rather than a platform or architecture this mapping
// invented.
var nodeAbsentIdentity = probe.NodePlatformIdentity(probe.NodePlatformUnknown, "")

// Build maps one run to the payload. It is the only producer of the DTO: every
// field of Payload is filled here, so a structure added to the run's own values
// cannot reach the document without a deliberate edit to this function. It
// measures nothing and touches no machine: the values it emits are the run's
// own, copied or derived.
func Build(in Input) Payload {
	unresolved, notMeasured := coverage(in.Results)
	completeness := CompletenessComplete
	if len(unresolved) > 0 {
		completeness = CompletenessIncomplete
	}

	return Payload{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   generatedAt(in.Run),
		Tool:          ToolInfo{Name: version.Name, Version: in.ToolVersion},
		Run: RunInfo{
			Completeness: completeness,
			Unresolved:   unresolved,
			NotMeasured:  notMeasured,
			Concurrency:  in.Run.Concurrency,
			RunBudgetMS:  in.Run.RunBudget.Milliseconds(),
		},
		Targets:       targets(in.Declared),
		Probes:        probes(in.Results),
		Findings:      findings(in.Diagnosis.Findings),
		OpenQuestions: openQuestions(in.Diagnosis.OpenQuestions),
		Node:          node(in),
		Transports:    transports(in.Transports),
	}
}

// generatedAt stamps the document from the run's single injected clock, in
// RFC3339 UTC, so only that clock moves a timestamp (design §3.3).
func generatedAt(options probe.Options) string {
	if options.Clock == nil {
		// An effective Options always carries a clock; a nil one is a caller that
		// handed over something other than the bounds it ran under. The zero time
		// is reported rather than the machine's own clock, so the mapping can
		// never put an uninjectable timestamp into a document whose determinism
		// claim depends on the clock being the run's.
		return time.Time{}.UTC().Format(time.RFC3339)
	}
	return options.Clock.Now().UTC().Format(time.RFC3339)
}

// targets echoes the effective declared set and resolves the hub address from
// it: the hub entry is the effective target of egress.hub.direct, so the
// address the payload reports is the resolved "host:port" the run would dial,
// with the documented default port already applied, rather than the raw flag
// value. A run with no hub entry reports JSON null.
func targets(declared []probe.EffectiveTarget) TargetsInfo {
	info := TargetsInfo{Declared: make([]DeclaredTarget, 0, len(declared))}
	for _, target := range declared {
		info.Declared = append(info.Declared, DeclaredTarget{
			Probe:    target.Probe,
			Target:   target.Address(),
			Protocol: target.Protocol,
		})
		if target.Probe != hubProbeName {
			continue
		}
		address := target.Address()
		info.Hub = &address
	}
	return info
}

// probes builds one row per registered probe the run reported, in
// probe.Registry() order whatever order the results arrived in. A result whose
// probe the registry does not declare is not part of the run's contract and is
// ignored — here and in the coverage lists alike — because the registry is the
// run's closed probe set.
func probes(results []probe.Result) []ProbeRow {
	rows := make([]ProbeRow, 0, len(probe.Registry()))
	for _, registration := range probe.Registry() {
		result, reported := resultFor(results, registration.Name)
		if !reported {
			continue
		}
		rows = append(rows, probeRow(result))
	}
	return rows
}

// nullableTarget renders one target field the way design D3 requires: JSON null
// when the run carried no target, the verbatim string otherwise. A script must
// not have to tell an empty string from an absent target, because a measurement
// that was never made has no target at all while a measurement that was made
// names the address it dialed. The pointer is the whole representation: the key
// stays present (no `omitempty`), and nothing else in the document is nullable.
func nullableTarget(target string) *string {
	if target == "" {
		return nil
	}
	return &target
}

// probeRow copies one result into its row. The aggregate verdict and reason are
// carried verbatim from the result — the run's own answer, never recomputed
// here — and the row's resolution is the resolution of the observation the
// measurement layer's own reduction selected.
func probeRow(result probe.Result) ProbeRow {
	return ProbeRow{
		Name:         result.Probe,
		Kind:         result.Kind,
		Target:       nullableTarget(result.Target),
		Verdict:      result.Verdict,
		Resolution:   aggregateResolution(result),
		Reason:       result.Reason,
		Detail:       result.Detail,
		ElapsedMS:    result.Elapsed.Milliseconds(),
		Observations: observationRows(result.Observations),
	}
}

// observationRows copies the probe's observations in the order the probe
// reported them, each value verbatim.
func observationRows(observations []probe.Observation) []ObservationRow {
	rows := make([]ObservationRow, 0, len(observations))
	for _, observation := range observations {
		rows = append(rows, ObservationRow{
			Label:      observation.Label,
			Target:     nullableTarget(observation.Target),
			Verdict:    observation.Verdict,
			Resolution: observation.Resolution,
			Reason:     observation.Reason,
			Detail:     observation.Detail,
		})
	}
	return rows
}

// resultCarriesNoObservation reports whether a reported result measured nothing:
// the probe returned but carries no observation for the mapping to read. It is
// one predicate on purpose. aggregateResolution maps this case to the unresolved
// resolution probeRow publishes, and coverage classifies the same case as making
// the run incomplete, so the row and the coverage list cannot disagree about one
// result (R-HR-NF-02, R-HR-NF-03).
func resultCarriesNoObservation(result probe.Result) bool {
	return len(result.Observations) == 0
}

// aggregateResolution reports how far the probe's answer got: the resolution of
// the observation probe.Aggregate reduces the probe to. The ranking — fail above
// indeterminate above pass, the earlier observation winning a tie — lives in
// that function, and feeding it the running winner and the next observation
// finds exactly the observation it selected, so this mapping holds no second
// copy of the documented order. A result that reported no observation measured
// nothing, so its resolution is the absence of an answer, matching the
// indeterminate internal-error verdict Aggregate gives that same result — and
// that empty case is resultCarriesNoObservation, the same predicate coverage
// reads.
func aggregateResolution(result probe.Result) probe.Resolution {
	if resultCarriesNoObservation(result) {
		return probe.Unresolved
	}
	observations := result.Observations
	worst := observations[0]
	for _, candidate := range observations[1:] {
		if verdict, _ := probe.Aggregate([]probe.Observation{worst, candidate}); verdict != worst.Verdict {
			worst = candidate
		}
	}
	return worst.Resolution
}

// findings copies the diagnosis's findings in their own order, with the rule id,
// conclusion and dependency names verbatim. Evidence is deliberately not copied:
// the payload does not carry it.
func findings(list []diagnosis.Finding) []FindingRow {
	rows := make([]FindingRow, 0, len(list))
	for _, finding := range list {
		rows = append(rows, FindingRow{
			Question:   finding.Question,
			Rule:       finding.Rule,
			Conclusion: finding.Conclusion,
			DependsOn:  copyStrings(finding.DependsOn),
		})
	}
	return rows
}

// openQuestions copies the reasoning layer's open questions in their own order.
func openQuestions(list []diagnosis.OpenQuestion) []OpenQuestionRow {
	rows := make([]OpenQuestionRow, 0, len(list))
	for _, question := range list {
		rows = append(rows, OpenQuestionRow{
			Question:     question.Question,
			NeededStates: copyStrings(question.NeededStates),
		})
	}
	return rows
}

// node reads the run's classification. The platform and architecture come from
// the local.env observation's own target, split by the probe's own inverse so
// the two halves are the values the probe measured rather than a second parse of
// its format, and the note is the classification's own verbatim detail. A run
// that never classified the machine reports the unknown identity: no platform is
// invented.
//
// `NodeInfo.Refused` stays in the payload shape for compatibility, and it is
// always false: the reasoning layer no longer declares a refusal rule, because
// native Windows is a supported classification as of Herdr 0.9.1. Nothing here
// reads the platform string to set it — a platform name is a classification, not
// a refusal.
func node(in Input) NodeInfo {
	info := NodeInfo{}
	platform, arch, _ := probe.SplitNodePlatformIdentity(nodeAbsentIdentity)
	info.Platform, info.Arch = string(platform), arch

	observation, classified := classificationObservation(in.Results)
	if !classified {
		return info
	}
	info.Note = observation.Detail
	if platform, arch, ok := probe.SplitNodePlatformIdentity(observation.Target); ok {
		info.Platform, info.Arch = string(platform), arch
	}
	return info
}

// classificationObservation finds the observation local.env reported. The probe
// reports one observation, so its first is its classification; a result with
// none is not a classification this mapping may read.
func classificationObservation(results []probe.Result) (probe.Observation, bool) {
	for _, result := range results {
		if result.Probe != localEnvProbeName || len(result.Observations) == 0 {
			continue
		}
		return result.Observations[0], true
	}
	return probe.Observation{}, false
}

// transports copies the paired rows sorted by name. The sort is stable and runs
// on a copy, so equal names keep their arrival order, the caller's slice is
// untouched, and no map iteration can influence the output.
func transports(rows []TransportRow) []TransportInfo {
	sorted := make([]TransportRow, len(rows))
	copy(sorted, rows)
	slices.SortStableFunc(sorted, func(a, b TransportRow) int { return cmp.Compare(a.Name, b.Name) })

	infos := make([]TransportInfo, 0, len(sorted))
	for _, row := range sorted {
		infos = append(infos, TransportInfo{
			Name:     row.Name,
			Viable:   row.Feasibility.Viable,
			Reason:   row.Feasibility.Reason,
			Requires: requirementRows(row.Feasibility.Requires),
			Notes:    copyStrings(row.Feasibility.Notes),
		})
	}
	return infos
}

// requirementRows copies one transport's evaluated prerequisite rows in
// declaration order, each verbatim.
func requirementRows(requirements []transport.Requirement) []RequirementRow {
	rows := make([]RequirementRow, 0, len(requirements))
	for _, requirement := range requirements {
		rows = append(rows, RequirementRow{
			Kind:      requirement.Kind,
			Satisfied: requirement.Satisfied,
			Detail:    requirement.Detail,
		})
	}
	return rows
}

// coverage derives the run's completeness and its two coverage lists from the
// results' observations by resolution, never from their verdicts (design D5's
// reconciliation): an attempted observation that produced no answer makes the
// run incomplete and names its probe in run.unresolved, while an observation
// that was never attempted leaves the run complete and names its probe in
// run.not_measured. The lists follow probe.Registry() order and name each probe
// once, so the order cannot depend on the results' arrival order or on how many
// observations a probe reported.
//
// A reported result that carries no observation at all belongs with the
// unresolved, not with the not-measured: the probe returned and measured
// nothing, which is already the unresolved resolution aggregateResolution gives
// its row. The row and this list read that empty case through the same
// predicate, so a run cannot publish an unresolved probe row beside a complete
// run (R-HR-NF-02, R-HR-NF-03).
func coverage(results []probe.Result) (unresolved, notMeasured []string) {
	unresolved = make([]string, 0)
	notMeasured = make([]string, 0)
	for _, registration := range probe.Registry() {
		result, reported := resultFor(results, registration.Name)
		if !reported {
			continue
		}
		if resultCarriesNoObservation(result) || hasResolution(result.Observations, probe.Unresolved) {
			unresolved = append(unresolved, registration.Name)
		}
		if hasResolution(result.Observations, probe.NotMeasured) {
			notMeasured = append(notMeasured, registration.Name)
		}
	}
	return unresolved, notMeasured
}

// hasResolution reports whether any observation of the probe got that far.
func hasResolution(observations []probe.Observation, resolution probe.Resolution) bool {
	for _, observation := range observations {
		if observation.Resolution == resolution {
			return true
		}
	}
	return false
}

// resultFor finds the result the run reported for one registered probe, so the
// payload's probe rows and its coverage lists read one run's answers.
func resultFor(results []probe.Result, name string) (probe.Result, bool) {
	for _, result := range results {
		if result.Probe == name {
			return result, true
		}
	}
	return probe.Result{}, false
}

// copyStrings copies a string slice so an empty list serialises as [] rather
// than null, and so the payload cannot alias a slice its caller may still edit.
func copyStrings(values []string) []string {
	copied := make([]string, len(values))
	copy(copied, values)
	return copied
}
