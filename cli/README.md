# fovea (CLI)

Single static Go binary implementing the [fovea spec v0.2](../spec/v0.2/00-overview.md)
against an **assessment directory** (header + cells).

## Commands

| Command | What it does | Exit non-zero when |
|---|---|---|
| `fovea init <dir>` | Reads `fovea.yaml`, computes the declared grid, creates every missing `<column>.<attribute>.yaml` as an `unassessed` skeleton. Refuses to run while the header itself has lint errors (no grid for an unowned assessment). | header invalid |
| `fovea lint <dir>` | Header checks + grid-closure ("every expected cell exists") + per-cell rule enforcement + copy-paste detection (≥85% definition similarity). | any error finding |
| `fovea score <dir> [--json]` | Computes the [scorecard](../spec/v0.2/13-scorecard.md): expected/present/missing cells, `pct_unassessed`, `na_unjustified`, `pct_by_design`, oldest `review_by` and overdue roadmaps, cells-per-attribute and per-family. Missing cells count as unassessed — closure is scored, not just linted. | lint had errors |
| `fovea render <dir> [--json]` | Markdown scorecard grid: attributes × the four column families, each block shown as the **worst** RAG inside it (Red > Amber > Grey > Green). | n/a |

Flags note: Go's `flag` stops at the first positional; pass `dir` after the
command, not between flags and nothing else.

## Anti-theater rules enforced (`lint`)

Rules are the point, not a style guide (spec 00-overview):

- **The grid is closing** — every expected cell must exist.
- **No empty answers** — non-`na` cells need a definition and ≥1
  manifestation.
- **The grid is honest** — `na` requires `na_reason`; `roadmap` requires a
  dated `review_by`; `by_design` measures require a `source`;
  `assessed` with every measure `org` contradicts itself.
- **The grid is written, not filled** — two cells with ≥85% similar
  definitions trip the copy-paste detector.
- **No invisible columns** — cell IDs must resolve against columns/attributes
  declared in the header.

## Build

```bash
cd cli
go build -o fovea ./cmd/fovea
```

## Example (against the dogfood skeleton)

```text
fovea lint assessments/macula-mesh-realm
  error fovea.yaml                 owner is unassigned
```

That error is the contract working: the dogfood skeleton deliberately
refuses to be claimed.
