package doctor

// This file is the doctor command's flag surface of design D3 and §7: the run
// input the entrypoint parses before the harness runs. It is deliberately only a
// parser. It performs no measurement, opens nothing, dials nothing and executes
// nothing; the hub address and the probe overrides it validates are validated
// through the probe layer's own resolvers, so the parse rules have one home and
// an unusable value is refused with the probe layer's own sentinel rather than
// with a sentence someone would have to parse.
//
// The surface is `herdr-reach doctor [flags]`: the `doctor` word selects the run
// mode and is consumed here, so the command line's shape lives with the rest of
// the flag rules rather than in the entrypoint. The two flag-only modes
// (`--version`, `-h`/`--help`) need no subcommand; a run flag without the word or
// a word this surface does not define is refused with ErrSubcommand naming the
// expected form, while an unknown flag keeps the surface's own unknown-flag
// refusal. Once the run mode is selected, every argument this file does not
// define — a stray positional argument, an unknown flag, a flag missing its
// value — is a usage error rather than an ignored word.
//
// No `net`, `tls` or `os/exec` primitive is constructed here. The address rules
// are applied by internal/probe's exported resolver (EffectiveTargets and
// ParseTargetOverride), which is the one place the declared set and the default
// port live; this file only hands values over and wraps what comes back.

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Luisalt20/herdr-reach/internal/probe"
	"github.com/Luisalt20/herdr-reach/internal/version"
)

// Options is one run's input: the flags the surface parsed and the measurement
// bounds the run is to use.
type Options struct {
	// JSON requests the machine-readable document on the standard output writer
	// in addition to the human projection on the standard error writer.
	JSON bool
	// Hub is the supplied hub address, verbatim, in "host[:port]" form. It is
	// empty exactly when no --hub was given; the documented default port is
	// applied by probe.EffectiveTargets when the run resolves its declared set.
	Hub string
	// Targets are the parsed --target overrides, in the order given. Each one
	// replaces one probe's declared target rather than appending to it.
	Targets []probe.TargetOverride
	// Version requests the tool's version instead of a run.
	Version bool
	// Help requests the usage text instead of a run. It is not a usage error:
	// a help request exits 0.
	Help bool
	// Run are the run's measurement bounds. ParseFlags leaves every field zero,
	// so the runner's documented defaults apply; a caller — a test, or a later
	// slice's entrypoint — may inject its own. The Clock field is always the
	// run's seam clock when the harness runs, because every timestamp in the
	// output must come from the one clock the run was given.
	Run probe.Options
}

// The flag surface's own sentinels. They exist for the facts that have no probe
// layer sentinel of their own — a flag that is not defined, a defined flag given
// no value or a value it does not take — so every refusal is branchable with
// errors.Is instead of by reading a message. The probe layer's sentinels
// (probe.ErrTargetAddress, probe.ErrUnknownProbe, probe.ErrLocalProbeOverride,
// probe.ErrTargetOverrideSyntax) travel inside UsageError unchanged.
var (
	// ErrUnknownFlag is a token the surface does not define: a flag that does
	// not exist, or a positional argument, which this surface accepts none of.
	ErrUnknownFlag = errors.New("doctor: unknown flag or argument")
	// ErrFlagValue is a defined flag given no value, or a value it does not
	// take. A flag that needs a value never consumes the next flag: a following
	// token that begins with "-" is reported as the missing value rather than
	// silently consumed.
	ErrFlagValue = errors.New("doctor: unusable flag value")
	// ErrSubcommand is a command line that selects no mode: the `doctor`
	// subcommand is missing (no arguments, or a flag where the command word
	// belongs) or is a word this surface does not define. The refusal names the
	// expected form.
	ErrSubcommand = errors.New("doctor: missing or unknown subcommand")
)

// UsageError is the typed error ParseFlags returns for input it cannot use. It
// wraps the probe layer's own sentinel when the refused value came from that
// layer, so a caller can branch on errors.Is against probe.ErrTargetAddress and
// friends and exit 2 without parsing prose.
type UsageError struct {
	// Err is the refusal: a probe-layer sentinel with its context, or one of
	// this file's own sentinels.
	Err error
}

// Error reports the refusal's message.
func (e *UsageError) Error() string { return e.Err.Error() }

// Unwrap returns the wrapped refusal, which is what makes errors.Is and
// errors.As reach it.
func (e *UsageError) Unwrap() error { return e.Err }

// usageText is the surface's usage, written when --help/-h is requested. It is
// generated from the version package's tool name so the command's own name has
// one home.
const usageText = "usage: " + version.Name + " doctor [flags]\n" +
	"\n" +
	"Measure this machine's reachability and report why a transport cannot work.\n" +
	"The run measures only; it changes nothing.\n" +
	"\n" +
	"  --json                     write the machine-readable document to standard output\n" +
	"  --hub host[:port]          measure this hub address (default port 22)\n" +
	"  --target probe=host[:port] replace one probe's declared target (repeatable)\n" +
	"  --version                  print the tool's version and exit\n" +
	"  -h, --help                 print this usage and exit\n" +
	"\n" +
	"With --json the document is the only thing on standard output; all human\n" +
	"output goes to standard error.\n"

// subcommand is the command word of the documented invocation,
// `herdr-reach doctor [flags]`. It is the only command word this surface
// defines; the two flag-only modes need none.
const subcommand = "doctor"

// ParseFlags parses the doctor command line into one run's Options.
//
// The accepted forms, and the mode each selects:
//
//   - `doctor [flags]` selects a run: the command word is consumed and the
//     remaining arguments parse as the run's flags.
//   - `--version` selects the version mode, with or without `doctor`; no
//     measurement is started.
//   - `-h` / `--help` selects the help mode, with or without `doctor`; no
//     measurement is started.
//
// Any other command line selects no mode and is refused: no argument at all or a
// run flag without the command word (`herdr-reach`, `herdr-reach --json`) is a
// missing subcommand, and a word this surface does not define
// (`herdr-reach diagnose`) is an unknown subcommand — both carry ErrSubcommand
// and name the expected form. An unknown flag that is not a run flag
// (`herdr-reach --bogus`) keeps this surface's existing unknown-flag refusal,
// because the token itself is the refused fact. Once the run mode is selected,
// every argument the surface does not define — an extra positional, an unknown
// flag, a flag missing its value — is refused exactly as before, with the
// sentinels this surface already had.
//
// The run rules are design D3's. `--hub host[:port]` keeps the address verbatim and
// is validated through probe.EffectiveTargets, so the documented default port,
// the bracketed-IPv6 host-only form and the refusals (an empty host, an
// unparsable port) are decided by the probe layer's one resolver. A repeatable
// `--target probe=host[:port]` is parsed through probe.ParseTargetOverride, so an
// unknown probe name or a malformed address is refused when the flag is read
// rather than after a run has started. `--json`, `--version` and `-h`/`--help`
// follow the same value rules.
//
// A refused argument returns the zero Options and a *UsageError. The zero value
// matters: a caller that ignored the error must not receive half-parsed input
// that would run as though the rest of the command line did not exist.
func ParseFlags(args []string) (Options, error) {
	var opts Options

	switch {
	case len(args) > 0 && args[0] == subcommand:
		args = args[1:]
	case flagOnlyMode(args):
		// The two flag-only modes need no subcommand; the flag loop below parses them.
	case len(args) == 0 || runFlagToken(args[0]):
		// A run flag without the command word selects no mode.
		return Options{}, subcommandError(args)
	case strings.HasPrefix(args[0], "-"):
		// An unknown flag keeps the surface's own refusal: the loop below reports it
		// as ErrUnknownFlag rather than as a missing subcommand, because the token
		// itself is the refused fact.
	default:
		// A word where the command word belongs is an unknown subcommand.
		return Options{}, subcommandError(args)
	}

	for index := 0; index < len(args); index++ {
		// Both "--flag value" and "--flag=value" are accepted; the value form
		// for a value-taking flag and the bare form for a boolean. The token is
		// split at its first "=" so a target value that itself contains "="
		// (--target probe=host) survives intact.
		name, value, hasValue := strings.Cut(args[index], "=")

		switch name {
		case "--json":
			if hasValue {
				return Options{}, flagValueError(name, "the flag takes no value")
			}
			opts.JSON = true

		case "--version":
			if hasValue {
				return Options{}, flagValueError(name, "the flag takes no value")
			}
			opts.Version = true

		case "-h", "--help":
			if hasValue {
				return Options{}, flagValueError(name, "the flag takes no value")
			}
			opts.Help = true

		case "--hub":
			resolved, next, err := flagValue(name, value, hasValue, args, index)
			if err != nil {
				return Options{}, err
			}
			index = next
			// The value is trimmed because surrounding whitespace is never part
			// of an address, and a value that is only whitespace is the empty
			// value D3 refuses rather than an absent --hub.
			hub := strings.TrimSpace(resolved)
			if hub == "" {
				return Options{}, &UsageError{Err: fmt.Errorf("%s: empty address: %w", name, probe.ErrTargetAddress)}
			}
			if _, err := probe.EffectiveTargets(probe.TargetInput{Hub: hub}); err != nil {
				return Options{}, &UsageError{Err: err}
			}
			opts.Hub = hub

		case "--target":
			resolved, next, err := flagValue(name, value, hasValue, args, index)
			if err != nil {
				return Options{}, err
			}
			index = next
			override, err := probe.ParseTargetOverride(resolved)
			if err != nil {
				return Options{}, &UsageError{Err: err}
			}
			opts.Targets = append(opts.Targets, override)

		default:
			return Options{}, &UsageError{Err: fmt.Errorf("%q: %w", args[index], ErrUnknownFlag)}
		}
	}

	return opts, nil
}

// flagOnlyMode reports whether the first argument is one of the two modes that
// need no subcommand: --version, -h or --help. The name is cut at its first "="
// so `--version=x` still reaches the flag loop, where it is refused as a value
// on a flag that takes none rather than reported as a missing subcommand.
func flagOnlyMode(args []string) bool {
	if len(args) == 0 {
		return false
	}
	name, _, _ := strings.Cut(args[0], "=")
	switch name {
	case "--version", "-h", "--help":
		return true
	}
	return false
}

// runFlagToken reports whether a token names one of the run mode's own flags. It
// is how a run flag used without the `doctor` word is told apart from an unknown
// flag: the run flag means the command word is missing, while an unknown flag
// keeps the surface's existing unknown-flag refusal. The name is cut at its first
// "=" so the --flag=value form is recognised too.
func runFlagToken(token string) bool {
	name, _, _ := strings.Cut(token, "=")
	switch name {
	case "--json", "--hub", "--target":
		return true
	}
	return false
}

// subcommandError is the refusal for a command line that selects no mode. It
// names the expected form, so a user is told the tool's own documented
// invocation, and it carries ErrSubcommand so the entrypoint can map the refusal
// to exit 2 without parsing prose. A word where the command word belongs is the
// unknown case and is named too; no argument at all, or a flag, is the missing
// case.
func subcommandError(args []string) error {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return &UsageError{Err: fmt.Errorf("unknown subcommand %q: expected %q: %w",
			args[0], version.Name+" "+subcommand+" [flags]", ErrSubcommand)}
	}
	return &UsageError{Err: fmt.Errorf("missing subcommand: expected %q: %w",
		version.Name+" "+subcommand+" [flags]", ErrSubcommand)}
}

// flagValue resolves one value-taking flag to its value and the index the caller
// must continue from.
//
// An inline value ("--hub=host") is used as given. A bare flag takes the next
// token, except when that token begins with "-": a flag is then missing its
// value and is refused rather than consuming the flag after it, which would
// silently turn "doctor --hub --json" into a run against a host literally named
// "--json".
func flagValue(name, value string, hasValue bool, args []string, index int) (string, int, error) {
	if hasValue {
		return value, index, nil
	}
	if index+1 >= len(args) || strings.HasPrefix(args[index+1], "-") {
		return "", index, flagValueError(name, "the flag needs a value")
	}
	return args[index+1], index + 1, nil
}

// flagValueError builds the refusal for a flag whose value the surface cannot
// use, in the message context that names the flag.
func flagValueError(name, reason string) error {
	return &UsageError{Err: fmt.Errorf("%s: %s: %w", name, reason, ErrFlagValue)}
}

// writeUsage writes the surface's usage text to w and returns the writer's error,
// so the caller can exit 2 rather than pretend the help request was answered when
// its stream refused the text.
func writeUsage(w io.Writer) error {
	_, err := io.WriteString(w, usageText)
	return err
}
