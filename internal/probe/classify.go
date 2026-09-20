package probe

// This file is the classification table of design §5.1: the single place where
// a raw fact a probe observed becomes the (resolution, verdict, reason) triple
// its observation reports. Every probe classifies through Classify/Observe and
// nothing else selects a reason code inline, so one observable can never be
// classified two ways — the property the spec's timeout requirement demands.
//
// Nothing here reasons about the network. It maps one fact to one code.

// Purpose is the declared question a probe asks of its observable. The same
// raw fact classifies differently by purpose — a dial that ran out the probe's
// own budget is the measurement itself for a reachability question, and merely
// an ambiguity otherwise — so the table is keyed by purpose as well as by what
// was seen (design D2, design §5.1).
type Purpose string

const (
	// PurposePortReachability asks "is this declared TCP port reachable?".
	// Its own budget expiring is the measurement, so a blackholed port is a
	// definite negative rather than an ambiguity.
	PurposePortReachability Purpose = "port_reachability"
	// PurposeNameResolution asks "does this declared name resolve
	// authoritatively?".
	PurposeNameResolution Purpose = "name_resolution"
	// PurposeUDPReachability asks D8's narrow question: "does a UDP datagram to
	// this edge draw any reply inside the probe budget?".
	PurposeUDPReachability Purpose = "udp_reachability"
	// PurposeTLSCertificate asks "which chain does this target present, and is
	// its issuer the declared expected publisher?".
	PurposeTLSCertificate Purpose = "tls_certificate"
	// PurposeTLSTrustStore asks "does the local trust store accept this
	// chain?".
	PurposeTLSTrustStore Purpose = "tls_truststore"
	// PurposeSSHDConfiguration asks what the local sshd is and how it is
	// configured.
	PurposeSSHDConfiguration Purpose = "sshd_configuration"
	// PurposePlatformClassification asks what this machine is (local.env).
	PurposePlatformClassification Purpose = "platform_classification"
)

// Observable is a raw fact a probe can see, before any judgement is attached
// to it. The vocabulary is closed for the same reason the reason-code set is:
// the table is total over it and reports a fact it does not recognise as an
// internal failure rather than guessing what it might have meant.
type Observable string

const (
	// ObsTCPEstablished is a TCP connection that completed.
	ObsTCPEstablished Observable = "tcp_established"
	// ObsTCPRefused is a TCP connection the far end refused (ECONNREFUSED).
	ObsTCPRefused Observable = "tcp_refused"
	// ObsTCPReset is a TCP connection reset after it was established.
	ObsTCPReset Observable = "tcp_reset"
	// ObsBannerNotSSH is a reachable port whose banner is not an SSH
	// identification string, which is what an intercepting proxy looks like.
	ObsBannerNotSSH Observable = "banner_not_ssh"
	// ObsSSHBannerReceived is an SSH identification string read from a reachable
	// port: the far end answered and answered as an SSH server. It is the positive
	// counterpart of ObsBannerNotSSH, and it is a stronger fact than
	// ObsTCPEstablished — a connection that completed says the port is reachable,
	// while this says what is listening on it.
	ObsSSHBannerReceived Observable = "ssh_banner_received"
	// ObsProbeBudgetExpired is the probe's own budget expiring with no
	// response to its declared port.
	ObsProbeBudgetExpired Observable = "probe_budget_expired"
	// ObsResolverAuthoritativeNegative is a resolver answering that the name
	// does not exist.
	ObsResolverAuthoritativeNegative Observable = "resolver_authoritative_negative"
	// ObsResolverUnavailable is a resolver that timed out, returned SERVFAIL
	// or does not exist. It says nothing about the destination.
	ObsResolverUnavailable Observable = "resolver_unavailable"
	// ObsProbeIgnoredBudget is a probe that neither returned nor honoured its
	// context. The runner sees this, never the probe.
	ObsProbeIgnoredBudget Observable = "probe_ignored_budget"
	// ObsRunCancelled is a run context cancelled while probes were in flight.
	ObsRunCancelled Observable = "run_cancelled"
	// ObsRunBudgetExhausted is the run's global budget reaching its limit.
	ObsRunBudgetExhausted Observable = "run_budget_exhausted"
	// ObsUDPResponse is any UDP datagram arriving from the edge inside the
	// probe budget.
	ObsUDPResponse Observable = "udp_response"
	// ObsUDPSilence is a UDP socket producing neither a reply nor an error.
	ObsUDPSilence Observable = "udp_silence"
	// ObsUDPUnreachable is an ICMP port-unreachable surfaced by the socket.
	ObsUDPUnreachable Observable = "udp_port_unreachable"
	// ObsUDPOtherError is any other UDP socket error.
	ObsUDPOtherError Observable = "udp_error_other"
	// ObsTLSVerified is a chain that verified against the verifier in use,
	// with the issuer among the declared expected publishers.
	ObsTLSVerified Observable = "tls_verified"
	// ObsTLSVerifyFailed is a chain that failed verification.
	ObsTLSVerifyFailed Observable = "tls_verification_failed"
	// ObsTLSIssuerUnexpected is a verified chain whose issuer is outside the
	// declared expected set for the target. The run records the divergence but
	// cannot establish whether the publisher changed or the traffic is
	// intercepted, so the table reports it as unresolved rather than as a
	// failure, and the wording never accuses.
	ObsTLSIssuerUnexpected Observable = "tls_issuer_unexpected"
	// ObsTLSHandshakeError is a handshake error that is neither a verification
	// failure nor an unexpected issuer.
	ObsTLSHandshakeError Observable = "tls_handshake_error"
	// ObsTrustStoreRejectsChain is the local trust pool rejecting the chain.
	ObsTrustStoreRejectsChain Observable = "truststore_rejects_chain"
	// ObsTrustStoreVerifierUnavailable is a platform trust verifier that
	// cannot answer at all, as with macOS system roots.
	ObsTrustStoreVerifierUnavailable Observable = "truststore_verifier_unavailable"
	// ObsTrustStoreVerifierBypassed is a certificate-file or
	// certificate-directory environment override, which bypasses the platform
	// verifier and would make its answer mean something else.
	ObsTrustStoreVerifierBypassed Observable = "truststore_verifier_bypassed"
	// ObsSSHDBinaryAbsent is an sshd binary that is not installed.
	ObsSSHDBinaryAbsent Observable = "sshd_binary_absent"
	// ObsSSHDBinaryPresent is an sshd binary at the documented path. It is the
	// positive counterpart of ObsSSHDBinaryAbsent: the presence of the binary is
	// a measurement of this machine in its own right (R-HR-18), and a measured
	// pass is only expressible if the table has a row for it.
	ObsSSHDBinaryPresent Observable = "sshd_binary_present"
	// ObsSSHDConfigDivergent is a written configuration that differs from the
	// effective one.
	ObsSSHDConfigDivergent Observable = "sshd_config_divergent"
	// ObsSSHDConfigMatches is a written configuration the effective
	// configuration agrees with for every directive the written file sets. It is
	// the positive counterpart design §5.1 does not print: §5.2's
	// `SSHD_PRESENT_CONFIGURED` requires a healthy sshd to be reportable, and
	// the table is the only place a reason code may be chosen.
	ObsSSHDConfigMatches Observable = "sshd_config_matches"
	// ObsSSHDServiceRunning is the service manager reporting an active sshd unit.
	ObsSSHDServiceRunning Observable = "sshd_service_running"
	// ObsSSHDServiceNotRunning is the service manager answering that no queried
	// sshd unit is active. It is a measured negative about this machine's sshd.
	ObsSSHDServiceNotRunning Observable = "sshd_service_not_running"
	// ObsCapabilityExcluded is a capability this slice's zero-execution
	// boundary never attempts, such as `sshd -T` with no production command
	// runner wired.
	ObsCapabilityExcluded Observable = "capability_excluded"
	// ObsCommandDenied is a command seam that denied the execution.
	ObsCommandDenied Observable = "command_denied"
	// ObsHubInputMissing is a run with no hub address supplied.
	ObsHubInputMissing Observable = "hub_input_missing"
	// ObsPlatformSignalsUnknown is a set of platform signals matching no
	// supported classification.
	ObsPlatformSignalsUnknown Observable = "platform_signals_unknown"
	// ObsPlatformSignalsClassified is a set of platform signals matching one of
	// the supported classifications: Linux, macOS or WSL2 as a node, or native
	// Windows (which is classified in order to be refused).
	ObsPlatformSignalsClassified Observable = "platform_signals_classified"
	// ObsNodePlatformUnsupported is a node classified as native Windows, which
	// this tool refuses to operate on.
	ObsNodePlatformUnsupported Observable = "node_platform_unsupported"
	// ObsInternalFailure is a fact no row of the table recognises.
	ObsInternalFailure Observable = "internal_failure"
)

// RawObservation is one fact a probe saw. Kind is what the table matches on;
// Wording is the operating system's verbatim text, carried through to
// Observation.Detail and never used to choose a reason code (R-HR-07).
type RawObservation struct {
	Kind    Observable
	Wording string
}

// Classification is the triple the table assigns to a raw observation. Its
// three fields always agree: a measured classification carries a definite
// verdict and the two non-measured resolutions carry Indeterminate.
type Classification struct {
	Resolution Resolution
	Verdict    Verdict
	Reason     ReasonCode
}

// classificationRow is one row of the ordered table. An empty purpose matches
// every purpose and an empty observable matches every observable, which is how
// the runner-classified facts and the table's total last row are expressed.
type classificationRow struct {
	purpose    Purpose
	observable Observable
	class      Classification
}

// classificationTable is the ordered classification table of design §5.1. Rows
// are evaluated in declaration order and the first match wins.
//
// The last row is deliberately total, so the table never falls outside itself:
// a fact no row recognises is reported as an internal failure, never as a pass
// and never as a block.
var classificationTable = []classificationRow{
	// Reachability questions: the declared question has a definite answer.
	// A budget expiry belongs here rather than with the ambiguities precisely
	// because the declared question is "is this port reachable?" — for that
	// question, no answer inside the budget is the measurement.
	{PurposePortReachability, ObsTCPEstablished, Classification{Measured, Pass, ReasonOK}},
	{PurposePortReachability, ObsTCPRefused, Classification{Measured, Fail, ReasonConnRefused}},
	{PurposePortReachability, ObsTCPReset, Classification{Measured, Fail, ReasonConnReset}},
	{PurposePortReachability, ObsBannerNotSSH, Classification{Measured, Fail, ReasonBannerNotSSH}},
	// A port that answered with an SSH identification string is the positive half of
	// the same question, and it has its own observable rather than borrowing
	// `tcp_established`: a completed connection says the port is reachable, while the
	// identification string says what answered. Design §5.1 prints the negative row
	// ("banner is not SSH") and not this one, which the sshd and platform rows above
	// also record: the table is the only place a reason code may be chosen, so the
	// positive outcome of a question the probes actually ask has to have a row.
	{PurposePortReachability, ObsSSHBannerReceived, Classification{Measured, Pass, ReasonOK}},
	{PurposePortReachability, ObsProbeBudgetExpired, Classification{Measured, Fail, ReasonBudgetExpired}},
	// No hub was supplied, so the attempt was not made: not measured, and
	// never a blocked hub.
	{PurposePortReachability, ObsHubInputMissing, Classification{NotMeasured, Indeterminate, ReasonInputMissingHub}},
	// A reachability probe may also be unable to attempt its dial at all: the run
	// may never have been given a dialer, or the dial seam may have refused. Both
	// are attempts that were not made, with the two reason codes design §5.1
	// obligation 2 requires to stay distinguishable, and the same two rows are
	// what every later reachability probe needs for the deny-all test default.
	{PurposePortReachability, ObsCapabilityExcluded, Classification{NotMeasured, Indeterminate, ReasonCapabilityExcluded}},
	{PurposePortReachability, ObsCommandDenied, Classification{NotMeasured, Indeterminate, ReasonCommandDenied}},

	// Name resolution: an authoritative negative is a measurement of the name;
	// a resolver that did not answer measured nothing. The two absences a resolver
	// can produce are the same two a dial can: a capability the run was never given
	// and a seam that refused. They are rows here as well as under reachability
	// because the name question is asked by probes that have no port to blame, and
	// obligation 2 requires the two to stay distinguishable wherever they happen.
	{PurposeNameResolution, ObsResolverAuthoritativeNegative, Classification{Measured, Fail, ReasonDNSNoSuchHost}},
	{PurposeNameResolution, ObsResolverUnavailable, Classification{Unresolved, Indeterminate, ReasonDNSUnresolved}},
	{PurposeNameResolution, ObsCapabilityExcluded, Classification{NotMeasured, Indeterminate, ReasonCapabilityExcluded}},
	{PurposeNameResolution, ObsCommandDenied, Classification{NotMeasured, Indeterminate, ReasonCommandDenied}},

	// The runner's own facts are facts about the run, not answers to the
	// probe's question, so they classify the same way whatever was asked.
	{"", ObsProbeIgnoredBudget, Classification{Unresolved, Indeterminate, ReasonProbeTimeout}},
	{"", ObsRunCancelled, Classification{Unresolved, Indeterminate, ReasonRunCancelled}},
	{"", ObsRunBudgetExhausted, Classification{Unresolved, Indeterminate, ReasonRunBudgetExceeded}},

	// D8's narrow UDP question. Only a reply is a measurement; silence is
	// ambiguity by construction and is never reported as a block.
	{PurposeUDPReachability, ObsUDPResponse, Classification{Measured, Pass, ReasonUDPResponseReceived}},
	{PurposeUDPReachability, ObsUDPSilence, Classification{Unresolved, Indeterminate, ReasonUDPSilence}},
	{PurposeUDPReachability, ObsUDPUnreachable, Classification{Measured, Fail, ReasonUDPUnreachable}},
	{PurposeUDPReachability, ObsUDPOtherError, Classification{Unresolved, Indeterminate, ReasonUDPErrorUnclassified}},
	// A datagram question can also be unable to attempt its exchange at all: no
	// packet dialer was injected for the run, or the packet seam refused. Both are
	// attempts that were not made, with the two reason codes obligation 2 requires
	// to stay distinguishable, and neither is a claimed absence of a reply.
	{PurposeUDPReachability, ObsCapabilityExcluded, Classification{NotMeasured, Indeterminate, ReasonCapabilityExcluded}},
	{PurposeUDPReachability, ObsCommandDenied, Classification{NotMeasured, Indeterminate, ReasonCommandDenied}},

	// TLS on the wire: the issuer and the verification code are part of the
	// measurement, and only a verified chain whose issuer is a declared
	// expected publisher is a pass.
	{PurposeTLSCertificate, ObsTLSVerified, Classification{Measured, Pass, ReasonOK}},
	{PurposeTLSCertificate, ObsTLSVerifyFailed, Classification{Measured, Fail, ReasonTLSVerifyFailed}},
	// A verified chain whose issuer is outside the declared set records a divergence the
	// run cannot settle: a publisher that rotated and an interception look identical here,
	// so the fact is an absence — the run attempted the question and could not answer it —
	// rather than a failure. The wording records the observed publisher and the declared
	// set it was compared against, and states the run cannot distinguish the two.
	{PurposeTLSCertificate, ObsTLSIssuerUnexpected, Classification{Unresolved, Indeterminate, ReasonTLSIssuerUnexpected}},
	{PurposeTLSCertificate, ObsTLSHandshakeError, Classification{Unresolved, Indeterminate, ReasonTLSHandshakeUnresolved}},
	// A chain measurement can also be unable to run at all: no verifier was
	// injected for the run, or the verifier seam refused. Both are attempts that
	// were not made, never a rejected chain and never a claimed interception, and
	// the two reason codes stay distinguishable (design §5.1 obligation 2).
	{PurposeTLSCertificate, ObsCapabilityExcluded, Classification{NotMeasured, Indeterminate, ReasonCapabilityExcluded}},
	{PurposeTLSCertificate, ObsCommandDenied, Classification{NotMeasured, Indeterminate, ReasonCommandDenied}},

	// The local trust store: a Linux pool can accept or reject the chain; a
	// verifier that cannot answer, or one an environment override bypasses, is
	// never a pass.
	{PurposeTLSTrustStore, ObsTLSVerified, Classification{Measured, Pass, ReasonOK}},
	{PurposeTLSTrustStore, ObsTrustStoreRejectsChain, Classification{Measured, Fail, ReasonTrustStoreRejectsChain}},
	{PurposeTLSTrustStore, ObsTrustStoreVerifierUnavailable, Classification{Unresolved, Indeterminate, ReasonTrustStorePlatformUnavailable}},
	{PurposeTLSTrustStore, ObsTrustStoreVerifierBypassed, Classification{Unresolved, Indeterminate, ReasonTrustStoreOverridePlatformBypass}},
	// The trust-store question is asked through the same handshake the chain
	// question is, so it has the same two absences and the same answer to an
	// answerless handshake. None of the three is a verdict about the pool: a
	// handshake that produced nothing measured nothing, and the two ways the
	// attempt can go unmade stay distinguishable (design §5.1 obligation 2).
	{PurposeTLSTrustStore, ObsTLSHandshakeError, Classification{Unresolved, Indeterminate, ReasonTLSHandshakeUnresolved}},
	// A platform the seam could not identify is the same fact here as it is for
	// `local.env`: the signals were read and matched nothing, so no platform is
	// assumed (R-HR-29). Without this row the fact would fall through to the total
	// row and be reported as an internal failure, which would name this package as
	// broken rather than name the missing platform signal.
	{PurposeTLSTrustStore, ObsPlatformSignalsUnknown, Classification{Unresolved, Indeterminate, ReasonPlatformUnknown}},
	{PurposeTLSTrustStore, ObsCapabilityExcluded, Classification{NotMeasured, Indeterminate, ReasonCapabilityExcluded}},
	{PurposeTLSTrustStore, ObsCommandDenied, Classification{NotMeasured, Indeterminate, ReasonCommandDenied}},

	// The local sshd: the presence of the binary, the service state and the
	// configuration in force are measured facts, and each has a positive and a
	// negative row. The two absences design §5.1 prints last — a capability this
	// slice never attempts and a denied command — are attempts that were not made,
	// never failures.
	//
	// The not-running row reuses `sshd_absent` deliberately: the closed reason set
	// holds no code for "installed but not running", and adding one is a contract
	// change (design §3.5). `sshd_absent` is the vocabulary's own "this node is not
	// serving sshd" code, and the observation's label and detail name which half
	// was missing, so a consumer can still tell a stopped service from an absent
	// binary. A dedicated code is the honest home for it; the gap is recorded in
	// PR 6's apply evidence.
	{PurposeSSHDConfiguration, ObsSSHDBinaryPresent, Classification{Measured, Pass, ReasonOK}},
	{PurposeSSHDConfiguration, ObsSSHDBinaryAbsent, Classification{Measured, Fail, ReasonSSHDAbsent}},
	{PurposeSSHDConfiguration, ObsSSHDServiceRunning, Classification{Measured, Pass, ReasonOK}},
	{PurposeSSHDConfiguration, ObsSSHDServiceNotRunning, Classification{Measured, Fail, ReasonSSHDAbsent}},
	{PurposeSSHDConfiguration, ObsSSHDConfigMatches, Classification{Measured, Pass, ReasonOK}},
	{PurposeSSHDConfiguration, ObsSSHDConfigDivergent, Classification{Measured, Fail, ReasonSSHDConfigDivergence}},
	{PurposeSSHDConfiguration, ObsCapabilityExcluded, Classification{NotMeasured, Indeterminate, ReasonCapabilityExcluded}},
	{PurposeSSHDConfiguration, ObsCommandDenied, Classification{NotMeasured, Indeterminate, ReasonCommandDenied}},

	// This machine: signals matching one of the supported classifications are a
	// measured positive, signals matching none are unresolved, and a classified
	// native Windows node is this tool's measured negative answer rather than an
	// error.
	//
	// The classified row is the positive counterpart design §5.1 does not print:
	// §5.3 requires `local.env` to pass with a healthy environment, and a measured
	// pass is only expressible if the table has a row for it. It is declared here,
	// with the rest of the table, rather than by the probe that first needs it.
	{PurposePlatformClassification, ObsPlatformSignalsClassified, Classification{Measured, Pass, ReasonOK}},
	{PurposePlatformClassification, ObsPlatformSignalsUnknown, Classification{Unresolved, Indeterminate, ReasonPlatformUnknown}},
	{PurposePlatformClassification, ObsNodePlatformUnsupported, Classification{Measured, Fail, ReasonNodePlatformUnsupported}},

	// An internal failure the probe itself noticed is a fact like any other:
	// it is universal because a failure is not an answer to any declared
	// question, and its unresolved outcome is never a pass.
	{"", ObsInternalFailure, Classification{Unresolved, Indeterminate, ReasonInternalError}},

	// Total row: nothing recognised the observation, so it is reported as an
	// internal failure instead of being guessed at. Never a pass.
	{"", "", Classification{Unresolved, Indeterminate, ReasonInternalError}},
}

// Classify applies the ordered classification table to one raw observation and
// returns the triple the probe's declared purpose assigns to it.
//
// It is a pure function of its two inputs, so two probes asking the same
// declared question can never classify one observable two ways. A purpose and
// observable pair the table does not document falls through to the table's
// total last row and is reported as an internal failure.
func Classify(purpose Purpose, raw RawObservation) Classification {
	for _, row := range classificationTable {
		if row.purpose != "" && row.purpose != purpose {
			continue
		}
		if row.observable != "" && row.observable != raw.Kind {
			continue
		}
		return row.class
	}
	// Unreachable while the table's last row is total. Kept so this function
	// stays total even if that row is later edited away: the zero
	// Classification would otherwise leak an empty resolution and reason.
	return Classification{Unresolved, Indeterminate, ReasonInternalError}
}

// Observe builds the observation a probe reports for one raw fact. It is the
// only path from a raw fact to a reason code: the table chooses the triple and
// the operating system's wording is carried into Detail unchanged, so no
// classification can depend on how an error was phrased (R-HR-07).
//
// A fact the table does not recognise still yields a valid observation: one
// that is unresolved, indeterminate and carries the internal-error reason.
func Observe(label, target string, purpose Purpose, raw RawObservation) Observation {
	class := Classify(purpose, raw)
	return Observation{
		Label:      label,
		Target:     target,
		Resolution: class.Resolution,
		Verdict:    class.Verdict,
		Reason:     class.Reason,
		Detail:     raw.Wording,
	}
}
