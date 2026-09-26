#!/usr/bin/env bash
# This exists to give fovea-pin-staleness.sh the fovea Action's release tags:
# [{"name","sha"}] for every tag, and "created_at" on the newest vX.Y.Z,
# which is the tagger date of an annotated tag or the commit date of a
# lightweight one.
#
# usage: fovea-pin-tags.sh OWNER/REPO
# env: FOVEA_TOKEN (optional, raises the rate limit), GITHUB_API_URL
#      (default https://api.github.com; set by Actions, also on GHES)
# Prints the JSON and exits 0, or prints nothing and exits non-zero.
set -euo pipefail

repo="$1"
api="${GITHUB_API_URL:-https://api.github.com}"
here="$(cd "$(dirname "$0")" && pwd)"

get() {
  local auth=()
  if [ -n "${FOVEA_TOKEN-}" ]; then
    auth=(-H "Authorization: Bearer $FOVEA_TOKEN")
  fi
  curl -fsS --connect-timeout 5 --max-time 10 --retry 1 \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    ${auth[@]+"${auth[@]}"} "$api/repos/$repo/$1"
}

all='[]'
for page in $(seq 1 10); do
  batch="$(get "tags?per_page=100&page=$page" | jq -c 'map({name, sha: .commit.sha})')"
  all="$(jq -c --argjson b "$batch" '. + $b' <<< "$all")"
  [ "$(jq 'length' <<< "$batch")" -lt 100 ] && break
done

newest="$(jq -r -f "$here/lib/newest-release.jq" <<< "$all" | jq -r '.name // empty')"
if [ -z "$newest" ]; then
  echo "$all"
  exit 0
fi

read -r type obj < <(get "git/ref/tags/$newest" | jq -r '"\(.object.type) \(.object.sha)"')
case "$type" in
  tag) created="$(get "git/tags/$obj" | jq -er '.tagger.date')" ;;
  commit) created="$(get "git/commits/$obj" | jq -er '.committer.date')" ;;
  *) echo "unexpected object type '$type' for tag $newest" >&2; exit 1 ;;
esac

jq -c --arg n "$newest" --arg c "$created" 'map(if .name == $n then . + {created_at: $c} else . end)' <<< "$all"
