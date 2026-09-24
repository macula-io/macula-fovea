# Mapping: NIST CSF 2.0

CSF's six Functions map onto the fovea structure as follows:

| CSF Function | Fovea counterpart |
|---|---|
| Govern | The trust-anchor register + the `admin` column + the `socio_legal` environment column. Governance appears as cells and as the instantiation protocol. |
| Identify | The system landscape + the inventory side of the `data` family (`at_rest`/`in_motion`/`in_use` columns) + the asset-aware side of the trust-anchor register. |
| Protect | `countermeasures` arrays in every cell. |
| Detect | `detection` arrays. |
| Respond | Cells cover it implicitly via countermeasures; the recovery split (below) is fovea's contribution where CSF 1.x underweighted it. |
| Recover | `recovery` arrays — fovea's deliberate addition, not CSF 1.x's purview; CSF 2.0 makes recovery explicit and fovea follows. |

The scorecard's per-family roll-up is therefore a legible *summary view* of
CSF coverage: a row with poor "Detect" status is exactly what a CSF
profile review would flag. Mapping is for communication across frameworks,
not compliance; fovea is not a CSF profile in itself.
