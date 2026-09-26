#!/usr/bin/env bash
# This exists so every branch of the Action's pin-staleness check is proven
# offline: each case under testdata/pin-staleness/<case>/ gives the ref, the
# tag list (tags.json; absent means the lookup failed) and "now", and the
# exact stdout and job-summary text the check must produce.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
check="$here/fovea-pin-staleness.sh"
fail=0
for d in "$here"/testdata/pin-staleness/*/; do
  name="$(basename "$d")"
  work="$(mktemp -d)"
  tags="$work/absent.json"
  [ -f "$d/tags.json" ] && tags="$d/tags.json"
  GITHUB_STEP_SUMMARY="$work/summary" \
    bash "$check" "$(cat "$d/ref")" "$tags" "$(cat "$d/now")" > "$work/out" 2> "$work/err"
  code=$?
  touch "$work/summary"
  ok=1
  [ "$code" -eq 0 ] || { echo "FAIL $name: exit $code, want 0 (the check never fails a consumer's job)"; cat "$work/err"; ok=0; }
  if ! diff -u "$d/expect.stdout" "$work/out" > "$work/diff.out"; then
    echo "FAIL $name: stdout"; cat "$work/diff.out"; ok=0
  fi
  if ! diff -u "$d/expect.summary" "$work/summary" > "$work/diff.summary"; then
    echo "FAIL $name: job summary"; cat "$work/diff.summary"; ok=0
  fi
  [ "$ok" -eq 1 ] && echo "ok   $name" || fail=$((fail + 1))
  rm -rf "$work"
done
[ "$fail" -eq 0 ] || { echo "$fail case(s) failed"; exit 1; }
