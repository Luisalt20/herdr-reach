package probe_test

// This file is design §6.2's third level of the no-egress proof: a static guard
// over the module's own non-test Go sources.
//
// The guard asserts that the direct construction of real network and process
// primitives — net.Dial*, net.Lookup*, net.ListenPacket, tls.Dial* and the
// os/exec package — appears only in the two files allowed to hold it:
// internal/probe/real.go, the production seam set, and cmd/herdr-reach/main.go,
// which wires it. Whatever it reports elsewhere belongs in real.go; the
// allow-list is not to be widened to make a report go away.
//
// It is a static check and is recorded as one. It narrows RG-7 and RG-10 — a
// build now fails when a real primitive appears in the wrong file — and it does
// not replace the CI workflow R11 owns: that workflow keeps its own selector and
// its own sentinel tests, and this test runs under the unit suite, so dropping
// the CI job does not drop this guard.
//
// Test sources are out of scope by decision: the suite's scripted seams are
// asserted by the deny-all sentinel tests, and a guard that read test files would
// have to special-case every fixture that names a primitive. The counter-case
// below proves the scanner can fail on a forbidden construction, so a permissive
// parser cannot pass vacuously.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// allowedProductionFiles are the only non-test sources where a real network or
// process primitive may be constructed, repository-relative and slash-separated.
// internal/probe/real.go is the production seam set itself; cmd/herdr-reach/main.go
// is the entrypoint that builds it.
var allowedProductionFiles = []string{
	"internal/probe/real.go",
	"cmd/herdr-reach/main.go",
}

// entrypointFile is where the production seam constructor is called. Design §6.2
// states it as "production seams are constructed in exactly one place
// (internal/probe/real.go, called from main.go); tests never call it", and this
// guard asserts the call half of that sentence structurally.
const entrypointFile = "cmd/herdr-reach/main.go"

// minSourceFiles is the positive control's floor: the module's non-test sources
// number in the dozens, so a walk that read fewer than this has failed to read
// the module and must fail rather than pass vacuously. The floor is deliberately
// far below the real count — it guards against a broken walk, not against the
// module shrinking.
const minSourceFiles = 25

// sourceFile is one parsed non-test source, with the file set its positions are
// resolved against.
type sourceFile struct {
	// path is the file's path relative to the scanned root, slash-separated, so
	// a finding names the repository location a reviewer needs.
	path string
	file *ast.File
	fset *token.FileSet
}

// finding is one construction the guard reports: where it is and what it is.
type finding struct {
	path string
	line int
	what string
}

// String renders the finding as the "file:line: what" form a failure message
// uses, so a report is actionable without opening the scanner.
func (f finding) String() string {
	return fmt.Sprintf("%s:%d: %s", f.path, f.line, f.what)
}

// moduleRoot walks up from the test's working directory — the package directory
// under `go test` — to the directory holding go.mod, which is the root the guard
// scans. The walk keeps the guard independent of where the package sits in the
// tree and of how many levels it has.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolving the working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("walked to the filesystem root without finding go.mod")
		}
		dir = parent
	}
}

// parseNonTestSources parses every non-test .go file under root. A source the
// parser cannot read fails the guard: a file the scanner silently skipped is a
// file it cannot vouch for.
func parseNonTestSources(t *testing.T, root string) []sourceFile {
	t.Helper()
	var sources []sourceFile
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			// A vendored tree is not this module's own source, and .git holds no
			// source at all.
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", relative, err)
		}
		sources = append(sources, sourceFile{path: filepath.ToSlash(relative), file: parsed, fset: fset})
		return nil
	})
	if err != nil {
		t.Fatalf("reading the sources under %s: %v", root, err)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].path < sources[j].path })
	return sources
}

// forbiddenFindings reports every forbidden construction in the sources that are
// not on the allow-list. Allowed files are skipped, not exempted from parsing:
// the positive controls assert they were read, so the guard can name the files
// the confinement holds for.
func forbiddenFindings(sources []sourceFile, allowed []string) []finding {
	permitted := make(map[string]bool, len(allowed))
	for _, path := range allowed {
		permitted[path] = true
	}
	var findings []finding
	for _, source := range sources {
		if permitted[source.path] {
			continue
		}
		findings = append(findings, forbiddenIn(source)...)
	}
	return findings
}

// forbiddenIn reports every forbidden construction in one parsed file.
//
// The scan is on the syntax tree, never on text: a comment that mentions
// `net.Dial` is not a construction, and a construction cannot hide in a string.
// Imports are read first so a restricted package is recognised under the local
// name the file actually gave it — an alias does not evade the guard — and a
// dot-import is reported itself, because it makes construction unnameable in the
// file that would be doing it.
func forbiddenIn(source sourceFile) []finding {
	restricted := make(map[string]string) // local import name -> import path
	var findings []finding

	for _, spec := range source.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		switch path {
		case "os/exec":
			findings = append(findings, finding{
				path: source.path,
				line: source.fset.Position(spec.Pos()).Line,
				what: `import "os/exec"`,
			})
		case "net", "crypto/tls":
			name := path
			if path == "crypto/tls" {
				name = "tls"
			}
			if spec.Name != nil {
				name = spec.Name.Name
			}
			switch name {
			case ".":
				findings = append(findings, finding{
					path: source.path,
					line: source.fset.Position(spec.Pos()).Line,
					what: fmt.Sprintf("dot-import of %q hides construction", path),
				})
			case "_":
				// A blank import performs no construction.
			default:
				restricted[name] = path
			}
		}
	}

	ast.Inspect(source.file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		path, ok := restricted[identifier.Name]
		if !ok {
			return true
		}
		if forbiddenSelector(path, selector.Sel.Name) {
			findings = append(findings, finding{
				path: source.path,
				line: source.fset.Position(selector.Sel.Pos()).Line,
				what: identifier.Name + "." + selector.Sel.Name,
			})
		}
		return true
	})
	return findings
}

// forbiddenSelector reports whether naming selector through importPath is one of
// the constructions the guard confines. The prefixes are the design's own:
// net.Dial* covers Dial, DialTimeout, DialUDP and the Dialer value; net.Lookup*
// covers every resolver entry point; net.ListenPacket is the datagram listener;
// tls.Dial* covers both TLS dialing entry points.
func forbiddenSelector(importPath, selector string) bool {
	switch importPath {
	case "net":
		return strings.HasPrefix(selector, "Dial") ||
			strings.HasPrefix(selector, "Lookup") ||
			selector == "ListenPacket"
	case "crypto/tls":
		return strings.HasPrefix(selector, "Dial")
	}
	return false
}

// productionSeamsCalls returns every call to the production seam constructor, so
// the guard can assert the design's sentence "production seams are constructed in
// exactly one place (internal/probe/real.go, called from main.go); tests never
// call it" as a property of the tree rather than as prose.
func productionSeamsCalls(sources []sourceFile) []finding {
	var calls []finding
	for _, source := range sources {
		ast.Inspect(source.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "ProductionSeams" {
				return true
			}
			calls = append(calls, finding{
				path: source.path,
				line: source.fset.Position(selector.Sel.Pos()).Line,
				what: "ProductionSeams",
			})
			return true
		})
	}
	return calls
}

// TestNoRealNetworkConstructionGuard is design §6.2 level 3: the module's own
// sources are the evidence, and the two allowed files are the only place a real
// primitive may be built. The subtests are ordered so a broken scanner fails a
// positive control before it can pass the confinement check vacuously.
func TestNoRealNetworkConstructionGuard(t *testing.T) {
	root := moduleRoot(t)
	sources := parseNonTestSources(t, root)

	t.Run("positive control: the module's non-test sources were read", func(t *testing.T) {
		if len(sources) < minSourceFiles {
			t.Fatalf("read %d non-test sources under %s, want at least %d: a scanner that read nothing must fail rather than pass vacuously", len(sources), root, minSourceFiles)
		}
	})

	t.Run("positive control: both allowed production files were read", func(t *testing.T) {
		read := make(map[string]bool, len(sources))
		for _, source := range sources {
			read[source.path] = true
		}
		for _, path := range allowedProductionFiles {
			if !read[path] {
				t.Errorf("%s was not among the parsed non-test sources, so the guard cannot vouch for the file it allows", path)
			}
		}
	})

	t.Run("forbidden constructions stay in the production files", func(t *testing.T) {
		for _, reported := range forbiddenFindings(sources, allowedProductionFiles) {
			t.Errorf("%s names %s outside the production files; move it into internal/probe/real.go (or, for the entrypoint's wiring, cmd/herdr-reach/main.go) instead of widening the allow-list", reported.String(), reported.what)
		}
	})

	t.Run("counter-case: a forbidden construction outside the allowed files is reported", func(t *testing.T) {
		// The fixture is written to a temporary directory outside the repository
		// and removed by the test framework; nothing is added to the module to
		// make the scanner fail on purpose.
		scratch := t.TempDir()
		fixture := filepath.Join(scratch, "offender.go")
		const source = `package scratch

import (
	"crypto/tls"
	"net"
	"os/exec"
)

var (
	_ = net.Dial
	_ = net.LookupHost
	_ = net.ListenPacket
	_ = tls.Dial
	_ = exec.Command
)
`
		if err := os.WriteFile(fixture, []byte(source), 0o600); err != nil {
			t.Fatalf("writing the scratch fixture: %v", err)
		}

		findings := forbiddenFindings(parseNonTestSources(t, scratch), allowedProductionFiles)
		reported := make(map[string]bool, len(findings))
		for _, found := range findings {
			if found.path != "offender.go" {
				t.Errorf("a finding names %s, want the scratch fixture offender.go", found.path)
			}
			reported[found.what] = true
		}
		for _, want := range []string{`import "os/exec"`, "net.Dial", "net.LookupHost", "net.ListenPacket", "tls.Dial"} {
			if !reported[want] {
				t.Errorf("the scanner did not report %s in the scratch fixture; a permissive parser must fail this case", want)
			}
		}
	})

	t.Run("the production constructor is called only from the entrypoint", func(t *testing.T) {
		calls := productionSeamsCalls(sources)
		if len(calls) == 0 {
			t.Fatalf("no non-test source calls ProductionSeams, so the production seam set is never constructed")
		}
		for _, call := range calls {
			if call.path != entrypointFile {
				t.Errorf("%s calls ProductionSeams; the production seam set is constructed only in %s, and tests never call it", call.String(), entrypointFile)
			}
		}
	})
}
