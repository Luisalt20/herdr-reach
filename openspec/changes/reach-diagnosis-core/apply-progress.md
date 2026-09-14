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
