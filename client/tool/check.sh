#!/usr/bin/env bash
# Runs the client checks — pub get, analyze, test — and saves everything they
# print to .check.log, so the result can be read without scrolling a terminal.
#
#   tool/check.sh            run once, print the result
#   tool/check.sh --watch    stay running; run again every time .check-request
#                            changes (write anything new into that file)
#
# The watch mode exists so an assistant working on this code without Flutter
# of its own can ask for a run and read the answer. It only ever runs the three
# commands below. Stop it with Ctrl-C.
set -u
cd "$(dirname "$0")/.."

run() {
  local request="${1:-manual}"
  local out=".check.log.tmp"
  {
    echo "request: $request"
    echo "started: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo
    echo "== flutter pub get"
    flutter pub get 2>&1 | tail -n 8
    echo "exit: ${PIPESTATUS[0]}"
    echo
    echo "== flutter analyze"
    flutter analyze 2>&1
    echo "exit: $?"
    echo
    echo "== flutter test"
    flutter test 2>&1
    echo "exit: $?"
    echo
    echo "finished: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  } > "$out"
  # Renamed into place only when complete, so a reader never sees half a log.
  mv "$out" .check.log
}

if [ "${1:-}" != "--watch" ]; then
  run
  cat .check.log
  exit 0
fi

echo "Watching for check requests in $(pwd). Leave this running; Ctrl-C to stop."
last="$(cat .check-request 2>/dev/null || true)"
while true; do
  request="$(cat .check-request 2>/dev/null || true)"
  if [ -n "$request" ] && [ "$request" != "$last" ]; then
    last="$request"
    echo "[$(date +%H:%M:%S)] running checks ($request)"
    run "$request"
    echo "[$(date +%H:%M:%S)] done — see .check.log"
  fi
  sleep 2
done
