# Changelog

Two version lines live in this repository and are kept apart:

- **Tool** versions (`fovea` CLI and GitHub Action): `vX.Y.Z` git tags,
  reported by `fovea --version`. Consumers pin these by commit sha.
- **Spec** versions (`spec/vX.Y/`): declared by an assessment header as
  `fovea: "X.Y"`. A tool release says which spec versions it reads.

## Unreleased

- **Spec v0.6**: a second probe, `station_release` version 1 (16-probes). A
  station's signed endpoint record names the release it runs; the claim holds
  when that release is a tag of the station's public image whose digest the
  tag's own release build signed (Sigstore keyless). Unlike `kx_group`, a
  refusal breaks it: both sides are authenticated. The assessment format and
  the observation record are unchanged.
- **lint** reads spec 0.6 and refuses `station_release` in a header before
  0.6 (`probe_unknown`): the registry says from which spec version each probe
  may be declared.
- **verify** judges `station_release` records: `signed_release` accepted is
  `holding`, refused `broken`, inconclusive `unknown`.

## Tool 0.3.0 (2026-09-30), reads spec 0.2 to 0.5

Install with `go install github.com/macula-io/macula-fovea/cli/cmd/fovea@v0.3.0`
(module tag `cli/v0.3.0`).

- **`fovea verify --json`**, for one record and with `--chain`: the same
  verdict as JSON on stdout (per record: slot, signer key id, observer, claim,
  target, state and its code, outcomes, observed and signed times, seq and
  prev, the assessment sha; per chain: continuous, the counts, every finding
  by its spec code with gaps as ranges and forks with their hashes, refusals
  by file, and the records in seq order). No booleans, times in ms and ISO
  8601, hex lowercase. Human output and exit codes are unchanged.

## Tool 0.2.0 (2026-09-30), reads spec 0.2 to 0.5

Reads spec 0.4 and 0.5 (below), and adds `fovea verify`: the offline
verifier of a signed claim observation, and with `--chain` the continuity
check of a kept history of them. Install with
`go install github.com/macula-io/macula-fovea/cli/cmd/fovea@v0.2.0` (the Go
module's tag is `cli/v0.2.0`), or pin the Action by the release's sha.

### Spec 0.5 (2026-09-30)

Spec v0.5 lets a kept history of observations be checked for continuity. It
changes only the observation record; an assessment reads exactly as in v0.4,
and a header may declare 0.4 or 0.5.

- **Chained observations** (15-observations): each record carries `seq` and
  `prev` (SHA-256 of the previous record's wire bytes), per observer, claim
  and station. The observer keeps its place across restarts; a lost place is
  a visible restart. A verifier given a kept set reports gaps, forks, broken
  links, out-of-order, late and late-signed records (signed more than 5
  minutes after observed), and restarts; a set is continuous only under
  `policy.publish: every_result`, with no restart in it.
- **Keeping a history** (informative): a keeper fetches each slot more often
  than the cadence and keeps each record's fetch time, which shows a late
  publication; a keeper publishing every 15 minutes makes a `broken`
  claim public within about 15 minutes.
- The lint reads a 0.5 header as 0.4 (`valid_complete_v0_5`).
- **`fovea verify`** (added 2026-09-29, spec v0.4 records): the nine offline
  verification steps of 15-observations against the realm key, the observer's
  realm member endorsement and the assessment revision, read from the git
  history of `--ref`.

### Spec 0.4 (2026-09-29)

Spec v0.4 lets a claim be re-checked, not only written.

- **Evidence on measures:** `doc`, `test`, `scenario` or `probe`.
- **`assessed` needs executable evidence:** a `test`, `scenario` or `probe`
  on one of the cell's measures (`assessed_without_executable_evidence`).
- **`system` is an identifier** (`[a-z][a-z0-9_.-]*`) in a v0.4 header.
- **Targets and a publication policy** in the header: targets name each
  station by its address (an IP literal in one canonical form) and its node
  id; the policy says what an observer publishes, how often it observes (a
  minute to seven days), which claims are suspended, and which observers'
  node ids a reader trusts.
- **Observations** (spec 15): each is a macula record of domain type 0x23,
  signed by the observer's node identity key and bound to the realm by its
  realm member endorsement. It carries `system`, `claim_id`, the station's
  address and `station_node`, the declaration's `expected`, the `outcomes`
  and the `state`. A verifier checks it offline against the realm key and
  the assessment revision it names: that the signer is a listed observer,
  that the state follows from the outcomes, and that the record is that
  revision's own claim. The contract section is marked as the one
  `fovea verify` depends on.
- **The probe registry** (spec 16): `kx_group` v1, one group per attempt;
  an attempt is `accepted` only when the declared station proves its
  identity in its handshake challenge; `refused` is handshake_failure only;
  and a round is judged `holding`, `broken` or `unknown`, with what each
  does not prove.
- 19 new lint rules, each with its case; v0.2 and v0.3 assessments refuse
  the new fields (`field_needs_v0_4`) rather than ignore them.
- The Cucumber report cross-check and `pct_executable_evidence` wait for a
  later spec version; the check, if it lands, is `fovea check-evidence`, since `fovea
  verify` checks signed observations.

## Tool 0.1.0 (2026-09-26), reads spec 0.2 and 0.3

The first tool release. The lint now enforces the spec it implements, and
every rule is tested.

- **Releases and pinning.** `fovea --version` reports the tool version
  (baked at build, `dev` otherwise) apart from the spec versions it reads.
  Every consumer run of the Action checks its own pin: a branch or tag ref
  warns; a sha more than 14 days behind the newest release warns and adds
  a job-summary line; an unknown answer is a notice, never a failure.
  This repo pins its own third-party actions by sha, checks that in CI,
  and runs Dependabot for actions and Go modules.

- **Fixed grid.** A header declares exactly the 16 spec columns, each in its
  own family, plus `x_` extensions; the core five attributes; only
  `possession` and `utility` as extensions, each disabled one with a
  written justification. Dropping, moving, doubling or inventing a column
  or attribute is a lint error.
- **Cell rules.** A present `unassessed` cell is an error. Owner checks
  ignore case and spacing. `roadmap` cells need a definition and
  manifestations; any cell with a `roadmap` measure needs `review_by`.
- **One N/A test.** A whitespace `na_reason` is unjustified in lint, score
  and render alike; the v0.3 grid shows it as unfinished (red).
- **Open gaps** list unassigned owners and overdue roadmaps, and are no
  longer cut at 40.
- **Paths.** `cells_dir` works without a trailing slash; `init` creates it
  when missing and never overwrites an existing cell file.
- **Output.** Diagnostics go to stderr; `score --json` stdout is valid JSON
  even when lint fails. `render --format md|html|json` is accepted as the
  spec writes it. Unknown flags exit 2.
- **Issues bridge.** Refuses to act when the header or a cell fails to
  load (a YAML typo used to close that cell's issue); no token without
  `--dry-run` is an error; 30 s API timeout; pull requests are never edited
  or closed.
- **Action.** Inputs pass through environment variables instead of being
  interpolated into bash (script injection).
- **Removed `schema/`.** No tool loaded the JSON Schemas and they
  contradicted the CLI. The CLI's `lint` is the reference implementation;
  its rule cases under `cli/internal/core/testdata/` seed a language-neutral
  conformance suite. Spec 12 (v0.2 and v0.3) carries an erratum.
- **Spec errata and editorial fixes** (v0.3): 12 status propagation, 13
  unjustified N/A in the grid, 00 "This is v0.3", 11 wording.
- **CI.** gofmt, `go vet`, `go test -race`, build, and an end-to-end run of
  `action.yml` on every push and pull request.
- `.gitignore` no longer hides `cli/cmd/fovea/`.
- **No empty answers** (spec 12 hard rule 1): `measures_empty` and
  `detection_empty` (detection may be empty only with an `na_reason`
  arguing why). `assessed_all_org` no longer passes a measureless cell.
- **Render** runs the full lint: a lint-failing assessment still gets its
  scorecard, but the findings go to stderr and the exit code is 1; v0.3
  open gaps carry the header's lint errors. `render --html` emits HTML for
  0.2 headers too, and its metrics show the real lint error count.
- **Case format**: `expect: {errors: [{rule, where}]}`, compared as an
  exact multiset; every rule code has a case directory named after it, and
  a test keeps it that way. `cells_dir_unreadable` is reported at
  `fovea.yaml`.
- `issues --dry-run` without a token says existing issues were not
  consulted.
- Flags are checked per command (`lint --json` exits 2); `--help` exits 0.
- Spec v0.3 13: the RAG table itself now splits `na` by whether
  `na_reason` is written (erratum).

## Spec 0.3 (2026-09-25)

- Scorecard becomes coverage-aware: per-block `authored/total` next to the
  (unchanged) worst-RAG letter, plus an open-gaps section naming missing /
  unassessed / unjustified-NA / overdue cells (13-scorecard).
- CLI `render` branches on the header's spec version: 0.2 headers keep the
  frozen grid, 0.3 headers get the coverage-aware grid.
- GitHub Action writes artifacts: score JSON + rendered scorecard into an
  artifact directory, plus a step-summary block.

## Spec 0.2 (2026-09-25)

- Initial versioned spec (`spec/v0.2/`): 00-overview, 10-axes, 11-attributes, 12-cell-schema, 13-scorecard, 14-instantiation.
- JSON Schemas for header and cell (removed in tool 0.1.0: unused).
- Dogfood assessment skeleton for `macula-mesh-realm`.
- `packs/` and `templates/` structure.

## Spec 0.1 (never released)

Unnumbered earlier drafts circulated as documents; v0.2 is the first versioned
specification.
