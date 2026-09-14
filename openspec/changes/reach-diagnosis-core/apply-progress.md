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
