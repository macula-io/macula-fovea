# 14 — Instantiation protocol

The matrix does not start at the first cell. Four artifacts, **in order**,
because the fourth cannot mean what it means without the first three.
The protocol exists so that two assessors — or an assessor and an auditor
six months later — reach the same shape.

## Step 1 — Trust-anchor register

`trust-anchors.md`. The named set of things whose compromise makes every
other defense meaningless. For a typical governed mesh: the root CA key and
its ceremony; the org CA keys; the build-and-release pipeline key; the
attestation root (TPM vendor cert chain) if attestation exists. **The rule:**
if a thing's compromise would require rebuilding the system, it's on the
list. If it isn't on the list, the cells' `by_design` claims fail at
review — they imply an unstated anchor.

## Step 2 — System landscape

`landscape.md`. One diagram, as small as possible, naming every process,
store, link, and actor the assessment covers. The test is bidirectional:
every column of the matrix must locate at least one element on the
diagram, and every element must be under at least one column. Elements in
the diagram that are out of scope are drawn dashed and explicitly listed
as excluded **with a reason**.

## Step 3 — Matrix header (`fovea.yaml`)

Commit to the regime *before* any cell exists: which attributes are core
(and therefore must exist), which extensions are enabled, which columns
(including any `x_` extensions) are declared. The header is what the scorecard
uses to know how many cells to expect — 16 columns × 5 core attributes =
80 base cells, plus 16 per enabled extension.

## Step 4 — Cells

Fill the grid. Fill it in attribute-major or column-major order; do not
make the mistake of writing only where fear is highest — the *reason* there
is a grid. Fill `unassessed` cells eiter by answering or by declaring `na`
with reason. Do not leave placeholders without status updates; the cell
template enforces every field.

## Step 5 — Scorecard

`fovea score`, then `fovea render`. Review the roll-ups: any
column-family/attribute quadrant that is disproportionately red or amber
is where the next cycle's effort goes — the scorecard's only job is to make
that visible to whoever decides.

## Anti-patterns this protocol exists to kill

| Anti-pattern | Why |
|---|---|
| Threat model written directly by the product's own authors, with no landscape review | The landscape is the mechanism for arguing over *what is being defended*. |
| Cells filled with the same boilerplate, cell after cell | Copy-paste is the death of assessment value; lint catches it. |
| "We'll cover that later" | `roadmap` + `review_by` *is* "later"; there's no unwritten later. |
| Trust anchors never named | Then `by_design` was never defined, and the cells lied politely. |
