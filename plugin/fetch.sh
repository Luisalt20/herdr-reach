#!/bin/sh
# fetch.sh installs the herdr-reach binary this plugin runs.
#
# It is the manifest's [[build]] command, so `herdr plugin install` fetches a
# released binary for this platform before Herdr registers the plugin. Linking a
# working tree does not run build commands; run this script yourself when you
# develop against a local checkout.
#
# The asset and the release's own SHA256SUMS are downloaded together and the
# asset is checked against the digest the release published. The install fails
# loudly on any mismatch: Herdr stores this binary and later executes it, and
# Herdr does not review or sandbox plugin code (plugins doc, "Trust and
# security"). An unverified binary never reaches plugin/bin.
#
# Herdr runs build commands with the plugin directory as the working directory,
# and this script anchors the install to its own directory anyway; either way
# the binary lands in bin/, which the runtime entrypoints exec relative to the
# plugin directory.
set -eu

# TOOL_TAG is the one home of the tool's release tag in this plugin. It must
# match the manifest's `version` and the GitHub release tag: the manifest
# mirrors the release so the two move together.
TOOL_TAG="v0.1.0-beta.2"
REPO="Luisalt20/herdr-reach"

# Map this machine to the release's asset suffix. The plugin declares linux and
# macOS because it runs on the machine whose Herdr holds the UI; anything else
# refuses the install instead of guessing an asset that was never built.
os=$(uname -s)
arch=$(uname -m)
case "${os}/${arch}" in
  Linux/x86_64 | Linux/amd64) suffix=linux_amd64 ;;
  Linux/aarch64 | Linux/arm64) suffix=linux_arm64 ;;
  Darwin/x86_64 | Darwin/amd64) suffix=darwin_amd64 ;;
  Darwin/arm64 | Darwin/aarch64) suffix=darwin_arm64 ;;
  *)
    echo "herdr-reach: unsupported platform ${os}/${arch}: this plugin installs release binaries for linux and macOS on x86_64 and arm64 only" >&2
    exit 1
    ;;
esac

asset="herdr-reach_${TOOL_TAG}_${suffix}"
base="https://github.com/${REPO}/releases/download/${TOOL_TAG}"

# Anchor the install to this script's own directory. Herdr guarantees the plugin
# directory as the working directory, and anchoring also keeps a developer's
# manual `sh plugin/fetch.sh` from installing the binary somewhere else.
plugin_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)

# The download and the checksum check live in a private temp directory, removed
# on every exit — success or failure — so a rejected download leaves no binary
# behind.
tmp=$(mktemp -d "${TMPDIR:-/tmp}/herdr-reach.XXXXXX")
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

# A failed download is a failed install: never continue past it.
if ! curl -fsSL -o "${tmp}/${asset}" "${base}/${asset}"; then
  echo "herdr-reach: could not download ${base}/${asset}" >&2
  exit 1
fi
if ! curl -fsSL -o "${tmp}/SHA256SUMS" "${base}/SHA256SUMS"; then
  echo "herdr-reach: could not download ${base}/SHA256SUMS" >&2
  exit 1
fi

# SHA256SUMS lists every platform, so only the line for this exact asset is
# checked: the other lines name binaries that are not on this machine and
# checking them would say nothing about the bytes that were downloaded. The
# digest is not embedded here on purpose; the release's published file is the
# authority, and the platform's own tool does the arithmetic (sha256sum on
# Linux, shasum on macOS).
if command -v sha256sum >/dev/null 2>&1; then
  checksum_check="sha256sum -c"
elif command -v shasum >/dev/null 2>&1; then
  checksum_check="shasum -a 256 -c"
else
  echo "herdr-reach: no SHA-256 tool found: need sha256sum (Linux) or shasum -a 256 (macOS) to verify the download" >&2
  exit 1
fi

if ! grep " ${asset}\$" "${tmp}/SHA256SUMS" > "${tmp}/${asset}.sha256"; then
  echo "herdr-reach: ${asset} is not listed in the release's SHA256SUMS: refusing to install an unverified binary" >&2
  exit 1
fi

# The check runs from the temp directory so the filename on the line resolves to
# the file that was just downloaded.
if ! (cd "$tmp" && $checksum_check "${asset}.sha256"); then
  echo "herdr-reach: checksum mismatch for ${asset}: the download does not match the release's SHA256SUMS; refusing to install" >&2
  exit 1
fi

# Install only after the bytes are verified. The runtime entrypoints exec
# bin/herdr-reach relative to the plugin directory.
mkdir -p "${plugin_dir}/bin"
cp "${tmp}/${asset}" "${plugin_dir}/bin/herdr-reach"
chmod 0755 "${plugin_dir}/bin/herdr-reach"

# Print the version so the install log shows which release was installed; the
# checksum proved the bytes, this proves the build.
"${plugin_dir}/bin/herdr-reach" --version
