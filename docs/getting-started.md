# Getting started — zero to a lint-clean grid

Ten minutes, one service. This walks the exact command sequence; the
normative rules it obeys live in [`spec/v0.3/`](../spec/v0.3/00-overview.md).

## 0. Build the CLI

```bash
cd cli && go build -o fovea ./cmd/fovea
```

(Go 1.27 — `.tool-versions` at the repo root pins it for asdf/mise.)

## 1. Scaffold an assessment directory

```
security/fovea/
├── fovea.yaml          # the header: attributes + columns in force
├── trust-anchors.md    # named roots whose compromise totals the assessment
├── landscape.md        # the diagram the matrix indexes
└── cells/              # generated next
```

Minimal `fovea.yaml`:

```yaml
fovea: "0.3"
system: my-service
assessment_date: 2026-09-25
owner: you@example.org          # lint fails on "unassigned"
attributes:
  core: [confidentiality, integrity, availability, authenticity, accountability]
  enabled: [possession]            # extensions: possession, utility
  disabled_justifications:         # every extension not enabled needs one
    utility: >-
      An intact but useless asset is an availability failure here.
columns:                           # all 16 spec columns, each in its family
  actors: [internal, external, trusted_partner, machine_agent]
  lifecycle: [create, acquire, deliver, operate, admin, decommission]
  data: [at_rest, in_motion, in_use]
  environment: [physical_natural, socio_legal, temporal]
cells_dir: cells/
```

Write `trust-anchors.md` (the keys, CAs, pipelines whose compromise breaks
everything) and `landscape.md` (a diagram; every column must name something
on it and everything on it must be under some column) — see
[spec 14](../spec/v0.3/14-instantiation.md).

## 2. Generate the grid

```bash
fovea init security/fovea
```

96 skeletons appear (80 core cells plus 16 for `possession`), each
`unassessed`; `init` creates `cells/` if needed and never overwrites an
existing file. Refusing to run until the header is clean (owner claimed,
grid complete, extensions justified) is the framework being armed, not
broken.

## 3. Write cells

One file per `column.attribute`. Fill the *cold* families first
(`decommission`, `socio_legal`, `temporal`) — that's where the questions
you wouldn't have asked live. Every `by_design` measure needs a `source`;
every `roadmap` cell needs `review_by`; every `na` needs `na_reason`.
Template: [`templates/cell.yaml`](../templates/cell.yaml).

## 4. Lint until clean

```bash
fovea lint security/fovea
```

Expect two errors per untouched skeleton: `cell_unassessed` and
`cell_owner_unassigned`. Once a cell is answered (`assumed`, `assessed` or
`roadmap`) it also needs a definition and at least one manifestation.
`lint` exits 0 only with no findings; [reference.md](reference.md) lists
every rule code.

## 5. Score and render

```bash
fovea score security/fovea --json   # headline metrics, machine-readable
fovea render security/fovea         # markdown scorecard
fovea render security/fovea --html  # GitHub job-summary scorecard (emoji RAG)
fovea render security/fovea --json  # grid, coverage and open gaps for CI
```

## 6. Wire CI

```yaml
- uses: macula-io/macula-fovea@main
  with: { dir: security/fovea, command: lint }
```

Do **not** wire the hard gate until lint is clean — a red run on an
unfinished grid is noise, not signal. (Or keep the lint step
`continue-on-error: true` while filling, and harden it at completion.)

## 7. The promotion loop

Amber → green by *verifying* (`assumed` → `assessed`: a test or live check
proves the sourced claim). Roadmap → green by *implementing* the gap and
recording the evidence. Grey stays grey — justified N/A is correct, not a
failure. Green is earned by evidence, never by editing statuses.
