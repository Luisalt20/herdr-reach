# Apply Progress — `reach-diagnosis-core`

**Change**: `reach-diagnosis-core` · **Slice**: **PR 1 of 20** — WU1 “Module bootstrap and `internal/version`”
**Branch**: `reach-diagnosis-core-pr1-bootstrap` (created by the orchestrator, branched from `main`, base commit `d375243`; chain strategy `stacked-to-main`, so PR 1 targets `main`)
**Date**: 2026-09-14 · **Artifact store**: `both` (this file + Engram mirror `sdd/reach-diagnosis-core/apply-progress`)
**Strict TDD**: active — `openspec/config.yaml` declares `strict_tdd: true` with runner `go test ./...`
**Skill resolution**: `paths-injected` — read `/home/luisalt20/.config/opencode/skills/go-testing/SKILL.md` and `/home/luisalt20/.config/opencode/skills/work-unit-commits/SKILL.md` before writing code; no registry fallback was needed
**Delivery path consumed**: `auto-chain` / `stacked-to-main` — this run implements **only** the assigned slice and stops at its PR boundary
**Commit status**: nothing committed, staged, pushed or branched by this phase; work is left in the working tree for the orchestrator

---

## Structured status consumed

| Field | Value |
|---|---|
| `schemaName` / `schemaVersion` | `gentle-ai.sdd-status` / `2` |
| `changeName` | `reach-diagnosis-core` |
| `nextRecommended` | `apply` |
| `applyState` | `ready` |
| `dependencies.apply` / `.verify` / `.archive` | `ready` / `blocked` / `blocked` |
| `actionContext.mode` | `repo-local` |
| `actionContext.workspaceRoot` | `/home/luisalt20/projects/close/herdr-reach` |
| `actionContext.allowedEditRoots` | `["/home/luisalt20/projects/close/herdr-reach"]` |
| `artifactStore` | `openspec` (native) / `both` declared by the parent prompt; files written under `openspec/changes/reach-diagnosis-core/` and mirrored to Engram |
| `taskProgress` before this run | 165 total / 0 completed / 165 pending |
| `taskProgress` after this run | 165 total / **6 completed** / 159 pending |
| `artifacts.applyProgress` before this run | `missing` (this run creates it) |
| `actionContext` warnings | none |
| Work-unit ownership markers | all six PR 1 rows carry the terminal `<!-- sdd-owner: implementation -->` marker; no malformed, duplicate or unsupported marker was found in the file |

**Attempt context**: the harness reports one bounded attempt already active for this work unit (`token sha256:88dc9dd9…d85e09`) with a 150-changed-line ceiling. Per the parent prompt, the parent owns `sdd-attempt acquire`/`settle`; this executor did **not** call the native attempt command. The overage against that ceiling is disclosed in “Workload and PR boundary”.

---

## Completed tasks and their persisted checkbox updates

All six PR 1 rows were flipped from `- [ ]` to `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (lines 136–141) as the work completed, then re-read to confirm.

| # | Task (short) | Persisted update | Evidence |
|---|---|---|---|
| 1 | BOOTSTRAP — create `go.mod` (`module github.com/Luisalt20/herdr-reach`, `go 1.25.10`, no `toolchain`, zero requires, no `go mod tidy`) | `tasks.md:136` → `- [x]` | `go.mod` is 3 lines: module line, blank line, `go 1.25.10`; no `require`, no `toolchain`, no `go.sum` |
| 2 | RED — write `internal/version/version_test.go` and prove the failure | `tasks.md:137` → `- [x]` | `go test ./...` → exit 1, `internal/version: no non-test Go files … [build failed]` — a compile-level failure **inside** `internal/version`, so the module path resolves and the test can fail for the right reason |
| 3 | GREEN — implement `internal/version/version.go` | `tasks.md:138` → `- [x]` | First honest pass: `ok github.com/Luisalt20/herdr-reach/internal/version 0.005s`, exit 0 |
| 4 | TRIANGULATE — ldflags-injection shape + dev-build case with unset `Commit` | `tasks.md:139` → `- [x]` | Injection table (3 cases) + `TestUnsetCommitIsNeverFabricated` + the added `TestEmptyInjectionFallsBackToDevVersion` (own RED→GREEN cycle) |
| 5 | REFACTOR + GATE — comments/package doc tidy, `go test`, `go vet`, `gofmt -l`, `go.sum` situation | `tasks.md:140` → `- [x]` | `go test -count=1 ./...` exit 0 · `go vet ./...` exit 0 · `gofmt -l .` exit 0 with empty output · `go.sum` **absent** |
| 6 | Record the D4/RG-12 audience decision and that `README.md` is out of scope | `tasks.md:141` → `- [x]` | Recorded under “Deviations from design” (D4 note below) |

---

## Files changed (PR 1 file list only)

| Path | Status | Authored lines | Purpose |
|---|---|---|---|
| `go.mod` | added | 3 | The module: ratified path `github.com/Luisalt20/herdr-reach`, `go 1.25.10`, no `toolchain`, zero requires |
| `internal/version/version.go` | added | 52 | `Name`, `DevVersion`, `Version`, `Commit`, `Date` and `String()` |
| `internal/version/version_test.go` | added | 122 | Default identity, ldflags-injection shape, dev-build honesty, empty-injection fallback — external test package `version_test` importing the module path |
| `openspec/changes/reach-diagnosis-core/tasks.md` | modified | 6 lines changed in place | Six PR 1 checkboxes `- [ ]` → `- [x]` |
| `openspec/changes/reach-diagnosis-core/apply-progress.md` | added | — | This artifact |
| `go.sum` | **not created** | 0 | Expected: the module has zero non-stdlib dependencies, so `go.sum` is *absent by design*, not missing. No `go mod tidy` was run, so nothing toolchain-generated and empty exists either |

**Total authored additions for the code slice: 177 lines** (`3 + 52 + 122`), all additions, zero deletions. No file outside the PR 1 list was created or modified; `README.md`, `PRD.md`, the proposal, the specs, the design, `explore.md`, `research.md`, `preproposal.md` and `openspec/config.yaml` are untouched.

---

## Test commands run — exact commands and exit status

| # | Command | Exit | Observed output (abridged) |
|---|---|---|---|
| 1 | `go test ./...` (after `go.mod` only) | **1** | `go: warning: "./..." matched no packages` / `no packages to test` |
| 2 | `go test ./...` (RED, test file without implementation) | **1** | `github.com/Luisalt20/herdr-reach/internal/version: no non-test Go files in …/internal/version` / `FAIL … [build failed]` |
| 3 | `go test ./...` (GREEN) | **0** | `ok github.com/Luisalt20/herdr-reach/internal/version 0.005s` — **the first honest `go test ./...` pass in this repository** |
| 4 | `go test -v ./internal/version/` | **0** | `PASS` — `TestUnsetDefaults`, `TestLdflagsInjectionShape` (3 subtests), `TestUnsetCommitIsNeverFabricated` |
| 5 | `go test ./...` (TRIANGULATE RED) | **1** | `--- FAIL: TestEmptyInjectionFallsBackToDevVersion` — `String() with an empty injected Version = "herdr-reach ", want "herdr-reach 0.0.0-dev"` |
| 6 | `go test ./...` (TRIANGULATE GREEN) | **0** | `ok …/internal/version 0.005s` |
| 7 | mutation check: `String()` reduced to `return Name` via `perl -0pi -e` on a backup copy | **1** | Six failures across all four test functions — the suite has teeth; original restored from `/tmp/version.go.bak` |
| 8 | `go test -count=1 ./...` (post-restore, uncached, REFACTOR gate) | **0** | `ok github.com/Luisalt20/herdr-reach/internal/version 0.005s` |
| 9 | `go vet ./...` (config `quality.typecheck`) | **0** | no output |
| 10 | `gofmt -l .` (config `quality.format`) | **0** | empty output — nothing unformatted |
| 11 | `go build ./...` | **0** | no output |
| 12 | `go test -race -count=1 ./...` | **0** | `ok …/internal/version 1.019s` |
| 13 | `ls -l go.sum` | **2** (not found) | `ls: cannot access 'go.sum': No such file or directory` — the expected absent state |

**Acceptance evidence of this slice = run 3** (the first honest `go test ./...`), with runs 8–12 as the gate. Run 1 explains why RG-11 required this slice to exist before any other: before `go.mod` plus one package, the canonical command cannot run at all.

---

## TDD Cycle Evidence

RED → GREEN → TRIANGULATE → REFACTOR per PR 1 task row. RED was produced **before** any production code existed; the only non-test artifact written first was `go.mod`, which the task itself marks as the enabling artifact that no failing test can precede.

| Task row | Phase | Evidence produced | Observed failure | Observed pass |
|---|---|---|---|---|
| BOOTSTRAP | enabling artifact (TDD not applicable) | `go.mod` written; `go test ./...` run | `"./..." matched no packages` / `no packages to test`, exit 1 — the pre-bootstrap state RG-11 records | n/a — bootstrap carries no behaviour of its own; its acceptance is run 3 |
| RED | RED | `internal/version/version_test.go` written before `version.go` | exit 1: `no non-test Go files in …/internal/version` / `[build failed]` — the failure is *inside* `internal/version`, proving both that the module path resolves and that the test fails for the right reason | n/a |
| GREEN | GREEN | `internal/version/version.go` implemented | (previous step) | exit 0: `ok …/internal/version 0.005s`; `go test -v` shows `TestUnsetDefaults` PASS |
| TRIANGULATE | TRIANGULATE RED→GREEN | `TestLdflagsInjectionShape` (3 table cases incl. non-default `Commit`/`Date`), `TestUnsetCommitIsNeverFabricated`, then the new `TestEmptyInjectionFallsBackToDevVersion` | exit 1: `String() with an empty injected Version = "herdr-reach ", want "herdr-reach 0.0.0-dev"` — the blank-identity case genuinely failed before the fallback existed | exit 0: `ok …/internal/version 0.005s` with the fallback in place |
| REFACTOR + GATE | REFACTOR | Package doc rewritten around the ldflags invocation; `String()` documented to never carry commit/date; comments explain *why* per case, no code restructured; `/tmp/version.go.bak` removed by restore | exit 1 on the deliberate mutation (`return Name`) — six failures across all four test functions | exit 0: `go test -count=1 ./...`, `go vet ./...`, `gofmt -l .` (empty), `go build ./...`, `go test -race -count=1 ./...` |
| Record D4/RG-12 evidence | evidence | Written into this artifact (see below) | n/a | n/a |

**Triangulation depth.** Four independent angles on one behaviour: (1) unset defaults, (2) three injected shapes including a pre-release version, (3) the dev build with an unset `Commit` asserting six fabricated tokens never appear, (4) the degenerate empty injection. The mutation run proves all four can fail rather than merely passing.

---

## Deviations from design

| # | Deviation | Why | Design reference | Follow-up owner |
|---|---|---|---|---|
| — | **No functional deviation from design §7's file plan.** `go.mod`, `internal/version/version.go`, `internal/version/version_test.go` are exactly the three paths the design and the PR 1 row name. | — | design §7 `internal/version/` table; design D4 | n/a |
| 1 | `Name` and `DevVersion` are exported constants, beyond the four members design names (`Version`, `Commit`, `Date`, `String()`) | `String()` must render the documented `name version` shape, and `Name`/`DevVersion` are that shape's two literals; exporting them keeps the single home rule and lets later slices reference the name without duplicating the string. They are additive and use no 1.26-only feature | design §3.3 payload `"tool": { "name": "herdr-reach", "version": "0.0.0-dev" }`; design §7 `version.go` row | `sdd-verify` may treat as a trivial refinement |
| 2 | `String()` guards an empty `Version` by returning `DevVersion` | `-X …Version=` is a legal link flag that would otherwise render the non-identifying string `"herdr-reach "`. Reporting the documented dev version is honest; inventing a version would not be | governing principle (PRD §14.5) as restated in `openspec/config.yaml` context | `sdd-verify` |
| 3 | `Commit` and `Date` are deliberately **not** rendered by `String()` | The `name version` shape is documented; appending optional build metadata would make the identity depend on build flags, and an unset `Commit` must never render a fabricated commit. Both variables exist for linker injection and are asserted to hold exactly what was injected | design §7 `version.go` row (“`Version`, `Commit`, `Date` variables + `String()` for the payload”); PR 1 TRIANGULATE row | R1b/`--version` slice decides whether any consumer renders them |
| 4 | The test file is an **external** test package (`package version_test`) importing `github.com/Luisalt20/herdr-reach/internal/version` | Importing the ratified module path proves the module path resolves (RG-11 evidence), not merely that a directory exists | tasks PR 1 file list; `openspec/config.yaml` ratified module path | n/a |

**D4 / RG-12 evidence recorded per task row 6.** `go 1.25.10` with **no** `toolchain` line is the deliberate audience decision: it matches the floor `README.md` already declares publicly, so a contributor on 1.25.10 is not silently forced to download the author's `go1.26.6`. The cost — a 1.26-only language feature would fail for that contributor — is accepted and caught by the declared floor rather than by accident; this slice uses no 1.26-only feature. `README.md` (and its badge, which is what makes the discrepancy visible) is explicitly **out of scope** for this change, so the version string was not “fixed” to match the local toolchain. Local environment for the acceptance evidence: `go version go1.26.6 linux/arm64` building the module under `go 1.25.10` semantics.

---

## Workload and PR boundary

| Field | Value |
|---|---|
| Slice | PR 1 of 20 — `Module bootstrap and internal/version` (WU1) |
| PR 1 estimate in `tasks.md` | 60–100 lines (point ≈80) |
| Host attempt ceiling | 150 changed lines |
| **Actual authored additions** | **177 lines** (0 deletions) |
| Chain per-PR cohesion ceiling | 600 changed lines (user-approved) — 177 is well inside it |
| Review budget (session canonical) | 400 changed lines — 177 is inside it |
| Budget status | **Over the slice estimate and over the 150-line attempt ceiling; inside both review budgets** |
| PR boundary | Starts at `main` (base `d375243`) and ends at `internal/version` compiling and passing; PR 2 begins at `internal/probe/probe.go` and is **not** started |
| Rollback boundary | Delete `go.mod` and `internal/version/`; the repository returns to the greenfield state RG-11 documents (accepted there). `tasks.md` checkbox lines 136–141 revert to `- [ ]`; this artifact is the only other PR 1 file |
| Rollback independence | Nothing outside PR 1 depends on these files yet — no other package exists — so the revert removes no unrelated work |

**Why 177 > 150, stated honestly.** The overage is `internal/version/version_test.go` (122 lines) and `internal/version/version.go` (52). The test price is the four triangulation angles the TRIANGULATE row demands (unset defaults, three injected shapes, the never-fabricate-a-commit dev case, and the empty-injection fallback discovered during triangulation) plus the comments that explain *why* each case exists. It can be reduced only by deleting a test case or stripping explanatory comments, and the review-budget rule forbids both (“never shrink a diff by deleting comments, blank lines, docs, or tests”). No code-golf pass was attempted for that reason. The honest count is reported instead of a trimmed one, with a `size:exception` recommendation for the attempt ceiling only — the two review thresholds (400 session, 600 chain) are not exceeded, so **no change to the chain strategy or PR splitting is required, and PR 2 must not be pulled into this slice to compensate**.

## Remaining unchecked tasks

**Inside this slice: none.** All six PR 1 checkbox rows are `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (re-read after the edit: 6 checked, 159 unchecked).

The change-wide remainder is **159 checkbox lines in PR 2 – PR 20**, which belong to later chained slices on later branches (PR 2 targets this branch; the chain is `stacked-to-main`) and are **not** part of this work unit. The next unchecked line in the artifact, verbatim, is `tasks.md:148`:

```
- [ ] **RED** — write `internal/probe/probe_test.go` with `TestResultInvariants`: `Verdict ∈ {pass,fail} ⟺ Resolution == measured`; `Resolution ∈ {unresolved, not_measured} ⇒ Verdict == indeterminate`; `TestAggregateOrder` asserting `fail > indeterminate > pass`. Prove the failure first. <!-- sdd-owner: implementation -->
```

The remaining slices continue at `tasks.md:142` (PR 2 heading) through `tasks.md:443` (end of PR 20), locatable in the same artifact.

---

## Risks

| # | Risk | Status / handling |
|---|---|---|
| 1 | 177 authored lines against a 150-line attempt ceiling | Disclosed above with the exact count and the reason it cannot shrink without deleting tests/comments. Recommend the maintainer acknowledge `size:exception` for the **attempt ceiling**; both review thresholds are satisfied |
| 2 | `go.sum` absent | Expected, verified: zero non-stdlib dependencies. Any future `go mod tidy` that creates an empty `go.sum` is a toolchain artifact, not a missing deliverable |
| 3 | The test file mutates package-level variables to simulate ldflags injection | Each case restores via `defer`, so no case leaks state; cases run sequentially and no test uses `t.Parallel()`. If a later slice parallelises tests, this pattern would need revisiting |
| 4 | PR 1 evidence runs on `go1.26.6` while `go.mod` declares `go 1.25.10` | Intended by D4; the code uses no 1.26-only feature and `gofmt`/`vet` are clean. A 1.25.10 toolchain run is not available locally and remains an unverified gap the design already accepts |
| 5 | `--version` rendering is not implemented here | Out of this slice by design: `flags.go` lands in PR 18 and `main.go` in PR 19. `Commit`/`Date` are carried and asserted, not yet consumed |

---

# Apply Progress — PR 2 of 20 — WU2 + WU3 · Measurement types, the closed reason-code set, and the classification table

**Change**: `reach-diagnosis-core` · **Slice**: **PR 2 of 20** — “Measurement types, the closed reason-code set, and the classification table” (WU2 + WU3)
**Branch**: `feat/probe-vocabulary`, stacked on PR 1's branch `feat/module-bootstrap` (chain strategy `stacked-to-main`, so PR 2 targets PR 1's branch and **not** `main`)
**Date**: 2026-09-14 · **Artifact store**: `both` (this file + Engram mirror `sdd/reach-diagnosis-core/apply-progress`)
**Strict TDD**: active — `openspec/config.yaml` declares `strict_tdd: true` with runner `go test ./...`; RED → GREEN → TRIANGULATE → REFACTOR followed for every row of this slice
**Skill resolution**: `paths-injected` — read `/home/luisalt20/.config/opencode/skills/go-testing/SKILL.md` and `/home/luisalt20/.config/opencode/skills/work-unit-commits/SKILL.md` before writing code; no registry fallback was needed
**Delivery path consumed**: `auto-chain` / `stacked-to-main` — this run implements **only** the assigned slice and stops at its PR boundary; PR 1's entry above is preserved unchanged
**Commit status**: nothing committed, staged, pushed or branched by this phase; the work is left in the working tree for the orchestrator
**How to read this file**: PR 2's entry is the section below. PR 1's closing “Remaining unchecked tasks” paragraph described the state at the end of PR 1; the current remainder is restated at the end of this section.

---

## Structured status consumed (PR 2)

| Field | Value |
|---|---|
| `schemaName` / `schemaVersion` | `gentle-ai.sdd-status` / `2` |
| `changeName` | `reach-diagnosis-core` |
| `nextRecommended` | `apply` |
| `applyState` | `ready` |
| `dependencies.apply` / `.verify` / `.archive` | `ready` / `blocked` / `blocked` |
| `actionContext.mode` | `repo-local` |
| `actionContext.workspaceRoot` | `/home/luisalt20/projects/close/herdr-reach` |
| `actionContext.allowedEditRoots` | `["/home/luisalt20/projects/close/herdr-reach"]` |
| `artifactStore` | `openspec` (native) / `both` declared by the parent prompt; files written under `openspec/changes/reach-diagnosis-core/` and mirrored to Engram |
| `taskProgress` before this run | 165 total / 6 completed / 159 pending (PR 1) |
| `taskProgress` after this run | 165 total / **16 completed** / 149 pending |
| `actionContext` warnings | none |
| Work-unit ownership markers | all ten PR 2 rows carry the terminal `<!-- sdd-owner: implementation -->` marker; a whole-file check found 165 markers for 165 checkbox rows, with no malformed, duplicate, unsupported or non-terminal marker |

**Attempt context**: the harness reports one bounded attempt already active for this work unit (`token sha256:a4af4df2…7ae13b`) with a **900**-changed-line ceiling. Per the parent prompt, the parent owns `sdd-attempt acquire`/`settle`; this executor did **not** call the native attempt command. The overage against that ceiling is disclosed in “Workload and PR boundary”.

---

## Completed tasks and their persisted checkbox updates (PR 2)

All ten PR 2 rows were flipped from `- [ ]` to `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (lines 148–157) as the work completed, then re-read to confirm.

| # | Task (short) | Persisted update | Evidence |
|---|---|---|---|
| 1 | RED — `probe_test.go` with `TestResultInvariants` + `TestAggregateOrder`; prove the failure | `tasks.md:148` → `- [x]` | `go test ./...` exit 1: `github.com/…/internal/probe: no non-test Go files … [build failed]` — the failure is the missing implementation inside the new package |
| 2 | GREEN — `probe.go` (`Probe`, `ProbeKind`, `Verdict`, `Resolution`, `Observation`, `Result`, `Aggregate`) | `tasks.md:149` → `- [x]` | `go test -v ./internal/probe/` exit 0: `TestResultInvariants` (9 subtests, 4 valid cells) and `TestAggregateOrder` (8 subtests) PASS |
| 3 | RED — the reason-code case: closed, unique, documented declaration order, `AllReasonCodes()` order | `tasks.md:150` → `- [x]` | `go test ./...` exit 1: `undefined: probe.AllReasonCodes` (2 errors), after which the set itself and its order were unverifiable |
| 4 | GREEN — `reason.go` with the constants plus `AllReasonCodes()` | `tasks.md:151` → `- [x]` | `go test ./...` exit 0; `gofmt -l .` exit 0 with empty output |
| 5 | TRIANGULATE — never promote unresolved to `pass`, never turn not-measured into `fail`, reason = worst observation's | `tasks.md:152` → `- [x]` | `TestAggregateNeverPromotesAnUnmeasuredObservation` (5 cases) + reason assertions added to `TestAggregateOrder`; teeth proven by mutation (rank of `Indeterminate` set to `Pass`) → exit 1 with 3 failing subtests, restore → exit 0 |
| 6 | REFACTOR + GATE — one owner per concept, `gofmt`, `go vet`, `go test` | `tasks.md:153` → `- [x]` | No reason-code string literal exists outside `reason.go` (grep exit 1 = no match); `gofmt -l .` exit 0 (empty); `go vet ./...` exit 0; `go test -count=1 ./...` exit 0 |
| 7 | RED — `classify_test.go`: one case per design §5.1 row, all codes reachable, `TestWordingDrift` | `tasks.md:154` → `- [x]` | `go test ./...` exit 1: `undefined: probe.Purpose`, `probe.Observable`, `probe.Classification`, … (`too many errors`) — the table's vocabulary did not exist |
| 8 | GREEN — `classify.go` as the ordered table keyed by observation + declared purpose | `tasks.md:155` → `- [x]` | `go test -count=1 ./...` exit 0: all 29 §5.1 row cases, the reachability case and the wording-drift cases PASS |
| 9 | TRIANGULATE — control case (unclassifiable ⇒ `internal_error`/`platform_unknown`, never `pass`) + one observable through two probes with one purpose ⇒ one code | `tasks.md:156` → `- [x]` | `TestUnclassifiableObservationIsNeverAPass` (5 cases) + `TestOneObservableNeverClassifiesTwoWays` (29 subtests); teeth proven by three mutations, each exit 1 (see “Mutation evidence”) |
| 10 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/`, full suite | `tasks.md:157` → `- [x]` | `gofmt -l .` exit 0 (empty) · `go vet ./...` exit 0 · `go test -count=1 ./...` exit 0 · `go test -race -count=1 ./...` exit 0 · verbose run: 9 top-level functions, 94 PASS lines including subtests |

---

## Files changed (PR 2 file list only)

| Path | Status | Authored lines | Purpose |
|---|---|---|---|
| `internal/probe/probe.go` | added | 146 | `Probe`, `ProbeKind`, `Verdict`, `Resolution`, `Observation`, `Result` and `Aggregate([]Observation) (Verdict, ReasonCode)` |
| `internal/probe/reason.go` | added | 149 | `ReasonCode` plus the closed set of 28 codes (design §3.5's enumeration) and `AllReasonCodes()` in declaration order |
| `internal/probe/classify.go` | added | 271 | The ordered classification table of design §5.1: `Purpose`, `Observable`, `RawObservation`, `Classification`, `Classify`, `Observe` |
| `internal/probe/probe_test.go` | added | 270 | `TestResultInvariants`, `TestReasonCodeSetIsClosed`, `TestAggregateOrder`, `TestAggregateNeverPromotesAnUnmeasuredObservation` |
| `internal/probe/classify_test.go` | added | 342 | `TestClassificationTable` (29 §5.1 rows), `TestEveryReasonCodeIsReachable`, `TestUnclassifiableObservationIsNeverAPass`, `TestOneObservableNeverClassifiesTwoWays`, `TestWordingDrift` |
| `openspec/changes/reach-diagnosis-core/tasks.md` | modified | 10 lines changed in place (10 deletions + 10 additions = 20 changed lines) | Ten PR 2 checkboxes `- [ ]` → `- [x]` |
| `openspec/changes/reach-diagnosis-core/apply-progress.md` | modified | this appended section | Cumulative PR 2 evidence; PR 1's entry untouched |

**Authored additions for the code slice: 1,178 lines** (146 + 149 + 271 + 270 + 342), all additions, zero deletions — every code and test file is new. Composition of those 1,178 lines, measured with `wc -l` plus blank/comment tallies:

| Kind | Lines | Share |
|---|---|---|
| Production code | 249 | 21% |
| Test code | 492 | 42% |
| Comments (design rationale + per-constant meaning) | 366 | 31% |
| Blank separator lines | 71 | 6% |

No file outside the five PR 2 paths and the two artifact files was created or modified; `README.md`, `PRD.md`, the proposal, the specs, the design, `explore.md`, `research.md`, `preproposal.md` and `openspec/config.yaml` are untouched. No new dependency was added; no linter or CI configuration was introduced.

---

## Test commands run — exact commands and exit status (PR 2)

| # | Command | Exit | Observed output (abridged) |
|---|---|---|---|
| 1 | `go test ./...` (task 1 RED, test file without implementation) | **1** | `github.com/Luisalt20/herdr-reach/internal/probe: no non-test Go files in …/internal/probe` / `FAIL … [build failed]` |
| 2 | `go test -v ./internal/probe/` (task 2 GREEN) | **0** | `--- PASS: TestResultInvariants` (9 subtests) · `--- PASS: TestAggregateOrder` (8 subtests) |
| 3 | `go test ./...` (task 3 RED) | **1** | `internal/probe/probe_test.go:130:15: undefined: probe.AllReasonCodes`, `:164:11: undefined: probe.AllReasonCodes` |
| 4 | `go test ./...` (task 4 GREEN) | **0** | `ok …/internal/probe` · `ok …/internal/version` |
| 5 | mutation (task 5): `case Indeterminate: return 1` → `return 0` in `Aggregate`'s rank | **1** | `--- FAIL: TestAggregateOrder/indeterminate_outranks_pass` (`Aggregate = ("pass","ok")`), plus the two “never a pass” promotion cases |
| 6 | `go test -count=1 ./...` (task 5 restore) | **0** | `ok …/internal/probe` · `ok …/internal/version` |
| 7 | one-owner check: grep for the 28 reason-code literals in `probe.go`/`reason.go`, excluding `reason.go` | **1** (no match) | No reason-code string literal exists outside `reason.go` |
| 8 | `gofmt -l .` (task 6 gate) | **0** | empty output |
| 9 | `go vet ./...` (task 6 gate) | **0** | no output |
| 10 | `go test -count=1 ./...` (task 6 gate) | **0** | `ok …/internal/probe` · `ok …/internal/version` |
| 11 | `go test ./...` (task 7 RED) | **1** | `undefined: probe.Purpose`, `probe.Observable`, `probe.Classification`, `probe.PurposePortReachability`, … (`too many errors`) |
| 12 | `go test -count=1 ./...` (task 8 GREEN) | **0** | `ok …/internal/probe 0.009s` · `ok …/internal/version 0.009s` |
| 13 | mutation A (task 9): UDP silence row changed to `Measured/Fail` | **1** | `--- FAIL: TestClassificationTable/udp_silence` — the RG-5 wrong-classification risk is caught |
| 14 | mutation B (task 9): `ObsProbeBudgetExpired` row loses its `PurposePortReachability` gate | **1** | `--- FAIL: TestUnclassifiableObservationIsNeverAPass/a_dial_budget_expiry_on_a_question_that_is_not_about_reachability` (`= {measured fail budget_expired}`, want `{unresolved indeterminate internal_error}`) |
| 15 | mutation C (task 9): `ReasonUDPSilence` deleted from the declaration-order list | **1** | `--- FAIL: TestEveryReasonCodeIsReachable` and `--- FAIL: TestReasonCodeSetIsClosed` (`AllReasonCodes() has 27 codes, the documented set has 28`) |
| 16 | `go test -count=1 ./...` (task 9 restore) | **0** | `ok …/internal/probe` · `ok …/internal/version` |
| 17 | `gofmt -l .` (task 10 gate) | **0** | empty output |
| 18 | `go vet ./...` (task 10 gate) | **0** | no output |
| 19 | `go test -count=1 ./...` (task 10 gate) | **0** | `ok …/internal/probe 0.016s` · `ok …/internal/version 0.016s` |
| 20 | `go test -race -count=1 ./...` (task 10 gate) | **0** | `ok …/internal/probe 1.024s` · `ok …/internal/version 1.018s` |
| 21 | `go test -count=1 -v ./internal/probe/` (final inventory) | **0** | 9 top-level test functions, 94 `--- PASS` lines including subtests |

**Runtime harness**: **N/A for this slice**, with the reason stated rather than implied — the slice's deliverable is the measurement vocabulary and the classification table, which perform no I/O at all: no dialer, resolver, packet socket, command runner, clock, filesystem or process boundary exists yet, and `internal/probe`'s seams land in PR 3. The closest executable evidence is the package's own test binary (runs 2, 12, 19, 20, 21), which exercises every table row and every aggregate path in-process. PR 3 onward owns the seam-level runtime scenarios.

---

## TDD Cycle Evidence (PR 2)

RED → GREEN → TRIANGULATE → REFACTOR per PR 2 task row. Every RED was produced before the production code it exercises existed, and each RED failure names the missing symbol or the missing package file rather than a wrong assertion.

| Task row | Phase | Evidence produced | Observed failure (RED) | Observed pass (GREEN) |
|---|---|---|---|---|
| 1 RED — invariants + aggregate order | RED | `internal/probe/probe_test.go` written before any non-test file | exit 1: `no non-test Go files in …/internal/probe` / `[build failed]` — the package exists and the test fails for the right reason | n/a |
| 2 GREEN — `probe.go` | GREEN | `probe.go` implemented (`Probe`, `ProbeKind`, `Verdict`, `Resolution`, `Observation`, `Result`, `Aggregate`) | (previous row) | exit 0: `TestResultInvariants` 9 subtests PASS, `TestAggregateOrder` 8 subtests PASS |
| 3 RED — closed reason-code set | RED | `TestReasonCodeSetIsClosed` added (28-code literal set, order, uniqueness, copy semantics) | exit 1: `undefined: probe.AllReasonCodes` (two call sites) | n/a |
| 4 GREEN — `reason.go` | GREEN | 28 constants in design §3.5 order + `AllReasonCodes()` returning a copy | (previous row) | exit 0: `go test ./...`; `gofmt -l .` empty |
| 5 TRIANGULATE — no promotion, worst reason | TRIANGULATE | `TestAggregateNeverPromotesAnUnmeasuredObservation` (5 cases) and reason assertions on `TestAggregateOrder` | Cases pass against the GREEN implementation; teeth proven by mutation #5 (`Indeterminate` ranked as `Pass`) → exit 1 with 3 failing subtests | exit 0 after restore |
| 6 REFACTOR + GATE — one owner per concept | REFACTOR | Doc/comment pass only; no code restructured. Ownership verified by grep (run 7) | n/a | exit 0: `gofmt -l .` (empty), `go vet ./...`, `go test -count=1 ./...` |
| 7 RED — classification table | RED | `classify_test.go` written (29 §5.1 rows, reachability, wording drift) before `classify.go` | exit 1: `undefined: probe.Purpose`, `probe.Observable`, `probe.Classification`, `probe.PurposePortReachability`, … (`too many errors`) | n/a |
| 8 GREEN — `classify.go` | GREEN | Ordered table (30 rows incl. the explicit internal-failure row and the total row) + `Classify` + `Observe` | (previous row) | exit 0: all 29 row cases, reachability and wording-drift cases PASS |
| 9 TRIANGULATE — control case + single-code guarantee | TRIANGULATE | `TestUnclassifiableObservationIsNeverAPass` (5 cases incl. the purpose-mismatch and the non-reachability budget expiry) and `TestOneObservableNeverClassifiesTwoWays` (29 subtests + the two SSH probes) | Cases pass against the GREEN implementation; teeth proven by mutations #13, #14, #15, each exit 1 | exit 0 after restores |
| 10 REFACTOR + GATE — final gates | REFACTOR | One code improvement (an explicit `ObsInternalFailure` row so a probe has a keyed internal-failure path, keeping the total row as the unknown-fact safety net); trailing blank lines and one `t.Fatalf`→`t.Fatal` tidied | n/a | exit 0: `gofmt -l .` (empty), `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...` |

**Triangulation depth.** Nine top-level test functions give four independent angles on the vocabulary (exhaustive 9-cell resolution × verdict matrix; the 28-code closed set and its order; the aggregate order plus the five promotion/sibling cases; the empty-observation case) and four on the table (all 29 design §5.1 rows; total reachability of the closed set; the unclassifiable/control paths including the purpose mismatch and the purpose-gated budget expiry; purity plus wording drift). Three mutations, each caught, prove the cases can fail rather than merely passing.

---

## Deviations from design (PR 2)

| # | Deviation | Why | Design reference | Follow-up owner |
|---|---|---|---|---|
| 1 | **The closed set has 28 codes, not 27.** Design §3.5's heading says `(27)` while the set it enumerates contains 28 distinct codes (`ok` is the only code two §5.1 rows share, so 29 table rows → 28 codes). All 28 are implemented. | Dropping any one code would leave at least one design §5.1 row unclassifiable — there is no candidate code to delete. The enumeration (content) outranks the heading (count), and the test pins the enumeration and prints the discrepancy in its failure message. | design §3.5 (heading vs enumeration); design §5.1 (29 rows); tasks PR 2 row 3 (“exactly the 27 codes”) | `sdd-verify` must adjudicate; `docs/diagnosis-report.md` (PR 17) must document 28 codes, and its `docs_test.go` set-equality check will surface the same mismatch |
| 2 | The table's Go vocabulary is chosen here: `Purpose`, `Observable`, `RawObservation`, `Classification`, `Classify`, `Observe`, and the seven `Purpose*` / 29 `Obs*` constants. | Design §5.1 and D2 specify the table's *content* (“one raw observation … `(Resolution, ReasonCode)`”, keyed by observation plus the probe's declared purpose) but name no Go identifiers. The purposes correspond one-to-one with the §5.1 row classes, and every documented purpose and observable is used by at least one row. | design D2; design §5.1 | later probe PRs (PR 5–PR 9) must declare the purpose and observable that match their real declared question |
| 3 | `Aggregate` over an **empty** observation list returns `(indeterminate, internal_error)`. | Design §3.1 documents the aggregate order but not the empty case. Returning `pass` with no evidence would fabricate success; reporting an internal failure is the honest reading and matches §5.1's last row. | design §3.1; design §5.1 last row; PRD §14.5 | `sdd-verify` may treat as a trivial refinement |
| 4 | `Classify` keeps a defensive post-loop return mirroring the table's total last row. | Go requires a terminating statement after a `for` loop, and the zero `Classification` (empty resolution, empty reason) must never leak. The total row is the documented path; the return is the structural safety net for a future edit of that row. | design §5.1 last row | n/a |
| 5 | `ObsInternalFailure` has its own universal row in addition to the total row. | A probe that catches an internal failure should report a *keyed* fact rather than rely on passing an unrecognised string; the total row remains the safety net for facts no row knows. | design §5.1 last row | PR 4/PR 5 probes that catch an unexpected error |
| 6 | The `budget_expired` row is documented for `PurposePortReachability` **only**. | Design §5.1 says the budget expiry is a definite negative “only because the probe's declared question is ‘is this port reachable’”. For any other declared question the same fact is not documented, so it falls to the total row (`internal_error`) instead of being borrowed as a failure. A UDP probe whose budget expires reports `udp_silence`, which is what D8 requires. | design §5.1 (row 5, Class column); design D8 (RG-5) | PR 8's QUIC probe must report `udp_silence`, never `probe_budget_expired` |
| 7 | `AllReasonCodes()` returns a copy of the closed set. | The set is a contract quoted by both projections; handing out the package's own slice would let a caller reorder or shrink it. Asserted by a test. | design §3.5 (“adding or renaming a code is a contract change”) | n/a |
| 8 | `ReasonCode` and `ReasonInternalError` were declared during the **task-2 GREEN** step, before the closed-set task. | `probe.go`'s required signature `Aggregate([]Observation) (Verdict, ReasonCode)` cannot compile without the reason vocabulary's type, and the empty-observation case needs one code. The remaining 27 codes and `AllReasonCodes()` landed in the task-4 GREEN step, so task 3's RED (`undefined: probe.AllReasonCodes`) was a genuine RED for the set itself. | tasks PR 2 rows 2 and 4; design §3.5 | n/a |
| 9 | Both test files are **external** test packages (`package probe_test`) importing the module path. | Importing `github.com/Luisalt20/herdr-reach/internal/probe` proves the exported vocabulary is sufficient to state design §3.1's invariants and §5.1's table without reaching into the package; it also matches PR 1's choice. | tasks PR 2 file list; design §8 test plan | n/a |

---

## Workload and PR boundary (PR 2)

| Field | Value |
|---|---|
| Slice | PR 2 of 20 — “Measurement types, the closed reason-code set, and the classification table” (WU2 + WU3) |
| PR 2 estimate in `tasks.md` | 470–720 lines (point ≈595) |
| Host attempt ceiling | 900 changed lines |
| **Actual authored code + tests** | **1,178 lines** (0 deletions; all five files are new): 249 production, 492 test, 366 comment, 71 blank |
| Artifact changes | `tasks.md` 10 lines changed (10 + 10 = **20** changed lines) + this appended section (**193** added lines) |
| **Total counted changed lines for the work unit** | **1,391** (1,178 authored code + tests, 193 artifact, 20 checkbox flips) |
| Chain per-PR cohesion ceiling | 600 changed lines (user-approved) |
| Review budget (session canonical) | 400 changed lines |
| Budget status | **Over the slice estimate, over the 600-line chain ceiling, over the 900-line attempt ceiling and over the 400-line session budget** (1,391 counted changed lines) |
| PR boundary | Starts at PR 1's branch state (`go.mod` + `internal/version/` only) and ends at `internal/probe` compiling and passing with the vocabulary, the closed reason-code set and the classification table in place. `internal/probe/seams.go` (PR 3) is **not** started, and no later-slice file exists |
| Rollback boundary | Delete the five files under `internal/probe/`; the module returns to its PR 1 state (`go.mod` + `internal/version/`) with no other package affected. `tasks.md` lines 148–157 revert to `- [ ]`; this appended section is the only other PR 2 change |
| Rollback independence | Nothing consumes `internal/probe` yet — PR 3 onward imports it — so reverting removes no unrelated work and leaves PR 1's slice green |

**Why 1,178 > 900, stated honestly.** The overage is not padding: 492 of the 1,178 lines are the two test files, whose size is set by the task rows themselves — one case per one of the 29 design §5.1 rows, all 28 codes reachable, the 9-cell resolution × verdict matrix, the five aggregate promotion cases, the wording-drift cases and the exhaustive control cases. Another 366 lines are comments, of which 138 are the per-row and per-constant explanations of design §5.1 and §3.5 (the design states the codes' meaning, so the code documents the same meaning beside each constant). The production code is 249 lines. The review-budget rule forbids reaching a number by deleting tests, control cases, triangulation cases, comments, docs or blank lines, and no `size:exception` was assumed by the human, so **no code-golf pass was attempted**. An honest 900-line slice would have to drop either the 29-row transcription or the closed-set/aggregate suites, which is exactly the coverage this slice exists to provide.

**Two remedies exist, both owned by the orchestrator, and neither was taken unilaterally.** (a) Record `size:exception` for this attempt and the chain ceiling, accepting PR 2 as one cohesive reviewable unit. (b) Apply the second-split boundary `tasks.md` already names for this group: PR 2a = WU2, the vocabulary and the closed reason-code set (`probe.go` 146 + `reason.go` 149 + `probe_test.go` 270 = **565** authored lines, inside the 600-line ceiling); PR 2b = WU3, the classification table (`classify.go` 271 + `classify_test.go` 342 = **613** authored lines, 13 over the chain ceiling but well inside the 900-line attempt ceiling). `internal/probe/reason.go` and `probe.go` are consumed by `classify.go`, so the split is strictly ordered and PR 2b would target PR 2a's branch. **This executor did not split the slice or switch branches**: the parent prompt assigned PR 2 as one unit and forbade branch operations.

---

## Remaining unchecked tasks (PR 2 view)

**Inside this slice: none.** All ten PR 2 checkbox rows are `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (re-read after the edits: **16 checked, 149 unchecked**; 165 rows total). The ownership markers were re-checked: 165 `<!-- sdd-owner: implementation -->` markers for 165 rows, every one terminal, none malformed, duplicate, unsupported or non-terminal.

The change-wide remainder is **149 checkbox lines in PR 3 – PR 20**, which belong to later chained slices on later branches (PR 2 targets PR 1's branch; the chain is `stacked-to-main`) and are **not** part of this work unit. The next unchecked line in the artifact, verbatim, is `tasks.md:164`:

```
- [ ] **RED** — write `seams_test.go` asserting `DenyAllSeams()` returns the sentinel error for dial, lookup, TLS verification and packet operations; the command runner denies everything; `FS` reports not-exist and an empty environment; `Platform` reports `unknown`; `Clock` is a deterministic stepper. Prove the failure first. <!-- sdd-owner: implementation -->
```

---

## Risks (PR 2)

| # | Risk | Status / handling |
|---|---|---|
| 1 | 1,178 authored lines against a 900-line attempt ceiling, a 600-line chain ceiling and a 400-line session budget | Disclosed above with the composition and the reason it cannot shrink without deleting tests, cases or comments. Recommend the maintainer either acknowledge `size:exception` or apply the named WU2/WU3 second split (565 / 613 authored lines) |
| 2 | design §3.5's heading says 27 codes while its enumeration has 28 (and §5.1 has 29 rows, not 27) | Implemented the enumeration (content over count) and pinned it in a test with an explanatory failure message. `sdd-verify` must adjudicate before `docs/diagnosis-report.md` (PR 17) documents the set |
| 3 | Purpose gating is an interpretation: a `probe_budget_expired` fact under a non-reachability purpose resolves to `internal_error` rather than a failure | Asserted by a dedicated control case and proven to have teeth by mutation. Later probes must name the observable that matches their true declared question (a UDP probe reports `udp_silence`) |
| 4 | “Only the classification table selects a reason code” is enforced by convention plus tests, not by a static guard | PR 3's seam tests and PR 19's static no-real-network guard narrow this; a dedicated guard on reason-code literals could be added there if the maintainer wants it enforced statically |
| 5 | `Observe`/`Classify` are exported, so a later probe could still hand-build an `Observation` with an inconsistent triple | The vocabulary's invariant is asserted over every table row, and later per-probe tests assert it over produced results; the structural fix (making the fields unexported) would break the JSON mapping's needs, so it is not taken here |
| 6 | The `internal_error` code is reached only through unclassifiable paths | Covered by the control cases (unknown fact, purpose mismatch, non-reachability budget expiry) and included in the reachability test, so it cannot silently lose coverage |

### Orchestrator note — delivery split (2026-09-14)

PR 2 exceeded the chain's per-PR ceiling (1178 authored lines against a task estimate of 470–720), so its
delivery is split at the boundary the plan already named: **PR 2a** carries the measurement vocabulary and the
closed reason-code set (`probe.go`, `reason.go`, `probe_test.go`), and **PR 2b** carries the classification
table (`classify.go`, `classify_test.go`) plus this artifact update. The implementation is unchanged and all
ten tasks belong to one work unit; only the review boundary moved, because a 1178-line review is not a
review. The reason-code count was also reconciled here: the closed set holds **28** codes, not the 27 the
design heading claimed, and the design heading was corrected with a dated note.

---

# Apply Progress — PR 3 of 20 — WU4 + WU5 · Seams, the deny-all test default, and the declared target set

**Change**: `reach-diagnosis-core` · **Slice**: **PR 3 of 20** — “Seams, the deny-all test default, and the declared target set” (WU4 + WU5)
**Branch**: `feat/probe-seams`, stacked on PR 2's branch `feat/probe-classification` (chain strategy `stacked-to-main`, so PR 3 targets PR 2's branch and **not** `main`)
**Date**: 2026-09-14 · **Artifact store**: `both` (this file + Engram mirror `sdd/reach-diagnosis-core/apply-progress`, split as the mirror note at the end records)
**Strict TDD**: active — `openspec/config.yaml` declares `strict_tdd: true` with runner `go test ./...`; RED → GREEN → TRIANGULATE → REFACTOR followed for every row of this slice
**Skill resolution**: `paths-injected` — read `/home/luisalt20/.config/opencode/skills/go-testing/SKILL.md` and `/home/luisalt20/.config/opencode/skills/work-unit-commits/SKILL.md` before writing code; no registry fallback was needed
**Delivery path consumed**: `auto-chain` / `stacked-to-main` — this run implements **only** the assigned slice and stops at its PR boundary; the PR 1 and PR 2 entries above are preserved unchanged
**Commit status**: nothing committed, staged, pushed or branched by this phase; the work is left in the working tree for the orchestrator
**How to read this file**: PR 3's entry is the section below. The current change-wide remainder is restated at the end of this section.

---

## Structured status consumed (PR 3)

| Field | Value |
|---|---|
| `schemaName` / `schemaVersion` | `gentle-ai.sdd-status` / `2` |
| `changeName` | `reach-diagnosis-core` |
| `nextRecommended` | `apply` |
| `applyState` | `ready` |
| `dependencies.apply` / `.verify` / `.archive` | `ready` / `blocked` / `blocked` |
| `actionContext.mode` | `repo-local` |
| `actionContext.workspaceRoot` | `/home/luisalt20/projects/close/herdr-reach` |
| `actionContext.allowedEditRoots` | `["/home/luisalt20/projects/close/herdr-reach"]` — every file written lives inside it |
| `artifactStore` | `openspec` (native) / `both` declared by the parent prompt; files written under `openspec/changes/reach-diagnosis-core/` and mirrored to Engram |
| `taskProgress` before this run | 165 total / 16 completed / 149 pending (PR 1 + PR 2) |
| `taskProgress` after this run | 165 total / **24 completed** / 141 pending |
| `actionContext` warnings | none |
| Work-unit ownership markers | all eight PR 3 rows carry the terminal `<!-- sdd-owner: implementation -->` marker; a whole-file re-check found 165 markers for 165 checkbox rows, none malformed, duplicate, unsupported or non-terminal |
| `applyState: all_done`? | no — implementation continues, so editing was permitted |

**Attempt context**: the harness reports one bounded attempt already active for this work unit (`token sha256:f64eb5f0…684f93`) with a **1700**-changed-line ceiling. Per the parent prompt the parent owns `sdd-attempt acquire`/`settle`; this executor did **not** call the native attempt command. The overage against that ceiling is disclosed in “Workload and PR boundary”.

---

## Completed tasks and their persisted checkbox updates (PR 3)

All eight PR 3 rows were flipped from `- [ ]` to `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (lines 164–171) as the work completed, then re-read to confirm.

| # | Task (short) | Persisted update | Evidence |
|---|---|---|---|
| 1 | RED — `seams_test.go` asserting every `DenyAllSeams()` capability returns the sentinel; prove the failure | `tasks.md:164` → `- [x]` | `go test ./...` exit 1: `undefined: probe.DenyAllSeams`, `probe.ErrSeamDenied` (×5), `probe.TLSVerification` |
| 2 | GREEN — `seams.go`: eight seam interfaces, `Seams` passed explicitly, `DenyAllSeams()` | `tasks.md:165` → `- [x]` | `go test -count=1 ./...` exit 0; eight capability subtests PASS (dial, lookup, TLS, packet, command, filesystem, platform, clock) |
| 3 | TRIANGULATE — nil `CommandRunner` = capability excluded vs deny-all runner = denied; neither yields `pass` | `tasks.md:166` → `- [x]` | `TestCommandFactDistinguishesMissingCapabilityFromDenial` (4 fact cases + 3 property subtests), plus `TestDenyAllSeamsNeverReportsARejectedChain`, `TestDenyAllSeamsIsFreshPerCall`, `TestStepperClockIsScriptedAndConcurrencySafe`; teeth proven by mutations A–C (each exit 1) |
| 4 | REFACTOR + GATE — inspection test that no seam is a package-level variable; `gofmt`, `go vet`, `go test ./internal/probe/` | `tasks.md:167` → `- [x]` | `TestNoPackageLevelSeamVariable` (AST guard over the package's own sources, vocabulary read from `seams.go`); four guard mutations each exit 1; focused `go test -run 'TestDenyAllSeams\|TestEffectiveTargets'` exit 0; `gofmt -l .` exit 0 (empty); `go vet ./...` exit 0; `go test -race -count=1 ./internal/probe/` exit 0 |
| 5 | RED — `targets_test.go`: one home, ten probes with `(host, port, protocol)` triples, override replaces, three usage errors | `tasks.md:168` → `- [x]` | `go test ./...` exit 1: `undefined: probe.DeclaredTargets`, `probe.ProtocolLocal`, `probe.ProtocolTCP/UDP/TLS`, `probe.ProbeDeclaration`, `probe.EffectiveTarget`, … |
| 6 | GREEN — `targets.go`: declared set, per-probe triples, `EffectiveTargets`, override parser | `tasks.md:169` → `- [x]` | `gofmt -l .` exit 0 (empty) · `go vet ./...` exit 0 · `go test -count=1 ./...` exit 0 (nine declaration, override, hub and error tests) |
| 7 | TRIANGULATE — closed set after an override; every declared triple reachable from the declaration | `tasks.md:170` → `- [x]` | `TestEffectiveSetIsClosedAfterAnOverride` (3 subtests: reachability from `DeclaredTargets()`, one override per probe, no triple escaping the declaration); teeth proven by mutations A–E (each exit 1) |
| 8 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/` | `tasks.md:171` → `- [x]` | Review added address validation for hand-built overrides (`ErrTargetAddress`) plus `TestDeclaredTargetsReturnsACopy`; one duplicate base-case test removed (see “Deviations”). Final: `gofmt -l .` exit 0 (empty) · `go vet ./...` exit 0 · `go test -count=1 ./...` exit 0 · `go test -race -count=1 ./...` exit 0 |

---

## Files changed (PR 3 file list only)

| Path | Status | Authored lines | Purpose |
|---|---|---|---|
| `internal/probe/seams.go` | added | 354 | The eight seam interfaces (`Dialer`, `Resolver`, `TLSVerifier`, `PacketDialer`/`PacketConn`, `CommandRunner`, `Clock`, `FS`, `Platform`), `TLSVerification`, `ErrSeamDenied`, `ErrTLSVerification`, the `Seams` struct passed explicitly, `CommandFact`, `DenyAllSeams()`, `NewStepperClock()` and the denying implementations |
| `internal/probe/seams_test.go` | added | 407 | `TestDenyAllSeamsDenyEveryCapability` (8 capability subtests), the nil-versus-denied triangulation, the rejected-chain and freshness cases, the scripted-clock contract, and `TestNoPackageLevelSeamVariable` (AST guard) |
| `internal/probe/targets.go` | added | 382 | The declared target set (ten probes, specification order), `Protocol`, `Target`, `ProbeDeclaration`, `EffectiveTarget`, `DeclaredTargets()`, `TargetInput`, `TargetOverride`, `EffectiveTargets()`, `ParseTargetOverride()`, `parseAddress()`, the four typed usage errors |
| `internal/probe/targets_test.go` | added | 591 | Fifteen tests: declaration completeness and PRD §1.1 triples, one-home guard, override replacement/default-port/hub/error matrices, the closed-set triangulation, and accessor copy isolation |
| `openspec/changes/reach-diagnosis-core/tasks.md` | modified | 8 lines changed in place (8 deletions + 8 additions = 16 changed lines) | Eight PR 3 checkboxes `- [ ]` → `- [x]` |
| `openspec/changes/reach-diagnosis-core/apply-progress.md` | modified | this appended section | Cumulative PR 3 evidence; the PR 1 and PR 2 sections are untouched |

**Authored additions for the code slice: 1,734 lines** (354 + 407 + 382 + 591), all additions, zero deletions — every code and test file is new. Composition of those 1,734 lines, measured with `wc -l` plus blank/comment tallies:

| File | Total | Production/test code | Comments | Blank |
|---|---|---|---|---|
| `seams.go` | 354 | 141 | 173 | 40 |
| `seams_test.go` | 407 | 334 | 44 | 29 |
| `targets.go` | 382 | 224 | 133 | 25 |
| `targets_test.go` | 591 | 519 | 63 | 37 |

No file outside the four PR 3 paths and the two artifact files was created or modified; `README.md`, `PRD.md`, the proposal, the specs, the design, `explore.md`, `research.md`, `preproposal.md` and `openspec/config.yaml` are untouched. No new dependency was added (the seam files import only the standard library: `context`, `crypto/tls`, `errors`, `fmt`, `io/fs`, `net`, `os`, `strconv`, `strings`, `sync`, `time`, and `go/ast`, `go/parser`, `go/token` in the test); no linter or CI configuration was introduced.

---

## Test commands run — exact commands and exit status (PR 3)

| # | Command | Exit | Observed output (abridged) |
|---|---|---|---|
| 1 | `go test ./...` (task 1 RED) | **1** | `internal/probe/seams_test.go:19:17: undefined: probe.DenyAllSeams`, `:24:28: undefined: probe.ErrSeamDenied` (×5), `:47:29: undefined: probe.TLSVerification` |
| 2 | `gofmt -l . && go vet ./... && go test -count=1 ./...` (task 2 GREEN) | **0** | `ok …/internal/probe 0.013s` · `ok …/internal/version 0.007s`; `gofmt` printed nothing |
| 3 | `go test -count=1 ./internal/probe/` (triangulation first run) | **1** | `--- FAIL: TestCommandFact…/neither_fact_is_ever_a_pass` — `CommandFact(<nil>) resolution = "unresolved", want "not_measured"` (see “Deviations” #4) |
| 4 | mutation A (task 3): the error argument examined before the nil capability | **1** | `--- FAIL: …/a_nil_runner_stays_excluded_even_when_an_error_is_handed_in` — `CommandFact("sshd -T", boom).Kind = "command_denied", want "capability_excluded"` |
| 5 | mutation B (task 3): `stepperClock.Now` without its mutex, under `-race` | **1** | `WARNING: DATA RACE` · `--- FAIL: TestStepperClockIsScriptedAndConcurrencySafe` |
| 6 | mutation C (task 3): denial classified as `ObsCapabilityExcluded` | **1** | two failing subtests — kind mismatch and `excluded and denied share the reason code "capability_excluded"` |
| 7 | guard mutations (task 4), one at a time: `var m1 = denyAllDialer{}`, `var m2 Dialer`, `var m3 = DenyAllSeams()`, `var m4 = NewStepperClock(…)` | **1** each | `seams.go: [m1] is a package-level denyAllDialer value…`, `[m2] is a package-level Dialer…`, `[m3] is a package-level DenyAllSeams value…`, `[m4] is a package-level NewStepperClock value…` |
| 8 | `go test -count=1 -run 'TestDenyAllSeams\|TestEffectiveTargets' ./internal/probe/` (task 4 gate) | **0** | `ok …/internal/probe 0.008s` |
| 9 | `gofmt -l .` · `go vet ./...` · `go test -race -count=1 ./internal/probe/` (task 4 gate) | **0** | `gofmt` empty output; `vet` no output; `ok …/internal/probe 1.053s` |
| 10 | `go test -count=1 ./...` (task 5 RED) | **1** | `undefined: probe.DeclaredTargets` (×2), `probe.ProtocolLocal`, `probe.ProtocolTCP/UDP/TLS`, `probe.ProbeDeclaration`, `probe.EffectiveTarget`, … |
| 11 | `gofmt -l . && go vet ./... && go test -count=1 ./...` (task 6 GREEN) | **0** | `ok …/internal/probe 0.011s` · `ok …/internal/version 0.007s` |
| 12 | mutation A (task 7): overrides append instead of replace | **1** | `--- FAIL: TestOverrideReplacesRatherThanAppends` — `repeating the override produced 2 targets, want 1` |
| 13 | mutation B (task 7): the hub address is never resolved | **1** | `--- FAIL: TestHubAddressBecomesTheDeclaredTarget/supplied_with_a_port` — `hub probe has 0 effective targets` |
| 14 | mutation C (task 7): the declared default port is not applied | **1** | `--- FAIL: …/egress.ssh.known=example.com` — `effective target = "example.com:0", want "example.com:22"` |
| 15 | mutation D (task 7): a colon-bearing non-IP accepted as a host | **1** | `--- FAIL: TestOverrideErrorsAreTyped/address_with_colon-bearing_host_but_no_port` — `error = <nil>, want probe: unparsable target address` |
| 16 | mutation E (task 7): a declared host literal leaked into `reason.go` | **1** | `--- FAIL: TestDeclaredHostsHaveOneHome` — `reason.go declares the target host "www.cloudflare.com"; the declared set lives only in targets.go` |
| 17 | mutation (task 8): `DeclaredTargets()` returns the package slice itself | **1** | `--- FAIL: TestDeclaredTargetsReturnsACopy` — `declaration[0] was edited through the accessor: {Probe:mutated …}` |
| 18 | `go test -count=1 -v ./internal/probe/` (final inventory) | **0** | 151 `=== RUN` lines and 151 `--- PASS` lines (15 new top-level tests, 106 of the lines are new subtests) |
| 19 | `gofmt -l .` (final gate) | **0** | empty output — nothing unformatted |
| 20 | `go vet ./...` (config `quality.typecheck`) | **0** | no output |
| 21 | `go test -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 0.014s` · `ok …/internal/version 0.007s` |
| 22 | `go test -race -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 1.053s` · `ok …/internal/version 1.019s` |

**Runtime harness**: **N/A for this slice as an end-to-end run, with the seam-level harness now in place.** The slice declares the capability set and the dialed set but nothing measures yet — no probe exists until PR 5 and no runner until PR 4 — so there is no run to execute. What *is* executable is the deny-all harness itself, and it runs: every capability of `DenyAllSeams()` is called by a test and asserted to deny (run 2), which is exactly design §6.2's first proof level and the base every later probe test starts from. PR 4 onwards owns the run-level scenarios (bounded run, streaming, cancellation).

---

## TDD Cycle Evidence (PR 3)

RED → GREEN → TRIANGULATE → REFACTOR per PR 3 task row. Every RED was produced before the production file it exercises existed, and each RED failure names the missing symbol or the missing package file rather than a wrong assertion.

| Task row | Phase | Evidence produced | Observed failure (RED) | Observed pass (GREEN) |
|---|---|---|---|---|
| 1 RED — deny-all sentinels | RED | `internal/probe/seams_test.go` written before any non-test file existed | exit 1: `undefined: probe.DenyAllSeams`, `ErrSeamDenied` (×5), `TLSVerification` — the seam vocabulary the test names did not exist | n/a |
| 2 GREEN — `seams.go` | GREEN | Eight interfaces, `Seams`, `ErrSeamDenied`/`ErrTLSVerification`, `DenyAllSeams()`, `NewStepperClock()` | (previous row) | exit 0: eight capability subtests PASS (dial, lookup, TLS verification, packet dial, command, filesystem, platform, clock); `gofmt` empty |
| 3 TRIANGULATE — nil vs deny-all | TRIANGULATE | `TestCommandFactDistinguishesMissingCapabilityFromDenial`, `TestDenyAllSeamsNeverReportsARejectedChain`, `TestDenyAllSeamsIsFreshPerCall`, `TestStepperClockIsScriptedAndConcurrencySafe` | One genuine RED: run 3 failed on the first draft of the “never a pass” assertion (`CommandFact(<nil>) resolution = "unresolved", want "not_measured"`), which exposed that a *ran* command has no denial fact and falls to the table's total row; the assertion was narrowed to the two facts that exist (recorded as deviation #4). Teeth proven by mutations A–C (runs 4–6, each exit 1, one of them a real `DATA RACE`) | exit 0 after the fix; suite restored green after every mutation |
| 4 REFACTOR + GATE — no package-level seam var | REFACTOR | `TestNoPackageLevelSeamVariable` + `seamVocabulary`/`declaredName`/`valueName`; doc pass on `seams.go`, no code restructured | The guard's first draft missed an inferred-type global (`var m1 = denyAllDialer{}`) — mutation run 7 caught the gap, so the forbidden vocabulary is now read from `seams.go` itself (types **and** functions) and all four patterns (interface, implementation, `Seams` constructor, clock constructor) fail | exit 0: focused run, `gofmt -l .` (empty), `go vet ./...`, `go test -race ./internal/probe/` |
| 5 RED — declared target set | RED | `targets_test.go` written before `targets.go` | exit 1: `undefined: probe.DeclaredTargets`, `ProtocolLocal`, `ProtocolTCP/UDP/TLS`, `ProbeDeclaration`, `EffectiveTarget`, … | n/a |
| 6 GREEN — `targets.go` | GREEN | Declaration of ten probes, `EffectiveTargets`, `ParseTargetOverride`, `parseAddress`, four typed errors | (previous row) | exit 0: `gofmt -l .` (empty), `go vet ./...`, `go test -count=1 ./...` |
| 7 TRIANGULATE — closed set | TRIANGULATE | `TestEffectiveSetIsClosedAfterAnOverride` (reachability from the declaration, one override per probe, nothing escaping the declaration) | Cases pass against the GREEN implementation; teeth proven by mutations A–E (runs 12–16, each exit 1) | exit 0 after each restore |
| 8 REFACTOR + GATE — final gates | REFACTOR | Review added address validation for hand-built overrides plus its cases, added `TestDeclaredTargetsReturnsACopy`, and removed one duplicate base-case test (deviation #6); documents re-read against the code, no code restructured | The new copy test was mutation-checked (run 17, exit 1) rather than trusted | exit 0: `gofmt -l .` (empty), `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...` |

**Triangulation depth.** Six independent angles on the seam set (each capability's sentinel; nil-versus-denied as two different facts and two different reason codes; a denied handshake never reading as a rejected chain; a global-free seam set proven by AST; a fresh value per call; a scripted clock that is deterministic, monotonic and concurrency-safe) and four on the declared set (the declaration read back probe by probe; every declared triple reachable from the declaration; replacement rather than appending, including a repeated flag and a hub-then-override sequence; no effective triple outside the declaration or an override). **Twelve mutations, every one caught**, including one real `DATA RACE` and one genuine RED that changed the assertion rather than the code.

---

## Deviations from design (PR 3)

| # | Deviation | Why | Design reference | Follow-up owner |
|---|---|---|---|---|
| 1 | The deny-all sentinel is exported as `ErrSeamDenied`; design §6.2 calls it `errNetworkForbiddenInTest`. | The seam set is used by tests in other packages (the doctor and command-surface suites assert the dialed set and the zero-exec counter), and an unexported sentinel cannot be asserted from there with `errors.Is`. The design's name describes the same sentinel; only its visibility changed. | design §6.2 | PR 18/19 suites consume it as `probe.ErrSeamDenied` |
| 2 | `TLSVerifier.Verify` returns `(TLSVerification, error)` instead of design §6.1's `(Observation, error)`, and a rejected chain is signalled by an error wrapping the exported `ErrTLSVerification`. | Returning an `Observation` would force the seam to choose a reason code (and, for `tls.truststore`, to guess which of two purposes it is answering), contradicting §5.1's rule that `classify.go` is the only place a code is chosen. The seam now reports the chain's issuer and verification code verbatim and how the attempt ended; the probe maps that to an observable and the table picks the code. | design §6.1 (seam table), design §5.1 | PR 8 (`tls.interception`) and PR 9 (`tls.truststore`) consume it; they must name the observable that matches their declared purpose |
| 3 | `Seams.CommandFact(command, err)` is added; design §6.1–6.2 names no such helper. | The nil-versus-explicit distinction the triangulation row demands must be *observable*, not merely inspectable: a nil `CommandRunner` has to read as `ObsCapabilityExcluded` and a denying runner as `ObsCommandDenied`. One small documented function keeps that mapping in the seam file instead of duplicating it in every probe that needs a command. | design §5.1 obligation 2, design §6.1–6.2 | PR 6 (`local.sshd`) is its first consumer |
| 4 | A `CommandFact` call with an injected runner and no error returns the zero `RawObservation` (no denial fact) rather than a not-measured fact. | The command *ran*; whether its output means anything is the probe's judgement, not the seam's. The first draft of the triangulation asserted not-measured for all four combinations, and the RED (run 3) showed the claim was wrong; the assertion was narrowed to the two denial facts, and “never a pass” is still asserted across all four. | design §5.1 (total row) | PR 6 |
| 5 | The declared set declares **all ten** probes, with `ProtocolLocal` and no endpoints for `local.env`/`local.sshd` and `ProtocolTCP` with no host for `egress.hub.direct`. | “The declared set lives in one place” needs one place that knows every probe *name*, otherwise an override naming a local probe could not be refused and the probe-name set would live in the registry (PR 5) instead. The hub's host is run input rather than a constant (design D3), so its entry declares the protocol and the documented default port only. | design D3, design D10, design §7 (`targets.go`) | PR 5's registry must enumerate the same ten names in the same order; PR 18's flag layer maps the four typed errors to exit code 2 |
| 6 | `parseAddress` rejects a colon-bearing value that is not an IP literal (`not:an:address`) instead of treating it as a host-only value, and `EffectiveTargets` validates a hand-built override's host and port range. | Design D3's host-only fallback exists for `[::1]`-style values; a host name cannot contain a colon, so accepting one would declare a target that can never resolve — a measurement of nothing dressed as one. The extra validation covers a caller that builds a `TargetOverride` without the parser. | design D3 (parse rule), design D10 | PR 18 (flags) reports both as usage errors |
| 7 | One duplicate test was removed during the row-8 REFACTOR: the no-input base case in `targets_test.go` was fully re-asserted by the closed-set triangulation (which additionally counts reachable triples) and by the hub test's “no hub supplied” subtest. | Removing a duplicate assertion is deduplication, not coverage reduction: both requirements stay covered by the remaining tests, and no other case, control case, comment or blank line was removed anywhere in this slice. | tasks PR 3 rows 5 and 7 | n/a |
| 8 | Both test files are **external** test packages (`package probe_test`) importing the module path, as in PR 1 and PR 2. | It proves the seams and the declaration are usable from outside the package — which is what the doctor suite, the flag layer and the payload mapper will be — and it keeps the guards (AST, one-home) reading the package as a consumer would. The AST guard reads the sources as files, so it needs no privileged access. | tasks PR 3 file list; design §8 test plan | n/a |

---

## Workload and PR boundary (PR 3)

| Field | Value |
|---|---|
| Slice | PR 3 of 20 — “Seams, the deny-all test default, and the declared target set” (WU4 + WU5) |
| PR 3 estimate in `tasks.md` | 430–660 lines (point ≈545) |
| Host attempt ceiling | 1,700 counted changed lines |
| **Actual authored code + tests** | **1,734 lines** (0 deletions; all four files are new) — 578 production, 853 test, 236 comment, 106 blank |
| Artifact changes | `tasks.md` 8 lines changed (8 + 8 = **16** changed lines) + this appended section |
| **Total counted changed lines for the work unit** | **≈1,950** including this artifact section |
| Chain per-PR cohesion ceiling | 600 changed lines (user-approved) |
| Review budget (session canonical) | 400 changed lines |
| Budget status | **Over the slice estimate, over the 600-line chain ceiling, over the 1,700-line attempt ceiling and over the 400-line session budget** |
| PR boundary | Starts at PR 2's branch state (`internal/probe` vocabulary + classification table) and ends at `internal/probe` compiling and passing with the seam set, the deny-all default and the declared target set in place. `internal/probe/runner.go` (PR 4), `registry.go`/`local.go` (PR 5) and every later-slice file are **not** started |
| Rollback boundary | Delete the four files under `internal/probe/` (`seams.go`, `seams_test.go`, `targets.go`, `targets_test.go`); the module returns to its PR 2 state with the vocabulary, the reason-code set and the classification table still green. `tasks.md` lines 164–171 revert to `- [ ]`; this appended section is the only other PR 3 change |
| Rollback independence | Nothing consumes the seams or the target set yet — the runner (PR 4) is the first consumer — so the revert removes no unrelated work and leaves PR 1 and PR 2 green |

**Why 1,734 > 1,700, stated honestly.** The two files that make the slice reviewable are 998 of the 1,734 lines (407 + 591), and their size is set by the task rows: eight capability subtests, the nil-versus-denied triangulation with its property cases, the AST guard and its four-pattern vocabulary, fifteen target tests including three override error families, the hub resolution matrix and the closed-set triangulation. Another 236 lines are comments — 173 in `seams.go`, much of it the per-seam rationale for *why* each seam exists and why a global would be wrong, which is the point design §6.1 makes. The review-budget rule forbids reaching a number by deleting tests, control cases, triangulation cases, comments, docs or blank lines, and no `size:exception` was assumed, so **no code-golf pass was attempted**. The only reduction taken was the removal of one genuinely duplicated assertion (deviation #7).

**Two remedies exist, both owned by the orchestrator, and neither was taken unilaterally.** (a) Record `size:exception` for the attempt ceiling and the chain ceiling, accepting PR 3 as one cohesive reviewable unit. (b) Apply the second-split boundary `tasks.md` already names for this group: **PR 3a = WU4**, the seams and `DenyAllSeams` (`seams.go` 354 + `seams_test.go` 407 = **761** authored lines, above the 600-line chain ceiling but well inside the 1,700-line attempt ceiling); **PR 3b = WU5**, the declared target set and overrides (`targets.go` 382 + `targets_test.go` 591 = **973** authored lines). The split is strictly ordered — `targets.go` does not import the seams and nothing else consumes either yet, so both halves compile on the PR 2 state — and PR 3b would target PR 3a's branch. **This executor did not split the slice or touch branches**: the parent assigned PR 3 as one unit.

---

## Remaining unchecked tasks (PR 3 view)

**Inside this slice: none.** All eight PR 3 checkbox rows are `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (re-read after the edits: **24 checked, 141 unchecked**; 165 rows total). The ownership markers were re-checked: 165 `<!-- sdd-owner: implementation -->` markers for 165 rows, every one terminal.

The change-wide remainder is **141 checkbox lines in PR 4 – PR 20**, which belong to later chained slices on later branches (PR 3 targets PR 2's branch; the chain is `stacked-to-main`) and are **not** part of this work unit. The next unchecked line in the artifact, verbatim, is `tasks.md:178`:

```
- [ ] **RED** — write `runner_test.go` with a counting seam proving the concurrency ceiling: never more than `Options.Concurrency` probes in flight, the default is 4, and a test-only elevated value is honoured. Prove the failure first. <!-- sdd-owner: implementation -->
```

---

## Risks (PR 3)

| # | Risk | Status / handling |
|---|---|---|
| 1 | 1,734 authored lines against a 1,700-line attempt ceiling, a 600-line chain ceiling and a 400-line session budget | Disclosed above with the composition and the reason it cannot shrink without deleting tests, cases or comments. Recommend the maintainer either acknowledge `size:exception` or apply the named WU4/WU5 second split (761 / 973 authored lines) |
| 2 | The seam shapes are consumed by five later slices (PR 4, 5, 6, 7, 8, 9) | Each interface is minimal and documented against design §6.1, and deviations #1–#3 record the two deliberate shape changes. A later slice that needs a different shape must record it as a deviation rather than widening `Seams` silently |
| 3 | `TLSVerifier`'s error-based signal (`errors.Is(err, ErrTLSVerification)`) is untested against a real handshake | Expected: no TLS probe exists yet. PR 8 must prove that a rejected chain is measured/fail while any other handshake error is unresolved, with a scripted verifier — the two must not collapse |
| 4 | `CommandFact` maps *any* runner error to `command_denied` | Correct for this slice because the production command runner is nil (design §6.2) and only test runners exist; when PR 6 needs to separate "denied" from "ran and failed", that distinction belongs in the probe's observable choice, not in this helper |
| 5 | The AST guard only forbids the vocabulary declared in `seams.go` | Stated in the test's own coverage note: a seam implementation declared in `real.go` (PR 19) is not in the vocabulary, and a var declared as `any` and filled with a dialer would evade it. PR 19's static construction guard is the complementary check |
| 6 | `parseAddress` accepts unbracketed IPv6 host-only values (`::1`) as design D3 requires, while rejecting unbracketed colon-bearing non-IP values | Covered by tests in both directions (`[2001:db8::1]` resolves with the default port; `not:an:address` is refused). PR 18's flag layer inherits the same behaviour for `--hub` |
| 7 | The declaration duplicates each probe's port twice (its `Targets` and its `DefaultPort`) for the probes that declare endpoints | Deliberate and asserted: the default port is the value an override without one resolves to, and `TestDeclarationHoldsTheTenProbes` fails if a probe's default port is absent or nonsense. A future edit that moves one without the other is caught by that test |

### Engram mirror note

This file is the **authoritative** artefact. Its Engram mirror is split because the merged text exceeds the store's content limit: part 1 (PR 1, PR 2 and the structured-status and completion tables that precede this section) is saved under the topic `sdd/reach-diagnosis-core/apply-progress`, and part 2 (this PR 3 section) under `sdd/reach-diagnosis-core/apply-progress/part2`. Read the repository file for the complete, current record.

---

# Apply Progress — PR 4 of 20 — WU6 + WU7 · Runner: bounds, streaming, run budget, cancellation and the three timeout paths

**Change**: `reach-diagnosis-core` · **Slice**: **PR 4 of 20** — “Runner: bounds, streaming, run budget, cancellation and the three timeout paths” (WU6 + WU7)
**Branch**: `feat/probe-runner`, stacked on PR 3's branch `feat/probe-targets` (chain strategy `stacked-to-main`, so PR 4 targets PR 3's branch and **not** `main`)
**Date**: 2026-09-14 · **Artifact store**: `both` (this file + Engram mirror `sdd/reach-diagnosis-core/apply-progress`, split as the mirror note at the end records)
**Strict TDD**: active — `openspec/config.yaml` declares `strict_tdd: true` with runner `go test ./...`; RED → GREEN → TRIANGULATE → REFACTOR followed for every row of this slice
**Skill resolution**: `paths-injected` — read `/home/luisalt20/.config/opencode/skills/go-testing/SKILL.md` and `/home/luisalt20/.config/opencode/skills/work-unit-commits/SKILL.md` before writing code; no registry fallback was needed
**Delivery path consumed**: `auto-chain` / `stacked-to-main` — this run implements **only** the assigned slice and stops at its PR boundary; the PR 1, PR 2 and PR 3 entries above are preserved unchanged
**Commit status**: nothing committed, staged, pushed or branched by this phase; the work is left in the working tree for the orchestrator
**How to read this file**: PR 4's entry is the section below. The current change-wide remainder is restated at the end of this section.

---

## Structured status consumed (PR 4)

| Field | Value |
|---|---|
| `schemaName` / `schemaVersion` | `gentle-ai.sdd-status` / `2` |
| `changeName` | `reach-diagnosis-core` |
| `nextRecommended` | `apply` |
| `applyState` | `ready` |
| `dependencies.apply` / `.verify` / `.archive` | `ready` / `blocked` / `blocked` |
| `actionContext.mode` | `repo-local` |
| `actionContext.workspaceRoot` | `/home/luisalt20/projects/close/herdr-reach` |
| `actionContext.allowedEditRoots` | `["/home/luisalt20/projects/close/herdr-reach"]` — every file written lives inside it |
| `artifactStore` | `both` declared by the parent prompt (native `openspec`); files written under `openspec/changes/reach-diagnosis-core/` and mirrored to Engram |
| `taskProgress` before this run | 165 total / 24 completed / 141 pending (PR 1 + PR 2 + PR 3) |
| `taskProgress` after this run | 165 total / **34 completed** / 131 pending |
| `actionContext` warnings | none |
| Work-unit ownership markers | all ten PR 4 rows carry the terminal `<!-- sdd-owner: implementation -->` marker; the PR 4 block contains no malformed, duplicate, unsupported or non-terminal marker |
| `applyState: all_done`? | no — implementation continues, so editing was permitted |

**Attempt context**: the harness reports one bounded attempt already active for this work unit (`token sha256:d358dc6b…f25257`) with a **2000**-changed-line ceiling. Per the parent prompt the parent owns `sdd-attempt acquire`/`settle`; this executor did **not** call the native attempt command. The counted-line position against that ceiling is in “Workload and PR boundary”.

---

## Completed tasks and their persisted checkbox updates (PR 4)

All ten PR 4 rows were flipped from `- [ ]` to `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (lines 178–187) as the work completed, then re-read to confirm. `git diff` on that file reports exactly `10 insertions(+), 10 deletions(-)` — ten `- [ ]` → `- [x]` flips and nothing else.

| # | Task (short) | Persisted update | Evidence |
|---|---|---|---|
| 1 | RED — `runner_test.go` with a counting fixture proving the ceiling; prove the failure | `tasks.md:178` → `- [x]` | `go test ./...` → `FAIL … [build failed]`: `undefined: probe.Options` (four sites) and `undefined: probe.NewRunner` — the runner vocabulary did not exist |
| 2 | GREEN — bounded concurrency plus streaming with `Options{Concurrency, ProbeTimeout, RunBudget, Clock}` at construction | `tasks.md:179` → `- [x]` | `TestRunnerHonoursConcurrencyCeiling` PASS with three subtests: default ceiling 4, elevated 6 honoured, serial 1 honoured; peak counted inside the fixtures always equalled the effective ceiling |
| 3 | TRIANGULATE — hanging probe ⇒ runner-produced `(unresolved, probe_timeout)`, run inside budget, every other probe reported | `tasks.md:180` → `- [x]` | `TestHangingProbeIsAbandonedByTheRunner` PASS (run 28 ms inside its 2 s budget, `probe_timeout`, start count 1, the fast pass and the refused fail both survive); teeth proven by mutation A |
| 4 | TRIANGULATE — streaming: first result before the slowest probe finishes | `tasks.md:181` → `- [x]` | `TestRunnerStreamsResultsBeforeTheSlowestProbeFinishes` PASS (the fast result arrives while the gated slow probe is provably still measuring; nothing else streams until it is released) plus `TestRunnerReturnsEveryProbeOnceInProbeOrder` (6 probes answering in reverse order) |
| 5 | REFACTOR + RACE — one deadline/abandon implementation for both bound paths; hand `-race ./internal/probe/` | `tasks.md:182` → `- [x]` | `abandonment` + `abandon()` + `runEndedCause()` extracted so the probe bound and the run-level reasons share one classifier; `gofmt -l .` exit 0 (empty) · `go vet ./...` exit 0 · `go test -count=1 ./internal/probe/` exit 0 · `go test -race -count=1 ./internal/probe/` exit 0 (1.307 s) |
| 6 | RED — bounded run with millisecond values | `tasks.md:183` → `- [x]` | `go test -run 'TestRunBudget' ./internal/probe/` → exit 1: `the run took 321.238916ms, want it bounded by its 40ms budget instead of a full wave of 80ms probes` — a genuine RED: the budget was accepted but not yet enforced |
| 7 | GREEN — run budget in `runner.go`, `run_budget_exceeded` at the runner layer only | `tasks.md:184` → `- [x]` | `TestRunBudgetBoundsASlowSuite` PASS: 8 probes slower than the budget, every probe bounded with `(unresolved, run_budget_exceeded)`, run returned in ~40 ms of its 40 ms budget; focused runner suite exit 0 |
| 8 | TRIANGULATE — cancellation returns promptly, affected probes `(unresolved, run_cancelled)`, never a pass | `tasks.md:185` → `- [x]` | Written as its own RED→GREEN inside the row (see “TDD Cycle Evidence”): RED exit 1 (`reason = "probe_timeout", want "run_cancelled"`), then `TestCancelStopsInFlightMeasurements` PASS with the measured pre-cancellation pass preserved, a fabricated post-cancellation pass refused, and the three unstarted probes never started |
| 9 | TRIANGULATE — `TestTimeoutPathsAreDistinct`: three distinct `(resolution, reason)` pairs, no fourth collapsing pair | `tasks.md:186` → `- [x]` | PASS: `(measured, budget_expired)` / `(unresolved, probe_timeout)` / `(unresolved, udp_silence)`, pairwise distinct, verdicts `fail` / `indeterminate` / `indeterminate`, none a pass; teeth proven by mutation B |
| 10 | REFACTOR + RACE — `gofmt`, `go vet`, `go test ./...`, hand `go test -race ./...` | `tasks.md:187` → `- [x]` | `gofmt -l .` exit 0 (empty output) · `go vet ./...` exit 0 · `go test -count=1 ./...` exit 0 · `go test -race -count=1 ./...` exit 0 |

---

## Files changed (PR 4 file list only)

| Path | Status | Authored lines | Purpose |
|---|---|---|---|
| `internal/probe/runner.go` | added | 352 (188 production code, 131 comment, 33 blank) | `Options` with documented defaults, `NewRunner`, `Runner.Options`, `Run` (bounded concurrency, per-probe bound, run budget, cancellation, streaming `emit`, ordered results), `awaitProbe`, `runEndedCause`, `abandonment` with its three builders, `abandon`, `fillIdentity`, `wallClock` |
| `internal/probe/runner_test.go` | added | 673 (527 test code, 89 comment, 57 blank) | Seven test functions and the fixture vocabulary: `runnerFixture` with an attempt counter, `runnerGate` (peak in-flight counter), classification-table-built fixture results, and one case per task row |
| `openspec/changes/reach-diagnosis-core/tasks.md` | modified | 10 lines changed in place (10 deletions + 10 additions = 20 changed lines) | Ten PR 4 checkboxes `- [ ]` → `- [x]` |
| `openspec/changes/reach-diagnosis-core/apply-progress.md` | modified | this appended section | Cumulative PR 4 evidence; the PR 1, PR 2 and PR 3 sections are untouched |

**Authored additions for the code slice: 1,025 lines** (352 + 673), all additions, zero deletions — both files are new. No file outside the two PR 4 paths and the two artifact files was created or modified; `README.md`, `PRD.md`, the proposal, the specs, the design, `explore.md`, `research.md`, `preproposal.md` and `openspec/config.yaml` are untouched. No new dependency was added (only `context`, `sync/atomic`, `time` and the standard test packages); no linter or CI configuration was introduced.

---

## Test commands run — exact commands and exit status (PR 4)

| # | Command | Exit | Observed output (abridged) |
|---|---|---|---|
| 1 | `go test ./...` (task 1 RED) | non-zero — `FAIL … [build failed]` | `undefined: probe.Options` at `runner_test.go:126/129/130/131`, `undefined: probe.NewRunner` at `:142`. (The pipeline also ran `head`, so the printed shell status was `head`'s; the `FAIL` line is `go test`'s own verdict, which is non-zero on a build failure.) |
| 2 | `gofmt -l . && go vet ./... && go test -count=1 -run 'TestRunnerHonoursConcurrencyCeiling' -v ./internal/probe/` (task 2 GREEN) | **0** | `--- PASS: TestRunnerHonoursConcurrencyCeiling` plus the three subtests; `gofmt` printed nothing; `vet` printed nothing |
| 3 | `go test -count=1 -run 'TestRunner\|TestHanging' ./internal/probe/` (tasks 3+4) | **0** | correct alternation form is `'TestRunner\|TestHanging'` in the shell with a Go regexp `|` (see the note below); `TestHangingProbeIsAbandonedByTheRunner`, `TestRunnerStreamsResultsBeforeTheSlowestProbeFinishes`, `TestRunnerReturnsEveryProbeOnceInProbeOrder` all PASS |
| 4 | mutation A (task 3): the abandoned observable changed from `ObsProbeIgnoredBudget` to `ObsTCPEstablished` | **1** | `--- FAIL: TestHangingProbeIsAbandonedByTheRunner` — `hanging reason = "internal_error", want "probe_timeout": the runner classifies the probe it abandoned` (the unattributed observable falls to the table's total row). Restored from `/tmp/runner.go.bak`; rerun exit 0 |
| 5 | `gofmt -l .` · `go vet ./...` · `go test -count=1 ./internal/probe/` (task 5 gate) | **0** each | `gofmt` empty output; `vet` no output; `ok …/internal/probe 0.262s` |
| 6 | `go test -race -count=1 ./internal/probe/` (task 5 hand race run) | **0** | `ok …/internal/probe 1.307s` |
| 7 | `go test -count=1 -run 'TestRunBudget' ./internal/probe/` (task 6 RED) | **1** | `the run took 321.238916ms, want it bounded by its 40ms budget instead of a full wave of 80ms probes` |
| 8 | `gofmt -l . && go vet ./... && go test -count=1 -run 'TestRunner|TestHanging|TestRunBudget' ./internal/probe/` (task 7 GREEN) | **0** | `ok …/internal/probe 0.298s` |
| 9 | `go test -count=1 -run 'TestCancel' ./internal/probe/` (task 8 RED, inside the row) | **1** | `cancelled result "fixture.honours-cancel" reason = "probe_timeout", want "run_cancelled"` |
| 10 | `gofmt -l . && go vet ./... && go test -count=1 -run 'TestRunner|TestHanging|TestRunBudget|TestCancel' ./internal/probe/` (task 8 GREEN) | **0** | `ok …/internal/probe 0.297s` |
| 11 | `go test -count=1 -run 'TestTimeoutPaths' -v ./internal/probe/` (task 9) | **0** | `--- PASS: TestTimeoutPathsAreDistinct (0.10s)` |
| 12 | mutation B (task 9): the abandoned observable changed from `ObsProbeIgnoredBudget` to `ObsUDPSilence` | **1** | `--- FAIL: TestTimeoutPathsAreDistinct` — `a probe that ignores its bound is the runner's unresolved timeout: reason = "internal_error", want "probe_timeout"`. Restored; rerun exit 0 |
| 13 | `gofmt -l .` (task 10 gate) | **0** | empty output — nothing unformatted |
| 14 | `go vet ./...` (config `quality.typecheck`) | **0** | no output |
| 15 | `go test -count=1 ./...` (task 10 gate) | **0** | `ok …/internal/probe 0.404s` · `ok …/internal/version 0.007s` |
| 16 | `go test -race -count=1 ./...` (task 10 gate) | **0** | `ok …/internal/probe 1.452s` · `ok …/internal/version 1.018s` |
| 17 | `go test -count=1 -run 'TestRunner|TestHanging|TestRunBudget|TestCancel|TestTimeoutPaths' -v ./internal/probe/` (the work-unit map's `Unit verification` command) | **0** | seven top-level functions PASS (`TestRunnerHonoursConcurrencyCeiling` with 3 subtests, `TestHangingProbeIsAbandonedByTheRunner`, `TestRunnerStreamsResultsBeforeTheSlowestProbeFinishes`, `TestRunnerReturnsEveryProbeOnceInProbeOrder`, `TestRunBudgetBoundsASlowSuite`, `TestCancelStopsInFlightMeasurements`, `TestTimeoutPathsAreDistinct`) |
| 18 | `go test -count=1 -v ./internal/probe/` (final inventory) | **0** | 161 `--- PASS` lines across the whole package |

**Note on the map's focused command.** `tasks.md`'s rows and the work-unit map write the filter as `-run 'TestRunner\|TestHanging\|…'`. Go's regexp engine (RE2) treats `\|` as a **literal pipe**, so that exact string matches no test — run 17 reproduced it (`ok … [no tests to run]`, exit 0) before the working form with bare `|` was used. This is an artifact-typo observation, not a code defect; the ten checkboxes are unaffected. Any later slice quoting the same filter should use `|`.

**Runtime harness**: **partly available and exercised in-process, with no end-to-end run.** The slice executes a real concurrent scheduler with real timers, so the harness *is* the runner itself: runs 3, 7, 9, 11 and 17 drive it with millisecond bounds, a gated fixture, a hanging fixture and a cancelled context, and the race gate (run 16) runs the same scheduler under the detector. There is still no end-to-end command: no CLI (`cmd/herdr-reach` lands in PR 18/19), no registry (PR 5) and no real probe (PR 5–PR 9), so nothing can be measured against a network yet. `DenyAllSeams()` is not invoked by the runner — probes bring their own seams — so this slice's boundary is exactly “the scheduler is proven, the suite it will run does not exist yet”.

---

## TDD Cycle Evidence (PR 4)

RED → GREEN → TRIANGULATE → REFACTOR per PR 4 task row. The first RED was produced before `runner.go` existed at all (`undefined: probe.Options`, `probe.NewRunner`); the two later REDs are genuine behaviour failures of the code as it stood at that moment, not compile errors.

| Task row | Phase | Evidence produced | Observed failure (RED) | Observed pass (GREEN) |
|---|---|---|---|---|
| 1 RED — concurrency ceiling | RED | `runner_test.go` written with the fixture vocabulary (`runnerFixture`, `runnerGate`) and `TestRunnerHonoursConcurrencyCeiling` before any `runner.go` | exit non-zero: `undefined: probe.Options` (×4), `undefined: probe.NewRunner` / `FAIL … [build failed]` | n/a |
| 2 GREEN — bounded concurrency + streaming | GREEN | `runner.go`: `Options`/`withDefaults`/`NewRunner`/`Runner.Options`, `Run` with the ceiling, per-probe `probeCtx` bound, `emit`, ordered results, `awaitProbe`, `abandon` | (previous row) | exit 0: ceiling test PASS; peak in-flight counted by the fixtures equalled the effective ceiling in all three subtests |
| 3 TRIANGULATE — hanging probe | TRIANGULATE | `TestHangingProbeIsAbandonedByTheRunner` (hanging + fast pass + refused fail) and `runnerGate`-based negative checks | Case passes against the GREEN implementation; teeth proven by mutation A (`ObsTCPEstablished` for the abandoned fact) → exit 1 | exit 0 after restore |
| 4 TRIANGULATE — streaming | TRIANGULATE | `TestRunnerStreamsResultsBeforeTheSlowestProbeFinishes` (gated slow fixture, streamed `emit`, negative “nothing else streamed yet” assertion, probe-order check) and `TestRunnerReturnsEveryProbeOnceInProbeOrder` | Cases pass against the GREEN implementation; the negative assertion is what would fail if `Run` buffered the suite (no result until the slow fixture is released) | exit 0 |
| 5 REFACTOR + RACE — one abandon path | REFACTOR | `abandonment` struct plus `probeBoundExceeded`/`abandon`/`runEndedCause`; the duplicated result-building removed, doc comments rewritten, no behaviour change; hand race run recorded | n/a — refactor only | exit 0: focused + full package tests, `gofmt` (empty), `vet`, `-race ./internal/probe/` (1.307 s) |
| 6 RED — bounded run | RED | `TestRunBudgetBoundsASlowSuite`: 8 fixtures slower (80 ms) than the injected 40 ms run budget with a 300 ms probe bound | exit 1: `the run took 321.238916ms` — four sequential waves of slow probes, no budget enforcement, results carried the probe's own reason | n/a |
| 7 GREEN — run budget | GREEN | `Run`: `time.NewTimer(opts.RunBudget)`, `budgetExceeded atomic.Bool`, `cancelRun`, nil-ed timer channel, the unstarted remainder reported through the same `abandon` path, and the run-ended override in `awaitProbe` | (previous row) | exit 0: every probe bounded with `(unresolved, run_budget_exceeded)`, run ended in ~40 ms of its 40 ms budget |
| 8 TRIANGULATE — cancellation | TRIANGULATE (own RED→GREEN) | `TestCancelStopsInFlightMeasurements` written **first**: an early measured pass streamed before cancellation, two in-flight fixtures (one honest failure, one fabricated pass), three never-started fixtures | exit 1: `cancelled result "fixture.honours-cancel" reason = "probe_timeout", want "run_cancelled"` — the row's own RED, taken before the cancellation code existed | exit 0 after adding `runCancelled`, the `runCtx.Err()` cause branch, the `ctxDone` select branch and the “never start against a dead context” guard |
| 9 TRIANGULATE — three timeout paths | TRIANGULATE | `TestTimeoutPathsAreDistinct`: probe-own-budget fixture (5 ms), bound-ignoring fixture, UDP-silence fixture, pairwise-distinct pair assertion, verdict assertions | Case passes against the GREEN implementation; teeth proven by mutation B (abandoned fact as `ObsUDPSilence`) → exit 1 | exit 0 after restore |
| 10 REFACTOR + RACE — final gates | REFACTOR | `abandon` doc corrected for the three causes, runner observation label widened to `"runner"`, `Run`'s doc comment gained the explicit ceiling boundary; no behaviour change | n/a — refactor only | exit 0: `gofmt -l .` (empty), `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...` |

**Strict-TDD row interleaving, disclosed.** Task row 7's text names both `run_budget_exceeded` and `run_cancelled`, but writing the cancellation code before the cancellation test would have been production code without a failing test. The rows were therefore interleaved: row 7 delivered the run budget (RED in row 6), and row 8's cancellation test was written first and observed failing before cancellation was implemented. Both rows end complete; no row's deliverable is missing.

**Triangulation depth.** Four independent angles on admission and delivery (the ceiling counted inside the fixtures at three ceilings; probe order preserved against reverse completion order; streaming proven positively and negatively; the run bounded by wall time rather than by probe count) and four on failure handling (a probe that never returns; a run budget shorter than every probe; cancellation with a measured result preserved and a fabricated pass refused; three timeout paths that must stay three). **Two mutations, each caught**, plus **two genuine behaviour REDs** (budget and cancellation) that failed for their own reason rather than for a missing symbol.

---

## Deviations from design (PR 4)

| # | Deviation | Why | Design reference | Follow-up owner |
|---|---|---|---|---|
| 1 | `Options.ProbeTimeout` defaults to **10 s** and is settable; the design names no default for the per-probe bound. | R-HR-NF-09 requires every probe to be bounded, so a default is mandatory and there is deliberately no “unbounded” value (a non-positive duration resolves to the default). Ten seconds is a judgment call with stated arithmetic: with a ceiling of four, ten probes drain in three waves, worst case 30 s inside the 60 s run budget. | design §7 (`runner.go` row: “per-probe bound”), R-HR-NF-09 | `sdd-verify` may adjudicate; PR 18's doctor wiring consumes it |
| 2 | A runner-classified observation carries an **empty `Target`**, and its label is `"runner"`. | `Probe` declares no `Target()`: a probe's address lives behind its own constructor, and the hub probe's address is run input. The runner therefore cannot name a target it never knew, and inventing one would attribute a timeout to an address that may not be the one measured. `probe.go`'s `Observation` doc says an empty target happens “only for a not-measured observation”; that sentence is now narrower than the truth, and `probe.go` is outside this slice's file list, so the discrepancy is recorded here instead of edited there. | design §3.1 (`Observation.Target`), design §7 (`runner.go` row) | PR 16's payload mapper can fill the declared target from `EffectiveTargets` for these results; PR 18 decides whether it does |
| 3 | A probe that returns **after the run ended** (budget or cancellation) has its result replaced by the runner's reason, even when it returned a measured pass. | The run stopped collecting answers; accepting a late pass would let a probe that woke on cancellation decide the run's result, which the spec forbids (“each affected probe reports the cancelled outcome rather than a fabricated pass”, R-HR-NF-09). `TestCancelStopsInFlightMeasurements` asserts exactly this with a deliberately fabricating fixture. | design §5.1 (obligation 3), R-HR-NF-09 | n/a — asserted by test |
| 4 | The concurrency ceiling bounds the probes the runner has **started and still awaits**, not the goroutines in the process. | A probe that ignores its bound is abandoned rather than waited for (the whole point of R-HR-NF-02), and Go cannot kill its goroutine, so it can outlive the slot it held. Stating the ceiling as “what the runner admits” is the honest reading; a ceiling over live goroutines would require waiting for a hung probe, which is the failure the requirement exists to prevent. Documented on `Run`. | design D9, R-HR-NF-02 | PR 19's no-real-network guard and PR 5+ probes must not rely on a hung goroutine being collected |
| 5 | `Runner.Options()` is added to read the **effective** bounds back. | The default must be assertable (the row demands “the default is 4”), and the payload's `run.concurrency` / `run.run_budget_ms` need one home for what the run actually used rather than a second copy of the default constants. | design §3.3 (`run.concurrency`, `run_budget_ms`), design D9 | PR 16 reads it when building the payload |
| 6 | `Run(ctx, probes, emit)` — the runner receives **no targets and no seams**. | `probe.go` already documents that a probe's targets and seams are supplied to its constructor, so `Run` needs nothing but its context; design §4's data-flow sketch shows `Run(ctx, probes, targets, seams)`, and this is the refinement of that sketch rather than a different contract. It also keeps the fixture probes of `runner_test.go` free of seam plumbing. | design §4 (data flow), `internal/probe/probe.go` (`Probe.Run` doc) | PR 18's doctor wiring passes probes that already carry their own seams |
| 7 | A `wallClock` fallback exists for a nil `Options.Clock`. | Production injects the run's own seam clock (design §3.3: one clock moves every timestamp); the fallback only keeps a zero `Options` value usable and an elapsed value from staying unset. No test relies on it, and `DenyAllSeams()`'s stepper clock is what the deterministic cases inject. | design §3.3, design §6.2 | PR 18 must inject `seams.Clock` so the fallback is never the production path |
| 8 | `Run` returns `nil` for an empty probe list. | Not documented anywhere; an empty suite has no results and `nil` is the honest answer. No doctor call path can reach it with the ten-probe registry. | design §4, design §7 | n/a |
| 9 | Task rows 7 and 8 were **interleaved** so each behaviour had its own failing test (see “Strict-TDD row interleaving”). | Strict TDD forbids writing cancellation before a failing test for it; delivering the whole row 7 text first would have done exactly that. | strict-TDD gate, `openspec/config.yaml` (`strict_tdd: true`) | n/a — disclosed and complete |
| 10 | The runner does **not** convert a panicking probe into a result. | No task row requires it. A panic in a real probe would currently take the process down rather than become `internal_error`; adding recovery would be a behaviour beyond this slice's rows and would need its own case and comment. Recorded as a risk instead of implemented silently. | R-HR-NF-02 (hanging, not panicking), design §5.1 (`internal_error` row) | A later slice or the verify phase may decide; `ObsInternalFailure` already exists for a probe that catches its own failure |

---

## Workload and PR boundary (PR 4)

| Field | Value |
|---|---|
| Slice | PR 4 of 20 — “Runner: bounds, streaming, run budget, cancellation and the three timeout paths” (WU6 + WU7) |
| PR 4 estimate in `tasks.md` | 400–610 lines (point ≈505) |
| Host attempt ceiling | 2,000 counted changed lines |
| **Actual authored code + tests** | **1,025 lines** (0 deletions; both files are new): 188 production code, 527 test code, 220 comment, 90 blank |
| Artifact changes | `tasks.md` 10 lines changed (10 + 10 = **20** changed lines) + this appended section (**191** added lines) |
| **Total counted changed lines for the work unit** | **1,236** (1,025 authored + 20 checkbox + 191 artifact) |
| Chain per-PR cohesion ceiling | 600 changed lines (user-approved) |
| Review budget (session canonical) | 400 changed lines |
| Budget status | **Over the slice estimate, over the 600-line chain ceiling and over the 400-line session budget; inside the 2,000-line attempt ceiling** |
| PR boundary | Starts at PR 3's branch state (`internal/probe` vocabulary + classification table + seams + declared target set) and ends at `internal/probe` compiling and passing with the runner in place. `internal/probe/registry.go` and `local.go` (PR 5) are **not** started, and no later-slice file exists |
| Rollback boundary | Delete `internal/probe/runner.go` and `internal/probe/runner_test.go`; the module returns to its PR 3 state with the vocabulary, classification table, seams and target set still green. `tasks.md` lines 178–187 revert to `- [ ]`; this appended section is the only other PR 4 change |
| Rollback independence | Nothing consumes the runner yet — the doctor wiring lands in PR 18 and the ten real probes in PR 5–PR 9 — so the revert removes no unrelated work and leaves PR 1–PR 3 green |

**Why 1,025 > 600, stated honestly.** The overage is the test file, not padding: 527 of the 1,025 lines are `runner_test.go`, and its size is set by the rows themselves — a counting fixture that measures the peak *inside* the measurement boundary at three ceilings, a gated fixture for streaming, a hanging fixture, a run-budget fixture set of eight slow probes, a cancellation fixture set of six probes including a deliberately misbehaving one, and a three-row timeout table with pairwise distinctness. Another 220 lines are comments, most of them the *why* of the abandonment paths (the three causes and the ordering between them), which is exactly the code a reviewer has to trust. The review-budget rule forbids reaching a number by deleting tests, cases, comments or blank lines, and no `size:exception` was assumed, so **no code-golf pass was attempted** and nothing was compressed to fit.

**The second-split boundary `tasks.md` already names for this group was considered and is available.** The plan's boundary is “the concurrency ceiling, hanging-probe handling and streaming (WU6) first, then the run budget, cancellation and the three-timeout-paths table (WU7)”. A WU6/WU7 split would cut roughly `runner.go` 250 / `runner_test.go` 380 against the other half, and the split is strictly ordered (WU7's budget and cancellation build on WU6's `abandon`/`awaitProbe` machinery). **This executor did not split the slice or touch branches**: the parent assigned PR 4 as one unit, and a split moves a review boundary rather than the work. The named boundary is recorded for the orchestrator to apply if it wants the smaller review.

---

## Remaining unchecked tasks (PR 4 view)

**Inside this slice: none.** All ten PR 4 checkbox rows are `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (re-read after the edits: **34 checked, 131 unchecked**; 165 rows total; `git diff --stat` on that file shows exactly 10 insertions and 10 deletions). The PR 4 ownership markers were re-checked: ten rows, ten terminal `<!-- sdd-owner: implementation -->` markers.

The change-wide remainder is **131 checkbox lines in PR 5 – PR 20**, which belong to later chained slices on later branches (PR 4 targets PR 3's branch; the chain is `stacked-to-main`) and are **not** part of this work unit. The next unchecked line in the artifact, verbatim, is `tasks.md:194`:

```
- [ ] **RED** — add the registry enumeration case to `probe_test.go`: exactly the ten declared probes, each with one of the four kinds, enumeration order stable across runs, and no eleventh entry. <!-- sdd-owner: implementation -->
```

---

## Risks (PR 4)

| # | Risk | Status / handling |
|---|---|---|
| 1 | 1,025 authored lines against a 600-line chain ceiling (2,000-line attempt ceiling satisfied) | Disclosed above with the composition. The plan's own WU6/WU7 second-split boundary is available and named; the orchestrator owns that decision, and nothing was compressed to fit |
| 2 | A probe that **panics** is not converted into a result (deviation #10) | No task row requires it and the runner has no panic case. A panicking real probe would currently take the process down; `ObsInternalFailure` already exists if a later slice adds `recover`. Carried as a known gap, not silently closed |
| 3 | An abandoned probe's goroutine can outlive its concurrency slot (deviation #4) | Inherent: Go cannot kill a goroutine and the runner refuses to wait for a probe that ignores its bound. Documented on `Run`; the ceiling is stated as “admitted and awaited”. `-race ./...` reports nothing |
| 4 | The runner-classified observation's empty `Target` contradicts `probe.go`'s `Observation` doc comment | Recorded as deviation #2 because `probe.go` is outside this slice's file list. Either PR 16's mapper fills the declared target or a later slice corrects the comment; until then the discrepancy is a documentation debt, not a behavioural one |
| 5 | `ProbeTimeout`'s 10 s default is a judgment call | Deviation #1. It cannot express “unbounded”, so the worst case is bounded; if a hand-run shows a probe needing longer, that is a constant change (the D9 pattern for concurrency) |
| 6 | A budget that expires while a `done` receive is also ready can admit one more probe in that same scheduler iteration | The extra probe is still reported as `(unresolved, run_budget_exceeded)`, the run's wall time is still bounded by the timer, and the case is covered by `TestRunBudgetBoundsASlowSuite`'s wall-time assertion. Disclosed rather than defended as impossible |
| 7 | The map's focused test filter as written (`-run '…\|…'`) matches no test under RE2 | Observed and reproduced (run 17's first form). The working command uses bare `|`; recorded here so no later slice repeats the empty run |
| 8 | Real timers enforce the bounds while the injected clock reports elapsed values | Deliberate: an injected scripted clock cannot fire a deadline, so enforcement must be wall-clock while `Elapsed`/`generated_at` come from the ONE injected clock (design §3.3). Tests therefore assert bounds by wall time and never assert exact elapsed values |

### Engram mirror note

This file is the **authoritative** artefact. Its Engram mirror is split in **three** parts, because the merged text does not fit the store's 50,000-character content limit in two: part 1 — the file header plus the PR 1 and PR 2 sections — is saved under the topic `sdd/reach-diagnosis-core/apply-progress`; part 2 — the PR 3 section — under `sdd/reach-diagnosis-core/apply-progress/part2`; and part 3 — this PR 4 section — under `sdd/reach-diagnosis-core/apply-progress/part3`. Each mirror part states the artefact path, this file's byte size and its SHA-256 digest, so the mirror can be checked against the repository copy, and each states that the repository file is authoritative. The PR 3 section's own older note still describes the earlier two-part shape; it is an earlier section of this artefact and is deliberately not rewritten here.

---

# Apply Progress — PR 5 of 20 — WU8 + WU9 · `local.env`: registry, platform classification, native-Windows refusal and WSL2 limits

**Change**: `reach-diagnosis-core` · **Slice**: **PR 5 of 20** — “`local.env`: registry, platform classification, native-Windows refusal and WSL2 limits” (WU8 + WU9)
**Branch**: `feat/probe-registry-localenv`, stacked on PR 4's branch `feat/probe-runner` (chain strategy `stacked-to-main`, so PR 5 targets PR 4's branch and **not** `main`)
**Date**: 2026-09-14 · **Artifact store**: `both` (this file + Engram mirror, split as the mirror note at the end records)
**Strict TDD**: active — `openspec/config.yaml` declares `strict_tdd: true` with runner `go test ./...`; RED → GREEN → TRIANGULATE → REFACTOR followed for every row of this slice, and two REDs were **re-created** rather than recorded after the fact (see “Strict-TDD integrity note”)
**Skill resolution**: `paths-injected` — read `/Users/jack.smith/.config/opencode/skills/go-testing/SKILL.md` and `/Users/jack.smith/.config/opencode/skills/work-unit-commits/SKILL.md` before writing code; no registry fallback was needed
**Delivery path consumed**: `auto-chain` / `stacked-to-main` — this run implements **only** the assigned slice and stops at its PR boundary; the PR 1, PR 2, PR 3 and PR 4 sections above are preserved unchanged
**Commit status**: nothing committed, staged, pushed or branched by this phase; the work is left in the working tree for the orchestrator
**How to read this file**: PR 5's entry is the section below. The current change-wide remainder is restated at the end of this section.

---

## Structured status consumed (PR 5)

| Field | Value |
|---|---|
| `schemaName` / `schemaVersion` | `gentle-ai.sdd-status` / `2` |
| `changeName` | `reach-diagnosis-core` |
| `nextRecommended` | `apply` |
| `applyState` | `ready` |
| `dependencies.apply` / `.verify` / `.archive` | `ready` / `blocked` / `blocked` |
| `actionContext.mode` | `repo-local` |
| `actionContext.workspaceRoot` | `/Users/jack.smith/projects/work/herdr-reach` |
| `actionContext.allowedEditRoots` | `["/Users/jack.smith/projects/work/herdr-reach"]` — every file written lives inside it |
| `artifactStore` | `both` declared by the parent prompt (native `openspec`); files written under `openspec/changes/reach-diagnosis-core/` and mirrored to Engram |
| `taskProgress` before this run | 165 total / 34 completed / 131 pending (PR 1 + PR 2 + PR 3 + PR 4) |
| `taskProgress` after this run | 165 total / **43 completed** / 122 pending |
| `actionContext` warnings | none |
| Work-unit ownership markers | all nine PR 5 rows carry the terminal `<!-- sdd-owner: implementation -->` marker; the whole file still holds 165 markers for 165 checkbox rows, none malformed, duplicate, unsupported or non-terminal |
| `applyState: all_done`? | no — implementation continues, so editing was permitted |

**Attempt context**: the harness reports one bounded attempt already active for this work unit (`token sha256:71b2aa5283a109d2c0fab439c60273886ac562cd3342be3332b5156a2150b2ca`) with a **2,000**-changed-line ceiling, generous because the ledger charges every file the work unit touches. Per the parent prompt the parent owns `sdd-attempt acquire`/`settle`; this executor did **not** call the native attempt command. The counted-line position against that ceiling is in “Workload and PR boundary”.

---

## Completed tasks and their persisted checkbox updates (PR 5)

All nine PR 5 rows were flipped from `- [ ]` to `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (lines 196–204) as the work completed, then re-read to confirm. `git diff --stat` on that file reports exactly `9 insertions(+), 9 deletions(-)` — nine flips and nothing else.

| # | Task (short) | Persisted update | Evidence |
|---|---|---|---|
| 1 | RED — registry enumeration case in `probe_test.go` (ten probes, four kinds, stable order, no eleventh entry) | `tasks.md:196` → `- [x]` | `go test ./...` → `FAIL … [build failed]`: `undefined: probe.Registry` (`:241`, `:287`, `:301`, `:329`, `:345`), `undefined: probe.ProbeRegistration` (`:300`), `undefined: probe.Probes` (`:339`) — the registry vocabulary did not exist |
| 2 | GREEN — `registry.go` with the ordered registry and a constructor per probe, `local.env` only | `tasks.md:197` → `- [x]` | `go test -count=1 -run 'TestRegistry' -v ./internal/probe/` exit 0: `TestRegistryEnumeratesTheDeclaredProbes` and `TestRegistryBuildsOnlyRegisteredProbes` PASS; ten entries, one built (`local.env`) |
| 3 | GREEN — `local.env` classification in `local.go` (Linux, macOS, WSL2, native Windows plus architecture) over the injected `Platform` seam | `tasks.md:198` → `- [x]` | The four-seam case is green (run 6 below): every classification asserted with its architecture; `gofmt -l .` empty and `go vet ./...` clean at each step |
| 4 | RED + TRIANGULATE — script all four platform seams with their arch; assert no provisioning action (R-HR-29) | `tasks.md:199` → `- [x]` | `go test -count=1 -run 'TestLocalEnv'` exit 1: `--- FAIL: TestLocalEnvClassifiesEachPlatform/native_windows` — `a scripted native windows machine measured "unresolved", want "measured"`; then green. Five further cases (unknown/degrade ×5, no-provisioning guard over 8 platforms, FS tripwire + `WSL_DISTRO_NAME`/`WSL_INTEROP` isolation, scripted-clock elapsed, identity round-trip) |
| 5 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/` | `tasks.md:200` → `- [x]` | `gofmt -l .` exit 0 (empty output) · `go vet ./...` exit 0 (no output) · `go test -count=1 ./internal/probe/` exit 0 |
| 6 | RED — refusal case: native Windows ⇒ `(measured, fail, node_platform_unsupported)` naming WSL2 as the supported path | `tasks.md:201` → `- [x]` | `go test -count=1 -run 'TestLocalEnvRefusesNativeWindows'` exit 1: both subtests `verdict = "pass", want "fail"` — a genuine behaviour RED, not a compile error |
| 7 | GREEN — implement the refusal plus the degrade path to `(unresolved, platform_unknown)` with no default guess | `tasks.md:202` → `- [x]` | `go test -count=1 ./...` exit 0 after the refusal; the degrade path is asserted by the five unclassified cases (deny-all seam, `freebsd`, linux with unknown arch, linux with empty arch, nil seam) and `TestLocalEnvRefusesNativeWindows` passes with both measured and unreported architectures |
| 8 | TRIANGULATE — WSL2 detail limited to documented semantics (milliseconds, 60000, Windows 11), no child-of-init rule and no `-1` sentinel (RG-4); WSL2 without systemd is detection-only and names a later slice | `tasks.md:203` → `- [x]` | `go test -count=1 -run 'TestLocalEnvWSL2TextIsDocumentedSemanticsOnly'` exit 1 first (7 failures: `milliseconds`, `60000`, `Windows 11` in both systemd states, plus the missing later-slice clause), then exit 0; nine mutations M1–M9 prove the guards can fail |
| 9 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/` | `tasks.md:204` → `- [x]` | Review added the macOS architecture-guard case and the identity round-trip case (both mutation-checked) and one `Run` doc sentence; final gates: `gofmt -l .` empty · `go vet ./...` clean · `go test -count=1 ./...` exit 0 · `go test -race -count=1 ./...` exit 0 |

---

## Files changed (PR 5)

| Path | Status | Authored lines | Purpose |
|---|---|---|---|
| `internal/probe/registry.go` | added | 134 (46 code, 81 comment, 7 blank) | The ten probe-name constants, `ProbeFactory`, `ProbeRegistration`, the ordered ten-entry `registry` declaration (one constructor filled), `Registry()` returning a copy, and `Probes(seams)` building the landed subset in registry order |
| `internal/probe/local.go` | added | 345 (148 code, 167 comment, 30 blank) | `NodePlatform` and its five values, `NodePlatformIdentity`/`SplitNodePlatformIdentity`, `platformSignals`, the `localEnv` probe (`Name`/`Kind`/`Run`/`now`/`observe`), `classifyPlatform` with the four classifications and the degrade, and the per-platform wording builders |
| `internal/probe/local_test.go` | added | 605 (439 code, 117 comment, 49 blank) | `scriptedPlatform`, `tripwireFS`, `localEnvBuild`, `runLocalEnv`, `provisioningTokens`; nine test functions: four-platform classification, five unclassified/degrade cases, the native-Windows refusal (2 subtests), the RG-4 WSL2 guard, the no-provisioning guard over 8 platforms, the seam-isolation tripwire, the scripted-clock elapsed case, and the identity round-trip |
| `internal/probe/probe_test.go` | modified | +166 / −0 | `declaredProbeRegistry` (PRD §5.1 transcribed), `probeKinds`, `TestRegistryEnumeratesTheDeclaredProbes`, `TestRegistryBuildsOnlyRegisteredProbes` |
| `internal/probe/classify.go` | modified (enabling edit — see deviation #1) | +14 / −3 | `ObsPlatformSignalsClassified` and the positive `PurposePlatformClassification` row the design's table omits while §5.3 requires `local.env` to pass |
| `openspec/changes/reach-diagnosis-core/tasks.md` | modified | 9 lines changed in place (9 deletions + 9 additions = 18 changed lines) | Nine PR 5 checkboxes `- [ ]` → `- [x]` |
| `openspec/changes/reach-diagnosis-core/apply-progress.md` | modified | this appended section | Cumulative PR 5 evidence; the PR 1–PR 4 sections are untouched |

**Authored code + tests: 1,264 additions and 3 deletions = 1,267 counted changed lines** (134 + 345 + 605 + 166 additions; `classify.go` +14/−3). No file outside the four assigned paths, the one disclosed enabling path and the two artifact files was created or modified; `README.md`, `PRD.md`, the proposal, the specs, the design, `explore.md`, `research.md`, `preproposal.md` and `openspec/config.yaml` are untouched. No new dependency was added (production code imports only `context`, `fmt`, `strings` and `time`; the tests add `context`, `errors`, `os`, `strings`, `testing`, `time`); no linter or CI configuration was introduced.

---

## Test commands run — exact commands and exit status (PR 5)

| # | Command | Exit | Observed output (abridged) |
|---|---|---|---|
| 1 | `go test ./...` (task 1 RED) | non-zero — `FAIL … [build failed]` | `internal/probe/probe_test.go:241:20: undefined: probe.Registry` (five sites), `:300:22: undefined: probe.ProbeRegistration`, `:339:17: undefined: probe.Probes`. (The invocation piped into `head`, so the printed shell status was `head`'s; the `FAIL … [build failed]` line is `go test`'s own verdict, which is non-zero on a build failure.) |
| 2 | `go test -count=1 -run 'TestRegistry' -v ./internal/probe/` (task 2 GREEN) | **0** | `--- PASS: TestRegistryEnumeratesTheDeclaredProbes`, `--- PASS: TestRegistryBuildsOnlyRegisteredProbes` |
| 3 | `gofmt -l .` · `go vet ./...` · `go test -count=1 ./...` (task 2/3 gate) | **0** each | `gofmt` empty output; `vet` no output; `ok …/internal/probe 0.405s` · `ok …/internal/version 0.006s` |
| 4 | `go test -count=1 -run 'TestLocalEnv' ./internal/probe/` (task 4 RED) | **1** | `--- FAIL: TestLocalEnvClassifiesEachPlatform/native_windows` — `local_test.go:208: a scripted native windows machine measured "unresolved", want "measured"`. Every other case passed: the three supported platforms, the five unclassified cases, the guards and the clock case |
| 5 | `gofmt -w internal/probe/local_test.go` + `gofmt -l .` · `go vet ./...` · `go test -count=1 ./...` (task 4 GREEN) | **0** each | Full suite green; `gofmt` empty output |
| 6 | `go test -count=1 -run 'TestLocalEnvRefusesNativeWindows' ./internal/probe/` (task 6 RED) | **1** | `--- FAIL: TestLocalEnvRefusesNativeWindows/a_measured_architecture` and `…/an_architecture_the_seam_did_not_report` — `verdict = "pass", want "fail"` |
| 7 | `gofmt -l .` · `go vet ./...` · `go test -count=1 ./...` (task 7 GREEN) | **0** each | `ok …/internal/probe 0.406s` · `ok …/internal/version 0.008s` |
| 8 | `go test -count=1 -run 'TestLocalEnvWSL2TextIsDocumentedSemanticsOnly' ./internal/probe/` (task 8 RED) | **1** | Seven failures: `the WSL2 text does not state the documented semantics "milliseconds"`, `…"60000"`, `…"Windows 11"` — each in both the systemd-present and systemd-absent states — plus `the WSL2 text without systemd does not state that enabling it belongs to a later slice` |
| 9 | `gofmt -l .` · `go vet ./...` · `go test -count=1 ./...` (task 8 GREEN) | **0** each | `ok …/internal/probe 0.414s` · `ok …/internal/version 0.007s` |
| 10 | mutations M1–M9, one at a time, each against a `/tmp` backup (see “Mutation evidence”) | **1** each | Every mutation was caught by the case it targets; after each restore `diff` confirmed the file was byte-identical to the backup and the suite was green again |
| 11 | `go test -count=1 -run 'TestRegistry\|TestLocalEnv\|TestNodePlatformIdentity' ./internal/probe/` (the work-unit map's `Unit verification` command, **bare pipes**) | **0** | `ok …/internal/probe 0.009s` — note 12 below |
| 12 | `go test -count=1 -run 'TestRegistry\|TestLocalEnv\|TestNodePlatformIdentity' -v ./internal/probe/` | **0** | Nine top-level functions PASS (23 `--- PASS` lines including subtests): the two registry cases and the seven `local.env` cases listed above plus the identity round-trip |
| 13 | `gofmt -l .` (final gate) | **0** | empty output — nothing unformatted |
| 14 | `go vet ./...` (config `quality.typecheck`) | **0** | no output |
| 15 | `go test -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 0.406s` · `ok …/internal/version 0.007s` |
| 16 | `go test -race -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 1.461s` · `ok …/internal/version 1.019s` |
| 17 | `go test -count=1 -v ./internal/probe/` (final inventory) | **0** | 41 top-level test functions, 191 `--- PASS` lines including subtests across the package |

**Note on the map's focused command (2).** `tasks.md`'s work-unit map writes PR 5's unit-verification filter as `-run 'TestRegistry\|TestLocalEnv'`, and Go's RE2 engine reads `\|` as a **literal pipe**, so that exact string matches no test — run 12's first form reproduced it (`ok … [no tests to run]`, exit 0) before the working form with bare `|` was used. This is the same artifact typo PR 4 recorded as risk #7; the nine checkboxes are unaffected.

**Runtime harness**: **N/A as an end-to-end run, with the probe-level harness now reachable in process.** There is still no CLI (`cmd/herdr-reach` lands in PR 18/19), no `doctor` wiring (PR 18) and no network measurement: the two probes that exist for a real machine are this one and the runner. What *is* exercised is the production path a run takes to this probe — `Registry()` → the declared entry → its constructor → `Run` — driven with scripted platform seams and the deny-all capability set, which is design §6.2's first proof level for a probe. PR 6 onward adds the transport probes; PR 20 owns the writes-nothing and zero-exec proof.

---

## TDD Cycle Evidence (PR 5)

RED → GREEN → TRIANGULATE → REFACTOR per PR 5 task row. Four REDs were produced before the behaviour they exercise existed, and three of them are genuine *behaviour* failures rather than compile errors: the native-Windows classification (`unresolved`, want `measured`), the refusal (`pass`, want `fail`) and the WSL2 documented semantics (missing text).

| Task row | Phase | Evidence produced | Observed failure (RED) | Observed pass (GREEN) |
|---|---|---|---|---|
| 1 RED — registry enumeration | RED | `probe_test.go` gained `declaredProbeRegistry` (PRD §5.1 transcribed literally), `probeKinds`, `TestRegistryEnumeratesTheDeclaredProbes` and `TestRegistryBuildsOnlyRegisteredProbes` before any `registry.go` existed | exit non-zero: `undefined: probe.Registry` (×5), `probe.ProbeRegistration`, `probe.Probes` / `FAIL … [build failed]` | n/a |
| 2 GREEN — `registry.go` | GREEN | The ten name constants, `ProbeFactory`, `ProbeRegistration`, the ordered ten-entry table (only `local.env` built), `Registry()` as a copy, `Probes(seams)` in registry order | (previous row) | exit 0: both registry cases PASS |
| 3 GREEN — `local.env` classification | GREEN | `local.go`: the classification vocabulary, the identity helpers, the probe and `classifyPlatform`; `classify.go` gained the positive platform row it needed (deviation #1) | (previous row) | exit 0 after GREEN 2 below; `gofmt`/`vet` clean at every step |
| 4 RED + TRIANGULATE — four platform seams, no provisioning | RED → TRIANGULATE | `local_test.go`: `scriptedPlatform`, `tripwireFS`, `localEnvBuild`, `runLocalEnv`, `TestLocalEnvClassifiesEachPlatform`, `TestLocalEnvUnknownPlatformIsNotGuessed`, `TestLocalEnvOffersNoProvisioningAction`, `TestLocalEnvReadsOnlyInjectedSeams`, `TestLocalEnvElapsedComesFromTheInjectedClock` | exit 1: `--- FAIL: TestLocalEnvClassifiesEachPlatform/native_windows` — `measured "unresolved", want "measured"`; the other cases passed against the classification that existed, which is what makes the RED specific | exit 0 after GREEN 2 added the Windows classification branch |
| 5 REFACTOR + GATE | REFACTOR | No code restructured; comments reviewed and the gates run | n/a | exit 0: `gofmt -l .` (empty), `go vet ./...`, `go test -count=1 ./internal/probe/` |
| 6 RED — the refusal | RED | `TestLocalEnvRefusesNativeWindows` written first, with two subtests (a measured architecture, an unreported one) | exit 1: both subtests `verdict = "pass", want "fail"` | n/a |
| 7 GREEN — refusal + degrade | GREEN | `classifyPlatform`'s Windows branch now returns `ObsNodePlatformUnsupported` and `windowsWording` names WSL2 as the supported path; the degrade path implemented with the classification is asserted by the five unclassified cases | (previous row) | exit 0: the refusal case passes; the full suite is green |
| 8 TRIANGULATE — WSL2 documented semantics | TRIANGULATE (own RED→GREEN) | `TestLocalEnvWSL2TextIsDocumentedSemanticsOnly` written first: required tokens (`milliseconds`, `60000`, `Windows 11`), forbidden tokens (`-1`, child-of-init spellings, `/init`, `keepalive`, `watchdog`, `persist`), and the systemd-absent detection-only clause | exit 1: seven failures — the three documented-semantics tokens missing in both systemd states, plus the missing later-slice clause | exit 0 after `wsl2Wording` implemented the documented sentence and the later-slice clause; teeth proven by M1 and M2 |
| 9 REFACTOR + GATE | REFACTOR | Review pass: one `Run` doc sentence about the target, the macOS architecture-guard case and the identity round-trip case added (both mutation-checked as M9 and by the splitter's malformed-value assertions), comments re-read against the code; no behaviour change | n/a | exit 0: `gofmt -l .` (empty), `go vet ./...`, `go test -count=1 ./...`, `go test -race -count=1 ./...` |

**Strict-TDD integrity note.** Two rows are recorded honestly rather than flatteringly. (a) The registry cannot compile without a probe constructor — `tasks.md`'s own sequencing note 2 says so — so rows 2 and 3 landed in one GREEN after RED 1; rows 6–8 then each got their own RED. (b) The refusal text and the WSL2 documented semantics were **first written with the classification**, which would have made rows 6 and 8 test-after-code; both were deliberately backed out, the REDs were re-run and observed (runs 6 and 8), and only then were they implemented again. The final code is byte-identical to the intent; the evidence above is the honest record of the sequence, not a reconstruction.

---

## Mutation evidence (PR 5)

Every mutation was applied to a `/tmp` backup of the file, run against the focused case, and restored; `diff` then confirmed the restore and the package was re-run green.

| # | Mutation | Case that caught it | Observed failure |
|---|---|---|---|
| M1 | The WSL2 text gained `, and only children of /init keep the instance alive` | `TestLocalEnvWSL2TextIsDocumentedSemanticsOnly` | exit 1 — `the WSL2 text states the undocumented "child of init"` and `"/init"` |
| M2 | The WSL2 text gained `(set vmIdleTimeout=-1 to disable)` | `TestLocalEnvWSL2TextIsDocumentedSemanticsOnly` | exit 1 — the `-1` sentinel is caught as a substring |
| M3 | The Windows branch reported `ObsPlatformSignalsClassified` instead of `ObsNodePlatformUnsupported` | `TestLocalEnvRefusesNativeWindows` | exit 1 — both subtests `verdict = "pass", want "fail"` |
| M4 | The Linux branch dropped its `archReported()` guard | `TestLocalEnvUnknownPlatformIsNotGuessed` | exit 1 — the unknown-architecture and empty-architecture cases classified as Linux |
| M5 | The registry's `egress.cf.7844` and `egress.cf.443` entries were swapped | `TestRegistryEnumeratesTheDeclaredProbes` | exit 1 — both positions and the declared-target-set cross-check disagree |
| M6 | `Registry()` returned the package slice instead of a copy | `TestRegistryEnumeratesTheDeclaredProbes` | exit 1 — `Registry() handed out the package's own registry slice` |
| M7 | `egress.quic` was registered with kind `ProbeEgress` | `TestRegistryEnumeratesTheDeclaredProbes` | exit 1 — the kind disagrees with the transcribed PRD §5.1 table |
| M8 | The positive `PurposePlatformClassification` row was removed from `classify.go` | `TestLocalEnvClassifiesEachPlatform` | exit 1 — all five supported-platform subtests fell to the table's total row |
| M9 | The macOS branch dropped its `archReported()` guard | `TestLocalEnvUnknownPlatformIsNotGuessed` | exit 1 — the new macOS-unknown-architecture case classified as macOS |

**Triangulation depth.** Four independent angles on the registry (positional transcription, four-kind closure, stability plus copy isolation, and every built probe matching its declaration) and five on the probe (four platform classifications read back through the package's splitter, five distinct unclassified gaps, the refusal with two architecture states, the RG-4 text guard in both systemd states, and the seam-isolation tripwire with the process environment set to look like WSL2). **Nine mutations, every one caught**, and three genuine behaviour REDs.

---

## Deviations from design (PR 5)

| # | Deviation | Why | Design reference | Follow-up owner |
|---|---|---|---|---|
| 1 | **`internal/probe/classify.go` was edited, which is outside the PR 5 file list.** One observable (`ObsPlatformSignalsClassified`) and one row (`PurposePlatformClassification` → measured/pass/`ok`) were added, +14/−3. | Design §5.1's platform rows are only “signals match no supported classification” and “classified native Windows”. Neither can express a healthy machine, yet §5.3 requires `local.env` to **pass** on Linux. Without the row the probe could only report `internal_error` through the table's total row, and the table is the only place a reason code may be chosen. PR 2's own record anticipated this (“later probe PRs must declare the purpose and observable that match their real declared question”). The edit is purely additive, in the same package, and the row's coverage is proven by M8. | design §5.1 (platform rows), design §5.3 (“`local.env` passes”), PR 2 apply-progress deviation #2 | `sdd-verify` must adjudicate; the orchestrator may prefer to move this row into a PR 5a/PR 5b boundary or to fold it into a review of PR 2 |
| 2 | `Registry()` returns `[]ProbeRegistration` (name, kind, constructor), not `[]Probe`, and `Probes(seams)` builds the landed subset. | Design §4's sketch says `probe.Registry() // exactly ten probes`, which cannot hold while nine probes have no implementation: returning nine placeholder probes would fabricate results for probes no slice has written, and returning only `local.env` would fail the “exactly ten” scenario. The declaration carries the enumeration contract instead, and `Probes` can never produce an eleventh entry (asserted). | design §4 (data flow sketch), design §7 (`registry.go` row), diagnosis spec “exactly the ten declared probes” | PR 18's doctor wiring calls `Probes(seams)`; `sdd-verify` may want the sketch re-worded |
| 3 | `NodePlatform`, its five values, and the identity pair `NodePlatformIdentity`/`SplitNodePlatformIdentity` (with `PlatformIdentitySeparator`) are exported although no consumer exists yet. | The payload's `node.platform` and `node.arch` (design §3.3) are exactly the two halves, and PR 9's accessors and PR 12's `NODE_*` rules need them apart. A format known only to `local.go` would be re-derived by string surgery in another package, which is what the single-home rule exists to prevent. The round-trip and the malformed-value cases are asserted now. | design §3.3 (`node` object), design §5.2 (`node.platform` rules) | PR 10/PR 12/PR 16 consume them; `sdd-verify` may treat the pair as a refinement |
| 4 | A supported platform whose architecture was **not** reported degrades to `(unresolved, platform_unknown)` naming the missing half. The refusal is the exception: it rests on the operating system alone. | R-HR-29 requires the classification *and* the architecture, and a node whose architecture was not measured is exactly the node whose binaries cannot be chosen (PRD §13: “Hub and node architectures differ — detect and report. Never copy a binary between them”). Calling it supported would present a classification whose binary-compatibility half is missing. The refusal is a statement about the operating system, so an unreported architecture neither weakens nor fabricates it. | R-HR-29, PRD §13, design §5.1 (`platform_unknown` row) | `sdd-verify` |
| 5 | A **nil** `Platform` seam resolves to `(unresolved, platform_unknown)`, not to `not_measured`/`capability_excluded`. | The table's `ObsCapabilityExcluded` row is scoped to `PurposeSSHDConfiguration`; a platform-scoped variant is a larger contract change than this slice should make on another slice's file. The detail says a seam was never injected, so the gap is named rather than hidden, and nothing is guessed. | design §5.1 (attempted-vs-not-attempted rule), design §6.2 | PR 6/PR 9 own the `capability_excluded` shape; a verifier may prefer not-measured here |
| 6 | WSL2's **systemd state lives in the verbatim detail**, not in a machine-readable field, and the observation's target carries only `<platform>/<arch>`. | The reason-code set is closed and holds no code for “systemd is not the running service manager”; a second observation would need a new code, which is a documented contract change (design §3.5). The text is asserted (RG-4, detection-only) but the reasoning layer cannot yet match it structurally. | design §5.2 (`NODE_WSL2_SYSTEMD_ABSENT`: “WSL2 ∧ systemd not enabled”), design §3.5 (closed set) | **PR 12 must extend the match** — either `Need` gains an observation label/observable, or the probe gains a structured systemd field. Recorded as risk 3 |
| 7 | The nine unlanded registry entries carry **nil** constructors, and every later probe slice must fill one in — an edit to `registry.go` that PR 6–PR 9's file lists do not name. | The full ten-entry declaration is what the enumeration contract needs, and a nil constructor is the honest representation of “no slice has written this probe yet”. The alternative (a placeholder probe) would fabricate a measurement. | design §7 (`registry.go` row), tasks.md sequencing note 2 | PR 6–PR 9: one line each; the enumeration case keeps the names, kinds and order honest |
| 8 | `local.env` reads **no** FS seam, although design §6.1 lists `FS` among its consumers. | The classification needs no file and no environment variable: WSL2 and systemd are Platform signals, and reading `/proc/version` or `WSL_DISTRO_NAME` directly would classify the machine the test runs on. The tripwire case proves the FS seam was never called and that the process environment (`WSL_DISTRO_NAME`, `WSL_INTEROP`, `WSLENV` set) cannot move the classification. | design §6.1 (seam table), design §6.2 (deny-all default) | A later slice that wants env-based corroboration goes through `FS.Getenv`; recorded so the table and the code are not silently inconsistent |
| 9 | Both new test files are **external** test packages (`package probe_test`), as in PR 1–PR 4. | The probe is reached through `Registry()` → constructor → `Run`, which is the production path, and the helpers (`holds`) and the guards are readable as a consumer would read them. | tasks PR 5 file list; design §8 test plan | n/a |

---

## Workload and PR boundary (PR 5)

| Field | Value |
|---|---|
| Slice | PR 5 of 20 — “`local.env`: registry, platform classification, native-Windows refusal and WSL2 limits” (WU8 + WU9) |
| PR 5 estimate in `tasks.md` | 380–570 lines (point ≈475) |
| Host attempt ceiling | 2,000 counted changed lines |
| **Actual authored code + tests** | **1,264 additions / 3 deletions = 1,267 counted lines**: `registry.go` 134, `local.go` 345, `local_test.go` 605, `probe_test.go` +166, `classify.go` +14/−3 |
| Artifact changes | `tasks.md` 9 lines changed (9 + 9 = **18** changed lines) + this appended section |
| **Total counted changed lines for the work unit** | **1,499** (1,264 authored + 3 deletions + 18 checkbox + 214 this section) |
| Chain per-PR cohesion ceiling | 1,000 changed lines (user-approved, revised from 600 on 2026-09-14) |
| Review budget (session canonical) | 400 changed lines |
| Budget status | **Over the slice estimate, over the 1,000-line chain ceiling and over the 400-line session budget; inside the 2,000-line attempt ceiling** |
| PR boundary | Starts at PR 4's branch state (`internal/probe` vocabulary + classification table + seams + declared target set + runner) and ends at `internal/probe` compiling and passing with the ordered registry and `local.env` in place. PR 6 (`local.sshd` + `egress.hub.direct`) is **not** started: no `internal/probe/egress.go`, `quic.go`, `tls.go`, `internal/diagnosis/`, `internal/transport/`, `internal/report/`, `internal/doctor/`, `cmd/` or `docs/diagnosis-report.md` exists |
| Rollback boundary | Delete `internal/probe/registry.go`, `local.go` and `local_test.go`; revert `probe_test.go`'s two added test functions plus their two transcription variables and `classify.go`'s +14/−3. The module returns to its PR 4 state with the vocabulary, the classification table, the seams, the declared target set and the runner still green. `tasks.md` lines 196–204 revert to `- [ ]`; this appended section is the only other PR 5 change |
| Rollback independence | Nothing consumes the registry or `local.env` yet — the doctor wiring lands in PR 18 and the other probes in PR 6–PR 9 — so the revert removes no unrelated work and leaves PR 1–PR 4 green |

**Why 1,267 > 1,000, stated honestly, and the boundary the plan already names.** The overage is the probe suite and its comments, not padding: 605 of the 1,264 lines are `local_test.go` (nine test functions, eight platforms in the wording guard, the FS tripwire, the clock case and the identity round-trip), 166 are the transcription and guards of the enumeration case, and 430 of `local.go`/`registry.go` are the comments that carry the *why* of each classification branch and of the registry's declaration-not-construction shape — exactly the code a reviewer has to trust. The review-budget rule forbids reaching a number by deleting tests, control cases, triangulation cases, comments, docs or blank lines, and no `size:exception` was assumed, so **nothing was compressed**; the honest count is reported instead.

Per the parent's instruction, the boundary the task plan already names is reported rather than applied here: **PR 5a = WU8** — the registry plus the platform classifications (`registry.go` 134 + `probe_test.go` 166 + `classify.go` 17 + the classification parts of `local.go` ≈ 291 and `local_test.go` ≈ 448) ≈ **1,056 counted lines**; **PR 5b = WU9** — the refusal, the unknown-platform degrade and the documented WSL2 limits (the Windows branch and `windowsWording`, `wsl2Wording`, and their two cases, ≈ 54 + 157) ≈ **211 counted lines**. The split is strictly ordered (the refusal and the WSL2 text are branches of the classification WU8 introduces), and PR 5b would target PR 5a's branch. The first half is still marginally above the 1,000-line ceiling (by ≈ 5 %), almost entirely because the 166-line enumeration case belongs with the registry; moving that case to the second half would put WU8 at ≈ 890 and WU9 at ≈ 377, which is the only adjustment that would bring both halves inside the ceiling. **This executor did not split the slice or touch branches**: the parent assigned PR 5 as one unit, and a split moves a review boundary rather than the work.

---

## Remaining unchecked tasks (PR 5 view)

**Inside this slice: none.** All nine PR 5 checkbox rows are `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (re-read after the edits: **43 checked, 122 unchecked**; 165 rows total; `git diff --stat` on that file shows exactly 9 insertions and 9 deletions). The ownership markers were re-checked: 165 `<!-- sdd-owner: implementation -->` markers for 165 rows, every one terminal.

The change-wide remainder is **122 checkbox lines in PR 6 – PR 20**, which belong to later chained slices on later branches (PR 5 targets PR 4's branch; the chain is `stacked-to-main`) and are **not** part of this work unit. The next unchecked line in the artifact, verbatim, is `tasks.md:211` — PR 6's first row, which is where the next slice resumes:

```
- [ ] **RED** — add the divergence case: written ≠ effective configuration ⇒ `(measured, fail, sshd_config_divergence)` with both configurations in the verbatim detail (R-HR-18). <!-- sdd-owner: implementation -->
```

PR 6 must also fill the `local.sshd` constructor slot in `internal/probe/registry.go` (deviation #7) and add `internal/probe/egress.go`; PR 7–PR 9 do the same for the remaining eight entries.

---

## Risks (PR 5)

| # | Risk | Status / handling |
|---|---|---|
| 1 | 1,267 counted lines against a 1,000-line chain ceiling (2,000-line attempt ceiling satisfied) | Disclosed above with the composition and the named WU8/WU9 boundary, including the measurement that shows the first half still lands at ≈ 1,056. The orchestrator owns the split decision; nothing was compressed to fit |
| 2 | `classify.go` was edited although it is outside the assigned file list (deviation #1) | Disclosed with the exact reason (design §5.3 requires a pass the table cannot express), the size (+14/−3, additive), and the mutation that proves the new row is covered (M8). `sdd-verify` must adjudicate the boundary |
| 3 | WSL2's systemd state has no machine-readable home (deviation #6) | The text is asserted in both states, but PR 12's `NODE_WSL2_SYSTEMD_ABSENT` rule cannot match it structurally yet. The exact interface gap is recorded with two candidate shapes; until then the finding would have to rest on an observation label or on detail text, and the latter is what R-HR-07 forbids |
| 4 | `Registry()` returns registrations, not `[]Probe` (deviation #2) | The design's sketch is narrower than the slice can be; the enumeration scenario is still asserted in full. If the maintainer prefers `Registry() []Probe`, the change is small but would reintroduce the placeholder-probe problem the declaration avoids |
| 5 | Nine registry entries have nil constructors, and later slices' file lists do not name `registry.go` (deviation #7) | Flagged for PR 6–PR 9 with the one-line edit each needs. The enumeration case fails if a later slice moves a name, a kind or a position, so the wiring cannot drift silently |
| 6 | A nil `Platform` seam resolves to `platform_unknown` rather than not-measured (deviation #5) | Disclosed; the detail names the missing seam so the gap is visible. The alternative needs a platform-scoped `capability_excluded` row in `classify.go`, which is a larger edit to the file this slice already had to touch once |
| 7 | The provisioning-token guard is a wording guard, not a proof | Stated in the test's own doc comment: it catches text that offers or claims a change, and the structural proof (tree digests, zero-exec counter, closed dialed set) is PR 20's per design §6.3. The probe also reads nothing through the FS seam (tripwire case), so it has no path to change anything |
| 8 | `local.env`'s observation target is an identity (`linux/aarch64`), not an address | Deliberate and documented on `Run`: a local probe dials nothing, so the identity is what it measured. PR 16's payload mapper must echo `probes[].target` for a local probe as this value rather than as `null`; the payload's `node.platform`/`node.arch` come from splitting it |
| 9 | `ProbeFactory` takes only `Seams`, so a probe needing run input (the hub address, overrides) must widen the signature in its own slice | Accepted for this slice because `local.env` needs nothing else, and the widening is a normal, reviewable edit in the slice that needs it. Recorded so PR 6–PR 9 do not silently add package-level run input instead |
| 10 | Real-platform classification is only exercised through seams | Intended: the suite never runs on Windows, so the refusal is proven without a Windows host. A production `Platform` (PR 19, `real.go`) reading `runtime.GOOS`/`GOARCH` is the remaining unverified step, and it lands with the no-real-network guard |

### Engram mirror note

This file is the **authoritative** artefact. Its Engram mirror is now split in **four** parts because the merged text exceeds the store's 50,000-character content limit: part 1 — the file header plus the PR 1 and PR 2 sections — under the topic `sdd/reach-diagnosis-core/apply-progress`; part 2 — the PR 3 section — under `sdd/reach-diagnosis-core/apply-progress/part2`; part 3 — the PR 4 section — under `sdd/reach-diagnosis-core/apply-progress/part3`; and part 4 — this PR 5 section — under `sdd/reach-diagnosis-core/apply-progress/part4`.

Every part states the artefact path, this file's byte size and its SHA-256 digest **as recorded when that part was saved**, and that the repository file is authoritative. Parts 1–3 therefore carry their own slice's snapshot and their digests no longer match this file, which has grown by two sections since; part 4 carries the merged file's size and digest as of this run. A mirror part is a readable convenience copy of this artefact, not a second source of truth: read this file for the complete, current record. The PR 3 and PR 4 sections' own older notes still describe the earlier two- and three-part shapes; they are earlier sections of this artefact and are deliberately not rewritten here.

# Apply Progress — PR 6 of 20 — WU10 + WU11 · `local.sshd` and `egress.hub.direct`

**Change**: `reach-diagnosis-core` · **Slice**: **PR 6 of 20** — “`local.sshd` and `egress.hub.direct`” (WU10 + WU11)
**Branch**: `feat/probe-sshd-hub`, stacked on PR 5's branch `feat/probe-registry-localenv` (chain strategy `stacked-to-main`, so PR 6 targets PR 5's branch and **not** `main`)
**Date**: 2026-09-14 · **Artifact store**: `both` (this file + Engram mirror, split as the mirror note at the end records)
**Strict TDD**: active — `openspec/config.yaml` declares `strict_tdd: true` with runner `go test ./...`; RED → GREEN → TRIANGULATE → REFACTOR followed for both halves of this slice
**Skill resolution**: `paths-injected` — read `/home/luisalt20/.config/opencode/skills/go-testing/SKILL.md` and `/home/luisalt20/.config/opencode/skills/work-unit-commits/SKILL.md` before writing code; no registry fallback was needed
**Delivery path consumed**: `auto-chain` / `stacked-to-main` — this run implements **only** the assigned slice and stops at its PR boundary; the PR 1–PR 5 sections above are preserved unchanged
**Commit status**: nothing committed, staged, pushed or branched by this phase; the work is left in the working tree for the orchestrator
**How to read this file**: PR 6's entry is the section below. The current change-wide remainder is restated at the end of this section.

---

## Structured status consumed (PR 6)

| Field | Value |
|---|---|
| `schemaName` / `schemaVersion` | `gentle-ai.sdd-status` / `2` |
| `changeName` | `reach-diagnosis-core` |
| `nextRecommended` | `apply` |
| `applyState` | `ready` |
| `dependencies.apply` / `.verify` / `.archive` | `ready` / `blocked` / `blocked` |
| `actionContext.mode` | `repo-local` |
| `actionContext.workspaceRoot` | `/home/luisalt20/projects/close/herdr-reach` |
| `actionContext.allowedEditRoots` | `["/home/luisalt20/projects/close/herdr-reach"]` — every file written lives inside it |
| `artifactStore` | `both` declared by the parent prompt (native `openspec`); files written under `openspec/changes/reach-diagnosis-core/` and mirrored to Engram |
| `taskProgress` before this run | 165 total / 43 completed / 122 pending (PR 1 + PR 2 + PR 3 + PR 4 + PR 5) |
| `taskProgress` after this run | 165 total / **52 completed** / 113 pending |
| `actionContext` warnings | none |
| Work-unit ownership markers | all nine PR 6 rows carry the terminal `<!-- sdd-owner: implementation -->` marker; the whole file still holds 165 markers for 165 checkbox rows, none malformed, duplicate, unsupported or non-terminal |
| `applyState: all_done`? | no — implementation continues, so editing was permitted |

**Attempt context**: the harness reports one bounded attempt already active for this work unit (`token sha256:da57554a6e523d04cff6731a570b8cf5cb6f607312b47cb7315deb6640a6a86b`) with a **2,000**-changed-line ceiling. Per the parent prompt the parent owns `sdd-attempt acquire`/`settle`; this executor did **not** call the native attempt command. The counted-line position against that ceiling is in “Workload and PR boundary”.

---

## Completed tasks and their persisted checkbox updates (PR 6)

All nine PR 6 rows were flipped from `- [ ]` to `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (lines 211–219) as the work completed, then re-read to confirm. `git diff --stat` on that file reports exactly `9 insertions(+), 9 deletions(-)` — nine flips and nothing else (52 checked, 113 unchecked of 165).

| # | Task (short) | Persisted update | Evidence |
|---|---|---|---|
| 1 | RED — divergence case: written ≠ effective ⇒ `(measured, fail, sshd_config_divergence)` with both configurations in the verbatim detail | `tasks.md:211` → `- [x]` | `go test -count=1 -run 'TestLocalSshd' ./internal/probe/` → exit 1, 13 call sites: `the registry declares "local.sshd" without a constructor, so no run could measure it` — a behaviour RED, not a compile error |
| 2 | GREEN — `local.sshd` as three separately reportable observations over `CommandRunner`/`FS`, absent binary `(measured, fail, sshd_absent)` with installation named as a later slice | `tasks.md:212` → `- [x]` | Focused run exit 0: `TestLocalSshdReportsThreeSeparateObservations`, `…ReportsWrittenVersusEffectiveDivergence` (both configurations asserted verbatim in observation **and** result detail), `…ReportsAnAbsentBinary` (`not present`, `not part of this run`, `later slice`) |
| 3 | TRIANGULATE — nil runner ⇒ not-measured/`capability_excluded` with the capability named; deny-all runner ⇒ not-measured/`command_denied`; indeterminate control case | `tasks.md:213` → `- [x]` | `TestLocalSshdDegradesWhenTheCommandCapabilityIsMissing` (both command observations, capability named, result `indeterminate`), `TestLocalSshdDegradesWhenTheCommandSeamDenies` (both `command_denied`, and the two reasons are asserted different) |
| 4 | TRIANGULATE — the result is never `pass` when any observation was not measured; all local input read through the injected readers | `tasks.md:214` → `- [x]` | `TestLocalSshdNeverPassesWhenAnObservationWasNotMeasured` (six scripts, 0 passes, result == `Aggregate` of the observations every time), `TestLocalSshdReadsOnlyInjectedReaders` (exact filesystem calls and exact commands) |
| 5 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestLocalSshd` | `tasks.md:215` → `- [x]` | `gofmt -l .` exit 0 (empty output) · `go vet ./...` exit 0 (no output) · focused `TestLocalSshd` exit 0 |
| 6 | RED — `egress_test.go` with the hub cases: supplied target, no-hub not-measured, refused dial, blackholed port | `tasks.md:216` → `- [x]` | `go test -count=1 -run 'TestEgressHub' ./internal/probe/` → exit 1, `FAIL … [build failed]`: `undefined: probe.DefaultDialBudget` at six sites — the egress vocabulary did not exist |
| 7 | GREEN — `egress.hub.direct` in `egress.go` over `EffectiveTargets` and the injected `Dialer` | `tasks.md:217` → `- [x]` | Focused run exit 0: supplied target (`203.0.113.10:2222` and the documented default-port case `hub.example.com:22`), no-hub not-measured with empty target, refused `conn_refused`, blackholed `budget_expired` with the probe's own deadline asserted |
| 8 | TRIANGULATE — not-measured and refused distinguishable, no “blocked” when no attempt was made, no transport viability from hub reachability | `tasks.md:218` → `- [x]` | `TestEgressHubDeniedOrMissingDialCapabilityIsNotMeasured` (the two not-measured reasons differ from each other and from `conn_refused`), `TestEgressHubEstablishesNoTransportViability` (five outcomes, five forbidden tokens, no match) |
| 9 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestEgressHub` | `tasks.md:219` → `- [x]` | `gofmt -l .` exit 0 (empty) · `go vet ./...` exit 0 · focused `TestEgressHub` exit 0 · full suite and `-race` suite exit 0 |

---

## Files changed (PR 6)

| Path | Status | Authored lines | Purpose |
|---|---|---|---|
| `internal/probe/local.go` | modified | +428 / −10 | The `local.sshd` half: the documented local inputs (binary path, written configuration, service query, `sshd -T`), the three observations, `invoke` (the four command outcomes), `sshdServiceActive`, the minimal directive comparison (`parseSSHDConfig`, `sshdDivergences`), plus the shared `runClockNow` and the `newLocalEnv` factory-signature change |
| `internal/probe/local_test.go` | modified | +701 / −1 | The `local.sshd` suite: `scriptedFS`/`scriptedFileInfo`/`scriptedRunner`, the two build helpers, the provisioning wording guard, and nine test functions (three observations, divergence, absence, six-script never-pass table, missing capability, denying seam, service state ×3, unclassifiable input ×2, reads-only-injected) |
| `internal/probe/egress.go` | **added** | 243 | `egress.hub.direct`: `DefaultDialBudget`, the probe, `declaredTarget` (through `EffectiveTargets`), `dialFact` and `dialTimedOut` |
| `internal/probe/egress_test.go` | **added** | 436 | The hub suite: `scriptedConn`/`scriptedDialer`, `hubSeams`/`hubBuild`/`runHub`/`hubText`, and six test functions |
| `internal/probe/classify.go` | modified (enabling edit — deviation #1) | +40 / −3 | Three new observables (`ObsSSHDBinaryPresent`, `ObsSSHDConfigMatches`, `ObsSSHDServiceRunning`, `ObsSSHDServiceNotRunning`) with their `PurposeSSHDConfiguration` rows, and the two `PurposePortReachability` rows for `ObsCapabilityExcluded`/`ObsCommandDenied` |
| `internal/probe/registry.go` | modified (enabling edit — deviation #2) | +47 / −18 | `ProbeFactory` widened to `func(seams Seams, targets TargetInput) Probe`, `ProbesFor` added, `Probes(seams)` kept as the zero-input wrapper, and the `local.sshd` + `egress.hub.direct` constructors filled |
| `openspec/changes/reach-diagnosis-core/tasks.md` | modified | 9 lines changed in place (9 deletions + 9 additions = 18 changed lines) | Nine PR 6 checkboxes `- [ ]` → `- [x]` |
| `openspec/changes/reach-diagnosis-core/apply-progress.md` | modified | this appended section | Cumulative PR 6 evidence; the PR 1–PR 5 sections are untouched |

**Authored code + tests: 1,927 counted lines** (43 + 438 + 702 + 65 + 243 + 436). No file outside the four assigned paths, the two disclosed enabling paths and the two artifact files was created or modified; `README.md`, `PRD.md`, the proposal, the specs, the design, `explore.md`, `research.md`, `preproposal.md` and `openspec/config.yaml` are untouched. No new dependency was added (production code imports only `context`, `errors`, `fmt`, `io/fs`, `strings`, `syscall`, `time`; the tests add `net`, `os`, `reflect`); no linter, no CI configuration and no `go.sum` was introduced.

---

## Test commands run — exact commands and exit status (PR 6)

| # | Command | Exit | Observed output (abridged) |
|---|---|---|---|
| 1 | `go test -count=1 -run 'TestLocalSshd' ./internal/probe/` (task 1 RED) | **1** | `--- FAIL: TestLocalSshdReportsThreeSeparateObservations … the registry declares "local.sshd" without a constructor, so no run could measure it` — 13 such failures across the nine new cases |
| 2 | `gofmt -w internal/probe/local_test.go` then the same focused run (task 2 GREEN) | **0** | all nine `local.sshd` functions PASS (13 subtests included) |
| 3 | mutations M1–M7, one at a time against `/tmp` backups (see “Mutation evidence”) | **1** each | every mutation caught by the case it targets; `diff` confirmed each restore and the package was green again afterwards |
| 4 | `gofmt -l .` · `go vet ./...` · `go test -count=1 -run 'TestLocalSshd' ./internal/probe/` (task 5 gate) | **0** each | `gofmt` empty output; `vet` no output; focused run `ok … 0.009s` |
| 5 | `go test -count=1 -run 'TestEgressHub' ./internal/probe/` (task 6 RED) | **1** | `# …internal/probe_test [build failed]` — `internal/probe/egress_test.go:330:49: undefined: probe.DefaultDialBudget` (six sites) |
| 6 | the same focused run (task 7 GREEN) | **0** | all six `egress.hub.direct` functions PASS (ten subtests) |
| 7 | mutation E7, first attempt: `perl -0pi -e 's/and nothing else/…/'` | **0** | **not caught**, and the reason is instructive: perl's `-0` slurps the file, so an unanchored substitution hit the *header comment* (“…and nothing else egress”) instead of the wording. Re-run anchored on the wording line: exit 1, caught by `TestEgressHubEstablishesNoTransportViability/a_hub_that_accepted_the_connection`. Recorded rather than hidden: a mutation that does not reach the code proves nothing |
| 8 | mutations E1–E9 (E7 re-anchored), one at a time against `/tmp` backups | **1** each | every mutation caught by the case it targets; restores verified with `diff` and the package re-run green |
| 9 | `gofmt -l .` · `go vet ./...` · `go test -count=1 -run 'TestEgressHub' ./internal/probe/` (task 9 gate) | **0** each | `gofmt` empty; `vet` silent; focused run `ok … 0.009s` |
| 10 | `go test -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 0.409s` · `ok …/internal/version 0.007s` |
| 11 | `go test -race -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 1.478s` · `ok …/internal/version 1.018s` |
| 12 | `go test -count=1 -run 'TestLocalSshd\|TestEgressHub' ./internal/probe/` (the work-unit map's `Unit verification` command, **bare pipes**) | **0** | `ok …/internal/probe 0.010s` |
| 13 | `go test -count=1 -run 'TestLocalSshd\|TestEgressHub' ./internal/probe/` (the map's **escaped** form, reproduced as written) | **0** | `ok …/internal/probe 0.009s [no tests to run]` — the artifact's `\|` typo again (PR 4 risk #7, PR 5 note 12): under RE2 it matches a literal pipe and selects nothing |
| 14 | `go test -count=1 -run 'TestLocalSshd\|TestEgressHub' -v ./internal/probe/` (focused inventory) | **0** | 15 top-level functions PASS, 35 `--- PASS` lines including subtests |
| 15 | `go test -count=1 -v ./internal/probe/` (final inventory) | **0** | 56 top-level functions PASS across the package, 226 `--- PASS` lines including subtests |

**Runtime harness**: **N/A as an end-to-end run, exercised in process at both probe boundaries.** There is still no CLI (`cmd/herdr-reach` lands in PR 18/19), no `doctor` wiring (PR 18) and no real network: the slices that would make a real machine reachable are PR 18–PR 20. What *is* exercised is the production path a run takes — `Registry()`/`ProbesFor(seams, input)` → the declared entry → its constructor → `Run` — with scripted filesystem, command runner and dialer seams over the deny-all base of design §6.2, which is the first proof level the design names for a probe. The runner is **not** exercised with these two probes yet (no test runs `Runner.Run` over the real registry): PR 18's doctor owns that, and PR 6's halves are proven at the probe boundary the tasks row names. The unmeasured live step remains the production `CommandRunner` (nil by design) and the real socket shapes.

---

## TDD Cycle Evidence (PR 6)

RED → GREEN → TRIANGULATE → REFACTOR per PR 6 task row, in the two halves the slice merges (`local.sshd` first, then `egress.hub.direct`). Two REDs were produced before the code they exercise existed; both are behaviour failures rather than missing-symbol failures inside a file that already exists.

| Task row | Phase | Evidence produced | Observed failure (RED) | Observed pass (GREEN) |
|---|---|---|---|---|
| 1 RED — divergence | RED | `local_test.go` gained the whole `local.sshd` suite (nine functions) before any `local.sshd` code existed | exit 1: 13 × `the registry declares "local.sshd" without a constructor, so no run could measure it` | n/a |
| 2 GREEN — three observations | GREEN | `local.go`: the documented inputs, `observeBinary`/`observeService`/`observeEffectiveConfig`, `invoke`, `sshdServiceActive`, `parseSSHDConfig`, `sshdDivergences`, `runClockNow`; `classify.go`: the four new observables and their rows; `registry.go`: the `local.sshd` constructor | (previous row) | exit 0: three-observation, divergence and absence cases PASS; full suite green |
| 3 TRIANGULATE — degradation | TRIANGULATE | `TestLocalSshdDegradesWhenTheCommandCapabilityIsMissing`, `…WhenTheCommandSeamDenies` | Cases pass against the GREEN implementation; teeth proven by M2 (capability classified as a pass) and M7 (denial fact dropped), each exit 1 | exit 0 after each restore |
| 4 TRIANGULATE — never a pass, injected readers only | TRIANGULATE | `TestLocalSshdNeverPassesWhenAnObservationWasNotMeasured` (six scripts, counters asserting the table's composition), `TestLocalSshdReadsOnlyInjectedReaders` (exact calls, `HOME` pointed at an empty temp dir), plus `TestLocalSshdServiceStateIsSeparatelyReportable` and `TestLocalSshdUnclassifiableLocalInputIsUnresolved` under the same row | Cases pass against the GREEN implementation; teeth proven by M3 (stopped service as a pass), M4 (absence reported as presence), M5 (result hardcoded to pass) and M6 (an environment read added), each exit 1 | exit 0 after each restore |
| 5 REFACTOR + GATE | REFACTOR | `runClockNow` extracted so both local probes take their timestamps one way; `localEnv.now` delegates to it (no behaviour change); comments re-read against the code | n/a — refactor only | exit 0: `gofmt -l .` (empty), `go vet ./...`, focused suite |
| 6 RED — hub cases | RED | `egress_test.go` written before `egress.go` existed | exit 1: build failure, `undefined: probe.DefaultDialBudget` (six sites) | n/a |
| 7 GREEN — `egress.hub.direct` | GREEN | `egress.go`: `DefaultDialBudget`, the probe, `declaredTarget`, `dialFact`, `dialTimedOut`; `classify.go`: the two reachability rows; `registry.go`: the hub constructor | (previous row) | exit 0: all six hub functions PASS |
| 8 TRIANGULATE — distinguishability, no block, no viability | TRIANGULATE | `TestEgressHubDeniedOrMissingDialCapabilityIsNotMeasured`, `TestEgressHubEstablishesNoTransportViability` | Cases pass against the GREEN implementation; teeth proven by E1, E2, E3, E8 and E7 (re-anchored), each exit 1 | exit 0 after each restore |
| 9 REFACTOR + GATE | REFACTOR | The pass wording rewritten (“…and nothing else”) after the wording guard caught the phrase “nothing about a transport” in the probe's **own** text — the guard was right and the text was fixed, not the guard; comments re-read | exit 1 on the mutation E7 before the wording change (see run 7) | exit 0: `gofmt -l .` (empty), `go vet ./...`, focused suite, full suite, `-race` suite |

**Strict-TDD integrity note.** (a) The `local.sshd` RED is a registry-constructor failure, because a probe the registry cannot build cannot be measured at all: that is the honest first failing state of this task, and it is a *behaviour* failure (the probe is declared and unbuildable), not a missing symbol in the suite. (b) The egress RED is a compile failure — the probe's exported budget constant did not exist — which is the same shape PR 2/PR 3/PR 5 recorded for vocabulary that a slice introduces. (c) Rows 3, 4 and 8 are TRIANGULATE rows: their cases were written against the GREEN implementation, which strict TDD allows for triangulation, and every one of them was mutation-checked rather than trusted (16 mutations, all caught). (d) The wording fix in row 9 was made by changing the probe's text, not by weakening the guard: the test that failed was asserting exactly what the row requires.

---

## Mutation evidence (PR 6)

Every mutation was applied to a `/tmp` backup, run against the focused case, and restored; `diff` then confirmed the restore and the package was re-run green.

| # | Mutation | Case that caught it | Observed failure |
|---|---|---|---|
| M1 | `sshdDivergences` result ignored, so divergence is never reported | `TestLocalSshdReportsWrittenVersusEffectiveDivergence` | exit 1 — the observation reported agreement over a divergent file |
| M2 | `ObsCapabilityExcluded` classified `Measured/Pass` | `TestLocalSshdNeverPassesWhenAnObservationWasNotMeasured`, `…DegradesWhenTheCommandCapabilityIsMissing` | exit 1 — a missing capability was reported as a pass |
| M3 | `ObsSSHDServiceNotRunning` classified `Measured/Pass` | `TestLocalSshdNeverPassesWhenAnObservationWasNotMeasured`, `…ServiceStateIsSeparatelyReportable` | exit 1 — a stopped service passed |
| M4 | An absent binary reported with `ObsSSHDBinaryPresent` | `TestLocalSshdReportsAnAbsentBinary`, the never-pass table | exit 1 — absence became a pass |
| M5 | `Run` hardcoded `Pass`/`ReasonOK` instead of `Aggregate` | three functions | exit 1 — the result disagreed with its observations, including for absent binaries |
| M6 | A `Getenv` call added to the binary check | `TestLocalSshdReadsOnlyInjectedReaders` | exit 1 — the exact-call assertion saw an environment read |
| M7 | The denial fact dropped (`CommandFact(command, nil)`) | `TestLocalSshdDegradesWhenTheCommandSeamDenies` | exit 1 — a denied command stopped being reported as denied |
| E1 | `ObsHubInputMissing` classified `Measured/Fail/conn_refused` | `TestEgressHubWithoutHubInputIsNotMeasuredAsBlocked` | exit 1 — a measurement that never happened became a refusal |
| E2 | A refused dial mapped to `budget_expired` | `TestEgressHubRefusedDialIsAMeasuredFailure` | exit 1 — refusal and expiry collapsed |
| E3 | The dial timeout branch disabled | `TestEgressHubBlackholedPortIsTheProbesOwnBudget` | exit 1 — a blackholed port stopped being `budget_expired` |
| E4 | The probe set no dial deadline (`WithCancel` instead of `WithTimeout`) | same case | exit 1 — `the probe dialed without a deadline, so the expiry cannot be its own` |
| E5 | `DefaultDialBudget = DefaultProbeTimeout` | same case | exit 1 — the ordering assertion that keeps `budget_expired` distinct from `probe_timeout` failed |
| E6 | A hardcoded address dialed instead of the resolved target | `TestEgressHubTargetIsExactlyTheSuppliedAddress` and two more | exit 1 — the target and the dialed set disagreed |
| E7 | The pass wording gained “blocked” (first attempt landed in a comment; re-anchored on the wording line) | `TestEgressHubEstablishesNoTransportViability` | exit 1 — the wording guard found the token |
| E8 | The `ErrSeamDenied` branch disabled | `TestEgressHubDeniedOrMissingDialCapabilityIsNotMeasured` | exit 1 — a denied dial became an unclassified failure |
| E9 | The established connection never closed | `TestEgressHubTargetIsExactlyTheSuppliedAddress` | exit 1 — both success subtests reported the connection was left open |

**Triangulation depth.** Five independent angles on `local.sshd` (the three observations with their own outcomes; the written-versus-effective divergence with both texts asserted in two places; absence with its later-slice wording; six different reasons for an observation to be missing, with the result's own reduction re-derived every time; and the exact filesystem/command call list with the process environment pointed somewhere useless) and four on `egress.hub.direct` (the target as supplied **and** after the documented default port, against the dialer's own record; the four outcomes of a reachability question, each distinguishable from the others; the probe's own deadline measured and ordered against the runner's bound; and five outcomes scanned for a viability claim). **Sixteen mutations, every one caught**, including one that had to be re-anchored before it proved anything.

---

## Deviations from design (PR 6)

| # | Deviation | Why | Design reference | Follow-up owner |
|---|---|---|---|---|
| 1 | **`internal/probe/classify.go` was edited, which is outside the PR 6 file list** (+40/−3): three new observables and four new `PurposeSSHDConfiguration` rows, plus two `PurposePortReachability` rows for `ObsCapabilityExcluded`/`ObsCommandDenied`. | Design §5.1's sshd rows are only `sshd_absent`, `sshd_config_divergence`, `capability_excluded` and `command_denied`: a healthy binary, a running service, an agreeing configuration, and a **dial** that was never attempted are all inexpressible without them, and the table is the only place a reason code may be chosen (PR 2's own record anticipated this). Purely additive, same package, proven by M2/M3 and by the hub degradation cases. | design §5.1 (sshd and reachability rows), design §5.2 (`SSHD_PRESENT_CONFIGURED` requires a pass), PR 2 deviation #2, PR 5 deviation #1 | `sdd-verify` must adjudicate; the orchestrator may prefer to fold the rows into a review of PR 2 |
| 2 | **`internal/probe/registry.go` was edited beyond the `local.sshd` slot** (+47/−18): `ProbeFactory` is now `func(seams Seams, targets TargetInput) Probe`, `ProbesFor(seams, targets)` is the input-aware builder, `Probes(seams)` stays as the zero-input wrapper, and both slots are filled. | The hub's target is run input (design D3), and a `ProbeFactory` that took only `Seams` could not deliver it: the alternatives were a package-level copy of the input (which design §6.1 forbids and the AST guard in `seams_test.go` exists to catch) or re-resolving the hub address inside `Run` from nothing. PR 5's risk #9 and PR 3's file note both predicted this widening as “a normal, reviewable edit in the slice that needs it”. `Probes(seams)` is kept so the existing enumeration case compiles unchanged; it builds the hub probe with no input, which the probe reports as `input_missing_hub` rather than inventing a target (asserted). | design §4 (data-flow sketch), design §6.1, design D3, PR 5 deviation #7 / risk #9 | PR 18's doctor must call `ProbesFor(seams, opts)`; `sdd-verify` should re-word design §4's `Probes(seams)` |
| 3 | A **stopped service** is reported with the `sshd_absent` code, because the closed reason set has no code for “installed but not running”. | Design §3.5 makes adding a code a contract change, and the parent's instruction was explicit: do not invent one. `sshd_absent` is the vocabulary's own “this node is not serving sshd” code; the observation's label (`service state`), its target (`sshd.service,ssh.service`) and its detail (the service manager's verbatim answer) say which half was missing, and the detail deliberately never claims anything needs installing. | design §3.5 (closed set), design §5.1 (sshd rows), design §5.2 (`SSHD_ABSENT`) | **PR 12**: `SSHD_ABSENT`'s conclusion text (“installation is a later slice”) would be wrong for a stopped service; a dedicated `sshd_not_running` code is the honest fix and is recorded as risk #2 |
| 4 | The service query is `systemctl is-active sshd.service ssh.service` — two unit names in one read-only query — and the answer is interpreted from the **verbatim output**, not from the exit status. | Distributions differ in the unit name (Debian/Ubuntu `ssh.service`, the RHEL family `sshd.service`), and guessing from this process's own signals would classify the machine the test runs on. `is-active` exits non-zero when no queried unit is active, so the answer always arrives beside an error on a stopped service: treating every non-nil error as a denial would report a stopped service as an excluded capability. Hence `invoke` distinguishes “denied” (the sentinel) from “answered with a non-zero status” (output present) from “produced nothing classifiable”. | design §5.1 (`command_denied`, `capability_excluded`), design §6.1 (`CommandRunner` consumer), design §6.2 obligation 2 | PR 12 (the rule layer matches the probe state, not the command); a Linux/macOS hand-run is a verify-phase obligation |
| 5 | `DefaultDialBudget = 4s` is a new exported constant, and the ordering `DefaultDialBudget < DefaultProbeTimeout` is asserted by test rather than by construction. | The design requires the probe's own budget to be shorter than the runner's bound (obligation 3, RG-8) but names no value for it. Exporting it lets the later reachability probes share one budget, and the test makes the ordering a checked property instead of a comment. The probe cannot read the runner's *effective* bound (probes never see `Options`), so the claim is a property of the defaults — recorded as risk #4. | design §5.1 obligation 3, design §5.1 (`budget_expired` row), design D9 | PR 7 must reuse the constant; PR 18 must not configure `ProbeTimeout` below it |
| 6 | The written-versus-effective comparison is a **minimal directive-set comparison**: first occurrence wins, names case-insensitive, whitespace collapsed; `Include` is not followed and compiled-in defaults are not modelled. | The design asks for “written ≠ effective” and “show both”, not for a configuration parser. Comparing every directive `sshd -T` prints would report a divergence on every real machine (it prints its defaults too), and inventing include-following or quoting rules would add unverifiable semantics. A directive the written file sets whose effective value differs — or which the effective configuration does not mention at all, which is the “the file in force is not this file” case — is a divergence. | design §5.2 (`SSHD_PRESENT_CONFIG_DIVERGENT`: “names both configurations”), PRD §13 (“Report that the written config is not the effective config, and show both”), R-HR-18 | `sdd-verify` may want the limitation stated in `docs/diagnosis-report.md` (PR 17) |
| 7 | `Result.Target` for `local.sshd` is the documented binary path, while each observation carries the path or unit it actually read. | A probe with three local subjects has no single remote target, and `probe.go` says only a *not-measured* observation carries an empty target. The binary path is the probe's subject and is stable; the per-observation targets are the precise values. | design §3.1 (`Observation.Target`), design §3.3 (`probes[].target`), PR 4 deviation #2 | PR 16's payload mapper must echo this value for `local.sshd` rather than `null` |
| 8 | A **nil `FS`** resolves to `not_measured`/`capability_excluded` for the observations that needed it (binary presence, written configuration), where PR 5's nil `Platform` resolves to `unresolved`/`platform_unknown`. | The table's `ObsCapabilityExcluded` row is scoped to `PurposeSSHDConfiguration`, which is exactly this probe's purpose, so the honest not-measured row already exists here — unlike the platform case, where a platform-scoped row would have been a larger edit to another slice's file. | design §5.1 obligation 2, PR 5 deviation #5 | n/a — the asymmetry is deliberate and now explicit |
| 9 | The two new test files are **external** test packages (`package probe_test`), as in PR 1–PR 5, and the probe's local inputs (paths, command lines, unit names) are repeated as literals in the test. | Reaching the probes through `Registry()`/`ProbesFor` is the production path, and repeating the documented inputs means a rename in `local.go` breaks the suite instead of silently moving what the probe reads. The exact-call assertions are what make “reads only injected readers” checkable. | tasks PR 6 file list; design §8 test plan | n/a |
| 10 | `sshdServiceActive` compares the answer line to `"active"` case-insensitively, i.e. it interprets the service manager's own vocabulary. | Something has to decide whether the answer means a running service, and the alternative (the exit status) is unavailable without an `ExitCode()` accessor the seam does not have. The reason code is still chosen by the classification table, and the verbatim answer is carried in the detail, so the interpretation is visible rather than hidden. | R-HR-07, design §5.1 | PR 12/PR 17 may want a structured service-state field if the reasoning layer must match it |
| 11 | `runClockNow` now backs both local probes, and `newLocalEnv` takes the widened factory signature. | Small dedup inside the same file, no behaviour change: the two probes need the same clock rule (only an injected clock moves a timestamp) and the same constructor shape. Asserted by the existing `TestLocalEnvElapsedComesFromTheInjectedClock`. | design §3.3, PR 5 code | n/a |

---

## Workload and PR boundary (PR 6)

| Field | Value |
|---|---|
| Slice | PR 6 of 20 — “`local.sshd` and `egress.hub.direct`” (WU10 + WU11) |
| PR 6 estimate in `tasks.md` | 410–630 lines (point ≈520) |
| Host attempt ceiling | 2,000 counted changed lines |
| **Actual authored code + tests** | **1,927 counted lines**: `local.go` +428/−10 (438), `local_test.go` +701/−1 (702), `egress.go` 243 (new), `egress_test.go` 436 (new), `classify.go` +40/−3 (43), `registry.go` +47/−18 (65) |
| Artifact changes | `tasks.md` 9 lines changed (9 + 9 = **18** changed lines) + this appended section |
| **Total counted changed lines for the work unit** | **≈2,175** (1,927 authored + 18 checkbox + this section) |
| Chain per-PR cohesion ceiling | 1,000 changed lines (user-approved, revised from 600 on 2026-09-14) |
| Review budget (session canonical) | 400 changed lines |
| Budget status | **Over the slice estimate, over the 1,000-line chain ceiling, over the 400-line session budget and over the 2,000-line attempt ceiling** |
| PR boundary | Starts at PR 5's branch state (`internal/probe` vocabulary + classification table + seams + declared target set + runner + registry with `local.env`) and ends at `internal/probe` compiling and passing with `local.sshd` and `egress.hub.direct` landed and registered. **PR 7 is not started**: no `egress.ssh.*`, `egress.cf.*`, `quic.go`, `tls.go`, `internal/diagnosis/`, `internal/transport/`, `internal/report/`, `internal/doctor/`, `cmd/` or `docs/diagnosis-report.md` exists |
| Rollback boundary | Revert the `local.sshd` additions in `local.go`/`local_test.go` and delete `egress.go`/`egress_test.go`; revert `classify.go`'s +40/−3 and `registry.go`'s +47/−18. The module returns to its PR 5 state with `local.env`, the vocabulary, the classification table, the seams, the declared target set and the runner still green. `tasks.md` lines 211–219 revert to `- [ ]`; this appended section is the only other PR 6 change |
| Rollback independence | Nothing outside `internal/probe` consumes either probe — the doctor wiring lands in PR 18 and the reasoning layer in PR 9+ — so the revert removes no unrelated work and leaves PR 1–PR 5 green |

**Why 1,927 > 1,000, stated honestly, and the boundary the plan already names.** The overage is not padding. 702 lines are `local_test.go` (nine functions: the three-observation case, the divergence case with both configurations asserted twice, absence with its later-slice wording, a six-script never-pass table, the two degradation cases, the ×3 service-state table, the ×2 unclassifiable-input table, and the exact-call isolation case), 436 are `egress_test.go` (six functions covering the target, the not-measured outcome, refusal, the probe's own budget with its deadline measured, the two degradations, and the no-viability wording guard), and 416 of `local.go`'s 428 added lines are the `local.sshd` half — of which roughly 190 are the comments that carry the *why* of the three observations, the four command outcomes and the comparison's stated limits, which is exactly the code a reviewer has to trust. The review-budget rule forbids reaching a number by deleting tests, control cases, triangulation cases, comments, docs or blank lines, and no `size:exception` was assumed, so **nothing was compressed**; the honest count is reported instead.

The boundary the task plan already names is therefore reported rather than applied, measured from the code as written: **PR 6a = WU10 (`local.sshd`)** — `local.go`'s `local.sshd` block (416 lines), `local_test.go`'s `local.sshd` block (696), the sshd rows and observables in `classify.go` (≈25), the `local.sshd` slot and the `ProbeFactory`/`ProbesFor` widening in `registry.go` (≈40) ≈ **1,180 counted lines** (still ≈18 % above the 1,000-line ceiling, almost entirely because the suite that proves the three observations and their degradations belongs with the probe); **PR 6b = WU11 (`egress.hub.direct`)** — `egress.go` (243), `egress_test.go` (436), the two reachability rows in `classify.go` (≈8) and the hub slot in `registry.go` (≈1) ≈ **688 counted lines**, inside the ceiling. The split is strictly ordered: PR 6b's probe resolves its target through the declared target set that PR 3 landed and is registered in the same `registry` table PR 6a extends with the `ProbeFactory`/`ProbesFor` widening, and PR 6b would target PR 6a's branch. **This executor did not split the slice or touch branches**: the parent assigned PR 6 as one unit, and a split moves a review boundary rather than the work. Against the 2,000-line attempt ceiling the work unit is ≈175 lines over; nothing was trimmed to reach it.

---

## Remaining unchecked tasks (PR 6 view)

**Inside this slice: none.** All nine PR 6 checkbox rows are `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (re-read after the edits: **52 checked, 113 unchecked**; 165 rows total; `git diff --stat` on that file shows exactly 9 insertions and 9 deletions). The ownership markers were re-checked: 165 `<!-- sdd-owner: implementation -->` markers for 165 rows, every one terminal.

The change-wide remainder is **113 checkbox lines in PR 7 – PR 20**, which belong to later chained slices on later branches (PR 6 targets PR 5's branch; the chain is `stacked-to-main`) and are **not** part of this work unit. The next unchecked line in the artifact, verbatim, is `tasks.md:226` — PR 7's first row, which is where the next slice resumes:

```
- [ ] **RED** — add one case per outcome for both probes: banner received; a non-SSH banner ⇒ measured fail/`banner_not_ssh`; refused and reset ⇒ measured fail; authoritative resolver negative ⇒ measured fail/`dns_no_such_host` while a resolver timeout ⇒ unresolved/`dns_unresolved` (the two must not collapse); plus each probe's `indeterminate` control case. <!-- sdd-owner: implementation -->
```

PR 7 extends `internal/probe/egress.go` and `egress_test.go` (both now exist) and fills the two SSH-probe registry slots; PR 8 fills the remaining four (`egress.cf.7844`, `egress.cf.443`, `egress.quic`, `tls.interception`, `tls.truststore`).

---

## Risks (PR 6)

| # | Risk | Status / handling |
|---|---|---|
| 1 | 1,927 authored lines against a 1,000-line chain ceiling and ≈175 lines over the 2,000-line attempt ceiling | Disclosed above with the composition, the measured PR 6a/PR 6b boundary the plan already names (≈1,180 / ≈688), and the explicit statement that nothing was compressed to fit |
| 2 | A stopped sshd service reuses `sshd_absent` (deviation #3) | The closed reason set has no code for “installed but not running”, and inventing one is a contract change. The observation's label, target and verbatim detail disambiguate it, and the detail never claims installation. **PR 12's `SSHD_ABSENT` conclusion text must not say “installation is a later slice” for a stopped service**; a dedicated code is the honest fix |
| 3 | The service unit names are a distribution-level guess | Two names, one read-only query, verbatim answer in the detail. A machine that names its unit differently reports a measured negative that the detail explains; the alternative (guessing from process signals) would measure the wrong machine |
| 4 | The `DefaultDialBudget < DefaultProbeTimeout` ordering holds for the **defaults**; a caller could configure a shorter `ProbeTimeout` | Probes never see the runner's `Options`, so the ordering cannot be enforced from inside the probe. Asserted for the defaults, and flagged for PR 18: the doctor must not configure a per-probe bound below `DefaultDialBudget`, or a blackholed hub would be reported as `probe_timeout` instead of `budget_expired` |
| 5 | The written-versus-effective comparison does not follow `Include`, model quoting, or compiled-in defaults | Stated in the code beside the comparison and recorded as deviation #6. PR 17's `docs/diagnosis-report.md` should state the limitation where a user reads what the probe claims |
| 6 | The real socket shapes behind `dialFact` are unexercised | The tests script the errors a real dial returns (`ECONNREFUSED`, `ECONNRESET`, a deadline expiry) and the probe's own deadline is measured, but no live blackholed port or refused connection is dialed in this suite — the project's own rule forbids real egress in tests. The verify phase's hand-run on the motivating network is the remaining evidence, and it already exists for the wider probe suite |
| 7 | `ProbeFactory`'s signature change (deviation #2) | `Probes(seams)` keeps the old call sites compiling, but a future caller that forgets `ProbesFor` gets a hub probe that honestly reports `input_missing_hub` — visible in `run.not_measured`, not silent. PR 18 must use `ProbesFor(seams, opts)` |
| 8 | `Probes(seams)`'s zero-input path is a convenience that can hide a wiring mistake | Documented on the function, asserted by the hub's no-input case, and the honest outcome (not measured, never blocked) is what a forgotten input produces. A reviewer may prefer removing it once `probe_test.go` can be edited |
| 9 | `sshdServiceActive` interprets the service manager's vocabulary (deviation #10) | The reason code still comes from the table and the verbatim answer travels in the detail. If PR 12 needs to match a structured service state, the probe must grow a field rather than the rule grow string parsing |
| 10 | The artifact's focused filters (`-run 'TestLocalSshd\|TestEgressHub'`, and the map's `\|` forms) select nothing under RE2 | Reproduced again (run 13) and recorded since PR 4 risk #7: copy the filters with bare `|`. The nine checkboxes are unaffected |

### Engram mirror note

This file is the **authoritative** artefact. Its Engram mirror is now split in **five** parts because the merged text exceeds the store's 50,000-character content limit: part 1 — the file header plus the PR 1 and PR 2 sections — under the topic `sdd/reach-diagnosis-core/apply-progress`; part 2 — the PR 3 section — under `sdd/reach-diagnosis-core/apply-progress/part2`; part 3 — the PR 4 section — under `sdd/reach-diagnosis-core/apply-progress/part3`; part 4 — the PR 5 section — under `sdd/reach-diagnosis-core/apply-progress/part4`; and part 5 — this PR 6 section — under `sdd/reach-diagnosis-core/apply-progress/part5`.

Every part states the artefact path, this file's byte size and its SHA-256 digest **as recorded when that part was saved**, and that the repository file is authoritative. Parts 1–4 therefore carry their own slice's snapshot and their digests no longer match this file, which has grown by one section since; part 5 carries the merged file's size and digest as of this run. A mirror part is a readable convenience copy of this artefact, not a second source of truth: read this file for the complete, current record. The PR 3, PR 4 and PR 5 sections' own older notes still describe the earlier two-, three- and four-part shapes; they are earlier sections of this artefact and are deliberately not rewritten here.

---

# Apply Progress — PR 7 of 20 — WU12 + WU13 · The egress reachability probes: public SSH and the Cloudflare edge regions

**Change**: `reach-diagnosis-core` · **Slice**: **PR 7 of 20** — "the egress reachability probes: public SSH and the Cloudflare edge regions" (WU12 + WU13)
**Branch**: `feat/probe-egress-reachability`, stacked on PR 6's branch `feat/probe-sshd-hub` (chain strategy `stacked-to-main`, so PR 7 targets PR 6's branch and **not** `main`)
**Date**: 2026-09-14 · **Artifact store**: `both` (this file + Engram mirror, split as the mirror note at the end records)
**Strict TDD**: active — `openspec/config.yaml` declares `strict_tdd: true` with runner `go test ./...`; RED → GREEN → TRIANGULATE → REFACTOR followed for both halves of this slice
**Skill resolution**: `paths-injected` — read `/home/luisalt20/.config/opencode/skills/go-testing/SKILL.md` and `/home/luisalt20/.config/opencode/skills/work-unit-commits/SKILL.md` before writing code; no registry fallback was needed
**Delivery path consumed**: `auto-chain` / `stacked-to-main` — this run implements **only** the assigned slice and stops at its PR boundary; the PR 1–PR 6 sections above are preserved unchanged
**Commit status**: nothing committed, staged, pushed or branched by this phase; the work is left in the working tree for the orchestrator
**How to read this file**: PR 7's entry is the section below. The current change-wide remainder is restated at the end of this section.

---

## Structured status consumed (PR 7)

| Field | Value |
|---|---|
| `schemaName` / `schemaVersion` | `gentle-ai.sdd-status` / `2` |
| `changeName` | `reach-diagnosis-core` |
| `nextRecommended` | `apply` |
| `applyState` | `ready` |
| `dependencies.apply` / `.verify` / `.archive` | `ready` / `blocked` / `blocked` |
| `actionContext.mode` | `repo-local` |
| `actionContext.workspaceRoot` | `/home/luisalt20/projects/close/herdr-reach` |
| `actionContext.allowedEditRoots` | `["/home/luisalt20/projects/close/herdr-reach"]` — every file written lives inside it |
| `artifactStore` | `both` declared by the parent prompt (native `openspec`); files written under `openspec/changes/reach-diagnosis-core/` and mirrored to Engram |
| `taskProgress` before this run | 165 total / 52 completed / 113 pending (PR 1 + PR 2 + PR 3 + PR 4 + PR 5 + PR 6) |
| `taskProgress` after this run | 165 total / **60 completed** / 105 pending |
| `actionContext` warnings | none |
| Work-unit ownership markers | all eight PR 7 rows carry the terminal `<!-- sdd-owner: implementation -->` marker; the whole file still holds 165 markers for 165 checkbox rows, none malformed, duplicate, unsupported or non-terminal |
| `applyState: all_done`? | no — implementation continues, so editing was permitted |

**Attempt context**: the status reports one bounded attempt already active for this work unit (`token sha256:7f46227bd0aee8f37dc39aa4d05cacce0f1f61d5c364681180edba4687eb4843`) with a **3,000**-changed-line ceiling. Per the parent prompt the parent owns `sdd-attempt acquire`/`settle`; this executor did **not** call the native attempt command. The counted-line position against that ceiling is in "Workload and PR boundary".

---

## Completed tasks and their persisted checkbox updates (PR 7)

All eight PR 7 rows were flipped from `- [ ]` to `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (lines 226–233) as the work completed, then re-read to confirm. `git diff --stat` on that file reports exactly `8 insertions(+), 8 deletions(-)` — eight flips and nothing else (60 checked, 105 unchecked of 165).

| # | Task (short) | Persisted update | Evidence |
|---|---|---|---|
| 1 | RED — one case per outcome for both SSH probes: banner received; non-SSH banner ⇒ measured fail/`banner_not_ssh`; refused and reset ⇒ measured fail; authoritative resolver negative ⇒ measured fail/`dns_no_such_host` vs resolver timeout ⇒ unresolved/`dns_unresolved`; plus each probe's `indeterminate` control case | `tasks.md:226` → `- [x]` | `go test -count=1 -run 'TestEgressSsh' ./internal/probe/` → **exit 1**, 53 behaviour failures, every one `the registry built no "egress.ssh.known"/"egress.ssh.443" probe, so no run could measure it` — the probes are declared and unbuildable, not missing symbols |
| 2 | GREEN — implement both probes in `egress.go` through the classification table only | `tasks.md:227` → `- [x]` | Focused run **exit 0**, 57 `--- PASS` lines including subtests: banner (pass/`ok` with the banner verbatim), non-SSH banner, refused, reset, the two name-level outcomes on both surfaces, the two not-measured capabilities and the dial-budget expiry |
| 3 | TRIANGULATE — partial-banner case: a truncated read that still carries an SSH identification string classifies as SSH, verbatim detail preserved (R-HR-07) | `tasks.md:228` → `- [x]` | `TestEgressSshProbesPreserveAPartialIdentificationString`: two packets `SSH` + `-2.0-OpenS` ⇒ pass with `"SSH-2.0-OpenS"` in the detail; a read truncated to `SSH-` ⇒ pass; `SSH` alone ⇒ `banner_not_ssh`; a reset and a deadlined read beside them; the read deadline asserted per case |
| 4 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestEgressSsh` | `tasks.md:229` → `- [x]` | `gofmt -l .` exit 0 (empty output) · `go vet ./...` exit 0 (no output) · focused `TestEgressSsh` exit 0 |
| 5 | RED — both regions pass; one region fails while the other passes ⇒ the split is visible per observation and the aggregate is `fail`; a fully blackholed target ⇒ measured fail/`budget_expired`; plus the `indeterminate` control case | `tasks.md:230` → `- [x]` | `go test -count=1 -run 'TestEgressCF' ./internal/probe/` → **exit 1**, 29 behaviour failures, every one `the registry built no "egress.cf.7844"/"egress.cf.443" probe, so no run could measure it` |
| 6 | GREEN — both probes produce one observation per region (`region1`, `region2`) while the registry still contains exactly ten probes (D10) | `tasks.md:231` → `- [x]` | Focused run **exit 0**: two observations labelled `tcp 7844 region1` / `tcp 7844 region2` (and `tcp 443 regionN`), both declared addresses dialed, both connections closed, `len(probe.Registry()) == 10` asserted, one registry entry per probe |
| 7 | TRIANGULATE — the declared target set for these probes carries both regions, an override that replaces it is reflected per observation and the closed-set assertion still holds | `tasks.md:232` → `- [x]` | `TestEgressCFDeclaredRegionsSurviveAnOverride`: both regions and their labels read from `DeclaredTargets()`; an override of `egress.cf.443` yields exactly one observation and one dial at `198.51.100.7:443`; the effective set total is `declared − 2 + 1` with every other probe unchanged; a per-region name split (region1 `dns_no_such_host`, region2 pass) surfaces per observation |
| 8 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestEgressCF` | `tasks.md:233` → `- [x]` | `gofmt -l .` exit 0 (empty) · `go vet ./...` exit 0 · focused `TestEgressCF` exit 0 · full suite and `-race` suite exit 0 |

---

## Files changed (PR 7)

| Path | Status | Authored lines | Purpose |
|---|---|---|---|
| `internal/probe/egress.go` | modified | +540 / −10 (550) | The file header rewritten to describe the three egress question families; the two SSH subject/prefix constants and the identification-string bound; `dialFact` split into a hub wrapper plus the shared `dialFactFor(subject, …)`; the shared reachability helpers (`resolverFact`, `isAuthoritativeNegative`, `isNameResolutionError`, `declaredEndpoints`, `declaredEndpoint`, `reachAttempt`); the `egressSSH` probe with `readIdentificationString`, `carriesSSHIdentification` and `bannerFact`; the `egressCF` probe measuring one observation per declared region |
| `internal/probe/egress_test.go` | modified | +1165 / −8 (1173) | The PR 7 fixtures (`scriptedResolver`, `scriptedBannerConn` with chunked reads, `scriptedDialFunc`, `cfDialer`, `reachSeams`, `egressBuild`, `runSsh`/`runCF`, `declaredAddresses`/`declaredAddress`, `declaredTargetsFor`, `cfDialerFunc`, the two probe tables) and thirteen test functions: seven for the SSH half and six for the Cloudflare half. The two hub assertions that read the whole result text were renamed `hubText` → `resultText`, because the helper is no longer hub-specific |
| `internal/probe/classify.go` | modified (enabling edit — deviation #1) | +21 / −1 (22) | One new observable (`ObsSSHBannerReceived`) with its `PurposePortReachability` row, and the two `PurposeNameResolution` rows for `ObsCapabilityExcluded`/`ObsCommandDenied` that the name-resolution step needs (design §5.1 obligation 2) |
| `internal/probe/registry.go` | modified (enabling edit — deviation #2) | +14 / −11 (25) | The four PR 7 constructors filled in (`egress.ssh.known`, `egress.ssh.443`, `egress.cf.7844`, `egress.cf.443`); the names, the kinds and the order are untouched, and the registry still holds exactly ten entries |
| `openspec/changes/reach-diagnosis-core/tasks.md` | modified | 8 lines changed in place (8 deletions + 8 additions = 16 changed lines) | Eight PR 7 checkboxes `- [ ]` → `- [x]` |
| `openspec/changes/reach-diagnosis-core/apply-progress.md` | modified | this appended section | Cumulative PR 7 evidence; the PR 1–PR 6 sections are untouched |

**Authored code + tests: 1,770 counted lines** (550 + 1,173 + 22 + 25). No file outside the two assigned paths, the two disclosed enabling paths and the two artifact files was created or modified; `README.md`, `PRD.md`, the proposal, the specs, the design, `explore.md`, `research.md`, `preproposal.md` and `openspec/config.yaml` are untouched. No new dependency was added (production code imports only `bytes`, `context`, `errors`, `fmt`, `io`, `net`, `strconv`, `strings`, `syscall`, `time` — all standard library; the tests add `io`, `net`, `reflect`); no linter, no CI configuration and no `go.sum` was introduced.

---

## Test commands run — exact commands and exit status (PR 7)

| # | Command | Exit | Observed output (abridged) |
|---|---|---|---|
| 1 | `go test -count=1 -run 'TestEgressSsh' ./internal/probe/` (task 1 RED) | **1** | 53 failures, all `the registry built no "egress.ssh.known" probe, so no run could measure it` (the 443 probe likewise) — a behaviour RED: the probes are declared, registered by name and unbuildable |
| 2 | `gofmt -w internal/probe/egress.go internal/probe/egress_test.go internal/probe/classify.go internal/probe/registry.go` then the same focused run (task 2 GREEN) | **0** | `ok …/internal/probe 0.009s`; 57 `--- PASS` lines including subtests |
| 3 | `go test -count=1 ./...` (after the SSH half) | **0** | `ok …/internal/probe 0.412s` · `ok …/internal/version 0.008s` — the hub suite still green after `dialFact` was split |
| 4 | mutations S1–S5, one at a time against `/tmp` backups (see "Mutation evidence") | **1** each | every mutation caught by the case it targets; `diff` confirmed each restore and the package was green again afterwards |
| 5 | `go test -count=1 -run 'TestEgressCF' ./internal/probe/` (task 5 RED) | **1** | 29 failures, all `the registry built no "egress.cf.7844" probe, so no run could measure it` (the 443 probe likewise) |
| 6 | the same focused run (task 6 GREEN) | **0** | `ok …/internal/probe 0.009s` |
| 7 | mutations C1–C5, one at a time against `/tmp` backups | **1** each | every mutation caught by the case it targets; restores verified with `diff` and the package re-run green |
| 8 | `gofmt -l .` · `go vet ./...` · `go test -count=1 -run 'TestEgressSsh' ./internal/probe/` · `-run 'TestEgressCF'` (tasks 4 and 8 gates) | **0** each | `gofmt` empty output; `vet` no output; both focused runs `ok` |
| 9 | `go test -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 0.418s` · `ok …/internal/version 0.009s` |
| 10 | `go test -race -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 1.497s` · `ok …/internal/version 1.018s` |
| 11 | `go test -count=1 -run 'TestEgressSsh|TestEgressCF' ./internal/probe/` (the map's two filters, **bare pipe**) | **0** | `ok …/internal/probe 0.016s` |
| 12 | `go test -count=1 -run 'TestEgressSsh\|TestEgressCF' ./internal/probe/` (the map's **escaped** form, reproduced as written) | **0** | `ok …/internal/probe 0.008s [no tests to run]` — the artifact's `\|` typo again (PR 4 risk #7, PR 5 note 12, PR 6 run 13): under RE2 it matches a literal pipe and selects nothing |
| 13 | `go test -count=1 -run 'TestEgressSsh|TestEgressCF|TestEgressHub' -v ./internal/probe/` (focused inventory) | **0** | 19 top-level functions PASS (6 hub + 7 SSH + 6 Cloudflare) |
| 14 | `go test -count=1 -v ./internal/probe/` (final inventory) | **0** | 69 top-level functions PASS across the package, 248 `--- PASS` lines including subtests |

**Runtime harness**: **N/A as an end-to-end run, exercised in process at both new probe boundaries.** There is still no CLI (`cmd/herdr-reach` lands in PR 18/19), no `doctor` wiring (PR 18) and no real network: the slices that would make a real machine reachable are PR 18–PR 20. What *is* exercised is the production path a run takes — `Registry()`/`ProbesFor(seams, input)` → the declared entry → its constructor → `Run` — with scripted resolver, dialer, connection and clock seams over the deny-all base of design §6.2, which is the first proof level the design names for a probe. The unmeasured live steps remain the real socket shapes (a real SSH banner, a real NXDOMAIN, a real blackholed region) and the real registry interaction with the runner, both of which belong to the verify phase's hand-run and to PR 18's doctor.

---

## TDD Cycle Evidence (PR 7)

RED → GREEN → TRIANGULATE → REFACTOR per PR 7 task row, in the two halves the slice merges (the two SSH destination probes first, then the two per-region Cloudflare edge probes, which is also the boundary the task plan names). Both REDs are behaviour failures — the probe is declared, registered and unbuildable — rather than missing symbols inside a file that already exists.

| Task row | Phase | Evidence produced | Observed failure (RED) | Observed pass (GREEN) |
|---|---|---|---|---|
| 1 RED — SSH cases | RED | `egress_test.go` gained the PR 7 fixtures and the seven SSH test functions before any PR 7 production code existed | exit 1: 53 × `the registry built no "egress.ssh.known"/"egress.ssh.443" probe, so no run could measure it` | n/a |
| 2 GREEN — both probes | GREEN | `egress.go`: the constants, the `dialFactFor` split, `resolverFact`/`isAuthoritativeNegative`/`isNameResolutionError`, `declaredEndpoints`/`declaredEndpoint`, `reachAttempt`, the `egressSSH` probe, `readIdentificationString`, `carriesSSHIdentification`, `bannerFact`; `classify.go`: the banner observable and three rows; `registry.go`: the two SSH constructors | (previous row) | exit 0: 57 PASS lines; banner pass, non-SSH banner, refused, reset, both name-level outcomes on both surfaces, both not-measured capabilities, dial-budget expiry |
| 3 TRIANGULATE — partial banner | TRIANGULATE | `TestEgressSshProbesPreserveAPartialIdentificationString` (five cases × two probes), plus the read-surface reset and deadlined-read cases | Cases pass against the GREEN implementation; teeth proven by S5 (the read stops after one chunk), S1 (the identification string is never recognised) and S4 (no read deadline), each exit 1 | exit 0 after each restore |
| 4 REFACTOR + GATE | REFACTOR | `dialFact` split into the hub wrapper plus `dialFactFor` so four probes share one mapping and the hub's text is byte-identical (the hub suite re-ran green); comments re-read against the code | n/a — refactor only (the hub suite is the regression gate) | exit 0: `gofmt -l .` (empty), `go vet ./...`, focused suite, full suite |
| 5 RED — Cloudflare cases | RED | `egress_test.go` gained `declaredTargetsFor`, `cfDialerFunc` and the six Cloudflare test functions, with no Cloudflare production code in place | exit 1: 29 × `the registry built no "egress.cf.7844"/"egress.cf.443" probe, so no run could measure it` | n/a |
| 6 GREEN — both probes | GREEN | `egress.go`: the `egressCF` probe (`Run` with the per-region reduction, `observe` with one measurement per declared region); `registry.go`: the two Cloudflare constructors | (previous row) | exit 0: both regions observed separately, the split visible, the blackholed budget measured, the indeterminate control table |
| 7 TRIANGULATE — declared set and override | TRIANGULATE | `TestEgressCFDeclaredRegionsSurviveAnOverride` (both regions declared, an override replaced not appended, the closed-set total recomputed, a per-region name split) | Cases pass against the GREEN implementation; teeth proven by C4 (the override is ignored) and C3 (the result names only the first region), each exit 1 | exit 0 after each restore |
| 8 REFACTOR + GATE | REFACTOR | `strings.Join` extracted for the multi-region target and detail exactly as `local.sshd` reports its three observations; comments re-read | exit 1 on C1 (the split stops after the first region) and C2 (the aggregate hardcoded to pass) before the gate | exit 0: `gofmt -l .` (empty), `go vet ./...`, focused suite, full suite, `-race` suite |

**Strict-TDD integrity note.** (a) Both REDs are registry-constructor failures, because a probe the registry cannot build cannot be measured at all: that is the honest first failing state of these rows, and it is a *behaviour* failure (declared and unbuildable), not a missing symbol in the suite. (b) Rows 3 and 7 are TRIANGULATE rows: their cases were written against the GREEN implementation, which strict TDD allows for triangulation, and every one of them was mutation-checked rather than trusted (ten mutations, all caught). (c) Row 4's refactor was verified by the landed hub suite rather than by new assertions, so the refactor could not change the hub's text: the suite's exact-wording assertions are the gate. (d) No comment, control case or triangulation case was removed or compressed to reach any number; the counts are reported in "Workload and PR boundary".

---

## Mutation evidence (PR 7)

Every mutation was applied to a `/tmp` backup, run against the focused case, and restored; `diff` then confirmed the restore and the package was re-run green.

| # | Mutation | Case that caught it | Observed failure |
|---|---|---|---|
| S1 | `bannerFact`'s SSH branch disabled (`case false`) | the five SSH banner cases | exit 1 — a port speaking SSH stopped being a pass |
| S2 | The dial's name-resolution branch removed from `reachAttempt` | `TestEgressSshProbesDistinguishANameThatDoesNotExistFromAResolverThatDidNotAnswer` (dialer surface) | exit 1 — a name-level dial failure became an internal error instead of `dns_no_such_host`/`dns_unresolved` |
| S3 | `isAuthoritativeNegative` always false | the same case (resolver surface) | exit 1 — the authoritative negative collapsed into `dns_unresolved` |
| S4 | The identification-string read set no deadline | `TestEgressSshProbesPreserveAPartialIdentificationString` | exit 1 — `the probe read the identification string without a deadline` |
| S5 | The read stopped after its first chunk (`for total == 0`) | the two-packet partial-banner case | exit 1 — a partial identification string stopped being recognised |
| C1 | The region loop stopped after the first region (`targets[:1]`) | the two-region and split cases | exit 1 — the second region had no observation |
| C2 | `Run` hardcoded `Pass`/`ReasonOK` instead of `Aggregate` | the split and indeterminate cases | exit 1 — a failing region was aggregated as a pass |
| C3 | `Result.Target` set to the first observation's target | the declared-set assertion | exit 1 — the second region's address disappeared from the result |
| C4 | `declaredEndpoints` read the declarations instead of the effective set | the override case | exit 1 — the override was ignored |
| C5 | The per-region connection never closed | the region-pass cases | exit 1 — `the connection to … was left open` |

**Triangulation depth.** Seven independent angles on the SSH probes (the banner as the positive signal against the dialer's own record of the declared address; the non-SSH banner as a measured negative; refusal and reset as two distinct measured negatives; the name-level pair on both the resolver and the dialer surface; two not-measured capabilities kept distinguishable; a chunked, truncated and deadlined read; and fourteen outcome/subject pairs scanned for a transport claim) and six on the Cloudflare probes (per-region observations matched to the declaration; the split in both positions; the blackholed budget with its deadline measured; five unanswered outcomes scanned for a block; the declared set against an override with the closed-set total recomputed; and five outcome sets scanned for a transport claim). **Ten mutations, every one caught.**

---

## Deviations from design (PR 7)

| # | Deviation | Why | Design reference | Follow-up owner |
|---|---|---|---|---|
| 1 | **`internal/probe/classify.go` was edited, which is outside the PR 7 file list** (+21/−1): one new observable (`ObsSSHBannerReceived`, `ssh_banner_received`) with its `PurposePortReachability` pass row, and two `PurposeNameResolution` rows for `ObsCapabilityExcluded`/`ObsCommandDenied`. | Design §5.1 prints the negative half only ("TCP connected, banner is not SSH"). The positive half of the same question is inexpressible without an observable of its own, and borrowing `tcp_established` would make two different raw facts share one row. The two name-resolution rows are the same gap: the resolver step is a probe step that can have no capability or a denied seam, and obligation 2 requires those to be not-measured facts with their own codes rather than a fall-through. Purely additive, same package, proven by S1 and by the not-measured controls. | design §5.1 (SSH banner row, obligation 2), PR 2 deviation #2, PR 6 deviation #1 | `sdd-verify` must adjudicate; the orchestrator may prefer to fold the rows into a review of PR 2 |
| 2 | **`internal/probe/registry.go` was edited beyond the two SSH slots, in the same slice** (+14/−11): all four PR 7 constructors are filled. | Four slots, four constructors, one edit each; the names, kinds and order are untouched and the enumeration test still asserts exactly ten. Filling only the SSH slots and then re-editing the same table in the same slice would add a review hop with no behavioural difference. | design §4, design D10, PR 5 deviation #7 / PR 6 deviation #2 | n/a — the CF half is part of this slice; PR 8 fills the remaining four slots |
| 3 | The SSH probes **pre-resolve** the declared name through the injected `Resolver` and then dial the **declared** address, so a production dialer resolves the name a second time inside the dial. | The design names `Resolver` as the seam of "probes whose declared target is a name" (§6.1), and the spec's replay scripts dialers **and** resolvers: leaving it unused would make the seam dead and the disambiguation requirement untestable at the probe boundary. Dialing the resolved address instead would put an address into the dialed set that the declaration does not contain, breaking "the dialed set equals the declared set" (R-HR-NF-10, design §6.2 level 2). The cost — one extra lookup per probe in production — is paid deliberately and stated in the code. | design §6.1, design §6.2 level 2, R-HR-NF-10, PRD §1.1 | `sdd-verify` may want the cost restated in `docs/diagnosis-report.md` (PR 17) |
| 4 | A dialer's **own** name-resolution failure is classified under `PurposeNameResolution` (and so is a resolver-seam failure), while a port-level dial failure stays under `PurposePortReachability`. | A name that does not exist is the same fact whichever seam surfaced it, and letting two seams produce two codes for it is the wrong-classification risk RG-8 names. The hub probe is **not** changed to match: its target is run input, its slice is PR 6, and its tests pin the landed behaviour, so a name-shaped `--hub` that does not resolve still reports `internal_error` today (risk #1). | design §5.1 (the two name-resolution rows, RG-8), design §5.1 obligation 3 | **PR 7's mapping is the one to copy**; the hub's is a residual gap recorded below |
| 5 | `Result.Target` for a multi-region probe is the declared set rendered in declaration order (`region1…:7844, region2…:7844`), not `null` and not the first region. | design §3.3 types `probes[].target` as one string, and a probe with two endpoints has no single target; naming only the first would hide the region the probe measured, and `null` would read as "no target" for a probe that measured two. Each observation still carries its own exact address, exactly as `local.sshd` does for its three subjects (PR 6 deviation #7). | design §3.1 (`Observation.Target`), design §3.3 (`probes[].target`), design D10, PR 6 deviation #7 | PR 16's payload mapper must echo this string for the two edge probes rather than `null` |
| 6 | The SSH identification-string read is a **bounded accumulation**: it reads until it holds an identification string, a line terminator, or `sshBannerLimit` bytes, with a deadline from the run's clock (falling back to the wall clock when none was injected). | A single `Read` can return `SSH` before `-2.0-…`, which would classify a real SSH server as `banner_not_ssh`; RFC 4253 permits a server to send other lines first, and the bound and the line scan are what handle both. `net.Conn` carries no context, so the deadline is the only bound a read can be given, and the fallback is what keeps a run with no injected clock from waiting forever. | RFC 4253 §4.2, R-HR-07, design §3.3 | `sdd-verify` may want the bound and the line scan stated in the doc (PR 17) |
| 7 | `hubText` was renamed `resultText` and its two call sites updated. | The helper joins every string a result carries and is used by three probe groups now; a name that says "hub" on a Cloudflare assertion would mislead a reader of the suite. Two call sites, same file, no assertion changed. | tasks PR 7 file list (this file is assigned to the slice) | n/a |
| 8 | The two not-measured capabilities are asserted with a `wantContains` word ("no resolver", "no dialer", "denied") rather than by comparing the whole detail string. | The detail carries socket and resolver wording that exists to be quotable, and asserting it in full would make the suite a second copy of the code. The stable assertions stay structural (resolution, verdict, reason, target, the two reasons being different); one word per case proves the text names what was missing. | R-HR-07, design §5.1 obligation 2 | n/a |

---

## Workload and PR boundary (PR 7)

| Field | Value |
|---|---|
| Slice | PR 7 of 20 — "the egress reachability probes: public SSH and the Cloudflare edge regions" (WU12 + WU13) |
| PR 7 estimate in `tasks.md` | 400–600 lines (point ≈500) |
| Host attempt ceiling | 3,000 counted changed lines |
| **Actual authored code + tests** | **1,770 counted lines**: `egress.go` +540/−10 (550), `egress_test.go` +1165/−8 (1,173), `classify.go` +21/−1 (22), `registry.go` +14/−11 (25) |
| Artifact changes | `tasks.md` 8 lines changed (8 + 8 = **16** changed lines) + this appended section |
| **Total counted changed lines for the work unit** | **≈1,796** (1,770 authored + 16 checkbox + this section) |
| Chain per-PR cohesion ceiling | 1,000 changed lines (user-approved, revised from 600 on 2026-09-14) |
| Review budget (session canonical) | 400 changed lines |
| Budget status | **Over the slice estimate, over the 1,000-line chain ceiling and over the 400-line session budget; inside the 3,000-line attempt ceiling** |
| PR boundary | Starts at PR 6's branch state (`internal/probe` vocabulary + classification table + seams + declared target set + runner + registry with `local.env`, `local.sshd`, `egress.hub.direct`) and ends at `internal/probe` compiling and passing with the two public-SSH probes and the two per-region Cloudflare edge probes landed and registered (six of the ten constructors filled). **PR 8 is not started**: no `quic.go`, `tls.go`, `internal/diagnosis/`, `internal/transport/`, `internal/report/`, `internal/doctor/`, `cmd/` or `docs/diagnosis-report.md` exists, and the four remaining registry slots (`egress.quic`, `tls.interception`, `tls.truststore`) are still nil |
| Rollback boundary | Revert `egress.go`'s PR 7 additions (the header, the constants, the `dialFactFor` split, the shared reachability helpers, the `egressSSH` and `egressCF` groups), revert `egress_test.go` to its PR 6 state (including the two `hubText` call sites), revert `classify.go`'s +21/−1 and `registry.go`'s four constructor slots. The module returns to its PR 6 state with the hub probe, `local.env`, `local.sshd`, the vocabulary, the classification table, the seams, the declared target set and the runner still green. `tasks.md` lines 226–233 revert to `- [ ]`; this appended section is the only other PR 7 change |
| Rollback independence | Nothing outside `internal/probe` consumes either probe group — the doctor wiring lands in PR 18 and the reasoning layer in PR 9+ — so the revert removes no unrelated work and leaves PR 1–PR 6 green |

**Why 1,770 > 1,000, stated honestly, and the boundary the plan already names.** The overage is not padding. 1,173 lines are `egress_test.go` — 272 of them the PR 7 fixtures (a chunked banner connection, an address-scripted dialer, a per-host resolver, the declared-set readers) and the rest thirteen test functions that between them script 53 SSH subtests and 29 Cloudflare subtests over the deny-all base — and 550 are `egress.go`, of which roughly 240 are the comments that carry the *why* of the two purposes, the four dial outcomes, the two name-level outcomes and the per-region split, which is exactly the code a reviewer has to trust. The review-budget rule forbids reaching a number by deleting tests, control cases, triangulation cases, comments, docs or blank lines, and no `size:exception` was assumed, so **nothing was compressed**; the honest count is reported instead.

Measuring the code as written, the boundary the task plan names splits this slice into:

- **PR 7a = WU12 (the two SSH destination probes)** — `egress.go`: the header, the constants, the `dialFactFor` split, the shared reachability helpers (`resolverFact` … `reachAttempt`, 184 lines) and the `egressSSH` group (199 lines), ≈ **442 counted lines**; `egress_test.go`: the PR 7 fixtures (272 lines) and the seven SSH test functions (468 lines), ≈ **740 counted lines**; `classify.go`'s banner observable and name-resolution rows (≈22) and `registry.go`'s two SSH constructors (≈13). Total ≈ **1,217 counted lines**, still ≈22 % above the 1,000-line ceiling — almost entirely because the shared fixtures and the six-way outcome table belong with the first probe group that needs them.
- **PR 7b = WU13 (the two per-region Cloudflare edge probes)** — `egress.go`'s `egressCF` group (98 lines) and `egress_test.go`'s six Cloudflare test functions (407 lines), ≈ **505 counted lines**, inside the ceiling. PR 7b depends on PR 7a for the shared `reachAttempt`/`declaredEndpoints` helpers and the `resolverFact` mapping, and would target PR 7a's branch.

**This executor did not split the slice or touch branches**: the parent assigned PR 7 as one unit, and a split moves a review boundary rather than the work. Against the 3,000-line attempt ceiling the work unit is comfortably inside it (≈1,796).

---

## Remaining unchecked tasks (PR 7 view)

**Inside this slice: none.** All eight PR 7 checkbox rows are `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (re-read after the edits: **60 checked, 105 unchecked**; 165 rows total; `git diff --stat` on that file shows exactly 8 insertions and 8 deletions). The ownership markers were re-checked: 165 `<!-- sdd-owner: implementation -->` markers for 165 rows, every one terminal.

The change-wide remainder is **105 checkbox lines in PR 8 – PR 20**, which belong to later chained slices on later branches (PR 7 targets PR 6's branch; the chain is `stacked-to-main`) and are **not** part of this work unit. The next unchecked line in the artifact, verbatim, is `tasks.md:240` — PR 8's first row, which is where the next slice resumes:

```
- [ ] **RED** — write `quic_test.go` with the four D8 outcomes and all four reason codes: any UDP reply ⇒ `(measured, pass, udp_response_received)`; silence ⇒ `(unresolved, udp_silence)`; ICMP port-unreachable ⇒ `(measured, fail, udp_unreachable)`; any other socket error ⇒ `(unresolved, udp_error_unclassified)`; plus the `indeterminate` control case. <!-- sdd-owner: implementation -->
```

PR 8 adds `internal/probe/quic.go`, `quic_test.go`, `tls.go` and `tls_test.go` (its own files, not this one) and fills the `egress.quic` and `tls.interception` registry slots. PR 9 fills `tls.truststore`, the last of the ten.

---

## Risks (PR 7)

| # | Risk | Status / handling |
|---|---|---|
| 1 | A **name-shaped `--hub`** whose name does not resolve is still reported by the hub probe as `internal_error`/unresolved, while the four PR 7 probes report `dns_no_such_host`/`dns_unresolved` for the same fact (deviation #4) | The hub's mapping belongs to PR 6's reviewed slice, so this run did not silently change it. The gap is real: a user with `--hub nosuch.example:22` reads an internal error where the honest answer is "the name does not exist". **Smallest fix**: route the hub's dial errors through the same DNS branch `reachAttempt` uses; it is a three-line change in `egress.go` plus one hub case, and it belongs in the slice that reviews it |
| 2 | A name-level `dns_no_such_host` is a **measured failure** at the probe layer | Design §5.1 calls it "a fact about the name, not about the network" and the diagnosis layer (PR 10+) must not read it as "the destination is blocked". The probe's detail says exactly that; PR 10's rule table is where the conclusion is worded, and PR 11's counterfactual cases must include this state |
| 3 | The declared region targets are **constants** (`region1/2.v2.argotunnel.com`) that Cloudflare can change | Measured, not assumed: a changed host reports `dns_no_such_host` with the resolver's own wording, and `--target egress.cf.443=…` replaces the endpoint without a rebuild (asserted). The constants are also repeated as literals in the suite, so a rename in `targets.go` breaks the suite rather than silently moving what is measured |
| 4 | Two bounded waits per SSH probe inside one runner bound (dial 4 s + banner read 4 s vs the runner's 10 s) | `2*probe.DefaultDialBudget > probe.DefaultProbeTimeout` is asserted, so a future edit to either constant fails the suite instead of turning a blackholed port into `probe_timeout`. Extends PR 6 risk #4 (a caller must not configure `ProbeTimeout` below the probe's own budget) |
| 5 | The SSH banner bound (255 bytes) and the line scan are local decisions | Each is stated in the code with its reason (RFC 4253 §4.2), and the truncated cases pin the behaviour: `SSH-2.0-OpenS` and `SSH-` classify as SSH, `SSH` alone does not. A server that sends other lines before its identification string is handled; a server that sends more than 255 bytes before it is not, and that is the protocol's own ceiling rather than this bound |
| 6 | The real socket shapes behind `dialFactFor`, `resolverFact` and `bannerFact` are unexercised | The tests script the errors a real dial, resolver and read return (`ECONNREFUSED`, `ECONNRESET`, a deadline expiry, `net.DNSError` with `IsNotFound`/`IsTimeout`) and assert the probe's own deadlines, but no live banner, live NXDOMAIN or live blackholed region is dialed in this suite — the project's own rule forbids real egress in tests. The verify phase's hand-run on the motivating network is the remaining evidence |
| 7 | `egress_test.go` now holds three probe groups and its two shared helpers (`resultText`, the `scripted*` seams) are used by all of them | The PR 6 helpers were kept byte-identical apart from the two renamed call sites, so the hub's assertions are unchanged; the new fixtures are additive and named for what they script. A reviewer may still prefer the file split per group, which PR 8 can do when it adds `quic_test.go` and `tls_test.go` |
| 8 | The artifact's focused filters (`-run 'TestEgressSsh\|TestEgressCF'`) select nothing under RE2 | Reproduced again (run 12) and recorded since PR 4 risk #7 (PR 5 note 12, PR 6 run 13): copy the filters from the map with a **bare** `|`. The eight checkboxes are unaffected |
| 9 | The per-region result target is a joined string (deviation #5) | PR 16's payload mapper must echo it as one string rather than `null` or the first region, and PR 17's human projection must not read it as a single endpoint. Both observations carry their own address either way, so no consumer loses the split |

### Engram mirror note

This file is the **authoritative** artefact. Its Engram mirror is now split in **six** parts because the merged text exceeds the store's 50,000-character content limit: part 1 — the file header plus the PR 1 and PR 2 sections — under the topic `sdd/reach-diagnosis-core/apply-progress`; part 2 — the PR 3 section — under `sdd/reach-diagnosis-core/apply-progress/part2`; part 3 — the PR 4 section — under `sdd/reach-diagnosis-core/apply-progress/part3`; part 4 — the PR 5 section — under `sdd/reach-diagnosis-core/apply-progress/part4`; part 5 — the PR 6 section — under `sdd/reach-diagnosis-core/apply-progress/part5`; and part 6 — this PR 7 section — under `sdd/reach-diagnosis-core/apply-progress/part6`.

Every part states the artefact path, this file's byte size and its SHA-256 digest **as recorded when that part was saved**, and that the repository file is authoritative. Parts 1–5 therefore carry their own slice's snapshot and their digests no longer match this file, which has grown by one section since; part 6 carries the merged file's size and digest as of this run. A mirror part is a readable convenience copy of this artefact, not a second source of truth: read this file for the complete, current record. The PR 3–PR 6 sections' own older notes still describe the earlier two-, three-, four- and five-part shapes; they are earlier sections of this artefact and are deliberately not rewritten here.

# Apply Progress — PR 8 of 20 — WU14 + WU15 · The protocol probes: `egress.quic` and `tls.interception`

**Change**: `reach-diagnosis-core` · **Slice**: **PR 8 of 20** — "the protocol probes: `egress.quic` and `tls.interception`" (WU14 + WU15)
**Branch**: `feat/probe-quic-tls`, stacked on PR 7's branch `feat/probe-egress-reachability` (chain strategy `stacked-to-main`, so PR 8 targets PR 7's branch and **not** `main`)
**Date**: 2026-09-14 · **Artifact store**: `both` (this file + Engram mirror, split as the mirror note at the end records)
**Strict TDD**: active — `openspec/config.yaml` declares `strict_tdd: true` with runner `go test ./...`; RED → GREEN → TRIANGULATE → REFACTOR followed for both halves of this slice
**Skill resolution**: `paths-injected` — read `/home/luisalt20/.config/opencode/skills/go-testing/SKILL.md` and `/home/luisalt20/.config/opencode/skills/work-unit-commits/SKILL.md` before writing code; no registry fallback was needed
**Delivery path consumed**: `auto-chain` / `stacked-to-main` — this run implements **only** the assigned slice and stops at its PR boundary; the PR 1–PR 7 sections above are preserved unchanged
**Commit status**: nothing committed, staged, pushed or branched by this phase; the work is left in the working tree for the orchestrator
**How to read this file**: PR 8's entry is the section below. The current change-wide remainder is restated at the end of this section.

---

## Structured status consumed (PR 8)

| Field | Value |
|---|---|
| `schemaName` / `schemaVersion` | `gentle-ai.sdd-status` / `2` |
| `changeName` | `reach-diagnosis-core` |
| `nextRecommended` | `apply` |
| `applyState` | `ready` |
| `dependencies.apply` / `.verify` / `.archive` | `ready` / `blocked` / `blocked` |
| `actionContext.mode` | `repo-local` |
| `actionContext.workspaceRoot` | `/home/luisalt20/projects/close/herdr-reach` |
| `actionContext.allowedEditRoots` | `["/home/luisalt20/projects/close/herdr-reach"]` — every file written lives inside it |
| `artifactStore` | `both` declared by the parent prompt (native `openspec`); files written under `openspec/changes/reach-diagnosis-core/` and mirrored to Engram |
| `taskProgress` before this run | 165 total / 60 completed / 105 pending (PR 1 through PR 7) |
| `taskProgress` after this run | 165 total / **68 completed** / 97 pending |
| `actionContext` warnings | none |
| Work-unit ownership markers | all eight PR 8 rows carry the terminal `<!-- sdd-owner: implementation -->` marker; the whole file still holds 165 markers for 165 checkbox rows, none malformed, duplicate, unsupported or non-terminal |
| `applyState: all_done`? | no — implementation continues, so editing was permitted |

**Attempt context**: the status reports one bounded attempt already active for this work unit (`token sha256:b428920ea5767c5fb59c36beafc49927fb0ddb2482d60b3d3dc78270f488358d`) with a **3,000**-changed-line ceiling. Per the parent prompt the parent owns `sdd-attempt acquire`/`settle`; this executor did **not** call the native attempt command. The counted-line position against that ceiling is in "Workload and PR boundary".

---

## Completed tasks and their persisted checkbox updates (PR 8)

All eight PR 8 rows were flipped from `- [ ]` to `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (lines 240–247) as the work completed, then re-read to confirm. `git diff --stat` on that file reports exactly `8 insertions(+), 8 deletions(-)` — eight flips and nothing else (68 checked, 97 unchecked of 165).

| # | Task (short) | Persisted update | Evidence |
|---|---|---|---|
| 1 | RED — write `quic_test.go` with the four D8 outcomes and all four reason codes: any UDP reply ⇒ `(measured, pass, udp_response_received)`; silence ⇒ `(unresolved, udp_silence)`; ICMP port-unreachable ⇒ `(measured, fail, udp_unreachable)`; any other socket error ⇒ `(unresolved, udp_error_unclassified)`; plus the `indeterminate` control case | `tasks.md:240` → `- [x]` | `go test -count=1 -run 'TestEgressQuic|TestQuicProbe' ./internal/probe/` → **exit 1**, 19 × `the registry built no "egress.quic" probe, so no run could measure it` across 21 failing subtests — a behaviour RED: the probe is declared, registered by name and unbuildable |
| 2 | GREEN — implement `quic.go` as a raw UDP datagram over the injected `PacketDialer`/`PacketConn`; add no QUIC dependency | `tasks.md:241` → `- [x]` | Focused run **exit 0**: the four outcomes over both declared regions, each observation labelled `udp 7844 regionN`, both declared addresses dialed over exactly the `udp` network with a probe-owned deadline, both sockets closed, `len(probe.Registry()) == 10` asserted, and `TestQuicProbeAddsNoQuicDependency` proving `go.mod` still carries no requirement |
| 3 | TRIANGULATE — assert the RG-5 property: a silent socket is never `fail`, the detail never claims the path is blocked, and the declared narrow question is stated in the detail text | `tasks.md:242` → `- [x]` | `TestEgressQuicSilenceIsNeverAFailureAndClaimsOnlyTheQuestion`: seven scripts (reply, silence, ICMP, other error, failed dial, no capability, deny-all) scanned for the forbidden tokens `block`/`viable`/`viability`/`transport`/`recommend`; silence asserted `(unresolved, indeterminate, udp_silence)` and never `fail`; a split run (region1 pass, region2 silent) aggregating to the unanswered region; the narrow question asserted in every outcome's detail |
| 4 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestEgressQuic` | `tasks.md:243` → `- [x]` | `gofmt -l .` exit 0 (empty output) · `go vet ./...` exit 0 (no output) · focused `TestEgressQuic` exit 0 · full package suite exit 0 |
| 5 | RED — write `tls_test.go` with: verified chain and declared expected issuer ⇒ `(measured, pass, ok)`; verification failure ⇒ `(measured, fail, tls_verify_failed)` with the verification code in the detail; issuer outside the declared expected set ⇒ `(measured, fail, tls_issuer_unexpected)` with wording that says "not in the declared expected set" and never accuses; other handshake error ⇒ `(unresolved, tls_handshake_unresolved)`; plus the `indeterminate` control case | `tasks.md:244` → `- [x]` | `go test -count=1 -run 'TestTLSInterception' ./internal/probe/` → **exit 1**, 13 × `the registry built no "tls.interception" probe, so no run could measure it` — a behaviour RED for the same reason as row 1 |
| 6 | GREEN — implement `tls.interception` over the injected `TLSVerifier` with the declared expected issuer set | `tasks.md:245` → `- [x]` | Focused run **exit 0**: verified-with-declared-issuer pass, verification failure with the code in the detail, issuer outside the declared set as a measured failure, other handshake error unresolved, and the two not-measured capabilities kept distinguishable |
| 7 | TRIANGULATE — assert the injected verifying configuration is unchanged after the run and that no unverified retry was performed, and that the observed issuer and verification code appear in the result (R-HR-04) | `tasks.md:246` → `- [x]` | `TestTLSInterceptionLeavesTheVerifyingConfigurationUnchanged` (three outcomes ×: exactly one `Verify` call at the declared address, `InsecureSkipVerify` false, `ServerName` the declared host, `RootCAs` nil, and the at-attempt snapshot equal to the post-run configuration) and `TestTLSInterceptionNamesTheObservedIssuerAndVerificationCode` (three outcomes ×: the observed issuer and the verification code both present in the result) |
| 8 | REFACTOR + GATE — `gofmt -l .`, `go vet ./...`, `go test ./internal/probe/ -run TestTLSInterception` | `tasks.md:247` → `- [x]` | `gofmt -l .` exit 0 (empty) · `go vet ./...` exit 0 · focused `TestTLSInterception` exit 0 · full suite and `-race` suite exit 0 |

---

## Files changed (PR 8)

| Path | Status | Authored lines | Purpose |
|---|---|---|---|
| `internal/probe/quic.go` | added | +285 (285) | D8's datagram probe: the header stating the narrow question and why no QUIC stack is involved; `udpProbePayload`, `udpReadLimit`, `udpDeclaredQuestion`, `udpSubject`, `udpFactLabel`; the `egressQuic` probe (`Run` with the per-region reduction, `observe`, `measure`, `exchangeDeadline`); the `packetAddr` net.Addr built from the declared target rather than resolved; `udpObserve`, `packetErrorFact`, `udpReplyFact` |
| `internal/probe/quic_test.go` | added | +549 (549) | The packet-seam fixtures (`scriptedPacketConn` with a per-address template, `scriptedPacketDialer`, `packetSeams`, `runQuic`) and five test functions: the four-outcome table, the indeterminate control table, the RG-5 silence/no-claim/split case, the declared-regions-and-override case, and the no-dependency case |
| `internal/probe/tls.go` | added | +230 (230) | R-HR-04's chain probe: the header stating the four outcomes and why verification is never weakened; `declaredExpectedIssuers` (the declared expected publisher set) with `expectedIssuerFor`; the `tlsInterception` probe (`Run`, `observe`); `tlsVerifyingConfig`; `tlsObserve`, `tlsChainFact` |
| `internal/probe/tls_test.go` | added | +388 (388) | The verifier fixture (`scriptedTLSVerifier`, `tlsConfigSnapshot`/`snapshotTLSConfig`, `tlsSeams`, `runTLSInterception`) and four test functions: the outcome table (including the never-accuses scan), the indeterminate control table, the unchanged-configuration/no-retry case, and the issuer/verification-code case |
| `internal/probe/classify.go` | modified (enabling edit — deviation #1) | +12 / −0 (12) | Four new rows, purely additive: `capability_excluded` and `command_denied` under `PurposeUDPReachability` and under `PurposeTLSCertificate`, with the comments that say why each purpose needs them (design §5.1 obligation 2) |
| `internal/probe/registry.go` | modified (enabling edit — deviation #2) | +9 / −7 (16) | The `egress.quic` and `tls.interception` constructors filled in; the ordering comment extended to record that PR 8 lands two more constructors and leaves exactly one slot (`tls.truststore`) nil. The names, the kinds and the order are untouched, and the registry still holds exactly ten entries |
| `openspec/changes/reach-diagnosis-core/tasks.md` | modified | 8 lines changed in place (8 deletions + 8 additions = 16 changed lines) | Eight PR 8 checkboxes `- [ ]` → `- [x]` |
| `openspec/changes/reach-diagnosis-core/apply-progress.md` | modified | this appended section | Cumulative PR 8 evidence; the PR 1–PR 7 sections are untouched |
| `openspec/changes/reach-diagnosis-core/apply.md` (attempt candidate) | not written | — | No `apply.md` exists or was created; the attempt bookkeeping belongs to the parent's `sdd-attempt` commands, which this executor did not call |

**Authored code + tests: 1,480 counted lines** (285 + 549 + 230 + 388 + 12 + 16). No file outside the four assigned paths, the two disclosed enabling paths and the two artifact files was created or modified; `README.md`, `PRD.md`, the proposal, the specs, the design, `explore.md`, `research.md`, `preproposal.md` and `openspec/config.yaml` are untouched. No new dependency was added (production code imports only `context`, `crypto/tls`, `errors`, `fmt`, `net`, `strconv`, `strings`, `syscall`, `time` — all standard library; the tests add `os` and `reflect`); no linter, no CI configuration and no `go.sum` was introduced.

---

## Test commands run — exact commands and exit status (PR 8)

| # | Command | Exit | Observed output (abridged) |
|---|---|---|---|
| 1 | `go test -count=1 -run 'TestEgressQuic\|TestQuicProbe' ./internal/probe/` (task 1 RED) | **1** | 19 × `the registry built no "egress.quic" probe, so no run could measure it`; 21 failing subtests across the four test functions |
| 2 | `gofmt -w internal/probe/quic.go internal/probe/quic_test.go internal/probe/classify.go internal/probe/registry.go` then the same focused run (task 2 GREEN) | **0** | `ok …/internal/probe 0.010s` |
| 3 | `go test -count=1 -v -run 'TestEgressQuic\|TestQuicProbe' ./internal/probe/` | **0** | 22 `--- PASS` lines including subtests |
| 4 | `go test -count=1 -run 'TestTLSInterception' ./internal/probe/` (task 5 RED, with the `tls.interception` slot still nil) | **1** | 13 × `the registry built no "tls.interception" probe, so no run could measure it` |
| 5 | `gofmt -w internal/probe/tls.go internal/probe/registry.go` then the same focused run (task 6 GREEN) | **0** | `ok …/internal/probe 0.008s` |
| 6 | `go test -count=1 -v -run 'TestEgressQuic\|TestQuicProbe\|TestTLSInterception' ./internal/probe/` | **0** | 39 `--- PASS` lines including subtests |
| 7 | mutations Q1–Q6 and T1–T7, one at a time against `/tmp` backups (see "Mutation evidence") | **1** each | every mutation caught by the case it targets; `diff -q` confirmed every restore, and the package was re-run green afterwards |
| 8 | `gofmt -l .` · `go vet ./...` (tasks 4 and 8 gates) | **0** each | `gofmt` empty output; `vet` no output |
| 9 | `go test -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 0.414s` · `ok …/internal/version 0.007s` |
| 10 | `go test -race -count=1 ./...` (final gate) | **0** | `ok …/internal/probe 1.510s` · `ok …/internal/version 1.021s` |
| 11 | `go test -count=1 -v ./internal/probe/` (final inventory) | **0** | 78 top-level functions PASS across the package (69 before this slice + 9 new: five in `quic_test.go`, four in `tls_test.go`) |
| 12 | `go test -count=1 -run 'TestEgressQuic\|TestTLSInterception' ./internal/probe/` (the map's filter, **bare pipe**) | **0** | `ok …/internal/probe 0.012s` |
| 13 | `go test -count=1 -run 'TestEgressQuic\\\|TestTLSInterception' ./internal/probe/` (the map's **escaped** form, reproduced as written) | **0** | `ok …/internal/probe 0.008s [no tests to run]` — the artifact's `\|` typo again (PR 4 risk #7, PR 5 note 12, PR 6 run 13, PR 7 run 12): under RE2 it matches a literal pipe and selects nothing |

**Runtime harness**: **N/A as an end-to-end run, exercised in process at both new probe boundaries.** There is still no CLI (`cmd/herdr-reach` lands in PR 18/19), no `doctor` wiring (PR 18) and no real network: the slices that would make a real machine reachable are PR 18–PR 20. What *is* exercised is the production path a run takes — `Registry()`/`ProbesFor(seams, input)` → the declared entry → its constructor → `Run` — with a scripted packet dialer/socket and a scripted TLS verifier over the deny-all base of design §6.2, which is the first proof level the design names for a probe. The unmeasured live steps remain the real socket shapes (a real UDP reply, a real ICMP port-unreachable, a real silent edge) and the real TLS handshake against a live chain, both of which belong to the verify phase's hand-run — design §9 names exactly these two: the D8 raw-UDP comparison against `cloudflared`'s own QUIC log lines, and the PRD §1.1 chain.

---

## TDD Cycle Evidence (PR 8)

RED → GREEN → TRIANGULATE → REFACTOR per PR 8 task row, in the two halves the slice merges (the QUIC probe first, then `tls.interception`, which is also the boundary the task plan names). Both REDs are behaviour failures — the probe is declared, registered and unbuildable — rather than missing symbols inside a file that already exists.

| Task row | Phase | Evidence produced | Observed failure (RED) | Observed pass (GREEN) |
|---|---|---|---|---|
| 1 RED — QUIC cases | RED | `quic_test.go` gained the packet-seam fixtures and all four QUIC test functions before any QUIC production code existed | exit 1: 19 × `the registry built no "egress.quic" probe, so no run could measure it` | n/a |
| 2 GREEN — the QUIC probe | GREEN | `quic.go` (285 lines); `classify.go`: the two `PurposeUDPReachability` not-measured rows; `registry.go`: the `egress.quic` constructor | (previous row) | exit 0: 22 PASS lines; reply/`udp_response_received`, silence/`udp_silence`, ICMP/`udp_unreachable`, other error/`udp_error_unclassified` over both declared regions, deadlines set, sockets closed, no dependency added |
| 3 TRIANGULATE — RG-5 and the declared set | TRIANGULATE | `TestEgressQuicSilenceIsNeverAFailureAndClaimsOnlyTheQuestion` (seven outcome scripts scanned for five forbidden tokens, the silence triple, and a split run), `TestEgressQuicDeclaredRegionsSurviveAnOverride`, `TestQuicProbeAddsNoQuicDependency` | Cases pass against the GREEN implementation; teeth proven by Q1 (silence classified as a failure), Q3 (the declared question dropped), Q4 (only the first region measured), Q5 (no socket deadline) and Q6 (socket never closed), each exit 1 | exit 0 after each restore |
| 4 REFACTOR + GATE | REFACTOR | The `packetErrorFact` default wording corrected so a dial failure is not described as a read that failed (it now says the exchange produced no reply, which is true whether the socket failed before or after the datagram went out); comments re-read against the code | n/a — refactor only (the QUIC suite is the regression gate) | exit 0: `gofmt -l .` (empty), `go vet ./...`, focused suite, full suite |
| 5 RED — TLS cases | RED | `tls_test.go` gained the verifier fixture, the snapshot helper and all four TLS test functions, with `tls.go` and the `tls.interception` constructor still absent | exit 1: 13 × `the registry built no "tls.interception" probe, so no run could measure it` | n/a |
| 6 GREEN — the chain probe | GREEN | `tls.go` (230 lines) with `declaredExpectedIssuers`, `expectedIssuerFor`, `tlsVerifyingConfig` and `tlsChainFact`; `classify.go`: the two `PurposeTLSCertificate` not-measured rows; `registry.go`: the `tls.interception` constructor | (previous row) | exit 0: verified/`ok`, verify failure/`tls_verify_failed` with the code, unexpected issuer/`tls_issuer_unexpected`, handshake error/`tls_handshake_unresolved`, and the two not-measured capabilities |
| 7 TRIANGULATE — configuration, retry and the reported pair | TRIANGULATE | `TestTLSInterceptionLeavesTheVerifyingConfigurationUnchanged` and `TestTLSInterceptionNamesTheObservedIssuerAndVerificationCode` | Cases pass against the GREEN implementation; teeth proven by T1 (`InsecureSkipVerify` true), T2 (the server name dropped), T5 (a retry after an error), T6 (`expectedIssuerFor` always true) and T7 (the nil-verifier branch removed, which panics), each exit 1 | exit 0 after each restore |
| 8 REFACTOR + GATE | REFACTOR | Comments re-read against the code; the duplicate multi-observation join deliberately left in `quic.go` rather than extracted, because the shared home would be `egress.go`, which is outside this slice's allowed file list (deviation #7) | n/a — refactor only | exit 0: `gofmt -l .` (empty), `go vet ./...`, focused suite, full suite, `-race` suite |

**Strict-TDD integrity note.** (a) Both REDs are registry-constructor failures, because a probe the registry cannot build cannot be measured at all: that is the honest first failing state of these rows, and it is a *behaviour* failure (declared and unbuildable), not a missing symbol in the suite. (b) Rows 3 and 7 are TRIANGULATE rows: their cases were written against the GREEN implementation, which strict TDD allows for triangulation, and every one of them was mutation-checked rather than trusted (thirteen mutations, all caught). (c) Row 4's and row 8's refactors were verified by the suites they already had, and both suites are the gate for the wording change in row 4: the RG-5 token scan and the four-outcome table are what pins `packetErrorFact`'s text. (d) No comment, control case or triangulation case was removed or compressed to reach any number; the counts are reported in "Workload and PR boundary".

---

## Mutation evidence (PR 8)

Every mutation was applied to a `/tmp` backup, run against the focused case, and restored; `diff -q` then confirmed the restore and the package was re-run green.

| # | Mutation | Case that caught it | Observed failure |
|---|---|---|---|
| Q1 | Silence classified as an ICMP port-unreachable (`ObsUDPSilence` → `ObsUDPUnreachable` in `udpReplyFact`) | the four-outcome table and the control table | exit 1 — `observation 0 = ("measured", "fail", "udp_unreachable"), want ("unresolved", "indeterminate", "udp_silence")` |
| Q2 | The denied-seam branch removed from `packetErrorFact` (`case errors.Is(err, ErrSeamDenied)` → `case false`) | the indeterminate control table | exit 1 — `observation 0 = ("unresolved", "indeterminate", "udp_error_unclassified"), want ("not_measured", "indeterminate", "command_denied")` |
| Q3 | `udpObserve` no longer appends the declared question | the four-outcome table and the silence case | exit 1 — `does not state the declared narrow question` |
| Q4 | The region loop stops after the first region (`targets[:1]`) | the four-outcome table, the split case and the override case | exit 1 — expected two observations, got one (`want one per declared region`) |
| Q5 | The socket deadline is never set | the four-outcome table | exit 1 — `carried no deadline, so the probe bounded nothing` |
| Q6 | The packet socket is never closed | the four-outcome table | exit 1 — `the probe left the socket for … open` |
| T1 | `InsecureSkipVerify` set to `true` | the unchanged-configuration case | exit 1 — `the probe disabled certificate verification to obtain a result` |
| T2 | The declared server name dropped from the verifying configuration | the unchanged-configuration case | exit 1 — `the verifying configuration's server name = "", want the declared host "www.cloudflare.com"` |
| T3 | The expected-issuer test inverted in `tlsChainFact` | the outcome table | exit 1 — `observation = ("measured", "fail", "tls_issuer_unexpected"), want ("measured", "pass", "ok")` |
| T4 | The `ErrTLSVerification` branch removed (`case false`) | the outcome table | exit 1 — `observation = ("unresolved", "indeterminate", "tls_handshake_unresolved"), want ("measured", "fail", "tls_verify_failed")` |
| T5 | A retry performed after a failed `Verify` | the unchanged-configuration case | exit 1 — `the verifier was invoked 2 times, want exactly one: an unverified retry is forbidden (R-HR-04)` |
| T6 | `expectedIssuerFor` made to accept any declared entry (`if strings.TrimSpace(expected) != ""`; the first form of this mutation, a literal `if true`, was a build failure rather than a behavioural mutation and does not count) | the outcome table and the issuer/code case | exit 1 — the unexpected-issuer outcome reclassified as a pass |
| T7 | The nil-verifier branch removed (`if false`) | the indeterminate control table | exit 1 — `panic: runtime error: invalid memory address or nil pointer dereference` in the `no capability at all` case, i.e. a missing capability reported by a crash instead of a not-measured fact |

**Triangulation depth.** Thirteen mutations, every one caught. On the QUIC half: five forbidden-token scans over seven outcome scripts; the four reason-code triples asserted per region and in the aggregate; the declared question asserted in every outcome's detail; both declared regions asserted against the declaration; the per-address recipients and deadlines asserted from the seam's own record; the two not-measured capabilities asserted to stay distinguishable; and the split run asserted to aggregate to the unanswered region. On the TLS half: the four outcomes asserted with their reason codes; the never-accuses scan over the unexpected-issuer detail; the at-attempt configuration snapshot compared against the post-run configuration; exactly one `Verify` call asserted in every case; and the observed issuer and verification code asserted to reach the result in every definite outcome.

---

## Deviations from design (PR 8)

| # | Deviation | Why | Design reference | Follow-up owner |
|---|---|---|---|---|
| 1 | **`internal/probe/classify.go` was edited, which is outside the PR 8 file list** (+12/−0): `capability_excluded` and `command_denied` rows under `PurposeUDPReachability` and under `PurposeTLSCertificate`. | Design §5.1 prints the definite outcomes for both questions and neither absence. Both probes can be unable to attempt their measurement at all — no seam injected, or a seam that refused — and obligation 2 requires those two facts to be not-measured with their own distinguishable codes rather than to fall through to `internal_error`. Purely additive, same package, no existing row moved or reordered; proven by Q2 (the denied-seam branch removed) and the two control tables. | design §5.1 (obligation 2), PR 7 deviation #1 | `sdd-verify` must adjudicate; the orchestrator may prefer to fold the rows into a review of PR 2 |
| 2 | **`internal/probe/registry.go` was edited** (+9/−7): both constructors filled and the ordering comment extended. | Two slots, two constructors, one edit each; the names, kinds and order are untouched, the enumeration test still asserts exactly ten, and `tls.truststore` is still nil for PR 9. Extending the comment in the same edit keeps the table's own record of which slice landed what from going stale. | design §4, design D8/D10, PR 7 deviation #2 | n/a — PR 9 fills the last slot |
| 3 | The **declared expected issuer set** lives in `tls.go` (`declaredExpectedIssuers`, unexported) rather than in `targets.go`'s declaration. | `targets.go` was outside this slice's file list, and the set is part of the chain measurement's contract rather than of "where the probe measures": `targets.go` says the probe measures `www.cloudflare.com:443`, and this declaration says what publisher that host is expected to present (PRD §1.1). The consequence is disclosed: the declared expected set is **not** echoed by the declared target set the payload reads, so a consumer can compare the observed issuer only against the probe's constant. | design §5.1 ("issuer not in the declared expected set for the target"), design §3.3, PRD §1.1 | PR 9 shares the declaration (same file); PR 16/PR 17 may want it echoed in the payload and the doc |
| 4 | The datagram is sent to a locally constructed `net.Addr` (`packetAddr`), not to one obtained from `net.ResolveUDPAddr`. | Naming an address is not measuring it, and a resolution call inside the probe would be a real network operation outside a seam — exactly what the seam set exists to prevent, and what PR 19's static guard narrows. The address string is the declared target's own `host:port`, so the dialed set still equals the declared set. | design §6.1 (seam table), design §6.2, R-HR-NF-10 | PR 19's `real.go` owns the production `DialPacket`; the static guard must not need to allow a resolution call here |
| 5 | The QUIC probe takes **no resolver step**, unlike the four TCP reachability probes: a name-level dial failure classifies as `udp_error_unclassified`. | Design §6.1's seam table lists only `PacketDialer`/`PacketConn` for `egress.quic`, and design D8's declared question is about the datagram exchange, not about the name. The failure is therefore unresolved and its wording is the socket's verbatim text — not dressed up, but also not attributed to the name. | design D8, design §6.1, RG-5 | `sdd-verify` may want this stated in the output; PR 16/PR 17 word the human projection |
| 6 | The **verifying configuration is built inside `tls.go`** and asserted through the injected verifier (the test captures the pointer and snapshots its fields at the attempt). | No seam in `Seams` carries a `*tls.Config` and `seams.go` is outside this slice's file list. Building it fresh per run and passing it once satisfies R-HR-04's "never weakened, never retried" without adding public surface; the snapshot-then-compare assertion is what makes "unchanged after the run" observable from outside. | R-HR-04, design §6.1, design §5.1 | PR 9's `tls.truststore` will want the same configuration contract; PR 19's `real.go` implements the production verifier |
| 7 | `quic.go`'s `Run` **repeats** the multi-observation join (`strings.Join` over targets and details) that `egressCF.Run` performs in `egress.go`. | The shared home for that helper is `egress.go`, which is outside this slice's allowed file list, and editing it would move PR 7's reviewed code inside a PR 8 diff. The duplication is three statements and is recorded rather than silently left for a reader to notice. | design §7 (`egress.go` / `quic.go` file map), PR 7 risk #7 | A later slice that already owns `egress.go`, or the verify phase, may extract it |
| 8 | `TestQuicProbeAddsNoQuicDependency` reads `../../go.mod` from the test. | Design D8's rejected alternative was a third-party QUIC stack, and "no dependency was added" is otherwise an assertion about a file no test reads. The path is the module root relative to the package directory `go test` runs in; a move of `go.mod` fails the test loudly instead of skipping it. | design D8, design §11 (no non-stdlib dependency in R1a) | PR 19/PR 20 own the broader dependency and no-egress guards; this case is the D8-specific one |
| 9 | The QUIC probe's silence outcome is reached through `dialTimedOut`, the same deadline-expiry property helper the TCP probes use. | The property ("a socket reported that its deadline expired") is the same fact in both places, and reusing the helper keeps one definition of it. The *classification* is not shared: a deadline expiry is `budget_expired`/fail for a port-reachability question and `udp_silence`/unresolved for the datagram question, and `TestUnclassifiableObservationIsNeverAPass` already pins that an expiry on a non-reachability purpose is not borrowed as a failure. | design §5.1 obligation 3, RG-8 | n/a |

---

## Workload and PR boundary (PR 8)

| Field | Value |
|---|---|
| Slice | PR 8 of 20 — "the protocol probes: `egress.quic` and `tls.interception`" (WU14 + WU15) |
| PR 8 estimate in `tasks.md` | 320–490 lines (point ≈405) |
| Host attempt ceiling | 3,000 counted changed lines |
| **Actual authored code + tests** | **1,480 counted lines**: `quic.go` +285, `quic_test.go` +549, `tls.go` +230, `tls_test.go` +388, `classify.go` +12/−0 (12), `registry.go` +9/−7 (16) |
| Artifact changes | `tasks.md` 8 lines changed (8 + 8 = **16** changed lines) + this appended section |
| **Total counted changed lines for the work unit** | **≈1,830** (1,480 authored + 16 checkbox + this section) |
| Chain per-PR cohesion ceiling | 1,000 changed lines (user-approved, revised from 600 on 2026-09-14) |
| Review budget (session canonical) | 400 changed lines |
| Budget status | **3.7× the slice estimate, over the 1,000-line chain ceiling and over the 400-line session budget; inside the 3,000-line attempt ceiling** |
| PR boundary | Starts at PR 7's branch state (`internal/probe` vocabulary + classification table + seams + declared target set + runner + registry with `local.env`, `local.sshd`, `egress.hub.direct`, the two public-SSH probes and the two per-region Cloudflare edge probes) and ends at `internal/probe` compiling and passing with D8's datagram probe and the chain-on-the-wire probe landed and registered (eight of the ten constructors filled). **PR 9 is not started**: no `tls.truststore` implementation, no `internal/diagnosis/`, `internal/transport/`, `internal/report/`, `internal/doctor/`, `cmd/` or `docs/diagnosis-report.md` exists, and `tls.truststore` is the only registry slot still nil |
| Rollback boundary | Delete `quic.go`, `quic_test.go`, `tls.go` and `tls_test.go`; revert `classify.go`'s +12 and `registry.go`'s +9/−7 (including the comment). The module returns to its PR 7 state with the eight previously landed probes, the vocabulary, the classification table, the seams, the declared target set and the runner still green. `tasks.md` lines 240–247 revert to `- [ ]`; this appended section is the only other PR 8 change |
| Rollback independence | Nothing outside `internal/probe` consumes either probe — the doctor wiring lands in PR 18 and the reasoning layer in PR 9+ — so the revert removes no unrelated work and leaves PR 1–PR 7 green |

**Why 1,480 > 1,000, stated honestly, and the boundary the plan already names.** The overage is not padding. 549 lines are `quic_test.go` — the packet-seam fixtures plus four test functions that between them script four outcome scripts × two regions, six control shapes, seven forbidden-token scans and a split run — and 388 are `tls_test.go` — the verifier fixture with its configuration snapshot plus four test functions that script three definite outcomes, three control shapes and two close-to-the-spec scenarios (unchanged configuration, one call, issuer + code reported). The production code is 515 lines, of which roughly 250 are the comments that carry the *why* of the narrow question, the four UDP outcomes, the never-weakens-verification rule and the never-accuses wording — exactly the code a reviewer has to trust in these two probes. The review-budget rule forbids reaching a number by deleting tests, control cases, triangulation cases, comments, docs or blank lines, and no `size:exception` was assumed, so **nothing was compressed**; the honest count is reported instead. The estimate in `tasks.md` (≈405) under-counted by 3.7×, consistent with the revision recorded in the forecast (the first three units measured 347/1400/1945 against estimates of 80/595/545).

Measuring the code as written, the boundary the task plan names splits this slice into:

- **PR 8a = WU14 (`egress.quic`)** — `quic.go` 285, `quic_test.go` 549, the two `PurposeUDPReachability` rows in `classify.go` 6, and the `egress.quic` constructor in `registry.go` 2, ≈ **842 counted lines** — inside the 1,000-line ceiling on its own.
- **PR 8b = WU15 (`tls.interception`)** — `tls.go` 230, `tls_test.go` 388, the two `PurposeTLSCertificate` rows in `classify.go` 6, and the `tls.interception` constructor in `registry.go` 2, ≈ **626 counted lines** — inside the ceiling too. PR 8b depends on PR 8a only for the shared test fixtures (`egressBuild`, `declaredAddress`, `resultText`), which PR 7 already landed in `egress_test.go` and `quic_test.go` does not add to.
- **Shared enabling edit**: the ordering comment in `registry.go` (12 changed lines), which describes both constructors and cannot be attributed to one half.

Two-thirds of the overage is test code, and the split the plan names does bring each half under the ceiling; the only reason the halves are not already separate reviews is that the parent assigned PR 8 as one work unit and this executor does not move review boundaries on its own.

**This executor did not split the slice or touch branches**: the parent assigned PR 8 as one unit, and a split moves a review boundary rather than the work. Against the 3,000-line attempt ceiling the work unit is inside it (≈1,830).

---

## Remaining unchecked tasks (PR 8 view)

**Inside this slice: none.** All eight PR 8 checkbox rows are `- [x]` in `openspec/changes/reach-diagnosis-core/tasks.md` (re-read after the edits: **68 checked, 97 unchecked**; 165 rows total; `git diff --stat` on that file shows exactly 8 insertions and 8 deletions). The ownership markers were re-checked: 165 `<!-- sdd-owner: implementation -->` markers for 165 rows, every one terminal.

The change-wide remainder is **97 checkbox lines in PR 9 – PR 20**, which belong to later chained slices on later branches (PR 8 targets PR 7's branch; the chain is `stacked-to-main`) and are **not** part of this work unit. The next unchecked line in the artifact, verbatim, is `tasks.md:254` — PR 9's first row, which is where the next slice resumes:

```
- [ ] **RED** — add the Linux controls: an accepting pool ⇒ `(measured, pass, ok)`; a rejected chain ⇒ `(measured, fail, truststore_rejects_chain)`. <!-- sdd-owner: implementation -->
```

PR 9 adds `tls.truststore` to `tls.go`/`tls_test.go` (extending the two files this slice created, including the `declaredExpectedIssuers` declaration and `tlsVerifyingConfig`) and fills the last registry slot, plus `internal/diagnosis/facts.go` and the accessor cases.

---

## Risks (PR 8)

| # | Risk | Status / handling |
|---|---|---|
| 1 | A **silent UDP socket** is the most likely live outcome on a filtered network, and it is deliberately reported as `unresolved` rather than as a negative | The classification table maps `ObsUDPSilence` to unresolved/indeterminate and Q1 proves the outcome table and the control table catch a regression. Consequence disclosed: a run against a network that drops QUIC will usually be **incomplete** (exit code 1 once PR 18 wires it), not clean, because silence makes the run incomplete by design. That is the honest outcome and it is what design §5.2's `CF_HTTP2_ADVISED_QUIC_UNCONFIRMED` path is built for |
| 2 | A **name-level failure** on the QUIC probe reports `udp_error_unclassified` rather than `dns_no_such_host` (deviation #5) | design §6.1 lists no resolver seam for `egress.quic`. The detail carries the socket's verbatim text, so nothing is fabricated; what is missing is the attribution. The verify phase's hand-run and PR 16/PR 17's wording are where a reader would notice |
| 3 | The **declared expected issuer set is not echoed** by the declared target set (deviation #3) | The pass/fail decision is still evidence-backed (the observed issuer and the verification code are always in the result), but a consumer cannot recompute the set from the payload. PR 16/PR 17 own whether it is echoed |
| 4 | `expectedIssuerFor` compares **case-insensitively** on trimmed values | This is a local decision with a stated reason (publishers vary in case between chains, and a case-only difference must not become a reported interception). It is a value comparison, never a comparison of error text, so R-HR-07 is unaffected; a stricter exact comparison would be the alternative and is recorded here as the trade-off |
| 5 | The QUIC payload is a **labelled marker**, not a QUIC packet | Deliberate (D8): the probe has no QUIC stack, so its payload cannot be a handshake and must not pretend to be one. A remote endpoint that answers only valid QUIC packets will stay silent, and silence is unresolved — which is the designed degradation, not a defect |
| 6 | The **real socket shapes** behind `packetErrorFact`, `udpReplyFact` and the production TLS verifier are unexercised | The tests script `ECONNREFUSED`, a deadline expiry and an arbitrary socket error, and assert the probe's own deadline and one-call discipline, but no live UDP reply, live ICMP port-unreachable or live handshake is performed — the project's own rule forbids real egress in tests. Design §9's verify-phase hand-run is the remaining evidence |
| 7 | `tls.interception` **shares `tls.go` and `tls_test.go`** with PR 9's `tls.truststore` | Deliberate (design §7's file map). PR 9 extends both files, and the shared `declaredExpectedIssuers`/`tlsVerifyingConfig` are the contract it will build on; the two probes' questions differ (a live chain vs the local pool), so the test functions stay separate |
| 8 | The artifact's focused filter (`-run 'TestEgressQuic\|TestTLSInterception'`) selects nothing under RE2 | Reproduced again (run 13) and recorded since PR 4 risk #7 (PR 5 note 12, PR 6 run 13, PR 7 run 12): copy the filter from the map with a **bare** `|`. The eight checkboxes are unaffected |
| 9 | The PR 8 estimate (≈405) was **3.7× under** the authored 1,480 | The forecast already records that its estimates measured 2–3× low (the first three units measured 347/1400/1945 against 80/595/545); this slice is inside that band at its upper end, mostly because the QUIC and TLS outcome tables script every state per region and the two R-HR-04 properties (unchanged configuration, no retry) need their own fixtures. No task was dropped to fit |

### Engram mirror note

This file is the **authoritative** artefact. Its Engram mirror is now split in **seven** parts because the merged text far exceeds the store's 50,000-character content limit: part 1 — the file header plus the PR 1 and PR 2 sections — under the topic `sdd/reach-diagnosis-core/apply-progress`; part 2 — the PR 3 section — under `sdd/reach-diagnosis-core/apply-progress/part2`; part 3 — the PR 4 section — under `sdd/reach-diagnosis-core/apply-progress/part3`; part 4 — the PR 5 section — under `sdd/reach-diagnosis-core/apply-progress/part4`; part 5 — the PR 6 section — under `sdd/reach-diagnosis-core/apply-progress/part5`; part 6 — the PR 7 section — under `sdd/reach-diagnosis-core/apply-progress/part6`; and part 7 — this PR 8 section — under `sdd/reach-diagnosis-core/apply-progress/part7`.

Every part states the artefact path, this file's byte size and its SHA-256 digest **as recorded when that part was saved**, and that the repository file is authoritative. Parts 1–6 therefore carry their own slice's snapshot and their digests no longer match this file, which has grown by one section since; part 7 carries the merged file's size and digest as of this run (267,631 bytes, `41f802a7fa9ed4d95693c6155e31b45fba334fee6b41d926ece0b3d908ebeee5`, measured immediately before this note was appended, so this note's own lines are not covered by that digest). A mirror part is a readable convenience copy of this artefact, not a second source of truth: read this file for the complete, current record. The PR 3–PR 7 sections' own older notes still describe the earlier two-, three-, four-, five- and six-part shapes; they are earlier sections of this artefact and are deliberately not rewritten here.
