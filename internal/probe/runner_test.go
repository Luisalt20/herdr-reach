package probe_test

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// runnerFixture is a scripted probe. Fixture probes live in this file on purpose
// (tasks.md PR 4): every case here needs a probe whose timing, cancellation
// honouring and classification the test controls exactly, which no measurement of
// a real network can give it, and the runner then reviews without the ten real
// probes that land in PR 5–PR 9.
type runnerFixture struct {
	name string
	kind probe.ProbeKind
	run  func(ctx context.Context) probe.Result

	// starts counts how many times the runner actually called this probe. A
	// runner-produced result is only meaningful if the probe behind it was really
	// attempted, so the cases assert this instead of assuming it.
	starts atomic.Int64
}

// Name is the fixture's stable probe name.
func (f *runnerFixture) Name() string { return f.name }

// Kind is the fixture's probe kind.
func (f *runnerFixture) Kind() probe.ProbeKind { return f.kind }

// Run performs the scripted measurement and counts the attempt.
func (f *runnerFixture) Run(ctx context.Context) probe.Result {
	f.starts.Add(1)
	return f.run(ctx)
}

// runnerObservation builds one observation through the classification table, so a
// fixture reports exactly what a real probe reports for the same raw fact rather
// than a hand-written triple the table could not produce.
func runnerObservation(label, target string, purpose probe.Purpose, kind probe.Observable, wording string) probe.Observation {
	return probe.Observe(label, target, purpose, probe.RawObservation{Kind: kind, Wording: wording})
}

// runnerResult reduces observations the way a probe must, so a fixture's Result
// satisfies the same invariants probe_test.go asserts over the vocabulary.
func runnerResult(obs ...probe.Observation) probe.Result {
	verdict, reason := probe.Aggregate(obs)
	lines := make([]string, 0, len(obs))
	for _, o := range obs {
		lines = append(lines, o.Label+": "+o.Detail)
	}
	return probe.Result{Verdict: verdict, Reason: reason, Observations: obs, Detail: strings.Join(lines, "\n")}
}

// reachableFixture is a well-behaved fixture whose declared reachability question
// is answered positively and immediately.
func reachableFixture(name string) *runnerFixture {
	return &runnerFixture{
		name: name,
		kind: probe.ProbeEgress,
		run: func(context.Context) probe.Result {
			return runnerResult(runnerObservation("tcp 22", "203.0.113.10:22",
				probe.PurposePortReachability, probe.ObsTCPEstablished, "connection established"))
		},
	}
}

// runnerGate holds gated fixtures inside their measurement until the test releases
// them, and records the highest number that were inside Run at one moment.
//
// That peak is the concurrency ceiling's evidence: it is counted by the fixtures
// themselves, at the measurement boundary, not inferred from the runner's own
// accounting. A runner that admitted a probe it was not supposed to start would
// be seen here.
type runnerGate struct {
	entered  chan struct{}
	release  chan struct{}
	inFlight atomic.Int64
	peak     atomic.Int64
}

// newRunnerGate returns a gate sized for count fixtures, so no fixture blocks on
// reporting its entry after the test stops reading entries.
func newRunnerGate(count int) *runnerGate {
	return &runnerGate{entered: make(chan struct{}, count), release: make(chan struct{})}
}

// probe returns one gated fixture. Every gated fixture blocks inside Run until the
// test closes the release channel, so the number of probes the runner admitted is
// observable while the run is still in flight.
func (g *runnerGate) probe(name string) *runnerFixture {
	return &runnerFixture{
		name: name,
		kind: probe.ProbeEgress,
		run: func(context.Context) probe.Result {
			inside := g.inFlight.Add(1)
			for {
				peak := g.peak.Load()
				if inside <= peak || g.peak.CompareAndSwap(peak, inside) {
					break
				}
			}
			g.entered <- struct{}{}
			<-g.release
			g.inFlight.Add(-1)
			return runnerResult(runnerObservation("tcp 22", "203.0.113.10:22",
				probe.PurposePortReachability, probe.ObsTCPEstablished, "connection established"))
		},
	}
}

// TestRunnerHonoursConcurrencyCeiling is the bounded-concurrency case of design D9
// and §7: the runner never has more than Options.Concurrency measurements in
// flight, the default is four, and a value injected for a test is honoured rather
// than ignored in favour of the default.
func TestRunnerHonoursConcurrencyCeiling(t *testing.T) {
	const probeCount = 10

	cases := []struct {
		name          string
		options       probe.Options
		wantEffective int
	}{
		{"the default ceiling is four", probe.Options{}, 4},
		{"an elevated test-only value is honoured", probe.Options{Concurrency: 6}, 6},
		{"a serial ceiling of one is honoured", probe.Options{Concurrency: 1}, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gate := newRunnerGate(probeCount)
			probes := make([]probe.Probe, 0, probeCount)
			for i := 0; i < probeCount; i++ {
				probes = append(probes, gate.probe(fmt.Sprintf("fixture.%02d", i)))
			}

			runner := probe.NewRunner(tc.options)
			if got := runner.Options().Concurrency; got != tc.wantEffective {
				t.Fatalf("effective concurrency = %d, want %d", got, tc.wantEffective)
			}

			results := make(chan []probe.Result, 1)
			go func() { results <- runner.Run(context.Background(), probes, nil) }()

			for entered := 0; entered < tc.wantEffective; entered++ {
				select {
				case <-gate.entered:
				case <-time.After(2 * time.Second):
					t.Fatalf("only %d fixtures entered their measurement, want %d", entered, tc.wantEffective)
				}
			}

			// While the ceiling is reached nothing else may enter. The window is
			// generous on purpose: an unenforced ceiling exceeds it immediately.
			select {
			case <-gate.entered:
				t.Fatalf("a %dth fixture entered while the ceiling was %d", tc.wantEffective+1, tc.wantEffective)
			case <-time.After(50 * time.Millisecond):
			}
			if peak := gate.peak.Load(); peak > int64(tc.wantEffective) {
				t.Fatalf("peak fixtures in flight = %d, want at most %d", peak, tc.wantEffective)
			}

			close(gate.release)
			select {
			case run := <-results:
				if len(run) != probeCount {
					t.Fatalf("run returned %d results, want %d", len(run), probeCount)
				}
				if peak := gate.peak.Load(); peak != int64(tc.wantEffective) {
					t.Fatalf("peak fixtures in flight = %d, want exactly %d", peak, tc.wantEffective)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("the gated run did not finish after its fixtures were released")
			}
		})
	}
}

// TestHangingProbeIsAbandonedByTheRunner is the hanging-probe case of R-HR-NF-02
// and PRD §4.4: the runner does not trust a probe to return. A probe that ignores
// both its context and its bound is abandoned, reported by the runner as an
// unresolved timeout, and it neither holds the run open nor keeps any other probe
// from being reported.
func TestHangingProbeIsAbandonedByTheRunner(t *testing.T) {
	// Never closed: the fixture below blocks here for the rest of the test binary's
	// life, which is exactly the misbehaviour the runner has to survive.
	forever := make(chan struct{})

	hanging := &runnerFixture{
		name: "fixture.hanging",
		kind: probe.ProbeEgress,
		run: func(context.Context) probe.Result {
			<-forever
			return probe.Result{}
		},
	}
	fast := reachableFixture("fixture.fast")
	refused := &runnerFixture{
		name: "fixture.refused",
		kind: probe.ProbeEgress,
		run: func(context.Context) probe.Result {
			return runnerResult(runnerObservation("tcp 22", "203.0.113.10:22",
				probe.PurposePortReachability, probe.ObsTCPRefused, "connect: connection refused"))
		},
	}

	const probeTimeout = 25 * time.Millisecond
	const runBudget = 2 * time.Second
	runner := probe.NewRunner(probe.Options{Concurrency: 3, ProbeTimeout: probeTimeout, RunBudget: runBudget})

	started := time.Now()
	run := runner.Run(context.Background(), []probe.Probe{hanging, fast, refused}, nil)
	elapsed := time.Since(started)

	if elapsed >= runBudget {
		t.Fatalf("the run took %s, want it inside its injected budget of %s", elapsed, runBudget)
	}
	if elapsed < probeTimeout {
		t.Fatalf("the run took %s, want it to have waited for the %s probe bound", elapsed, probeTimeout)
	}
	if len(run) != 3 {
		t.Fatalf("run returned %d results, want 3: every other probe must still be reported", len(run))
	}
	if hanging.starts.Load() != 1 {
		t.Fatalf("the hanging fixture was started %d times, want the runner to have attempted it once", hanging.starts.Load())
	}

	got := run[0]
	if got.Probe != "fixture.hanging" {
		t.Fatalf("result 0 names %q, want the hanging fixture", got.Probe)
	}
	if got.Verdict != probe.Indeterminate {
		t.Fatalf("hanging verdict = %q, want %q: an abandoned probe measured nothing", got.Verdict, probe.Indeterminate)
	}
	if got.Reason != probe.ReasonProbeTimeout {
		t.Fatalf("hanging reason = %q, want %q: the runner classifies the probe it abandoned", got.Reason, probe.ReasonProbeTimeout)
	}
	if len(got.Observations) != 1 || got.Observations[0].Resolution != probe.Unresolved {
		t.Fatalf("hanging observations = %+v, want one unresolved observation produced by the runner", got.Observations)
	}
	if got.Verdict == probe.Pass {
		t.Fatal("an abandoned probe was reported as a pass")
	}
	for i, want := range []string{"fixture.fast", "fixture.refused"} {
		if run[i+1].Probe != want {
			t.Fatalf("result %d names %q, want %q", i+1, run[i+1].Probe, want)
		}
		if run[i+1].Verdict == "" || run[i+1].Reason == "" {
			t.Fatalf("result %q carries no verdict or reason: %+v", want, run[i+1])
		}
	}
	if run[1].Verdict != probe.Pass || run[2].Verdict != probe.Fail {
		t.Fatalf("other verdicts = %q, %q; want the fast pass and the refused fail to survive the hanging probe", run[1].Verdict, run[2].Verdict)
	}
}

// TestRunnerStreamsResultsBeforeTheSlowestProbeFinishes is the streaming case of
// design §7: the runner delivers each result as it settles instead of buffering
// the suite. The proof is negative as well as positive — the fast fixture's result
// arrives while the slow fixture is provably still measuring, and nothing else is
// delivered until that fixture is released.
func TestRunnerStreamsResultsBeforeTheSlowestProbeFinishes(t *testing.T) {
	release := make(chan struct{})
	slow := &runnerFixture{
		name: "fixture.slow",
		kind: probe.ProbeEgress,
		run: func(ctx context.Context) probe.Result {
			select {
			case <-release:
			case <-ctx.Done():
			}
			return runnerResult(runnerObservation("tcp 22", "203.0.113.10:22",
				probe.PurposePortReachability, probe.ObsTCPEstablished, "connection established"))
		},
	}
	fast := reachableFixture("fixture.fast")

	streamed := make(chan probe.Result, 2)
	runner := probe.NewRunner(probe.Options{Concurrency: 2, ProbeTimeout: 2 * time.Second, RunBudget: 5 * time.Second})
	finished := make(chan []probe.Result, 1)
	go func() {
		finished <- runner.Run(context.Background(), []probe.Probe{slow, fast}, func(result probe.Result) {
			streamed <- result
		})
	}()

	var first probe.Result
	select {
	case first = <-streamed:
	case <-time.After(2 * time.Second):
		t.Fatal("no result was streamed before the slow fixture finished")
	}
	if first.Probe != "fixture.fast" {
		t.Fatalf("first streamed result = %q, want the fast fixture: the runner buffered the suite", first.Probe)
	}

	select {
	case second := <-streamed:
		t.Fatalf("streamed %q while the slow fixture was still measuring: the runner is not streaming", second.Probe)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case run := <-finished:
		if len(run) != 2 {
			t.Fatalf("run returned %d results, want 2", len(run))
		}
		if run[0].Probe != "fixture.slow" || run[1].Probe != "fixture.fast" {
			t.Fatalf("results = %q, %q; want probe order regardless of completion order", run[0].Probe, run[1].Probe)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the run did not finish after the slow fixture was released")
	}

	select {
	case last := <-streamed:
		if last.Probe != "fixture.slow" {
			t.Fatalf("second streamed result = %q, want the slow fixture", last.Probe)
		}
	case <-time.After(time.Second):
		t.Fatal("the slow fixture's result was never streamed")
	}
}

// TestRunnerReturnsEveryProbeOnceInProbeOrder pins the delivery contract that the
// streaming cases rely on: results arrive in completion order, but the returned
// suite is one result per probe in probe order. Fixtures answer in the reverse of
// probe order, so completion order and probe order genuinely differ and an
// implementation that returned completion order would be caught.
func TestRunnerReturnsEveryProbeOnceInProbeOrder(t *testing.T) {
	const count = 6
	probes := make([]probe.Probe, 0, count)
	for i := 0; i < count; i++ {
		delay := time.Duration(count-i) * 3 * time.Millisecond
		probes = append(probes, &runnerFixture{
			name: fmt.Sprintf("fixture.%02d", i),
			kind: probe.ProbeEgress,
			run: func(context.Context) probe.Result {
				// A well-behaved probe inside a generous bound: the sleep only
				// shuffles completion order, which is the point of the case.
				time.Sleep(delay)
				return runnerResult(runnerObservation("tcp 22", "203.0.113.10:22",
					probe.PurposePortReachability, probe.ObsTCPEstablished, "connection established"))
			},
		})
	}

	runner := probe.NewRunner(probe.Options{Concurrency: 3, ProbeTimeout: time.Second, RunBudget: 5 * time.Second})
	run := runner.Run(context.Background(), probes, nil)
	if len(run) != count {
		t.Fatalf("run returned %d results, want %d", len(run), count)
	}
	seen := make(map[string]int, count)
	for i, result := range run {
		if result.Probe != probes[i].Name() {
			t.Fatalf("result %d names %q, want %q (probe order)", i, result.Probe, probes[i].Name())
		}
		seen[result.Probe]++
	}
	for _, p := range probes {
		if seen[p.Name()] != 1 {
			t.Fatalf("probe %q has %d results, want exactly 1", p.Name(), seen[p.Name()])
		}
	}
}

// TestRunBudgetBoundsASlowSuite is the bounded-run case of R-HR-NF-09: an
// injected run budget bounds the whole suite even when every probe is slower than
// that budget. Every probe still gets a bounded result — the runner reports the
// ones it could not collect — and the run ends inside the budget instead of
// waiting out four waves of slow probes.
func TestRunBudgetBoundsASlowSuite(t *testing.T) {
	const probeCount = 8
	const probeDelay = 80 * time.Millisecond
	const runBudget = 40 * time.Millisecond

	slowFixture := func(name string) *runnerFixture {
		return &runnerFixture{
			name: name,
			kind: probe.ProbeEgress,
			run: func(ctx context.Context) probe.Result {
				select {
				case <-time.After(probeDelay):
				case <-ctx.Done():
				}
				// The probe's own honest answer for its declared question: the
				// port never answered inside the budget it was given. The runner
				// overrides this when the run itself ended first.
				return runnerResult(runnerObservation("tcp 7844", "203.0.113.10:7844",
					probe.PurposePortReachability, probe.ObsProbeBudgetExpired, "dial tcp 203.0.113.10:7844: i/o timeout"))
			},
		}
	}

	probes := make([]probe.Probe, 0, probeCount)
	for i := 0; i < probeCount; i++ {
		probes = append(probes, slowFixture(fmt.Sprintf("fixture.%02d", i)))
	}

	runner := probe.NewRunner(probe.Options{Concurrency: 2, ProbeTimeout: 300 * time.Millisecond, RunBudget: runBudget})
	if got := runner.Options().RunBudget; got != runBudget {
		t.Fatalf("effective run budget = %s, want the injected %s", got, runBudget)
	}

	started := time.Now()
	run := runner.Run(context.Background(), probes, nil)
	elapsed := time.Since(started)

	if elapsed >= probeDelay {
		t.Fatalf("the run took %s, want it bounded by its %s budget instead of a full wave of %s probes", elapsed, runBudget, probeDelay)
	}
	if len(run) != probeCount {
		t.Fatalf("run returned %d results, want %d: every probe must have a bounded result", len(run), probeCount)
	}
	startedProbes := 0
	for i, result := range run {
		if n := probes[i].(*runnerFixture).starts.Load(); n > 0 {
			startedProbes += int(n)
		}
		if result.Probe != probes[i].Name() {
			t.Fatalf("result %d names %q, want %q", i, result.Probe, probes[i].Name())
		}
		if result.Verdict != probe.Indeterminate {
			t.Fatalf("result %q verdict = %q, want %q: the run ended before the measurement was collected", result.Probe, result.Verdict, probe.Indeterminate)
		}
		if result.Reason != probe.ReasonRunBudgetExceeded {
			t.Fatalf("result %q reason = %q, want %q", result.Probe, result.Reason, probe.ReasonRunBudgetExceeded)
		}
		if len(result.Observations) != 1 || result.Observations[0].Resolution != probe.Unresolved {
			t.Fatalf("result %q observations = %+v, want one unresolved observation", result.Probe, result.Observations)
		}
		if result.Verdict == probe.Pass {
			t.Fatalf("result %q was reported as a pass although the run ended first", result.Probe)
		}
	}
	if startedProbes == 0 {
		t.Fatal("no fixture was attempted at all; the budget case measured nothing")
	}
}

// TestCancelStopsInFlightMeasurements is the cancellation case of R-HR-NF-09:
// cancelling the run returns promptly instead of waiting out any bound, every
// probe affected by the cancellation is reported as the runner's cancelled fact
// and never as a pass, and the probes that had not started are stopped rather
// than measured against a dead context.
//
// Two of the in-flight fixtures are deliberately awkward. One reports its own
// honest failure for the cancelled attempt; the other fabricates a pass. The
// runner must report both as cancelled, because a probe that woke on cancellation
// does not get to decide the run's result.
func TestCancelStopsInFlightMeasurements(t *testing.T) {
	entered := make(chan struct{}, 2)

	honouring := &runnerFixture{
		name: "fixture.honours-cancel",
		kind: probe.ProbeEgress,
		run: func(ctx context.Context) probe.Result {
			entered <- struct{}{}
			<-ctx.Done()
			return runnerResult(runnerObservation("tcp 22", "203.0.113.10:22",
				probe.PurposePortReachability, probe.ObsProbeBudgetExpired, "dial tcp 203.0.113.10:22: context canceled"))
		},
	}
	fabricating := &runnerFixture{
		name: "fixture.fabricates-a-pass",
		kind: probe.ProbeEgress,
		run: func(ctx context.Context) probe.Result {
			entered <- struct{}{}
			<-ctx.Done()
			// The misbehaviour this case exists for: a cancelled probe claiming
			// success. The runner must not launder it into the run's result.
			return runnerResult(runnerObservation("tcp 22", "203.0.113.10:22",
				probe.PurposePortReachability, probe.ObsTCPEstablished, "connection established"))
		},
	}
	early := reachableFixture("fixture.early")
	neverStarted := func(name string) *runnerFixture {
		return &runnerFixture{
			name: name,
			kind: probe.ProbeEgress,
			run: func(context.Context) probe.Result {
				return runnerResult(runnerObservation("tcp 22", "203.0.113.10:22",
					probe.PurposePortReachability, probe.ObsTCPEstablished, "connection established"))
			},
		}
	}

	probes := []probe.Probe{early, honouring, fabricating, neverStarted("fixture.pending.0"), neverStarted("fixture.pending.1"), neverStarted("fixture.pending.2")}
	runner := probe.NewRunner(probe.Options{Concurrency: 2, ProbeTimeout: 5 * time.Second, RunBudget: 5 * time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	earlyResult := make(chan probe.Result, 1)
	finished := make(chan []probe.Result, 1)
	started := time.Now()
	go func() {
		finished <- runner.Run(ctx, probes, func(result probe.Result) {
			if result.Probe == "fixture.early" {
				earlyResult <- result
			}
		})
	}()

	// The measured result has to be in hand before the cancellation, otherwise
	// "the unaffected result survives" would be a race rather than an assertion.
	select {
	case result := <-earlyResult:
		if result.Verdict != probe.Pass {
			t.Fatalf("the early fixture's result = %q, want %q before cancellation", result.Verdict, probe.Pass)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the early fixture's result was never streamed")
	}
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatalf("only %d fixtures entered their measurement, want 2 in flight before cancelling", i)
		}
	}

	cancel()

	var run []probe.Result
	select {
	case run = <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("the cancelled run did not return promptly")
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("the cancelled run took %s, want it back before the 5s bounds ever mattered", elapsed)
	}
	if len(run) != len(probes) {
		t.Fatalf("run returned %d results, want %d: a cancelled run still reports every probe", len(run), len(probes))
	}

	if run[0].Probe != "fixture.early" || run[0].Verdict != probe.Pass || run[0].Reason != probe.ReasonOK {
		t.Fatalf("the probe that finished before cancellation = %+v, want its measured pass preserved", run[0])
	}

	passes := 0
	for i, result := range run {
		if result.Verdict == probe.Pass {
			passes++
		}
		if i == 0 {
			continue
		}
		if result.Verdict != probe.Indeterminate {
			t.Fatalf("cancelled result %q verdict = %q, want %q", result.Probe, result.Verdict, probe.Indeterminate)
		}
		if result.Reason != probe.ReasonRunCancelled {
			t.Fatalf("cancelled result %q reason = %q, want %q", result.Probe, result.Reason, probe.ReasonRunCancelled)
		}
		if len(result.Observations) != 1 || result.Observations[0].Resolution != probe.Unresolved {
			t.Fatalf("cancelled result %q observations = %+v, want one unresolved observation", result.Probe, result.Observations)
		}
	}
	if passes != 1 {
		t.Fatalf("the cancelled run reported %d passes, want exactly the one measured before cancellation", passes)
	}

	for i := 3; i < len(probes); i++ {
		if n := probes[i].(*runnerFixture).starts.Load(); n != 0 {
			t.Fatalf("probe %q was started %d times after the run was cancelled, want 0", probes[i].Name(), n)
		}
	}
}

// TestTimeoutPathsAreDistinct is obligation 3 of design §5.1 and the RG-8
// wrong-classification risk: three timeouts that look alike from a distance must
// stay three different facts. A probe's own budget expiring is a measured
// negative; a probe that ignores its bound is the runner's unresolved timeout; UDP
// silence is the measurement's own unresolved ambiguity. The test asserts three
// distinct (resolution, reason) pairs and that no fourth pair collapses any two of
// them into one.
func TestTimeoutPathsAreDistinct(t *testing.T) {
	forever := make(chan struct{})

	budgetExpired := &runnerFixture{
		name: "fixture.probe-budget-expired",
		kind: probe.ProbeEgress,
		run: func(ctx context.Context) probe.Result {
			// The probe's own budget (5ms) is far shorter than the runner's bound
			// (100ms). The blackholed port never answers inside it, and for a
			// reachability question that expiry is the measurement itself.
			select {
			case <-time.After(5 * time.Millisecond):
			case <-ctx.Done():
			}
			return runnerResult(runnerObservation("tcp 7844", "203.0.113.10:7844",
				probe.PurposePortReachability, probe.ObsProbeBudgetExpired, "dial tcp 203.0.113.10:7844: i/o timeout after 5ms"))
		},
	}
	ignoresBudget := &runnerFixture{
		name: "fixture.ignores-budget",
		kind: probe.ProbeEgress,
		run: func(context.Context) probe.Result {
			<-forever
			return probe.Result{}
		},
	}
	udpSilence := &runnerFixture{
		name: "fixture.udp-silence",
		kind: probe.ProbeProto,
		run: func(context.Context) probe.Result {
			// D8's narrow datagram question: the socket produced neither a reply
			// nor an error, which is ambiguous by construction and never a block.
			return runnerResult(runnerObservation("udp 7844 region1", "region1.v2.argotunnel.com:7844",
				probe.PurposeUDPReachability, probe.ObsUDPSilence, "no datagram reply and no socket error inside the probe budget"))
		},
	}

	runner := probe.NewRunner(probe.Options{Concurrency: 3, ProbeTimeout: 100 * time.Millisecond, RunBudget: 2 * time.Second})
	run := runner.Run(context.Background(), []probe.Probe{budgetExpired, ignoresBudget, udpSilence}, nil)
	if len(run) != 3 {
		t.Fatalf("run returned %d results, want 3", len(run))
	}

	cases := []struct {
		name           string
		wantResolution probe.Resolution
		wantReason     probe.ReasonCode
		wantVerdict    probe.Verdict
	}{
		{"the probe's own budget expiring is a measured failure", probe.Measured, probe.ReasonBudgetExpired, probe.Fail},
		{"a probe that ignores its bound is the runner's unresolved timeout", probe.Unresolved, probe.ReasonProbeTimeout, probe.Indeterminate},
		{"UDP silence is the measurement's own unresolved ambiguity", probe.Unresolved, probe.ReasonUDPSilence, probe.Indeterminate},
	}

	seen := make(map[string]string, len(cases))
	for i, tc := range cases {
		result := run[i]
		if result.Probe != []string{"fixture.probe-budget-expired", "fixture.ignores-budget", "fixture.udp-silence"}[i] {
			t.Fatalf("result %d names %q, want %q", i, result.Probe, cases[i].name)
		}
		if result.Verdict != tc.wantVerdict {
			t.Fatalf("%s: verdict = %q, want %q", tc.name, result.Verdict, tc.wantVerdict)
		}
		if result.Reason != tc.wantReason {
			t.Fatalf("%s: reason = %q, want %q", tc.name, result.Reason, tc.wantReason)
		}
		if result.Verdict == probe.Pass {
			t.Fatalf("%s: a timeout path was reported as a pass", tc.name)
		}
		if len(result.Observations) != 1 {
			t.Fatalf("%s: observations = %+v, want exactly one", tc.name, result.Observations)
		}
		obs := result.Observations[0]
		if obs.Resolution != tc.wantResolution {
			t.Fatalf("%s: resolution = %q, want %q", tc.name, obs.Resolution, tc.wantResolution)
		}
		pair := string(obs.Resolution) + "/" + string(obs.Reason)
		if previous, collapsed := seen[pair]; collapsed {
			t.Fatalf("%s collapsed onto %q, already reported by %s", tc.name, pair, previous)
		}
		seen[pair] = tc.name
	}
	if len(seen) != 3 {
		t.Fatalf("the three timeout paths produced %d distinct (resolution, reason) pairs, want 3", len(seen))
	}
	if ignoresBudget.starts.Load() != 1 {
		t.Fatalf("the bound-ignoring fixture was started %d times, want the runner to have attempted it once", ignoresBudget.starts.Load())
	}
}
