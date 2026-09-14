package probe

// This file is the declared target set of design §6.2 and D10: the single place
// where every (host, port, protocol) triple this tool will measure is written
// down, plus the resolution of run input — the hub address and the repeatable
// --target overrides — into the effective set the run will actually dial.
//
// The property the design buys with one declaration is checkable from outside the
// package: EffectiveTargets is a pure function of the declaration and the run
// input, so "the dialed set equals the declared set" (R-HR-NF-10, PRD §13) is an
// assertion about two values rather than a claim about probe code. No probe may
// declare a host of its own, and an override replaces a probe's targets instead
// of appending to them, which is what keeps the set closed and computable.

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Protocol is the wire protocol a probe's declared targets are measured over. It
// is the third element of the (host, port, protocol) triple the payload echoes.
type Protocol string

const (
	// ProtocolTCP is a stream connection. Reachability probes measure over it.
	ProtocolTCP Protocol = "tcp"
	// ProtocolUDP is a datagram exchange. Only egress.quic measures over it.
	ProtocolUDP Protocol = "udp"
	// ProtocolTLS is a TLS handshake, which both tls probes measure over.
	ProtocolTLS Protocol = "tls"
	// ProtocolLocal marks a probe that measures this machine and therefore dials
	// nothing at all. It is not a wire protocol, and a probe that declares it has
	// no target an override could replace.
	ProtocolLocal Protocol = "local"
)

// Target is one declared endpoint: where a probe measures, and which observation
// the measurement belongs to when one probe declares several endpoints.
type Target struct {
	// Host is the declared host, verbatim: a name or an IP literal.
	Host string
	// Port is the declared port, always resolved (the probe's default is applied
	// before a Target reaches the effective set).
	Port int
	// Label distinguishes a probe's observations when it declares more than one
	// endpoint, for example "region1". It is empty when the probe declares one.
	Label string
}

// ProbeDeclaration is one probe's entry in the declared target set: the protocol
// it measures over, the port an override without one resolves to, and the
// endpoints it declares. A probe that measures the local machine declares the
// local protocol and no endpoints; the hub probe declares the stream protocol and
// no host, because its address is run input rather than a constant (design D3).
type ProbeDeclaration struct {
	Probe       string
	Protocol    Protocol
	DefaultPort int
	Targets     []Target
}

// EffectiveTarget is one entry of the effective declared set: the probe that
// declares it and the resolved triple it will dial. This is the value the payload
// echoes and the value the dialed-set assertion compares against, so it carries
// no nesting and no pointer.
type EffectiveTarget struct {
	Probe    string
	Protocol Protocol
	Host     string
	Port     int
	Label    string
}

// Address is the "host:port" form of the target, bracketed correctly for an IPv6
// literal. It is the string every projection echoes, so the resolved target a
// user reads is the same string the assertion compares.
func (t EffectiveTarget) Address() string {
	return net.JoinHostPort(t.Host, strconv.Itoa(t.Port))
}

// TargetOverride is one parsed --target value: the probe whose declaration it
// replaces, and the address it replaces it with. Port is zero when the value
// named only a host, in which case the probe's declared default port applies.
type TargetOverride struct {
	Probe string
	Host  string
	Port  int
}

// TargetInput is the run input EffectiveTargets resolves the effective declared
// set from. Hub is the optional hub address in "host[:port]" form and is empty
// when no hub was supplied; Overrides are the parsed --target values, applied in
// the order the user gave them.
type TargetInput struct {
	Hub       string
	Overrides []TargetOverride
}

// The sentinels below are the usage errors design D10 names. They are typed so
// the flag layer reports exit code 2 for any of them without parsing prose, and
// so no malformed input can become a silently different measurement.
var (
	// ErrUnknownProbe is an override naming a probe that is not declared.
	ErrUnknownProbe = errors.New("probe: target override names an unknown probe")
	// ErrLocalProbeOverride is an override for a probe that measures this
	// machine. Such a probe has no remote target, so replacing its targets would
	// declare a measurement nothing performs.
	ErrLocalProbeOverride = errors.New("probe: target override names a probe that measures no remote target")
	// ErrTargetAddress is an address that is not a usable "host[:port]": an empty
	// host, an empty port, a port outside 1..65535, or a colon-bearing value that
	// is not an IP literal.
	ErrTargetAddress = errors.New("probe: unparsable target address")
	// ErrTargetOverrideSyntax is a --target value that is not
	// "<probe-name>=<host[:port]>".
	ErrTargetOverrideSyntax = errors.New("probe: target override is not <probe-name>=<host[:port]>")
)

// The declared constants of the target set. Keeping each host in one constant is
// what lets the two region probes and the two TLS probes share a host without
// either of them being able to drift from the other.
const (
	// hubProbe names the probe whose target is the supplied hub address.
	hubProbe = "egress.hub.direct"
	// hubDefaultPort is the documented default port applied when the hub address
	// omits one (design D3).
	hubDefaultPort = 22
	// tlsProbeHost is PRD §1.1's measured TLS target: the host whose chain
	// verified with issuer Let's Encrypt/ISRG and verification code 0.
	tlsProbeHost = "www.cloudflare.com"
	// region1Host and region2Host are the two Cloudflare edge regions. Both are
	// probed (design D10), because "one region reachable" is materially different
	// from "both blocked".
	region1Host = "region1.v2.argotunnel.com"
	region2Host = "region2.v2.argotunnel.com"
)

// declaredTargetSet is the single home of the declared target set. It holds all
// ten probes in specification order, including the probes that declare no remote
// target, so an override naming any probe can be resolved and an override naming
// something else can be refused.
//
// Each entry's DefaultPort is the port an override without one resolves to; for
// the probes that declare endpoints it is the declared port itself, so "the
// declared target" and "the shown target" can never disagree.
var declaredTargetSet = []ProbeDeclaration{
	// This machine: nothing leaves the process, so nothing is declared.
	{Probe: "local.env", Protocol: ProtocolLocal},
	{Probe: "local.sshd", Protocol: ProtocolLocal},

	// The hub: its address is run input, never a constant, so the declaration
	// carries only the protocol and the documented default port.
	{Probe: hubProbe, Protocol: ProtocolTCP, DefaultPort: hubDefaultPort},

	// PRD §1.1's evidence: SSH is not blocked as a protocol, on either port.
	{
		Probe: "egress.ssh.known", Protocol: ProtocolTCP, DefaultPort: 22,
		Targets: []Target{{Host: "github.com", Port: 22}},
	},
	{
		Probe: "egress.ssh.443", Protocol: ProtocolTCP, DefaultPort: 443,
		Targets: []Target{{Host: "ssh.github.com", Port: 443}},
	},

	// Both edge regions on the tunnel port and on HTTPS, and both again for the
	// datagram question. One endpoint per region, so a split result survives into
	// the observations instead of being hidden behind a single aggregate.
	{
		Probe: "egress.cf.7844", Protocol: ProtocolTCP, DefaultPort: 7844,
		Targets: []Target{
			{Host: region1Host, Port: 7844, Label: "region1"},
			{Host: region2Host, Port: 7844, Label: "region2"},
		},
	},
	{
		Probe: "egress.cf.443", Protocol: ProtocolTCP, DefaultPort: 443,
		Targets: []Target{
			{Host: region1Host, Port: 443, Label: "region1"},
			{Host: region2Host, Port: 443, Label: "region2"},
		},
	},
	{
		Probe: "egress.quic", Protocol: ProtocolUDP, DefaultPort: 7844,
		Targets: []Target{
			{Host: region1Host, Port: 7844, Label: "region1"},
			{Host: region2Host, Port: 7844, Label: "region2"},
		},
	},

	// The chain on the wire and the chain against the local trust store, measured
	// against the host PRD §1.1 recorded as interception-free.
	{
		Probe: "tls.interception", Protocol: ProtocolTLS, DefaultPort: 443,
		Targets: []Target{{Host: tlsProbeHost, Port: 443}},
	},
	{
		Probe: "tls.truststore", Protocol: ProtocolTLS, DefaultPort: 443,
		Targets: []Target{{Host: tlsProbeHost, Port: 443}},
	},
}

// DeclaredTargets returns the declared target set, in specification order, as a
// deep copy: a caller may read it and may never edit the declaration through it.
//
// It is exported because "the declared set lives in one place" is a property an
// external test — and later the documentation and the payload — must be able to
// check without importing probe code or reading this file.
func DeclaredTargets() []ProbeDeclaration {
	declarations := make([]ProbeDeclaration, 0, len(declaredTargetSet))
	for _, declaration := range declaredTargetSet {
		copied := declaration
		copied.Targets = append([]Target(nil), declaration.Targets...)
		declarations = append(declarations, copied)
	}
	return declarations
}

// declarationFor finds one probe's declaration.
func declarationFor(name string) (ProbeDeclaration, bool) {
	for _, declaration := range declaredTargetSet {
		if declaration.Probe == name {
			return declaration, true
		}
	}
	return ProbeDeclaration{}, false
}

// EffectiveTargets resolves the declared set and the run input into the exact set
// of endpoints the run will dial, in declaration order.
//
// The rules, in the order they apply:
//
//  1. a supplied hub address replaces the hub probe's target, with the documented
//     default port applied when the address omits one (design D3);
//  2. each override replaces that probe's declared targets — it never appends —
//     so repeating a flag for one probe leaves exactly one target, and an
//     override built by hand rather than parsed is validated here too;
//  3. a probe that declares the local protocol or declares no endpoint
//     contributes nothing, because nothing measurable would come from it;
//  4. an unknown probe name, an address that cannot be parsed, or an override for
//     a probe with no remote target is an error and yields no set at all.
//
// The function is pure: the same input always resolves to the same set, which is
// what makes "the dialed set equals the declared set" (R-HR-NF-10) a comparison
// of two values.
func EffectiveTargets(in TargetInput) ([]EffectiveTarget, error) {
	replaced := map[string][]Target{}

	if in.Hub != "" {
		declaration, ok := declarationFor(hubProbe)
		if !ok {
			// Unreachable while the declaration holds the hub probe; the check
			// keeps the resolver honest if that entry is ever removed.
			return nil, fmt.Errorf("hub address %q: %w", in.Hub, ErrUnknownProbe)
		}
		host, port, err := parseAddress(in.Hub)
		if err != nil {
			return nil, fmt.Errorf("hub address %q: %w", in.Hub, err)
		}
		if port == 0 {
			port = declaration.DefaultPort
		}
		replaced[hubProbe] = []Target{{Host: host, Port: port}}
	}

	for _, override := range in.Overrides {
		declaration, ok := declarationFor(override.Probe)
		if !ok {
			return nil, fmt.Errorf("%q: %w", override.Probe, ErrUnknownProbe)
		}
		if declaration.Protocol == ProtocolLocal {
			return nil, fmt.Errorf("%q: %w", override.Probe, ErrLocalProbeOverride)
		}
		// A caller may build an override without going through the parser, so the
		// address is validated here as well: an empty host or a port outside the
		// legal range must never become a declared dial.
		if override.Host == "" || override.Port < 0 || override.Port > 65535 {
			return nil, fmt.Errorf("%q: address %q:%d: %w", override.Probe, override.Host, override.Port, ErrTargetAddress)
		}
		port := override.Port
		if port == 0 {
			port = declaration.DefaultPort
		}
		replaced[override.Probe] = []Target{{Host: override.Host, Port: port}}
	}

	var effective []EffectiveTarget
	for _, declaration := range declaredTargetSet {
		if declaration.Protocol == ProtocolLocal {
			continue
		}
		targets := declaration.Targets
		if replacement, ok := replaced[declaration.Probe]; ok {
			targets = replacement
		}
		for _, target := range targets {
			effective = append(effective, EffectiveTarget{
				Probe:    declaration.Probe,
				Protocol: declaration.Protocol,
				Host:     target.Host,
				Port:     target.Port,
				Label:    target.Label,
			})
		}
	}
	return effective, nil
}

// ParseTargetOverride parses one repeatable --target value of the form
// "<probe-name>=<host[:port]>".
//
// The probe name is resolved against the declaration here, so an unknown name or
// a probe that measures this machine is refused at the moment the flag is read
// rather than after a run has started. A value that names only a host keeps a port
// of zero, which EffectiveTargets resolves to the probe's declared default port.
func ParseTargetOverride(value string) (TargetOverride, error) {
	probeName, address, found := strings.Cut(value, "=")
	if !found {
		return TargetOverride{}, fmt.Errorf("%q: %w", value, ErrTargetOverrideSyntax)
	}
	probeName = strings.TrimSpace(probeName)
	address = strings.TrimSpace(address)
	if probeName == "" {
		return TargetOverride{}, fmt.Errorf("%q: %w", value, ErrUnknownProbe)
	}
	declaration, ok := declarationFor(probeName)
	if !ok {
		return TargetOverride{}, fmt.Errorf("%q: %w", probeName, ErrUnknownProbe)
	}
	if declaration.Protocol == ProtocolLocal {
		return TargetOverride{}, fmt.Errorf("%q: %w", probeName, ErrLocalProbeOverride)
	}
	host, port, err := parseAddress(address)
	if err != nil {
		return TargetOverride{}, fmt.Errorf("%q: %w", address, err)
	}
	return TargetOverride{Probe: probeName, Host: host, Port: port}, nil
}

// parseAddress splits a caller-supplied "host[:port]" into its parts, with a port
// of zero meaning the value omitted one.
//
// The rule is design D3's: net.SplitHostPort first, and when it fails because
// there is no port the whole value is a host, with a bracketed IPv6 literal
// unwrapped. An empty host, an empty port, a port outside 1..65535, or a
// colon-bearing value that is not an IP literal is an error: declaring a host
// that can never resolve would be a measurement of nothing, dressed as one.
func parseAddress(raw string) (string, int, error) {
	if raw == "" {
		return "", 0, fmt.Errorf("empty address: %w", ErrTargetAddress)
	}
	host, portText, err := net.SplitHostPort(raw)
	if err != nil {
		// No port at all: the value is a host. A bracketed IPv6 literal arrives
		// here as "[::1]" and is unwrapped; any other colon-bearing value must be
		// an IP literal, because a host name cannot contain a colon.
		host = strings.TrimSuffix(strings.TrimPrefix(raw, "["), "]")
		if host == "" {
			return "", 0, fmt.Errorf("empty host in %q: %w", raw, ErrTargetAddress)
		}
		if strings.Contains(host, ":") && net.ParseIP(host) == nil {
			return "", 0, fmt.Errorf("host %q is not an IP literal: %w", host, ErrTargetAddress)
		}
		return host, 0, nil
	}
	if host == "" {
		return "", 0, fmt.Errorf("empty host in %q: %w", raw, ErrTargetAddress)
	}
	if portText == "" {
		return "", 0, fmt.Errorf("empty port in %q: %w", raw, ErrTargetAddress)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return "", 0, fmt.Errorf("port %q in %q is not a number: %w", portText, raw, ErrTargetAddress)
	}
	if port < 1 || port > 65535 {
		return "", 0, fmt.Errorf("port %d in %q is out of range: %w", port, raw, ErrTargetAddress)
	}
	return host, port, nil
}
