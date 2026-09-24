# Mappings (spec annex)

Fovea deliberately does **not** reinvent what established frameworks already
cover. These mappings exist so that a fovea matrix can be *read* as
ISO/controls evidence without the writer learning every framework's
vocabulary. Each is a lightweight annex, not a normative part of the spec.

| Framework | File | Fovea's relationship to it |
|---|---|---|
| ISO/IEC 27001:2022 Annex A | [`iso-27001-2022-annexA.md`](iso-27001-2022-annexA.md) | Cells map to controls; the grid is where controls are *satisfied*, the matrix the evidence where they are. |
| NIST CSF 2.0 | [`nist-csf-2.0.md`](nist-csf-2.0.md) | Fovea's core-five rows approximate CSF's Identify/Protect/Detect/Respond/Recover organization; the scorecard's per-family column maps closely. |
| STRIDE | [`stride-per-cell.md`](stride-per-cell.md) | STRIDE asks "what could go wrong per *element*"; fovea's cells are the *column* view. Use STRIDE to *generate* the manifestations list inside a cell. |
| ATT&CK | [`attck-tactics.md`](attck-tactics.md) | Cells' manifestations link to ATT&CK techniques where applicable; fovea is threat-model-space, ATT&CK incident-space. |

Each file documents the axis-to-axis relationship: what maps cleanly, what
needs manual translation, and what has no mapping (which is appropriately
disclosed).
