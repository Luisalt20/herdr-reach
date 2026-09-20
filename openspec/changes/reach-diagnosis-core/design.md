# Design — `reach-diagnosis-core` (R1a: headless diagnosis core)

**Change**: `reach-diagnosis-core` — slice R1a of the approved roadmap R1–R12
**Phase**: design · **Date**: 2026-09-14
**Artifact store**: both — this file and Engram topic `sdd/reach-diagnosis-core/design` (type `architecture`, project `herdr-reach`)
**Inputs read before designing**: proposal (`sdd/reach-diagnosis-core/proposal`, obs. 797), the three specs (`sdd/reach-diagnosis-core/spec`, obs. 798), exploration (obs. 791), research (obs. 793, `outcome: partial`), pre-proposal handoff (obs. 792), PRD §5.1/§5.2/§7.1/§8/§9/§11/§13/§14.5, `openspec/config.yaml`.
**Skill resolution**: `paths-injected` — `/home/luisalt20/.config/opencode/skills/cognitive-doc-design/SKILL.md` read before writing.
**Preflight consumed**: execution mode `auto`, artifact store `hybrid`, delivery strategy `ask-on-risk`, review budget 400 changed lines, chain strategy deferred, `size:exception` never inferred. This phase inferred no new SDD choice.
**Scope override consumed**: this repository is the Go project `github.com/Luisalt20/herdr-reach`. The packaged "centre on `packages/coding-agent`" rule does not apply; the layout below is PRD §7.1's own sketch plus the deviations recorded in §2.
**No implementation**: this artifact adds no Go file, no `go.mod`, no scaffolding. The only file written is this one (plus its memory mirror).

---

## Quick path for a reviewer

1. **§1** — the ten decisions (D1–D10) the proposal left open, each with its rejected alternative.
2. **§2** — the deviation record, checkable line by line against PRD §7.1 and §5.1/§5.2.
3. **§3** — the contracts: result shape, payload, exit codes, reason codes, rule ids.
4. **§5** — the classification table and the rule table (the product's centrepiece) plus the five design obligations the spec phase raised.
5. **§6** — the seam set, the deny-all test default, and what proves no real egress.
6. **§7 + §8** — file plan and test plan, read together.
7. **§9 + §10** — rollout/rollback and the review-budget position.

Intentionally **not** reviewed here: any TUI (R1b), any write or `Step` (R3), `PairingBundle`/`NodeReceipt` bodies (R2), and behaviour of real `cloudflared`/`sshd` (R5).

---

## 1. Decisions D1–D10

Each decision states what is chosen, why, and the alternative that was rejected with its cost.

### D1 — Package boundaries, dependency direction and `Result` ownership

**Decision.** Six new packages plus the entrypoint, in dependency order:

```
cmd/herdr-reach      →  doctor, probe, diagnosis, transport, report, version
internal/doctor      →  probe, diagnosis, transport, report, version   (run harness, exit codes)
internal/transport   →  diagnosis, probe (types)
internal/report      →  diagnosis, transport, probe (types), version
internal/diagnosis   →  probe (value types only)                        ← the one deviation
internal/probe       →  stdlib + injected seams only
internal/version     →  stdlib
```

`internal/probe` owns the shared measurement types (`Probe`, `ProbeKind`, `Verdict`, `Resolution`, `Result`, `Observation`, `ReasonCode`) because PRD §5.1 already places `Probe`/`Result`/`Verdict` in `probe.go`, and because a second owner would make `Result` a two-step migration later. `internal/probe` never imports `diagnosis`, `transport` or `report`; `diagnosis` imports `probe` for value types and **never calls a measurement function**; `transport` states reasons, `diagnosis` states facts.

**Why.** The port-vs-protocol rule is the most valuable logic in the tool. Putting it in `probe/egress.go` (PRD §7.1's own sketch) puts the reasoning in the same package as `net`, which makes "the reasoning layer is unchanged when a transport is added" (NF-04) untestable and invites rule duplication per adapter.

**Trade-off.** `internal/diagnosis` can never be compiled without `probe`, so the reasoning package transitively links `net`/`crypto/tls`. Accepted because the deliverable is one binary and the property that matters ("the rule table needs no network to be tested") is enforced by the seam default and by two guard tests (`§6.2`, `§6.4`), not by package isolation. Cost of the deviation: one extra package and one paragraph of documentation.

**Rejected alternative.** A shared `internal/measure`/`internal/model` package holding `Result`/`Diagnosis` so neither `probe` nor `diagnosis` owns them. Rejected because it adds a package *and* a migration of PRD §5.1's already-named type location, for an isolation benefit that the guard tests already deliver. Also rejected: PRD-literal placement of the rule inside `internal/probe/egress.go`, because NF-04 then has no boundary to hold fixed.

### D2 — Classification rules, the rule table, and where each lives

**Decision.** Two separate, purely declarative tables:

| Table | Home | What it answers |
|---|---|---|
| **Classification table** | `internal/probe/classify.go` | For one raw observation (dial error, absence, silence, platform signal): `(Resolution, ReasonCode)`. Applied by every probe so one observable is never classified two ways. Enumerated in §5.1. |
| **Rule table** | `internal/diagnosis/rules.go` | For a question (e.g. `ssh.destination`): the first matching rule's conclusion, or an explicit open question. Declarative data, no I/O, no `net`. Enumerated in §5.2. |

The split is deliberate: only the probe knows whether silence means "port unreachable" *for its own declared question*, so measurement classification is probe-level; whether the network fact implies a destination block is engine-level.

**Why one table per concept.** The spec requires timeout classification to be "resolved once in one ordered rule table"; the classification table is that table. It is a pure function of an observation plus the probe's declared purpose, so the whole unit is testable with hand-written observations and zero network.

**Trade-off.** Two places to look when a new reason code is wanted. Mitigated by the rule that a new reason code arrives only with a classification-table row and a documented entry in `docs/diagnosis-report.md` (§3.4).

**Rejected alternative.** A single table in `diagnosis` classifying raw errors: rejected because `diagnosis` would then need `net` error types and would own measurement semantics (and `net`-typed switches are exactly the Go-version-sensitive code the spec wants out of the reasoning layer). Also rejected: per-probe inline classification, because two probes could classify the same `i/o timeout` differently.

### D3 — Hub address representation, flag syntax, and how absence appears

**Decision.** `--hub host[:port]`, applying the documented default port `22` when the port is omitted. Parse rule: `net.SplitHostPort` first; on failure the value is treated as a host only, except that a bracketed IPv6 literal `[::1]` is accepted as host-only. Empty host or unparsable port ⇒ usage error, exit `2`, nothing measured. The resolved target is echoed in both projections.

Absence is represented as a **`not_measured` observation with `reason: "input_missing_hub"`** and JSON `"target": null` (a pointer field, so the payload distinguishes "no target" from a target string; an empty host can never reach a probe because it is a usage error).

**Why.** The product decision is already recorded: absent hub ⇒ not measured, never blocked. `null` is the only representation a script cannot confuse with a hostname, and it keeps the not-measured path distinguishable from `203.0.113.10:22` failing.

**Trade-off.** One nullable field in the payload and one branch in the mapper. Rejected alternatives: a `"(not supplied)"` sentinel string (indistinguishable from a hostname by a script) and omitting the field (a script then cannot tell "not measured" from "field renamed").

Also decided here: **no config file and no env var in R1a.** The file format that carries the hub address is R2's `PairingBundle`; inventing one now would be replaced immediately.

### D4 — `go` directive, toolchain pin, TUI dependency versions

**Decision.**

| Item | Decision |
|---|---|
| `go.mod` module | `github.com/Luisalt20/herdr-reach` (ratified) |
| `go` directive | `go 1.25.10` |
| `toolchain` line | **Absent** |
| Non-stdlib dependencies in R1a | **None** — so `go.sum` is expected to be *absent*, not missing |
| TUI dependency versions (Bubbletea/Bubbles/Lipgloss) | **Not pinned in R1a**; the constraint is recorded and the choice is owned by R1b |

`go 1.25.10` matches the floor the README already declares publicly, so the audience the README promises is exactly the audience that can build the module. No `toolchain` line means a contributor on 1.25.10 is not silently forced to download the author's 1.26.6. R1a code must therefore avoid 1.26-only language features; the local 1.26.6 toolchain builds it fine under 1.25 semantics, and that is the configuration the acceptance evidence runs in.

**Trade-off.** The author's observed toolchain (`go1.26.6`) is newer than the declared directive, so contributors may use an older toolchain than the author runs; that is the intended widening, and the cost is that a 1.26-only API would fail for those contributors — caught by the declared floor rather than by accident. On TUI versions: pinning three unverified version strings now would assert something the research record does not establish (§14.5), and `go mod tidy` would delete unused requirements anyway. R1b must pin all three to the same, verified versions (PRD §14.1) at the moment it adds them, in `go.mod` as the single home.

**Rejected alternatives.** `go 1.26.6` (matches the machine, silently narrows the audience and contradicts the README badge, and the README edit is explicitly out of this change). `go 1.25.10` **plus** `toolchain go1.26.6` (forces a download on contributors and turns a wider audience into a narrower one at the first build). Pinning TUI versions now by name (unverifiable in this phase; a fabricated pin).

### D5 — Exit codes, stream split, payload shape, reason-code vocabulary

**Decision.** Full contract in §3. In brief: exit `0` = measurement completed (including a negative answer and including a native-Windows refusal), `1` = run incomplete, `2` = usage or internal error. `--json` writes exactly one JSON document to stdout; all human text goes to stderr; without `--json`, stdout stays empty. `schema_version` is `"1"`. Reason codes are a closed set of 27 constants in `internal/probe/reason.go`, documented with the payload and the exit codes in `docs/diagnosis-report.md`, and checked by a test.

**Why.** The product decision (a negative answer is success) is already recorded, and the spec requires the documented location to be where a script author reads it — which cannot be the README, because this change must not modify it.

**Trade-off.** A new repository-facing document (`docs/diagnosis-report.md`) and a test that parses it; in exchange, the doc cannot drift from the constants, and adding a code is mechanically a contract change. Rejected alternative: documenting the codes in code comments only (a script author does not read Go) and documenting them in the README (out of scope for this change).

**Reconciliation recorded.** The proposal says exit `1` denotes "any `indeterminate`". The spec is tighter: only an **attempted** probe that resolved unresolved makes the run incomplete, and a not-measured probe must not. Because not-measured observations also carry verdict `indeterminate` (PRD §5.1 allows only three verdicts), completeness is computed over **observations** by resolution, not over verdicts. This design follows the spec; the proposal's looser phrasing is a simplification, not a conflict.

### D6 — Conditional requirements without a third viability state

**Decision.** Viability stays a strict boolean. A prerequisite is a first-class, machine-readable row:

```
Requirement{ Kind, Satisfied bool, Detail }      // Kind from a closed set
Feasibility{Viable bool, Reason string, Notes []string, Requires []Requirement}
```

Closed `Kind` set for R1a: `hostname`, `zone_membership`, `third_party_permission`, `hub_address`, `sshd_effective_config`. When any requirement row is unsatisfied, `Viable` is **false**, the `Reason` names the unmet prerequisite, and a `Note` states what the user must supply so the row is not read as "this network blocks it". `Candidate.Requires() []Requirement` (PRD §5.2) declares the prerequisite set statically; `Feasibility.Requires` carries the evaluated rows; a test asserts every declared kind appears among the evaluated rows.

Concretely for `cloudflare-tunnel` on a reachable edge with no hostname: `viable:false`, `requires:[{kind:"hostname",satisfied:false}]`, reason names the missing hostname, note says the edge is reachable and this is an account prerequisite, not a network block.

**Why.** §14.5 forbids a "sort of viable" state, and "not ready to use" is only unambiguously true if `viable` is false. Keeping the unmet prerequisite in a typed array means a script sees the reason without parsing prose.

**Trade-off.** The most attractive transport reads `viable:false` on a healthy network, which is discouraging until the note is read; the alternative is a boolean that means "possibly", which is worse. `Feasibility` gains a field versus PRD §5.2's three-field struct — recorded as a deviation in §2.

**Rejected alternatives.** A third state such as `partial` / `conditional` (explicitly ruled out by the spec and by §14.5). `viable:true` + a note (weak for scripts and a fabricated success for a transport that cannot work yet). `requires` only in the DTO with the internal struct left faithful (then `Feasible` has no way to report satisfaction, so the DTO value would be invented at the mapping layer).

### D7 — Transport interface shape

**Decision.** Both contracts exist in `internal/transport/contract.go`:

- `Candidate` — `Name()`, `Requires() []Requirement`, `Feasible(diagnosis.Diagnosis) Feasibility`. This is the **registration** contract; the four R1a adapters implement it, and a later detect-only adapter (or a test-only transport) needs nothing more.
- `Transport` — embeds `Candidate` and adds `PlanHub`, `PlanNode`, `Verify` (PRD §5.2). The four R1a adapters also implement it, and each of those three members returns a typed `ErrNotImplementedInThisPhase` naming the member and its owner slice (`PlanHub`/`PlanNode` → R3, `Verify` → R5/R6). **No value is returned alongside the error** — never an empty slice plus a nil error.

The types those three members need are declared once, in `internal/transport/pending.go`, as empty placeholder structs (`PairingBundle`, `Step`, `Handle`) with a doc comment naming the owning slice and the removal trigger. That file is the single migration point when R2/R3/R5 define the real types.

**Why.** The spec makes the loud failure a MUST: a plan-producing member must exist and must fail with a typed error naming the member and owner slice, so no caller can receive a plausible-looking empty plan. A registry that accepted only `Transport` would force every future detect-only adapter to carry three failing stubs, hence the two-level contract.

**Trade-off.** Empty placeholder structs ship, and the three signatures change when R2/R3 land (four adapters × three methods, mechanical). Rejected alternative: a narrow `Candidate` grown later — no placeholders, but the spec's loud-failure requirement would be unimplementable in this slice and a caller could get an empty plan for as long as the interface does not exist. Also rejected: `any`-typed parameters to avoid placeholders (weaker typing for no saved churn).

### D8 — What `egress.quic` measures, and the evidence that settles it

**Decision.** Raw UDP datagram, **no QUIC dependency in R1a**. The probe's declared question is narrow and is stated in its own detail text: *does a UDP datagram to the tunnel edge on 7844 elicit any UDP response inside the probe budget?*

| Observation | Resolution | Reason code |
|---|---|---|
| Any UDP response datagram | measured / `pass` | `udp_response_received` |
| No response, no error, budget expired | unresolved / `indeterminate` | `udp_silence` |
| ICMP port-unreachable surfaced by the socket | measured / `fail` | `udp_unreachable` |
| Any other socket error | unresolved / `indeterminate` | `udp_error_unclassified` |

`pass` claims only "UDP to this edge and port was not silently dropped"; the probe never says "QUIC works" and never says "QUIC is blocked". A real QUIC handshake would be the only way to establish QUIC viability, and the answer it would change is only *"recommend HTTP/2 or not"* — a question the unresolved path already answers conservatively (advise HTTP/2, naming the unresolved measurement, §5.2 rule `CF_HTTP2_ADVISED_QUIC_UNCONFIRMED`). Paying for a heavy QUIC stack and its TLS policy to sharpen a recommendation this slice does not enforce is the wrong trade.

**Evidence that settles it (verify-phase obligation, not a unit test).** A hand-run on the motivating network comparing this probe's raw-UDP result against `cloudflared`'s own QUIC log lines (PRD §8.2's transcript is the reference: `UDP Connectivity region1/region2 … FAIL QUIC connection failed`). The transcript is recorded in the verify phase. If it shows the probe's `pass` never occurs live, the probe remains a negative-signal-only probe and **nothing downstream changes**, because the Cloudflare adapter's advice does not depend on which non-`pass` state occurred.

**Trade-off.** A probe that cannot establish the positive case on a real network (a raw datagram will rarely draw a reply from a QUIC edge) — accepted because its failure modes are exactly the ones that matter (ICMP ⇒ definite negative; silence ⇒ never reported as a block), and because it costs no dependency and no TLS policy surface. Rejected alternatives: a full QUIC handshake via a third-party stack (heavy dependency, larger review surface, a new trust decision inside the probe); treating UDP silence as `fail` (explicitly forbidden by the spec and the source of RG-5's wrong HTTP/2 recommendation).

### D9 — Default probe concurrency

**Decision.** Default concurrency **4**, settable at construction (`probe.Options.Concurrency`), not exposed as a CLI flag. A test asserts the bound is honoured (never more than N probes in flight, measured by a seam that counts concurrent entries).

**Why.** Ten simultaneous outbound connections from a machine that may be under an endpoint agent's inspection is a self-inflicted burst; four keeps a full run well inside the 60 s budget while staying gentle. No PRD requirement covers this, so it is recorded as a judgment call.

**Trade-off.** A full run can be slower than an unbounded burst would be (bounded by the slowest probe group rather than the slowest probe). Rejected alternative: a `--concurrency` flag — it turns a tuning judgment into a public contract that a user can raise to 10, which is the exact burst the default avoids. If a hand-run on the motivating network shows agent interference at 4, the constant drops; that is a code change, not a flag.

### D10 — `region2`, and target-list overridability

**Decision.** **Both** edge regions are probed, inside the existing probes rather than as extra registry entries (the registry must contain exactly ten probes): `egress.cf.7844`, `egress.cf.443` and `egress.quic` each produce **one observation per region**, so a split result (`region1` reachable, `region2` not) is representable instead of hidden behind one aggregate.

The declared target set lives in one place, `internal/probe/targets.go`, and is overridable by repeatable `--target <probe-name>=<host[:port]>`, which **replaces** that probe's declared targets (never appends), so the effective set is closed and computable. An unknown probe name or a malformed address is a usage error (exit `2`). The effective declared set is echoed in the payload so "the dialed set equals the declared set" is checkable from the output alone.

**Why.** PRD §8.2's measured transcript fails on both regions, and "one region reachable" is materially different from "both blocked" — a fact the Cloudflare adapter and the user both need. Overridability is required by the spec (`R-HR-NF-10`), and replace-not-append is what keeps the closed-set assertion honest.

**Trade-off.** Two more observations per edge probe and one more flag. Rejected alternatives: probing only `region1` (loses a measurable difference and contradicts the recorded transcript); a `--targets-file` config file (invents a format that R2 replaces, and adds a file read to a slice whose proof is "reads nothing outside the seams"); no override at all (spec violation).

---

## 2. Recorded deviations

Three deviations from PRD §7.1/§5.1/§5.2 are deliberate and are the only ones this design introduces. Everything else matches the specification's own sketch.

| # | Deviation | Why | Rejected alternative |
|---|---|---|---|
| DEV-1 | New package `internal/diagnosis` holds classification consumption and the rule table; PRD §7.1 puts the rule in `internal/probe/egress.go` and lists no `diagnosis` package | Keeps reasoning free of `net`, makes NF-04 a testable boundary, and gives the rule table one unit-testable home | PRD-literal placement in `probe/egress.go` (rule shares a package with I/O; NF-04 unprovable) |
| DEV-2 | `probe.Result` gains `Resolution`, `Reason` and `Observations []Observation` beyond PRD §5.1's six fields | The spec requires a stable machine reason code beside verbatim detail, and requires `local.sshd`'s three sub-observations to be separately reportable while the verdict vocabulary stays exactly `pass`/`fail`/`indeterminate` | Deriving the reason code in the report layer (duplicates parsing, drifts across Go versions) and encoding not-measured with a fourth verdict (violates PRD §5.1's enum) |
| DEV-3 | `transport.Feasibility` gains `Requires []Requirement` beyond PRD §5.2's `Viable`/`Reason`/`Notes` | The spec requires a machine-readable prerequisite; without a typed row the mapper would have to invent satisfaction | DTO-only `requires` (internal representation then cannot record satisfaction), or `viable:true` with prose (fabricated success) |

`internal/doctor` is **not** a deviation: PRD §7.1 lists `doctor/doctor.go` ("environment checks for the local machine"), which is exactly the run harness this design places there.

---

## 3. Contracts

### 3.1 Measurement types (`internal/probe`)

```go
type Verdict string      // "pass" | "fail" | "indeterminate"  (PRD §5.1 — unchanged)
type Resolution string   // "measured" | "unresolved" | "not_measured"

type Observation struct {
    Label      string      // "tcp 7844 region1", "effective sshd config", "binary present"
    Target     string      // "" only for a not-measured observation
    Resolution Resolution
    Verdict    Verdict     // pass/fail for measured; indeterminate for the other two
    Reason     ReasonCode
    Detail     string      // verbatim, quotable
}

type Result struct {      // PRD §5.1 fields kept at top level
    Probe    string
    Kind     ProbeKind
    Target   string
    Verdict  Verdict      // worst-of aggregate over Observations (documented order)
    Reason   ReasonCode   // reason of the worst observation
    Detail   string       // per-observation lines, verbatim
    Elapsed  time.Duration
    Observations []Observation
}
```

Invariants, each backed by a test: `Verdict ∈ {pass, fail}` ⟺ `Resolution == measured`; `Resolution ∈ {unresolved, not_measured} ⇒ Verdict == indeterminate`; aggregate order is `fail > indeterminate > pass`, so an unresolved observation can never be promoted to `pass` and a not-measured one can never be turned into `fail`. Completeness is computed from observations, not verdicts (§1 D5 reconciliation).

### 3.2 Transport contract (`internal/transport`)

```go
type Requirement struct { Kind RequirementKind; Satisfied bool; Detail string }
type Feasibility struct { Viable bool; Reason string; Notes []string; Requires []Requirement }

type Candidate interface {
    Name() string
    Requires() []Requirement                                   // static declaration
    Feasible(d diagnosis.Diagnosis) Feasibility                 // evaluated, no measurement
}

type Transport interface {
    Candidate
    PlanHub(PairingBundle) ([]Step, error)
    PlanNode(PairingBundle) ([]Step, error)
    Verify(ctx context.Context, h Handle) ([]Result, error)
}
```

`Reason` precedence, documented and tested: (1) a measured blocking observation, (2) an unresolved dependency of the decision, (3) an unmet requirement. `!Viable ⟹ Reason != ""` holds by construction and by a loop assertion over the four adapters. A `Viable` transport carries at least one named measured observation in `Notes`.

> **Dated note — 2026-09-20, PR 13 (S4) adjudication.** `diagnosis.Finding` gains `Evidence []Fact` — the observations the fired rule matched, filled by `Diagnose` from the facts it already holds — because `Feasible(d diagnosis.Diagnosis)` is fixed here and R-HR-06's scenario requires a rejection to name the measured target and its port, which the prose `Conclusion` cannot supply structurally. The payload shape of §3.3 is unchanged. Three calls taken from this design rather than from prose: `Verify` returns `([]probe.Result, error)` (the unqualified `Result` above is `probe.Result`, since D7's placeholder list names only `PairingBundle`, `Step` and `Handle`); the R1a `Requires()` mapping is one requirement set per adapter — `direct-ssh` declares `hub_address`, `reverse-ssh` declares `hub_address` and `sshd_effective_config`, and PR 14 adds `cloudflare-tunnel` (`hostname`, `zone_membership`) and `tailscale` (`third_party_permission`), which is the whole closed kind set of D6; and a prerequisite is `Satisfied` when the run supplied or measured it, so a measured block leaves `Satisfied` true while `Viable` is false, because the block is a network fact rather than a missing input.

> **Dated note — 2026-09-20, PR 14 (S5).** The `cloudflare.edge` half of `cloudflare-tunnel`'s viability is satisfied by a **measured pass**, and a measured pass includes `CF_EDGE_PARTIAL`: a split means at least one declared endpoint answered inside the budget, so the edge is never reported as the fact that rejects this transport — the unmet account prerequisite is what keeps it non-viable, and the split is named as the qualification instead. `CF_EDGE_UNREACHABLE` (every measured endpoint failed, no measured success, no unanswered attempt) is the measured block that takes this section's first precedence slot, and `CF_EDGE_UNRESOLVED` the unresolved dependency that takes the second. The account note D6 requires is emitted whenever the edge half is satisfied, so a partly reachable edge cannot be read as a network block.

> **Dated note — 2026-09-20, PR 15 (S6).** The `cloudflared` pin statement lives in `internal/transport/pin_note.go` and is carried on every `cloudflare-tunnel` row's notes, whatever the measured edge state and whether or not the HTTP/2 advice fired: R-HR-16 constrains the statement's wording, not its presence, and a reader who has to pin a client needs the record even on a run whose edge did not answer. The HTTP/2 advice is informational in the same way — it never changes `Viable`, which no R1a input can make true — and the notes it contributes state its boundary: this slice recommends and does not enforce, so no fallback was applied and no configuration was written. R-HR-16's single home is proved by scanning for the statement's **exact text** across the repository's non-test sources and shipped docs, excluding this change's own artifacts and the ODD tracking directory; the project's prose about the upstream report in `README.md` and `PRD.md` is not the statement, which is why the proof counts the text and not a fragment like `#1673`.

### 3.3 JSON payload (`internal/report`, `schema_version: "1"`)

One document on stdout, ordered, no maps. Shape (field names as emitted):

```jsonc
{
  "schema_version": "1",
  "generated_at": "2026-09-14T12:00:00Z",          // one injected clock, RFC3339 UTC
  "tool": { "name": "herdr-reach", "version": "0.0.0-dev" },
  "run": { "completeness": "complete|incomplete", "unresolved": ["tls.truststore"],
           "not_measured": ["egress.hub.direct"], "concurrency": 4, "run_budget_ms": 60000 },
  "targets": { "hub": "203.0.113.10:22",            // null when no --hub was supplied
               "declared": [ { "probe": "...", "target": "...", "protocol": "tcp|udp|tls" } ] },
  "probes": [ { "name", "kind", "target", "verdict", "resolution", "reason",
                "detail", "elapsed_ms",
                "observations": [ { "label", "target", "verdict", "resolution", "reason", "detail" } ] } ],
  "findings": [ { "question", "rule", "conclusion", "depends_on": ["probe", "..."] } ],
  "open_questions": [ { "question": "...", "needed_states": ["..."] } ],
  "node": { "platform": "linux|macos|wsl2|windows-native|unknown",
            "arch": "...", "refused": false, "note": "..." },
  "transports": [ { "name", "viable", "reason", "requires": [ { "kind", "satisfied", "detail" } ],
                    "notes": ["..."] } ]
}
```

Rules: probes in registry order; `observations` in declaration order; `transports` sorted by name; `elapsed_ms` an integer; `generated_at` from the single injected clock; no field derived from treating an unresolved observation as a `pass`; the DTO is produced by exactly one mapping function. A test asserts the exact key set of each level, so an added boolean or renamed field is a deliberate act.

### 3.4 Exit codes (documented in `docs/diagnosis-report.md`, constants in `internal/doctor/exit.go`)

| Code | Meaning | Evidence |
|---|---|---|
| `0` | Measurement completed — including "no transport viable" and a native-Windows refusal | every probe resolved (no unresolved observation) |
| `1` | Run incomplete — at least one probe was attempted and resolved unresolved | `run.completeness == "incomplete"` |
| `2` | Usage or internal error — no diagnosis is presented as completed; stdout stays empty | bad flag/value, unknown `--target` name, `--hub` unusable, JSON write failure |

A not-measured observation alone never changes the exit code but always appears in `run.not_measured`. A default live run **with no `--hub`** on Linux therefore exits `0` with a named coverage gap (hub not measured; `sshd -T` not measured by this slice's boundary). A default live run on **macOS exits `1`**, because `tls.truststore` is attempted and unresolved there by decision (RG-3) — stated here explicitly so nobody reads the Linux expectation as universal.

### 3.5 Reason codes — closed set (28), `internal/probe/reason.go`

_Count reconciled 2026-09-14: this heading said 27 while the enumeration below lists 28 distinct codes, and the classification table needs all of them (`ok` is the only code two rows share). The enumeration is authoritative; the PR 2 apply evidence records the check._

`ok`, `conn_refused`, `conn_reset`, `budget_expired`, `dns_no_such_host`, `dns_unresolved`, `banner_not_ssh`, `probe_timeout`, `run_cancelled`, `run_budget_exceeded`, `udp_response_received`, `udp_silence`, `udp_unreachable`, `udp_error_unclassified`, `tls_verify_failed`, `tls_issuer_unexpected`, `tls_handshake_unresolved`, `truststore_rejects_chain`, `truststore_platform_unavailable`, `truststore_override_platform_bypass`, `sshd_absent`, `sshd_config_divergence`, `capability_excluded`, `command_denied`, `input_missing_hub`, `platform_unknown`, `node_platform_unsupported`, `internal_error`.

**Rule: adding or renaming a code is a contract change.** The code is stable across Go versions and operating systems by construction: it is chosen from the classification table (§5.1), never parsed out of an error string, and the OS-specific wording lives only in `detail`. A test parses the table in `docs/diagnosis-report.md` and asserts set equality with `probe.AllReasonCodes()`, so a code cannot be added without updating the documented set.

### 3.6 Rule ids

Fact questions use a derived id: `<PROBE_NAME_UPPERCASED_WITH_UNDERSCORES>_<STATE>`, where state ∈ `PASS | FAIL | UNRESOLVED | NOT_MEASURED` (e.g. `EGRESS_HUB_DIRECT_FAIL`). Derived questions use hand-named ids (§5.2). A test asserts that every registered probe × every possible resolution state has exactly one rule id, so no observable is unclassified. Rule ids appear verbatim in the payload and in the human projection (PRD §5.3: the report must quote which rule fired), and are documented in the same doc as the reason codes.

---

## 4. Data flow

```
cmd/herdr-reach/main.go
  parse flags                      → doctor.Options (hub, targets override, json, version)
  build production seams           → probe.Seams            (the only file constructing real net/tls)
  os.Exit(doctor.Run(ctx, opts, seams))

doctor.Run
  probe.EffectiveTargets(opts)                       // declared set + overrides, one place
  probe.Registry()                                   // exactly ten probes, deterministic order
  probe.NewRunner(opts.Round).Run(ctx, probes, targets, seams)
      → streamed []probe.Result, per-probe bound, global 60 s budget, concurrency 4, cancellable
  diagnosis.Diagnose(results)                        // pure: findings + open questions
  transport.Evaluate(transport.Registry(), diagnosis) // pure: viable + reason + requires + notes
  report.Build(clock, version, targets, results, diagnosis, feasibility)   // one mapping
  report.WriteHuman(stderr)      // always
  report.WriteJSON(stdout)       // only with --json
  → exit code from completeness
```

> **Dated note — 2026-09-20, PR 16 (S7).** The mapping takes a documented input struct rather than the illustrative positional call above, because §3.3's `run` requires `concurrency` and `run_budget_ms` while that call carries neither: the honest source of both, and of the single clock that stamps `generated_at`, is the run's **effective `probe.Options`** after its documented defaults, which is exactly what `Runner.Options()` reports. `transport.Feasibility` carries no transport name while §3.3 requires one per row, so the transports reach the mapping paired with their names in a report-local type, which makes a name impossible to misalign with its viability silently. The mapping is the only producer of the DTO, and design D3's nullable target is what makes an absent target JSON `null` instead of an empty string: the probe rows, their observations and `targets.hub` are the only nullable fields, and every key is always present.

Nothing in this path writes, creates, moves or deletes anything; the only outbound traffic is the declared measurements (§6).

---

## 5. Classification, reasoning and the five spec obligations

### 5.1 Classification table (`internal/probe/classify.go`)

| Observable (raw) | Resolution | Reason code | Class |
|---|---|---|---|
| TCP established (reachability probe) | measured | `ok` | definite positive |
| TCP refused (`ECONNREFUSED`) | measured / `fail` | `conn_refused` | definite negative |
| TCP reset after connect | measured / `fail` | `conn_reset` | definite negative |
| TCP connected, banner is not SSH (for `egress.ssh.*`) | measured / `fail` | `banner_not_ssh` | definite negative (interception or proxy) |
| **probe's own dial budget expired** | measured / `fail` | `budget_expired` | definite negative **only because the probe's declared question is "is this port reachable"** |
| resolver authoritative negative (no such host) | measured / `fail` | `dns_no_such_host` | definite negative |
| resolver timeout / SERVFAIL / no resolver | unresolved | `dns_unresolved` | ambiguous: not a measurement of the destination |
| **probe ignored its budget and never returned** | unresolved | `probe_timeout` | runner-classified — never probe-classified |
| run context cancelled | unresolved | `run_cancelled` | runner-classified |
| global run budget exhausted | unresolved | `run_budget_exceeded` | runner-classified |
| UDP reply received at the edge | measured | `udp_response_received` | narrow positive (D8) |
| **UDP silence, no error** | unresolved | `udp_silence` | ambiguous by construction; never a block |
| ICMP port-unreachable surfaced by the socket | measured / `fail` | `udp_unreachable` | definite negative |
| other UDP socket error | unresolved | `udp_error_unclassified` | ambiguous |
| TLS verified, issuer in the declared expected set | measured | `ok` | definite positive |
| TLS verification failed (code in `detail`) | measured / `fail` | `tls_verify_failed` | definite negative |
| issuer not in the declared expected set for the target | measured / `fail` | `tls_issuer_unexpected` | interception candidate; wording says "not in the declared expected set", never accuses |
| TLS handshake error, neither of the above | unresolved | `tls_handshake_unresolved` | ambiguous |
| local trust pool rejects the chain (Linux) | measured / `fail` | `truststore_rejects_chain` | definite negative |
| macOS platform verifier cannot answer | unresolved | `truststore_platform_unavailable` | capability attempted and unusable |
| `SSL_CERT_FILE`/`SSL_CERT_DIR` override bypasses the platform verifier | unresolved | `truststore_override_platform_bypass` | capability bypassed — never a false pass |
| `sshd` binary absent | measured / `fail` | `sshd_absent` | measured negative (with "installation is a later slice") |
| written ≠ effective configuration | measured / `fail` | `sshd_config_divergence` | definite negative |
| `sshd -T` capability excluded by this slice's zero-execution boundary | **not_measured** | `capability_excluded` | attempt not made by design |
| command seam denied the execution | **not_measured** | `command_denied` | attempt not made; reason names the capability |
| no `--hub` supplied | **not_measured** | `input_missing_hub` | attempt not made for lack of input |
| platform signals match no supported classification | unresolved | `platform_unknown` | ambiguous; no default guess |
| classified native Windows | measured / `fail` | `node_platform_unsupported` | measured negative answer for this tool's purpose |
| unexpected internal failure | unresolved | `internal_error` | never `pass` |

**The three timeout paths stay distinct** (obligation 3): a reachability probe's own budget is shorter than the runner's per-probe bound, so a blackholed port is `fail`/`budget_expired` **produced by the probe**; a probe that ignores its context is `indeterminate`/`probe_timeout` **produced by the runner**; UDP silence is `indeterminate`/`udp_silence`. One test scripts all three side by side and asserts the three distinct `(resolution, reason)` pairs — collapsing any two is the wrong-classification risk RG-8.

**Attempted vs not attempted** is the rule that separates `unresolved` from `not_measured` (obligations 1 and 2): an attempt was made and produced nothing classifiable ⇒ `unresolved` (and it makes the run incomplete); no attempt was made because an input or capability is outside this slice's boundary ⇒ `not_measured` (and it does not). A platform capability that is *attempted and unusable* (macOS trust store) is `unresolved` by decision; a capability this slice **never attempts** (`sshd -T`, because no production `CommandRunner` is wired) is `not_measured`.

**Obligation 2, consequence made visible.** `doctor.Run` constructs seams with **no command runner**. A nil runner is treated by every probe as "capability excluded", so the live `sshd -T` observation is not-measured with `capability_excluded` and the capability name in `detail`; wiring a deny-all runner instead yields `command_denied`. Both outcomes are tested, so the deferral is provably a boundary and not a broken probe. The gap appears in `run.not_measured`, in the human projection's coverage section, and in the `local.sshd` finding's conclusion text.

> **Dated note — 2026-09-20, issue #59.** The §5.1 row *"issuer not in the declared expected set for the target"* is revised: it is no longer `measured / fail` but `unresolved` with reason code `tls_issuer_unexpected`. A run on a healthy network with no interception observed `www.cloudflare.com:443` presenting a chain that verified (code `0`) and whose publisher was `Google Trust Services/GlobalSign` — a rotation of the publisher PRD §1.1 recorded — while the machine's own trust store accepted the same chain. The single `Let's Encrypt/ISRG` entry in `declaredExpectedIssuers` turns the first rotation into a guaranteed false failure, and a failure has to be earned as much as a pass. The probe still fails when the chain does not verify (`tls_verify_failed`), which is the one chain fact it can establish; the divergence is now reported for exactly what the run can and cannot establish — the chain verified, the observed publisher, the declared set it was compared against, and that this run cannot distinguish a publisher change from an interception. The observation is therefore an absence, the run is incomplete and exits `1`; where a plugin host reads a non-zero action status as a failed action (issue #58 tracks that reading for Herdr), a publisher divergence now reaches it. `declaredExpectedIssuers` is deliberately not expanded: adding `Google Trust Services` because it was observed once would be a copy of an observation that could drift, not a declaration. This note supersedes the §5.1 row above and the §5.2 sentence that the failure state names the unexpected issuer; the history above is left as it was written.

### 5.2 Rule table (`internal/diagnosis/rules.go`)

Mechanism: `[]Rule{ID, Question, Match []Need, Conclusion, DependsOn}` where `Need{Probe, Label?, Resolution, Verdict}` is an exact observable state. The probe, the resolution and the verdict are always required; `Label` is optional, and an empty label means "any observation of the probe" while a non-empty one matches only the observation the probe reported under that label. The label exists because one probe can report several observations with identical triples — `local.sshd`'s absent binary and its stopped service are both measured failures carrying the vocabulary's shared `sshd_absent` code — and this table's rows for that probe have to tell them apart. Evaluation: for each question, the **first matching rule wins**; every rule is a pure predicate over the observations. **There is no fall-through default**: a question with no matching rule emits an explicit `open_question` naming the states the table needed, spelling a labeled state as `<probe> (<label>) <STATE>`. `DependsOn` is exactly the set of observables in `Match`, which is how the payload can show what each conclusion rests on.

**Unresolved propagation is structural, not a check** (obligation 5): a rule that needs `measured/pass` or `measured/fail` cannot fire when the observation is unresolved or not measured, so the confident conclusion is simply unreachable, and the weaker rule — which requires the unresolved state — is the one that matches and names the unresolved probe. No rendering decision is involved.

> **Dated note — 2026-09-19, PR 12 adjudication (S3).** Three parts of this table could not coexist with §3.6 or with the `Need` triple as written. PR 12 settled them before writing any id:
>
> 1. **The `tls.interception` and `tls.truststore` rows are restated to their derived ids** (`<PROBE>_<STATE>`). Their own `Requires` column is *a derived state of the probe*, so the hand-named spellings were renames of the derived fact rules rather than a second interpretation, and two of them (`TRUSTSTORE_UNRESOLVED_PLATFORM` and `TRUSTSTORE_UNRESOLVED_OVERRIDE`) are one `(tls.truststore, unresolved, indeterminate)` triple — shipping them would have left one id unreachable and a third state unnamed. The platform/override split stays where the vocabulary already carries it: the reason code (`truststore_platform_unavailable` vs `truststore_override_platform_bypass`) and the verbatim detail the conclusion prints, which is exactly what RG-3 requires the output to state. The two fact questions therefore keep §5.2's own question names (`tls.interception`, `tls.truststore`); only `local.sshd` keeps a `.fact` suffix, because its group owns the probe's name.
> 2. **`local.sshd`'s rows are expressed through `Need.Label`** (mechanism above), which is what lets an absent binary, a stopped service and a diverging configuration be told apart and keeps `SSHD_ABSENT` unreachable for a stopped service.
> 3. **`NODE_WSL2_SYSTEMD_ABSENT` is not implemented in R1a.** The service-manager signal has no structured home — the observation's target carries `<platform>|<arch>` and the systemd state lives only in the verbatim detail — and matching a conclusion on wording is forbidden (R-HR-07), so the id belongs to the slice that carries the signal structurally. The WSL2 wording and its derived fact conclusion already report the systemd state and that enabling it is a later slice, which is the RG-4 scenario.
>
> Consequence inherited by the documentation slice: `docs/diagnosis-report.md`'s rule-id table is compared with `AllRuleIDs()` by set equality, so it lists the derived tls ids and none of the spellings above.

| Question | Rule id | Requires | Conclusion |
|---|---|---|---|
| `ssh.public_22` | `EGRESS_SSH_KNOWN_PASS/FAIL/UNRESOLVED/NOT_MEASURED` | derived state of `egress.ssh.known` | the plain fact |
| `ssh.public_443` | `EGRESS_SSH_443_*` | derived state of `egress.ssh.443` | the plain fact |
| `hub.reachability` | `EGRESS_HUB_DIRECT_PASS/FAIL/UNRESOLVED/NOT_MEASURED` | derived state of `egress.hub.direct` | the plain fact; `FAIL` names target **and** port; `NOT_MEASURED` states the measurement was not made and names the missing input |
| `ssh.destination` | `SSH_DEST_BLOCKED_BY_PUBLIC_SSH` | `ssh.public_22 = pass` ∧ `hub.reachability = fail` | "outbound SSH is allowed; the hub address `<target>` is blocked" — the PRD §13 case |
| | `SSH_DEST_BLOCKED_BY_PUBLIC_SSH_443` | `ssh.public_22 ≠ pass` ∧ `ssh.public_443 = pass` ∧ `hub.reachability = fail` | same conclusion, naming :443 as the allowed port |
| | `SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED` | `hub.reachability = fail` ∧ (`ssh.public_22` or `ssh.public_443` ∈ {unresolved, not_measured}) | **weaker**: the hub did not answer, and whether outbound SSH is allowed is *not established*; names the unresolved probe; never "SSH is blocked" |
| | `SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_FAILED` | `hub.reachability = fail` ∧ `ssh.public_22 = fail` ∧ `ssh.public_443 ≠ pass` | weaker: both public targets and the hub failed; protocol-level vs destination-level is not established |
| | `SSH_NO_DEST_BLOCK_OBSERVED` | `hub.reachability = pass` ∧ `ssh.public_22 = pass` | no destination block observed |
| | `SSH_DEST_BLOCK_NOT_ASSESSED` | `hub.reachability ∈ {unresolved, not_measured}` | nothing is claimed about the hub; this is the default live case |
| `egress.quic` | `EGRESS_QUIC_PASS/FAIL/UNRESOLVED/NOT_MEASURED` | derived state | the narrow fact of D8 |
| `cloudflare.edge` | `CF_EDGE_REACHABLE` / `CF_EDGE_PARTIAL` / `CF_EDGE_UNREACHABLE` / `CF_EDGE_UNRESOLVED` | per-region observations of `egress.cf.443` (and `egress.cf.7844`) | at least one region reachable / split / all measured regions failed / no measured success and at least one unresolved |
| `cloudflare.http2` | `CF_HTTP2_ADVISED_QUIC_FAILED` | `egress.quic = fail` ∧ `cloudflare.edge ∈ {reachable, partial}` | advise HTTP/2, state the post-quantum trade-off, state that this slice recommends and does not enforce |
| | `CF_HTTP2_ADVISED_QUIC_UNCONFIRMED` | `egress.quic ∈ {unresolved, not_measured}` ∧ edge reachable | same advice, and the note names the unresolved measurement it rests on |
| | `CF_NO_HTTP2_ADVICE_QUIC_USABLE` | `egress.quic = pass` | no downgrade recommended |
| | `CF_HTTP2_ADVISORY_NOT_ASSESSED` | edge unreachable/unresolved | no advice |
| `tls.interception` | `TLS_INTERCEPTION_PASS` / `TLS_INTERCEPTION_FAIL` / `TLS_INTERCEPTION_UNRESOLVED` / `TLS_INTERCEPTION_NOT_MEASURED` (derived) | derived state of `tls.interception` | observed issuer + verification code always present; the failure state names the unexpected issuer or the verification failure through its reason code |
| `tls.truststore` | `TLS_TRUSTSTORE_PASS` / `TLS_TRUSTSTORE_FAIL` / `TLS_TRUSTSTORE_UNRESOLVED` / `TLS_TRUSTSTORE_NOT_MEASURED` (derived) | derived state of `tls.truststore` | the platform limitation is emitted **in the output**, not only in code (RG-3); the two unresolved reasons are told apart by the reason code and the verbatim detail the conclusion carries |
| `local.sshd` | `SSHD_PRESENT_CONFIGURED` | binary present ∧ effective config measured ∧ no divergence (each observation by its `Need.Label`) | positive; the service state is deliberately not required, and the fact layer reports it as its own finding |
| | `SSHD_PRESENT_CONFIG_DIVERGENT` | divergence measured | names both configurations (never "success") |
| | `SSHD_ABSENT` | binary measured absent (by its `Need.Label`) | states that installation is a later slice; unreachable for a stopped service |
| | `SSHD_EFFECTIVE_CONFIG_NOT_MEASURED` | effective-config observation not measured | weaker: names the excluded capability; this is the default live case |
| `node.platform` | `LOCAL_ENV_PASS` / `LOCAL_ENV_FAIL` / `LOCAL_ENV_UNRESOLVED` | derived state of `local.env` | classification + arch |
| | `NODE_PLATFORM_SUPPORTED` / `NODE_PLATFORM_REFUSED_NATIVE_WINDOWS` / `NODE_PLATFORM_UNKNOWN` | classification result | the refusal names WSL2 as the supported path; no transport is viable for a refused node |
| | ~~`NODE_WSL2_SYSTEMD_ABSENT`~~ **deferred** | WSL2 ∧ systemd not enabled | **not implemented in this change** — the service-manager signal has no structured home and matching on wording is forbidden; see the dated PR 12 note above |

WSL2 text is limited to the documented semantics (milliseconds idle, default 60000, Windows 11 only). The child-of-init rule and the `vmIdleTimeout=-1` sentinel must not appear anywhere in R1a output (RG-4); a test asserts their absence from both projections.

### 5.3 Obligation 1 — not measured vs unresolved vs incomplete, in the payload

| Concept | Result field | Payload | Exit code |
|---|---|---|---|
| not measured | `resolution: "not_measured"`, `reason` names the missing input/capability, `target` null for the hub | `probes[].resolution`, listed in `run.not_measured` | unchanged |
| attempted, unresolved | `resolution: "unresolved"`, `verdict: "indeterminate"` | `probes[].resolution`, listed in `run.unresolved` | `1` |
| incomplete run | — | `run.completeness: "incomplete"` + named unresolved probes | `1` |

A default live invocation on Linux exits `0` with a healthy environment: `local.env` passes; `local.sshd` reports the binary observation and one not-measured observation; the four egress probes and the two TLS probes are scripted-measured in tests and, live, are measured or fail; `egress.hub.direct` is not measured because no `--hub` was given; no observation is unresolved ⇒ complete ⇒ `0`, with the coverage gaps named in `run.not_measured` and printed in the human projection.

### 5.4 Obligation 4 — reason codes and rule ids have one documented home

`docs/diagnosis-report.md` documents the exit codes, the stdout/stderr split, the payload shape with its `schema_version`, the 28 reason codes, and the rule-id scheme, with the statement that adding a code or rule id is a contract change. `internal/report/docs_test.go` parses the two tables and asserts equality with `probe.AllReasonCodes()` and `diagnosis.AllRuleIDs()`. The `cloudflared` pin note has its own single home in code (`internal/transport/pin_note.go`), and the HTTP/2 trade-off text has one home in `internal/transport/cloudflare.go`, so the human and machine projections cannot drift.

---

## 6. Seams, the deny-all test default, and the no-egress proof

### 6.1 Seam set (`internal/probe/seams.go`)

| Seam | Shape | Used by |
|---|---|---|
| `Dialer` | `DialContext(ctx, network, addr) (net.Conn, error)` | all TCP reachability probes, TLS probes |
| `Resolver` | `LookupHost(ctx, host) ([]string, error)` | probes whose declared target is a name |
| `TLSVerifier` | `Verify(ctx, target, cfg) (Observation, error)` with an injectable `*tls.Config` and root pool | `tls.interception`, `tls.truststore` |
| `PacketDialer` / `PacketConn` | `DialPacket(ctx, network, addr)`, then `WriteTo` / `ReadFrom` / `SetDeadline` / `Close` | `egress.quic` |
| `CommandRunner` | `Run(ctx, name, args...) (stdout, stderr []byte, err error)` | `local.sshd` service state and effective config |
| `Clock` | `Now() time.Time` | `elapsed_ms`, `generated_at` |
| `FS` | `ReadFile(path)`, `Stat(path)`, `Getenv(name)` | `local.env`, `local.sshd` written config, trust-store overrides |
| `Platform` | `GOOS()`, `Arch()`, WSL2/systemd signals | `local.env` classification |

All eight are fields of one injected `probe.Seams` struct, passed explicitly — no package-level `var dial = net.Dial` swapping (it cannot run under `t.Parallel()`, hides coupling, and lets a sub-test forget to swap and then dial the real network).

> **Dated note — 2026-09-20, PR 19 (S10).** Three conventions of the production seam set, recorded because later slices inherit them. **(1)** The `PacketDialer` is built over a **connected** datagram socket (`net.Dial` with `udp`), not `net.ListenPacket`: binding a socket to a *remote* unicast address fails with `EADDRNOTAVAIL` (verified locally), and a wildcard bind would additionally hide the ICMP port-unreachable the classification table reads as `udp_unreachable`. `net.ListenPacket` therefore stays inside the guard's forbidden set rather than becoming the production path. **(2)** The chain digest is `<leaf issuer organization>/<topmost certificate short name>` — `Let's Encrypt/ISRG` for the chain PRD §1.1 recorded — and it is exactly the string `declaredExpectedIssuers` compares against, so the fold in `real.go` and the declared set in `tls.go` are two halves of one contract; the suite pins that they agree, and the fold is a deliberate reduction, not a claim about the wire, because a real chain carries one issuer per certificate. **(3)** The verifier refuses a nil configuration and one with `InsecureSkipVerify` set, and never mutates the caller's config (R-HR-04); the refusals are pinned by the internal-package cases of `real_test.go`, while `real.go`'s live path has no test by design — the production seam set is called only from `main.go` — so its live behaviour stays a verify-phase hand-run obligation. A fourth, smaller convention: the entrypoint answers a refused command line by asking the harness for its help mode and discarding that mode's code, because the usage text has exactly one home in the flag surface.

### 6.2 Test default: `probe.DenyAllSeams()`

One exported constructor returns seams where every dial, lookup, TLS verification and packet operation fails with a distinctive sentinel error (`errNetworkForbiddenInTest`), the command runner denies everything, `FS` reports `fs.ErrNotExist` for every path and an empty environment, `Platform` reports `unknown`, and `Clock` is a scripted monotonic stepper (so elapsed values are deterministic in golden files, per the spec's "only the injected clock moves the timestamps"). Every probe unit test starts from `DenyAllSeams()` and overrides exactly the one seam the case needs.

**How a test proves no real egress happens — three levels, none of them a syscall trace:**

1. **Sentinel reachability.** A test asserts that the deny-all dialer/resolver/packet seams themselves return the sentinel when called, so the default cannot silently become permissive.
2. **Closed dialed set.** `doctor_run_test.go` injects a recording dialer built on the deny-all base (it records `(host, port, protocol)` and then fails) and asserts set equality with `probe.EffectiveTargets(opts)` — the declared set, after overrides. Nothing leaves the process, and an undeclared target would fail the assertion.
3. **Static guard.** `internal/probe/guard_test.go` parses the module's `.go` sources and asserts that direct construction of real network or process primitives (`net.Dial*`, `net.Lookup*`, `tls.Dial*`, `os/exec`) appears only in `internal/probe/real.go` and `cmd/herdr-reach/main.go`. This is a static check, recorded as such: it narrows RG-7/RG-10, it does not replace the CI workflow that R11 owns.

Production seams are constructed in exactly one place (`internal/probe/real.go`, called from `main.go`); tests never call it, and — the strongest fact — the production `CommandRunner` is left **nil** in R1a, so no code path can execute a third-party binary even if a probe asked.

### 6.3 The "writes nothing" proof

`cmd/herdr-reach/main_test.go` calls the CLI entrypoint in-process (`run(args, stdio, seams)`) with `t.Setenv("HOME", t.TempDir())` and the working directory moved to a second `t.TempDir()`, digests both trees recursively before and after, and asserts they are byte-identical, the command runner is called zero times in the production shape, and the dialed set equals the declared set; `internal/doctor/guard_test.go` carries the proof levels that only need the harness. Coverage boundary, stated rather than over-credited (RG-14): a temp-`HOME` digest cannot prove the absence of writes outside `HOME`/CWD; R1a holds that by construction (no writer code path and no write-capable dependency is imported), and the residual gap stays in the risk register.

> **Dated note — 2026-09-20, PR 20 (S11).** The "zero calls" reading is the production shape's: no runner is wired there, so no call is possible. `local.sshd` asks a non-nil runner for the service state and the effective configuration, and the deny-all runner denies both, so a test run with a runner wired records two denied attempts rather than zero calls — the denial, not the count, is what the case pins, and the attempts are pinned by their own case. R1a's no-execution boundary is checked statically in `internal/probe/guard_test.go`: it reads the composite literal `ProductionSeams` returns and fails if that literal wires a `CommandRunner` (absent or explicitly nil is the absence the boundary requires), with a counter-case fixture proving the check can fail.

### 6.4 How a test-only transport proves NF-04 without touching the reasoning layer

`internal/transport/testtransport_test.go` registers a `testTunnel` that implements only `Candidate` with a fixed viability and reason, and asserts: the produced feasibility list contains it with its stated viability and reason; every `diagnosis` finding is byte-identical to the run without it; and `internal/diagnosis/agnostic_test.go` — a guard that reads the `diagnosis` package's own **non-test** sources — asserts they contain none of the strings `direct-ssh`, `reverse-ssh`, `cloudflare-tunnel`, `tailscale`, `argotunnel`, `7844`, so per-transport knowledge provably lives outside the reasoning layer. Two scoping decisions are part of the guard rather than accidents of it: it reads production sources only (the PRD §1.1 replay in `matrix_test.go` quotes the matrix's literal rows, which is acceptance evidence rather than reasoning knowledge), and it removes the registered probe names before searching (`egress.cf.7844` is exactly such a name, and it belongs to the measurement layer), with a positive control so a guard that read nothing fails instead of passing silently.

---

## 7. File plan (every path the change adds)

**Module root**

| Path | Purpose |
|---|---|
| `go.mod` | Module `github.com/Luisalt20/herdr-reach`, `go 1.25.10`, no `toolchain`, zero requires |

No `go.sum` is expected: R1a is stdlib-only. If the toolchain leaves an empty one behind, that is a toolchain artifact, not a missing deliverable.

**`cmd/herdr-reach/`**

| Path | Purpose |
|---|---|
| `main.go` | Thin entrypoint: build production seams, call `doctor.Run`, `os.Exit` |
| `flags.go` | `doctor` subcommand, `--json`, `--hub`, `--target`, `--version`, `-h` → `doctor.Options` |
| `flags_test.go` | Flag parsing, defaults, usage errors |
| `main_test.go` | Exit-code plumbing at the process boundary (in-process `run`) |

**`internal/doctor/`**

| Path | Purpose |
|---|---|
| `doctor.go` | Run harness: targets → runner → diagnosis → feasibility → report → exit code |
| `exit.go` | `ExitOK`/`ExitIncomplete`/`ExitUsage` constants (documented contract) |
| `doctor_test.go` | End-to-end in-process run over scripted seams; exit-code matrix |
| `guard_test.go` | Writes-nothing, no-exec, closed-dialed-set proof (§6.3) |

> **Dated note — 2026-09-20, PR 18 (S9).** The flag surface lives in `internal/doctor/flags.go` rather than under `cmd/herdr-reach/`: that directory is a `package main`, its `main.go` belongs to PR 19, and a `package main` with no `func main` fails `go build ./...` — which the CI's five `Cross-Build` jobs run, and those are deliberately not required checks, so the slice would have landed with five red jobs that nothing blocks. The `cmd/` directory therefore appears with the binary that justifies it, and PR 19's `main.go` calls the exported parser from here. The command line is `herdr-reach doctor [flags]`, with the subcommand owned by the flag parser so the documented shape is testable where it is defined; `--version` writes the version to standard output and `-h`/`--help` writes usage to standard error, both exiting `0`, because a version query is the answer a caller asked for rather than a run's human prose, and a help request is not a usage error. The exit-code constants ↔ document-table assertion lives in `internal/doctor/doctor_test.go`, which is why it is not in `internal/report/docs_test.go`.

**`internal/probe/`**

| Path | Purpose |
|---|---|
| `probe.go` | `Probe`, `ProbeKind`, `Verdict`, `Resolution`, `Observation`, `Result`, aggregate function |
| `reason.go` | The closed reason-code set + `AllReasonCodes()` |
| `classify.go` | The classification table (§5.1) |
| `seams.go` | The eight seam interfaces, `Seams`, `DenyAllSeams()` |
| `real.go` | Production seams — the only file constructing real network primitives |
| `targets.go` | Declared target set, overrides, `EffectiveTargets`, per-probe protocol triples |
| `runner.go` | Bounded concurrency, per-probe bound, 60 s run budget, streaming channel, cancellation, runner-side `probe_timeout` |
| `registry.go` | The ten probes, deterministic order |
| `local.go` | `local.env`, `local.sshd` (three observations) |
| `egress.go` | `egress.hub.direct`, `egress.ssh.known`, `egress.ssh.443`, `egress.cf.7844`, `egress.cf.443` (per-region observations) |
| `quic.go` | `egress.quic` (narrow UDP question, D8) |
| `tls.go` | `tls.interception`, `tls.truststore` |
| `probe_test.go` | Result invariants, aggregate order, control cases (`fail` + `indeterminate`) per probe |
| `classify_test.go` | One case per classification-table row |
| `runner_test.go` | Bounds, concurrency ceiling, hanging probe, cancellation, streaming before completion |
| `seams_test.go` | Sentinel reachability of `DenyAllSeams`; nil-runner behaviour |
| `targets_test.go` | Declared set, override replaces, unknown name ⇒ error |
| `local_test.go`, `egress_test.go`, `quic_test.go`, `tls_test.go` | Per-probe scenarios incl. the three timeout paths and the macOS/override trust-store cases |
| `guard_test.go` | Static no-real-network guard (§6.2) |

**`internal/diagnosis/`**

| Path | Purpose |
|---|---|
| `facts.go` | Observation accessors that turn `[]Result` into matchable states |
| `rules.go` | The ordered declarative rule table (§5.2) |
| `findings.go` | `Finding`, `OpenQuestion`, `Diagnosis`, conclusion texts |
| `ids.go` | `AllRuleIDs()`, derived-id construction for fact questions |
| `diagnose.go` | `Diagnose([]Result) Diagnosis` — the pure entry point |
| `rules_test.go` | One case per rule id, including every unresolved-suppression path |
| `matrix_test.go` | PRD §1.1 six-row replay as a named acceptance test + counterfactuals |
| `unresolved_test.go` | Unresolved weakens only dependent conclusions; unrelated findings unchanged |
| `agnostic_test.go` | No per-transport strings in the reasoning package (§6.4) |

**`internal/transport/`**

| Path | Purpose |
|---|---|
| `contract.go` | `Candidate`, `Transport`, `Requirement`, `Feasibility`, `RequirementKind` closed set |
| `pending.go` | Placeholder `PairingBundle`, `Step`, `Handle` + `ErrNotImplementedInThisPhase` |
| `registry.go` | The four V1 transports, deterministic order |
| `directssh.go`, `reversessh.go`, `tailscale.go`, `cloudflare.go` | The four adapters: `Feasible` + typed not-implemented plan/verify members; the HTTP/2 trade-off text lives in `cloudflare.go` |
| `pin_note.go` | The single home of the `cloudflared` pin statement (RG-1 wording) |
| `contract_test.go` | `!Viable ⟹ Reason != ""` over all four; declared ⊆ evaluated requirements; no third viability value |
| `notimplemented_test.go` | Each plan/verify member fails with the typed error naming member + owner slice, and returns no value |
| `feasibility_test.go` | Finding-set × adapter table incl. blocked-but-detected and unmeasured-transport cases |
| `cloudflare_test.go` | Hostname prerequisite as a requirement; QUIC-failed/unconfirmed/OK advice paths; PQ trade-off wording; no "applied" claim |
| `pin_test.go` | Pin note names #1673, 2026.6.0 only, later releases unknown, 2026.5.1 checksums; one home |
| `testtransport_test.go` | NF-04 test-only transport (§6.4) |
| `registry_test.go` | Exactly four transports, stable order, nothing else registered |

**`internal/report/`**

| Path | Purpose |
|---|---|
| `report.go` | Payload DTO types with explicit JSON field names |
| `map.go` | The single mapping function from internal structures |
| `human.go` | Human projection (probe rows, findings, coverage gaps, transports) to stderr |
| `map_test.go` | Golden payload with a fixed clock; exact key sets per level; determinism |
| `human_test.go` | Unresolved rendered as unresolved; incomplete named; blocked target preserved |
| `docs_test.go` | Doc table ↔ `AllReasonCodes()` / `AllRuleIDs()` / exit-code constants |

> **Dated note — 2026-09-20, PR 17 (S8).** The human projection renders the **already built payload** rather than the internal structures, so the two projections of one run read one source and cannot drift; its section order and column geometry are pinned by the suite, because a layout change should be a deliberate act. The document's two vocabularies are machine-checked by `internal/report/docs_test.go`, which parses them by their headings (`## Reason codes`, `## Rule ids`) and asserts set equality with `probe.AllReasonCodes()` and `diagnosis.AllRuleIDs()` in both directions; renaming a heading therefore fails the suite rather than silently skipping the check. The exit-code constants ↔ table assertion lives in `internal/doctor/doctor_test.go` (PR 18), because `internal/doctor` imports `internal/report` and the reverse import would invert the dependency direction. The document records the `cloudflared` pin's single home instead of restating it, which is what keeps PR 15's one-home guard green while `docs/**` remains part of its scan.

**`internal/version/`**

| Path | Purpose |
|---|---|
| `version.go` | `Version`, `Commit`, `Date` variables + `String()` for the payload |
| `version_test.go` | Defaults when unset (ldflags-injected later) |

**Docs**

| Path | Purpose |
|---|---|
| `docs/diagnosis-report.md` | Exit codes, stream split, payload shape + `schema_version`, reason-code table, rule-id scheme, "adding one is a contract change" |

---

## 8. Test plan (strict TDD)

Every task carries its test first (`strict_tdd: true`). Gates: `go test ./...`, `go vet ./...`, `gofmt -l .` — no linter exists and none is invented.

| Area | Test shape | Proves |
|---|---|---|
| Probe registry | enumeration test | exactly ten probes, four kinds, stable order |
| Result invariants | table + loop assertions | verdict/resolution consistency; aggregate order never promotes unresolved to pass |
| Per-probe control cases (§8.4 PRD) | table-driven per probe | each probe can return `fail`, and each can return `indeterminate`; the macOS trust-store probe satisfies pass/fail controls on Linux plus the macOS unresolved case |
| Classification table | one case per row (§5.1) | wording drift does not move the code; identical codes across OSes |
| Timeout coherence | three side-by-side cases | probe budget ⇒ `fail`/`budget_expired`; hanging probe ⇒ `indeterminate`/`probe_timeout` (runner-classified); UDP silence ⇒ `indeterminate`/`udp_silence` |
| Not-measured vs unresolved | paired cases | absent hub ⇒ not_measured (`input_missing_hub`); refused hub dial ⇒ measured fail; nil runner vs deny-all runner |
| Runner | bounds, ceiling, cancellation, streaming | concurrency ceiling honoured; full run inside injected millisecond budget; first result available before the slowest probe ends; cancellation stops in-flight dials with a cancelled outcome (never a pass) |
| Target set | declared vs recording dialer | dialed triples == effective declared set; override replaces; no extra outbound attempt |
| Matrix replay | literal six rows of PRD §1.1 + counterfactuals | `SSH_DEST_BLOCKED_BY_PUBLIC_SSH` fires and names `203.0.113.10:22`; both-pass ⇒ no block conclusion; public fail/not-measured ⇒ weaker rule id |
| Unresolved propagation | per-question cases | a confident rule cannot fire on unresolved input; the weaker rule fires and names the probe; unrelated findings unchanged |
| Reason/rule closure | enumeration + docs parse | every probe × resolution has a rule id; code set == documented set |
| NF-03 at three layers | engine, DTO, human | `indeterminate` is never encoded or rendered as `pass` |
| Exit codes | matrix over scripted runs | `0` completed/negative/refused node; `1` incomplete; `2` usage; `--json` stdout is exactly one document and no human text |
| Determinism | repeated runs + clock-only change | byte-identical payloads; only `generated_at` and elapsed move with the clock |
| Human projection | golden with pinned geometry | coverage gaps and unresolved rows visible; blocked target string present in both projections |
| Writes nothing | in-process run with temp HOME/CWD | byte-identical trees; command runner called zero times; dialed set closed |
| No real egress | sentinel + recording dialer + static guard | §6.2, three levels |
| NF-04 | test-only transport + `diagnosis` source guard | adding a transport changes no reasoning file |
| Pin note / WSL2 text | string assertions in both projections | RG-1 wording; RG-4 exclusions; no "applied"/"enforced" claim |

`-race` is run by hand during apply (the runner is concurrent; a race there is a correctness bug), even though the configured command is `go test ./...`.

---

## 9. Rollout, verification evidence, rollback

**Work-unit order** (the tasks phase turns these into reviewable units; it also owns the chained-split decision):

1. Module bootstrap — `go.mod` at the ratified path, `internal/version`, first honest `go test ./...` (RG-11: the canonical command is unrunnable until this unit lands).
2. `internal/probe` value types, reason codes, classification table, seams + `DenyAllSeams`, targets.
3. Runner (concurrency, bounds, streaming, cancellation) + registry.
4. The ten probes one at a time, each with its control tests.
5. `internal/diagnosis` fact accessors + rule table + matrix replay.
6. `internal/transport` contract, placeholders, registry, four adapters, not-implemented test.
7. `internal/report` DTO + mapping + human projection + docs + `docs/diagnosis-report.md`.
8. `internal/doctor` harness + CLI flags + guard tests.

**Verification evidence beyond unit tests** (things that genuinely cannot be a unit test, recorded in the verify phase): the PRD §1.1 six-row matrix reproduced by hand on the motivating network and compared with the replayed verdict; the D8 raw-UDP comparison against `cloudflared`'s QUIC log lines; the macOS trust-store behaviour on a real Mac (unresolved + limitation text).

**Rollback.** R1a is additive and mutating-nothing, so rollback is a revert: new files disappear; no machine state, service, key, token or config entry was ever created. Reverting only the engine commits leaves `doctor --json` unavailable and keeps the probe layer. No migration, no data, no compatibility window.

---

## 10. Risks carried and review-budget position

| # | Carried risk | Handling in this design |
|---|---|---|
| RG-1 | `cloudflared` pin is report-only | Single-home wording in `pin_note.go` + string tests; no version range, no "fixed" |
| RG-2/RG-8 | Wrong classification / `indeterminate` read as `pass` | §5.1 classification table, §5.2 structural suppression, NF-03 at three layers |
| RG-3 | macOS trust store cannot be resolved | `indeterminate` + limitation text in the output; documented live-run consequence: macOS exits `1` |
| RG-4 | WSL2 undocumented rules | Text limited to documented semantics; absence assertions |
| RG-5 | QUIC false negative | Narrow declared question (D8); silence ⇒ unresolved; advice never claims enforcement |
| RG-6 | HTTP/2 read as equivalent | Trade-off wording requirement + tests; no "applied" claim |
| RG-7/RG-10 | No CI workflow; tests could dial the real network | Deny-all default, recording dialer, static guard — and the gap is stated, not assumed closed |
| RG-9 | Interface stubs | Full contract + typed error + placeholder file + test asserting no value is returned |
| RG-11 | Bootstrap friction | Work unit 1 is exactly the module |
| RG-12 | Toolchain ambiguity | `go 1.25.10`, no `toolchain`, audience widening recorded |
| RG-13 | Hub address | `--hub` flag, `null` target when absent, file format deferred to R2 |
| RG-14 | Guard-test reach | Coverage boundary documented; residual gap stays open |

**Review budget.** R1a is expected to exceed 400 changed lines; this design does not claim an exception and does not select a chain strategy. The tasks phase estimates the work units above against the real plan and, under `ask-on-risk`, pauses to ask for the chained split when the overrun is confirmed. Candidate axes are the eight work units in §9; no chain is pre-selected here.

**Open for the tasks phase (not decisions, sequencing and size only):** how the eight units map to commits/PRs; whether the `docs/diagnosis-report.md` contract ships with the report unit or separately; and where the verify-phase hand-run evidence is recorded.