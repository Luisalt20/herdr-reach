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

// seamsCommandRunnerIndex returns the index of the CommandRunner field in the
// Seams struct declared in one parsed source, reading the declaration's field
// order rather than hard-coding it. The boolean is false when the struct or the
// field cannot be resolved, which the no-runner case treats as a broken field
// lookup rather than a reason to skip the positional check.
func seamsCommandRunnerIndex(source sourceFile) (int, bool) {
	for _, declaration := range source.file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.TYPE {
			continue
		}
		for _, spec := range general.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != "Seams" {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			index := 0
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					// An embedded field occupies one positional slot.
					index++
					continue
				}
				for _, name := range field.Names {
					if name.Name == "CommandRunner" {
						return index, true
					}
					index++
				}
			}
		}
	}
	return 0, false
}

// isExplicitNil reports whether an expression is the nil identifier.
func isExplicitNil(expr ast.Expr) bool {
	identifier, ok := expr.(*ast.Ident)
	return ok && identifier.Name == "nil"
}

// noRunnerFindings reports every way ProductionSeams can leave its no-runner
// boundary: a return that is not a direct Seams composite literal (an indirect
// return is exactly what a static check cannot follow), a keyed CommandRunner
// field set to anything but the explicit nil, and a positional CommandRunner
// slot set to anything but the explicit nil. The positional slot's index is read
// from the Seams struct declaration, so a reordered struct cannot silently move
// the check to the wrong element. It protects R-HR-02 and design §6.2's
// no-execution boundary; it narrows that boundary rather than proving it, and
// the deny-all sentinel tests remain the behavioural half.
func noRunnerFindings(source sourceFile, commandRunnerIndex int) []finding {
	var findings []finding
	found := false
	for _, declaration := range source.file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "ProductionSeams" || function.Body == nil {
			continue
		}
		found = true
		ast.Inspect(function.Body, func(node ast.Node) bool {
			returns, ok := node.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			if len(returns.Results) != 1 {
				findings = append(findings, finding{
					path: source.path,
					line: source.fset.Position(returns.Pos()).Line,
					what: "ProductionSeams returns without a single value",
				})
				return true
			}
			literal, ok := returns.Results[0].(*ast.CompositeLit)
			if !ok {
				findings = append(findings, finding{
					path: source.path,
					line: source.fset.Position(returns.Pos()).Line,
					what: "ProductionSeams returns an indirect value",
				})
				return true
			}
			if identifier, ok := literal.Type.(*ast.Ident); !ok || identifier.Name != "Seams" {
				findings = append(findings, finding{
					path: source.path,
					line: source.fset.Position(returns.Pos()).Line,
					what: "ProductionSeams returns a non-Seams literal",
				})
				return true
			}
			findings = append(findings, commandRunnerFieldFindings(source, literal, commandRunnerIndex)...)
			return true
		})
	}
	if !found {
		findings = append(findings, finding{
			path: source.path,
			what: "ProductionSeams was not found",
		})
	}
	return findings
}

// commandRunnerFieldFindings reports every CommandRunner field or slot of one
// Seams composite literal that is not the explicit nil R1a leaves: a keyed field
// is matched by name, and a positional slot is matched by the index read from the
// struct declaration. An absent field and the explicit nil are the same absence,
// while any runner value is an execution path the no-execution boundary forbids
// (R-HR-02, design §6.2). It is a static check like its siblings: it narrows the
// boundary the deny-all sentinel tests already exercise.
func commandRunnerFieldFindings(source sourceFile, literal *ast.CompositeLit, commandRunnerIndex int) []finding {
	var findings []finding
	position := 0
	for _, element := range literal.Elts {
		keyValue, keyed := element.(*ast.KeyValueExpr)
		if keyed {
			key, ok := keyValue.Key.(*ast.Ident)
			if !ok || key.Name != "CommandRunner" {
				continue
			}
			if isExplicitNil(keyValue.Value) {
				// The explicit nil is the same absence as an omitted field: R1a's
				// boundary is that no runner is wired, and nil wires none.
				continue
			}
			findings = append(findings, finding{
				path: source.path,
				line: source.fset.Position(keyValue.Pos()).Line,
				what: "Seams literal wires CommandRunner",
			})
			continue
		}
		if position == commandRunnerIndex && !isExplicitNil(element) {
			findings = append(findings, finding{
				path: source.path,
				line: source.fset.Position(element.Pos()).Line,
				what: "Seams literal wires CommandRunner at its positional slot",
			})
		}
		position++
	}
	return findings
}

// writeFixture writes one scratch source into path, failing the case when it
// cannot be written. The fixture lives outside the repository and is removed by
// the test framework.
func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing the scratch fixture %s: %v", path, err)
	}
}

// onlySource returns the one parsed source named name under root, failing the
// case when it was not parsed, so a counter-case cannot pass on a fixture that
// was never read.
func onlySource(t *testing.T, root, name string) sourceFile {
	t.Helper()
	for _, source := range parseNonTestSources(t, root) {
		if source.path == name {
			return source
		}
	}
	t.Fatalf("the scratch fixture %s was not parsed", name)
	return sourceFile{}
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
		var realSource, seamsSource sourceFile
		for _, source := range sources {
			switch source.path {
			case "internal/probe/real.go":
				realSource = source
			case "internal/probe/seams.go":
				seamsSource = source
			}
		}
		if realSource.file == nil {
			t.Fatalf("internal/probe/real.go was not among the parsed non-test sources, so the no-execution boundary cannot be checked")
		}
		if seamsSource.file == nil {
			t.Fatalf("internal/probe/seams.go was not among the parsed non-test sources, so the CommandRunner field cannot be located")
		}
		commandRunnerIndex, resolved := seamsCommandRunnerIndex(seamsSource)
		if !resolved {
			t.Fatalf("the CommandRunner field could not be resolved from the Seams struct in internal/probe/seams.go, so the positional form cannot be checked")
		}
		for _, reported := range noRunnerFindings(realSource, commandRunnerIndex) {
			t.Errorf("%s leaves the production seam set able to execute a command; R1a wires no CommandRunner, so no code path can execute a third-party binary (R-HR-02, design §6.2)", reported.String())
		}

		// Control: the resolver fails on a struct without the field, so a broken
		// field lookup cannot silently pass the positional check.
		controlScratch := t.TempDir()
		writeFixture(t, filepath.Join(controlScratch, "seams.go"), "package scratch\n\ntype Seams struct {\n\tDialer Dialer\n}\n")
		if _, resolved := seamsCommandRunnerIndex(onlySource(t, controlScratch, "seams.go")); resolved {
			t.Errorf("the resolver reported a CommandRunner index for a struct without the field; a broken field lookup must fail the control")
		}

		// Counter-cases: the keyed form, the positional form and the indirect form
		// (built earlier and returned as an identifier) are all reported, so a
		// check that reads nothing or follows no return cannot pass vacuously.
		cases := []struct {
			name   string
			source string
		}{
			{
				name: "keyed",
				source: `package scratch

func ProductionSeams() Seams {
	return Seams{CommandRunner: runner{}}
}
`,
			},
			{
				name: "positional",
				source: `package scratch

func ProductionSeams() Seams {
	return Seams{dialer{}, resolver{}, verifier{}, packetDialer{}, runner{}, clock{}, fs{}, platform{}}
}
`,
			},
			{
				name: "indirect",
				source: `package scratch

func ProductionSeams() Seams {
	base := Seams{CommandRunner: nil}
	base.CommandRunner = runner{}
	return base
}
`,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				fixtureScratch := t.TempDir()
				writeFixture(t, filepath.Join(fixtureScratch, "runner.go"), tc.source)
				if findings := noRunnerFindings(onlySource(t, fixtureScratch, "runner.go"), commandRunnerIndex); len(findings) == 0 {
					t.Errorf("the scan did not report the %s bypass form; a check that follows no returns must fail this case", tc.name)
				}
			})
		}
	})
}
