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

| Step | Deliverable | Dependent on |
|---|---|---|
| v0.2 | This spec | — |
| v0.3 | The grid *closure* becomes enforceable: built-in extensions beyond possession/utility as sanctioned options | Evidence from ≥2 assessments that one extension is the same shape everywhere |
| v0.4 | The scorecard gets a *conformance class*: "level 1 = zeros on lint", "level 2 = no unassigned owners", "level 3 = no unjustified N/A" | enough assessments to see one's own cultural failure modes |

## The future repo boundary

Anything the spec says that the code doesn't enforce yet is *pending*. The
CLI will be developed in this repo; no mesh-served surface (`mcl-fovea`)
until two real assessment directories pass the lint without being modified
to pass.
