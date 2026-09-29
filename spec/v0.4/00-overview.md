# 00 — Overview

Fovea is a framework for producing **complete, honest security assessments**
of a system. It is deliberately small: a taxonomy (what to ask), a cell
schema (how to answer), and a scorecard (how to read the result). Everything
else — likelihood modeling, kill chains, controls catalogues — is delegated
to established methods referenced in [mappings/](../mappings/README.md).

## v0.4 delta

v0.3 made an assessment complete; v0.4 lets a claim in it be **re-checked**.
`assessed` used to be a claim with no re-check: a cell went green once, the
defense regressed, and the green stayed. v0.4 changes four normative places
and leaves the axes, the attributes and the scorecard (13) as v0.3 has them:

1. **Evidence on measures** (12-cell-schema): each measure may carry
   evidence of kind `doc`, `test`, `scenario` or `probe`.
2. **`assessed` needs executable evidence** (12-cell-schema): at least one
   `test`, `scenario` or `probe` on one of the cell's measures. A cell whose
   evidence is only cited is `assumed`, whatever its author believes.
3. **Targets and a publication policy in the header** (14-instantiation):
   what a probe observes, by IP address, and what an observer may publish
   about it.
4. **Observations** (15-observations, 16-probes): a probe declaration names
   a claim; an observer checks it on the running system and signs what it
   saw as a record anyone can verify offline against the realm key. The
   claim states are `holding`, `broken` and `unknown`.

Assessments written under 0.2 and 0.3 keep their reading, and the lint
refuses v0.4 fields in them rather than ignore them. Opting into 0.4 is a
header change plus, for each `assessed` cell, its executable evidence.

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

## Claim states

An observation (15-observations) gives a probe-declared claim one of three
states. They are observed, never authored: no cell carries one.

| State | Meaning |
|---|---|
| `holding` | Every expectation of the probe was observed as expected, in one round. |
| `broken` | An attempt the declared station itself authenticated contradicts the claim (16-probes: only an acceptance can break a claim). |
| `unknown` | Neither: nothing answered, the probe could not conclude, or what it saw cannot be relied on. A probe that cannot reach its target yields `unknown`, never `holding`. |

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
| `assessed` | Answer verified, and re-checkable: at least one measure carries `test`, `scenario` or `probe` evidence (12-cell-schema). | Yes. |
| `roadmap` | The threat is accepted as real; the defense does not yet exist and is scheduled. | Yes, disclosed, must have `review_by`. |
| `na` | Cell not applicable; **must** carry `na_reason`. | Yes, disclosed. |

## Anti-theater rules

Linting (see 13-scorecard) fails an assessment when:

- two cells share a threat definition beyond a similarity threshold —
  copy-paste is the primary symptom of a workbook that was filled, not
  written;
- a cell has zero manifestations;
- a `roadmap` cell lacks `owner` or `review_by`;
- a cell's status is `assessed` but all its measures are `org`;
- a cell's status is `assessed` but none of its measures carries executable
  evidence.

## Versioning

See [spec/README.md](../README.md). This is **v0.4**. The taxonomy is
therefore frozen at release and evolves only through numbered versions —
decisions about axes and attributes are made in spec PRs, not in prose
debates inside individual assessments.
