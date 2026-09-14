package probe

// This file is D8's datagram probe: `egress.quic`, the question "does one UDP datagram to
// the Cloudflare tunnel edge on 7844 draw any reply inside the probe's own budget?".
//
// The declared question is deliberately narrow, and it is stated in the probe's own
// detail text rather than left to a reader's assumption. A reply proves one thing — the
// datagram was not silently dropped on the way to the edge — and nothing more: the probe
// never says QUIC works and never says QUIC is blocked. Silence proves nothing either: a
// UDP datagram that draws no answer may have been filtered, may have reached a server
// that does not answer bare datagrams, or may simply have been ignored, and the three are
// indistinguishable from here. That is why silence is `unresolved`/`udp_silence` and
// never a failure (design D8, RG-5); a silent socket must never produce the wrong
// downgrade advice.
//
// No QUIC stack is involved. The probe sends one raw datagram over the injected
// `PacketDialer`/`PacketConn` seam and reads at most one reply, so the measurement costs
// no dependency and no trust decision (design D8's rejected alternative was a full QUIC
// handshake through a third-party stack). A real handshake would be the only way to
// establish QUIC viability, and the only question it would sharpen — "recommend HTTP/2 or
// not" — is already answered conservatively from the unresolved path.
//
// Both declared edge regions are measured, one observation each (design D10), because
// "one region reachable" is materially different from "both silent". Every region's
// socket is closed by this probe, and every outcome classifies through classify.go's
// table and nothing else, so one raw fact can never become two different codes.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// udpProbePayload is the datagram the probe sends. Its content cannot make this a
	// QUIC handshake — the probe has no QUIC stack — and it must not pretend to be
	// one: the declared question is only whether any reply comes back, so the payload
	// is a labelled marker a reader of the code can recognise.
	udpProbePayload = "herdr-reach udp reachability probe"
	// udpReadLimit bounds the datagram the probe reads. A datagram's own maximum is
	// bounded by the link's MTU in practice; the buffer only has to be large enough to
	// hold any reply this question cares about, and a reply is recognised by arriving
	// rather than by its length.
	udpReadLimit = 1500
	// udpDeclaredQuestion is the narrow question stated in every observation's detail
	// text. It is quoted here rather than in the wording functions so that no outcome
	// can omit it: the reader of a silent measurement in particular must see what was
	// and was not asked.
	udpDeclaredQuestion = "the declared question is narrow: does one UDP datagram to this edge and port draw any reply inside the probe's own budget? Silence here means only that no answer on UDP arrived, so it is reported as unresolved and never as a failure."
	// udpSubject names what was dialed in the not-measured wordings, so a reader is
	// told which capability was missing rather than which probe missed it.
	udpSubject = "the Cloudflare edge region over UDP"
	// udpFactLabel names the datagram question when no endpoint could be resolved at
	// all: it names the question rather than a region that was never dialed.
	udpFactLabel = "cloudflare edge udp"
)

// egressQuic is the `egress.quic` probe. It carries the run's seams and the run's
// declared target input, and it resolves its targets at Run time through the one
// declaration in targets.go so that an override is honoured (design D10).
type egressQuic struct {
	seams   Seams
	targets TargetInput
}

// newEgressQUIC builds `egress.quic` from the run's seams and its declared target input.
// The probe's targets are declared constants in targets.go, so the input travels to it
// only so that an override of those endpoints is honoured through the one declaration.
func newEgressQUIC(seams Seams, targets TargetInput) Probe {
	return &egressQuic{seams: seams, targets: targets}
}

// Name is the probe's stable identifier, declared once in registry.go.
func (p *egressQuic) Name() string { return probeNameEgressQUIC }

// Kind is the question family the probe belongs to: the datagram question separates a
// blocked destination from a blocked protocol, which is what ProbeProto means.
func (p *egressQuic) Kind() ProbeKind { return ProbeProto }

// now reads the run's clock, or the zero time when no clock was injected, so only an
// injected clock can move an elapsed value (design §3.3).
func (p *egressQuic) now() time.Time { return runClockNow(p.seams) }

// Run measures every declared region, in declaration order, and reports one observation
// per region plus the reduction of all of them.
//
// The result's target is the declared set rendered in declaration order, because a probe
// with two endpoints has no single target and naming only the first would hide the region
// the probe measured; each observation still carries its own exact address. The result's
// detail is the per-observation lines verbatim, exactly as the other multi-observation
// probes report theirs.
func (p *egressQuic) Run(ctx context.Context) Result {
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

// observe measures each declared region once, in declaration order, and keeps measuring
// the remaining regions after one has produced no answer: the split is the answer, and
// stopping at the first silence would hide the region that replied.
func (p *egressQuic) observe(ctx context.Context) []Observation {
	targets, unresolved, ok := declaredEndpoints(p.Name(), p.targets)
	if !ok {
		return []Observation{udpObserve(udpFactLabel, "", unresolved)}
	}
	observations := make([]Observation, 0, len(targets))
	for _, target := range targets {
		observations = append(observations, p.measure(ctx, target))
	}
	return observations
}

// measure performs one region's datagram exchange and reports what it observed.
//
// The steps are the order of the questions: whether the run was given the capability to
// open a datagram socket, whether the socket could be opened, whether the datagram could
// be sent, and finally what came back. Each step that cannot proceed reports what was
// missing instead of claiming anything about the region, and the socket's deadline is the
// probe's own budget, so silence is observed by the probe rather than waited for by the
// runner.
func (p *egressQuic) measure(ctx context.Context, target EffectiveTarget) Observation {
	address := target.Address()
	label := fmt.Sprintf("udp %d", target.Port)
	if target.Label != "" {
		label += " " + target.Label
	}

	if p.seams.PacketDialer == nil {
		return udpObserve(label, address, RawObservation{
			Kind:    ObsCapabilityExcluded,
			Wording: fmt.Sprintf("dial udp %s: no packet dialer is injected for this run, so %s was not measured and nothing is claimed about it", address, udpSubject),
		})
	}

	dialCtx, cancel := context.WithTimeout(ctx, DefaultDialBudget)
	defer cancel()
	conn, err := p.seams.PacketDialer.DialPacket(dialCtx, "udp", address)
	if err != nil {
		return udpObserve(label, address, packetErrorFact(address, err))
	}
	if conn == nil {
		return udpObserve(label, address, RawObservation{
			Kind:    ObsInternalFailure,
			Wording: fmt.Sprintf("dial udp %s: the packet dialer reported a socket it did not return, so no datagram could be sent and nothing is claimed about %s", address, udpSubject),
		})
	}
	// Closing is the probe's own duty: a probe that leaves a socket behind would leak
	// one per region per run.
	defer func() { _ = conn.Close() }()

	if err := conn.SetDeadline(p.exchangeDeadline()); err != nil {
		return udpObserve(label, address, RawObservation{
			Kind:    ObsUDPOtherError,
			Wording: fmt.Sprintf("udp %s: the socket refused a deadline (%v), so the exchange could not be bounded and nothing is claimed about %s", address, err, udpSubject),
		})
	}
	if _, err := conn.WriteTo([]byte(udpProbePayload), packetAddr{host: target.Host, port: target.Port}); err != nil {
		return udpObserve(label, address, packetErrorFact(address, err))
	}

	buffer := make([]byte, udpReadLimit)
	n, _, err := conn.ReadFrom(buffer)
	return udpObserve(label, address, udpReplyFact(address, n, err))
}

// exchangeDeadline is the instant the datagram exchange must stop by: the probe's own
// budget from now.
//
// It comes from the run's clock when one was injected, so a scripted run reads
// deterministically, and from the wall clock when none was: a run with no clock must
// still bound its exchange rather than wait forever for a reply that never comes.
func (p *egressQuic) exchangeDeadline() time.Time {
	now := p.now()
	if now.IsZero() {
		now = time.Now()
	}
	return now.Add(DefaultDialBudget)
}

// packetAddr is the net.Addr a datagram is sent to. It is built from the declared target
// rather than resolved through the standard library: naming an address is not measuring
// it, and a resolution call here would be a real network operation outside a seam, which
// is exactly what the seam set exists to prevent.
type packetAddr struct {
	host string
	port int
}

// Network reports the datagram network.
func (packetAddr) Network() string { return "udp" }

// String reports the address in the same "host:port" form the declared set echoes.
func (a packetAddr) String() string { return net.JoinHostPort(a.host, strconv.Itoa(a.port)) }

// udpObserve classifies one raw datagram fact for one declared region through the
// classification table — never inline — and states the probe's declared question in the
// detail text, so no outcome can quietly widen what was asked (design D8).
func udpObserve(label, address string, raw RawObservation) Observation {
	raw.Wording = strings.TrimSpace(raw.Wording + " " + udpDeclaredQuestion)
	return Observe(label, address, PurposeUDPReachability, raw)
}

// packetErrorFact turns one socket failure — opening the socket, or sending the datagram
// — into the raw fact the classification table sees.
//
// The mapping is by error identity, never by message text (R-HR-07). A denial from the
// seam is an attempt that was not made. An ICMP port-unreachable surfaced anywhere is the
// same definite negative. Everything else is an unclassified socket error: it is
// unresolved, and its wording is carried verbatim rather than borrowed into an answer the
// error does not establish. Silence is not in this function at all, because silence is
// what a read reports when nothing arrived, not what an error reports.
func packetErrorFact(address string, err error) RawObservation {
	wording := fmt.Sprintf("udp %s: %v", address, err)
	switch {
	case errors.Is(err, ErrSeamDenied):
		return RawObservation{
			Kind:    ObsCommandDenied,
			Wording: wording + fmt.Sprintf(" (the packet seam denied the attempt, so %s was not measured)", udpSubject),
		}
	case errors.Is(err, syscall.ECONNREFUSED):
		return RawObservation{
			Kind:    ObsUDPUnreachable,
			Wording: wording + " (the socket surfaced an ICMP port-unreachable, so the far end answered that nothing is listening on this UDP port: a definite negative for this datagram)",
		}
	default:
		return RawObservation{
			Kind:    ObsUDPOtherError,
			Wording: wording + " (the socket failed, so the datagram exchange produced no reply: the error is carried here verbatim and is not dressed up as an answer)",
		}
	}
}

// udpReplyFact turns one read result into the raw fact the classification table sees.
//
// The order of the branches is the order of the evidence. A read that returned at all is
// a reply, whatever its length: the declared question is whether any reply arrives.
// Otherwise the ways a read can end are checked in the order that keeps the definite
// negative apart from the two ambiguities: an ICMP port-unreachable is a measured
// failure, a deadline expiry with nothing read is silent (unresolved), and any other
// error is an unclassified ambiguity (unresolved) — never a failure.
func udpReplyFact(address string, n int, err error) RawObservation {
	switch {
	case err == nil:
		return RawObservation{
			Kind:    ObsUDPResponse,
			Wording: fmt.Sprintf("udp %s: a %d-byte datagram arrived from the edge, so this datagram to this edge and port was not silently dropped", address, n),
		}
	case errors.Is(err, syscall.ECONNREFUSED):
		return RawObservation{
			Kind:    ObsUDPUnreachable,
			Wording: fmt.Sprintf("udp %s: %v, which the socket surfaced as an ICMP port-unreachable: the far end answered that nothing is listening on this UDP port, a definite negative for this datagram", address, err),
		}
	case dialTimedOut(err):
		return RawObservation{
			Kind:    ObsUDPSilence,
			Wording: fmt.Sprintf("udp %s: no answer on UDP arrived inside the probe's own %s budget, and the socket reported no error at all", address, DefaultDialBudget),
		}
	default:
		return RawObservation{
			Kind:    ObsUDPOtherError,
			Wording: fmt.Sprintf("udp %s: %v, a socket error that is neither a reply nor an ICMP port-unreachable, carried here verbatim and not dressed up as an answer", address, err),
		}
	}
}
