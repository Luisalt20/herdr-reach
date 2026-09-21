# herdr-reach for Herdr

Measure what this machine's network allows and read the answer inside Herdr.

This plugin runs [`herdr-reach doctor`](https://github.com/Luisalt20/herdr-reach) on the machine
where Herdr is running and shows its human report — in a pane, or through an action that lands in
the plugin log.

## What it does not do

The doctor measures and recommends. **It installs nothing, configures nothing, provisions no
transport, opens no port, and connects no machines.** A `viable` row in the report means the run
measured every requirement of that path *from where it ran*; it does not mean a transport exists
or was set up. The plugin is a view onto that measurement, so it inherits the same discipline:
it runs the tool and shows what the tool found.

If the report advises you to pin a client version or verify a checksum before you install
something yourself, that advice is addressed to **you**. Nothing in this plugin acts on it.

## Install

```sh
herdr plugin install Luisalt20/herdr-reach/plugin
```

The install runs one build step, `fetch.sh`, which downloads the `herdr-reach` binary for your
platform from a release of this repository and verifies it against that release's own
`SHA256SUMS`. If the download fails, the asset is not listed there, or the digest does not match,
**the install fails and no binary is installed**. This matters because Herdr does not review or
sandbox plugin code: the checksum is what makes the binary this plugin runs the one the release
published.

The plugin declares `platforms = ["linux", "macos"]`: it runs on the machine whose Herdr holds
the UI, which is the machine each action measures. There is no Windows plugin: the published
beta binaries have been run on real Windows machines for measurement (issues #59 and #72, and
the tester report behind issues #78 and #79), but no plugin is offered there, and WSL2 remains
the Windows path this tool would provision.

## Configure

The plugin reads one `KEY=value` file from its config directory. Print the directory with:

```sh
herdr plugin config-dir herdr-reach.diagnose
```

Then create `herdr-reach.conf` in it:

```sh
HERDR_REACH_HUB=hub.example.com:22
HERDR_REACH_NODE=node.example.com
```

Both keys are optional; set the side(s) you know. The value is always **the other side's**
address, never this machine's:

- on the node, `HERDR_REACH_HUB` is the hub's address;
- on the hub, `HERDR_REACH_NODE` is the node's address.

One machine can carry both when it plays both roles. The doctor itself has no role switch:
`--hub` always names the address to measure from this machine toward the other side, and the role
you invoke decides which configured address that is. The last assignment wins if a key appears
more than once.

## Use

Once installed, the plugin registers three entrypoints:

| Entrypoint | What it runs |
|:---|:---|
| Action `herdr-reach: diagnose this machine as the node` | `doctor --hub <HERDR_REACH_HUB>` |
| Action `herdr-reach: diagnose this machine as the hub` | `doctor --hub <HERDR_REACH_NODE>` |
| Pane `herdr-reach report` | `doctor --hub` with whichever address is configured (the hub's preferred) |

Invoke an action with `herdr plugin action invoke herdr-reach.diagnose.diagnose-node` (or
`...diagnose-hub`), and open the pane with
`herdr plugin pane open --plugin herdr-reach.diagnose --entrypoint report`. The pane holds its
output until you press enter, so the report stays readable before the split closes.

`doctor` is read-only: it writes nothing to the machine, executes no third-party binary, and
dials only the targets it declares. Its report goes to standard error, and the machine-readable
document to standard output; the plugin shows the report.

With no address configured the run still happens, without `--hub`: the report names the missing
measurement as `not measured` instead of guessing, and the missing address alone does not change
the exit code. Other probes can still leave the run incomplete — the exit section below says
which.

## Exit codes

`doctor` reports through its exit code, and the plugin passes that code through unchanged:

| Code | Meaning |
|:---|:---|
| `0` | Measurement completed — including "no transport viable"; that is a result, not a failure of the run. |
| `1` | Run incomplete — at least one probe was attempted and produced no answer; `run.completeness` is the field that says so. |
| `2` | Usage or internal error — no diagnosis is presented as completed, and standard output stays empty. |

Herdr derives a plugin action's status from the process exit code, so an action lands in the
plugin log as `failed` when `doctor` exits `1`. **That means "incomplete measurement", not "the
plugin broke".** Exit `1` is the common outcome on a real network, not an exception:
`tls.interception` and `egress.quic` are attempted and produce no answer, a macOS run adds
`tls.truststore`, and a verified TLS chain whose issuing organization is outside the declared
expected set is one of the causes. The report is complete for every question it answered; the
`unresolved` line names the question that was attempted and could not be settled, and this is a
result, not a failure of the run. The field to read is `run.completeness`, never the action's
status. Both entrypoints print this explanation on standard error after the report and still exit
`1`; the plugin never rewrites the tool's status. A status outside `0`, `1`, and `2` is outside
the documented set, and the plugin names it as such before passing it through.

## Known limit

The doctor does not measure the node's Herdr version. A node running Herdr older than 0.9.1 is
classified as supported here — `refused` is always `false` — while it also cannot host a
saved-machine connection; telling those two cases apart needs a version measurement this tool does
not make, and the report carries the caveat in the node classification's own note.

## More

[Testing herdr-reach](../docs/beta-testing.md) is the full picture of what the shipped binary
measures, how to read its report, and what to send back when a run surprises you.
