package transport_test

// This file is R-HR-16's suite for the cloudflared pin statement: the content the research record
// establishes, the claims the statement must not make, the proof that the exact statement has
// exactly one home in the repository's code and shipped docs, and the proof that the feasibility
// notes render that one source rather than a copy of their own.

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/transport"
)

// versionToken matches a dotted release number, so the content case can assert exactly which
// versions the statement names instead of trusting a substring check.
var versionToken = regexp.MustCompile(`\b20\d{2}\.\d+\.\d+\b`)

// TestPinNoteStatesOnlyWhatTheResearchRecordEstablishes is R-HR-16's content case: the statement
// names issue #1673 with its state, names version 2026.6.0 only, states that later releases'
// behaviour is unknown, cites release 2026.5.1's published SHA256 checksums, and addresses the
// install advice to the reader.
func TestPinNoteStatesOnlyWhatTheResearchRecordEstablishes(t *testing.T) {
	statement := transport.PinNote()
	if strings.TrimSpace(statement) == "" {
		t.Fatal("PinNote() returned an empty statement")
	}

	wants := []struct {
		why  string
		text string
	}{
		{"it names the upstream issue", "#1673"},
		{"it states the report is open", "open"},
		{"it states the report is uncommented", "uncommented"},
		{"it names the version the report names", "2026.6.0"},
		{"it says the report names that version only", "names that version only"},
		{"it states later releases' behaviour is unknown", "later releases"},
		{"it states that the behaviour is unknown", "unknown"},
		{"it cites the release whose checksums are published", "2026.5.1"},
		{"it cites published checksums", "SHA256 checksums"},
		{"it advises pinning to the last release the report does not implicate", "last release the report does not implicate"},
		{"it advises verifying the checksum before installing", "verify its published checksum before installing"},
	}
	for _, want := range wants {
		if !strings.Contains(statement, want.text) {
			t.Errorf("the statement does not carry %q (%s): %s", want.text, want.why, statement)
		}
	}

	// "Names 2026.6.0 only" is a version-set property, not a phrase: the report's implicated
	// version and the release whose checksums are cited are the only versions the statement names.
	versions := versionToken.FindAllString(statement, -1)
	slices.Sort(versions)
	if want := []string{"2026.5.1", "2026.6.0"}; !slices.Equal(versions, want) {
		t.Errorf("the statement names the versions %v, want exactly %v: the report implicates 2026.6.0 only and 2026.5.1 is cited for its published checksums", versions, want)
	}
	if got := strings.Count(statement, "2026.6.0"); got != 1 {
		t.Errorf("the statement names 2026.6.0 %d times, want once: it is the one version the report implicates", got)
	}
}

// TestPinNoteClaimsNoRangeFixOrAction is R-HR-16's prohibition case. Each assertion is the
// affirmative claim the statement must not make; the past-tense action words cannot appear at all,
// because the statement's only action sentence is advice addressed to the reader and its final
// sentence disclaims any action of the tool's own.
func TestPinNoteClaimsNoRangeFixOrAction(t *testing.T) {
	statement := transport.PinNote()

	rangeClaims := []string{
		"and later", "or later", "onwards", "and above", "and newer",
		"from 2026.6.0", "2026.6.0+", "affected range", "range of versions",
		"all releases", "every release", "versions after 2026.6.0",
	}
	for _, claim := range rangeClaims {
		if strings.Contains(statement, claim) {
			t.Errorf("the statement carries the version-range claim %q: the report establishes one implicated version, not a range", claim)
		}
	}

	fixClaims := []string{"fix", "corrected", "resolved", "patched"}
	for _, claim := range fixClaims {
		if strings.Contains(strings.ToLower(statement), claim) {
			t.Errorf("the statement carries the fix claim %q: no release note documents a fix", claim)
		}
	}

	// Past-tense action words are the tense a claim about this run would need. The advice uses the
	// present tense ("pin", "verify", "installing") and the final sentence says the tool does not
	// act, so none of these may appear.
	actionClaims := []string{"installed", "verified", "pinned", "applied", "written", "changed"}
	for _, claim := range actionClaims {
		if strings.Contains(strings.ToLower(statement), claim) {
			t.Errorf("the statement carries %q, which would claim an action this slice did not take", claim)
		}
	}
}

// TestPinNoteHasExactlyOneHomeInCodeAndShippedDocs is R-HR-16's single-home proof. The scan is
// scoped and controlled, and it asserts the exact statement, never a fragment: README.md and
// PRD.md legitimately describe the upstream report as the project's prose, and counting a fragment
// like `#1673` would count their prose instead of the tool's statement.
//
// Scope: every non-test .go file and every .md file under the repository root, excluding .git,
// openspec/** (the change's own artifacts carry the research record) and odd/** (the ODD tracking
// directory); test files are excluded because a suite quotes the text to assert it. The controls
// fail a scan that read implausibly few files, and a scan that found the statement zero or two
// times, so no broken scan can pass silently.
func TestPinNoteHasExactlyOneHomeInCodeAndShippedDocs(t *testing.T) {
	statement := transport.PinNote()
	root := filepath.Join("..", "..")

	var (
		filesRead   int
		occurrences int
		carriers    []string
	)
	documents := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "openspec", "odd":
				return fs.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if ext := filepath.Ext(name); ext != ".go" && ext != ".md" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		filesRead++
		switch relative {
		case "README.md", "PRD.md":
			documents[relative] = string(content)
		}
		if found := strings.Count(string(content), statement); found > 0 {
			occurrences += found
			carriers = append(carriers, relative)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("the repository could not be scanned: %v", err)
	}

	// Positive control: a scan that read nothing, or read the wrong directory, must fail instead of
	// passing silently. The repository ships more than twenty code and doc files in scope, so a
	// floor of twenty is a floor a broken listing falls under while later files can grow past it.
	const minScannedFiles = 20
	if filesRead < minScannedFiles {
		t.Fatalf("the scan read %d files, below the floor of %d: a scan that read nothing must fail rather than pass silently", filesRead, minScannedFiles)
	}

	// Positive control: the scan must have read the two documents whose prose carries the research
	// record, or its exclusion of them from the count proves nothing. Their prose names the issue
	// fragment but is not the tool's statement, and the statement must not appear in them.
	for _, name := range []string{"README.md", "PRD.md"} {
		content, read := documents[name]
		if !read {
			t.Fatalf("the scan did not read %s, so its exclusion from the statement count proves nothing", name)
		}
		if !strings.Contains(content, "#1673") {
			t.Errorf("%s does not contain the issue fragment #1673, so the scan cannot show the fragment is not what it counts", name)
		}
		if strings.Contains(content, statement) {
			t.Errorf("%s contains the exact statement: the tool's statement must live in pin_note.go alone", name)
		}
	}

	// The exact statement occurs exactly once, in pin_note.go. Zero occurrences fail too: a scan
	// that found nothing must not pass silently.
	if occurrences != 1 {
		t.Fatalf("the exact statement occurs %d times across %v, want exactly once in internal/transport/pin_note.go", occurrences, carriers)
	}
	if want := []string{filepath.Join("internal", "transport", "pin_note.go")}; !slices.Equal(carriers, want) {
		t.Errorf("the statement's one home is %v, want %v", carriers, want)
	}
}

// TestPinNoteReachesFeasibilityNotesFromTheSingleSource asserts the two projections cannot drift:
// the cloudflare-tunnel notes carry exactly the statement PinNote() returns, as one note, so the
// machine-readable payload (the feasibility rows, which PR 16 maps) and the human projection (which
// PR 17 renders from the same rows) both read the single home.
func TestPinNoteReachesFeasibilityNotesFromTheSingleSource(t *testing.T) {
	d := diagnosis.Diagnose([]probe.Result{cfEdgeReachable()})
	registry, rows := registryRows(t, d)
	row := rowFor(t, registry, rows, "cloudflare-tunnel")
	if !slices.Contains(row.Notes, transport.PinNote()) {
		t.Fatalf("the cloudflare-tunnel notes %q do not carry PinNote() verbatim: both projections must render the single source", row.Notes)
	}
	count := 0
	for _, note := range row.Notes {
		if note == transport.PinNote() {
			count++
		}
	}
	if count != 1 {
		t.Errorf("the statement appears %d times among the notes, want exactly once: a second copy in the notes is a second home", count)
	}
}
