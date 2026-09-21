// Package probe_test exercises the measurement vocabulary from outside the
// package: the invariants every Observation must satisfy, the aggregate order
// every Result's verdict is reduced through, and later the closed reason-code
// set. Keeping the tests external proves the exported vocabulary is enough to
// state the design's invariants without reaching into the package.
package probe_test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"
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
	// design §3.5 records the set as it stood when native Windows was a refusal
	// (28 codes). Removing the platform-refusal code when native Windows became a
	// supported classification leaves the 27 distinct codes below, and `ok`
	// remains the only code two §5.1 rows share.
	if len(documentedReasonCodes) != 27 {
		t.Errorf("documented set has %d codes, want the 27 enumerated in this slice", len(documentedReasonCodes))
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

// ---------------------------------------------------------------------------
// Issue #68 — a detail names the address it measured exactly once
// ---------------------------------------------------------------------------

// TestDetailsNameTheMeasuredAddressOnce asserts issue #68's property over every wording
// that quotes the error that ended an attempt: the detail names the address the attempt
// was against exactly once.
//
// The standard library formats its own network errors as "<operation> <address>:
// <cause>" — "dial tcp 127.0.0.1:1: connect: connection refused", "read tcp
// 192.0.2.5:53124->198.51.100.7:443: i/o timeout", "lookup nosuch.example: no such host"
// — so a detail that joins the probe's own "<operation> <address>:" prefix to one names
// the address twice, and the doubling reaches every conclusion that quotes the detail.
//
// The property is asserted, not the sentence: a wording edit may rewrite these sentences
// freely, while a case that compared whole strings would have to be rewritten beside
// every edit and would not notice the doubling returning inside a differently shaped
// sentence. The denied-seam cases are the control — ErrSeamDenied names no address, so
// there the probe's own prefix is the only thing that says what was attempted, and a
// repair that dropped the address from every detail would fail those cases.
//
// Each case also pins the classification the same input produced before the wording was
// repaired. Classify chooses a resolution, a verdict and a reason code together, from one
// table row, so a matching reason code is the same classification the run reported then:
// the repair is shown to be wording only, and no conclusion built from these facts moves.
func TestDetailsNameTheMeasuredAddressOnce(t *testing.T) {
	hubAddress := testHubAddress
	sshAddress := declaredAddress(t, testSSHKnownProbe)
	sshHost, _, err := net.SplitHostPort(sshAddress)
	if err != nil {
		t.Fatalf("the declared public-SSH target %q is not host:port: %v", sshAddress, err)
	}
	quicAddresses := declaredAddresses(t, testQuicProbe)
	tlsAddress := declaredAddress(t, testTLSInterceptionProbe)
	trustStoreAddress := declaredAddress(t, testTLSTrustStoreProbe)

	// networkError builds what the standard library's own network code returns: an
	// *net.OpError whose message is "<operation> <network> <address>: <cause>". The
	// scripted address is a net.Addr like any other, so the message is the one a real
	// dial, read or write carries.
	networkError := func(operation, network, address string, cause error) error {
		return &net.OpError{Op: operation, Net: network, Addr: scriptedAddr(address), Err: cause}
	}
	// refused is the dial failure the released binary printed, wrapped the way net
	// wraps it, so its message reads "connect: connection refused" rather than the bare
	// syscall wording.
	refused := func(operation, network, address string) error {
		return networkError(operation, network, address, os.NewSyscallError("connect", syscall.ECONNREFUSED))
	}
	// packetErrors scripts one datagram socket per declared edge region, each socket's
	// error naming its own region: the two regions are measured separately, so a shared
	// error would name the wrong region's address.
	packetErrors := func(script func(address string) *scriptedPacketConn) *scriptedPacketDialer {
		perAddress := make(map[string]*scriptedPacketConn, len(quicAddresses))
		for _, address := range quicAddresses {
			perAddress[address] = script(address)
		}
		return &scriptedPacketDialer{perAddress: perAddress}
	}

	cases := []struct {
		name string
		// run scripts one measurement and returns the address each observation's
		// detail must name exactly once, in observation order. For a resolution
		// failure it is the host the resolver's own error names; everywhere else it is
		// the observation's own target.
		run func(t *testing.T) (probe.Result, []string)
		// wantReasons pins the reason code of every observation, in order.
		wantReasons []probe.ReasonCode
	}{
		{
			// egress.go dialFactFor: the error already names the operation and the
			// address, so the probe must use it verbatim.
			name: "a refused hub dial",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := &scriptedDialer{err: refused("dial", "tcp", hubAddress)}
				return runHub(t, dialer, probe.TargetInput{Hub: hubAddress}), []string{hubAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonConnRefused},
		},
		{
			// egress.go dialFactFor's budget branch: the probe's own sentence follows
			// the error, which already names the address.
			name: "a hub dial that expired the probe's own budget",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := &scriptedDialer{err: networkError("dial", "tcp", hubAddress, os.ErrDeadlineExceeded)}
				return runHub(t, dialer, probe.TargetInput{Hub: hubAddress}), []string{hubAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonBudgetExpired},
		},
		{
			// egress.go dialFactFor's default branch, whose sentence is appended to
			// the error that already names the address.
			name: "a hub dial that failed for a reason no row names",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := &scriptedDialer{err: networkError("dial", "tcp", hubAddress, errors.New("no route to host"))}
				return runHub(t, dialer, probe.TargetInput{Hub: hubAddress}), []string{hubAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonInternalError},
		},
		{
			// Control: a denied seam names no address, so the probe's own prefix is
			// the only thing that says what was dialed.
			name: "a hub dial the seam denied",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := &scriptedDialer{err: probe.ErrSeamDenied}
				return runHub(t, dialer, probe.TargetInput{Hub: hubAddress}), []string{hubAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonCommandDenied},
		},
		{
			// Control: a dialer whose own error names neither operation nor address
			// still gets the probe's prefix, so the detail says what was dialed.
			name: "a hub dial whose own error names nothing",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := &scriptedDialer{err: errors.New("the scripted dialer refused the attempt")}
				return runHub(t, dialer, probe.TargetInput{Hub: hubAddress}), []string{hubAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonInternalError},
		},
		{
			// egress.go resolverFact: the resolver's own error names the lookup.
			name: "a name the resolver answered authoritatively",
			run: func(t *testing.T) (probe.Result, []string) {
				resolver := &scriptedResolver{err: &net.DNSError{Err: "no such host", Name: sshHost, IsNotFound: true}}
				return runSsh(t, testSSHKnownProbe, reachSeams(resolver, &scriptedDialer{}), probe.TargetInput{}), []string{sshHost}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonDNSNoSuchHost},
		},
		{
			// egress.go resolverFact's unresolved branch.
			name: "a resolver that did not answer",
			run: func(t *testing.T) (probe.Result, []string) {
				resolver := &scriptedResolver{err: &net.DNSError{Err: "i/o timeout", Name: sshHost, IsTimeout: true}}
				return runSsh(t, testSSHKnownProbe, reachSeams(resolver, &scriptedDialer{}), probe.TargetInput{}), []string{sshHost}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonDNSUnresolved},
		},
		{
			// The same fact raised by the dialer's own resolution, which net reports
			// as "dial tcp: lookup <host>: no such host".
			name: "a name the dialer's own resolution did not find",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := &scriptedDialer{err: &net.OpError{Op: "dial", Net: "tcp", Err: &net.DNSError{Err: "no such host", Name: sshHost, IsNotFound: true}}}
				resolver := &scriptedResolver{addresses: []string{"203.0.113.7"}}
				return runSsh(t, testSSHKnownProbe, reachSeams(resolver, dialer), probe.TargetInput{}), []string{sshHost}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonDNSNoSuchHost},
		},
		{
			// Control: a denied lookup carries no address.
			name: "a lookup the seam denied",
			run: func(t *testing.T) (probe.Result, []string) {
				resolver := &scriptedResolver{err: probe.ErrSeamDenied}
				return runSsh(t, testSSHKnownProbe, reachSeams(resolver, &scriptedDialer{}), probe.TargetInput{}), []string{sshHost}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonCommandDenied},
		},
		{
			// egress.go bannerFact's reset branch.
			name: "a read the far end reset",
			run: func(t *testing.T) (probe.Result, []string) {
				conn := &scriptedBannerConn{readErr: networkError("read", "tcp", sshAddress, syscall.ECONNRESET)}
				dialer := &scriptedDialer{conn: conn}
				resolver := &scriptedResolver{addresses: []string{"203.0.113.7"}}
				return runSsh(t, testSSHKnownProbe, reachSeams(resolver, dialer), probe.TargetInput{}), []string{sshAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonConnReset},
		},
		{
			// egress.go bannerFact's default read-failure branch.
			name: "a read that failed for a reason no row names",
			run: func(t *testing.T) (probe.Result, []string) {
				conn := &scriptedBannerConn{readErr: networkError("read", "tcp", sshAddress, errors.New("connection aborted"))}
				dialer := &scriptedDialer{conn: conn}
				resolver := &scriptedResolver{addresses: []string{"203.0.113.7"}}
				return runSsh(t, testSSHKnownProbe, reachSeams(resolver, dialer), probe.TargetInput{}), []string{sshAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonInternalError},
		},
		{
			// egress.go bannerFact's budget branch, the read-side expiry beside the
			// dial-side one above.
			name: "a read that expired the probe's own budget",
			run: func(t *testing.T) (probe.Result, []string) {
				conn := &scriptedBannerConn{readErr: networkError("read", "tcp", sshAddress, os.ErrDeadlineExceeded)}
				dialer := &scriptedDialer{conn: conn}
				resolver := &scriptedResolver{addresses: []string{"203.0.113.7"}}
				return runSsh(t, testSSHKnownProbe, reachSeams(resolver, dialer), probe.TargetInput{}), []string{sshAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonBudgetExpired},
		},
		{
			// Control: a far end that sends nothing ends the read with io.EOF, which
			// names no address, so this wording keeps the probe's own prefix — the fix
			// must not have dropped the address from the branch that never doubled it.
			name: "a read the far end ended without sending anything",
			run: func(t *testing.T) (probe.Result, []string) {
				conn := &scriptedBannerConn{readErr: io.EOF}
				dialer := &scriptedDialer{conn: conn}
				resolver := &scriptedResolver{addresses: []string{"203.0.113.7"}}
				return runSsh(t, testSSHKnownProbe, reachSeams(resolver, dialer), probe.TargetInput{}), []string{sshAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonBannerNotSSH},
		},
		{
			// Control: a denied packet seam carries no address, and each region's
			// detail is named by the probe's own prefix.
			name: "a packet socket the seam denied",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := &scriptedPacketDialer{err: fmt.Errorf("dial udp: %w", probe.ErrSeamDenied)}
				return runQuic(t, packetSeams(dialer), probe.TargetInput{}), quicAddresses
			},
			wantReasons: []probe.ReasonCode{probe.ReasonCommandDenied, probe.ReasonCommandDenied},
		},
		{
			// quic.go packetErrorFact's refusal branch, reached through a failing
			// write.
			name: "a datagram write the socket refused",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := packetErrors(func(address string) *scriptedPacketConn {
					return &scriptedPacketConn{writeErr: networkError("write", "udp", address, os.NewSyscallError("write", syscall.ECONNREFUSED))}
				})
				return runQuic(t, packetSeams(dialer), probe.TargetInput{}), quicAddresses
			},
			wantReasons: []probe.ReasonCode{probe.ReasonUDPUnreachable, probe.ReasonUDPUnreachable},
		},
		{
			// quic.go packetErrorFact's default branch.
			name: "a datagram write that failed for a reason no row names",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := packetErrors(func(address string) *scriptedPacketConn {
					return &scriptedPacketConn{writeErr: networkError("write", "udp", address, errors.New("network is down"))}
				})
				return runQuic(t, packetSeams(dialer), probe.TargetInput{}), quicAddresses
			},
			wantReasons: []probe.ReasonCode{probe.ReasonUDPErrorUnclassified, probe.ReasonUDPErrorUnclassified},
		},
		{
			// quic.go udpReplyFact's refusal branch.
			name: "a datagram read the socket refused",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := packetErrors(func(address string) *scriptedPacketConn {
					return &scriptedPacketConn{readErr: networkError("read", "udp", address, syscall.ECONNREFUSED)}
				})
				return runQuic(t, packetSeams(dialer), probe.TargetInput{}), quicAddresses
			},
			wantReasons: []probe.ReasonCode{probe.ReasonUDPUnreachable, probe.ReasonUDPUnreachable},
		},
		{
			// quic.go udpReplyFact's default branch.
			name: "a datagram read that failed for a reason no row names",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := packetErrors(func(address string) *scriptedPacketConn {
					return &scriptedPacketConn{readErr: networkError("read", "udp", address, errors.New("socket is not connected"))}
				})
				return runQuic(t, packetSeams(dialer), probe.TargetInput{}), quicAddresses
			},
			wantReasons: []probe.ReasonCode{probe.ReasonUDPErrorUnclassified, probe.ReasonUDPErrorUnclassified},
		},
		{
			// Control: silence embeds no error at all; the probe names each region
			// itself and the detail must keep doing so.
			name: "datagram silence",
			run: func(t *testing.T) (probe.Result, []string) {
				dialer := packetErrors(func(address string) *scriptedPacketConn {
					return &scriptedPacketConn{readErr: networkError("read", "udp", address, os.ErrDeadlineExceeded)}
				})
				return runQuic(t, packetSeams(dialer), probe.TargetInput{}), quicAddresses
			},
			wantReasons: []probe.ReasonCode{probe.ReasonUDPSilence, probe.ReasonUDPSilence},
		},
		{
			// tls.go tlsChainFact's handshake-error branch: the production verifier
			// returns the dial error unchanged, address and all.
			name: "a chain handshake that dialed nothing",
			run: func(t *testing.T) (probe.Result, []string) {
				verifier := &scriptedTLSVerifier{err: refused("dial", "tcp", tlsAddress)}
				return runTLSInterception(t, tlsSeams(verifier), probe.TargetInput{}), []string{tlsAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonTLSHandshakeUnresolved},
		},
		{
			// tls.go tlsTrustStoreFact's handshake-error branch, for the same reason.
			name: "a trust-store handshake that dialed nothing",
			run: func(t *testing.T) (probe.Result, []string) {
				verifier := &scriptedTLSVerifier{err: refused("dial", "tcp", trustStoreAddress)}
				platform := scriptedPlatform{goos: testTLSTrustStoreLinux, arch: "amd64"}
				return runTLSTruststore(t, trustStoreSeams(verifier, platform, &scriptedEnv{}), probe.TargetInput{}), []string{trustStoreAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonTLSHandshakeUnresolved},
		},
		{
			// Control: tls.go's denied-verifier branch, where the denial names no
			// address and the prefix is the only thing that says what was measured.
			name: "a chain handshake the seam denied",
			run: func(t *testing.T) (probe.Result, []string) {
				verifier := &scriptedTLSVerifier{err: probe.ErrSeamDenied}
				return runTLSInterception(t, tlsSeams(verifier), probe.TargetInput{}), []string{tlsAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonCommandDenied},
		},
		{
			// Control: the trust-store probe's denied-verifier branch.
			name: "a trust-store handshake the seam denied",
			run: func(t *testing.T) (probe.Result, []string) {
				verifier := &scriptedTLSVerifier{err: probe.ErrSeamDenied}
				platform := scriptedPlatform{goos: testTLSTrustStoreLinux, arch: "amd64"}
				return runTLSTruststore(t, trustStoreSeams(verifier, platform, &scriptedEnv{}), probe.TargetInput{}), []string{trustStoreAddress}
			},
			wantReasons: []probe.ReasonCode{probe.ReasonCommandDenied},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, addresses := tc.run(t)
			if len(result.Observations) != len(tc.wantReasons) {
				t.Fatalf("the run reported %d observations, want %d", len(result.Observations), len(tc.wantReasons))
			}
			if len(addresses) != len(result.Observations) {
				t.Fatalf("the case names %d addresses for %d observations", len(addresses), len(result.Observations))
			}
			for i, observation := range result.Observations {
				if got := strings.Count(observation.Detail, addresses[i]); got != 1 {
					t.Errorf("observation %d names %q %d times, want exactly once: %q", i, addresses[i], got, observation.Detail)
				}
				if observation.Reason != tc.wantReasons[i] {
					t.Errorf("observation %d reason = %q, want %q: the wording must not move a classification", i, observation.Reason, tc.wantReasons[i])
				}
			}
			if len(result.Observations) == 1 {
				// The result's detail is what a conclusion quotes, so the property is
				// asserted on it as well.
				if got := strings.Count(result.Detail, addresses[0]); got != 1 {
					t.Errorf("the result names %q %d times, want exactly once: %q", addresses[0], got, result.Detail)
				}
			}
		})
	}
}
