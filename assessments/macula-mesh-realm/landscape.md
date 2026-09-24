# System Landscape — macula-mesh-realm

*Placeholder — the spec's step 2. This file exists to be replaced by a real
landscape diagram; nothing here is final or authoritative.*

## Elements (to be diagrammed)

- **Stations** — the mesh routers. At minimum: several generalized stations,
  plus any dedicated role-stations (seed, relay, gateway-adjacent).
- **Realm** — the governance boundary; the CA hierarchy that issues
  domain/org/leaf identities. Runs partly "off-wire" (ceremonies) and partly
  as mesh service(s).
- **Org** — the membership unit within a realm; org CA, leaf certificates,
  revocation lists.
- **Nodes / daemons** — application processes attached to a station, with or
  without their own TUN device.
- **External links** — the portions of flow that cross the public internet
  or adversary-adjacent media (up-links, satellites, tactical RF).

## Scope markers

- `in` of scope per fovea.yaml: the items above.
- Exclusions: none declared yet.

## Risks to surface in the landscape

1. The **single greatest visual risk** the diagram must answer: *where is
   the trust-anchor register located relative to each peer?* If the realm
   root CA is reachable as a network service, the "fractal, no-gateway"
   claim is compromised from its own diagram; it must be drawn off-wire.
2. The **first-hop into the mesh** — where the deliver/decommission
   boundaries touch the external world.
3. Which links are *observable* (i.e., higher threat under `in_motion`).

## Status

`unassessed` — to be replaced with a diagram and the per-element
ownership table that fovea's instantiation step 2 requires.
