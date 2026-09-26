# Changelog

All notable spec versions and tooling milestones.

## Unreleased (tooling; spec stays 0.3)

The lint now enforces the spec it implements, and every rule is tested.

- **Fixed grid.** A header declares exactly the 16 spec columns, each in its
  own family, plus `x_` extensions; the core five attributes; only
  `possession` and `utility` as extensions, each disabled one with a
  written justification. Dropping, moving, doubling or inventing a column
  or attribute is a lint error.
- **Cell rules.** A present `unassessed` cell is an error. Owner checks
  ignore case and spacing. `roadmap` cells need a definition and
  manifestations; any cell with a `roadmap` measure needs `review_by`.
- **One N/A test.** A whitespace `na_reason` is unjustified in lint, score
  and render alike; the v0.3 grid shows it as unfinished (red).
- **Open gaps** list unassigned owners and overdue roadmaps, and are no
  longer cut at 40.
- **Paths.** `cells_dir` works without a trailing slash; `init` creates it
  when missing and never overwrites an existing cell file.
- **Output.** Diagnostics go to stderr; `score --json` stdout is valid JSON
  even when lint fails. `render --format md|html|json` is accepted as the
  spec writes it. Unknown flags exit 2.
- **Issues bridge.** Refuses to act when the header or a cell fails to
  load (a YAML typo used to close that cell's issue); no token without
  `--dry-run` is an error; 30 s API timeout; pull requests are never edited
  or closed.
- **Action.** Inputs pass through environment variables instead of being
  interpolated into bash (script injection).
- **Removed `schema/`.** No tool loaded the JSON Schemas and they
  contradicted the CLI. The CLI's `lint` is the reference implementation;
  its rule cases under `cli/internal/core/testdata/` seed a language-neutral
  conformance suite. Spec 12 (v0.2 and v0.3) carries an erratum.
- **Spec errata and editorial fixes** (v0.3): 12 status propagation, 13
  unjustified N/A in the grid, 00 "This is v0.3", 11 wording.
- **CI.** gofmt, `go vet`, `go test -race`, build, and an end-to-end run of
  `action.yml` on every push and pull request.
- `.gitignore` no longer hides `cli/cmd/fovea/`.

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
- JSON Schemas for header and cell (removed later: unused, see Unreleased).
- Dogfood assessment skeleton for `macula-mesh-realm`.
- `packs/` and `templates/` structure.

## [0.1] — never released

Unnumbered earlier drafts circulated as documents; v0.2 is the first versioned
specification.
