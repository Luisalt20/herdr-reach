package probe_test

// This file is the per-probe suite of `local.env` (PRD §5.1, R-HR-29, R-HR-30).
//
// Every case starts from the deny-all seam set of design §6.2 and overrides the
// platform seam only, so a case states the machine it describes instead of
// inheriting the machine the test happens to run on. That is the whole reason the
// classification takes its signals through a seam: a suite that had to be run on
// Windows to test the Windows refusal would never test it at all.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// scriptedPlatform is the Platform seam a local.env case injects: every signal
// the probe can read is a field, so a case is a description of a machine.
type scriptedPlatform struct {
	goos    string
	arch    string
	wsl2    bool
	systemd bool
}

// GOOS reports the scripted operating system.
func (p scriptedPlatform) GOOS() string { return p.goos }

// Arch reports the scripted architecture.
func (p scriptedPlatform) Arch() string { return p.arch }

// WSL2 reports the scripted WSL2 signal.
func (p scriptedPlatform) WSL2() bool { return p.wsl2 }

// Systemd reports the scripted service-manager signal.
func (p scriptedPlatform) Systemd() bool { return p.systemd }

// tripwireFS is an FS seam that records every read a probe attempted. A case can
// therefore prove that the classification needed none, which is a stronger claim
// than "the seams answer nothing": a probe that never called the seam at all may
// not be silently reading the process environment instead.
type tripwireFS struct {
	calls []string
}

// ReadFile records the attempt and refuses it.
func (f *tripwireFS) ReadFile(path string) ([]byte, error) {
	f.calls = append(f.calls, "ReadFile "+path)
	return nil, errors.New("tripwire filesystem: local.env must not read files")
}

// Stat records the attempt and refuses it.
func (f *tripwireFS) Stat(path string) (os.FileInfo, error) {
	f.calls = append(f.calls, "Stat "+path)
	return nil, errors.New("tripwire filesystem: local.env must not stat paths")
}

// Getenv records the attempt and answers "".
func (f *tripwireFS) Getenv(name string) string {
	f.calls = append(f.calls, "Getenv "+name)
	return ""
}

// localEnvBuild looks up the registry entry for local.env and builds the probe
// through it. Reaching the probe the way a run reaches it keeps the registration,
// the declared name and the declared kind inside every case's evidence.
func localEnvBuild(t *testing.T, seams probe.Seams) probe.Probe {
	t.Helper()
	for _, entry := range probe.Registry() {
		if entry.Name != "local.env" {
			continue
		}
		if entry.New == nil {
			t.Fatalf("the registry declares %q without a constructor, so no run could measure it", entry.Name)
		}
		built := entry.New(seams)
		if built.Name() != entry.Name {
			t.Fatalf("the local.env constructor built %q, the registry declares %q", built.Name(), entry.Name)
		}
		if built.Kind() != entry.Kind {
			t.Fatalf("local.env reports kind %q, the registry declares %q", built.Kind(), entry.Kind)
		}
		return built
	}
	t.Fatal("the registry does not declare local.env")
	return nil
}

// runLocalEnv runs local.env over one scripted platform, with the deny-all seams
// of design §6.2 behind every other capability.
func runLocalEnv(t *testing.T, platform probe.Platform) probe.Result {
	t.Helper()
	seams := probe.DenyAllSeams()
	seams.Platform = platform
	return localEnvBuild(t, seams).Run(context.Background())
}

// provisioningTokens are the strings that would show `local.env` offering or
// performing a change to the machine, which R-HR-29 forbids this slice from
// doing: provisioning, persisting, reconfiguring and remediating are all out of
// scope, and the probe's text may name such work only as work it is not doing.
// The list is a wording guard over the probe's own output — the structural proof
// that no write happens at all is the tree digests and the zero-exec counter of
// PR 20 (design §6.3), not this list.
var provisioningTokens = []string{
	"sudo",
	"apt",
	"brew",
	"systemctl",
	"chmod",
	"chown",
	"wsl --",
	"will install",
	"will enable",
	"will configure",
	"will create",
	"will write",
	"will modify",
	"has been applied",
	"is enforced",
}

// assertNoProvisioningAction fails the case if the probe's text offers or claims a
// change to the machine.
func assertNoProvisioningAction(t *testing.T, detail string) {
	t.Helper()
	for _, token := range provisioningTokens {
		if strings.Contains(detail, token) {
			t.Errorf("detail offers or claims a provisioning action: %q appears in %q", token, detail)
		}
	}
}

// TestLocalEnvClassifiesEachPlatform scripts all four platform seams of R-HR-29
// and asserts each machine is classified with its architecture. The classification
// is read back out of the observation's target through the package's own splitter,
// so the case proves the value a consumer receives, not the value the probe meant.
//
// Native Windows is asserted here only as a classification: that the node is
// classified and reported with its architecture. Its refusal is its own case
// below, added with the refusal itself.
func TestLocalEnvClassifiesEachPlatform(t *testing.T) {
	cases := []struct {
		name     string
		platform scriptedPlatform
		want     probe.NodePlatform
		wantArch string
	}{
		{
			name:     "linux with systemd",
			platform: scriptedPlatform{goos: "linux", arch: "aarch64", systemd: true},
			want:     probe.NodePlatformLinux, wantArch: "aarch64",
		},
		{
			name:     "linux without systemd",
			platform: scriptedPlatform{goos: "linux", arch: "x86_64"},
			want:     probe.NodePlatformLinux, wantArch: "x86_64",
		},
		{
			name:     "macos",
			platform: scriptedPlatform{goos: "darwin", arch: "arm64"},
			want:     probe.NodePlatformMacOS, wantArch: "arm64",
		},
		{
			name:     "wsl2 with systemd",
			platform: scriptedPlatform{goos: "linux", arch: "x86_64", wsl2: true, systemd: true},
			want:     probe.NodePlatformWSL2, wantArch: "x86_64",
		},
		{
			name:     "wsl2 without systemd",
			platform: scriptedPlatform{goos: "linux", arch: "x86_64", wsl2: true},
			want:     probe.NodePlatformWSL2, wantArch: "x86_64",
		},
		{
			name:     "native windows",
			platform: scriptedPlatform{goos: "windows", arch: "amd64"},
			want:     probe.NodePlatformWindowsNative, wantArch: "amd64",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runLocalEnv(t, tc.platform)

			if result.Probe != "local.env" {
				t.Fatalf("result names the probe %q, want %q", result.Probe, "local.env")
			}
			if result.Kind != probe.ProbeLocal {
				t.Fatalf("result kind = %q, want %q", result.Kind, probe.ProbeLocal)
			}
			if len(result.Observations) != 1 {
				t.Fatalf("result carries %d observations, want the 1 classification local.env measures", len(result.Observations))
			}

			observation := result.Observations[0]
			if !holds(observation) {
				t.Fatalf("observation does not satisfy the measurement vocabulary's invariant: %+v", observation)
			}
			if observation.Label != "platform" {
				t.Fatalf("observation label = %q, want %q", observation.Label, "platform")
			}
			if observation.Resolution != probe.Measured {
				t.Fatalf("a scripted %s machine measured %q, want %q", tc.name, observation.Resolution, probe.Measured)
			}
			if result.Verdict != observation.Verdict || result.Reason != observation.Reason {
				t.Fatalf("the result's reduction disagrees with its only observation: %+v vs %+v", result, observation)
			}

			platform, arch, ok := probe.SplitNodePlatformIdentity(observation.Target)
			if !ok {
				t.Fatalf("observation target %q is not a platform identity", observation.Target)
			}
			if platform != tc.want || arch != tc.wantArch {
				t.Fatalf("classification = (%q, %q), want (%q, %q)", platform, arch, tc.want, tc.wantArch)
			}
			if result.Target != observation.Target {
				t.Fatalf("result target = %q, observation target = %q", result.Target, observation.Target)
			}
			if !strings.Contains(observation.Detail, tc.wantArch) {
				t.Errorf("detail does not report the architecture: %q", observation.Detail)
			}
			if strings.TrimSpace(observation.Detail) == "" {
				t.Error("detail is empty: a measurement's verbatim text must be quotable")
			}
			assertNoProvisioningAction(t, observation.Detail)
		})
	}
}

// TestLocalEnvUnknownPlatformIsNotGuessed is R-HR-29's second scenario: a platform
// whose signals match no supported classification degrades to an explicit
// unresolved outcome naming the missing signal, and never to a default guess.
//
// Three ways to have no classification are covered, because they are three
// different gaps a reader has to be able to tell apart in words even though they
// share one reason code: nothing reported at all, an operating system outside the
// supported set, and a supported operating system whose architecture half was not
// reported.
func TestLocalEnvUnknownPlatformIsNotGuessed(t *testing.T) {
	cases := []struct {
		name       string
		platform   probe.Platform
		wantSignal string
	}{
		{
			name:       "the deny-all seam reports nothing",
			platform:   probe.DenyAllSeams().Platform,
			wantSignal: "unknown",
		},
		{
			name:       "an operating system outside the supported set",
			platform:   scriptedPlatform{goos: "freebsd", arch: "amd64"},
			wantSignal: "freebsd",
		},
		{
			name:       "a supported operating system with an unknown architecture",
			platform:   scriptedPlatform{goos: "linux", arch: "unknown"},
			wantSignal: "no architecture",
		},
		{
			name:       "a supported operating system with an empty architecture",
			platform:   scriptedPlatform{goos: "linux", arch: ""},
			wantSignal: "no architecture",
		},
		{
			// macOS is a supported node platform, so its architecture half is
			// guarded exactly as Linux's is: the guard belongs to the classification
			// rather than to one operating system's branch.
			name:       "a macos node with an unknown architecture",
			platform:   scriptedPlatform{goos: "darwin", arch: "unknown"},
			wantSignal: "no architecture",
		},
		{
			name:       "no platform seam was injected",
			platform:   nil,
			wantSignal: "no platform seam",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runLocalEnv(t, tc.platform)

			if result.Verdict != probe.Indeterminate {
				t.Fatalf("verdict = %q, want %q: an unclassified machine is not a negative answer", result.Verdict, probe.Indeterminate)
			}
			if result.Reason != probe.ReasonPlatformUnknown {
				t.Fatalf("reason = %q, want %q", result.Reason, probe.ReasonPlatformUnknown)
			}
			if result.Verdict == probe.Pass {
				t.Fatal("an unclassified machine was reported as a pass")
			}

			observation := result.Observations[0]
			if observation.Resolution != probe.Unresolved {
				t.Fatalf("resolution = %q, want %q", observation.Resolution, probe.Unresolved)
			}
			platform, arch, ok := probe.SplitNodePlatformIdentity(observation.Target)
			if !ok {
				t.Fatalf("observation target %q is not a platform identity", observation.Target)
			}
			if platform != probe.NodePlatformUnknown || arch != "unknown" {
				t.Fatalf("unclassified identity = (%q, %q), want (%q, %q)", platform, arch, probe.NodePlatformUnknown, "unknown")
			}
			if !strings.Contains(observation.Detail, tc.wantSignal) {
				t.Errorf("detail does not name the missing signal %q: %q", tc.wantSignal, observation.Detail)
			}
			assertNoProvisioningAction(t, observation.Detail)
		})
	}
}

// TestLocalEnvRefusesNativeWindows is R-HR-30: a node classified as native
// Windows is refused, and the refusal explains itself by naming WSL2 as the
// supported path. The refusal is the measurement's own negative answer — the
// operating system answered, and the answer is that this tool will not operate on
// it — so it is a measured fail with the platform refusal's reason code, never a
// usage error and never an internal one.
//
// The case also asserts the two things the refusal must not do: present a
// transport as viable, and claim any change. A refused node has no viable
// transport, and this slice changes nothing on any node.
func TestLocalEnvRefusesNativeWindows(t *testing.T) {
	cases := []struct {
		name     string
		platform scriptedPlatform
		wantArch string
	}{
		{
			name:     "a measured architecture",
			platform: scriptedPlatform{goos: "windows", arch: "amd64"},
			wantArch: "amd64",
		},
		{
			// The refusal rests on the operating system, so an unreported
			// architecture neither weakens it nor fabricates one.
			name:     "an architecture the seam did not report",
			platform: scriptedPlatform{goos: "windows", arch: "unknown"},
			wantArch: "unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runLocalEnv(t, tc.platform)

			if len(result.Observations) != 1 {
				t.Fatalf("the refusal carries %d observations, want 1", len(result.Observations))
			}
			observation := result.Observations[0]
			if observation.Resolution != probe.Measured {
				t.Fatalf("resolution = %q, want %q: the operating system answered", observation.Resolution, probe.Measured)
			}
			if observation.Verdict != probe.Fail {
				t.Fatalf("verdict = %q, want %q", observation.Verdict, probe.Fail)
			}
			if observation.Reason != probe.ReasonNodePlatformUnsupported {
				t.Fatalf("reason = %q, want %q", observation.Reason, probe.ReasonNodePlatformUnsupported)
			}
			if result.Verdict != probe.Fail || result.Reason != probe.ReasonNodePlatformUnsupported {
				t.Fatalf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Fail, probe.ReasonNodePlatformUnsupported)
			}

			platform, arch, ok := probe.SplitNodePlatformIdentity(observation.Target)
			if !ok {
				t.Fatalf("observation target %q is not a platform identity", observation.Target)
			}
			if platform != probe.NodePlatformWindowsNative || arch != tc.wantArch {
				t.Fatalf("refused identity = (%q, %q), want (%q, %q)", platform, arch, probe.NodePlatformWindowsNative, tc.wantArch)
			}

			if !strings.Contains(observation.Detail, "WSL2") {
				t.Errorf("the refusal does not name WSL2 as the supported path: %q", observation.Detail)
			}
			if !strings.Contains(observation.Detail, "supported path") {
				t.Errorf("the refusal does not say that WSL2 is the supported path: %q", observation.Detail)
			}
			if strings.Contains(observation.Detail, "viable") {
				t.Errorf("the refusal presents a transport as viable: %q", observation.Detail)
			}
			for _, claim := range []string{"applied", "enforced"} {
				if strings.Contains(observation.Detail, claim) {
					t.Errorf("the refusal claims a change was %s: %q", claim, observation.Detail)
				}
			}
			assertNoProvisioningAction(t, observation.Detail)
		})
	}
}

// TestLocalEnvWSL2TextIsDocumentedSemanticsOnly is RG-4 and design §5.2's
// NODE_WSL2_SYSTEMD_ABSENT row.
//
// Two opposite obligations meet in the WSL2 text. It must repeat the semantics
// that are documented — vmIdleTimeout counts milliseconds of idle, its default is
// 60000, and the setting exists only on Windows 11 — and it must not repeat the
// rules that are not: the child-of-init shutdown rule and the `-1` sentinel were
// not verifiable in the reference material, so they are excluded from this
// slice's output entirely and another slice owes their measurement. It must also
// not describe keepalive, watchdog or persistence behaviour, none of which this
// slice measured.
//
// The second half is the systemd signal: when systemd is not the running service
// manager the text says so and names the enabling work as a later slice's, which
// is detection and no repair.
func TestLocalEnvWSL2TextIsDocumentedSemanticsOnly(t *testing.T) {
	wording := func(t *testing.T, systemd bool) string {
		t.Helper()
		result := runLocalEnv(t, scriptedPlatform{goos: "linux", arch: "x86_64", wsl2: true, systemd: systemd})
		if len(result.Observations) != 1 {
			t.Fatalf("the WSL2 result carries %d observations, want 1", len(result.Observations))
		}
		return result.Detail
	}

	withSystemd := wording(t, true)
	withoutSystemd := wording(t, false)

	// The documented semantics are present in both states: they describe the
	// environment, not the service manager.
	for _, want := range []string{"milliseconds", "60000", "Windows 11"} {
		for _, got := range []string{withSystemd, withoutSystemd} {
			if !strings.Contains(got, want) {
				t.Errorf("the WSL2 text does not state the documented semantics %q: %q", want, got)
			}
		}
	}

	// The undocumented rules are absent, in both states, spelled the ways they
	// are commonly written. `-1` is checked as a substring because the sentinel is
	// exactly that: a value that must not appear, not a phrase to paraphrase.
	forbidden := []string{
		"-1",
		"child of init",
		"child-of-init",
		"init process",
		"/init",
		"keepalive",
		"keep-alive",
		"watchdog",
		"persist",
	}
	for _, token := range forbidden {
		for _, got := range []string{withSystemd, withoutSystemd} {
			if strings.Contains(got, token) {
				t.Errorf("the WSL2 text states the undocumented %q: %q", token, got)
			}
		}
	}

	// The service-manager signal is reported, not assumed: the two states must
	// produce different texts, and only the absent case may name work a later
	// slice owns.
	if withSystemd == withoutSystemd {
		t.Fatalf("the WSL2 text does not report the systemd signal at all: %q", withSystemd)
	}
	if !strings.Contains(withSystemd, "systemd is the running service manager") {
		t.Errorf("the WSL2 text with systemd does not report it: %q", withSystemd)
	}
	if strings.Contains(withSystemd, "later slice") {
		t.Errorf("the WSL2 text with systemd claims work by a later slice: %q", withSystemd)
	}
	if !strings.Contains(withoutSystemd, "systemd is not the running service manager") {
		t.Errorf("the WSL2 text without systemd does not detect it: %q", withoutSystemd)
	}
	if !strings.Contains(withoutSystemd, "later slice") {
		t.Errorf("the WSL2 text without systemd does not state that enabling it belongs to a later slice: %q", withoutSystemd)
	}

	assertNoProvisioningAction(t, withSystemd)
	assertNoProvisioningAction(t, withoutSystemd)
}

// TestNodePlatformIdentityRoundTrip is the triangulation of the identity the
// observation's target carries: the value local.env writes and the value a
// consumer reads back are the same two facts, for every classification the probe
// can produce. An identity that could not be split would force every consumer to
// guess where the classification ends, and a malformed value must be refused
// rather than read as a classification it never carried.
func TestNodePlatformIdentityRoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		platform probe.NodePlatform
		arch     string
		wantArch string
	}{
		{"linux", probe.NodePlatformLinux, "aarch64", "aarch64"},
		{"macos", probe.NodePlatformMacOS, "arm64", "arm64"},
		{"wsl2", probe.NodePlatformWSL2, "x86_64", "x86_64"},
		{"native windows", probe.NodePlatformWindowsNative, "amd64", "amd64"},
		{"unknown platform", probe.NodePlatformUnknown, "unknown", "unknown"},
		{"an architecture the seam did not report", probe.NodePlatformLinux, "", "unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			identity := probe.NodePlatformIdentity(tc.platform, tc.arch)
			platform, arch, ok := probe.SplitNodePlatformIdentity(identity)
			if !ok {
				t.Fatalf("%q does not split back into a platform identity", identity)
			}
			if platform != tc.platform || arch != tc.wantArch {
				t.Fatalf("%q splits to (%q, %q), want (%q, %q)", identity, platform, arch, tc.platform, tc.wantArch)
			}
		})
	}

	for _, malformed := range []string{"", "linux", "/aarch64", "linux/", "/"} {
		if _, _, ok := probe.SplitNodePlatformIdentity(malformed); ok {
			t.Errorf("SplitNodePlatformIdentity(%q) reported a classification the value never carried", malformed)
		}
	}
}

// TestLocalEnvOffersNoProvisioningAction applies the R-HR-29 prohibition to every
// text the probe can produce: the classification, the refusal and every
// unclassified case. A probe that measures and offers nothing to do about what it
// measured is the whole boundary of this slice.
func TestLocalEnvOffersNoProvisioningAction(t *testing.T) {
	platforms := []probe.Platform{
		scriptedPlatform{goos: "linux", arch: "aarch64", systemd: true},
		scriptedPlatform{goos: "linux", arch: "x86_64"},
		scriptedPlatform{goos: "darwin", arch: "arm64"},
		scriptedPlatform{goos: "linux", arch: "x86_64", wsl2: true, systemd: true},
		scriptedPlatform{goos: "linux", arch: "x86_64", wsl2: true},
		scriptedPlatform{goos: "windows", arch: "amd64"},
		scriptedPlatform{goos: "freebsd", arch: "amd64"},
		nil,
	}

	for _, platform := range platforms {
		result := runLocalEnv(t, platform)
		assertNoProvisioningAction(t, result.Detail)
		for _, observation := range result.Observations {
			assertNoProvisioningAction(t, observation.Detail)
		}
	}
}

// TestLocalEnvReadsOnlyInjectedSeams proves two properties at once.
//
// The first is that the classification reads nothing through the FS seam it was
// handed: the tripwire records every attempt and the case asserts the record is
// empty, so "the probe needed no file and no environment variable" is a fact
// rather than a claim.
//
// The second is that the classification never reads the process environment. The
// WSL2 variables a real WSL2 machine carries are set for the whole case, and the
// classification must still be the one the seam reported: a probe that consulted
// the environment would call this Linux machine WSL2 (or classify a scripted WSL2
// machine from the test host's variables).
func TestLocalEnvReadsOnlyInjectedSeams(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu-24.04")
	t.Setenv("WSL_INTEROP", "/run/WSL/8_interop")
	t.Setenv("WSLENV", "WSL_DISTRO_NAME")

	tripwire := &tripwireFS{}
	seams := probe.DenyAllSeams()
	seams.Platform = scriptedPlatform{goos: "linux", arch: "aarch64", systemd: true}
	seams.FS = tripwire

	result := localEnvBuild(t, seams).Run(context.Background())

	if len(tripwire.calls) != 0 {
		t.Fatalf("local.env read the filesystem seam: %v", tripwire.calls)
	}

	platform, arch, ok := probe.SplitNodePlatformIdentity(result.Target)
	if !ok {
		t.Fatalf("observation target %q is not a platform identity", result.Target)
	}
	if platform != probe.NodePlatformLinux || arch != "aarch64" {
		t.Fatalf("classification = (%q, %q), want (%q, %q): the seam's signals are the only input", platform, arch, probe.NodePlatformLinux, "aarch64")
	}
}

// TestLocalEnvElapsedComesFromTheInjectedClock asserts the timing contract of
// design §3.3: when the run injects a clock, the probe's elapsed value comes from
// that clock (the deny-all stepper advances by one millisecond per reading, so the
// value is exactly one millisecond rather than whatever the machine's clock said),
// and when the run injects none the probe reports zero and leaves the value to the
// runner instead of reaching for the wall clock.
func TestLocalEnvElapsedComesFromTheInjectedClock(t *testing.T) {
	platform := scriptedPlatform{goos: "linux", arch: "aarch64", systemd: true}

	scripted := probe.DenyAllSeams()
	scripted.Platform = platform
	fromClock := localEnvBuild(t, scripted).Run(context.Background())
	if fromClock.Elapsed != time.Millisecond {
		t.Fatalf("elapsed with a scripted clock = %s, want %s", fromClock.Elapsed, time.Millisecond)
	}

	withoutClock := probe.DenyAllSeams()
	withoutClock.Platform = platform
	withoutClock.Clock = nil
	unfilled := localEnvBuild(t, withoutClock).Run(context.Background())
	if unfilled.Elapsed != 0 {
		t.Fatalf("elapsed without a clock = %s, want 0 so the runner fills it", unfilled.Elapsed)
	}
}
