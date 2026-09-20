package probe

// This file is the local probe group of design §7: `local.env` (what this machine
// is) and `local.sshd` (what this machine's sshd is). The two halves answer two
// different questions and share only the rule that shapes the whole package:
// every local fact is read through an injected seam, never from the process.
//
// `local.env` answers one question: what is this machine? It classifies the
// platform the run is happening on — Linux, macOS, WSL2 or native Windows — and
// reports the architecture it was classified with (R-HR-29), and it refuses a
// native-Windows node while stating that the refusal is this tool's provisioning
// scope and naming WSL2 as the Windows path it handles today (R-HR-30).
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
	"errors"
	"fmt"
	"io/fs"
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
	// NodePlatformWindowsNative is a node running Windows itself. Upstream Herdr
	// supports a Windows server as of 0.9.1, but this tool does not provision a
	// native Windows node yet, so the classification is refused (R-HR-30).
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
// question. The run's declared target input is ignored: a probe that measures this
// machine declares the local protocol and no remote target (targets.go), so the
// hub address and the target overrides are nothing to do with it.
func newLocalEnv(seams Seams, _ TargetInput) Probe { return &localEnv{seams: seams} }

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
func (p *localEnv) now() time.Time { return runClockNow(p.seams) }

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
		// is that this tool does not provision this node. It is reported as the
		// measurement's own negative answer rather than as a usage or internal
		// error (R-HR-30), and its text names WSL2 as the Windows path this tool
		// handles today so a reader is never left to guess one. The architecture
		// is reported verbatim when the seam measured it and as unknown
		// otherwise: the refusal rests on the operating system, not on the
		// architecture, so an unreported architecture neither weakens the
		// refusal nor permits a fabricated one.
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
// itself. The classification is still a refusal, but its reason is this tool's
// scope and not the platform's: upstream Herdr supports a Windows server as of
// 0.9.1 — measured 2026-09-20, when `machine add` saved a Windows 11 24H2 host
// and the hub listed the agents running natively on it — while this tool
// provisions Linux/systemd, macOS/launchd and WSL2 nodes only. The text names
// WSL2 as the Windows path this tool handles today and claims no change to the
// machine it refused to operate on (R-HR-30).
//
// The refusal rests on the classification alone, which leaves one gap this text
// does not close: the probe does not measure the node's Herdr version, so a node
// running a server older than 0.9.1 is refused here although it also cannot host
// a saved-machine connection. Telling those two cases apart would need a version
// measurement this slice does not make.
func windowsWording(signals platformSignals) string {
	return fmt.Sprintf("native Windows node (GOOS %q, architecture %q): upstream Herdr supports a Windows server as of 0.9.1, but this tool does not provision a native Windows node yet, and WSL2 is the Windows path this tool handles today; nothing was changed",
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

// --- `local.sshd` -------------------------------------------------------------
//
// `local.sshd` answers one question about this machine (R-HR-18): is an sshd
// serving here, and is the configuration in force the one that was written? It
// answers it as three separately reportable observations, because the three
// answers fail differently and a reader has to be able to tell them apart (DEV-2):
//
//  1. `binary present` — an sshd binary at the documented path, read through the
//     injected FS seam. Absence is a measured negative, and its detail states that
//     installing sshd is not part of this run: this slice detects, it does not
//     provision (R-HR-02, R-HR-29).
//  2. `service state` — a read-only service-manager query through the injected
//     CommandRunner. The query asks for the unit's state and changes nothing.
//  3. `effective config` — the written configuration read through the FS seam
//     compared with the configuration actually in force as `sshd -T` reports it
//     through the CommandRunner. When the two disagree the observation is a
//     measured negative and the detail carries both configurations verbatim,
//     which is what R-HR-18 requires and PRD §13's "show both" means.
//
// Degradation is the point of this half of the file (design §5.1 obligation 2). A
// run with no command runner was never given the capability, so the two
// command-derived observations are not measured and name the capability they
// needed. A run whose command seam denies the execution is not measured either,
// with a *different* reason code, so "the capability was never there" and "the
// capability refused" stay distinguishable. Either way the probe is never a pass:
// it passes only when all three observations were measured, which the aggregate
// order makes structural rather than a matter of care.
//
// The closed reason-code set holds no code for "installed but not running". The
// service-state negative therefore reuses `sshd_absent` — the vocabulary's own
// "this node is not serving sshd" code — while the observation's label and detail
// say which half was missing. That is a deliberate reuse of a closed set rather
// than an invented code (design §3.5: adding a code is a contract change), and it
// is recorded in this slice's apply evidence with the gap named: a dedicated code
// is the honest home for a stopped service.
//
// The comparison is deliberately minimal. sshd's configuration grammar is larger
// than one file, so the probe compares the directive names and values the written
// file sets — first occurrence wins, names case-insensitive — against the values
// `sshd -T` reports for the same names. It does not follow `Include`, does not
// model quoting and does not apply compiled-in defaults: a directive the written
// file does not set cannot disagree with the configuration in force, and PRD §13's
// case is exactly a written directive whose effective value differs or is absent.

const (
	// localSSHDBinaryPath is the documented location of the sshd binary. One path
	// keeps "where did you look" answerable from the detail, and it is where every
	// supported platform installs it.
	localSSHDBinaryPath = "/usr/sbin/sshd"
	// localSSHDConfigPath is the written configuration this probe reads.
	localSSHDConfigPath = "/etc/ssh/sshd_config"
	// localSSHDCommand and localSSHDConfigFlag are the effective-configuration
	// question: `sshd -T` prints the configuration in force and runs as the
	// invoking user. A configuration it cannot print is not a divergence, which is
	// why the invocation has its own outcome below.
	localSSHDCommand    = "sshd"
	localSSHDConfigFlag = "-T"
	// localSSHDServiceCommand and localSSHDServiceQuery are the service-state
	// question. `is-active` reports a unit's state and changes nothing.
	localSSHDServiceCommand = "systemctl"
	localSSHDServiceQuery   = "is-active"
	// The two unit names a distribution may ship, queried together: Debian and
	// Ubuntu name the unit `ssh.service` and the RHEL family names it
	// `sshd.service`. One query with both names asks the service manager the actual
	// question — "is either serving?" — instead of guessing a distribution.
	localSSHDServiceUnitSSHD = "sshd.service"
	localSSHDServiceUnitSSH  = "ssh.service"
	// The three observation labels. They are the separately reportable answers
	// R-HR-18 names, and they are stable because the reasoning layer and the human
	// projection both quote them.
	labelSSHDBinary  = "binary present"
	labelSSHDService = "service state"
	labelSSHDConfig  = "effective config"
)

// sshdServiceUnits is the unit list the service query asks about, in the order the
// command carries it. It is read-only: nothing here is ever written to.
var sshdServiceUnits = []string{localSSHDServiceUnitSSHD, localSSHDServiceUnitSSH}

// runClockNow reads the run's clock, or the zero time when the run injected no
// clock seam. Both local probes take their timestamps this way, so only an
// injected clock can move an elapsed value (design §3.3), and a probe without a
// clock leaves the value to the runner rather than reaching for the wall clock.
func runClockNow(seams Seams) time.Time {
	if seams.Clock == nil {
		return time.Time{}
	}
	return seams.Clock.Now()
}

// localSSHD is the `local.sshd` probe. It carries the run's seams and nothing
// else, and it reads only two of them: the filesystem and the command runner. A
// probe that reached the filesystem or executed a binary directly would measure
// the machine the test runs on instead of the machine the case describes, and
// would put an exec outside the run's capability set (R-HR-02).
type localSSHD struct {
	seams Seams
}

// newLocalSSHD builds the probe from the run's injected seams. The run's declared
// target input is ignored, for the same reason `local.env` ignores it: this probe
// measures this machine and declares no remote target.
func newLocalSSHD(seams Seams, _ TargetInput) Probe { return &localSSHD{seams: seams} }

// Name is the probe's stable identifier, declared once in registry.go.
func (p *localSSHD) Name() string { return probeNameLocalSSHD }

// Kind is the question family the probe belongs to.
func (p *localSSHD) Kind() ProbeKind { return ProbeLocal }

// Run performs the three measurements in declaration order and reports them as
// three observations plus the reduction of all three.
//
// The result's target is the probe's local subject, the documented binary path: a
// probe that measures this machine dials nothing, and each observation carries the
// path or unit it actually read, so the caller can tell which local fact the
// result is about.
//
// The context travels into the two command invocations, so a run that is cancelled
// or that exhausts its budget stops waiting on a command rather than leaving one
// behind. The commands themselves are bounded by the runner's per-probe bound; a
// probe-level budget would add a second, invisible deadline to the same question.
func (p *localSSHD) Run(ctx context.Context) Result {
	started := p.now()
	observations := []Observation{
		p.observeBinary(),
		p.observeService(ctx),
		p.observeEffectiveConfig(ctx),
	}
	verdict, reason := Aggregate(observations)
	details := make([]string, 0, len(observations))
	for _, observation := range observations {
		details = append(details, observation.Detail)
	}
	return Result{
		Probe:        p.Name(),
		Kind:         p.Kind(),
		Target:       localSSHDBinaryPath,
		Verdict:      verdict,
		Reason:       reason,
		Detail:       strings.Join(details, "\n"),
		Elapsed:      p.now().Sub(started),
		Observations: observations,
	}
}

// now reads the run's clock, or the zero time when none was injected, exactly as
// `local.env` does.
func (p *localSSHD) now() time.Time { return runClockNow(p.seams) }

// observeBinary reports whether an sshd binary is installed at the documented
// path.
//
// Absence is a measurement, not a skip: the filesystem answered, and the answer is
// that nothing is there. It is a definite negative for a node that is supposed to
// serve SSH, and the detail states that installing sshd is work owned by a later
// slice rather than by this run, so the report is honest about the boundary
// instead of offering a change this slice will not make.
//
// A check that fails for any other reason is not absence: nothing answered about
// the path itself, so the observation is unresolved and carries the failure
// verbatim. Reporting it as absent would be a fabricated measurement.
func (p *localSSHD) observeBinary() Observation {
	if p.seams.FS == nil {
		return Observe(labelSSHDBinary, localSSHDBinaryPath, PurposeSSHDConfiguration, RawObservation{
			Kind:    ObsCapabilityExcluded,
			Wording: localSSHDBinaryPath + ": no filesystem seam is injected for this run, so the sshd binary cannot be checked",
		})
	}
	if _, err := p.seams.FS.Stat(localSSHDBinaryPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Observe(labelSSHDBinary, localSSHDBinaryPath, PurposeSSHDConfiguration, RawObservation{
				Kind: ObsSSHDBinaryAbsent,
				Wording: fmt.Sprintf("the sshd binary is not present at %s: no sshd is installed here, and installing it is not part of this run but work owned by a later slice; nothing was changed",
					localSSHDBinaryPath),
			})
		}
		return Observe(labelSSHDBinary, localSSHDBinaryPath, PurposeSSHDConfiguration, RawObservation{
			Kind:    ObsInternalFailure,
			Wording: fmt.Sprintf("%s: %v (the path could not be checked, so its absence is not claimed)", localSSHDBinaryPath, err),
		})
	}
	return Observe(labelSSHDBinary, localSSHDBinaryPath, PurposeSSHDConfiguration, RawObservation{
		Kind:    ObsSSHDBinaryPresent,
		Wording: fmt.Sprintf("the sshd binary is present at %s", localSSHDBinaryPath),
	})
}

// observeService asks the service manager whether an sshd unit is active.
//
// The answer decides the observation, not the exit status: `systemctl is-active`
// exits 3 when no queried unit is active, so a stopped service arrives as output
// beside an error, and the verbatim answer is what a reader compares against their
// own `systemctl` run. An answer naming an active unit is a measured pass; an
// answer naming none is a measured negative about this machine's sshd.
func (p *localSSHD) observeService(ctx context.Context) Observation {
	target := strings.Join(sshdServiceUnits, ",")
	args := append([]string{localSSHDServiceQuery}, sshdServiceUnits...)
	stdout, denial := p.invoke(ctx, localSSHDServiceCommand, args...)
	if denial.Kind != "" {
		return Observe(labelSSHDService, target, PurposeSSHDConfiguration, denial)
	}
	answer := strings.TrimSpace(string(stdout))
	if sshdServiceActive(string(stdout)) {
		return Observe(labelSSHDService, target, PurposeSSHDConfiguration, RawObservation{
			Kind: ObsSSHDServiceRunning,
			Wording: fmt.Sprintf("the sshd service is running: %s reported %q for %s",
				commandLine(localSSHDServiceCommand, args...), answer, target),
		})
	}
	return Observe(labelSSHDService, target, PurposeSSHDConfiguration, RawObservation{
		Kind: ObsSSHDServiceNotRunning,
		Wording: fmt.Sprintf("the sshd service is not running: %s answered %q for %s, so no sshd unit is active on this machine; this run reports the state it read and changes nothing",
			commandLine(localSSHDServiceCommand, args...), answer, target),
	})
}

// sshdServiceActive reports whether the service manager named an active unit.
//
// `systemctl is-active` prints one state per queried unit, so the rule is "any
// line is active" rather than "the whole answer is active": with both unit names
// queried, a machine that ships either one is a machine serving sshd. The
// comparison is case-insensitive because the state the service manager prints is
// its own vocabulary, not this package's.
func sshdServiceActive(answer string) bool {
	for _, line := range strings.Split(answer, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "active") {
			return true
		}
	}
	return false
}

// observeEffectiveConfig compares the written configuration with the
// configuration in force.
//
// Three boundaries are deliberate. First, the written file is read before the
// command runs: if it cannot be read, nothing was compared and the observation
// says so instead of running a command whose answer would be unused. Second, a
// missing written file is not a divergence — sshd then runs on its compiled-in
// defaults — so the effective configuration is reported with the absence stated,
// and no divergence is claimed. Third, a check that fails for a reason other than
// absence is unresolved: nothing was compared, and the failure is carried
// verbatim.
//
// The two configurations appear verbatim in the detail of both the divergent and
// the agreeing case, so a reader can check the comparison rather than trust it
// (R-HR-07).
func (p *localSSHD) observeEffectiveConfig(ctx context.Context) Observation {
	command := commandLine(localSSHDCommand, localSSHDConfigFlag)
	if p.seams.FS == nil {
		return Observe(labelSSHDConfig, localSSHDConfigPath, PurposeSSHDConfiguration, RawObservation{
			Kind:    ObsCapabilityExcluded,
			Wording: localSSHDConfigPath + ": no filesystem seam is injected for this run, so the written configuration cannot be read",
		})
	}
	writtenBytes, err := p.seams.FS.ReadFile(localSSHDConfigPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Observe(labelSSHDConfig, localSSHDConfigPath, PurposeSSHDConfiguration, RawObservation{
			Kind:    ObsInternalFailure,
			Wording: fmt.Sprintf("%s: %v (the written configuration could not be read, so it was not compared with the configuration in force)", localSSHDConfigPath, err),
		})
	}
	writtenPresent := err == nil

	stdout, denial := p.invoke(ctx, localSSHDCommand, localSSHDConfigFlag)
	if denial.Kind != "" {
		return Observe(labelSSHDConfig, localSSHDConfigPath, PurposeSSHDConfiguration, denial)
	}
	effectiveText := strings.TrimSpace(string(stdout))

	if !writtenPresent {
		return Observe(labelSSHDConfig, localSSHDConfigPath, PurposeSSHDConfiguration, RawObservation{
			Kind: ObsSSHDConfigMatches,
			Wording: fmt.Sprintf("the written configuration file %s is not present, so no written directive disagrees with the configuration in force; the effective configuration reported by %s is: %s",
				localSSHDConfigPath, command, effectiveText),
		})
	}

	written := parseSSHDConfig(string(writtenBytes))
	effective := parseSSHDConfig(effectiveText)
	if diverged := sshdDivergences(written, effective); len(diverged) > 0 {
		return Observe(labelSSHDConfig, localSSHDConfigPath, PurposeSSHDConfiguration, RawObservation{
			Kind: ObsSSHDConfigDivergent,
			Wording: fmt.Sprintf("the written configuration at %s is not the configuration in force: %s disagrees with it on %d of the %d directive(s) the written file sets (%s); written configuration of %s: %s; effective configuration reported by %s: %s",
				localSSHDConfigPath, command, len(diverged), len(written.names), strings.Join(diverged, ", "),
				localSSHDConfigPath, string(writtenBytes), command, effectiveText),
		})
	}
	return Observe(labelSSHDConfig, localSSHDConfigPath, PurposeSSHDConfiguration, RawObservation{
		Kind: ObsSSHDConfigMatches,
		Wording: fmt.Sprintf("%s agrees with the configuration in force reported by %s for every one of the %d directive(s) the written file sets; written configuration of %s: %s; effective configuration reported by %s: %s",
			localSSHDConfigPath, command, len(written.names), localSSHDConfigPath, string(writtenBytes), command, effectiveText),
	})
}

// invoke runs one command through the run's command seam and reports either its
// answer or the fact that explains why no answer exists.
//
// Four outcomes are deliberately distinguished, because they are four different
// facts (design §5.1 obligation 2):
//
//   - no command runner is injected: the capability was never given to this run,
//     so the fact is ObsCapabilityExcluded and the command is named;
//   - the seam denied the execution: the capability was given and refused, so the
//     fact is ObsCommandDenied — a different fact with a different code;
//   - the command answered, including with a non-zero exit status such as
//     `systemctl is-active` returning 3 for a stopped unit: the answer is returned
//     and the caller judges it;
//   - the command produced nothing classifiable (an error without an answer): the
//     fact is ObsInternalFailure with the verbatim error.
//
// The captured streams are folded into a denial's wording so a reader sees what
// the command said instead of only that it failed, and the exit status is never
// used to choose a reason code: R-HR-07 puts every code in the classification
// table, and the verbatim text is what a reader quotes.
func (p *localSSHD) invoke(ctx context.Context, name string, args ...string) ([]byte, RawObservation) {
	command := commandLine(name, args...)
	if p.seams.CommandRunner == nil {
		return nil, p.seams.CommandFact(command, nil)
	}
	stdout, stderr, err := p.seams.CommandRunner.Run(ctx, name, args...)
	if err == nil {
		return stdout, RawObservation{}
	}
	if errors.Is(err, ErrSeamDenied) {
		fact := p.seams.CommandFact(command, err)
		fact.Wording = withCapturedOutput(fact.Wording, stdout, stderr)
		return nil, fact
	}
	if len(stdout) > 0 {
		// The command ran and answered with a non-zero status. Treating every
		// non-nil error as a denial would report a stopped service as an excluded
		// capability, which is a different fact about a different capability.
		return stdout, RawObservation{}
	}
	return nil, RawObservation{
		Kind:    ObsInternalFailure,
		Wording: withCapturedOutput(fmt.Sprintf("%s: %v (the command produced no answer to classify)", command, err), stdout, stderr),
	}
}

// commandLine renders an invocation the way the detail text and the tests quote
// it: the command name and its arguments separated by single spaces.
func commandLine(name string, args ...string) string {
	return strings.Join(append([]string{name}, args...), " ")
}

// withCapturedOutput appends the streams a command captured to its wording. An
// empty stream adds nothing, so a denial that captured nothing reads as the
// denial alone instead of as an empty label.
func withCapturedOutput(wording string, stdout, stderr []byte) string {
	if len(stdout) > 0 {
		wording += "; stdout: " + strings.TrimSpace(string(stdout))
	}
	if len(stderr) > 0 {
		wording += "; stderr: " + strings.TrimSpace(string(stderr))
	}
	return wording
}

// sshdConfig is one sshd configuration text reduced to the directives it sets.
// names keeps the order the directives first appear in, so a comparison's detail
// text is stable rather than map-ordered.
type sshdConfig struct {
	names  []string
	values map[string]string
}

// parseSSHDConfig reduces a configuration text to the directive values it sets.
//
// The grammar modelled here is the part this probe needs and no more: one
// directive per line, the name first and the value after it, a line whose first
// non-space character is `#` being a comment, blank lines ignored, directive names
// case-insensitive, whitespace inside a value collapsed, and the first occurrence
// of a name winning (sshd keeps the first value it reads). It does not follow
// `Include`, does not model quoting or escapes, and does not apply compiled-in
// defaults: a configuration larger than this question would need more, and stating
// the limit is better than inventing semantics silently.
func parseSSHDConfig(text string) sshdConfig {
	config := sshdConfig{values: map[string]string{}}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.ToLower(fields[0])
		value := strings.Join(fields[1:], " ")
		if _, seen := config.values[name]; seen {
			continue
		}
		config.values[name] = value
		config.names = append(config.names, name)
	}
	return config
}

// sshdDivergences returns the directives the written file sets whose value the
// effective configuration does not agree with, in the order the written file sets
// them. A written directive the effective configuration does not mention at all is
// a divergence too: the configuration in force did not come from this file, which
// is exactly PRD §13's "the written config is not the effective config".
func sshdDivergences(written, effective sshdConfig) []string {
	var diverged []string
	for _, name := range written.names {
		effectiveValue, reported := effective.values[name]
		if !reported || effectiveValue != written.values[name] {
			diverged = append(diverged, name)
		}
	}
	return diverged
}
