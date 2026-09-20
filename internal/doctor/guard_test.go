package doctor_test

// This file is the doctor side of design §6.2's no-egress proof and the
// end-to-end honesty suite of PR 20: the sentinel reachability of the deny-all
// set (level 1), the closed dialed set measured by the recording seam set (level
// 2), PRD §1.1's replay through the real pipeline, and the RG-4/RG-6 absence
// assertions over one complete run's two projections.
//
// The static half of the proof (level 3) lives in internal/probe/guard_test.go,
// where it parses the module's sources; the writes-nothing half lives in
// cmd/herdr-reach/main_test.go, where the real entrypoint is reachable. This file
// carries the levels that need the harness, and every case drives the pipeline
// through doctor.Run rather than inspecting its internals.

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/doctor"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// TestSentinelReachabilityOfTheDenyAllSeams is design §6.2 level 1: the deny-all
// seam set itself must refuse, capability by capability, so the default cannot
// silently become permissive. A permissive default would make every probe unit
// test measure the machine the suite happens to run on, and the sentinel is the
// only fact that separates "the seam denied this attempt" from "the far end
// answered".
func TestSentinelReachabilityOfTheDenyAllSeams(t *testing.T) {
	seams := probe.DenyAllSeams()
	ctx := context.Background()

	t.Run("a dial", func(t *testing.T) {
		if _, err := seams.Dialer.DialContext(ctx, "tcp", "203.0.113.10:22"); !errors.Is(err, probe.ErrSeamDenied) {
			t.Errorf("the deny-all dialer returned %v, want the seam sentinel", err)
		}
	})

	t.Run("a lookup", func(t *testing.T) {
		if _, err := seams.Resolver.LookupHost(ctx, "github.com"); !errors.Is(err, probe.ErrSeamDenied) {
			t.Errorf("the deny-all resolver returned %v, want the seam sentinel", err)
		}
	})

	t.Run("a packet exchange", func(t *testing.T) {
		// The socket is the exchange's gate: the deny-all set returns no conn,
		// so the exchange cannot start on a denied socket and no datagram can be
		// written or read through it.
		conn, err := seams.PacketDialer.DialPacket(ctx, "udp", "203.0.113.10:7844")
		if !errors.Is(err, probe.ErrSeamDenied) {
			t.Errorf("the deny-all packet dialer returned %v, want the seam sentinel", err)
		}
		if conn != nil {
			t.Errorf("the deny-all packet dialer returned a conn, so an exchange could start on a denied socket")
		}
	})

	t.Run("a TLS verification", func(t *testing.T) {
		if _, err := seams.TLSVerifier.Verify(ctx, "www.cloudflare.com:443", &tls.Config{ServerName: "www.cloudflare.com"}); !errors.Is(err, probe.ErrSeamDenied) {
			t.Errorf("the deny-all verifier returned %v, want the seam sentinel", err)
		}
	})

	t.Run("a command", func(t *testing.T) {
		if _, _, err := seams.CommandRunner.Run(ctx, "sshd", "-T"); !errors.Is(err, probe.ErrSeamDenied) {
			t.Errorf("the deny-all command runner returned %v, want the seam sentinel", err)
		}
	})
}

// TestDialedSetEqualsTheDeclaredSet is design §6.2 level 2: a whole run over the
// recording seam set dials exactly the declared set, after the hub address and a
// --target override, and nothing else.
//
// The semantics of a record: an attempt is recorded by whichever seam measures
// that protocol — the TCP dialer, the packet dialer or the TLS verifier — and is
// then denied, so nothing leaves the process. An attempt against a target the
// declaration does not carry appears as an undeclared triple, and a declared
// target no seam attempted appears as a missing one; either direction fails the
// assertion.
func TestDialedSetEqualsTheDeclaredSet(t *testing.T) {
	args := []string{"doctor", "--json", "--hub", "203.0.113.10:2222", "--target", "egress.cf.7844=192.0.2.30:7844"}
	opts, err := doctor.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags(%q) returned an error: %v", args, err)
	}

	recording := newRecordingSeams()
	var stdout, stderr bytes.Buffer
	got := doctor.Run(context.Background(), opts, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, recording.seams)

	if got != doctor.ExitOK {
		t.Fatalf("Run returned exit %d, want %d: a denied attempt is not-measured, never unresolved", got, doctor.ExitOK)
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
}

// TestMatrixReplayThroughTheRealPipeline replays PRD §1.1's matrix through the
// real pipeline and asserts the acceptance case end to end (R-HR-03, PRD §13):
// the run exits 0, no field marks it incomplete, and both projections carry the
// hub target with its port, the verbatim failing detail and the stable rule id
// that names the destination block.
func TestMatrixReplayThroughTheRealPipeline(t *testing.T) {
	const failingDetail = "dial tcp 203.0.113.10:22: connection refused"

	var stdout, stderr bytes.Buffer
	got := doctor.Run(context.Background(), doctor.Options{JSON: true, Hub: "203.0.113.10:22"},
		doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, matrixReplaySeams())

	if got != doctor.ExitOK {
		t.Fatalf("Run returned exit %d, want %d: the replay is a completed measurement, including its measured negative", got, doctor.ExitOK)
	}

	document := decodeRunDocument(t, stdout.Bytes())
	if document.Run.Completeness != "complete" {
		t.Errorf("run.completeness is %q, want %q", document.Run.Completeness, "complete")
	}
	if len(document.Run.Unresolved) != 0 {
		t.Errorf("run.unresolved is %v, want no unresolved probe", document.Run.Unresolved)
	}
	// No field may mark the run incomplete, in either projection: a reader who
	// only skims the human text must not be told the run failed to finish.
	if strings.Contains(stdout.String(), "incomplete") || strings.Contains(stderr.String(), "completeness: incomplete") {
		t.Errorf("a projection marks the completed replay as incomplete")
	}

	if document.Targets.Hub == nil || *document.Targets.Hub != "203.0.113.10:22" {
		t.Errorf("targets.hub is %v, want the declared hub address with its port", document.Targets.Hub)
	}
	hub := rowFor(t, document, "egress.hub.direct")
	if hub.Target == nil || *hub.Target != "203.0.113.10:22" {
		t.Errorf("the hub probe row target is %v, want the declared address with its port", hub.Target)
	}
	if hub.Resolution != "measured" || hub.Verdict != "fail" {
		t.Errorf("egress.hub.direct is %s/%s, want a measured failure", hub.Resolution, hub.Verdict)
	}
	if hub.Reason != string(probe.ReasonConnRefused) {
		t.Errorf("the hub row carries reason %q, want the stable probe code %q", hub.Reason, probe.ReasonConnRefused)
	}
	if hub.Detail != failingDetail {
		t.Errorf("the hub detail is %q, want the verbatim failing detail %q", hub.Detail, failingDetail)
	}

	finding := findingFor(t, document, "ssh.destination")
	if finding.Rule != "SSH_DEST_BLOCKED_BY_PUBLIC_SSH" {
		t.Errorf("ssh.destination fired %q, want %q", finding.Rule, "SSH_DEST_BLOCKED_BY_PUBLIC_SSH")
	}
	for _, want := range []string{"203.0.113.10:22", failingDetail} {
		if !strings.Contains(finding.Conclusion, want) {
			t.Errorf("the destination-block conclusion does not carry %q: %q", want, finding.Conclusion)
		}
	}

	human := stderr.String()
	for _, want := range []string{"203.0.113.10:22", failingDetail, string(probe.ReasonConnRefused), "SSH_DEST_BLOCKED_BY_PUBLIC_SSH"} {
		if !strings.Contains(human, want) {
			t.Errorf("the human projection does not carry %q", want)
		}
	}
}

// TestForbiddenStringsAbsent asserts the RG-4 and RG-6 wording boundaries over
// one complete run's two projections.
//
// RG-4: the WSL2 text carries the documented vmIdleTimeout semantics —
// milliseconds of idle, the default 60000, Windows 11 only — and neither the
// child-of-init shutdown rule nor the `vmIdleTimeout=-1` sentinel appears
// anywhere, because neither is documented in the reference material this slice
// could verify.
//
// RG-6: the run produces the HTTP/2 recommendation (the positive control that
// keeps the check from passing vacuously), and no projection claims the fallback
// was applied or a configuration written, that the HTTP/2 path is equivalent to
// QUIC, or that anything was enforced.
func TestForbiddenStringsAbsent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	got := doctor.Run(context.Background(), doctor.Options{JSON: true},
		doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, wsl2ReplaySeams())

	if got != doctor.ExitOK {
		t.Fatalf("Run returned exit %d, want %d for a complete WSL2 replay", got, doctor.ExitOK)
	}

	document := decodeRunDocument(t, stdout.Bytes())
	if document.Node.Platform != "wsl2" {
		t.Fatalf("node.platform is %q, want the scripted WSL2 classification", document.Node.Platform)
	}
	for _, want := range []string{"milliseconds", "60000", "Windows 11"} {
		if !strings.Contains(document.Node.Note, want) {
			t.Errorf("the WSL2 note does not carry the documented semantics %q: %q", want, document.Node.Note)
		}
	}
	if !strings.Contains(stderr.String(), "milliseconds of idle") {
		t.Errorf("the human projection does not carry the WSL2 wording")
	}

	// The positive control: the run must actually carry the HTTP/2 advice, or the
	// absence of an enforcement claim below would prove nothing.
	if !strings.Contains(stderr.String(), "this slice recommends and does not enforce") {
		t.Fatalf("the run produced no HTTP/2 recommendation, so the no-enforcement assertion would pass vacuously")
	}

	projections := []struct {
		name string
		text string
	}{
		{name: "the machine document", text: stdout.String()},
		{name: "the human projection", text: stderr.String()},
	}
	for _, projection := range projections {
		for _, forbidden := range []string{"child-of-init", "child of init", "vmIdleTimeout=-1", "vmIdleTimeout = -1"} {
			if strings.Contains(projection.text, forbidden) {
				t.Errorf("%s carries the forbidden WSL2 text %q (RG-4)", projection.name, forbidden)
			}
		}
		for _, forbidden := range []string{"has been applied", "was applied to", "a configuration was written", "equivalent", "enforced"} {
			if strings.Contains(projection.text, forbidden) {
				t.Errorf("%s claims HTTP/2 enforcement or equivalence with %q (RG-6)", projection.name, forbidden)
			}
		}
	}
}
