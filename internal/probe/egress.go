package probe

// This file is the egress probe group of design §7: what this network permits. Three
// question families live here — `egress.hub.direct` (does the hub address this run was
// given accept a connection?), the two public-SSH probes (`egress.ssh.known`,
// `egress.ssh.443`: does a well-known public host speak SSH on this port?) and the two
// Cloudflare edge probes (`egress.cf.7844`, `egress.cf.443`: is the tunnel edge
// reachable, per region?).
//
// The hub probe's three properties shape it, and they matter more than its length.
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
// The hub probe dials and closes. It does not read a banner: whether the far end
// speaks SSH is the public-host probes' question, and answering it here would put
// a second question inside a probe the registry declares as one.
//
// The public-SSH probes ask the one question PRD §1.1 measured: does a well-known
// public host speak SSH on the declared port (:22, and again on the allowed port
// :443)? Their targets are declared constants rather than run input, and they read
// the identification string the far end sends first, because a port that answered is
// not the same fact as a port that answered with SSH. Their two name-level outcomes —
// a resolver that says the name does not exist, and a resolver that did not answer —
// are what separate "SSH is blocked" from "the destination is blocked", and they are
// deliberately not collapsed.
//
// Every probe here classifies only through classify.go's table, so one raw fact can
// never become two different codes depending on which probe saw it.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
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
	// sshBannerFactLabel names the SSH fact when no endpoint could be resolved at
	// all. It names the question rather than a port, for the same reason the hub's
	// label does.
	sshBannerFactLabel = "ssh banner"
	// publicSSHSubject names what was dialed in every not-measured wording an SSH
	// probe reports, so a reader is told which capability was missing rather than
	// which probe missed it.
	publicSSHSubject = "the public SSH target"
	// cloudflareEdgeSubject names what was dialed in the not-measured wordings the
	// per-region edge probes report, for the same reason.
	cloudflareEdgeSubject = "the Cloudflare edge region"
	// cloudflareEdgeFactLabel names the edge fact when no endpoint could be resolved
	// at all: it names the question rather than a region that was never dialed.
	cloudflareEdgeFactLabel = "cloudflare edge"
	// sshIdentificationPrefix is what every SSH identification string begins with
	// (RFC 4253 §4.2): the first line a server sends, before the key exchange.
	sshIdentificationPrefix = "SSH-"
	// sshBannerLimit bounds the identification-string read. An identification string
	// is at most 255 bytes including its line terminator (RFC 4253 §4.2), so the bound
	// is the protocol's own rather than a guess: a far end that keeps talking past it
	// is not sending an identification string.
	sshBannerLimit = 255
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

// addressPrefix returns the "<operation> <address>: " prefix a detail uses to say what a
// failed attempt was made against, or "" when the error's own message already names the
// address.
//
// The standard library formats its own network errors as "<operation> <address>:
// <cause>" — "dial tcp 127.0.0.1:1: connect: connection refused", "read tcp
// 192.0.2.5:53124->198.51.100.7:443: i/o timeout", "lookup nosuch.example: no such host"
// — so joining this prefix to one names the address twice in a detail the tool presents
// as verbatim evidence, and the doubling reaches every conclusion that quotes the detail.
// Each detail names the address once either way: the error keeps it when the error carries
// it, and the probe adds it when the error does not — a denied seam, or a dialer whose own
// error names nothing — because otherwise the detail would not say what was measured.
//
// Which reason code the fact becomes is still chosen from the error's identity in
// classify.go's table (R-HR-07): the only thing read from the text here is whether the
// text already contains the address it is about to quote.
func addressPrefix(operation, address string, err error) string {
	if err != nil && address != "" && strings.Contains(err.Error(), address) {
		return ""
	}
	return fmt.Sprintf("%s %s: ", operation, address)
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
	return dialFactFor("the hub", address, err)
}

// dialFactFor is dialFact for a probe whose subject is not the hub. subject names
// what was dialed, so the not-measured wordings read correctly for every
// reachability probe in this file ("the hub was not measured", "the public SSH
// target was not measured"); the four outcome mappings are shared, so one dial
// failure can never be classified two ways (RG-8).
func dialFactFor(subject, address string, err error) RawObservation {
	wording := fmt.Sprintf("%s%v", addressPrefix("dial tcp", address, err), err)
	switch {
	case errors.Is(err, ErrSeamDenied):
		return RawObservation{
			Kind:    ObsCommandDenied,
			Wording: wording + fmt.Sprintf(" (the dial seam denied the attempt, so %s was not measured)", subject),
		}
	case dialTimedOut(err):
		return RawObservation{
			Kind: ObsProbeBudgetExpired,
			Wording: fmt.Sprintf("%s after the probe's own %s dial budget, which is shorter than the runner's %s per-probe bound: the declared port did not answer inside the probe's own budget, which for the question \"is this port reachable?\" is the measurement itself",
				wording, DefaultDialBudget, DefaultProbeTimeout),
		}
	case errors.Is(err, syscall.ECONNREFUSED):
		return RawObservation{Kind: ObsTCPRefused, Wording: wording}
	case errors.Is(err, syscall.ECONNRESET):
		return RawObservation{Kind: ObsTCPReset, Wording: wording}
	default:
		return RawObservation{
			Kind:    ObsInternalFailure,
			Wording: wording + fmt.Sprintf(" (the dial failed for a reason no row of the classification table names, so nothing is claimed about %s)", subject),
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

// resolverFact turns one name-resolution failure into the raw fact the
// classification table sees, the way dialFactFor does for a dial failure.
//
// The mapping is by error identity, never by message text (R-HR-07), and it keeps
// apart the two facts that must never collapse: a resolver answering
// authoritatively that the name does not exist is a measurement *of the name* (a
// definite negative about the destination's name), while a resolver that timed out,
// returned SERVFAIL or was never able to ask produced no answer at all (an ambiguity,
// never a claimed block). Reporting the first as the second would turn "we could not
// find out" into "the destination is blocked", which is precisely the mistake
// PRD §1.1 exists to prevent; reporting the second as the first would fabricate a
// negative the resolver never gave.
//
// isAuthoritativeNegative reads net.DNSError's own IsNotFound field rather than the
// message, because the message is the operating system's wording and wording must
// never move a classification.
func resolverFact(host string, err error) RawObservation {
	wording := fmt.Sprintf("%s%v", addressPrefix("resolve", host, err), err)
	switch {
	case errors.Is(err, ErrSeamDenied):
		return RawObservation{
			Kind:    ObsCommandDenied,
			Wording: wording + " (the resolver seam denied the lookup, so nothing is claimed about the name)",
		}
	case isAuthoritativeNegative(err):
		return RawObservation{
			Kind:    ObsResolverAuthoritativeNegative,
			Wording: wording + " (the resolver answered authoritatively that the name does not exist: a fact about the name, not about the network)",
		}
	default:
		return RawObservation{
			Kind:    ObsResolverUnavailable,
			Wording: wording + " (the resolver produced no answer at all, so the destination was not measured and nothing is claimed about it)",
		}
	}
}

// isAuthoritativeNegative reports whether a resolver error is the resolver's own
// "this name does not exist" answer.
func isAuthoritativeNegative(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr) && dnsErr.IsNotFound
}

// isNameResolutionError reports whether a dial error is the dialer's own name
// resolution failing, rather than anything about the port. A dialer that resolves
// the name itself — as the standard library's does — surfaces a name-level failure
// through the dial call, and that fact belongs to the name question whichever seam
// raised it.
func isNameResolutionError(err error) bool {
	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr)
}

// declaredEndpoints resolves the run's declared target set and returns this probe's
// endpoints, in declaration order. A probe that declares two regions measures both;
// an override replaces them, so the count is the effective one rather than the
// declared one (design D10).
//
// The resolution goes through EffectiveTargets rather than through a private lookup:
// the declared set, the documented default port and the override rules live in
// exactly one place (targets.go), and a probe that re-derived them could disagree
// with the declaration the payload echoes (R-HR-NF-10).
//
// A failed resolution or a probe with no declared endpoint is reported as an internal
// failure rather than guessed at: the declaration is this package's own contract, so
// a probe missing from it is a defect in this package and not a fact about the
// network.
func declaredEndpoints(name string, in TargetInput) ([]EffectiveTarget, RawObservation, bool) {
	effective, err := EffectiveTargets(in)
	if err != nil {
		return nil, RawObservation{
			Kind:    ObsInternalFailure,
			Wording: fmt.Sprintf("the declared target set for %s could not be resolved from the run's input, so no measurement was attempted: %v", name, err),
		}, false
	}
	var mine []EffectiveTarget
	for _, target := range effective {
		if target.Probe == name {
			mine = append(mine, target)
		}
	}
	if len(mine) == 0 {
		return nil, RawObservation{
			Kind:    ObsInternalFailure,
			Wording: fmt.Sprintf("%s declares no endpoint in the declared target set, so this probe had nothing to measure", name),
		}, false
	}
	return mine, RawObservation{}, true
}

// declaredEndpoint is declaredEndpoints for a probe that declares exactly one
// endpoint, which is every reachability probe in this file except the two
// per-region Cloudflare edge probes.
func declaredEndpoint(name string, in TargetInput) (EffectiveTarget, RawObservation, bool) {
	endpoints, unresolved, ok := declaredEndpoints(name, in)
	if !ok {
		return EffectiveTarget{}, unresolved, false
	}
	return endpoints[0], RawObservation{}, true
}

// reachAttempt performs the shared body of the TCP reachability probes this slice
// lands: the two public-SSH probes and the two Cloudflare edge probes. The hub probe
// keeps its own body: its target is run input with a not-measured case of its own,
// and its behaviour is fixed by the slice that landed it.
//
// It returns the open connection when the attempt reached the far end — the caller
// owns closing it — together with the observation that answers the port question.
// When the attempt did not reach the far end, the connection is nil and the returned
// observation is the terminal answer. A probe that must read something from the far
// end replaces the established observation with what it read; the connection
// completing is not yet the answer to a question about the service on that port.
//
// The attempt is shared so that one dial failure can never be classified two ways,
// which is the RG-8 wrong-classification risk this package's single classification
// table exists to remove. subject names what was dialed in the not-measured wordings
// and label is the observation label the calling probe reports.
//
// The declared name is resolved through the injected resolver before the dial, which
// is what lets a name-level failure be attributed to the name — "no such host" is a
// measurement of the name, while a resolver that did not answer is not a measurement
// at all — instead of being buried in a socket error. The dial itself goes to the
// declared address, so "the dialed set equals the declared set" (R-HR-NF-10) stays a
// comparison of two values rather than a claim about how a name was resolved: the
// cost is that a production dialer resolves the name a second time inside the dial,
// and the reason for paying it is that the alternative — dialing a resolved address —
// would put an address into the dialed set that the declaration does not contain.
func reachAttempt(ctx context.Context, seams Seams, target EffectiveTarget, subject, label string) (net.Conn, Observation) {
	address := target.Address()

	if seams.Resolver == nil {
		return nil, Observe(label, address, PurposeNameResolution, RawObservation{
			Kind:    ObsCapabilityExcluded,
			Wording: fmt.Sprintf("resolve %s: no resolver is injected for this run, so %s could not be attempted and nothing is claimed about it", target.Host, address),
		})
	}
	addresses, err := seams.Resolver.LookupHost(ctx, target.Host)
	if err != nil {
		return nil, Observe(label, address, PurposeNameResolution, resolverFact(target.Host, err))
	}
	if len(addresses) == 0 {
		// A resolver that answered with nothing has answered about the name: the
		// name has no address. That is a measurement of the name, not an ambiguity
		// about the network.
		return nil, Observe(label, address, PurposeNameResolution, RawObservation{
			Kind:    ObsResolverAuthoritativeNegative,
			Wording: fmt.Sprintf("resolve %s: the resolver answered with no address at all, which is a fact about the name and not about the network", target.Host),
		})
	}

	if seams.Dialer == nil {
		return nil, Observe(label, address, PurposePortReachability, RawObservation{
			Kind:    ObsCapabilityExcluded,
			Wording: fmt.Sprintf("dial tcp %s: no dialer is injected for this run, so %s could not be measured", address, subject),
		})
	}
	dialCtx, cancel := context.WithTimeout(ctx, DefaultDialBudget)
	defer cancel()
	conn, err := seams.Dialer.DialContext(dialCtx, "tcp", address)
	if err != nil {
		// A dialer whose own name resolution failed has surfaced the same fact the
		// resolver seam would have reported, so it is classified the same way: by the
		// name, and under the name question, whoever raised it.
		if isNameResolutionError(err) {
			return nil, Observe(label, address, PurposeNameResolution, resolverFact(target.Host, err))
		}
		return nil, Observe(label, address, PurposePortReachability, dialFactFor(subject, address, err))
	}
	if conn == nil {
		// A dialer that reports success without a connection leaves nothing to read:
		// reporting the port as established would be a positive claim about a socket
		// nobody holds.
		return nil, Observe(label, address, PurposePortReachability, RawObservation{
			Kind:    ObsInternalFailure,
			Wording: fmt.Sprintf("dial tcp %s: the dialer reported a connection it did not return, so nothing could be read from the far end and nothing is claimed about %s", address, subject),
		})
	}
	return conn, Observe(label, address, PurposePortReachability, RawObservation{
		Kind:    ObsTCPEstablished,
		Wording: fmt.Sprintf("dial tcp %s: the connection was established and closed without reading or writing, which measures TCP reachability of the declared address and nothing else", address),
	})
}

// egressSSH is one of the two public-SSH destination probes of PRD §5.1:
// `egress.ssh.known` (a well-known public host on :22) and `egress.ssh.443` (the same
// question on the allowed port :443). One implementation serves both because they ask
// the same question of two declared endpoints, and PRD §1.1's evidence is exactly
// their comparison: SSH reachable on one port and not on the other is a fact about the
// port, and it is the fact that separates "SSH is blocked" from "the destination is
// blocked".
type egressSSH struct {
	name    string
	seams   Seams
	targets TargetInput
}

// newEgressSSHKnown builds `egress.ssh.known` from the run's seams and its declared
// target input. The probe's target is a declared constant rather than run input, so
// the input travels to it only so that an override of its endpoint is honoured through
// the one declaration in targets.go.
func newEgressSSHKnown(seams Seams, targets TargetInput) Probe {
	return &egressSSH{name: probeNameEgressSSHKnown, seams: seams, targets: targets}
}

// newEgressSSH443 builds `egress.ssh.443`: the same question on the port a
// locked-down network is likelier to allow (PRD §1.1).
func newEgressSSH443(seams Seams, targets TargetInput) Probe {
	return &egressSSH{name: probeNameEgressSSH443, seams: seams, targets: targets}
}

// Name is the probe's stable identifier, declared once in registry.go.
func (p *egressSSH) Name() string { return p.name }

// Kind is the question family the probe belongs to.
func (p *egressSSH) Kind() ProbeKind { return ProbeEgress }

// now reads the run's clock, or the zero time when no clock was injected, so only an
// injected clock can move an elapsed value (design §3.3).
func (p *egressSSH) now() time.Time { return runClockNow(p.seams) }

// Run performs the one measurement this probe declares and reports it as one
// observation.
func (p *egressSSH) Run(ctx context.Context) Result {
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

// observe asks the probe's one question: does the declared host speak SSH on the
// declared port?
//
// The steps are the order of the questions. Which endpoint was declared, whether the
// name resolves, whether the run was given the capability to dial, and finally what
// the far end sent. Each step that cannot proceed reports what was missing instead of
// claiming anything about the destination.
func (p *egressSSH) observe(ctx context.Context) Observation {
	target, unresolved, ok := declaredEndpoint(p.Name(), p.targets)
	if !ok {
		return Observe(sshBannerFactLabel, "", PurposePortReachability, unresolved)
	}
	address := target.Address()
	label := fmt.Sprintf("tcp %d ssh banner", target.Port)

	conn, observation := reachAttempt(ctx, p.seams, target, publicSSHSubject, label)
	if conn == nil {
		return observation
	}
	// The connection completing is not yet the answer: the answer is the
	// identification string the far end sends first, so the established observation
	// reachAttempt produced is replaced by what was read. The probe owns closing the
	// socket either way.
	defer func() { _ = conn.Close() }()

	banner, err := readIdentificationString(conn, p.readDeadline())
	return Observe(label, address, PurposePortReachability, bannerFact(address, banner, err))
}

// readDeadline is the instant the identification-string read must stop by: the
// probe's own budget from now, exactly as the dial was bounded.
//
// It comes from the run's clock when one was injected, so a scripted run reads
// deterministically, and from the wall clock when none was: a run with no clock must
// still bound its read rather than wait forever on a far end that accepted the
// connection and then said nothing. net.Conn carries no context, so a deadline is the
// only bound a read can be given.
func (p *egressSSH) readDeadline() time.Time {
	now := p.now()
	if now.IsZero() {
		now = time.Now()
	}
	return now.Add(DefaultDialBudget)
}

// readIdentificationString reads what the far end sends for the SSH identification
// string (RFC 4253 §4.2), up to sshBannerLimit bytes, and reports the bytes it read
// and the error that ended the read.
//
// It stops as soon as the bytes it holds carry an SSH identification string or a line
// terminator, so a peer that answers in more than one packet — or whose stream ends
// mid-identification — is still classified from what arrived, while a peer that keeps
// talking is bounded. The deadline is the probe's own budget: without it a far end
// that accepts the connection and then says nothing would hold the probe until the
// runner abandons it, and "said nothing" would be reported as a hung probe rather
// than as this probe's own measured negative.
func readIdentificationString(conn net.Conn, deadline time.Time) ([]byte, error) {
	if err := conn.SetReadDeadline(deadline); err != nil {
		return nil, err
	}
	buffer := make([]byte, sshBannerLimit)
	total := 0
	for total < len(buffer) {
		n, err := conn.Read(buffer[total:])
		total += n
		if err != nil {
			return buffer[:total], err
		}
		if n == 0 || carriesSSHIdentification(buffer[:total]) || bytes.ContainsRune(buffer[:total], '\n') {
			break
		}
	}
	return buffer[:total], nil
}

// carriesSSHIdentification reports whether the bytes read hold an SSH identification
// string: a line beginning with the documented prefix.
//
// Looking at every line rather than only at the first is deliberate, because RFC 4253
// §4.2 permits a server to send other lines before its identification string. A
// partially read identification string still counts, which is what makes a truncated
// read classify as what it is.
func carriesSSHIdentification(data []byte) bool {
	for _, line := range bytes.Split(data, []byte("\n")) {
		if bytes.HasPrefix(bytes.TrimRight(line, "\r"), []byte(sshIdentificationPrefix)) {
			return true
		}
	}
	return false
}

// bannerFact turns what was read from an accepted connection into the raw fact the
// classification table sees.
//
// The order of the branches is the order of the evidence. Bytes carrying an SSH
// identification string are the positive answer and are recognised first, so a
// truncated read that still carries the prefix is classified as SSH rather than as a
// failure. Only then do the ways a read can end matter: a deadline expiry inside the
// probe's own budget is this probe's measured negative, a reset is the far end's
// negative, and any other error is reported verbatim as an internal failure, never
// borrowed into a negative the error does not establish. Anything else — including a
// far end that sent nothing before closing — is not an SSH identification string, and
// that is exactly what `banner_not_ssh` says.
//
// The bytes read are quoted verbatim into the detail, so the operating system's (or
// the far end's) own evidence stays quotable beside the stable code (R-HR-07).
func bannerFact(address string, data []byte, err error) RawObservation {
	read := strconv.Quote(string(data))
	switch {
	case carriesSSHIdentification(data):
		return RawObservation{
			Kind:    ObsSSHBannerReceived,
			Wording: fmt.Sprintf("tcp %s: read %d bytes (%s); the first line is an SSH identification string, so this port speaks SSH", address, len(data), read),
		}
	case dialTimedOut(err):
		return RawObservation{
			Kind: ObsProbeBudgetExpired,
			Wording: fmt.Sprintf("%sthe connection was accepted but %v inside the probe's own %s budget with no identification string: for the question \"does this port speak SSH?\" nothing inside the budget is the measurement itself",
				addressPrefix("tcp", address, err), err, DefaultDialBudget),
		}
	case errors.Is(err, syscall.ECONNRESET):
		return RawObservation{Kind: ObsTCPReset, Wording: fmt.Sprintf("%s%v", addressPrefix("tcp", address, err), err)}
	case err != nil && !errors.Is(err, io.EOF):
		return RawObservation{
			Kind:    ObsInternalFailure,
			Wording: fmt.Sprintf("%s%v (reading the identification string failed for a reason no row of the classification table names, so nothing is claimed about this port)", addressPrefix("tcp", address, err), err),
		}
	case len(data) == 0:
		ending := "without ending the connection"
		if err != nil {
			ending = fmt.Sprintf("(the read ended with %v)", err)
		}
		return RawObservation{
			Kind:    ObsBannerNotSSH,
			Wording: fmt.Sprintf("tcp %s: the far end sent no identification string at all %s, and an SSH server sends its identification string first, so this port is not serving SSH", address, ending),
		}
	default:
		return RawObservation{
			Kind:    ObsBannerNotSSH,
			Wording: fmt.Sprintf("tcp %s: read %d bytes (%s), which is not an SSH identification string: something answered on this port and it is not an SSH server (an intercepting proxy looks like this)", address, len(data), read),
		}
	}
}

// egressCF is one of the two per-region Cloudflare edge probes of PRD §5.1:
// `egress.cf.7844` (the tunnel port) and `egress.cf.443` (the fallback path over
// HTTPS). Both edge regions are declared and both are measured (design D10), and each
// region becomes its own observation, because "one region reachable" is materially
// different from "both blocked" — a difference the Cloudflare adapter and the user
// both need, and one a single aggregate would hide.
//
// The probe measures TCP reachability and nothing more: it reads no banner, performs
// no handshake and establishes nothing about a tunnel. Whether a tunnel is viable is a
// question for internal/transport over these measurements, and it cannot be answered
// here.
type egressCF struct {
	name    string
	seams   Seams
	targets TargetInput
}

// newEgressCF7844 builds `egress.cf.7844`: the port Cloudflare's tunnel egress
// listens on.
func newEgressCF7844(seams Seams, targets TargetInput) Probe {
	return &egressCF{name: probeNameEgressCF7844, seams: seams, targets: targets}
}

// newEgressCF443 builds `egress.cf.443`: the same edge over the port a locked-down
// network is likelier to allow.
func newEgressCF443(seams Seams, targets TargetInput) Probe {
	return &egressCF{name: probeNameEgressCF443, seams: seams, targets: targets}
}

// Name is the probe's stable identifier, declared once in registry.go.
func (p *egressCF) Name() string { return p.name }

// Kind is the question family the probe belongs to.
func (p *egressCF) Kind() ProbeKind { return ProbeEgress }

// now reads the run's clock, or the zero time when no clock was injected, so only an
// injected clock can move an elapsed value (design §3.3).
func (p *egressCF) now() time.Time { return runClockNow(p.seams) }

// Run measures every declared region, in declaration order, and reports one
// observation per region plus the reduction of all of them.
//
// The result's target is the declared set rendered in declaration order, because a
// probe with two endpoints has no single target and naming only the first would hide
// the region the probe actually measured; each observation still carries its own exact
// address. The result's detail is the per-observation lines verbatim, exactly as
// `local.sshd` reports its three.
func (p *egressCF) Run(ctx context.Context) Result {
	started := p.now()
	observations := p.observe(ctx)
	verdict, reason := Aggregate(observations)
	details := make([]string, 0, len(observations))
	targets := make([]string, 0, len(observations))
	for _, observation := range observations {
		details = append(details, observation.Detail)
		targets = append(targets, observation.Target)
	}
	return Result{
		Probe:        p.Name(),
		Kind:         p.Kind(),
		Target:       strings.Join(targets, ", "),
		Verdict:      verdict,
		Reason:       reason,
		Detail:       strings.Join(details, "\n"),
		Elapsed:      p.now().Sub(started),
		Observations: observations,
	}
}

// observe measures each declared region once, in declaration order.
//
// It keeps measuring the remaining regions after one has failed: the split is the
// answer, and stopping at the first failure would report "the edge is unreachable"
// where the measurement says "one region is reachable". An override replaces the
// declared regions, so this loop follows the effective set rather than the declaration
// (design D10).
func (p *egressCF) observe(ctx context.Context) []Observation {
	targets, unresolved, ok := declaredEndpoints(p.Name(), p.targets)
	if !ok {
		return []Observation{Observe(cloudflareEdgeFactLabel, "", PurposePortReachability, unresolved)}
	}
	observations := make([]Observation, 0, len(targets))
	for _, target := range targets {
		label := fmt.Sprintf("tcp %d", target.Port)
		if target.Label != "" {
			label += " " + target.Label
		}
		conn, observation := reachAttempt(ctx, p.seams, target, cloudflareEdgeSubject, label)
		if conn != nil {
			// The connection is the measurement; nothing is read or written on it.
			// Closing is the probe's own duty, and each region's connection is closed
			// before the next region is dialed, so no socket lingers past its region.
			_ = conn.Close()
		}
		observations = append(observations, observation)
	}
	return observations
}
