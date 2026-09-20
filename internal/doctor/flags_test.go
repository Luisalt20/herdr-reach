package doctor_test

// This file is the flag surface's suite of design D3 and §7: the flags that parse
// into doctor.Options, the hub address rules — the documented default port, the
// bracketed IPv6 host-only form, the usage errors — the repeatable --target
// values, and the two non-run modes' streams. A usage error is asserted to be a
// typed error that carries the probe layer's own sentinel where one exists, and
// to start no measurement at all: the tripwire seam set below counts every
// capability call, and the control proves the tripwire is live.

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Luisalt20/herdr-reach/internal/doctor"
	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/version"
)

// --- the usage-error tripwire ------------------------------------------------

// tripwire is a seam set in which every capability call is counted. It is how a
// case asserts that a usage error started no measurement: a run that began would
// have read a seam, and the count would say so. The wrappers delegate to the
// deny-all set, so the counted run is a real (refused) run rather than a mock.
type tripwire struct {
	calls atomic.Int64
}

// newTripwire returns an unused tripwire.
func newTripwire() *tripwire { return &tripwire{} }

// bump records one capability call.
func (t *tripwire) bump() { t.calls.Add(1) }

// seams returns the counting seam set.
func (t *tripwire) seams() probe.Seams {
	base := probe.DenyAllSeams()
	base.Dialer = countingDialer{inner: base.Dialer, trip: t}
	base.Resolver = countingResolver{inner: base.Resolver, trip: t}
	base.TLSVerifier = countingTLSVerifier{inner: base.TLSVerifier, trip: t}
	base.PacketDialer = countingPacketDialer{inner: base.PacketDialer, trip: t}
	base.CommandRunner = countingCommandRunner{inner: base.CommandRunner, trip: t}
	base.Clock = countingClock{inner: base.Clock, trip: t}
	base.FS = countingFS{inner: base.FS, trip: t}
	base.Platform = countingPlatform{inner: base.Platform, trip: t}
	return base
}

// countingDialer counts dial attempts.
type countingDialer struct {
	inner probe.Dialer
	trip  *tripwire
}

// DialContext counts the attempt and delegates it.
func (d countingDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d.trip.bump()
	return d.inner.DialContext(ctx, network, addr)
}

// countingResolver counts name lookups.
type countingResolver struct {
	inner probe.Resolver
	trip  *tripwire
}

// LookupHost counts the lookup and delegates it.
func (r countingResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	r.trip.bump()
	return r.inner.LookupHost(ctx, host)
}

// countingTLSVerifier counts handshakes.
type countingTLSVerifier struct {
	inner probe.TLSVerifier
	trip  *tripwire
}

// Verify counts the handshake and delegates it.
func (v countingTLSVerifier) Verify(ctx context.Context, target string, cfg *tls.Config) (probe.TLSVerification, error) {
	v.trip.bump()
	return v.inner.Verify(ctx, target, cfg)
}

// countingPacketDialer counts datagram socket attempts.
type countingPacketDialer struct {
	inner probe.PacketDialer
	trip  *tripwire
}

// DialPacket counts the attempt and delegates it.
func (d countingPacketDialer) DialPacket(ctx context.Context, network, addr string) (probe.PacketConn, error) {
	d.trip.bump()
	return d.inner.DialPacket(ctx, network, addr)
}

// countingCommandRunner counts command invocations.
type countingCommandRunner struct {
	inner probe.CommandRunner
	trip  *tripwire
}

// Run counts the invocation and delegates it.
func (r countingCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	r.trip.bump()
	return r.inner.Run(ctx, name, args...)
}

// countingFS counts filesystem and environment reads.
type countingFS struct {
	inner probe.FS
	trip  *tripwire
}

// ReadFile counts the read and delegates it.
func (f countingFS) ReadFile(path string) ([]byte, error) {
	f.trip.bump()
	return f.inner.ReadFile(path)
}

// Stat counts the stat and delegates it.
func (f countingFS) Stat(path string) (os.FileInfo, error) {
	f.trip.bump()
	return f.inner.Stat(path)
}

// Getenv counts the environment read and delegates it.
func (f countingFS) Getenv(name string) string {
	f.trip.bump()
	return f.inner.Getenv(name)
}

// countingPlatform counts platform readings.
type countingPlatform struct {
	inner probe.Platform
	trip  *tripwire
}

// GOOS counts the reading and delegates it.
func (p countingPlatform) GOOS() string {
	p.trip.bump()
	return p.inner.GOOS()
}

// Arch counts the reading and delegates it.
func (p countingPlatform) Arch() string {
	p.trip.bump()
	return p.inner.Arch()
}

// WSL2 counts the reading and delegates it.
func (p countingPlatform) WSL2() bool {
	p.trip.bump()
	return p.inner.WSL2()
}

// Systemd counts the reading and delegates it.
func (p countingPlatform) Systemd() bool {
	p.trip.bump()
	return p.inner.Systemd()
}

// countingClock counts clock readings.
type countingClock struct {
	inner probe.Clock
	trip  *tripwire
}

// Now counts the reading and delegates it.
func (c countingClock) Now() time.Time {
	c.trip.bump()
	return c.inner.Now()
}

// runFromArgs is the entrypoint rule in miniature: parse the flags, and only when
// the parse succeeded run the harness. PR 19's main does exactly this around
// os.Exit; the test needs the same shape to assert that a usage error never
// reaches the measurement layer.
func runFromArgs(ctx context.Context, args []string, stdio doctor.Stdio, seams probe.Seams) int {
	opts, err := doctor.ParseFlags(args)
	if err != nil {
		fmt.Fprintln(stdio.Stderr, err)
		return doctor.ExitUsage
	}
	return doctor.Run(ctx, opts, stdio, seams)
}

// --- parsing -----------------------------------------------------------------

// TestFlagsParseIntoOptions asserts every flag of the surface parses into the
// Options field it describes, including the repeatable --target in the order the
// values were given.
func TestFlagsParseIntoOptions(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want doctor.Options
	}{
		{
			name: "the machine-readable flag",
			args: []string{"doctor", "--json"},
			want: doctor.Options{JSON: true},
		},
		{
			name: "the hub address",
			args: []string{"doctor", "--hub", "203.0.113.10:2222"},
			want: doctor.Options{Hub: "203.0.113.10:2222"},
		},
		{
			name: "the hub address in its --hub=value form",
			args: []string{"doctor", "--hub=203.0.113.10:2222"},
			want: doctor.Options{Hub: "203.0.113.10:2222"},
		},
		{
			name: "repeatable target overrides keep their order",
			args: []string{"doctor", "--target", "egress.hub.direct=198.51.100.7:2222", "--target", "egress.ssh.known=198.51.100.8"},
			want: doctor.Options{Targets: []probe.TargetOverride{
				{Probe: "egress.hub.direct", Host: "198.51.100.7", Port: 2222},
				{Probe: "egress.ssh.known", Host: "198.51.100.8"},
			}},
		},
		{
			name: "the version request",
			args: []string{"--version"},
			want: doctor.Options{Version: true},
		},
		{
			name: "the short help request",
			args: []string{"-h"},
			want: doctor.Options{Help: true},
		},
		{
			name: "the long help request",
			args: []string{"--help"},
			want: doctor.Options{Help: true},
		},
		{
			name: "the flags together",
			args: []string{"doctor", "--json", "--hub", "203.0.113.10", "--target", "egress.cf.443=198.51.100.9:443"},
			want: doctor.Options{
				JSON: true,
				Hub:  "203.0.113.10",
				Targets: []probe.TargetOverride{
					{Probe: "egress.cf.443", Host: "198.51.100.9", Port: 443},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := doctor.ParseFlags(tc.args)
			if err != nil {
				t.Fatalf("ParseFlags(%q) returned an error: %v", tc.args, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseFlags(%q) = %+v, want %+v", tc.args, got, tc.want)
			}
		})
	}
}

// TestFlagsSubcommandSelectsTheMode pins the documented invocation: `doctor` as the first argument
// selects a run and the remaining arguments are its flags, the two flag-only modes work with and
// without it, and every other first argument is a typed ErrSubcommand refusal that names the
// expected form and starts no run.
func TestFlagsSubcommandSelectsTheMode(t *testing.T) {
	accepted := []struct {
		name string
		args []string
		want doctor.Options
	}{
		{name: "the bare subcommand selects a run with the zero options", args: []string{"doctor"}},
		{
			name: "the subcommand passes the run flags through",
			args: []string{"doctor", "--json", "--hub", "203.0.113.10"},
			want: doctor.Options{JSON: true, Hub: "203.0.113.10"},
		},
		{name: "a version request after the subcommand", args: []string{"doctor", "--version"}, want: doctor.Options{Version: true}},
		{name: "a help request after the subcommand", args: []string{"doctor", "--help"}, want: doctor.Options{Help: true}},
		{name: "a version request without the subcommand", args: []string{"--version"}, want: doctor.Options{Version: true}},
		{name: "a short help request without the subcommand", args: []string{"-h"}, want: doctor.Options{Help: true}},
	}
	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			got, err := doctor.ParseFlags(tc.args)
			if err != nil {
				t.Fatalf("ParseFlags(%q) returned an error: %v", tc.args, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseFlags(%q) = %+v, want %+v", tc.args, got, tc.want)
			}
		})
	}

	expectedForm := version.Name + " doctor [flags]"
	refused := []struct {
		name string
		args []string
		// wantWord, when set, is the offending word the refusal must name.
		wantWord string
	}{
		{name: "no arguments at all", args: nil},
		{name: "a run flag without the subcommand", args: []string{"--json"}},
		{name: "an unknown subcommand", args: []string{"diagnose", "--json"}, wantWord: "diagnose"},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := doctor.ParseFlags(tc.args)
			if err == nil {
				t.Fatalf("ParseFlags(%q) accepted a command line that selects no mode", tc.args)
			}
			var usage *doctor.UsageError
			if !errors.As(err, &usage) {
				t.Fatalf("the error %v is not a typed UsageError", err)
			}
			if !errors.Is(err, doctor.ErrSubcommand) {
				t.Errorf("the error %v does not carry ErrSubcommand", err)
			}
			if !strings.Contains(err.Error(), expectedForm) {
				t.Errorf("the refusal %q does not name the expected form %q", err.Error(), expectedForm)
			}
			if tc.wantWord != "" && !strings.Contains(err.Error(), tc.wantWord) {
				t.Errorf("the refusal %q does not name the offending word %q", err.Error(), tc.wantWord)
			}
			if !reflect.DeepEqual(opts, doctor.Options{}) {
				t.Errorf("a refused parse returned %+v, want the zero Options", opts)
			}
		})
	}
}

// TestFlagsHubAddressRules asserts the documented D3 resolution: a supplied
// address is kept verbatim, the documented default port 22 applies when the port
// is omitted, and a bracketed IPv6 literal is accepted as a host-only value. The
// resolution is asserted through probe.EffectiveTargets, the one place the
// declared set is resolved, rather than through a second parse in the test.
func TestFlagsHubAddressRules(t *testing.T) {
	cases := []struct {
		name        string
		hub         string
		wantAddress string
	}{
		{name: "an explicit port is kept", hub: "203.0.113.10:2222", wantAddress: "203.0.113.10:2222"},
		{name: "an omitted port resolves to 22", hub: "203.0.113.10", wantAddress: "203.0.113.10:22"},
		{name: "a host name keeps its explicit port", hub: "hub.example.com:2200", wantAddress: "hub.example.com:2200"},
		{name: "a host name without a port resolves to 22", hub: "hub.example.com", wantAddress: "hub.example.com:22"},
		{name: "a bracketed IPv6 literal is accepted as host-only", hub: "[::1]", wantAddress: "[::1]:22"},
		{name: "a bracketed IPv6 literal keeps its port", hub: "[2001:db8::1]:2222", wantAddress: "[2001:db8::1]:2222"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := doctor.ParseFlags([]string{"doctor", "--hub", tc.hub})
			if err != nil {
				t.Fatalf("ParseFlags(--hub %q) returned an error: %v", tc.hub, err)
			}
			if opts.Hub != tc.hub {
				t.Fatalf("Options.Hub is %q, want the supplied value %q", opts.Hub, tc.hub)
			}

			targets, err := probe.EffectiveTargets(probe.TargetInput{Hub: opts.Hub})
			if err != nil {
				t.Fatalf("EffectiveTargets(%q) returned an error: %v", opts.Hub, err)
			}
			address := ""
			for _, target := range targets {
				if target.Probe == "egress.hub.direct" {
					address = target.Address()
					break
				}
			}
			if address != tc.wantAddress {
				t.Errorf("the resolved hub target is %q, want %q", address, tc.wantAddress)
			}
		})
	}
}

// TestParseFlagsUsageErrorsAreTyped asserts every refused input is a typed
// UsageError carrying the probe layer's own sentinel where one exists, so the
// entrypoint can map the refusal to exit 2 without parsing prose. The zero
// Options value is asserted too: a caller that ignored the error must not receive
// half-parsed input.
func TestParseFlagsUsageErrorsAreTyped(t *testing.T) {
	cases := []struct {
		name          string
		args          []string
		wantSentinels []error
	}{
		{
			name:          "an unknown flag in run mode",
			args:          []string{"doctor", "--bogus"},
			wantSentinels: []error{doctor.ErrUnknownFlag},
		},
		{
			name:          "a boolean flag given a value in run mode",
			args:          []string{"doctor", "--json=yes"},
			wantSentinels: []error{doctor.ErrFlagValue},
		},
		{
			name:          "a flag with no value at the end in run mode",
			args:          []string{"doctor", "--hub"},
			wantSentinels: []error{doctor.ErrFlagValue},
		},
		{
			name:          "a flag whose next token is another flag in run mode",
			args:          []string{"doctor", "--hub", "--json"},
			wantSentinels: []error{doctor.ErrFlagValue},
		},
		{
			name:          "an empty hub value",
			args:          []string{"doctor", "--hub", ""},
			wantSentinels: []error{probe.ErrTargetAddress},
		},
		{
			name:          "an empty hub host",
			args:          []string{"doctor", "--hub", ":22"},
			wantSentinels: []error{probe.ErrTargetAddress},
		},
		{
			name:          "an unparsable hub port",
			args:          []string{"doctor", "--hub", "203.0.113.10:notaport"},
			wantSentinels: []error{probe.ErrTargetAddress},
		},
		{
			name:          "an empty hub port",
			args:          []string{"doctor", "--hub", "203.0.113.10:"},
			wantSentinels: []error{probe.ErrTargetAddress},
		},
		{
			name:          "an unknown target probe name",
			args:          []string{"doctor", "--target", "no.such.probe=203.0.113.10"},
			wantSentinels: []error{probe.ErrUnknownProbe},
		},
		{
			name:          "a malformed target address",
			args:          []string{"doctor", "--target", "egress.hub.direct=203.0.113.10:notaport"},
			wantSentinels: []error{probe.ErrTargetAddress},
		},
		{
			name:          "a target value without an address",
			args:          []string{"doctor", "--target", "egress.hub.direct"},
			wantSentinels: []error{probe.ErrTargetOverrideSyntax},
		},
		{
			name:          "a target override for a local probe",
			args:          []string{"doctor", "--target", "local.env=203.0.113.10"},
			wantSentinels: []error{probe.ErrLocalProbeOverride},
		},
		{
			name:          "no command word at all",
			args:          nil,
			wantSentinels: []error{doctor.ErrSubcommand},
		},
		{
			name:          "a run flag without the subcommand",
			args:          []string{"--json"},
			wantSentinels: []error{doctor.ErrSubcommand},
		},
		{
			name:          "an unknown subcommand",
			args:          []string{"diagnose", "--json"},
			wantSentinels: []error{doctor.ErrSubcommand},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := doctor.ParseFlags(tc.args)
			if err == nil {
				t.Fatalf("ParseFlags(%q) accepted unusable input", tc.args)
			}
			var usage *doctor.UsageError
			if !errors.As(err, &usage) {
				t.Fatalf("the error %v is not a typed UsageError", err)
			}
			for _, sentinel := range tc.wantSentinels {
				if !errors.Is(err, sentinel) {
					t.Errorf("the error %v does not carry the sentinel %v", err, sentinel)
				}
			}
			if !reflect.DeepEqual(opts, doctor.Options{}) {
				t.Errorf("a refused parse returned %+v, want the zero Options", opts)
			}
		})
	}
}

// TestParseUsageErrorStartsNoMeasurement asserts a usage error reaches the exit
// code 2 without starting a measurement: the tripwire seam set records every
// capability call, and the accepted-run control proves the tripwire is live, so
// the zero counts are a real absence rather than a dead counter.
func TestParseUsageErrorStartsNoMeasurement(t *testing.T) {
	usageErrors := []struct {
		name string
		args []string
	}{
		{name: "no command word", args: nil},
		{name: "a run flag without the subcommand", args: []string{"--json"}},
		{name: "an unknown subcommand", args: []string{"diagnose", "--json"}},
		{name: "an unknown flag in run mode", args: []string{"doctor", "--bogus"}},
		{name: "an empty hub host in run mode", args: []string{"doctor", "--hub", ":22"}},
		{name: "an unparsable hub port in run mode", args: []string{"doctor", "--hub", "203.0.113.10:notaport"}},
		{name: "an unknown target probe in run mode", args: []string{"doctor", "--target", "no.such.probe=203.0.113.10"}},
		{name: "a malformed target address in run mode", args: []string{"doctor", "--target", "egress.hub.direct=203.0.113.10:notaport"}},
	}

	for _, tc := range usageErrors {
		t.Run(tc.name, func(t *testing.T) {
			trip := newTripwire()
			var stdout, stderr bytes.Buffer
			got := runFromArgs(context.Background(), tc.args,
				doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, trip.seams())

			if got != doctor.ExitUsage {
				t.Fatalf("the usage error exited %d, want %d", got, doctor.ExitUsage)
			}
			if trip.calls.Load() != 0 {
				t.Errorf("the refused input touched the seams %d times; it must start no measurement", trip.calls.Load())
			}
			if stdout.Len() != 0 {
				t.Errorf("the refused input wrote %q to the output writer", stdout.String())
			}
			if stderr.Len() == 0 {
				t.Errorf("the refusal was not reported on the error writer")
			}
		})
	}

	t.Run("control: an accepted parse reaches the measurement layer", func(t *testing.T) {
		trip := newTripwire()
		var stdout, stderr bytes.Buffer
		got := runFromArgs(context.Background(), []string{"doctor", "--json", "--hub", "203.0.113.10"},
			doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, trip.seams())

		if got == doctor.ExitUsage {
			t.Fatalf("the control run was refused; the parser accepted a value it must accept")
		}
		if trip.calls.Load() == 0 {
			t.Fatalf("the tripwire observed no seam call for an accepted run, so it cannot witness a refusal")
		}
		if stdout.Len() == 0 {
			t.Fatalf("the control run produced no machine-readable document")
		}
	})
}

// --- the two non-run modes ---------------------------------------------------

// TestStreamVersionPrintsToStdoutAndExitsZero pins the stream contract the issue
// fixed: --version is the answer a caller asked for, so it is written to the
// standard output writer and exits 0, and it starts no run at all.
func TestStreamVersionPrintsToStdoutAndExitsZero(t *testing.T) {
	opts, err := doctor.ParseFlags([]string{"--version"})
	if err != nil {
		t.Fatalf("ParseFlags(--version) returned an error: %v", err)
	}

	trip := newTripwire()
	var stdout, stderr bytes.Buffer
	got := doctor.Run(context.Background(), opts, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, trip.seams())

	if got != doctor.ExitOK {
		t.Fatalf("Run returned exit %d, want %d", got, doctor.ExitOK)
	}
	if strings.TrimSpace(stdout.String()) != version.String() {
		t.Errorf("stdout is %q, want the tool's version %q", stdout.String(), version.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr carries %q; a version request produces no human run text", stderr.String())
	}
	if trip.calls.Load() != 0 {
		t.Errorf("the version request touched the seams %d times; it is not a run", trip.calls.Load())
	}
}

// TestStreamHelpPrintsToStderrAndExitsZero pins the other stream contract: a help
// request is not a usage error, so it writes usage to the standard error writer
// and exits 0, and it starts no run.
func TestStreamHelpPrintsToStderrAndExitsZero(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			opts, err := doctor.ParseFlags(args)
			if err != nil {
				t.Fatalf("ParseFlags(%q) returned an error: %v", args, err)
			}

			trip := newTripwire()
			var stdout, stderr bytes.Buffer
			got := doctor.Run(context.Background(), opts, doctor.Stdio{Stdout: &stdout, Stderr: &stderr}, trip.seams())

			if got != doctor.ExitOK {
				t.Fatalf("Run returned exit %d, want %d for a help request", got, doctor.ExitOK)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout carries %q; usage belongs on the error writer", stdout.String())
			}
			if !strings.Contains(stderr.String(), "usage:") || !strings.Contains(stderr.String(), "--hub") {
				t.Errorf("stderr does not carry the usage text: %q", stderr.String())
			}
			if trip.calls.Load() != 0 {
				t.Errorf("the help request touched the seams %d times; it is not a run", trip.calls.Load())
			}
		})
	}
}
