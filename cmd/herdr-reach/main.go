// Command herdr-reach is the headless doctor of R1a: one measurement run per
// invocation and nothing else.
//
// The entrypoint is deliberately thin. It builds the production seam set — the
// one construction site of real network primitives, internal/probe/real.go —
// parses the command line through the doctor package's flag surface, and exits
// with the code the harness returned. It measures nothing itself, writes nothing
// to the machine, and starts no process: the production seam set carries no
// command runner, so no path reachable from here can execute a third-party
// binary (R-HR-02).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Luisalt20/herdr-reach/internal/doctor"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// main is the process boundary and nothing more: the streams are the process's
// own, the seams are the production set, and the exit code is run's.
func main() {
	os.Exit(run(os.Args[1:], doctor.Stdio{Stdout: os.Stdout, Stderr: os.Stderr}, probe.ProductionSeams()))
}

// run is the entrypoint's in-process shape: parse the command line, answer a
// refusal on the error writer, or hand the parsed options to doctor.Run. It
// returns the exit code instead of calling os.Exit, so the whole path — including
// a usage error and a failed projection write — is reachable in process by the
// suite (design §4).
//
// A refused command line prints the typed usage error and the surface's own
// usage text to the error writer and returns ExitUsage. The usage text has
// exactly one home, the doctor package, so the refusal asks for it the way the
// documented surface does — a help request — rather than carrying a second copy
// of the text that would drift at the first flag change. No seam is touched on
// that path: the harness answers a help request before any measurement.
func run(args []string, stdio doctor.Stdio, seams probe.Seams) int {
	opts, err := doctor.ParseFlags(args)
	if err != nil {
		fmt.Fprintln(stdio.Stderr, err)
		// The help mode's return code is deliberately discarded: the request was
		// answered with usage, but the command line itself was refused, and
		// ExitUsage is what a refusal exits with.
		_ = doctor.Run(context.Background(), doctor.Options{Help: true}, stdio, seams)
		return doctor.ExitUsage
	}
	return doctor.Run(context.Background(), opts, stdio, seams)
}
