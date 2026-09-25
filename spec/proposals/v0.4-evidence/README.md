# Proposal — v0.4: executable evidence (the Cucumber link)

**Status: PROPOSED. Not normative.** Nothing in this directory is enforced by
the current CLI; no header may declare `fovea: "0.4"` yet. It exists so the
evidence model can be argued, amended, and — after at least one dogfood
demonstrates it — promoted into `spec/v0.4/` proper.

## The problem this solves

`assessed` ("verified by inspection of code, configuration, or test") is a
claim with no re-check: a cell goes green once, a defense regresses, the
green stays. Fovea's weakness is verification honesty — which is exactly the
one thing Cucumber/Gherkin exists to provide: documentation that stays true
because it is executed.

## The link, in one sentence

**Cells specify the security property; scenarios execute it; the CI report
is the only thing allowed to call a cell `assessed`.**

| Cell field | Gherkin counterpart |
|---|---|
| threat definition | the Scenario's stated purpose |
| manifestations | `Given` — the system in the vulnerable state |
| countermeasure (`by_design`) | `Then` — the property holds, the attack fails |
| the adversary's move | `When` |

## Proposed changes

1. **Evidence kinds on measures** (schema addition):

   ```yaml
   countermeasures:
     - measure: Retracted clips are unservable everywhere.
       status: by_design
       source: security/scenarios/retraction.feature
       evidence:
         - kind: scenario
           ref: retraction.feature  # feature file + scenario name
           runner: godog            # godog | whitebread | cucumber
         - kind: test               # executable, CI-run
           ref: apps/query_tube/test/retraction_tests.erl
         - kind: doc                # the fallback: a citation, not proof
           ref: README.md#retraction
   ```

2. **Status rule (promoted):** `assessed` requires at least one *executable*
   evidence kind (`test` or `scenario`) whose CI run passed. A cell with only
   `doc` evidence is `assumed`, whatever its author believes.

3. **`fovea verify <dir> --report cucumber.json`** — consumes the standard
   Cucumber JSON report, cross-references every `scenario` evidence ref, and
   fails the build if a scenario named by an `assessed` cell is missing or
   failed. A failed scenario is a lint error, not a scorecard shade.

4. **New headline metric:** `pct_executable_evidence` — measures with
   test/scenario evidence ÷ all `by_design` measures. Amber-to-green progress
   becomes measurable per assessment.

## The honest boundary

Scenario evidence applies to **behavioral** properties: authorization,
retraction enforcement, rate limits, scan gates, boot refusal, revocation,
content-address verification. It does **not** apply to physical seizure,
socio-legal compulsion, insider culture, or harvest-now/decrypt-later — those
stay `doc`/`test` evidenced. The framework must never pretend a `.feature`
file can prove physical custody.

## Open questions for the review

1. Should a *failing* scenario demote the cell to `roadmap` (state change
   authored by CI), or only block the build? (Proposal: block only — CI
   writing statuses is a bigger step than CI reading reports.)
2. Runner matrix: godog (Go) for the CLI, WhiteBread (Erlang) for the
   services, plain Cucumber for polyglot repos — one `runner` field is
   enough, but who runs what in CI is deploy-specific.
3. Does `test` evidence need the same report cross-check (e.g. TAP/CT logs),
   or is scenario-report consumption the only v0.4 gate?

## Worked example

```gherkin
# security/scenarios/retraction.feature — evidence for operate.confidentiality
Scenario: A retracted clip is unservable everywhere
  Given a published clip with mcid "abc"
  And the owner retracts it
  When a consumer looks the clip up
  Then the lookup returns not_found
  And the watch stream refuses the mcid
```

Passing this in CI is what turns mcl-tube's sharpest roadmap cell green —
automatically, by evidence, not by editing a YAML status.
