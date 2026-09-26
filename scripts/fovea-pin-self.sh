#!/usr/bin/env bash
# This exists so every consumer run of the fovea Action reports how old its
# own pin is. It is the Action step's whole body: learn the Action's ref and
# repository, list the release tags when the ref is a sha, and let
# fovea-pin-staleness.sh decide. It never fails the job.
#
# env (all set by action.yml from contexts, never interpolated into bash):
#   FOVEA_ACTION_REF         github.action_ref
#   FOVEA_ACTION_REPOSITORY  github.action_repository
#   FOVEA_TOKEN              github.token, for the tag lookup
#   GITHUB_ACTION_PATH, RUNNER_TEMP, GITHUB_STEP_SUMMARY, GITHUB_API_URL,
#   GITHUB_ENV (FOVEA_RESOLVED_REF is exported there for the build step)
#   FOVEA_PIN_CHECK_TIMEOUT  seconds the tag lookup may take (default 60)
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"

ref="${FOVEA_ACTION_REF-}"
repo="${FOVEA_ACTION_REPOSITORY-}"
# Composite actions have been known to see an empty github.action_ref
# (actions/runner#2473). The runner downloads a remote action to
# .../_actions/<owner>/<repo>/<ref>, so the path says the same thing. A
# local action (uses: ./) runs from the workspace and has no ref at all.
# The greedy ^.* anchors on the LAST /_actions/ segment; trailing slashes
# are dropped first.
path="${GITHUB_ACTION_PATH-}"
while [[ "$path" == */ ]]; do path="${path%/}"; done
if [ -z "$ref" ] && [[ "$path" =~ ^.*/_actions/([^/]+)/([^/]+)/(.+)$ ]]; then
  repo="${repo:-${BASH_REMATCH[1]}/${BASH_REMATCH[2]}}"
  ref="${BASH_REMATCH[3]}"
fi

# The build step bakes the resolved ref into the binary as its version.
if [ -n "$ref" ] && [ -n "${GITHUB_ENV-}" ] && [[ "$ref" != *$'\n'* ]]; then
  printf 'FOVEA_RESOLVED_REF=%s\n' "$ref" >> "$GITHUB_ENV"
fi
export FOVEA_ACTION_REPOSITORY="${repo:-macula-io/macula-fovea}"

tags="${RUNNER_TEMP:-/tmp}/fovea-action-tags.json"
rm -f "$tags"
if [[ "$ref" =~ ^[0-9a-f]{40}$ ]]; then
  # Bounded: a stalled API degrades to the notice, never holds the job.
  bound=()
  if command -v timeout >/dev/null 2>&1; then
    bound=(timeout "${FOVEA_PIN_CHECK_TIMEOUT:-60}")
  fi
  if ${bound[@]+"${bound[@]}"} "$here/fovea-pin-tags.sh" "$FOVEA_ACTION_REPOSITORY" > "$tags.part" 2>/dev/null; then
    mv "$tags.part" "$tags"
  else
    rm -f "$tags.part"
  fi
fi

bash "$here/fovea-pin-staleness.sh" "$ref" "$tags" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" ||
  echo "::notice title=fovea action pin::the pin check itself failed; staleness is unknown"
exit 0
