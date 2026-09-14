package probe

// This file is the R-HR-04 chain probe: `tls.interception`, the question "which chain does
// the declared TLS target present, and is its issuer the declared expected publisher?".
//
// The measurement is one handshake through the injected `TLSVerifier` seam, and the probe
// is deliberately not the place where a certificate is trusted: the verifier owns the
// socket, the root pool and the verification code, and this probe owns only the declared
// question, the verifying configuration it hands over, and the classification of what came
// back.
//
// Four outcomes are reported, and the vocabulary keeps them distinct (design §5.1):
//
//   - the chain verified and its issuer is in the declared expected set — a measured pass;
//   - the chain failed verification — a measured failure whose detail carries the
//     verifier's own code;
//   - the chain verified but its issuer is outside the declared expected set — a measured
//     failure whose wording says exactly that. It is the interception candidate R-HR-04
//     exists for, and the wording never accuses: it reports the observed issuer and the
//     declared set it was compared against, and claims nothing about why they differ;
//   - the handshake produced no answer — unresolved. A handshake that produced nothing is
//     not a rejected chain, and collapsing the two would report an interception-free
//     network whenever a handshake failed for an unrelated reason.
//
// Verification is never weakened to obtain a result. The configuration the probe hands to
// the verifier disables no verification, supplies no root pool of its own — so the platform
// verifier's answer is the one measured — and the probe verifies exactly once, whatever
// came back. A verifier seam that denied the handshake, or no verifier injected at all,
// are attempts that were not made and are reported as such.
//
// The declared expected issuer set lives here, beside the probe that consumes it, because
// it is part of this measurement's contract rather than a property of the declared target
// set: targets.go declares where the probe measures, and this declaration says what
// publisher that target is expected to present. It is compared as a declared value, never
// parsed out of an error, so wording cannot move the classification (R-HR-07).

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// tlsChainFactLabel names the chain fact when no endpoint could be resolved at all:
	// it names the question rather than a host that was never measured.
	tlsChainFactLabel = "tls chain"
	// tlsSubject names what was measured in the not-measured wordings, so a reader is
	// told which capability was missing rather than which probe missed it.
	tlsSubject = "the chain presented by the declared TLS target"
)

// declaredExpectedIssuers is the declared expected publisher set, per declared TLS host:
// the target's chain must verify, and its issuer must be one of these values, for the
// measurement to be a pass. The set for the host of PRD §1.1 is the one that measurement
// recorded (`www.cloudflare.com`, issuer `Let's Encrypt/ISRG`, verification code 0); the
// comment on the entry is deliberately the reason it is there rather than a copy of an
// observed value that could drift.
var declaredExpectedIssuers = map[string][]string{
	tlsProbeHost: {"Let's Encrypt/ISRG"},
}

// expectedIssuerFor reports whether an observed issuer is in the declared expected set for
// a host.
//
// The comparison is on the declared value, case-insensitively and with surrounding space
// ignored: certificate publishers vary in case between chains, and a case-only difference
// must not turn a pass into a reported interception. It is never a comparison of error
// text, so two ways of wording the same failure cannot classify differently.
func expectedIssuerFor(host, issuer string) bool {
	for _, expected := range declaredExpectedIssuers[host] {
		if strings.EqualFold(strings.TrimSpace(issuer), strings.TrimSpace(expected)) {
			return true
		}
	}
	return false
}

// tlsInterception is the `tls.interception` probe. It carries the run's seams and the
// run's declared target input.
type tlsInterception struct {
	seams   Seams
	targets TargetInput
}

// newTLSInterception builds `tls.interception` from the run's seams and its declared
// target input. The probe's target is a declared constant in targets.go, so the input
// travels to it only so that an override of that endpoint is honoured through the one
// declaration.
func newTLSInterception(seams Seams, targets TargetInput) Probe {
	return &tlsInterception{seams: seams, targets: targets}
}

// Name is the probe's stable identifier, declared once in registry.go.
func (p *tlsInterception) Name() string { return probeNameTLSInterception }

// Kind is the question family the probe belongs to.
func (p *tlsInterception) Kind() ProbeKind { return ProbeTLS }

// now reads the run's clock, or the zero time when no clock was injected, so only an
// injected clock can move an elapsed value (design §3.3).
func (p *tlsInterception) now() time.Time { return runClockNow(p.seams) }

// Run performs the one handshake this probe declares and reports it as one observation.
func (p *tlsInterception) Run(ctx context.Context) Result {
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

// observe asks the probe's one question: what chain does the declared target present?
//
// The steps are the order of the questions: which target was declared, whether the run was
// given the capability to verify at all, and what the handshake reported. Each step that
// cannot proceed reports what was missing instead of claiming anything about the chain.
//
// The handshake is bounded by the caller's context alone. It is one call, and the runner's
// per-probe bound is the bound that applies to it; the verifier must honour that context,
// which its own contract requires.
func (p *tlsInterception) observe(ctx context.Context) Observation {
	target, unresolved, ok := declaredEndpoint(p.Name(), p.targets)
	if !ok {
		return Observe(tlsChainFactLabel, "", PurposeTLSCertificate, unresolved)
	}
	address := target.Address()
	label := fmt.Sprintf("tls %d chain", target.Port)

	if p.seams.TLSVerifier == nil {
		return tlsObserve(label, address, RawObservation{
			Kind:    ObsCapabilityExcluded,
			Wording: fmt.Sprintf("tls %s: no TLS verifier is injected for this run, so %s was not measured and nothing is claimed about it", address, tlsSubject),
		})
	}

	cfg := tlsVerifyingConfig(target.Host)
	verification, err := p.seams.TLSVerifier.Verify(ctx, address, cfg)
	return tlsObserve(label, address, tlsChainFact(target.Host, address, verification, err))
}

// tlsVerifyingConfig returns the verifying configuration the handshake runs with. It is
// built fresh for every run, so one run's configuration can never leak into the next.
//
// Verification is never weakened to obtain a result (R-HR-04): certificate verification is
// left on, the root pool is deliberately left nil so the platform verifier decides, and the
// declared server name is set from the target so a production verifier verifies the chain
// against the host the run declared. Nothing may retry without verification, and the
// configuration is not modified after the verifier receives it.
func tlsVerifyingConfig(host string) *tls.Config {
	return &tls.Config{
		ServerName: host,
		// The zero value is already false; it is written out because it is the
		// property the spec's scenario asserts and a reader must not have to know the
		// zero value to see it.
		InsecureSkipVerify: false,
	}
}

// tlsObserve classifies one raw chain fact through the classification table — never
// inline — and carries the operating system's or verifier's wording into the detail
// unchanged (R-HR-07).
func tlsObserve(label, address string, raw RawObservation) Observation {
	return Observe(label, address, PurposeTLSCertificate, raw)
}

// tlsChainFact turns one handshake's result into the raw fact the classification table
// sees.
//
// The order of the branches is the order of the evidence. A denial from the seam is an
// attempt that was not made, and it is checked first so a denied seam can never be misread
// as a rejected chain. A verification failure is a measurement of the target and carries
// the verifier's own code into the detail. Any other error produced no answer at all and
// is unresolved. Only when the handshake verified does the issuer decide between the pass
// and the interception candidate, and that wording reports the observed issuer beside the
// declared set it was compared against: it never accuses the network of causing the
// difference.
//
// The observed issuer and the verification code are quoted into every definite outcome's
// detail, because R-HR-04 requires the result itself to carry both. An empty value is
// reported as such rather than omitted, so a reader can tell "the verifier reported no
// code" from "the code was dropped".
func tlsChainFact(host, address string, verification TLSVerification, err error) RawObservation {
	issuer := verification.Issuer
	code := verification.VerificationCode
	if code == "" {
		code = "the verifier reported no code"
	}
	switch {
	case errors.Is(err, ErrSeamDenied):
		return RawObservation{
			Kind:    ObsCommandDenied,
			Wording: fmt.Sprintf("tls %s: %v (the verifier seam denied the handshake, so %s was not measured)", address, err, tlsSubject),
		}
	case errors.Is(err, ErrTLSVerification):
		return RawObservation{
			Kind: ObsTLSVerifyFailed,
			Wording: fmt.Sprintf("tls %s: the chain failed verification (observed issuer %q, verification code %q): %v; the failure is reported as the measurement it is, and verification was not disabled or retried to obtain a different result",
				address, issuer, code, err),
		}
	case err != nil:
		return RawObservation{
			Kind: ObsTLSHandshakeError,
			Wording: fmt.Sprintf("tls %s: the handshake produced no answer (observed issuer %q, verification code %q): %v; the failure is neither a verification failure nor a concluded issuer mismatch, so nothing is claimed about the chain",
				address, issuer, code, err),
		}
	case expectedIssuerFor(host, issuer):
		return RawObservation{
			Kind: ObsTLSVerified,
			Wording: fmt.Sprintf("tls %s: the chain verified and its observed issuer %q is in the declared expected set for %s (verification code %q)",
				address, issuer, host, code),
		}
	default:
		return RawObservation{
			Kind: ObsTLSIssuerUnexpected,
			Wording: fmt.Sprintf("tls %s: the chain verified and its observed issuer %q is not in the declared expected set for %s (verification code %q). The result records the observed issuer and the declared set it was compared against, and makes no claim about why the two differ",
				address, issuer, host, code),
		}
	}
}
