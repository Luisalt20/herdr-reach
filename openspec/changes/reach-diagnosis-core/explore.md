# Exploration — `reach-diagnosis-core`

**Change**: `reach-diagnosis-core` (slice 1 of the herdr-reach SDD program)
**Date**: 2026-09-14
**Phase**: explore (no implementation, no scaffolding, no source files written)
**Artifact store**: hybrid — this file plus Engram topic `sdd/reach-diagnosis-core/explore`
**Preflight consumed**: execution mode `auto`, artifact store `hybrid`, delivery strategy `ask-on-risk`,
review budget 400 changed lines, chain strategy deferred, `size:exception` never inferred.

---

## 1. Ground truth read before any conclusion

| Source | What it establishes |
|---|---|
| `PRD.md` (1166 lines, v0.1.0-draft) | Full product spec: §1 problem, §1.1 measured matrix, §4 flows/screens, §5 core abstractions, §5.1 probe table, §7 architectural package layout, §8 traps, §9 requirements, §10 screens, §11 edge cases, §12 out-of-scope, §13 metrics, §14 notes, §14.5 governing rule |
| `README.md` | Public front page; roadmap Phases 0–6; Go 1.25.10 badge; "10 probes"; four design principles |
| `openspec/config.yaml` | Ratified module path `github.com/Luisalt20/herdr-reach`; strict TDD `go test ./...`; typecheck `go vet ./...`; format `gofmt -l .`; `quality.lint` empty (do not invent); greenfield state; artifact language English |
| Repository state | Only `PRD.md`, `README.md`, `LICENSE`, `.gitignore`, `docs/assets/**.svg`, `openspec/config.yaml`. No `go.mod`, no Go sources, no `contracts/`, no CI workflow |
| Toolchain observed | `go1.26.6 linux/arm64` vs README badge `1.25.10` — discrepancy recorded in config, deliberately unresolved |

Consequences that constrain every option below:

1. **`go test ./...` currently fails** ("directory prefix . does not contain main module"). The first apply phase of this change must create `go.mod` with the ratified module path before the configured test command can run at all.
2. **No linter exists and none may be invented.** `go vet` + `gofmt -l .` are the only quality gates.
3. **No `contracts/` directory exists.** Slice 1 does not need it unless the diagnosis report is promoted to a cross-machine contract (§5.4, Q15).
4. **No CI workflow exists.** "Never dial the real network in CI" is currently a property of the test design, not of automation. Enforceable only by construction (deny-all test dialer) until a workflow is added.
5. **The PRD is a product spec, not a single change.** The SDD program must chain changes; slice 1 must be honest about what it does not deliver (§4 below).

---

## 2. What slice 1 is, and its boundary

**Candidate scope (as given)**: Go module bootstrap, CLI/TUI skeleton, the probe suite, the diagnosis
engine, transport feasibility (detect + recommend, with reasons), the verdict screen, machine-readable
`--json`. **Mutates nothing on the user's machine.**

This maps exactly onto README roadmap **Phase 1 ("Diagnosis")**, PRD §5.1 (Diagnosis Engine), PRD §4.4
(the product's center of gravity), and PRD §12's "Expanded `doctor` mode" ("Nearly free, since the probe
suite already exists. Likely to ship early").

**The headline finding of this exploration**: the §5.1 probe table is *exactly* slice 1. All ten probes
fit, none requires a mutation, and every one of them is offline-testable behind a seam. That makes
slice 1 both the correct first slice and the one with the highest ratio of product value to blast radius:
it is useful on its own (`herdr-reach doctor`), it is the credibility engine for the governing rule
(§14.5), and a bug in it cannot damage a work laptop.

| In slice 1 | Explicitly out of slice 1 |
|---|---|
| `go.mod` / `cmd/herdr-reach` bootstrap, flag parsing, exit codes | Any file write outside the user's build tree |
| `internal/probe` with the ten §5.1 probes + injected seams | `internal/{environ,pair,shedr,sshcfg,sysfile,contracts}` |
| `internal/diagnosis` (pure reasoning: classification, disambiguation, verdict) | `PairingBundle` / `NodeReceipt` (R-HR-08…11) |
| `internal/transport` registry + feasibility for 4 adapters (detect + recommend) | `PlanHub` / `PlanNode` / `Verify` bodies, Step framework, dry-run, undo |
| TUI shell, router, role screen, diagnose screen, verdict screen | Plan / Apply / Verify / Receipt / Prerequisite / Status / hub screens |
| `--json` report + raw-evidence view | `sshd` provisioning, `cloudflared` install, unit files, WSL2 keepalive |
| Zero mutation, provable | `herdr` invocation (R-HR-26/27) |

---

## 3. Requirement coverage map

Coverage vocabulary:

- **Covered** — the slice's implementation satisfies the requirement's testable intent.
- **Partial** — an explicitly named part is covered; the remainder is deferred to a named slice.
- **Deferred** — not in this slice.
- **By-construction** — satisfied only because the slice mutates nothing / invokes no external binary; still needs an explicit test, because "we did not write the bug yet" is not a guarantee.

### 3.1 Functional requirements (R-HR-01…30)

| ID | Coverage | Note |
|---|---|---|
| R-HR-01 | **Covered** | TUI begins with role selection and branches on the answer. Node branch is implemented; hub branch must be an explicit "hub flow ships in a later slice" screen, never a half-built flow (governing rule). |
| R-HR-02 | **Covered** | Diagnose before recommend; nothing is modified before a plan — in this slice nothing is modified at all. |
| R-HR-03 | **Covered** | Port-vs-protocol disambiguation is the slice's core logic (§5.3). |
| R-HR-04 | **Covered** | `tls.interception` reports issuer + verification code. |
| R-HR-05 | **Partial** | Probe + recommendation note covered. *Enforcing* `TUNNEL_TRANSPORT_PROTOCOL=http2` in a unit file is deferred to the Cloudflare provisioning slice. |
| R-HR-06 | **Covered** | Verdict lists every considered transport with viable/not **and a reason in both cases**. Reason is mandatory (`!Viable ⟹ Reason != ""`), testable as a property. |
| R-HR-07 | **Covered** | Raw evidence view + `--json` for the full diagnosis. |
| R-HR-08 | **Deferred** | Pairing artifacts + JSON Schema (`contracts/pairing/v1`) → pairing slice. |
| R-HR-09 | **Deferred** | No artifact exists yet; the no-secrets property is trivially held and must not be claimed. |
| R-HR-10 | **Deferred** | Fingerprint display/compare both sides → pairing slice. |
| R-HR-11 | **Deferred** | Nonce binding → pairing slice. |
| R-HR-12 | **Deferred** | Plan screen with every path → mutation-primitives slice. |
| R-HR-13 | **Deferred** | Backup-before-write → `internal/sysfile` slice. |
| R-HR-14 | **Deferred** | Dry-run → `internal/sysfile` / Step slice. |
| R-HR-15 | **Deferred** | Partial-failure reporting + undo → Step slice. |
| R-HR-16 | **Partial** | Slice 1 carries the pin as *knowledge*: the verdict/notes for `cloudflare-tunnel` must state the pinned client version and the exit criterion (PRD §8.3) and the code must reserve the constant's home. SHA256 + self-version verification before install is deferred to the install slice. |
| R-HR-17 | **Deferred** | Refusing a mismatched checksum only matters where something is installed. |
| R-HR-18 | **Partial** | `local.sshd` reads binary presence, unit state and *effective* config (`sshd -T`) and reports divergence between written and effective config. Hardening and write-back are deferred. |
| R-HR-19 | **Partial** | No key is installed or verified in this slice, so the requirement is vacuous here — but the evidence model must be shaped to carry **what and where** (owner, mode, absolute path) from day one (PRD §8.4), or the later verification slices will retrofit it badly. This is a design obligation, not an implementation one. |
| R-HR-20 | **Deferred** | User-scope-first services → `internal/environ` slice. |
| R-HR-21 | **Partial** | WSL2 *detection* (`local.env`) covered; the three-mechanism fix is deferred. |
| R-HR-22 | **Deferred** | Keepalive watchdog → WSL2 slice. |
| R-HR-23 | **Deferred** | Same slice; no keepalive is created here, so it cannot misbehave. |
| R-HR-24 | **By-construction** | Nothing is disabled or reconfigured. Needs an explicit test: slice 1 performs no write and executes no third-party binary. |
| R-HR-25 | **Partial** | The SSH-reachability probe seam is covered. End-to-end hub→node authentication is deferred; it needs both provisioning slices. |
| R-HR-26 | **Deferred** | `herdr machine add` interactive → `internal/shedr` slice. |
| R-HR-27 | **By-construction** | Slice 1 never invokes or touches `herdr`. Needs an explicit test/assertion. |
| R-HR-28 | **Deferred** | `remove` → lifecycle slice. |
| R-HR-29 | **Partial** | Platform classification (Linux / macOS / WSL2 / native Windows) is covered; *provisioning* on each platform is deferred to environ + WSL2 slices. |
| R-HR-30 | **Covered** | Refusing an unsupported node (native Windows) with an explanation is diagnosis-shaped and cheap; it belongs in this slice, not later. |

### 3.2 Non-functional requirements (R-HR-NF-01…10)

| ID | Coverage | Note |
|---|---|---|
| R-HR-NF-01 | **Covered** | TUI never blocks on a probe: streaming runner + `Model.Update()` free of I/O (§5.2). |
| R-HR-NF-02 | **Covered** | Bounded timeouts; a hanging probe resolves to `indeterminate` and the run is never reported complete. |
| R-HR-NF-03 | **Covered** | Core invariant, and directly testable across the DTO, the TUI render and the verdict. |
| R-HR-NF-04 | **Covered** | Registry + interface; asserted by a *test-only* transport registered in a test with no change to `internal/diagnosis`. |
| R-HR-NF-05 | **Deferred** | No `Environment` interface in this slice (nothing that persists). Defining it now would ship an unused abstraction. |
| R-HR-NF-06 | **Deferred** | Applies to pairing artifacts. Slice 1's `--json` is not a cross-machine contract, but should carry `schema_version` so it *can* be promoted later (Q15). |
| R-HR-NF-07 | **Covered** | Bubbletea + Bubbles + Lipgloss. Version pinning is Q5. |
| R-HR-NF-08 | **By-construction** | No mutation ⇒ nothing to be idempotent about. Must not be presented as demonstrated. |
| R-HR-NF-09 | **Covered** | Full node diagnosis under 60 s with every probe bounded; testable offline with injected short timeouts and scripted dialers. |
| R-HR-NF-10 | **Covered** | No telemetry. Requires a test that the set of dialed targets is closed and declared (the ten probes' declared targets only). |

### 3.3 §5.1 probe table

All ten probes are in slice 1. Two carry scope risk.

| Probe | Slice 1 | Risk / dependency |
|---|---|---|
| `local.env` | Covered | Needs a portable OS/arch/systemd/launchd/WSL2 classifier with no external binary where possible. |
| `local.sshd` | Covered | `sshd -T` may require privilege in some configurations → degrade to `indeterminate` with a reason, never to `pass` (Q4). |
| `egress.hub.direct` | Covered | **Requires a hub address source**, which does not exist yet: the PRD's `PairingBundle` carries it, and pairing is deferred (Q2). |
| `egress.ssh.known` | Covered | Target list must be configurable/declared (Q3, NF-10). |
| `egress.ssh.443` | Covered | Same. |
| `egress.cf.7844` | Covered | Edge endpoints `region1`/`region2` — decide whether both are probed (Q3). |
| `egress.cf.443` | Covered | Same. |
| `egress.quic` | Covered **at design risk** | Highest-uncertainty probe in the product: raw UDP silence is not proof of blockage, and a real QUIC handshake pulls in a heavy dependency (Q8). |
| `tls.interception` | Covered | Needs a root configuration seam so tests never hit the network; live issuer comparison is a manual/verify-time evidence item. |
| `tls.truststore` | **Partial / at risk** | Go does not read the macOS keychain without `cgo` or shelling out. Linux (`crypto/x509` system pool) is straightforward; macOS needs an explicit decision (Q9). |

### 3.4 §8 traps

| Trap | Coverage |
|---|---|
| §8.1 port blocked ≠ protocol blocked | **Covered** — the disambiguation rule is slice 1's centerpiece. |
| §8.2 QUIC blocked silently, forcing HTTP/2 | **Partial** — detection + recommendation note + the post-quantum trade-off text are covered; unit-file enforcement is deferred (R-HR-05). |
| §8.3 pinned `cloudflared` + SHA256 + self-reported version | **Partial** — pin knowledge and exit criterion are surfaced in slice 1's verdict/notes; enforcement is deferred (R-HR-16/17). |
| §8.4a control tests ("a probe that cannot fail is not a probe") | **Partial** — the obligation applies *now*: every slice-1 probe needs a case proving it can return `fail`/`indeterminate`. Encode as a TDD rule (§6). |
| §8.4b content **and** location | **Partial** — no file is verified in slice 1; the evidence/`Result` shape must be able to carry location before the first verification slice (§5.4, R-HR-19). |

### 3.5 §11 edge cases reachable in slice 1

| Edge case | Slice 1 behavior required |
|---|---|
| A probe hangs | Resolve to `indeterminate` after the timeout, continue the suite, never present the run as complete. Covered. |
| TLS interception detected | Report issuer + verification code; identify which process would fail. Covered at detection level. |
| Trust store rejects the inspecting CA | Explain what must be imported; never silently disable verification. Covered (macOS mechanism: Q9). |
| `sshd` absent | Report; the install offer belongs to a later slice, so the verdict must say the install step is not part of this slice. Covered as report-only. |
| `sshd -T` disagrees with the written file | Report both. Covered by `local.sshd`. |
| Node is native Windows | Stop, explain Herdr does not support Windows as an SSH target, offer WSL2 as the supported path. Covered (R-HR-30). |
| Node is WSL2 without systemd | Detect and report; enabling systemd is a later slice. Covered as report-only. |
| QUIC fails, TCP succeeds | Force HTTP/2 in the generated unit (deferred) + state the trade-off in the plan (partial: verdict note now). |
| Docker Desktop on a WSL2 node | Warn that it can terminate the VM and that the watchdog is the answer; do not touch its configuration. Partial — a `local.env` detail/note is enough for slice 1. |
| Hub and node architectures differ | Not applicable (no pairing yet). |
| No hostname available | The verdict must state that a named tunnel is impossible without one and stop pretending. Partial — the prerequisite screen is deferred, but the *honest verdict text* belongs here. |

---

## 4. Proposed program roadmap (the artifact to approve before slice 1's proposal)

### 4.1 Slicing rules used

1. **One package family or one adapter per change**, matching PRD §7.1 — that is the natural boundary that keeps each change inside the 400-line review budget.
2. **A slice must be verifiable without the slice above it.** Diagnosis, pairing, and mutation primitives have no external prerequisites; provisioning slices need the ones below them.
3. **Nothing mutates a user machine until the write primitives have their own change with their own tests.** PRD §7.2 ("Steps, Not Side Effects") is the framework; slice 1 deliberately has no Step type because nothing applies anything.
4. **README roadmap phases are the coarse ordering**; this table only says where each phase has to be cut so reviews stay small.

### 4.2 Roadmap

| # | Change | Scope | PRD / reqs | Depends on | Why here |
|---|---|---|---|---|---|
| **R1** | `reach-diagnosis-core` (this) | Module + CLI/TUI skeleton + 10 probes + diagnosis engine + transport feasibility + verdict + `--json`, zero mutation | §5.1, §4.4, R-HR-01…07, 24, 30, NF-01…04, 07, 09, 10 | none | Measurement is the product's precondition and the only slice that is useful alone. Zero blast radius. Retires the governing-rule risk first. |
| R1a | *(optional split of R1)* `reach-probe-core` | `go.mod`, `internal/probe`, `internal/diagnosis`, `internal/report`, headless `doctor --json` | as R1 | none | Recommended if R1 exceeds the review budget: pure logic + JSON is the largest and most testable half (§8.1 R1). |
| R1b | *(optional split of R1)* `reach-tui-diagnosis` | TUI shell, router, role/diagnose/verdict screens, streaming adapter | as R1 | R1a | Screens are render/test-heavy and reviewable in isolation; they consume an already-tested model. |
| **R2** | `reach-pairing-contracts` | `internal/pair`, `contracts/pairing/v1` schemas, bundle/receipt codecs, fingerprint display/compare, nonce binding, hub pairing + node receipt screens | §5.4, §7.7, R-HR-08…11, NF-06 | R1 | P0-dense, purely data, no privilege, no mutation. Freezing the artifact pipeline before the Cloudflare slice prevents rework of plan/verify/evidence shapes (the receipt carries `[]Result`). |
| **R3** | `reach-mutation-primitives` | `internal/sysfile` (write-with-backup, atomic replace, mode-aware), `internal/sshcfg` (surgical diffs), `Step` + dry-run + undo framework, plan screen | §7.2, R-HR-12…15, NF-08 | R1 | Highest blast radius per line in the whole product (a work laptop's config files). All its tests are offline (`t.TempDir()`, golden fixtures). Do it before any transport complexity exists, so review attention is on correctness, not on network logic. *R2/R3 are independent; the order can be swapped without rework.* |
| **R4** | `reach-environ-core` | `internal/environ` interface + `ServiceSpec`, `linux-systemd` (user scope + linger preferred), `macos-launchd` | §5.3, §6.1, R-HR-20, NF-05 | R3 | Cloudflare/hub provisioning cannot write a unit without it. It is also the prerequisite for the WSL2 slice. User-scope-first is a stated differentiator, so it must exist before anything assumes a system unit. |
| **R5** | `reach-cloudflare-node` | Node-side: pinned `cloudflared` install (SHA256 + self-version), `sshd` provisioning/hardening/effective-config verification, tunnel unit with `TUNNEL_TRANSPORT_PROTOCOL=http2`, node verification evidence, receipt emission | §8.2, §8.3, R-HR-05, 16…19, 25, 29 | R2, R3, R4 | The workhorse transport, and the first slice that mutates a machine. Likely needs its own split (install / sshd / tunnel+verify) to stay in budget — see §8.2. |
| **R6** | `reach-hub-flow` | Receipt consumption + fingerprint confirmation, `~/.ssh/config` entry, forwarder service, `0600` token file, end-to-end SSH verification, `internal/shedr` + `herdr machine add` screen | §4.3, §14.2, R-HR-25…27, 29 | R2, R3, R4 (+R5 to verify against a real node) | Completes README Phase 2 (node + hub + pairing). Splitting hub from node keeps two large, independent flows out of one review. |
| **R7** | `reach-wsl2-lifetime` | WSL2 detection hardening, `vmIdleTimeout`, `/init`-child keepalive watchdog, forced-`wsl --shutdown` recovery verification | §6.2, §6.3, R-HR-21…23, 29 | R4 (and R5 for a tunnel to restart) | README's stated differentiator and the least documented work anywhere in the program; it deserves a dedicated change with its own evidence artifact (the ≤20 s measurement, PRD §6.2). |
| **R8** | `reach-direct-paths` | `direct-ssh` + `reverse-ssh` provisioning first-class (feasibility already exists from R1) | §5.2, README Phase 4 | R3, R4, R6 | "Try the cheap path first" — but only after the expensive path works end to end, so the comparison has a reference. |
| **R9** | `reach-lifecycle` | `status`, `verify`, `disconnect`, `remove`, `doctor` expansion | §4.5, §12, R-HR-28 | R6 | Requires something provisioned to manage. `remove` must be able to enumerate exactly what earlier slices created. |
| **R10** | `reach-devtunnels-adapter` | Feasibility research spike + adapter *only if* raw TCP/SSH works headlessly | §12, README Phase 5 | R1 (+R3/R4 if it ships) | Removes the single biggest prerequisite (a domain). It may legitimately close as "not viable, here is the evidence" — the spike is the deliverable. |
| **R11** | `reach-distribution` | `scripts/install.sh`, Homebrew tap, goreleaser, publish `--json`/pairing schemas, CI workflow | README Install, §13 | R6+ | The CLI contract must be stable before it is published. This is also where a CI workflow with the deny-network test policy would live. |
| **R12** | `reach-recipes` | Export/import a solved network profile with a redaction story | §12, README Phase 6 | R6, R9 | Needs hostname/token redaction, which only makes sense once the artifacts it redacts are stable. |

### 4.3 Mapping to README roadmap

| README phase | Covered by |
|---|---|
| 0. Design | done (PRD + README) |
| 1. Diagnosis | **R1 (= this change)** |
| 2. Cloudflare transport + node/hub flows + pairing | R2 → R3 → R4 → R5 → R6 |
| 3. Persistence (systemd/launchd/WSL2) | R4 (Linux/macOS) → R7 (WSL2) |
| 4. Direct paths | R8 |
| 5. Dev Tunnels | R10 |
| 6. Recipes | R12 |
| cross-cutting | R9 lifecycle, R11 distribution |

### 4.4 Ordering rationale, stated as reasons rather than preferences

1. **R1 first** because every later slice consumes `Result`/`Diagnosis`/`Feasibility`, and because a wrong measurement makes every later slice's verification meaningless (PRD §13 "Silent failures: zero").
2. **R2 before R5** because the receipt is defined as `[]Result` plus fingerprints; if the pairing contract lands after provisioning, the provisioning slice is rewritten.
3. **R3 before R5/R6** because `Step`, backup and dry-run are the mechanisms those slices are *reviewed* through (R-HR-12…15); retrofitting them after provisioning implies rewriting the provisioning code path.
4. **R4 before R5** because a unit file is an environment artifact, and user-scope-first (R-HR-20) is a design commitment that must precede the code that writes units.
5. **R5 before R6** to avoid building a hub flow against an imagined node. The node produces the receipt the hub consumes; building them in one change would double the review surface and hide the pairing contract.
6. **R7 (WSL2) after R4** because it extends the environment abstraction, and after R5 because its verification is "the tunnel comes back", which requires a tunnel.
7. **R10 (Dev Tunnels) last among transports** because a negative result is as valuable as a positive one and it is the only slice whose *existence* depends on an upstream capability that has not been verified.
8. **R11 last** because distribution freezes public contracts.

---

## 5. Slice-1 architecture options with real tradeoffs

### 5.1 Package boundaries inside `internal/`

| Option | Shape | Tradeoff |
|---|---|---|
| **A. PRD-literal** | `internal/probe` (incl. the disambiguation rule in `egress.go`), `internal/transport`, `internal/tui{,/screens}`, `internal/version`, `cmd/herdr-reach` | Zero deviation from PRD §7.1, which is a spec statement. Cost: `probe` holds both I/O and reasoning, so the most valuable logic in the tool (the rule) shares a package with `net` and becomes harder to test in isolation and harder to reason about. |
| **B. Flat minimal** | everything in `internal/probe` + a thin `internal/tui` | Fewest packages, fastest to write. Cost: the transport adapters and the JSON DTO end up in `probe`, and NF-04's "no changes to the diagnosis engine" becomes unprovable because there is no engine boundary to hold fixed. |
| **C. Dependency-ordered (recommended)** | `cmd/herdr-reach`; `internal/probe` (measurement + seams only); `internal/diagnosis` (pure: classify → disambiguate → verdict); `internal/transport` (registry + `Requires`/`Feasible` adapters); `internal/report` (versioned JSON DTO); `internal/tui` + `internal/tui/screens`; `internal/version` | `internal/probe` never imports `internal/diagnosis`; `transport` imports `diagnosis`; `report` imports `diagnosis`; the TUI imports all three and owns no logic. Cost: one recorded deviation from PRD §7.1 (no `diagnosis` package is listed there) and ~1 extra package of wiring. Benefit: the rule is a pure function of `[]Result`, the JSON contract is decoupled from internal structs, and NF-04 becomes an actual test rather than a hope. |

**Recommendation**: Option C, with the deviation recorded in the slice's design artifact next to the
PRD §7.1 package diagram. Rationale: PRD §5.1 names a "Diagnosis Engine" as one of the four
abstractions that carry the design, and §7.1's file list is a directory sketch rather than a
constraint on package count. If the reviewer prefers strict §7.1 fidelity, Option A is acceptable
provided the rule is still written as a pure function in `egress.go` and the JSON DTO still lives
in its own file, so a later extraction costs one move rather than a rewrite.

Dependency direction (must be acyclic, worth stating because it is easy to get wrong):

```
cmd → tui → {diagnosis, transport, report, probe}
transport → diagnosis
report    → diagnosis
diagnosis → probe           (types only: Result, Verdict, Kind)
probe     → (stdlib, injected seams)
```

`diagnosis` importing `probe` for its value types keeps `Result` single-sourced. The alternative —
a `internal/model` or `internal/core` package holding `Result`/`Diagnosis` so both others import it —
is defensible and adds one package; it is the safer choice if `diagnosis` ever needs to be consumed
by a slice that must not import `probe` (for example a future `status`). Decide in design; do not
decide by accident.

### 5.2 Concurrent probes that stream without blocking the TUI

| Option | Mechanism | Tradeoff |
|---|---|---|
| A. Blocking batch in one `tea.Cmd` | Run all probes, return one `AllDoneMsg` | Simplest. Violates R-HR-NF-01/02 and §4.4's "probes stream into `Probes`" — a 12 s probe freezes the screen. Rejected. |
| B. `errgroup` + mutex-protected slice + periodic refresh | Workers append; TUI polls on `tea.Tick` | No channel plumbing, but the screen updates on the tick, not on the result, and the runner is entangled with the TUI's refresh cadence. Rejected. |
| C. Streamed results over a channel, TUI awaits one result per `tea.Cmd` (recommended) | `probe.Runner` starts one goroutine per probe (bounded), each wrapped in its own timeout, sending `Result` on a buffered channel plus a completion signal. `internal/tui` adapts: a `tea.Cmd` reads one `Result` and returns `ResultMsg`; after each `ResultMsg` the model re-issues the command. The runner itself imports no TUI package and is testable with plain Go. | Canonical Bubbletea streaming shape; progressive fill (the §4.2 mock); runner unit-testable without `teatest`. Cost: a small adapter in `tui`, plus care with cancellation. |
| D. Coalesced batches | As C, but emit a `BatchMsg` every N ms | Fewer re-renders on slow terminals. Only 10 probes exist, so this is premature; keep as a documented option if a render storm appears on large terminals. |

Non-negotiable details for C, each of which is a test in §6:

1. **Per-probe timeout plus a global budget.** Each probe gets `context.WithTimeout` with its own value; the run as a whole is bounded by NF-09's 60 s. In tests the values are injected (tens of milliseconds) so the bound is provable in milliseconds.
2. **The runner must not trust the probe to return.** A probe that ignores `ctx` must not hang the run: the runner selects on the probe's channel and on `ctx.Done()`, records `indeterminate` with reason `timeout`, and abandons the goroutine. This is precisely R-HR-NF-02 and §11's hanging-probe row.
3. **Bounded concurrency.** Ten probes fired at once is a burst of outbound connections from a machine that may be under an endpoint agent's inspection. A small semaphore (default 4, configurable) avoids a self-inflicted probe storm. No PRD requirement covers this; it is a judgment call, and it should be recorded as such (Q10).
4. **Deterministic display order.** Rows come from the registry order, not arrival order, so the diagnose screen never reorders under the user (and golden tests are stable).
5. **Cancellation.** Quitting or navigating away cancels the run context; a dial in flight must honor `ctx` (hence the `DialContext` seam) so the process can exit promptly.
6. **Completion is separate from "all rows filled".** The verdict must be able to say "run incomplete: 2 indeterminate" rather than implying the suite finished. §11 requires "never present the run as complete".

### 5.3 Where the port-versus-protocol disambiguation rule lives

Three candidates:

| Option | Placement | Tradeoff |
|---|---|---|
| A. PRD-literal | `internal/probe/egress.go` | Faithful to §7.1's comment ("TCP matrix, port-vs-protocol rule"). Cost: the rule is in the package that dials, so testing it means dragging probing along, and #NF-04's "engine unchanged" is untestable. |
| B. Pure function in `internal/diagnosis` (recommended) | `diagnosis.Classify([]Result) Findings` + `diagnosis.Recommend(Findings, []Feasibility) Verdict` | The rule becomes a pure, table-driven function over data, so the §1.1 measured matrix is literally the test table. `--json` can quote which rule fired. Cost: recorded deviation from §7.1 + the `Result` type import described in §5.1. |
| C. Inside each transport adapter | Each adapter inspects raw results | Worst option: the rule is duplicated per adapter, so `direct-ssh` and `reverse-ssh` can disagree about the same network. Contradicts R-HR-06 ("a reason in both cases") and the single-source principle behind §8.1. Rejected. |

Shape of the recommendation (design-level, not code):

- The rule is an **ordered declarative table** of `condition → conclusion`, evaluated against classified findings. Conditions are predicates over probe verdicts (by probe name), conclusions are named constants. Ordered evaluation makes precedence explicit and reviewable — the §5.1 snippet is exactly one rule in that table.
- Each conclusion carries a **stable rule id** (e.g. `SSH_ALLOWED_DEST_BLOCKED`) so that the verdict, the `--json` report and a future issue report all name the same thing. This is the cheapest way to make "it says why" machine-checkable and supportable.
- **Facts come from the engine, reasons come from the adapters.** `diagnosis` states what is true about the network; each `Transport` states why it is or is not viable given those facts (`Feasible(d Diagnosis)` per §5.2). This split is what makes NF-04 real: adding a transport adds reasons, never rules.
- **Negative conclusions are first-class.** "`reverse-ssh` not viable — hub :22 unreachable" must be produced by the same machinery as the positive recommendation; §4.4 says the rejected list is as valuable as the recommendation.
- **`indeterminate` poisons only what it touches.** If `egress.ssh.known` is `indeterminate`, the rule cannot conclude "SSH is allowed". The engine must emit an explicitly weaker finding (e.g. `ssh_protocol_unknown`) and the verdict must say so, never fall through to a default. This is R-HR-NF-03 applied to the rule table, and it needs its own tests.

### 5.4 One diagnosis model for `--json` and the TUI

| Option | Shape | Tradeoff |
|---|---|---|
| A. Shared struct with `json` tags | The TUI renders the same struct that is marshalled | Zero mapping code. Cost: internal refactors silently change a public, scriptable output; `Result.Detail` (raw Go error strings) leaks into the contract; no version field. |
| B. Internal model + thin versioned DTO (recommended) | `internal/report.Report` with `schema_version`, ordered arrays, stable field names, mapped from `diagnosis.Verdict` in one place | Public output is decoupled from internal types; deterministic ordering and normalization are explicit; `schema_version` allows later promotion to `contracts/` (Q15). Cost: ~60 lines of mapping plus golden tests, and the risk that the DTO drifts from the model — mitigated by one mapping function and a golden file. |
| C. TUI renders from the marshalled JSON | Single source of truth by construction | Rejects type safety and forces a decode round trip on every re-render. The raw-evidence view also wants richer text than the contract. Rejected. |

Decisions the DTO forces, all of which should be settled in design rather than discovered in review:

1. **Exit codes.** `0` = diagnosis completed, `1` = run incomplete (any `indeterminate`) or no viable transport, `2` = usage/internal error. The open rule is whether "no viable transport" is an error: the tool succeeded at measuring, so arguably `0` with a `viable: false` payload. Scripts need one answer; recommend `0` for a completed measurement regardless of outcome, `1` only for an incomplete run, and document it (Q6).
2. **Streams.** `--json` on stdout only; all human/TUI text on stderr. Otherwise `herdr-reach doctor --json | jq` breaks whenever a progress line escapes.
3. **Determinism.** Probes in registry order, transports sorted, no map iteration in output, `Elapsed` as a duration rounded to ms, and no wall-clock field other than one `generated_at` that tests inject a fixed clock into. Golden files depend on this.
4. **Raw vs stable text.** `Detail` is verbatim and quotable (PRD §5.1: `"dial tcp: i/o timeout after 4s"`) but Go error strings vary by version and OS. The DTO therefore needs a **stable machine reason code** (`reason: "timeout" | "refused" | "dns" | "tls_verify" | "reset" | "blocked_by_policy" | ...`) alongside the verbatim detail. Whether that lives as a new `Result` field (a deviation from the §5.1 struct) or as a derived classification is a design decision with a real tradeoff: a field is honest and typed but deviates from the spec's struct; a derivation keeps the spec shape but duplicates parsing logic. Recommend the field, recorded as a deviation.
5. **Conditionally viable transports.** `cloudflare-tunnel` is only viable *given* a hostname the tool cannot create (R-HR-16, §11). `Feasibility` has `Viable bool / Reason / Notes`, which cannot express "viable if you supply X" without ambiguity. Options: `viable: true` + a note (weak for scripts), or a third state (deviates from §5.2's struct, and a tri-state invites the exact "sort of pass" confusion §14.5 forbids), or `viable` plus a separate `requires []string` array in the JSON DTO (DTO-only extension; the internal struct stays faithful). Recommend the last (Q7).
6. **`tls.truststore` on macOS.** `crypto/x509` does not read the keychain by default; the options are `cgo` (`SECURITY_FRAMEWORK_LOADED` style toolchain burden), shelling out to `security find-certificate` (adds a process dependency and a parser), or reporting `indeterminate` on macOS with a reason (honest, zero risk, small capability loss). Recommend reporting `indeterminate` on macOS in slice 1 and recording the limitation — it is the only option fully consistent with §14.5 (Q9).

### 5.5 Keeping the diagnosis testable without network access

The seam list, all injected, no global mutable hooks as the primary mechanism:

| Seam | Interface | Why it must exist |
|---|---|---|
| `Dialer` | `DialContext(ctx, network, addr) (net.Conn, error)` | TCP matrix, banner reads, TLS; the single most-used seam |
| `Resolver` | `LookupHost(ctx, host) ([]string, error)` | DNS failure is a distinct reason code, not a generic dial error |
| `TLSProber` | handshake with an injectable `*tls.Config` and root pool | Issuer/verify-code assertions offline; also the only way `tls.truststore` is testable |
| `QUICProber` | send/receive against `net.PacketConn` | UDP silence vs response must be scripted, not witnessed |
| `CommandRunner` | `Run(ctx, name, args...) (stdout, stderr, err)` | `sshd -T`, service state, `sw_vers`; also the future `herdr`/`cloudflared` boundary pattern |
| `Clock` | `Now()` / timer injection | Bounded-timeout assertions, deterministic `Elapsed`/`generated_at` |
| Local readers | narrow `Stat`/`ReadFile` wrappers | `/etc/ssh/sshd_config*`, `wsl.conf`, `.wslconfig` fixtures under `t.TempDir()` |

**Injection style**: plain structs with dependencies plus functional options
(`egress.New(egress.WithDialer(d))`), so the default in `main` is real and the default in tests is
scripted. **Explicitly rejected as the primary mechanism**: package-level `var dial = net.Dial`
swapping — it cannot be used with `t.Parallel()`, it hides coupling, and it invites a test that
accidentally performs a real dial because a sub-test forgot to swap.

**The hostile default**: unit tests construct a **deny-all dialer** that fails every dial with a
distinctive error ("network access attempted in unit test"). Any probe that reaches for the real
network fails its own test instead of silently passing because the CI box has internet. This is what
makes the no-egress policy enforceable before a CI workflow exists (§8.2 R9).

What is genuinely not unit-testable offline, and must therefore be *evidence*, not a test: real QUIC
behavior behind an endpoint agent, real TLS interception, real `sshd -T` output on each target distro,
and macOS trust-store behavior. Those become manual verification steps whose transcripts are recorded
in the verify phase of the slices that depend on them (starting with §1.1's matrix reproduction in
this slice's verify).

---

## 6. Strict-TDD strategy for slice 1

`openspec/config.yaml` sets `strict_tdd: true` with `go test ./...`, and the go-testing skill dictates
the patterns. Applied to this slice:

| Target | Pattern (go-testing decision gate) | Notes |
|---|---|---|
| Each probe's classification (pass/fail/indeterminate) | Table-driven with a scripted `Dialer`; `t.Run(tt.name, ...)` named by scenario ("refused connection to hub", "blackholed dial times out") | Scenarios, not inputs. Every probe needs at least one case per verdict class. |
| Dialer failure taxonomy | Table over refused / timeout / NXDOMAIN / reset / TCP-ok-TLS-verify-fail / TLS-intercepted-issuer / non-SSH banner / UDP silence | This table *is* the reason-code contract; it is where "destination blocked" vs "protocol blocked" is proven. |
| Diagnosis rule table | Table-driven, with the six measured rows of PRD §1.1 as literal cases plus counterfactuals (both probes pass; both fail; hub probe `indeterminate`; known-host probe `indeterminate`) | The motivation case (§13: "correctly identifies 'destination blocked, SSH allowed'") becomes a named test case. |
| `indeterminate` never `pass` (NF-03) | Dedicated tests at three layers: engine output, JSON DTO, rendered verdict | §14.5 is the governing rule; it deserves more than one assertion. |
| Control tests (§8.4) | Per-probe obligation: a case asserting the probe returns `fail` and one asserting it returns `indeterminate` | "A probe that cannot fail is not a probe" — an explicit review checklist item, not folklore. |
| Transport feasibility | Table across 4 adapters × finding sets; plus property-style assertions in a loop: `!Viable ⟹ Reason != ""`; `Viable && prerequisites ⟹ notes/requires non-empty` | R-HR-06's "reason in both cases" becomes mechanical. |
| NF-04 (additive transport) | A test-only adapter registered in a test; assert the resulting verdict includes it unchanged | Proves the requirement instead of asserting it in prose. |
| NF-09 (bounded, <60 s) | Injected short timeouts; assert a full run with a uniformly slow scripted dialer completes inside the budget and that a hanging dialer yields `indeterminate` within the bound | Fake clock for `Elapsed`; the real timeout mechanism is exercised because it is context-based. |
| Runner streaming | Plain Go test: collect results from the channel, assert progressive delivery (a result is available before the slowest probe finishes) | The runner imports no TUI package, so `teatest` is not needed to prove streaming. |
| TUI state transitions | `Model.Update()` driven directly with `ResultMsg`/`DoneMsg` sequences; assert one row per probe, stable order, incomplete-run marking | Per the skill: direct `Update` for state, not `teatest`. |
| Full interactive flow | `teatest.NewTestModel()` only if role → diagnose → verdict navigation needs an integration check | Keep it minimal; it is the flakiest test type in the suite. |
| Rendered verdict | Golden file with pinned width/height and a forced ASCII color profile (`termenv.Ascii`) | Update only through the repo's `-update` path, then re-run without it (skill rule). |
| `--json` report | Golden file with a fixed clock; assert field ordering and `schema_version` | Determinism rules from §5.4 are the contract; golden files make drift a review event. |
| "Writes nothing" (R-HR-24, R-HR-27, R-HR-02) | Run the CLI with `HOME=$(t.TempDir())` and a temp CWD; assert the tree is byte-identical afterwards; plus a test that the process never executes a third-party binary (deny-all `CommandRunner`) | The "no mutation" promise is the slice's whole safety claim and must be proven, not stated. |

**Never tested with real network egress in CI** — and how each is closed instead:

| Forbidden in CI | Replacement |
|---|---|
| DNS resolution of `github.com`, `ssh.github.com`, `*.argotunnel.com` | Scripted `Resolver` table |
| TCP dials to any real host | Deny-all `Dialer` by default in tests |
| TLS handshakes to the Cloudflare edge or any intercepting middlebox | Injected `tls.Config` + self-signed fixture chains under `t.TempDir()` |
| UDP/QUIC to the edge | Scripted packet-conn fake (response / silence / ICMP-less drop) |
| `sshd -T`, service state, `sw_vers` | `CommandRunner` fixtures + `t.TempDir()` config files |
| The real `herdr`/`cloudflared` binaries | Not invoked at all in slice 1 |

Also: no test may depend on wall-clock duration beyond generous upper bounds, no test may assume a
particular OS-provided network stack, and `-race` should be run by hand during apply even though the
configured command is `go test ./...` (the runner has concurrent goroutines; a race here is a
correctness bug in the streaming design).

---

## 7. Open questions the proposal must answer, with the evidence that settles each

| # | Question | Evidence that settles it |
|---|---|---|
| Q1 | **Is a dial timeout `fail` or `indeterminate`?** §1.1 reports a timeout as BLOCKED; §11 says a hanging probe resolves to `indeterminate`. These can conflict for the same observable. | An explicit rule table plus one hand-run measurement on the motivating network distinguishing (a) connection refused, (b) silently dropped (blackhole), (c) slow-but-eventually-open. The rule should be: definitive negatives (`refused`, `reset`, DNS NXDOMAIN, TLS verify error) ⇒ `fail`; absence of a response inside the probe's own budget ⇒ `indeterminate` **unless** the probe's purpose is "is this port reachable", in which case the timeout is the measurement — decided once, documented, and tested for the hub probe and the public-host probe separately. |
| Q2 | **Where does the hub address come from in slice 1**, given pairing is deferred and PRD §5.1 assumes "the configured port" with no config package in §7.1? | Options: `--hub host[:port]` flag (zero new surface), an env var, or a small read-only config file. Settle by deciding what the acceptance criterion in §13 needs and by not inventing a config format that the pairing slice will replace. Recommend flag + optional read-only file, with the file format explicitly deferred. |
| Q3 | **Hard-coded probe targets or a configurable list?** §1.1 uses `github.com:22`, `ssh.github.com:443`, `region1.v2.argotunnel.com:7844/443`, `www.cloudflare.com:443`; §8.2 also showed `region2`. | A decision plus a declared, overridable target list, and one test asserting the dialed target set equals the declared set (this is also how NF-10 becomes checkable). Settle by checking whether `region2` adds diagnostic value or only latency, using the motivating network. |
| Q4 | **Does any probe need privilege** (`sshd -T`, reading `/etc/ssh/sshd_config*`)? | Run `sshd -T` as a non-root user on Linux, macOS and WSL2 and record what it emits. If any platform requires root, that probe degrades to `indeterminate` with a reason instead of `fail` — and the verdict must say which capability is missing. |
| Q5 | **`go` directive in `go.mod` and the exact Bubbletea/Bubbles/Lipgloss versions.** README claims Go 1.25.10, the observed toolchain is go1.26.6, and §14.1 requires "identical versions to the ecosystem". | `go version` output (recorded in config), plus a decision on which Go version the project supports publicly. This is a one-way door for contributors: `go 1.25` widens the audience, `go 1.26` matches the author's machine. Also decide whether a `toolchain` line is pinned. |
| Q6 | **Exit-code semantics for `--json`.** Is "no viable transport" a failure? | A written contract plus one script that consumes it. Recommend `0` = completed measurement, `1` = incomplete run (any `indeterminate`), `2` = usage/internal error, and document that "no viable transport" is a successful measurement with a negative answer. |
| Q7 | **How is a conditionally viable transport represented?** (`cloudflare-tunnel` is viable *given* a hostname.) | A decision between `viable:true` + notes, a third state, or a DTO-only `requires []string`. Recommend DTO-only `requires`, keeping the internal struct faithful to §5.2 and avoiding a "sort of pass" state that §14.5 forbids. |
| Q8 | **What does `egress.quic` actually measure**: raw UDP reachability or a real QUIC handshake? Raw UDP silence is indistinguishable from a blocked-but-quiet path, and a real handshake may pull in a QUIC library the rest of V1 does not need. | Compare a hand-run raw-UDP probe against `cloudflared`'s own QUIC log lines on the motivating network (§8.2's transcript is the reference). If they disagree, the probe must report `indeterminate` rather than claim a block. This is the probe most likely to produce a false negative, so the evidence is mandatory before the spec is written. |
| Q9 | **How is `tls.truststore` implemented on macOS** without `cgo`? | Decide between `cgo`, shelling to `security find-certificate` (needs the `CommandRunner` seam and a parser) or `indeterminate` with a documented limitation. Evidence: attempt a Go `crypto/x509` system-pool verification of a keychain-trusted CA on macOS and record the result. |
| Q10 | **Default probe concurrency.** A 10-way burst of outbound connections from a machine running an endpoint agent is a self-inflicted risk; nothing in the PRD covers it. | A judgment call, recorded as such: default 4, configurable, and a test that the semaphore is actually honored. If any hand-run shows the burst causing rate-limiting or agent interference on the motivating network, lower it. |
| Q11 | **Does slice 1 write any file or keep any state** (probe cache, last-diagnosis for a future `status`)? | Decide explicitly. Recommend no writes at all: it keeps the no-mutation claim absolute and testable, and leaves probe freshness to the lifecycle slice. Evidence: §4.5's `status` ("probe freshness") presumes a cache, so the decision must be recorded as a deliberate deferral rather than an oversight. |
| Q12 | **How are TUI renders tested without flakiness** (ANSI codes, terminal width, color profile)? | Pin width/height, force `termenv.Ascii`, and use golden files; verify by running the suite in both a TTY and a pipe and confirming identical output. |
| Q13 | **Does slice 1 fit the 400-line review budget?** Module + CLI + TUI skeleton + 10 probes + engine + 4 feasibility adapters + verdict screen + JSON DTO + tests is plausibly 1200–1800 changed lines. | Estimate during proposal writing. If the estimate exceeds the budget, this is an `ask-on-risk` decision, not a silent exception: split into R1a (probe core + engine + JSON, headless) and R1b (TUI screens) as proposed in §4.2. Do not invent `size:exception`. |
| Q14 | **Should the role-selection screen ship in slice 1?** R-HR-01 is P0 and says the TUI *must begin* by asking. | Recommend yes, with an honest hub-branch placeholder. The alternative — a node-only binary — would violate R-HR-01's literal text and force a screen-shell change later. |
| Q15 | **Is the diagnosis JSON report promoted to `contracts/`?** PRD §7.7 only lists pairing artifacts, but §4.4 says "`--json` everywhere" and users will script it. | Decide in design. Recommend: not in slice 1 (no schema file, no cross-machine consumer yet) but include `schema_version` so promotion is additive, and reserve the `contracts/diagnosis/v1` path in the design note. |
| Q16 | **What exactly is "raw evidence" in `--json`?** Verbatim Go error strings are not a stable contract, yet §5.1's `Detail` is explicitly quotable and every screen must show it. | The Q1/§5.4 reason-code decision settles this: verbatim `detail` plus a stable `reason` code. Evidence: attempt to golden-test a real `net.OpError` string across two Go versions and observe the drift. |

---

## 8. Risks

### 8.1 Slice-1 risks

| # | Risk | Impact | Mitigation |
|---|---|---|---|
| R1 | **Scope exceeds the review budget** (Q13) | A 1500-line single PR erodes review quality, and the review budget is an explicit session choice | Pre-plan the R1a/R1b split; if the proposal's estimate is over 400 lines, raise it as an `ask-on-risk` decision *before* apply |
| R2 | **A probe's classification is wrong** (timeout read as blocked, or blocked read as indeterminate) | Breaks the product's single credibility claim (§14.5, §13 "silent failures: zero") and every downstream slice's verification | Q1's rule table, reason codes, per-probe control tests, and the §1.1 matrix as a named acceptance test |
| R3 | **`egress.quic` produces a false negative** | Wrongly forces HTTP/2 (a real, if small, post-quantum downgrade) or reports a blocked network as open | Q8's evidence requirement; prefer `indeterminate` over an unsupported claim |
| R4 | **`tls.truststore` scope creep into macOS/keychain work** | Slice 1 grows into a platform-integration project | Q9: report `indeterminate` on macOS in this slice, document the limitation |
| R5 | **TUI test flakiness** (teatest, ANSI, timing) | Erodes trust in `go test ./...` and slows every later slice | Test `Model.Update()` directly, golden renders with pinned profile and geometry, minimal use of `teatest`, no duration-sensitive assertions |
| R6 | **Accidental real network use in tests** | Non-deterministic, unreviewable test failures, and the NF-10 "no telemetry" story becomes murky | Deny-all dialer as the test default; a test asserting the dialed target set is closed and declared |
| R7 | **`transport.Transport` interface stubs.** Go has no partial interfaces: defining §5.2's interface in slice 1 means `PlanHub`/`PlanNode`/`Verify` exist before anything can implement them | Either dead code ships, or the interface changes later and touches every adapter | Two honest options, decide in design: (a) define the full interface now and have stubs return a typed `ErrNotImplementedInThisPhase`, *with a test asserting they return it* (so no caller can ever receive a fabricated plan); or (b) define a narrower `Candidate` interface (`Name`/`Requires`/`Feasible`) now and grow it in the provisioning slice. (a) is PRD-faithful and avoids churn; (b) avoids dead code. Recommendation: (a), because §5.2 is a stated interface and because "returns not-implemented" is a testable, honest behavior |
| R8 | **Rule/data duplication between the engine and the adapters** | Adapters disagree about the same network; R-HR-06 becomes unreliable | §5.3's split: engine states facts, adapters state reasons; a test-only adapter proves the boundary |
| R9 | **No CI workflow exists**, so "no network in CI" is only as strong as the deny-all dialer | A future contributor adds a real-network test and nothing catches it | Deny-all default now; add the workflow in the distribution slice (R11), or explicitly accept the gap in slice 1 and record it |
| R10 | **Greenfield bootstrap friction**: `go test ./...` fails until `go.mod` exists, and the repo has no linter | Apply cannot run the configured test command until the module exists, and a first red verify looks like a failure | The apply phase's first commit is the module bootstrap; state in the tasks that the first honest `go test ./...` can only run after `go.mod`, `go.sum` and at least one package exist |
| R11 | **Artifact-store divergence** (openspec file vs Engram mirror) | Two versions of the same exploration, and later phases read the stale one | This file is the reviewable source the proposal cites; the Engram mirror is written from the same text in the same phase |

### 8.2 Downstream risks the roadmap must carry (not slice-1 work, but roadmap consequences)

| Risk | Why it matters to the roadmap |
|---|---|
| R5 (`reach-cloudflare-node`) will exceed 400 lines | It must be pre-split (install / sshd / tunnel+verify) or the split decided during its own proposal; do not discover it at apply time |
| WSL2 recovery is only provable by killing the environment | The WSL2 slice's verification cannot be a `go test`; it needs a recorded measurement (PRD §6.2's ≤20 s), so its spec must define the evidence artifact |
| Dev Tunnels may be a negative result | R10's deliverable is the evidence, not the adapter; the roadmap must allow the change to close without an adapter |
| `cloudflared` pin has a stated exit criterion | The pin and its dated note need a single home (a constant next to the Cloudflare transport), never duplicated in docs |

---

## 9. Candidate acceptance criteria for slice 1 (input to the later spec)

Traceable, mostly offline, each with the evidence it produces:

1. `go test ./...`, `go vet ./...` and `gofmt -l .` pass with `go.mod` at `github.com/Luisalt20/herdr-reach` (R-…, config).
2. All ten §5.1 probes exist, are independently runnable, and each has a passing control case proving it can return `fail` and one proving it can return `indeterminate` (§8.4).
3. The §1.1 measured matrix, replayed against scripted dialers, yields "SSH is allowed; the destination is blocked" and recommends a transport whose reason names the blocked hub (R-HR-03, R-HR-06, §13 "diagnosis accuracy").
4. The verdict lists all four adapters with viable/not and a non-empty reason for each rejection (R-HR-06, §4.4).
5. `herdr-reach doctor --json` emits a schema-versioned report with stable ordering, no `indeterminate` mapped to `pass`, and no human text on stdout (R-HR-07, R-HR-NF-03).
6. A run with a hanging probe completes, marks it `indeterminate`, and the report says the run is incomplete (R-HR-NF-02, §11).
7. A full node diagnosis completes within the 60 s budget with every probe individually bounded (R-HR-NF-09).
8. A native-Windows node is refused with an explanation naming WSL2 as the supported path (R-HR-30, §11).
9. Running the CLI leaves the filesystem byte-identical (temp `HOME`), executes no third-party binary, and dials only the declared target set (R-HR-02, 24, 27, NF-10).
10. Adding a transport in a test requires no change under `internal/diagnosis` (R-HR-NF-04).

---

## 10. Handoff to the next phases

**For the proposal**: approve or amend §4's roadmap first (it is the artifact the user approves), then
scope slice 1 to R1 — or R1a + R1b if the size estimate exceeds the review budget (Q13). The proposal
must carry the problem statement (PRD §1.1's measured matrix in the user's own words), and its scope
section must state the "zero mutation" boundary explicitly.

**For the spec**: §3's coverage table becomes the requirement delta — write requirements only for what
this slice implements, and record the deferred IDs with their target slice so nothing looks forgotten.
`R-HR-*` IDs should be cited rather than renumbered, since the PRD is the source of truth.

**For the design**: settle, with tradeoffs recorded per `rules.design.require_tradeoffs`: package
boundaries (§5.1, including the recorded §7.1 deviation), the streaming runner (§5.2), the rule table
and its rule ids (§5.3), the DTO and its exit codes/streams/determinism/reason codes (§5.4), the seam
set and the deny-all test default (§5.5), the `Transport` interface stub strategy (R7), the `go.mod`
Go directive and TUI dependency versions (Q5), and the hub-address source (Q2).

**For the tasks**: order as module bootstrap → seams + `Result`/`Verdict` types + runner → probes one
at a time (each with its control tests) → engine rule table → feasibility adapters → DTO + golden →
TUI shell/router/screens → "writes nothing" guard tests → verify. Every task carries its test first
(`strict_tdd: true`).

**Not decided here, deliberately**: anything the PRD leaves open, anything requiring a real network
measurement (Q1, Q3, Q4, Q8, Q9), and the delivery split (that is an `ask-on-risk` decision at
proposal time, per session preflight).
