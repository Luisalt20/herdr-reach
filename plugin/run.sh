#!/bin/sh
# run.sh is the action entrypoint: it points `herdr-reach doctor` at the other
# side of the link, then execs the binary.
#
# The role ($1) does not change what doctor does. The tool is role-agnostic and
# --hub always names the other side of the link; the role only decides which
# configured address that is. `node` reads HERDR_REACH_HUB (the hub's address),
# `hub` reads HERDR_REACH_NODE (the node's address). The config file lives in
# HERDR_PLUGIN_CONFIG_DIR, the directory Herdr documents for user-editable
# configuration and injects at runtime.
#
# With no address configured the run still happens, without --hub: doctor
# reports the missing measurement as `not measured` instead of guessing. That
# is a valid run, so a missing configuration must not fail the action.
set -eu

role="${1:-}"
case "$role" in
  node) key=HERDR_REACH_HUB ;;
  hub) key=HERDR_REACH_NODE ;;
  *)
    echo "usage: run.sh node|hub" >&2
    exit 2
    ;;
esac

# Read the key with sed, tolerating an absent file or key and letting the last
# assignment win if the key repeats, the way a shell sourcing the file would.
address=""
config="${HERDR_PLUGIN_CONFIG_DIR:-}/herdr-reach.conf"
if [ -n "${HERDR_PLUGIN_CONFIG_DIR:-}" ] && [ -f "$config" ]; then
  address=$(sed -n "s/^${key}=//p" "$config" | tail -n 1)
fi

# $0 keeps the binary path correct whether Herdr runs the command from the
# plugin directory or a developer runs `sh plugin/run.sh` from the repository.
binary="$(dirname "$0")/bin/herdr-reach"

if [ -n "$address" ]; then
  exec "$binary" doctor --hub "$address"
fi

echo "herdr-reach: no ${key} configured; running without --hub, so the other side's address is reported as 'not measured'." >&2
exec "$binary" doctor
