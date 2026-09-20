package probe_test

// This file is the R-HR-04 chain suite: the `tls.interception` probe (PR 8), which asks
// one question of the declared TLS target —
//
//	which chain does this target present, and is its issuer the declared expected publisher?
//
// — and reports four outcomes: a chain that verifies with an issuer in the declared
// expected set (a measured pass), a chain that fails verification (a measured failure
// carrying the verification code in its detail), a chain whose issuer is outside the
// declared expected set (unresolved: the wording records the observed publisher and the
// declared set, and states that the run cannot distinguish a publisher change from an
// interception, so it accuses no one and dismisses nothing), and a handshake that produced
// no answer (unresolved). No verifier injected, or a verifier seam that refused, are two
// distinguishable not-measured facts.
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
	"os"
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
// failure carrying the verification code; an issuer outside the declared expected set is
// unresolved — the wording records the divergence and states that the run cannot tell a
// publisher change from an interception, and never accuses; any other handshake error is
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
			name:           "an issuer outside the declared expected set is unresolved, never a failure and never an accusation",
			verifier:       &scriptedTLSVerifier{verification: probe.TLSVerification{Issuer: "Acme Inspection CA", VerificationCode: "0"}},
			wantResolution: probe.Unresolved,
			wantVerdict:    probe.Indeterminate,
			wantReason:     probe.ReasonTLSIssuerUnexpected,
			wantDetail:     []string{"Acme Inspection CA", "not in the declared expected set", "0", "cannot distinguish a publisher change from an interception", "unresolved"},
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
// after the run, and is used exactly once — no unverified retry. It covers the pass, the
// verification failure and the recorded publisher divergence, because a result that is not
// a pass is where a retry-without-verification would be tempting.
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
			wantStatus: "unresolved",
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

// ---------------------------------------------------------------------------
// PR 9 — `tls.truststore`: the chain against the local trust store (R-HR-04, RG-3)
// ---------------------------------------------------------------------------
//
// The cases below extend this file with the second TLS question of PRD §5.1: does the
// local trust store accept the chain the declared TLS target presents? It is the same
// declared target as `tls.interception` and the same handshake seam, but a different
// question: the chain probe asks what the peer presented, and this probe asks what this
// machine's own pool does with it.
//
// Two honest degradations are the reason the probe exists in this shape, and both are
// properties of the *result*, not of a comment:
//
//   - on macOS Go cannot enumerate the system roots, so the probe reports unresolved and
//     the documented limitation travels in the result detail (RG-3); and
//   - a certificate-file or certificate-directory environment override replaces the
//     platform pool, so the probe reports unresolved rather than claiming a pass about a
//     pool this run did not use.
//
// The test names follow the focused filter the task plan documents for this slice
// (`-run TestTLSTruststore`) so the documented command selects them; the probe's own
// identifiers keep Go's spelling (`tlsTrustStore`).

const (
	// testTLSTrustStoreProbe is the local-trust-store probe, as the registry declares it.
	testTLSTrustStoreProbe = "tls.truststore"
	// testTLSTrustStoreLinux and testTLSTrustStoreMacOS are the two platform seam values
	// the cases script. They are Go's own operating-system vocabulary, which is what the
	// Platform seam reports.
	testTLSTrustStoreLinux = "linux"
	testTLSTrustStoreMacOS = "darwin"
)

// scriptedEnv is the FS seam a trust-store case injects: an environment holding exactly
// the variables the case names, and a tripwire for every other read. It records every
// call, so a case can assert that the probe read the two documented override variables
// and nothing else — the difference between "the seam answered nothing" and "the seam
// was never asked".
type scriptedEnv struct {
	// env maps each variable the case sets to its value; an absent name reads as "".
	env map[string]string
	// calls records every seam call in order.
	calls []string
}

// ReadFile records the attempt and refuses it: the local trust store is not a file the
// probe reads, so a read here would be a measurement outside the declared question.
func (f *scriptedEnv) ReadFile(path string) ([]byte, error) {
	f.calls = append(f.calls, "ReadFile "+path)
	return nil, fmt.Errorf("tripwire filesystem: the trust-store probe must read no file, but asked for %s", path)
}

// Stat records the attempt and refuses it, for the same reason.
func (f *scriptedEnv) Stat(path string) (os.FileInfo, error) {
	f.calls = append(f.calls, "Stat "+path)
	return nil, fmt.Errorf("tripwire filesystem: the trust-store probe must stat no path, but asked for %s", path)
}

// Getenv records the lookup and answers from the case's environment.
func (f *scriptedEnv) Getenv(name string) string {
	f.calls = append(f.calls, "Getenv "+name)
	return f.env[name]
}

// trustStoreSeams builds the seam set one trust-store case describes: the deny-all set of
// design §6.2 with the verifier, the platform and the environment replaced. A nil value
// is the zero capability — the run was never given one — which is a different fact from a
// seam that answers or refuses, and each is a case of its own.
func trustStoreSeams(verifier probe.TLSVerifier, platform probe.Platform, environment probe.FS) probe.Seams {
	seams := probe.DenyAllSeams()
	seams.TLSVerifier = verifier
	seams.Platform = platform
	seams.FS = environment
	return seams
}

// runTLSTruststore runs the local-trust-store probe over one scripted run, the way a run
// reaches it: through the registry, with the run's seams and its declared target input.
func runTLSTruststore(t *testing.T, seams probe.Seams, targets probe.TargetInput) probe.Result {
	t.Helper()
	return egressBuild(t, testTLSTrustStoreProbe, seams, targets).Run(context.Background())
}

// TestTLSTruststoreReportsTheLinuxControls is R-HR-04's first trust-store scenario: on a
// platform whose verifier can answer, an accepting pool is a measured pass and a rejected
// chain is a measured failure. Both cases script the verifier's answer, and both assert
// the properties that make the answer attributable to *this machine's* pool: the
// configuration the probe verified with names the declared host, leaves verification on
// and supplies no root pool of its own, and the environment was asked only about the two
// documented override variables.
func TestTLSTruststoreReportsTheLinuxControls(t *testing.T) {
	address := declaredAddress(t, testTLSTrustStoreProbe)
	wantAddress := fmt.Sprintf("%s:%d", testTLSTargetHost, testTLSTargetPort)
	if address != wantAddress {
		t.Fatalf("targets.go declares %q for %q, want the documented %q", address, testTLSTrustStoreProbe, wantAddress)
	}

	rejection := fmt.Errorf("%w: x509: certificate signed by unknown authority", probe.ErrTLSVerification)

	cases := []struct {
		name           string
		verifier       *scriptedTLSVerifier
		wantResolution probe.Resolution
		wantVerdict    probe.Verdict
		wantReason     probe.ReasonCode
		wantDetail     []string
	}{
		{
			name:           "a trust pool that accepts the chain",
			verifier:       &scriptedTLSVerifier{verification: probe.TLSVerification{Issuer: testTLSExpectedIssuer, VerificationCode: "0"}},
			wantResolution: probe.Measured,
			wantVerdict:    probe.Pass,
			wantReason:     probe.ReasonOK,
			wantDetail:     []string{testTLSExpectedIssuer, "0", "accepted"},
		},
		{
			name: "a trust pool that rejects the chain",
			verifier: &scriptedTLSVerifier{
				verification: probe.TLSVerification{Issuer: "Acme Local Root", VerificationCode: "1"},
				err:          rejection,
			},
			wantResolution: probe.Measured,
			wantVerdict:    probe.Fail,
			wantReason:     probe.ReasonTrustStoreRejectsChain,
			wantDetail:     []string{"x509: certificate signed by unknown authority", "1", "rejected"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			platform := scriptedPlatform{goos: testTLSTrustStoreLinux, arch: "amd64"}
			environment := &scriptedEnv{}
			result := runTLSTruststore(t, trustStoreSeams(tc.verifier, platform, environment), probe.TargetInput{})

			if result.Probe != testTLSTrustStoreProbe {
				t.Fatalf("result names the probe %q, want %q", result.Probe, testTLSTrustStoreProbe)
			}
			if result.Kind != probe.ProbeTLS {
				t.Fatalf("result kind = %q, want %q", result.Kind, probe.ProbeTLS)
			}
			if result.Target != address {
				t.Errorf("result target = %q, want the declared %q", result.Target, address)
			}
			if len(result.Observations) != 1 {
				t.Fatalf("the trust-store probe reported %d observations, want 1", len(result.Observations))
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
			for _, want := range tc.wantDetail {
				if !strings.Contains(observation.Detail, want) {
					t.Errorf("the detail does not carry %q: %q", want, observation.Detail)
				}
			}

			// The answer is attributable to this machine's pool only if the probe
			// verified once, with the declared host, verification on and no root pool
			// of its own — the platform verifier is what must answer.
			if len(tc.verifier.calls) != 1 {
				t.Fatalf("the verifier was invoked %d times, want exactly one", len(tc.verifier.calls))
			}
			if tc.verifier.calls[0] != address {
				t.Errorf("the verifier was asked for %q, want the declared %q", tc.verifier.calls[0], address)
			}
			if len(tc.verifier.configs) != 1 {
				t.Fatalf("the verifier recorded %d configurations, want one", len(tc.verifier.configs))
			}
			cfg := tc.verifier.configs[0]
			if cfg == nil {
				t.Fatal("the probe verified with no configuration at all, so the pool in force could not be identified")
			}
			if cfg.InsecureSkipVerify {
				t.Error("the probe disabled certificate verification to obtain a result")
			}
			if cfg.ServerName != testTLSTargetHost {
				t.Errorf("the verifying configuration's server name = %q, want the declared host %q", cfg.ServerName, testTLSTargetHost)
			}
			if cfg.RootCAs != nil {
				t.Error("the probe supplied a root pool, so the local trust store would not be the pool that answered")
			}

			// The environment seam was asked about the two documented overrides and
			// nothing else: any other variable would be an undocumented input to the
			// classification.
			for _, call := range environment.calls {
				if call != "Getenv SSL_CERT_FILE" && call != "Getenv SSL_CERT_DIR" {
					t.Errorf("the probe read an undocumented environment input: %q", call)
				}
			}
		})
	}
}

// TestTLSTruststoreNeverClaimsWhenThePoolInForceIsNotTheLocalStore is RG-3's own
// property case: the two conditions that make a verdict unattributable — the macOS
// platform verifier and a certificate-environment override — are reported as unresolved
// with the reason that says which one it is, the documented macOS limitation travels in
// the result itself rather than only in a comment, and no case can come out as a pass.
// The scripted verifier would answer "verified, expected issuer" if it were consulted, so
// a pass here would be exactly the false pass RG-3 forbids.
func TestTLSTruststoreNeverClaimsWhenThePoolInForceIsNotTheLocalStore(t *testing.T) {
	address := declaredAddress(t, testTLSTrustStoreProbe)

	cases := []struct {
		name        string
		platform    probe.Platform
		environment probe.FS
		wantReason  probe.ReasonCode
		wantDetail  []string
	}{
		{
			name:        "macOS: the documented system-roots limitation",
			platform:    scriptedPlatform{goos: testTLSTrustStoreMacOS, arch: "arm64"},
			environment: &scriptedEnv{},
			wantReason:  probe.ReasonTrustStorePlatformUnavailable,
			wantDetail: []string{
				"cannot enumerate macOS system roots",
				"keychain trust is only visible through the platform verifier",
				"no explicit root pool",
				"unresolved",
			},
		},
		{
			name:        "a certificate-file override bypasses the platform verifier",
			platform:    scriptedPlatform{goos: testTLSTrustStoreLinux, arch: "amd64"},
			environment: &scriptedEnv{env: map[string]string{"SSL_CERT_FILE": "/etc/herdr-reach/local-ca.pem"}},
			wantReason:  probe.ReasonTrustStoreOverridePlatformBypass,
			wantDetail:  []string{"SSL_CERT_FILE", "/etc/herdr-reach/local-ca.pem", "bypass", "unresolved"},
		},
		{
			name:        "a certificate-directory override bypasses the platform verifier",
			platform:    scriptedPlatform{goos: testTLSTrustStoreLinux, arch: "amd64"},
			environment: &scriptedEnv{env: map[string]string{"SSL_CERT_DIR": "/etc/ssl/certs.d"}},
			wantReason:  probe.ReasonTrustStoreOverridePlatformBypass,
			wantDetail:  []string{"SSL_CERT_DIR", "/etc/ssl/certs.d", "bypass", "unresolved"},
		},
		{
			name:        "both overrides set: the file override is the one reported",
			platform:    scriptedPlatform{goos: testTLSTrustStoreLinux, arch: "amd64"},
			environment: &scriptedEnv{env: map[string]string{"SSL_CERT_FILE": "/etc/herdr-reach/local-ca.pem", "SSL_CERT_DIR": "/etc/ssl/certs.d"}},
			wantReason:  probe.ReasonTrustStoreOverridePlatformBypass,
			wantDetail:  []string{"SSL_CERT_FILE", "/etc/herdr-reach/local-ca.pem", "bypass"},
		},
		{
			name:        "an override on macOS still reports the bypass",
			platform:    scriptedPlatform{goos: testTLSTrustStoreMacOS, arch: "arm64"},
			environment: &scriptedEnv{env: map[string]string{"SSL_CERT_FILE": "/etc/herdr-reach/local-ca.pem"}},
			wantReason:  probe.ReasonTrustStoreOverridePlatformBypass,
			wantDetail:  []string{"SSL_CERT_FILE", "bypass"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The verifier answers "verified, expected issuer": if the probe
			// consulted it, the case below would be a pass.
			verifier := &scriptedTLSVerifier{verification: probe.TLSVerification{Issuer: testTLSExpectedIssuer, VerificationCode: "0"}}
			result := runTLSTruststore(t, trustStoreSeams(verifier, tc.platform, tc.environment), probe.TargetInput{})

			if len(result.Observations) != 1 {
				t.Fatalf("the trust-store probe reported %d observations, want 1", len(result.Observations))
			}
			observation := result.Observations[0]
			if observation.Resolution != probe.Unresolved || observation.Verdict != probe.Indeterminate {
				t.Errorf("observation = (%q, %q), want (%q, %q)", observation.Resolution, observation.Verdict, probe.Unresolved, probe.Indeterminate)
			}
			if observation.Reason != tc.wantReason {
				t.Errorf("observation reason = %q, want %q", observation.Reason, tc.wantReason)
			}
			if observation.Verdict == probe.Pass {
				t.Error("an unattributable trust-store answer was reported as a pass")
			}
			if observation.Verdict == probe.Fail {
				t.Error("an unattributable trust-store answer was reported as a rejection of the chain")
			}
			if result.Verdict != probe.Indeterminate || result.Reason != tc.wantReason {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Indeterminate, tc.wantReason)
			}

			// RG-3: the limitation is in the result the run produces, not only in
			// the probe's reason code or in a comment beside it.
			for _, want := range tc.wantDetail {
				if !strings.Contains(result.Detail, want) {
					t.Errorf("the result's own detail does not carry %q: %q", want, result.Detail)
				}
			}
			if !strings.Contains(resultText(result), "tls "+address) {
				t.Errorf("the result does not name the declared target %q: %q", address, resultText(result))
			}

			// The pool that would have answered is not the pool the probe declares,
			// so no handshake is performed: consulting the verifier would measure a
			// different question and its answer would have to be discarded.
			if len(verifier.calls) != 0 {
				t.Errorf("the verifier was invoked %d times for an unattributable measurement, want none: %v", len(verifier.calls), verifier.calls)
			}
		})
	}
}

// TestTLSTruststoreReportsTheIndeterminateControlCase covers every remaining way the
// trust-store question can end without a verdict: a capability the run was never given
// (the verifier, the environment, the platform), a platform the seam could not identify,
// a verifier seam that refused, and a handshake that produced no answer. None of them is
// a pass and none is a rejection of the chain.
func TestTLSTruststoreReportsTheIndeterminateControlCase(t *testing.T) {
	linux := scriptedPlatform{goos: testTLSTrustStoreLinux, arch: "amd64"}
	verified := &scriptedTLSVerifier{verification: probe.TLSVerification{Issuer: testTLSExpectedIssuer, VerificationCode: "0"}}

	cases := []struct {
		name           string
		verifier       probe.TLSVerifier
		platform       probe.Platform
		environment    probe.FS
		wantResolution probe.Resolution
		wantReason     probe.ReasonCode
	}{
		{
			"no verifier capability at all",
			nil, linux, &scriptedEnv{},
			probe.NotMeasured, probe.ReasonCapabilityExcluded,
		},
		{
			"a verifier seam that denies everything",
			probe.DenyAllSeams().TLSVerifier, linux, &scriptedEnv{},
			probe.NotMeasured, probe.ReasonCommandDenied,
		},
		{
			"no environment capability at all",
			verified, linux, nil,
			probe.NotMeasured, probe.ReasonCapabilityExcluded,
		},
		{
			"no platform capability at all",
			verified, nil, &scriptedEnv{},
			probe.NotMeasured, probe.ReasonCapabilityExcluded,
		},
		{
			"a platform the seam could not identify",
			verified, scriptedPlatform{goos: "unknown", arch: "amd64"}, &scriptedEnv{},
			probe.Unresolved, probe.ReasonPlatformUnknown,
		},
		{
			"a platform the seam did not report at all",
			verified, scriptedPlatform{goos: "", arch: "amd64"}, &scriptedEnv{},
			probe.Unresolved, probe.ReasonPlatformUnknown,
		},
		{
			"a handshake that produced no answer",
			&scriptedTLSVerifier{err: errors.New("tls: first record does not look like a TLS handshake")}, linux, &scriptedEnv{},
			probe.Unresolved, probe.ReasonTLSHandshakeUnresolved,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runTLSTruststore(t, trustStoreSeams(tc.verifier, tc.platform, tc.environment), probe.TargetInput{})

			if len(result.Observations) != 1 {
				t.Fatalf("the trust-store probe reported %d observations, want 1", len(result.Observations))
			}
			observation := result.Observations[0]
			if observation.Resolution != tc.wantResolution || observation.Verdict != probe.Indeterminate || observation.Reason != tc.wantReason {
				t.Errorf("observation = (%q, %q, %q), want (%q, %q, %q)", observation.Resolution, observation.Verdict, observation.Reason,
					tc.wantResolution, probe.Indeterminate, tc.wantReason)
			}
			if observation.Verdict == probe.Pass || observation.Verdict == probe.Fail {
				t.Errorf("an unanswered trust-store question reported the definite verdict %q", observation.Verdict)
			}
			if result.Verdict != probe.Indeterminate || result.Reason != tc.wantReason {
				t.Errorf("result = (%q, %q), want (%q, %q)", result.Verdict, result.Reason, probe.Indeterminate, tc.wantReason)
			}
		})
	}
}

// TestTLSTruststoreClosesTheTenProbeRegistry is the gate the task plan names for this
// slice: `tls.truststore` is the tenth and last probe, so after it the registry builds
// every declared probe and no slot is left without a constructor. A registry that still
// skipped an entry would be a run that silently measures nine probes and reports ten.
func TestTLSTruststoreClosesTheTenProbeRegistry(t *testing.T) {
	registry := probe.Registry()
	if len(registry) != 10 {
		t.Fatalf("the registry holds %d probes, want the ten of PRD §5.1", len(registry))
	}

	var unbuilt []string
	for _, entry := range registry {
		if entry.New == nil {
			unbuilt = append(unbuilt, entry.Name)
		}
	}
	if len(unbuilt) != 0 {
		t.Fatalf("the registry declares %d probes without a constructor: %v", len(unbuilt), unbuilt)
	}

	built := probe.Probes(probe.DenyAllSeams())
	if len(built) != len(registry) {
		t.Fatalf("the registry built %d probes, its entries promise %d", len(built), len(registry))
	}

	found := false
	for _, builtProbe := range built {
		if builtProbe.Name() != testTLSTrustStoreProbe {
			continue
		}
		found = true
		if builtProbe.Kind() != probe.ProbeTLS {
			t.Errorf("%s reports kind %q, want %q", testTLSTrustStoreProbe, builtProbe.Kind(), probe.ProbeTLS)
		}
	}
	if !found {
		t.Fatalf("the ten built probes do not include %q, the last probe of PRD §5.1", testTLSTrustStoreProbe)
	}
}
