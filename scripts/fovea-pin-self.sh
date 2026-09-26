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
#   GITHUB_ACTION_PATH, RUNNER_TEMP, GITHUB_STEP_SUMMARY, GITHUB_API_URL
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"

ref="${FOVEA_ACTION_REF-}"
repo="${FOVEA_ACTION_REPOSITORY-}"
# Composite actions have been known to see an empty github.action_ref
# (actions/runner#2473). The runner downloads a remote action to
# .../_actions/<owner>/<repo>/<ref>, so the path says the same thing. A
# local action (uses: ./) runs from the workspace and has no ref at all.
if [ -z "$ref" ] && [[ "${GITHUB_ACTION_PATH-}" =~ /_actions/([^/]+)/([^/]+)/(.+)$ ]]; then
  repo="${repo:-${BASH_REMATCH[1]}/${BASH_REMATCH[2]}}"
  ref="${BASH_REMATCH[3]}"
fi
export FOVEA_ACTION_REPOSITORY="${repo:-macula-io/macula-fovea}"

tags="${RUNNER_TEMP:-/tmp}/fovea-action-tags.json"
rm -f "$tags"
if [[ "$ref" =~ ^[0-9a-f]{40}$ ]]; then
  if "$here/fovea-pin-tags.sh" "$FOVEA_ACTION_REPOSITORY" > "$tags.part" 2>/dev/null; then
    mv "$tags.part" "$tags"
  else
    rm -f "$tags.part"
  fi
fi

bash "$here/fovea-pin-staleness.sh" "$ref" "$tags" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" ||
  echo "::notice title=fovea action pin::the pin check itself failed; staleness is unknown"
exit 0
