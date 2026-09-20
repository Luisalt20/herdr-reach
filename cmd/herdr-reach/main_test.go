package main

// This file drives the entrypoint in process, at the boundary the process would
// exit from: run(args, stdio, seams) is the shape main wraps in os.Exit, so the
// stream split and the exit-code plumbing are reachable without spawning the
// binary.
//
// Every case passes scripted seams: the deny-all default of design §6.2 with
// exactly the one capability a case needs overridden. ProductionSeams is never
// called here — a test that built it would measure the machine the suite happens
// to run on, and the static guard in internal/probe asserts that only main.go
// calls it.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/doctor"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/version"
)

// --- scripted machines -------------------------------------------------------

// linuxPlatform scripts a classified Linux machine: the deny-all default cannot
// classify a platform, and an unclassifiable one makes the run incomplete for a
// reason this file did not script. With the platform answered, a run over the
// deny-all seams is complete — every refused capability reads as an attempt that
// was not made — so an exit-code difference in a case is that case's own.
type linuxPlatform struct{}

// GOOS reports the scripted operating system.
func (linuxPlatform) GOOS() string { return "linux" }

// Arch reports the scripted architecture.
func (linuxPlatform) Arch() string { return "amd64" }

// WSL2 reports no WSL2 signal: a machine a case did not describe must not claim
// one.
func (linuxPlatform) WSL2() bool { return false }

// Systemd reports no service-manager signal, for the same reason.
func (linuxPlatform) Systemd() bool { return false }

// unresolvedVerifier answers every handshake with a plain error: an attempt that
// produced no answer, which is unresolved and makes the run incomplete. It is
// deliberately not ErrSeamDenied (an attempt that was not made) and not
// ErrTLSVerification (a measured rejection), because those are different facts.
type unresolvedVerifier struct{}

// Verify reports that the handshake produced no answer.
func (unresolvedVerifier) Verify(context.Context, string, *tls.Config) (probe.TLSVerification, error) {
	return probe.TLSVerification{}, errors.New("the handshake produced no answer")
}

// scriptedSeams is the deny-all default with a classified Linux platform.
func scriptedSeams() probe.Seams {
	seams := probe.DenyAllSeams()
	seams.Platform = linuxPlatform{}
	return seams
}

// failingWriter refuses every write and counts the attempts, so a case can
// assert the document was attempted exactly once and reported as unwritten
// rather than silently dropped.
type failingWriter struct{ writes int }

// Write refuses the bytes and reports the failure.
func (w *failingWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errors.New("the stream refused the write")
}

// runDocument is the subset of the schema_version "1" document this file reads:
// the completeness the exit code must agree with. The payload's full key set is
// pinned by the report suite.
type runDocument struct {
	Run struct {
		Completeness string `json:"completeness"`
	} `json:"run"`
}

// decodeDocument parses stdout as exactly one machine-readable document. The
// second Decode must find EOF: a projection that wrote the document twice, or
// mixed a sentence into it, is not the one-document contract R-HR-07 states, and
// this helper fails instead of accepting the first value it can read.
func decodeDocument(t *testing.T, stdout []byte) runDocument {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	var document runDocument
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("stdout did not parse as one machine-readable document: %v\nstdout: %q", err, stdout)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("stdout carried more than one JSON value: %v\nstdout: %q", err, stdout)
	}
	return document
}

// --- the stream split --------------------------------------------------------

// TestStreamSplitJSONKeepsTheDocumentAloneOnStdout is R-HR-07's first scenario at
// the entrypoint: with --json the output writer carries exactly one parseable
// document and no human text, while the human report lands on the error writer.
func TestStreamSplitJSONKeepsTheDocumentAloneOnStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := run([]string{"doctor", "--json", "--hub", "203.0.113.10"}, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, scriptedSeams())

	if got != doctor.ExitOK {
		t.Fatalf("run returned exit %d, want %d for a complete run", got, doctor.ExitOK)
	}
	document := decodeDocument(t, stdout.Bytes())
	if document.Run.Completeness != "complete" {
		t.Errorf("run.completeness is %q, want %q", document.Run.Completeness, "complete")
	}
	if (got == doctor.ExitOK) != (document.Run.Completeness == "complete") {
		t.Errorf("exit code %d and run.completeness %q disagree", got, document.Run.Completeness)
	}
	if strings.Contains(stdout.String(), "PROBES") {
		t.Errorf("the human projection leaked into the output writer: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "PROBES") {
		t.Errorf("the human projection did not reach the error writer: %q", stderr.String())
	}
}

// TestStreamSplitWithoutJSONLeavesStdoutEmpty is the other half: the human
// projection is the run's only output, and it is written to the error writer.
func TestStreamSplitWithoutJSONLeavesStdoutEmpty(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := run([]string{"doctor", "--hub", "203.0.113.10"}, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, scriptedSeams())

	if got != doctor.ExitOK {
		t.Fatalf("run returned exit %d, want %d for a complete run", got, doctor.ExitOK)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout carries %q; without --json the output writer stays empty", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Errorf("the human report did not reach the error writer")
	}
}

// --- the two flag-only modes -------------------------------------------------

// TestEntrypointVersionWritesToStdoutAndExitsZero pins the version stream: the
// version is the answer a caller asked for, so it goes to the output writer and
// the run exits 0.
func TestEntrypointVersionWritesToStdoutAndExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := run([]string{"--version"}, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, scriptedSeams())

	if got != doctor.ExitOK {
		t.Fatalf("run returned exit %d, want %d for a version request", got, doctor.ExitOK)
	}
	if got, want := stdout.String(), version.String()+"\n"; got != want {
		t.Errorf("stdout carries %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr carries %q; a version request produces no human run text", stderr.String())
	}
}

// TestEntrypointHelpWritesUsageToStderrAndExitsZero pins the help stream: a help
// request is not a usage error, so usage goes to the error writer and the run
// exits 0.
func TestEntrypointHelpWritesUsageToStderrAndExitsZero(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(args, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, scriptedSeams())

			if got != doctor.ExitOK {
				t.Fatalf("run returned exit %d, want %d for a help request", got, doctor.ExitOK)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout carries %q; usage belongs on the error writer", stdout.String())
			}
			if !strings.Contains(stderr.String(), "usage:") || !strings.Contains(stderr.String(), "--hub") {
				t.Errorf("stderr does not carry the usage text: %q", stderr.String())
			}
		})
	}
}

// --- the exit-code plumbing --------------------------------------------------

// TestEntrypointUsageErrorExitsTwoWithEmptyStdout is the refused-command-line
// path: the typed usage error and the surface's usage text reach the error
// writer, the output writer stays empty, and the run exits 2 without starting a
// measurement.
func TestEntrypointUsageErrorExitsTwoWithEmptyStdout(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{name: "no command word", args: nil},
		{name: "an unknown flag", args: []string{"--bogus"}},
		{name: "an empty hub host in run mode", args: []string{"doctor", "--hub", ":22"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := run(tc.args, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, scriptedSeams())

			if got != doctor.ExitUsage {
				t.Fatalf("run returned exit %d, want %d", got, doctor.ExitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("the refusal wrote %q to the output writer", stdout.String())
			}
			if !strings.Contains(stderr.String(), "usage:") {
				t.Errorf("the refusal did not carry the usage text on the error writer: %q", stderr.String())
			}
		})
	}
}

// TestEntrypointFailedDocumentWriteExitsTwo is the internal-error path: the
// document write fails, the run reports exit 2 rather than presenting the
// diagnosis as completed, and the writer saw exactly one attempted document.
func TestEntrypointFailedDocumentWriteExitsTwo(t *testing.T) {
	stdout := &failingWriter{}
	var stderr bytes.Buffer
	got := run([]string{"doctor", "--json", "--hub", "203.0.113.10"}, doctor.Stdio{Stdout: stdout, Stderr: &stderr}, scriptedSeams())

	if got != doctor.ExitUsage {
		t.Fatalf("run returned exit %d, want %d for a failed document write", got, doctor.ExitUsage)
	}
	if stdout.writes != 1 {
		t.Errorf("the output writer saw %d writes, want exactly one attempted document", stdout.writes)
	}
}

// TestRunUnresolvedMeasurementExitsOne drives an attempted measurement that
// produced no answer through the entrypoint: the run is incomplete, exits 1, and
// the document it produced states the same completeness the exit code does.
func TestRunUnresolvedMeasurementExitsOne(t *testing.T) {
	seams := scriptedSeams()
	seams.TLSVerifier = unresolvedVerifier{}

	var stdout, stderr bytes.Buffer
	got := run([]string{"doctor", "--json", "--hub", "203.0.113.10"}, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, seams)

	if got != doctor.ExitIncomplete {
		t.Fatalf("run returned exit %d, want %d for a run with an unresolved measurement", got, doctor.ExitIncomplete)
	}
	document := decodeDocument(t, stdout.Bytes())
	if document.Run.Completeness != "incomplete" {
		t.Errorf("run.completeness is %q, want %q", document.Run.Completeness, "incomplete")
	}
	if (got == doctor.ExitOK) != (document.Run.Completeness == "complete") {
		t.Errorf("exit code %d and run.completeness %q disagree", got, document.Run.Completeness)
	}
}

// --- the writes-nothing proof (design §6.3, R-HR-02, R-HR-24, R-HR-27) -------
//
// The proof runs the real entrypoint in process with HOME and the working
// directory pointed at two fresh temp trees, digests both trees recursively
// before and after, and asserts the run left them byte-identical, invoked no
// command and dialed exactly the declared set. Its coverage boundary is stated
// rather than over-credited (RG-14): a temp-HOME digest cannot prove the absence
// of writes outside HOME/CWD. R1a holds that by construction — no writer code
// path and no write-capable dependency is imported — and the residual gap stays
// in the risk register rather than being closed by this test.
//
// The recording seam set below mirrors the production command shape
// (internal/probe/real.go leaves the production CommandRunner nil), so the
// writes-nothing runs have no execution path at all; TestNoExecEntrypointHasNoExecutionPath
// proves the counter is live by wiring it in a control.

// recordedTriple is one outbound attempt in the (host, port, protocol) form the
// declared target set uses. Each recording seam records the triple the protocol
// it owns measures, so the comparison with probe.EffectiveTargets compares like
// values rather than a seam's private idea of what it dialed.
type recordedTriple struct {
	Host     string
	Port     int
	Protocol probe.Protocol
}

// recordingSeams is the recording seam set the writes-nothing proof runs over:
// every TCP dial, packet socket and TLS handshake is recorded as the triple its
// protocol measures and then denied with probe.ErrSeamDenied. An attempt is
// therefore visible in the record, nothing leaves the process, and an attempt
// against a target the declaration does not carry appears as an undeclared
// triple.
//
// The resolver is the one seam that answers instead of denying, and the reason is
// structural rather than permissive: the name-based probes resolve before they
// dial, so a denying resolver would stop them before the dial seam that measures
// their protocol was reached and the declared set could never be compared. The
// answer is a scripted documentation address and never leaves the process.
type recordingSeams struct {
	mu       sync.Mutex
	triples  map[recordedTriple]bool
	commands *countingCommandRunner
	seams    probe.Seams
}

// newRecordingSeams builds a fresh recording set with the counting command
// runner wired.
func newRecordingSeams() *recordingSeams {
	recording := &recordingSeams{
		triples:  make(map[recordedTriple]bool),
		commands: &countingCommandRunner{},
	}
	seams := probe.DenyAllSeams()
	seams.Platform = linuxPlatform{}
	seams.Resolver = answeringResolver{}
	seams.Dialer = recordingDialer{recording: recording}
	seams.PacketDialer = recordingPacketDialer{recording: recording}
	seams.TLSVerifier = recordingTLSVerifier{recording: recording}
	seams.CommandRunner = recording.commands
	recording.seams = seams
	return recording
}

// withoutCommandRunner returns the recording set in the production command
// shape: internal/probe/real.go leaves the production CommandRunner nil, so a
// run in that shape has no execution path at all.
func (r *recordingSeams) withoutCommandRunner() probe.Seams {
	seams := r.seams
	seams.CommandRunner = nil
	return seams
}

// record adds one outbound attempt to the set.
func (r *recordingSeams) record(triple recordedTriple) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.triples[triple] = true
}

// recorded returns a copy of the recorded triple set.
func (r *recordingSeams) recorded() map[recordedTriple]bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	recorded := make(map[recordedTriple]bool, len(r.triples))
	for triple := range r.triples {
		recorded[triple] = true
	}
	return recorded
}

// recordingDialer records every TCP dial attempt and denies it.
type recordingDialer struct{ recording *recordingSeams }

// DialContext records the declared address and returns the denial.
func (d recordingDialer) DialContext(_ context.Context, _, addr string) (net.Conn, error) {
	d.recording.record(tripleFromAddress(addr, probe.ProtocolTCP))
	return nil, probe.ErrSeamDenied
}

// recordingPacketDialer records every packet socket attempt and denies it.
type recordingPacketDialer struct{ recording *recordingSeams }

// DialPacket records the declared address and returns the denial.
func (d recordingPacketDialer) DialPacket(_ context.Context, _, addr string) (probe.PacketConn, error) {
	d.recording.record(tripleFromAddress(addr, probe.ProtocolUDP))
	return nil, probe.ErrSeamDenied
}

// recordingTLSVerifier records every handshake attempt and denies it.
type recordingTLSVerifier struct{ recording *recordingSeams }

// Verify records the declared address and returns the denial.
func (v recordingTLSVerifier) Verify(_ context.Context, target string, _ *tls.Config) (probe.TLSVerification, error) {
	v.recording.record(tripleFromAddress(target, probe.ProtocolTLS))
	return probe.TLSVerification{}, probe.ErrSeamDenied
}

// tripleFromAddress parses the "host:port" address a seam was handed. An address
// that cannot be parsed is recorded with port -1, which no declared target can
// carry, so a malformed attempt fails the set comparison instead of being
// silently dropped from it.
func tripleFromAddress(address string, protocol probe.Protocol) recordedTriple {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return recordedTriple{Host: address, Port: -1, Protocol: protocol}
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return recordedTriple{Host: host, Port: -1, Protocol: protocol}
	}
	return recordedTriple{Host: host, Port: port, Protocol: protocol}
}

// answeringResolver answers every lookup with one scripted documentation
// address. It is the seam that lets a name-based probe reach the dial seam that
// measures its protocol; the answer never leaves the process.
type answeringResolver struct{}

// LookupHost returns the scripted address.
func (answeringResolver) LookupHost(context.Context, string) ([]string, error) {
	return []string{"192.0.2.1"}, nil
}

// countingCommandRunner records every invocation and denies it, so a case can
// assert both that the run attempted an execution and that the attempt never
// became one.
type countingCommandRunner struct {
	mu    sync.Mutex
	calls []commandInvocation
}

// commandInvocation is one recorded invocation: the command line the probe asked
// for and the runner's answer.
type commandInvocation struct {
	Command string
	Err     error
}

// Run records the invocation and returns the denial.
func (r *countingCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
	command := name
	if len(args) > 0 {
		command += " " + strings.Join(args, " ")
	}
	r.mu.Lock()
	r.calls = append(r.calls, commandInvocation{Command: command, Err: probe.ErrSeamDenied})
	r.mu.Unlock()
	return nil, nil, probe.ErrSeamDenied
}

// invocations returns a copy of the recorded invocations.
func (r *countingCommandRunner) invocations() []commandInvocation {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]commandInvocation(nil), r.calls...)
}

// declaredTripleSet converts the effective declared set into the recorded form,
// so set equality is a comparison of two values of the same type.
func declaredTripleSet(effective []probe.EffectiveTarget) map[recordedTriple]bool {
	declared := make(map[recordedTriple]bool, len(effective))
	for _, target := range effective {
		declared[recordedTriple{Host: target.Host, Port: target.Port, Protocol: target.Protocol}] = true
	}
	return declared
}

// tripleSetDifference reports the two directions of the set comparison: attempts
// that are not declared, and declared targets that were never attempted. An
// empty result is set equality.
func tripleSetDifference(got, want map[recordedTriple]bool) []string {
	var difference []string
	for triple := range got {
		if !want[triple] {
			difference = append(difference, fmt.Sprintf("undeclared attempt recorded: %s:%d %s", triple.Host, triple.Port, triple.Protocol))
		}
	}
	for triple := range want {
		if !got[triple] {
			difference = append(difference, fmt.Sprintf("declared target never attempted: %s:%d %s", triple.Host, triple.Port, triple.Protocol))
		}
	}
	slices.Sort(difference)
	return difference
}

// digestTree returns a recursive digest of root: every entry's relative path and
// mode, and the content of every regular file. A created, removed, renamed or
// chmodded entry changes the digest, and so does any content change, which is
// what makes "the trees are byte-identical" checkable from outside the run.
func digestTree(t *testing.T, root string) string {
	t.Helper()
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(hash, "%s\x00%s\x00", relative, info.Mode())
		if entry.Type().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			hash.Write(data)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("digesting %s: %v", root, err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// TestNoWriteEntrypointLeavesTheTreesUntouched is design §6.3's writes-nothing
// proof at the real entrypoint: with HOME and the working directory pointed at
// two fresh temp trees, a completed run leaves both trees byte-identical, dials
// exactly the declared set — with a hub address and a --target override in the
// second case, so the declared set is not trivially the constant one — and has
// no command runner to invoke.
func TestNoWriteEntrypointLeavesTheTreesUntouched(t *testing.T) {
	home := t.TempDir()
	workdir := t.TempDir()
	t.Setenv("HOME", home)

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("reading the working directory: %v", err)
	}
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("moving the working directory into the temp tree: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restoring the working directory: %v", err)
		}
	})

	runs := []struct {
		name string
		args []string
	}{
		{name: "the default declared set", args: []string{"doctor", "--json"}},
		{name: "a hub address and a target override", args: []string{"doctor", "--json", "--hub", "203.0.113.10:2222", "--target", "egress.ssh.known=192.0.2.20:2200"}},
	}

	for _, tc := range runs {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := doctor.ParseFlags(tc.args)
			if err != nil {
				t.Fatalf("ParseFlags(%q) returned an error: %v", tc.args, err)
			}
			recording := newRecordingSeams()
			beforeHome, beforeWorkdir := digestTree(t, home), digestTree(t, workdir)

			var stdout, stderr bytes.Buffer
			got := run(tc.args, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, recording.withoutCommandRunner())

			// A refused run would leave the trees untouched for the wrong reason,
			// so the completed run is part of the proof rather than a nicety.
			if got != doctor.ExitOK {
				t.Fatalf("run returned exit %d, want %d for a completed run", got, doctor.ExitOK)
			}
			if stdout.Len() == 0 || stderr.Len() == 0 {
				t.Fatalf("the run produced no projections, so the digest assertions would be vacuous")
			}

			afterHome, afterWorkdir := digestTree(t, home), digestTree(t, workdir)
			if beforeHome != afterHome {
				t.Errorf("the run changed the temp home tree (before %s, after %s)", beforeHome, afterHome)
			}
			if beforeWorkdir != afterWorkdir {
				t.Errorf("the run changed the temp working tree (before %s, after %s)", beforeWorkdir, afterWorkdir)
			}

			// The production command shape wires no runner at all, so no call is
			// possible; TestNoExecEntrypointHasNoExecutionPath proves the counter
			// is live by wiring it in a control.
			if invocations := recording.commands.invocations(); len(invocations) != 0 {
				t.Errorf("the run invoked the command runner %d times (%v); the production shape wires no runner", len(invocations), invocations)
			}

			want, err := probe.EffectiveTargets(probe.TargetInput{Hub: opts.Hub, Overrides: opts.Targets})
			if err != nil {
				t.Fatalf("EffectiveTargets(%+v) returned an error: %v", opts, err)
			}
			recorded := recording.recorded()
			if len(recorded) == 0 {
				t.Fatalf("the recording set recorded no attempt at all, so the set comparison would pass vacuously")
			}
			if difference := tripleSetDifference(recorded, declaredTripleSet(want)); len(difference) > 0 {
				t.Errorf("the dialed set is not the declared set:\n%s", strings.Join(difference, "\n"))
			}
		})
	}

	// The counter-case that makes the digest assertion load-bearing: the same
	// comparison must fail when something was written into one of the trees. The
	// failure is asserted, never swallowed, so a digest that ignored content or
	// mode would fail this case instead of passing the proof silently.
	t.Run("control: a write into the tree fails the digest comparison", func(t *testing.T) {
		before := digestTree(t, home)
		if err := os.WriteFile(filepath.Join(home, "control-write"), []byte("this write must be seen"), 0o600); err != nil {
			t.Fatalf("writing the control file: %v", err)
		}
		after := digestTree(t, home)
		if before == after {
			t.Fatalf("the digest comparison did not notice a file written into the temp home tree, so the writes-nothing assertion cannot fail")
		}
	})
}

// TestNoExecEntrypointHasNoExecutionPath pins the two shapes of the execution
// question. The production shape wires no command runner at all — the seam that
// is the only path to a third-party binary — so a run under it has no execution
// path and invokes nothing. The control wires the counting runner and shows the
// counter is live: the run attempts exactly the two command-derived observations
// of local.sshd and every attempt is denied, so the seam is the only thing
// between the run and an execution and nothing executed.
//
// The design's §6.3 sentence "the command runner was called zero times" holds
// for the production shape and only for it: local.sshd queries the service
// manager and the effective configuration through any non-nil runner
// (internal/probe/local.go), so a run given a runner calls it twice. This case
// records that fact instead of hiding it.
func TestNoExecEntrypointHasNoExecutionPath(t *testing.T) {
	production := newRecordingSeams().withoutCommandRunner()
	if production.CommandRunner != nil {
		t.Fatalf("the production-shaped recording set wires a command runner")
	}

	recording := newRecordingSeams()
	var stdout, stderr bytes.Buffer
	got := run([]string{"doctor", "--json", "--hub", "203.0.113.10"}, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, recording.seams)
	if got != doctor.ExitOK {
		t.Fatalf("run returned exit %d, want %d for a completed run", got, doctor.ExitOK)
	}

	calls := recording.commands.invocations()
	want := []string{"systemctl is-active sshd.service ssh.service", "sshd -T"}
	if len(calls) != len(want) {
		t.Fatalf("the run invoked the command runner %d times (%v), want the %d command-derived observations of local.sshd", len(calls), calls, len(want))
	}
	for index, call := range calls {
		if call.Command != want[index] {
			t.Errorf("invocation %d is %q, want %q", index, call.Command, want[index])
		}
		if !errors.Is(call.Err, probe.ErrSeamDenied) {
			t.Errorf("invocation %d returned %v, want the seam sentinel: the run must not execute", index, call.Err)
		}
	}
}
