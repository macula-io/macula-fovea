# Packs

A **pack** is a set of pre-filled, reviewed cells for a recurring technology
class — a way to start an assessment with *reasonable defaults for the
terrain*, not a blank grid. Packs live here and are referenced from cells by
`(pack, cell-id)` tuples; the CLI merges them at `fovea init` time.

## Format

```
packs/<pack-name>/
  pack.yaml          # name, version, applicability statement, maintainer
  cells/*.yaml       # candidate cells, same schema as per-assessment cells
  NOTES.md           # designer's notes: when to include, when to strip
```

Packs are *suggestions*, not doctrines:

- they only ever produce `assumed`-or-weaker statuses; an assessment may
  promote them (`assumed → assessed`) but **never** import them as
  `assessed` directly;
- an assessment overrides a pack cell by writing its own cell file in its
  own `cells/` dir — packs never leak content into assessments silently.

## Planned packs

| Pack | Status | Coverage |
|---|---|---|
| `network-mesh-generic` | planned | DHT/gossip/P2P meshes; cells for the classic "hub is a beacon", eclipse, byzantine gossip |
| `ca-pki-governance`   | planned | CA hierarchies, key ceremonies, CRL/OCSP; the `admin` column cells |
| `supply-chain-cicd`   | planned | `create`/`acquire` cells for dependency poison, build tamper, provenance |

Packs stay generic: domain-specific content (e.g. "macula-station" cells)
belongs in the *assessment* dir, not in a pack.
