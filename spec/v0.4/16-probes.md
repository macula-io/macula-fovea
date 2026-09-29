# 16 — Probes

A probe is a check an observer runs against a running system to observe one
claim. This section is the registry: a probe declaration (12-cell-schema)
may name only a probe and version listed here, and an observer implements
exactly the semantics written here. A new probe, or a changed meaning, is a
new version in a new spec version.

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
