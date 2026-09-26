# fovea (CLI)

Single static Go binary implementing the fovea spec
([v0.3](../spec/v0.3/00-overview.md) and the archival
[v0.2](../spec/v0.2/00-overview.md); both share the grid and cell rules, v0.3
changes only the scorecard) against an **assessment directory** (header +
cells). It is the reference implementation of the spec: where the spec says
"the lint", it means this `lint`.

## Commands

| Command | What it does | Exit non-zero when |
|---|---|---|
| `fovea init <dir>` | Reads `fovea.yaml`, computes the declared grid, creates `cells_dir` if missing and every missing `<column>.<attribute>.yaml` as an `unassessed` skeleton. Only adds files: an existing cell, even one that fails to parse, is never overwritten. Refuses to run while the header has findings. | header invalid |
| `fovea lint <dir> [--github]` | Header checks (the fixed grid, attributes, owner, version) + grid closure (every expected cell exists) + per-cell rules + copy-paste detection (>=85% definition similarity). `--github` emits workflow commands that annotate the offending files in a PR. | any error finding |
| `fovea score <dir> [--json]` | Computes the [scorecard](../spec/v0.3/13-scorecard.md): expected/present/missing cells, `pct_unassessed`, `na_unjustified`, `pct_by_design`, oldest `review_by` and overdue roadmaps, cells per attribute and per family. Missing cells count as unassessed. With `--json`, stdout is valid JSON even when lint fails. | lint had errors |
| `fovea render <dir> [--format md\|html\|json]` | The scorecard. v0.2 headers get the frozen worst-RAG grid; v0.3 headers get the coverage-aware grid (`R 2/6`) and the complete open-gaps list. `--html` and `--json` are short for the formats. | header unreadable |
| `fovea issues <dir> [--dry-run] [--check] [--repo o/r] [--token t]` | Syncs `roadmap` cells to GitHub issues, idempotently via a `<!-- fovea-cell: <id> -->` body marker. Refuses to act when the header or any cell fails to load. `--check` fails when a roadmap cell's linked issue is closed. See [issues-bridge](../docs/issues-bridge.md). | load findings, no token without `--dry-run`, `--check` trap, API errors |

`dir` defaults to the current directory and may come before or after the
flags. An unknown flag exits 2 with usage. Results go to stdout; load
errors, notes and warnings go to stderr, so `fovea score --json dir >
score.json` stays parseable.

## Lint rules

Every finding carries a stable rule code; `docs/reference.md` lists them
all. The rules, by family:

- **The grid is fixed** (spec 10, 11): the header declares exactly the 16
  spec columns, each in its own family, plus `x_`-prefixed extension
  columns; exactly the core five attributes; only `possession` and `utility`
  as extensions; a non-blank justification for each extension not enabled.
- **The grid is closing**: every expected cell exists; no cell id names an
  undeclared column or attribute; the file name matches the id.
- **Every cell is answered**: `unassessed` is an error; `owner` is claimed
  (`unassigned` in any case or spacing is not a claim).
- **No empty answers**: `assumed`, `assessed` and `roadmap` cells need a
  definition and at least one manifestation.
- **The grid is honest**: `na` needs a non-blank `na_reason`; `roadmap`
  cells, and any cell with a `roadmap` measure, need a `review_by` ISO date;
  `by_design` measures need a `source`; `assessed` with every measure `org`
  contradicts itself.
- **The grid is written, not filled**: two cells with >=85% similar
  definitions trip the copy-paste detector.

Each rule has a case under
[`internal/core/testdata/`](internal/core/testdata/): an assessment
directory plus a `case.yaml` naming the exact error rules lint must raise.
Cases overlay the lint-clean 80-cell `valid_complete` base, so each one
changes only what its rule is about. `go test ./...` runs them all.

## GitHub Action

`action.yml` at the repo root wraps the CLI for consumers:

```yaml
- uses: macula-io/macula-fovea@main
  with:
    dir: security/fovea
    command: lint          # lint | score | render | issues
    args: ""               # extra flags, e.g. --check for issues
```

The action builds the binary with `setup-go` and runs `lint` with
`--github`, so findings appear as inline annotations on the assessment's
files. It then writes `score.json`, `scorecard.md` and `scorecard.html` into
`artifact-dir` (default `fovea-artifacts/`), even when lint fails, so a red
run leaves its report behind, and appends the HTML scorecard to the job
summary. Inputs reach the shell only through environment variables. Upload
the artifacts with:

```yaml
- uses: actions/upload-artifact@v4
  if: always()
  with:
    name: fovea-artifacts
    path: fovea-artifacts/
```

## Build and test

```bash
cd cli
go build -o fovea ./cmd/fovea
go test -race ./...
```

The Go version is pinned in `.tool-versions` at the repo root. CI runs
gofmt, `go vet`, the race-enabled tests and the build on every push and
pull request.

## Example (against the dogfood skeleton)

```text
fovea lint assessments/macula-mesh-realm
fovea lint: macula-mesh-realm (96 cells expected)
  error acquire.accountability             missing cell (run: fovea init)
  ...
  error fovea.yaml                         owner is unassigned
  ...
  97 error(s), 0 warning(s)
```

That red run is the contract working: the dogfood skeleton has one of its
96 cells and nobody has claimed it.
