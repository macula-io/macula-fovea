# 14 — Instantiation protocol

The matrix does not start at the first cell. Four artifacts, **in order**,
because the fourth cannot mean what it means without the first three.
The protocol exists so that two assessors — or an assessor and an auditor
six months later — reach the same shape.

## Step 1 — Trust-anchor register

`trust-anchors.md`. The named set of things whose compromise makes every
other defense meaningless. For a typical governed mesh: the root CA key and
its ceremony; the org CA keys; the build-and-release pipeline key; the
attestation root (TPM vendor cert chain) if attestation exists. **The rule:**
if a thing's compromise would require rebuilding the system, it's on the
list. If it isn't on the list, the cells' `by_design` claims fail at
review — they imply an unstated anchor.

## Step 2 — System landscape

`landscape.md`. One diagram, as small as possible, naming every process,
store, link, and actor the assessment covers. The test is bidirectional:
every column of the matrix must locate at least one element on the
diagram, and every element must be under at least one column. Elements in
the diagram that are out of scope are drawn dashed and explicitly listed
as excluded **with a reason**.

## Step 3 — Matrix header (`fovea.yaml`)

Commit to the regime *before* any cell exists: which attributes are core
(and therefore must exist), which extensions are enabled, which columns
(including any `x_` extensions) are declared. The header is what the scorecard
uses to know how many cells to expect — 16 columns × 5 core attributes =
80 base cells, plus 16 per enabled extension.

**`system` (v0.4).** From v0.4 the header's `system` is a lowercase identifier,
`[a-z][a-z0-9_.-]*`: observations carry it first in their record's subject,
before a 0x00 byte (15-observations), so it can hold no 0x00 and one spelling
names one assessment.

### Targets and policy (v0.4)

An assessment with probe declarations says, in its header, **what** they
observe and **what** an observer may publish.

```yaml
targets:
  fleet_stations:                  # a name probes refer to
    kind: macula_station           # the only target kind v0.6 defines
    stations:                      # each: where it answers, and who it must be
      - address: "192.0.2.10:4433"
        node_id: "0011111111111111111111111111111111111111111111111111111111111111"
      - address: "[2001:db8::10]:4433"
        node_id: "0022222222222222222222222222222222222222222222222222222222222222"
policy:
  publish: state_changes           # every_result | state_changes
  cadence: PT1H                    # ISO 8601 duration, a minute to seven days
  suspended: []                    # claim ids an observer must not observe
  observers:                       # node ids whose observations count
    - "0033333333333333333333333333333333333333333333333333333333333333"
```

- **Addresses are IP literals, in one canonical form.** An observation that
  resolved a name would observe whatever the resolver said; the target is the
  address. Each address is written exactly one way, because observations
  carry it as their record's subject (15-observations): IPv4 in dotted
  decimal (an IPv4-mapped IPv6 address is written as IPv4), or IPv6 in
  brackets, compressed per RFC 5952 in lowercase with no zone id; then `:` and
  the port in decimal without leading zeros. `[2001:db8::10]:4433`, never
  `[2001:DB8:0:0::10]:04433`. An address appears once in the whole header,
  and is neither the unspecified address nor a multicast one.
- **Each station is named by its node id** (64 lowercase hex digits). An
  observer counts a handshake as the station's only when the station proves
  that identity (16-probes); another node answering at the address makes the
  attempt `inconclusive`. A station's node id is stable across restarts and
  address changes, and a changed address is a new assessment revision.
- **`publish`**: `every_result` publishes every observation; `state_changes`
  publishes an observation whose state differs from the station's previous one
  for the claim, and before the latest record expires signs a fresh record
  of the same round (a new `version` and `created_at`, and from v0.5 a new
  `seq` and `prev`; the same `observed_at` and outcomes), so a reader always finds a live one. An observer signs and keeps
  every observation either way.
- **`cadence`**: how often a claim is observed, from a minute to seven days.
  Days, hours, minutes and seconds only (`P1D`, `PT1H`, `PT15M`), since a
  month has no fixed length; seven days is the most an observation's record
  lives (15-observations).
- **`suspended`**: claims the observer must not observe until a later
  revision removes them, for example while a target is being rebuilt. Each
  must be a claim the assessment declares.
- **`observers`**: the node ids of the observers whose observations of this
  assessment a reader trusts. Every admitted member of a realm can sign a
  record of the observation type; this list is what tells a verifier which
  ones speak for this assessment (15-observations). An observer is never one
  of the declared stations: a station cannot observe itself.

A header with probe declarations and no `policy`, or a policy naming no
observer, is a lint error: nothing would say what may be published or whose
observations count.

## Step 4 — Cells

Fill the grid. Fill it in attribute-major or column-major order; do not
make the mistake of writing only where fear is highest — the *reason* there
is a grid. Fill `unassessed` cells eiter by answering or by declaring `na`
with reason. Do not leave placeholders without status updates; the cell
template enforces every field.

## Step 5 — Scorecard

`fovea score`, then `fovea render`. Review the roll-ups: any
column-family/attribute quadrant that is disproportionately red or amber
is where the next cycle's effort goes — the scorecard's only job is to make
that visible to whoever decides.

## Anti-patterns this protocol exists to kill

| Anti-pattern | Why |
|---|---|
| Threat model written directly by the product's own authors, with no landscape review | The landscape is the mechanism for arguing over *what is being defended*. |
| Cells filled with the same boilerplate, cell after cell | Copy-paste is the death of assessment value; lint catches it. |
| "We'll cover that later" | `roadmap` + `review_by` *is* "later"; there's no unwritten later. |
| Trust anchors never named | Then `by_design` was never defined, and the cells lied politely. |
