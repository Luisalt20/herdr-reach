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
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"strings"
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
