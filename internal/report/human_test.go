package report_test

// This file is the human projection's suite (design §7, §8): the projection is
// rendered to an injected writer and only to it, the same payload renders
// byte-identically twice, an unresolved measurement is rendered as unresolved
// with its own reason and verbatim detail and never with success wording, an
// incomplete run names its unresolved probes and its coverage gaps, the blocked
// target and the verbatim failing detail survive beside the stable reason code,
// and the table geometry — headers and column shape — is pinned so a reflow is a
// deliberate edit rather than an accident.

import (
	"bytes"
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/report"
)

// --- fixtures ----------------------------------------------------------------

// blockedHubDetail is the verbatim detail of PRD §1.1's failing hub measurement:
// the declared address was dialed and its own budget expired. It is a constant
// so the fixture and the assertion quote the same bytes.
const blockedHubDetail = "dial tcp 203.0.113.10:22: i/o timeout after the probe's dial budget"

// unresolvedInput is the complete fixture with one probe attempted and left
// unresolved: tls.truststore's platform verifier cannot answer (RG-3), which
// makes the run incomplete, while the hub stays a not-measured coverage gap.
func unresolvedInput() report.Input {
	input := fullInput()
	input.Results = []probe.Result{
		resultOf("egress.hub.direct", probe.ProbeEgress, "", time.Millisecond,
			"no hub address was supplied for this run, so the hub measurement was not made",
			notMeasuredObservation("hub target", probe.ReasonInputMissingHub,
				"no hub address was supplied for this run, so the hub measurement was not made")),
		resultOf("tls.truststore", probe.ProbeTLS, "www.cloudflare.com:443", 2*time.Millisecond,
			"the platform verifier cannot answer for this target",
			unresolvedObservation("trust store", "www.cloudflare.com:443", probe.ReasonTrustStorePlatformUnavailable,
				"the platform verifier cannot answer for this target")),
	}
	input.Diagnosis = diagnosis.Diagnose(input.Results)
	return input
}

// blockedHubInput is PRD §1.1's replay: a public SSH measurement passed, so the
// protocol is not blocked, and the hub-directed measurement failed on its own
// dial budget, so the destination is. The failing detail is the fixture's own
// constant, so the projection cannot carry a paraphrase the assertion accepts.
func blockedHubInput() report.Input {
	input := fullInput()
	input.Results = []probe.Result{
		resultOf("egress.hub.direct", probe.ProbeEgress, "203.0.113.10:22", 4*time.Second, blockedHubDetail,
			failObservation("tcp 22", "203.0.113.10:22", probe.ReasonBudgetExpired, blockedHubDetail)),
		resultOf("egress.ssh.known", probe.ProbeEgress, "github.com:22", 3*time.Millisecond,
			"dial tcp github.com:22: the connection was established and closed",
			passObservation("tcp 22", "github.com:22",
				"dial tcp github.com:22: the connection was established and closed without reading or writing")),
	}
	input.Diagnosis = diagnosis.Diagnose(input.Results)
	return input
}

// --- helpers -----------------------------------------------------------------

// renderHuman renders one payload through the projection into a buffer, failing
// the case when the projection itself reports a writer failure.
func renderHuman(t *testing.T, payload report.Payload) string {
	t.Helper()
	var out bytes.Buffer
	if err := report.WriteHuman(&out, payload); err != nil {
		t.Fatalf("WriteHuman returned an error: %v", err)
	}
	return out.String()
}

// projectionRow returns the projection's line whose first column is name: the
// row starts at column zero and is followed by the separator space, so a
// prefix collision cannot return a different probe's row.
func projectionRow(t *testing.T, projection, name string) string {
	t.Helper()
	for _, line := range strings.Split(projection, "\n") {
		if strings.HasPrefix(line, name+" ") {
			return line
		}
	}
	t.Fatalf("the projection carries no row for %q", name)
	return ""
}

// coverageValue returns the value of one COVERAGE line, so the cases assert the
// completeness sentence and the two coverage lists instead of searching the
// whole projection for a word.
func coverageValue(t *testing.T, projection, key string) string {
	t.Helper()
	prefix := key + ": "
	for _, line := range strings.Split(projection, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	t.Fatalf("the projection carries no %q line", key)
	return ""
}

// hasField reports whether text carries field as a whitespace-separated field.
// It is how the success-wording cases look for the verdict and reason vocabulary
// without matching a substring of a longer word such as "bypass".
func hasField(text, field string) bool {
	return slices.Contains(strings.Fields(text), field)
}

// paddedCell pads one expected table cell to width, the way the projection pads
// its own cells. The geometry case builds its expectations from the documented
// widths here rather than from the implementation, so a width change fails
// unless both sides are edited.
func paddedCell(text string, width int) string {
	if len(text) >= width {
		return text
	}
	return text + strings.Repeat(" ", width-len(text))
}

// captureStdStreams runs fn with os.Stdout and os.Stderr redirected to pipes and
// returns everything written to each. The reads run concurrently, so a
// projection that wrote to a standard stream cannot deadlock the case.
func captureStdStreams(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	outRead, outWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe for stdout failed: %v", err)
	}
	errRead, errWrite, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe for stderr failed: %v", err)
	}

	previousOut, previousErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outWrite, errWrite

	type readResult struct {
		data []byte
		err  error
	}
	outReads := make(chan readResult, 1)
	errReads := make(chan readResult, 1)
	go func() {
		data, err := io.ReadAll(outRead)
		outReads <- readResult{data: data, err: err}
	}()
	go func() {
		data, err := io.ReadAll(errRead)
		errReads <- readResult{data: data, err: err}
	}()

	fn()

	os.Stdout, os.Stderr = previousOut, previousErr
	outWrite.Close()
	errWrite.Close()

	outResult := <-outReads
	errResult := <-errReads
	outRead.Close()
	errRead.Close()
	if outResult.err != nil {
		t.Fatalf("reading the captured stdout failed: %v", outResult.err)
	}
	if errResult.err != nil {
		t.Fatalf("reading the captured stderr failed: %v", errResult.err)
	}
	return string(outResult.data), string(errResult.data)
}

// failingWriter is an io.Writer that refuses every write, so the projection case
// can assert WriteHuman propagates the failure instead of swallowing it.
type failingWriter struct{ err error }

// Write refuses the write.
func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

// --- the projection goes only to its writer ----------------------------------

// TestHumanProjectionWritesOnlyToTheInjectedWriter asserts the projection
// reaches the writer it was handed and neither standard stream (R-HR-07: the
// human projection must not pollute standard output).
func TestHumanProjectionWritesOnlyToTheInjectedWriter(t *testing.T) {
	payload := report.Build(fullInput())

	var buffer bytes.Buffer
	var writeErr error
	stdout, stderr := captureStdStreams(t, func() {
		writeErr = report.WriteHuman(&buffer, payload)
	})
	if writeErr != nil {
		t.Fatalf("WriteHuman returned an error: %v", writeErr)
	}
	if buffer.Len() == 0 {
		t.Fatal("WriteHuman wrote nothing to the injected writer")
	}
	if stdout != "" {
		t.Errorf("WriteHuman wrote %q to os.Stdout: the projection must reach only the writer it was handed", stdout)
	}
	if stderr != "" {
		t.Errorf("WriteHuman wrote %q to os.Stderr: the projection must reach only the writer it was handed", stderr)
	}
}

// TestHumanProjectionIsByteIdenticalAcrossRenders asserts identical payloads
// render byte-identically, and that a second build of the same input renders the
// same text (R-HR-07's determinism requirement).
func TestHumanProjectionIsByteIdenticalAcrossRenders(t *testing.T) {
	payload := report.Build(fullInput())
	first := renderHuman(t, payload)
	second := renderHuman(t, payload)
	if first != second {
		t.Errorf("two renders of the same payload differ:\nfirst:  %q\nsecond: %q", first, second)
	}
	if third := renderHuman(t, report.Build(fullInput())); third != first {
		t.Errorf("a second build of the same input renders differently:\nfirst: %q\nthird: %q", first, third)
	}
}

// TestHumanProjectionReturnsWriterErrors asserts a refusing writer is reported
// to the caller rather than swallowed: the caller must be able to exit 2 instead
// of pretending the projection was written.
func TestHumanProjectionReturnsWriterErrors(t *testing.T) {
	wantErr := errors.New("the writer refused the write")
	err := report.WriteHuman(failingWriter{err: wantErr}, report.Build(fullInput()))
	if !errors.Is(err, wantErr) {
		t.Errorf("WriteHuman error = %v, want it to carry %v: a writer failure must not be reported as success", err, wantErr)
	}
}

// --- unresolved is never rendered as success ---------------------------------

// TestHumanProjectionRendersUnresolvedAsUnresolved asserts an attempted
// measurement that produced no answer is rendered with its own verdict,
// resolution, reason code and verbatim detail, and that no success wording is
// attributed to it (R-HR-NF-03).
func TestHumanProjectionRendersUnresolvedAsUnresolved(t *testing.T) {
	payload := report.Build(unresolvedInput())
	projection := renderHuman(t, payload)

	row := projectionRow(t, projection, "tls.truststore")
	for _, want := range []string{
		string(probe.Indeterminate),
		string(probe.Unresolved),
		string(probe.ReasonTrustStorePlatformUnavailable),
		"the platform verifier cannot answer for this target",
	} {
		if !strings.Contains(row, want) {
			t.Errorf("the unresolved tls.truststore row %q does not carry %q", row, want)
		}
	}
	for _, banned := range []string{string(probe.Pass), string(probe.ReasonOK)} {
		if hasField(row, banned) {
			t.Errorf("the unresolved tls.truststore row %q carries the success field %q: unresolved is never rendered as success", row, banned)
		}
	}
	if strings.Contains(row, "success") {
		t.Errorf("the unresolved tls.truststore row %q carries success wording", row)
	}
}

// TestHumanProjectionIncompleteRunNamesUnresolvedProbesAndGaps asserts the
// coverage section says the run is incomplete, names the unresolved probe, and
// shows the not-measured coverage gap (R-HR-NF-02).
func TestHumanProjectionIncompleteRunNamesUnresolvedProbesAndGaps(t *testing.T) {
	payload := report.Build(unresolvedInput())
	if payload.Run.Completeness != report.CompletenessIncomplete {
		t.Fatalf("the fixture must be an incomplete run, got completeness %q", payload.Run.Completeness)
	}
	projection := renderHuman(t, payload)

	if got := coverageValue(t, projection, "completeness"); got != string(report.CompletenessIncomplete) {
		t.Errorf("coverage completeness = %q, want %q: the run must say it is incomplete", got, report.CompletenessIncomplete)
	}
	if got := coverageValue(t, projection, "unresolved"); got != "tls.truststore" {
		t.Errorf("coverage unresolved = %q, want tls.truststore: the unresolved probe must be named", got)
	}
	if got := coverageValue(t, projection, "not measured"); got != "egress.hub.direct" {
		t.Errorf("coverage not measured = %q, want egress.hub.direct: the not-measured gap must be shown", got)
	}
}

// TestHumanProjectionCompleteRunDoesNotClaimIncompleteness asserts a complete
// run's coverage text does not say the run was incomplete and names no
// unresolved probe, while its not-measured gaps stay visible (R-HR-NF-02).
func TestHumanProjectionCompleteRunDoesNotClaimIncompleteness(t *testing.T) {
	payload := report.Build(fullInput())
	if payload.Run.Completeness != report.CompletenessComplete {
		t.Fatalf("the fixture must be a complete run, got completeness %q", payload.Run.Completeness)
	}
	projection := renderHuman(t, payload)

	if got := coverageValue(t, projection, "completeness"); got != string(report.CompletenessComplete) {
		t.Errorf("coverage completeness = %q, want %q", got, report.CompletenessComplete)
	}
	if got := coverageValue(t, projection, "unresolved"); got != "none" {
		t.Errorf("coverage unresolved = %q, want none: a complete run names no unresolved probe", got)
	}
	if got := coverageValue(t, projection, "not measured"); !strings.Contains(got, "egress.hub.direct") || !strings.Contains(got, "local.sshd") {
		t.Errorf("coverage not measured = %q, want both coverage gaps named", got)
	}
}

// --- raw evidence survives ---------------------------------------------------

// TestHumanProjectionPreservesBlockedTargetBesideReasonAndDetail asserts PRD
// §1.1's blocked hub survives into the human projection: the measured target,
// the verdict, the resolution, the stable reason code and the verbatim failing
// detail are all on the same measurement's row, and the rule that conclusion
// fired from is quoted (R-HR-07).
func TestHumanProjectionPreservesBlockedTargetBesideReasonAndDetail(t *testing.T) {
	payload := report.Build(blockedHubInput())
	projection := renderHuman(t, payload)

	row := projectionRow(t, projection, "egress.hub.direct")
	for _, want := range []string{"203.0.113.10:22", string(probe.ReasonBudgetExpired), blockedHubDetail, string(probe.Fail), string(probe.Measured)} {
		if !strings.Contains(row, want) {
			t.Errorf("the hub row %q does not carry %q beside the reason code", row, want)
		}
	}

	finding := projectionRow(t, projection, "ssh.destination")
	if !strings.Contains(finding, "SSH_DEST_BLOCKED_BY_PUBLIC_SSH") {
		t.Errorf("the ssh.destination finding %q does not quote the rule that fired", finding)
	}
	if !strings.Contains(finding, "203.0.113.10:22") {
		t.Errorf("the ssh.destination finding %q does not name the blocked target", finding)
	}
}

// --- geometry ----------------------------------------------------------------

// TestHumanProjectionPinsTheTableGeometry asserts the exact header lines and one
// full row, so a width, separator or column-order change fails here instead of
// silently reflowing the projection.
func TestHumanProjectionPinsTheTableGeometry(t *testing.T) {
	projection := renderHuman(t, report.Build(fullInput()))

	wantProbeHeader := paddedCell("PROBE", 18) + " " + paddedCell("TARGET", 30) + " " + paddedCell("VERDICT", 13) + " " + paddedCell("RESOLUTION", 12) + " " + paddedCell("REASON", 36) + " " + "DETAIL"
	wantFindingHeader := paddedCell("QUESTION", 20) + " " + paddedCell("RULE", 46) + " " + "CONCLUSION"
	wantTransportHeader := paddedCell("NAME", 20) + " " + paddedCell("VIABLE", 10) + " " + "REASON"
	for _, want := range []string{wantProbeHeader, wantFindingHeader, wantTransportHeader} {
		if !strings.Contains(projection, want+"\n") {
			t.Errorf("the projection does not carry the header line %q", want)
		}
	}

	// One full row: the absent target is the absent marker, never an empty
	// string, and the verdict, resolution, reason and detail follow their columns.
	wantHubRow := paddedCell("egress.hub.direct", 18) + " " + paddedCell("<absent>", 30) + " " + paddedCell("indeterminate", 13) + " " + paddedCell("not_measured", 12) + " " + paddedCell("input_missing_hub", 36) + " " + "no hub address was supplied for this run, so the hub measurement was not made"
	if !strings.Contains(projection, wantHubRow+"\n") {
		t.Errorf("the projection does not carry the pinned hub row %q", wantHubRow)
	}

	// The DETAIL column starts where the five columns before it end.
	const detailColumn = 114
	detail := "linux node (architecture \"aarch64\"): systemd is the running service manager; this run detects and reports the environment and changes nothing"
	row := projectionRow(t, projection, "local.env")
	if at := strings.Index(row, detail); at != detailColumn {
		t.Errorf("the local.env detail starts at column %d, want %d", at, detailColumn)
	}
}

// TestHumanProjectionIndentsMultiLineDetails asserts a detail the probe reported
// on several lines keeps every line verbatim and indents the continuation to the
// DETAIL column, so an embedded newline cannot break the table's shape.
func TestHumanProjectionIndentsMultiLineDetails(t *testing.T) {
	const (
		firstLine  = "region1: the connection was established"
		secondLine = "region2: the connection was refused"
	)
	input := fullInput()
	input.Results = []probe.Result{
		resultOf("egress.cf.443", probe.ProbeEgress, "region1.v2.argotunnel.com:443", 3*time.Millisecond,
			firstLine+"\n"+secondLine,
			passObservation("tcp 443 region1", "region1.v2.argotunnel.com:443", firstLine),
			failObservation("tcp 443 region2", "region2.v2.argotunnel.com:443", probe.ReasonConnRefused, secondLine)),
	}
	input.Diagnosis = diagnosis.Diagnose(input.Results)

	lines := strings.Split(renderHuman(t, report.Build(input)), "\n")
	rowAt := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "egress.cf.443 ") {
			rowAt = i
			break
		}
	}
	if rowAt < 0 {
		t.Fatal("the projection carries no egress.cf.443 row")
	}
	if !strings.Contains(lines[rowAt], firstLine) {
		t.Errorf("the egress.cf.443 row %q does not carry the first detail line %q", lines[rowAt], firstLine)
	}
	if rowAt+1 >= len(lines) {
		t.Fatalf("the egress.cf.443 row has no continuation line for %q", secondLine)
	}
	if want := strings.Repeat(" ", 114) + secondLine; lines[rowAt+1] != want {
		t.Errorf("the second detail line = %q, want %q indented to the DETAIL column", lines[rowAt+1], want)
	}
}

// --- section order and the transports ----------------------------------------

// TestHumanProjectionSectionOrder asserts the projection's fixed order: the tool
// identity and timestamp, the node classification, the probes, the findings, the
// coverage section and the transports.
func TestHumanProjectionSectionOrder(t *testing.T) {
	projection := renderHuman(t, report.Build(fullInput()))
	order := []string{"tool: ", "generated_at: ", "node: ", "node refused: ", "node note: ", "\nPROBES\n", "\nFINDINGS\n", "\nCOVERAGE\n", "\nTRANSPORTS\n"}
	previous := -1
	for _, marker := range order {
		at := strings.Index(projection, marker)
		if at < 0 {
			t.Fatalf("the projection does not carry the section marker %q", marker)
		}
		if at <= previous {
			t.Errorf("the section marker %q appears at %d, before the previous marker at %d: the order is fixed", marker, at, previous)
		}
		previous = at
	}
}

// TestHumanProjectionRendersTransportFeasibility asserts each transport row
// carries its name, viability and reason, and that its requirements and notes
// follow it.
func TestHumanProjectionRendersTransportFeasibility(t *testing.T) {
	projection := renderHuman(t, report.Build(fullInput()))

	row := projectionRow(t, projection, "direct-ssh")
	for _, want := range []string{"direct-ssh", "not viable", "the hub measurement was not made"} {
		if !strings.Contains(row, want) {
			t.Errorf("the direct-ssh row %q does not carry %q", row, want)
		}
	}
	for _, want := range []string{"requires:", "hub_address", "unsatisfied", "supply --hub host[:port] so the hub-directed dial can be measured", "notes:", "hub reachability was not measured"} {
		if !strings.Contains(projection, want) {
			t.Errorf("the transports section does not carry %q", want)
		}
	}
}
