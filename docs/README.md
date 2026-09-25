# Fovea Documentation Index

This directory is **operator-facing**: who opens fovea, what each document is
for, in which order to read.

| Order | Document | For | Purpose |
|---|---|---|---|
| 1 | [`getting-started.md`](getting-started.md) | New users | Zero to a lint-clean grid in ten minutes: scaffold, generate, fill, lint, score, render, wire CI, promote amber→green. |
| 2 | [`reference.md`](reference.md) | Everyone, often | Statuses, measure claim-strengths, RAG mapping, every lint rule, CLI flags, and the pitfalls that have bitten real assessments. |
| 3 | [`reading-the-matrix.md`](reading-the-matrix.md) | Readers, reviewers, auditors | How to open a fovea assessment and extract what matters in five minutes. Trust-anchor register first, scorecard second, amber cells third. |
| 4 | [`writing-an-assessment.md`](writing-an-assessment.md) | Assessor, tech leads | How the instantiation protocol runs: anchors → landscape → header → cells → scorecard, and how to write a cell that survives the lint. |
| 5 | [`maintaining-the-spec.md`](maintaining-the-spec.md) | Spec maintainers | How the spec itself is versioned, PR-disciplined, and spawned into new specs; the meta-rules that keep taxonomy sane. |

The normative contract is not in these docs; it is in [`spec/v0.2/`](../spec/v0.2/00-overview.md).
These docs explain how to run the spec; the spec explains what the rules are.
