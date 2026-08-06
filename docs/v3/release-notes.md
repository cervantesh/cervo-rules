# v3 Release Candidate Notes

Notes for the current release candidate, followed by the ones it supersedes.
Every entry here remains a release candidate, not GA final.

## v3.0.0-rc.5

`v3.0.0-rc.5` makes a policy able to state the conditions under which it
authorizes, and makes the previous release's guard seam actually usable.

### Compound predicates over typed facts

A v3 policy could say which operation goes to which target. It could not say
under what numeric conditions: every predicate over facts lived in hand-written
consumer Go, outside the policy and outside the `PolicyHash`.

- The vocabulary declares a fact's name and type in a new `facts:` block; the
  policy declares its `min`, `max` and `default` in its own `facts:` block.
- Routes and denies take a `when:` predicate. Composition is `all` / `any` /
  `not`, nestable, over leaves that compare one fact against a literal, against
  another fact of the same type, or consult a named condition.
- Every threshold lives in the policy file, over whose raw bytes the
  `PolicyHash` is taken, so retuning a limit is an auditable change.
- Predicates compile to Go boolean expressions in the generated policy. There is
  no runtime evaluator and no new dependency: the grammar is a closed tagged
  form, deliberately too small to express anything but a guard.
- A fact that is absent with no declared default, unparseable, non-finite, or
  outside its declared bounds fails the decision with the new
  `missing_fact` / `invalid_fact` codes. It is never reported as a predicate
  that did not hold.
- Generated engines populate `DecisionTrace.Steps` with the rule id and the leaf
  that decided it. Trace stays opt-in per decision.

See [compound-predicates.md](compound-predicates.md) and
[ADR 0016](../adr/0016-compound-predicates-over-typed-facts.md).

### Fixes that make earlier releases usable

- Generated `mergeRuntimeConfig` dropped `PolicyRuntimeConfig.Conditions`, and
  since `DefaultConfig` cannot supply an evaluator, every condition-gated policy
  failed to build. The `conditions` / `requires` feature added in rc.4 was
  unusable end to end.
- `generate -test-out` emitted a test file that did not compile for any policy
  without a `tests:` block, and a test that always failed for any policy
  declaring `conditions:`.

### Breaking for malformed policies

Each closes a case where a policy generated cleanly but could not be audited or
could not be built. None affects a policy that was already correct.

- A deny naming an operation absent from the vocabulary is rejected. It
  previously generated, compiled, and never fired.
- Two routes on one operation are rejected at `check`. The generator previously
  emitted a Go map literal with duplicate constant keys, so only the consumer's
  build failed.
- `deny.id` is required. Since `reason` falls back to `id`, a deny with neither
  denied every operation with an empty reason and an unnamed trace step.

### Other changes

- `denies` is an ordered list evaluated in authored order, first match wins.
  `deny.operation` is optional; absent applies the rule to every operation.
- `cervorules-vocabgen` emits a `Fact*` constant per declared fact.
- `testkit.RuntimeCase` gains `WantErrorCode` and `WantTraceStepNames`, so a
  consumer can certify the refusal path and the explanation, not only the happy
  decision.
- CI validates every shipped policy document against the published JSON Schemas
  (`scripts/ci/validate-schemas.py`). Nothing previously read the schemas, so
  schema/parser divergence was invisible; the divergences that existed are
  closed.

## v3.0.0-rc.4

Issues: symbolic guard layer, caller-owned error redaction.

- Added the optional `ontology` package: entity types, predicate domain/range
  signatures, disjoint type sets, functional properties, and declarative
  lifecycles with terminal-state detection. See
  [ADR 0014](../adr/0014-symbolic-guard-layer.md).
- Added `core.Condition`, `core.Conditions`, `core.ConditionFunc` and
  `core.ConditionSet`, plus `conditions:` and `requires:` in the DSL. Note the
  rc.5 fix above: the seam did not work in rc.4.
- **Breaking.** `core.Error` no longer infers value sensitivity from the field
  name; redaction is explicit via `Sensitive` or caller-owned via
  `core.RedactFields`. See
  [ADR 0015](../adr/0015-caller-owned-error-redaction.md).

## v3.0.0-rc.3

Issues: #398, #403, #472, #473, #474, #478, #480, #481, #487, #489.

Repository organization, package-script layout, and v3 root-boundary hardening.

- v3 docs live behind topic indexes in `docs/v3`, `docs/performance`,
  `docs/change-management`, `docs/operations`, and `docs/reports`.
- Operational scripts are grouped by domain under `scripts/ci`,
  `scripts/release`, `scripts/performance`, and `scripts/sonar`.
- v3 lives at the repository root after the physical v2/v3 split.
- v2 maintenance moved to `CervoSoft/cervo-rules-v2` with final cut `v2.1.0`.
- Future breaking work will not carry v2 compatibility shims forward.

## Breaking Changes Since v2

- Public primitive names are `Operation`, `Target`, and `Executor`.
- v3 removes public `Capability`, `Service`, and `Provider` aliases.
- Generated policies use `runtime.PolicyFactory`; `BuildPolicy` is not part of
  the v3 generated runtime API.
- v3 DSL field names are `operation`, `target`, and `executor`.
- `RoutingPhase` is removed from v3; indexed and linear routing are explicit.
- Facts are optional and budgeted; trace is opt-in.
- Observability reports are versioned and low-cardinality by default.

## Machine-Readable Contracts

- v3 schema bundle under `schemas/v3`.
- Public API inventory in `docs/v3/public-api-inventory.json`.
- Legacy migration report command remains available from `CervoSoft/cervo-rules-v2`.
- Consumer conformance contract in `testkit`.
- Package release marker: `release_module=github.com/cervantesh/cervo-rules/v3`.

## Verification Before Tag

```bash
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
scripts/release/check.sh v3.0.0-rc.5 dist-release-check-v3.0.0-rc.5
```

Windows PowerShell operators can run the same check through:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/release/check.ps1 v3.0.0-rc.5 dist-release-check-v3.0.0-rc.5
```

## Verification After Package Publish

```bash
scripts/release/verify-generic-package.sh v3.0.0-rc.5
scripts/release/verify-oci-tools.sh v3.0.0-rc.5
```
