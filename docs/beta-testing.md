# Beta testing herdr-reach

`herdr-reach doctor` is one read-only measurement run per invocation: it measures the machine it
runs on, reports what it found, and writes nothing to the machine. This guide is for the beta
testers running it on the two machines that matter. It covers what you download, how you verify
it, which run answers which question, how to read the answer, and what to send back.

The beta does not install, configure or provision anything, and it does not connect two machines.
It produces evidence. Everything a report says is a measurement taken where the run happened; a
transport row describes a path this run measured, never a path this run configured.

## What you are downloading

Beta releases are published as **pre-releases** on the repository's
[releases page](https://github.com/Luisalt20/herdr-reach/releases). A beta tag carries a hyphen,
for example `v0.1.0-beta.1`. Each release attaches one binary per platform plus a `SHA256SUMS`
file:

| Platform | Artifact |
|:---|:---|
| Linux, x86-64 | `herdr-reach_<tag>_linux_amd64` |
| Linux, ARM64 | `herdr-reach_<tag>_linux_arm64` |
| macOS, Intel | `herdr-reach_<tag>_darwin_amd64` |
| macOS, Apple silicon | `herdr-reach_<tag>_darwin_arm64` |
| Windows, x86-64 | `herdr-reach_<tag>_windows_amd64.exe` |

`<tag>` is the release you downloaded, for example `v0.1.0-beta.1`. Download your binary and
`SHA256SUMS` from the same release, and verify both the bytes and the build before you run it.

**The checksum proves the bytes** are the ones the release published. `SHA256SUMS` lists all five
platform binaries, so check the line for the one you actually downloaded:

```bash
grep "herdr-reach_<tag>_linux_arm64$" SHA256SUMS | sha256sum -c -        # Linux
grep "herdr-reach_<tag>_darwin_arm64$" SHA256SUMS | shasum -a 256 -c -   # macOS
```

Substitute the artifact name for your platform from the table above. Checking the whole file would
report the four binaries that are not on your machine as unreadable, which says nothing about the
one you have.

On Windows, compare by eye against the matching line of `SHA256SUMS`:

```powershell
Get-FileHash .\herdr-reach_<tag>_windows_amd64.exe -Algorithm SHA256
```

**The version proves the build** is that release and not something else:

```bash
./herdr-reach_<tag>_linux_amd64 --version
```

It must print `herdr-reach <tag>`, for example `herdr-reach v0.1.0-beta.1`. A binary built from
source with no release metadata reports `herdr-reach 0.0.0-dev`, so this check is also what tells
a release build from a development one.

The beta binaries are unsigned and not notarised. macOS may quarantine one downloaded through a
browser and refuse it on first run (`xattr -d com.apple.quarantine <binary>` clears the
attribute), and Windows may show a SmartScreen warning. Neither is a bug in the tool.

## The two profiles

Two machines are involved, and the vocabulary is deliberate:

- the **node** is the machine that has to connect out: the locked-down machine, the work laptop,
  anything on CGNAT. It is the machine that is hard to reach.
- the **hub** is the machine that hosts the orchestrator (Herdr): a VPS, a home server, a Mac
  mini. It is where the node's agents should appear.

`herdr-reach doctor` measures **the machine it runs on**. `--hub <host[:port]>` always names the
address of **the other side** of the link being diagnosed; the default port is 22. That is the
single most important thing to understand before reading the output: the same command answers a
different question depending on which machine runs it.

This beta has no role switch. There is no `--role` flag and no prompt asking which machine this
is: the tool is role-agnostic by design, and the role question belongs to the interface that is
not part of this beta. It measures this machine plus the address you point it at, and choosing
that address is the profile.

### On the node

```bash
herdr-reach doctor --hub <hub address>
```

`<hub address>` is the hub's hostname or address, optionally with a port (`hub.example.com` or
`hub.example.com:22`). This run answers, from the node's network:

- can this machine reach the hub's address on its SSH port, and what happens when it tries;
- is egress blocked at the **destination** or at the **protocol**: the run measures SSH to
  well-known public hosts on ports 22 and 443 beside the hub measurement, so a passing public
  measurement next to a failing hub measurement says the block is on the hub's address rather
  than on SSH itself;
- is the public internet reachable, and is the Cloudflare edge reachable on its tunnel and HTTPS
  ports, including whether the UDP path answers;
- is an `sshd` present on this machine at the documented path — the daemon a transport would
  eventually reach. The configuration *in force* is not measured in this beta: the run is given
  no command runner, so it reports that question as `not measured` instead of reading silence as
  "configured";
- what platform this is and whether the tool supports it. Native Windows as a node is measured and
  refused by this tool: upstream Herdr supports a Windows server as of 0.9.1, but this tool does
  not provision a native Windows node yet, so WSL2 is the Windows path it handles today. The tool
  does not measure the node's Herdr version, so a node running an older server is refused here
  although it also cannot host a saved-machine connection.

### On the hub

```bash
herdr-reach doctor --hub <node address>
```

`<node address>` is whatever address the hub has for the node — the node's hostname or address
and port, as far as the hub can test it. The same questions are asked from the hub's side, and
two of them read differently:

- does the hub reach the node's SSH port: the direct path a hub-to-node connection would use;
- is the hub's own `sshd` present — the endpoint an outbound path from the node can terminate
  on. As on the node, the configuration in force is reported as `not measured` rather than
  assumed.

Read the two reports together. A node that cannot reach the hub is one measurement; a hub that
cannot reach the node is another; and the findings name the measurement each conclusion rests on
rather than summarising across machines.

### The transports section

Both profiles end with a `TRANSPORTS` section. Its four entries are the ways a node can be
reached — `direct-ssh`, `reverse-ssh`, `cloudflare-tunnel` and `tailscale` — each marked `viable`
or `not viable` with the reason the verdict rests on and the requirements that were checked.

The measurement rule applies here too: on a hub run, these rows describe the paths a node would
use, measured from the hub. `viable` means this run measured every requirement of that path
satisfied *from where the tool ran*; it does not mean the transport is configured, and nothing in
this beta configures one.

## Reading the output

One run produces two projections of the same result:

- the **human report**, written to standard error: the one to read;
- the **`--json` document**, written to standard output when you pass `--json`: the one to send.

```bash
herdr-reach doctor --hub <address> --json > report.json
```

With `--json` the document is the only thing on standard output, so the redirection is safe; the
human report still goes to standard error. Without `--json`, standard output stays empty. Use the
report to understand the run and the document to report it.

The human report opens with the tool and its version, the generation time and the node
classification, then prints four sections:

- `PROBES`: one row per measurement, with its target, verdict, resolution, reason code and detail;
- `FINDINGS`: one row per question, the rule id that fired, and the conclusion;
- `COVERAGE`: the completeness of the run and its two coverage lists;
- `TRANSPORTS`: the four rows described above.

### `not measured` and `unresolved` are not failures

`COVERAGE` uses two words that look like problems and are not:

- **`not measured`** means the run never attempted a measurement. Usually there was no input for
  it (no `--hub` was supplied), or the capability is outside this beta's boundary. The tool lists
  the gap instead of implying the measurement passed.
- **`unresolved`** means the run attempted a measurement and the attempt produced no answer: a
  resolver that did not reply, a handshake that did not complete, a platform verifier that cannot
  answer here. It is the absence of a result, not a result of "blocked".

A `fail` verdict is a measured negative — blocked, refused, absent — and it is a completed
answer. `unresolved` is no answer at all. The difference is the tool's whole point: when it
cannot establish something it says so and names why, instead of guessing.

### One `unresolved` to read carefully: an unexpected TLS publisher

`tls.interception` compares the publisher of the chain it observed against a **declared expected
publisher set** for that host. That set was recorded from a single hand-run (`Let's Encrypt/ISRG`),
while a large CDN legitimately serves its fronts from several certificate authorities: a live run on
2026-09-20 observed `Google Trust Services/GlobalSign` for `www.cloudflare.com` on a healthy path.

When the chain verifies but its publisher is outside that declared set, the probe reports
**`unresolved`** — not a `fail`. The run establishes two things (the chain verified, and the
publisher is not the one declared) and **cannot establish a third**: whether the difference is a
publisher change or an interception. Reporting it as a failure would accuse a network this
measurement cannot convict, so the divergence is left open and named as open.

Two consequences to expect from that:

- **The run is incomplete**, so it exits `1`. The question was attempted and could not be settled;
  that is a result, and the `unresolved` line names it.
- It is **not** evidence of interception. The reason code worth reporting on its own is
  `tls_verify_failed` — a chain that does not verify at all — which is a different and more serious
  signal.

The declared set is a declaration, not a record of what has been observed: an entry is added when
there is a stated reason for it, never by copying a publisher that a run happened to see.

### The completeness line

`completeness:` summarises the two:

- `complete`: no probe was attempted and left unresolved. A complete run can still report blocked
  paths — completeness is about whether the attempted questions were answered, not about whether
  the answers are good.
- `incomplete`: at least one probe was attempted and produced no answer. Those probes are named
  on the `unresolved:` line below.

A `not measured` probe does not make a run incomplete, and it still appears on the `not measured:`
line. A run with no `--hub` on Linux is the ordinary case: complete, exit `0`, with the hub
measurement named as a coverage gap.

### Exit codes

| Code | Meaning |
|:---|:---|
| `0` | A measurement completed — including "blocked" answers and a refused native-Windows node. Those are results, not failures of the run. |
| `1` | The run is incomplete: at least one attempted measurement produced no answer. Since 2026-09-20 this includes a verified chain whose publisher is outside the declared expected set — see the section above. |
| `2` | Usage or internal error — a bad flag, an unusable address, a failed write. No diagnosis is presented as completed, and standard output stays empty. |

One platform expectation to know before you file a bug: on macOS the run cannot resolve the local
trust store by decision, so even a default run reports it `unresolved` and exits `1`. On Linux
the same run exits `0`. The report states which case you are in.

## What the tool guarantees

In this beta, a `doctor` run:

- **is read-only**: it writes nothing to the machine — no file, no service, no configuration, no
  receipt; its only output is the report and the document;
- **executes no third-party binary**: no `ssh`, no package manager, no helper process. The one
  command-backed capability, the sshd configuration in force, is reported as `not measured`
  rather than executed;
- **dials only the targets it declares**: every target it will dial is listed in the document
  under `targets.declared`, and the run dials exactly that set;
- **reports reasons instead of guessing**: every measurement carries a reason code from a closed
  set, and something the run cannot establish is reported `unresolved` rather than assumed. An
  indeterminate measurement is never presented as a pass.

## What to send back

When a run surprises you — or fails — send:

1. the `--json` document for the run
   (`herdr-reach doctor --hub <address> --json > report.json`);
2. the version (`<your binary> --version`) and the platform you ran it on, including the
   architecture;
3. the exact command you ran, copied verbatim so the address and flags are not lost;
4. what you expected and what happened. The run reports what it measured; what it cannot know is
   what your network was supposed to do, and that comparison is what finds bugs.

Open an issue on the repository and attach the document. If the tool says `unresolved`, report it
as it stands rather than guessing a cause: the refusal to guess is what makes the report
trustworthy.

This is a **pre-release built to be tested**. Failures are the point: an incomplete run, an
unexpected block, or a reason code that does not match your network is exactly what a beta is
for — not a sign that you are using it wrong.
