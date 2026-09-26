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

[ "$fail" -eq 0 ] || { echo "$fail case(s) failed"; exit 1; }
