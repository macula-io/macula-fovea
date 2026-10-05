# 16 — Probes

A probe is a check an observer runs against a running system to observe one
claim. This section is the registry: a probe declaration (12-cell-schema)
may name only a probe and version listed here, and an observer implements
exactly the semantics written here. A new probe, or a changed meaning, is a
new version in a new spec version.

| Probe | Version | Declarable from |
|---|---|---|
| `kx_group` | 1 | v0.4 |
| `station_release` | 1 | v0.6 |

Each entry says what the probe observes, what one attempt returns, what an
expectation may contain, and how a round of attempts is judged into a claim
state (00-overview): `holding`, `broken` or `unknown`.

## `kx_group`, version 1

**Observes** a `macula_station` target, station by station: which key
exchange groups the station declared at an address completes a QUIC/TLS 1.3
handshake on, as that station.

**One attempt** offers exactly one group, because the group a QUIC handshake
settled on cannot be read by the client; an attempt that offers one group and
completes proves that group was negotiated. Everything but the offered group
is what a real macula client sends (ALPN `macula`, TLS 1.3, ML-DSA-87
signature verification of the station's key), so a refusal differs from a real
client's handshake only in the group. An attempt returns one of:

| Outcome | Meaning |
|---|---|
| `accepted` | The handshake completed on the one group offered, **and the answering endpoint proved it is the station the target declares at this address** (below). |
| `refused` | The endpoint ended the handshake with TLS alert handshake_failure (40), and only that alert. |
| `inconclusive` | Anything else: no answer in time, another alert (a stack may send insufficient_security, 71, for no common group), another close, a failure on the observer's side, or a completed handshake that did not prove the declared station. |

**Proving the station.** Completing TLS proves only that the endpoint holds
*some* ML-DSA-87 key. After it, the observer sends the opener a macula client
sends on its control stream and reads the station's `challenge`, as macula's
`plans/DESIGN_PQ_HANDSHAKE_FRAMES.md` specifies it. The attempt is `accepted`
only if, exactly as a macula client checks them:

- the challenge's `identity_key` derives to the node id the target declares
  for this address (14-instantiation);
- its `tls_binding` verifies under that key, names that node id, is for the
  leaf certificate this connection presented, and is within its
  `not_before`/`not_after` at the time of the attempt;
- its `tls_status` verifies under that key, is for that binding, and is
  within its `issued_at`/`expires_at` at the time of the attempt;
- and every other check a macula client makes on a challenge passes, among
  them that the leaf does not carry the identity key itself (key purpose
  reuse) and that the binding's `use` is `tls`. The observer then closes without sending CONNECT, so it
never asks the station to admit it: the probe never carries a session.

**What a station sees.** A classical attempt ends inside TLS. A post-quantum
attempt is a connection that sent the opener, received the challenge and
closed before CONNECT; stations count and log it as such. At the policy's
cadence (14-instantiation) that is a handful of connections per station per
round, which is not an attack and should not be read as one.

**Vocabulary.** An expectation names groups from this list, each at most
once across `accepted` and `refused`, and an expectation with `refused`
groups names at least one `accepted` group (see the asymmetry below: a
refusal counts only beside an acceptance):

| Group | Kind |
|---|---|
| `secp384r1_mlkem1024` | post-quantum hybrid |
| `secp256r1_mlkem768` | post-quantum hybrid |
| `x25519` | classical |
| `secp256r1` | classical |
| `secp384r1` | classical |

**A round** attempts every expected group once against one station of the
target (its address, as that node id). It is judged:

| State | When |
|---|---|
| `broken` | A group expected `refused` was `accepted`. |
| `holding` | Every group expected `accepted` was `accepted`, every group expected `refused` was `refused`, and at least one group was `accepted`. |
| `unknown` | Otherwise. |

Why the rules are asymmetric:

- **An acceptance is authenticated; a refusal is not.** An accepted attempt is
  bound to the declared station by its identity key and TLS binding. A refusal
  travels in a QUIC Initial-space close that anyone on the path who sees the
  connection id can forge. So only an acceptance can break a claim, and a
  refusal can only support one. That makes `holding` as strong as the path
  between observer and station: an on-path party that forges handshake_failure
  for the classical attempts makes a false `holding`. `broken` it cannot make.
- **handshake_failure has three causes.** A station sends it when no key
  exchange group, no signature scheme or no cipher suite is in common. A
  classical group's `refused` is therefore evidence about key exchange only in
  a round where a post-quantum group was `accepted` from the same station,
  which proves the signature scheme and cipher suite are shared and leaves the
  group as the only difference. `holding` requires exactly that.
- **Silence is not a verdict.** A round with any `inconclusive` attempt, or
  a group expected `accepted` that was `refused`, is `unknown`, never
  `holding`.

A target with several stations is observed station by station: each is its
own round and its own observation (15-observations).

**What "only" covers.** `holding` says the station refused every classical
group the expectation lists and completed every post-quantum one it lists;
it says nothing about a group the vocabulary does not name (x448, secp521r1,
the ffdhe groups, a pure ML-KEM group). An assessment claims "only
post-quantum" in the sense of the groups it expects, and a new group enters
the vocabulary in a new version of this probe.

## `station_release`, version 1 (from v0.6)

**Observes** a `macula_station` target, station by station: whether the
release a station says it runs is a signed public release. Only a header that
declares `fovea: "0.6"` or later may declare it.

**One attempt** per station takes three steps, in order:

1. **What the station says.** The observer fetches, from the mesh DHT, the
   station endpoint record (macula record type 0x12) stored under the
   endpoint key of the node id the target declares for this address. The
   record must verify as macula verifies a record, at the time of the attempt,
   and its signer's key id must be that node id. Its `station_version`, text
   of 1 to 64 bytes, is the release the station names: call it V.
2. **The release.** The observer asks the registry `ghcr.io`, anonymously and
   over TLS, for the manifest of `ghcr.io/macula-io/macula-station:V`,
   accepting the OCI image index, Docker manifest list and OCI and Docker
   image manifest media types, and takes the digest of the manifest it serves
   for the tag: the tag's top-level descriptor, what `cosign verify`
   resolves.
3. **The signature.** The observer verifies a Sigstore keyless signature on
   that digest, against Sigstore's public trust root and transparency log (as
   `cosign verify` does), whose certificate names all of:
   - OIDC issuer `https://token.actions.githubusercontent.com`;
   - identity `https://github.com/macula-io/macula-ci-images/.github/workflows/attest-image.yml@`
     followed by any ref (the release workflow that signs every
     macula-station image);
   - GitHub workflow repository `macula-io/macula-station`;
   - GitHub workflow ref `refs/tags/v` followed by V: the build of that
     release tag, and no other build.

   The certificate binds the caller (the station's repository and its tag),
   not the contents of the reusable workflow it called: whoever can change
   `attest-image.yml` in `macula-io/macula-ci-images` is in the trust base of
   this probe, beside Sigstore.

An attempt returns, for its one vocabulary entry `signed_release`:

| Outcome | Meaning |
|---|---|
| `accepted` | All three steps held. |
| `refused` | Step 1 held, and the station's signed V is no signed release: V cannot be an OCI tag, or the registry answered with the OCI distribution error code `MANIFEST_UNKNOWN` for the tag, or the digest carries no signature, or none whose certificate names the four values above. |
| `inconclusive` | Anything else: no endpoint record, a record that does not verify or that another key signed, a record without `station_version`, any other registry answer (`NAME_UNKNOWN`, `DENIED`, `UNAUTHORIZED`, a status without an error code: a package made private or moved is not a station's doing), a trust root or transparency log that did not answer or answered otherwise, an observer-side failure, no answer in time. |

**Vocabulary.** `signed_release`, the only entry. An expectation names it as
`accepted`, and names nothing `refused`: expecting a station to run no signed
release is not a claim this probe can hold.

**A round** is one attempt against one station of the target. It is judged:

| State | When |
|---|---|
| `holding` | `signed_release` was `accepted`. |
| `broken` | `signed_release` was `refused`. |
| `unknown` | `signed_release` was `inconclusive`. |

**Why a refusal can break this claim** (where a `kx_group` refusal cannot):
both of its sides are authenticated. V is under the station's own signature,
bound to the declared node id; the absence of the tag is the registry's answer
over TLS, and a missing or non-matching signature is a cryptographic finding,
not a network event. Whatever the observer could not finish is `inconclusive`,
never `refused`.

**What it says, and what it does not.** `holding` says the station names, under
its signature, a release whose image the station's own release build signed
for that tag. It does not say the bytes the station runs are that image: the
release is self-attested, and only remote attestation would prove the running
code. It does say that a station naming an unreleased, unsigned or re-tagged
build is seen, and signed, as `broken`; a git tag moved and built again is a
new, correctly signed release under the same name, and holds. It is as strong
as Sigstore's trust root, the release workflow's repository and the
registry's TLS.

**What a station sees.** Nothing: the probe reads the DHT and the registry and
never connects to the station itself.

A target with several stations is observed station by station: each is its
own round and its own observation (15-observations).
