# Reading a fovea matrix (in five minutes)

You are a reviewer, an auditor, or the next tech lead. Here's how to get the
shape of an assessment without reading all of it.

## 1. `fovea.yaml` — the regime statement

Top of the directory. It tells you which attributes are in force (core five,
plus any Parkerian extensions) and which columns the system chose for itself.
Every cell file and the scorecard inherit from this. If the regime looks wrong
(for example `possession` disabled on a system likely to be physically seized),
the first challenge goes to the header, not the cells.

## 2. `trust-anchors.md` — "what breaks if broken?"

Second read. If an entry on this register is not covered by a proper ceremony
elsewhere, the entire assessment inherits a broken anchor — every `by_design`
claim in every cell now depends on a named weakness. The register must be
*complete* (everything the system decries as a root is there) and *conservative*
(nothing smaller than total-scope entries).

## 3. The scorecard

Top-level summary, computed. Scan in this order:

1. **Headline metrics** — `pct_unassessed` must be zero at "final".
   `na_unjustified` must be zero. If either isn't, the assessment isn't done
   regardless of what the prose says.
2. **Column-family roll-ups** — disproportionate red/amber in `decommission`
   is a classic discovery; it means renovation, not certification.
3. **`pct_by_design`** — a low number is not bad; "engineered fact is small,
   obligation is large" is frequently the honest shape. What is bad is a
   high number of `by_design` claims without sources; audit three, then four.

## 4. The cells — read the amber ones first

You do not have to read 85 cells. The scorecard's roll-up tells you where
the risk lives. Read:

- every `roadmap` cell, checking `review_by` against today;
- every `na` cell, checking the reason against your own judgment — a lazy N/A
  is the most frequent lie;
- any cell the coverage roll-up suggests is load-bearing (e.g. anything in
  `admin` for a system with a CA).

## 5. The drill for auditors

Three randomized drills expose a template-theater assessment:

| Drill | How |
|---|---|
| Duplicate threat | Ask for the two most *similar* cells. If they share a definition, it was filled, not written — lint should have caught it. |
| Owner attribution | Pick a cell, ask "who owns this?" If the answer is the author of the cell, not the owner, the framework was used as a diary. |
| Anchor coverage | Pick a random trust anchor; ask, "is its compromise reversible in one quarter?" If the answer is "yes," the register has the wrong candidates. |

The matrix is therefore not a list of findings; it is a way to find the
finding.
