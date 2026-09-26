#!/usr/bin/env bash
# This exists so the tag lookup behind the pin-staleness check is proven
# offline: a local HTTP server serves each case's api/ tree as the GitHub
# REST API, and the fetched tag list must equal expect.json (or the fetch
# must fail with no output, when expect.fail exists).
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
fetch="$here/fovea-pin-tags.sh"
fail=0
for d in "$here"/testdata/pin-tags/*/; do
  name="$(basename "$d")"
  port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1])')"
  python3 -m http.server "$port" --bind 127.0.0.1 --directory "$d/api" >/dev/null 2>&1 &
  srv=$!
  for _ in $(seq 50); do
    curl -s -o /dev/null "http://127.0.0.1:$port/" && break
    sleep 0.1
  done
  out="$(GITHUB_API_URL="http://127.0.0.1:$port" FOVEA_TOKEN=test bash "$fetch" o/r 2>/dev/null)"
  code=$?
  kill "$srv" 2>/dev/null; wait "$srv" 2>/dev/null
  if [ -f "$d/expect.fail" ]; then
    if [ "$code" -ne 0 ] && [ -z "$out" ]; then echo "ok   $name"; else echo "FAIL $name: exit $code, output '$out'; want a failure with no output"; fail=$((fail + 1)); fi
    continue
  fi
  if [ "$code" -eq 0 ] && [ "$(jq -S . <<< "$out" 2>/dev/null)" = "$(jq -S . "$d/expect.json")" ]; then
    echo "ok   $name"
  else
    echo "FAIL $name: exit $code"; echo "got:  $out"; echo "want: $(jq -c . "$d/expect.json")"; fail=$((fail + 1))
  fi
done
[ "$fail" -eq 0 ] || { echo "$fail case(s) failed"; exit 1; }
