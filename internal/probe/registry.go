package probe

// This file is the probe registry of design §4 and §7: the one ordered
// declaration of the ten probes of PRD §5.1, and the only place that says which
// probes a run contains and in which order.
//
// The registry is a declaration rather than a construction. Each entry carries
// the probe's stable name, the kind it reports, and the constructor that builds
// it from the run's injected seams; Probes builds the entries that have a
// constructor. That is what lets this file be the single home of the probe set
// while the set is still being built: the enumeration contract — ten names,
// their kinds, and their order — is fixed from the first probe onward, and every
// later probe slice fills in one more constructor without moving a name, a kind
// or a position. An entry with no constructor is a probe whose slice has not
// landed yet; it is never a probe that measures nothing, and nothing here
// fabricates a result for it.
//
// The order is specification order, which is also the order of targets.go's
// declared target set. Design §3.3 requires the payload to echo probes in
// registry order, so the order is part of the contract rather than an
// implementation detail: it must not depend on map iteration or on the order in
// which probes happen to finish.
//
// Nothing here builds a probe at package initialisation time. A constructor is
// called by Probes, with the seams of the run it belongs to, so no global probe
// can carry one run's seams into another run — the reason design §6.1 forbids
// package-level seam state.

// The probe names of PRD §5.1. They live here, beside the order they appear in,
// because the registry is the probe set's single home: a probe's Name method
// returns its own constant, so a name cannot drift between the declaration and
// the probe that answers to it. The declared target set (targets.go) names the
// same probes in the same order, and the registry's enumeration test asserts the
// two agree.
const (
	// probeNameLocalEnv is this machine's environment classification.
	probeNameLocalEnv = "local.env"
	// probeNameLocalSSHD is the local sshd binary, service state and effective
	// configuration.
	probeNameLocalSSHD = "local.sshd"
	// probeNameEgressHub is the direct TCP measurement of the supplied hub
	// address.
	probeNameEgressHub = "egress.hub.direct"
	// probeNameEgressSSHKnown is SSH to a well-known public host on port 22: the
	// port-versus-protocol disambiguator.
	probeNameEgressSSHKnown = "egress.ssh.known"
	// probeNameEgressSSH443 is the same public SSH question on port 443.
	probeNameEgressSSH443 = "egress.ssh.443"
	// probeNameEgressCF7844 is the Cloudflare edge on the tunnel port.
	probeNameEgressCF7844 = "egress.cf.7844"
	// probeNameEgressCF443 is the Cloudflare edge on HTTPS.
	probeNameEgressCF443 = "egress.cf.443"
	// probeNameEgressQUIC is D8's narrow UDP datagram question.
	probeNameEgressQUIC = "egress.quic"
	// probeNameTLSInterception is the chain presented on the wire.
	probeNameTLSInterception = "tls.interception"
	// probeNameTLSTrustStore is the chain against the local trust store.
	probeNameTLSTrustStore = "tls.truststore"
)

// ProbeFactory builds one probe from the seams of the run it belongs to and the
// run's declared target input. It is the constructor type of a registry entry: a
// probe is built per run, never per process, so two runs cannot share the seams
// of one machine's measurement.
//
// The target input travels with the seams because a probe's declared target may
// be run input rather than a constant (design D3: the hub address is supplied with
// --hub). A probe that measures this machine ignores it; a probe whose target is
// run input resolves it through EffectiveTargets here, at construction, so the
// probe's Run performs exactly its declared measurement.
type ProbeFactory func(seams Seams, targets TargetInput) Probe

// ProbeRegistration is one probe's place in the registry: the name and kind the
// enumeration contract fixes, and the constructor that builds it.
//
// New is nil while the probe's own slice has not landed. A nil constructor is
// reported as a missing implementation by the caller that needs one (Probes
// simply does not build it), and it is never treated as a probe that resolves to
// something: a run must not be able to claim it measured a probe it has no code
// for.
type ProbeRegistration struct {
	// Name is the probe's stable identifier, e.g. "egress.hub.direct".
	Name string
	// Kind is the question family the probe belongs to.
	Kind ProbeKind
	// New builds the probe from the run's seams. Nil until the probe lands.
	New ProbeFactory
}

// registry is the ordered probe registry: exactly the ten probes of PRD §5.1, in
// specification order, with a constructor on each probe whose slice has landed.
//
// This slice lands local.env, local.sshd, egress.hub.direct (PR 5 and PR 6) and the
// two public-SSH probes and two Cloudflare edge probes (PR 7). Every later probe slice
// replaces one nil constructor with its own, which is the only edit to this table those
// slices need: the names, the kinds and the order are already the contract, and the
// enumeration test fails if any of them moves.
//
// The landed probes are not the same shape. local.env and local.sshd measure this
// machine and ignore the target input; egress.hub.direct has no declared host of its
// own — its address is run input — and resolves it here; the two SSH probes declare one
// endpoint each and the two edge probes declare two regions each, and they take the
// target input so an override of those endpoints is honoured through one declaration.
// That is why the factory takes both values rather than seams alone.
var registry = []ProbeRegistration{
	{Name: probeNameLocalEnv, Kind: ProbeLocal, New: newLocalEnv},
	{Name: probeNameLocalSSHD, Kind: ProbeLocal, New: newLocalSSHD},
	{Name: probeNameEgressHub, Kind: ProbeEgress, New: newEgressHub},
	{Name: probeNameEgressSSHKnown, Kind: ProbeEgress, New: newEgressSSHKnown},
	{Name: probeNameEgressSSH443, Kind: ProbeEgress, New: newEgressSSH443},
	{Name: probeNameEgressCF7844, Kind: ProbeEgress, New: newEgressCF7844},
	{Name: probeNameEgressCF443, Kind: ProbeEgress, New: newEgressCF443},
	{Name: probeNameEgressQUIC, Kind: ProbeProto},
	{Name: probeNameTLSInterception, Kind: ProbeTLS},
	{Name: probeNameTLSTrustStore, Kind: ProbeTLS},
}

// Registry returns the ordered probe registry as a copy: the ten entries of
// PRD §5.1, in specification order, each with the name and kind a consumer may
// rely on.
//
// The copy matters for the same reason DeclaredTargets returns one: the
// enumeration order and the name set are a contract the payload and the human
// report both echo, so a caller must not be able to reorder or shrink the probe
// set through the value it read.
func Registry() []ProbeRegistration {
	entries := make([]ProbeRegistration, len(registry))
	copy(entries, registry)
	return entries
}

// Probes builds the probes the registry can construct, in registry order, for a
// run whose declared target input is empty.
//
// A run with run input — a hub address, target overrides — must call ProbesFor
// instead: this function exists for the callers that have no input to hand over
// (and for the registry's own enumeration case), and a probe whose target is run
// input reports the missing input as a not-measured observation rather than
// inventing a target, so the omission is visible in the run's output instead of
// silently measuring something else.
func Probes(seams Seams) []Probe {
	return ProbesFor(seams, TargetInput{})
}

// ProbesFor builds the probes the registry can construct, in registry order, from
// the run's seams and its declared target input.
//
// A registry entry with no constructor is skipped rather than replaced by a
// placeholder: this slice's registry builds the probes whose slices have landed,
// and the probes that follow are built by the slices that implement them.
// Callers that need to know what a run is missing read Registry and compare the
// entries whose New is nil, so a gap is reported as a declared probe that was not
// built instead of as a probe that answered with nothing.
//
// The target input is handed to each probe as a value, so no probe can rebind the
// run's input for the probe after it. The overrides inside it are a slice: a probe
// must read them and never edit them, which is the same rule the declared set's
// accessors follow.
func ProbesFor(seams Seams, targets TargetInput) []Probe {
	var built []Probe
	for _, entry := range registry {
		if entry.New == nil {
			continue
		}
		built = append(built, entry.New(seams, targets))
	}
	return built
}
