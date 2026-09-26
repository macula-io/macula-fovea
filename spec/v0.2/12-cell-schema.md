# 12 — The cell schema

One file per cell, YAML, linted by the `fovea` CLI, which is the reference
implementation of this spec.

> **Erratum (2026-09-26).** This section said cell files are "validated
> against `schema/cell.schema.json` after YAML to JSON conversion". Nothing
> loaded those JSON Schemas, and they contradicted the CLI (the header
> schema accepted only `fovea: "0.2"`; `fovea init` skeletons failed the
> cell schema), so they were removed. The `fovea` CLI's `lint` is the
> reference implementation of these rules; a language-neutral conformance
> suite follows it.

## Required structure

```yaml
id: in_motion.confidentiality      # <column>.<attribute>, both declared in the header
status: assumed                    # unassessed | assumed | assessed | roadmap | na
owner: alice@example.org           # lint fails on the literal string "unassigned"
threat:
  definition: >
    What this adversary, in this phase, with this data state and under this
    condition, does to this attribute.
  manifestations:
    - Concrete, observable ways the threat appears.
    - Must not be empty; two cells may not share a definition >85% similar.
defense:
  detection:                       # how do we know it's happening?
    - measure: >
        What the measure is.
      status: org                  # by_design | roadmap | org
      source: >                    # evidence: doc link, test, code path (required for by_design)
        ...
  countermeasures:                 # how we stop or repel it
    - measure: ...
      status: by_design
      source: ...
  recovery:                        # how we restore after it happens
    - measure: ...
      status: roadmap
review_by: 2027-03-31              # ISO date; required for roadmap status
na_reason: >                       # required iff status: na
  ...
notes: >
  ...
```

## Measure statuses (claim strength)

| Status | Meaning | Rule |
|---|---|---|
| `by_design` | The defense is guaranteed by the architecture itself — not by policy, configuration, or habitual practice. | Requires `source`. |
| `roadmap` | The defense is agreed and scheduled; it does not exist yet. | Requires `review_by` on the *cell*. |
| `org` | The defense is outside the product's scope: an organization must do it (key ceremonies, exit procedures, monitoring staffing). The framework doesn't pretend the product saves you here. | Requires `owner`. |

## The two hard rules

1. **No empty answers.** Detection may be empty **only** with
   `na_reason` saying why detection is structurally impossible for this
   threat (some are; you have to argue it).
2. **No invisible columns.** The file's `id` must resolve to a column and
   an attribute declared in the enclosing assessment's header. A cell that
   dot-joins invented words is a lint error.

## Status propagation (informative)

The scorecard (13-scorecard) reads only `status` and `review_by`; nothing
else in the cell affects roll-up. If a cell contains both `by_design` and
`roadmap` measures, the *cell* status remains `assumed` or `assessed` — the
honesty lives inside the measures, so the headline number isn't faked by
tone either.
