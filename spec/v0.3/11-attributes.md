# 11 — Attributes: core five + extensions

An **attribute** is "the thing being protected" in a cell. Attributes are the
rows of the matrix. There is no complete set of attributes accepted by the
industry; fovea standardizes **five core attributes** that are always
assessed, and treats Parker's hexad additions as **extensions** to be
enabled per system.

## Core five (always on)

| Attribute | Definition | Typical cell question |
|---|---|---|
| `confidentiality` | Information is disclosed only to authorized parties. | *Can this adversary read it?* |
| `integrity` | Information and function are unaltered except as authorized. | *Can this adversary change it without being noticed?* |
| `availability` | Information and function are present when authorized parties need them. | *Can this adversary stop authorized use?* |
| `authenticity` | The origin and identity claimed are genuine. | *Can this adversary claim to be someone/something it isn't?* |
| `accountability` | Actions can be attributed to their author afterwards. | *Could this adversary act and leave no trace?* |

The last two are why "CIA" alone was insufficient in v0.1-era thinking: a
mesh built on identities cares about who-spoke, and any governed system
cares about who-can-burn-the-key.

## Extensions

| Attribute | Definition | When to enable |
|---|---|---|
| `possession` | The asset is physically taken but not necessarily read or used. | **Node capture** is a real threat class — kiosks, field devices, adversary physical access. If your hardware can be picked up by an adversary and carried away, enable this. |
| `utility` | The asset exists, is authentic, is intact — and is useless. | Expired token that is still technically valid, certificate for an identity that no longer exists, "we have the data but not the schema". When semantics matter as much as protection. |

Each extension enabled adds a full attribute row across every column: 16
columns × 1 attribute = 16 cells. Enable them because a *documented threat
class* demands it, not because more rows feel more secure.

## Why not the full hexad, always?

Parker's hexad (CIA + authenticity + possession + utility) is the most
complete published attribute list; the "clean" CIA variant and its
"CIA+A2" half-step are more communicable. A straight union of both — treat
everything as always-on — yields two problems: **(a)** utility collapses
into availability in most systems (an intact-but-useless asset *is*
unavailable), and **(b)** the matrix doubles in size for systems where the
additions are inapplicable. The compromise: the attributes are core-of-five
as fixed rows, versions of possession and utility as optional rows whose
omission must be justified. The justification travels in the assessment
header, so a reader always knows *which* attribute regime was in force.

## Renaming policy

Attributes are part of the spec, not of an assessment. A rename or a new
extension is a spec PR and a version bump; see [spec/README](../README.md).
