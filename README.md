# macula-fovea

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](#license)
[![Spec](https://img.shields.io/badge/spec-v0.2-blueviolet)](spec/v0.2/00-overview.md)
[![GitHub Sponsors](https://img.shields.io/badge/GitHub%20Sponsors-support-ea4aaa.svg?logo=githubsponsors&logoColor=white)](https://github.com/sponsors/rgfaber)

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/macula-fovea-full-dark.svg">
    <img src="assets/macula-fovea-full-light.svg" alt="Macula Fovea" width="320">
  </picture>
</p>

<p align="center">
  <strong>Security assessment specified as code — every cell a question<br>with a status and an owner, not a heading with hope.</strong>
</p>

---

Fovea is the part of the macula with the densest photoreceptors: the place
with the sharpest vision. `macula-fovea` is the sharpest-eyes instrument of
the ecosystem: a small, explicit framework for answering *"what can go wrong
here, caused by whom, and what do we honestly have against it."*

> **Status, 2026-09-25:** the v0.2 **specification, schemas, and
> operator docs** are stable; the [`fovea`](cli/) CLI exists and enforces
> them — `init` / `lint` / `score` / `render` as a single static Go binary,
> dogfooded daily against `assessments/macula-mesh-realm/`, which
> deliberately still refuses to be claimed. `mcl-fovea` (mesh-served) and the
> GitHub Action wait for a second real assessment to exist.

## What is macula-fovea?

A **taxonomy with semantics**, versioned as a spec and enforced as data — not
a spreadsheet with vibes. Threat models kept in Word documents and wiki
tables rot quietly: cells go unfilled, copies drift, and nobody can diff "the
threat model" against "the code". Fovea moves the model into the repository
next to the code and makes it lintable, scoreable, and renderable.

**Start here:** [`docs/getting-started.md`](docs/getting-started.md) is the
ten-minute zero-to-clean-grid walkthrough;
[`docs/reference.md`](docs/reference.md) is the one-page statuses/rules/
pitfalls cheat sheet; [`spec/v0.3/`](spec/v0.3/00-overview.md) is the
normative contract everything else obeys.

Concretely, a fovea assessment is four artifacts, produced **in order**:

1. **Trust-anchor register** — the named things whose compromise breaks
   everything (domain CAs, root keys, build pipeline). If it isn't listed
   here, the assessment hasn't started.
2. **System landscape** — the diagram the matrix refers to. Every column of
   the matrix must name something on it; everything on it must be under at
   least one column.
3. **The matrix** — cells of *column × attribute*, each one a YAML file with
   a threat, defenses, a status, and an owner.
4. **The scorecard** — computed, never hand-written: roll-ups per attribute
   and per column family, plus the headline metric — the count of
   unjustified cells.

## The matrix

Seventeen columns in four families, defined in
[`spec/v0.2/10-axes.md`](spec/v0.2/10-axes.md):

| Family | Columns |
|---|---|
| **Actors** (origin × agency) | `internal` · `external` · `trusted_partner` · `machine_agent` |
| **Lifecycle** (the system over time) | `create` · `acquire` · `deliver` · `operate` · `admin` · `decommission` |
| **Data state** (metadata included) | `at_rest` · `in_motion` · `in_use` |
| **Environment** (the non-technical world) | `physical_natural` · `socio_legal` · `temporal` |

Columns are extensible via namespaced `x_` columns declared in the assessment
header — nothing hidden, nothing implicit.

## The attributes

**Core five, always assessed:** `confidentiality`, `integrity`,
`availability`, `authenticity`, `accountability`. Two Parkerian extensions,
`possession` (asset taken but unread — the node-capture case) and `utility`
(asset intact but useless), are enabled per assessment in
[`fovea.yaml`](assessments/macula-mesh-realm/fovea.yaml), with disabled ones
carrying a written justification. Definitions in
[`spec/v0.2/11-attributes.md`](spec/v0.2/11-attributes.md).

## A cell

One file per `column.attribute`, validated against
[`schema/cell.schema.json`](schema/cell.schema.json):

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

Every defense measure carries its own claim strength — `by_design`,
`roadmap`, or `org` (the organization's job, not the product's) — so the
matrix reports **engineered fact separately from intent**, and intent
separately from everyone else's homework. That's the entire point.

## The scorecard

Computed from cell statuses, greppable by humans and CI alike. The two
headline numbers: **pct unassessed** and **unjustified-NA count**. Full
semantics in [`spec/v0.2/13-scorecard.md`](spec/v0.2/13-scorecard.md).

## Layout

```
spec/v0.2/            # the framework itself, versioned
cli/                  # the fovea binary — init / lint / score / render
docs/                 # operator-facing: reading, writing, maintaining
schema/               # assessment + cell JSON Schemas (YAML validated via conversion)
packs/                # reusable pre-filled cell packs per technology class
templates/            # empty cell skeleton used by `fovea init`
assets/               # brand artwork (dark/light logo variants)
assessments/
  macula-mesh-realm/  # dogfood #1 — header, landscape, anchors, cells
```

## Relationship to other repos

| Repo | Role |
|---|---|
| [`macula-io/macula-station`](https://github.com/macula-io/macula-station) | Subject of the first dogfood assessment — the mesh substrate under evaluation. |
| [`macula-io/macula-realm`](https://github.com/macula-io/macula-realm) | The governance half of the same assessment: CA hierarchy, membership policy. |
| `macula-services` | Future home of `mcl-fovea`, the mesh-served variant, once the CLI's format is proven. |

## License

MIT. See [LICENSE](LICENSE).

---

<p align="center">
  <sub>The fovea doesn't see more than the retina. It sees what matters, sharply.</sub>
</p>
