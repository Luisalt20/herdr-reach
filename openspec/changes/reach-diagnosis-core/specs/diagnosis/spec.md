# diagnosis Specification

## Purpose

This domain defines the measurement layer of `herdr-reach` for slice R1a: the ten probes of the
specification's probe table (PRD §5.1), the result and verdict vocabulary they produce, the ordered
reasoning that turns measurements into facts, and the honesty rules that govern an unresolved
measurement. It exists so that every later slice reasons over a measurement whose provenance is
explicit and whose failures are visible. Probes measure only: nothing in this domain mutates the
measured machine. Requirement names cite the specification's own requirement ids (PRD §9) and never
renumber them.

## Requirements

### Requirement: The probe set is the specification's ten probes (PRD §5.1)

The system MUST provide the ten probes of the specification's probe table: `local.env`,
`local.sshd`, `egress.hub.direct`, `egress.ssh.known`, `egress.ssh.443`, `egress.cf.7844`,
`egress.cf.443`, `egress.quic`, `tls.interception` and `tls.truststore`. Each probe MUST declare one
of the specification's probe kinds (`local`, `egress`, `tls` or `proto`) and MUST be independently
runnable. A probe MUST be a measurement only and MUST NOT create, modify, move or delete anything on
the measured machine.

#### Scenario: The registry contains exactly the ten declared probes

- GIVEN the probe registry
- WHEN it is enumerated
- THEN it contains exactly the ten probes named above
- AND each has a stable name and one of the four probe kinds
- AND the enumeration order is deterministic across runs

#### Scenario: A probe measures only its own declared target

- GIVEN scripted dialer, resolver and TLS seams that answer for one probe's declared target
- WHEN that probe runs alone
- THEN only that probe's target is dialed
- AND no other probe is started as a side effect

#### Scenario: A probe writes nothing and executes nothing

- GIVEN a probe run with the deny-all command seam and injected file readers
- WHEN the probe completes
- THEN no external command was executed
- AND no file under the run's temporary home or working directory was created, modified or removed

### Requirement: Results carry the specification's verdict vocabulary (PRD §5.1)

Every probe MUST return a result carrying the probe name, probe kind, measured target, verdict,
verbatim detail and elapsed time. The verdict MUST be exactly one of `pass`, `fail` or
`indeterminate`; no boolean, numeric or free-text substitute MAY carry the measurement's meaning.

#### Scenario: A successful measurement is a pass

- GIVEN a scripted measurement that completes successfully within its budget
- WHEN the probe returns
- THEN the result carries the probe name, kind, target, verbatim detail and elapsed time
- AND the verdict is exactly `pass`

#### Scenario: An observation that cannot be classified is indeterminate

- GIVEN a scripted observation for which no classification rule applies
- WHEN the result is produced
- THEN the verdict is exactly `indeterminate`
- AND the verdict is never `pass`

### Requirement: Verbatim detail is preserved alongside a stable machine reason code (R-HR-07)

Every result MUST carry the verbatim observed detail, quotable as-is, together with a stable
machine-readable reason code drawn from one documented closed set. The reason code MUST be stable
across Go versions and operating systems, and the verbatim detail MUST NOT be the only
machine-readable signal.

#### Scenario: Wording drift does not move the classification

- GIVEN two observations that differ only in the operating system's wording of the same failure
- WHEN both results are produced
- THEN their reason codes are identical
- AND their verbatim details differ
- AND both details remain available for display

### Requirement: Probes are independent and a failed probe does not stop the run (PRD §4.4)

A failing, unresolved or not-measured probe MUST NOT prevent any other probe from being attempted or
reported. The suite MUST collect all available evidence before any conclusion is drawn.

#### Scenario: One broken probe does not stop the suite

- GIVEN one probe scripted to return `indeterminate` and one scripted to never return
- WHEN the run completes
- THEN every other probe of the ten has a result
- AND the run proceeds to reasoning with the results it obtained

### Requirement: Port-versus-protocol disambiguation for SSH (R-HR-03)

The engine MUST evaluate the specification's disambiguation rule: when SSH to a well-known public
host passes and the hub-directed TCP measurement fails, the conclusion MUST be that outbound SSH is
allowed and that the hub's address is blocked. That conclusion MUST be a named finding with a stable
rule id, and the reason MUST name the blocked target. The engine MUST state what is true about the
network and MUST NOT decide transport viability itself.

#### Scenario: The measured evidence matrix of PRD §1.1 replays to the correct conclusion

- GIVEN scripted dialers and resolvers that reproduce PRD §1.1's six rows: `github.com:22` passes,
  `ssh.github.com:443` passes, the hub's `203.0.113.10` on ports 22, 2222 and 443 resolves to no
  response inside the probe budget, `region1.v2.argotunnel.com:7844` passes, the same host on 443
  passes, and `www.cloudflare.com:443` passes with issuer `Let's Encrypt/ISRG` and verification code 0
- WHEN the diagnosis engine reasons over the results
- THEN the verdict states that outbound SSH works and that the destination is blocked
- AND the reason names `203.0.113.10` with its port
- AND the stable rule id that produced the conclusion is present in the output

#### Scenario: No destination-block conclusion without the passing public signal

- GIVEN the public SSH probe is `fail` or not measured while the hub-directed probe fails
- WHEN the engine reasons
- THEN the conclusion is not "outbound SSH works"
- AND the weaker finding actually established is emitted with its rule id

#### Scenario: Both signals passing yields no block conclusion

- GIVEN both the public SSH probe and the hub-directed probe pass
- WHEN the engine reasons
- THEN no destination-blocked conclusion is produced

### Requirement: Timeout classification is decided per probe purpose (R-HR-03, R-HR-NF-02)

Timeout classification MUST be resolved once in one ordered rule table and applied by every probe,
so one observation is never classified two ways. When a probe's declared question is whether a
declared TCP port is reachable, absence of a response within that probe's own budget is the
measurement and MUST be reported as `fail` with a timeout reason code. When the observable cannot
distinguish blockage from silence — UDP or QUIC egress, or a platform capability the tool cannot use
— the result MUST be `indeterminate` rather than a claimed block. A probe that never returns within
its bounded budget is a separate case and is governed by the hanging-probe requirement below.

#### Scenario: A blackholed TCP port resolves as a definitive negative

- GIVEN a scripted dialer that never answers for a declared port and a probe budget shorter than the
  runner's per-probe bound
- WHEN the reachability probe runs
- THEN the result is `fail` with the timeout reason code
- AND the result is produced by the probe itself, not by a runner-level abandonment
- AND the detail names the target and the elapsed budget

#### Scenario: UDP or QUIC silence is never reported as a block

- GIVEN a scripted packet seam that returns no response and no error
- WHEN the QUIC egress probe runs
- THEN the result is `indeterminate`
- AND the result is never `fail` and never claims the path is blocked

#### Scenario: A probe that ignores its budget is classified at the runner layer

- GIVEN a scripted probe that ignores context cancellation and never returns
- WHEN the run proceeds
- THEN that probe's result is `indeterminate` with a timeout reason code
- AND the classification comes from the runner, not from the probe

### Requirement: A measurement that could not be attempted is reported as not measured (R-HR-02)

A probe, or one of a probe's observations, that could not be attempted for lack of a required input
or capability MUST report an explicit not-measured outcome. That outcome MUST NOT be `fail`, MUST
NOT be presented as a blocked target, MUST NOT be presented as success, and MUST be distinguishable
by a consumer from an attempted-but-unresolved measurement. It MUST carry a stable reason code
naming what was missing.

#### Scenario: No hub supplied is not a blocked hub

- GIVEN a run with no hub address supplied
- WHEN the hub-directed probe is evaluated
- THEN the result is the not-measured outcome with a reason code naming the missing input
- AND no verdict states or implies that the hub is blocked
- AND no transport reports viability established from hub reachability

#### Scenario: A refused hub dial is a measurement, not a skip

- GIVEN a hub address supplied and a scripted dialer that refuses the connection
- WHEN the hub-directed probe runs
- THEN the result is `fail` with a refusal reason code
- AND the result is distinguishable from the not-measured outcome of the previous scenario

#### Scenario: An attempted but unresolved observation is not "not measured"

- GIVEN a probe that was attempted and produced no classifiable observation
- WHEN the result is produced
- THEN it is reported as unresolved and not as not measured

### Requirement: An unresolved measurement never yields a confident conclusion (R-HR-NF-03)

An `indeterminate` measurement MUST NOT contribute to a confident conclusion, and MUST weaken only
the conclusions that depend on it. When a conclusion depends on an unresolved measurement, the
engine MUST emit an explicitly weaker finding that names the unresolved dependency, or abstain. It
MUST NOT fall through to a default, to the opposite conclusion, or to a silent omission.

#### Scenario: An unresolved public-host signal weakens the SSH finding

- GIVEN the public SSH probe is `indeterminate` and the hub-directed probe fails
- WHEN the engine reasons
- THEN it emits a weaker finding stating that the SSH protocol state is unknown
- AND it does not conclude either "SSH is blocked" or "outbound SSH works"
- AND the weaker finding names the unresolved probe

#### Scenario: An unresolved probe does not poison unrelated conclusions

- GIVEN an unresolved probe whose measurement no conclusion depends on
- WHEN the engine reasons
- THEN the independent conclusions are unchanged
- AND the unresolved probe remains visible as a gap in the run's coverage

### Requirement: A hanging probe degrades completeness, never responsiveness (R-HR-NF-02)

The runner MUST NOT trust a probe to return. A probe that has not returned within its bounded budget
MUST be recorded as `indeterminate` with a timeout reason code, its in-flight work MUST be
abandoned, and the run MUST continue, finish and be reported as incomplete.

#### Scenario: A hanging probe resolves as unresolved and the run still ends

- GIVEN a scripted probe that ignores context cancellation and never returns
- WHEN the runner executes the suite
- THEN the run completes inside its run budget
- AND that probe's result is `indeterminate` with the timeout reason code
- AND every other probe has a result
- AND the run's completeness is reported as incomplete

### Requirement: Every probe is bounded and the run fits the 60-second budget (R-HR-NF-09)

Every probe MUST be bounded by its own timeout and the run MUST be bounded by the specification's
60-second budget. The bounds MUST be settable at construction so an offline test can prove them with
millisecond values. Cancelling the run MUST stop in-flight measurements rather than leaving them
running.

#### Scenario: Bounded probes finish a full run inside the budget

- GIVEN a scripted dialer that is uniformly slower than every injected per-probe bound
- WHEN a full run executes with millisecond bounds
- THEN every probe has a bounded result
- AND the run completes inside the injected run budget

#### Scenario: Cancellation stops in-flight measurements

- GIVEN a scripted dialer that observes context cancellation
- WHEN the run's context is cancelled mid-flight
- THEN the run returns promptly
- AND each affected probe reports the cancelled outcome rather than a fabricated pass

### Requirement: The measured target set is declared, closed and overridable (R-HR-NF-10)

The measured target set MUST be declared in one place, MUST be closed, and MUST be overridable by
configuration without changing the measuring code. No target outside the declared set MAY be
measured. The tool MUST NOT transmit telemetry or any measured data to a remote service; the only
outbound traffic in this slice is the declared measurements themselves.

#### Scenario: The dialed set equals the declared set exactly

- GIVEN a full run over all ten probes with scripted seams
- WHEN the set of dialed `(host, port, protocol)` triples is compared with the declared target set
- THEN the two sets are equal
- AND no additional outbound attempt was made

#### Scenario: An override moves the declaration, not the code

- GIVEN an override that replaces or extends the declared target list
- WHEN the run executes
- THEN the dialed triples follow the override
- AND no target outside the overridden declaration is measured

### Requirement: TLS interception is reported with issuer and verification code (R-HR-04)

The `tls.interception` probe MUST report the certificate issuer and the verification code observed
for a known host. When a chain is not the expected publisher's chain, the result MUST name the
interception rather than report a successful measurement. The probe MUST NOT disable verification,
retry without verification, or otherwise weaken correctness to obtain a result.

#### Scenario: An inspecting middlebox is named in the result

- GIVEN an injected TLS chain whose issuer is not the expected publisher for the target host
- WHEN the probe runs
- THEN the result carries the observed issuer and the verification code
- AND the result is not `pass`
- AND the reason code names the interception

#### Scenario: A verification failure does not silently disable verification

- GIVEN an injected chain that fails verification
- WHEN the probe runs
- THEN the result reports the verification failure with its code
- AND the injected verifying configuration is unchanged after the run
- AND no unverified retry was performed

### Requirement: The macOS trust-store limitation is stated in the output (R-HR-04)

The `tls.truststore` probe MUST report whether the local trust store accepts the chain. On macOS the
probe MUST report the unresolved outcome and the output itself MUST state the documented limitation:
Go cannot enumerate macOS system roots, and keychain trust is only visible through the platform
verifier when no explicit root pool is supplied. The probe MUST report `pass` and `fail` on Linux,
and MUST report the unresolved outcome — never `pass` — when the platform verifier would be bypassed,
for example when a certificate-file or certificate-directory environment override is set.

#### Scenario: The Linux probe can pass and can fail

- GIVEN a scripted platform seam reporting Linux and a trust pool that accepts the chain
- WHEN the probe runs
- THEN the result is `pass`
- AND GIVEN a chain the trust pool does not accept
- THEN the result is `fail` with a stable reason code

#### Scenario: macOS reports the unresolved outcome with the limitation stated

- GIVEN a scripted platform seam reporting macOS
- WHEN the probe runs
- THEN the result is `indeterminate`
- AND the documented limitation is present in the output itself
- AND the result is never `pass`

#### Scenario: A bypassed platform verifier never yields a false pass

- GIVEN a platform seam with certificate-file or certificate-directory overrides set
- WHEN the probe runs
- THEN the result is `indeterminate`
- AND the result is never `pass`

### Requirement: local.sshd reports written-versus-effective divergence and degrades honestly (R-HR-18)

The `local.sshd` probe MUST report the binary's presence, the service's state and the effective
configuration as separate observations, each with its own outcome. When the effective configuration
differs from the written configuration, both MUST be reported and the result MUST NOT be presented
as success. When a required capability is unavailable, the affected observation MUST report the
not-measured outcome, or `indeterminate` when an attempt was made and could not be resolved, always
naming the missing capability in a stable reason code; it MUST NOT report `pass`. All local input
MUST be read through the injected readers, so nothing outside those seams is read.

#### Scenario: Written and effective configuration diverge

- GIVEN injected readers whose effective configuration differs from the written configuration
- WHEN the probe runs
- THEN the result reports both configurations in the verbatim detail
- AND the result carries a stable reason code naming the divergence
- AND the probe does not report success

#### Scenario: A denied command seam degrades instead of passing

- GIVEN the deny-all command seam
- WHEN the probe runs
- THEN the command-derived observation reports the not-measured or unresolved outcome
- AND the reason code names the missing capability
- AND the result is never `pass`

#### Scenario: An absent sshd is reported, not installed

- GIVEN injected readers reporting no sshd binary
- WHEN the probe runs
- THEN the absence is reported as a measurement result with a stable reason code
- AND the output states that installing it is not part of this slice

### Requirement: The machine is classified by platform without provisioning (R-HR-29)

The `local.env` probe MUST classify the machine as Linux, macOS, WSL2 or native Windows and MUST
report the architecture. This slice classifies only: it MUST NOT provision, persist, reconfigure or
remediate anything on any platform, and an unknown platform MUST degrade to an explicit unresolved
outcome naming the missing signal rather than to a default guess.

#### Scenario: Each supported classification is reported

- GIVEN scripted platform seams for Linux, macOS, WSL2 and native Windows
- WHEN the probe runs
- THEN each machine reports the matching classification with its architecture
- AND no provisioning or configuration action is offered or performed

#### Scenario: An unknown platform is not guessed

- GIVEN a platform seam whose signals match no supported classification
- WHEN the probe runs
- THEN the result is `indeterminate`
- AND the reason code names the missing classification signal
- AND no platform is assumed

### Requirement: A native-Windows node is refused with WSL2 named as the supported path (R-HR-30)

The tool MUST refuse to operate on a node classified as native Windows and MUST explain why, naming
WSL2 as the supported path. The refusal MUST be reported as the measurement's negative answer, not
as a usage or internal error, and no transport MAY be presented as viable for that node.

#### Scenario: Native Windows is refused with the supported path named

- GIVEN a scripted classification of native Windows
- WHEN the run completes
- THEN the report contains the refusal and names WSL2 as the supported path
- AND no transport is reported viable for that node
- AND the report contains no planned or applied change

### Requirement: WSL2 output is limited to documented semantics (PRD §6.2, RG-4)

Any WSL2 output in this slice MUST be limited to detection and to the documented `vmIdleTimeout`
semantics: milliseconds of idle before the VM is shut down, default 60000, available only on
Windows 11. The output MUST NOT state the child-of-init shutdown rule, MUST NOT state a `-1`
sentinel, and MUST NOT describe keepalive, watchdog or persistence behaviour.

#### Scenario: The WSL2 note carries only documented semantics

- GIVEN a scripted classification of WSL2
- WHEN the report is produced
- THEN the WSL2 text contains the documented millisecond semantics
- AND it does not state the child-of-init shutdown rule and does not state a `-1` sentinel
- AND it does not claim any keepalive or persistence behaviour exists

#### Scenario: WSL2 without systemd is detected and reported only

- GIVEN a scripted WSL2 classification whose signals show systemd is not enabled
- WHEN the probe runs
- THEN the report states the detection
- AND enabling systemd is stated as work owned by a later slice, not performed here
