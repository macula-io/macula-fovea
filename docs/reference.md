# Reference — statuses, rules, flags, pitfalls

Everything the lint enforces, in one place. Normative: spec/v0.3.

## Cell statuses

| Status | Meaning | Lint demands |
|---|---|---|
| `unassessed` | Nobody has answered the cell yet | fails until written |
| `assumed` | Answer drafted from docs/design, not verified | nothing extra — but it renders amber |
| `assessed` | Verified by inspection of code, config, or test | cannot be all-`org` measures; `by_design` measures need `source` |
| `roadmap` | Gap accepted as real, defense scheduled | `review_by` (ISO date) |
| `na` | Not applicable | `na_reason` (written) |

## Measure statuses (claim strength)

| Status | Meaning | Lint demands |
|---|---|---|
| `by_design` | Guaranteed by the architecture itself | non-empty `source` |
| `roadmap` | Agreed, scheduled, does not exist yet | cell carries `review_by` |
| `org` | The organization's job, not the product's | cell carries an `owner` |

## RAG rendering

🟢 `assessed` · 🟡 `assumed`/`roadmap` · 🔴 `unassessed`/missing · ⚪ `na`
(all-N/A blocks show `- n/n`). Block letter = worst of its **non-N/A**
cells — justified N/As never drag a block down, unfinished cells always do.

## Lint rules (failures, not style)

- grid closure: every declared column × attribute must have a cell
- no invisible columns: cell `id` parts must be declared in the header
- zero manifestations on a non-`na` cell
- empty threat definition on a non-`na` cell
- `owner: unassigned` anywhere (cells and header)
- `na` without `na_reason`, `roadmap` without `review_by`
- `by_design` measure without `source`
- `assessed` cell whose measures are all `org`
- two threat definitions ≥85% similar (copy-paste detector)
- unknown status on a cell or measure

## CLI

```
fovea init  <dir>                 # generate missing skeletons
fovea lint  <dir> [--github]      # validate + anti-theater; --github emits PR annotations
fovea score <dir> [--json]        # headline metrics (exit 1 if lint has errors)
fovea render <dir> [--json|--html]# scorecard: markdown, JSON, or job-summary HTML
```

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
5. **`na` cells keep empty definitions.** The schema tolerates an empty
   threat on an `na` cell — the `na_reason` is the content.
6. **Don't chase grey.** Justified N/A is a correct terminal state; writing
   cells for surfaces that don't exist is how templates start lying.
