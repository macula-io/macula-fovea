# Input: an array of tags, each {"name": ..., ...}. Output: the tag whose
# name is the highest release vMAJOR.MINOR.PATCH by numeric semver, or null.
# Pre-releases (v1.0.0-rc.1) and non-version names (latest) are not releases.
map(select(.name | test("^v[0-9]+\\.[0-9]+\\.[0-9]+$")))
| sort_by(.name | ltrimstr("v") | split(".") | map(tonumber))
| last
