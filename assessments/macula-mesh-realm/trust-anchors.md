# Trust-Anchor Register — macula-mesh-realm

*Placeholder — the spec's step 1. Nothing on this list yet is binding;
each entry, when a real name/key is assigned, must be confirmed by a
signature from the relevant ceremony before it is considered live.*

| # | Anchor | Type | Present? | Ceremony | Compromise impact |
|---|--------|------|----------|----------|-------------------|
| 1 | *(Realm root CA key)* | Root authority | ☐ | ☐ | Total: every `by_design` claim in the matrix inherits a broken anchor. |
| 2 | *(Org A CA key)* | Org authority | ☐ | ☐ | Scope: all leaf cells under org A. |
| 3 | *(Release/build pipeline key)* | Supply-chain authority | ☐ | ☐ | Scope: all `create` and `acquire` cells. |
| 4 | *(Attestation root: TPM vendor cert chain)* | Hardware root | ☐ | ☐ | Scope: all `deliver`/`decommission` attestation claims (`roadmap` today). |

**Rules:** an anchor is only on this register when its compromise is *total*
(or at least assessment-totalling) — not every sensitive secret, only the
roots from which all else derives. Adding an entry here implies ceremony
documentation exists elsewhere in the artifact set.
