#!/usr/bin/env bash
# This exists so a consumer pinned to an old fovea Action sees it in their
# own CI: a dead Dependabot must not leave the security check silently stale.
#
# usage: fovea-pin-staleness.sh REF TAGS_JSON NOW
#   REF        the Action's own ref (github.action_ref); "" for a local path
#   TAGS_JSON  file with [{"name","sha","created_at"?}] from
#              fovea-pin-tags.sh; missing or unreadable = lookup failed.
#              created_at is needed on the newest release only.
#   NOW        ISO 8601 UTC time, e.g. 2026-09-26T12:00:00Z
# env: GITHUB_STEP_SUMMARY (optional), FOVEA_ACTION_REPOSITORY (for the
#      pin hint; default macula-io/macula-fovea)
#
# It prints GitHub workflow commands and always exits 0: staleness is a
# warning, never a reason to fail someone's security CI.
set -uo pipefail

ref="${1-}"
tags="${2-}"
now="${3-}"
repo="${FOVEA_ACTION_REPOSITORY:-macula-io/macula-fovea}"
grace_days=14
title="fovea action pin"
here="$(cd "$(dirname "$0")" && pwd)"

summary() {
  if [ -n "${GITHUB_STEP_SUMMARY-}" ]; then
    printf '%s\n' "$1" >> "$GITHUB_STEP_SUMMARY"
  fi
}

if [ -z "$ref" ]; then
  echo "$title: no ref (the Action runs from a local path); nothing to check"
  exit 0
fi

if ! [[ "$ref" =~ ^[0-9a-f]{40}$ ]]; then
  echo "::warning title=$title::the fovea Action is used at ref '$ref', a branch or tag, not a 40-character commit sha. Pin it: uses: $repo@<sha> # vX.Y.Z"
  exit 0
fi

if ! newest="$(jq -ce -f "$here/lib/newest-release.jq" "$tags" 2>/dev/null)"; then
  if jq -e 'type == "array"' "$tags" >/dev/null 2>&1; then
    echo "::notice title=$title::the fovea Action repository has no vX.Y.Z release tags yet; staleness of pin $ref is unknown"
  else
    echo "::notice title=$title::could not list the fovea Action's release tags (network, rate limit or permissions); staleness of pin $ref is unknown"
  fi
  exit 0
fi

name="$(jq -r '.name' <<< "$newest")"
sha="$(jq -r '.sha' <<< "$newest")"
created="$(jq -r '.created_at // empty' <<< "$newest")"

if [ "$sha" = "$ref" ]; then
  echo "$title: $ref is $name, the newest release"
  exit 0
fi

# A pin that is no release commit at all (a main commit, a fork) is not
# "behind": it was never a release, and the fix is to pin one.
if ! jq -e --arg r "$ref" 'any(.[]; (.name | test("^v[0-9]+\\.[0-9]+\\.[0-9]+$")) and .sha == $r)' "$tags" >/dev/null 2>&1; then
  echo "::warning title=$title::pinned to $ref, which is not a release commit; pin a release. The newest is $name ($sha)."
  exit 0
fi

# jq's fromdateiso8601 accepts only YYYY-MM-DDTHH:MM:SSZ. Each date is
# parsed on its own, so an offset, fractional seconds or garbage leads to
# the notice, never to an aborted script.
created_s=""
if [ -n "$created" ]; then
  created_s="$(jq -rn --arg c "$created" '$c | fromdateiso8601' 2>/dev/null)" || created_s=""
fi
now_s="$(jq -rn --arg n "$now" '$n | fromdateiso8601' 2>/dev/null)" || now_s=""
if [ -z "$created_s" ] || [ -z "$now_s" ]; then
  echo "::notice title=$title::the date of the newest release $name is unknown; staleness of pin $ref is unknown"
  exit 0
fi
age_days=$(( (now_s - created_s) / 86400 ))

if [ "$age_days" -gt "$grace_days" ]; then
  echo "::warning title=$title::pinned to $ref, which is not the newest release $name ($sha); $name is $age_days days old. Update the sha (Dependabot's github-actions ecosystem does this)."
  summary "**fovea action pin is stale:** \`$ref\` is not the newest release \`$name\`, which is $age_days days old."
  exit 0
fi

echo "$title: $ref is not the newest release $name, which is $age_days days old; within the $grace_days-day grace period"
exit 0
