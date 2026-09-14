package probe

// This file is the runner of design §7: the bounded-concurrency scheduler that
// executes the probe suite, streams each result as it settles, and owns the
// clock, the per-probe bound and the run budget the suite runs under.
//
// The runner owns those through Options set at construction, never through
// package state. A package-level default would be shared silently between the
// doctor's live run and a test's millisecond run, and the second run would
// inherit bounds it never asked for; here the effective bounds are readable back
// through Runner.Options instead of being guessed at.
//
// The rule the whole file is built around: the runner never trusts a probe to
// return. Every probe is started under its own context and awaited only until
// that bound expires. A probe that ignores the bound is abandoned and reported by
// the runner, because the run still has to finish and report the probes that did
// answer (PRD §4.4, R-HR-NF-02).

import (
	"context"
	"sync/atomic"
	"time"
)

const (
	// DefaultConcurrency is how many measurements the runner admits at once. D9
	// chooses four: ten simultaneous outbound connections from a machine that may
	// be under an endpoint agent's inspection is a self-inflicted burst, while
	// four keep a full run well inside its budget.
	DefaultConcurrency = 4

	// DefaultProbeTimeout bounds one probe. It is longer than a probe's own
	// budget, which is the ordering the timeout paths need: a probe's own, shorter
	// budget expires first and reports the measurement itself, and only a probe
	// that ignores even that is reported at the runner layer.
	//
	// Ten seconds is a judgment call, not a product requirement. With a ceiling of
	// four, ten probes drain in three waves, so the worst case is thirty seconds
	// inside the sixty-second run budget — and the probes that answer quickly still
	// finish in the first wave.
	DefaultProbeTimeout = 10 * time.Second

	// DefaultRunBudget is the specification's ceiling on a whole run: sixty
	// seconds (R-HR-NF-09, design §3.3's run_budget_ms).
	DefaultRunBudget = 60 * time.Second
)

// Options are one runner's injected bounds.
//
// Every field has a documented default applied by NewRunner, and there is
// deliberately no way to express "unbounded": a non-positive duration or a
// concurrency below one resolves to the default, so a run cannot be configured
// out of its own guarantees.
type Options struct {
	// Concurrency is the ceiling on measurements in flight. Default 4.
	Concurrency int
	// ProbeTimeout bounds one measurement. Default 10s.
	ProbeTimeout time.Duration
	// RunBudget bounds the whole run. Default 60s.
	RunBudget time.Duration
	// Clock is the run's single clock, used for every elapsed value the runner
	// reports. Default: the machine's own clock.
	Clock Clock
}

// withDefaults resolves every unset field to its documented default. It is the
// only place a default is applied, so Runner.Options can report exactly the
// bounds the run used rather than what the caller happened to leave unset.
func (o Options) withDefaults() Options {
	if o.Concurrency < 1 {
		o.Concurrency = DefaultConcurrency
	}
	if o.ProbeTimeout <= 0 {
		o.ProbeTimeout = DefaultProbeTimeout
	}
	if o.RunBudget <= 0 {
		o.RunBudget = DefaultRunBudget
	}
	if o.Clock == nil {
		o.Clock = wallClock{}
	}
	return o
}

// Runner executes probes under one set of bounds.
type Runner struct {
	options Options
}

// NewRunner returns a runner with options' bounds, defaults applied.
func NewRunner(options Options) *Runner {
	return &Runner{options: options.withDefaults()}
}

// Options returns the effective bounds this runner uses.
func (r *Runner) Options() Options { return r.options }

// outcome is one settled probe result on its way from the probe's watcher to the
// scheduling loop. The index is what lets results be delivered in completion
// order while the returned slice stays in probe order.
type outcome struct {
	index  int
	result Result
}

// Run executes probes and returns exactly one Result per probe, in the order the
// probes were given, whether the suite completed, ran out of budget or was
// cancelled. A probe that broke never stops another probe from being reported.
//
// Results are streamed: emit, when it is not nil, is called once per probe as
// soon as that probe's result settles, in completion order, so a caller can
// report a measurement before the slowest probe has finished instead of waiting
// for the whole suite. emit runs on Run's own goroutine and must not block the
// run; a caller that needs to hand results on should do so over a buffered
// channel.
//
// The concurrency ceiling bounds the probes the runner has started and still
// awaits. It deliberately does not bound the goroutines in the process: a probe
// that ignores its bound is abandoned rather than waited for, so it can outlive
// the slot it held. That is the cost of never trusting a probe to return, and it
// is the reason the ceiling is stated as what the runner admits rather than as
// what the machine is running.
func (r *Runner) Run(ctx context.Context, probes []Probe, emit func(Result)) []Result {
	if len(probes) == 0 {
		return nil
	}

	opts := r.options
	results := make([]Result, len(probes))
	settled := make([]bool, len(probes))
	remaining := len(probes)
	done := make(chan outcome, len(probes))

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	record := func(index int, result Result) {
		if settled[index] {
			return
		}
		settled[index] = true
		results[index] = result
		remaining--
		if emit != nil {
			emit(result)
		}
	}

	// The run budget is enforced from the run's start, not from the first probe's,
	// so a suite that spends time starting up still finishes inside its budget.
	// budgetC and ctxDone are nil-ed once the run has ended, so a spent timer or an
	// already-cancelled context never spins this loop.
	var budgetExceeded atomic.Bool
	budgetTimer := time.NewTimer(opts.RunBudget)
	defer budgetTimer.Stop()
	budgetC := budgetTimer.C
	ctxDone := ctx.Done()

	// runEnded records that the run stopped admitting probes, and endedCause is the
	// one abandonment every probe that never got its chance is reported with — the
	// same value the in-flight probes are classified through, so the two paths
	// cannot drift apart.
	runEnded := false
	var endedCause func(Probe) abandonment

	live := 0
	next := 0
	for remaining > 0 {
		if !runEnded {
			for next < len(probes) && live < opts.Concurrency {
				if runCtx.Err() != nil {
					// The run ended between iterations. The select below records the
					// cause; nothing may be started against a dead context.
					break
				}
				probe := probes[next]
				startedAt := opts.Clock.Now()
				live++
				go awaitProbe(runCtx, probe, opts, &budgetExceeded, startedAt, done, next)
				next++
			}
		}

		// A run that ended never starts another probe: the probes it never attempted
		// have no measurement to wait for, so the runner reports them now rather
		// than letting the unstarted remainder hold the run open.
		if runEnded {
			for ; next < len(probes); next++ {
				record(next, abandon(probes[next], endedCause(probes[next]), opts.Clock.Now(), opts.Clock))
			}
		}

		select {
		case reported := <-done:
			live--
			record(reported.index, reported.result)
		case <-budgetC:
			budgetC = nil
			ctxDone = nil
			runEnded = true
			endedCause = runBudgetExceeded
			budgetExceeded.Store(true)
			cancelRun()
		case <-ctxDone:
			ctxDone = nil
			budgetC = nil
			budgetTimer.Stop()
			runEnded = true
			endedCause = runCancelled
			cancelRun()
		}
	}

	return results
}

// awaitProbe starts one probe under its own bound and reports the settled result
// to the scheduling loop.
//
// The probe is called from its own goroutine because Go cannot kill a goroutine:
// the only way to stop waiting for a probe that ignores its bound is to stop
// waiting for it. The watcher therefore reports the runner's own fact once the
// bound expires, and the probe's goroutine is left to finish or not, holding no
// channel the run still depends on: its result channel is buffered, so even a
// late return cannot block.
//
// A probe that returns after the run itself has ended gets the runner's reason
// too. Its answer arrived after the run stopped collecting answers, and accepting
// it would let a probe that woke on cancellation decide the run's result.
func awaitProbe(runCtx context.Context, p Probe, opts Options, budgetExceeded *atomic.Bool, startedAt time.Time, done chan<- outcome, index int) {
	probeCtx, cancel := context.WithTimeout(runCtx, opts.ProbeTimeout)
	defer cancel()

	returned := make(chan Result, 1)
	go func() { returned <- p.Run(probeCtx) }()

	select {
	case result := <-returned:
		if runCtx.Err() != nil {
			done <- outcome{index: index, result: abandon(p, runEndedCause(runCtx, budgetExceeded, p, opts.ProbeTimeout), startedAt, opts.Clock)}
			return
		}
		done <- outcome{index: index, result: fillIdentity(p, result, startedAt, opts.Clock)}
	case <-probeCtx.Done():
		done <- outcome{index: index, result: abandon(p, runEndedCause(runCtx, budgetExceeded, p, opts.ProbeTimeout), startedAt, opts.Clock)}
	}
}

// runEndedCause names why the probe's context ended. The run's own budget comes
// first because it also cancels the run context; then the caller's cancellation;
// and when neither happened, the only remaining explanation is the probe's own
// bound. Exactly one implementation decides this, so the probe bound, the run
// budget and cancellation can never be reported as each other (RG-8).
func runEndedCause(runCtx context.Context, budgetExceeded *atomic.Bool, p Probe, timeout time.Duration) abandonment {
	switch {
	case budgetExceeded.Load():
		return runBudgetExceeded(p)
	case runCtx.Err() != nil:
		return runCancelled(p)
	default:
		return probeBoundExceeded(p, timeout)
	}
}

// abandonment is one reason the runner stopped awaiting a probe. Its two fields
// are exactly what the classification table needs, so the per-probe bound and the
// run-level reasons below cannot drift into different observables or different
// wording: they are built here and settled by one function.
type abandonment struct {
	observable Observable
	wording    string
}

// probeBoundExceeded is the per-probe bound's abandonment: the probe was given
// its bound and ignored it.
func probeBoundExceeded(p Probe, timeout time.Duration) abandonment {
	return abandonment{
		observable: ObsProbeIgnoredBudget,
		wording:    p.Name() + ": the probe did not return inside its " + timeout.String() + " bound and was abandoned",
	}
}

// runBudgetExceeded is the run budget's abandonment: the run's ceiling expired
// before this probe's measurement could be collected. The reason is the runner's
// own fact and is never chosen by a probe.
func runBudgetExceeded(p Probe) abandonment {
	return abandonment{
		observable: ObsRunBudgetExhausted,
		wording:    p.Name() + ": the run budget expired before this probe's measurement was collected and the run moved on",
	}
}

// runCancelled is the caller's cancellation's abandonment: the run was cancelled
// while this probe was in flight, so the attempt produced no answer the run is
// willing to report. A probe affected by cancellation is never a pass, however
// it answered on its way out.
func runCancelled(p Probe) abandonment {
	return abandonment{
		observable: ObsRunCancelled,
		wording:    p.Name() + ": the run was cancelled before this probe's measurement was collected",
	}
}

// abandon builds the result the runner reports for a probe it stopped awaiting,
// whatever stopped it: the probe's own bound, the run's budget or the caller's
// cancellation. No measurement exists for that probe, so the result is the
// runner's own fact: unresolved and indeterminate, never a pass, with the reason
// named in the detail so a reader can tell why the run gave up on it.
//
// The observation is classified by the same table every probe uses — the runner
// gets no private vocabulary — and its target is empty because a Probe declares no
// target: the address a probe measured lives behind its own constructor. The
// probe's name is in the result and the label, so the gap is named rather than
// silently attributed to a target the runner guessed.
func abandon(p Probe, why abandonment, startedAt time.Time, clock Clock) Result {
	obs := Observe("runner", "", "", RawObservation{Kind: why.observable, Wording: why.wording})
	return Result{
		Probe:        p.Name(),
		Kind:         p.Kind(),
		Verdict:      obs.Verdict,
		Reason:       obs.Reason,
		Detail:       obs.Detail,
		Elapsed:      clock.Now().Sub(startedAt),
		Observations: []Observation{obs},
	}
}

// fillIdentity fills in the identity and timing a probe left unset. The runner
// never overwrites a value the probe set — a probe may know its own timings better
// than the run's clock does — but an unnamed or untimed result would break the
// payload's key set, so the run's own facts fill the gaps.
func fillIdentity(p Probe, result Result, startedAt time.Time, clock Clock) Result {
	if result.Probe == "" {
		result.Probe = p.Name()
	}
	if result.Kind == "" {
		result.Kind = p.Kind()
	}
	if result.Elapsed == 0 {
		result.Elapsed = clock.Now().Sub(startedAt)
	}
	return result
}

// wallClock is the clock a runner falls back to when its Options carry none.
// Production always injects the run's own seam clock — design §3.3 keeps every
// timestamp in one clock's hands — and this fallback exists only so a zero Options
// value is usable rather than leaving an elapsed value unset.
type wallClock struct{}

// Now returns the machine's current time.
func (wallClock) Now() time.Time { return time.Now() }
