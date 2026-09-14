package probe

// This file is the local probe group of design §7. This slice lands `local.env`
// and nothing else local; `local.sshd` follows in its own slice.
//
// `local.env` answers one question: what is this machine? It classifies the
// platform the run is happening on — Linux, macOS, WSL2 or native Windows — and
// reports the architecture it was classified with (R-HR-29), and it refuses a
// native-Windows node while naming WSL2 as the supported path (R-HR-30).
//
// The classification is a pure function of the injected Platform seam. This file
// opens no file, reads no environment variable and inspects no kernel interface
// of its own: the seam owns those, which is what lets a test script Linux, macOS,
// WSL2 and native Windows without being on any of them, and what keeps the probe
// from asking the machine a question the run was never given a capability for.
// Whether the system is WSL2 and whether systemd is running are seam signals, not
// process lookups: reading them directly would classify the machine the test
// happens to run on instead of the machine the case describes.
//
// Provisioning is outside this slice entirely: `local.env` reports what the
// machine is and offers nothing to do about it. It installs, reconfigures,
// persists and remediates nothing (R-HR-29), and the only later work it names is
// work it states belongs to a later slice rather than to this run.
//
// WSL2 output is deliberately thin (RG-4). Only the Microsoft-documented
// vmIdleTimeout semantics are repeated: milliseconds of idle before the instance
// is shut down, a default of 60000, and Windows 11 only. The child-of-init
// shutdown rule and the `vmIdleTimeout=-1` sentinel are not documented in the
// reference material this slice could verify, so they appear nowhere in this
// file's output; a test asserts their absence from the text a run produces.

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// NodePlatform is the machine classification `local.env` reports. Its values are
// the ones design §3.3 records for the payload's `node.platform`, so a consumer
// echoes a classification this slice measured instead of inventing a second
// vocabulary for the same fact.
type NodePlatform string

const (
	// NodePlatformLinux is a Linux node with no WSL2 signal.
	NodePlatformLinux NodePlatform = "linux"
	// NodePlatformMacOS is a macOS node.
	NodePlatformMacOS NodePlatform = "macos"
	// NodePlatformWSL2 is a Linux node running inside WSL2. WSL2 is supported as
	// a node (R-HR-29); this slice detects it and states no more than that.
	NodePlatformWSL2 NodePlatform = "wsl2"
	// NodePlatformWindowsNative is a node running Windows itself, which cannot
	// host a supported node and is therefore refused (R-HR-30).
	NodePlatformWindowsNative NodePlatform = "windows-native"
	// NodePlatformUnknown is the classification reported when the machine could
	// not be classified at all. It is a reported absence, never a default guess.
	NodePlatformUnknown NodePlatform = "unknown"
)

const (
	// unknownSignal is the value the Platform seam documents for a system or an
	// architecture it could not identify (see DenyAllSeams): "the signal was not
	// reported", which is not the same claim as any real value.
	unknownSignal = "unknown"
	// goosLinux, goosDarwin and goosWindows are Go's own operating-system
	// vocabulary, which is the vocabulary the Platform seam reports.
	goosLinux   = "linux"
	goosDarwin  = "darwin"
	goosWindows = "windows"
	// platformFactLabel names the one fact local.env measures. It is the
	// observation's label rather than its target, because the target carries the
	// classification the machine was placed in.
	platformFactLabel = "platform"
)

// PlatformIdentitySeparator joins the two halves of the value `local.env` reports
// as its observation's target: the classification and the architecture, as in
// "linux/aarch64". It is exported because both halves are facts a consumer needs
// apart — the payload's `node.platform` and `node.arch` (design §3.3) are exactly
// those two values — and a format known only to this file would have to be
// re-derived by string surgery somewhere else.
const PlatformIdentitySeparator = "/"

// NodePlatformIdentity returns the target `local.env` reports for a
// classification: "<platform><separator><architecture>", with the unknown value
// in a half the platform seam did not report.
//
// Both halves are always present. A consumer must never have to tell "no target
// was attempted" from "the architecture was unknown", and an empty half would
// force exactly that distinction onto every reader of the value.
func NodePlatformIdentity(platform NodePlatform, arch string) string {
	return string(platform) + PlatformIdentitySeparator + archOrUnknown(arch)
}

// SplitNodePlatformIdentity is NodePlatformIdentity's inverse: it splits an
// identity into the classification and the architecture it was reported with.
// It reports false for a value that is not an identity this package produces —
// a missing separator and an empty half are both ill-formed — so a consumer
// cannot read a classification out of a string that never carried one.
func SplitNodePlatformIdentity(identity string) (NodePlatform, string, bool) {
	platform, arch, found := strings.Cut(identity, PlatformIdentitySeparator)
	if !found || platform == "" || arch == "" {
		return "", "", false
	}
	return NodePlatform(platform), arch, true
}

// archOrUnknown reports the architecture verbatim, or the unknown value when the
// platform seam reported none. An architecture is never invented: the node's
// architecture decides which binaries it can run (PRD §13), so a guessed one
// would be worse than an absent one.
func archOrUnknown(arch string) string {
	if arch == "" {
		return unknownSignal
	}
	return arch
}

// platformSignals is one reading of the Platform seam. It is captured once so the
// classification is a pure function of one value instead of a sequence of seam
// calls whose answers could differ between readings.
type platformSignals struct {
	// goos is the operating system the seam reported.
	goos string
	// arch is the architecture the seam reported, or the unknown value.
	arch string
	// wsl2 is the seam's WSL2 signal.
	wsl2 bool
	// systemd is the seam's service-manager signal.
	systemd bool
}

// archReported reports whether the seam identified the architecture. An absent
// architecture is a missing signal, not an empty string that can be padded over:
// the classification below refuses to call a node supported when the half of the
// answer that decides binary compatibility was not measured.
func (s platformSignals) archReported() bool {
	return s.arch != "" && s.arch != unknownSignal
}

// localEnv is the `local.env` probe of PRD §5.1. It carries the run's seams and
// nothing else, so two runs cannot share one measurement's inputs.
type localEnv struct {
	seams Seams
}

// newLocalEnv builds the probe from the run's injected seams. Only the platform
// seam is read; the rest travel with the run and are simply not this probe's
// question.
func newLocalEnv(seams Seams) Probe { return &localEnv{seams: seams} }

// Name is the probe's stable identifier, declared once in registry.go.
func (p *localEnv) Name() string { return probeNameLocalEnv }

// Kind is the question family the probe belongs to.
func (p *localEnv) Kind() ProbeKind { return ProbeLocal }

// Run classifies the machine and reports the classification as one observation.
//
// The result's target is the observation's classification identity rather than an
// address: a probe that measures this machine dials nothing, and the identity —
// which platform, which architecture — is what it measured instead.
//
// It deliberately does not read its context. The classification performs no I/O
// and cannot block, so there is nothing a cancelled context could stop; a run
// cancelled while this probe is in flight is reported by the runner with the
// runner's own reason, because run_cancelled is runner-classified and a probe
// that produced it would be claiming a fact about the run that only the runner
// owns (design §5.1).
//
// The elapsed value comes from the run's injected clock when it has one, so only
// the injected clock moves a timestamp (design §3.3); when the run supplied no
// clock, the elapsed value is left unset and the runner fills it from its own.
func (p *localEnv) Run(context.Context) Result {
	started := p.now()
	observation := p.observe()
	verdict, reason := Aggregate([]Observation{observation})
	return Result{
		Probe:        p.Name(),
		Kind:         p.Kind(),
		Target:       observation.Target,
		Verdict:      verdict,
		Reason:       reason,
		Detail:       observation.Detail,
		Elapsed:      p.now().Sub(started),
		Observations: []Observation{observation},
	}
}

// now reads the run's clock, or the zero time when the run injected no clock
// seam. The runner fills an elapsed value the probe left unset, so a probe
// without a clock reports zero rather than reaching for the wall clock, which
// would put an uninjectable timestamp into the output.
func (p *localEnv) now() time.Time {
	if p.seams.Clock == nil {
		return time.Time{}
	}
	return p.seams.Clock.Now()
}

// observe reads the platform seam once and classifies what it reported.
//
// A missing seam is not a machine that failed to answer: it is a run that was
// never given the capability, so the observation says exactly that and no
// platform is assumed.
func (p *localEnv) observe() Observation {
	platform := p.seams.Platform
	if platform == nil {
		return unknownPlatformObservation("no platform seam is injected for this run, so the machine cannot be classified; no platform is assumed")
	}
	return classifyPlatform(platformSignals{
		goos:    platform.GOOS(),
		arch:    platform.Arch(),
		wsl2:    platform.WSL2(),
		systemd: platform.Systemd(),
	})
}

// classifyPlatform turns one reading of the platform seam into the observation
// `local.env` reports.
//
// The branch order is the order of the answers. Native Windows is decided first
// and on the operating system alone, because the refusal does not depend on the
// architecture or on the service manager. WSL2 is decided before plain Linux
// because a WSL2 instance also reports GOOS "linux", so a later plain-Linux
// branch would swallow it. macOS comes last of the supported platforms. Every
// remaining case — an operating system outside the supported set, or a supported
// operating system whose architecture was not reported — degrades to an explicit
// unknown rather than to a default, because a node we cannot characterise is not
// a node we may call supported (R-HR-29).
func classifyPlatform(signals platformSignals) Observation {
	arch := archOrUnknown(signals.arch)

	switch {
	case signals.goos == goosWindows:
		// The refusal is measured: the operating system answered, and the answer
		// is that this tool will not operate on it. It is reported as the
		// measurement's own negative answer rather than as a usage or internal
		// error (R-HR-30), and its text names the supported path so a reader is
		// never left to guess one. The architecture is reported verbatim when the
		// seam measured it and as unknown otherwise: the refusal rests on the
		// operating system, not on the architecture, so an unreported architecture
		// neither weakens the refusal nor permits a fabricated one.
		return Observe(platformFactLabel,
			NodePlatformIdentity(NodePlatformWindowsNative, arch),
			PurposePlatformClassification,
			RawObservation{Kind: ObsNodePlatformUnsupported, Wording: windowsWording(signals)})

	case signals.goos == goosLinux && signals.wsl2 && signals.archReported():
		return Observe(platformFactLabel,
			NodePlatformIdentity(NodePlatformWSL2, arch),
			PurposePlatformClassification,
			RawObservation{Kind: ObsPlatformSignalsClassified, Wording: wsl2Wording(signals)})

	case signals.goos == goosLinux && signals.archReported():
		return Observe(platformFactLabel,
			NodePlatformIdentity(NodePlatformLinux, arch),
			PurposePlatformClassification,
			RawObservation{Kind: ObsPlatformSignalsClassified, Wording: linuxWording(signals)})

	case signals.goos == goosDarwin && signals.archReported():
		return Observe(platformFactLabel,
			NodePlatformIdentity(NodePlatformMacOS, arch),
			PurposePlatformClassification,
			RawObservation{Kind: ObsPlatformSignalsClassified, Wording: macosWording(signals)})

	case signals.goos == goosLinux || signals.goos == goosDarwin:
		// The operating system is one this tool supports; the architecture is the
		// half that was not reported. Calling the node supported anyway would
		// present a classification whose binary-compatibility half is missing.
		return unknownPlatformObservation(fmt.Sprintf("the platform seam reported %s but no architecture (%q): a node whose architecture was not reported is not classified as supported, because the node's architecture decides which binaries it can run; no architecture is assumed",
			signals.goos, signals.arch))

	default:
		// An operating system outside the supported set, or no operating-system
		// signal at all. The signal is quoted verbatim so the reader sees what the
		// seam actually said instead of a summary of it.
		return unknownPlatformObservation(fmt.Sprintf("the platform signals (GOOS %q, architecture %q) match no supported classification (linux, macOS, wsl2 or native Windows); no platform is assumed and no default is guessed",
			signals.goos, arch))
	}
}

// linuxWording is the text `local.env` reports for a Linux node: the
// classification, the architecture, and the service-manager signal the platform
// seam reported. It states what the run did — reach a conclusion from the
// signals it read — and nothing it could do about the machine.
func linuxWording(signals platformSignals) string {
	service := "systemd is the running service manager"
	if !signals.systemd {
		service = "systemd is not the running service manager"
	}
	return fmt.Sprintf("linux node (architecture %q): %s; this run detects and reports the environment and changes nothing",
		signals.arch, service)
}

// macosWording is the text `local.env` reports for a macOS node. macOS is
// supervised by launchd, which is why the service-manager question is answered by
// the classification itself rather than by the systemd signal.
func macosWording(signals platformSignals) string {
	return fmt.Sprintf("macos node (architecture %q): launchd is the service manager on macOS; this run detects and reports the environment and changes nothing",
		signals.arch)
}

// windowsWording is the text `local.env` reports for a node running Windows
// itself. A native-Windows node cannot host a supported node, so the
// classification is a refusal: it says what this tool will not do, names WSL2 as
// the supported path, and claims no change to the machine it refused to operate
// on (R-HR-30).
func windowsWording(signals platformSignals) string {
	return fmt.Sprintf("native Windows node (GOOS %q, architecture %q): this tool does not operate on Windows itself, and WSL2 is the supported path on a Windows machine; nothing was changed",
		signals.goos, archOrUnknown(signals.arch))
}

// wsl2Wording is the text `local.env` reports for a WSL2 node.
//
// Two rules shape it. First, it repeats only semantics that are documented:
// vmIdleTimeout counts milliseconds of idle, its default is 60000, and the
// setting exists only on Windows 11 (RG-4). The child-of-init shutdown rule and
// the `vmIdleTimeout=-1` sentinel are not documented in anything this slice could
// verify, so they appear nowhere here; their measurement belongs to another
// slice, and a test asserts their absence. Second, when systemd is not the
// running service manager the text says so and states that enabling it is work
// owned by a later slice: detection, no repair, and no claim that any keepalive,
// watchdog or persistence behaviour exists (R-HR-30, PRD §6.2).
func wsl2Wording(signals platformSignals) string {
	service := "systemd is the running service manager"
	if !signals.systemd {
		service = "systemd is not the running service manager in this instance, and enabling systemd is work owned by a later slice rather than by this run"
	}
	return fmt.Sprintf("wsl2 node (architecture %q): %s; vmIdleTimeout is documented as milliseconds of idle before the WSL2 instance is shut down, with a default of 60000, and the setting exists only on Windows 11; this run detects and reports the environment and changes nothing",
		signals.arch, service)
}

// unknownPlatformObservation builds the observation for a machine this probe
// could not classify. It is unresolved and indeterminate — never a pass, and
// never a fail that would claim a definite negative the signals do not support —
// and its target is the unknown identity rather than an empty string, so every
// consumer reads one shape.
func unknownPlatformObservation(wording string) Observation {
	return Observe(platformFactLabel,
		NodePlatformIdentity(NodePlatformUnknown, unknownSignal),
		PurposePlatformClassification,
		RawObservation{Kind: ObsPlatformSignalsUnknown, Wording: wording})
}
