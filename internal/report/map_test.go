package report_test

// This file pins the payload contract of design §3.3 and the single mapping that
// produces it: the exact key set at every level, the completeness and coverage
// derivation, determinism, the millisecond unit, the NF-03 no-promotion
// property, and the proof that the mapping is the payload's only constructor.
//
// The payload is asserted as the JSON document it is: every case marshals the
// value Build returns, decodes it generically, and makes claims about the
// document's own shape and types, so a Go type change that changes the wire
// shape fails here instead of in a later consumer.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/report"
	"github.com/Luisalt20/herdr-reach/internal/transport"
	"github.com/Luisalt20/herdr-reach/internal/version"
)

// testTime is the fixed instant the injected clock reports, so generated_at is
// the same string in every build of a case.
var testTime = time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

// fixedClock is the run's one injected clock: a value, so two builds over the
// same Input cannot observe two different times.
type fixedClock struct{ at time.Time }

// Now returns the fixed instant.
func (c fixedClock) Now() time.Time { return c.at }

// testOptions is an effective probe.Options: the documented defaults already
// resolved by probe.NewRunner, plus the single injected clock. The mapping
// applies no default of its own, so the payload's run values are these.
func testOptions() probe.Options {
	return probe.Options{
		Concurrency:  probe.DefaultConcurrency,
		ProbeTimeout: probe.DefaultProbeTimeout,
		RunBudget:    probe.DefaultRunBudget,
		Clock:        fixedClock{at: testTime},
	}
}

// --- observation and result builders -----------------------------------------

// passObservation is one measured positive fact.
func passObservation(label, target, detail string) probe.Observation {
	return probe.Observation{
		Label: label, Target: target,
		Resolution: probe.Measured, Verdict: probe.Pass, Reason: probe.ReasonOK, Detail: detail,
	}
}

// failObservation is one measured negative fact.
func failObservation(label, target string, reason probe.ReasonCode, detail string) probe.Observation {
	return probe.Observation{
		Label: label, Target: target,
		Resolution: probe.Measured, Verdict: probe.Fail, Reason: reason, Detail: detail,
	}
}

// unresolvedObservation is one attempted fact that produced no answer. Its
// verdict is indeterminate — the absence of an answer, never a pass.
func unresolvedObservation(label, target string, reason probe.ReasonCode, detail string) probe.Observation {
	return probe.Observation{
		Label: label, Target: target,
		Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: reason, Detail: detail,
	}
}

// notMeasuredObservation is one fact that was never attempted: no target was
// reached, so the observation carries none.
func notMeasuredObservation(label string, reason probe.ReasonCode, detail string) probe.Observation {
	return probe.Observation{
		Label:      label,
		Resolution: probe.NotMeasured, Verdict: probe.Indeterminate, Reason: reason, Detail: detail,
	}
}

// resultOf builds a probe result the way the probes themselves do: through the
// measurement layer's own Aggregate, so the aggregate verdict and reason of the
// fixture are the values the production path would have reported.
func resultOf(name string, kind probe.ProbeKind, target string, elapsed time.Duration, detail string, observations ...probe.Observation) probe.Result {
	verdict, reason := probe.Aggregate(observations)
	return probe.Result{
		Probe: name, Kind: kind, Target: target,
		Verdict: verdict, Reason: reason, Detail: detail,
		Elapsed: elapsed, Observations: observations,
	}
}

// --- fixtures ----------------------------------------------------------------

// fullDeclared is an effective declared set in declaration order, with the hub
// entry a supplied hub resolves to. It is exactly what probe.EffectiveTargets
// returns for a run with --hub 203.0.113.10.
func fullDeclared() []probe.EffectiveTarget {
	return []probe.EffectiveTarget{
		{Probe: "egress.hub.direct", Protocol: probe.ProtocolTCP, Host: "203.0.113.10", Port: 22},
		{Probe: "egress.ssh.known", Protocol: probe.ProtocolTCP, Host: "github.com", Port: 22},
		{Probe: "egress.cf.443", Protocol: probe.ProtocolTCP, Host: "region1.v2.argotunnel.com", Port: 443, Label: "region1"},
		{Probe: "egress.cf.443", Protocol: probe.ProtocolTCP, Host: "region2.v2.argotunnel.com", Port: 443, Label: "region2"},
	}
}

// declaredWithoutHub is the effective declared set of a run with no --hub: the hub entry is absent
// because the run never resolved one, while every other declared target is still resolved. It is
// the case the payload must report as targets.hub null and no hub row in targets.declared.
func declaredWithoutHub() []probe.EffectiveTarget {
	var declared []probe.EffectiveTarget
	for _, target := range fullDeclared() {
		if target.Probe == "egress.hub.direct" {
			continue
		}
		declared = append(declared, target)
	}
	return declared
}

// fullResults is a complete run's results: a healthy Linux node, an sshd whose
// binary is absent while its effective configuration was never attempted, a hub
// that was not measured because no hub was supplied, and one public SSH
// measurement. Nothing is unresolved, so the run is complete with two named
// coverage gaps.
func fullResults() []probe.Result {
	return []probe.Result{
		resultOf("local.env", probe.ProbeLocal, "linux/aarch64", 2*time.Millisecond,
			"linux node (architecture \"aarch64\"): systemd is the running service manager; this run detects and reports the environment and changes nothing",
			passObservation("platform", "linux/aarch64",
				"linux node (architecture \"aarch64\"): systemd is the running service manager; this run detects and reports the environment and changes nothing")),
		resultOf("local.sshd", probe.ProbeLocal, "", 5*time.Millisecond,
			"the sshd binary is absent from the documented path",
			failObservation("binary present", "/usr/sbin/sshd", probe.ReasonSSHDAbsent,
				"no sshd binary is present at the documented path: installing sshd is work owned by a later slice, and nothing was changed"),
			notMeasuredObservation("effective config", probe.ReasonCapabilityExcluded,
				"sshd -T was not attempted: this run has no command runner wired, and the capability is excluded by this slice's boundary")),
		resultOf("egress.hub.direct", probe.ProbeEgress, "", time.Millisecond,
			"no hub address was supplied for this run, so the hub measurement was not made",
			notMeasuredObservation("hub target", probe.ReasonInputMissingHub,
				"no hub address was supplied for this run, so the hub measurement was not made; supply --hub host[:port] to measure it")),
		resultOf("egress.ssh.known", probe.ProbeEgress, "github.com:22", 3*time.Millisecond,
			"dial tcp github.com:22: the connection was established and closed",
			passObservation("tcp 22", "github.com:22",
				"dial tcp github.com:22: the connection was established and closed without reading or writing")),
	}
}

// fullDiagnosis is the reasoning layer's output for fullResults: one finding for
// the hub's missing input, one open question for the destination group. The
// evidence is deliberately present — the payload does not carry it, and a case
// proves that mutating it changes nothing.
func fullDiagnosis() diagnosis.Diagnosis {
	hubObservation := notMeasuredObservation("hub target", probe.ReasonInputMissingHub,
		"no hub address was supplied for this run, so the hub measurement was not made; supply --hub host[:port] to measure it")
	return diagnosis.Diagnosis{
		Findings: []diagnosis.Finding{
			{
				Question:   "hub.reachability",
				Rule:       "EGRESS_HUB_DIRECT_NOT_MEASURED",
				Conclusion: "egress.hub.direct: not measured (input_missing_hub) — no hub address was supplied for this run",
				DependsOn:  []string{"egress.hub.direct"},
				Evidence: []diagnosis.Fact{{
					Probe: "egress.hub.direct", Resolution: probe.NotMeasured, Verdict: probe.Indeterminate,
					State: diagnosis.StateNotMeasured, Observation: hubObservation,
				}},
			},
		},
		OpenQuestions: []diagnosis.OpenQuestion{
			{Question: "ssh.destination", NeededStates: []string{"egress.ssh.known PASS", "egress.hub.direct NOT_MEASURED"}},
		},
	}
}

// fullTransports is the transport layer's evaluation for the same run, paired
// with names and delivered in an order the payload must sort.
func fullTransports() []report.TransportRow {
	return []report.TransportRow{
		{Name: "reverse-ssh", Feasibility: transport.Feasibility{
			Viable: false,
			Reason: "the hub measurement was not made: no hub observation exists for this run, so nothing is claimed about the hub",
			Notes:  []string{"hub reachability was not measured"},
			Requires: []transport.Requirement{
				{Kind: transport.KindHubAddress, Satisfied: false, Detail: "supply --hub host[:port] so the hub-directed dial can be measured"},
				{Kind: transport.KindSSHDEffectiveConfig, Satisfied: false, Detail: "an sshd whose configuration in force is the written one"},
			},
		}},
		{Name: "direct-ssh", Feasibility: transport.Feasibility{
			Viable: false,
			Reason: "the hub measurement was not made: no hub observation exists for this run, so nothing is claimed about the hub",
			Notes:  []string{"hub reachability was not measured"},
			Requires: []transport.Requirement{
				{Kind: transport.KindHubAddress, Satisfied: false, Detail: "supply --hub host[:port] so the hub-directed dial can be measured"},
			},
		}},
	}
}

// fullInput is one complete input: an effective run, a version, a resolved
// declared set, the run's results and diagnosis, and the paired feasibility rows.
func fullInput() report.Input {
	return report.Input{
		Run:         testOptions(),
		ToolVersion: "9.9.9-test",
		Declared:    fullDeclared(),
		Results:     fullResults(),
		Diagnosis:   fullDiagnosis(),
		Transports:  fullTransports(),
	}
}

// --- document helpers --------------------------------------------------------

// marshalPayload serialises the payload, failing the case when it cannot.
func marshalPayload(t *testing.T, payload report.Payload) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("the payload did not marshal: %v", err)
	}
	return data
}

// decodePayload serialises the payload and decodes it generically, so assertions
// are about the JSON document's own shape and types rather than about Go values.
func decodePayload(t *testing.T, payload report.Payload) map[string]any {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(marshalPayload(t, payload), &document); err != nil {
		t.Fatalf("the payload did not decode: %v", err)
	}
	return document
}

// object asserts the value at path is a JSON object and returns it.
func object(t *testing.T, path string, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s is %T, want a JSON object", path, value)
	}
	return object
}

// array asserts the value at path is a JSON array and returns it. An empty
// non-nil array is accepted; a JSON null decodes to a nil slice and fails here.
func array(t *testing.T, path string, value any) []any {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("%s is %T, want a JSON array", path, value)
	}
	return items
}

// keySet asserts the object's key set is exactly want: an added, renamed,
// removed or omitted field fails, which is what makes a field a deliberate act.
func keySet(t *testing.T, path string, object map[string]any, want ...string) {
	t.Helper()
	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s has keys %v, want exactly %v", path, got, want)
	}
}

// stringAt asserts the value at path is a JSON string and returns it.
func stringAt(t *testing.T, path string, value any) string {
	t.Helper()
	text, ok := value.(string)
	if !ok {
		t.Fatalf("%s is %T, want a JSON string", path, value)
	}
	return text
}

// targetAt asserts the JSON a target field owes: JSON null when the run carried no target, the
// verbatim string otherwise. The JSON value is asserted, not the Go zero value, because a script
// must be able to tell "no target" from an empty one (design D3).
func targetAt(t *testing.T, path string, value any, want string) {
	t.Helper()
	if want == "" {
		if value != nil {
			t.Errorf("%s = %#v, want JSON null: an empty target is a null field, never an empty string", path, value)
		}
		return
	}
	if got := stringAt(t, path, value); got != want {
		t.Errorf("%s = %q, want the verbatim target %q", path, got, want)
	}
}

// numberAt asserts the value at path is a JSON number and returns it.
func numberAt(t *testing.T, path string, value any) float64 {
	t.Helper()
	number, ok := value.(float64)
	if !ok {
		t.Fatalf("%s is %T, want a JSON number", path, value)
	}
	return number
}

// rowFor returns the decoded probe row named name.
func rowFor(t *testing.T, document map[string]any, name string) map[string]any {
	t.Helper()
	for _, item := range array(t, "probes", document["probes"]) {
		row := object(t, "probes[]", item)
		if stringAt(t, "probes[].name", row["name"]) == name {
			return row
		}
	}
	t.Fatalf("the payload carries no probe row %q", name)
	return nil
}

// resultNamed returns the reported result of one probe, failing the case when the fixture does not
// carry it.
func resultNamed(t *testing.T, results []probe.Result, name string) probe.Result {
	t.Helper()
	for _, result := range results {
		if result.Probe == name {
			return result
		}
	}
	t.Fatalf("the fixture carries no result for %q", name)
	return probe.Result{}
}

// booleanPaths returns the path of every JSON boolean in a decoded value, so a
// case can assert exactly which booleans the document is allowed to carry.
func booleanPaths(value any, path string) []string {
	var paths []string
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			paths = append(paths, booleanPaths(child, path+"."+key)...)
		}
	case []any:
		for i, child := range v {
			paths = append(paths, booleanPaths(child, fmt.Sprintf("%s[%d]", path, i))...)
		}
	case bool:
		paths = append(paths, path)
	}
	return paths
}

// nullPaths returns the path of every JSON null in a decoded value, so a case can assert exactly
// which fields the document is allowed to carry as null.
func nullPaths(value any, path string) []string {
	var paths []string
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			if child == nil {
				paths = append(paths, childPath)
				continue
			}
			paths = append(paths, nullPaths(child, childPath)...)
		}
	case []any:
		for i, child := range v {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if child == nil {
				paths = append(paths, childPath)
				continue
			}
			paths = append(paths, nullPaths(child, childPath)...)
		}
	}
	return paths
}

// stringValues returns every JSON string in a decoded value.
func stringValues(value any) []string {
	var values []string
	switch v := value.(type) {
	case map[string]any:
		for _, child := range v {
			values = append(values, stringValues(child)...)
		}
	case []any:
		for _, child := range v {
			values = append(values, stringValues(child)...)
		}
	case string:
		values = append(values, v)
	}
	return values
}

// --- the key set and the types -----------------------------------------------

// TestPayloadKeySetAtEveryLevelWithFixedClock pins the document's exact key set
// at every level and the JSON type of every field, including that the hub is
// JSON null when no hub was supplied and that numbers are numbers.
func TestPayloadKeySetAtEveryLevelWithFixedClock(t *testing.T) {
	input := fullInput()
	payload := report.Build(input)
	document := decodePayload(t, payload)

	keySet(t, "payload", document,
		"schema_version", "generated_at", "tool", "run", "targets",
		"probes", "findings", "open_questions", "node", "transports")

	if got := document["schema_version"]; got != "1" {
		t.Errorf("schema_version = %#v, want the string \"1\"", got)
	}
	if report.SchemaVersion != "1" {
		t.Errorf("report.SchemaVersion = %q, want \"1\"", report.SchemaVersion)
	}
	if got := stringAt(t, "generated_at", document["generated_at"]); got != "2026-09-14T12:00:00Z" {
		t.Errorf("generated_at = %q, want the fixed clock's RFC3339 UTC stamp 2026-09-14T12:00:00Z", got)
	}

	tool := object(t, "tool", document["tool"])
	keySet(t, "tool", tool, "name", "version")
	if got := stringAt(t, "tool.name", tool["name"]); got != version.Name {
		t.Errorf("tool.name = %q, want %q", got, version.Name)
	}
	if got := stringAt(t, "tool.version", tool["version"]); got != "9.9.9-test" {
		t.Errorf("tool.version = %q, want the injected build identity", got)
	}

	run := object(t, "run", document["run"])
	keySet(t, "run", run, "completeness", "unresolved", "not_measured", "concurrency", "run_budget_ms")
	if got := stringAt(t, "run.completeness", run["completeness"]); got != "complete" {
		t.Errorf("run.completeness = %q, want \"complete\": no observation in the fixture is unresolved", got)
	}
	if got := numberAt(t, "run.concurrency", run["concurrency"]); got != float64(probe.DefaultConcurrency) {
		t.Errorf("run.concurrency = %v, want the effective Options value %d", got, probe.DefaultConcurrency)
	}
	if got := numberAt(t, "run.run_budget_ms", run["run_budget_ms"]); got != float64(probe.DefaultRunBudget.Milliseconds()) {
		t.Errorf("run.run_budget_ms = %v, want %d", got, probe.DefaultRunBudget.Milliseconds())
	}

	targets := object(t, "targets", document["targets"])
	keySet(t, "targets", targets, "hub", "declared")
	if got := stringAt(t, "targets.hub", targets["hub"]); got != "203.0.113.10:22" {
		t.Errorf("targets.hub = %q, want the resolved hub address", got)
	}
	declared := array(t, "targets.declared", targets["declared"])
	if len(declared) != len(input.Declared) {
		t.Fatalf("targets.declared has %d entries, want %d", len(declared), len(input.Declared))
	}
	for i, item := range declared {
		row := object(t, fmt.Sprintf("targets.declared[%d]", i), item)
		keySet(t, fmt.Sprintf("targets.declared[%d]", i), row, "probe", "target", "protocol")
		if got, want := stringAt(t, "target", row["target"]), input.Declared[i].Address(); got != want {
			t.Errorf("targets.declared[%d].target = %q, want %q", i, got, want)
		}
		if got, want := stringAt(t, "protocol", row["protocol"]), string(input.Declared[i].Protocol); got != want {
			t.Errorf("targets.declared[%d].protocol = %q, want %q", i, got, want)
		}
	}

	probes := array(t, "probes", document["probes"])
	if len(probes) != len(input.Results) {
		t.Fatalf("probes has %d rows, want %d", len(probes), len(input.Results))
	}
	for i, item := range probes {
		path := fmt.Sprintf("probes[%d]", i)
		row := object(t, path, item)
		keySet(t, path, row, "name", "kind", "target", "verdict", "resolution", "reason", "detail", "elapsed_ms", "observations")
		result := resultNamed(t, input.Results, stringAt(t, path+".name", row["name"]))
		stringAt(t, path+".kind", row["kind"])
		targetAt(t, path+".target", row["target"], result.Target)
		stringAt(t, path+".verdict", row["verdict"])
		stringAt(t, path+".resolution", row["resolution"])
		stringAt(t, path+".reason", row["reason"])
		stringAt(t, path+".detail", row["detail"])
		numberAt(t, path+".elapsed_ms", row["elapsed_ms"])
		observations := array(t, path+".observations", row["observations"])
		if len(observations) != len(result.Observations) {
			t.Errorf("%s.observations has %d rows, want the result's %d", path, len(observations), len(result.Observations))
			continue
		}
		for j, observationItem := range observations {
			observationPath := fmt.Sprintf("%s.observations[%d]", path, j)
			observation := object(t, observationPath, observationItem)
			keySet(t, observationPath, observation, "label", "target", "verdict", "resolution", "reason", "detail")
			stringAt(t, observationPath+".label", observation["label"])
			targetAt(t, observationPath+".target", observation["target"], result.Observations[j].Target)
			stringAt(t, observationPath+".verdict", observation["verdict"])
			stringAt(t, observationPath+".resolution", observation["resolution"])
			stringAt(t, observationPath+".reason", observation["reason"])
			stringAt(t, observationPath+".detail", observation["detail"])
		}
	}

	findings := array(t, "findings", document["findings"])
	if len(findings) != len(input.Diagnosis.Findings) {
		t.Fatalf("findings has %d rows, want %d", len(findings), len(input.Diagnosis.Findings))
	}
	for i, item := range findings {
		path := fmt.Sprintf("findings[%d]", i)
		row := object(t, path, item)
		keySet(t, path, row, "question", "rule", "conclusion", "depends_on")
	}

	openQuestions := array(t, "open_questions", document["open_questions"])
	if len(openQuestions) != len(input.Diagnosis.OpenQuestions) {
		t.Fatalf("open_questions has %d rows, want %d", len(openQuestions), len(input.Diagnosis.OpenQuestions))
	}
	for i, item := range openQuestions {
		path := fmt.Sprintf("open_questions[%d]", i)
		row := object(t, path, item)
		keySet(t, path, row, "question", "needed_states")
	}

	node := object(t, "node", document["node"])
	keySet(t, "node", node, "platform", "arch", "refused", "note")
	stringAt(t, "node.platform", node["platform"])
	stringAt(t, "node.arch", node["arch"])
	if _, ok := node["refused"].(bool); !ok {
		t.Errorf("node.refused is %T, want a JSON boolean", node["refused"])
	}
	stringAt(t, "node.note", node["note"])

	transports := array(t, "transports", document["transports"])
	if len(transports) != len(input.Transports) {
		t.Fatalf("transports has %d rows, want %d", len(transports), len(input.Transports))
	}
	for i, item := range transports {
		path := fmt.Sprintf("transports[%d]", i)
		row := object(t, path, item)
		keySet(t, path, row, "name", "viable", "reason", "requires", "notes")
		stringAt(t, path+".name", row["name"])
		if _, ok := row["viable"].(bool); !ok {
			t.Errorf("%s.viable is %T, want a JSON boolean", path, row["viable"])
		}
		stringAt(t, path+".reason", row["reason"])
		for j, requirementItem := range array(t, path+".requires", row["requires"]) {
			requirementPath := fmt.Sprintf("%s.requires[%d]", path, j)
			requirement := object(t, requirementPath, requirementItem)
			keySet(t, requirementPath, requirement, "kind", "satisfied", "detail")
			stringAt(t, requirementPath+".kind", requirement["kind"])
			if _, ok := requirement["satisfied"].(bool); !ok {
				t.Errorf("%s.satisfied is %T, want a JSON boolean", requirementPath, requirement["satisfied"])
			}
			stringAt(t, requirementPath+".detail", requirement["detail"])
		}
		for j, note := range array(t, path+".notes", row["notes"]) {
			stringAt(t, fmt.Sprintf("%s.notes[%d]", path, j), note)
		}
	}
}

// TestPayloadNullHubIsJSONNullNotAnEmptyString pins targets.hub's own null: a run with no hub
// entry emits JSON null, which a script can tell apart from a hostname, and declares no hub row at
// all in targets.declared rather than a row with an empty target.
func TestPayloadNullHubIsJSONNullNotAnEmptyString(t *testing.T) {
	input := fullInput()
	input.Declared = declaredWithoutHub()
	payload := report.Build(input)
	raw := marshalPayload(t, payload)

	if !strings.Contains(string(raw), `"hub":null`) {
		t.Errorf("the raw document does not carry \"hub\":null: %s", raw)
	}
	document := decodePayload(t, payload)
	targets := object(t, "targets", document["targets"])
	if hub := targets["hub"]; hub != nil {
		t.Errorf("targets.hub = %#v, want JSON null: one of the payload's nullable targets", hub)
	}
	declared := array(t, "targets.declared", targets["declared"])
	if len(declared) == 0 {
		t.Fatal("targets.declared is empty, so the case cannot show that a run with no hub still declares its resolved set")
	}
	for i, item := range declared {
		row := object(t, fmt.Sprintf("targets.declared[%d]", i), item)
		if probeName := stringAt(t, fmt.Sprintf("targets.declared[%d].probe", i), row["probe"]); probeName == "egress.hub.direct" {
			t.Errorf("targets.declared[%d] is a hub entry: a run that supplied no hub declares no hub row", i)
		}
		stringAt(t, fmt.Sprintf("targets.declared[%d].target", i), row["target"])
	}
}

// TestPayloadNullableFieldsAreExactlyTheTargets pins the whole nullable set: the only JSON nulls a
// document may carry are the probe and observation target fields that have no target, plus
// targets.hub when the run supplied no hub. Every other field at every level is present with a
// non-null value, so an added nullable field is a deliberate act.
func TestPayloadNullableFieldsAreExactlyTheTargets(t *testing.T) {
	wantTargets := []string{
		"probes[1].target",
		"probes[1].observations[1].target",
		"probes[2].target",
		"probes[2].observations[0].target",
	}

	withHub := nullPaths(decodePayload(t, report.Build(fullInput())), "")
	slices.Sort(withHub)
	want := append([]string(nil), wantTargets...)
	slices.Sort(want)
	if !slices.Equal(withHub, want) {
		t.Errorf("the document's JSON nulls are %v, want exactly %v: the empty target fields are nullable and nothing else", withHub, want)
	}

	noHubInput := fullInput()
	noHubInput.Declared = declaredWithoutHub()
	noHub := nullPaths(decodePayload(t, report.Build(noHubInput)), "")
	slices.Sort(noHub)
	wantNoHub := append(append([]string(nil), wantTargets...), "targets.hub")
	slices.Sort(wantNoHub)
	if !slices.Equal(noHub, wantNoHub) {
		t.Errorf("the document's JSON nulls are %v, want exactly %v: targets.hub joins the nullable set when no hub was supplied", noHub, wantNoHub)
	}
}

// TestPayloadEmptyCollectionsSerializeAsArrays asserts that every collection is
// always present and serialises as `[]`, never as null, on a minimal run.
func TestPayloadEmptyCollectionsSerializeAsArrays(t *testing.T) {
	input := report.Input{Run: testOptions()}
	payload := report.Build(input)
	raw := string(marshalPayload(t, payload))
	document := decodePayload(t, payload)

	for _, fragment := range []string{
		`"declared":[]`, `"probes":[]`, `"findings":[]`, `"open_questions":[]`,
		`"transports":[]`, `"unresolved":[]`, `"not_measured":[]`,
	} {
		if !strings.Contains(raw, fragment) {
			t.Errorf("the raw document does not carry %s: an empty collection must serialise as [] and never as null: %s", fragment, raw)
		}
	}

	run := object(t, "run", document["run"])
	for _, name := range []string{"unresolved", "not_measured"} {
		if items := array(t, "run."+name, run[name]); len(items) != 0 {
			t.Errorf("run.%s = %v, want an empty array", name, items)
		}
	}
	targets := object(t, "targets", document["targets"])
	if items := array(t, "targets.declared", targets["declared"]); len(items) != 0 {
		t.Errorf("targets.declared = %v, want an empty array", items)
	}
	for _, name := range []string{"probes", "findings", "open_questions", "transports"} {
		if items := array(t, name, document[name]); len(items) != 0 {
			t.Errorf("%s = %v, want an empty array", name, items)
		}
	}
}

// --- registry order, observation order and transport sorting ------------------

// TestMappingFollowsRegistryOrderRegardlessOfResultOrder asserts the probes
// array follows probe.Registry() order whatever order the results arrived in,
// and that a result the registry does not declare is ignored.
func TestMappingFollowsRegistryOrderRegardlessOfResultOrder(t *testing.T) {
	input := fullInput()
	slices.Reverse(input.Results)
	input.Results = append(input.Results, resultOf("made.up.probe", probe.ProbeEgress, "", 0, "not a registered probe",
		passObservation("invented", "invented:1", "not part of the run's contract")))

	payload := report.Build(input)
	var got []string
	for _, row := range payload.Probes {
		got = append(got, row.Name)
	}
	want := []string{"local.env", "local.sshd", "egress.hub.direct", "egress.ssh.known"}
	if !slices.Equal(got, want) {
		t.Errorf("probes order = %v, want registry order %v regardless of the results' arrival order", got, want)
	}
}

// TestMappingKeepsTheObservationOrderTheProbeReported asserts each probe's
// observations are emitted in the order the probe reported them.
func TestMappingKeepsTheObservationOrderTheProbeReported(t *testing.T) {
	result := resultOf("local.sshd", probe.ProbeLocal, "", time.Millisecond, "three facts",
		failObservation("first", "/usr/sbin/sshd", probe.ReasonSSHDAbsent, "first detail"),
		notMeasuredObservation("second", probe.ReasonCapabilityExcluded, "second detail"),
		passObservation("third", "/etc/ssh/sshd_config", "third detail"))
	payload := report.Build(report.Input{Run: testOptions(), Results: []probe.Result{result}})
	if len(payload.Probes) != 1 {
		t.Fatalf("probes has %d rows, want 1", len(payload.Probes))
	}

	var got []string
	for _, observation := range payload.Probes[0].Observations {
		got = append(got, observation.Label)
	}
	if want := []string{"first", "second", "third"}; !slices.Equal(got, want) {
		t.Errorf("observation order = %v, want the reported order %v", got, want)
	}
}

// TestMappingSortsTransportsByName asserts the transports are sorted by name
// whatever order the paired rows arrived in, and that the input slice is not
// reordered by the mapping.
func TestMappingSortsTransportsByName(t *testing.T) {
	input := report.Input{
		Run: testOptions(),
		Transports: []report.TransportRow{
			{Name: "tailscale", Feasibility: transport.Feasibility{Viable: false, Reason: "third"}},
			{Name: "cloudflare-tunnel", Feasibility: transport.Feasibility{Viable: false, Reason: "first"}},
			{Name: "direct-ssh", Feasibility: transport.Feasibility{Viable: false, Reason: "second"}},
		},
	}
	payload := report.Build(input)

	var got []string
	for _, row := range payload.Transports {
		got = append(got, row.Name)
	}
	if want := []string{"cloudflare-tunnel", "direct-ssh", "tailscale"}; !slices.Equal(got, want) {
		t.Errorf("transports order = %v, want sorted by name %v", got, want)
	}
	if input.Transports[0].Name != "tailscale" {
		t.Errorf("the mapping reordered its input slice: %v", input.Transports)
	}
}

// TestMappingCarriesFindingsAndOpenQuestionsVerbatim asserts the diagnosis is
// echoed in its own order with rule, conclusion, depends_on and needed_states
// copied verbatim, and that the payload carries no field the design does not
// name — evidence in particular.
func TestMappingCarriesFindingsAndOpenQuestionsVerbatim(t *testing.T) {
	input := fullInput()
	payload := report.Build(input)

	if len(payload.Findings) != 1 {
		t.Fatalf("findings has %d rows, want 1", len(payload.Findings))
	}
	finding := payload.Findings[0]
	wantFinding := input.Diagnosis.Findings[0]
	if finding.Question != wantFinding.Question || finding.Rule != wantFinding.Rule || finding.Conclusion != wantFinding.Conclusion {
		t.Errorf("finding = %+v, want question/rule/conclusion verbatim from %+v", finding, wantFinding)
	}
	if !slices.Equal(finding.DependsOn, wantFinding.DependsOn) {
		t.Errorf("finding.depends_on = %v, want %v", finding.DependsOn, wantFinding.DependsOn)
	}

	if len(payload.OpenQuestions) != 1 {
		t.Fatalf("open_questions has %d rows, want 1", len(payload.OpenQuestions))
	}
	if got, want := payload.OpenQuestions[0].NeededStates, input.Diagnosis.OpenQuestions[0].NeededStates; !slices.Equal(got, want) {
		t.Errorf("open_question.needed_states = %v, want %v", got, want)
	}
}

// --- the node classification -------------------------------------------------

// TestMappingNodeReadsTheClassificationAndTheRefusal asserts node comes from the
// local.env observation, and refused from the refusal rule firing — and that a
// run that never classified the machine reports the unknown platform rather
// than inventing one.
func TestMappingNodeReadsTheClassificationAndTheRefusal(t *testing.T) {
	note := "native Windows node (GOOS \"windows\", architecture \"amd64\"): this tool does not operate on Windows itself, and WSL2 is the supported path on a Windows machine; nothing was changed"
	windowsResult := resultOf("local.env", probe.ProbeLocal, "windows-native/amd64", time.Millisecond, note,
		failObservation("platform", "windows-native/amd64", probe.ReasonNodePlatformUnsupported, note))
	refusal := diagnosis.Diagnosis{Findings: []diagnosis.Finding{{
		Question: "node.platform",
		Rule:     "NODE_PLATFORM_REFUSED_NATIVE_WINDOWS",
		Evidence: []diagnosis.Fact{{Probe: "local.env", State: diagnosis.StateFail}},
	}}}

	payload := report.Build(report.Input{Run: testOptions(), Results: []probe.Result{windowsResult}, Diagnosis: refusal})
	if payload.Node.Platform != "windows-native" || payload.Node.Arch != "amd64" {
		t.Errorf("node = %q/%q, want windows-native/amd64 from the classification's own target", payload.Node.Platform, payload.Node.Arch)
	}
	if !payload.Node.Refused {
		t.Error("node.refused = false, want true: the refusal rule fired")
	}
	if payload.Node.Note != note {
		t.Errorf("node.note = %q, want the classification's verbatim detail %q", payload.Node.Note, note)
	}

	linuxResult := resultOf("local.env", probe.ProbeLocal, "linux/aarch64", time.Millisecond, "linux detail",
		passObservation("platform", "linux/aarch64", "linux detail"))
	payload = report.Build(report.Input{Run: testOptions(), Results: []probe.Result{linuxResult}})
	if payload.Node.Platform != "linux" || payload.Node.Arch != "aarch64" || payload.Node.Refused {
		t.Errorf("node = %+v, want linux/aarch64 with no refusal", payload.Node)
	}

	// A run that never classified the machine must not invent a platform: the
	// classification absence is the probe's own unknown vocabulary.
	payload = report.Build(report.Input{Run: testOptions()})
	if payload.Node.Platform == "linux" || payload.Node.Platform != string(probe.NodePlatformUnknown) {
		t.Errorf("node.platform = %q, want the unknown classification, never a guessed platform", payload.Node.Platform)
	}
	if payload.Node.Refused {
		t.Error("a run that never classified the machine cannot have refused it")
	}
	if payload.Node.Note != "" {
		t.Errorf("node.note = %q, want empty: no classification means no verbatim detail", payload.Node.Note)
	}

	// A classification observation whose target is not an identity the probe's
	// own format recognises must not become a platform: the mapping reports the
	// unknown identity while still carrying the classification's verbatim detail.
	illFormed := resultOf("local.env", probe.ProbeLocal, "not-an-identity", time.Millisecond, "ill-formed detail",
		passObservation("platform", "not-an-identity", "ill-formed detail"))
	payload = report.Build(report.Input{Run: testOptions(), Results: []probe.Result{illFormed}})
	if payload.Node.Platform != string(probe.NodePlatformUnknown) {
		t.Errorf("node.platform = %q for an ill-formed classification target, want the unknown platform", payload.Node.Platform)
	}
	if payload.Node.Note != "ill-formed detail" {
		t.Errorf("node.note = %q, want the classification's verbatim detail", payload.Node.Note)
	}
}

// TestMappingAggregateResolutionFollowsTheWorstObservation asserts the row's
// resolution is the resolution of the observation the measurement layer's own
// reduction selects, in the documented order fail above indeterminate above
// pass and with the earlier observation keeping a tie. A verdict must never be
// carried with a resolution the same reduction would not produce.
func TestMappingAggregateResolutionFollowsTheWorstObservation(t *testing.T) {
	cases := []struct {
		name         string
		observations []probe.Observation
		verdict      probe.Verdict
		resolution   probe.Resolution
	}{
		{
			"a pass beside an unresolved attempt",
			[]probe.Observation{
				passObservation("binary present", "/usr/sbin/sshd", "present"),
				unresolvedObservation("effective config", "/etc/ssh/sshd_config", probe.ReasonInternalError, "no answer"),
			},
			probe.Indeterminate, probe.Unresolved,
		},
		{
			"a failure beside an unresolved attempt",
			[]probe.Observation{
				failObservation("binary present", "/usr/sbin/sshd", probe.ReasonSSHDAbsent, "absent"),
				unresolvedObservation("effective config", "/etc/ssh/sshd_config", probe.ReasonInternalError, "no answer"),
			},
			probe.Fail, probe.Measured,
		},
		{
			"two indeterminate absences keep the first reported",
			[]probe.Observation{
				notMeasuredObservation("service state", probe.ReasonCapabilityExcluded, "not attempted"),
				unresolvedObservation("effective config", "/etc/ssh/sshd_config", probe.ReasonInternalError, "no answer"),
			},
			probe.Indeterminate, probe.NotMeasured,
		},
		{
			"an unresolved attempt beside a later pass",
			[]probe.Observation{
				unresolvedObservation("effective config", "/etc/ssh/sshd_config", probe.ReasonInternalError, "no answer"),
				passObservation("binary present", "/usr/sbin/sshd", "present"),
			},
			probe.Indeterminate, probe.Unresolved,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := resultOf("local.sshd", probe.ProbeLocal, "", time.Millisecond, "aggregate", testCase.observations...)
			payload := report.Build(report.Input{Run: testOptions(), Results: []probe.Result{result}})
			if len(payload.Probes) != 1 {
				t.Fatalf("probes has %d rows, want 1", len(payload.Probes))
			}
			if got := payload.Probes[0].Verdict; got != testCase.verdict {
				t.Errorf("verdict = %q, want %q", got, testCase.verdict)
			}
			if got := payload.Probes[0].Resolution; got != testCase.resolution {
				t.Errorf("resolution = %q, want %q from the observation Aggregate selected", got, testCase.resolution)
			}
		})
	}
}

// --- the one-producer property -----------------------------------------------

// TestMappingIgnoresStructureThePayloadDoesNotCarry asserts the payload is a
// projection of the values the design names: mutating a structure it does not
// carry (a finding's evidence) leaves the document unchanged, while mutating a
// field it does carry moves it — the control that proves the first result is
// about the mapping and not about a mapping that ignores its input.
func TestMappingIgnoresStructureThePayloadDoesNotCarry(t *testing.T) {
	input := fullInput()
	before := marshalPayload(t, report.Build(input))

	input.Diagnosis.Findings[0].Evidence[0].State = diagnosis.StateUnresolved
	input.Diagnosis.Findings[0].Evidence[0].Observation.Detail = "an edited evidence detail the payload must not carry"
	input.Diagnosis.Findings[0].Evidence = append(input.Diagnosis.Findings[0].Evidence, diagnosis.Fact{
		Probe: "tls.truststore", Resolution: probe.Unresolved, Verdict: probe.Indeterminate,
		State:       diagnosis.StateUnresolved,
		Observation: unresolvedObservation("trust store", "www.cloudflare.com:443", probe.ReasonTrustStorePlatformUnavailable, "an appended evidence fact"),
	})
	after := marshalPayload(t, report.Build(input))
	if !bytes.Equal(before, after) {
		t.Errorf("editing Finding.Evidence changed the payload:\nbefore: %s\nafter:  %s", before, after)
	}

	input.Diagnosis.Findings[0].Conclusion = "an edited conclusion the payload does carry"
	changed := marshalPayload(t, report.Build(input))
	if bytes.Equal(before, changed) {
		t.Fatal("editing the finding's conclusion did not change the payload, so the case's mutation cannot prove anything")
	}
}

// TestOnlyProducerIsTheMappingInMapGo is the static half of the one-producer
// proof: the payload's types may be constructed only in map.go, so no second
// site can grow a document the mapping does not know about.
//
// The scan reads the package's own non-test sources. Three controls keep it
// falsifiable: the file floor fails a scan that read nothing or read the wrong
// directory, the declaration check fails a scan whose type list has drifted from
// report.go, and the per-type occurrence floor fails a type that is declared
// nowhere it is built.
func TestOnlyProducerIsTheMappingInMapGo(t *testing.T) {
	// dtoOutputTypes are the payload's own types, in the order of design §3.3.
	// Input and TransportRow are the mapping's input types and are constructed by
	// the caller, so they are deliberately not part of this scan.
	dtoOutputTypes := []string{
		"Payload", "ToolInfo", "RunInfo", "TargetsInfo", "DeclaredTarget",
		"ProbeRow", "ObservationRow", "FindingRow", "OpenQuestionRow",
		"NodeInfo", "TransportInfo", "RequirementRow",
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("the report package directory could not be read: %v", err)
	}
	type source struct {
		name    string
		content string
	}
	var sources []source
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("the production source %s could not be read: %v", name, err)
		}
		sources = append(sources, source{name: name, content: string(data)})
	}

	// Control 1: a scan that read nothing, or read the wrong directory, must fail
	// instead of passing silently. The package ships the DTO and the mapping as
	// two sources today, so two is a floor a broken listing falls under.
	const minProductionFiles = 2
	if len(sources) < minProductionFiles {
		t.Fatalf("the scan read %d production sources, want at least %d: a scan that read nothing must fail rather than pass silently", len(sources), minProductionFiles)
	}
	byName := map[string]string{}
	for _, src := range sources {
		byName[src.name] = src.content
	}
	for _, required := range []string{"report.go", "map.go"} {
		if _, read := byName[required]; !read {
			t.Fatalf("the scan did not read %s, so it cannot say where the payload types are constructed", required)
		}
	}

	for _, name := range dtoOutputTypes {
		// Control 2: the type must be declared in report.go, or the scan's list
		// has drifted from the DTO and its occurrences prove nothing.
		if !strings.Contains(byName["report.go"], "type "+name+" struct") {
			t.Fatalf("report.go does not declare a struct %s, so the scan's type list has drifted from the DTO", name)
		}

		// Control 3: every of the payload's own types must be constructed at
		// least once, and every construction site must be map.go.
		found := 0
		var carriers []string
		for _, src := range sources {
			count := strings.Count(src.content, name+"{")
			if count == 0 {
				continue
			}
			found += count
			carriers = append(carriers, src.name)
		}
		if found == 0 {
			t.Fatalf("no source constructs %s, so the one-producer scan cannot see the payload's own construction sites", name)
		}
		for _, carrier := range carriers {
			if carrier != "map.go" {
				t.Errorf("%s constructs %s: the payload's types may be constructed only in map.go, the single mapping", carrier, name)
			}
		}
	}
}

// TestMappingSpelledLayerConstantsRemainDeclared guards the three names the
// mapping spells because the layers that own them keep them unexported: the two
// probe names it must recognise in a result and the refusal rule id it reads
// from the diagnosis. If a layer renames any of them, the mapping would fall
// silently onto its absence path (no hub address, no classification, no
// refusal), so this case fails instead.
func TestMappingSpelledLayerConstantsRemainDeclared(t *testing.T) {
	var registered []string
	for _, registration := range probe.Registry() {
		registered = append(registered, registration.Name)
	}
	for _, name := range []string{"egress.hub.direct", "local.env"} {
		if !slices.Contains(registered, name) {
			t.Errorf("the mapping spells the probe name %q, which probe.Registry() no longer declares", name)
		}
	}
	if declared := diagnosis.AllRuleIDs(); !slices.Contains(declared, "NODE_PLATFORM_REFUSED_NATIVE_WINDOWS") {
		t.Errorf("the mapping spells the refusal rule id NODE_PLATFORM_REFUSED_NATIVE_WINDOWS, which diagnosis.AllRuleIDs() no longer declares: %v", declared)
	}
}

// --- coverage and completeness -----------------------------------------------

// TestCoverageNotMeasuredProbeIsNamedAndLeavesTheRunComplete asserts a
// not-measured observation appears with its resolution, its reason naming the
// missing input and a null target, is listed in run.not_measured, and does not
// make the run incomplete.
func TestCoverageNotMeasuredProbeIsNamedAndLeavesTheRunComplete(t *testing.T) {
	payload := report.Build(fullInput())
	if payload.Run.Completeness != report.CompletenessComplete {
		t.Errorf("run.completeness = %q, want %q: a not-measured observation never makes the run incomplete",
			payload.Run.Completeness, report.CompletenessComplete)
	}
	if !slices.Contains(payload.Run.NotMeasured, "egress.hub.direct") {
		t.Errorf("run.not_measured = %v, want egress.hub.direct named", payload.Run.NotMeasured)
	}
	if len(payload.Run.Unresolved) != 0 {
		t.Errorf("run.unresolved = %v, want none: no observation was attempted and left unanswered", payload.Run.Unresolved)
	}

	document := decodePayload(t, payload)
	row := rowFor(t, document, "egress.hub.direct")
	if got := stringAt(t, "egress.hub.direct.resolution", row["resolution"]); got != string(probe.NotMeasured) {
		t.Errorf("resolution = %q, want %q", got, probe.NotMeasured)
	}
	if got := stringAt(t, "egress.hub.direct.verdict", row["verdict"]); got != string(probe.Indeterminate) {
		t.Errorf("verdict = %q, want %q: a not-measured observation is never a pass", got, probe.Indeterminate)
	}
	if got := stringAt(t, "egress.hub.direct.reason", row["reason"]); got != string(probe.ReasonInputMissingHub) {
		t.Errorf("reason = %q, want %q naming the missing input", got, probe.ReasonInputMissingHub)
	}
	if target := row["target"]; target != nil {
		t.Errorf("the not-measured hub row's target = %#v, want JSON null: a target the run did not carry is null, never an empty string", target)
	}
	observations := array(t, "egress.hub.direct.observations", row["observations"])
	if len(observations) != 1 {
		t.Fatalf("the not-measured hub row carries %d observations, want one", len(observations))
	}
	observation := object(t, "egress.hub.direct.observations[0]", observations[0])
	if target := observation["target"]; target != nil {
		t.Errorf("the not-measured hub observation's target = %#v, want JSON null: the observation carries no target either", target)
	}
}

// TestCoverageUnresolvedProbeIsNamedAndMakesTheRunIncomplete asserts an
// attempted observation that produced no answer is listed in run.unresolved and
// makes the run incomplete (R-HR-NF-02).
func TestCoverageUnresolvedProbeIsNamedAndMakesTheRunIncomplete(t *testing.T) {
	input := fullInput()
	input.Results = append(input.Results, resultOf("tls.truststore", probe.ProbeTLS, "www.cloudflare.com:443", time.Millisecond,
		"the platform verifier could not answer",
		unresolvedObservation("trust store", "www.cloudflare.com:443", probe.ReasonTrustStorePlatformUnavailable,
			"the platform verifier cannot answer for this target")))

	payload := report.Build(input)
	if payload.Run.Completeness != report.CompletenessIncomplete {
		t.Errorf("run.completeness = %q, want %q: the unresolved attempt makes the run incomplete",
			payload.Run.Completeness, report.CompletenessIncomplete)
	}
	if !slices.Contains(payload.Run.Unresolved, "tls.truststore") {
		t.Errorf("run.unresolved = %v, want tls.truststore named", payload.Run.Unresolved)
	}
	if slices.Contains(payload.Run.NotMeasured, "tls.truststore") {
		t.Errorf("run.not_measured = %v, must not name an attempted probe: unresolved and not measured are distinct outcomes", payload.Run.NotMeasured)
	}
}

// TestCoverageListsAreDeduplicatedAndInRegistryOrder asserts both coverage lists
// name each probe once and follow registry order, independent of the order the
// results arrived in and of how many observations a probe reported.
func TestCoverageListsAreDeduplicatedAndInRegistryOrder(t *testing.T) {
	input := report.Input{
		Run: testOptions(),
		Results: []probe.Result{
			// Arrival order is deliberately the reverse of registry order.
			resultOf("tls.truststore", probe.ProbeTLS, "www.cloudflare.com:443", time.Millisecond, "attempted",
				unresolvedObservation("trust store", "www.cloudflare.com:443", probe.ReasonTrustStorePlatformUnavailable, "attempted and unanswered")),
			resultOf("egress.hub.direct", probe.ProbeEgress, "", time.Millisecond, "not attempted",
				notMeasuredObservation("hub target", probe.ReasonInputMissingHub, "no hub address was supplied")),
			resultOf("local.sshd", probe.ProbeLocal, "", time.Millisecond, "two facts never attempted",
				notMeasuredObservation("service state", probe.ReasonCapabilityExcluded, "no command runner is wired"),
				notMeasuredObservation("effective config", probe.ReasonCapabilityExcluded, "no command runner is wired")),
		},
	}

	payload := report.Build(input)
	if want := []string{"local.sshd", "egress.hub.direct"}; !slices.Equal(payload.Run.NotMeasured, want) {
		t.Errorf("run.not_measured = %v, want %v: deduplicated and in registry order", payload.Run.NotMeasured, want)
	}
	if want := []string{"tls.truststore"}; !slices.Equal(payload.Run.Unresolved, want) {
		t.Errorf("run.unresolved = %v, want %v: deduplicated and in registry order", payload.Run.Unresolved, want)
	}
}

// TestCompletenessIsDerivedFromResolutionNotVerdict asserts the reconciliation
// of design D5 at the payload level: two observations whose verdicts are both
// indeterminate produce different completeness solely because one was attempted
// and the other was not.
func TestCompletenessIsDerivedFromResolutionNotVerdict(t *testing.T) {
	notMeasured := resultOf("egress.hub.direct", probe.ProbeEgress, "", time.Millisecond, "not measured",
		notMeasuredObservation("hub target", probe.ReasonInputMissingHub, "no hub address was supplied"))
	payload := report.Build(report.Input{Run: testOptions(), Results: []probe.Result{notMeasured}})
	if payload.Run.Completeness != report.CompletenessComplete {
		t.Errorf("completeness = %q for a not-measured observation, want %q", payload.Run.Completeness, report.CompletenessComplete)
	}

	unresolved := resultOf("egress.hub.direct", probe.ProbeEgress, "203.0.113.10:22", time.Millisecond, "attempted",
		unresolvedObservation("tcp 22", "203.0.113.10:22", probe.ReasonInternalError, "the attempt produced no answer"))
	payload = report.Build(report.Input{Run: testOptions(), Results: []probe.Result{unresolved}})
	if payload.Run.Completeness != report.CompletenessIncomplete {
		t.Errorf("completeness = %q for an unresolved observation, want %q", payload.Run.Completeness, report.CompletenessIncomplete)
	}
}

// --- determinism -------------------------------------------------------------

// TestDeterminismRepeatedBuildsAreByteIdentical asserts identical inputs produce
// a byte-identical document (R-HR-07).
func TestDeterminismRepeatedBuildsAreByteIdentical(t *testing.T) {
	input := fullInput()
	first := marshalPayload(t, report.Build(input))
	second := marshalPayload(t, report.Build(input))
	if !bytes.Equal(first, second) {
		t.Errorf("two builds over the same input differ:\nfirst:  %s\nsecond: %s", first, second)
	}
}

// TestDeterminismResultOrderDoesNotChangeThePayload asserts shuffling the
// results' arrival order does not change the document: the probes follow the
// registry, not the input.
func TestDeterminismResultOrderDoesNotChangeThePayload(t *testing.T) {
	input := fullInput()
	before := marshalPayload(t, report.Build(input))
	slices.Reverse(input.Results)
	after := marshalPayload(t, report.Build(input))
	if !bytes.Equal(before, after) {
		t.Errorf("the results' arrival order changed the payload:\nbefore: %s\nafter:  %s", before, after)
	}
}

// TestDeterminismClockOnlyChangeMovesOnlyGeneratedAt asserts one injected clock
// is the only thing a timestamp change moves: the same run under a second clock
// produces a document whose every field but generated_at is unchanged.
func TestDeterminismClockOnlyChangeMovesOnlyGeneratedAt(t *testing.T) {
	first := fullInput()
	second := fullInput()
	later := testTime.Add(time.Hour)
	second.Run.Clock = fixedClock{at: later}

	firstDocument := decodePayload(t, report.Build(first))
	secondDocument := decodePayload(t, report.Build(second))

	if firstDocument["generated_at"] == secondDocument["generated_at"] {
		t.Fatalf("generated_at did not move with the clock: both %v", firstDocument["generated_at"])
	}
	delete(firstDocument, "generated_at")
	delete(secondDocument, "generated_at")
	if !reflect.DeepEqual(firstDocument, secondDocument) {
		t.Errorf("a clock-only change moved more than generated_at:\nfirst:  %#v\nsecond: %#v", firstDocument, secondDocument)
	}
}

// --- the millisecond unit ----------------------------------------------------

// TestElapsedMSIsIntegerMilliseconds asserts elapsed_ms is an integer number of
// milliseconds in one documented unit: a known duration with a sub-millisecond
// remainder truncates to its whole milliseconds.
func TestElapsedMSIsIntegerMilliseconds(t *testing.T) {
	result := resultOf("local.env", probe.ProbeLocal, "linux/aarch64",
		2*time.Second+750*time.Millisecond+400*time.Microsecond, "measured",
		passObservation("platform", "linux/aarch64", "measured"))
	payload := report.Build(report.Input{Run: testOptions(), Results: []probe.Result{result}})
	if len(payload.Probes) != 1 {
		t.Fatalf("probes has %d rows, want 1", len(payload.Probes))
	}

	if got := payload.Probes[0].ElapsedMS; got != 2750 {
		t.Errorf("elapsed_ms = %d, want 2750: an integer number of milliseconds", got)
	}
	if got := payload.Run.RunBudgetMS; got != probe.DefaultRunBudget.Milliseconds() {
		t.Errorf("run_budget_ms = %d, want %d", got, probe.DefaultRunBudget.Milliseconds())
	}

	document := decodePayload(t, payload)
	row := rowFor(t, document, "local.env")
	if got := numberAt(t, "local.env.elapsed_ms", row["elapsed_ms"]); got != 2750 {
		t.Errorf("the decoded elapsed_ms = %v, want the JSON number 2750", got)
	}
}

// --- NF-03: unresolved is never encoded as success ---------------------------

// TestNoPassPromotionOfUnresolvedProbe is R-HR-NF-03's payload proof. It asserts
// the probe's verdict verbatim and, structurally, that the document carries no
// field that could mark the unresolved measurement successful: the exact key set
// at every level is asserted, and the only booleans anywhere in the document are
// the three the design names.
func TestNoPassPromotionOfUnresolvedProbe(t *testing.T) {
	input := report.Input{
		Run: testOptions(),
		Results: []probe.Result{
			resultOf("egress.quic", probe.ProbeProto, "region1.v2.argotunnel.com:7844", 2*time.Millisecond,
				"udp datagrams were sent and none answered",
				unresolvedObservation("udp 7844 region1", "region1.v2.argotunnel.com:7844", probe.ReasonUDPSilence,
					"no reply arrived before the budget expired, so nothing is claimed about QUIC viability")),
		},
		Transports: []report.TransportRow{
			{Name: "direct-ssh", Feasibility: transport.Feasibility{
				Viable: false, Reason: "the hub measurement was not made",
				Requires: []transport.Requirement{
					{Kind: transport.KindHubAddress, Satisfied: false, Detail: "supply --hub host[:port]"},
				},
			}},
		},
	}

	payload := report.Build(input)
	if payload.Run.Completeness != report.CompletenessIncomplete {
		t.Errorf("run.completeness = %q, want %q", payload.Run.Completeness, report.CompletenessIncomplete)
	}
	if !slices.Contains(payload.Run.Unresolved, "egress.quic") {
		t.Errorf("run.unresolved = %v, want egress.quic named", payload.Run.Unresolved)
	}

	document := decodePayload(t, payload)
	row := rowFor(t, document, "egress.quic")
	if got := stringAt(t, "egress.quic.verdict", row["verdict"]); got != string(probe.Indeterminate) {
		t.Errorf("verdict = %q, want %q verbatim", got, probe.Indeterminate)
	}
	keySet(t, "egress.quic", row, "name", "kind", "target", "verdict", "resolution", "reason", "detail", "elapsed_ms", "observations")
	if got := stringAt(t, "egress.quic.resolution", row["resolution"]); got != string(probe.Unresolved) {
		t.Errorf("resolution = %q, want %q", got, probe.Unresolved)
	}

	// Structural absence of a success marker: the only booleans the document may
	// carry are the three fields the design names, so no derived boolean, status
	// or summary field can present the unresolved measurement as successful.
	gotBooleans := booleanPaths(document, "payload")
	slices.Sort(gotBooleans)
	wantBooleans := []string{
		"payload.node.refused",
		"payload.transports[0].requires[0].satisfied",
		"payload.transports[0].viable",
	}
	if !slices.Equal(gotBooleans, wantBooleans) {
		t.Errorf("the document carries booleans at %v, want exactly %v: no unnamed boolean may mark an unresolved measurement successful", gotBooleans, wantBooleans)
	}
	for _, value := range stringValues(row) {
		if value == string(probe.Pass) {
			t.Errorf("the unresolved probe's row carries the string %q, so a field maps it to a pass", value)
		}
	}
}
