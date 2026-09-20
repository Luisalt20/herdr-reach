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
// Test sources are out of scope for the primitive-construction scan by decision:
// the suite's scripted seams are asserted by the deny-all sentinel tests, and a
// guard that read test files there would have to special-case every fixture that
// names a primitive. They are not out of scope for the constructor invariant: a
// separate scan reads the module's _test.go files and fails if any of them calls
// ProductionSeams, because such a test would dial the real network. The
// counter-cases below prove both scanners can fail — a forbidden construction, a
// test-file constructor call — so a permissive parser cannot pass vacuously.

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

// minTestFiles is the constructor-invariant scan's positive control: the module
// ships test files in every package, so a walk that read fewer than this opened
// no tests and cannot vouch for their contents.
const minTestFiles = 10

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

// parseNonTestSources parses every non-test .go file under root, for the
// primitive-construction scan.
func parseNonTestSources(t *testing.T, root string) []sourceFile {
	return parseSources(t, root, false)
}

// parseTestSources parses every _test.go file under root, for the scan that
// asserts tests never construct the production seam set.
func parseTestSources(t *testing.T, root string) []sourceFile {
	return parseSources(t, root, true)
}

// parseSources parses the .go files under root whose test-ness matches tests,
// sorted by path. A source the parser cannot read fails the guard: a file the
// scanner silently skipped is a file it cannot vouch for.
func parseSources(t *testing.T, root string, tests bool) []sourceFile {
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
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		if strings.HasSuffix(name, "_test.go") != tests {
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
		name, path, ok := restrictedRoot(selector.X, restricted)
		if !ok {
			return true
		}
		if forbiddenSelector(path, selector.Sel.Name) {
			findings = append(findings, finding{
				path: source.path,
				line: source.fset.Position(selector.Sel.Pos()).Line,
				what: name + "." + selector.Sel.Name,
			})
		}
		return true
	})
	return findings
}

// restrictedRoot resolves the restricted import a selector chain is rooted at,
// walking through the parentheses, address-of expressions, composite literals
// and intermediate selectors that can stand between a package name and the
// construction inside it: net.DefaultResolver.LookupHost and
// (&net.Resolver{}).LookupHost both root at net. A chain rooted anywhere else —
// a local variable, a function result — yields no restricted import and is not
// the guard's business. Walking the chain is what reports a nested entry point
// rather than only the outermost selector.
func restrictedRoot(expr ast.Expr, restricted map[string]string) (string, string, bool) {
	switch node := expr.(type) {
	case *ast.Ident:
		path, ok := restricted[node.Name]
		return node.Name, path, ok
	case *ast.SelectorExpr:
		return restrictedRoot(node.X, restricted)
	case *ast.ParenExpr:
		return restrictedRoot(node.X, restricted)
	case *ast.UnaryExpr:
		return restrictedRoot(node.X, restricted)
	case *ast.CompositeLit:
		if node.Type == nil {
			return "", "", false
		}
		return restrictedRoot(node.Type, restricted)
	case *ast.IndexExpr:
		return restrictedRoot(node.X, restricted)
	}
	return "", "", false
}

// forbiddenSelector reports whether naming selector through importPath is one of
// the constructions the guard confines. The prefixes are the design's own:
// net.Dial* covers Dial, DialTimeout, DialUDP and the Dialer value; net.Lookup*
// covers every resolver entry point, and net.DefaultResolver and net.Resolver
// name the process-wide resolver and the resolver type a literal is built from,
// so a lookup rooted at either is caught; net.ListenPacket is the datagram
// listener; tls.Dial* covers both TLS dialing entry points.
func forbiddenSelector(importPath, selector string) bool {
	switch importPath {
	case "net":
		return strings.HasPrefix(selector, "Dial") ||
			strings.HasPrefix(selector, "Lookup") ||
			selector == "ListenPacket" ||
			selector == "DefaultResolver" ||
			selector == "Resolver"
	case "crypto/tls":
		return strings.HasPrefix(selector, "Dial")
	}
	return false
}

// productionSeamsCalls returns every call to the production seam constructor,
// in both the qualified form (probe.ProductionSeams()) and the bare form
// (ProductionSeams(), as an internal-package test would write it), so the guard
// can assert the design's sentence "production seams are constructed in exactly
// one place (internal/probe/real.go, called from main.go); tests never call it"
// as a property of the tree rather than as prose.
func productionSeamsCalls(sources []sourceFile) []finding {
	var calls []finding
	for _, source := range sources {
		ast.Inspect(source.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fun := call.Fun.(type) {
			case *ast.SelectorExpr:
				if fun.Sel.Name != "ProductionSeams" {
					return true
				}
				calls = append(calls, finding{
					path: source.path,
					line: source.fset.Position(fun.Sel.Pos()).Line,
					what: "ProductionSeams",
				})
			case *ast.Ident:
				if fun.Name != "ProductionSeams" {
					return true
				}
				calls = append(calls, finding{
					path: source.path,
					line: source.fset.Position(fun.Pos()).Line,
					what: "ProductionSeams",
				})
			}
			return true
		})
	}
	return calls
}

// productionSeamsLiteral returns the composite literal ProductionSeams returns,
// or reports that the constructor or its literal was not found, so the
// no-execution case can fail on a missing subject rather than pass vacuously.
func productionSeamsLiteral(source sourceFile) (*ast.CompositeLit, bool) {
	for _, declaration := range source.file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "ProductionSeams" || function.Body == nil {
			continue
		}
		var literal *ast.CompositeLit
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if literal != nil {
				return false
			}
			candidate, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if identifier, ok := candidate.Type.(*ast.Ident); ok && identifier.Name == "Seams" {
				literal = candidate
				return false
			}
			return true
		})
		if literal != nil {
			return literal, true
		}
	}
	return nil, false
}

// commandRunnerFieldFindings reports every CommandRunner field in one Seams
// composite literal: an absent field and the explicit nil R1a leaves are the same
// absence, while any runner value is an execution path the no-execution boundary
// forbids (R-HR-02, design §6.2). It is a static check like its siblings: it reads
// the constructor's own literal and narrows the boundary the deny-all sentinel
// tests already exercise.
func commandRunnerFieldFindings(source sourceFile, literal *ast.CompositeLit) []finding {
	var findings []finding
	for _, element := range literal.Elts {
		keyValue, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := keyValue.Key.(*ast.Ident)
		if !ok || key.Name != "CommandRunner" {
			continue
		}
		if value, ok := keyValue.Value.(*ast.Ident); ok && value.Name == "nil" {
			// The explicit nil is the same absence as an omitted field: R1a's
			// boundary is that no runner is wired, and nil wires none.
			continue
		}
		findings = append(findings, finding{
			path: source.path,
			line: source.fset.Position(keyValue.Pos()).Line,
			what: "Seams literal wires CommandRunner",
		})
	}
	return findings
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
	"context"
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
	_ = net.DefaultResolver
)

func offender() {
	_, _ = net.DefaultResolver.LookupHost(context.Background(), "example.com")
	_, _ = (&net.Resolver{}).LookupHost(context.Background(), "example.com")
}
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
		for _, want := range []string{`import "os/exec"`, "net.Dial", "net.LookupHost", "net.ListenPacket", "tls.Dial", "net.DefaultResolver", "net.Resolver"} {
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

	t.Run("the production constructor is never called from a test source", func(t *testing.T) {
		testSources := parseTestSources(t, root)
		if len(testSources) < minTestFiles {
			t.Fatalf("read %d test sources under %s, want at least %d: a scan that opened no tests cannot vouch that tests never construct the production seam set", len(testSources), root, minTestFiles)
		}
		for _, call := range productionSeamsCalls(testSources) {
			t.Errorf("%s calls ProductionSeams from a test source: a test that constructs the production seam set would dial the real network, and tests never call it", call.String())
		}

		// Counter-case: scratch test files that do call it are reported, in both
		// the qualified and the bare form, so neither a scan that read nothing nor
		// a detector that knows only one form can pass vacuously.
		scratch := t.TempDir()
		fixtures := map[string]string{
			"qualified_test.go": "package scratch\n\nimport \"github.com/Luisalt20/herdr-reach/internal/probe\"\n\nvar _ = probe.ProductionSeams()\n",
			"bare_test.go":      "package probe\n\nvar _ = ProductionSeams()\n",
		}
		for name, content := range fixtures {
			if err := os.WriteFile(filepath.Join(scratch, name), []byte(content), 0o600); err != nil {
				t.Fatalf("writing the scratch test fixture %s: %v", name, err)
			}
		}
		reported := make(map[string]bool)
		for _, call := range productionSeamsCalls(parseTestSources(t, scratch)) {
			reported[call.path+" "+call.what] = true
		}
		for _, want := range []string{"qualified_test.go ProductionSeams", "bare_test.go ProductionSeams"} {
			if !reported[want] {
				t.Errorf("the scan did not report %s in a scratch test fixture; a detector that knows only one call form must fail this case", want)
			}
		}
	})

	t.Run("the production seam set carries no command runner", func(t *testing.T) {
		var realSource sourceFile
		for _, source := range sources {
			if source.path == "internal/probe/real.go" {
				realSource = source
				break
			}
		}
		if realSource.file == nil {
			t.Fatalf("internal/probe/real.go was not among the parsed non-test sources, so the no-execution boundary cannot be checked")
		}
		literal, ok := productionSeamsLiteral(realSource)
		if !ok {
			t.Fatalf("ProductionSeams has no Seams composite literal to inspect, so the no-execution boundary cannot be checked")
		}
		for _, reported := range commandRunnerFieldFindings(realSource, literal) {
			t.Errorf("%s wires a CommandRunner into the production seam set; R1a leaves it nil so no code path can execute a third-party binary (R-HR-02, design §6.2)", reported.String())
		}

		// Counter-case: a fixture constructor whose literal wires a runner is
		// reported, so a check that read nothing or inspected nothing cannot pass
		// vacuously.
		scratch := t.TempDir()
		fixture := filepath.Join(scratch, "runner.go")
		const source = `package scratch

func ProductionSeams() Seams {
	return Seams{CommandRunner: runner{}}
}
`
		if err := os.WriteFile(fixture, []byte(source), 0o600); err != nil {
			t.Fatalf("writing the scratch fixture: %v", err)
		}
		fixtureSources := parseNonTestSources(t, scratch)
		var fixtureSource sourceFile
		for _, candidate := range fixtureSources {
			if candidate.path == "runner.go" {
				fixtureSource = candidate
			}
		}
		if fixtureSource.file == nil {
			t.Fatalf("the scratch fixture runner.go was not parsed, so the counter-case cannot run")
		}
		fixtureLiteral, ok := productionSeamsLiteral(fixtureSource)
		if !ok {
			t.Fatalf("the scratch fixture's ProductionSeams literal was not found")
		}
		if findings := commandRunnerFieldFindings(fixtureSource, fixtureLiteral); len(findings) == 0 {
			t.Errorf("the scan did not report the scratch fixture's CommandRunner field; a check that reads nothing must fail this case")
		}
	})
}
