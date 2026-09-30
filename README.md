# macula-fovea

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](#license)
[![Spec](https://img.shields.io/badge/spec-v0.5-blueviolet)](spec/v0.5/00-overview.md)
[![CI](https://github.com/macula-io/macula-fovea/actions/workflows/ci.yml/badge.svg)](https://github.com/macula-io/macula-fovea/actions/workflows/ci.yml)
[![GitHub Sponsors](https://img.shields.io/badge/GitHub%20Sponsors-support-ea4aaa.svg?logo=githubsponsors&logoColor=white)](https://github.com/sponsors/rgfaber)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/macula-fovea-full-dark.svg">
    <img src="assets/macula-fovea-full-light.svg" alt="Macula Fovea" width="320">
  </picture>
</p>

<p align="center">
  <strong>Security assessment specified as code: every cell a question<br>with a status and an owner, not a heading with hope.</strong>
</p>

---

Fovea is the part of the macula with the densest photoreceptors: the place
with the sharpest vision. `macula-fovea` is the sharpest-eyes instrument of
the ecosystem: a small, explicit framework for answering *"what can go wrong
here, caused by whom, and what do we honestly have against it."*

> **Status, 2026-09-30:** spec [v0.5](spec/v0.5/00-overview.md) is current:
> evidence on measures, `assessed` only with executable evidence, and probe
> declarations whose observations are signed records anyone can verify
> offline against the realm key (spec 15, 16), chained so that a kept
> history shows every missing, duplicated or late observation (v0.5).
> The [`fovea`](cli/) CLI (`init`, `lint`, `score`, `render`, `issues`, `verify`) is a
> single static Go binary and the reference implementation of the spec: its
> `lint` enforces the fixed grid, the status and content rules and the
> anti-theater rules, each one proven by a rule case under
> [`cli/internal/core/testdata/`](cli/internal/core/testdata/). A
> language-neutral conformance suite built from those cases comes next. The
> GitHub Action ([`action.yml`](action.yml)) runs the CLI in CI with PR
> annotations and scorecard artifacts. `assessments/macula-mesh-realm/` is an
> unfinished skeleton (1 of 96 cells, owner unassigned) and fails lint.

## What is macula-fovea?

A **taxonomy with semantics**, versioned as a spec and enforced as data, not
a spreadsheet with vibes. Threat models kept in Word documents and wiki
tables rot quietly: cells go unfilled, copies drift, and nobody can diff "the
threat model" against "the code". Fovea moves the model into the repository
next to the code and makes it lintable, scoreable, and renderable.

Fovea assesses any system, Macula or not: the spec, the lint, the scorecard,
signed observations, their chain and `fovea verify` describe and check
claims about whatever the assessment names. What is Macula-specific today is
how observations are signed and carried (macula records, checked against a
Macula realm key) and the one probe that exists (`kx_group`, which observes
Macula stations); probes of other systems are planned, not built.

**Start here:** [`docs/getting-started.md`](docs/getting-started.md) is the
ten-minute zero-to-clean-grid walkthrough;
[`docs/reference.md`](docs/reference.md) is the one-page statuses/rules/
pitfalls cheat sheet; [`spec/v0.5/`](spec/v0.5/00-overview.md) is the
normative contract everything else obeys.

Concretely, a fovea assessment is four artifacts, produced **in order**:

1. **Trust-anchor register**: the named things whose compromise breaks
   everything (domain CAs, root keys, build pipeline). If it isn't listed
   here, the assessment hasn't started.
2. **System landscape**: the diagram the matrix refers to. Every column of
   the matrix must name something on it; everything on it must be under at
   least one column.
3. **The matrix**: cells of *column × attribute*, each one a YAML file with
   a threat, defenses, a status, and an owner.
4. **The scorecard**: computed, never hand-written: roll-ups per attribute
   and per column family, plus the headline metric, the count of
   unjustified cells.

## The matrix

Sixteen columns in four families, defined in
[`spec/v0.5/10-axes.md`](spec/v0.5/10-axes.md). The grid is fixed: a header
declares all sixteen, each in its own family, and lint fails a header that
drops, moves or invents one.

| Family | Columns |
|---|---|
| **Actors** (origin × agency) | `internal` · `external` · `trusted_partner` · `machine_agent` |
| **Lifecycle** (the system over time) | `create` · `acquire` · `deliver` · `operate` · `admin` · `decommission` |
| **Data state** (metadata included) | `at_rest` · `in_motion` · `in_use` |
| **Environment** (the non-technical world) | `physical_natural` · `socio_legal` · `temporal` |

Columns are extensible via namespaced `x_` columns declared in the assessment
header: nothing hidden, nothing implicit.

## The attributes

**Core five, always assessed:** `confidentiality`, `integrity`,
`availability`, `authenticity`, `accountability`. Two Parkerian extensions,
`possession` (asset taken but unread: the node-capture case) and `utility`
(asset intact but useless), are enabled per assessment in
[`fovea.yaml`](assessments/macula-mesh-realm/fovea.yaml), with disabled ones
carrying a written justification. Definitions in
[`spec/v0.5/11-attributes.md`](spec/v0.5/11-attributes.md).

## A cell

One file per `column.attribute`, checked by `fovea lint` against
[`spec/v0.5/12-cell-schema.md`](spec/v0.5/12-cell-schema.md):

```yaml
id: in_motion.confidentiality
status: assumed            # unassessed | assumed | assessed | roadmap | na
owner: unassigned          # lint fails if this stays unassigned
threat:
  definition: ...
  manifestations: [ ... ]  # must differ from every other cell
defense:
  detection:      [ { measure: ..., status: org } ]
  countermeasures:[ { measure: ..., status: by_design } ]
  recovery:       [ { measure: ..., status: roadmap } ]
review_by: 2027-03-31      # the temporal column, applied everywhere
```

Every defense measure carries its own claim strength: `by_design`,
`roadmap`, or `org` (the organization's job, not the product's). So the
matrix reports **engineered fact separately from intent**, and intent
separately from everyone else's homework. That's the entire point.

## The scorecard

Computed from cell statuses, greppable by humans and CI alike. The two
headline numbers: **pct unassessed** and **unjustified-NA count**. Full
semantics in [`spec/v0.5/13-scorecard.md`](spec/v0.5/13-scorecard.md).

## Layout

```
spec/v0.5/            # the framework itself, current version (v0.2/ to v0.4/ archival)
spec/proposals/       # non-normative proposals (evidence, issues bridge)
spec/mappings/        # cross-walks to NIST CSF, ISO 27001, ATT&CK, STRIDE
cli/                  # the fovea binary: init / lint / score / render / issues
  internal/core/testdata/  # one assessment per lint rule, with expected findings
action.yml            # GitHub Action wrapping the CLI
.github/workflows/    # this repo's CI: gofmt, vet, race tests, build, scripts, action run
.github/dependabot.yml  # weekly updates for the pinned actions and Go modules
scripts/              # the Action's pin-staleness check and pin guard, with offline tests
docs/                 # operator-facing: getting started, reference, reading, writing
packs/                # the intended format for pre-filled cell packs (none yet)
templates/            # a cell skeleton to copy by hand
assets/               # brand artwork (dark/light logo variants)
assessments/
  macula-mesh-realm/  # dogfood #1: header, landscape, anchors, one seed cell
```

## Releases and pinning

The tool (the `fovea` CLI and the GitHub Action) is released as `vX.Y.Z`
git tags; `fovea --version` prints the tool version and the spec versions
it reads. The tool version and the spec version are separate lines: tool
0.1.0 reads assessments written to spec v0.2 and v0.3; tool 0.2.0 reads v0.2
to v0.5 and adds `fovea verify`. See
[`docs/CHANGELOG.md`](docs/CHANGELOG.md).

Use the Action pinned to the full commit sha of a release, with the tag as
a comment, never `@main` or a bare tag (both can move under you):

```yaml
- uses: macula-io/macula-fovea@<40-char sha of the release> # v0.2.0
  with:
    dir: security/fovea
    command: lint
```

Resolve the sha of a tag with
`gh api repos/macula-io/macula-fovea/commits/v0.2.0 --jq .sha`, and let
Dependabot keep the pin current. In `.github/dependabot.yml`:

```yaml
version: 2
updates:
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
```

On self-hosted runners, use runner 2.327.1 or newer: the Action and the
actions it pins run on Node 24.

Every run of the Action checks its own pin. A branch or tag ref gets a
warning. A sha more than 14 days behind the newest release gets a warning
and a line in the job summary, which is how a Dependabot that stopped
working shows up in your own CI. When the check cannot tell (no network,
rate limit, no releases yet) it says so in a notice. It never fails the
job.

## Relationship to other repos

| Repo | Role |
|---|---|
| [`macula-io/macula-station`](https://github.com/macula-io/macula-station) | Subject of the first dogfood assessment: the mesh substrate under evaluation. |
| [`macula-io/macula-realm`](https://github.com/macula-io/macula-realm) | The governance half of the same assessment: CA hierarchy, membership policy. |
| `macula-services/mcl-fovea` | The mesh-served observer being planned there; it implements the same rules and must pass the same conformance cases as this CLI. |

## License

MIT. See [LICENSE](LICENSE).

---

<p align="center">
  <sub>The fovea doesn't see more than the retina. It sees what matters, sharply.</sub>
</p>
