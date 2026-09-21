# Diagnosis report contract

`herdr-reach doctor` produces two projections of one run: a machine-readable JSON document on
standard output with `--json`, and a human report on standard error. This document is the contract
both projections quote: the exit codes, the stream split, the payload shape, and the two closed
vocabularies — reason codes and rule ids — that travel through the payload and the human report.

The document is not frozen as a schema file, and nothing here promises cross-machine compatibility
(R-HR-NF-06). The reason-code table and the rule-id table are asserted against the code's own
declarations by `internal/report/docs_test.go`: set equality with `probe.AllReasonCodes()` and
`diagnosis.AllRuleIDs()` in both directions, so a code or rule id added to the constants without a
row here — or a row without a constant — fails the suite.

## Exit codes

| Code | Meaning | Evidence that decides it |
| --- | --- | --- |
| `0` | Measurement completed — including "no transport viable" | every probe resolved (no unresolved observation) |
| `1` | Run incomplete — at least one probe was attempted and resolved unresolved | `run.completeness == "incomplete"` |
| `2` | Usage or internal error — no diagnosis is presented as completed; standard output stays empty | bad flag or flag value, unknown `--target` probe name, unusable `--hub` address, JSON write failure |

Read `run.completeness` and `run.unresolved`, never the code alone: exit `1` is common on a real
network and it is a result, not a failed run.

- A not-measured observation alone never changes the exit code, but it always appears in
  `run.not_measured` so the coverage gap stays visible.
- A run **with no `--hub`** names the hub row in `run.not_measured`, and that alone does not change
  the exit code. On a real network the run commonly exits `1` anyway: `tls.interception` and
  `egress.quic` are attempted and left unresolved — measured on 2026-09-21 with a Linux/arm64 VPS
  and on the macOS CI runner. No platform is documented as exiting `0` on a default run; the fields
  to read are `run.completeness` and `run.unresolved`.
- A default live run on **macOS exits `1`** for one more reason: `tls.truststore` is attempted and
  unresolved there by decision (RG-3). The enumeration half of that limitation was measured on a
  real macOS runner rather than assumed; the keychain half rests on `crypto/x509`'s own source,
  because the measurement performs no handshake (issue #81).
- A TLS chain that verified with an issuing-CA organization outside the declared expected set is
  an unresolved observation as of 2026-09-20 (see the `tls_issuer_unexpected` row below): the run
  attempted a question it could not settle, so it is incomplete and exits `1`. The compared value
  is that organization alone; the chain's anchor is reported beside the observed issuer as
  evidence and never decides the comparison (issue #78). Where a plugin host reads a non-zero
  action status as a failed action — the interaction issue #58 tracks for Herdr — a publisher
  divergence now reaches that reading.
- The constants are `ExitOK`, `ExitIncomplete` and `ExitUsage` in `internal/doctor/exit.go`. The
  assertion that those constants match this table lives in `internal/doctor/doctor_test.go`
  (PR 18), not in `internal/report/docs_test.go`: `internal/doctor` imports `internal/report`, and
  the reverse import would invert the dependency direction.

## Streams

- With `--json`, standard output carries exactly one machine-readable document and nothing else: no
  prose, progress line, warning or banner is mixed into it. All human-facing output goes to
  standard error.
- Without `--json`, standard output stays empty. The human report still goes to standard error.
- The split is what makes piping the machine-readable document into another program safe. On exit
  `2` no diagnosis is presented as completed and standard output stays empty.
- `--version` prints the tool's version to standard output and exits `0`; it is not a run and carries
  no diagnosis. A help request (`-h`/`--help`) prints usage to standard error and exits `0`.

## Payload shape

The document is one ordered JSON object. `schema_version` is `"1"`. Every key below is always
present; an empty collection serialises as `[]` rather than `null`. The only nullable fields are
`targets.hub`, `probes[].target` and `probes[].observations[].target`: each is JSON `null` exactly
when the run carried no target, never an empty string (design D3).

| Level | Keys |
| --- | --- |
| document | `schema_version`, `generated_at`, `tool`, `run`, `targets`, `probes`, `findings`, `open_questions`, `node`, `transports` |
| `tool` | `name`, `version` |
| `run` | `completeness`, `unresolved`, `not_measured`, `concurrency`, `run_budget_ms` |
| `targets` | `hub`, `declared` |
| `targets.declared[]` | `probe`, `target`, `protocol` |
| `probes[]` | `name`, `kind`, `target`, `verdict`, `resolution`, `reason`, `detail`, `elapsed_ms`, `observations` |
| `probes[].observations[]` | `label`, `target`, `verdict`, `resolution`, `reason`, `detail` |
| `findings[]` | `question`, `rule`, `conclusion`, `depends_on` |
| `open_questions[]` | `question`, `needed_states` |
| `node` | `platform`, `arch`, `refused`, `note` |
| `transports[]` | `name`, `viable`, `reason`, `requires`, `notes` |
| `transports[].requires[]` | `kind`, `satisfied`, `detail` |

- `probes` follow the registry order; `observations` follow the order the probe reported them;
  `transports` are sorted by name. No map iteration influences the document.
- `elapsed_ms` is an integer number of whole milliseconds, truncated rather than rounded up.
- `generated_at` is RFC3339 UTC, stamped by the run's single injected clock.
- No field is derived from treating an unresolved observation as a `pass`, and no derived boolean
  or summary field marks one as successful.
- One mapping function produces the whole document, so an internal structure change cannot silently
  change the payload.
- This slice freezes no schema file and writes none, and it promises no cross-machine
  compatibility. Promotion to a versioned wire contract later is additive (R-HR-NF-06).

## Reason codes

A reason code is the stable machine-readable outcome of one observation. It is chosen from the
classification table from what a probe observed — never parsed out of an error string — so the same
failure keeps the same code across Go versions and operating systems while the verbatim wording
stays in `detail`. The set is closed.

| Code | Meaning |
| --- | --- |
| `ok` | The observation answered its declared question positively: a measured pass. |
| `conn_refused` | A TCP connection the far end refused. |
| `conn_reset` | A TCP connection reset after it was established. |
| `budget_expired` | The probe's own dial budget expired on a reachability question: the declared port never answered, which for that declared question is the measurement itself. |
| `dns_no_such_host` | An authoritative resolver negative: the name does not exist, which is a fact about the name and not about the network. |
| `dns_unresolved` | The resolver did not answer at all (timeout, SERVFAIL, no resolver): not a measurement of the destination. |
| `banner_not_ssh` | A reachable port whose banner is not an SSH identification string: something answered, but not an SSH server. |
| `probe_timeout` | A probe that ignored its budget and never returned. Classified by the runner, never by the probe. |
| `run_cancelled` | The run context was cancelled while probes were in flight. Runner-classified. |
| `run_budget_exceeded` | The run's global budget was exhausted. Runner-classified. |
| `udp_response_received` | A UDP reply arrived from the edge before the probe budget expired; it claims only that the datagram was not silently dropped. |
| `udp_silence` | A UDP socket that produced neither a reply nor an error: ambiguous by construction, and never a block. |
| `udp_unreachable` | An ICMP port-unreachable surfaced by the socket: a definite negative. |
| `udp_error_unclassified` | Any other UDP socket error: ambiguous. |
| `tls_verify_failed` | A TLS chain that failed verification; the verification code is in `detail`. |
| `tls_issuer_unexpected` | A verified chain whose leaf's issuing-CA organization is not in the declared expected publisher set for the target. The compared value is that organization alone; the chain's anchor (the short name of its topmost certificate, which the local trust store selected) is evidence in `detail` and never decides the comparison (issue #78). The run records the observed issuer, the anchor and the declared set it was compared against and cannot establish whether the difference is a publisher change or an interception, so the observation is unresolved — it makes the run incomplete and exits `1`, and it is never a failure and never a pass. |
| `tls_handshake_unresolved` | A TLS handshake error that is neither a verification failure nor an unexpected issuer: ambiguous. |
| `truststore_rejects_chain` | The local trust pool rejected the chain: a definite negative. |
| `truststore_platform_unavailable` | A platform verifier that cannot answer at all (macOS system roots): the capability was attempted and is unusable. |
| `truststore_override_platform_bypass` | An `SSL_CERT_FILE`/`SSL_CERT_DIR` override bypasses the platform verifier, so its answer would not mean what the probe claims. Never a false pass. |
| `sshd_absent` | No sshd binary is present at the platform's documented path — `/usr/sbin/sshd` on Linux, macOS and WSL2, `C:\Windows\System32\OpenSSH\sshd.exe` on native Windows. Installing it is work owned by a later slice. |
| `sshd_config_divergence` | A written sshd configuration that differs from the configuration in force; both are in `detail`. |
| `capability_excluded` | A capability this slice never attempts (for example `sshd -T` with no production command runner wired): the attempt was not made, and a check that was not attempted never claims an absence. The `local.sshd` service-state question is the platform's own: on the POSIX platforms it is the systemd `is-active` query for the units a distribution ships, run only through an injected command runner; on a native-Windows node this slice declares the platform's own question — the state of the `sshd` service the OpenSSH Server capability installs, which a Get-Service-style query reports — and does not put it to the machine, whether or not a command runner was injected, so a Windows report names that platform question, carries no systemd vocabulary, and claims no service state (issue #79). |
| `command_denied` | A command seam denied the execution: the attempt was not made, and the reason names the capability. |
| `input_missing_hub` | A run with no hub address supplied: the attempt was not made for lack of input, which is never a blocked hub. |
| `platform_unknown` | A set of platform signals matching no supported classification: ambiguous, and no platform is assumed. |
| `internal_error` | An unexpected internal failure, including a fact the classification table does not recognise and a probe that reported no observation at all. Unresolved, never a pass. |

**Adding or renaming a reason code is a contract change.** The payload, the human projection and
this document quote the same closed set.

## Rule ids

A fact question's rule id is derived, not catalogued:
`<PROBE_NAME_UPPERCASED_WITH_UNDERSCORES>_<STATE>`, where the base is the probe whose observation
the rule states and `<STATE>` is one of `PASS`, `FAIL`, `UNRESOLVED` or `NOT_MEASURED`. For example
`EGRESS_HUB_DIRECT_FAIL` is the `egress.hub.direct` failure state and `LOCAL_ENV_PASS` is the
`local.env` pass state. The base is the probe's name and never the question's name: several fact
questions carry a human-facing name of their own (`hub.reachability` answers for
`egress.hub.direct`, `ssh.public_22` for `egress.ssh.known`) while every state of one probe shares
one id base.

A derived question's id is named by hand instead, because its conclusion is about several
observables at once and no single (probe, state) pair derives it: `SSH_DEST_BLOCKED_BY_PUBLIC_SSH`,
`CF_HTTP2_ADVISED_QUIC_FAILED` and the other group ids below. Every registered probe in every
observable state has exactly one rule id, so no observable is left unclassified.

Rule ids appear verbatim in the payload's `findings[].rule` and in the human projection, so a
reader can quote which rule fired.

| Rule id | Question | Meaning |
| --- | --- | --- |
| `LOCAL_ENV_FAIL` | `local.env` | The probe reported a measured failure. |
| `LOCAL_ENV_UNRESOLVED` | `local.env` | The probe was attempted and produced no answer. |
| `LOCAL_ENV_NOT_MEASURED` | `local.env` | The probe was never attempted. |
| `LOCAL_ENV_PASS` | `local.env` | The probe reported a measured pass. |
| `LOCAL_SSHD_FAIL` | `local.sshd.fact` | The probe reported a measured failure. |
| `LOCAL_SSHD_UNRESOLVED` | `local.sshd.fact` | The probe was attempted and produced no answer. |
| `LOCAL_SSHD_NOT_MEASURED` | `local.sshd.fact` | The probe was never attempted. |
| `LOCAL_SSHD_PASS` | `local.sshd.fact` | The probe reported a measured pass. |
| `EGRESS_HUB_DIRECT_FAIL` | `hub.reachability` | The probe reported a measured failure. |
| `EGRESS_HUB_DIRECT_UNRESOLVED` | `hub.reachability` | The probe was attempted and produced no answer. |
| `EGRESS_HUB_DIRECT_NOT_MEASURED` | `hub.reachability` | The probe was never attempted. |
| `EGRESS_HUB_DIRECT_PASS` | `hub.reachability` | The probe reported a measured pass. |
| `EGRESS_SSH_KNOWN_FAIL` | `ssh.public_22` | The probe reported a measured failure. |
| `EGRESS_SSH_KNOWN_UNRESOLVED` | `ssh.public_22` | The probe was attempted and produced no answer. |
| `EGRESS_SSH_KNOWN_NOT_MEASURED` | `ssh.public_22` | The probe was never attempted. |
| `EGRESS_SSH_KNOWN_PASS` | `ssh.public_22` | The probe reported a measured pass. |
| `EGRESS_SSH_443_FAIL` | `ssh.public_443` | The probe reported a measured failure. |
| `EGRESS_SSH_443_UNRESOLVED` | `ssh.public_443` | The probe was attempted and produced no answer. |
| `EGRESS_SSH_443_NOT_MEASURED` | `ssh.public_443` | The probe was never attempted. |
| `EGRESS_SSH_443_PASS` | `ssh.public_443` | The probe reported a measured pass. |
| `EGRESS_CF_7844_FAIL` | `egress.cf.7844` | The probe reported a measured failure. |
| `EGRESS_CF_7844_UNRESOLVED` | `egress.cf.7844` | The probe was attempted and produced no answer. |
| `EGRESS_CF_7844_NOT_MEASURED` | `egress.cf.7844` | The probe was never attempted. |
| `EGRESS_CF_7844_PASS` | `egress.cf.7844` | The probe reported a measured pass. |
| `EGRESS_CF_443_FAIL` | `egress.cf.443` | The probe reported a measured failure. |
| `EGRESS_CF_443_UNRESOLVED` | `egress.cf.443` | The probe was attempted and produced no answer. |
| `EGRESS_CF_443_NOT_MEASURED` | `egress.cf.443` | The probe was never attempted. |
| `EGRESS_CF_443_PASS` | `egress.cf.443` | The probe reported a measured pass. |
| `EGRESS_QUIC_FAIL` | `egress.quic` | The probe reported a measured failure. |
| `EGRESS_QUIC_UNRESOLVED` | `egress.quic` | The probe was attempted and produced no answer. |
| `EGRESS_QUIC_NOT_MEASURED` | `egress.quic` | The probe was never attempted. |
| `EGRESS_QUIC_PASS` | `egress.quic` | The probe reported a measured pass. |
| `TLS_INTERCEPTION_FAIL` | `tls.interception` | The probe reported a measured failure. Reachable through a chain that failed verification (`tls_verify_failed`); since 2026-09-20 a publisher outside the declared expected set is unresolved and fires `TLS_INTERCEPTION_UNRESOLVED` instead. |
| `TLS_INTERCEPTION_UNRESOLVED` | `tls.interception` | The probe was attempted and produced no answer. |
| `TLS_INTERCEPTION_NOT_MEASURED` | `tls.interception` | The probe was never attempted. |
| `TLS_INTERCEPTION_PASS` | `tls.interception` | The probe reported a measured pass. |
| `TLS_TRUSTSTORE_FAIL` | `tls.truststore` | The probe reported a measured failure. |
| `TLS_TRUSTSTORE_UNRESOLVED` | `tls.truststore` | The probe was attempted and produced no answer. |
| `TLS_TRUSTSTORE_NOT_MEASURED` | `tls.truststore` | The probe was never attempted. |
| `TLS_TRUSTSTORE_PASS` | `tls.truststore` | The probe reported a measured pass. |
| `SSH_DEST_BLOCKED_BY_PUBLIC_SSH` | `ssh.destination` | A public SSH measurement on port 22 passed and the hub-directed measurement failed: the protocol is not blocked, so the block is on the hub's address. |
| `SSH_DEST_BLOCKED_BY_PUBLIC_SSH_443` | `ssh.destination` | The same conclusion reached from the public measurement on port 443 instead of port 22. |
| `SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED` | `ssh.destination` | The hub-directed measurement failed while the public counterpart produced no answer or was never made: what the hub failure means for SSH egress is not established, and the conclusion names the public measurement that did not answer. |
| `SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_FAILED` | `ssh.destination` | The hub-directed measurement and both public measurements failed: destination-level and protocol-level blocks are not separable from this run. |
| `SSH_NO_DEST_BLOCK_OBSERVED` | `ssh.destination` | The hub's address accepted a connection beside a public SSH measurement that passed: no destination block was observed. |
| `SSH_DEST_BLOCK_NOT_ASSESSED` | `ssh.destination` | The hub-directed measurement produced no answer or was never made: nothing is claimed about the hub. This is the default live case with no `--hub`. |
| `CF_EDGE_REACHABLE` | `cloudflare.edge` | At least one declared Cloudflare edge endpoint answered on TCP and none failed: the edge is reachable. |
| `CF_EDGE_PARTIAL` | `cloudflare.edge` | Some declared edge endpoints answered and some failed: the declared endpoints disagree. |
| `CF_EDGE_UNREACHABLE` | `cloudflare.edge` | Every measured edge endpoint failed and no endpoint produced an absence: the edge is not reachable from this network. |
| `CF_EDGE_UNRESOLVED` | `cloudflare.edge` | No measured edge success beside at least one attempt that produced no answer: whether the edge is reachable is not established. |
| `CF_HTTP2_ADVISED_QUIC_FAILED` | `cloudflare.http2` | The datagram measurement failed beside a measured TCP edge path: the run recommends forcing the tunnel transport to HTTP/2, and states that it recommends rather than enforces. |
| `CF_HTTP2_ADVISED_QUIC_UNCONFIRMED` | `cloudflare.http2` | The datagram measurement produced no answer or was never made beside a measured TCP edge path: the same recommendation, resting on an unconfirmed datagram path, and the conclusion names the measurement it rests on. |
| `CF_NO_HTTP2_ADVICE_QUIC_USABLE` | `cloudflare.http2` | The datagram measurement drew a reply: no downgrade is recommended, because nothing measured a block. |
| `CF_HTTP2_ADVISORY_NOT_ASSESSED` | `cloudflare.http2` | The edge itself did not answer, was never measured or produced no answer: no HTTP/2 advice is given, because no measured TCP path could carry one. |
| `SSHD_PRESENT_CONFIG_DIVERGENT` | `local.sshd` | The written sshd configuration and the configuration in force were both measured and disagree: the node is present but its configuration diverges. |
| `SSHD_ABSENT` | `local.sshd` | No sshd binary is present at the platform's documented path — `/usr/sbin/sshd` on Linux, macOS and WSL2, `C:\Windows\System32\OpenSSH\sshd.exe` on native Windows — reached only through the binary observation's own label, so a stopped service cannot produce it. The check runs on every supported platform, so an absence measured anywhere is this rule. |
| `SSHD_EFFECTIVE_CONFIG_NOT_MEASURED` | `local.sshd` | The configuration in force was not measured: no claim is made about what configuration is in force. This is the default live case. |
| `SSHD_PRESENT_CONFIGURED` | `local.sshd` | The sshd binary is present and the configuration in force was measured and agrees with the written one. |
| `NODE_PLATFORM_UNKNOWN` | `node.platform` | The platform signals matched no supported classification: no platform is assumed. |
| `NODE_PLATFORM_SUPPORTED` | `node.platform` | The classification and architecture measured for this node; a detection, not a viability decision. For native Windows the support is version-dependent: the platform can host a supported Herdr server as of 0.9.1, and the run does not measure which Herdr version is installed. |

**Adding or renaming a rule id is a contract change.** The rule-id set is the complete vocabulary
of the rule table: every registered probe in every observable state has exactly one derived id, and
each hand-named group id is declared beside its rules.

## Contract changes and the pin record

- Adding or renaming a reason code or a rule id is a contract change. The constants and this
  document are updated in the same change; `internal/report/docs_test.go` refuses one without the
  other, in both directions.
- No schema file is frozen in this change, and no cross-machine compatibility is promised
  (R-HR-NF-06).
- The cloudflared pin statement's wording has exactly one home, in
  `internal/transport/pin_note.go`. Both projections render it from the cloudflare-tunnel
  feasibility notes, and this document does not restate it.
