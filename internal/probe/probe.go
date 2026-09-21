// Package probe owns the measurement vocabulary of herdr-reach: what a probe
// is, what a measurement's verdict and resolution may be, and the closed
// reason-code set every measurement's machine-readable outcome is drawn from.
//
// The vocabulary is deliberately pure. Nothing in this file dials, resolves,
// reads a file or runs a command: measurements are performed by probes over
// injected seams, and this file holds only the value types those measurements
// produce plus the reduction that turns a probe's observations into the single
// verdict its Result reports. Reasoning over the measurements lives in
// internal/diagnosis (DEV-1); the tool never decides transport viability here.
package probe

import (
	"context"
	"time"
)

// ProbeKind names the question family a probe belongs to (PRD §5.1).
type ProbeKind string

const (
	// ProbeLocal measures what this machine is and has.
	ProbeLocal ProbeKind = "local"
	// ProbeEgress measures what this network permits.
	ProbeEgress ProbeKind = "egress"
	// ProbeTLS measures whether something is intercepting TLS.
	ProbeTLS ProbeKind = "tls"
	// ProbeProto separates a blocked destination from a blocked protocol.
	ProbeProto ProbeKind = "proto"
)

// Verdict is a measurement's answer. It is exactly one of three values and is
// never silently treated as a pass: Indeterminate means "not known", not
// "fine" (PRD §5.1, PRD §14.5).
type Verdict string

const (
	// Pass is a definite positive answer to the probe's declared question.
	Pass Verdict = "pass"
	// Fail is a definite negative answer to the probe's declared question.
	Fail Verdict = "fail"
	// Indeterminate is the absence of an answer. It is never a pass.
	Indeterminate Verdict = "indeterminate"
)

// Resolution records how far a measurement got (DEV-2). A verdict alone cannot
// separate "the question was answered" from "the question was attempted and
// not answered" from "the question was never attempted", and the run's
// completeness and exit code depend on exactly that difference.
type Resolution string

const (
	// Measured means the probe's declared question was answered, so the
	// verdict is a definite pass or fail.
	Measured Resolution = "measured"
	// Unresolved means an attempt was made and produced nothing classifiable.
	// An unresolved observation makes the run incomplete.
	Unresolved Resolution = "unresolved"
	// NotMeasured means no attempt was made, because a required input or
	// capability is outside this slice's boundary. It never changes the
	// exit code and is never presented as a blocked target.
	NotMeasured Resolution = "not_measured"
)

// Observation is one fact a probe measured. It carries both the verbatim
// detail an operator reads and the stable machine reason code a script reads
// (R-HR-07), so neither has to be derived from the other.
//
// Label names the fact inside its probe ("tcp 7844 region1", "effective sshd
// config"). Target names the subject the observation is about — a dialed
// address, a local path, a unit list, a service name — and is empty when the
// observation is about no subject; resolution plays no part. egress.quic's
// internal-failure observation carries no target and is unresolved, while
// local.sshd's not-measured service and config observations still carry their
// unit list and config path.
type Observation struct {
	Label      string
	Target     string
	Resolution Resolution
	Verdict    Verdict
	Reason     ReasonCode
	Detail     string
}

// Result is one probe's outcome. It keeps PRD §5.1's fields at the top level
// and adds the resolution, reason and per-observation detail the diagnosis
// domain needs (DEV-2). Verdict and Reason are the worst-of reduction over
// Observations (see Aggregate); Detail carries the per-observation lines
// verbatim.
type Result struct {
	Probe        string
	Kind         ProbeKind
	Target       string
	Verdict      Verdict
	Reason       ReasonCode
	Detail       string
	Elapsed      time.Duration
	Observations []Observation
}

// Probe is one independently meaningful measurement (PRD §5.1).
// Implementations never mutate the measured machine. Because a measurement's
// targets and seams are supplied to the probe's constructor, Run needs nothing
// but its context.
type Probe interface {
	// Name is the probe's stable identifier, e.g. "egress.hub.direct".
	Name() string
	// Kind is the question family the probe belongs to.
	Kind() ProbeKind
	// Run performs the measurement. It must never modify the system.
	Run(ctx context.Context) Result
}

// rank orders the three verdicts by how much they may claim: fail (2) above
// indeterminate (1) above pass (0). Anything a measurement cannot establish
// therefore outranks a pass, which is what makes it impossible for an
// unresolved observation to be promoted into one.
func rank(v Verdict) int {
	switch v {
	case Fail:
		return 2
	case Indeterminate:
		return 1
	default:
		return 0
	}
}

// Aggregate reduces a probe's observations to the single verdict and reason
// its Result reports. The verdict is the worst observation's verdict in the
// documented order fail > indeterminate > pass, and the reason is that same
// observation's reason — so when a probe reports a failure beside a sibling it
// could not measure, the failure is explained by the failing observation and
// never by the sibling's not-measured reason.
//
// An empty observation list is not a pass. A probe that reported nothing has
// measured nothing, so it is reported as indeterminate with the internal-error
// reason rather than as a success the run cannot support.
func Aggregate(observations []Observation) (Verdict, ReasonCode) {
	if len(observations) == 0 {
		return Indeterminate, ReasonInternalError
	}
	worst := observations[0]
	for _, o := range observations[1:] {
		if rank(o.Verdict) > rank(worst.Verdict) {
			worst = o
		}
	}
	return worst.Verdict, worst.Reason
}
