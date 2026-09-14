// Package version carries the build identity of the herdr-reach binary.
//
// The four values below are the single home of "which build is this?". They are
// plain variables rather than constants so a release can overwrite them at link
// time:
//
//	go build -ldflags "\
//	  -X github.com/Luisalt20/herdr-reach/internal/version.Version=1.4.2 \
//	  -X github.com/Luisalt20/herdr-reach/internal/version.Commit=9f1c2ab \
//	  -X github.com/Luisalt20/herdr-reach/internal/version.Date=2026-09-14T12:00:00Z"
//
// A build without those flags is a legitimate dev build: it reports the
// documented development version and it never invents a commit, a date or a
// placeholder token to fill the gap.
package version

// Name is the tool name as it appears in the payload's `tool.name` field and in
// the version string.
const Name = "herdr-reach"

// DevVersion is the version reported when no release value was injected.
const DevVersion = "0.0.0-dev"

var (
	// Version is the build's version, overwritten at link time for releases.
	// It is never empty: an uninjected build reports DevVersion.
	Version = DevVersion

	// Commit is the VCS revision the binary was built from, injected at link
	// time. It is empty for a dev build, and an empty Commit is reported as
	// absent rather than substituted with a fabricated revision.
	Commit = ""

	// Date is the build timestamp injected at link time, empty for a dev build.
	Date = ""
)

// String returns the documented "name version" identity of this build, for
// example "herdr-reach 0.0.0-dev".
//
// An empty Version — the shape left by a degenerate `-X ...Version=` link flag —
// reports DevVersion instead, so the identity is never blank.
//
// Commit and Date are deliberately not part of the string: the version identity
// has one documented shape, and appending optional build metadata to it would
// make the shape depend on how the binary happened to be built.
func String() string {
	if Version == "" {
		return Name + " " + DevVersion
	}
	return Name + " " + Version
}
