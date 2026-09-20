#!/bin/sh
# run.sh is the action entrypoint: it points `herdr-reach doctor` at the other
# side of the link, captures the doctor's status, and explains a non-zero
# status on stderr without changing it. Herdr derives an action's log status
# from the process exit code, and exit 1 means an incomplete measurement -- a
# result, not a broken run.
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

# explain_status prints the meaning of a non-zero doctor status to stderr. The
# status itself is left to the caller: exiting 0 for an incomplete run would
# break doctor's contract and hide real errors from anything scripting this
# action. Keep the case in sync with report.sh, where the pane must tell the
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

# Capture the status instead of exec'ing so a non-zero one can be explained;
# the status itself is still the process's own. The failing command sits in an
# if condition so `set -e` does not fire before it is captured.
if [ -n "$address" ]; then
  if "$binary" doctor --hub "$address"; then status=0; else status=$?; fi
else
  echo "herdr-reach: no ${key} configured; running without --hub, so the other side's address is reported as 'not measured'." >&2
  if "$binary" doctor; then status=0; else status=$?; fi
fi

if [ "$status" -ne 0 ]; then
  explain_status "$status"
fi

exit "$status"
