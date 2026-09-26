# 00 — Overview

Fovea is a framework for producing **complete, honest security assessments**
of a system. It is deliberately small: a taxonomy (what to ask), a cell
schema (how to answer), and a scorecard (how to read the result). Everything
else — likelihood modeling, kill chains, controls catalogues — is delegated
to established methods referenced in [mappings/](../mappings/README.md).

## v0.3 delta

This version differs from v0.2 in exactly one normative place: the scorecard
(13-scorecard) gains a coverage-aware grid (per-block `authored/total` next
to the unchanged worst-RAG letter) and an open-gaps section. The worst-RAG
invariant, the headline metrics, the cell schema, the axes, and the
attributes are untouched. Assessments written under 0.2 keep their 0.2
reading; opting into 0.3 is a one-line header change.

## Philosophy

*Completeness through forced explicitness.* The framework's value is not in
what it tells you about security; it is in making it structurally impossible
to finish an assessment while a whole region of the threat space was never
considered. Three mechanisms do the work:

1. **The grid is closing.** Every combination of column × attribute must,
   at completion, be either filled, or marked `na` **with a written reason**.
   An empty cell is a lint failure, not a blank line.
2. **Claims are typed.** Content is separated from confidence: statuses
   (`assessed / assumed / roadmap / na`) and measure tags (`by_design /
   roadmap / org`) make engineered fact distinguishable from intent, and
   intent distinguishable from other people's obligations.
3. **The artifacts begin outside the code.** The trust-anchor register and
   the landscape come *first* (see [14-instantiation](14-instantiation.md)),
   so the matrix is anchored to named, adjudicable things rather than to an
   assessor's mental model.

## Artifact set

| Artifact | Required by | Purpose |
|---|---|---|
| Trust-anchor register | 14-instantiation | The named root-of-trust set whose compromise invalidates all cells' `by_design` claims. |
| System landscape | 14-instantiation | The diagram the matrix indexes. |
| Matrix (cells) | cell schema | One YAML file per `column.attribute`. |
| Scorecard | 13-scorecard | Computed rollup; CI-consumable. |

## Status taxonomy

| Status | Meaning | Allowed at "final"? |
|---|---|---|
| `unassessed` | Cell exists; nobody has answered it yet. | No — lint fails. |
| `assumed` | Answer drafted from documentation/design; not verified against a running system. | Yes, disclosed on the scorecard. |
| `assessed` | Answer verified by inspection of code, configuration, or test. | Yes. |
| `roadmap` | The threat is accepted as real; the defense does not yet exist and is scheduled. | Yes, disclosed, must have `review_by`. |
| `na` | Cell not applicable; **must** carry `na_reason`. | Yes, disclosed. |

## Anti-theater rules

Linting (see 13-scorecard) fails an assessment when:

- two cells share a threat definition beyond a similarity threshold —
  copy-paste is the primary symptom of a workbook that was filled, not
  written;
- a cell has zero manifestations;
- a `roadmap` cell lacks `owner` or `review_by`;
- a cell's status is `assessed` but all its measures are `org`.

## Versioning

See [spec/README.md](../README.md). This is **v0.3**. The taxonomy is
therefore frozen at release and evolves only through numbered versions —
decisions about axes and attributes are made in spec PRs, not in prose
debates inside individual assessments.
