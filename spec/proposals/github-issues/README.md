# Proposal — fovea issues: the matrix becomes the backlog

**Status: PROPOSED. Not normative.** Phases 1 and 2 are implemented as
`fovea issues` (see below and [docs/issues-bridge.md](../../../docs/issues-bridge.md));
the spec does not require them. It is
the "work out" half of the loop, symmetric to
[`v0.4-evidence`](../v0.4-evidence/README.md)'s "evidence in".

## The problem this solves

`roadmap` cells are already perfectly-formed work items — stable id, owner,
gap description, `review_by` date — but nothing turns them into work. The
scorecard shows them; the team's actual queue doesn't. A gap that is visible
but not actionable drifts, and the whole point of fovea is that gaps must not
drift.

## Proposed command

```
fovea issues <dir> [--dry-run] [--token GITHUB_TOKEN]
```

A **sync**, not a spammer. Idempotent across runs:

| Cell state | Action |
|---|---|
| `roadmap` | issue open, title `security: <cell.id>`, body = threat + manifestations + `review_by` + `<!-- fovea-cell: <cell.id> -->`, label `fovea/roadmap` |
| overdue (`review_by` in the past) | existing issue relabelled `fovea/overdue` |
| `assessed` or `na` | linked issue closed automatically |

## The idempotency contract

Every issue fovea creates carries a marker in its body:

```
<!-- fovea-cell: operate.confidentiality -->
```

On re-run, the command lists its own issues (by label `fovea`), matches on
the marker, updates bodies/labels in place, closes the ones whose cells
moved on, and opens only genuinely new ones. Issues without the marker are
never touched. Duplicate-issue CI tools fail exactly here; the marker is the
whole difference.

## The anti-theater lint rule (the interesting half)

`fovea issues --check` fails the build when:

> a `roadmap` cell has a linked issue that is **closed** but the cell's
> status is unchanged.

Closing the ticket without doing the work — or editing the cell — becomes a
lint error. The tracker and the threat model cannot drift apart.

**Implemented (phases 1+2):** `fovea issues` ships in the CLI — REST client
over `GITHUB_TOKEN` (stdlib only, no new deps), marker idempotency,
`--dry-run`, `--check`, `--repo` override with git-remote detection. Phase 3
(scenario-failure auto-issues) still depends on the v0.4 evidence work.

## Defaults and limits

- **Opt-in per status:** `roadmap` + `overdue` sync by default. Auto-opening
  `unassessed` cells would flood the tracker with 72 tickets; those stay in
  the scorecard's open-gaps list.
- **Scheduled, not per-push:** a cron workflow with `GITHUB_TOKEN` (write)
  runs the sync; pushes only run `--check`.
- **Label scheme:** `fovea`, `fovea/roadmap`, `fovea/overdue`.

## Synergy with v0.4 evidence

A scenario whose evidence fails in `fovea verify` auto-opens an issue
referencing both the cell and the failing feature file. Evidence in, work
out — one loop, closed.

## Open questions

1. Backend: hand-rolled REST client over `GITHUB_TOKEN` (self-contained, no
   deps) vs. shelling out to the `gh` CLI (less code, requires gh locally)?
   Proposal leans REST.
2. Should the sync also mirror `assumed` cells as low-priority issues
   ("verify me"), or is that tracker noise?
3. Repo scope: one repo per assessment directory, or a `repos:` mapping in
   `fovea.yaml` for multi-repo systems?
