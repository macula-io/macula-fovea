# 13 — The scorecard (v0.3)

The scorecard is the only derivative artifact: it exists so that the matrix
can be read in one page, and so CI can read it in one invocation.

## Semantics

Per cell, a RAG state is computed from `status`:

| Status | RAG |
|---|---|
| `assessed` | Green |
| `assumed` | Amber |
| `roadmap` | Amber, flagged with its `review_by` |
| `na` | Grey |
| `unassessed` | Red; lint has already failed the assessment |

Roll-ups are computed along both axes:

- **per attribute** — how is confidentiality, integrity, … covered across
  all columns?
- **per column family** — how are actors, lifecycle, data state, environment
  covered across the core-five attributes?

## v0.3 change: the coverage-aware grid

v0.2 printed a bare RAG letter per attribute×family block, computed as the
**worst** cell RAG in the block. That made one thing unambiguous (any
unfinished cell poisons its block) and one thing unreadable (a block that is
red because one cell is unwritten looks identical to a block that is red
because nothing was ever attempted).

v0.3 keeps the worst-RAG rule as an **invariant** — it is what makes the
scorecard incapable of lying green — and adds two layers of information that
v0.2 implied but never showed:

1. **Per-block coverage count.** Each grid cell reads `RAG authored/total`,
   e.g. `R 2/6`: the block is red, and 2 of its 6 cells are written. The
   red letter still means what it always meant; the fraction explains why.
2. **Open-gaps section.** After the grid, every unfinished item is named:
   missing cells, `unassessed` cells, unjustified `na`s, overdue roadmaps.
   Gap information moves from "implied by a red wall" to "stated as a list"
   — the list is what CI archives, and it is the honest answer to "what
   remains".

## Headline metrics (unchanged from v0.2)

| Metric | Definition | Purpose |
|---|---|---|
| `pct_unassessed` | unassessed cells / total cells | 0.0 at "final"; this *is* the completion test. |
| `na_unjustified` | `na` cells with missing or empty `na_reason` | Must be 0. The anti-theater tripwire. |
| `pct_by_design` | measures tagged `by_design` / total measures | Keeps "everything is roadmap/org" visible as a posture, not a confession buried in prose. |
| `oldest_review_by` | earliest (i.e. most overdue) `review_by` in any roadmap cell | Nothing rots silently. |

## Presentation

`fovea render --format md` prints the v0.3 grid and open-gaps section;
`fovea render --json` includes `grid_v03`, `coverage` and `open_gaps` so CI
can archive them without parsing markdown; `fovea score --json` carries the
headline metrics unchanged.

## Anti-theater as lint

The following are lint-level failures, not conventions:

- two cells with threat definitions >85% similar (copy-paste detection);
- any cell with zero manifestations;
- a `roadmap` cell without `review_by`, or an `assumed` cell without an
  `owner` attribution;
- a `na` cell without `na_reason`;
- a cell whose measures are *all* `org` when the status is `assessed` —
  if the product does nothing here, the cell is `na`, not `assessed`.

The lint is the framework. Everything else in this directory is
documentation *for* the lint.
