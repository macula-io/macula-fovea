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
| 6 | [`issues-bridge.md`](issues-bridge.md) | Operators, CI | The roadmap-cells-to-GitHub-issues sync: markers, idempotency, the closed-issue trap, and the CI recipe with the `issues: write` permission gotcha. |

## Worked examples (reference implementations)

Two complete, lint-clean 96-cell assessments exist in the wild and are the
best study material after the docs:

- `macula-services/mcl-echo/security/fovea/` — a stateless hello-world
  service; many justified N/As, thin data family.
- `macula-services/mcl-tube/security/fovea/` — an event-sourced video
  service with real content at rest; a streaming procedure, an owner UI,
  and a retraction-confidentiality roadmap cell.

Read their `fovea.yaml` (scope discipline), `trust-anchors.md` (what counts
as an anchor), and the `roadmap` cells (how gaps are recorded honestly).

The normative contract is not in these docs; it is in [`spec/v0.3/`](../spec/v0.3/00-overview.md).
These docs explain how to run the spec; the spec explains what the rules are;
the `fovea` CLI's `lint` is the reference implementation that enforces them
(one test case per rule under `cli/internal/core/testdata/`).
