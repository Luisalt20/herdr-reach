// Package probe_test exercises the measurement vocabulary from outside the
// package: the invariants every Observation must satisfy, the aggregate order
// every Result's verdict is reduced through, and later the closed reason-code
// set. Keeping the tests external proves the exported vocabulary is enough to
// state the design's invariants without reaching into the package.
package probe_test

import (
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// holds restates the design §3.1 invariants that every observation must
// satisfy: a definite verdict is allowed only on a measured observation, and
// the two non-measured resolutions are always indeterminate. It is deliberately
// written here rather than exported by the package, so the tests state the
// contract instead of merely reusing an implementation of it.
func holds(o probe.Observation) bool {
	if o.Resolution == probe.Measured {
		return o.Verdict == probe.Pass || o.Verdict == probe.Fail
	}
	return o.Verdict == probe.Indeterminate
}

// observation builds the observation one case is about. Label, target and
// detail are realistic on purpose: the fixtures should read like measurements
// rather than like anonymous structs.
func observation(resolution probe.Resolution, verdict probe.Verdict, reason probe.ReasonCode) probe.Observation {
	return probe.Observation{
		Label:      "tcp 22",
		Target:     "203.0.113.10:22",
		Resolution: resolution,
		Verdict:    verdict,
		Reason:     reason,
		Detail:     "dial tcp 203.0.113.10:22: i/o timeout after 4s",
	}
}

// TestResultInvariants enumerates the whole resolution × verdict matrix and
// pins which cells the vocabulary allows. A measurement that cannot be
// expressed as one of the four allowed cells is a vocabulary bug, so the
// invalid cells are asserted to be invalid rather than merely skipped.
func TestResultInvariants(t *testing.T) {
	cases := []struct {
		name       string
		resolution probe.Resolution
		verdict    probe.Verdict
		wantValid  bool
	}{
		{"measured pass", probe.Measured, probe.Pass, true},
		{"measured fail", probe.Measured, probe.Fail, true},
		{"measured indeterminate", probe.Measured, probe.Indeterminate, false},
		{"unresolved pass", probe.Unresolved, probe.Pass, false},
		{"unresolved fail", probe.Unresolved, probe.Fail, false},
		{"unresolved indeterminate", probe.Unresolved, probe.Indeterminate, true},
		{"not measured pass", probe.NotMeasured, probe.Pass, false},
		{"not measured fail", probe.NotMeasured, probe.Fail, false},
		{"not measured indeterminate", probe.NotMeasured, probe.Indeterminate, true},
	}

	valid := 0
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := observation(tc.resolution, tc.verdict, probe.ReasonOK)
			if got := holds(o); got != tc.wantValid {
				t.Fatalf("invariant over %s = %v, want %v", tc.name, got, tc.wantValid)
			}
			if !tc.wantValid {
				return
			}
			// A single observation must survive the reduction unchanged:
			// Aggregate may not invent a verdict the observation did not carry.
			gotVerdict, _ := probe.Aggregate([]probe.Observation{o})
			if gotVerdict != tc.verdict {
				t.Fatalf("Aggregate over %s = %q, want %q", tc.name, gotVerdict, tc.verdict)
			}
		})
		if tc.wantValid {
			valid++
		}
	}

	// The matrix above is exhaustive, so exactly four of its nine cells can be
	// valid. A change that quietly adds a fifth cell changes the vocabulary.
	if valid != 4 {
		t.Errorf("valid cells = %d, want 4 (the matrix is exhaustive)", valid)
	}
}

// documentedReasonCodes is the closed set of design §3.5, transcribed in the
// order the design prints it. It is deliberately a literal list rather than a
// reference to the package's own vocabulary: the point of the case is to
// detect drift between the documented set and the implementation.
var documentedReasonCodes = []probe.ReasonCode{
	"ok",
	"conn_refused",
	"conn_reset",
	"budget_expired",
	"dns_no_such_host",
	"dns_unresolved",
	"banner_not_ssh",
	"probe_timeout",
	"run_cancelled",
	"run_budget_exceeded",
	"udp_response_received",
	"udp_silence",
	"udp_unreachable",
	"udp_error_unclassified",
	"tls_verify_failed",
	"tls_issuer_unexpected",
	"tls_handshake_unresolved",
	"truststore_rejects_chain",
	"truststore_platform_unavailable",
	"truststore_override_platform_bypass",
	"sshd_absent",
	"sshd_config_divergence",
	"capability_excluded",
	"command_denied",
	"input_missing_hub",
	"platform_unknown",
	"node_platform_unsupported",
	"internal_error",
}

// TestReasonCodeSetIsClosed asserts that the reason-code set is exactly the
// documented one, is free of duplicates, and is returned in declaration order.
// Closure matters because both projections quote the code verbatim: an
// undocumented code would reach a consumer that has no contract for it.
func TestReasonCodeSetIsClosed(t *testing.T) {
	got := probe.AllReasonCodes()

	if len(got) != len(documentedReasonCodes) {
		t.Fatalf("AllReasonCodes() has %d codes, the documented set has %d", len(got), len(documentedReasonCodes))
	}
	// Closure, uniqueness and order in one pass: a missing, extra or reordered
	// code all fail here.
	for i, want := range documentedReasonCodes {
		if got[i] != want {
			t.Errorf("AllReasonCodes()[%d] = %q, want %q", i, got[i], want)
		}
	}

	seen := make(map[probe.ReasonCode]bool, len(got))
	for i, code := range got {
		if code == "" {
			t.Errorf("AllReasonCodes()[%d] is empty", i)
		}
		if seen[code] {
			t.Errorf("AllReasonCodes()[%d] = %q is a duplicate", i, code)
		}
		seen[code] = true
	}

	// The count is pinned to the enumeration rather than to the design heading:
	// design §3.5's heading says "(27)" while the set it prints contains 28
	// distinct codes (`ok` is the only code two §5.1 rows share). The
	// enumeration wins, because every one of the 28 is needed by a §5.1 row.
	if len(documentedReasonCodes) != 28 {
		t.Errorf("documented set has %d codes, want the 28 enumerated in design §3.5", len(documentedReasonCodes))
	}

	// The closed set is a copy: a caller cannot reorder or shrink the contract.
	got[0] = "mutated"
	if probe.AllReasonCodes()[0] != documentedReasonCodes[0] {
		t.Errorf("AllReasonCodes() handed out the package's own slice")
	}
}

// TestAggregateOrder pins the documented aggregate order fail > indeterminate
// > pass: a Result's verdict is the worst observation's verdict, whatever
// order the observations arrive in, and its reason is that same observation's
// reason.
func TestAggregateOrder(t *testing.T) {
	passed := observation(probe.Measured, probe.Pass, probe.ReasonOK)
	failed := observation(probe.Measured, probe.Fail, probe.ReasonConnRefused)
	ambiguous := observation(probe.Unresolved, probe.Indeterminate, probe.ReasonDNSUnresolved)

	cases := []struct {
		name         string
		observations []probe.Observation
		wantVerdict  probe.Verdict
		wantReason   probe.ReasonCode
	}{
		{"a pass alone stays a pass", []probe.Observation{passed}, probe.Pass, probe.ReasonOK},
		{"a fail alone stays a fail", []probe.Observation{failed}, probe.Fail, probe.ReasonConnRefused},
		{"an unresolved observation alone stays indeterminate", []probe.Observation{ambiguous}, probe.Indeterminate, probe.ReasonDNSUnresolved},
		{"fail outranks pass", []probe.Observation{passed, failed}, probe.Fail, probe.ReasonConnRefused},
		{"fail outranks indeterminate", []probe.Observation{ambiguous, failed}, probe.Fail, probe.ReasonConnRefused},
		{"indeterminate outranks pass", []probe.Observation{passed, ambiguous}, probe.Indeterminate, probe.ReasonDNSUnresolved},
		{"observation order does not change the verdict", []probe.Observation{failed, passed}, probe.Fail, probe.ReasonConnRefused},
		// A probe that reported nothing has measured nothing, so it cannot be a
		// success the run has no evidence for.
		{"no observations is never a pass", nil, probe.Indeterminate, probe.ReasonInternalError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotVerdict, gotReason := probe.Aggregate(tc.observations)
			if gotVerdict != tc.wantVerdict || gotReason != tc.wantReason {
				t.Fatalf("Aggregate = (%q, %q), want (%q, %q)", gotVerdict, gotReason, tc.wantVerdict, tc.wantReason)
			}
		})
	}
}

// TestAggregateNeverPromotesAnUnmeasuredObservation triangulates the honesty
// property the aggregate order exists for: an unresolved or not-measured
// observation is never promoted to a pass, a not-measured observation is never
// turned into a failure, and the reason reported for a failure always belongs
// to the observation that failed rather than to a sibling that was not
// measured.
func TestAggregateNeverPromotesAnUnmeasuredObservation(t *testing.T) {
	pass := observation(probe.Measured, probe.Pass, probe.ReasonOK)
	fail := observation(probe.Measured, probe.Fail, probe.ReasonBudgetExpired)
	ambiguous := observation(probe.Unresolved, probe.Indeterminate, probe.ReasonUDPSilence)
	missingHub := observation(probe.NotMeasured, probe.Indeterminate, probe.ReasonInputMissingHub)
	excluded := observation(probe.NotMeasured, probe.Indeterminate, probe.ReasonCapabilityExcluded)

	cases := []struct {
		name         string
		observations []probe.Observation
		wantVerdict  probe.Verdict
		wantReason   probe.ReasonCode
	}{
		{
			"an unresolved sibling is never a pass",
			[]probe.Observation{pass, ambiguous},
			probe.Indeterminate, probe.ReasonUDPSilence,
		},
		{
			"a not-measured sibling is never a pass",
			[]probe.Observation{pass, missingHub},
			probe.Indeterminate, probe.ReasonInputMissingHub,
		},
		{
			"a not-measured sibling is never turned into the failure",
			[]probe.Observation{fail, missingHub},
			probe.Fail, probe.ReasonBudgetExpired,
		},
		{
			"a failing sibling still outranks both unmeasured kinds",
			[]probe.Observation{ambiguous, missingHub, fail},
			probe.Fail, probe.ReasonBudgetExpired,
		},
		{
			"the first of two unmeasured siblings explains the gap",
			[]probe.Observation{excluded, ambiguous},
			probe.Indeterminate, probe.ReasonCapabilityExcluded,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotVerdict, gotReason := probe.Aggregate(tc.observations)
			if gotVerdict != tc.wantVerdict || gotReason != tc.wantReason {
				t.Fatalf("Aggregate = (%q, %q), want (%q, %q)", gotVerdict, gotReason, tc.wantVerdict, tc.wantReason)
			}
			if tc.wantVerdict == probe.Fail {
				// The failure reason must be the failing observation's, so it can
				// never be a code that means "not attempted".
				for _, unmeasured := range []probe.ReasonCode{probe.ReasonInputMissingHub, probe.ReasonCapabilityExcluded} {
					if gotReason == unmeasured {
						t.Fatalf("failure reason = %q, which means the observation was not measured", gotReason)
					}
				}
			}
		})
	}
}
