package report

// This file is the human projection of design §7: it renders the payload Build
// produced to one injected writer, so both projections read one source and
// cannot drift.
//
// It is a projection and nothing else. WriteHuman measures nothing, reads no
// file, consults no clock and writes to nothing but the writer it is handed: it
// copies the payload's own values into text, and the only error it can produce
// is the writer's, which it returns so the caller can exit 2 instead of
// pretending the projection was written.
//
// The text is deliberately tabular. Probe rows, findings and transports are
// tables with fixed column widths, and a probe's detail is free text at the end
// of its row; the geometry is a decision recorded here, so a layout change is an
// edit to this file's constants rather than a side effect of a value's length.

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The projection's geometry. Every width is chosen to fit the closed vocabulary
// it carries — probe names, verdicts, resolutions, reason codes, rule ids and
// requirement kinds are all bounded by their own declarations — so a row cannot
// shift a later column by being longer than its slot. A value that does outgrow
// its column shifts the columns after it rather than being truncated: the raw
// evidence is never cut to fit a layout.
const (
	humanProbeWidth        = 18
	humanTargetWidth       = 30
	humanVerdictWidth      = 13
	humanResolutionWidth   = 12
	humanReasonWidth       = 36
	humanQuestionWidth     = 20
	humanRuleWidth         = 46
	humanTransportWidth    = 20
	humanViabilityWidth    = 10
	humanRequirementWidth  = 22
	humanSatisfactionWidth = 11

	// humanFindingConclusionColumn is where a finding's CONCLUSION column starts.
	humanFindingConclusionColumn = humanQuestionWidth + humanRuleWidth + 2
)

// humanAbsent is how the projection renders a value the payload does not carry.
// The payload distinguishes an absent target (JSON null) from an empty string
// (design D3), and the human projection must keep that distinction: a reader may
// not mistake "no target was measured" for "the target is the empty string".
const humanAbsent = "<absent>"

// humanEmptyList is how a section that carries no rows is rendered, so an empty
// list is visibly empty rather than an ambiguous heading with nothing under it.
const humanEmptyList = "(none)"

// WriteHuman renders one payload as the human projection and writes it to w.
//
// The text is built completely before the single write, so a failed write leaves
// the caller with the error and nothing else: the projection is either reported
// whole or reported as not written. A writer failure is returned, never
// swallowed, because the caller's exit code depends on knowing the human output
// did not reach its stream.
func WriteHuman(w io.Writer, payload Payload) error {
	if _, err := io.WriteString(w, humanProjection(payload)); err != nil {
		return fmt.Errorf("writing the human projection: %w", err)
	}
	return nil
}

// humanProjection builds the whole projection as one string. The section order
// is fixed: the tool identity and timestamp, the node classification, one row
// per probe, the findings, the coverage section and the transports.
func humanProjection(payload Payload) string {
	var out strings.Builder
	humanHeader(&out, payload)
	humanProbes(&out, payload.Probes)
	humanFindings(&out, payload.Findings)
	humanCoverage(&out, payload.Run)
	humanTransports(&out, payload.Transports)
	return out.String()
}

// humanHeader writes the tool identity, the payload's own generated_at — the
// projection holds no clock and prints no timestamp the payload did not carry —
// and the node classification.
func humanHeader(out *strings.Builder, payload Payload) {
	out.WriteString("tool: " + humanText(strings.TrimSpace(payload.Tool.Name+" "+payload.Tool.Version)) + "\n")
	out.WriteString("generated_at: " + humanText(payload.GeneratedAt) + "\n")
	out.WriteString("node: " + humanText(humanNodeIdentity(payload.Node)) + "\n")
	out.WriteString("node refused: " + strconv.FormatBool(payload.Node.Refused) + "\n")
	out.WriteString("node note: " + humanText(payload.Node.Note) + "\n")
}

// humanNodeIdentity renders the classification's platform and architecture as
// one identity, the shape local.env's own target uses. An architecture the run
// never measured is omitted rather than rendered as a dangling separator.
func humanNodeIdentity(node NodeInfo) string {
	if node.Arch == "" {
		return node.Platform
	}
	return node.Platform + "/" + node.Arch
}

// humanProbes writes one row per probe, in the payload's order, and the table's
// header through the same row builder the rows use, so the header cannot drift
// from the columns. A multi-line detail's continuation lines are indented to the
// DETAIL column the rendered row actually reached — an overlong probe name or
// target shifts it right — so the alignment follows the row rather than the
// nominal geometry.
func humanProbes(out *strings.Builder, rows []ProbeRow) {
	out.WriteString("\nPROBES\n")
	header, _ := humanProbeRow("PROBE", "TARGET", "VERDICT", "RESOLUTION", "REASON", "DETAIL")
	out.WriteString(header + "\n")
	if len(rows) == 0 {
		out.WriteString(humanEmptyList + "\n")
		return
	}
	for _, row := range rows {
		lines := humanTextLines(row.Detail)
		line, detailColumn := humanProbeRow(row.Name, humanTarget(row.Target), string(row.Verdict), string(row.Resolution), string(row.Reason), lines[0])
		out.WriteString(line + "\n")
		for _, line := range lines[1:] {
			out.WriteString(strings.Repeat(" ", detailColumn) + line + "\n")
		}
	}
}

// humanFindings writes one row per finding: the question, the rule id that
// fired, and the conclusion verbatim.
func humanFindings(out *strings.Builder, rows []FindingRow) {
	out.WriteString("\nFINDINGS\n")
	out.WriteString(humanFindingRow("QUESTION", "RULE", "CONCLUSION") + "\n")
	if len(rows) == 0 {
		out.WriteString(humanEmptyList + "\n")
		return
	}
	for _, row := range rows {
		lines := humanTextLines(row.Conclusion)
		out.WriteString(humanFindingRow(row.Question, row.Rule, lines[0]) + "\n")
		for _, line := range lines[1:] {
			out.WriteString(strings.Repeat(" ", humanFindingConclusionColumn) + line + "\n")
		}
	}
}

// humanCoverage writes the run's completeness and its two coverage lists. The
// completeness value is the payload's own: an incomplete run says so, and its
// unresolved probes are named right below it, while a complete run names no
// unresolved probe and still shows every not-measured gap (R-HR-NF-02).
func humanCoverage(out *strings.Builder, run RunInfo) {
	out.WriteString("\nCOVERAGE\n")
	out.WriteString("completeness: " + humanText(string(run.Completeness)) + "\n")
	out.WriteString("unresolved: " + humanNameList(run.Unresolved) + "\n")
	out.WriteString("not measured: " + humanNameList(run.NotMeasured) + "\n")
}

// humanNameList renders one coverage list; an empty list is "none" rather than a
// blank line, so the section's shape does not depend on the run.
func humanNameList(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

// humanTransports writes one row per transport: its name, viability and reason,
// followed by its evaluated requirements and its notes. The notes are the
// payload's own strings, so the cloudflared pin statement reaches this
// projection from its single home rather than being restated here.
func humanTransports(out *strings.Builder, rows []TransportInfo) {
	out.WriteString("\nTRANSPORTS\n")
	out.WriteString(humanTransportRow("NAME", "VIABLE", "REASON") + "\n")
	if len(rows) == 0 {
		out.WriteString(humanEmptyList + "\n")
		return
	}
	for _, row := range rows {
		out.WriteString(humanTransportRow(row.Name, humanViability(row.Viable), humanText(row.Reason)) + "\n")
		if len(row.Requires) == 0 {
			out.WriteString("  requires: " + humanEmptyList + "\n")
		} else {
			out.WriteString("  requires:\n")
			for _, requirement := range row.Requires {
				out.WriteString(humanRequirementRow(requirement) + "\n")
			}
		}
		if len(row.Notes) == 0 {
			out.WriteString("  notes: " + humanEmptyList + "\n")
			continue
		}
		out.WriteString("  notes:\n")
		for _, note := range row.Notes {
			out.WriteString("    " + note + "\n")
		}
	}
}

// humanViability renders the strict boolean decision as a word: "viable" or
// "not viable", never a bare true/false a reader could skim past.
func humanViability(viable bool) string {
	if viable {
		return "viable"
	}
	return "not viable"
}

// humanSatisfaction renders one requirement's satisfaction state as a word.
func humanSatisfaction(satisfied bool) string {
	if satisfied {
		return "satisfied"
	}
	return "unsatisfied"
}

// humanTarget renders one probe's measured target: the payload's own string when
// it carried one, the absent marker when it did not.
func humanTarget(target *string) string {
	if target == nil {
		return humanAbsent
	}
	return *target
}

// humanText renders a value the payload may not carry. It never turns an empty
// string into an empty cell, because an empty cell is indistinguishable from a
// missing one in a table.
func humanText(value string) string {
	if value == "" {
		return humanAbsent
	}
	return value
}

// humanTextLines splits one free-text value into the lines the projection
// prints: the first completes the row and every later line is a continuation the
// caller indents to its column. The value is split on newlines and never
// reworded, so a probe's verbatim multi-line detail keeps every line.
func humanTextLines(value string) []string {
	return strings.Split(humanText(value), "\n")
}

// humanProbeRow builds one probe-table line through humanRow, so the header and
// the rows share one column shape. It also reports the column where the DETAIL
// cell begins in the rendered line, which is what the continuation lines of a
// multi-line detail are indented to; an overlong earlier cell shifts that column
// right, exactly as the rendered row shows.
func humanProbeRow(cells ...string) (string, int) {
	return humanRow(cells, humanProbeWidth, humanTargetWidth, humanVerdictWidth, humanResolutionWidth, humanReasonWidth)
}

// humanFindingRow builds one finding-table line through humanRow.
func humanFindingRow(cells ...string) string {
	line, _ := humanRow(cells, humanQuestionWidth, humanRuleWidth)
	return line
}

// humanTransportRow builds one transport-table line through humanRow.
func humanTransportRow(cells ...string) string {
	line, _ := humanRow(cells, humanTransportWidth, humanViabilityWidth)
	return line
}

// humanRequirementRow builds one requirement line: the kind and the satisfaction
// state padded to their columns, then the detail verbatim.
func humanRequirementRow(requirement RequirementRow) string {
	return fmt.Sprintf("    %-*s %-*s %s",
		humanRequirementWidth, string(requirement.Kind),
		humanSatisfactionWidth, humanSatisfaction(requirement.Satisfied),
		humanText(requirement.Detail))
}

// humanRow joins one table row and reports where its last cell begins: every cell
// before the last is padded to its column width and separated by one space, and
// the last cell is appended verbatim. An empty last cell leaves no trailing
// padding behind it, so a row with nothing in its final column does not end in
// invisible whitespace.
//
// The reported column is the rune column of the last cell's first character in
// the line this function just built. Returning it from beside the rendering that
// decides it is what keeps a continuation-line indent from drifting from the
// actual row: a cell that outgrows its width shifts the column right and is
// never truncated to keep the geometry nominal.
//
// Padding counts runes, not bytes, so a value outside ASCII still occupies one
// column per visible character.
func humanRow(cells []string, widths ...int) (string, int) {
	var line strings.Builder
	last := len(cells) - 1
	column := 0
	lastColumn := 0
	for i, cell := range cells {
		if i > 0 {
			line.WriteByte(' ')
			column++
		}
		if i == last {
			lastColumn = column
		}
		line.WriteString(cell)
		runes := utf8.RuneCountInString(cell)
		column += runes
		if i == last {
			break
		}
		width := widths[i]
		if runes > width {
			width = runes
		}
		for pad := width - runes; pad > 0; pad-- {
			line.WriteByte(' ')
			column++
		}
	}
	if cells[last] == "" {
		return strings.TrimRight(line.String(), " "), lastColumn
	}
	return line.String(), lastColumn
}
