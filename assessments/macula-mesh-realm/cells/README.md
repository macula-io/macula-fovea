# Cells

One YAML file per `column.attribute` the header (`../fovea.yaml`) declares in
force, named `<column>.<attribute>.yaml`, validated by
[`../../../../schema/cell.schema.json`](../../../../schema/cell.schema.json).
Contents per cell obey [spec/v0.2/12-cell-schema.md](../../../../spec/v0.2/12-cell-schema.md).

`fovea init` (planned CLI) generates the full grid of empty skeletons with
status `unassessed` from the header — 16 columns × 5 core attributes, plus 16
more because `possession` is enabled here. The current seed files
(`in_motion.confidentiality.yaml`) are hand-written worked examples to
anchor the style: **statuses are honest**, i.e. `assumed` (documented but
unverified against a running mesh) until tested, `roadmap` where the defense
is announced but unbuilt, `by_design` only with a `source` pointing at the
authority (spec text or sibling-repo docs) that guarantees it.

Coverage is a lint condition — cells are *not all expected* to be filled
before the grid exists, but the grid *does* have to exist before the
assessment can be read.
