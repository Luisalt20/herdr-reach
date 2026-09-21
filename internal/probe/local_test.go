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
	"fmt"
	"io/fs"
	"os"
	"reflect"
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
		built := entry.New(seams, probe.TargetInput{})
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
// Native Windows is asserted here as a supported classification, with the same
// measured shape as Linux and macOS. The version caveat its wording must carry is
// asserted by its own case below.
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
			// Windows became a supported classification in Herdr 0.9.1, so its
			// architecture half is guarded exactly as Linux's and macOS's are: a
			// supported operating system whose architecture was not reported is not
			// classified as supported.
			name:       "a native windows node with an unknown architecture",
			platform:   scriptedPlatform{goos: "windows", arch: "unknown"},
			wantSignal: "no architecture",
		},
		{
			name:       "a native windows node with an empty architecture",
			platform:   scriptedPlatform{goos: "windows", arch: ""},
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

// TestLocalEnvClassifiesNativeWindowsAsSupported is R-HR-30 applied in the
// direction that is easy to forget: a pass has to be as earned as a failure.
// Native Windows is classified the way Linux and macOS are — a measured,
// supported classification — because upstream Herdr supports a Windows server as
// of 0.9.1, measured 2026-09-20 when `machine add` saved a Windows 11 24H2 host
// and the hub listed the agents running natively on it.
//
// The caveat travels with the classification, in the classification's own
// wording: the tool does not measure the node's Herdr version, so a node running
// a server older than 0.9.1 is classified supported here while being unable to
// host a saved-machine connection. The wording also keeps the boundary that
// remains: the provisioning slices still do not cover native Windows.
//
// The case also asserts the two things the classification must not do: present a
// transport as viable, and claim any change. Detection is not provisioning, and
// this slice changes nothing on any node.
func TestLocalEnvClassifiesNativeWindowsAsSupported(t *testing.T) {
	result := runLocalEnv(t, scriptedPlatform{goos: "windows", arch: "amd64"})

	if len(result.Observations) != 1 {
		t.Fatalf("the classification carries %d observations, want 1", len(result.Observations))
	}
	observation := result.Observations[0]
	if observation.Resolution != probe.Measured {
		t.Fatalf("resolution = %q, want %q: the operating system answered", observation.Resolution, probe.Measured)
	}
	if observation.Verdict != probe.Pass {
		t.Fatalf("verdict = %q, want %q: a platform Herdr supports is not a negative answer", observation.Verdict, probe.Pass)
	}
	if observation.Reason != probe.ReasonOK {
		t.Fatalf("reason = %q, want %q", observation.Reason, probe.ReasonOK)
	}
	if result.Verdict != probe.Pass || result.Reason != probe.ReasonOK {
		t.Fatalf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Pass, probe.ReasonOK)
	}

	platform, arch, ok := probe.SplitNodePlatformIdentity(observation.Target)
	if !ok {
		t.Fatalf("observation target %q is not a platform identity", observation.Target)
	}
	if platform != probe.NodePlatformWindowsNative || arch != "amd64" {
		t.Fatalf("classification identity = (%q, %q), want (%q, %q)", platform, arch, probe.NodePlatformWindowsNative, "amd64")
	}

	// The version caveat must be in the classification's own wording, not only in
	// the docs: the conclusion is what a reader and an agent act on.
	for _, want := range []string{
		"can host a supported Herdr server as of 0.9.1",
		"does not measure which Herdr version is installed",
		"older server cannot host a saved-machine connection",
		"provisioning slices still do not cover native Windows",
		"changes nothing",
	} {
		if !strings.Contains(observation.Detail, want) {
			t.Errorf("the classification does not carry %q: %q", want, observation.Detail)
		}
	}
	for _, stale := range []string{"unsupported", "refused", "WSL2 is the supported path", "cannot host a supported Herdr server"} {
		if strings.Contains(observation.Detail, stale) {
			t.Errorf("the classification still carries the stale claim %q: %q", stale, observation.Detail)
		}
	}
	if strings.Contains(observation.Detail, "viable") {
		t.Errorf("the classification presents a transport as viable: %q", observation.Detail)
	}
	for _, claim := range []string{"applied", "enforced"} {
		if strings.Contains(observation.Detail, claim) {
			t.Errorf("the classification claims a change was %s: %q", claim, observation.Detail)
		}
	}
	assertNoProvisioningAction(t, observation.Detail)
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

// --- local.sshd ---------------------------------------------------------------
//
// The suite below is the `local.sshd` half of this slice (PRD §5.1, R-HR-18).
// Every case starts from the deny-all seam set of design §6.2 and replaces the
// two seams this probe is allowed to read — the filesystem and the command
// runner — so a case describes one machine's sshd instead of the machine the
// test happens to run on.
//
// The documented local inputs are repeated here as literals on purpose. What the
// probe reads — one binary path and one written configuration file per platform,
// one service query and one effective-configuration command — is part of its
// contract, so a rename in local.go has to break this suite rather than silently
// move the probe's inputs. The commands are also asserted exactly, which is the
// strongest statement available here that this probe runs nothing that could
// change the machine.
const (
	// testSSHDBinaryPath and testSSHDConfigPath are the documented sshd locations
	// on the POSIX platforms: Linux, macOS and WSL2, the last of which reports
	// GOOS "linux" and is therefore a POSIX platform here.
	testSSHDBinaryPath = "/usr/sbin/sshd"
	testSSHDConfigPath = "/etc/ssh/sshd_config"
	// testSSHDBinaryPathWindows and testSSHDConfigPathWindows are the documented
	// locations of the OpenSSH server that ships with Windows. They are the pair a
	// native-Windows node is checked at, and the POSIX pair above must not be
	// touched there.
	testSSHDBinaryPathWindows = `C:\Windows\System32\OpenSSH\sshd.exe`
	testSSHDConfigPathWindows = `C:\ProgramData\ssh\sshd_config`
	// testSSHDConfigCommand is the command that reports the effective
	// configuration.
	testSSHDConfigCommand = "sshd -T"
	// testSSHDServiceCommand is the service-manager query for the sshd unit. It
	// is a read-only query: it asks the service manager for the unit's state and
	// changes nothing.
	testSSHDServiceCommand = "systemctl is-active sshd.service ssh.service"
)

// writtenConfigAgreeingWithEffect is a written configuration the effective
// configuration of runningSSHDRunner agrees with: the same directives with the
// same values, parsed case-insensitively by name.
const writtenConfigAgreeingWithEffect = "PermitRootLogin no\nPasswordAuthentication no\n"

// scriptedFileInfo is the os.FileInfo a scriptedFS reports for a path it holds.
// Only existence matters to this probe, so every other attribute is the zero
// value: a case must not be able to pass because of a size or a mode it never
// stated.
type scriptedFileInfo struct {
	path string
}

// Name reports the scripted path.
func (i scriptedFileInfo) Name() string { return i.path }

// Size reports zero: this probe reads existence, not size.
func (scriptedFileInfo) Size() int64 { return 0 }

// Mode reports a regular file's permission bits.
func (scriptedFileInfo) Mode() fs.FileMode { return 0o644 }

// ModTime reports the zero time: no case measures a timestamp through the FS.
func (scriptedFileInfo) ModTime() time.Time { return time.Time{} }

// IsDir reports false: every scripted path is a file.
func (scriptedFileInfo) IsDir() bool { return false }

// Sys reports no underlying data.
func (scriptedFileInfo) Sys() any { return nil }

// scriptedFS is the FS seam a local.sshd case injects. It records every call, so
// a case can assert exactly which local inputs the probe read, and it can be
// scripted to fail for a reason other than absence, which is a different fact
// from "this path is not there".
type scriptedFS struct {
	// files maps each path that exists to the contents ReadFile returns.
	files map[string]string
	// statErrs maps a path to a Stat failure that is not "not exist".
	statErrs map[string]error
	// readErrs maps a path to a ReadFile failure that is not "not exist".
	readErrs map[string]error
	// calls records every seam call in order.
	calls []string
}

// ReadFile records the attempt and returns the scripted contents or a failure.
func (f *scriptedFS) ReadFile(path string) ([]byte, error) {
	f.calls = append(f.calls, "ReadFile "+path)
	if err, ok := f.readErrs[path]; ok {
		return nil, err
	}
	contents, ok := f.files[path]
	if !ok {
		return nil, fmt.Errorf("%s: %w", path, fs.ErrNotExist)
	}
	return []byte(contents), nil
}

// Stat records the attempt and reports the scripted existence or a failure.
func (f *scriptedFS) Stat(path string) (os.FileInfo, error) {
	f.calls = append(f.calls, "Stat "+path)
	if err, ok := f.statErrs[path]; ok {
		return nil, err
	}
	if _, ok := f.files[path]; !ok {
		return nil, fmt.Errorf("%s: %w", path, fs.ErrNotExist)
	}
	return scriptedFileInfo{path: path}, nil
}

// Getenv records the attempt and answers "": this probe reads no environment
// variable, and the record is what proves it.
func (f *scriptedFS) Getenv(name string) string {
	f.calls = append(f.calls, "Getenv "+name)
	return ""
}

// scriptedAnswer is what one scripted command invocation returned.
type scriptedAnswer struct {
	stdout string
	stderr string
	err    error
}

// scriptedRunner is the CommandRunner seam a local.sshd case injects. It is
// keyed by the exact command line, so a case states what the probe runs and
// fails loudly for any invocation the case did not script.
type scriptedRunner struct {
	answers map[string]scriptedAnswer
	calls   []string
}

// Run records the invocation and returns the scripted answer.
func (r *scriptedRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	line := strings.Join(append([]string{name}, args...), " ")
	r.calls = append(r.calls, line)
	answer, ok := r.answers[line]
	if !ok {
		return nil, nil, fmt.Errorf("the scripted runner has no answer for %q", line)
	}
	return []byte(answer.stdout), []byte(answer.stderr), answer.err
}

// sshdFiles is a scripted filesystem for a machine where sshd is installed and
// the written configuration is the one the case supplies.
func sshdFiles(written string) *scriptedFS {
	return &scriptedFS{files: map[string]string{
		testSSHDBinaryPath: "",
		testSSHDConfigPath: written,
	}}
}

// sshdRunner is a scripted command runner for a machine whose sshd service is
// active and whose effective configuration is the caller's.
func sshdRunner(effective string) *scriptedRunner {
	return &scriptedRunner{answers: map[string]scriptedAnswer{
		testSSHDServiceCommand: {stdout: "active\ninactive\n"},
		testSSHDConfigCommand:  {stdout: effective},
	}}
}

// localSSHDSeams builds the seam set one case describes: the deny-all set of
// design §6.2 with the filesystem and the command runner replaced.
func localSSHDSeams(fsSeam probe.FS, runner probe.CommandRunner) probe.Seams {
	seams := probe.DenyAllSeams()
	seams.FS = fsSeam
	seams.CommandRunner = runner
	return seams
}

// localSSHDBuild looks up the registry entry for local.sshd and builds the probe
// through it, exactly as a run reaches it.
func localSSHDBuild(t *testing.T, seams probe.Seams) probe.Probe {
	t.Helper()
	for _, entry := range probe.Registry() {
		if entry.Name != "local.sshd" {
			continue
		}
		if entry.New == nil {
			t.Fatalf("the registry declares %q without a constructor, so no run could measure it", entry.Name)
		}
		built := entry.New(seams, probe.TargetInput{})
		if built.Name() != entry.Name {
			t.Fatalf("the local.sshd constructor built %q, the registry declares %q", built.Name(), entry.Name)
		}
		if built.Kind() != entry.Kind {
			t.Fatalf("local.sshd reports kind %q, the registry declares %q", built.Kind(), entry.Kind)
		}
		return built
	}
	t.Fatal("the registry does not declare local.sshd")
	return nil
}

// runLocalSSHD runs local.sshd over one scripted machine.
func runLocalSSHD(t *testing.T, seams probe.Seams) probe.Result {
	t.Helper()
	return localSSHDBuild(t, seams).Run(context.Background())
}

// sshdProvisioningTokens is the wording guard for this probe: the strings that
// would show local.sshd offering or claiming a change to the machine. It is
// `provisioningTokens` without "systemctl", because measuring the service state
// reads the service manager through a read-only query — which the exact-command
// assertion of TestLocalSshdReadsOnlyInjectedReaders pins — so the word itself
// is part of the measurement's own detail rather than an offer to change
// anything.
var sshdProvisioningTokens = []string{
	"sudo",
	"apt",
	"brew",
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

// assertNoSSHDProvisioningAction fails the case if the probe's text offers or
// claims a change to the machine.
func assertNoSSHDProvisioningAction(t *testing.T, detail string) {
	t.Helper()
	for _, token := range sshdProvisioningTokens {
		if strings.Contains(detail, token) {
			t.Errorf("local.sshd detail offers or claims a provisioning action: %q appears in %q", token, detail)
		}
	}
}

// TestLocalSshdReportsThreeSeparateObservations is R-HR-18's first sentence: the
// binary's presence, the service's state and the effective configuration are
// three observations, each with its own outcome, and all three measured means
// the probe passes.
func TestLocalSshdReportsThreeSeparateObservations(t *testing.T) {
	result := runLocalSSHD(t, localSSHDSeams(
		sshdFiles(writtenConfigAgreeingWithEffect),
		sshdRunner("permitrootlogin no\npasswordauthentication no\n"),
	))

	if result.Probe != "local.sshd" {
		t.Fatalf("result names the probe %q, want %q", result.Probe, "local.sshd")
	}
	if result.Kind != probe.ProbeLocal {
		t.Fatalf("result kind = %q, want %q", result.Kind, probe.ProbeLocal)
	}
	wantLabels := []string{"binary present", "service state", "effective config"}
	if len(result.Observations) != len(wantLabels) {
		t.Fatalf("local.sshd reported %d observations, want the %d it must report separately (binary presence, service state, effective configuration)",
			len(result.Observations), len(wantLabels))
	}
	for i, want := range wantLabels {
		observation := result.Observations[i]
		if observation.Label != want {
			t.Errorf("observation %d label = %q, want %q", i, observation.Label, want)
		}
		if !holds(observation) {
			t.Errorf("observation %q does not satisfy the measurement vocabulary's invariant: %+v", observation.Label, observation)
		}
		if observation.Resolution != probe.Measured {
			t.Errorf("observation %q resolution = %q, want %q: a scripted healthy machine was fully measured", observation.Label, observation.Resolution, probe.Measured)
		}
		if observation.Verdict != probe.Pass || observation.Reason != probe.ReasonOK {
			t.Errorf("observation %q = (%q, %q), want (%q, %q)", observation.Label, observation.Verdict, observation.Reason, probe.Pass, probe.ReasonOK)
		}
		if strings.TrimSpace(observation.Detail) == "" {
			t.Errorf("observation %q carries no verbatim detail", observation.Label)
		}
		assertNoSSHDProvisioningAction(t, observation.Detail)
	}
	if result.Verdict != probe.Pass || result.Reason != probe.ReasonOK {
		t.Fatalf("result = (%q, %q), want (%q, %q) for a machine whose three observations all measured", result.Verdict, result.Reason, probe.Pass, probe.ReasonOK)
	}
	if result.Target != testSSHDBinaryPath {
		t.Errorf("result target = %q, want the probe's local subject %q", result.Target, testSSHDBinaryPath)
	}
	verdict, reason := probe.Aggregate(result.Observations)
	if result.Verdict != verdict || result.Reason != reason {
		t.Fatalf("the result's reduction %s/%s disagrees with Aggregate over its observations %s/%s", result.Verdict, result.Reason, verdict, reason)
	}
	assertNoSSHDProvisioningAction(t, result.Detail)
}

// TestLocalSshdReportsWrittenVersusEffectiveDivergence is R-HR-18's central
// case: a written configuration the effective configuration does not agree with
// is a measured failure, both configurations appear in the verbatim detail, and
// the probe never reports success.
func TestLocalSshdReportsWrittenVersusEffectiveDivergence(t *testing.T) {
	written := "PermitRootLogin no\nPasswordAuthentication no\n"
	effective := "permitrootlogin yes\npasswordauthentication no\n"

	result := runLocalSSHD(t, localSSHDSeams(sshdFiles(written), sshdRunner(effective)))

	observation, ok := observationByLabel(result, "effective config")
	if !ok {
		t.Fatalf("local.sshd reported no effective-configuration observation: %+v", result.Observations)
	}
	if observation.Resolution != probe.Measured {
		t.Errorf("divergence resolution = %q, want %q: the divergence is a measured fact, not an ambiguity", observation.Resolution, probe.Measured)
	}
	if observation.Verdict != probe.Fail || observation.Reason != probe.ReasonSSHDConfigDivergence {
		t.Errorf("divergence = (%q, %q), want (%q, %q)", observation.Verdict, observation.Reason, probe.Fail, probe.ReasonSSHDConfigDivergence)
	}
	if result.Verdict != probe.Fail || result.Reason != probe.ReasonSSHDConfigDivergence {
		t.Errorf("result = (%q, %q), want (%q, %q): a divergent configuration must not be reported as success", result.Verdict, result.Reason, probe.Fail, probe.ReasonSSHDConfigDivergence)
	}
	if result.Verdict == probe.Pass {
		t.Fatal("a divergent configuration was reported as a pass")
	}

	for _, want := range []string{"PermitRootLogin no", "permitrootlogin yes", "permitrootlogin"} {
		if !strings.Contains(observation.Detail, want) {
			t.Errorf("the divergence detail does not carry %q verbatim: %q", want, observation.Detail)
		}
	}
	if !strings.Contains(result.Detail, "PermitRootLogin no") || !strings.Contains(result.Detail, "permitrootlogin yes") {
		t.Errorf("the result detail does not carry both configurations: %q", result.Detail)
	}
	assertNoSSHDProvisioningAction(t, observation.Detail)
}

// TestLocalSshdReportsAnAbsentBinary is R-HR-18's absence case: an absent sshd
// binary is a measurement — a definite negative answer — and its text states
// that installing it is not part of this run.
func TestLocalSshdReportsAnAbsentBinary(t *testing.T) {
	files := &scriptedFS{files: map[string]string{testSSHDConfigPath: writtenConfigAgreeingWithEffect}}
	result := runLocalSSHD(t, localSSHDSeams(files, sshdRunner("permitrootlogin no\npasswordauthentication no\n")))

	observation, ok := observationByLabel(result, "binary present")
	if !ok {
		t.Fatalf("local.sshd reported no binary observation: %+v", result.Observations)
	}
	if observation.Resolution != probe.Measured {
		t.Errorf("absence resolution = %q, want %q", observation.Resolution, probe.Measured)
	}
	if observation.Verdict != probe.Fail || observation.Reason != probe.ReasonSSHDAbsent {
		t.Errorf("absence = (%q, %q), want (%q, %q)", observation.Verdict, observation.Reason, probe.Fail, probe.ReasonSSHDAbsent)
	}
	if observation.Target != testSSHDBinaryPath {
		t.Errorf("absence target = %q, want %q: the path that was checked", observation.Target, testSSHDBinaryPath)
	}
	if !strings.Contains(observation.Detail, "not present") {
		t.Errorf("the absence detail does not state what it looked for: %q", observation.Detail)
	}
	if !strings.Contains(observation.Detail, "not part of this run") {
		t.Errorf("the absence detail does not state that installing sshd is not part of this run: %q", observation.Detail)
	}
	if !strings.Contains(observation.Detail, "later slice") {
		t.Errorf("the absence detail does not name a later slice as the owner of the installation: %q", observation.Detail)
	}
	if result.Verdict != probe.Fail || result.Reason != probe.ReasonSSHDAbsent {
		t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Fail, probe.ReasonSSHDAbsent)
	}
	assertNoSSHDProvisioningAction(t, observation.Detail)
}

// windowsSSHDFiles is the filesystem one native-Windows case scripts: the written
// configuration at the Windows configuration path, the Windows sshd binary
// depending on the case, and both POSIX paths tripwired.
//
// The tripwire keeps issue #66's lesson as the test's mechanism: on Windows a path
// beginning with "/" resolves against the working directory, so a probe that asked
// for one would measure whichever directory the process started in. If the probe
// asks anyway, the seam answers with a failure that is not absence, so the
// observation cannot silently become a fabricated answer — and the exact call log
// the case asserts fails as well.
func windowsSSHDFiles(binaryPresent bool) *scriptedFS {
	files := &scriptedFS{
		files: map[string]string{testSSHDConfigPathWindows: writtenConfigAgreeingWithEffect},
		statErrs: map[string]error{
			testSSHDBinaryPath: errors.New("tripwire: the POSIX sshd path must not be stat'ed on a native-Windows node"),
		},
		readErrs: map[string]error{
			testSSHDConfigPath: errors.New("tripwire: the POSIX sshd configuration path must not be read on a native-Windows node"),
		},
	}
	if binaryPresent {
		files.files[testSSHDBinaryPathWindows] = ""
	}
	return files
}

// TestLocalSshdChecksTheWindowsPathsOnNativeWindows is issue #72's case: a
// native-Windows node has its own documented sshd locations, so the probe checks
// them and reports a real measurement instead of the not-applicable branch issues
// #66 and #67 needed while the POSIX path was the only documented one.
//
// The case scripts both outcomes — the binary present and the binary absent at the
// Windows path — and asserts the binary observation's target, the result's target
// and the effective-configuration observation's target all name the Windows path
// that was actually touched. The POSIX pair must not be touched: the filesystem
// seam answers any POSIX path with a non-absence failure, and the exact call log is
// asserted, which together catch the working-directory dependence the platform path
// exists to remove.
//
// The result's aggregate carries the service question issue #79 settled: on a
// native-Windows node that observation is not measured, so a present binary leaves
// the result indeterminate with the capability excluded, and an absent binary's
// measured negative still outranks it.
func TestLocalSshdChecksTheWindowsPathsOnNativeWindows(t *testing.T) {
	cases := []struct {
		name              string
		binaryPresent     bool
		wantBinaryVerdict probe.Verdict
		wantBinaryReason  probe.ReasonCode
		wantResultVerdict probe.Verdict
		wantResultReason  probe.ReasonCode
	}{
		{
			name:              "the sshd binary is present at the Windows path",
			binaryPresent:     true,
			wantBinaryVerdict: probe.Pass,
			wantBinaryReason:  probe.ReasonOK,
			wantResultVerdict: probe.Indeterminate,
			wantResultReason:  probe.ReasonCapabilityExcluded,
		},
		{
			name:              "the sshd binary is absent from the Windows path",
			binaryPresent:     false,
			wantBinaryVerdict: probe.Fail,
			wantBinaryReason:  probe.ReasonSSHDAbsent,
			wantResultVerdict: probe.Fail,
			wantResultReason:  probe.ReasonSSHDAbsent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := windowsSSHDFiles(tc.binaryPresent)
			seams := localSSHDSeams(files, sshdRunner("permitrootlogin no\npasswordauthentication no\n"))
			seams.Platform = scriptedPlatform{goos: "windows", arch: "amd64"}

			result := runLocalSSHD(t, seams)

			binary, ok := observationByLabel(result, "binary present")
			if !ok {
				t.Fatalf("local.sshd reported no binary observation: %+v", result.Observations)
			}
			if binary.Resolution != probe.Measured {
				t.Errorf("the Windows binary observation resolution = %q, want %q: the Windows path applies on this node, so the check is an attempt that produced an answer",
					binary.Resolution, probe.Measured)
			}
			if binary.Verdict != tc.wantBinaryVerdict || binary.Reason != tc.wantBinaryReason {
				t.Errorf("the Windows binary observation = (%q, %q), want (%q, %q)", binary.Verdict, binary.Reason, tc.wantBinaryVerdict, tc.wantBinaryReason)
			}
			if binary.Target != testSSHDBinaryPathWindows {
				t.Errorf("the Windows binary observation target = %q, want the path that was checked, %q", binary.Target, testSSHDBinaryPathWindows)
			}
			if !strings.Contains(binary.Detail, testSSHDBinaryPathWindows) {
				t.Errorf("the Windows binary detail does not name the path that was checked: %q", binary.Detail)
			}
			if result.Target != testSSHDBinaryPathWindows {
				t.Errorf("result target = %q, want the path that was checked, %q", result.Target, testSSHDBinaryPathWindows)
			}
			if result.Verdict != tc.wantResultVerdict || result.Reason != tc.wantResultReason {
				t.Errorf("result = (%q, %q), want (%q, %q): the effective configuration was measured and the service question was disclosed as excluded (issue #79)",
					result.Verdict, result.Reason, tc.wantResultVerdict, tc.wantResultReason)
			}

			config, ok := observationByLabel(result, "effective config")
			if !ok {
				t.Fatalf("local.sshd reported no effective-configuration observation: %+v", result.Observations)
			}
			if config.Resolution != probe.Measured || config.Verdict != probe.Pass || config.Reason != probe.ReasonOK {
				t.Errorf("the Windows effective-configuration observation = (%q, %q, %q), want (%q, %q, %q): the Windows configuration path was handed to the probe and agrees with the effective configuration",
					config.Resolution, config.Verdict, config.Reason, probe.Measured, probe.Pass, probe.ReasonOK)
			}
			if config.Target != testSSHDConfigPathWindows {
				t.Errorf("the Windows effective-configuration observation target = %q, want the path that was read, %q", config.Target, testSSHDConfigPathWindows)
			}
			if !strings.Contains(config.Detail, testSSHDConfigPathWindows) {
				t.Errorf("the Windows effective-configuration detail does not name the path that was read: %q", config.Detail)
			}

			wantFilesystem := []string{
				"Stat " + testSSHDBinaryPathWindows,
				"ReadFile " + testSSHDConfigPathWindows,
			}
			if !reflect.DeepEqual(files.calls, wantFilesystem) {
				t.Errorf("the probe's filesystem calls = %v, want exactly %v: on a native-Windows node only the Windows pair may be touched, and every POSIX path is tripwired", files.calls, wantFilesystem)
			}
		})
	}
}

// TestLocalSshdWindowsServiceQuestionIsDeclaredNotAsked is issue #79's case on
// the Windows seam: the service-state observation names the platform's own
// question — the state of the sshd service the OpenSSH Server capability
// installs, which a Get-Service-style query would report — and does not put it to
// the machine. No systemd invocation and no systemd unit name may appear anywhere
// in the run's detail or targets, and no runner is asked the question even when
// one is injected.
//
// The observation is `capability_excluded`: the vocabulary's own "the attempt was
// not made by design" fact, already used by this probe for a run with no command
// runner, and reused here so no observable, reason code or table row is added.
// No Windows interpretation of any answer is declared: a reading this slice
// cannot execute or test would manufacture a "not running" negative out of an
// answer it never read, and the slice that wires the platform's query declares
// the question and its interpretation together.
func TestLocalSshdWindowsServiceQuestionIsDeclaredNotAsked(t *testing.T) {
	const windowsServiceName = "sshd"

	cases := []struct {
		name   string
		runner probe.CommandRunner
	}{
		{
			name:   "a command runner is injected",
			runner: sshdRunner("permitrootlogin no\npasswordauthentication no\n"),
		},
		{
			name:   "no command runner is injected",
			runner: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seams := localSSHDSeams(windowsSSHDFiles(true), tc.runner)
			seams.Platform = scriptedPlatform{goos: "windows", arch: "amd64"}

			result := runLocalSSHD(t, seams)

			observation, ok := observationByLabel(result, "service state")
			if !ok {
				t.Fatalf("local.sshd reported no service-state observation: %+v", result.Observations)
			}
			if observation.Resolution != probe.NotMeasured || observation.Verdict != probe.Indeterminate {
				t.Errorf("the Windows service observation = (%q, %q), want (%q, %q): this slice does not put the question to the machine",
					observation.Resolution, observation.Verdict, probe.NotMeasured, probe.Indeterminate)
			}
			if observation.Reason != probe.ReasonCapabilityExcluded {
				t.Errorf("the Windows service observation reason = %q, want %q: the fact is an attempt that was not made by design",
					observation.Reason, probe.ReasonCapabilityExcluded)
			}
			if observation.Target != windowsServiceName {
				t.Errorf("the Windows service observation target = %q, want the service name %q: a systemd unit name is not a question this machine has",
					observation.Target, windowsServiceName)
			}
			for _, want := range []string{"Get-Service", windowsServiceName, "does not put that question to the machine"} {
				if !strings.Contains(observation.Detail, want) {
					t.Errorf("the Windows service detail does not name %q: %q", want, observation.Detail)
				}
			}

			// The systemd vocabulary must not survive anywhere in a native-Windows
			// run: not in an observation's target or detail, not in the probe's own
			// target and not in the joined detail.
			texts := []string{result.Target, result.Detail}
			for _, reported := range result.Observations {
				texts = append(texts, reported.Target, reported.Detail)
			}
			for _, text := range texts {
				if strings.Contains(text, "systemctl") {
					t.Errorf("a native-Windows run names the systemd invocation: %q", text)
				}
				if strings.Contains(text, ".service") {
					t.Errorf("a native-Windows run names a systemd unit: %q", text)
				}
			}

			if result.Verdict != probe.Indeterminate || result.Reason != probe.ReasonCapabilityExcluded {
				t.Errorf("the Windows result = (%q, %q), want (%q, %q): the service question was declared, not measured, beside a present binary",
					result.Verdict, result.Reason, probe.Indeterminate, probe.ReasonCapabilityExcluded)
			}

			if runner, ok := tc.runner.(*scriptedRunner); ok {
				wantCommands := []string{testSSHDConfigCommand}
				if !reflect.DeepEqual(runner.calls, wantCommands) {
					t.Errorf("the probe's commands = %v, want exactly %v: the Windows service question is declared, not asked", runner.calls, wantCommands)
				}
			}
		})
	}
}

// TestLocalSshdChecksThePosixPathOnEveryPosixPlatform is the other side of issue
// #66 and issue #72: the platform-dependent path choice must not catch a platform
// the POSIX path is for. The three platforms are scripted through the same seam,
// and WSL2 is the sharpest of them — it is a Windows machine running Linux, and its
// seam reports GOOS "linux", so a choice that read "Windows" from anything but the
// operating system would swallow it. Every case asserts the binary observation's
// target and the exact filesystem call log, and keeps today's outcome: present
// passes, absent is a measured `sshd_absent` failure.
func TestLocalSshdChecksThePosixPathOnEveryPosixPlatform(t *testing.T) {
	linux := scriptedPlatform{goos: "linux", arch: "x86_64", systemd: true}
	macos := scriptedPlatform{goos: "darwin", arch: "arm64"}
	wsl2 := scriptedPlatform{goos: "linux", arch: "x86_64", wsl2: true}

	cases := []struct {
		name        string
		platform    scriptedPlatform
		present     bool
		wantVerdict probe.Verdict
		wantReason  probe.ReasonCode
	}{
		{"linux and the binary is present", linux, true, probe.Pass, probe.ReasonOK},
		{"linux and the binary is absent", linux, false, probe.Fail, probe.ReasonSSHDAbsent},
		{"macos and the binary is present", macos, true, probe.Pass, probe.ReasonOK},
		{"macos and the binary is absent", macos, false, probe.Fail, probe.ReasonSSHDAbsent},
		{"wsl2 and the binary is present", wsl2, true, probe.Pass, probe.ReasonOK},
		{"wsl2 and the binary is absent", wsl2, false, probe.Fail, probe.ReasonSSHDAbsent},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := &scriptedFS{files: map[string]string{testSSHDConfigPath: writtenConfigAgreeingWithEffect}}
			if tc.present {
				files.files[testSSHDBinaryPath] = ""
			}
			seams := localSSHDSeams(files, sshdRunner("permitrootlogin no\npasswordauthentication no\n"))
			seams.Platform = tc.platform

			result := runLocalSSHD(t, seams)

			observation, ok := observationByLabel(result, "binary present")
			if !ok {
				t.Fatalf("local.sshd reported no binary observation: %+v", result.Observations)
			}
			if observation.Resolution != probe.Measured {
				t.Errorf("the binary observation resolution = %q, want %q: the POSIX path applies on this platform, so the check is a measurement", observation.Resolution, probe.Measured)
			}
			if observation.Verdict != tc.wantVerdict || observation.Reason != tc.wantReason {
				t.Errorf("the binary observation = (%q, %q), want (%q, %q)", observation.Verdict, observation.Reason, tc.wantVerdict, tc.wantReason)
			}
			if observation.Target != testSSHDBinaryPath {
				t.Errorf("the binary observation target = %q, want the POSIX path %q on this platform", observation.Target, testSSHDBinaryPath)
			}
			wantFilesystem := []string{"Stat " + testSSHDBinaryPath, "ReadFile " + testSSHDConfigPath}
			if !reflect.DeepEqual(files.calls, wantFilesystem) {
				t.Errorf("the probe's filesystem calls = %v, want exactly %v: every POSIX platform keeps the POSIX pair and touches no Windows path", files.calls, wantFilesystem)
			}
		})
	}
}

// TestLocalSshdNeverPassesWhenAnObservationWasNotMeasured triangulates the
// honesty rule the whole slice rests on: the probe may only pass when every one
// of its observations was measured, and its result is exactly the reduction of
// those observations. Each case scripts a different reason for an observation to
// be missing, and none of them may produce a pass.
func TestLocalSshdNeverPassesWhenAnObservationWasNotMeasured(t *testing.T) {
	agreeing := writtenConfigAgreeingWithEffect
	agreeingEffective := "permitrootlogin no\npasswordauthentication no\n"
	withoutBinary := &scriptedFS{files: map[string]string{testSSHDConfigPath: agreeing}}
	failingStat := &scriptedFS{
		files:    map[string]string{testSSHDConfigPath: agreeing},
		statErrs: map[string]error{testSSHDBinaryPath: errors.New("stat /usr/sbin/sshd: permission denied")},
	}
	inactiveService := &scriptedRunner{answers: map[string]scriptedAnswer{
		testSSHDServiceCommand: {stdout: "inactive\ninactive\n"},
		testSSHDConfigCommand:  {stdout: agreeingEffective},
	}}

	cases := []struct {
		name  string
		seams probe.Seams
	}{
		{"no command runner is injected", localSSHDSeams(&scriptedFS{files: sshdFiles(agreeing).files}, nil)},
		{"the command seam denies everything", localSSHDSeams(&scriptedFS{files: sshdFiles(agreeing).files}, probe.DenyAllSeams().CommandRunner)},
		{"no filesystem seam is injected", localSSHDSeams(nil, sshdRunner(agreeingEffective))},
		{"the binary is absent", localSSHDSeams(withoutBinary, sshdRunner(agreeingEffective))},
		{"the binary check fails for a reason other than absence", localSSHDSeams(failingStat, sshdRunner(agreeingEffective))},
		{"the service is not running", localSSHDSeams(&scriptedFS{files: sshdFiles(agreeing).files}, inactiveService)},
	}

	passes := 0
	unmeasured := 0
	failures := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runLocalSSHD(t, tc.seams)

			measured := true
			for _, observation := range result.Observations {
				if !holds(observation) {
					t.Errorf("observation %q does not satisfy the measurement vocabulary's invariant: %+v", observation.Label, observation)
				}
				if observation.Resolution != probe.Measured {
					measured = false
				}
				if strings.TrimSpace(observation.Detail) == "" {
					t.Errorf("observation %q carries no verbatim detail", observation.Label)
				}
				assertNoSSHDProvisioningAction(t, observation.Detail)
			}
			if !measured && result.Verdict == probe.Pass {
				t.Fatalf("the probe passed while an observation was not measured: %+v", result.Observations)
			}
			verdict, reason := probe.Aggregate(result.Observations)
			if result.Verdict != verdict || result.Reason != reason {
				t.Fatalf("the result's reduction %s/%s disagrees with Aggregate over its observations %s/%s", result.Verdict, result.Reason, verdict, reason)
			}
		})
		// The case-level counters are read after the sub-test so the table's
		// composition is itself asserted: a table where every case failed would
		// not exercise the pass rule at all.
		result := runLocalSSHD(t, tc.seams)
		if result.Verdict == probe.Pass {
			passes++
		} else {
			if result.Verdict == probe.Fail {
				failures++
			}
			unmeasured++
		}
	}

	if passes != 0 {
		t.Errorf("%d of the %d unmeasured cases produced a pass, want 0", passes, len(cases))
	}
	if unmeasured != len(cases) {
		t.Errorf("%d of the %d cases were expected to report a gap, got %d", unmeasured, len(cases), unmeasured)
	}
	if failures == 0 {
		t.Error("no case in the table produced a measured failure, so the table does not cover the failure path")
	}
}

// TestLocalSshdDegradesWhenTheCommandCapabilityIsMissing is the first half of
// the degradation row: with no command runner injected the two command-derived
// observations are not measured, the capability each needed is named, and the
// probe is never a pass (design §6.2, design §5.1 obligation 2).
func TestLocalSshdDegradesWhenTheCommandCapabilityIsMissing(t *testing.T) {
	result := runLocalSSHD(t, localSSHDSeams(
		&scriptedFS{files: sshdFiles(writtenConfigAgreeingWithEffect).files},
		nil,
	))

	if result.Verdict == probe.Pass {
		t.Fatal("the probe passed with no command capability at all")
	}
	if result.Verdict != probe.Indeterminate {
		t.Errorf("result verdict = %q, want %q: an excluded capability is not a negative answer", result.Verdict, probe.Indeterminate)
	}

	for _, want := range []struct{ label, capability string }{
		{"service state", testSSHDServiceCommand},
		{"effective config", testSSHDConfigCommand},
	} {
		observation, ok := observationByLabel(result, want.label)
		if !ok {
			t.Fatalf("local.sshd reported no %q observation: %+v", want.label, result.Observations)
		}
		if observation.Resolution != probe.NotMeasured || observation.Verdict != probe.Indeterminate {
			t.Errorf("%q = (%q, %q), want (%q, %q)", want.label, observation.Resolution, observation.Verdict, probe.NotMeasured, probe.Indeterminate)
		}
		if observation.Reason != probe.ReasonCapabilityExcluded {
			t.Errorf("%q reason = %q, want %q", want.label, observation.Reason, probe.ReasonCapabilityExcluded)
		}
		if !strings.Contains(observation.Detail, want.capability) {
			t.Errorf("%q detail does not name the missing capability %q: %q", want.label, want.capability, observation.Detail)
		}
	}

	binaryObservation, ok := observationByLabel(result, "binary present")
	if !ok {
		t.Fatalf("local.sshd reported no binary observation: %+v", result.Observations)
	}
	if binaryObservation.Resolution != probe.Measured || binaryObservation.Verdict != probe.Pass {
		t.Errorf("the filesystem-derived observation = (%q, %q), want (%q, %q): its capability was injected", binaryObservation.Resolution, binaryObservation.Verdict, probe.Measured, probe.Pass)
	}
}

// TestLocalSshdDegradesWhenTheCommandSeamDenies is the second half of the
// degradation row: an injected runner that denies the execution must be
// distinguishable from a capability that was never injected, and neither may be
// reported as a pass.
func TestLocalSshdDegradesWhenTheCommandSeamDenies(t *testing.T) {
	missing := runLocalSSHD(t, localSSHDSeams(&scriptedFS{files: sshdFiles(writtenConfigAgreeingWithEffect).files}, nil))
	denied := runLocalSSHD(t, localSSHDSeams(
		&scriptedFS{files: sshdFiles(writtenConfigAgreeingWithEffect).files},
		probe.DenyAllSeams().CommandRunner,
	))

	if denied.Verdict == probe.Pass {
		t.Fatal("the probe passed with a deny-all command seam")
	}
	for _, label := range []string{"service state", "effective config"} {
		observation, ok := observationByLabel(denied, label)
		if !ok {
			t.Fatalf("local.sshd reported no %q observation: %+v", label, denied.Observations)
		}
		if observation.Resolution != probe.NotMeasured || observation.Verdict != probe.Indeterminate {
			t.Errorf("%q = (%q, %q), want (%q, %q)", label, observation.Resolution, observation.Verdict, probe.NotMeasured, probe.Indeterminate)
		}
		if observation.Reason != probe.ReasonCommandDenied {
			t.Errorf("%q reason = %q, want %q: the seam denied the execution", label, observation.Reason, probe.ReasonCommandDenied)
		}
		if !strings.Contains(observation.Detail, "denied") {
			t.Errorf("%q detail does not name the denial: %q", label, observation.Detail)
		}
	}

	if missing.Reason == denied.Reason {
		t.Fatalf("a missing capability and a denying seam report the same reason %q; design §5.1 obligation 2 requires two", missing.Reason)
	}
}

// TestLocalSshdServiceStateIsSeparatelyReportable asserts that the service
// observation answers on its own: an active unit passes, and a unit that is not
// running is a measured negative rather than a guess.
//
// The closed reason-code set holds no code for "installed but not running", so
// the not-running answer reuses `sshd_absent`, the vocabulary's own "this node
// is not serving sshd" code; the observation's label and detail name the service
// state verbatim, and the detail deliberately does not claim that anything needs
// installing. Recorded as a deviation, not as a silent choice.
func TestLocalSshdServiceStateIsSeparatelyReportable(t *testing.T) {
	effective := "permitrootlogin no\npasswordauthentication no\n"

	cases := []struct {
		name        string
		service     scriptedAnswer
		wantVerdict probe.Verdict
		wantReason  probe.ReasonCode
	}{
		{
			name:        "the service manager reports the unit active",
			service:     scriptedAnswer{stdout: "active\ninactive\n"},
			wantVerdict: probe.Pass, wantReason: probe.ReasonOK,
		},
		{
			// `systemctl is-active` exits non-zero when no queried unit is
			// active, so the answer arrives with an error beside it. The output
			// is still the measurement.
			name:        "the unit is not running and the query exits non-zero",
			service:     scriptedAnswer{stdout: "inactive\ninactive\n", err: errors.New("exit status 3")},
			wantVerdict: probe.Fail, wantReason: probe.ReasonSSHDAbsent,
		},
		{
			name:        "the unit is not running and the query exits zero",
			service:     scriptedAnswer{stdout: "inactive\ninactive\n"},
			wantVerdict: probe.Fail, wantReason: probe.ReasonSSHDAbsent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &scriptedRunner{answers: map[string]scriptedAnswer{
				testSSHDServiceCommand: tc.service,
				testSSHDConfigCommand:  {stdout: effective},
			}}
			result := runLocalSSHD(t, localSSHDSeams(sshdFiles(writtenConfigAgreeingWithEffect), runner))

			observation, ok := observationByLabel(result, "service state")
			if !ok {
				t.Fatalf("local.sshd reported no service-state observation: %+v", result.Observations)
			}
			if observation.Resolution != probe.Measured {
				t.Errorf("service state resolution = %q, want %q: the service manager answered", observation.Resolution, probe.Measured)
			}
			if observation.Verdict != tc.wantVerdict || observation.Reason != tc.wantReason {
				t.Errorf("service state = (%q, %q), want (%q, %q)", observation.Verdict, observation.Reason, tc.wantVerdict, tc.wantReason)
			}
			if tc.wantVerdict == probe.Fail {
				if !strings.Contains(observation.Detail, "inactive") {
					t.Errorf("the not-running detail does not carry the service manager's own answer: %q", observation.Detail)
				}
				if strings.Contains(observation.Detail, "install") {
					t.Errorf("the not-running detail claims something needs installing: %q", observation.Detail)
				}
			}
			assertNoSSHDProvisioningAction(t, observation.Detail)
		})
	}
}

// TestLocalSshdAsksThePosixServiceQuestionUnchanged pins the POSIX half of issue
// #79's platform choice byte for byte: the same `systemctl is-active` invocation
// for the same two unit names, the same joined target and the same wording, on
// every platform the POSIX question describes — Linux, macOS, WSL2 and a run with
// no platform seam. The service question is platform-chosen now, so this case is
// the control that the choice did not touch the platforms it already described.
func TestLocalSshdAsksThePosixServiceQuestionUnchanged(t *testing.T) {
	const wantDetail = `the sshd service is running: systemctl is-active sshd.service ssh.service reported "active" for sshd.service,ssh.service`

	cases := []struct {
		name     string
		platform probe.Platform
	}{
		{"linux", scriptedPlatform{goos: "linux", arch: "x86_64", systemd: true}},
		{"macos", scriptedPlatform{goos: "darwin", arch: "arm64"}},
		{"wsl2", scriptedPlatform{goos: "linux", arch: "x86_64", wsl2: true}},
		{"no platform seam", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &scriptedRunner{answers: map[string]scriptedAnswer{
				testSSHDServiceCommand: {stdout: "active\n"},
				testSSHDConfigCommand:  {stdout: "permitrootlogin no\npasswordauthentication no\n"},
			}}
			seams := localSSHDSeams(sshdFiles(writtenConfigAgreeingWithEffect), runner)
			seams.Platform = tc.platform

			result := runLocalSSHD(t, seams)

			observation, ok := observationByLabel(result, "service state")
			if !ok {
				t.Fatalf("local.sshd reported no service-state observation: %+v", result.Observations)
			}
			if observation.Target != "sshd.service,ssh.service" {
				t.Errorf("the POSIX service observation target = %q, want %q", observation.Target, "sshd.service,ssh.service")
			}
			if observation.Detail != wantDetail {
				t.Errorf("the POSIX service wording changed:\n got %q\nwant %q", observation.Detail, wantDetail)
			}
			if observation.Resolution != probe.Measured || observation.Verdict != probe.Pass || observation.Reason != probe.ReasonOK {
				t.Errorf("the POSIX service observation = (%q, %q, %q), want (%q, %q, %q)",
					observation.Resolution, observation.Verdict, observation.Reason, probe.Measured, probe.Pass, probe.ReasonOK)
			}
			if result.Verdict != probe.Pass || result.Reason != probe.ReasonOK {
				t.Errorf("the POSIX result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Pass, probe.ReasonOK)
			}
			wantCommands := []string{testSSHDServiceCommand, testSSHDConfigCommand}
			if !reflect.DeepEqual(runner.calls, wantCommands) {
				t.Errorf("the probe's commands = %v, want exactly %v in that order", runner.calls, wantCommands)
			}
		})
	}
}

// TestLocalSshdUnclassifiableLocalInputIsUnresolved asserts the error path the
// table's internal-failure row exists for: a local input that fails for a reason
// other than absence is an attempt that produced nothing classifiable, never a
// fabricated absence and never a pass.
func TestLocalSshdUnclassifiableLocalInputIsUnresolved(t *testing.T) {
	agreeing := writtenConfigAgreeingWithEffect
	effective := "permitrootlogin no\npasswordauthentication no\n"

	cases := []struct {
		name     string
		fsSeam   probe.FS
		wantText string
	}{
		{
			name: "the binary check fails for a reason other than absence",
			fsSeam: &scriptedFS{
				files:    map[string]string{testSSHDConfigPath: agreeing},
				statErrs: map[string]error{testSSHDBinaryPath: errors.New("stat /usr/sbin/sshd: permission denied")},
			},
			wantText: "permission denied",
		},
		{
			name: "the written configuration cannot be read",
			fsSeam: &scriptedFS{
				files:    map[string]string{testSSHDBinaryPath: ""},
				readErrs: map[string]error{testSSHDConfigPath: errors.New("open /etc/ssh/sshd_config: permission denied")},
			},
			wantText: "permission denied",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runLocalSSHD(t, localSSHDSeams(tc.fsSeam, sshdRunner(effective)))

			found := false
			for _, observation := range result.Observations {
				if observation.Reason != probe.ReasonInternalError {
					continue
				}
				found = true
				if observation.Resolution != probe.Unresolved {
					t.Errorf("%q resolution = %q, want %q: the attempt was made and produced no classifiable answer", observation.Label, observation.Resolution, probe.Unresolved)
				}
				if observation.Verdict == probe.Pass {
					t.Errorf("%q was reported as a pass", observation.Label)
				}
				if !strings.Contains(observation.Detail, tc.wantText) {
					t.Errorf("%q detail does not carry the failure verbatim: %q", observation.Label, observation.Detail)
				}
			}
			if !found {
				t.Fatalf("no observation reported the unclassifiable input: %+v", result.Observations)
			}
			if result.Verdict == probe.Pass {
				t.Fatal("the probe passed while a local input failed")
			}
		})
	}
}

// TestLocalSshdReadsOnlyInjectedReaders asserts the second half of the
// triangulation row. The probe's entire local input is the two seams, so the
// calls it makes are asserted exactly: one existence check of the documented
// binary path, one read-only service query, one read of the documented written
// configuration and one effective-configuration command. Nothing else is
// touched — no environment variable, no other path — and the answers a case
// scripts are the answers the probe reports, which is what proves the seams are
// the only inputs.
func TestLocalSshdReadsOnlyInjectedReaders(t *testing.T) {
	// A real home directory would tempt a probe to look outside its seams; the
	// case makes one that contains nothing the probe could use.
	t.Setenv("HOME", t.TempDir())

	written := "Port 2222\nPermitRootLogin no\n"
	effective := "port 2222\npermitrootlogin no\n"
	files := sshdFiles(written)
	runner := sshdRunner(effective)

	result := runLocalSSHD(t, localSSHDSeams(files, runner))

	wantFilesystem := []string{
		"Stat " + testSSHDBinaryPath,
		"ReadFile " + testSSHDConfigPath,
	}
	if !reflect.DeepEqual(files.calls, wantFilesystem) {
		t.Errorf("the probe's filesystem calls = %v, want exactly %v in that order", files.calls, wantFilesystem)
	}
	wantCommands := []string{testSSHDServiceCommand, testSSHDConfigCommand}
	if !reflect.DeepEqual(runner.calls, wantCommands) {
		t.Errorf("the probe's commands = %v, want exactly %v in that order", runner.calls, wantCommands)
	}

	if !strings.Contains(result.Detail, "Port 2222") {
		t.Errorf("the probe's detail does not carry the written configuration it was handed: %q", result.Detail)
	}
	if !strings.Contains(result.Detail, "port 2222") {
		t.Errorf("the probe's detail does not carry the effective configuration it was handed: %q", result.Detail)
	}
	if result.Verdict != probe.Pass {
		t.Errorf("result verdict = %q, want %q: the two handed configurations agree", result.Verdict, probe.Pass)
	}
}

// observationByLabel finds one observation of a result by its label.
func observationByLabel(result probe.Result, label string) (probe.Observation, bool) {
	for _, observation := range result.Observations {
		if observation.Label == label {
			return observation, true
		}
	}
	return probe.Observation{}, false
}
