package doctor_test

// This file is the doctor harness's suite of design §4 and §7: the exit matrix
// over scripted seams, the stdout/stderr split of the two projections, the
// harness's use of the run's effective bounds, and the assertion that the
// exit-code constants are exactly the codes the documented table states.
//
// Every case drives the harness through doctor.Run and reads the run's own
// output, never the harness's internals. The payload is inspected by parsing the
// machine-readable document, so "the exit code and run.completeness agree" is an
// assertion about what a script would actually read rather than about a value the
// test recomputed from the same inputs. The seams are scripted rather than
// mocked: the deny-all default of design §6.2 supplies a machine whose denied
// capabilities read as not-measured attempts, and the platform seam decides only
// which machine the run classifies. That is enough to reach every exit code
// without a socket, a file or a process.

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Luisalt20/herdr-reach/internal/doctor"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// --- scripted machines -------------------------------------------------------

// scriptedPlatform is the one seam beyond the deny-all default that a case
// scripted as a machine needs: the classification is a pure function of it, so a
// case states "linux/amd64" or "windows/amd64" without being on either machine.
type scriptedPlatform struct {
	goos string
	arch string
	// wsl2 is the scripted WSL2 signal. It is false unless a case describes a
	// WSL2 instance: a machine a case did not describe must not claim one.
	wsl2 bool
}

// GOOS reports the scripted operating system.
func (p scriptedPlatform) GOOS() string { return p.goos }

// Arch reports the scripted architecture.
func (p scriptedPlatform) Arch() string { return p.arch }

// WSL2 reports the scripted WSL2 signal.
func (p scriptedPlatform) WSL2() bool { return p.wsl2 }

// Systemd reports no service-manager signal, for the same reason.
func (p scriptedPlatform) Systemd() bool { return false }

// linuxSeams is the smallest scripted machine the exit matrix can run on. Every
// capability of the deny-all default still refuses, and every probe reports a
// refused capability as a not-measured observation — which never makes a run
// incomplete. The platform seam is the one capability that cannot be denied, and
// it answers a classified Linux, so local.env is a measured pass and local.sshd's
// binary is a measured absence. The run is therefore complete and carries a
// measured negative answer, which is exactly the state exit 0 describes.
func linuxSeams() probe.Seams {
	seams := probe.DenyAllSeams()
	seams.Platform = scriptedPlatform{goos: "linux", arch: "amd64"}
	return seams
}

// refusingDialer answers every dial with the far end's refusal. It is the
// measured-negative script: the hub-directed probe dials and the far end answers
// that nothing is listening, which is a measurement of the address rather than an
// ambiguity.
type refusingDialer struct{}

// DialContext reports the refused connection and returns no conn.
func (refusingDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return nil, syscall.ECONNREFUSED
}

// hangingDialer ignores the dial context entirely and waits for the test to
// release it. Ignoring the context is the point: a probe that honours its budget
// reports its own measured outcome, while this one has to be abandoned by the
// runner, which is the hanging-probe case exit 1 covers.
type hangingDialer struct {
	release <-chan struct{}
}

// DialContext waits for the release and then answers with an error. The value is
// never classified: by the time it arrives the run has stopped collecting it.
func (d hangingDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	<-d.release
	return nil, errors.New("the hanging dial was released after the run abandoned it")
}

// unresolvedVerifier answers every handshake with a plain error: an attempt that
// produced no answer, which is unresolved and makes the run incomplete. It is
// deliberately not ErrSeamDenied (an attempt that was not made) and not
// ErrTLSVerification (a measured rejection), because those are different facts.
type unresolvedVerifier struct{}

// Verify reports that the handshake produced no answer.
func (unresolvedVerifier) Verify(context.Context, string, *tls.Config) (probe.TLSVerification, error) {
	return probe.TLSVerification{}, errors.New("the handshake produced no answer")
}

// --- payload reading ---------------------------------------------------------

// runDocument is the subset of the schema_version "1" document the exit matrix
// reads. It is deliberately partial: the matrix asserts the harness's contract —
// completeness, coverage, the resolved hub, the probe rows it reasons about and
// the transport verdicts — and the payload's full key set is pinned by the report
// suite.
type runDocument struct {
	Run struct {
		Completeness string   `json:"completeness"`
		Unresolved   []string `json:"unresolved"`
		NotMeasured  []string `json:"not_measured"`
		Concurrency  int      `json:"concurrency"`
		RunBudgetMS  int64    `json:"run_budget_ms"`
	} `json:"run"`
	Targets struct {
		Hub *string `json:"hub"`
	} `json:"targets"`
	Probes     []probeRow     `json:"probes"`
	Findings   []findingRow   `json:"findings"`
	Transports []transportRow `json:"transports"`
	Node       struct {
		Platform string `json:"platform"`
		Arch     string `json:"arch"`
		Refused  bool   `json:"refused"`
		Note     string `json:"note"`
	} `json:"node"`
}

// findingRow is one entry of the document's findings array.
type findingRow struct {
	Question   string   `json:"question"`
	Rule       string   `json:"rule"`
	Conclusion string   `json:"conclusion"`
	DependsOn  []string `json:"depends_on"`
}

// probeRow is one entry of the document's probes array.
type probeRow struct {
	Name       string  `json:"name"`
	Target     *string `json:"target"`
	Verdict    string  `json:"verdict"`
	Resolution string  `json:"resolution"`
	Reason     string  `json:"reason"`
	Detail     string  `json:"detail"`
}

// transportRow is one entry of the document's transports array.
type transportRow struct {
	Name   string `json:"name"`
	Viable bool   `json:"viable"`
}

// decodeRunDocument parses stdout as exactly one machine-readable document. The
// second Decode must find EOF: a projection that wrote the document twice, or
// mixed a sentence into it, is not the one-document contract R-HR-07 states, and
// this helper fails instead of accepting the first value it can read.
func decodeRunDocument(t *testing.T, stdout []byte) runDocument {
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

// rowFor returns the document's probe row for name, failing the case when the
// row is absent: a conclusion about a probe must fail loudly rather than compare
// a zero value against it.
func rowFor(t *testing.T, document runDocument, name string) probeRow {
	t.Helper()
	for _, row := range document.Probes {
		if row.Name == name {
			return row
		}
	}
	t.Fatalf("the document carries no probe row for %q", name)
	return probeRow{}
}

// containsName reports whether names carries name.
func containsName(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}

// --- the exit matrix ---------------------------------------------------------

// TestExitMatrixOverScriptedSeams is the exit matrix of design §3.4 and R-HR-07:
// a completed measurement — including a negative answer, a run with no viable
// transport and a supported native-Windows classification — exits 0; a run in
// which at least one probe was attempted and resolved unresolved exits 1,
// including the hanging probe the runner had to abandon; and a not-measured
// measurement alone never moves the code. Every case also asserts that the exit
// code and the document's run.completeness agree, so the two can never drift
// apart.
func TestExitMatrixOverScriptedSeams(t *testing.T) {
	const (
		complete   = "complete"
		incomplete = "incomplete"
	)

	cases := []struct {
		name string
		// seams builds a fresh seam set per case: a case must not be able to
		// observe another case's overrides.
		seams func() probe.Seams
		opts  doctor.Options
		// wantExit is the exit code the evidence decides.
		wantExit int
		// wantCompleteness is the run.completeness the same evidence must state.
		wantCompleteness string
		// wantUnresolved and wantNotMeasured, when set, must appear in the
		// corresponding coverage list.
		wantUnresolved  string
		wantNotMeasured string
		// wantHubAbsent asserts targets.hub is JSON null.
		wantHubAbsent bool
		// wantNegatedProbe, when set, must be a measured failure carrying the
		// refused-connection reason: the negative answer that is still success.
		wantNegatedProbe string
		// wantAbandonedReason, when set together with wantUnresolved, is the
		// reason the abandoned probe must carry (the runner's own fact).
		wantAbandonedReason string
		// wantAllTransportsUnviable asserts the document reports no viable
		// transport at all.
		wantAllTransportsUnviable bool
		// wantNodePlatform, wantNodeArch and wantNodeRefused assert the node
		// classification when the case scripts a machine; wantNodeRefused pins the
		// payload field that must stay false now that no platform is refused.
		wantNodePlatform string
		wantNodeArch     string
		wantNodeRefused  bool
	}{
		{
			name: "a measured negative answer is a completed run",
			seams: func() probe.Seams {
				seams := linuxSeams()
				seams.Dialer = refusingDialer{}
				return seams
			},
			opts:                      doctor.Options{JSON: true, Hub: "203.0.113.10"},
			wantExit:                  doctor.ExitOK,
			wantCompleteness:          complete,
			wantNegatedProbe:          "egress.hub.direct",
			wantAllTransportsUnviable: true,
		},
		{
			name:             "no viable transport is still a completed run",
			seams:            linuxSeams,
			opts:             doctor.Options{JSON: true},
			wantExit:         doctor.ExitOK,
			wantCompleteness: complete,
			wantHubAbsent:    true,
			wantNotMeasured:  "egress.hub.direct",
			// The hub was never attempted, so no transport may be reported viable
			// on the strength of hub reachability; here none is viable at all.
			wantAllTransportsUnviable: true,
		},
		{
			name: "a supported native-Windows classification is a completed run",
			seams: func() probe.Seams {
				seams := probe.DenyAllSeams()
				seams.Platform = scriptedPlatform{goos: "windows", arch: "amd64"}
				return seams
			},
			opts:             doctor.Options{JSON: true},
			wantExit:         doctor.ExitOK,
			wantCompleteness: complete,
			wantNodePlatform: "windows-native",
			wantNodeArch:     "amd64",
			wantNodeRefused:  false,
		},
		{
			name: "an attempted probe that resolved unresolved makes the run incomplete",
			seams: func() probe.Seams {
				seams := linuxSeams()
				seams.TLSVerifier = unresolvedVerifier{}
				return seams
			},
			opts:             doctor.Options{JSON: true},
			wantExit:         doctor.ExitIncomplete,
			wantCompleteness: incomplete,
			wantUnresolved:   "tls.interception",
		},
		{
			name: "a probe the runner had to abandon makes the run incomplete",
			seams: func() probe.Seams {
				release := make(chan struct{})
				t.Cleanup(func() { close(release) })
				seams := linuxSeams()
				seams.Dialer = hangingDialer{release: release}
				return seams
			},
			opts: doctor.Options{
				JSON: true,
				Hub:  "203.0.113.10",
				// The probe bound is injected in milliseconds so the case does
				// not wait out the production default; the runner's own fact —
				// the probe ignored its bound — is what the document must state.
				Run: probe.Options{ProbeTimeout: 20 * time.Millisecond, RunBudget: time.Second},
			},
			wantExit:            doctor.ExitIncomplete,
			wantCompleteness:    incomplete,
			wantUnresolved:      "egress.hub.direct",
			wantAbandonedReason: string(probe.ReasonProbeTimeout),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			got := doctor.Run(context.Background(), tc.opts, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, tc.seams())

			if got != tc.wantExit {
				t.Fatalf("Run returned exit %d, want %d", got, tc.wantExit)
			}
			if stdout.Len() == 0 {
				t.Fatalf("the run wrote no machine-readable document to stdout")
			}
			if stderr.Len() == 0 {
				t.Fatalf("the run wrote no human projection to stderr")
			}

			document := decodeRunDocument(t, stdout.Bytes())
			if document.Run.Completeness != tc.wantCompleteness {
				t.Errorf("run.completeness is %q, want %q", document.Run.Completeness, tc.wantCompleteness)
			}
			if (got == doctor.ExitOK) != (document.Run.Completeness == complete) {
				t.Errorf("exit code %d and run.completeness %q disagree", got, document.Run.Completeness)
			}
			if tc.wantUnresolved != "" && !containsName(document.Run.Unresolved, tc.wantUnresolved) {
				t.Errorf("run.unresolved %v does not name %q", document.Run.Unresolved, tc.wantUnresolved)
			}
			if tc.wantNotMeasured != "" && !containsName(document.Run.NotMeasured, tc.wantNotMeasured) {
				t.Errorf("run.not_measured %v does not name %q", document.Run.NotMeasured, tc.wantNotMeasured)
			}
			if tc.wantHubAbsent && document.Targets.Hub != nil {
				t.Errorf("targets.hub is %q, want JSON null for a run with no hub", *document.Targets.Hub)
			}
			if tc.wantNegatedProbe != "" {
				row := rowFor(t, document, tc.wantNegatedProbe)
				if row.Resolution != "measured" || row.Verdict != "fail" {
					t.Errorf("%s is %s/%s, want a measured failure", tc.wantNegatedProbe, row.Resolution, row.Verdict)
				}
				if row.Reason != string(probe.ReasonConnRefused) {
					t.Errorf("%s carries reason %q, want %q", tc.wantNegatedProbe, row.Reason, probe.ReasonConnRefused)
				}
			}
			if tc.wantAbandonedReason != "" {
				row := rowFor(t, document, tc.wantUnresolved)
				if row.Resolution != "unresolved" {
					t.Errorf("%s is %q, want unresolved", tc.wantUnresolved, row.Resolution)
				}
				if row.Reason != tc.wantAbandonedReason {
					t.Errorf("%s carries reason %q, want %q", tc.wantUnresolved, row.Reason, tc.wantAbandonedReason)
				}
			}
			if tc.wantAllTransportsUnviable {
				if len(document.Transports) == 0 {
					t.Fatalf("the document carries no transport row to judge")
				}
				for _, row := range document.Transports {
					if row.Viable {
						t.Errorf("transport %q is reported viable in a run where none can be", row.Name)
					}
				}
			}
			if tc.wantNodePlatform != "" {
				if document.Node.Platform != tc.wantNodePlatform {
					t.Errorf("node.platform is %q, want %q", document.Node.Platform, tc.wantNodePlatform)
				}
				if document.Node.Arch != tc.wantNodeArch {
					t.Errorf("node.arch is %q, want %q", document.Node.Arch, tc.wantNodeArch)
				}
				if document.Node.Refused != tc.wantNodeRefused {
					t.Errorf("node.refused is %v, want %v", document.Node.Refused, tc.wantNodeRefused)
				}
			}
		})
	}
}

// TestExitUsageAndInternalFailures is the exit-2 half of the matrix: unusable run
// input the parser refused, an unusable hub the harness refused before measuring,
// and an internal failure whose projection could not reach its stream. In every
// case no diagnosis is presented as completed and the output writer carries no
// human text; the unknown-flag case asserts the typed error the entrypoint maps
// to this code.
func TestExitUsageAndInternalFailures(t *testing.T) {
	t.Run("an unknown flag is a typed usage error", func(t *testing.T) {
		_, err := doctor.ParseFlags([]string{"--bogus"})
		if err == nil {
			t.Fatalf("ParseFlags accepted an unknown flag")
		}
		if !errors.Is(err, doctor.ErrUnknownFlag) {
			t.Fatalf("the error %v does not carry ErrUnknownFlag", err)
		}
		var usage *doctor.UsageError
		if !errors.As(err, &usage) {
			t.Fatalf("the error %v is not a typed UsageError", err)
		}
		if doctor.ExitUsage != 2 {
			t.Fatalf("ExitUsage is %d, want 2 for an unknown flag", doctor.ExitUsage)
		}
	})

	t.Run("an unusable hub value is refused before any measurement", func(t *testing.T) {
		trip := newTripwire()
		var stdout, stderr bytes.Buffer
		got := doctor.Run(context.Background(), doctor.Options{JSON: true, Hub: ":22"},
			doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, trip.seams())

		if got != doctor.ExitUsage {
			t.Fatalf("Run returned exit %d, want %d", got, doctor.ExitUsage)
		}
		if stdout.Len() != 0 {
			t.Errorf("the output writer received %q; no diagnosis may be presented as completed", stdout.String())
		}
		if stderr.Len() == 0 {
			t.Errorf("the refusal was not reported on the error writer")
		}
		if trip.calls.Load() != 0 {
			t.Errorf("the run touched the seams %d times; unusable input must start no measurement", trip.calls.Load())
		}
	})

	t.Run("a document write failure exits two with only the document offered to output", func(t *testing.T) {
		refused := &recordingFailingWriter{}
		var stderr bytes.Buffer
		got := doctor.Run(context.Background(), doctor.Options{JSON: true},
			doctor.Stdio{Stdout: refused, Stderr: &stderr}, linuxSeams())

		if got != doctor.ExitUsage {
			t.Fatalf("Run returned exit %d, want %d for a failed document write", got, doctor.ExitUsage)
		}
		if len(refused.attempts) != 1 {
			t.Fatalf("the output writer saw %d writes, want exactly the one document", len(refused.attempts))
		}
		// The writer was offered the document and nothing else: no human text,
		// no banner and no partial sentence about the run.
		if !bytes.HasPrefix(refused.attempts[0], []byte("{")) {
			t.Errorf("the output writer was offered %q, which is not the document", refused.attempts[0])
		}
		if bytes.Contains(refused.attempts[0], []byte("PROBES")) {
			t.Errorf("the output writer was offered human text: %q", refused.attempts[0])
		}
	})

	t.Run("a human projection write failure exits two with output untouched", func(t *testing.T) {
		refused := &recordingFailingWriter{}
		var stdout bytes.Buffer
		got := doctor.Run(context.Background(), doctor.Options{JSON: true},
			doctor.Stdio{Stdout: &stdout, Stderr: refused}, linuxSeams())

		if got != doctor.ExitUsage {
			t.Fatalf("Run returned exit %d, want %d for a failed human write", got, doctor.ExitUsage)
		}
		// The document is written after the human projection, so a failed human
		// write must leave the output writer untouched: there is no half-run
		// document to mistake for a completed diagnosis.
		if stdout.Len() != 0 {
			t.Errorf("the output writer received %q after a failed human write", stdout.String())
		}
	})
}

// recordingFailingWriter is an io.Writer that refuses every write and records
// what it was offered, so a case can assert both the failure and that nothing
// else was ever sent to that stream.
type recordingFailingWriter struct {
	attempts [][]byte
}

// Write records the offered bytes and refuses them.
func (w *recordingFailingWriter) Write(p []byte) (int, error) {
	w.attempts = append(w.attempts, append([]byte(nil), p...))
	return 0, errors.New("the stream refused the write")
}

// --- the stream split --------------------------------------------------------

// TestDoctorSplitsTheMachineDocumentFromTheHumanReport is R-HR-07's split: with
// the machine-readable flag stdout carries exactly one parseable document and no
// human text while the human report lands on stderr; without the flag stdout
// stays empty and the human report still lands on stderr.
func TestDoctorSplitsTheMachineDocumentFromTheHumanReport(t *testing.T) {
	t.Run("with the machine-readable flag", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		got := doctor.Run(context.Background(), doctor.Options{JSON: true},
			doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, linuxSeams())

		if got != doctor.ExitOK {
			t.Fatalf("Run returned exit %d, want %d", got, doctor.ExitOK)
		}
		if stdout.Len() == 0 {
			t.Fatalf("stdout is empty; the document was not written")
		}
		// decodeRunDocument also proves the "exactly one document" half: a
		// second value after the document fails there.
		decodeRunDocument(t, stdout.Bytes())
		if bytes.Contains(stdout.Bytes(), []byte("PROBES")) || bytes.Contains(stdout.Bytes(), []byte("completeness: ")) {
			t.Errorf("stdout carries human text beside the document: %q", stdout.String())
		}
		if !strings.Contains(stderr.String(), "PROBES") {
			t.Errorf("the human report did not land on stderr: %q", stderr.String())
		}
	})

	t.Run("without the machine-readable flag", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		got := doctor.Run(context.Background(), doctor.Options{},
			doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, linuxSeams())

		if got != doctor.ExitOK {
			t.Fatalf("Run returned exit %d, want %d", got, doctor.ExitOK)
		}
		if stdout.Len() != 0 {
			t.Errorf("stdout carries %q; without the flag it must stay empty", stdout.String())
		}
		if !strings.Contains(stderr.String(), "PROBES") {
			t.Errorf("the human report did not land on stderr: %q", stderr.String())
		}
	})
}

// TestDoctorReportEchoesTheEffectiveRunOptions asserts the report is built from
// the run's effective probe.Options, not from the caller's struct: an injected
// bound is what run.concurrency and run.run_budget_ms report, and a zero-value
// bound reports the documented defaults rather than zero.
func TestDoctorReportEchoesTheEffectiveRunOptions(t *testing.T) {
	t.Run("an injected bound is reported", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		opts := doctor.Options{JSON: true, Run: probe.Options{
			Concurrency: 2,
			RunBudget:   1500 * time.Millisecond,
		}}
		if got := doctor.Run(context.Background(), opts, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, linuxSeams()); got != doctor.ExitOK {
			t.Fatalf("Run returned exit %d, want %d", got, doctor.ExitOK)
		}
		document := decodeRunDocument(t, stdout.Bytes())
		if document.Run.Concurrency != 2 {
			t.Errorf("run.concurrency is %d, want the injected 2", document.Run.Concurrency)
		}
		if document.Run.RunBudgetMS != 1500 {
			t.Errorf("run.run_budget_ms is %d, want the injected 1500", document.Run.RunBudgetMS)
		}
	})

	t.Run("an unset bound reports the documented default", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if got := doctor.Run(context.Background(), doctor.Options{JSON: true},
			doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, linuxSeams()); got != doctor.ExitOK {
			t.Fatalf("Run returned exit %d, want %d", got, doctor.ExitOK)
		}
		document := decodeRunDocument(t, stdout.Bytes())
		if document.Run.Concurrency != probe.DefaultConcurrency {
			t.Errorf("run.concurrency is %d, want the documented default %d", document.Run.Concurrency, probe.DefaultConcurrency)
		}
		if want := probe.DefaultRunBudget.Milliseconds(); document.Run.RunBudgetMS != want {
			t.Errorf("run.run_budget_ms is %d, want the documented default %d", document.Run.RunBudgetMS, want)
		}
	})
}

// --- the documented table ----------------------------------------------------

// diagnosisDocRelativePath is the contract document as the repository root names
// it. This package's test binary runs in internal/doctor, so the document is two
// levels up.
const diagnosisDocRelativePath = "docs/diagnosis-report.md"

// TestExitDocumentedCodesMatchTheConstants asserts the exit-code constants are
// exactly the codes the document's "Exit codes" table states, in both
// directions. The assertion lives here rather than in the report package because
// internal/doctor imports internal/report and the reverse import would invert
// the dependency direction (design §7's PR 17 note).
func TestExitDocumentedCodesMatchTheConstants(t *testing.T) {
	document := readDiagnosisDoc(t)
	documented := exitCodeTableRows(t, document)
	// The positive control: a parse that found no table row must fail loudly
	// instead of passing as a comparison with nothing.
	if len(documented) == 0 {
		t.Fatalf("the exit-code table under %q yielded no rows: a parse that found nothing must not pass silently", "## Exit codes")
	}

	want := []string{
		strconv.Itoa(doctor.ExitOK),
		strconv.Itoa(doctor.ExitIncomplete),
		strconv.Itoa(doctor.ExitUsage),
	}
	missing, extra := setDifference(want, documented), setDifference(documented, want)
	if len(missing) > 0 || len(extra) > 0 {
		t.Errorf("the documented exit codes are not the constants' values:\nmissing from the document: %v\nextra in the document:     %v", missing, extra)
	}
}

// readDiagnosisDoc reads the contract document from the repository root, failing
// the case when it is absent or empty.
func readDiagnosisDoc(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(diagnosisDocRelativePath)))
	if err != nil {
		t.Fatalf("%s could not be read from the repository root: %v", diagnosisDocRelativePath, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		t.Fatalf("%s is empty", diagnosisDocRelativePath)
	}
	return string(data)
}

// exitCodeTableRows returns the first column of the first markdown table after
// the exact "## Exit codes" heading line, with the inline-code backticks stripped.
// The header and separator rows are skipped and the table ends at the first
// non-table line after it started; a missing heading fails the case, so the table
// cannot silently stop being found.
func exitCodeTableRows(t *testing.T, document string) []string {
	t.Helper()
	lines := strings.Split(document, "\n")

	headingAt := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Exit codes" {
			headingAt = i
			break
		}
	}
	if headingAt < 0 {
		t.Fatalf("%s has no heading %q", diagnosisDocRelativePath, "## Exit codes")
	}

	var (
		rows      []string
		tableRows int
	)
	for _, line := range lines[headingAt+1:] {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			if tableRows > 0 {
				break
			}
			continue
		}
		cells := strings.Split(trimmed, "|")
		if len(cells) < 3 {
			continue
		}
		cell := strings.Trim(strings.TrimSpace(cells[1]), "`")
		tableRows++
		if tableRows == 1 || strings.Trim(cell, "-: ") == "" {
			// The header row, then the separator row.
			continue
		}
		if cell == "" {
			t.Fatalf("a row of the exit-code table has an empty first column")
		}
		rows = append(rows, cell)
	}
	return rows
}

// setDifference returns the entries of a that are not in b, in a's order.
func setDifference(a, b []string) []string {
	present := make(map[string]bool, len(b))
	for _, entry := range b {
		present[entry] = true
	}
	var difference []string
	for _, entry := range a {
		if !present[entry] {
			difference = append(difference, entry)
		}
	}
	return difference
}

// --- the recording seam set (design §6.2, level 2) ---------------------------

// recordedTriple is one outbound attempt in the (host, port, protocol) form the
// declared target set uses. Each recording seam records the triple the protocol
// it owns measures, so the comparison with probe.EffectiveTargets compares like
// values rather than a seam's private idea of what it dialed.
type recordedTriple struct {
	Host     string
	Port     int
	Protocol probe.Protocol
}

// recordingSeams is the recording seam set of design §6.2 level 2: every TCP
// dial, packet socket and TLS handshake is recorded as the triple its protocol
// measures and then denied with probe.ErrSeamDenied. An attempt is therefore
// visible in the record, nothing leaves the process, and an attempt against a
// target the declaration does not carry appears as an undeclared triple.
//
// The resolver is the one seam that answers instead of denying, and the reason is
// structural rather than permissive: the name-based probes resolve before they
// dial, so a denying resolver would stop them before the dial seam that measures
// their protocol was reached and the declared set could never be compared. The
// answer is a scripted documentation address and never leaves the process.
//
// The command runner is wired to the counter so a case can observe every
// invocation. The writes-nothing proof asks for withoutCommandRunner — the
// production command shape — because internal/probe/real.go leaves the
// production CommandRunner nil and only a shape with no runner can be called
// zero times.
type recordingSeams struct {
	mu       sync.Mutex
	triples  map[recordedTriple]bool
	commands *recordingCommandRunner
	seams    probe.Seams
}

// newRecordingSeams builds a fresh recording set with the counting command
// runner wired.
func newRecordingSeams() *recordingSeams {
	recording := &recordingSeams{
		triples:  make(map[recordedTriple]bool),
		commands: &recordingCommandRunner{},
	}
	seams := probe.DenyAllSeams()
	seams.Platform = scriptedPlatform{goos: "linux", arch: "amd64"}
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
// run in that shape has no execution path at all. It is the shape the
// writes-nothing proof runs under.
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

// recordingCommandRunner records every invocation and denies it, so a case can
// assert both that the run attempted an execution and that the attempt never
// became one.
type recordingCommandRunner struct {
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
func (r *recordingCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, []byte, error) {
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
func (r *recordingCommandRunner) invocations() []commandInvocation {
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
	sort.Strings(difference)
	return difference
}

// findingFor returns the document's finding for question, failing the case when
// the finding is absent: a conclusion must fail loudly rather than compare a
// zero value against it.
func findingFor(t *testing.T, document runDocument, question string) findingRow {
	t.Helper()
	for _, finding := range document.Findings {
		if finding.Question == question {
			return finding
		}
	}
	t.Fatalf("the document carries no finding for question %q", question)
	return findingRow{}
}

// --- the PRD §1.1 replay seams -----------------------------------------------

// matrixReplaySeams builds the scripted seams PRD §1.1's matrix implies, through
// the real pipeline: the hub address refuses, the two public SSH targets answer
// with an identification string, the Cloudflare edge accepts a connection, and
// the chain verifies with the issuer and code the specification recorded. The
// seams the matrix does not script — the datagram probe and the command-derived
// local.sshd observations — stay denied, which is a not-measured absence and
// never an unresolved one, so the run is complete.
func matrixReplaySeams() probe.Seams {
	seams := probe.DenyAllSeams()
	seams.Platform = scriptedPlatform{goos: "linux", arch: "amd64"}
	seams.Resolver = answeringResolver{}
	seams.Dialer = scriptedMatrixDialer{}
	seams.TLSVerifier = verifyingTLSVerifier{}
	return seams
}

// wsl2ReplaySeams is matrixReplaySeams on a WSL2 node, so the run's two
// projections carry the WSL2 wording the RG-4 assertions inspect.
func wsl2ReplaySeams() probe.Seams {
	seams := matrixReplaySeams()
	seams.Platform = scriptedPlatform{goos: "linux", arch: "amd64", wsl2: true}
	return seams
}

// scriptedMatrixDialer answers the addresses PRD §1.1 measured: the hub address
// refuses, the two public SSH targets speak SSH, and every other declared target
// accepts a connection.
type scriptedMatrixDialer struct{}

// DialContext answers the declared address.
func (scriptedMatrixDialer) DialContext(_ context.Context, _, addr string) (net.Conn, error) {
	switch addr {
	case "203.0.113.10:22":
		return nil, syscall.ECONNREFUSED
	case "github.com:22", "ssh.github.com:443":
		return &scriptedConn{banner: []byte("SSH-2.0-herdr-reach-scripted\r\n")}, nil
	default:
		return &scriptedConn{}, nil
	}
}

// scriptedConn is a connection a scripted dialer hands back: an optional SSH
// identification string is readable once, the deadline calls are accepted, and
// closing is a no-op. It is deliberately not a socket: a scripted seam must not
// be able to reach one by accident.
type scriptedConn struct {
	banner []byte
	read   bool
}

// Read returns the scripted banner once, then EOF.
func (c *scriptedConn) Read(p []byte) (int, error) {
	if c.read || len(c.banner) == 0 {
		return 0, io.EOF
	}
	c.read = true
	return copy(p, c.banner), nil
}

// Write accepts the bytes without doing anything with them.
func (c *scriptedConn) Write(p []byte) (int, error) { return len(p), nil }

// Close releases nothing: the connection was never a socket.
func (c *scriptedConn) Close() error { return nil }

// LocalAddr reports no address, for the same reason.
func (c *scriptedConn) LocalAddr() net.Addr { return nil }

// RemoteAddr reports no address, for the same reason.
func (c *scriptedConn) RemoteAddr() net.Addr { return nil }

// SetDeadline accepts the bound without a socket to enforce it on.
func (c *scriptedConn) SetDeadline(time.Time) error { return nil }

// SetReadDeadline accepts the bound without a socket to enforce it on.
func (c *scriptedConn) SetReadDeadline(time.Time) error { return nil }

// SetWriteDeadline accepts the bound without a socket to enforce it on.
func (c *scriptedConn) SetWriteDeadline(time.Time) error { return nil }

// verifyingTLSVerifier answers every handshake with the publisher and anchor PRD
// §1.1 recorded: the leaf's issuing organization, the anchor that recording
// machine's store selected, and the ok code. It refuses a configuration that
// disables verification, so the scripted seam cannot become the place where
// R-HR-04 is weakened.
type verifyingTLSVerifier struct{}

// Verify reports the recorded issuing organization, anchor and code.
func (verifyingTLSVerifier) Verify(_ context.Context, _ string, cfg *tls.Config) (probe.TLSVerification, error) {
	if cfg == nil || cfg.InsecureSkipVerify {
		return probe.TLSVerification{}, errors.New("the scripted verifier refuses a configuration that disables verification")
	}
	return probe.TLSVerification{Issuer: "Let's Encrypt", Anchor: "ISRG", VerificationCode: "0"}, nil
}
