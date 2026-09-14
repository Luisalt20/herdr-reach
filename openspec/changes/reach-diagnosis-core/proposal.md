# Proposal — `reach-diagnosis-core` (R1a: headless diagnosis core)

**Change**: `reach-diagnosis-core` — slice **R1a** of the approved roadmap R1–R12
**Date**: 2026-09-14
**Phase**: proposal (no implementation, no scaffolding, no `go.mod`, no source files written)
**Artifact store**: hybrid — this file plus Engram topic `sdd/reach-diagnosis-core/proposal`
**Depends on**: exploration (obs. 791), research (obs. 793, `outcome: partial`), orchestrator-confirmed pre-proposal handoff (obs. 792)
**Preflight consumed**: execution mode `auto`, artifact store `hybrid`, delivery strategy `ask-on-risk`,
review budget 400 changed lines, chain strategy deferred, `size:exception` never inferred.

---

## Summary

Build the headless measurement core of `herdr-reach`: a Go module, the ten probes of PRD §5.1, a pure
diagnosis engine that implements the port-vs-protocol disambiguation rule, transport feasibility for the
four V1 adapters as **detect-and-recommend only**, and a schema-versioned `--json` report. R1a writes
nothing to the user's machine and executes no third-party binary, and proves both.

Why R1a first: every later slice consumes `Result` / `Diagnosis` / `Feasibility`, and a wrong measurement
makes every downstream verification meaningless. R1a is also the only slice that is useful on its own
(`herdr-reach doctor`), and a bug in it cannot damage a work laptop.

---

## Problem statement

Herdr 0.9 multi-machine assumes SSH reachability. On the machines this product exists for — corporate
laptops behind endpoint agents, ISP CGNAT, egress blocked by destination — that assumption fails, and the
failure is indistinguishable from a network that blocks SSH itself. Measured from a locked-down corporate
machine on a FortiClient network (PRD §1.1):

| Target | Port | Result | What it proves |
|---|---|---|---|
| `github.com` | 22 | **OK** | SSH is not blocked as a protocol |
| `ssh.github.com` | 443 | **OK** | SSH-over-443 works when the port is allowed |
| Your VPS | 22, 2222, 443 | **BLOCKED** | The block is on the **destination**, not the protocol |
| `region1.v2.argotunnel.com` | 7844 | **OK** | Cloudflare tunnel egress is allowed |
| `region1.v2.argotunnel.com` | 443 | **OK** | Same, over HTTPS |
| `www.cloudflare.com` | 443 | **OK** | No TLS interception (`issuer = Let's Encrypt/ISRG`, `verify code 0`) |

In the specification's own words: *"Read carefully what that table says. Every conclusion a human draws
from 'my corporate network blocks SSH' is wrong. SSH works. The destination is blocked. The tunnel edge is
reachable. The only thing missing was the measurement."*

Today the same conclusion costs eleven undocumented traps (PRD §1), each of which fails silently and looks
like something else: discovering that inbound is hopeless, that Quick Tunnels are HTTP-only and cannot
carry SSH, that a named tunnel needs a domain **and** account membership for the zone, that
`cloudflared access ssh` is reported to ignore service tokens on 2026.6.0, that Herdr cannot target native
Windows, that systemd services do not keep a WSL2 instance alive, that `vmIdleTimeout` is a second
shutdown mechanism — and finally getting a correct key into the wrong file. *"Each numbered step is an
evening lost by a competent developer."*

**Why measurement must precede provisioning.** Provisioning is the irreversible half: it picks a transport,
changes machine configuration, and can cost a domain purchase. The evidence above shows the naive
measurement ("SSH is blocked") points at the wrong transport, so provisioning without measurement
systematically installs the wrong thing and then has to explain itself. PRD §13 makes this measurable —
`diagnosis accuracy on the motivating case` = *correctly identifies "destination blocked, SSH allowed"*,
`silent failures` = zero — and §14.5 makes it governing: **when the tool cannot establish something, it says
so, and it says why.** R1a exists to make that rule true before anything is allowed to mutate a machine.

---

## Intent

1. Turn §1.1's measured matrix into executable, offline-reproducible logic, so the tool's central claim is
   testable rather than asserted.
2. Ship a headless `doctor` surface (`herdr-reach doctor [--json]`) whose output is deterministic, versioned
   and scriptable.
3. Establish the shared value types (`Result`, `Verdict`, `Diagnosis`, `Feasibility`) and the seam set that
   every later slice consumes, so R2–R12 build on a frozen measurement vocabulary instead of inventing one.
4. Carry the confirmed pin knowledge (client-side `cloudflared` version + exit criterion) as *stated
   knowledge with a single home*, without pretending the tool verified or installed anything.

---

## Scope

### In scope for R1a

| Area | What R1a delivers |
|---|---|
| Module bootstrap | `go.mod` at the ratified path `github.com/Luisalt20/herdr-reach`; first runnable `go test ./...` |
| `internal/probe` | The ten PRD §5.1 probes, each independently runnable, behind injected seams |
| Probe runner | Bounded per-probe timeouts, global 60 s budget, bounded concurrency, streamed results, cancellation |
| `internal/diagnosis` | Pure reasoning: classify → disambiguate → verdict, as an ordered rule table with stable rule ids |
| `internal/transport` | Registry + `Requires`/`Feasible` for `direct-ssh`, `reverse-ssh`, `cloudflare-tunnel`, `tailscale` — **detect-and-recommend only** |
| `internal/report` | `schema_version`-carrying JSON DTO, one mapping function, deterministic ordering |
| CLI | Headless `doctor` with human text on stderr and `--json` on stdout; documented exit codes |
| Safety proof | "Writes nothing" guard tests: byte-identical sandbox tree, no third-party exec, closed dial target set |

### Out of scope, with the slice that owns it

| Out of scope | Owner |
|---|---|
| TUI shell, router, role / diagnose / verdict screens (with it, `R-HR-01`, `NF-01`, `NF-07`) | **R1b** `reach-tui-diagnosis` |
| Any file write, `Step`, plan/apply/undo, `internal/{sysfile,sshcfg}`, plan screen | R3 `reach-mutation-primitives` |
| `PairingBundle` / `NodeReceipt`, `contracts/`, fingerprint/nonce | R2 `reach-pairing-contracts` |
| `internal/environ`, user-scope units, `launchd` | R4 `reach-environ-core` |
| `cloudflared` install + SHA256 + self-version verification, `sshd` provisioning/hardening, forcing `TUNNEL_TRANSPORT_PROTOCOL=http2` in a unit | R5 `reach-cloudflare-node` |
| Hub flow, `~/.ssh/config`, `internal/shedr`, `herdr machine add` | R6 `reach-hub-flow` |
| WSL2 keepalive, `vmIdleTimeout` handling, watchdog recovery | R7 `reach-wsl2-lifetime` |
| `status` / `verify` / `disconnect` / `remove`, `control-socket` lifecycle | R9 `reach-lifecycle` |
| Microsoft Dev Tunnels adapter (and its unverified raw-TCP / service-principal questions) | R10 `reach-devtunnels-adapter` |
| Packaging, brew tap, goreleaser, published schemas, **CI workflow** | R11 `reach-distribution` |
| Native Windows node support | Blocked upstream; never in V1 (PRD §2.1) |

**Scope boundary on adapters.** R1a implements feasibility only for the four adapters in PRD §5.2. Dev
Tunnels is explicitly *not* one of them: it is R10, and its open research questions (SSH through a dev
tunnel, service-principal login) must not leak into R1a as either a claim or an adapter.

### Zero-mutation boundary (explicit)

R1a **does not write, create, modify, move or delete any file on the user's machine outside its own build
tree**, does not install or invoke `cloudflared`, `sshd`, `herdr` or any other third-party binary, and does
not create services, units, config entries, keys or tokens. Concretely:

- No code path in R1a performs a filesystem write outside test fixtures under `t.TempDir()`.
- No code path in R1a executes an external command; `CommandRunner` exists only as an injected seam whose
  production implementation is unused in R1a, and whose test default denies everything.
- The tool performs outbound network *measurements* only (TCP/TLS/UDP probes), and only against the declared
  target set.

**How it is proven — the "writes nothing" test.** A test invokes the CLI with `HOME` and the working
directory pointed at a fresh temp tree, records a recursive digest of that tree, runs the diagnosis, and
asserts the tree is byte-identical afterwards. Companion assertions in the same test: the injected
`CommandRunner` is called zero times (any third-party exec attempt fails the test by design), and the set of
dialed `(host, port, protocol)` triples equals the declared probe target set exactly.

**What that proof does not cover** (stated so a reviewer does not over-credit it): a temp-`HOME` digest
cannot prove the absence of writes to paths outside `HOME`/CWD (for example `/tmp` or a cache directory keyed
to another variable). R1a closes this by construction (no writer code exists and no write-capable dependency
is imported) rather than by observation, and this residual gap is carried in the risk register. A
syscall-level trace is deliberately not proposed here; it would add a platform-specific test dependency to a
slice whose whole premise is portability.

---

## Affected areas

| Path | Change |
|---|---|
| `go.mod`, `go.sum` | **New** — module `github.com/Luisalt20/herdr-reach` |
| `cmd/herdr-reach/**` | **New** — headless entrypoint, flag parsing, exit codes |
| `internal/probe/**` | **New** — ten probes, seams, runner |
| `internal/diagnosis/**` | **New** — classification, rule table, verdict |
| `internal/transport/**` | **New** — registry + four feasibility adapters |
| `internal/report/**` | **New** — versioned JSON DTO |
| `internal/version/**` | **New** — version string plumbing |
| `PRD.md`, `README.md`, `LICENSE`, `docs/assets/**` | **Unchanged** |
| `openspec/**` | Unchanged except the artifacts of this change |
| User machine state | **Untouched** — no writes, no services, no binaries, no config |

**Recorded deviation to settle in design, not here.** PRD §7.1 does not list an `internal/diagnosis`
package; the exploration recommends it (`internal/probe` measures, `internal/diagnosis` reasons,
`internal/transport` explains why a transport is or is not viable) as a recorded deviation, with the
dependency direction `cmd → transport/report → diagnosis → probe (types only)`. The design phase must record
that deviation next to the §7.1 diagram or fall back to the faithful layout. The proposal does not decide it.

---

## Requirement coverage at R1a granularity

Coverage vocabulary: **Covered** / **Partial** (named part now) / **By-construction** (held only because
nothing is mutated — still needs an explicit test) / **Deferred** (owner slice named).

| ID | R1a | Note |
|---|---|---|
| R-HR-01 | Deferred → **R1b** | R1a ships no TUI, so R1 as a whole satisfies this only once R1b lands. Stated rather than implied. |
| R-HR-02 | Covered | Probe before recommend; nothing modified at all in R1a |
| R-HR-03 | Covered | Port-vs-protocol disambiguation is the engine's centerpiece |
| R-HR-04 | Covered | `tls.interception` reports issuer + verification code |
| R-HR-05 | Partial | Detect + recommend + trade-off note; `http2` unit enforcement → R5 |
| R-HR-06 | Covered | All four adapters listed with viable/not and a reason in both cases (`!Viable ⟹ Reason != ""`) |
| R-HR-07 | Covered | Raw evidence in the human view; full diagnosis as `--json` |
| R-HR-08…11 | Deferred → R2 | No pairing artifact exists; nothing is claimed about it |
| R-HR-12…15 | Deferred → R3 | No plan, no backup, no dry-run, no undo |
| R-HR-16 | Partial | Pin knowledge + exit-criterion home only (see risk RG-1); install/SHA256/self-version → R5 |
| R-HR-17 | Deferred → R5 | Nothing is installed |
| R-HR-18 | Partial | `local.sshd` reads binary presence, unit state, `sshd -T` and reports written-vs-effective divergence; hardening → R5 |
| R-HR-19 | Partial | No key installed; the evidence shape must carry **what and where** (owner, mode, absolute path) from day one |
| R-HR-20 | Deferred → R4 | Nothing is persisted |
| R-HR-21 | Partial | WSL2 *detection* only; documented-semantics note at most (see risk RG-4) |
| R-HR-22, 23 | Deferred → R7 | No keepalive exists here, so it cannot misbehave |
| R-HR-24 | By-construction | Needs the explicit no-write test |
| R-HR-25 | Partial | SSH-reachability probe seam only; end-to-end auth needs R5+R6 |
| R-HR-26 | Deferred → R6 | `herdr` is never invoked |
| R-HR-27 | By-construction | Needs the explicit no-exec assertion |
| R-HR-28 | Deferred → R9 | Nothing was created to remove |
| R-HR-29 | Partial | Platform classification only (Linux / macOS / WSL2 / native Windows); provisioning → R4/R5/R7 |
| R-HR-30 | Covered | Refuse a native-Windows node, naming WSL2 as the supported path |
| NF-01 | Deferred → **R1b** | No TUI in R1a |
| NF-02 | Covered | Hanging probe resolves to `indeterminate`; the run is never reported complete |
| NF-03 | Covered | Core invariant, asserted at engine, DTO and rendering layers |
| NF-04 | Covered | Proved by a test-only transport registered in a test, with no change under `internal/diagnosis` |
| NF-05 | Deferred → R4 | No `Environment` interface here; defining it now would ship an unused abstraction |
| NF-06 | Deferred → R2/R11 | `--json` carries `schema_version` so promotion to `contracts/diagnosis/v1` is additive; no schema file yet |
| NF-07 | Deferred → **R1b** | Bubbletea/Bubbles/Lipgloss arrive with the TUI |
| NF-08 | By-construction | Nothing mutates, so idempotency is held vacuously and must not be presented as demonstrated |
| NF-09 | Covered | Full diagnosis under 60 s with every probe bounded; provable offline with injected short timeouts |
| NF-10 | Covered | No telemetry; the dialed target set is closed, declared and asserted |

---

## Evidence status carried into this proposal

The research record is `outcome: partial`, and the user accepted it as such. Nothing below is upgraded by
this proposal.

**Fully source-backed (citable as fact)**

- Tunnel transport: `auto` configures QUIC and falls back to HTTP/2 over TCP when UDP fails;
  `TUNNEL_TRANSPORT_PROTOCOL` accepts `auto|http2|quic`; port 7844 is UDP for QUIC and TCP for HTTP/2.
- Cloudflare states post-quantum key agreements are not supported when using HTTP/2.
- The macOS trust path: Go verifies via the platform APIs with `opts.Roots == nil`, implemented with
  cgo-less Security.framework wrappers; setting `SSL_CERT_FILE`/`SSL_CERT_DIR` prevents those APIs from being
  used.
- Release `2026.5.1` exists with published SHA256 checksums.
- `vmIdleTimeout` is documented as milliseconds idle before VM shutdown, default 60000, Windows 11 only.

**Report-only, single-source (must not be presented as established)**

- The `cloudflared` 2026.6.0 service-token regression: upstream issue #1673 is open, zero comments, an
  unlabelled report from a non-member reporter. The PRD's generalization *"at or above 2026.6.0"* is **not
  established**, and no release note documents a fix.

**Unverified (carried as unknowns, never as facts)**

- Whether releases after 2026.6.0 are affected by, or fix, that report.
- Whether an SSH client can complete a session through `devtunnel connect`, and whether Dev Tunnels supports
  service-principal or managed-identity login. *R10 only; not used by R1a.*
- The WSL2 child-of-init shutdown rule and the `vmIdleTimeout=-1` sentinel: both appear only in unfetched
  GitHub issue snippets, not in Microsoft reference documentation.

---

## Risk register

Status column: **V** = verified by the research record, **RO** = report-only single-source, **UV** =
unverified, **D** = delivery/engineering risk.

| # | Risk | Status | Impact on R1a | Mitigation |
|---|---|---|---|---|
| RG-1 | The `cloudflared` service-token report is single-source, and the "at or above 2026.6.0" range is unverified | **RO / UV** | R1a must state the pin *as knowledge*. If its verdict text repeats the PRD's version-range claim, the tool's first product-facing sentence is a fabrication, violating §14.5 | Word the note as: issue #1673, open, uncommented, names **2026.6.0 only**; `2026.5.1` publishes SHA256 checksums; later releases' behaviour is **unknown**. Provide exactly one home for the constant + dated exit criterion (next to the Cloudflare transport). No version range, no "confirmed fixed" |
| RG-2 | An `indeterminate` presented as `pass` anywhere | **D** | Breaks the product's only credibility claim and every downstream slice's verification | Assert NF-03 at three layers (engine output, JSON DTO, human render); a run with any `indeterminate` exits `1` and is reported as incomplete. §1.1's six rows become literal engine test cases plus counterfactuals |
| RG-3 | The macOS trust-store probe cannot do what the Linux one does: Go cannot enumerate macOS system roots, and keychain trust is only visible through the platform verifier with nil `Roots` | **V** (mechanism) + **D** | Shipping an "always indeterminate" probe on macOS looks like a broken probe and could invite a cgo/shell-out scope creep | Confirmed decision: report `indeterminate` on macOS and state the documented limitation **in the output**. Prove the probe can pass and fail on Linux with control cases, and add a case where `SSL_CERT_FILE`/`SSL_CERT_DIR` is set (the platform verifier would be bypassed) → `indeterminate`, never a false pass |
| RG-4 | The WSL2 lifetime rule is not documented: neither the child-of-init shutdown rule nor `vmIdleTimeout=-1` appears in Microsoft reference docs; only the millisecond semantics are verified | **UV** | Any WSL2 note in R1a that repeats those rules as documented fact would be unestablished | R1a's WSL2 output is limited to detection plus the documented semantics (milliseconds idle, default 60000, Windows 11 only). The child-of-init rule and the `-1` sentinel are excluded from R1a text entirely; R7 owns them and owes the measurement |
| RG-5 | `egress.quic` false negative: raw UDP silence is not proof of blockage, and a real QUIC handshake pulls in a dependency the rest of V1 may not need | **D** | Wrongly recommends forcing HTTP/2 (a real post-quantum downgrade on the tunnel transport) or reports a blocked path as open | Design must settle what the probe measures against a recorded hand-run comparison; when the signal is ambiguous the probe returns `indeterminate` rather than claiming a block |
| RG-6 | Recommendation wording could imply the HTTP/2 fallback is equivalent to QUIC | **V** | Cloudflare documents that HTTP/2 forfeits post-quantum key agreement on the tunnel transport; a silently equivalent-looking recommendation trades a security property without saying so | The verdict note states the trade-off explicitly and states that R1a only *recommends* — enforcement is R5. Never claim a fallback was applied |
| RG-7 | Accidental real network use in tests | **D** | Non-deterministic, unreviewable failures; the NF-10 no-telemetry story becomes murky | Deny-all `Dialer`/`Resolver`/`CommandRunner` as the test default; a test asserting the dialed target set equals the declared set. No CI workflow exists to enforce this (see RG-10) |
| RG-8 | Wrong probe classification (timeout read as blocked, or blocked read as `indeterminate`) | **D** | The disambiguation rule is the product's centerpiece; a wrong classification invalidates the recommendation | One documented rule table resolved once, reason codes alongside verbatim detail, per-probe control tests proving `fail` and `indeterminate`, and §1.1's matrix as a named acceptance test |
| RG-9 | `Transport` interface stubs: Go has no partial interfaces, so `PlanHub`/`PlanNode`/`Verify` would exist before anything implements them | **D** | Either dead code ships, or callers can receive a fabricated empty plan | Design must choose: full §5.2 interface with a typed not-implemented error **and a test asserting it** (recommended, PRD-faithful), or a narrower `Candidate` interface grown later. A caller must never receive a plausible-looking empty plan |
| RG-10 | No CI workflow exists, so "never dial the real network" is a property of test construction only | **D** | A future contributor adds a real-network test and nothing catches it | Deny-all default now; the workflow lands in R11. The gap is recorded here rather than assumed closed |
| RG-11 | Greenfield bootstrap friction: `go test ./...` fails today ("directory prefix . does not contain main module") | **D** | A first red verify looks like a failure and could be misread as a regression | The first work unit is exactly the module bootstrap; the tasks artifact must state that the canonical test command only becomes runnable once `go.mod`, `go.sum` and one package exist |
| RG-12 | Toolchain ambiguity: observed `go1.26.6` vs the README's declared Go 1.25.10 | **D** | The `go` directive is a one-way door for contributors; choosing by accident narrows the audience silently | Design decides the `go` directive and whether `toolchain` is pinned, with the audience trade-off recorded. The bootstrap unit must not resolve it implicitly |
| RG-13 | `egress.hub.direct` needs a hub address, and pairing (which carries it) is deferred | **D** | A headless slice could invent a config format that R2 immediately replaces | Design picks the smallest surface (a `--hub host[:port]` flag is the exploration's recommendation) and explicitly defers the file format. R1a must be honest when no hub is supplied instead of guessing one |
| RG-14 | The zero-mutation test's reach | **D** | Over-crediting the proof would itself be a fabricated guarantee | The test's coverage boundary is documented in this proposal; the residual gap is closed by construction, not by observation |
| RG-15 | Review-budget overrun | **D** | See below | Chained split decided in the tasks phase under `ask-on-risk` |
| RG-16 | Artifact-store divergence (this file vs its Engram mirror) | **D** | Later phases could read a stale copy | This file is the reviewable source; the Engram mirror is written from the same text in the same phase |

### Review-budget risk, stated honestly

**R1a is expected to exceed the 400-changed-line review budget.** The exploration's estimate for the whole of
R1 for the module, probes, engine, four feasibility adapters, report DTO, TUI and tests is 1200–1800 changed
lines; R1a is the larger, more test-dense half (ten probes plus the rule table plus the JSON contract, all
under `strict_tdd: true`). This proposal therefore does **not** claim an exception and does **not** assume
`size:exception`. Delivery strategy is `ask-on-risk`: the chained-split decision lands in the **tasks**
phase, where the workload is planned as reviewable work units and the split can be made against a real
estimate. Candidate chain axes for that decision, offered as input and not as a choice (chain strategy stays
deferred): (1) module bootstrap + seams + `Result`/`Verdict` + runner, (2) the ten probes with their control
tests, (3) engine rule table + classification, (4) transport feasibility for four adapters, (5) report DTO +
`doctor --json` + the writes-nothing guard tests.

---

## Rollback

R1a is a purely additive, non-mutating change, so rollback is a revert.

| Rollback trigger | Action | Consequence |
|---|---|---|
| Any success criterion fails after merge | Revert the change's commits | New files (`go.mod`, `cmd/**`, `internal/**`) are removed; no user-machine state, service, key, token or config entry was ever created, so there is nothing to undo |
| A wrong classification is found in the rule table | Revert the engine commits only; the probe layer stays | `doctor --json` becomes unavailable; no user impact |
| The probe suite proves harmful on a real machine | Remove the binary; optionally revert the module | The repository returns to greenfield, and `go test ./...` returns to its current unrunnable state — an accepted consequence, recorded so it is not mistaken for a defect |

No migration, no data, no persistence, no backward-compatibility window. Because R1a mutates nothing,
rollback carries no user-facing risk — which is the reason this slice is first.

---

## Success criteria

Every criterion below is checkable by a reviewer from the repository alone. Criteria marked **[G]** are
direct applications of the governing rule (§14.5): nothing is presented as success that was not established.

**Build and toolchain**

- [ ] `go test ./...`, `go vet ./...` and `gofmt -l .` pass with `go.mod` declaring
      `github.com/Luisalt20/herdr-reach`. *Note for the reviewer*: the canonical test command only becomes
      runnable after the bootstrap work unit; the tasks artifact must order and label that explicitly, and
      the first honest test run is its acceptance evidence.
- [ ] No linter is introduced; `quality.lint` stays empty until someone decides otherwise.

**Probes**

- [ ] All ten PRD §5.1 probes exist and are independently runnable.
- [ ] Each probe has a control case proving it can return `fail` and one proving it can return
      `indeterminate` — "a probe that cannot fail is not a probe" (§8.4). The macOS trust-store probe
      satisfies its `pass`/`fail` controls on Linux and additionally asserts `indeterminate` on macOS
      (RG-3).
- [ ] The dialed `(host, port, protocol)` set equals the declared probe-target set exactly, and the target
      list is declared in one place and overridable.
- [ ] `local.sshd` reports written-vs-effective configuration divergence, and degrades to `indeterminate`
      with the missing capability named when `sshd -T` is unavailable or needs privilege.
- [ ] A native-Windows node is refused with an explanation naming WSL2 as the supported path.

**Diagnosis and reasoning**

- [ ] Replaying §1.1's six-row matrix against scripted dialers yields *"SSH is allowed; the destination is
      blocked"* and the verdict names the blocked hub, with the rule id recorded in output. This is the PRD
      §13 `diagnosis accuracy` case, as a named test.
- [ ] `!Viable ⟹ Reason != ""` holds for every transport, enforced mechanically over the four adapters.
- [ ] An `indeterminate` probe cannot yield a confident conclusion that depends on it; the engine emits an
      explicitly weaker finding instead of falling through to a default.
- [ ] Adding a transport in a test requires no change under `internal/diagnosis` (NF-04).

**Report and honesty — [G]**

- [ ] `doctor --json` emits `schema_version`, is deterministically ordered (registry order, sorted
      transports, no map iteration), uses one injected clock for `generated_at`, and writes no human text to
      stdout.
- [ ] Exit codes are documented and tested: `0` = measurement completed, `1` = run incomplete (any
      `indeterminate`), `2` = usage/internal error; "no viable transport" is a **completed** measurement with
      a negative answer, not an error.
- [ ] `indeterminate` is never rendered as, or encoded as, `pass` — asserted at the engine layer, the JSON
      DTO and the human render.
- [ ] A run containing a hanging probe completes, marks that probe `indeterminate`, and reports the run as
      **incomplete**; the verdict never implies the suite finished (§11).
- [ ] Verbatim evidence is preserved alongside a stable machine reason code, so the humane quote and the
      scriptable classification come from the same result.
- [ ] **[G]** The `cloudflared` pin note names issue #1673 with its state, names **2026.6.0 only**, records
      that later releases' behaviour is unknown, and cites the published `2026.5.1` SHA256 evidence — no
      version-range generalization (RG-1).
- [ ] **[G]** Any WSL2 output is limited to documented semantics; the child-of-init rule and the
      `vmIdleTimeout=-1` sentinel do not appear (RG-4).
- [ ] **[G]** The HTTP/2 recommendation states the post-quantum trade-off and does not claim the fallback was
      applied (RG-6).
- [ ] **[G]** The macOS trust-store limitation is emitted in the output itself, not only in the code (RG-3).
- [ ] **[G]** A reviewer can trace every claim in the verdict and the JSON report to either a probe result or
      a cited source with its status. Nothing is asserted at a higher confidence than its source.

**Safety**

- [ ] The "writes nothing" test passes: with `HOME` and CWD in a temp tree, the tree is byte-identical after
      a run, the `CommandRunner` is called zero times, and the dialed set matches the declared set.
- [ ] The tool never invokes `herdr`, `cloudflared` or any other third-party binary (R-HR-27).
- [ ] A full diagnosis completes within the 60 s budget with every probe individually bounded (proven in
      milliseconds with injected short timeouts), and cancelling the run stops in-flight dials.

---

## Open decisions for the design phase

Listed so they are settled deliberately; none of them changes this proposal's scope.

| # | Decision |
|---|---|
| D1 | Package boundaries, including the recorded PRD §7.1 deviation (`internal/diagnosis`) and the `Result` ownership question |
| D2 | The classification rule table: timeout as `fail` vs `indeterminate`, resolved once, documented and tested per probe purpose |
| D3 | Hub address source for `egress.hub.direct` in a headless slice (RG-13) |
| D4 | The `go` directive (and whether `toolchain` is pinned) plus TUI dependency versions (RG-12) |
| D5 | Exit-code contract, stdout/stderr split, and reason-code vocabulary |
| D6 | Representation of a conditionally viable transport (`cloudflare-tunnel` needs a hostname the tool cannot create) — a third "sort of viable" state is ruled out by §14.5 |
| D7 | `Transport` interface stub strategy (RG-9) |
| D8 | What `egress.quic` actually measures, and the evidence that settles it (RG-5) |
| D9 | Default probe concurrency (endpoint agents make a ten-way burst a self-inflicted risk; a small bounded default is the exploration's recommendation) |
| D10 | Whether `region2` is probed in addition to `region1`, and whether the target list is overridable by flag |

---

## Proposal question round

This phase could not interview the user directly, so the questions it would have asked are recorded here.
They are product/PRD-shaped and deliberately do **not** re-open the confirmed decisions (R1a/R1b split,
research accepted as partial, macOS `indeterminate`, approved roadmap). If the answers contradict the
assumptions below, this proposal should be amended before the spec phase.

**Questions**

1. **Day-one interface for the hub address.** `egress.hub.direct` measures "TCP to the hub on the configured
   port", but pairing (which carries the hub address) is R2. Should R1a expose a `--hub host[:port]` flag and
   report honestly when it is absent, or should it ship with no hub input at all and mark that probe as not
   measured on day one? This decides what the user types in their first successful run.
2. **How should "no viable transport" read?** Is a completed measurement that finds nothing viable a success
   with a negative answer (exit `0`, `viable: false`), or a failure (non-zero exit)? The proposal assumes
   success-with-a-negative-answer; scripts and the product's promise both depend on this.
3. **Product-facing wording for the `cloudflared` pin.** The PRD §8.3 sentence says "versions at or above
   2026.6.0". Research establishes only that 2026.6.0 is *reported* by one upstream issue. Should the PRD text
   be softened to the report-only wording, and is it acceptable that the tool's own note then says a range is
   unverified? The proposal assumes yes to both, to satisfy §14.5.
4. **Is `doctor --json` a stable product contract now?** The proposal keeps `schema_version` but no schema
   file, deferring `contracts/diagnosis/v1` until there is a second consumer. Is that the right product
   boundary, or should R1a already freeze the schema because users will script it?
5. **What does a user see between R1a and R1b?** R1a is headless (no role screen), so `R-HR-01`, `NF-01` and
   `NF-07` are satisfied only when R1b lands. Is a headless CLI-only intermediate state acceptable, or should
   R1a and R1b be planned to land together as one user-visible milestone?

**Assumptions needing review or correction**

- R1a's deliverable includes the "writes nothing" *proof*, not just the intent — and its coverage boundary is
  documented rather than widened with a syscall tracer (RG-14).
- Default probe concurrency is a small bounded number chosen as a judgment call, with no PRD requirement
  behind it (D9).
- The chained split is decided in the tasks phase; no exception is assumed and no chain strategy is chosen
  here.
- R1a carries pin knowledge but no verification of an installed `cloudflared`, and says so in its output.

---

## Delivery and review workload

| Aspect | Position |
|---|---|
| Review budget | 400 changed lines |
| Expectation | R1a **will exceed it** — ten probes, the rule table, four feasibility adapters, the JSON DTO and their TDD tests |
| Response | `ask-on-risk`: the tasks phase plans reviewable work units and, when the estimate confirms the overrun, pauses to ask for the chained split |
| Not done here | No `size:exception` inferred, no chain strategy chosen, no `contracts/` schema frozen |
| First work unit | The Go module bootstrap — `go.mod` at `github.com/Luisalt20/herdr-reach` plus `go.sum` and one package — because it is exactly what makes `go test ./...` runnable for the first time |
| Test policy | `strict_tdd: true`; every task carries its test first; `go test ./...`, `go vet ./...`, `gofmt -l .` are the only configured gates (no linter exists and none is invented) |
| Never in tests | Real DNS, TCP, TLS or UDP egress; `sshd -T`; the macOS keychain; `herdr`/`cloudflared` (not invoked at all in R1a) |

---

## Next step

Run the **spec** phase for `reach-diagnosis-core` at R1a scope: turn the coverage table and success criteria
into requirements with acceptance criteria, citing PRD `R-HR-*` and `R-HR-NF-*` ids rather than renumbering,
and record the deferred ids with their owner slices so nothing looks forgotten. Then design (D1–D10), then
tasks — where the module bootstrap is ordered first and the chained-split decision is raised under
`ask-on-risk`.

---

## Question round answers (orchestrator-recorded, 2026-09-14)

This phase could not interview the user, so the orchestrator put the questions to the user and records the
answers here. They are product decisions and they are binding for spec and design.

| # | Question | Answer | Effect on this proposal |
|---|---|---|---|
| 1 | Day-one hub address input | A `--hub host[:port]` flag; when it is absent, `egress.hub.direct` is reported as **not measured**, never as blocked | Confirms the assumption in RG-13. Design owns the flag syntax and the "not measured" representation (D3). |
| 2 | Semantics of "no viable transport" | A completed measurement with a negative answer is a **success**: exit `0`, `viable: false` | Confirms the exit-code contract in "Report and honesty" and D5. |
| 3 | Product-facing wording for the `cloudflared` pin | The specification's §8.3 sentence is corrected to the report-only wording in a **separate work unit**, and the tool's own note uses that same wording | RG-1's mitigation becomes an explicit deliverable; the specification edit is its own reviewable work unit rather than an implicit edit inside code. |
| 4 | `doctor --json` as a product contract | **Lightweight**: `schema_version` in the payload, no `contracts/diagnosis/v1` schema file in R1a; the schema is frozen only when a second consumer appears | Confirms NF-06's deferred status and the D5/D6 boundary. |
| 5 | What a user sees between R1a and R1b | R1a ships as a headless CLI (`herdr-reach doctor`); R1b follows as its own change, and `R-HR-01`, `NF-01`, `NF-07` stay marked **Deferred → R1b** | Confirms the R1a/R1b split and the coverage table's honesty. |

No other assumption in this proposal changes because of these answers.