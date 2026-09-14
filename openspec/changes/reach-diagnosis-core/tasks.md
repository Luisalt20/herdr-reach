# Tasks — `reach-diagnosis-core` (R1a: headless diagnosis core)

**Change**: `reach-diagnosis-core` — slice R1a of the approved roadmap R1–R12
**Phase**: tasks (third pass: consolidation of the decided chain into cohesive PRs) · **Date**: 2026-09-14
**Artifact store**: both — this file, mirrored in Engram as two observations: `sdd/reach-diagnosis-core/tasks` (part 1) and `sdd/reach-diagnosis-core/tasks/part2` (part 2)
**Inputs read before writing**: this change's `proposal.md`, the three specs (`specs/diagnosis`, `specs/transport-feasibility`, `specs/cli-report`), `design.md` (authoritative), `openspec/config.yaml`, the previous tasks pass (38 units), and the parent's consolidation instruction (≈15–20 cohesive PRs; the per-PR ceiling was ≈600 lines then and is 1000 lines since 2026-09-14).
**Skill resolution**: `paths-injected` — `/home/luisalt20/.config/opencode/skills/work-unit-commits/SKILL.md` and `/home/luisalt20/.config/opencode/skills/chained-pr/SKILL.md` read before writing.
**Preflight consumed**: execution mode `auto`, artifact store `hybrid`, delivery strategy `ask-on-risk`, review budget 400 changed lines, `size:exception` never inferred. The human has chosen chained delivery, named the chain strategy `stacked-to-main`, and approved a per-PR cohesion ceiling (600 lines, revised to 1000 on 2026-09-14 after three units measured 2–3× their estimates). This pass executes those decisions and decides nothing new.
**No implementation**: this artifact adds no Go file, no `go.mod`, no scaffolding. The only file written is this one (plus its memory mirror).

---

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ≈7,600–11,800 (`additions + deletions`: authored code + strict-TDD tests + `docs/diagnosis-report.md`); point estimate ≈9,600 |
| 400-line budget risk | Medium — every PR's point estimate is below the chain ceiling (600 lines, revised to 1000 on 2026-09-14); the widest ranges straddle it and each carries a named second-split boundary |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → … → PR 20 (20 stacked PRs; the 38 work units WU1–WU38 are consolidated into 20 cohesive groups) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

```text
Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium
```

**Budget ceiling, stated as a decision and not an oversight.** The per-PR ceiling for this chain is **1000 changed lines** as of 2026-09-14: a deliberate, user-approved exception to the canonical 400-line threshold, revised upward from 600 because the first three units measured 347, 1400 and 1945 counted lines against estimates of 80, 595 and 545, and their splits still landed at 565–1176 lines. The 600 figure was not reachable in this codebase without fragmenting reviews into pieces smaller than a coherent unit. No PR in the map below exceeds it at its point estimate, so no `size:exception` is required or assumed. The budget constrains slicing only: no comment, blank line, doc, control case, triangulation case or test is deleted or compressed to reach a number.

**Reading the focused test commands in the map.** Those `-run` filters are written with markdown-escaped pipes (`\|`) so the table renders. When copying one, use the rendered form with a bare `|`: Go's `-run` takes an RE2 regexp, where `\|` matches a literal pipe and selects zero tests, which would report a green run that executed nothing.

**Why `Decision needed before apply: No`.** The human already made the three decisions this pass would otherwise defer: chained delivery, the chain strategy `stacked-to-main`, and the per-PR cohesion ceiling for this chain (600 lines then, 1000 since 2026-09-14). What remains is execution under those decisions, plus the per-PR budget check stated in the map below.

**Chain strategy, concretely.** PR 1 targets the main branch; PR 2 targets PR 1's branch; PR 3 targets PR 2's branch; and so on through PR 20. Merges happen in order, so each PR's diff shows only its own group. Branch naming: `reach-diagnosis-core/pr-<n>-<slug>`. If a diff shows another group's work (a polluted diff), retarget or rebase until only the current group appears — treat it as a base bug, not a review task.

**How the budget was computed per PR.** Each row's estimate covers authored `additions + deletions` for its own code, its own tests and its own docs, and is the sum of the member units' points (ranges from the member units' honest lower and upper bounds). Generated or vendored content does not exist in this change.

**Estimate revision, stated honestly.** The first pass printed 4,200–6,100 for the whole slice; the second pass revised it to ≈7,600–11,800 (point ≈9,600) and this pass keeps that band unchanged. Nothing is shrunk to return it to the earlier width. The reason it moved is unchanged and real: two control cases per probe (can it `fail`, can it be `indeterminate`, PRD §8.4), one case per rule id (~40 rule ids), one case per classification row (27 rows), the three side-by-side timeout paths, and the determinism/coverage/NF-03 assertions at three layers. Consolidation changes grouping only.

**Consolidation record (38 units → 20 PRs).** The previous pass sliced the same scope into 38 stacked PRs. That fragmentation cost — 38 branches, 38 reviews and 38 merges in strict order before anything is usable — was rejected by the user in favour of ≈15–20 cohesive groups under a per-PR ceiling (600 lines at the time, 1000 since 2026-09-14). The mapping is exhaustive and non-overlapping:

| PR | Member units (previous pass) |
|---|---|
| 1 | WU1 |
| 2 | WU2, WU3 |
| 3 | WU4, WU5 |
| 4 | WU6, WU7 |
| 5 | WU8, WU9 |
| 6 | WU10, WU11 |
| 7 | WU12, WU13 |
| 8 | WU14, WU15 |
| 9 | WU16, WU17b |
| 10 | WU17a, WU18 |
| 11 | WU19, WU20 |
| 12 | WU21, WU22 |
| 13 | WU23, WU24 |
| 14 | WU25, WU26 |
| 15 | WU27, WU28 |
| 16 | WU29, WU30 |
| 17 | WU31, WU32 |
| 18 | WU33, WU34 |
| 19 | WU35, WU36 |
| 20 | WU37, WU38 |

**Two groups are budget-forced rather than theme-pure, and are disclosed instead of hidden.** No contiguous three-unit group anywhere in the sequence fits under 600 lines — the smallest is 645 lines (`egress.quic` + `tls.interception` + `tls.truststore`) and the second smallest 650 — so a 20-PR chain cannot be assembled from theme-pure pairs alone. Exactly two groups therefore cross the previous per-question grouping:

- **PR 6** merges WU10 (`local.sshd`) with WU11 (`egress.hub.direct`). Second-split boundary if the group is disputed or lands above 600: `local.sshd` first, then `egress.hub.direct`.
- **PR 9** merges WU16 (`tls.truststore`) with WU17b (the `facts.go` accessors). It is framed as the measurement→reasoning boundary: the last probe and the accessors that turn probe results into matchable reasoning states. Second-split boundary: `tls.truststore` first, then the accessors.

Every other group is organised around a single reviewable theme: the measurement vocabulary with its closed reason-code set and the classification table that consumes it (PR 2); the injected seams with the declared target set (PR 3); the runner with its budget, cancellation and three timeout paths (PR 4); the platform question, refusal and documented WSL2 limits (PR 5); probes grouped by the question they answer rather than one probe per PR (PR 6–PR 8); the reason-code/rule-id vocabularies with the rule mechanism and the rule groups that share it (PR 9–PR 12); the transport contract with its adapters (PR 13–PR 14); the recommendation wording with the feasibility evidence table (PR 15); the payload with its coverage properties (PR 16); the human projection with its contract document (PR 17); the command surface with its guards and the honesty suite (PR 18–PR 20).

**Coverage is unchanged.** All 164 checkbox tasks of the 38 units appear in this file exactly once, each in one PR (165 total: the re-slice below adds one `REFACTOR + GATE` line for its second half). No control case, rule-id case, classification row, guard test, exit-code row or documentation deliverable was dropped. WU17 was the one unit re-sliced (WU17a = `ids.go` + the rule-id coverage case → PR 10; WU17b = `facts.go` + the accessor cases → PR 9), keeping all of its tasks and splitting its range into the same 250–360 lines. The only file-plan movement is that `internal/diagnosis/facts.go` lands in PR 9 and `internal/diagnosis/ids.go` in PR 10 — one package, two PRs, no file added or dropped against design §7.

**What a reviewer checks for every PR (the three required statements)** are the `Unit verification`, `Isolation check` and `Rollback boundary` columns of the work-unit map, one row per PR.

---

## Sequencing notes

1. **`go.mod` first (PR 1).** RG-11: `go test ./...` fails today with "directory prefix . does not contain main module". The canonical command only becomes runnable once `go.mod` and one package exist, so PR 1 is exactly that and its acceptance evidence is the first honest `go test ./...` run. `go.sum` is expected to be **absent** (stdlib-only); a toolchain-created empty file is an artifact, not a missing deliverable.
2. **The registry lands with `local.env` (PR 5), not with the runner.** Design §7's file plan is unchanged; the registry cannot compile before a probe constructor exists. PR 5 opens with the enumeration test; later probe PRs extend the registry by one or two entries and add no enumerable surprise.
3. **`internal/doctor/exit.go` lands with PR 18.** The exit-code constant ↔ documentation assertion therefore lives in `internal/doctor/doctor_test.go` (PR 18), while `internal/report/docs_test.go` (PR 17) asserts the reason-code set and the rule-id set. This avoids an import cycle (`doctor` imports `report`).
4. **Reason codes ship with the measurement types (PR 2), not with the classification table.** The closed set is a vocabulary; the table is behaviour. Splitting them keeps the classification group free to carry 27 row cases plus the wording-drift case.
5. **The reasoning layer lands in four ordered steps (PR 9 → PR 12).** Accessors (`facts.go`, PR 9) → rule-id scheme (`ids.go`) with the mechanism, findings and derived fact rules (PR 10) → the destination-block group with the §1.1 replay and the Cloudflare groups (PR 11) → the remaining rule groups with the suppression and agnosticism suites (PR 12). Each step compiles on the previous step's exported types only.
6. **Both projections must exist before the string-level honesty assertions.** RG-4 (no child-of-init rule, no `-1` sentinel), RG-6 (no "applied"/"enforced" claim) and the blocked-target survival assertions are placed in PR 20, where `report.WriteHuman` and `report.WriteJSON` are both reachable from one in-process run.
7. **No file is added or dropped against design §7.** The 20 groups are sequencing and size only; one path moves between PRs (`facts.go` in PR 9, `ids.go` in PR 10) and no path is invented.

---

## Work-unit map

| PR | Deliverable (files) | Reqs | Est. lines | Budget risk | Unit verification | Isolation check | Rollback boundary |
|---|---|---|---|---|---|---|---|
| 1 | `go.mod`; `internal/version/version.go` + test | RG-11, RG-12, R-HR-NF-06 ctx | 60–100 (≈80) | Low | `go test ./...` (first honest run), `go vet ./...`, `gofmt -l .` | `go.mod` fields + recorded `go test` output; no README/PRD edit rode along | Delete `go.mod` + `internal/version/`; repo returns to greenfield (accepted, RG-11) |
| 2 | `internal/probe/probe.go`, `reason.go`, `classify.go`, `probe_test.go`, `classify_test.go` | PRD §5.1, R-HR-07, R-HR-03, R-HR-NF-02, RG-8, design §3.1/§3.5/§5.1 | 470–720 (≈595) | Medium | `go test ./internal/probe/ -run 'TestResultInvariants\|TestClassificationTable'` + full suite | 27 constants countable against design §3.5 and all 27 rows comparable one-by-one with design §5.1 without reading any probe | Delete the five files; module returns to PR 1 state |
| 3 | `internal/probe/seams.go`, `seams_test.go`, `targets.go`, `targets_test.go` | RG-7, RG-10, R-HR-NF-10, D10, RG-13, design §6.1–6.2 | 430–660 (≈545) | Medium | `go test ./internal/probe/ -run 'TestDenyAllSeams\|TestEffectiveTargets'`; `-race` | Every seam's sentinel is asserted from the test side, no package-level seam variable exists, and the declared set lives in exactly one file | Delete the four files; PR 2 stays green and no consumer exists yet |
| 4 | `internal/probe/runner.go`, `runner_test.go` | PRD §4.4, R-HR-NF-02, R-HR-NF-09, D9, obligation 3, RG-8 | 400–610 (≈505) | Medium | `go test ./internal/probe/ -run 'TestRunner\|TestHanging\|TestRunBudget\|TestCancel\|TestTimeoutPaths'`; hand `-race` | Fixture probes live in `runner_test.go`, so the ceiling, the bound and streaming review without the ten real probes; the three timeout paths are one table | Revert this PR only; callers supply their own probes and PR 3 stays green |
| 5 | `internal/probe/registry.go`, `local.go`, `local_test.go`, `probe_test.go` (enumeration case) | PRD §5.1, R-HR-29, R-HR-30, DEV-2, RG-4 | 380–570 (≈475) | Low | `go test ./internal/probe/ -run 'TestRegistry\|TestLocalEnv'` | Count/order/kinds are declared in `registry.go` and asserted in one test; platform cases script every seam and the refusal names its own requirement | Delete `registry.go` and both the `local.env` and refusal parts of `local.go`; the runner stays usable |
| 6 | `local.go`, `local_test.go` (additions); `egress.go`, `egress_test.go` | R-HR-18, obligations 1–2, R-HR-02, R-HR-03, RG-13 | 410–630 (≈520) | Medium | `go test ./internal/probe/ -run 'TestLocalSshd\|TestEgressHub'` | Each merged half names its own requirement in its own test file, and both not-measured cases (nil runner vs deny-all runner; absent hub) sit in those files | Revert this PR; `local.env` and the registry stay green; split boundary named in the consolidation record |
| 7 | `egress.go`, `egress_test.go` (additions) | PRD §5.1, R-HR-03, R-HR-07, PRD §8.4, D10, RG-8 | 400–600 (≈500) | Medium | `go test ./internal/probe/ -run 'TestEgressSsh\|TestEgressCF'` | Each probe's declared question plus its pass/fail/indeterminate controls are stated in its own test block; the region split is visible per observation | Revert this PR; PR 6's hub probe stays green |
| 8 | `quic.go`, `quic_test.go`, `tls.go`, `tls_test.go` | D8, RG-5, R-HR-NF-02, R-HR-04 | 320–490 (≈405) | Low | `go test ./internal/probe/ -run 'TestEgressQuic\|TestTLSInterception'` | The four D8 outcomes, the "silence is never a block" assertion and the no-unverified-retry assertion are separate, readable tables | Delete the four files; PR 7 stays green |
| 9 | `tls.go`, `tls_test.go` (additions); `internal/diagnosis/facts.go`, `rules_test.go` (accessor cases) | R-HR-04, RG-3, PRD §5.1, design §3.6, DEV-2 | 300–440 (≈365) | Low | `go test ./internal/probe/ -run TestTLSTruststore`; `go test ./internal/diagnosis/ -run TestFacts` | The Linux controls, macOS limitation text and override case are one file; the accessors are testable with hand-written results, no probe and no CLI | Delete `facts.go` + the accessor cases and revert the trust-store additions; PR 8 stays green |
| 10 | `internal/diagnosis/ids.go`, `rules.go`, `findings.go`, `diagnose.go`, `rules_test.go` (additions) | PRD §5.1, design §3.6/§5.2, R-HR-NF-03, DEV-2 | 400–610 (≈505) | Medium | `go test ./internal/diagnosis/ -run 'TestRuleIDsCoverEveryProbeState\|TestRulesOneCasePerID\|TestNoFallThrough'` | The id scheme and the mechanism (first match, no default, open question) review without any specific transport or probe | Revert this PR; PR 9's accessors stay |
| 11 | `rules.go` (additions), `matrix_test.go`, `rules_test.go` (additions) | R-HR-03, PRD §13, §1.1, RG-2, design §5.2 | 410–610 (≈510) | Medium | `go test ./internal/diagnosis/ -run 'TestMatrixReplay\|TestCloudflareRules'` | The six literal §1.1 rows with their counterfactuals, and the four edge states plus four HTTP/2 advice states, are two readable tables | Revert this PR; the mechanism and the other rule groups stay |
| 12 | `rules.go`, `rules_test.go` (additions), `unresolved_test.go`, `agnostic_test.go` | R-HR-04, R-HR-18, R-HR-29/30, RG-3, R-HR-NF-03, R-HR-NF-04, obligations 1–2 | 440–650 (≈545) | Medium | `go test ./internal/diagnosis/` | Every id in the group has one case; suppression is structural, so the test reads the table's match sets rather than a rendering path | Revert this PR; PR 11's groups stay |
| 13 | `internal/transport/contract.go`, `pending.go`, `directssh.go`, `reversessh.go`, `registry.go`, `contract_test.go` | PRD §5.2, §14.5, RG-9, D6–D7, R-HR-06 | 410–610 (≈510) | Medium | `go test ./internal/transport/ -run 'TestContractTypes\|TestContractInvariants\|TestRegistry'` | The closed kind set, strict boolean viability and the typed error are checkable without any adapter; the four invariants run as a loop over the registry | Delete the six files; PR 12 stays green and nothing imports them yet |
| 14 | `tailscale.go`, `notimplemented_test.go`, `cloudflare.go`, `registry_test.go` | RG-9, R-HR-06, PRD §5.2, §14.5 | 380–570 (≈475) | Low | `go test ./internal/transport/ -run 'TestNotImplemented\|TestRegistry\|TestCloudflareRequirements'` | Every registered adapter's three members are proven to fail loudly and return no value; exactly four transports and the unsatisfied hostname row are one file each | Delete the four files; the registry returns to two adapters |
| 15 | `cloudflare.go` (advice block), `pin_note.go`, `pin_test.go`, `feasibility_test.go`, `testtransport_test.go` | R-HR-05, RG-6, R-HR-16, RG-1, R-HR-06, R-HR-NF-04, design §6.4 | 420–620 (≈520) | Medium | `go test ./internal/transport/` | `pin_note.go` is the only file holding the pin wording, so one grep proves the single home; the finding-set × adapter table and the test-only transport read `diagnosis` output only | Delete `pin_note.go` + `pin_test.go` + the two test files; revert the advice block only |
| 16 | `internal/report/report.go`, `map.go`, `map_test.go` | R-HR-NF-06, R-HR-NF-02, R-HR-NF-03, cli-report determinism, design §3.3/§5.3 | 460–680 (≈570) | Medium | `go test ./internal/report/ -run 'TestPayloadKeySets\|TestCoverage\|TestDeterminism\|TestNoPassPromotion'` | The exact key set per payload level, coverage/completeness and determinism are asserted with a fixed clock; no other producer exists | Delete the three files; nothing imports `report` yet |
| 17 | `internal/report/human.go`, `human_test.go`, `docs/diagnosis-report.md`, `internal/report/docs_test.go` | R-HR-07, R-HR-NF-02, R-HR-NF-03, R-HR-NF-06, design §5.4 | 390–580 (≈485) | Low | `go test ./internal/report/ -run 'TestHumanProjection\|TestDocsMatchCodeSets'` | Golden human output with the unresolved row, coverage gap and blocked target in one file; the doc tables are parsed and compared with `AllReasonCodes()`/`AllRuleIDs()`, so drift fails the build | Delete the doc, the human projection and their tests; the JSON projection stays |
| 18 | `internal/doctor/exit.go`, `doctor.go`, `doctor_test.go`; `cmd/herdr-reach/flags.go`, `flags_test.go` | R-HR-07, R-HR-NF-02, R-HR-30, RG-13, cli-report hub, design §3.4 | 430–650 (≈540) | Medium | `go test ./internal/doctor/ -run 'TestExitMatrix\|TestDocMatchesExitCodes'`; `go test ./cmd/herdr-reach/ -run TestFlags` | The whole exit matrix runs over scripted seams with no CLI binary and no real network; every parse rule and usage error is one table that starts no measurement | Delete the five files; the probe and report layers stay |
| 19 | `cmd/herdr-reach/main.go`, `main_test.go`, `internal/probe/real.go`, `internal/probe/guard_test.go` | R-HR-07, cli-report §1, design §4/§6.2 level 3, RG-7, RG-10 | 360–530 (≈445) | Low | `go test ./cmd/herdr-reach/ -run TestStreamSplit`; `go test ./internal/probe/ -run TestStaticNoRealNetwork`; `go test ./...` | The stream split is asserted at the in-process boundary; the guard names the two allowed files and prints the offending path on failure | Delete the four files; the CLI disappears, the libraries stay |
| 20 | `internal/doctor/guard_test.go`, `internal/doctor/doctor_test.go` (e2e additions) | R-HR-02, R-HR-24, R-HR-27, RG-14, PRD §13, §1.1, RG-2, RG-4, RG-6, R-HR-NF-09 | 390–580 (≈485) | Medium | `go test ./...`, `go vet ./...`, `gofmt -l .`, hand `go test -race ./...` | Two temp-tree digests, the zero-exec counter and the dialed-set comparison are in one file; one complete run yields both projections, so the honesty string assertions are checkable together | Delete the two files; the proof disappears, the behaviour does not change |

**Chain shape** (each PR body repeats this with 📍 on itself, per the chained-pr skill):

```
main ← PR 1        bootstrap: module + version
     ← PR 2 … PR 5  measurement vocabulary, classification, seams, targets, runner, local.env
     ← PR 6 … PR 8  the probes, grouped by the question they answer
     ← PR 9 … PR 12 reasoning: accessors, rule ids, mechanism, rule groups, suppression
     ← PR 13 … PR 15 transport feasibility: contract, four adapters, recommendation wording
     ← PR 16 … PR 17 projections: JSON payload, human report, contract document
     ← PR 18 … PR 20 the command surface, the safety proofs and the honesty suite
```

---

## Work units

### PR 1 — WU1 · Module bootstrap and `internal/version`

`Files`: `go.mod`, `internal/version/version.go`, `internal/version/version_test.go`. `Depends on`: nothing (main).

- [x] **BOOTSTRAP (enabling artifact — no failing test can precede it)** — create `go.mod` with `module github.com/Luisalt20/herdr-reach`, `go 1.25.10`, no `toolchain` line and zero requires; do not run `go mod tidy` against the empty module. <!-- sdd-owner: implementation -->
- [x] **RED** — write `internal/version/version_test.go` asserting the unset defaults (`Version` = `"0.0.0-dev"`, `String()` returns the documented `name version` shape) and run `go test ./...`; the failure must be a compile/assert failure inside `internal/version`, proving the package exists and the test can fail for the right reason. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `internal/version/version.go` with `Version`, `Commit`, `Date` and `String()`; run `go test ./...` and record the first honest pass. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add the ldflags-injection shape (non-default `Commit`/`Date`) and a dev-build case where an unset `Commit` never renders a fabricated commit. <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — tidy comments and package doc; run `go test ./...`, `go vet ./...`, `gofmt -l .` and record the exact commands, exit status and the `go.sum` situation (absent, or toolchain-created and empty). <!-- sdd-owner: implementation -->
- [x] Record in the unit evidence that `go 1.25.10` with no `toolchain` line is the deliberate audience decision (D4, RG-12) and that `README.md` is out of this change's scope. <!-- sdd-owner: implementation -->

### PR 2 — WU2 + WU3 · Measurement types, the closed reason-code set, and the classification table

`Files`: `internal/probe/probe.go`, `internal/probe/reason.go`, `internal/probe/classify.go`, `internal/probe/probe_test.go`, `internal/probe/classify_test.go`. `Depends on`: PR 1.
Second-split boundary if this group lands above 600: the vocabulary and the closed reason-code set (WU2) first, then the classification table (WU3), since the table consumes the vocabulary and nothing consumes the table yet.

- [x] **RED** — write `internal/probe/probe_test.go` with `TestResultInvariants`: `Verdict ∈ {pass,fail} ⟺ Resolution == measured`; `Resolution ∈ {unresolved, not_measured} ⇒ Verdict == indeterminate`; `TestAggregateOrder` asserting `fail > indeterminate > pass`. Prove the failure first. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `probe.go`: `Probe`, `ProbeKind`, `Verdict`, `Resolution`, `Observation`, `Result` and `Aggregate([]Observation) (Verdict, ReasonCode)`; run the invariant test to green. <!-- sdd-owner: implementation -->
- [x] **RED** — add the reason-code case to `probe_test.go`: the set is closed, unique, in the documented declaration order, exactly the 28 codes of design §3.5, and `AllReasonCodes()` returns that order. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `reason.go` with the 28 constants plus `AllReasonCodes()`. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — an unresolved observation is never promoted to `pass` and a not-measured one is never turned into `fail`, even when a sibling observation is `pass` or `fail`; `Aggregate` returns the worst observation's reason. <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — one owner per concept (codes only in `reason.go`); run `gofmt -l .`, `go vet ./...`, `go test ./...`. <!-- sdd-owner: implementation -->
- [x] **RED** — write `classify_test.go` with one case per design §5.1 row, asserting `(Resolution, ReasonCode)` and that all 28 reason codes are reachable; include `TestWordingDrift`, where two OS wordings of one failure yield identical codes and differing verbatim details. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `classify.go` as the ordered table keyed by observation plus the probe's declared purpose; every probe classifies only through it (no inline reason selection). <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add the control case: an unclassifiable observation yields `internal_error`/`platform_unknown` and never `pass`; and one raw observable classified through two probes with the same declared purpose cannot produce two codes. <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/` and the full suite. <!-- sdd-owner: implementation -->

### PR 3 — WU4 + WU5 · Seams, the deny-all test default, and the declared target set

`Files`: `internal/probe/seams.go`, `internal/probe/seams_test.go`, `internal/probe/targets.go`, `internal/probe/targets_test.go`. `Depends on`: PR 2.
Second-split boundary if this group lands above 600: the seams and `DenyAllSeams` (WU4) first, then the declared target set and overrides (WU5).

- [x] **RED** — write `seams_test.go` asserting `DenyAllSeams()` returns the sentinel error for dial, lookup, TLS verification and packet operations; the command runner denies everything; `FS` reports not-exist and an empty environment; `Platform` reports `unknown`; `Clock` is a deterministic stepper. Prove the failure first. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `seams.go`: the eight seam interfaces (`Dialer`, `Resolver`, `TLSVerifier`, `PacketDialer`/`PacketConn`, `CommandRunner`, `Clock`, `FS`, `Platform`), the `Seams` struct passed explicitly (no package-level seam variables) and `DenyAllSeams()`. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add the nil-seam cases: a nil `CommandRunner` is treated by consumers as "capability excluded" while an explicit deny-all runner yields "denied"; both must be distinguishable and neither may yield `pass` (design §5.1 obligation 2). <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — assert by inspection test that no seam is a package-level variable; run `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/`. <!-- sdd-owner: implementation -->
- [x] **RED** — write `targets_test.go` asserting: the declared set lives only in `targets.go`; ten probes are declared with their `(host, port, protocol)` triples; an override **replaces** that probe's declared targets and never appends; an unknown probe name, an empty host or an unparsable address is an error. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `targets.go`: the declared target set, per-probe protocol triples, `EffectiveTargets` and the override parser. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add the closed-set case: after an override the effective declared set is exactly what `EffectiveTargets` returns, and every declared triple is reachable from the declaration without reading probe code (R-HR-NF-10). <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/`. <!-- sdd-owner: implementation -->

### PR 4 — WU6 + WU7 · Runner: bounds, streaming, run budget, cancellation and the three timeout paths

`Files`: `internal/probe/runner.go`, `internal/probe/runner_test.go`. `Depends on`: PR 3.
Second-split boundary if this group lands above 600: the concurrency ceiling, hanging-probe handling and streaming (WU6) first, then the run budget, cancellation and the three-timeout-paths table (WU7).

- [x] **RED** — write `runner_test.go` with a counting seam proving the concurrency ceiling: never more than `Options.Concurrency` probes in flight, the default is 4, and a test-only elevated value is honoured. Prove the failure first. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement bounded concurrency plus streaming in `runner.go` with `Options{Concurrency, ProbeTimeout, RunBudget, Clock}` settable at construction. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add the hanging-probe case: a fixture probe that ignores context cancellation returns `(unresolved, probe_timeout)` produced by the runner, the run still finishes inside its injected budget, and every other fixture probe has a result (PRD §4.4, R-HR-NF-02). <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add streaming: the first result is delivered before the slowest probe finishes, so the runner does not buffer the suite. <!-- sdd-owner: implementation -->
- [x] **REFACTOR + RACE** — extract the deadline/abandon logic so one implementation serves both bound paths; record a hand `go test -race ./internal/probe/` run. <!-- sdd-owner: implementation -->
- [x] **RED** — add the bounded-run case with millisecond values: a seam uniformly slower than every per-probe bound still yields a bounded result for every probe and the run ends inside the injected run budget (R-HR-NF-09). <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement the run budget and cancellation propagation in `runner.go`, producing `run_budget_exceeded` and `run_cancelled` at the runner layer only. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add cancellation: cancelling mid-flight returns promptly, each affected probe reports `(unresolved, run_cancelled)`, and no cancelled probe is ever reported as `pass`. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add `TestTimeoutPathsAreDistinct`: the probe's own budget expiring ⇒ `(measured, budget_expired)`; a probe ignoring its budget ⇒ `(unresolved, probe_timeout)` from the runner; UDP silence ⇒ `(unresolved, udp_silence)`. Assert three distinct `(resolution, reason)` pairs and no fourth collapsing pair (RG-8). <!-- sdd-owner: implementation -->
- [x] **REFACTOR + RACE** — `gofmt -l .`, `go vet ./...`, `go test ./...` and a hand `go test -race ./...`. <!-- sdd-owner: implementation -->

### PR 5 — WU8 + WU9 · `local.env`: registry, platform classification, native-Windows refusal and WSL2 limits

`Files`: `internal/probe/registry.go`, `internal/probe/local.go`, `internal/probe/local_test.go`, `internal/probe/probe_test.go` (enumeration case). `Depends on`: PR 4.
Second-split boundary if this group lands above 600: the registry plus the four platform classifications (WU8) first, then the refusal, the unknown-platform degrade and the documented WSL2 limits (WU9).

- [x] **RED** — add the registry enumeration case to `probe_test.go`: exactly the ten declared probes, each with one of the four kinds, enumeration order stable across runs, and no eleventh entry. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `registry.go` with the ordered registry and a constructor per probe, starting with `local.env` only, and extend it as each probe lands. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `local.env` classification in `local.go` (Linux, macOS, WSL2, native Windows plus architecture) reading only the injected `Platform`/`FS` seams. <!-- sdd-owner: implementation -->
- [x] **RED + TRIANGULATE** — in `local_test.go`, script all four platform seams and assert each classification with its arch; assert the probe offers no provisioning action (R-HR-29). <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/`. <!-- sdd-owner: implementation -->
- [x] **RED** — add the refusal case: native Windows ⇒ `(measured, fail, node_platform_unsupported)` naming WSL2 as the supported path (R-HR-30). <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement the refusal in `local.env`, plus the degrade path where signals matching no supported classification yield `(unresolved, platform_unknown)` with no default guess. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — assert the WSL2 detail contains only the documented semantics (milliseconds idle, default 60000, Windows 11 only) and contains neither the child-of-init rule nor a `-1` sentinel (RG-4); assert WSL2 without systemd is detection-only and names the enabling work as a later slice. <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/`. <!-- sdd-owner: implementation -->

### PR 6 — WU10 + WU11 · `local.sshd` and `egress.hub.direct`

`Files`: `internal/probe/local.go`, `internal/probe/local_test.go` (additions); `internal/probe/egress.go`, `internal/probe/egress_test.go`. `Depends on`: PR 5.
**Budget-forced merge** (disclosed in the consolidation record): no three-unit group fits under the 600-line ceiling, so these two probe units share one PR. Second-split boundary if disputed or above 600: `local.sshd` first, then `egress.hub.direct`.

- [x] **RED** — add the divergence case: written ≠ effective configuration ⇒ `(measured, fail, sshd_config_divergence)` with both configurations in the verbatim detail (R-HR-18). <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `local.sshd` as three separately reportable observations (binary presence, service state, effective configuration) over the injected `CommandRunner`/`FS`, with an absent binary as `(measured, fail, sshd_absent)` stating that installation is a later slice. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add the degradation cases: nil `CommandRunner` ⇒ not-measured/`capability_excluded` with the capability named; deny-all runner ⇒ not-measured/`command_denied`; plus the probe's `indeterminate` control case. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — assert the result is never `pass` whenever any observation was not measured, and that all local input is read through the injected readers. <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestLocalSshd`. <!-- sdd-owner: implementation -->
- [x] **RED** — write `egress_test.go` with the three hub cases: with `--hub` the declared target is exactly the supplied `host[:port]`; with no hub the observation is not-measured/`input_missing_hub` with an empty target; a refused dial is `(measured, fail, conn_refused)`; a blackholed port is `(measured, fail, budget_expired)` produced by the probe whose budget is shorter than the runner bound. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement `egress.hub.direct` in `egress.go` over `EffectiveTargets` and the injected `Dialer`. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — assert not-measured and refused are distinguishable, no observation says or implies "blocked" when no attempt was made, and no transport viability is established from hub reachability (R-HR-02, RG-13). <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestEgressHub`. <!-- sdd-owner: implementation -->

### PR 7 — WU12 + WU13 · The egress reachability probes: public SSH and the Cloudflare edge regions

`Files`: `internal/probe/egress.go`, `internal/probe/egress_test.go` (additions). `Depends on`: PR 6.
Second-split boundary if this group lands above 600: the two SSH destination probes (WU12) first, then the two per-region Cloudflare probes (WU13).

- [x] **RED** — add one case per outcome for both probes: banner received; a non-SSH banner ⇒ measured fail/`banner_not_ssh`; refused and reset ⇒ measured fail; authoritative resolver negative ⇒ measured fail/`dns_no_such_host` while a resolver timeout ⇒ unresolved/`dns_unresolved` (the two must not collapse); plus each probe's `indeterminate` control case. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement both probes in `egress.go` through the classification table only. <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — add the partial-banner case: a truncated read that still carries an SSH identification string classifies as SSH, and the verbatim detail is preserved alongside the stable reason code (R-HR-07). <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestEgressSsh`. <!-- sdd-owner: implementation -->
- [x] **RED** — add: both regions pass; one region fails while the other passes ⇒ the split is visible per observation and the aggregate is `fail`; a fully blackholed target ⇒ measured fail/`budget_expired`; plus the `indeterminate` control case. <!-- sdd-owner: implementation -->
- [x] **GREEN** — implement both probes producing one observation per region (`region1`, `region2`) while the registry still contains exactly ten probes (D10). <!-- sdd-owner: implementation -->
- [x] **TRIANGULATE** — assert the declared target set for these probes carries both regions, so an override that replaces it is reflected per observation and the closed-set assertion still holds. <!-- sdd-owner: implementation -->
- [x] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestEgressCF`. <!-- sdd-owner: implementation -->

### PR 8 — WU14 + WU15 · The protocol probes: `egress.quic` and `tls.interception`

`Files`: `internal/probe/quic.go`, `internal/probe/quic_test.go`, `internal/probe/tls.go`, `internal/probe/tls_test.go`. `Depends on`: PR 7.
Second-split boundary if this group lands above 600: the QUIC probe (WU14) first, then `tls.interception` (WU15).

- [ ] **RED** — write `quic_test.go` with the four D8 outcomes and all four reason codes: any UDP reply ⇒ `(measured, pass, udp_response_received)`; silence ⇒ `(unresolved, udp_silence)`; ICMP port-unreachable ⇒ `(measured, fail, udp_unreachable)`; any other socket error ⇒ `(unresolved, udp_error_unclassified)`; plus the `indeterminate` control case. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `quic.go` as a raw UDP datagram over the injected `PacketDialer`/`PacketConn`; add no QUIC dependency. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert the RG-5 property: a silent socket is never `fail`, the detail never claims the path is blocked, and the declared narrow question is stated in the detail text. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestEgressQuic`. <!-- sdd-owner: implementation -->
- [ ] **RED** — write `tls_test.go` with: verified chain and declared expected issuer ⇒ `(measured, pass, ok)`; verification failure ⇒ `(measured, fail, tls_verify_failed)` with the verification code in the detail; issuer outside the declared expected set ⇒ `(measured, fail, tls_issuer_unexpected)` with wording that says "not in the declared expected set" and never accuses; other handshake error ⇒ `(unresolved, tls_handshake_unresolved)`; plus the `indeterminate` control case. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `tls.interception` over the injected `TLSVerifier` with the declared expected issuer set. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert the injected verifying configuration is unchanged after the run and that no unverified retry was performed, and that the observed issuer and verification code appear in the result (R-HR-04). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestTLSInterception`. <!-- sdd-owner: implementation -->

### PR 9 — WU16 + WU17b · `tls.truststore` and the fact accessors

`Files`: `internal/probe/tls.go`, `internal/probe/tls_test.go` (additions); `internal/diagnosis/facts.go`, `internal/diagnosis/rules_test.go` (accessor cases). `Depends on`: PR 8.
**Budget-forced merge** (disclosed in the consolidation record): the last probe and the measurement→reasoning bridge — the accessors that turn `[]probe.Result` into matchable reasoning states. Second-split boundary if disputed or above 600: `tls.truststore` first, then `facts.go`.

- [ ] **RED** — add the Linux controls: an accepting pool ⇒ `(measured, pass, ok)`; a rejected chain ⇒ `(measured, fail, truststore_rejects_chain)`. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `tls.truststore`: macOS ⇒ `(unresolved, truststore_platform_unavailable)` with the documented limitation in the result detail; `SSL_CERT_FILE`/`SSL_CERT_DIR` override set ⇒ `(unresolved, truststore_override_platform_bypass)`. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert the macOS limitation text is in the result itself (RG-3), and that the override case is `indeterminate` and never `pass`. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestTLSTruststore`, then the full probe suite with the ten probes registered. <!-- sdd-owner: implementation -->
- [ ] **RED** — add the accessor cases to `internal/diagnosis/rules_test.go`: a multi-observation probe exposes every observation's own state (DEV-2), and a not-measured observation maps to `NOT_MEASURED` rather than to a `fail` or `pass` state; prove the failure first (compile-level `RED` is expected here, as the package is new). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `internal/diagnosis/facts.go`: accessors turning `[]probe.Result` into matchable `(probe, resolution, verdict)` states. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — accessors only in `facts.go`; `gofmt -l .`, `go vet ./...`, `go test ./internal/diagnosis/`. <!-- sdd-owner: implementation -->

### PR 10 — WU17a + WU18 · Rule-id vocabulary, the rule mechanism and the derived fact rules

`Files`: `internal/diagnosis/ids.go`, `rules.go`, `findings.go`, `diagnose.go`, `rules_test.go` (additions). `Depends on`: PR 9.
Second-split boundary if this group lands above 600: the rule-id scheme with its coverage case (WU17a) first, then the mechanism, findings, `Diagnose` and the derived fact rules (WU18).

- [ ] **RED** — add `TestRuleIDsCoverEveryProbeState` to `rules_test.go`: every registered probe × `{PASS, FAIL, UNRESOLVED, NOT_MEASURED}` has exactly one rule id, and `AllRuleIDs()` has no duplicate and no orphan. Prove the failure first. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `ids.go`: derived `<PROBE_NAME_UPPERCASED>_<STATE>` ids for the four states and `AllRuleIDs()` in declaration order. <!-- sdd-owner: implementation -->
- [ ] **RED** — add the first-match case: for each question the first matching rule wins, there is **no** fall-through default, and a question with no matching rule emits an explicit open question naming the states the table needed. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `rules.go` (`[]Rule{ID, Question, Match []Need, Conclusion, DependsOn}` with `Need{Probe, Resolution, Verdict}`, plus the derived fact rules for the ten probes), `findings.go` (`Finding`, `OpenQuestion`, `Diagnosis`, conclusion texts) and `diagnose.go` (`Diagnose([]Result) Diagnosis`). <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — add one case per derived fact rule id, asserting the conclusion is the plain fact and `DependsOn` equals the matched observable set. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — ids only in `ids.go`, conclusion texts only in `findings.go`; `gofmt -l .`, `go vet ./...`, `go test ./internal/diagnosis/`. <!-- sdd-owner: implementation -->

### PR 11 — WU19 + WU20 · The destination-block group with the §1.1 replay, and the Cloudflare rule groups

`Files`: `internal/diagnosis/rules.go` (additions), `internal/diagnosis/matrix_test.go`, `internal/diagnosis/rules_test.go` (additions). `Depends on`: PR 10.
Second-split boundary if this group lands above 600: the `ssh.destination` group with the §1.1 matrix replay (WU19) first, then the Cloudflare edge and HTTP/2 advice groups (WU20).

- [ ] **RED** — write `matrix_test.go` replaying the six literal rows of PRD §1.1 with scripted results: the `ssh.destination` conclusion is "outbound SSH is allowed; the hub address is blocked", the reason names `203.0.113.10` with its port, and rule id `SSH_DEST_BLOCKED_BY_PUBLIC_SSH` is present (R-HR-03, PRD §13). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement the `ssh.destination` group in `rules.go`: `SSH_DEST_BLOCKED_BY_PUBLIC_SSH`, `..._443`, `SSH_DEST_BLOCK_UNESTABLISHED_PUBLIC_UNRESOLVED`, `..._FAILED`, `SSH_NO_DEST_BLOCK_OBSERVED`, `SSH_DEST_BLOCK_NOT_ASSESSED`. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — add the counterfactuals: both signals passing ⇒ no block conclusion; public probe `fail` or not-measured while the hub fails ⇒ the weaker rule fires and never claims "outbound SSH works"; hub unresolved or not-measured ⇒ `SSH_DEST_BLOCK_NOT_ASSESSED` with nothing claimed about the hub. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/diagnosis/ -run TestMatrixReplay`. <!-- sdd-owner: implementation -->
- [ ] **RED** — add one case per id: `CF_EDGE_REACHABLE`, `CF_EDGE_PARTIAL`, `CF_EDGE_UNREACHABLE`, `CF_EDGE_UNRESOLVED`, `CF_HTTP2_ADVISED_QUIC_FAILED`, `CF_HTTP2_ADVISED_QUIC_UNCONFIRMED`, `CF_NO_HTTP2_ADVICE_QUIC_USABLE`, `CF_HTTP2_ADVISORY_NOT_ASSESSED`. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement both question groups in `rules.go` over the per-region observations of PR 7. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert the unconfirmed path names the unresolved measurement it rests on, and that a working QUIC measurement produces no downgrade advice. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/diagnosis/`. <!-- sdd-owner: implementation -->

### PR 12 — WU21 + WU22 · The remaining rule groups with the suppression and agnosticism suites

`Files`: `internal/diagnosis/rules.go`, `internal/diagnosis/rules_test.go` (additions), `internal/diagnosis/unresolved_test.go`, `internal/diagnosis/agnostic_test.go`. `Depends on`: PR 11.
Second-split boundary if this group lands above 600: the TLS, `local.sshd` and node-platform rule groups (WU21) first, then the unresolved-propagation and agnosticism suites (WU22).

- [ ] **RED** — add one case per id: `TLS_INTERCEPTION_DETECTED`, `TLS_CHAIN_AS_EXPECTED`, `TLS_INTERCEPTION_UNRESOLVED`, `TRUSTSTORE_ACCEPTS_CHAIN`, `TRUSTSTORE_REJECTS_CHAIN`, `TRUSTSTORE_UNRESOLVED_PLATFORM`, `TRUSTSTORE_UNRESOLVED_OVERRIDE`, `SSHD_PRESENT_CONFIGURED`, `SSHD_PRESENT_CONFIG_DIVERGENT`, `SSHD_ABSENT`, `SSHD_EFFECTIVE_CONFIG_NOT_MEASURED`, `LOCAL_ENV_*`, `NODE_PLATFORM_SUPPORTED`, `NODE_PLATFORM_REFUSED_NATIVE_WINDOWS`, `NODE_PLATFORM_UNKNOWN`, `NODE_WSL2_SYSTEMD_ABSENT`. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement the four question groups in `rules.go`, with `SSHD_EFFECTIVE_CONFIG_NOT_MEASURED` as the default live case and no transport viability derived here. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert the RG-3 platform limitation reaches a conclusion text through both `TRUSTSTORE_UNRESOLVED_*` ids, and that no id exists beyond the four states. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/diagnosis/`. <!-- sdd-owner: implementation -->
- [ ] **RED** — write `unresolved_test.go`: a rule needing `measured/pass` or `measured/fail` is unreachable on unresolved or not-measured input, the weaker rule fires and names the unresolved probe, and independent conclusions are byte-identical to the run without the unresolved probe (R-HR-NF-03). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement the structural matching that makes the confident rule unreachable plus the completeness computation over observations by resolution (never over verdicts), so no fall-through default is needed. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — add the obligation 1/2 case: a not-measured observation produces a not-measured conclusion naming what was missing, an attempted-but-unresolved observation produces an unresolved conclusion naming the probe, and NF-03 holds at the engine layer. <!-- sdd-owner: implementation -->
- [ ] **RED + GREEN** — write `agnostic_test.go`, which reads the `diagnosis` package's own sources and asserts none of `direct-ssh`, `reverse-ssh`, `cloudflare-tunnel`, `tailscale`, `argotunnel`, `7844` appear (NF-04, design §6.4). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/diagnosis/`. <!-- sdd-owner: implementation -->

### PR 13 — WU23 + WU24 · The transport contract, placeholders, registry and the two SSH adapters

`Files`: `internal/transport/contract.go`, `internal/transport/pending.go`, `internal/transport/directssh.go`, `internal/transport/reversessh.go`, `internal/transport/registry.go`, `internal/transport/contract_test.go`. `Depends on`: PR 12.
Second-split boundary if this group lands above 600: the contract types, placeholders and typed error (WU23) first, then the two SSH adapters, the registry and the contract loop (WU24).

- [ ] **RED** — write `contract_test.go` with the type-level cases: the `RequirementKind` set is closed (`hostname`, `zone_membership`, `third_party_permission`, `hub_address`, `sshd_effective_config`), viability is a strict boolean with no third value in the type, `Feasibility` carries `Requires`, and the typed error text names the member and the owner slice. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `contract.go` (`Requirement`, `RequirementKind`, `Feasibility`, `Candidate`, `Transport`) and `pending.go` (placeholder `PairingBundle`, `Step`, `Handle` with a doc comment naming the owning slice and the removal trigger, plus `ErrNotImplementedInThisPhase`). <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — register a fixture type implementing only `Candidate` in the test and assert it reports its requirement list; assert no value is returned alongside the typed error. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/transport/`. <!-- sdd-owner: implementation -->
- [ ] **RED** — add the loop to `contract_test.go` over `transport.Registry()`: `!Viable ⟹ Reason != ""`; a viable transport names at least one measured observation in `Notes`; every kind declared by `Requires()` appears among the evaluated `Feasibility.Requires` rows. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement the two SSH adapters (`Feasible` only, derived from the `diagnosis` facts) and `registry.go` with the deterministic order. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert each rejection names the specific measured target or observation, and that a not-measured hub dependency yields a reason stating the measurement is missing rather than asserting a block (R-HR-06). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — no per-transport branch inside `diagnosis`; `gofmt -l .`, `go vet ./...`, `go test ./...` and re-run the agnostic guard. <!-- sdd-owner: implementation -->

### PR 14 — WU25 + WU26 · The `tailscale` and `cloudflare-tunnel` adapters, the not-implemented proof and registry closure

`Files`: `internal/transport/tailscale.go`, `internal/transport/notimplemented_test.go`, `internal/transport/cloudflare.go`, `internal/transport/registry_test.go`. `Depends on`: PR 13.
Second-split boundary if this group lands above 600: the `tailscale` adapter with the not-implemented proof (WU25) first, then `cloudflare-tunnel` feasibility with the registry closure (WU26).

- [ ] **RED** — write `notimplemented_test.go`: for every registered adapter, `PlanHub`, `PlanNode` and `Verify` fail with the typed error naming the member and the owner slice (`PlanHub`/`PlanNode` → R3, `Verify` → R5/R6). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `tailscale.go` (`Feasible` plus the three failing members) and register it. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert no value is returned alongside the error (the slice/handle is nil in every case) and the error is `errors.Is`-matchable, so a caller can never receive a plausible empty plan (RG-9). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/transport/ -run TestNotImplemented`. <!-- sdd-owner: implementation -->
- [ ] **RED** — write `registry_test.go`: exactly four transports in a stable order, nothing else registered, and no adapter researched for a later slice (for example Microsoft Dev Tunnels) present. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `cloudflare.go` `Feasible` plus `Requires()` rows (`hostname`, `zone_membership`) and complete `registry.go` with the fourth entry. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — on a reachable edge with no hostname: `viable == false`, a non-empty unsatisfied `requires` row naming the hostname, a note stating the prerequisite is unsatisfied, and no output describing the transport as ready to use (PRD §5.2, §14.5). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/transport/ -run 'TestRegistry|TestCloudflareRequirements'`. <!-- sdd-owner: implementation -->

### PR 15 — WU27 + WU28 · Recommendation wording, the pin note and the feasibility evidence table

`Files`: `internal/transport/cloudflare.go` (advice block), `internal/transport/pin_note.go`, `internal/transport/pin_test.go`, `internal/transport/feasibility_test.go`, `internal/transport/testtransport_test.go`. `Depends on`: PR 14.
Second-split boundary if this group lands above 600: the HTTP/2 advice and the pin note (WU27) first, then the evidence table and the NF-04 test transport (WU28).

- [ ] **RED** — write `pin_test.go`: the pin statement names upstream issue #1673 with its state (open, uncommented), names version 2026.6.0 **only**, states that later releases' behaviour is unknown, cites release 2026.5.1's published SHA256 checksums, asserts no version-range claim and no claim of installing, verifying or pinning anything on the machine, and asserts exactly one home for the wording (RG-1, R-HR-16). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `pin_note.go` as the single home and the QUIC-failed/unconfirmed/usable/not-assessed advice in `cloudflare.go`. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert the advice states that HTTP/2 forfeits post-quantum key agreement and that this slice recommends and does not enforce; assert no output claims a fallback was applied or a configuration written; assert working QUIC produces no downgrade advice (R-HR-05, RG-6). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/transport/`. <!-- sdd-owner: implementation -->
- [ ] **RED** — write `feasibility_test.go`: a finding-set × adapter table covering viable, non-viable, detected-but-blocked and unmeasured-dependency cases; for the PRD §1.1 replay the direct-SSH and reverse-SSH reasons name the blocked hub target with its port; no reason asserts a measurement that was never made (R-HR-06). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — extract the shared feasibility helpers the table exposes, without introducing a per-transport branch into `diagnosis`. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — write `testtransport_test.go`: register a `testTunnel` implementing only `Candidate`, assert the feasibility list contains it with its stated viability and reason, and assert every `diagnosis` finding is byte-identical to the same run without it (NF-04, design §6.4). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./...` and confirm the agnostic guard still passes with all four adapters present. <!-- sdd-owner: implementation -->

### PR 16 — WU29 + WU30 · Report DTO, the single mapping, coverage, completeness and determinism

`Files`: `internal/report/report.go`, `internal/report/map.go`, `internal/report/map_test.go`. `Depends on`: PR 15.
Second-split boundary if this group lands above 600: the DTO and the single mapping with its key-set cases (WU29) first, then the coverage, completeness, determinism and NF-03 cases (WU30).

- [ ] **RED** — write `map_test.go` asserting, with a fixed clock, the exact key set at every payload level (`schema_version`, `generated_at`, `tool`, `run`, `targets`, `probes`, `findings`, `open_questions`, `node`, `transports`) and that `schema_version` is `"1"`. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `report.go` (DTO types with explicit JSON field names) and `map.go` (the single mapping function), with probes in registry order, transports sorted by name and no map iteration. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert the mapping is the only producer: an internal structure changed without touching `map.go` leaves the payload unchanged, and `elapsed_ms` is an integer in one documented unit. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — one mapping, no duplicated field names; `gofmt -l .`, `go vet ./...`, `go test ./internal/report/`. <!-- sdd-owner: implementation -->
- [ ] **RED** — add the coverage case: a not-measured observation appears with `resolution: "not_measured"` and a reason naming the missing input while `run.not_measured` lists it; an unresolved observation appears in `run.unresolved` with `completeness: "incomplete"`; a not-measured probe alone does not make the run incomplete; `targets.hub` is JSON `null` when no hub was supplied (R-HR-NF-02, design §5.3). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — extend the mapping for the coverage and completeness fields without adding a derived success boolean anywhere. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — add the determinism case: repeated runs on the same inputs are byte-identical; probes follow registry order and transports are sorted; a clock-only change moves only `generated_at` and the elapsed values. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — add the NF-03 payload assertion: an unresolved probe carries `verdict: "indeterminate"` verbatim, no derived boolean, status or summary field marks it successful, and no field maps it to `pass`. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/report/`. <!-- sdd-owner: implementation -->

### PR 17 — WU31 + WU32 · Human projection and `docs/diagnosis-report.md` with its drift test

`Files`: `internal/report/human.go`, `internal/report/human_test.go`, `docs/diagnosis-report.md`, `internal/report/docs_test.go`. `Depends on`: PR 16.
Second-split boundary if this group lands above 600: the human projection (WU31) first, then the contract document with its drift test (WU32).

- [ ] **RED** — write `human_test.go`: the projection is written to the injected writer, renders an unresolved measurement as unresolved with its reason and never with success wording, names the unresolved probes when the run is incomplete, and shows the coverage gaps. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `human.go` with probe rows, findings, the coverage section and the transports. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert the blocked target string and the verbatim failing detail survive alongside the stable reason code, and that the completeness text never implies the suite finished when it did not (R-HR-07, R-HR-NF-02). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/report/ -run TestHumanProjection`. <!-- sdd-owner: implementation -->
- [ ] **RED** — write `docs_test.go` parsing the doc's two tables and asserting set equality with `probe.AllReasonCodes()` and `diagnosis.AllRuleIDs()`. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — write `docs/diagnosis-report.md`: the exit-code table, the stdout/stderr split, the payload shape with `schema_version`, the table of all 28 reason codes, the rule-id scheme, and the statement that adding a code or rule id is a contract change. No schema file is written (R-HR-NF-06). <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert a code added to the constants without a doc row, or a row without a constant, fails; document that the exit-code constants ↔ doc-table assertion lives in `internal/doctor/doctor_test.go` (PR 18) to avoid an import cycle. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./...`. <!-- sdd-owner: implementation -->

### PR 18 — WU33 + WU34 · The `doctor` harness, exit codes and the flag surface

`Files`: `internal/doctor/exit.go`, `internal/doctor/doctor.go`, `internal/doctor/doctor_test.go`, `cmd/herdr-reach/flags.go`, `cmd/herdr-reach/flags_test.go`. `Depends on`: PR 17.
Second-split boundary if this group lands above 600: the harness and the exit matrix (WU33) first, then the flag parsing and hub resolution (WU34).

- [ ] **RED** — write the exit matrix in `doctor_test.go` over scripted seams: `0` for a completed run including a negative answer, no viable transport and a native-Windows refusal; `1` when at least one probe was attempted and resolved unresolved (including the hanging probe); `2` for an unknown flag, an unusable flag value and an internal error with no diagnosis presented as completed; a not-measured probe alone leaves the exit code unchanged (R-HR-07, R-HR-30). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `exit.go` (`ExitOK`, `ExitIncomplete`, `ExitUsage`) and `doctor.go`: `EffectiveTargets` → `Registry` → runner → `diagnosis.Diagnose` → `transport.Evaluate` → `report.Build` → `WriteHuman(stderr)` always, `WriteJSON(stdout)` only with `--json`. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert the exit-code constants ↔ `docs/diagnosis-report.md` table, and that `run.completeness` and the exit code agree in every matrix case. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/doctor/`. <!-- sdd-owner: implementation -->
- [ ] **RED** — write `flags_test.go`: `--json`, `--hub`, repeatable `--target`, `--version` and `-h` parse into `doctor.Options`; the default hub port `22` applies when the port is omitted; a bracketed IPv6 literal is accepted as host-only; an empty host, an unparsable port, an unknown `--target` probe name and a malformed address are usage errors. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `flags.go` with the D3 parse rules; unusable input returns a usage error and starts no measurement run. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — add the hub-resolution cases: a supplied `host:port` becomes the hub target exactly and the resolved target is stated in the output; an omitted port resolves to 22 by the one documented rule; no hub yields not-measured/`input_missing_hub`, never "blocked", and no transport is viable on the strength of hub reachability (RG-13, R-HR-02). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./cmd/herdr-reach/`. <!-- sdd-owner: implementation -->

### PR 19 — WU35 + WU36 · Entrypoint wiring, stream split, production seams and the static no-real-network guard

`Files`: `cmd/herdr-reach/main.go`, `cmd/herdr-reach/main_test.go`, `internal/probe/real.go`, `internal/probe/guard_test.go`. `Depends on`: PR 18.
Second-split boundary if this group lands above 600: the entrypoint wiring and production seams (WU35) first, then the static guard (WU36).

- [ ] **RED** — write `main_test.go` driving the in-process `run(args, stdio, seams)`: with `--json` stdout parses as exactly one document and carries no prose, progress text or banner while the human text is on stderr; without `--json` stdout is empty; `--version` prints the version and exits `0`; exit codes are plumbed through. <!-- sdd-owner: implementation -->
- [ ] **GREEN** — implement `main.go` (build production seams, call `doctor.Run`, `os.Exit`) and `internal/probe/real.go`, the only file constructing real network primitives, with the production `CommandRunner` left **nil** so no code path can execute a third-party binary in R1a. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert a failing JSON write exits `2` with stdout empty and no diagnosis presented as completed. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./...`. <!-- sdd-owner: implementation -->
- [ ] **RED** — write `guard_test.go`, parsing the module's `.go` sources and asserting that direct construction of `net.Dial*`, `net.Lookup*`, `tls.Dial*` and `os/exec` appears only in `internal/probe/real.go` and `cmd/herdr-reach/main.go` (design §6.2 level 3). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — move into `real.go` whatever construction the guard reports outside those two files; do not widen the allowed list to make it pass. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — add the counter-case: a source string placed in a scratch fixture outside the allowed files fails the guard, so a permissive parser cannot pass vacuously; record in the test comment that this is a static check that narrows RG-7/RG-10 and does not replace the CI workflow R11 owns. <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + GATE** — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestStaticNoRealNetwork`. <!-- sdd-owner: implementation -->

### PR 20 — WU37 + WU38 · Writes-nothing proof, closed dialed set and the end-to-end honesty suite

`Files`: `internal/doctor/guard_test.go`, `internal/doctor/doctor_test.go` (end-to-end additions). `Depends on`: PR 19.
Second-split boundary if this group lands above 600: the writes-nothing, no-exec and closed-dialed-set proof (WU37) first, then the end-to-end honesty suite and the final gate (WU38).

- [ ] **RED** — run the CLI in-process with `HOME` and the working directory pointed at two fresh `t.TempDir()` trees, record recursive digests before and after, and assert byte-identical trees, zero command-runner calls, and set equality between dialed `(host, port, protocol)` triples and `probe.EffectiveTargets(opts)` (R-HR-02, R-HR-24, R-HR-27, design §6.3). <!-- sdd-owner: implementation -->
- [ ] **GREEN** — fix only inside the named files if the guard fails; never by weakening an assertion. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — add the "a write attempt cannot pass the guard" counter-case: a deliberate in-test write to the temp tree must fail the digest assertion. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — complete the three-level no-egress proof: sentinel reachability re-asserted from the doctor side (level 1) and the closed dialed set measured by a recording dialer built on the deny-all base (level 2). <!-- sdd-owner: implementation -->
- [ ] State the coverage boundary in the test comment: a temp-`HOME` digest cannot prove the absence of writes outside `HOME`/CWD; the slice closes that by construction (no writer code path, no write-capable dependency) and the residual gap stays in the risk register (RG-14). <!-- sdd-owner: implementation -->
- [ ] **RED** — add the PRD §1.1 replay through the real pipeline: it exits `0`, and both the human and machine projections carry the hub target string, the verbatim failing detail and the stable reason code (PRD §13, §1.1, RG-2). <!-- sdd-owner: implementation -->
- [ ] **RED + GREEN** — add `TestForbiddenStringsAbsent` over one complete run's two projections: the WSL2 text contains the documented millisecond semantics and neither the child-of-init rule nor a `-1` sentinel (RG-4); no output claims an HTTP/2 fallback was applied or enforced (RG-6). Fix only inside the owning units' files. <!-- sdd-owner: implementation -->
- [ ] **TRIANGULATE** — assert a complete run finishes inside an injected millisecond run budget with every probe bounded, and that a hanging-probe variant makes the run incomplete with exit `1` in both projections (R-HR-NF-02, R-HR-NF-09). <!-- sdd-owner: implementation -->
- [ ] **REFACTOR + FINAL GATE** — run `gofmt -l .`, `go vet ./...`, `go test ./...` and a hand `go test -race ./...`; record the exact commands and results as the unit's evidence; confirm no linter was introduced (`quality.lint` stays empty). <!-- sdd-owner: implementation -->
- [ ] Record the verify-phase hand-run obligations that no unit test can discharge: the §1.1 matrix reproduced by hand on the motivating network, the D8 raw-UDP result compared with `cloudflared`'s QUIC log lines, and the macOS trust-store behaviour on a real Mac. <!-- sdd-owner: implementation -->

---

## Cross-cutting gates (every work unit)

| Gate | Command | Note |
|---|---|---|
| Tests | `go test ./...` | The canonical command from `openspec/config.yaml`; unrunnable until PR 1 lands (RG-11). |
| Vet | `go vet ./...` | Configured typecheck. |
| Format | `gofmt -l .` | Must print nothing. |
| Race (hand run in apply) | `go test -race ./...` | The runner is concurrent; a race there is a correctness bug. |
| Lint | — | `quality.lint` is empty; no linter is introduced or invented. |
| Diff hygiene | `git diff --stat` against the parent branch | Each PR must show only its own group's files, or the diff is retargeted/re-based before review. |
| Budget | `git diff --numstat` summed per PR | Each PR must stay at or under the 600-line chain ceiling; a group that would exceed it is split at the boundary named in its section. |

Never in tests: real DNS, TCP, TLS or UDP egress; `sshd -T`; the macOS keychain; `herdr` or `cloudflared` (never invoked in R1a).

## Chain protocol for every PR

Each PR targets the previous PR's branch (PR 1 targets main), merges in order, and its body carries: the chain-shape diagram with 📍 on itself, its start state, its finished state, its dependency (the previous PR), what is explicitly out of scope for it, the unit verification command and result, the budget measurement, and the rollback boundary from the map. A reviewer accepts or rejects one group without reading the branches after it; a diff that shows another group's work is a base bug to fix, not to review.

## Out of scope carried forward (not tasks of this change)

- **Product specification §8.3 wording correction — handled outside this change.** The orchestrator corrects the specification's `cloudflared` pin sentence to the report-only wording as a **direct commit outside this SDD cycle**. It is not a task here and not a unit of the chain; the tool-side obligation lives in PR 15 (`pin_note.go` + `pin_test.go`).
- **Verify-phase hand-run evidence** (design §9): the PRD §1.1 six-row matrix reproduced by hand on the motivating network against the replayed verdict, the D8 raw-UDP comparison with `cloudflared`'s QUIC log lines, and the macOS trust-store behaviour on a real Mac. Not unit tests; recorded in the verify phase (and named as a PR 20 evidence obligation).
- **CI workflow** that would enforce "never dial the real network" continuously (RG-10), owned by R11. R1a narrows the gap with PR 3's deny-all default, PR 19's static guard and PR 20's recording dialer, and says so rather than assuming it closed.
- **Linter selection** (`quality.lint` empty): no decision made here and none invented.
- **R1b TUI work** (bubbletea/bubbles/lipgloss, role screen, `R-HR-01`, `NF-01`, `NF-07`), **R2–R12 slices**, and every mutation path (`Step`, plan/apply/undo, `internal/sysfile`, `internal/sshcfg`, `internal/environ`, `internal/shedr`, `contracts/`): owned by their own changes and untouched here.
