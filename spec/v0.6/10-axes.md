# 10 — Axes: the four column families

A **column** is a lens. Every cell is the intersection of exactly one column
and one attribute. The four families answer: *caused by whom*, *during what
phase of the system's life*, *in what data state*, and *under what
non-technical condition*.

## Actors — `internal`, `external`, `trusted_partner`, `machine_agent`

Origin and agency, deliberately kept separate (origin = where from, agency =
human or not).

| Column | Definition |
|---|---|
| `internal` | Human agents with legitimate inside access who abuse or misuse it: administrators, developers, operators acting maliciously or erroneously. |
| `external` | Human agents without legitimate access: intruders, nation-state operators, hacktivists. |
| `trusted_partner` | Entities trusted by the system but not part of it: vendors, MSPs, SDK and package maintainers, cross-signed authorities. Neither inside nor outside; the SolarWinds column. |
| `machine_agent` | Automated actors: bots, worms, scrapers, agentic processes acting on behalf of anyone, including "the system's own automation behaving badly". |

## Lifecycle — `create`, `acquire`, `deliver`, `operate`, `admin`, `decommission`

The system **over time**. A threat that cannot occur in operation may occur
in creation (malicious dependency) or disposal (key remnant on a retired
device).

| Column | Definition |
|---|---|
| `create` | Design and build: source code, toolchain, dependencies, CI. Threats enter before the system exists: poisoned dependencies, compromised toolchain, backdoored libraries. |
| `acquire` | Procure and trust software or hardware from third parties: vendor integrity, license and provenance verification, counterfeit or tampered components. The buy-vs-build exposure. |
| `deliver` | Distribution and provisioning: package signing, first-boot bootstrap, seed distribution, attestation before first trust. |
| `operate` | Normal runtime: every threat live systems face during use. Historically the only column people write about — which is why this family exists. |
| `admin` | Administration of trust: CA ceremonies, key rotation, revocation issuance, configuration change, access management. Quiet, rare, and catastrophic when wrong. |
| `decommission` | Retirement, transfer, destruction: key remnants, stale caches, captured or resold hardware, unreclaimed accounts. The most commonly forgotten column in real assessments. |

## Data state — `at_rest`, `in_motion`, `in_use`

Where the data physically is when threatened.

| Column | Definition |
|---|---|
| `at_rest` | Stored: on disk, in backups, in databases — encrypted or not. |
| `in_motion` | In transit across any link, internal or external. |
| `in_use` | In memory during processing: RAM contents, CPU caches, side-channel exposure while computation proceeds. |

**Rule: data includes metadata and derived data.** Traffic patterns, timing,
access logs, and key identifiers are data. For any mesh or overlay network
this is not a footnote; the metadata domain is frequently the more valuable
target, and assessments that skip it underreport SIGINT exposure.

## Environment — `physical_natural`, `socio_legal`, `temporal`

The world the system is powerless to keep out.

| Column | Definition |
|---|---|
| `physical_natural` | Physical and natural events: destruction, disaster, jamming, RF degradation, causal hardware failure, kinetic seizure of equipment. |
| `socio_legal` | Human law and politics acting as an adversary: lawful-seizure subpoenas, sanctions, export controls, censorship regimes, jurisdictional pressure on operators. Adversary zero is a court, not a hacker. |
| `temporal` | Time as an adversary: algorithm deprecation (harvest-now/decrypt-later, quantum), certificate and license expiry, key aging, trust decay, record retention obligations colliding with secrecy. |

## Extensibility

Assessments may add columns under the reserved prefix `x_` (e.g.
`x_supply_chain`) **declared in the header** (see
[12-cell-schema](12-cell-schema.md)). Undeclared columns are a lint error:
an extension that isn't written down isn't an extension, it's a loophole.
