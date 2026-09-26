#!/usr/bin/env bash
# This exists so the Action step's glue is proven offline: how it learns its
# own ref and repository (github.action_ref, falling back to the runner's
# _actions/<owner>/<repo>/<ref> download path), and that nothing it meets,
# including an unreachable API, fails the consumer's job.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
self="$here/fovea-pin-self.sh"
fail=0
SHA="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

# expect NAME WANT_SUBSTRING -- ENV...
expect() {
  local name="$1" want="$2"; shift 3
  local out code
  out="$(env -i PATH="$PATH" RUNNER_TEMP="$(mktemp -d)" GITHUB_API_URL="http://127.0.0.1:9" "$@" bash "$self" 2>&1)"
  code=$?
  if [ "$code" -eq 0 ] && [[ "$out" == *"$want"* ]]; then
    echo "ok   $name"
  else
    echo "FAIL $name: exit $code, output: $out"; echo "     want substring: $want"; fail=$((fail + 1))
  fi
}

expect context_branch "used at ref 'main', a branch or tag" -- \
  FOVEA_ACTION_REF=main FOVEA_ACTION_REPOSITORY=macula-io/macula-fovea GITHUB_ACTION_PATH=/w/_actions/macula-io/macula-fovea/main
expect path_fallback_tag "used at ref 'v0.1.0', a branch or tag" -- \
  FOVEA_ACTION_REF= FOVEA_ACTION_REPOSITORY= GITHUB_ACTION_PATH=/home/runner/work/_actions/macula-io/macula-fovea/v0.1.0
expect path_fallback_repo "uses: some-fork/macula-fovea@<sha>" -- \
  FOVEA_ACTION_REF= FOVEA_ACTION_REPOSITORY= GITHUB_ACTION_PATH=/home/runner/work/_actions/some-fork/macula-fovea/main
expect local_path "no ref (the Action runs from a local path)" -- \
  FOVEA_ACTION_REF= FOVEA_ACTION_REPOSITORY= GITHUB_ACTION_PATH=/home/runner/work/repo/repo
expect sha_api_unreachable "staleness of pin $SHA is unknown" -- \
  FOVEA_ACTION_REF="$SHA" FOVEA_ACTION_REPOSITORY=macula-io/macula-fovea GITHUB_ACTION_PATH="/w/_actions/macula-io/macula-fovea/$SHA"
expect sha_path_fallback_unreachable "staleness of pin $SHA is unknown" -- \
  FOVEA_ACTION_REF= FOVEA_ACTION_REPOSITORY= GITHUB_ACTION_PATH="/w/_actions/macula-io/macula-fovea/$SHA"

expect path_fallback_last_actions_segment "used at ref 'z', a branch or tag" -- \
  FOVEA_ACTION_REF= FOVEA_ACTION_REPOSITORY= GITHUB_ACTION_PATH=/w/_actions/_actions/x/y/z
expect path_fallback_last_actions_repo "uses: x/y@<sha>" -- \
  FOVEA_ACTION_REF= FOVEA_ACTION_REPOSITORY= GITHUB_ACTION_PATH=/w/_actions/_actions/x/y/z
expect path_fallback_trailing_slash_sha "staleness of pin $SHA is unknown" -- \
  FOVEA_ACTION_REF= FOVEA_ACTION_REPOSITORY= GITHUB_ACTION_PATH="/w/_actions/macula-io/macula-fovea/$SHA/"

# The resolved ref is exported for the build step (tool version), and only
# when there is one.
# exported NAME WANT_LINE_OR_EMPTY -- ENV...
exported() {
  local name="$1" want="$2"; shift 3
  local envfile got
  envfile="$(mktemp)"
  env -i PATH="$PATH" RUNNER_TEMP="$(mktemp -d)" GITHUB_API_URL="http://127.0.0.1:9" GITHUB_ENV="$envfile" "$@" bash "$self" >/dev/null 2>&1
  got="$(grep '^FOVEA_RESOLVED_REF=' "$envfile" || true)"
  if [ "$got" = "$want" ]; then echo "ok   $name"; else echo "FAIL $name: GITHUB_ENV has '$got', want '$want'"; fail=$((fail + 1)); fi
  rm -f "$envfile"
}
exported export_context_ref "FOVEA_RESOLVED_REF=main" -- \
  FOVEA_ACTION_REF=main FOVEA_ACTION_REPOSITORY=macula-io/macula-fovea GITHUB_ACTION_PATH=/w/_actions/macula-io/macula-fovea/main
exported export_path_fallback_sha "FOVEA_RESOLVED_REF=$SHA" -- \
  FOVEA_ACTION_REF= FOVEA_ACTION_REPOSITORY= GITHUB_ACTION_PATH="/w/_actions/macula-io/macula-fovea/$SHA/"
exported export_nothing_for_local_path "" -- \
  FOVEA_ACTION_REF= FOVEA_ACTION_REPOSITORY= GITHUB_ACTION_PATH=/home/runner/work/repo/repo

# A tag lookup that stalls is cut off and degrades to the notice.
port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1])')"
python3 -c '
import http.server, sys, time
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self): time.sleep(120)
    def log_message(self, *a): pass
http.server.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
' "$port" &
stall=$!
sleep 0.5
start=$SECONDS
out="$(env -i PATH="$PATH" RUNNER_TEMP="$(mktemp -d)" GITHUB_API_URL="http://127.0.0.1:$port" FOVEA_PIN_CHECK_TIMEOUT=2 \
  FOVEA_ACTION_REF="$SHA" FOVEA_ACTION_REPOSITORY=macula-io/macula-fovea bash "$self" 2>&1)"
code=$?
took=$((SECONDS - start))
kill "$stall" 2>/dev/null; wait "$stall" 2>/dev/null
if [ "$code" -eq 0 ] && [ "$took" -le 8 ] && [[ "$out" == *"staleness of pin $SHA is unknown"* ]]; then
  echo "ok   lookup_stall_times_out (${took}s)"
else
  echo "FAIL lookup_stall_times_out: exit $code after ${took}s, output: $out"; fail=$((fail + 1))
fi

[ "$fail" -eq 0 ] || { echo "$fail case(s) failed"; exit 1; }
