# cli-report Specification

## Purpose

This domain defines the headless `doctor` surface of `herdr-reach` for slice R1a: its input, its two
projections (human text and machine-readable payload), its documented exit codes, the determinism its
output must hold, and the proof that a run changes nothing and executes no third-party binary. The
interactive TUI surface is owned by a later change and is not specified here. Requirement names cite
the specification's own requirement ids (PRD §9).

## Requirements

### Requirement: The headless doctor surface separates machine output from human output (R-HR-07)

The tool MUST expose a headless `doctor` command. With the machine-readable flag, standard output
MUST carry exactly one valid machine-readable document containing the full diagnosis, with no human
text, progress line or banner mixed into it. All human-facing output MUST go to standard error, so
that piping the machine-readable document into another program is always safe. The command MUST
perform measurements only.

#### Scenario: The machine-readable document is the only thing on standard output

- GIVEN a run with the machine-readable flag and scripted seams
- WHEN the command completes
- THEN standard output parses as exactly one machine-readable document
- AND no prose, progress text or warning appears on standard output
- AND the human-facing text appears on standard error

#### Scenario: The human projection never pollutes standard output

- GIVEN a run without the machine-readable flag
- WHEN the command completes
- THEN the human report is written to standard error
- AND standard output carries no human text

### Requirement: The hub address is an optional input and its absence is honest (RG-13)

The command MUST accept an optional hub address in the form `host[:port]`. When supplied, the
hub-directed probe's target MUST be exactly the supplied address, with one documented default port
applied when the port is omitted, and the resolved target MUST be stated in the output. When no hub
address is supplied, the hub-directed probe MUST be reported per the diagnosis domain's
not-measured requirement: the output MUST state that the hub measurement was not made, MUST NOT
state or imply that the hub is blocked, and MUST NOT report transport viability established from hub
reachability. Unusable input MUST be reported as a usage error.

#### Scenario: The supplied hub address becomes the measured target

- GIVEN a run with an explicit hub address including a port
- WHEN the command completes
- THEN the hub-directed probe's target equals the supplied address exactly
- AND the resolved target is present in the output

#### Scenario: An omitted port resolves by one documented rule and is stated

- GIVEN a run with a hub address that omits the port
- WHEN the command completes
- THEN the documented default port is applied
- AND the resolved target including that port is stated in the output

#### Scenario: No hub address yields not measured, not blocked

- GIVEN a run with no hub address
- WHEN the command completes
- THEN the output states that the hub measurement was not made
- AND no output states or implies that the hub is blocked
- AND no transport is reported viable on the strength of hub reachability

#### Scenario: Unusable input is a usage error

- GIVEN a hub address with an empty host or an unparsable port
- WHEN the command runs
- THEN it fails with a usage error on standard error
- AND no measurement run is presented as completed

### Requirement: Exit codes are documented and testable (R-HR-07)

The command MUST return `0` for a completed measurement, including a negative answer such as no
viable transport or a refused unsupported node platform; `1` when the run is incomplete because at
least one probe was attempted and resolved unresolved; and `2` for a usage or internal error. A
not-measured probe MUST NOT by itself make the run incomplete, but the run's output MUST expose it so
a caller can see the coverage gap. The exit-code contract MUST be documented where a script author
reads it and MUST be covered by tests.

#### Scenario: A negative answer is a completed measurement

- GIVEN a run in which every probe resolved and no transport is viable
- WHEN the command completes
- THEN the machine-readable document reports no viable transport
- AND the exit code is `0`

#### Scenario: An unresolved probe makes the run incomplete

- GIVEN a run with at least one probe that was attempted and resolved unresolved
- WHEN the command completes
- THEN the run is reported as incomplete
- AND the exit code is `1`

#### Scenario: Usage and internal errors are distinct

- GIVEN an unknown flag or an unusable flag value
- WHEN the command runs
- THEN the exit code is `2`
- AND no diagnosis is presented as completed

#### Scenario: A refused node platform is a negative answer, not an error

- GIVEN a node classified as native Windows
- WHEN the command completes
- THEN the report contains the refusal naming WSL2 as the supported path
- AND the exit code is `0`
- AND the exit code is not `2`

#### Scenario: A not-measured probe stays visible without changing the result

- GIVEN a run in which a probe could not be attempted for lack of a required input
- WHEN the command completes
- THEN that probe is reported as not measured
- AND the exit code is not changed by that probe alone

### Requirement: The JSON payload carries a schema version and no schema is frozen in this slice (R-HR-NF-06)

The machine-readable document MUST carry a schema version field. This slice MUST NOT freeze a schema
file, MUST NOT write one, and MUST NOT promise cross-machine compatibility; promotion to a versioned
wire contract later MUST be additive. The document MUST be produced by one mapping from the internal
diagnosis, so an internal structure change does not silently change the payload.

#### Scenario: The payload is versioned

- GIVEN a completed run
- WHEN the machine-readable document is produced
- THEN it carries a non-empty schema version field

#### Scenario: One mapping owns the payload shape

- GIVEN an internal diagnosis structure changed without changing the mapping
- WHEN the document is produced
- THEN the payload is unchanged
- AND no schema file is required to read it

### Requirement: Identical inputs produce byte-identical output (R-HR-07)

For identical inputs, scripted seams and injected clock, the machine-readable document MUST be
byte-identical across runs. Probes MUST be ordered by the registry, transports MUST be sorted, no map
iteration may influence the output, the generated-at value MUST come from one injected clock, and
elapsed values MUST be rendered in one documented unit.

#### Scenario: Repeated runs are byte-identical

- GIVEN the same scripted seams and the same injected clock
- WHEN the command runs more than once
- THEN each run's machine-readable document is byte-identical to the others

#### Scenario: Only the injected clock moves the timestamps

- GIVEN two runs whose only difference is the injected clock value
- WHEN the documents are compared
- THEN only the generated-at field and elapsed values differ
- AND probe and transport ordering is unchanged

### Requirement: Unresolved is never encoded or rendered as success (R-HR-NF-03)

The machine-readable document MUST carry each probe's verdict verbatim and MUST NOT carry any derived
boolean, status or summary that presents an unresolved measurement as success. The human projection
MUST render an unresolved measurement as unresolved, with its reason, and MUST NOT attribute success
wording to it.

#### Scenario: The payload never turns unresolved into a pass

- GIVEN a run in which a probe resolved unresolved
- WHEN the machine-readable document is produced
- THEN that probe's verdict is exactly `indeterminate`
- AND no boolean or summary field marks it as successful
- AND no field maps it to `pass`

#### Scenario: The human projection says unresolved

- GIVEN the same run
- WHEN the human projection is produced
- THEN the unresolved probe is marked as unresolved with its reason
- AND no success wording is attributed to it

### Requirement: An incomplete run is reported as incomplete (R-HR-NF-02)

The report MUST state run completeness separately from the individual probe rows, MUST name the
unresolved probes, and MUST NOT present a run as complete when any probe was attempted and resolved
unresolved. The verdict text MUST NOT imply that the suite finished when it did not.

#### Scenario: A hanging probe makes the run incomplete in both projections

- GIVEN a run containing a probe that never returns
- WHEN the command completes
- THEN both projections state that the run is incomplete
- AND the unresolved probe is named
- AND the verdict does not imply that the suite finished
- AND the exit code is `1`

#### Scenario: A not-measured probe is distinguished from an unresolved one

- GIVEN a run containing a probe that could not be attempted and a probe that was attempted and
  resolved unresolved
- WHEN the report is produced
- THEN the two are reported as distinct outcomes
- AND the completeness statement reflects the unresolved probe rather than the not-measured one

### Requirement: Raw evidence is preserved in both projections (R-HR-07)

Both projections MUST carry, for every probe, the measured target, the verdict, the verbatim detail
and the stable reason code, so that a user can read the raw result and disagree with the tool. No
summary, headline or recommendation MAY drop the named failing target.

#### Scenario: The blocked target survives into both projections

- GIVEN the PRD §1.1 evidence replay
- WHEN both projections are produced
- THEN the hub target string is present in the human projection and in the machine-readable document
- AND the verbatim detail of the failing measurement is preserved
- AND the stable reason code accompanies it

### Requirement: A run writes nothing and executes no third-party binary (R-HR-02, R-HR-24, R-HR-27)

A full run MUST leave the machine unchanged: with the home directory and the working directory
pointed at a fresh temporary tree, the tree MUST be byte-identical after the run. The production
shape wires no command runner, so a production run calls none and has no execution path at all;
with a test's denying runner wired, every attempt is denied and nothing is executed. The tool
MUST NOT invoke `herdr`, `cloudflared`, `sshd`, `security` or any other third-party binary. The set
of dialed `(host, port, protocol)` triples MUST equal the declared target set exactly. The proof's
coverage is limited to the home directory and the working directory; it MUST NOT be presented as
proof that no path outside them was touched, a gap this slice closes by construction because no
writer code path and no write-capable dependency exists.

#### Scenario: The temporary tree is byte-identical after a run

- GIVEN a fresh temporary home and working directory with recorded recursive digests
- WHEN a full run completes
- THEN the recursive digests are unchanged
- AND no file was created, modified or removed

#### Scenario: No third-party binary is executed

- GIVEN the deny-all command runner
- WHEN a full run completes
- THEN every command attempt was denied
- AND no third-party binary was executed
- AND the production shape wires no command runner, so no execution path exists even if a probe asks

#### Scenario: Only the declared targets are dialed

- GIVEN a full run with scripted seams
- WHEN the dialed triples are collected
- THEN they equal the declared target set exactly
- AND no other outbound attempt was made

#### Scenario: A write attempt cannot pass the guard

- GIVEN a construction that attempts any write during a run
- WHEN the guard test executes
- THEN the run fails the test rather than reporting success
