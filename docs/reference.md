# Reference — statuses, rules, flags, pitfalls

Everything the lint enforces, in one place. Normative: spec/v0.3.

## Cell statuses

| Status | Meaning | Lint demands |
|---|---|---|
| `unassessed` | Nobody has answered the cell yet | always fails: answer it or declare `na` |
| `assumed` | Answer drafted from docs/design, not verified | definition, manifestations; renders amber |
| `assessed` | Verified by inspection of code, config, or test | cannot be all-`org` measures; `by_design` measures need `source` |
| `roadmap` | Gap accepted as real, defense scheduled | `review_by` (ISO date), definition, manifestations |
| `na` | Not applicable | `na_reason` (written, not blank) |

## Measure statuses (claim strength)

| Status | Meaning | Lint demands |
|---|---|---|
| `by_design` | Guaranteed by the architecture itself | non-empty `source` |
| `roadmap` | Agreed, scheduled, does not exist yet | cell carries `review_by`, whatever its status |
| `org` | The organization's job, not the product's | cell carries an `owner` |

## RAG rendering

🟢 `assessed` · 🟡 `assumed`/`roadmap` · 🔴 `unassessed`/missing · ⚪ `na`
(all-N/A blocks show `- n/n`). Block letter = worst of its cells, **skipping
justified N/As**: those never drag a block down, while unfinished cells
(including an `na` without a reason) always do.

## Lint rules (failures, not style)

Every finding carries a rule code. Each code has a case under
`cli/internal/core/testdata/<code>/` (the cases seed the conformance suite).

| Rule code | Fails when |
|---|---|
| `header_version_unknown` | `fovea:` is not a spec version the CLI knows (0.2, 0.3) |
| `header_owner_unassigned` | header `owner` is empty or `unassigned` (any case, any spacing) |
| `grid_missing_column` | a spec column is absent from the header |
| `grid_column_wrong_family` | a spec column is declared under another family |
| `grid_column_unknown` | a column is neither a spec column nor `x_`-prefixed |
| `grid_column_duplicate` | a column is declared twice |
| `grid_family_unknown` | `columns:` has a key other than actors, lifecycle, data, environment |
| `grid_core_attribute_missing` | one of the core five is absent from `attributes.core` |
| `grid_core_attribute_invalid` | an extension is listed under `core` |
| `grid_attribute_unknown` | an attribute (core, enabled, or justification key) is not in the spec |
| `grid_attribute_duplicate` | an attribute is listed twice |
| `grid_extension_unjustified` | `possession` or `utility` is neither enabled nor justified (blank is not a justification) |
| `grid_extension_contradiction` | an extension is enabled and also justified as disabled |
| `cells_dir_unreadable` | `cells_dir` does not exist or cannot be read |
| `cell_parse` | a cell file is not valid YAML for the cell shape |
| `cell_id_filename_mismatch` | the file name is not `<id>.yaml` |
| `grid_missing_cell` | a declared column x attribute has no cell |
| `cell_id_undeclared` | a cell id names a column or attribute the header does not declare |
| `cell_status_unknown` | status is not one of the five |
| `cell_unassessed` | a present cell is still `unassessed` |
| `cell_owner_unassigned` | cell `owner` is empty or `unassigned` (any case, any spacing) |
| `definition_empty` | an `assumed`, `assessed` or `roadmap` cell has no threat definition |
| `manifestations_empty` | an `assumed`, `assessed` or `roadmap` cell has zero manifestations |
| `na_without_reason` | an `na` cell has an empty or whitespace-only `na_reason` |
| `roadmap_without_review_by` | a `roadmap` cell has no `review_by` |
| `roadmap_measure_without_review_by` | any cell with a `roadmap` measure has no `review_by` |
| `review_by_not_iso_date` | a `review_by` is present but not `YYYY-MM-DD` |
| `measure_status_unknown` | a measure status is not `by_design`, `roadmap` or `org` |
| `by_design_without_source` | a `by_design` measure has no `source` |
| `assessed_all_org` | an `assessed` cell whose measures are all `org` |
| `definition_copy_paste` | two threat definitions are >=85% similar (word-set Jaccard) |

An overdue `review_by` is not a lint error; the scorecard reports it
(`overdue_roadmap`, `oldest_review_by`, and the v0.3 open gaps).

## CLI

```
fovea init   <dir>                      # generate missing skeletons (never overwrites)
fovea lint   <dir> [--github]           # all rules above; --github emits PR annotations
fovea score  <dir> [--json]             # headline metrics (exit 1 if lint has errors)
fovea render <dir> [--format md|html|json]  # scorecard; --html / --json are short forms
fovea issues <dir> [--dry-run] [--check] [--repo o/r] [--token t]
```

Results go to stdout, diagnostics to stderr: `fovea score --json dir >
score.json` is valid JSON even on a failing run.

Metrics: `expected/present/missing cells`, `pct_unassessed`,
`na_unjustified`, `pct_by_design`, `oldest_review_by`, `overdue_roadmap`,
per-attribute and per-family counts.

## Pitfalls (all of these have bitten real assessments)

1. **YAML `: ` trap.** A plain scalar containing colon+space parses as a
   mapping: quote the string or use a `>-` block scalar. Manifestations and
   `source:` lines are the usual victims.
2. **`owner: unassigned` is a lint error, by design.** Claim the header and
   every written cell before expecting a clean run.
3. **Cell id = filename.** `cells/in_motion.confidentiality.yaml` must
   contain `id: in_motion.confidentiality` — mismatch is a lint error.
4. **Spec version in the header.** `fovea: "0.3"` — 0.2 headers get the
   frozen 0.2 scorecard; opt in deliberately.
5. **`na` cells keep empty definitions.** Lint tolerates an empty threat
   on an `na` cell: the `na_reason` is the content, and it must be real
   text, not whitespace.
6. **Don't chase grey.** Justified N/A is a correct terminal state; writing
   cells for surfaces that don't exist is how templates start lying.
