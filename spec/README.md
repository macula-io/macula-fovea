# Fovea Specification

Versions live in directories (`v0.2/`, later `v0.3/`...). The newest
directory is normative; older ones are archival and must not be edited
except for errata marked as such.

The spec changes by PR, with the version bumped when any of the following
changes:

- the column set or attribute set (add, remove, rename, re-scope),
- the cell schema (required fields, enums, status semantics),
- the scorecard semantics (roll-up rules, headline metrics).

Editorial fixes (typos, clarifying examples) do not require a version bump.

Current version: [v0.3](v0.3/00-overview.md).

Proposals under review live in [`proposals/`](proposals/) — non-normative,
not lint-enforced, not declarable in a header until promoted.
