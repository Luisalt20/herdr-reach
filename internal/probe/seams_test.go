package probe_test

import (
	"context"
	"crypto/tls"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// TestDenyAllSeamsDenyEveryCapability is design §6.2's first proof level: the
// deny-all default itself must deny, or the default could silently become
// permissive and every probe test would start dialing the real network. Each
// capability is asserted in its own subtest so a seam that started answering
// fails its own case instead of hiding behind a sibling's pass.
func TestDenyAllSeamsDenyEveryCapability(t *testing.T) {
	seams := probe.DenyAllSeams()
	ctx := context.Background()

	t.Run("dial", func(t *testing.T) {
		conn, err := seams.Dialer.DialContext(ctx, "tcp", "203.0.113.10:22")
		if !errors.Is(err, probe.ErrSeamDenied) {
			t.Fatalf("DialContext error = %v, want the deny-all sentinel", err)
		}
		if conn != nil {
			t.Fatalf("DialContext returned a connection (%v) beside its denial", conn)
		}
	})

	t.Run("lookup", func(t *testing.T) {
		hosts, err := seams.Resolver.LookupHost(ctx, "region1.v2.argotunnel.com")
		if !errors.Is(err, probe.ErrSeamDenied) {
			t.Fatalf("LookupHost error = %v, want the deny-all sentinel", err)
		}
		if len(hosts) != 0 {
			t.Fatalf("LookupHost returned %v beside its denial, want no addresses", hosts)
		}
	})

	t.Run("tls verification", func(t *testing.T) {
		verification, err := seams.TLSVerifier.Verify(ctx, "www.cloudflare.com:443", &tls.Config{})
		if !errors.Is(err, probe.ErrSeamDenied) {
			t.Fatalf("Verify error = %v, want the deny-all sentinel", err)
		}
		if verification != (probe.TLSVerification{}) {
			t.Fatalf("Verify returned %+v beside its denial, want the zero verification", verification)
		}
	})

	t.Run("packet dial", func(t *testing.T) {
		conn, err := seams.PacketDialer.DialPacket(ctx, "udp", "region1.v2.argotunnel.com:7844")
		if !errors.Is(err, probe.ErrSeamDenied) {
			t.Fatalf("DialPacket error = %v, want the deny-all sentinel", err)
		}
		if conn != nil {
			t.Fatalf("DialPacket returned a connection (%v) beside its denial", conn)
		}
	})

	t.Run("command", func(t *testing.T) {
		stdout, stderr, err := seams.CommandRunner.Run(ctx, "sshd", "-T")
		if !errors.Is(err, probe.ErrSeamDenied) {
			t.Fatalf("Run error = %v, want the deny-all sentinel", err)
		}
		if len(stdout) != 0 || len(stderr) != 0 {
			t.Fatalf("Run captured stdout=%q stderr=%q beside its denial, want nothing", stdout, stderr)
		}
	})

	t.Run("filesystem", func(t *testing.T) {
		if data, err := seams.FS.ReadFile("/etc/ssh/sshd_config"); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("ReadFile error = %v, want fs.ErrNotExist (data %q)", err, data)
		}
		info, err := seams.FS.Stat("/etc/ssh/sshd_config")
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("Stat error = %v, want fs.ErrNotExist", err)
		}
		if info != nil {
			t.Fatalf("Stat returned %v beside fs.ErrNotExist, want no file info", info)
		}
		if got := seams.FS.Getenv("SSL_CERT_FILE"); got != "" {
			t.Fatalf("Getenv(SSL_CERT_FILE) = %q, want the empty environment", got)
		}
	})

	t.Run("platform", func(t *testing.T) {
		if got := seams.Platform.GOOS(); got != "unknown" {
			t.Fatalf("GOOS() = %q, want %q so the machine classifies as unknown", got, "unknown")
		}
		if got := seams.Platform.Arch(); got != "unknown" {
			t.Fatalf("Arch() = %q, want %q beside an unknown system", got, "unknown")
		}
		if seams.Platform.WSL2() {
			t.Fatalf("WSL2() = true on a seam that was never given a platform")
		}
		if seams.Platform.Systemd() {
			t.Fatalf("Systemd() = true on a seam that was never given a platform")
		}
	})

	t.Run("clock", func(t *testing.T) {
		first := seams.Clock.Now()
		second := seams.Clock.Now()
		if !second.After(first) {
			t.Fatalf("deny-all clock is not monotonic: %v then %v", first, second)
		}
		if step := second.Sub(first); step <= 0 {
			t.Fatalf("deny-all clock step = %v, want a positive scripted step", step)
		}
	})
}

// TestCommandFactDistinguishesMissingCapabilityFromDenial is design §5.1
// obligation 2 at the seam boundary: a run that was never given a command runner
// has an excluded capability, while a run whose runner refused had the capability
// denied. The two must be distinguishable — and neither may become a pass.
func TestCommandFactDistinguishesMissingCapabilityFromDenial(t *testing.T) {
	denying := probe.DenyAllSeams()
	_, _, denialErr := denying.CommandRunner.Run(context.Background(), "sshd", "-T")
	if denialErr == nil {
		t.Fatal("the deny-all command runner did not deny")
	}

	cases := []struct {
		name     string
		seams    probe.Seams
		err      error
		wantKind probe.Observable
	}{
		{"a nil runner was never given the capability", probe.Seams{}, nil, probe.ObsCapabilityExcluded},
		{"a nil runner stays excluded even when an error is handed in", probe.Seams{}, errors.New("boom"), probe.ObsCapabilityExcluded},
		{"an injected deny-all runner denied the command", denying, denialErr, probe.ObsCommandDenied},
		{"an injected runner that succeeded leaves the fact to the caller", denying, nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.seams.CommandFact("sshd -T", tc.err); got.Kind != tc.wantKind {
				t.Fatalf("CommandFact(%q, %v).Kind = %q, want %q", "sshd -T", tc.err, got.Kind, tc.wantKind)
			}
		})
	}

	t.Run("neither denial fact is ever a pass", func(t *testing.T) {
		for _, seams := range []probe.Seams{{}, denying} {
			for _, err := range []error{nil, denialErr} {
				fact := seams.CommandFact("sshd -T", err)
				class := probe.Classify(probe.PurposeSSHDConfiguration, fact)
				if class.Verdict == probe.Pass {
					t.Fatalf("CommandFact(%v) classified as a pass: %+v", err, class)
				}
				if fact.Kind == "" {
					// The command ran, so there is no denial fact; judging its output
					// is the caller's job.
					continue
				}
				if class.Resolution != probe.NotMeasured {
					t.Fatalf("CommandFact(%v) resolution = %q, want %q", err, class.Resolution, probe.NotMeasured)
				}
			}
		}
	})

	t.Run("the two facts carry different reason codes", func(t *testing.T) {
		excluded := probe.Classify(probe.PurposeSSHDConfiguration, probe.Seams{}.CommandFact("sshd -T", nil))
		denied := probe.Classify(probe.PurposeSSHDConfiguration, denying.CommandFact("sshd -T", denialErr))
		if excluded.Reason == denied.Reason {
			t.Fatalf("excluded and denied share the reason code %q, want distinguishable codes", excluded.Reason)
		}
		if excluded.Reason != probe.ReasonCapabilityExcluded {
			t.Fatalf("missing capability reason = %q, want %q", excluded.Reason, probe.ReasonCapabilityExcluded)
		}
		if denied.Reason != probe.ReasonCommandDenied {
			t.Fatalf("denied command reason = %q, want %q", denied.Reason, probe.ReasonCommandDenied)
		}
	})

	t.Run("the fact names the command it could not run", func(t *testing.T) {
		wording := probe.Seams{}.CommandFact("sshd -T", nil).Wording
		if !strings.Contains(wording, "sshd -T") {
			t.Fatalf("CommandFact wording = %q, want the missing capability named", wording)
		}
	})

	t.Run("a command that ran leaves the judgement to the caller", func(t *testing.T) {
		if fact := denying.CommandFact("sshd -T", nil); fact != (probe.RawObservation{}) {
			t.Fatalf("CommandFact with no error = %+v, want no denial fact", fact)
		}
	})
}

// TestDenyAllSeamsNeverReportsARejectedChain keeps the seam denials apart from
// the measurements they must never imitate: a denied handshake is not a chain the
// verifier judged, so it must not read as an interception failure.
func TestDenyAllSeamsNeverReportsARejectedChain(t *testing.T) {
	_, err := probe.DenyAllSeams().TLSVerifier.Verify(context.Background(), "www.cloudflare.com:443", &tls.Config{})
	if !errors.Is(err, probe.ErrSeamDenied) {
		t.Fatalf("deny-all Verify error = %v, want the deny-all sentinel", err)
	}
	if errors.Is(err, probe.ErrTLSVerification) {
		t.Fatalf("a denied seam reported a rejected chain: %v", err)
	}
}

// TestDenyAllSeamsIsFreshPerCall proves the default holds no shared state: a test
// that overrides a field must not change what the next test starts from.
func TestDenyAllSeamsIsFreshPerCall(t *testing.T) {
	mutated := probe.DenyAllSeams()
	mutated.Clock = probe.NewStepperClock(time.Unix(12345, 0), time.Hour)
	mutated.CommandRunner = nil
	mutated.Platform = nil

	fresh := probe.DenyAllSeams()
	if fresh.CommandRunner == nil || fresh.Platform == nil {
		t.Fatal("overriding one Seams value removed a capability from the next default")
	}
	if _, err := fresh.Dialer.DialContext(context.Background(), "tcp", "203.0.113.10:22"); !errors.Is(err, probe.ErrSeamDenied) {
		t.Fatalf("the default stopped denying after another value was overridden: %v", err)
	}
	origin := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	if got := fresh.Clock.Now(); !got.Equal(origin) {
		t.Fatalf("deny-all clock started at %v, want the scripted origin %v", got, origin)
	}
	if got := probe.DenyAllSeams().Clock.Now(); !got.Equal(origin) {
		t.Fatalf("deny-all clock is not scripted: second run started at %v, want %v", got, origin)
	}
}

// TestStepperClockIsScriptedAndConcurrencySafe covers the clock seam's own
// contract: a documented origin, a fixed step, identical readings for identical
// scripts, and one distinct reading per call even when the runner's probes read
// the shared clock concurrently.
func TestStepperClockIsScriptedAndConcurrencySafe(t *testing.T) {
	origin := time.Date(2026, time.September, 14, 12, 0, 0, 0, time.UTC)
	stepping := probe.NewStepperClock(origin, 3*time.Millisecond)
	if got := stepping.Now(); !got.Equal(origin) {
		t.Fatalf("first reading = %v, want the origin %v", got, origin)
	}
	if got, want := stepping.Now(), origin.Add(3*time.Millisecond); !got.Equal(want) {
		t.Fatalf("second reading = %v, want origin + step %v", got, want)
	}

	first := probe.NewStepperClock(origin, time.Second)
	second := probe.NewStepperClock(origin, time.Second)
	for i := 0; i < 5; i++ {
		if a, b := first.Now(), second.Now(); !a.Equal(b) {
			t.Fatalf("two identical scripts disagreed at reading %d: %v vs %v", i, a, b)
		}
	}

	shared := probe.NewStepperClock(origin, time.Millisecond)
	const readers = 64
	readings := make(chan time.Time, readers)
	var wg sync.WaitGroup
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			readings <- shared.Now()
		}()
	}
	wg.Wait()
	close(readings)

	seen := make(map[time.Time]bool, readers)
	for reading := range readings {
		if seen[reading] {
			t.Fatalf("the stepper handed out %v twice", reading)
		}
		seen[reading] = true
	}
	if len(seen) != readers {
		t.Fatalf("the stepper produced %d distinct readings, want %d", len(seen), readers)
	}
}

// TestNoPackageLevelSeamVariable is design §6.1's structural rule: the seam set is
// injected, so no package-level variable may hold a seam, a Seams value, or an
// implementation of one. A global such as `var dial = net.Dial` that a test swaps
// cannot run under t.Parallel(), hides which capability a measurement actually
// needed, and lets a sub-test that forgets to swap reach the real network.
//
// The forbidden vocabulary is read from seams.go itself — every type and every
// function that file declares — so a new seam type joins the guard the moment it
// is declared, without the guard being told about it.
//
// Coverage boundary: the guard refuses a package-level var whose declared type,
// composite-literal value or constructor call belongs to that vocabulary. It is a
// static check of the documented pattern, not a proof against a var declared as
// `any` and filled with a dialer; and a seam implementation declared outside
// seams.go (real.go is where the production one lands) is not in the vocabulary,
// which is why the later static construction guard exists as well.
func TestNoPackageLevelSeamVariable(t *testing.T) {
	forbidden := seamVocabulary(t, "seams.go")

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	scanned := 0
	sawSeamsFile := false
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		scanned++
		if name == "seams.go" {
			sawSeamsFile = true
		}
		for _, declaration := range file.Decls {
			gen, ok := declaration.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if named := declaredName(value.Type); forbidden[named] {
					t.Errorf("%s: %v is a package-level %s; seams must be injected through Seams", name, value.Names, named)
				}
				for _, initial := range value.Values {
					if named := valueName(initial); forbidden[named] {
						t.Errorf("%s: %v is a package-level %s value; seams must be injected through Seams", name, value.Names, named)
					}
				}
			}
		}
	}
	if scanned == 0 || !sawSeamsFile {
		t.Fatalf("the guard scanned %d package sources (seams.go seen: %v), so it proves nothing", scanned, sawSeamsFile)
	}
}

// seamVocabulary returns every type and function name declared in one source
// file: the vocabulary that carries a seam, whether the name is an interface or an
// implementation of one.
func seamVocabulary(t *testing.T, name string) map[string]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	names := map[string]bool{}
	for _, declaration := range file.Decls {
		switch decl := declaration.(type) {
		case *ast.GenDecl:
			if decl.Tok != token.TYPE {
				continue
			}
			for _, spec := range decl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					names[typeSpec.Name.Name] = true
				}
			}
		case *ast.FuncDecl:
			names[decl.Name.Name] = true
		}
	}
	if len(names) == 0 {
		t.Fatalf("%s declares no types or functions, so the guard would pass vacuously", name)
	}
	return names
}

// declaredName names the type an expression declares, unwrapping a pointer and
// ignoring the package qualifier: `*net.Dialer` and `Dialer` both name "Dialer",
// which is what makes the guard compare against the seam vocabulary.
func declaredName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.Ident:
		return typed.Name
	case *ast.StarExpr:
		return declaredName(typed.X)
	case *ast.SelectorExpr:
		return declaredName(typed.Sel)
	default:
		return ""
	}
}

// valueName names the type an initializer produces: the type of a composite
// literal, or the name of the function it calls.
func valueName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.CompositeLit:
		return declaredName(typed.Type)
	case *ast.CallExpr:
		return declaredName(typed.Fun)
	default:
		return ""
	}
}
