# The issues bridge — roadmap cells become the backlog

`fovea issues` publishes the assessment's `roadmap` cells as GitHub issues,
idempotently, and polices the link between tracker and matrix.

## The mechanism

Every issue fovea creates carries a marker in its body —
`<!-- fovea-cell: operate.confidentiality -->`. On re-run the command lists
its own issues (label `fovea`), matches on the marker, updates bodies and
labels in place, closes issues whose cells left `roadmap`, and opens only
genuinely new ones. Issues without the marker are never touched.

| Cell state | Action |
|---|---|
| `roadmap` | issue open, title `security: <cell.id>`, label `fovea/roadmap` |
| `roadmap` with `review_by` in the past | additionally labelled `fovea/overdue` |
| `assessed` or `na` | linked issue closed |

## Commands

```
fovea issues <dir> --dry-run          # decisions only, no writes
fovea issues <dir> --check            # the trap (below)
fovea issues <dir>                    # the real sync (needs GITHUB_TOKEN)
```

Options: `--repo owner/name` (default: detect from `git remote origin`),
`--token` (default: `$GITHUB_TOKEN`).

## The trap

`--check` fails the build when a `roadmap` cell has a linked issue that is
**closed** but the cell's status is unchanged — closing the ticket without
doing the work (or editing the cell) is a lint error. It also warns when an
overdue cell's issue isn't labelled `fovea/overdue`.

## CI wiring (the complete recipe, permissions included)

The default `GITHUB_TOKEN` is **read-only for issues**; the sync job must
declare `issues: write` or the API answers 403. Push-time check and weekly
sync:

```yaml
on:
  schedule: [{ cron: "17 6 * * 1" }]   # weekly
  workflow_dispatch:
  push: { paths: ["security/fovea/**"] }

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: macula-io/macula-fovea@main
        with: { dir: security/fovea, command: issues, args: --check }
        env: { GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}" }

  sync:
    if: github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-latest
    permissions: { contents: read, issues: write }   # ← required
    steps:
      - uses: actions/checkout@v4
      - uses: macula-io/macula-fovea@main
        with: { dir: security/fovea, command: issues }
        env: { GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}" }
```

## Idempotency contract

Safe to run on every push if you want; the marker makes it a sync, not a
spam. First run opens one issue per roadmap cell; every later run only
reconciles diffs.
