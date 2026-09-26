# Writing a fovea assessment

The instantiation protocol commits you to a **shape for the whole exercise
before the threat-space is entered**, which is precisely the point: shape
trumps bias.

## Phase 0 — Scope discipline

Before `fovea init`, write `scope.in` and `scope.out` in `fovea.yaml`. If you
cannot give a one-paragraph "out" — "vendor-employed firmware on the
attestation chip, their CAs on production SDRs, physical security of halls
managed by others" — you are about to fill in a grid that has no outer edge.

## Phase 1 — Trust anchors (with enclosure)

Collect the root-of-trust list. The standard mistake is to make it long; the
rule is *assessment-totalling*:

> If this breaks, is every other defense in this matrix revocable within one
> bounded change (re-issue certs, rotate one key), or is the whole system
> gone? Only the latter is an anchor.

Trusted anchors that are *partly* strong (HSMs that can be re-initialized on
compromise) still count; their tamper-readiness is a property of the anchor's
*ceremony*, not of the anchor as an object.

## Phase 2 — Landscape

One diagram. Every column of the matrix must point to something on it, and
every element must be pointed to by *at least* one column. The two-way test
is the only test that justifies the diagram.

If you find yourself drawing arrows to "trust" or "org," stop; the landscape
is about stations, directories, pipes, and real processes. "Trust" appears
only as a boundary, or as an anchor you already registered.

## Phase 3 — Header

Commit the regime: every extension enabled or justified as disabled (lint
fails a blank justification), all sixteen spec columns in their families,
custom `x_` columns if any
(this is the moment to declare `x_supply_chain` if you need it; you do not
need it for most meshes — `create` and `acquire` cover it), assessment date,
owner, team.

## Phase 4 — Cells

Fill the grid from the outside in: cold-field columns (`decommission`,
`socio_legal`, `temporal`) first. By the time you reach the columns you
already knew about (`operate`, `internal`), you will have written things you
otherwise would never have asked. Writing cells in fear-order writes the
threat-model you already had; writing them in matrix-order writes the one
you didn't.

## Phase 5 — Score and review

`fovea score` computes the headline metrics and `fovea render` the roll-up
with its open gaps; review it warm, three months
later, with someone who was not the writer. Any block that is *strictly*
green is either masterfully engineered or a working demonstration of
self-validation. Ask for the sources.

## What makes a good cell

| Dimension | Good | Bad |
|---|---|---|
| Definition | Names the actor, phase, data state, and propert in the cell's terms | A generic STRIDE statement duplicated across six cells |
| Manifestations | Observable, singular, specific | Threat restated as bullet |
| Countermeasures | `by_design` with a source artifact | "we encrypt data" |
| Recovery | One concrete action ("rotate cert, revoke via SWIM") | "be resilient" |
| Owner | A person or a team that can sign | "the team" |

## The lint is the framework

If you find yourself thinking "we'll fix the lint findings after the next
audit," rewrite the sentence in past tense: *we fixed it after this audit,
before the report*. Nothing else — no policy, no attestation, no firewall —
compensates for a workbook that has been filed in an illegitimate state.
