#!/usr/bin/env bash
# This exists so every third-party action this repo runs stays pinned to a
# full commit sha with its release tag as a comment, and a regression to a
# moving ref such as @v4 or @main fails CI.
#
# usage: scripts/check-pinned-actions.sh [FILE...]
# default files: action.yml or action.yaml, and every workflow under
# .github/workflows/ (*.yml and *.yaml), whichever exist.
# Local actions (uses: ./...) and docker:// images are out of scope.
set -euo pipefail

if [ "$#" -eq 0 ]; then
  shopt -s nullglob
  set -- action.yml action.yaml .github/workflows/*.yml .github/workflows/*.yaml
  shopt -u nullglob
  existing=()
  for f in "$@"; do
    [ -f "$f" ] && existing+=("$f")
  done
  if [ "${#existing[@]}" -eq 0 ]; then
    echo "no action.yml or workflow files found" >&2
    exit 1
  fi
  set -- "${existing[@]}"
fi

pinned='^[[:space:]]*(-[[:space:]]+)?uses:[[:space:]]+[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+@[0-9a-f]{40}[[:space:]]+#[[:space:]]+v[0-9]+(\.[0-9]+)*[[:space:]]*$'
bad=0
for f in "$@"; do
  n=0
  while IFS= read -r line; do
    n=$((n + 1))
    case "$line" in
      *uses:*) ;;
      *) continue ;;
    esac
    # skip comments, local actions and docker images
    if [[ "$line" =~ ^[[:space:]]*# ]] || [[ "$line" =~ uses:[[:space:]]+\./ ]] || [[ "$line" =~ uses:[[:space:]]+docker:// ]]; then
      continue
    fi
    if ! [[ "$line" =~ $pinned ]]; then
      echo "::error file=$f,line=$n,title=unpinned action::$(echo "$line" | sed 's/^[[:space:]]*//') (pin to a 40-char commit sha with '# vX.Y.Z')"
      bad=$((bad + 1))
    fi
  done < "$f"
done
if [ "$bad" -gt 0 ]; then
  echo "$bad unpinned action reference(s)" >&2
  exit 1
fi
echo "all action references are pinned by sha"
