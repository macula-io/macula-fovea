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
      evidence:                    # v0.4: optional on any measure; see "Evidence"
        - kind: probe
          claim: kx_post_quantum_only
          probe: kx_group
          version: 1
          target: fleet_stations   # declared in the header's targets
          expect:
            accepted: [secp384r1_mlkem1024, secp256r1_mlkem768]
            refused: [x25519, secp256r1, secp384r1]
        - kind: test
          ref: native/probe/src/lib.rs
  recovery:                        # how we restore after it happens
    - measure: ...
      status: roadmap
review_by: 2027-03-31              # ISO date; required for roadmap status
# na_reason: required iff status: na; on an answered cell it may carry the
# hard-rule-1 argument that detection is structurally impossible
na_reason: >
  ...
notes: >
  ...
```

> **Erratum (2026-09-26).** The `na_reason` comment above read only
> "required iff status: na", which contradicted hard rule 1 below: an
> answered cell (`assumed`, `assessed`, `roadmap`) may leave detection empty
> when its `na_reason` argues that detection is structurally impossible.
> `na_reason` is required iff the status is `na`; on an answered cell it is
> optional and carries that argument.

## Measure statuses (claim strength)

| Status | Meaning | Rule |
|---|---|---|
| `by_design` | The defense is guaranteed by the architecture itself — not by policy, configuration, or habitual practice. | Requires `source`. |
| `roadmap` | The defense is agreed and scheduled; it does not exist yet. | Requires `review_by` on the *cell*. |
| `org` | The defense is outside the product's scope: an organization must do it (key ceremonies, exit procedures, monitoring staffing). The framework doesn't pretend the product saves you here. | Requires `owner`. |

## Evidence (v0.4)

A measure may list evidence: what shows the measure is real. Four kinds:

| Kind | Fields | What it is |
|---|---|---|
| `doc` | `ref` | A citation: a document, a README section, a design. Not executable. |
| `test` | `ref` | A test that runs in CI: a path, or a path and a test name. |
| `scenario` | `ref`, `runner` | An executable scenario (Gherkin): `ref` names the feature file and scenario; `runner` is `godog`, `whitebread` or `cucumber`. |
| `probe` | `claim`, `probe`, `version`, `target`, `expect` | A check an observer runs against the running system (16-probes). |

**`assessed` needs executable evidence.** An `assessed` cell carries at least
one `test`, `scenario` or `probe` on one of its measures. `doc` evidence alone
leaves a cell `assumed`. The lint checks that the evidence is declared; that a
test or scenario passed is CI's business, and that a probe's claim holds is
the observer's (15-observations). v0.4 does not let CI or an observer change a
cell's authored status.

**A probe declaration** names:

- `claim`: the claim's stable id, `[a-z][a-z0-9_]*`, unique in the
  assessment. Every observation of the claim carries it.
- `probe` and `version`: a probe the registry (16-probes) defines.
- `target`: a target the header declares (14-instantiation), of the kind the
  probe observes.
- `expect`: what the probe must see, in its own vocabulary (16-probes). A
  group expected both ways, or a group the probe does not know, is a lint
  error, and so is an empty expectation.

A probe declares a claim about a `by_design` measure in practice, but the
lint does not require it: the claim's evidence is its observations.

## The two hard rules

1. **No empty answers.** Detection may be empty **only** with
   `na_reason` saying why detection is structurally impossible for this
   threat (some are; you have to argue it).
2. **No invisible columns.** The file's `id` must resolve to a column and
   an attribute declared in the enclosing assessment's header. A cell that
   dot-joins invented words is a lint error.

## Status propagation (informative)

The scorecard's RAG roll-up (13-scorecard) reads only `status`, and for
`na` whether `na_reason` is written; nothing else in the cell affects
roll-up. If a cell contains both `by_design` and
`roadmap` measures, the *cell* status remains `assumed` or `assessed` — the
honesty lives inside the measures, so the headline number isn't faked by
tone either.

> **Erratum (2026-09-26).** This section said the scorecard reads only
> `status` and `review_by`. The headline metrics read more: `pct_by_design`
> counts measure statuses and `na_unjustified` reads `na_reason`. The
> sentence now describes the roll-up, which is what it meant.
