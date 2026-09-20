package version_test

// This file guards the Herdr plugin's version declarations against the drift
// that shipped v0.1.0-beta.3 without bumping them. The plugin is a hybrid: its
// scripts come from main, while fetch.sh downloads the binary of the release
// the manifest declares — a manifest left at the previous version installs the
// previous binary under the new scripts, and a marketplace user has no way to
// tell.
//
// plugin/herdr-plugin.toml's top-level `version` and plugin/fetch.sh's
// TOOL_TAG are the two homes of that declaration, and they must name the same
// release: TOOL_TAG is "v" followed by the manifest's version. The release
// workflow enforces the shape on the tag being cut; this test enforces the
// same shape on the manifest and asserts the pair stays together.
//
// The parses are anchored to the start of a line on purpose, because the
// failure this guards is a human forgetting a bump: a declaration that is
// commented out, named only in the prose beside it, or absent must not satisfy
// the guard. The counter-cases at the bottom prove each parser refuses a
// commented line, an absent declaration, an ambiguous pair and an empty file,
// so a broken scan fails rather than passing vacuously.
//
// The content is normalized from CRLF to LF before either parse runs: git
// checks the files out with \r\n on Windows, where the line anchors would
// otherwise match nothing and the guard would fail on every Windows checkout.
// The verdict must not depend on the machine's core.autocrlf setting, and the
// CRLF counter-cases below hold the parsers to the same answer for those bytes.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	// manifestRelativePath is the manifest the installed plugin reads.
	manifestRelativePath = "plugin/herdr-plugin.toml"

	// fetchRelativePath is the build script that downloads the binary.
	fetchRelativePath = "plugin/fetch.sh"
)

// semverShape is the release workflow's tag shape with the leading v removed:
// <major>.<minor>.<patch>, optionally followed by a hyphenated suffix. The
// workflow already refuses a tag that does not match it, and the manifest
// carries that tag's version, so this guard reuses the one shape instead of
// inventing a second one.
var semverShape = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$`)

// manifestVersionDecl matches a top-level TOML declaration on its own line.
// The line anchors are the point: `# version = "..."` is a comment, and only a
// real assignment may satisfy the guard.
var manifestVersionDecl = regexp.MustCompile(`(?m)^version = "([^"]*)"$`)

// toolTagDecl matches fetch.sh's TOOL_TAG assignment on its own line, with the
// same anchoring: the comment beside the assignment names TOOL_TAG freely, and
// a commented-out assignment must not count.
var toolTagDecl = regexp.MustCompile(`(?m)^TOOL_TAG="([^"]*)"$`)

// TestPluginDeliversTheVersionItDeclares is the drift guard: it reads both
// declarations from the repository root, asserts the manifest's version is the
// release workflow's shape, and asserts TOOL_TAG is exactly "v" followed by
// that version. A failure names both values, because the reader's next action
// is to bump one of them.
func TestPluginDeliversTheVersionItDeclares(t *testing.T) {
	root := moduleRoot(t)
	manifest := readRepoFile(t, root, manifestRelativePath)
	script := readRepoFile(t, root, fetchRelativePath)

	declaredVersion, err := manifestVersion(manifest)
	if err != nil {
		t.Fatalf("the manifest does not expose the version it declares: %v", err)
	}
	declaredTag, err := toolTag(script)
	if err != nil {
		t.Fatalf("fetch.sh does not expose the tag it downloads: %v", err)
	}

	if !semverShape.MatchString(declaredVersion) {
		t.Errorf("plugin/herdr-plugin.toml declares version %q, which is not <major>.<minor>.<patch> optionally with a hyphenated suffix: the release workflow refuses to tag anything else, so the manifest must carry a version that could be released", declaredVersion)
	}
	if want := "v" + declaredVersion; declaredTag != want {
		t.Errorf("plugin version drift: plugin/herdr-plugin.toml declares version %q, so plugin/fetch.sh's TOOL_TAG must be %q, but it is %q: the manifest and TOOL_TAG move together and name the release the plugin delivers", declaredVersion, want, declaredTag)
	}
}

// TestPluginVersionParsersRejectCommentsAndAbsence is the guard's
// positive-control half. Each case below is content a scan might meet, and the
// parse must refuse every shape that is not exactly one real declaration on its
// own line: a commented-out declaration, a different key, a second ambiguous
// declaration and an empty file all fail. This is what keeps the guard in the
// test above honest — it cannot be satisfied by the prose beside a declaration
// or by a line that was commented out instead of updated. The CRLF cases are
// the acceptance side of the same control: the declaration with the line
// endings a Windows checkout produces must still extract the value, so the
// parse cannot stop matching on one platform and keep passing on another.
func TestPluginVersionParsersRejectCommentsAndAbsence(t *testing.T) {
	manifestCases := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{"a declaration on its own line", "version = \"1.2.3\"\n", "1.2.3", false},
		{"a declaration on a CRLF checkout parses the same value", toCRLF("version = \"1.2.3\"\n"), "1.2.3", false},
		{"a pre-release declaration", "version = \"1.2.3-beta.1\"\n", "1.2.3-beta.1", false},
		{"a commented-out declaration is not a declaration", "# version = \"1.2.3\"\n", "", true},
		{"another key is not the version", "min_herdr_version = \"0.7.0\"\n", "", true},
		{"two declarations are ambiguous", "version = \"1.2.3\"\nversion = \"1.2.4\"\n", "", true},
		{"an empty file found nothing", "", "", true},
	}
	for _, tc := range manifestCases {
		t.Run("manifest/"+tc.name, func(t *testing.T) {
			got, err := manifestVersion(tc.content)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("manifestVersion(%q) = %q, want an error: a commented, absent or ambiguous declaration must not satisfy the guard", tc.content, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("manifestVersion(%q) returned an error: %v", tc.content, err)
			}
			if got != tc.want {
				t.Errorf("manifestVersion(%q) = %q, want %q", tc.content, got, tc.want)
			}
		})
	}

	toolCases := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{"an assignment on its own line", "TOOL_TAG=\"v1.2.3\"\n", "v1.2.3", false},
		{"an assignment on a CRLF checkout parses the same value", toCRLF("TOOL_TAG=\"v1.2.3\"\n"), "v1.2.3", false},
		{"a commented-out assignment is not an assignment", "# TOOL_TAG=\"v1.2.3\"\n", "", true},
		{"prose that names TOOL_TAG is not the assignment", "# TOOL_TAG is the one home of the release tag.\n", "", true},
		{"two assignments are ambiguous", "TOOL_TAG=\"v1.2.3\"\nTOOL_TAG=\"v1.2.4\"\n", "", true},
		{"an empty file found nothing", "", "", true},
	}
	for _, tc := range toolCases {
		t.Run("fetch/"+tc.name, func(t *testing.T) {
			got, err := toolTag(tc.content)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("toolTag(%q) = %q, want an error: a commented, absent or ambiguous assignment must not satisfy the guard", tc.content, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("toolTag(%q) returned an error: %v", tc.content, err)
			}
			if got != tc.want {
				t.Errorf("toolTag(%q) = %q, want %q", tc.content, got, tc.want)
			}
		})
	}
}

// manifestVersion returns the one top-level version declaration in a plugin
// manifest, or an error naming why there is no single declaration.
func manifestVersion(content string) (string, error) {
	return oneDeclaration("plugin/herdr-plugin.toml's top-level version", manifestVersionDecl, content)
}

// toolTag returns the one TOOL_TAG assignment in fetch.sh, or an error naming
// why there is no single assignment.
func toolTag(content string) (string, error) {
	return oneDeclaration("plugin/fetch.sh's TOOL_TAG", toolTagDecl, content)
}

// oneDeclaration requires exactly one match: zero means the scan found nothing
// and must fail rather than pass, and more than one means the reader cannot
// tell which declaration the plugin delivers. The content is normalized from
// CRLF to LF first, so both declarations are matched the same way whatever
// line endings the checkout uses.
func oneDeclaration(what string, decl *regexp.Regexp, content string) (string, error) {
	matches := decl.FindAllStringSubmatch(normalizeLineEndings(content), -1)
	switch {
	case len(matches) == 0:
		return "", fmt.Errorf("%s has no declaration on its own line: found 0, want exactly 1", what)
	case len(matches) > 1:
		values := make([]string, len(matches))
		for i, match := range matches {
			values[i] = match[1]
		}
		return "", fmt.Errorf("%s has %d declarations (%s), want exactly 1", what, len(matches), strings.Join(values, ", "))
	default:
		return matches[0][1], nil
	}
}

// normalizeLineEndings converts the \r\n a Windows checkout hands the guard to
// the \n the line-anchored parses expect. It is the one normalization point:
// both declarations are parsed through oneDeclaration above.
func normalizeLineEndings(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

// toCRLF re-spells a known-good declaration the way a Windows checkout does,
// so the counter-cases can ask each parser for the same value it gives the LF
// bytes. It shapes test input; the parsers themselves own the normalization.
func toCRLF(content string) string {
	return strings.ReplaceAll(content, "\n", "\r\n")
}

// moduleRoot walks up from the test's working directory — the package
// directory under `go test` — to the directory holding go.mod, which is the
// root both files are read from. The walk keeps the guard independent of where
// the package sits in the tree and of how many levels it has.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolving the working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("walked to the filesystem root without finding go.mod")
		}
		dir = parent
	}
}

// readRepoFile reads one repository-relative file and refuses to hand back an
// empty one: a scan that read nothing must fail rather than pass. A read error
// fails for the same reason — the guard cannot vouch for bytes it never saw.
func readRepoFile(t *testing.T, root, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("reading %s from the repository root: %v", relative, err)
	}
	if strings.TrimSpace(string(content)) == "" {
		t.Fatalf("%s is empty: a scan that read nothing must fail rather than pass", relative)
	}
	return string(content)
}
