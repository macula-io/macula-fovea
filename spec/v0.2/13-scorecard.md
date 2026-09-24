# 13 — The scorecard

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

## Headline metrics

| Metric | Definition | Purpose |
|---|---|---|
| `pct_unassessed` | unassessed cells / total cells | 0.0 at "final"; this *is* the completion test. |
| `na_unjustified` | `na` cells with missing or empty `na_reason` | Must be 0. The anti-theater tripwire. |
| `pct_by_design` | measures tagged `by_design` / total measures | Keeps "everything is roadmap/org" visible as a posture, not a confession buried in prose. |
| `oldest_review_by` | earliest (i.e. most overdue) `review_by` in any roadmap cell | Nothing rots silently. |

## Presentation

`fovea render --format md` produces a Markdown scorecard (grid of RAG
letters with per-family subtotals); `fovea render --format json` produces
the machine-readable version for dashboards and CI annotation;
`fovea render --format pdf` is the deliverable for humans in conference
rooms.

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
