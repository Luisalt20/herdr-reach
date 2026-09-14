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
