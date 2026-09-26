# Maintaining the fovea spec

The spec, not any assessment, is what gets versioned.

## Rules of the game

1. **No taxonomy debates in assessments.** When a needed column is discovered
   in the field, open a spec PR. The field's assessment marks the cell `na`
   with a reason referencing the spec PR — never an `x_` declaration that is
   really a spec bug.
2. **Versions are forked, not branched.** A version bump creates a new
   directory `spec/vN.M/`; older versions remain normative for assessments
   they issued, forever. No retroactive amendments.
3. **Extensions are the death of spec, cinnamon of assessments.** If an `x_`
   appears in more than two assessments independently, it wants to be a core
   column — promote it in the next spec version.

## What counts as a version bump

| Change | Bump? |
|---|---|
| Typo, clarifying example, additional anti-pattern | No |
| Renaming a column family (the four families) | No — the families are structural, names may be stable across versions |
| Adding/removing/renaming a *column* or *attribute* | Yes |
| Adjusting the cell schema in a way that breaks YAML files | Yes |
| New headline metric on the scorecard | Yes, minor-conservative |

## The roadmap ahead

| Step | Deliverable | State |
|---|---|---|
| v0.2 | Axes, attributes, cell schema, scorecard, instantiation protocol | released 2026-09-25, archival |
| v0.3 | Coverage-aware scorecard: `authored/total` per block and an open-gaps list | released 2026-09-25, current |
| v0.4 | Executable evidence on measures ([proposal](../spec/proposals/v0.4-evidence/README.md)) | proposed, not normative |

## Spec, reference implementation, conformance

The `fovea` CLI in `cli/` is the reference implementation: its `lint` is
what the spec means by "the lint". Every rule it enforces has a case under
`cli/internal/core/testdata/<rule>/`, an assessment plus the exact findings
expected, and those cases seed a language-neutral conformance suite. Any
other implementation (`mcl-fovea`, the mesh-served observer in
`macula-services`) must pass the same cases. A spec change that adds or
alters a rule lands with its case and the CLI change in the same PR.

Corrections to a released version are errata: marked as such in the text,
dated, and never a change of meaning (see `spec/README.md`).
