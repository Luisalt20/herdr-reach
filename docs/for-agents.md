# herdr-reach for agents

## Who this is for

You are an agent. A human asked you to make a machine reachable to their Herdr, and you have this
repository's binary. You do not need the tool to provision anything — you can execute — but you
cannot measure a network you are not on, and no document can tell you which of the four transports
the network in front of you permits. That is what `herdr-reach doctor` produces. **The tool carries
the machine-specific measurement, this document carries the general playbook, and only the
combination acts.** Read this file, run the tool on the machine in question, and work from its
output. Human-facing installation and prerequisites live in the [README](../README.md); what the
shipped binary does and does not do is in [Testing herdr-reach](beta-testing.md); the payload contract
and its closed vocabularies are in [Diagnosis report contract](diagnosis-report.md).

## How to run it

```bash
herdr-reach doctor --hub <hub address> --json > report.json
```

- `doctor` is the only command in this slice, and it measures the machine it runs on. It
  changes nothing (see "What the tool will not do").
- `--hub <host[:port]>` always names **the other side** of the link: run it on the node with the
  hub's address, on the hub with the node's address. Port 22 is the default. There is no role flag;
  the address is what selects the profile.
- **Stream split:** with `--json` the machine-readable document is the only thing on standard
  output; the human report goes to standard error. Without `--json`, standard output stays empty.
  The redirection above is therefore safe.
- **Exit codes:**

| Code | Meaning |
|:---|:---|
| `0` | The measurement completed — including "no transport is viable". A measured negative is still a result. |
| `1` | The run is incomplete: at least one probe was attempted and produced no answer. **This is a result, not a failure.** Use the document; do not discard it. |
| `2` | Usage or internal error. No diagnosis is presented and standard output is empty; fix the invocation and run again. |

An exit `1` is common and is a result, not a failed run: on a real network `tls.interception` and
`egress.quic` are attempted and left unresolved (measured on 2026-09-21 on a Linux/arm64 VPS and
on the macOS CI runner), and a default macOS run adds `tls.truststore`, attempted and unresolved
there by decision. The enumeration half of that macOS limitation was measured on a real macOS
runner, not assumed; the keychain half rests on `crypto/x509`'s source because the measurement
performs no handshake (issue #81). A Linux run with no `--hub` names the hub measurement as a
coverage gap and answers the rest. Read `run.completeness` and `run.unresolved`, not the code
alone.

## How to read the payload

One run produces one JSON object. Its levels, and what each is for:

```
schema_version   "1"
generated_at     RFC3339 UTC timestamp
tool             {name, version}
run              {completeness, unresolved[], not_measured[], concurrency, run_budget_ms}
targets          {hub, declared[]}
probes[]         {name, kind, target, verdict, resolution, reason, detail, elapsed_ms, observations[]}
findings[]       {question, rule, conclusion, depends_on[]}
open_questions[] {question, needed_states[]}
node             {platform, arch, refused, note}
transports[]     {name, viable, reason, requires[], notes[]}
```

`probes` has one row per registered probe — ten in this build — in registry order. `targets.declared`
is the resolved dialed set: eleven rows with a `--hub` on the default port, ten without it.
`findings` has one row per question the rule table answered (the table declares fifteen), and
`open_questions` has one row per question no rule matched: usually empty, but read it and never
assume it. Every key is always present; the only nullable fields are `targets.hub`, `probes[].target`
and `probes[].observations[].target`, and null there means "the run carried no target", never an
empty string.

`schema_version` is `"1"`. No schema file is frozen and no cross-machine compatibility is promised,
so a consumer must tolerate additive change. `generated_at` is RFC3339 UTC from the run's single
clock: it is when this measurement was taken, which is what makes the document evidence rather than
a claim about now. `tool.name` and `tool.version` identify the build that produced it — compare the
version with `--version` before acting on a document of unknown origin.

### `run` — whether the picture can be trusted at all

- `completeness`: `complete` means no attempted probe was left unanswered. `incomplete` means at
  least one was attempted and produced no answer. **Before any conclusion, name every entry in
  `run.unresolved[]`**; a conclusion that depends on an unresolved probe is not available. A
  complete run can still report failures — completeness grades the answers, not the network.
- `unresolved[]`: the probes with at least one attempted observation that produced no answer, one
  entry per probe. These make the run incomplete and exit `1`.
- `not_measured[]`: probes with at least one observation that was never attempted. A coverage gap,
  not a blocked target, and never a reason to call the run incomplete.
- `concurrency` and `run_budget_ms`: the effective bounds the run used, not what a caller left
  unset. When a probe carries `run_budget_exceeded`, this ceiling is why.

### `targets` — what was actually dialed

- `hub`: the resolved hub address in `host:port` form, or `null` when no hub was supplied.
- `declared[]`: one row per declared target — `probe`, `target`, `protocol` — in declaration order.
  This is the exact set the run dials; an endpoint that is not here was not measured. The property
  the payload supports is "the dialed set equals the declared set".

### `probes` — the facts

Each row carries `name`, `kind` (`local`, `egress`, `tls`, `proto`), `target`, `verdict`,
`resolution`, `reason`, `detail`, `elapsed_ms`, and `observations[]`.

- `verdict`: `pass`, `fail` or `indeterminate`, the worst-of reduction over the row's observations.
  `resolution` says how far the measurement got: `measured`, `unresolved` or `not_measured`. A fail
  beside an absence is reported as a fail and explained by the failing observation, never by the
  sibling it could not measure.
- `target`: null exactly when the measurement carried none; otherwise a host and port you can dial
  by hand.
- `reason`: **the stable code to branch on.** The set is closed and documented in
  [Diagnosis report contract](diagnosis-report.md#reason-codes); never parse `detail`, which is
  verbatim operating-system wording. What the codes license:
  - `ok` — the declared question was answered positively.
  - Definite negatives: `conn_refused`, `conn_reset`, `budget_expired` (on a declared
    reachability question, no answer inside the budget *is* the measurement), `dns_no_such_host`
    (a fact about the name, not the network), `banner_not_ssh` (something answered, but not an
    SSH server), `udp_unreachable`, `tls_verify_failed`, `truststore_rejects_chain`,
    `sshd_absent`, `sshd_config_divergence`.
  - Absences: `input_missing_hub` (no hub was supplied), `capability_excluded` and
    `command_denied` (the attempt was not made by this slice's design). These are never blocks.
  - Ambiguities, all `unresolved`: `dns_unresolved`, `udp_silence` (a UDP socket that produced
    neither a reply nor an error), `udp_error_unclassified`, `tls_issuer_unexpected` (a verified
    chain whose leaf's issuing-CA organization is outside the declared expected set; the chain's
    anchor is reported beside it as evidence and never compared, so a publisher change and an
    interception look identical here), `tls_handshake_unresolved`, `truststore_platform_unavailable`,
    `truststore_override_platform_bypass`, `platform_unknown`, `probe_timeout`, `run_cancelled`,
    `run_budget_exceeded`, `internal_error`.
  - `udp_response_received` claims only that a datagram was not silently dropped; it is not a claim
    that QUIC works.
  - Native Windows is a supported classification as of Herdr 0.9.1: `local.env` reports `ok`, and
    the classification's own detail — which the `node.platform` conclusion quotes — carries the
    caveat that the run does not measure which Herdr version is installed.
- `observations[]`: the per-fact rows (`label`, `target`, `verdict`, `resolution`, `reason`,
  `detail`) in the order the probe reported them. Read these when declared endpoints can disagree:
  one edge region answering beside another failing is a split the aggregate verdict hides.

### `findings` — the conclusions the tool derived

Each row is `question`, `rule`, `conclusion`, `depends_on[]`.

- `rule` says which conclusion fired. A fact question's id is derived as
  `<PROBE_NAME_UPPERCASED_WITH_UNDERSCORES>_<STATE>`, with `<STATE>` one of `PASS`, `FAIL`,
  `UNRESOLVED` or `NOT_MEASURED` — `EGRESS_HUB_DIRECT_FAIL` is the hub-directed measurement
  failing. Derived group questions carry hand-named ids (`SSH_DEST_BLOCKED_BY_PUBLIC_SSH`,
  `CF_EDGE_PARTIAL`, and so on). Every registered probe in every observable state has exactly one
  rule id.
- `depends_on[]` names the probes the conclusion rests on, so a finding is traceable to the
  measurements that produced it.
- `conclusion` is prose for a human. Do not parse it; read the rule id and the probes' reasons.
- If a question is absent from `findings`, look for it in `open_questions[]`: it is
  `{question, needed_states[]}` — no rule matched, so **no conclusion exists**, and `needed_states`
  names the observable states the table needed. Do not supply one yourself.

### `node` — what this machine is

- `platform` and `arch`: the classification and architecture measured for the node. A run that never
  classified the machine reports the unknown identity rather than inventing one. `windows-native`
  means the platform can host a supported Herdr server **as of 0.9.1**; the tool does not measure
  which Herdr version is installed, so a node running an older server is classified supported here
  while being unable to host a saved-machine connection.
- `refused`: always `false`. The field remains in the payload shape for compatibility, but native
  Windows is a supported classification as of Herdr 0.9.1, so nothing refuses a platform. Do not
  branch on it. The provisioning slices still do not cover native Windows, so WSL2 remains the
  Windows path this tool provisions today.
- `note`: the classification's own detail, verbatim. A platform the signals could not classify is
  reported as unknown with no platform assumed. For `windows-native`, `note` carries the version
  caveat.

### `transports` — the decision surface

Each row is `name`, `viable`, `reason`, `requires[]`, `notes[]`. The four candidates are
`cloudflare-tunnel`, `direct-ssh`, `reverse-ssh` and `tailscale`.

- `viable` is a strict boolean: this run measured **every** requirement satisfied **from where the
  tool ran**. It does not mean the transport is configured, working or persistent, and a `true` is
  never a promise of success.
- `reason` is non-empty in both cases and names the highest-precedence fact behind the verdict.
- `requires[]` rows are `kind`, `satisfied`, `detail`:
  - `satisfied: false` means the prerequisite was not supplied or measured; `detail` says what to
    supply or fix. This is the row to act on.
  - `satisfied: true` beside `viable: false` means the prerequisite itself is present but a
    measured block rejects the transport — the hub-address row works this way: the address was
    supplied and dialed, and the measurement rejected it. Supplying the input again will not clear
    it. A measured block also appears among the transport's `notes[]`, and the `reason` always
    names it.
  - The five kinds are closed: `hostname` (a hostname in the user's Cloudflare account),
    `zone_membership` (membership in the zone that owns its domain), `third_party_permission` (a
    third-party tunnel adapter permitted on this machine), `hub_address` (the address supplied with
    `--hub`), `sshd_effective_config` (an sshd whose configuration in force is the written one).
- `notes[]`: the measured observations the verdict rests on, what to supply for an unsatisfied row,
  the HTTP/2 advice when the run has one, and the tool's own pin statement for the `cloudflared`
  client. Read the pin text from here; do not retype a version from memory.

## The decision procedure

Run this in order. Nothing below replaces a measurement.

1. **Run on the node first**, with the hub's address:
   `herdr-reach doctor --hub <hub address> --json > report.json`.
2. **Check `run.completeness`.** If `incomplete`, read `run.unresolved[]`, name those probes, and
   either re-run to settle them or hold back every conclusion that depends on them. An unresolved
   probe is not a pass.
3. **Read `node`.** `refused` is always `false` in this build: native Windows is a supported
   classification as of Herdr 0.9.1, so nothing is refused. Read `platform` and `note` for the
   classification and its caveat; `platform_unknown` means no platform was assumed, and a
   `windows-native` node is classified supported while the provisioning slices still do not cover
   it.
4. **Find the viable transport** — the row with `viable: true`. If one exists, that is the
   transport. Read its `notes[]` for the measurement it rests on and its `requires[]` rows (all
   satisfied) for what it assumes.
5. **If none is viable, choose by failure mode, not preference.** A transport rejected by a measured
   block is not the candidate to satisfy — the block does not disappear by supplying input. The
   candidate is the one whose unsatisfied `requires[]` rows are things the user or you can supply or
   fix. Read each row's `detail` for exactly what.
6. **Supply what the rows name, then re-run** so the change is measured. The tool never acts; a
   requirement satisfied outside it stays unmeasured by it until the next run.
7. **Never conclude from expectation.** If the run did not measure something, say that;
   `not_measured` and `unresolved` are answers about the run, not claims about the network. When the
   measurement disagrees with the human's expectation, the measurement is the evidence.

## Playbook per transport

What follows is derived from the requirement rows the transports declare in code, so the document
and the payload cannot disagree about what is missing. In every case: `viable` means measured from
where the tool ran, not configured.

### `direct-ssh`

- **What the row means.** The hub accepted a TCP connection from where the tool ran, on the address
  supplied with `--hub`. It is the cheapest path — no third party, no domain.
- **Requires.** One row: `hub_address`. Viable exactly when the hub-directed measurement passed.
- **How to read a non-viable row.** `fail` means the address was dialed and did not accept; the
  reason names the measured target, its port and the reason code. `unresolved` means the attempt
  produced no answer — no rejection is claimed, and this is not a decision. `not_measured` with
  `input_missing_hub` means no hub was supplied; run again with `--hub`.
- **What to supply.** Nothing beyond a hub address that accepts TCP from the node. If the address is
  blocked by destination (see "The traps"), stop raising the same address and pick a path that
  reaches the hub through something else.

### `reverse-ssh`

- **What the row means.** The node dials the hub and the hub accepts and forwards. Zero third-party
  dependencies; the cost is that the node must serve SSH to the hub.
- **Requires.** `hub_address` and `sshd_effective_config`: the hub's address must pass its
  measurement, and the node's sshd must exist at the platform's documented location — `/usr/sbin/sshd`
  and `/etc/ssh/sshd_config` on Linux, macOS and WSL2, `C:\Windows\System32\OpenSSH\sshd.exe`
  and `C:\ProgramData\ssh\sshd_config` on native Windows — with its configuration in force equal
  to the written one.
- **How to read a non-viable row.** Precedence in the reason: a measured hub rejection first, then a
  measured sshd rejection (`SSHD_PRESENT_CONFIG_DIVERGENT` or `SSHD_ABSENT`), then an unanswered hub
  attempt, then an unmet requirement. `SSHD_EFFECTIVE_CONFIG_NOT_MEASURED` is the default live case:
  this slice is given no command runner, so it reports the capability as excluded rather than
  reading silence as "configured". That is an absence, not a claim that the sshd is broken — but it
  leaves the row unsatisfied and the transport not viable.
- **What to supply.** An sshd whose configuration in force is the written one. The tool will not run
  `sshd -T`; if you can execute, establish that evidence by hand and treat the payload's excluded
  row as what it is: the tool's boundary, not a verdict. A binary that is present is not a
  configuration, and a stopped service is reported as its own measurement on the POSIX platforms —
  the platform question there is the systemd `is-active` query. On native Windows this slice
  declares the platform's own `sshd` service question and does not put it to the machine, so it
  claims no service state there (issue #79). The positive conclusion requires the binary present
  *and* the configuration in force agreeing with the written one, so neither half stands in for the
  other.

### `cloudflare-tunnel`

- **What the row means.** The node dials out to the Cloudflare edge and the edge carries the tunnel
  to a hostname; no inbound port on either side. This is the transport that motivated the project.
- **Requires.** `hostname` and `zone_membership`. This slice has no input for either and no
  probe observes them, so both rows are unsatisfied by construction and **the transport is never
  viable in this build**. That is the honest answer for a slice with no account input, not a
  measured rejection of the tunnel; read the edge half and the notes for what is actually
  established.
- **How to read the evidence.** The `reason` and `notes` carry the edge state: reachable or partly
  reachable satisfies the edge half (a partial row names the split), while `CF_EDGE_UNREACHABLE` is
  a measured block and `CF_EDGE_UNRESOLVED` establishes nothing. Where the edge is measured
  reachable, the notes say explicitly that the unmet hostname is an account prerequisite to supply,
  not a network block.
- **What to supply.** A hostname inside a zone the user's Cloudflare account owns, and the account
  membership to create the tunnel and its DNS record. Both rows' `detail` strings say this; the
  human-facing prerequisites are in the [README](../README.md). Quick Tunnels cannot carry SSH, and
  a borrowed DNS record is not enough — see "The traps".
- **The notes are the operational payload.** They carry the measured edge observations, the HTTP/2
  recommendation when the datagram measurement failed or was never confirmed (recommend-only, with
  its post-quantum trade-off stated — no fallback was applied and no configuration was written), the
  account note above, and the tool's pin statement for the `cloudflared` client. Read the pin text
  from the payload's notes.

### `tailscale`

- **What the row means.** Detect-and-report only. The adapter is never silently relied upon.
- **Requires.** One row: `third_party_permission` — whether the endpoint agent or security policy
  permits the adapter on this machine.
- **How to read the row.** The adapter reads no measurement and names no target: no probe in this
  build observes whether a tunnel adapter is installed or permitted, so the row is not viable, the
  reason says the measurement is missing, and **no rejection of the transport is claimed**. This is
  a coverage gap the detection slices will close, not a statement that Tailscale cannot work here.
- **What to supply.** The policy check the row's detail names. A permitted adapter is a prerequisite
  for a later slice; this slice cannot measure or provision it.

## The traps

Each of these fails silently and looks like something else. Each is carried here with its
consequence.

- **The block may be on the destination, not on SSH.** "My corporate network blocks SSH" is usually
  wrong: SSH to a public host on 22 and 443 passes while every port to *your* hub fails, which means
  the block is on the destination's address. The measurement that settles it is the pair of public
  SSH probes beside the hub probe, and the `ssh.destination` finding states which side the evidence
  supports. Consequence: do not change transports because a human says "SSH is blocked"; read the
  finding, and note that a transport reaching the hub through a different address is a different
  question from the protocol being blocked. ([PRD](../PRD.md) §1.1, §8.1.)
- **`systemd` services do not keep a WSL2 instance alive.** Only children of Microsoft's `/init` do.
  A perfectly healthy `sshd` or `cloudflared` unit does not count, which is why the failure looks
  like a network fault. Consequence: on WSL2, persistence needs an `/init`-child keepalive that is
  dumb by design; the environment's lifetime is a separate problem from any service's health.
  ([PRD](../PRD.md) §6.2.)
- **`vmIdleTimeout` is a second, independent shutdown mechanism.** It defaults to 60 seconds and
  shuts the VM down even when an `/init` child exists. Consequence: fixing only the init-child
  problem does not keep the instance up, and relying on `vmIdleTimeout` alone does not either — the
  mechanisms are independent and each needs an answer. ([PRD](../PRD.md) §6.2.)
- **A named tunnel needs a zone the account owns; a borrowed DNS record is not enough.** A tunnel
  can only publish on a hostname inside a zone the account controls. Consequence: a friend's DNS
  record dead-ends; what works is account membership with tunnel and DNS permissions, NS delegation
  of the subdomain, or the owner creating the tunnel and handing over credentials. ([PRD](../PRD.md)
  §1, §11; the `zone_membership` requirement row.)
- **A correct key in the wrong file produces a correct fingerprint and a silent failure.** The
  content is right, the location is wrong (`sudo su` moved `~` to `/root`), and the verification
  passes anyway. Consequence: verify content **and** location — owner, mode and absolute path — and
  never accept "the fingerprint matches" as sufficient evidence on its own. ([PRD](../PRD.md) §1,
  §8.4.)
- **A listener's port must be checked before it is written.** The port a configuration will listen
  on may already be held by another daemon; on the motivating machine, 2222 was already taken by
  sshd. Consequence: check what holds the port before writing a listener for it, or the write
  succeeds and the listener never binds.
- **UDP/QUIC silence is not a block.** A UDP socket that produced neither a reply nor an error is
  ambiguous by construction; the run reports it `unresolved`, and a reply claims only that the
  datagram was not silently dropped. A failed datagram measurement beside a measured TCP path
  produces an HTTP/2 recommendation that the tool states as a recommendation — this slice never
  applies it. The trade-off is real and carried with the advice: forcing HTTP/2 forfeits
  post-quantum key agreement on the tunnel transport, not on the SSH session inside it.
  ([PRD](../PRD.md) §8.2; [README](../README.md).)
- **A `cloudflared` release is implicated by an open upstream report.** The report says the client
  ignores service tokens on the `access ssh` and `access tcp` paths, falling into a browser flow
  that can never complete headlessly. The pin statement — what to pin, which checksums to verify,
  and what the report does and does not establish — is carried verbatim in the `cloudflare-tunnel`
  notes; read it there rather than relying on memory. ([PRD](../PRD.md) §8.3.)
- **Docker Desktop can take the WSL2 VM down with it.** The remedy is a self-healing watchdog,
  because the tool does not disable other software to protect itself. Consequence: persistence on
  WSL2 must survive an environment killed by something outside it, and verification must exercise
  that kill rather than trust "the service is active". ([PRD](../PRD.md) §1, §6.3.)

## What the tool will not do

`doctor` measures and recommends; it does not act. In one run it:

- **installs nothing** — no package manager, no `cloudflared`, no `sshd`;
- **writes nothing** — no file, service, configuration, receipt or backup: its only output is the
  two projections;
- **executes no third-party binary** — the command-backed questions, the sshd configuration in
  force and the POSIX service state, are reported as excluded coverage gaps rather than run, and
  the native-Windows service question is declared and not put to the machine (issue #79);
- **connects no machines** — the two sides cannot reach each other, which is the problem; it
  measures only from where it runs;
- **dials only what it declares** — `targets.declared` is the exact dialed set;
- **sends nothing anywhere** — no telemetry, no account, no backend.

That division is what makes it safe to run anywhere, including on a machine you are not yet allowed
to change: a measurement that cannot change state can be taken before any authorization to change
anything. The output is the evidence a plan would rest on; the plan, the provisioning and the
verification of a working link belong to slices that do not exist yet.

## The discipline you must not break

The payload is built so that these rules are checkable. Breaking them makes your conclusion stronger
than the measurement, which is the one failure this tool exists to prevent.

- **`unresolved` is never `pass`.** An attempted measurement that produced no answer is not a
  success, not an absence, and not a block. Name it before any conclusion, and do not let it
  silently become either outcome.
- **A `fail` must be earned.** `fail` is a measured negative — refused, reset, absent, verified
  broken — never the absence of an answer. `budget_expired` on a declared reachability question is
  the measurement of no answer within the budget; `udp_silence` is not a block; `not_measured` is
  not a failure.
- **Nothing is stated above the confidence of the measurement.** A `viable` row is measured from
  where the tool ran; it is not a configured transport, a working connection, or persistence.
  Reasons and rule ids quote the measurement; do not paraphrase them into a diagnosis the run did
  not make.
- **Branch on the closed vocabularies.** Reason codes and rule ids are contract sets, compared
  against the code's own declarations; `detail` and `conclusion` are prose and may change without
  being a contract change. If a code or rule is not in
  [Diagnosis report contract](diagnosis-report.md), it is not part of the vocabulary — do not invent
  one, and do not fill a gap the report marks as unestablished.
- **A question with no rule is open, not answered.** If `findings` does not answer a question and
  `open_questions` names it, the correct report is the needed states, not a guess.
