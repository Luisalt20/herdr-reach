# Contributing to herdr-reach

This repository is public, and the rules below are the same ones the automation
enforces. Nothing here is a suggestion: every item is a required status check or a
branch-ruleset rule, and a pull request that ignores one of them will not merge.

## The flow

1. **Open an issue first.** Say what is wrong or what you want to add, with the
   evidence you already have. A maintainer adds the `status:approved` label once the
   scope is agreed — that label is the difference between "someone is working on it"
   and "this change was agreed to".
2. **Do not start before the issue is approved.** Work that lands without it is
   rejected by CI, not by a person's mood.
3. **Branch as `type/description`** — `feat/cloudflare-runbook`,
   `fix/sshd-divergence-report`. Allowed types: `feat`, `fix`, `chore`, `docs`,
   `style`, `refactor`, `perf`, `test`, `build`, `ci`, `revert`.
4. **Commit in Conventional Commits.** `feat(probe): add the QUIC probe`,
   `docs: state the pin as report-only`. Allowed types as above, optional lowercase
   scope in parentheses, optional `!` for a breaking change:
   `feat(contract)!: drop the third viability state`.
5. **Open the pull request against `main`** and put `Closes #<issue>` in the body.
6. **Label it with exactly one `type:*` label.** A change that is two things at once
   is usually two changes.

## The gates

| Check | What it enforces | Where |
|---|---|---|
| `Go Format` | `gofmt -l .` prints nothing | `.github/workflows/ci.yml` |
| `Unit Tests` | `go vet ./...` and `go test -count=1 ./...` | `.github/workflows/ci.yml` |
| `Race Detector` | `go test -race -count=1 ./...` | `.github/workflows/ci.yml` |
| `Platform Tests (macos-latest\|windows-latest)` | the suite on the other supported hosts | `.github/workflows/ci.yml` |
| `Cross-Build` | the release targets compile (`linux`/`darwin` × `amd64`/`arm64`, `windows/amd64`) | `.github/workflows/ci.yml` |
| `No-Real-Network Guard` | the deny-all seam set still denies, with a zero-match guard | `.github/workflows/ci.yml` |
| `Conventional Commits` | every commit subject follows the convention | `.github/workflows/pr-check.yml` |
| `Branch Name` | the head branch is `type/description` | `.github/workflows/pr-check.yml` |
| `PR Size` | at most 400 changed lines, or the `size:exception` label | `.github/workflows/pr-check.yml` |
| `Type Label` | exactly one `type:*` label | `.github/workflows/pr-check.yml` |
| `Linked Issue` | the body carries `Closes #N` | `.github/workflows/pr-check.yml` |
| `Issue Approved` | every linked issue is open with `status:approved` | `.github/workflows/pr-check.yml` |

Two of those checks exist as CI jobs rather than as branch-ruleset rules for a plain
reason: **GitHub refuses `commit_message_pattern` and `branch_name_pattern` on a
user-owned repository** (HTTP 422 — they are organization-only rules). The convention
still needs a home, so it lives in `pr-check.yml` and is a required check like the
rest.

## The review budget

**400 changed lines**, and the number is not arbitrary: it is the point where one
reviewer can still hold the change in their head and hold an opinion about it. A pull
request over the budget will be asked to split, and the repository's own history is the
evidence for why — the largest slice landed here so far was 1,868 authored lines in one
commit, and reviewing it honestly took longer than writing it.

Above the budget there are exactly two honest outcomes:

- **split it** — preferred, and the plan artifacts already name the split boundary for
  most slices of the current change;
- **`size:exception`** — a maintainer-applied label that downgrades the size check to a
  warning. It is a decision with a name attached, never a silent pass.

Never shrink a diff by deleting tests, control cases, comments or blank lines to fit the
budget. The budget protects reviewers; it is not a reason to weaken evidence.

## What the tests are expected to prove

The project's governing rule, from the specification, outranks every other decision:

> When the tool cannot establish something, it says so, and it says why.

That is why contributions are expected to arrive with evidence rather than assertion:

- **a probe that cannot fail is not a probe** — every new probe carries a control case
  proving it can fail and one proving it can be `indeterminate`;
- **an absence is never upgraded** — `indeterminate` and `not measured` must never be
  presented, encoded or rendered as success;
- **no real egress in tests** — the deny-all seam set is the default, and a test asserts
  the dialed target set equals the declared one;
- **a `go test -run` filter that matches nothing exits 0** — so focused selectors are
  checked for a real selection (`=== RUN` count), and the CI guard fails on an empty
  match.

## Commit shape

One work unit per commit, tests and docs alongside the behaviour they describe. Every
commit message states what changed and why in the body when the reason is not obvious
from the subject. A commit that discloses a deviation from the design, or a budget
overrun, is doing its job: the disclosures are the review's raw material, not noise.

## Release and versioning

Releases are automated for tags: `.github/workflows/release.yml` publishes a pre-release on a
`v<major>.<minor>.<patch>` tag, but **choosing the tag and the version stays a manual decision**, and
so does the plugin's declaration of it. Cutting a release therefore means setting
`plugin/herdr-plugin.toml`'s `version` and `plugin/fetch.sh`'s `TOOL_TAG` to the tag before the tag
exists: the plugin declares the version of the tool it delivers, a script-only change needs no bump,
and a release does. Two guards make forgetting impossible rather than merely discouraged — a Go test
that the two declarations agree, and a release preflight that refuses a tag the plugin does not
declare.

`go.mod` declares `go 1.25.10` deliberately, so a contributor on that toolchain is not silently
forced to download a newer one; do not raise the directive without an explicit decision recorded in
the change's artifacts.

The Herdr plugin declares the version of the tool it delivers: a release sets
`plugin/herdr-plugin.toml`'s `version` and `plugin/fetch.sh`'s `TOOL_TAG` to the tag
being cut, because the plugin's scripts come from `main` while `fetch.sh` downloads
the binary the manifest declares. A script-only change needs no bump and keeps
delivering the last released binary; a release does. Forgetting is impossible rather
than merely discouraged: the Go suite fails when the manifest and `TOOL_TAG`
disagree, and the release preflight fails when the manifest does not declare the
version being tagged.
