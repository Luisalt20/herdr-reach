package diagnosis_test

// This file is NF-04's static guard: the reasoning layer states network facts and carries no
// per-transport knowledge, so adding a transport never requires touching `internal/diagnosis`
// (design §6.4).
//
// Scope: the package's own production sources. Files ending in `_test.go` are excluded, and that
// exclusion is deliberate: `matrix_test.go` replays PRD §1.1's literal rows, which is acceptance
// evidence rather than reasoning knowledge, and this guard's own token list below is a test
// literal. Reading them would measure the tests' vocabulary instead of the layer's.
//
// Registered probe names are stripped before searching, because a probe name is the measurement
// layer's vocabulary and the reasoning layer must name the probes it reasons over: `egress.cf.7844`
// is how the table says which probe a need belongs to, and the token `7844` is only forbidden as
// transport knowledge. Stripping removes the measurement vocabulary and leaves the words that would
// betray per-transport knowledge.
//
// Three controls make the guard falsifiable, following the project's own "a probe that cannot fail
// is not a probe" discipline. The file-count floor fails a guard that read nothing or read the
// wrong directory. The stripped-name count fails a guard whose stripping never ran: a guard that
// searched the corpus without stripping would fail on a probe name today, and a future corpus
// without one would make the clean result mean nothing. The un-stripped-token control fails a guard
// whose forbidden list no longer matches the corpus at all, which would make the stripping step
// decorative rather than load-bearing.

import (
	"os"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// forbiddenTransportTokens lists the per-transport words that must never reach the reasoning layer:
// the transport identifiers of the R1a candidates and of the tunnel engines they drive. A
// registered probe name may contain one of these as a substring (`egress.cf.7844` contains `7844`),
// which is exactly why the guard strips probe names first.
var forbiddenTransportTokens = []string{
	"direct-ssh",
	"reverse-ssh",
	"cloudflare-tunnel",
	"tailscale",
	"argotunnel",
	"7844",
}

// TestAgnosticProductionSourcesCarryNoTransportKnowledge is R-HR-NF-04's static check over the
// reasoning layer's own sources.
func TestAgnosticProductionSourcesCarryNoTransportKnowledge(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("the diagnosis package directory could not be read: %v", err)
	}

	type source struct {
		name    string
		content string
	}
	var sources []source
	var raw strings.Builder
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
		raw.Write(data)
	}

	// Control 1: a guard that read nothing, or read a different directory, must fail instead of
	// passing silently. The design's own file plan names four reasoning files (facts.go, ids.go,
	// rules.go, findings.go) and the package ships diagnose.go as a fifth, so four is a floor that
	// a broken listing falls under while a later file split does not fail the guard for growing.
	const minProductionFiles = 4
	if len(sources) < minProductionFiles {
		t.Fatalf("the guard read %d production sources, want at least %d: the reasoning layer's sources are the corpus this guard is about, so reading fewer means it saw nothing or saw the wrong directory", len(sources), minProductionFiles)
	}

	// The measurement layer's vocabulary is stripped here. Control 2: if no registered probe name
	// appeared at all, the stripping step never proved anything, and the guard fails loudly instead
	// of reporting a clean corpus it never examined.
	stripped := 0
	for i := range sources {
		for _, entry := range probe.Registry() {
			stripped += strings.Count(sources[i].content, entry.Name)
			sources[i].content = strings.ReplaceAll(sources[i].content, entry.Name, "")
		}
	}
	if stripped == 0 {
		t.Fatalf("no registered probe name was found in the production sources, so the stripping step cannot be load-bearing: the reasoning layer must name the probes it reasons over")
	}

	// Control 3: the un-stripped corpus must contain at least one forbidden token, or stripping
	// would be decoration and a clean result would be vacuous.
	rawCorpus := raw.String()
	rawHasForbidden := false
	for _, token := range forbiddenTransportTokens {
		if strings.Contains(rawCorpus, token) {
			rawHasForbidden = true
			break
		}
	}
	if !rawHasForbidden {
		t.Fatalf("the un-stripped production corpus contains none of the forbidden tokens %v, so the guard's stripping step is not load-bearing and a clean result would prove nothing", forbiddenTransportTokens)
	}

	// The assertion this guard exists for: after the measurement vocabulary is stripped, no
	// per-transport word remains in any production source of the reasoning layer.
	for _, src := range sources {
		for _, token := range forbiddenTransportTokens {
			if strings.Contains(src.content, token) {
				t.Errorf("the production source %s carries the transport token %q after registered probe names were stripped, so per-transport knowledge leaked into the reasoning layer (R-HR-NF-04)", src.name, token)
			}
		}
	}
}
