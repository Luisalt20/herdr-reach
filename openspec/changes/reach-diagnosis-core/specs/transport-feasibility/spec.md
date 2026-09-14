# transport-feasibility Specification

## Purpose

This domain defines how `herdr-reach` decides, for slice R1a, whether each of the specification's
four V1 transports can work on the measured network, and how it explains the answer in both
directions. It consumes the diagnosis domain's findings and produces detect-and-recommend
feasibility only: no plan, no provisioning, no verification against a real transport. The domain
exists so that a user learns why the cheap path is closed and not merely that it is closed, and so
that adding a transport later is an interface implementation rather than a change to the reasoning
layer. Requirement names cite the specification's own requirement ids (PRD §9).

## Requirements

### Requirement: The adapter registry is the specification's four V1 transports (R-HR-06)

The registry MUST contain exactly the four V1 transports named by the specification: `direct-ssh`,
`reverse-ssh`, `cloudflare-tunnel` and `tailscale`. Every registered transport MUST be reported in a
deterministic order, and every registered transport MUST be considered on every run. No adapter
outside this set, including any adapter researched for a later slice, MAY be registered in this
slice.

#### Scenario: The registry is exactly the four declared transports

- GIVEN the transport registry
- WHEN it is enumerated
- THEN it contains exactly `direct-ssh`, `reverse-ssh`, `cloudflare-tunnel` and `tailscale`
- AND the enumeration order is stable across runs
- AND no other transport is registered

### Requirement: Every considered transport reports viability with a reason in both cases (R-HR-06)

Every considered transport MUST report whether it is viable and MUST carry a reason in both cases: a
non-viable transport MUST carry a non-empty reason, and a viable transport MUST carry the measured
observations that make it viable. A transport may never be reported without an explanation, and no
transport may be reported viable on the strength of an assumption instead of a measurement.

#### Scenario: No transport is reported without a reason

- GIVEN a finding set over which all four transports are evaluated
- WHEN the feasibility list is produced
- THEN every transport carries a non-empty reason
- AND every non-viable transport carries a non-empty reason
- AND every viable transport carries at least one named measured observation

#### Scenario: A detected but blocked adapter is reported blocked, never viable

- GIVEN a scripted detection that an adapter is installed but its connectivity is blocked
- WHEN its feasibility is produced
- THEN the transport is not viable
- AND the reason names the blocked observation
- AND the transport is never reported viable or assumed to work

### Requirement: A rejection reason names the measured observation that rejects the transport (R-HR-06)

Every rejection MUST name the specific measured target or observation that rejects the transport.
An adapter MUST NOT state a network fact it did not receive from the reasoning layer, and MUST NOT
invent a reason for a measurement that was never made.

#### Scenario: The matrix replay names the blocked hub in each rejection

- GIVEN the PRD §1.1 evidence replay in which the public SSH signal passes and the hub address is
  blocked
- WHEN feasibility is produced for the four transports
- THEN the direct and reverse SSH transports are not viable
- AND each reason names the blocked hub target and its port
- AND no reason claims a measurement that was not performed

#### Scenario: An unmeasured transport is honest about the gap

- GIVEN a transport whose viability depends on an observation that was not measured
- WHEN its feasibility is produced
- THEN it is not reported viable
- AND its reason states that the relevant measurement is missing
- AND the reason does not assert that the transport is blocked

### Requirement: A conditional prerequisite is reported as a requirement, not as a third viability state (PRD §5.2, §14.5)

Viability MUST be exactly true or false; there MUST be no third "sort of viable" state and no
enumeration, boolean or string in the payload that encodes one. When a transport's viability depends
on something only the user can supply — for example a hostname in a Cloudflare account — that
dependency MUST be reported as an explicit, machine-readable requirement with a note. A transport
with an unmet prerequisite MUST NOT be presented as ready to use.

#### Scenario: A hostname prerequisite is reported as a requirement

- GIVEN a network on which the Cloudflare edge is reachable but no hostname is available
- WHEN the Cloudflare transport's feasibility is produced
- THEN its requirement list is non-empty and names the missing hostname
- AND the note states that the prerequisite is not satisfied
- AND no output describes the transport as ready to use

#### Scenario: No third viability value exists anywhere in the output

- GIVEN the full feasibility payload
- WHEN it is inspected
- THEN every transport's viability is exactly true or false
- AND no additional viability state, severity or partial-credit value is encoded

### Requirement: Feasibility is detect-and-recommend only in this slice (R-HR-05, R-HR-02)

In this slice, transport feasibility MUST NOT plan, apply, install, verify, start, stop or otherwise
modify anything on the machine or on any remote system. Producing feasibility MUST NOT perform a
measurement of its own: it reads the diagnosis it was given and returns an answer and reasons.

#### Scenario: Producing feasibility touches nothing

- GIVEN the deny-all dialer and command seams and a complete finding set
- WHEN feasibility is produced for all four transports
- THEN no dial, command execution, file access outside injected readers, or configuration change
  occurred
- AND the produced result contains only viability, reasons and requirements

### Requirement: No caller can receive a fabricated plan or verification (RG-9)

Any member of the transport contract that produces a plan or verification in a later slice MUST, in
this slice, fail with an explicit typed not-implemented error that names the member and the slice
that owns it. It MUST NOT return an empty, nil or plausible-looking plan or evidence set, and MUST
NOT return a value that a caller could mistake for a successful result.

#### Scenario: A not-implemented member fails loudly

- GIVEN a registered transport and the deny-all seams
- WHEN a plan-producing or verification-producing member is invoked
- THEN the call fails with an explicit not-implemented error naming the member
- AND the error names the slice that will implement it
- AND no plan or evidence value is returned to the caller

### Requirement: Adding a transport requires no change to the reasoning layer (R-HR-NF-04)

Adding a transport MUST be achievable by implementing the transport contract, with no change to the
diagnosis or reasoning layer. The reasoning layer MUST state network facts and MUST NOT contain
per-transport knowledge; each transport MUST derive its own reasons from the facts it is given.

#### Scenario: A test-only transport appears without touching the reasoning layer

- GIVEN a test-only transport registered in a test
- WHEN feasibility is produced
- THEN the resulting verdict includes that transport with its stated viability and reason
- AND the factual findings produced by the reasoning layer are unchanged
- AND no file under the reasoning layer was modified for the test to pass

### Requirement: The HTTP/2 recommendation states the post-quantum trade-off and claims no enforcement (R-HR-05)

When the QUIC egress measurement does not succeed and the TCP edge is reachable, the recommendation
MUST state the documented trade-off that the HTTP/2 path does not support post-quantum key agreement
on the tunnel transport, and MUST state that this slice only recommends. It MUST NOT claim that the
fallback was applied, that a configuration was changed, or that the HTTP/2 path is equivalent to
QUIC. When QUIC succeeds, no HTTP/2 downgrade MAY be recommended.

#### Scenario: The QUIC-failed recommendation states the trade-off and recommends only

- GIVEN an unresolved or failed QUIC measurement and a reachable Cloudflare edge on 443
- WHEN the Cloudflare transport's notes are produced
- THEN the notes state that HTTP/2 forfeits post-quantum key agreement
- AND the notes state that this slice recommends and does not enforce
- AND no output claims a fallback was applied or a configuration written

#### Scenario: Working QUIC produces no downgrade recommendation

- GIVEN a passing QUIC measurement
- WHEN the Cloudflare transport's notes are produced
- THEN no HTTP/2 downgrade is recommended

### Requirement: The cloudflared pin is stated as report-only knowledge with a single home (R-HR-16)

Wherever this slice mentions the pinned client, it MUST state only what the research record
establishes: upstream issue #1673 is open, uncommented and names version 2026.6.0 only; the behaviour
of later releases is unknown; and release 2026.5.1 publishes SHA256 checksums. The statement MUST NOT
generalize to a version range, MUST NOT claim a fix, and MUST NOT claim that this slice installed,
verified or pinned anything on the machine. The statement MUST have exactly one home in the codebase,
so the human and machine-readable projections cannot drift apart.

#### Scenario: The pin note stays inside the established evidence

- GIVEN the Cloudflare transport's pin note
- WHEN it is produced in the report
- THEN it names issue #1673 with its state and names version 2026.6.0 only
- AND it states that later releases' behaviour is unknown
- AND it cites release 2026.5.1's published SHA256 checksums
- AND no version-range claim and no claim of installation or verification appears

#### Scenario: One source feeds both projections

- GIVEN the human projection and the machine-readable projection of the same run
- WHEN the pin note is compared between them
- THEN both carry the same text from the single declared source
- AND no second copy of the wording exists in the repository
