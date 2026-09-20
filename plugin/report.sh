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
# 0 measured, 1 incomplete, 2 usage error. A non-zero status is explained on
# stderr after the report, exactly as run.sh explains the action's status, so
# the pane and the action tell the same story.
set -eu

# explain_status prints the meaning of a non-zero doctor status to stderr. The
# status itself is left to the caller: exiting 0 for an incomplete run would
# break doctor's contract and hide real errors from anything scripting this
# entrypoint. Keep the case in sync with run.sh, where the action must tell the
# same story.
explain_status() {
  case "$1" in
    1)
      echo "herdr-reach: exit 1: the measurement completed and is incomplete: at least one probe was attempted and produced no answer." >&2
      echo "herdr-reach: the unresolved line in the report above names the question that could not be settled; this is a result, not a failure of the run." >&2
      ;;
    2)
      echo "herdr-reach: exit 2: usage or internal error: no diagnosis is presented as completed and standard output is empty." >&2
      ;;
    *)
      echo "herdr-reach: exit $1: outside doctor's documented exit codes (0 = measurement completed, 1 = incomplete run, 2 = usage or internal error)." >&2
      ;;
  esac
}

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

if [ "$status" -ne 0 ]; then
  explain_status "$status"
fi

# Hold the pane so the report can be read before it closes.
printf '\npress enter to close this report...'
# read fails at end of input, for example when the pane has no terminal input;
# that must not overwrite the doctor's exit status.
read -r _ || true

exit "$status"
