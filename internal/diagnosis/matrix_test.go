package diagnosis_test

// This file is PRD §1.1's six-row evidence matrix, replayed through the reasoning layer as a named
// acceptance test. The change's own acceptance criterion is this replay: PRD §13 measures diagnosis
// accuracy by comparing the tool's verdict with the manual matrix, and R-HR-03 makes the
// port-versus-protocol disambiguation a named finding with a stable rule id whose reason names the
// blocked target.
//
// The matrix is replayed as scripted results rather than by driving dialers, because the reasoning
// layer's input is a run's results and nothing else (design §4: `diagnose.Diagnose(results)`), and
// because a replay that needed a network would not be a test. Each row of PRD §1.1 is one probe's
// observation: the two public-SSH rows, the hub-directed row, the two Cloudflare edge rows and the
// TLS row. The placeholder `203.0.113.10` is the documentation address the specification itself
// uses for the hub; no real host is ever dialed or named as the hub here.

import (
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/diagnosis"
	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// prdSection11Row is one literal row of PRD §1.1's measured matrix, as the observation the probe
// that measured it reported.
//
// The six rows are kept whole rather than folded into one fixture, because the point of the replay
// is that a reader can compare the table in the specification with the table in the test, row by
// row, and see that the conclusion comes from the comparison of the two public-SSH rows with the
// hub-directed row and from nothing else.
type prdSection11Row struct {
	// probe is the probe of PRD §5.1 that measured the row.
	probe string
	// kind is that probe's question family.
	kind probe.ProbeKind
	// label is the observation's own label inside the probe ("tcp 22 ssh banner").
	label string
	// target is the address the row measured.
	target string
	// resolution and verdict are what the row's measurement produced.
	resolution probe.Resolution
	verdict    probe.Verdict
	// reason is the stable machine reason code of that outcome (R-HR-07).
	reason probe.ReasonCode
	// detail is the verbatim detail the probe reported.
	detail string
}

// prdSection11Rows is PRD §1.1's six rows in the specification's own order.
//
// One reading is stated here because it is a deliberate one and a reviewer should see it rather
// than infer it. The hub row of the specification reads "Your VPS | 22, 2222, 443 | BLOCKED", which
// is a human matrix over three hand-made attempts; a run declares one hub address (`--hub
// host[:port]`, design D3) and the hub probe reports one observation for it, so the run replayed
// here declares `203.0.113.10:22` and the row is reproduced at that port. The other two ports are
// the same row measured by hand and change nothing about the conclusion: the reason names the
// declared address with its port, and a run declaring `:2222` would name that port instead.
var prdSection11Rows = []prdSection11Row{
	{
		// "github.com | 22 | OK | SSH is not blocked as a protocol."
		probe:      "egress.ssh.known",
		kind:       probe.ProbeEgress,
		label:      "tcp 22 ssh banner",
		target:     "github.com:22",
		resolution: probe.Measured,
		verdict:    probe.Pass,
		reason:     probe.ReasonOK,
		detail:     `tcp github.com:22: read 40 bytes ("SSH-2.0-..."), the first line is an SSH identification string, so this port speaks SSH`,
	},
	{
		// "ssh.github.com | 443 | OK | SSH-over-443 works when the port is allowed."
		probe:      "egress.ssh.443",
		kind:       probe.ProbeEgress,
		label:      "tcp 443 ssh banner",
		target:     "ssh.github.com:443",
		resolution: probe.Measured,
		verdict:    probe.Pass,
		reason:     probe.ReasonOK,
		detail:     `tcp ssh.github.com:443: read 40 bytes ("SSH-2.0-..."), the first line is an SSH identification string, so this port speaks SSH`,
	},
	{
		// "Your VPS | 22, 2222, 443 | BLOCKED | The block is on the destination, not the protocol."
		// The declared hub address of this run, failing on the probe's own dial budget: for the
		// question "is this port reachable?" the expiry is the measurement (design §5.1).
		probe:      "egress.hub.direct",
		kind:       probe.ProbeEgress,
		label:      "tcp 22",
		target:     "203.0.113.10:22",
		resolution: probe.Measured,
		verdict:    probe.Fail,
		reason:     probe.ReasonBudgetExpired,
		detail:     "dial tcp 203.0.113.10:22: i/o timeout after the probe's own 4s dial budget, which is shorter than the runner's per-probe bound: the declared port did not answer inside the probe's own budget",
	},
	{
		// "region1.v2.argotunnel.com | 7844 | OK | Cloudflare tunnel egress is allowed."
		probe:      "egress.cf.7844",
		kind:       probe.ProbeEgress,
		label:      "tcp 7844 region1",
		target:     "region1.v2.argotunnel.com:7844",
		resolution: probe.Measured,
		verdict:    probe.Pass,
		reason:     probe.ReasonOK,
		detail:     "dial tcp region1.v2.argotunnel.com:7844: the connection was established and closed without reading or writing, which measures TCP reachability of the declared address and nothing else",
	},
	{
		// "region1.v2.argotunnel.com | 443 | OK | Same, over HTTPS."
		probe:      "egress.cf.443",
		kind:       probe.ProbeEgress,
		label:      "tcp 443 region1",
		target:     "region1.v2.argotunnel.com:443",
		resolution: probe.Measured,
		verdict:    probe.Pass,
		reason:     probe.ReasonOK,
		detail:     "dial tcp region1.v2.argotunnel.com:443: the connection was established and closed without reading or writing, which measures TCP reachability of the declared address and nothing else",
	},
	{
		// "www.cloudflare.com | 443 | OK | No TLS interception (issuer Let's Encrypt/ISRG, verify code 0)."
		probe:      "tls.interception",
		kind:       probe.ProbeTLS,
		label:      "tls verify www.cloudflare.com",
		target:     "www.cloudflare.com:443",
		resolution: probe.Measured,
		verdict:    probe.Pass,
		reason:     probe.ReasonOK,
		detail:     `verified www.cloudflare.com:443: issuer "Let's Encrypt/ISRG", verify code 0`,
	},
}

// matrixReplayRun builds the run PRD §1.1's six rows imply: one result per row, in row order.
func matrixReplayRun() []probe.Result {
	results := make([]probe.Result, 0, len(prdSection11Rows))
	for _, row := range prdSection11Rows {
		results = append(results, result(row.probe, row.kind, observation(
			row.label, row.target, row.resolution, row.verdict, row.reason, row.detail,
		)))
	}
	return results
}

// TestMatrixReplayDestinationBlocked is the change's named acceptance test: replaying PRD §1.1's
// six literal rows must produce the destination-block conclusion with the blocked target and its
// port, under the stable rule id SSH_DEST_BLOCKED_BY_PUBLIC_SSH (R-HR-03, PRD §13).
//
// The assertions are the specification's own THEN clauses: the verdict states that outbound SSH
// works and that the destination is blocked, the reason names `203.0.113.10` with its port, and the
// rule id that produced the conclusion is present. The replay also checks the negative half of the
// disambiguation — the conclusion rests on the two public-SSH and hub-directed measurements, and
// nothing in it is derived from the Cloudflare-edge or TLS rows, which passed in this run and must
// stay independent of the conclusion.
func TestMatrixReplayDestinationBlocked(t *testing.T) {
	got := diagnosis.Diagnose(matrixReplayRun())

	finding := findingFor(t, got, "ssh.destination")
	if finding.Rule != "SSH_DEST_BLOCKED_BY_PUBLIC_SSH" {
		t.Errorf("the six-row replay fired %q, want %q", finding.Rule, "SSH_DEST_BLOCKED_BY_PUBLIC_SSH")
	}

	// The conclusion states both halves of the disambiguation and names the blocked destination
	// with the port it was measured on.
	for _, want := range []string{
		"outbound SSH is allowed; the hub address",
		"203.0.113.10:22 is blocked",
	} {
		if !strings.Contains(finding.Conclusion, want) {
			t.Errorf("the destination-block conclusion %q does not carry %q", finding.Conclusion, want)
		}
	}

	// The hub's own fact finding carries the blocked target and its port verbatim, so a reader who
	// looks at the measurement rather than at the interpretation sees the same address.
	hub := findingFor(t, got, "hub.reachability")
	for _, want := range []string{"203.0.113.10:22", "measured and failed"} {
		if !strings.Contains(hub.Conclusion, want) {
			t.Errorf("the hub conclusion %q does not carry %q", hub.Conclusion, want)
		}
	}

	// The conclusion rests on exactly the two measurements PRD §1.1 compares, in the order the rule
	// declares them: the passing public SSH measurement and the failing hub-directed one.
	wantDepends := []string{"egress.ssh.known", "egress.hub.direct"}
	if len(finding.DependsOn) != len(wantDepends) {
		t.Fatalf("the conclusion depends on %v, want %v", finding.DependsOn, wantDepends)
	}
	for i, want := range wantDepends {
		if finding.DependsOn[i] != want {
			t.Errorf("the conclusion depends on %v, want %v", finding.DependsOn, wantDepends)
			break
		}
	}

	// The rule fired because it matched the passing public measurement: the conclusion quotes what
	// was read from the public target, which is the evidence the disambiguation turns on.
	if !strings.Contains(finding.Conclusion, "github.com:22") {
		t.Errorf("the conclusion %q does not name the public target that established SSH egress", finding.Conclusion)
	}
}
