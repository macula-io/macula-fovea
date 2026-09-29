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

**Observes** a `macula_station` target: which key exchange groups the
station completes a QUIC/TLS 1.3 handshake on.

**One attempt** offers exactly one group, because the group a QUIC handshake
settled on cannot be read by the client; an attempt that offers one group and
completes proves that group was negotiated. Everything but the offered group
is what a real macula client sends (ALPN `macula`, TLS 1.3, ML-DSA-87
signature verification of the station's key), so a refusal differs from a real
client's handshake only in the group. An attempt returns one of:

| Outcome | Meaning |
|---|---|
| `accepted` | The station completed the handshake on the one group offered, and proved possession of its ML-DSA-87 key. |
| `refused` | The station ended the handshake with TLS alert handshake_failure (40), and only that alert. |
| `inconclusive` | Anything else: no answer in time, another alert, another close, or a failure on the observer's side. |

An observer closes an accepted connection at once: the probe never carries a
session.

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

**A round** attempts every expected group once against one address of the
target. It is judged:

| State | When |
|---|---|
| `broken` | A group expected `refused` was `accepted`. |
| `holding` | Every group expected `accepted` was `accepted`, every group expected `refused` was `refused`, and at least one group was `accepted`. |
| `unknown` | Otherwise. |

Why the rules are asymmetric:

- **An acceptance is authenticated; a refusal is not.** The station signs an
  accepted handshake with its ML-DSA-87 key. A refusal travels in a QUIC
  Initial-space close that anyone on the path who sees the connection id can
  forge. So only an acceptance can break a claim, and a refusal can only
  support one.
- **handshake_failure has three causes.** A station sends it when no key
  exchange group, no signature scheme or no cipher suite is in common. A
  classical group's `refused` is therefore evidence about key exchange only in
  a round where a post-quantum group was `accepted` from the same station,
  which proves the signature scheme and cipher suite are shared and leaves the
  group as the only difference. `holding` requires exactly that.
- **Silence is not a verdict.** A round with any `inconclusive` attempt, or
  a post-quantum group refused, is `unknown`, never `holding`.

A target with several addresses is observed address by address: each
address is its own round and its own observation (15-observations).
