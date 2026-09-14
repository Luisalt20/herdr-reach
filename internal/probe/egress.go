package probe

// This file is the egress probe group of design §7. This slice lands
// `egress.hub.direct` and nothing else egress: the public-SSH probes and the two
// Cloudflare edge questions follow in their own slices, which extend this file
// rather than start another one.
//
// `egress.hub.direct` answers one question: does the hub address this run was
// given accept a TCP connection on its declared port? Three properties shape it,
// and they matter more than its length.
//
// First, its target is run input, never a constant (design D3). With `--hub` the
// target is exactly the supplied `host[:port]`, with the documented default port
// applied when the address omits one, resolved through the one declaration in
// targets.go — so "the dialed set equals the declared set" (R-HR-NF-10) stays a
// comparison of two values instead of a claim about probe code.
//
// Second, no hub supplied is not a blocked hub (R-HR-02, RG-13). The observation
// is not measured, its target is empty because nothing was attempted, and its
// reason names the missing input. The output never says "blocked": a block is a
// claim about a measurement, and there is no measurement here.
//
// Third, the probe's own dial budget is deliberately shorter than the runner's
// per-probe bound (design §5.1 obligation 3). A declared port that never answers
// expires the probe's own budget, and that expiry *is* the measurement for the
// question "is this port reachable?" — a definite negative. A probe that ignores
// even the longer runner bound is abandoned by the runner and reported as
// `probe_timeout`. Collapsing those two is the RG-8 wrong-classification risk, and
// the ordering of the two constants is asserted by a test rather than left to
// whoever edits one of them next.
//
// The probe dials and closes. It does not read a banner: whether the far end
// speaks SSH is the public-host probes' question, and answering it here would put
// a second question inside a probe the registry declares as one.

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"time"
)

const (
	// DefaultDialBudget is the dial budget every reachability probe uses. It is a
	// judgment call with a stated reason: it must expire before the runner's
	// per-probe bound, so the expiry is observed by the probe and reported as the
	// measurement rather than as the runner abandoning a hung probe. Four seconds
	// is half a typical TCP connect timeout and well inside DefaultProbeTimeout.
	//
	// It is exported because it is a property of the run's timing contract, not an
	// implementation detail of one probe: a later reachability probe must use the
	// same budget, and a test asserts both the budget and its ordering against the
	// runner's bound.
	DefaultDialBudget = 4 * time.Second
	// hubFactLabel names the hub fact when no address was resolved: the fact under
	// test is the run's input, so the label says so instead of naming a port that
	// was never dialed.
	hubTargetLabel = "hub target"
)

// egressHub is the `egress.hub.direct` probe. It carries the run's seams and the
// run's declared target input, and it resolves its target at construction time so
// that Run performs exactly one measurement.
type egressHub struct {
	seams   Seams
	targets TargetInput
}

// newEgressHub builds the probe from the run's seams and its declared target
// input. The hub address is run input rather than a constant, which is why the
// registry's factory takes the input as well as the seams: a probe whose target is
// supplied by the run must receive it, and a package-level copy of it would let
// two runs share one machine's input (design §6.1).
func newEgressHub(seams Seams, targets TargetInput) Probe {
	return &egressHub{seams: seams, targets: targets}
}

// Name is the probe's stable identifier, declared once in registry.go.
func (p *egressHub) Name() string { return probeNameEgressHub }

// Kind is the question family the probe belongs to.
func (p *egressHub) Kind() ProbeKind { return ProbeEgress }

// Run measures the declared hub address once and reports the answer as one
// observation.
//
// The result's target is the observation's target, so a run with no hub address
// reports an empty target in both places: a consumer must never have to guess
// whether "no target" means "not supplied" or "supplied and empty", and an empty
// hub address is a usage error one layer above (design D3).
//
// The context travels into the dial, so cancellation and the runner's own bound
// both reach the socket. The probe's budget narrows it further rather than
// replacing it: whichever ends first, the dial stops.
func (p *egressHub) Run(ctx context.Context) Result {
	started := p.now()
	observation := p.observe(ctx)
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

// now reads the run's clock, or the zero time when no clock was injected, so only
// an injected clock can move an elapsed value (design §3.3).
func (p *egressHub) now() time.Time { return runClockNow(p.seams) }

// observe performs the one measurement this probe declares.
//
// The order is the order of the questions: which target was declared, then
// whether the run was given the capability to dial it, then the dial itself. Each
// step that cannot proceed reports a not-measured fact naming what was missing,
// and none of them claims anything about the hub.
func (p *egressHub) observe(ctx context.Context) Observation {
	target, unresolved := p.declaredTarget()
	if unresolved.Kind != "" {
		return Observe(hubTargetLabel, "", PurposePortReachability, unresolved)
	}

	address := target.Address()
	label := fmt.Sprintf("tcp %d", target.Port)
	if target.Label != "" {
		label += " " + target.Label
	}

	if p.seams.Dialer == nil {
		return Observe(label, address, PurposePortReachability, RawObservation{
			Kind:    ObsCapabilityExcluded,
			Wording: fmt.Sprintf("dial tcp %s: no dialer is injected for this run, so the hub measurement could not be attempted", address),
		})
	}

	dialCtx, cancel := context.WithTimeout(ctx, DefaultDialBudget)
	defer cancel()
	conn, err := p.seams.Dialer.DialContext(dialCtx, "tcp", address)
	if err != nil {
		return Observe(label, address, PurposePortReachability, dialFact(address, err))
	}
	if conn != nil {
		// The connection is the measurement; it is closed immediately because
		// nothing is read or written on it. Closing is the probe's own duty: a
		// probe that leaves a socket behind would leak one per run.
		_ = conn.Close()
	}
	return Observe(label, address, PurposePortReachability, RawObservation{
		Kind:    ObsTCPEstablished,
		Wording: fmt.Sprintf("dial tcp %s: the connection was established and closed without reading or writing, which measures TCP reachability of the declared address and nothing else", address),
	})
}

// declaredTarget resolves the effective declared set and returns the hub entry, or
// the raw fact that explains why there is none.
//
// The resolution goes through EffectiveTargets rather than through a private
// lookup: the declared set, the documented default port and the override rules live
// in exactly one place (targets.go), and a probe that re-derived them could
// disagree with the declaration the payload echoes (R-HR-NF-10).
//
// A run with no hub address supplied is not an error: it is the documented
// not-measured case. A hub address that cannot be resolved into a target is a fact
// this probe did not expect — the flag layer refuses unusable input with a usage
// error — so it is reported as an unresolved internal failure rather than guessed
// at.
func (p *egressHub) declaredTarget() (EffectiveTarget, RawObservation) {
	effective, err := EffectiveTargets(p.targets)
	if err != nil {
		return EffectiveTarget{}, RawObservation{
			Kind:    ObsInternalFailure,
			Wording: fmt.Sprintf("the declared hub target could not be resolved from the run's input, so no measurement was attempted: %v", err),
		}
	}
	for _, target := range effective {
		if target.Probe == probeNameEgressHub {
			return target, RawObservation{}
		}
	}
	return EffectiveTarget{}, RawObservation{
		Kind:    ObsHubInputMissing,
		Wording: "no hub address was supplied for this run, so the hub measurement was not made: no target was attempted, and nothing is claimed about the hub's reachability; supply --hub host[:port] to measure it",
	}
}

// dialFact turns one failed dial into the raw fact the classification table sees.
//
// The mapping is by error identity, never by message text (R-HR-07): a denial from
// the seam, a connection the far end refused, a connection reset after it was
// established, and the probe's own budget expiring on a dial are four different
// facts with four different codes. Anything else is an unclassified dial failure:
// it is reported as an internal failure with the error verbatim rather than
// borrowed into a negative the error does not establish, because the closed
// reason-code set has no "dial failed for another reason" code and inventing one is
// a contract change (design §3.5).
//
// The timeout branch is written as a property rather than as a comparison against
// context.DeadlineExceeded alone: a socket reports its deadline expiry through its
// own error type, and the probe's claim — "this expiry is mine, because my budget
// is the shorter one on this dial context" — holds for every implementation of
// that property.
func dialFact(address string, err error) RawObservation {
	wording := fmt.Sprintf("dial tcp %s: %v", address, err)
	switch {
	case errors.Is(err, ErrSeamDenied):
		return RawObservation{
			Kind:    ObsCommandDenied,
			Wording: wording + " (the dial seam denied the attempt, so the hub was not measured)",
		}
	case dialTimedOut(err):
		return RawObservation{
			Kind: ObsProbeBudgetExpired,
			Wording: fmt.Sprintf("dial tcp %s: %v after the probe's own %s dial budget, which is shorter than the runner's %s per-probe bound: the declared port did not answer inside the probe's own budget, which for the question \"is this port reachable?\" is the measurement itself",
				address, err, DefaultDialBudget, DefaultProbeTimeout),
		}
	case errors.Is(err, syscall.ECONNREFUSED):
		return RawObservation{Kind: ObsTCPRefused, Wording: wording}
	case errors.Is(err, syscall.ECONNRESET):
		return RawObservation{Kind: ObsTCPReset, Wording: wording}
	default:
		return RawObservation{
			Kind:    ObsInternalFailure,
			Wording: wording + " (the dial failed for a reason no row of the classification table names, so nothing is claimed about the hub)",
		}
	}
}

// dialTimedOut reports whether a dial error is a deadline expiry. A socket's
// timeout error implements Timeout() bool, and a context deadline error does too;
// errors.As recognises either without this package naming a concrete error type it
// would then have to keep in step with the standard library.
func dialTimedOut(err error) bool {
	var timeout interface{ Timeout() bool }
	if errors.As(err, &timeout) {
		return timeout.Timeout()
	}
	return errors.Is(err, context.DeadlineExceeded)
}
