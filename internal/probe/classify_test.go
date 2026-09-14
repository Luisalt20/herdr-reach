package probe_test

import (
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// classificationCase is one row of design §5.1: the declared question a probe
// asks, the raw fact it saw, and the triple the table must assign to it.
type classificationCase struct {
	name       string
	purpose    probe.Purpose
	observable probe.Observable
	want       probe.Classification
}

// classificationCases transcribes design §5.1 row by row, in the order the
// design prints the table. Keeping it a literal transcription is the point:
// the per-row assertion below is then checkable by eye against the design, and
// the reachability assertion is computed from the same list rather than from
// the package's own vocabulary.
var classificationCases = []classificationCase{
	{
		"tcp established", probe.PurposePortReachability, probe.ObsTCPEstablished,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Pass, Reason: probe.ReasonOK},
	},
	{
		"tcp refused", probe.PurposePortReachability, probe.ObsTCPRefused,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonConnRefused},
	},
	{
		"tcp reset after connect", probe.PurposePortReachability, probe.ObsTCPReset,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonConnReset},
	},
	{
		"reachable port serving a non-SSH banner", probe.PurposePortReachability, probe.ObsBannerNotSSH,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonBannerNotSSH},
	},
	{
		"the probe's own dial budget expired", probe.PurposePortReachability, probe.ObsProbeBudgetExpired,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonBudgetExpired},
	},
	{
		"resolver authoritative negative", probe.PurposeNameResolution, probe.ObsResolverAuthoritativeNegative,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonDNSNoSuchHost},
	},
	{
		"resolver did not answer", probe.PurposeNameResolution, probe.ObsResolverUnavailable,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonDNSUnresolved},
	},
	{
		"runner abandoned a probe that ignored its budget", probe.PurposePortReachability, probe.ObsProbeIgnoredBudget,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonProbeTimeout},
	},
	{
		"runner observed a cancelled context", probe.PurposePortReachability, probe.ObsRunCancelled,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonRunCancelled},
	},
	{
		"runner observed the global budget exhausted", probe.PurposePortReachability, probe.ObsRunBudgetExhausted,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonRunBudgetExceeded},
	},
	{
		"udp reply received", probe.PurposeUDPReachability, probe.ObsUDPResponse,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Pass, Reason: probe.ReasonUDPResponseReceived},
	},
	{
		"udp silence", probe.PurposeUDPReachability, probe.ObsUDPSilence,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonUDPSilence},
	},
	{
		"icmp port unreachable surfaced by the socket", probe.PurposeUDPReachability, probe.ObsUDPUnreachable,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonUDPUnreachable},
	},
	{
		"any other udp socket error", probe.PurposeUDPReachability, probe.ObsUDPOtherError,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonUDPErrorUnclassified},
	},
	{
		"tls verified with the expected issuer", probe.PurposeTLSCertificate, probe.ObsTLSVerified,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Pass, Reason: probe.ReasonOK},
	},
	{
		"tls verification failed", probe.PurposeTLSCertificate, probe.ObsTLSVerifyFailed,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonTLSVerifyFailed},
	},
	{
		"issuer outside the declared expected set", probe.PurposeTLSCertificate, probe.ObsTLSIssuerUnexpected,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonTLSIssuerUnexpected},
	},
	{
		"tls handshake error, neither verification nor issuer", probe.PurposeTLSCertificate, probe.ObsTLSHandshakeError,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonTLSHandshakeUnresolved},
	},
	{
		"local trust pool rejects the chain", probe.PurposeTLSTrustStore, probe.ObsTrustStoreRejectsChain,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonTrustStoreRejectsChain},
	},
	{
		"platform trust verifier cannot answer", probe.PurposeTLSTrustStore, probe.ObsTrustStoreVerifierUnavailable,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonTrustStorePlatformUnavailable},
	},
	{
		"an environment override bypasses the platform verifier", probe.PurposeTLSTrustStore, probe.ObsTrustStoreVerifierBypassed,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonTrustStoreOverridePlatformBypass},
	},
	{
		"sshd binary absent", probe.PurposeSSHDConfiguration, probe.ObsSSHDBinaryAbsent,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonSSHDAbsent},
	},
	{
		"written configuration differs from the effective one", probe.PurposeSSHDConfiguration, probe.ObsSSHDConfigDivergent,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonSSHDConfigDivergence},
	},
	{
		"capability excluded by this slice's boundary", probe.PurposeSSHDConfiguration, probe.ObsCapabilityExcluded,
		probe.Classification{Resolution: probe.NotMeasured, Verdict: probe.Indeterminate, Reason: probe.ReasonCapabilityExcluded},
	},
	{
		"command seam denied the execution", probe.PurposeSSHDConfiguration, probe.ObsCommandDenied,
		probe.Classification{Resolution: probe.NotMeasured, Verdict: probe.Indeterminate, Reason: probe.ReasonCommandDenied},
	},
	{
		"no hub input supplied", probe.PurposePortReachability, probe.ObsHubInputMissing,
		probe.Classification{Resolution: probe.NotMeasured, Verdict: probe.Indeterminate, Reason: probe.ReasonInputMissingHub},
	},
	{
		"platform signals match no supported classification", probe.PurposePlatformClassification, probe.ObsPlatformSignalsUnknown,
		probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonPlatformUnknown},
	},
	{
		"classified native Windows node", probe.PurposePlatformClassification, probe.ObsNodePlatformUnsupported,
		probe.Classification{Resolution: probe.Measured, Verdict: probe.Fail, Reason: probe.ReasonNodePlatformUnsupported},
	},
}

// TestClassificationTable asserts the table's triple for every row of design
// §5.1, and asserts that each triple satisfies the measurement vocabulary's
// invariant. A row whose resolution and verdict disagree would make the
// vocabulary unable to express what the table decided.
func TestClassificationTable(t *testing.T) {
	for _, tc := range classificationCases {
		t.Run(tc.name, func(t *testing.T) {
			got := probe.Classify(tc.purpose, probe.RawObservation{Kind: tc.observable, Wording: "verbatim wording"})
			if got.Resolution != tc.want.Resolution || got.Verdict != tc.want.Verdict || got.Reason != tc.want.Reason {
				t.Fatalf("Classify(%q, %q) = (%q, %q, %q), want (%q, %q, %q)",
					tc.purpose, tc.observable,
					got.Resolution, got.Verdict, got.Reason,
					tc.want.Resolution, tc.want.Verdict, tc.want.Reason)
			}

			observed := probe.Observe("label", "203.0.113.10:22", tc.purpose, probe.RawObservation{Kind: tc.observable, Wording: "verbatim wording"})
			if !holds(observed) {
				t.Fatalf("table row %q produced an observation the vocabulary cannot express: %+v", tc.name, observed)
			}
			if observed.Resolution != got.Resolution || observed.Verdict != got.Verdict || observed.Reason != got.Reason {
				t.Fatalf("Observe(%q) disagrees with Classify: %+v vs %+v", tc.name, observed, got)
			}
		})
	}
}

// TestEveryReasonCodeIsReachable asserts that the closed set and the
// classification table describe the same vocabulary: every declared code is
// what some documented observation classifies to, and the table never produces
// a code outside the set.
func TestEveryReasonCodeIsReachable(t *testing.T) {
	reachable := make(map[probe.ReasonCode]string, len(classificationCases)+1)
	for _, tc := range classificationCases {
		reachable[tc.want.Reason] = tc.name
	}

	// The final row of design §5.1 is the table's total row: a fact no row
	// recognises is reported as an internal failure rather than guessed at.
	unknown := probe.Observe("label", "", probe.PurposePortReachability, probe.RawObservation{Kind: probe.Observable("unrecognised_fact")})
	reachable[unknown.Reason] = "unrecognised observation"

	closed := make(map[probe.ReasonCode]bool, len(probe.AllReasonCodes()))
	for _, code := range probe.AllReasonCodes() {
		closed[code] = true
		if _, ok := reachable[code]; !ok {
			t.Errorf("reason code %q is declared but no classification row reaches it", code)
		}
	}
	for code, where := range reachable {
		if !closed[code] {
			t.Errorf("classification row %q produces %q, which is not in the closed set", where, code)
		}
	}
}

// TestUnclassifiableObservationIsNeverAPass is the table's control case. A
// fact the table cannot place, a purpose and fact combination it does not
// document, and the platform case where the signals match nothing all resolve
// to an explicit internal failure or platform-unknown outcome — never a pass
// and never a claimed block.
func TestUnclassifiableObservationIsNeverAPass(t *testing.T) {
	cases := []struct {
		name    string
		purpose probe.Purpose
		kind    probe.Observable
		want    probe.Classification
	}{
		{
			"a fact no row recognises",
			probe.PurposePortReachability,
			probe.Observable("unrecognised_fact"),
			probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonInternalError},
		},
		{
			"the documented internal-failure fact",
			probe.PurposeUDPReachability,
			probe.ObsInternalFailure,
			probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonInternalError},
		},
		{
			"platform signals matching no supported classification",
			probe.PurposePlatformClassification,
			probe.ObsPlatformSignalsUnknown,
			probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonPlatformUnknown},
		},
		{
			// A dial budget expiry is a definite negative only because the
			// declared question is "is this port reachable?". Asked by a probe
			// whose question is something else, the same fact is not documented
			// and must not be borrowed as a failure.
			"a dial budget expiry on a question that is not about reachability",
			probe.PurposeUDPReachability,
			probe.ObsProbeBudgetExpired,
			probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonInternalError},
		},
		{
			"a fact that belongs to a different declared question",
			probe.PurposePortReachability,
			probe.ObsTrustStoreRejectsChain,
			probe.Classification{Resolution: probe.Unresolved, Verdict: probe.Indeterminate, Reason: probe.ReasonInternalError},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := probe.Classify(tc.purpose, probe.RawObservation{Kind: tc.kind, Wording: "verbatim wording"})
			if got != tc.want {
				t.Fatalf("Classify(%q, %q) = %+v, want %+v", tc.purpose, tc.kind, got, tc.want)
			}
			if got.Verdict == probe.Pass || got.Verdict == probe.Fail {
				t.Fatalf("unclassifiable observation produced the definite verdict %q", got.Verdict)
			}
			if got.Resolution == probe.Measured {
				t.Fatal("unclassifiable observation was reported as measured")
			}
			if observed := probe.Observe("label", "", tc.purpose, probe.RawObservation{Kind: tc.kind, Wording: "verbatim wording"}); !holds(observed) {
				t.Fatalf("Observe produced an observation the vocabulary cannot express: %+v", observed)
			}
		})
	}
}

// TestOneObservableNeverClassifiesTwoWays asserts the property the single
// ordered table exists for: classification is a pure function of the declared
// purpose and the observed fact, so two probes asking the same declared
// question cannot report different codes for the same fact, and no repeated
// call can drift.
func TestOneObservableNeverClassifiesTwoWays(t *testing.T) {
	// The two SSH destination probes of PRD §5.1 declare the same question.
	raw := probe.RawObservation{Kind: probe.ObsTCPRefused, Wording: "connect: connection refused"}
	sshKnown := probe.Observe("banner", "github.com:22", probe.PurposePortReachability, raw)
	ssh443 := probe.Observe("banner", "ssh.github.com:443", probe.PurposePortReachability, raw)
	if sshKnown.Reason != ssh443.Reason || sshKnown.Resolution != ssh443.Resolution || sshKnown.Verdict != ssh443.Verdict {
		t.Fatalf("one fact, one declared question, two classifications: %+v vs %+v", sshKnown, ssh443)
	}

	for _, tc := range classificationCases {
		t.Run(tc.name, func(t *testing.T) {
			first := probe.Classify(tc.purpose, probe.RawObservation{Kind: tc.observable, Wording: "first wording"})
			second := probe.Classify(tc.purpose, probe.RawObservation{Kind: tc.observable, Wording: "second wording"})
			if first != second {
				t.Fatalf("repeated classification drifted: %+v vs %+v", first, second)
			}
			again := probe.Observe("label", "target", tc.purpose, probe.RawObservation{Kind: tc.observable, Wording: "first wording"})
			if again.Resolution != first.Resolution || again.Verdict != first.Verdict || again.Reason != first.Reason {
				t.Fatalf("Observe and Classify disagree: %+v vs %+v", again, first)
			}
		})
	}
}

// TestWordingDrift asserts R-HR-07's scenario: two observations that differ
// only in the operating system's wording of the same failure keep an identical
// reason code, while both verbatim wordings stay available as the detail.
func TestWordingDrift(t *testing.T) {
	cases := []struct {
		name       string
		purpose    probe.Purpose
		observable probe.Observable
		first      string
		second     string
		wantReason probe.ReasonCode
	}{
		{
			name:       "a refused connection, phrased by two systems",
			purpose:    probe.PurposePortReachability,
			observable: probe.ObsTCPRefused,
			first:      "dial tcp 203.0.113.10:22: connect: connection refused",
			second:     "dial tcp 203.0.113.10:22: connect: Connection refused",
			wantReason: probe.ReasonConnRefused,
		},
		{
			name:       "an authoritative resolver negative, phrased by two resolvers",
			purpose:    probe.PurposeNameResolution,
			observable: probe.ObsResolverAuthoritativeNegative,
			first:      "lookup hub.invalid: no such host",
			second:     "lookup hub.invalid: Name or service not known",
			wantReason: probe.ReasonDNSNoSuchHost,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first := probe.Observe("tcp 22", "203.0.113.10:22", tc.purpose, probe.RawObservation{Kind: tc.observable, Wording: tc.first})
			second := probe.Observe("tcp 22", "203.0.113.10:22", tc.purpose, probe.RawObservation{Kind: tc.observable, Wording: tc.second})

			if first.Reason != tc.wantReason || second.Reason != tc.wantReason {
				t.Fatalf("reason codes = (%q, %q), want both %q", first.Reason, second.Reason, tc.wantReason)
			}
			if first.Reason != second.Reason {
				t.Fatalf("wording moved the reason code: %q vs %q", first.Reason, second.Reason)
			}
			if first.Resolution != second.Resolution || first.Verdict != second.Verdict {
				t.Fatalf("wording moved the classification: (%q, %q) vs (%q, %q)",
					first.Resolution, first.Verdict, second.Resolution, second.Verdict)
			}
			if first.Detail == second.Detail {
				t.Fatalf("verbatim details collapsed to one string: %q", first.Detail)
			}
			if first.Detail != tc.first || second.Detail != tc.second {
				t.Fatalf("details = (%q, %q), want (%q, %q)", first.Detail, second.Detail, tc.first, tc.second)
			}
		})
	}
}
