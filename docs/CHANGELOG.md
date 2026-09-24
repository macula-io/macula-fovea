# Changelog

All notable spec versions and tooling milestones.

## [0.3] — 2026-09-25

- Scorecard becomes coverage-aware: per-block `authored/total` next to the
  (unchanged) worst-RAG letter, plus an open-gaps section naming missing /
  unassessed / unjustified-NA / overdue cells (13-scorecard).
- CLI `render` branches on the header's spec version: 0.2 headers keep the
  frozen grid, 0.3 headers get the coverage-aware grid.
- GitHub Action writes artifacts: score JSON + rendered scorecard into an
  artifact directory, plus a step-summary block.

## [0.2] — 2026-09-25

- Initial versioned spec (`spec/v0.2/`): 00-overview, 10-axes, 11-attributes, 12-cell-schema, 13-scorecard, 14-instantiation.
- JSON Schemas for header and cell.
- Dogfood assessment skeleton for `macula-mesh-realm`.
- `packs/` and `templates/` structure.

## [0.1] — never released

Unnumbered earlier drafts circulated as documents; v0.2 is the first versioned
specification.
