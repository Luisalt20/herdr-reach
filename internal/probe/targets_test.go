package probe_test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/probe"
)

// declaredProbeNames is the probe set of the diagnosis spec (PRD §5.1), written
// out here so the declaration is checked against the specification rather than
// against itself. Order is part of the assertion: the registry that lands later
// enumerates probes in this order.
var declaredProbeNames = []string{
	"local.env",
	"local.sshd",
	"egress.hub.direct",
	"egress.ssh.known",
	"egress.ssh.443",
	"egress.cf.7844",
	"egress.cf.443",
	"egress.quic",
	"tls.interception",
	"tls.truststore",
}

// TestDeclarationHoldsTheTenProbes is the first property of the declared target
// set: it is the specification's ten probes, each with the protocol it measures
// over and the port an override without one resolves to. A probe that measures
// this machine declares no target at all, because nothing about it may be dialed.
func TestDeclarationHoldsTheTenProbes(t *testing.T) {
	declarations := probe.DeclaredTargets()
	if len(declarations) != len(declaredProbeNames) {
		t.Fatalf("declaration holds %d probes, want the %d of PRD §5.1", len(declarations), len(declaredProbeNames))
	}
	for i, want := range declaredProbeNames {
		if got := declarations[i].Probe; got != want {
			t.Fatalf("declaration[%d] = %q, want %q in specification order", i, got, want)
		}
	}
	for _, declaration := range declarations {
		switch declaration.Protocol {
		case probe.ProtocolLocal:
			if len(declaration.Targets) != 0 {
				t.Errorf("%s measures this machine but declares %d dialable targets", declaration.Probe, len(declaration.Targets))
			}
			if declaration.DefaultPort != 0 {
				t.Errorf("%s declares the local protocol but a default port of %d", declaration.Probe, declaration.DefaultPort)
			}
		case probe.ProtocolTCP, probe.ProtocolUDP, probe.ProtocolTLS:
			if declaration.DefaultPort < 1 {
				t.Errorf("%s declares %s without a default port", declaration.Probe, declaration.Protocol)
			}
		default:
			t.Errorf("%s declares unknown protocol %q", declaration.Probe, declaration.Protocol)
		}
		for _, target := range declaration.Targets {
			if target.Host == "" {
				t.Errorf("%s declares a target with no host: %+v", declaration.Probe, target)
			}
			if target.Port < 1 {
				t.Errorf("%s declares %s with no port", declaration.Probe, target.Host)
			}
		}
	}
}

// TestDeclarationCarriesTheMeasuredTriples checks the declared endpoints against
// PRD §1.1's measured evidence, which is the reason each of them exists. The
// region probes carry both regions (design D10) so a split result is
// representable; the local and hub probes carry no fixed host.
func TestDeclarationCarriesTheMeasuredTriples(t *testing.T) {
	declarations := map[string]probe.ProbeDeclaration{}
	for _, declaration := range probe.DeclaredTargets() {
		declarations[declaration.Probe] = declaration
	}

	want := map[string][]probe.Target{
		"egress.ssh.known": {{Host: "github.com", Port: 22}},
		"egress.ssh.443":   {{Host: "ssh.github.com", Port: 443}},
		"egress.cf.7844": {
			{Host: "region1.v2.argotunnel.com", Port: 7844, Label: "region1"},
			{Host: "region2.v2.argotunnel.com", Port: 7844, Label: "region2"},
		},
		"egress.cf.443": {
			{Host: "region1.v2.argotunnel.com", Port: 443, Label: "region1"},
			{Host: "region2.v2.argotunnel.com", Port: 443, Label: "region2"},
		},
		"egress.quic": {
			{Host: "region1.v2.argotunnel.com", Port: 7844, Label: "region1"},
			{Host: "region2.v2.argotunnel.com", Port: 7844, Label: "region2"},
		},
		"tls.interception": {{Host: "www.cloudflare.com", Port: 443}},
		"tls.truststore":   {{Host: "www.cloudflare.com", Port: 443}},
	}
	for name, wantTargets := range want {
		declaration, ok := declarations[name]
		if !ok {
			t.Fatalf("%s is missing from the declaration", name)
		}
		if len(declaration.Targets) != len(wantTargets) {
			t.Fatalf("%s declares %d targets, want %d: %+v", name, len(declaration.Targets), len(wantTargets), declaration.Targets)
		}
		for i, target := range wantTargets {
			if got := declaration.Targets[i]; got != target {
				t.Fatalf("%s target[%d] = %+v, want %+v", name, i, got, target)
			}
		}
	}

	for _, name := range []string{"local.env", "local.sshd", "egress.hub.direct"} {
		declaration, ok := declarations[name]
		if !ok {
			t.Fatalf("%s is missing from the declaration", name)
		}
		if len(declaration.Targets) != 0 {
			t.Fatalf("%s declares a fixed host %+v, but its address is not a constant", name, declaration.Targets)
		}
	}
}

// TestDeclaredHostsHaveOneHome asserts R-HR-NF-10's "declared in one place"
// literally: each declared host exists as a string literal in targets.go and in
// no other non-test source file of the package, so a probe that hardcoded a
// target of its own would fail here instead of quietly widening the dialed set.
//
// Coverage boundary: the check looks for the quoted literal, so a host assembled
// at run time from fragments would not be caught; it proves the documented
// pattern, not every aliasing trick.
func TestDeclaredHostsHaveOneHome(t *testing.T) {
	hosts := map[string]bool{}
	for _, declaration := range probe.DeclaredTargets() {
		for _, target := range declaration.Targets {
			hosts[target.Host] = true
		}
	}
	if len(hosts) == 0 {
		t.Fatal("the declaration holds no hosts, so this guard would pass vacuously")
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read the package directory: %v", err)
	}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		scanned++
		for host := range hosts {
			if !strings.Contains(string(source), fmt.Sprintf("%q", host)) {
				continue
			}
			if name != "targets.go" {
				t.Errorf("%s declares the target host %q; the declared set lives only in targets.go", name, host)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("the guard scanned no package sources, so it proves nothing")
	}
}

// TestOverrideReplacesRatherThanAppends is the closed-set rule of design D10: an
// override replaces the probe's declared targets, so the effective set stays
// exactly computable and no declared target survives beside the override.
func TestOverrideReplacesRatherThanAppends(t *testing.T) {
	override, err := probe.ParseTargetOverride("egress.cf.7844=203.0.113.7:7844")
	if err != nil {
		t.Fatalf("ParseTargetOverride: %v", err)
	}
	effective, err := probe.EffectiveTargets(probe.TargetInput{Overrides: []probe.TargetOverride{override}})
	if err != nil {
		t.Fatalf("EffectiveTargets: %v", err)
	}

	for _, target := range effective {
		if target.Probe != "egress.cf.7844" {
			continue
		}
		if got := target.Address(); got != "203.0.113.7:7844" {
			t.Fatalf("overridden target = %q, want the override address", got)
		}
		if target.Label != "" {
			t.Fatalf("overridden target kept the declared label %q", target.Label)
		}
	}
	if got := countProbe(effective, "egress.cf.7844"); got != 1 {
		t.Fatalf("egress.cf.7844 has %d effective targets after the override, want exactly the override", got)
	}
	for _, declaration := range probe.DeclaredTargets() {
		if declaration.Probe == "egress.cf.7844" || declaration.Protocol == probe.ProtocolLocal {
			continue
		}
		if got, want := countProbe(effective, declaration.Probe), len(declaration.Targets); got != want {
			t.Fatalf("%s has %d effective targets, want its declared %d left untouched", declaration.Probe, got, want)
		}
	}

	// A second override for the same probe replaces the first rather than adding
	// beside it: the flag is repeatable, and "replace" must stay true for it too.
	second, err := probe.ParseTargetOverride("egress.cf.7844=198.51.100.9:7844")
	if err != nil {
		t.Fatalf("ParseTargetOverride (second): %v", err)
	}
	effective, err = probe.EffectiveTargets(probe.TargetInput{Overrides: []probe.TargetOverride{override, second}})
	if err != nil {
		t.Fatalf("EffectiveTargets with two overrides: %v", err)
	}
	if got := countProbe(effective, "egress.cf.7844"); got != 1 {
		t.Fatalf("repeating the override produced %d targets, want 1", got)
	}
	for _, target := range effective {
		if target.Probe == "egress.cf.7844" && target.Address() != "198.51.100.9:7844" {
			t.Fatalf("repeated override resolved to %q, want the last one", target.Address())
		}
	}
}

// TestOverrideWithoutAPortUsesTheDeclaredDefaultPort keeps the port rule in one
// place: an override that names only a host resolves to the probe's declared
// default port, which is the same rule the hub address follows.
func TestOverrideWithoutAPortUsesTheDeclaredDefaultPort(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{"egress.ssh.known=example.com", "example.com:22"},
		{"egress.ssh.443=example.com", "example.com:443"},
		{"egress.cf.7844=example.com", "example.com:7844"},
		{"egress.quic=example.com", "example.com:7844"},
		{"tls.interception=example.com", "example.com:443"},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			override, err := probe.ParseTargetOverride(tc.value)
			if err != nil {
				t.Fatalf("ParseTargetOverride(%q): %v", tc.value, err)
			}
			effective, err := probe.EffectiveTargets(probe.TargetInput{Overrides: []probe.TargetOverride{override}})
			if err != nil {
				t.Fatalf("EffectiveTargets: %v", err)
			}
			for _, target := range effective {
				if target.Probe != override.Probe {
					continue
				}
				if got := target.Address(); got != tc.want {
					t.Fatalf("effective target = %q, want %q", got, tc.want)
				}
				return
			}
			t.Fatalf("%s produced no effective target", override.Probe)
		})
	}
}

// TestOverrideErrorsAreTyped pins the three usage errors design D10 names — an
// unknown probe name, an empty host and an unparsable address — plus the guard
// that a probe measuring this machine has no target to replace. Each is asserted
// with errors.Is, so the flag layer can turn any of them into exit code 2.
func TestOverrideErrorsAreTyped(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  error
	}{
		{"unknown probe name", "nosuch.probe=example.com:22", probe.ErrUnknownProbe},
		{"no equals sign", "egress.ssh.known", probe.ErrTargetOverrideSyntax},
		{"empty probe name", "=example.com:22", probe.ErrUnknownProbe},
		{"empty host", "egress.ssh.known=:22", probe.ErrTargetAddress},
		{"empty address", "egress.ssh.known=", probe.ErrTargetAddress},
		{"empty port", "egress.ssh.known=example.com:", probe.ErrTargetAddress},
		{"port is not a number", "egress.ssh.known=example.com:ssh", probe.ErrTargetAddress},
		{"port is out of range", "egress.ssh.known=example.com:70000", probe.ErrTargetAddress},
		{"address with colon-bearing host but no port", "egress.ssh.known=not:an:address", probe.ErrTargetAddress},
		{"local probe has no target to replace", "local.env=example.com:22", probe.ErrLocalProbeOverride},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := probe.ParseTargetOverride(tc.value)
			if !errors.Is(err, tc.want) {
				t.Fatalf("ParseTargetOverride(%q) error = %v, want %v", tc.value, err, tc.want)
			}
		})
	}

	t.Run("an override built by hand is rejected by EffectiveTargets too", func(t *testing.T) {
		_, err := probe.EffectiveTargets(probe.TargetInput{
			Overrides: []probe.TargetOverride{{Probe: "nosuch.probe", Host: "example.com", Port: 22}},
		})
		if !errors.Is(err, probe.ErrUnknownProbe) {
			t.Fatalf("EffectiveTargets error = %v, want %v", err, probe.ErrUnknownProbe)
		}
		_, err = probe.EffectiveTargets(probe.TargetInput{
			Overrides: []probe.TargetOverride{{Probe: "local.env", Host: "example.com", Port: 22}},
		})
		if !errors.Is(err, probe.ErrLocalProbeOverride) {
			t.Fatalf("EffectiveTargets error = %v, want %v", err, probe.ErrLocalProbeOverride)
		}
		for _, broken := range []probe.TargetOverride{
			{Probe: "egress.ssh.known", Host: ""},
			{Probe: "egress.ssh.known", Host: "example.com", Port: -1},
			{Probe: "egress.ssh.known", Host: "example.com", Port: 70000},
		} {
			if _, err := probe.EffectiveTargets(probe.TargetInput{Overrides: []probe.TargetOverride{broken}}); !errors.Is(err, probe.ErrTargetAddress) {
				t.Fatalf("EffectiveTargets(%+v) error = %v, want %v", broken, err, probe.ErrTargetAddress)
			}
		}
	})
}

// TestHubAddressBecomesTheDeclaredTarget covers design D3's rule at the target
// layer: the supplied hub address is the hub probe's target exactly, the
// documented default port applies when it is omitted, and an absent hub leaves
// the probe with no target rather than a blocked one.
func TestHubAddressBecomesTheDeclaredTarget(t *testing.T) {
	cases := []struct {
		name string
		hub  string
		want string
	}{
		{"supplied with a port", "203.0.113.10:2222", "203.0.113.10:2222"},
		{"supplied without a port", "203.0.113.10", "203.0.113.10:22"},
		{"bracketed IPv6 with a port", "[2001:db8::1]:2222", "[2001:db8::1]:2222"},
		{"bracketed IPv6 without a port", "[2001:db8::1]", "[2001:db8::1]:22"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			effective, err := probe.EffectiveTargets(probe.TargetInput{Hub: tc.hub})
			if err != nil {
				t.Fatalf("EffectiveTargets(hub %q): %v", tc.hub, err)
			}
			targets := targetsFor(effective, "egress.hub.direct")
			if len(targets) != 1 {
				t.Fatalf("hub probe has %d effective targets, want exactly the supplied hub", len(targets))
			}
			if got := targets[0].Address(); got != tc.want {
				t.Fatalf("hub target = %q, want %q", got, tc.want)
			}
			if got := targets[0].Protocol; got != probe.ProtocolTCP {
				t.Fatalf("hub target protocol = %q, want %q", got, probe.ProtocolTCP)
			}
		})
	}

	t.Run("no hub supplied leaves no hub target", func(t *testing.T) {
		effective, err := probe.EffectiveTargets(probe.TargetInput{})
		if err != nil {
			t.Fatalf("EffectiveTargets: %v", err)
		}
		if got := targetsFor(effective, "egress.hub.direct"); len(got) != 0 {
			t.Fatalf("hub probe declared %+v with no hub supplied, want nothing dialable", got)
		}
	})

	t.Run("an explicit override wins over the hub address", func(t *testing.T) {
		override, err := probe.ParseTargetOverride("egress.hub.direct=198.51.100.4:2222")
		if err != nil {
			t.Fatalf("ParseTargetOverride: %v", err)
		}
		effective, err := probe.EffectiveTargets(probe.TargetInput{
			Hub:       "203.0.113.10:22",
			Overrides: []probe.TargetOverride{override},
		})
		if err != nil {
			t.Fatalf("EffectiveTargets: %v", err)
		}
		targets := targetsFor(effective, "egress.hub.direct")
		if len(targets) != 1 || targets[0].Address() != "198.51.100.4:2222" {
			t.Fatalf("hub probe targets = %+v, want the explicit override alone", targets)
		}
	})

	t.Run("an unparsable hub address is a usage error", func(t *testing.T) {
		for _, hub := range []string{"203.0.113.10:ssh", ":22", "203.0.113.10:", "not:an:address"} {
			if _, err := probe.EffectiveTargets(probe.TargetInput{Hub: hub}); !errors.Is(err, probe.ErrTargetAddress) {
				t.Fatalf("EffectiveTargets(hub %q) error = %v, want %v", hub, err, probe.ErrTargetAddress)
			}
		}
	})
}

// countProbe counts a probe's effective targets.
func countProbe(effective []probe.EffectiveTarget, name string) int {
	return len(targetsFor(effective, name))
}

// targetsFor selects one probe's effective targets in declaration order.
func targetsFor(effective []probe.EffectiveTarget, name string) []probe.EffectiveTarget {
	var selected []probe.EffectiveTarget
	for _, target := range effective {
		if target.Probe == name {
			selected = append(selected, target)
		}
	}
	return selected
}

// assertEffectiveTargets compares two effective sets field by field, so a
// mismatch names the differing entry instead of printing two slices.
func assertEffectiveTargets(t *testing.T, got, want []probe.EffectiveTarget) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("effective set holds %d targets, want %d\n got: %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("effective[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestEffectiveSetIsClosedAfterAnOverride is R-HR-NF-10's closed-set property:
// the effective set is exactly the declaration after the run input has moved it,
// nothing in it escapes the declaration or an override, and every declared triple
// is reachable from DeclaredTargets() alone — no probe code is consulted, because
// there is no probe code to consult yet.
func TestEffectiveSetIsClosedAfterAnOverride(t *testing.T) {
	declarations := probe.DeclaredTargets()

	t.Run("every declared triple is reachable from the declaration", func(t *testing.T) {
		effective, err := probe.EffectiveTargets(probe.TargetInput{})
		if err != nil {
			t.Fatalf("EffectiveTargets: %v", err)
		}
		reachable := 0
		for _, declaration := range declarations {
			if declaration.Protocol == probe.ProtocolLocal || len(declaration.Targets) == 0 {
				continue
			}
			want := effectiveTargetsOf(declaration)
			assertEffectiveTargets(t, targetsFor(effective, declaration.Probe), want)
			reachable += len(want)
		}
		if want := declaredTargetCount(declarations); reachable != want {
			t.Fatalf("%d declared triples were reachable, want %d", reachable, want)
		}
	})

	t.Run("an override for every probe replaces every declared target", func(t *testing.T) {
		var overrides []probe.TargetOverride
		var want []probe.EffectiveTarget
		for i, declaration := range declarations {
			if declaration.Protocol == probe.ProtocolLocal || len(declaration.Targets) == 0 {
				continue
			}
			address := fmt.Sprintf("198.51.100.%d", i+1)
			override, err := probe.ParseTargetOverride(declaration.Probe + "=" + address)
			if err != nil {
				t.Fatalf("ParseTargetOverride(%s=%s): %v", declaration.Probe, address, err)
			}
			overrides = append(overrides, override)
			want = append(want, probe.EffectiveTarget{
				Probe:    declaration.Probe,
				Protocol: declaration.Protocol,
				Host:     address,
				Port:     declaration.DefaultPort,
			})
		}
		effective, err := probe.EffectiveTargets(probe.TargetInput{Overrides: overrides})
		if err != nil {
			t.Fatalf("EffectiveTargets: %v", err)
		}
		assertEffectiveTargets(t, effective, want)
	})

	t.Run("no effective triple escapes the declaration or an override", func(t *testing.T) {
		const overridden = "egress.cf.7844"
		override, err := probe.ParseTargetOverride(overridden + "=198.51.100.9:7844")
		if err != nil {
			t.Fatalf("ParseTargetOverride: %v", err)
		}
		effective, err := probe.EffectiveTargets(probe.TargetInput{Overrides: []probe.TargetOverride{override}})
		if err != nil {
			t.Fatalf("EffectiveTargets: %v", err)
		}

		allowed := map[string]bool{}
		for _, declaration := range declarations {
			for _, target := range declaration.Targets {
				allowed[tripleKey(declaration.Probe, declaration.Protocol, target.Host, target.Port)] = true
			}
		}
		allowed[tripleKey(override.Probe, probe.ProtocolTCP, override.Host, override.Port)] = true

		for _, entry := range effective {
			key := tripleKey(entry.Probe, entry.Protocol, entry.Host, entry.Port)
			if !allowed[key] {
				t.Fatalf("the effective set dials %s, which neither the declaration nor an override names", key)
			}
		}

		// Replacement, not appending: the probe's two declared endpoints are one,
		// and every other probe's declared endpoints are untouched.
		replaced := 0
		for _, declaration := range declarations {
			if declaration.Probe == overridden {
				replaced = len(declaration.Targets)
			}
		}
		if got, want := len(effective), declaredTargetCount(declarations)-replaced+1; got != want {
			t.Fatalf("effective set holds %d triples, want %d after replacing %d with one", got, want, replaced)
		}
	})
}

// effectiveTargetsOf is the declaration's own view of one probe's endpoints:
// building it without calling EffectiveTargets is what makes the comparison
// between the two an assertion rather than a tautology.
func effectiveTargetsOf(declaration probe.ProbeDeclaration) []probe.EffectiveTarget {
	targets := make([]probe.EffectiveTarget, 0, len(declaration.Targets))
	for _, target := range declaration.Targets {
		targets = append(targets, probe.EffectiveTarget{
			Probe:    declaration.Probe,
			Protocol: declaration.Protocol,
			Host:     target.Host,
			Port:     target.Port,
			Label:    target.Label,
		})
	}
	return targets
}

// declaredTargetCount counts the declared endpoints of the probes that dial
// something, which is the number the effective set holds when run input moves
// nothing.
func declaredTargetCount(declarations []probe.ProbeDeclaration) int {
	count := 0
	for _, declaration := range declarations {
		if declaration.Protocol == probe.ProtocolLocal {
			continue
		}
		count += len(declaration.Targets)
	}
	return count
}

// tripleKey renders one (host, port, protocol) triple together with its probe, so
// a set comparison can name the entry that escaped.
func tripleKey(probeName string, protocol probe.Protocol, host string, port int) string {
	return fmt.Sprintf("%s %s %s:%d", probeName, protocol, host, port)
}

// TestDeclaredTargetsReturnsACopy keeps R-HR-NF-10's single home honest: the
// accessor may be read freely, and editing what it returns must not move the
// declaration the run resolves — otherwise a caller could widen the measured set
// without changing a single declared line.
func TestDeclaredTargetsReturnsACopy(t *testing.T) {
	edited := probe.DeclaredTargets()
	for i := range edited {
		edited[i].Probe = "mutated"
		edited[i].Protocol = probe.ProtocolUDP
		edited[i].DefaultPort = 1
		for j := range edited[i].Targets {
			edited[i].Targets[j] = probe.Target{Host: "mutated.invalid", Port: 1, Label: "mutated"}
		}
	}

	fresh := probe.DeclaredTargets()
	for i, declaration := range fresh {
		if declaration.Probe == "mutated" {
			t.Fatalf("declaration[%d] was edited through the accessor: %+v", i, declaration)
		}
		if i < len(declaredProbeNames) && declaration.Probe != declaredProbeNames[i] {
			t.Fatalf("declaration[%d] = %q, want %q", i, declaration.Probe, declaredProbeNames[i])
		}
		for _, target := range declaration.Targets {
			if target.Host == "mutated.invalid" || target.Label == "mutated" || target.Port == 1 {
				t.Fatalf("%s kept an edited target %+v", declaration.Probe, target)
			}
		}
	}

	effective, err := probe.EffectiveTargets(probe.TargetInput{})
	if err != nil {
		t.Fatalf("EffectiveTargets: %v", err)
	}
	if got, want := len(effective), declaredTargetCount(fresh); got != want {
		t.Fatalf("effective set holds %d targets after the accessor was edited, want %d", got, want)
	}
}
