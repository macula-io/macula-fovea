# 15 — Observations

A probe declaration (12-cell-schema) names a claim; an **observer** checks it
on the running system, on the policy's cadence (14-instantiation), and turns
each round into an **observation**: one claim, one station, one state
(`holding`, `broken`, `unknown`; 00-overview), judged as the probe's registry
entry says (16-probes). This section defines the observation as a signed
record, so that anyone holding the realm's public key and the assessment can
verify it offline, without the mesh or the observer, and (from v0.5) links an
observer's records of one claim and station into a **chain**, so that a kept
history of them shows every missing, duplicated or late observation.

## What an observer does

- It observes only claims of an assessment revision it has adopted, and not
  the claims the policy suspends.
- It judges each round exactly as the probe's entry in 16-probes says.
- It signs **every** observation. `policy.publish` decides only which ones
  it publishes (14-instantiation).
- It never changes a cell: the authored status stays what the assessment says.
- It links every record it signs into its slot's chain (below), and never
  signs two records of one slot with the same `seq`.

## ⛔ THE CONTRACT `fovea verify` DEPENDS ON

> Everything from here to the end of this section is what an offline
> verifier checks. It changes only in a new spec version, never by erratum,
> and an implementation that signs observations follows it to the byte.

### The record

An observation is a **macula record** of domain type **`0x23`,
`mcl_fovea.claim_observation.v1`**, allocated in macula's
`plans/DESIGN_PQ_SIGNED_FRAMES_AND_RECORDS.md` ("Domain record types"). Its
envelope, encoding and signature are macula's, specified in that document
and not restated here:

- a signed object `{key, tbs, signature}` in deterministic CBOR, signed
  under the record label, the signer named by `key` alone;
- `tbs` holds `type` (`0x23`), `alg`, `version`, `created_at`, `expires_at`,
  `payload` and `subject`;
- `expires_at` at most 7 days after `created_at` (a domain record's maximum);
- the signing key is the **observer's node identity key**: the key its node
  id is derived from, and the key its realm member endorsement names.

**`subject`**: the bytes of `system`, one 0x00 byte, the bytes of `claim_id`,
one 0x00 byte, then the bytes of `target_address`, all UTF-8. A domain
record's slot is its signer key id, then its subject, so a record store keeps
one observation per observer, assessment, claim and station address.
`target_address` is in the header's canonical form (14-instantiation), so
every observation of an address writes the same subject. A reader who knows
an observer by node id finds the slot by fetching the observer's node record
and deriving its signer key id (`MACULA-KEY-ID-V1`) from the key.

**`payload`**: a map with exactly these text keys, and no other:

| Key | Type | Value |
|---|---|---|
| `system` | text | The assessment's `system`, as its header writes it: `[a-z][a-z0-9_.-]*` (14-instantiation). |
| `claim_id` | text | The declaration's `claim`: `[a-z][a-z0-9_]*`. |
| `realm_id` | bytes, 32 | The wire realm id: SHA-256 of the realm's name. |
| `state` | unsigned | 0 `holding`, 1 `broken`, 2 `unknown`. |
| `probe` | text | The declaration's `probe`, e.g. `kx_group`. |
| `probe_version` | unsigned | The declaration's `version`. |
| `target_address` | text | The one address observed, in the canonical form of 14-instantiation, exactly as the header writes it. |
| `station_node` | bytes, 32 | The node id the header declares for that address: the only node an `accepted` outcome in this record can be about (16-probes). |
| `expected` | map | The declaration's `expect`: each expected group, as text, to 0 (`accepted`) or 1 (`refused`). |
| `outcomes` | map | The same groups, each to its attempt's outcome: 0 `accepted`, 1 `refused`, 2 `inconclusive` (16-probes). |
| `assessment_sha` | bytes, 20 or 32 | The commit id of the adopted assessment revision that declares the claim. |
| `publish` | unsigned | The publication policy in force when it was signed: 0 `every_result`, 1 `state_changes`. It does not say this record was published. |
| `observed_at` | unsigned | When the round ended, in milliseconds since the Unix epoch; not after the record's `created_at`. |
| `seq` | unsigned | This record's place in its slot's chain: 0 for the first, then one more than the record before it. |
| `prev` | bytes, 32 | SHA-256 of the complete wire bytes of the slot's record with `seq` − 1; 32 zero bytes when `seq` is 0. |

No value is a boolean or a float: 0 and 1 stand in for yes and no.

A record with exactly the thirteen keys of v0.4 (no `seq`, no `prev`) is a
v0.4 observation: it verifies as v0.4's section 15 says and belongs to no
chain. A v0.5 observation may name an assessment revision of v0.4 or v0.5,
whose formats are the same.

### The chain

A **slot** is one signer key and one `subject`: one observer, assessment
system, claim and station address, as a record store keeps it. Every record
an observer signs in a slot, published or not, is the next link of that
slot's chain: `seq` one more than the last one it signed there, `prev` the
SHA-256 of that record's wire bytes. A record's **wire bytes** are the
deterministic CBOR of its signed object `{key, tbs, signature}`, exactly as
handed to the record store and as the store returns it; nobody re-encodes
them to hash them.

- **The observer keeps its place.** After signing a record and before it
  publishes it or signs the slot's next one, it writes the record's `seq`, the
  SHA-256 of its wire bytes, and the wire bytes themselves to storage that
  survives a restart, atomically, so a crash leaves either the old place or
  the new one. After a restart it publishes the kept record again if it had
  published it and it has not expired, so an honest crash between writing and publishing costs no
  link.
- **A lost place is a restart, never a silence.** An observer that has lost
  its place (a new volume) starts again at `seq` 0. A reader sees the
  restart, and a set with one is never continuous across it. A new signer
  key is a new slot, not a restart.
- **Continuity is claimed only under `every_result`.** Under
  `state_changes` the observer signs rounds it does not publish, so a kept
  history of published records has gaps by design, and a verifier reports
  them as unpublished rounds, never as a continuous chain.

### Verification

A verifier holds, beside the observation's record bytes:

- the realm's **public key** as carried, the **profile** (`pq_hybrid` or
  `pq_pure`) and the realm's **name**;
- the observer's **realm member endorsement** record bytes (macula record
  type `0x05`, signed by the realm key). The realm publishes it in the DHT
  under the member's endorsement slot and renews it before it expires; an
  observer may ship it beside its observations;
- the **assessment header** of the revision the observation names
  (`assessment_sha`), and the probe declaration in it: a git checkout of the
  assessment, which is offline too. Its `policy.observers` (14-instantiation)
  are the observer node ids the verifier trusts. Observers come from the
  revision the record names, whatever revision the verifier or the observer
  has adopted since.

It accepts the observation only if every step holds, at **T = the
observation's `created_at`**, so an observation stays verifiable after the
endorsement that covered it has expired:

1. The observation verifies as a macula record at time T, and its `type` is
   `0x23`; and its `created_at` is at most 5 minutes after the verifier's own
   clock (verifying at T alone would never see a record dated in the future).
2. Its payload has exactly the keys and types above, and no other: text keys,
   each once. `state` is 0, 1 or 2; every `expected` value is 0 or 1; every
   `outcomes` value is 0, 1 or 2; `observed_at` is not after `created_at`;
   `prev` is 32 zero bytes exactly when `seq` is 0; and its `subject` is
   `system`, 0x00, `claim_id`, 0x00, `target_address`.
3. The endorsement verifies as a macula record at time T, its `type` is
   `0x05`, and its key's key id (`MACULA-KEY-ID-V1`) is the key id of the
   realm's public key.
4. The endorsement's `realm_id` is SHA-256 of the realm's name, and equals
   the observation's `realm_id`.
5. The endorsement's `member_node` is the **node id derived from the
   observation's `key`** (macula's `macula_node_keys:node_id/2`; macula-go's
   `identity.NodeIDOf`). That is not the record's signer key id: a domain
   record names its signer by the `MACULA-KEY-ID-V1` key id of `key`, which
   for a node identity key differs from its node id, and comparing the two
   would refuse every genuine observation.
6. `valid_from` ≤ T ≤ `valid_until`, and the endorsement's window is at most
   30 days.
7. That node id is one of the header's `policy.observers`.
8. `outcomes` has exactly the groups of `expected`, and `state` is what
   16-probes' judgement of `probe` at `probe_version` gives for those
   outcomes against that expectation.
9. The record is the header's own claim. The header's `system` is the
   payload's; the header holds one probe declaration whose `claim` is
   `claim_id`, whose `probe` and `version` are `probe` and `probe_version` (a
   probe and version the verifier's registry knows, 16-probes), and whose
   `expect` is `expected`, the same groups on the same sides; and that
   declaration's target lists `target_address` with `node_id` equal to
   `station_node`; and (v0.5 records) the header's `policy.publish` is the
   payload's `publish`. A revision the verifier cannot obtain refuses here: the
   record's expectation is then the observer's word, not the claim's.

A refusal names the first step that failed. What an accepted observation
proves: an observer this assessment trusts, admitted to the realm, signed at
the time it states that it saw these outcomes from this station, and the
state follows from them. It does not prove:

- that the outcomes are true: they are the observer's word, and a refusal
  among them may have been forged on the path (16-probes);
- that the stated time is true: `created_at` is the observer's clock. A
  realm member endorsement withdrawn inside its window is invisible offline,
  so a withdrawn observer can backdate a record to any time up to the
  endorsement's `valid_until`, at most 30 days.

### Verifying a chain

A verifier given a set of records it holds for one slot (the same signer key
and `subject`) accepts each one by the steps above. Byte-identical records
count once. A record with v0.4's thirteen keys is reported as unchained and
takes no part below. The verifier then **partitions** the rest into chains
by their links: a record follows the record whose wire-bytes SHA-256 is its
`prev`; a record whose `prev` is 32 zero bytes heads a chain; a record whose
`prev` matches no record in the set heads a **fragment**. It reports:

Chains and fragments together are **groups**. Groups are ordered by
`created_at` span. The head of the first group is no gap. A later group
headed at seq 0 is C6. A later group whose head's seq is above the previous
group's highest is a gap in one chain: C1 names every seq between. A later
group whose head is neither is C2. Within a group, a record whose seq is not
one more than the record it follows is C3.

- **C1 gap**: as above.
- **C2 fork**: two different records following the same record, two groups
  whose `created_at` spans overlap, or a later group as above. The observer
  signed two histories; the set is not a chain.
- **C3 broken link**: as above.
- **C4 out of order**: a record whose `created_at` is not after that of the
  record it follows.
- **C5 late**: a record and the one it follows whose `observed_at` are
  further apart than the larger of the two `policy.cadence`s their revisions
  name, plus the smaller of 5 minutes and half that cadence.
- **C6 restart**: as above, a lost place. The report names the hole between
  the groups.
- **C7 signed late**: a record whose revision names `policy.publish`
  `every_result` and whose `created_at` is more than 5 minutes after its
  `observed_at` (under `state_changes` a refresh record is late by design).

C3, C4 and C5 compare only a record with the record it follows. The set is
**continuous** from its first `observed_at` to its last only if every record
is accepted, it shows none of C1 to C7, and every revision it names has
`policy.publish` `every_result`. A chain inside a set with C6 may be
continuous over its own span; the set is not.

What a continuous set proves: the observer signed these observations in this
order, dated them at most one cadence (and its slack) apart, and signed each
within 5 minutes of the time it states it observed. It does not prove that
those dates are true: an observer that fell silent and later signed the
missed rounds with earlier times, both `observed_at` and `created_at`,
produces a continuous set. Only a keeper that keeps the time it fetched each
record, or several keepers compared, shows such a late publication (below).
It does not prove that the observer signed nothing after the last record,
nor anything that an accepted observation does not prove.

## Informative: keeping a history

A record store keeps only the latest record of a slot, so a history exists
only if someone keeps it. A **keeper** fetches each slot from the record
store more often than the cadence, verifies what it fetches, and keeps every
new record with the time it fetched it. A record whose created_at is more
than 5 minutes before the keeper's previous fetch of the slot was published
late: by its own date it existed when the keeper last looked, and was not
there. A keeper reports it. A tombstone found in the slot is kept beside the chain;
the record it withdraws shows as a C1 gap. A keeper that fetches less often than the cadence loses records
that were published, and the gaps it then shows are its own, not the
observer's. Several independent keepers of one slot make a lost or withheld
record visible to every reader who compares them.

**Disclosure.** A keeper that publishes what it keeps publishes every state,
`broken` included, as soon as it fetches it: with a keeper fetching every
15 minutes, a broken claim is public within about 15 minutes of the round
that saw it. An assessment's owner who keeps a public history accepts that,
as the observation spec already requires every state to be published as
observed.

## Informative: how an observation travels

An observation's record bytes are the evidence wherever they are carried:
in a mesh fact, in a DHT slot, in a file. A transport's own signature is not
evidence: macula's publication signature does not reach a subscriber and
lives at most an hour, which is why the observation is its own record.
