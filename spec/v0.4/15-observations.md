# 15 — Observations

A probe declaration (12-cell-schema) names a claim; an **observer** checks it
on the running system, on the policy's cadence (14-instantiation), and turns
each round into an **observation**: one claim, one station, one state
(`holding`, `broken`, `unknown`; 00-overview), judged as the probe's registry
entry says (16-probes). This section defines the observation as a signed
record, so that anyone holding the realm's public key can verify it offline,
without the mesh, the observer or the assessment's repository.

## What an observer does

- It observes only claims of an assessment revision it has adopted, and not
  the claims the policy suspends.
- It judges each round exactly as the probe's entry in 16-probes says.
- It signs **every** observation. `policy.publish` decides only which ones
  it publishes (14-instantiation).
- It never changes a cell: the authored status stays what the assessment says.

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
| `system` | text | The assessment's `system`, as its header writes it. |
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

No value is a boolean or a float: 0 and 1 stand in for yes and no.

### Verification

A verifier holds, beside the observation's record bytes:

- the realm's **public key** as carried, the **profile** (`pq_hybrid` or
  `pq_pure`) and the realm's **name**;
- the observer's **realm member endorsement** record bytes (macula record
  type `0x05`, signed by the realm key). The realm publishes it in the DHT
  under the member's endorsement slot and renews it before it expires; an
  observer may ship it beside its observations;
- the **observer node ids** it trusts for this assessment: the header's
  `policy.observers` (14-instantiation) of the revision the observation names.

It accepts the observation only if every step holds, at **T = the
observation's `created_at`**, so an observation stays verifiable after the
endorsement that covered it has expired:

1. The observation verifies as a macula record at time T, and its `type` is
   `0x23`; and its `created_at` is at most 5 minutes after the verifier's own
   clock (verifying at T alone would never see a record dated in the future).
2. Its payload has exactly the keys and types above, and no other: text keys,
   each once. `state` is 0, 1 or 2; every `expected` value is 0 or 1; every
   `outcomes` value is 0, 1 or 2; `observed_at` is not after `created_at`;
   and its `subject` is `system`, 0x00, `claim_id`, 0x00, `target_address`.
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
7. That node id is one of the observer node ids the verifier trusts.
8. `outcomes` has exactly the groups of `expected`, and `state` is what
   16-probes' judgement of `probe` at `probe_version` gives for those
   outcomes against that expectation.

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

## Informative: how an observation travels

An observation's record bytes are the evidence wherever they are carried:
in a mesh fact, in a DHT slot, in a file. A transport's own signature is not
evidence: macula's publication signature does not reach a subscriber and
lives at most an hour, which is why the observation is its own record.
