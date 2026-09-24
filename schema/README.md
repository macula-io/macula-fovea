# schema/

JSON Schemas (Draft 2020-12) validating the fovea header and cell files
**after YAML→JSON conversion** — i.e., by the `fovea` CLI or any
independent validator that converts first.

| File | Validates |
|---|---|
| [`assessment.schema.json`](assessment.schema.json) | `fovea.yaml` header of an assessment |
| [`cell.schema.json`](cell.schema.json) | One `.yaml` cell file |

Conditional requirements (`na_reason` if `na`, `review_by` if `roadmap`,
`source` if `by_design`) are expressed as `if/then` in the schemas; lint
enforces them regardless.
