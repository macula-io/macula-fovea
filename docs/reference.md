# Reference — statuses, rules, flags, pitfalls

Everything the lint enforces, in one place. Normative: spec/v0.4.

## Cell statuses

| Status | Meaning | Lint demands |
|---|---|---|
| `unassessed` | Nobody has answered the cell yet | always fails: answer it or declare `na` |
| `assumed` | Answer drafted from docs/design, not verified | definition, manifestations, measures, detection (or an argued `na_reason`); renders amber |
| `assessed` | Verified, and re-checkable | as `assumed`, not all-`org` measures, and (v0.4) at least one `test`, `scenario` or `probe` evidence on a measure |
| `roadmap` | Gap accepted as real, defense scheduled | as `assumed`, plus `review_by` (ISO date) |
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

Every finding carries a rule code and a location (`where`): `fovea.yaml`
for header findings, the file name for a cell file that cannot be tied to
an id, the cell id otherwise. Each code has a case under
`cli/internal/core/testdata/<code>/`, whose `case.yaml` lists the exact
`(rule, where)` errors expected (the cases seed the conformance suite; the
format is described in `cli/README.md`).

| Rule code | Fails when |
|---|---|
| `header_version_unknown` | `fovea:` is not a spec version the CLI knows (0.2, 0.3, 0.4) |
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
| `cells_dir_unreadable` | `cells_dir` does not exist or cannot be read (reported at `fovea.yaml`) |
| `cell_parse` | a cell file is not valid YAML for the cell shape |
| `cell_id_filename_mismatch` | the file name is not `<id>.yaml` |
| `grid_missing_cell` | a declared column x attribute has no cell |
| `cell_id_undeclared` | a cell id names a column or attribute the header does not declare |
| `cell_status_unknown` | status is not one of the five |
| `cell_unassessed` | a present cell is still `unassessed` |
| `cell_owner_unassigned` | cell `owner` is empty or `unassigned` (any case, any spacing) |
| `definition_empty` | an `assumed`, `assessed` or `roadmap` cell has no threat definition |
| `manifestations_empty` | an `assumed`, `assessed` or `roadmap` cell has zero manifestations |
| `measures_empty` | an `assumed`, `assessed` or `roadmap` cell has no detection, countermeasure or recovery measure |
| `detection_empty` | such a cell has measures but no detection measure, and no `na_reason` arguing detection is structurally impossible (spec 12 hard rule 1) |
| `na_without_reason` | an `na` cell has an empty or whitespace-only `na_reason` |
| `roadmap_without_review_by` | a `roadmap` cell has no `review_by` |
| `roadmap_measure_without_review_by` | any cell with a `roadmap` measure has no `review_by` |
| `review_by_not_iso_date` | a `review_by` is present but not `YYYY-MM-DD` |
| `measure_status_unknown` | a measure status is not `by_design`, `roadmap` or `org` |
| `by_design_without_source` | a `by_design` measure has no `source` |
| `assessed_all_org` | an `assessed` cell whose measures are all `org` |
| `definition_copy_paste` | two threat definitions are >=85% similar (word-set Jaccard); reported on both cells of the pair |

v0.4 rules (evidence, probe declarations, targets, policy; spec 12, 14, 16):

| Rule code | Fails when |
|---|---|
| `field_needs_v0_4` | a v0.2 or v0.3 assessment uses `evidence`, `targets` or `policy` (refused, not ignored) |
| `evidence_kind_unknown` | an evidence `kind` is not `doc`, `test`, `scenario` or `probe` |
| `evidence_ref_empty` | `doc`, `test` or `scenario` evidence has no `ref` |
| `evidence_runner_unknown` | `scenario` evidence names a runner other than `godog`, `whitebread` or `cucumber` |
| `assessed_without_executable_evidence` | an `assessed` cell has no `test`, `scenario` or `probe` evidence on any measure |
| `probe_unknown` | a probe and version the spec's registry (16-probes) does not define |
| `probe_target_undeclared` | a probe names a target the header does not declare, or one of another kind |
| `probe_expectation_invalid` | an expectation names nothing, a group outside the probe's vocabulary, a group both ways, or refusals with no accepted group beside them |
| `claim_id_invalid` | a probe's `claim` is not `[a-z][a-z0-9_]*` |
| `claim_id_duplicate` | two probe declarations claim the same id; reported at each |
| `target_kind_unknown` | a target's `kind` is not `macula_station` |
| `target_address_invalid` | a target names no station, or a station address is not a unicast IP literal and port in canonical form (`192.0.2.10:4433`, `[2001:db8::10]:4433`), or an address appears twice in the header |
| `target_node_id_invalid` | a station's `node_id` is not 64 lowercase hex digits |
| `header_system_invalid` | a v0.4 header's `system` is not `[a-z][a-z0-9_.-]*` (observations carry it in their record's subject) |
| `policy_missing` | the assessment declares probes but no `policy` |
| `policy_observer_invalid` | the assessment declares probes and `policy.observers` names none, or an observer is not a node id (64 lowercase hex digits), or an observer is a declared station |
| `policy_publish_unknown` | `policy.publish` is not `every_result` or `state_changes` |
| `policy_cadence_invalid` | `policy.cadence` is not an ISO 8601 duration (days to seconds) from a minute to seven days |
| `policy_suspended_unknown_claim` | `policy.suspended` names a claim no probe declares |

An overdue `review_by` is not a lint error; the scorecard reports it
(`overdue_roadmap`, `oldest_review_by`, and the v0.3 open gaps).

## CLI

```
fovea init   <dir>                      # generate missing skeletons (never overwrites)
fovea lint   <dir> [--github]           # all rules above; --github emits PR annotations
fovea score  <dir> [--json]             # headline metrics (exit 1 if lint has errors)
fovea render <dir> [--format md|html|json]  # scorecard; exit 1 if lint has errors
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
4. **Spec version in the header.** `fovea: "0.4"` — 0.2 headers get the
   frozen 0.2 scorecard, 0.3 and 0.4 the coverage-aware one; 0.4 adds
   evidence, probes, targets and a policy, and an older header refuses those
   fields. Opt in deliberately.
5. **`na` cells keep empty definitions.** Lint tolerates an empty threat
   on an `na` cell: the `na_reason` is the content, and it must be real
   text, not whitespace.
6. **Don't chase grey.** Justified N/A is a correct terminal state; writing
   cells for surfaces that don't exist is how templates start lying.
