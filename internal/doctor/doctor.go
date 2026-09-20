// Package doctor owns the headless run harness of design §4: one call that ties
// the measurement, reasoning, transport and projection layers of one doctor run
// together, and the exit-code contract that call reports.
//
// The harness is deliberately thin and side-effect free. It measures nothing the
// runner did not measure, derives nothing the reasoning layer did not derive,
// decides nothing the transport layer did not decide, and writes nowhere except
// the two writers it is handed. It never calls os.Exit: Run returns the exit
// code and the entrypoint (cmd/herdr-reach, PR 19) exits with it, so the whole
// path — including a refused run and a failed projection write — is reachable
// in-process by the suite.
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/report"
	"github.com/Luisalt20/herdr-reach/internal/transport"
	"github.com/Luisalt20/herdr-reach/internal/version"
)

// Stdio is the pair of writers one run projects to. It is a value the caller
// supplies — production passes the process's own streams, a test passes buffers
// — so the harness never reaches for os.Stdout or os.Stderr and a case can drive
// a whole run without touching the process's streams.
type Stdio struct {
	// Stdout receives the machine-readable document, and nothing else, exactly
	// when the run asked for it.
	Stdout io.Writer
	// Stderr receives all human-facing output: the human projection of every
	// run, a refused run's error, and the usage text of a help request.
	Stderr io.Writer
}

// Run performs one doctor run and returns its exit code (design §4, §3.4).
//
// The flow is the design's data flow and nothing more: resolve the effective
// declared set, build the runner from one set of effective probe.Options, run
// the registry's probe suite, reason over the results, evaluate the transport
// candidates, map the whole run to one payload, write the human projection to
// the error writer always and the machine-readable document to the output writer
// only with --json, and return the code the payload's completeness decides.
//
// Two non-run modes are answered first, because neither is a measurement: a help
// request writes usage to the error writer, and a version request writes the
// tool's version to the output writer; both exit 0, and neither touches the
// seams. The help request is answered first when both are asked for, since its
// usage names the version flag.
//
// The run's clock always comes from the seams: a run that stamped values from
// two clocks could not support the determinism its output promises. The
// effective bounds are read back from the runner and handed to the report, so
// run.concurrency and run.run_budget_ms state what the run actually used rather
// than what the caller left unset.
//
// A refused run input or a projection that cannot reach its stream returns
// ExitUsage. In those cases no diagnosis is presented as completed, and the
// output writer is touched by the document alone: nothing else is ever written
// there.
func Run(ctx context.Context, opts Options, stdio Stdio, seams probe.Seams) int {
	if opts.Help {
		if err := writeUsage(stdio.Stderr); err != nil {
			return ExitUsage
		}
		return ExitOK
	}
	if opts.Version {
		if _, err := io.WriteString(stdio.Stdout, version.String()+"\n"); err != nil {
			return ExitUsage
		}
		return ExitOK
	}

	input := probe.TargetInput{Hub: opts.Hub, Overrides: opts.Targets}
	declared, err := probe.EffectiveTargets(input)
	if err != nil {
		// A run assembled by hand can carry an unusable hub address or override
		// that the flag parser never saw, so the refusal is made again here
		// before anything is measured. No seam is touched and no diagnosis is
		// presented.
		fmt.Fprintln(stdio.Stderr, version.Name+" doctor: "+err.Error())
		return ExitUsage
	}

	runOptions := opts.Run
	// The run's clock is the seam clock, always: the seam set is the run's one
	// clock home (design §6.1), and a clock left in opts.Run is not read, because
	// two clocks in one run could stamp values from two timelines.
	runOptions.Clock = seams.Clock
	runner := probe.NewRunner(runOptions)
	effective := runner.Options()

	results := runner.Run(ctx, probe.ProbesFor(seams, input), nil)
	diagnosisResult := diagnosis.Diagnose(results)

	// The registry's order is the contract order of the transport rows, and
	// Evaluate preserves it, so the name travels beside its own feasibility
	// instead of being matched to a verdict by position afterwards.
	candidates := transport.Registry()
	feasibilities := transport.Evaluate(candidates, diagnosisResult)
	transports := make([]report.TransportRow, 0, len(candidates))
	for index, candidate := range candidates {
		transports = append(transports, report.TransportRow{
			Name:        candidate.Name(),
			Feasibility: feasibilities[index],
		})
	}

	payload := report.Build(report.Input{
		Run:         effective,
		ToolVersion: version.Version,
		Declared:    declared,
		Results:     results,
		Diagnosis:   diagnosisResult,
		Transports:  transports,
	})

	if err := report.WriteHuman(stdio.Stderr, payload); err != nil {
		return ExitUsage
	}

	if opts.JSON {
		document, err := json.Marshal(payload)
		if err != nil {
			return ExitUsage
		}
		// One write of one document plus its terminator: a writer failure
		// cannot leave a second value or human text beside it, and the newline
		// keeps the document line-oriented without becoming a second write of
		// its own.
		document = append(document, '\n')
		if _, err := stdio.Stdout.Write(document); err != nil {
			return ExitUsage
		}
	}

	// The code is read from the built payload, not recomputed from the results:
	// the exit code and run.completeness are two readings of one value, so they
	// cannot disagree.
	if payload.Run.Completeness == report.CompletenessIncomplete {
		return ExitIncomplete
	}
	return ExitOK
}
