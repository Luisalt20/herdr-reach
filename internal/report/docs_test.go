package report_test

// This file is the drift test of design §5.4 and §7: docs/diagnosis-report.md is
// the repository-facing contract document, and its reason-code and rule-id
// tables are asserted as sets against the code's own declarations, in both
// directions. A code or rule id added to the constants without a document row
// fails here, and a document row without a constant fails too, so the document
// cannot drift away from the vocabulary it documents.
//
// The exit-code constants live in internal/doctor, which imports this package;
// their table is therefore asserted in internal/doctor/doctor_test.go (PR 18)
// rather than here, because importing that package from this one would invert
// the dependency direction.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// diagnosisDocRelativePath is the contract document as the repository root
// names it. The test binary's working directory is this package's directory, so
// the document is read two levels up from it.
const diagnosisDocRelativePath = "docs/diagnosis-report.md"

// The positive controls' floors. Each is the table's size at the time this case
// landed: a parse that found nothing, or only a fragment of a table, falls under
// its floor and fails loudly instead of passing as an empty set comparison.
const (
	reasonCodeTableFloor = 28
	ruleIDTableFloor     = 61
)

// readDiagnosisDoc reads the contract document from the repository root, failing
// the case when it is absent or empty: a missing document must fail rather than
// turn every set comparison into a comparison with nothing.
func readDiagnosisDoc(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(diagnosisDocRelativePath)))
	if err != nil {
		t.Fatalf("%s could not be read from the repository root: %v", diagnosisDocRelativePath, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		t.Fatalf("%s is empty", diagnosisDocRelativePath)
	}
	return string(data)
}

// docTableFirstColumn returns the first column of the first markdown table after
// the exact heading line, with the inline-code backticks stripped. The table's
// header row and its separator row are skipped; the table ends at the first
// non-table line after it started.
func docTableFirstColumn(t *testing.T, document, heading string) []string {
	t.Helper()
	lines := strings.Split(document, "\n")

	headingAt := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			headingAt = i
			break
		}
	}
	if headingAt < 0 {
		t.Fatalf("%s has no heading %q", diagnosisDocRelativePath, heading)
	}

	var (
		rows      []string
		tableRows int
	)
	for _, line := range lines[headingAt+1:] {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			if tableRows > 0 {
				break
			}
			continue
		}
		cells := strings.Split(trimmed, "|")
		if len(cells) < 3 {
			continue
		}
		cell := strings.Trim(strings.TrimSpace(cells[1]), "`")
		tableRows++
		if tableRows == 1 || strings.Trim(cell, "-: ") == "" {
			// The header row, then the separator row.
			continue
		}
		if cell == "" {
			t.Fatalf("a row of the table under %q has an empty first column", heading)
		}
		rows = append(rows, cell)
	}
	return rows
}

// assertSameSet fails when want and got differ in either direction and prints
// both differences, so the document can be corrected mechanically.
func assertSameSet(t *testing.T, what string, want, got []string) {
	t.Helper()
	missing, extra := setDifference(want, got), setDifference(got, want)
	if len(missing) == 0 && len(extra) == 0 {
		return
	}
	t.Errorf("%s does not match the code's declarations:\nmissing from the document: %v\nextra in the document:     %v", what, missing, extra)
}

// setDifference returns the entries of a that are not in b, in a's order.
func setDifference(a, b []string) []string {
	present := make(map[string]bool, len(b))
	for _, entry := range b {
		present[entry] = true
	}
	var difference []string
	for _, entry := range a {
		if !present[entry] {
			difference = append(difference, entry)
		}
	}
	return difference
}

// TestDocsReasonCodeTableMatchesTheClosedSet asserts the document's reason-code
// table is exactly probe.AllReasonCodes(), in both directions (design §3.5).
func TestDocsReasonCodeTableMatchesTheClosedSet(t *testing.T) {
	document := readDiagnosisDoc(t)
	documented := docTableFirstColumn(t, document, "## Reason codes")
	if len(documented) < reasonCodeTableFloor {
		t.Fatalf("the reason-code table yielded %d entries, below the floor of %d: a parse that found nothing must not pass silently", len(documented), reasonCodeTableFloor)
	}

	codes := probe.AllReasonCodes()
	want := make([]string, 0, len(codes))
	for _, code := range codes {
		want = append(want, string(code))
	}
	assertSameSet(t, "the reason-code table", want, documented)
}

// TestDocsRuleIDTableMatchesTheVocabulary asserts the document's rule-id table
// is exactly diagnosis.AllRuleIDs(), in both directions (design §3.6).
func TestDocsRuleIDTableMatchesTheVocabulary(t *testing.T) {
	document := readDiagnosisDoc(t)
	documented := docTableFirstColumn(t, document, "## Rule ids")
	if len(documented) < ruleIDTableFloor {
		t.Fatalf("the rule-id table yielded %d entries, below the floor of %d: a parse that found nothing must not pass silently", len(documented), ruleIDTableFloor)
	}

	assertSameSet(t, "the rule-id table", diagnosis.AllRuleIDs(), documented)
}
