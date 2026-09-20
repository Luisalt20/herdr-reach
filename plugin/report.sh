#!/bin/sh
# report.sh is the pane entrypoint: it shows the doctor report for whichever
# side this machine has configured and keeps the pane readable afterwards.
#
# It prefers HERDR_REACH_HUB and falls back to HERDR_REACH_NODE, because the
# report is worth showing from either machine; with neither configured it runs
# without --hub and doctor names the missing measurement instead of guessing.
# The value is always the other side's address, exactly as in run.sh.
#
# A split pane is an ordinary terminal pane, so it closes when its command
# exits and the report would disappear with it. The pane therefore waits for
# the reader before closing (the reference github-link-preview example holds
# its preview pane the same way), and the exit status stays the doctor's own:
# 0 measured, 1 incomplete, 2 usage error.
set -eu

config="${HERDR_PLUGIN_CONFIG_DIR:-}/herdr-reach.conf"
address=""
if [ -n "${HERDR_PLUGIN_CONFIG_DIR:-}" ] && [ -f "$config" ]; then
  # Last match wins for each key; the hub's address is the preferred one.
  address=$(sed -n 's/^HERDR_REACH_HUB=//p' "$config" | tail -n 1)
  if [ -z "$address" ]; then
    address=$(sed -n 's/^HERDR_REACH_NODE=//p' "$config" | tail -n 1)
  fi
fi

binary="$(dirname "$0")/bin/herdr-reach"

# Doctor is allowed to exit non-zero (1 means an incomplete measurement, not a
# failed run), so its status is captured and returned after the hold instead of
# ending the pane early.
status=0
if [ -n "$address" ]; then
  "$binary" doctor --hub "$address" || status=$?
else
  echo "herdr-reach: neither HERDR_REACH_HUB nor HERDR_REACH_NODE configured; running without --hub, so the other side's address is reported as 'not measured'." >&2
  "$binary" doctor || status=$?
fi

# Hold the pane so the report can be read before it closes.
printf '\npress enter to close this report...'
# read fails at end of input, for example when the pane has no terminal input;
# that must not overwrite the doctor's exit status.
read -r _ || true

exit "$status"
