package probe

// ReasonCode is the stable machine-readable outcome of one observation
// (R-HR-07). The set of legal codes is closed; every code is declared in this
// file and AllReasonCodes() returns them in declaration order (design §3.5).
//
// A code is chosen by the classification table (classify.go) from what a probe
// observed — never parsed out of an error string — so the same failure keeps
// the same code across Go versions and operating systems while the verbatim
// wording stays in Observation.Detail. Adding or renaming a code is a contract
// change: the JSON payload, the human projection and docs/diagnosis-report.md
// all quote the set.
type ReasonCode string

const (
	// ReasonOK is a measurement that answered its declared question
	// positively.
	ReasonOK ReasonCode = "ok"
	// ReasonConnRefused is a TCP connection the far end refused.
	ReasonConnRefused ReasonCode = "conn_refused"
	// ReasonConnReset is a TCP connection reset after it was established.
	ReasonConnReset ReasonCode = "conn_reset"
	// ReasonBudgetExpired is the probe's own dial budget expiring on a
	// reachability question: the declared port never answered, which for that
	// declared question is the measurement itself.
	ReasonBudgetExpired ReasonCode = "budget_expired"
	// ReasonDNSNoSuchHost is an authoritative resolver negative: the name does
	// not exist, which is a fact about the name and not about the network.
	ReasonDNSNoSuchHost ReasonCode = "dns_no_such_host"
	// ReasonDNSUnresolved is a resolver that did not answer at all (timeout,
	// SERVFAIL, no resolver). It is not a measurement of the destination.
	ReasonDNSUnresolved ReasonCode = "dns_unresolved"
	// ReasonBannerNotSSH is a reachable port whose banner is not an SSH
	// identification string: something answered, but not an SSH server.
	ReasonBannerNotSSH ReasonCode = "banner_not_ssh"
	// ReasonProbeTimeout is a probe that ignored its budget and never
	// returned. It is classified by the runner, never by the probe.
	ReasonProbeTimeout ReasonCode = "probe_timeout"
	// ReasonRunCancelled is a run context that was cancelled while probes were
	// in flight. Runner-classified.
	ReasonRunCancelled ReasonCode = "run_cancelled"
	// ReasonRunBudgetExceeded is the run's global budget being exhausted.
	// Runner-classified.
	ReasonRunBudgetExceeded ReasonCode = "run_budget_exceeded"
	// ReasonUDPResponseReceived is a UDP reply arriving from the edge before
	// the probe budget expired. It claims only that the datagram was not
	// silently dropped.
	ReasonUDPResponseReceived ReasonCode = "udp_response_received"
	// ReasonUDPSilence is a UDP socket that produced neither a reply nor an
	// error: ambiguous by construction and never a block.
	ReasonUDPSilence ReasonCode = "udp_silence"
	// ReasonUDPUnreachable is an ICMP port-unreachable surfaced by the socket,
	// which is a definite negative.
	ReasonUDPUnreachable ReasonCode = "udp_unreachable"
	// ReasonUDPErrorUnclassified is any other UDP socket error: ambiguous.
	ReasonUDPErrorUnclassified ReasonCode = "udp_error_unclassified"
	// ReasonTLSVerifyFailed is a TLS chain that failed verification, with the
	// verification code in the detail.
	ReasonTLSVerifyFailed ReasonCode = "tls_verify_failed"
	// ReasonTLSIssuerUnexpected is a verified chain whose issuer is not the
	// declared expected publisher for the target. The wording says the issuer
	// is not in the declared expected set and never accuses the network.
	ReasonTLSIssuerUnexpected ReasonCode = "tls_issuer_unexpected"
	// ReasonTLSHandshakeUnresolved is a TLS handshake error that is neither a
	// verification failure nor an unexpected issuer: ambiguous.
	ReasonTLSHandshakeUnresolved ReasonCode = "tls_handshake_unresolved"
	// ReasonTrustStoreRejectsChain is the local trust pool rejecting the
	// chain: a definite negative.
	ReasonTrustStoreRejectsChain ReasonCode = "truststore_rejects_chain"
	// ReasonTrustStorePlatformUnavailable is a platform verifier that cannot
	// answer at all (macOS system roots). The capability was attempted and is
	// unusable, so the result is unresolved and carries the limitation.
	ReasonTrustStorePlatformUnavailable ReasonCode = "truststore_platform_unavailable"
	// ReasonTrustStoreOverridePlatformBypass is an SSL_CERT_FILE/SSL_CERT_DIR
	// override that bypasses the platform verifier, so its answer would not
	// mean what the probe claims. Unresolved, never a false pass.
	ReasonTrustStoreOverridePlatformBypass ReasonCode = "truststore_override_platform_bypass"
	// ReasonSSHDAbsent is an sshd binary that is not installed. It is a
	// measured negative, and the detail states that installing it is a later
	// slice.
	ReasonSSHDAbsent ReasonCode = "sshd_absent"
	// ReasonSSHDConfigDivergence is a written configuration that differs from
	// the effective one, with both configurations in the detail.
	ReasonSSHDConfigDivergence ReasonCode = "sshd_config_divergence"
	// ReasonCapabilityExcluded is a capability this slice never attempts
	// (for example `sshd -T` with no production command runner wired). The
	// attempt was not made by design, so the observation is not measured.
	ReasonCapabilityExcluded ReasonCode = "capability_excluded"
	// ReasonCommandDenied is a command seam that denied the execution. The
	// attempt was not made, so the observation is not measured and the reason
	// names the capability.
	ReasonCommandDenied ReasonCode = "command_denied"
	// ReasonInputMissingHub is a run with no hub address supplied: the attempt
	// was not made for lack of input, which is never a blocked hub.
	ReasonInputMissingHub ReasonCode = "input_missing_hub"
	// ReasonPlatformUnknown is a set of platform signals matching no supported
	// classification. It is ambiguous and no platform is assumed.
	ReasonPlatformUnknown ReasonCode = "platform_unknown"
	// ReasonNodePlatformUnsupported is a node classified as native Windows.
	// The refusal is this tool's measured negative answer, not a usage error,
	// and the detail names WSL2 as the Windows path this tool handles today.
	ReasonNodePlatformUnsupported ReasonCode = "node_platform_unsupported"
	// ReasonInternalError is an unexpected internal failure, including a fact
	// the classification table does not recognise and a probe that reported no
	// observation at all. It is unresolved, never a pass.
	ReasonInternalError ReasonCode = "internal_error"
)

// allReasonCodes is the declaration order, which is also the order
// AllReasonCodes() returns and the order docs/diagnosis-report.md documents.
var allReasonCodes = []ReasonCode{
	ReasonOK,
	ReasonConnRefused,
	ReasonConnReset,
	ReasonBudgetExpired,
	ReasonDNSNoSuchHost,
	ReasonDNSUnresolved,
	ReasonBannerNotSSH,
	ReasonProbeTimeout,
	ReasonRunCancelled,
	ReasonRunBudgetExceeded,
	ReasonUDPResponseReceived,
	ReasonUDPSilence,
	ReasonUDPUnreachable,
	ReasonUDPErrorUnclassified,
	ReasonTLSVerifyFailed,
	ReasonTLSIssuerUnexpected,
	ReasonTLSHandshakeUnresolved,
	ReasonTrustStoreRejectsChain,
	ReasonTrustStorePlatformUnavailable,
	ReasonTrustStoreOverridePlatformBypass,
	ReasonSSHDAbsent,
	ReasonSSHDConfigDivergence,
	ReasonCapabilityExcluded,
	ReasonCommandDenied,
	ReasonInputMissingHub,
	ReasonPlatformUnknown,
	ReasonNodePlatformUnsupported,
	ReasonInternalError,
}

// AllReasonCodes returns the closed reason-code set in declaration order. The
// returned slice is a copy, so a caller cannot reorder or shrink the contract;
// callers must not assume otherwise.
func AllReasonCodes() []ReasonCode {
	out := make([]ReasonCode, len(allReasonCodes))
	copy(out, allReasonCodes)
	return out
}
