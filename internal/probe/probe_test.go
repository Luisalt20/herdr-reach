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

// declaredProbeRegistry is PRD §5.1's probe table, transcribed row by row in
// specification order: the ten probes, each with the single kind it declares.
// It is deliberately a literal transcription rather than a read of the
// declaration, so the enumeration case below detects drift between the
// specification and the registry instead of restating the registry to itself.
var declaredProbeRegistry = []struct {
	name string
	kind probe.ProbeKind
}{
	{"local.env", probe.ProbeLocal},
	{"local.sshd", probe.ProbeLocal},
	{"egress.hub.direct", probe.ProbeEgress},
	{"egress.ssh.known", probe.ProbeEgress},
	{"egress.ssh.443", probe.ProbeEgress},
	{"egress.cf.7844", probe.ProbeEgress},
	{"egress.cf.443", probe.ProbeEgress},
	{"egress.quic", probe.ProbeProto},
	{"tls.interception", probe.ProbeTLS},
	{"tls.truststore", probe.ProbeTLS},
}

// probeKinds is the closed set of probe kinds (PRD §5.1): a probe declares
// exactly one of these four, and a fifth value would be a specification change
// rather than an implementation detail.
var probeKinds = []probe.ProbeKind{probe.ProbeLocal, probe.ProbeEgress, probe.ProbeTLS, probe.ProbeProto}

// TestRegistryEnumeratesTheDeclaredProbes is the registry's enumeration case
// (PRD §5.1's first scenario): the registry holds exactly the ten declared
// probes, in specification order, each with one of the four kinds, with no
// eleventh entry, no duplicate, and an enumeration order that does not move
// between runs. Everything a run reports is echoed in registry order (design
// §3.3), so an unstable or extended enumeration would change output that must
// not depend on map iteration or on which probe finished first.
func TestRegistryEnumeratesTheDeclaredProbes(t *testing.T) {
	registry := probe.Registry()
	if len(registry) != len(declaredProbeRegistry) {
		t.Fatalf("the registry holds %d probes, want the %d of PRD §5.1", len(registry), len(declaredProbeRegistry))
	}

	seen := make(map[string]bool, len(registry))
	for i, want := range declaredProbeRegistry {
		entry := registry[i]
		if entry.Name != want.name || entry.Kind != want.kind {
			t.Errorf("registry[%d] = (%q, %q), want (%q, %q) in specification order",
				i, entry.Name, entry.Kind, want.name, want.kind)
		}
		if seen[entry.Name] {
			t.Errorf("registry[%d] = %q is a duplicate: one probe, one entry", i, entry.Name)
		}
		seen[entry.Name] = true

		// The kind must be one of the four documented values: a probe that
		// declares something else is a probe no consumer has a contract for.
		known := false
		for _, kind := range probeKinds {
			if entry.Kind == kind {
				known = true
				break
			}
		}
		if !known {
			t.Errorf("registry[%d] = %q declares kind %q, which is not one of the four probe kinds", i, entry.Name, entry.Kind)
		}
	}

	// No eleventh entry: every registered name is one of the declared ten. The
	// count plus the positional loop above already say so; this map says it
	// against the specification rather than against the registry's own order.
	declaredNames := make(map[string]bool, len(declaredProbeRegistry))
	for _, declared := range declaredProbeRegistry {
		declaredNames[declared.name] = true
	}
	for i, entry := range registry {
		if !declaredNames[entry.Name] {
			t.Errorf("registry[%d] = %q is not one of the ten declared probes", i, entry.Name)
		}
	}

	// Stability across runs: two enumerations of the same registry must be the
	// same sequence, because both the payload and the human report read it.
	again := probe.Registry()
	if len(again) != len(registry) {
		t.Fatalf("the second enumeration holds %d probes, the first %d", len(again), len(registry))
	}
	for i := range registry {
		if again[i].Name != registry[i].Name || again[i].Kind != registry[i].Kind {
			t.Errorf("enumeration %d differs between runs: (%q, %q) vs (%q, %q)",
				i, again[i].Name, again[i].Kind, registry[i].Name, registry[i].Kind)
		}
	}

	// The registry hands out a copy: a caller cannot reorder or shrink the
	// probe set through the value it read.
	registry[0] = probe.ProbeRegistration{Name: "mutated", Kind: probe.ProbeTLS}
	if probe.Registry()[0].Name != declaredProbeRegistry[0].name {
		t.Fatal("Registry() handed out the package's own registry slice")
	}

	// The registry is the probe set; the declared target set is the dialed set.
	// They name the same ten probes in the same order, so a probe cannot be
	// registered without a declaration or declared without a registration.
	declarations := probe.DeclaredTargets()
	if len(declarations) != len(registry) {
		t.Fatalf("the declared target set holds %d probes, the registry %d", len(declarations), len(registry))
	}
	for i, declaration := range declarations {
		if declaration.Probe != declaredProbeRegistry[i].name {
			t.Errorf("declared target set[%d] = %q, the registry declares %q in that position",
				i, declaration.Probe, declaredProbeRegistry[i].name)
		}
	}
}

// TestRegistryBuildsOnlyRegisteredProbes asserts the second half of the
// enumeration contract: the probes a run can actually build are the registry's
// own entries, in registry order, each reporting the name and kind its entry
// declares. A probe the registry cannot build is a probe whose own slice has not
// landed yet, and building one must never produce a differently named or
// differently kinded probe than the declaration promised.
func TestRegistryBuildsOnlyRegisteredProbes(t *testing.T) {
	registered := map[string]probe.ProbeKind{}
	buildable := 0
	for _, entry := range probe.Registry() {
		registered[entry.Name] = entry.Kind
		if entry.New != nil {
			buildable++
		}
	}
	if buildable == 0 {
		t.Fatal("the registry cannot build a single probe, so a run would measure nothing")
	}

	built := probe.Probes(probe.DenyAllSeams())
	if len(built) != buildable {
		t.Fatalf("the registry built %d probes, its entries promise %d", len(built), buildable)
	}

	position := 0
	for _, entry := range probe.Registry() {
		if entry.New == nil {
			continue
		}
		probeUnderTest := built[position]
		position++
		if probeUnderTest.Name() != entry.Name {
			t.Errorf("built probe %d reports %q, its registry entry declares %q", position-1, probeUnderTest.Name(), entry.Name)
		}
		if probeUnderTest.Kind() != entry.Kind {
			t.Errorf("built probe %q reports kind %q, its registry entry declares %q", entry.Name, probeUnderTest.Kind(), entry.Kind)
		}
	}

	// A built probe outside the registry would be the eleventh entry the
	// enumeration case forbids.
	for _, probeUnderTest := range built {
		kind, ok := registered[probeUnderTest.Name()]
		if !ok {
			t.Errorf("the registry built the unregistered probe %q", probeUnderTest.Name())
			continue
		}
		if kind != probeUnderTest.Kind() {
			t.Errorf("built probe %q reports kind %q, the registry declares %q", probeUnderTest.Name(), probeUnderTest.Kind(), kind)
		}
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
