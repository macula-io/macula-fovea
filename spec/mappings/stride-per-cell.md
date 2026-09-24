# Mapping: STRIDE per cell

Fovea treats STRIDE not as a competing taxonomy but as a **manifestation
generator** inside cells. The letter of the mapping:

| STRIDE letter | Typical fovea column | Family |
|---|---|---|
| Spoofing | `machine_agent`, `trusted_partner`, `admin` | Actors |
| Tampering | `create`, `acquire`, `deliver`, `operate`, `at_rest`, `in_motion`, `in_use` | Lifecycle / Data |
| Repudiation | `admin`, `operate`, most actors | Actors / Lifecycle |
| Information Disclosure | `at_rest`, `in_motion`, `in_use`, `socio_legal` | Data / Environment |
| Denial of Service | `operate`, `physical_natural`, `temporal`, any actor | Lifecycle / Environment |
| Elevation of Privilege | `internal`, `trusted_partner`, `admin` | Actors |

When writing a cell, enumerate manifestations per STRIDE row that maps to
the cell's column (the first column above), skipping those manifestly
inapplicable to the attribute; then rewrite them in system-specific language.
STRIDE-as-STRIDE words are not manifestations; they are the seed that
becomes one.
