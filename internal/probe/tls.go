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
//   - the chain verified but its issuer is outside the declared expected set — unresolved.
//     Publishers rotate, and the single declared entry means the first rotation lands here,
//     where the run cannot tell a rotation from an interception (the dated note of
//     2026-09-20 in the change's design.md records the measurement that settled this). The
//     wording reports the observed issuer and the declared set it was compared against and
//     states exactly what the run cannot establish: it never accuses, and it does not call
//     the divergence harmless either;
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
// and the recorded divergence, and that wording reports the observed issuer beside the
// declared set it was compared against and states that the run cannot distinguish a
// publisher change from an interception: it never accuses the network, and it does not
// call the divergence harmless either.
//
// The observed issuer and the verification code are quoted into every outcome the handshake
// produced, because R-HR-04 requires the result itself to carry both. An empty value is
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
			Wording: fmt.Sprintf("tls %s: the chain verified and its observed issuer %q is not in the declared expected set for %s (verification code %q); the result records the observed publisher and the declared set it was compared against, and this run cannot distinguish a publisher change from an interception, so the divergence is reported as unresolved rather than as a failure",
				address, issuer, host, code),
		}
	}
}

// ---------------------------------------------------------------------------
// `tls.truststore`: the same chain, against the local trust store (R-HR-04, RG-3)
// ---------------------------------------------------------------------------
//
// The second TLS question of PRD §5.1 is not "what did the peer present?" but "what
// does this machine's own pool do with it?". It is the same declared target and the
// same handshake seam as `tls.interception`, and a deliberately different question:
// the chain probe reads the chain, and this probe reads the local trust store.
//
// The probe exists in this shape because the honest answer is not always available, and
// the two degradations below are the reason it is written the way it is:
//
//   - On macOS Go cannot enumerate the system roots, and keychain trust is only visible
//     through the platform verifier when no explicit root pool is supplied. A pool the
//     probe cannot enumerate is a pool whose verdict the probe cannot attribute, so the
//     measurement is unresolved and the documented limitation travels in the result's
//     own detail (RG-3). The probe does not perform the handshake on that platform: an
//     answer the platform does not let this probe attribute to a pool would be an answer
//     about nothing the probe can name.
//   - An `SSL_CERT_FILE` or `SSL_CERT_DIR` override makes Go's certificate loader read
//     that file or directory *in place of* the platform pool. The handshake would then
//     verify against a pool this run supplied rather than against the machine's trust
//     store, so accepting the chain would be a false pass. The probe reports the
//     override instead, and again performs no handshake, because the pool in force is
//     not the pool the declared question names.
//
// Everything else follows the classification table (classify.go) and never chooses a
// reason code inline: the probe reports the platform's own wording and the table
// decides what the fact is. Verification is never weakened to obtain an answer — the
// configuration is the same `tlsVerifyingConfig` the chain probe uses, with the root
// pool deliberately left nil so the platform verifier is the one that answers.
//
// The capability ordering is deliberate and is the order of the questions: which target
// was declared, whether an override replaces the pool, whether the platform can be
// attributed at all, and finally whether a handshake can be performed. Each step that
// cannot proceed reports what was missing instead of claiming anything about the pool.

const (
	// trustStoreFactLabel names the trust-store fact when no endpoint could be
	// resolved at all: it names the question rather than a host that was never
	// measured.
	trustStoreFactLabel = "tls trust store"
	// trustStoreMacOSLimitation is the documented limitation R-HR-04 requires the
	// output itself to state (RG-3). It is a constant rather than a phrase written
	// into one wording so that the platform limitation has exactly one home, and so
	// that the reason code and the sentence a reader sees cannot drift apart.
	trustStoreMacOSLimitation = "Go cannot enumerate macOS system roots, and keychain trust is only visible through the platform verifier when no explicit root pool is supplied"
	// trustStoreSubject names what was measured in the not-measured wordings, so a
	// reader is told which capability was missing rather than which probe missed it.
	trustStoreSubject = "the local trust store's answer for the chain presented by the declared TLS target"
)

// trustStoreOverrideVariables are the two environment variables Go's certificate loader
// reads in place of the platform trust store (crypto/x509's own root loading). They are
// named once, in declaration order, so the probe's read order is deterministic when both
// are set and so the detail can name the variable that was found rather than guessing.
var trustStoreOverrideVariables = []string{"SSL_CERT_FILE", "SSL_CERT_DIR"}

// tlsTrustStore is the `tls.truststore` probe. It carries the run's seams and the run's
// declared target input. It reads three of the run's seams: the platform (to know
// whether a verdict can be attributed to the local pool at all), the filesystem (to read
// the two documented override variables) and the TLS verifier (to perform the one
// handshake). Nothing here reads the process environment or the real machine directly.
type tlsTrustStore struct {
	seams   Seams
	targets TargetInput
}

// newTLSTrustStore builds `tls.truststore` from the run's seams and its declared target
// input. The probe's target is a declared constant in targets.go, so the input travels to
// it only so that an override of that endpoint is honoured through the one declaration.
func newTLSTrustStore(seams Seams, targets TargetInput) Probe {
	return &tlsTrustStore{seams: seams, targets: targets}
}

// Name is the probe's stable identifier, declared once in registry.go.
func (p *tlsTrustStore) Name() string { return probeNameTLSTrustStore }

// Kind is the question family the probe belongs to.
func (p *tlsTrustStore) Kind() ProbeKind { return ProbeTLS }

// now reads the run's clock, or the zero time when no clock was injected, so only an
// injected clock can move an elapsed value (design §3.3).
func (p *tlsTrustStore) now() time.Time { return runClockNow(p.seams) }

// Run performs the one measurement this probe declares and reports it as one
// observation.
func (p *tlsTrustStore) Run(ctx context.Context) Result {
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

// observe asks the probe's one question in the order of its parts: which target was
// declared, whether an environment override replaces the pool that would answer, whether
// the platform's verifier can be attributed at all, and only then whether a handshake can
// be performed. Every step that cannot proceed reports what was missing.
//
// The handshake is bounded by the caller's context alone. It is one call, and the
// runner's per-probe bound is the bound that applies to it; the verifier must honour
// that context, which its own contract requires.
func (p *tlsTrustStore) observe(ctx context.Context) Observation {
	target, unresolved, ok := declaredEndpoint(p.Name(), p.targets)
	if !ok {
		return tlsTrustStoreObserve(trustStoreFactLabel, "", unresolved)
	}
	address := target.Address()
	label := fmt.Sprintf("tls %d truststore", target.Port)

	// The environment decides whether the platform pool is the pool that would
	// answer. A run that was never given the filesystem seam cannot know whether an
	// override is set, so it must not claim a verdict about the platform pool: the
	// missing capability is reported rather than a guess about the environment.
	if p.seams.FS == nil {
		return tlsTrustStoreObserve(label, address, RawObservation{
			Kind: ObsCapabilityExcluded,
			Wording: fmt.Sprintf("tls %s: no filesystem seam is injected for this run, so the certificate-environment overrides (%s) could not be read and %s was not measured",
				address, strings.Join(trustStoreOverrideVariables, ", "), trustStoreSubject),
		})
	}
	if name, value, set := trustStoreOverride(p.seams.FS); set {
		return tlsTrustStoreObserve(label, address, RawObservation{
			Kind: ObsTrustStoreVerifierBypassed,
			Wording: fmt.Sprintf("tls %s: %s=%q is set for this run, so Go loads that pool in place of the platform trust store and the platform verifier is bypassed; the chain is reported as unresolved rather than accepted or rejected, because an answer about that pool would not be an answer about %s",
				address, name, value, trustStoreSubject),
		})
	}

	// The platform decides whether a verdict can be attributed to the local trust
	// store at all. A run that was never given the platform seam cannot identify the
	// operating system, and an operating system the seam could not identify must not
	// be assumed to be one whose pool can be enumerated: both are reported rather
	// than turned into a default guess (R-HR-29).
	if p.seams.Platform == nil {
		return tlsTrustStoreObserve(label, address, RawObservation{
			Kind: ObsCapabilityExcluded,
			Wording: fmt.Sprintf("tls %s: no platform seam is injected for this run, so the operating system could not be identified and %s was not measured; no platform is assumed",
				address, trustStoreSubject),
		})
	}
	switch goos := platformGOOS(p.seams.Platform); goos {
	case goosDarwin:
		return tlsTrustStoreObserve(label, address, RawObservation{
			Kind: ObsTrustStoreVerifierUnavailable,
			Wording: fmt.Sprintf("tls %s: %s cannot be resolved on this platform, and the documented limitation is that %s. The capability was attempted and is unusable, so the chain is reported as unresolved rather than as accepted or rejected, and no handshake is performed: an answer the platform does not let this probe attribute to a pool would not be an answer about %s (R-HR-04, RG-3)",
				address, trustStoreSubject, trustStoreMacOSLimitation, trustStoreSubject),
		})
	case unknownSignal:
		return tlsTrustStoreObserve(label, address, RawObservation{
			Kind: ObsPlatformSignalsUnknown,
			Wording: fmt.Sprintf("tls %s: the platform seam reported %q, which identifies no operating system, so whether the platform verifier can answer was not established and %s was not measured; no platform is assumed",
				address, goos, trustStoreSubject),
		})
	}

	// The capability decides whether a handshake can be attempted at all. No
	// verifier is a capability the run was never given, never a rejected chain and
	// never an accepted one.
	if p.seams.TLSVerifier == nil {
		return tlsTrustStoreObserve(label, address, RawObservation{
			Kind: ObsCapabilityExcluded,
			Wording: fmt.Sprintf("tls %s: no TLS verifier is injected for this run, so %s was not measured and nothing is claimed about it",
				address, trustStoreSubject),
		})
	}

	cfg := tlsVerifyingConfig(target.Host)
	verification, err := p.seams.TLSVerifier.Verify(ctx, address, cfg)
	return tlsTrustStoreObserve(label, address, tlsTrustStoreFact(address, verification, err))
}

// platformGOOS reads the platform seam's operating system, or the unknown value when the
// seam reported none. An empty reading is the seam's own "not reported" value rather than
// a platform: a machine the seam could not name must not be treated as one it did.
func platformGOOS(platform Platform) string {
	goos := strings.TrimSpace(platform.GOOS())
	if goos == "" {
		return unknownSignal
	}
	return goos
}

// trustStoreOverride reads the two documented override variables through the run's
// filesystem seam and reports the first one that is set, with its value, or false when
// neither is. The declaration order decides which variable is reported when both are set,
// so the result is deterministic rather than dependent on map iteration.
func trustStoreOverride(seam FS) (name, value string, set bool) {
	for _, candidate := range trustStoreOverrideVariables {
		if setting := strings.TrimSpace(seam.Getenv(candidate)); setting != "" {
			return candidate, setting, true
		}
	}
	return "", "", false
}

// tlsTrustStoreObserve classifies one local-trust-store fact through the classification
// table — never inline — and carries the platform's wording into the detail unchanged
// (R-HR-07).
func tlsTrustStoreObserve(label, address string, raw RawObservation) Observation {
	return Observe(label, address, PurposeTLSTrustStore, raw)
}

// tlsTrustStoreFact turns one handshake's result into the raw fact the classification
// table sees, for the local-trust-store question.
//
// The order of the branches is the order of the evidence, and it is the same order the
// chain probe uses for the same reason: a denial from the seam is an attempt that was not
// made and is checked first, a verification failure is a measurement of the pool and
// carries the verifier's own code, and any other error produced no answer at all. Only a
// handshake that verified is an acceptance of the chain.
//
// The observed issuer and the verification code are quoted into every definite outcome's
// detail, because the chain the pool judged is part of that measurement: without them a
// reader could not tell which chain was accepted. An empty value is reported as such
// rather than omitted, so a reader can tell "the verifier reported no code" from "the code
// was dropped".
func tlsTrustStoreFact(address string, verification TLSVerification, err error) RawObservation {
	issuer := verification.Issuer
	code := verification.VerificationCode
	if code == "" {
		code = "the verifier reported no code"
	}
	switch {
	case errors.Is(err, ErrSeamDenied):
		return RawObservation{
			Kind:    ObsCommandDenied,
			Wording: fmt.Sprintf("tls %s: %v (the verifier seam denied the handshake, so %s was not measured)", address, err, trustStoreSubject),
		}
	case errors.Is(err, ErrTLSVerification):
		return RawObservation{
			Kind: ObsTrustStoreRejectsChain,
			Wording: fmt.Sprintf("tls %s: the local trust store rejected the chain (observed issuer %q, verification code %q): %v; the pool in force on this machine is what answered, and verification was not disabled or retried to obtain a different answer",
				address, issuer, code, err),
		}
	case err != nil:
		return RawObservation{
			Kind: ObsTLSHandshakeError,
			Wording: fmt.Sprintf("tls %s: the handshake produced no answer (observed issuer %q, verification code %q): %v; neither an acceptance nor a rejection of the chain was measured, so nothing is claimed about the local trust store",
				address, issuer, code, err),
		}
	default:
		return RawObservation{
			Kind: ObsTLSVerified,
			Wording: fmt.Sprintf("tls %s: the local trust store accepted the chain (observed issuer %q, verification code %q)",
				address, issuer, code),
		}
	}
}
