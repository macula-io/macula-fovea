#!/usr/bin/env bash
# This exists so the pin guard is proven to see every file GitHub reads:
# workflows as .yml and .yaml, and the Action as action.yml or action.yaml.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
guard="$here/check-pinned-actions.sh"
fail=0
PINNED='      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1'
LOOSE='      - uses: actions/checkout@v4'
FIRST_PARTY='    uses: macula-io/macula-ci-images/.github/workflows/c4-model.yml@83a76a358456bf2f7d6c919ad5e8fe7fd2162ce3'
FIRST_PARTY_LOOSE='    uses: macula-io/macula-ci-images/.github/workflows/c4-model.yml@main'
THIRD_PARTY_NO_TAG='      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1'

# case NAME WANT_EXIT FILE=CONTENT...
case_() {
  local name="$1" want="$2"; shift 2
  local d; d="$(mktemp -d)"
  mkdir -p "$d/.github/workflows"
  for spec in "$@"; do printf 'steps:\n%s\n' "${spec#*=}" > "$d/${spec%%=*}"; done
  (cd "$d" && bash "$guard" >/dev/null 2>&1)
  local code=$?
  if [ "$code" -eq "$want" ]; then echo "ok   $name"; else echo "FAIL $name: exit $code, want $want"; fail=$((fail + 1)); fi
  rm -rf "$d"
}

case_ pinned_yml_passes 0 ".github/workflows/ci.yml=$PINNED" "action.yml=$PINNED"
case_ loose_yml_fails 1 ".github/workflows/ci.yml=$LOOSE" "action.yml=$PINNED"
case_ loose_yaml_workflow_fails 1 ".github/workflows/ci.yml=$PINNED" ".github/workflows/release.yaml=$LOOSE" "action.yml=$PINNED"
case_ loose_action_yaml_fails 1 ".github/workflows/ci.yml=$PINNED" "action.yaml=$LOOSE"
case_ only_yaml_files_pinned_passes 0 ".github/workflows/ci.yaml=$PINNED" "action.yaml=$PINNED"

case_ first_party_workflow_by_sha_passes 0 ".github/workflows/c4.yml=$FIRST_PARTY" "action.yml=$PINNED"
case_ first_party_workflow_by_branch_fails 1 ".github/workflows/c4.yml=$FIRST_PARTY_LOOSE" "action.yml=$PINNED"
case_ third_party_sha_without_tag_fails 1 ".github/workflows/ci.yml=$THIRD_PARTY_NO_TAG" "action.yml=$PINNED"

[ "$fail" -eq 0 ] || { echo "$fail case(s) failed"; exit 1; }
