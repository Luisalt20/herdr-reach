// Package version_test pins the build identity contract of the herdr-reach
// binary. It is an external test package on purpose: importing the module path
// proves the ratified module path resolves (RG-11) rather than only that a
// directory exists.
package version_test

import (
	"strings"
	"testing"

	"github.com/Luisalt20/herdr-reach/internal/version"
)

// TestUnsetDefaults pins the dev-build identity. A build produced without
// -ldflags must still name itself honestly instead of rendering an empty or
// placeholder version.
func TestUnsetDefaults(t *testing.T) {
	if got, want := version.Version, "0.0.0-dev"; got != want {
		t.Errorf("unset Version = %q, want %q", got, want)
	}
	if got, want := version.Name, "herdr-reach"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
	if got, want := version.String(), "herdr-reach 0.0.0-dev"; got != want {
		t.Errorf("unset String() = %q, want %q", got, want)
	}
}

// TestLdflagsInjectionShape covers what `-ldflags -X` does to this package: it
// overwrites the three variables at link time. The injected values must be
// readable, must survive into String(), and must not change the documented
// "name version" shape.
func TestLdflagsInjectionShape(t *testing.T) {
	cases := []struct {
		name    string
		version string
		commit  string
		date    string
		want    string
	}{
		{
			name:    "dev build with no injected metadata",
			version: "0.0.0-dev",
			want:    "herdr-reach 0.0.0-dev",
		},
		{
			name:    "release build with full injected metadata",
			version: "1.4.2",
			commit:  "9f1c2ab",
			date:    "2026-09-14T12:00:00Z",
			want:    "herdr-reach 1.4.2",
		},
		{
			name:    "pre-release build with injected commit only",
			version: "1.5.0-rc.1",
			commit:  "0d3e7f9",
			want:    "herdr-reach 1.5.0-rc.1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restore := inject(tc.version, tc.commit, tc.date)
			defer restore()

			if got := version.Version; got != tc.version {
				t.Errorf("injected Version = %q, want %q", got, tc.version)
			}
			if got := version.Commit; got != tc.commit {
				t.Errorf("injected Commit = %q, want %q", got, tc.commit)
			}
			if got := version.Date; got != tc.date {
				t.Errorf("injected Date = %q, want %q", got, tc.date)
			}
			if got := version.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestUnsetCommitIsNeverFabricated is the dev-build case: when no commit was
// injected, no commit-shaped text may be invented to fill the gap. This is the
// version-layer application of the change's governing principle.
func TestUnsetCommitIsNeverFabricated(t *testing.T) {
	restore := inject("0.0.0-dev", "", "")
	defer restore()

	got := version.String()
	if got != "herdr-reach 0.0.0-dev" {
		t.Fatalf("String() = %q, want the documented %q shape", got, "herdr-reach 0.0.0-dev")
	}
	for _, fabricated := range []string{"unknown", "none", "n/a", "0000000", "HEAD", "dirty"} {
		if strings.Contains(strings.ToLower(got), fabricated) {
			t.Errorf("unset Commit rendered the fabricated token %q in %q", fabricated, got)
		}
	}
}

// TestEmptyInjectionFallsBackToDevVersion covers the degenerate injection
// `-X ...Version=`: the variable is overwritten with an empty string. An empty
// version would render a blank, non-identifying string ("herdr-reach "), so the
// documented dev version is reported instead. The fallback is a real value, not
// a fabricated one.
func TestEmptyInjectionFallsBackToDevVersion(t *testing.T) {
	restore := inject("", "", "")
	defer restore()

	if got, want := version.String(), "herdr-reach 0.0.0-dev"; got != want {
		t.Errorf("String() with an empty injected Version = %q, want %q", got, want)
	}
}

// inject mimics `-ldflags -X` and returns a restore function so table cases do
// not leak state into each other.
func inject(v, commit, date string) func() {
	previous := [3]string{version.Version, version.Commit, version.Date}
	version.Version, version.Commit, version.Date = v, commit, date
	return func() {
		version.Version, version.Commit, version.Date = previous[0], previous[1], previous[2]
	}
}
