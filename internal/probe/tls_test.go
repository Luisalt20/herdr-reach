package probe_test

// This file is the R-HR-04 chain suite: the `tls.interception` probe (PR 8), which asks
// one question of the declared TLS target —
//
//	which chain does this target present, and is its issuer the declared expected publisher?
//
// — and reports four outcomes: a chain that verifies with an issuer in the declared
// expected set (a measured pass), a chain that fails verification (a measured failure
// carrying the verification code in its detail), a chain whose issuer is outside the
// declared expected set (a measured failure whose wording says so and never accuses), and
// a handshake that produced no answer (unresolved). No verifier injected, or a verifier
// seam that refused, are two distinguishable not-measured facts.
//
// The cases script the verifier seam rather than a real handshake, and the verifier
// records the verifying configuration it was handed, so the two properties the spec's
// scenario asserts are checkable from outside: the configuration the probe passed is
// unchanged after the run, and the probe never retries without verification. The injected
// verifier also lets the observed issuer and the verification code be scripted, which is
// how the assertions that both appear in the result are made.

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

const (
	// testTLSInterceptionProbe is the chain probe, as the registry declares it.
	testTLSInterceptionProbe = "tls.interception"
	// testTLSTargetHost and testTLSTargetPort are the declared TLS target of PRD
	// §1.1, repeated here as literals from targets.go so a rename in the declaration
	// breaks this suite instead of silently moving what is measured.
	testTLSTargetHost = "www.cloudflare.com"
	testTLSTargetPort = 443
	// testTLSExpectedIssuer is the publisher the declared expected set holds for that
	// target (PRD §1.1: issuer `Let's Encrypt/ISRG`, verification code 0), transcribed
	// here rather than read from the probe, so a change to the declared set breaks this
	// suite instead of silently reclassifying a real chain.
	testTLSExpectedIssuer = "Let's Encrypt/ISRG"
)

// tlsConfigSnapshot is the verifying configuration as it was when the verifier saw it.
// `tls.Config` carries slices and so cannot be compared as a value; the fields that
// decide whether verification was weakened are captured instead, and compared against the
// configuration after the run so "unchanged" is asserted rather than assumed.
type tlsConfigSnapshot struct {
	serverName         string
	insecureSkipVerify bool
	rootCAsNil         bool
	minVersion         uint16
}

// snapshotTLSConfig captures the verifying configuration's decision-bearing fields.
func snapshotTLSConfig(cfg *tls.Config) tlsConfigSnapshot {
	return tlsConfigSnapshot{
		serverName:         cfg.ServerName,
		insecureSkipVerify: cfg.InsecureSkipVerify,
		rootCAsNil:         cfg.RootCAs == nil,
		minVersion:         cfg.MinVersion,
	}
}

// scriptedTLSVerifier is the TLSVerifier seam one chain case injects. It answers with the
// scripted verification and error, and records every target it was asked for, the
// configuration it was handed, and that configuration's decision-bearing fields at the
// moment of the attempt.
type scriptedTLSVerifier struct {
	// verification is what a Verify returns.
	verification probe.TLSVerification
	// err is what a Verify returns beside it.
	err error
	// calls records the targets verified, in order: more than one means the probe
	// retried, which the spec forbids.
	calls []string
	// configs records the configurations handed to the verifier, in order.
	configs []*tls.Config
	// snapshots records those configurations' fields at the moment of the attempt.
	snapshots []tlsConfigSnapshot
}

// Verify records the attempt and answers it from the case's script.
func (v *scriptedTLSVerifier) Verify(_ context.Context, target string, cfg *tls.Config) (probe.TLSVerification, error) {
	v.calls = append(v.calls, target)
	v.configs = append(v.configs, cfg)
	v.snapshots = append(v.snapshots, snapshotTLSConfig(cfg))
	return v.verification, v.err
}

// tlsSeams builds the seam set one chain case describes: the deny-all set of design §6.2
// with the verifier replaced. A nil verifier is the zero capability — the run was never
// given one — which is a different fact from a verifier that refuses, and both are cases
// of their own.
func tlsSeams(verifier probe.TLSVerifier) probe.Seams {
	seams := probe.DenyAllSeams()
	seams.TLSVerifier = verifier
	return seams
}

// runTLSInterception runs the chain probe over one scripted run, the way a run reaches
// it: through the registry, with the run's seams and its declared target input.
func runTLSInterception(t *testing.T, seams probe.Seams, targets probe.TargetInput) probe.Result {
	t.Helper()
	return egressBuild(t, testTLSInterceptionProbe, seams, targets).Run(context.Background())
}

// TestTLSInterceptionReportsEveryChainOutcome is R-HR-04's outcome table: a verified chain
// with the declared issuer is a measured pass; a verification failure is a measured
// failure carrying the verification code; an issuer outside the declared expected set is a
// measured failure whose wording says so and never accuses; any other handshake error is
// unresolved. Each case asserts the declared target, the observation triple, the aggregate,
// and the wording each outcome must carry.
func TestTLSInterceptionReportsEveryChainOutcome(t *testing.T) {
	address := declaredAddress(t, testTLSInterceptionProbe)
	wantAddress := fmt.Sprintf("%s:%d", testTLSTargetHost, testTLSTargetPort)
	if address != wantAddress {
		t.Fatalf("targets.go declares %q for %q, want the documented %q", address, testTLSInterceptionProbe, wantAddress)
	}

	verifyFailure := fmt.Errorf("%w: x509: certificate signed by unknown authority", probe.ErrTLSVerification)

	cases := []struct {
		name           string
		verifier       probe.TLSVerifier
		wantResolution probe.Resolution
		wantVerdict    probe.Verdict
		wantReason     probe.ReasonCode
		wantDetail     []string
		forbidDetail   []string
	}{
		{
			name:           "a verified chain with the declared issuer",
			verifier:       &scriptedTLSVerifier{verification: probe.TLSVerification{Issuer: testTLSExpectedIssuer, VerificationCode: "0"}},
			wantResolution: probe.Measured,
			wantVerdict:    probe.Pass,
			wantReason:     probe.ReasonOK,
			wantDetail:     []string{testTLSExpectedIssuer, "declared expected set", "0"},
		},
		{
			name: "a verification failure carries the verification code",
			verifier: &scriptedTLSVerifier{
				verification: probe.TLSVerification{Issuer: "Acme Local Root", VerificationCode: "1"},
				err:          verifyFailure,
			},
			wantResolution: probe.Measured,
			wantVerdict:    probe.Fail,
			wantReason:     probe.ReasonTLSVerifyFailed,
			wantDetail:     []string{"1", "x509: certificate signed by unknown authority"},
		},
		{
			name:           "an issuer outside the declared expected set never accuses",
			verifier:       &scriptedTLSVerifier{verification: probe.TLSVerification{Issuer: "Acme Inspection CA", VerificationCode: "0"}},
			wantResolution: probe.Measured,
			wantVerdict:    probe.Fail,
			wantReason:     probe.ReasonTLSIssuerUnexpected,
			wantDetail:     []string{"Acme Inspection CA", "not in the declared expected set", "0"},
			forbidDetail:   []string{"attacker", "malicious", "hijack", "middlebox", "man in the middle", "intercepting"},
		},
		{
			name:           "any other handshake error is unresolved",
			verifier:       &scriptedTLSVerifier{err: errors.New("tls: handshake failure")},
			wantResolution: probe.Unresolved,
			wantVerdict:    probe.Indeterminate,
			wantReason:     probe.ReasonTLSHandshakeUnresolved,
			wantDetail:     []string{"tls: handshake failure"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runTLSInterception(t, tlsSeams(tc.verifier), probe.TargetInput{})

			if result.Probe != testTLSInterceptionProbe {
				t.Fatalf("result names the probe %q, want %q", result.Probe, testTLSInterceptionProbe)
			}
			if result.Kind != probe.ProbeTLS {
				t.Fatalf("result kind = %q, want %q", result.Kind, probe.ProbeTLS)
			}
			if result.Target != address {
				t.Errorf("result target = %q, want the declared %q", result.Target, address)
			}
			if len(result.Observations) != 1 {
				t.Fatalf("the chain probe reported %d observations, want 1", len(result.Observations))
			}
			observation := result.Observations[0]
			if !holds(observation) {
				t.Fatalf("the observation does not satisfy the measurement vocabulary's invariant: %+v", observation)
			}
			if observation.Target != address {
				t.Errorf("observation target = %q, want %q", observation.Target, address)
			}
			if observation.Resolution != tc.wantResolution || observation.Verdict != tc.wantVerdict || observation.Reason != tc.wantReason {
				t.Errorf("observation = (%q, %q, %q), want (%q, %q, %q)", observation.Resolution, observation.Verdict, observation.Reason,
					tc.wantResolution, tc.wantVerdict, tc.wantReason)
			}
			if result.Verdict != tc.wantVerdict || result.Reason != tc.wantReason {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, tc.wantVerdict, tc.wantReason)
			}
			if tc.wantVerdict == probe.Fail && observation.Verdict == probe.Pass {
				t.Error("an interception candidate was reported as a pass")
			}
			for _, want := range tc.wantDetail {
				if !strings.Contains(observation.Detail, want) {
					t.Errorf("the detail does not carry %q: %q", want, observation.Detail)
				}
			}
			for _, forbidden := range tc.forbidDetail {
				if strings.Contains(strings.ToLower(observation.Detail), forbidden) {
					t.Errorf("the detail accuses with %q: %q", forbidden, observation.Detail)
				}
			}
		})
	}
}

// TestTLSInterceptionReportsTheIndeterminateControlCase covers every way the chain
// question can end without a definite answer: no verifier at all, a verifier that
// refused, a handshake error, and a verification error that names no chain problem the
// declared questions cover. None of them is a pass and none is a failure.
func TestTLSInterceptionReportsTheIndeterminateControlCase(t *testing.T) {
	cases := []struct {
		name       string
		verifier   probe.TLSVerifier
		wantRes    probe.Resolution
		wantReason probe.ReasonCode
	}{
		{
			"no capability at all",
			nil,
			probe.NotMeasured, probe.ReasonCapabilityExcluded,
		},
		{
			"a verifier seam that denies everything",
			probe.DenyAllSeams().TLSVerifier,
			probe.NotMeasured, probe.ReasonCommandDenied,
		},
		{
			"a handshake that produced no answer",
			&scriptedTLSVerifier{err: errors.New("tls: first record does not look like a TLS handshake")},
			probe.Unresolved, probe.ReasonTLSHandshakeUnresolved,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runTLSInterception(t, tlsSeams(tc.verifier), probe.TargetInput{})

			if len(result.Observations) != 1 {
				t.Fatalf("the chain probe reported %d observations, want 1", len(result.Observations))
			}
			observation := result.Observations[0]
			if observation.Resolution != tc.wantRes || observation.Verdict != probe.Indeterminate || observation.Reason != tc.wantReason {
				t.Errorf("observation = (%q, %q, %q), want (%q, %q, %q)", observation.Resolution, observation.Verdict, observation.Reason,
					tc.wantRes, probe.Indeterminate, tc.wantReason)
			}
			if observation.Verdict == probe.Pass || observation.Verdict == probe.Fail {
				t.Errorf("an unanswered chain question reported the definite verdict %q", observation.Verdict)
			}
			if result.Verdict != probe.Indeterminate || result.Reason != tc.wantReason {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Indeterminate, tc.wantReason)
			}
		})
	}
}

// TestTLSInterceptionLeavesTheVerifyingConfigurationUnchanged is the spec's second
// scenario: the configuration the probe verifies with is never weakened, is unchanged
// after the run, and is used exactly once — no unverified retry. It covers the definite
// outcomes, because a failure is where a retry-without-verification would be tempting.
func TestTLSInterceptionLeavesTheVerifyingConfigurationUnchanged(t *testing.T) {
	address := declaredAddress(t, testTLSInterceptionProbe)

	cases := []struct {
		name       string
		verifier   *scriptedTLSVerifier
		wantStatus string
	}{
		{
			name:       "a verified chain",
			verifier:   &scriptedTLSVerifier{verification: probe.TLSVerification{Issuer: testTLSExpectedIssuer, VerificationCode: "0"}},
			wantStatus: "pass",
		},
		{
			name: "a chain that failed verification",
			verifier: &scriptedTLSVerifier{
				verification: probe.TLSVerification{Issuer: "Acme Local Root", VerificationCode: "1"},
				err:          fmt.Errorf("%w: x509: certificate signed by unknown authority", probe.ErrTLSVerification),
			},
			wantStatus: "fail",
		},
		{
			name:       "a chain whose issuer is outside the declared set",
			verifier:   &scriptedTLSVerifier{verification: probe.TLSVerification{Issuer: "Acme Inspection CA", VerificationCode: "0"}},
			wantStatus: "fail",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runTLSInterception(t, tlsSeams(tc.verifier), probe.TargetInput{})

			if len(tc.verifier.calls) != 1 {
				t.Fatalf("the verifier was invoked %d times, want exactly one: an unverified retry is forbidden (R-HR-04)", len(tc.verifier.calls))
			}
			if tc.verifier.calls[0] != address {
				t.Errorf("the verifier was asked for %q, want the declared %q", tc.verifier.calls[0], address)
			}
			if len(tc.verifier.configs) != 1 {
				t.Fatalf("the verifier recorded %d configurations, want one", len(tc.verifier.configs))
			}
			cfg := tc.verifier.configs[0]
			if cfg == nil {
				t.Fatal("the probe verified with no configuration at all, so verification could not have been performed")
			}
			if cfg.InsecureSkipVerify {
				t.Error("the probe disabled certificate verification to obtain a result")
			}
			if cfg.ServerName != testTLSTargetHost {
				t.Errorf("the verifying configuration's server name = %q, want the declared host %q", cfg.ServerName, testTLSTargetHost)
			}
			if cfg.RootCAs != nil {
				t.Error("the probe supplied a root pool, so the platform verifier's answer would not be the one measured")
			}
			snapshot := tc.verifier.snapshots[0]
			if snapshot.insecureSkipVerify != cfg.InsecureSkipVerify ||
				snapshot.serverName != cfg.ServerName ||
				snapshot.rootCAsNil != (cfg.RootCAs == nil) ||
				snapshot.minVersion != cfg.MinVersion {
				t.Errorf("the verifying configuration changed after the run: %+v became server name %q, insecure %v, nil roots %v, min version %d",
					snapshot, cfg.ServerName, cfg.InsecureSkipVerify, cfg.RootCAs == nil, cfg.MinVersion)
			}
		})
	}
}

// TestTLSInterceptionNamesTheObservedIssuerAndVerificationCode is R-HR-04's own
// requirement: whichever chain was seen, the result carries the observed issuer and the
// verification code, so the detail is quotable beside the stable reason code.
func TestTLSInterceptionNamesTheObservedIssuerAndVerificationCode(t *testing.T) {
	cases := []struct {
		name         string
		verification probe.TLSVerification
		err          error
		wantReason   probe.ReasonCode
	}{
		{
			"a verified chain",
			probe.TLSVerification{Issuer: testTLSExpectedIssuer, VerificationCode: "0"},
			nil,
			probe.ReasonOK,
		},
		{
			"a rejected chain",
			probe.TLSVerification{Issuer: "Acme Local Root", VerificationCode: "12"},
			fmt.Errorf("%w: x509: certificate has expired", probe.ErrTLSVerification),
			probe.ReasonTLSVerifyFailed,
		},
		{
			"an unexpected issuer",
			probe.TLSVerification{Issuer: "Acme Inspection CA", VerificationCode: "3"},
			nil,
			probe.ReasonTLSIssuerUnexpected,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verifier := &scriptedTLSVerifier{verification: tc.verification, err: tc.err}
			result := runTLSInterception(t, tlsSeams(verifier), probe.TargetInput{})

			if result.Reason != tc.wantReason {
				t.Fatalf("result reason = %q, want %q", result.Reason, tc.wantReason)
			}
			text := resultText(result)
			if !strings.Contains(text, tc.verification.Issuer) {
				t.Errorf("the result does not carry the observed issuer %q: %q", tc.verification.Issuer, text)
			}
			if !strings.Contains(text, tc.verification.VerificationCode) {
				t.Errorf("the result does not carry the verification code %q: %q", tc.verification.VerificationCode, text)
			}
		})
	}
}
