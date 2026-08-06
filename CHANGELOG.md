# Changelog

All notable changes to CervoRules will be documented in this file.

## v3.0.0-rc.5 - 2026-08-05

### Added

- Added compound predicates over typed facts to the v3 policy DSL. The
  vocabulary declares a fact's name and type in a new `facts:` block; the policy
  declares its bounds and default in its own `facts:` block, and attaches a
  `when:` predicate to a route or deny. Composition is `all` / `any` / `not`,
  nestable, over leaves that compare one fact against a literal, against another
  fact of the same type, or consult a named condition. See
  [docs/v3/compound-predicates.md](docs/v3/compound-predicates.md).
- Predicates compile to Go boolean expressions in the generated policy. There is
  no evaluator at runtime and no new dependency: the grammar is a closed tagged
  form the JSON Schema validates, deliberately too small to express anything but
  a guard.
- Every threshold, bound and default lives in the policy file, over whose raw
  bytes the `PolicyHash` is taken. Editing a limit moves the hash.
- Added `core.ErrorCodeMissingFact` and `core.ErrorCodeInvalidFact`. A fact that
  is absent with no declared default, unparseable, non-finite, or outside its
  declared bounds fails the decision with a structured error rather than
  reporting a predicate as unsatisfied. `strconv.ParseFloat` accepts `"NaN"`,
  and a non-finite value passes every comparison without matching any, so the
  generated parser rejects it explicitly.
- Generated engines now populate `DecisionTrace.Steps`: one step per evaluated
  rule carrying the rule id, whether it matched, and which leaf decided it.
  Trace remains opt-in per decision, so an untraced decision allocates nothing
  for explanation.
- `cervorules-vocabgen` emits a `Fact*` constant per declared fact, so a
  misspelled metadata key in a consumer is a compile error.
- Added a `facts` count to the `check` snapshot.

### Changed

- **Breaking for malformed policies.** A deny naming an operation the vocabulary
  does not declare is now rejected at generation time. Previously it generated,
  compiled, and never fired: a typo in a deny rule was a silent fail-open.
- **Breaking for malformed policies.** Two routes on one operation are now
  rejected at `check` time. Previously the generator emitted a Go map literal
  with duplicate constant keys, so `check` and `generate` both passed and only
  the consumer's build failed.
- `denies` is now an ordered list evaluated in authored order, first match wins,
  instead of a map keyed by operation. A policy can declare several denies for
  the same operation, which is what a disjunction split across two rules needs.
- `deny.operation` is now optional. An absent operation applies the deny to
  every operation in the vocabulary.
- `deny.reason` defaults to the rule's `id` when absent. Previously such a deny
  produced an empty `Decision.Reason` in the audit record.
- **Breaking for malformed policies.** `deny.id` is now required. Since `reason`
  falls back to `id`, a deny with neither denied every operation with an empty
  reason and an unnamed trace step. The id is what a trace step reports and what
  an audit record keys on, so a deny that cannot be named is not auditable.
- A route's decision trace step is now named with the route `id`, falling back
  to the operation. It previously carried `reason`, which falls back to the
  literal `"route matched"` — a name that identifies nothing.

### Fixed

- Fixed `generate -test-out` emitting a test file that did not compile for any
  policy with no `tests:` block: the engine variable and the `core` import were
  emitted unconditionally but only referenced by generated cases.
- Fixed the generated test always failing for any policy declaring
  `conditions:`. It asserted that `DefaultConfig` validates, which a
  condition-gated policy can never do. It now asserts the fail-closed contract
  instead, then supplies a stub evaluator answering true for each declared
  condition so the declarative cases can run.
- Fixed generated `mergeRuntimeConfig` dropping `PolicyRuntimeConfig.Conditions`
  from the caller's config. Since `DefaultConfig` cannot supply an evaluator,
  the merged config always had none and `ValidateConfig` then rejected the
  build, which made every condition-gated policy impossible to build. The
  `conditions` / `requires` feature added in rc.4 was unusable end to end.

## v3.0.0-rc.4 - 2026-08-05

### Added

- Added the optional `ontology` package: a symbolic guard layer with entity
  types, predicate domain/range signatures, disjoint type sets, functional
  properties, and declarative lifecycles with terminal-state detection. See
  [ADR 0014](docs/adr/0014-symbolic-guard-layer.md).
- Added `core.Condition`, `core.Conditions`, `core.ConditionFunc` and
  `core.ConditionSet`: a named-guard seam a policy can consult before allowing
  an operation. Unknown or unwired conditions fail closed with a structured
  error rather than reporting `false`.
- Added `ontology.Guard`, which implements `core.Conditions` over an ontology
  and a caller-supplied snapshot resolver. `Guard.Explain` returns the
  violations behind a refusal; `Holds` returns the same answer without them.
- Added `conditions` and `requires` to the v3 policy DSL. A route or deny lists
  the named guards that must hold for it to apply, and `policygen` rejects a
  rule requiring an undeclared condition at generation time, so an unwired guard
  fails CI instead of a production decision.
- Added `runtime.PolicyRuntimeConfig.Conditions`. A generated policy that
  declares conditions but is built without an evaluator is rejected by
  `ValidateConfig` rather than silently allowing what the guards exist to stop.
- Added `ontology.WithRequestScope`, which resolves one snapshot per request
  instead of one per consulted condition, and refuses reuse across requests.
- Added `ontology.Constraint` and `Ontology.Custom` so consumers can add
  constraint families without waiting for upstream.
- Added the `guarded-refund` example and
  [docs/v3/symbolic-guards.md](docs/v3/symbolic-guards.md).

### Changed

- **Breaking behavior change.** `core.Error` no longer infers value sensitivity
  from the field name. Redaction is now explicit via the `Sensitive` marker, or
  caller-owned via `core.RedactFields` with `Error.RedactWith` /
  `Errors.RedactWith`. The previous hardcoded list put HTTP- and AI-shaped
  vocabulary inside a domain-neutral package. See
  [ADR 0015](docs/adr/0015-caller-owned-error-redaction.md) for migration.
- `core.Error.Error()` now withholds a value marked `Sensitive`. Previously the
  string form printed values that JSON serialization redacted, so logging an
  error bypassed redaction.

### Fixed

- Fixed a redaction false positive where the field `max_tokens` matched the
  substring `token` and had its value replaced with `[REDACTED]`, hiding a
  legitimate limit value. Field matching is now segment-based.

### Added (adapters)

- Added `httpadapter.Redactor()` and `httpadapter.SensitiveFieldNames()` so
  transport-shaped field names live with the transport adapter that owns them.

## v3.0.0-rc.3 - 2026-05-24

- Organized v3 documentation into topic directories with README indexes for
  v3 contracts, change management, performance, operations, and reports.
- Organized operational scripts by domain under `scripts/ci`, `scripts/release`,
  `scripts/performance`, and `scripts/sonar`.
- Enforced the v3 root package as marker-only with repoquality tests so v3 does
  not grow a compatibility facade.
- Consolidated the v2 root facade into a single `facade.go` file while keeping
  the existing v2 import path compatible.
- Moved repository schema validation tests into `internal/repoquality`.
- Recorded that the next breaking line will not carry v2 compatibility shims or
  root-facade behavior forward.

- Prepared v3 final docs for the `v3.0.0-rc.1` target, including
  `docs/v3/migration-v2-to-v3.md`, `docs/v3/api-reference.md`,
  `docs/v3/wiki-refresh.md`, and `docs/v3/release-notes.md` (#398-#403, PR
  pending).
- Added v3 release/package verification metadata so pre-release artifacts record
  `release_module=github.com/cervantesh/cervo-rules/v3`, include
  `release-dependencies.txt`, package `schemas/v3`, and verify v3 schemas in
  generic packages and OCI images (#392-#397, PR #437).
- Added v3 consumer conformance contracts and neutral fixtures in `v3/testkit`
  for generated-policy factories, facts adapters, package smoke, and a
  proxy-shaped neutral fixture (#382-#387, PR #435).
- Document PR #247 performance/routing/facts evidence, issue #248 bounded
  runner permission repair, and issue #249 repo/wiki documentation refresh
  workflow (#247, #248, #249, PR pending).
- Add optional SonarQube installation, scanner config, CI workflow, and
  documentation for CervoRules quality analysis (#242, PR pending).
- Update the local SonarQube compose image from inactive `lts-community` to the
  active `community` tag, add a guarded local reset helper, and guard it with
  tests (#244, PR pending).
- Added `AGENTS.md`, `.cervorules/agent-manifest.json`, and
  `docs/agent-quickstart.md` so AI agents can discover current version,
  package boundaries, required checks, policy workflows, and agnosticism rules
  without conversation context (#231-#238, PR pending).
- Refreshed README current-version language from `v2.0.0` to
  `v2.1.0-rc.1` and added drift tests for agent entrypoints, machine manifest,
  quickstart coverage, and agnosticism guidance (#233, #237, PR pending).
- Added automated RC docs drift checks for roadmap status, repo/wiki report
  inventory, snippet policy, changelog issue/PR references, and maturity
  evidence citations (#221, parent #214, PR pending).

## v2.0.1 - 2026-05-22

- Added modular public package guidance for selective imports through `core`, `runtime`, `limits`, `httpadapter`, `observe`, and `testkit`.
- Updated generated policy and vocabulary tooling to prefer modular v2 imports for generated consumers.
- Fixed `cervorules-vocabgen` default imports so generated vocabularies use `github.com/cervantesh/cervo-rules/v2/core`.
- Fixed packaged CLI schema discovery so release archives can find schemas beside the extracted executables.
- Verified CervoProxy against the modular v2 APIs before release promotion.

## v2.0.0 - 2026-05-21

- Changed the Go module path to `github.com/cervantesh/cervo-rules/v2`.
- Added structured `ValidationError` and `ValidationErrors` with stable codes and fields.
- Changed `CheckLimits` to return all `LimitViolations` instead of a single error.
- Changed `Engine.Decide` to return `DecisionResult` with decision, trace, observation, and diagnostics.
- Made predicates and actions context/error aware.
- Added generated `PolicyFactory` with explicit default config, validation, metadata, and context-aware build.
- Added error-first generated policy testkit helpers.
- Changed `Vocabulary` to an interface and added `StaticVocabulary` as the fixed-set implementation.
- Added `RuntimeOption` helpers and `NewPolicyRuntimeConfig`.
- Added v2 consumer adoption fixtures covering generated policies, HTTP classification, runtime options, disabled route enablement, `DecisionResult`, and limit checks.
- Verified CervoProxy migration to CervoRules v2 before release packaging.

## v0.1.2 - 2026-05-20

- Added tag-based the package registry publishing for CLI tools, schemas, checksums, and optional `cervorules-tools` OCI image.
- Added package consumption, release verification, smoke package, and cleanup documentation for packages.
- Added `PolicyRuntimeConfig` and `ValidatePolicyRuntimeConfig` for generated policy startup overrides.
- Added generic HTTP request classification helpers for consumers that want deterministic `RequestFacts` extraction.
- Updated generated policies to accept optional runtime overrides through `BuildPolicy(overrides ...PolicyRuntimeConfig)`.
- Added declarative `disabled_by_default` routes that can be enabled by runtime capability route overrides.
- Added public `MergePolicyRuntimeConfig`, precompiled `HTTPClassifier`, generated policy testkit contracts, and runtime env mapping documentation.
- Added stack-neutral `DecisionObservation` documentation and stable observability contract tests.
- Added generated example consumer verification that generates every example into a temporary module and runs generated tests.

## v0.1.1 - 2026-05-06

- Fixed policy generator output for audit class constants.

## v0.1.0 - 2026-05-06

- Added JSON schemas and in-tool JSON Schema validation for policy vocabulary and policy rules DSL files.
- Added `cervorules-policygen check` and `explain` workflows for agent-safe validation and dry runs.
- Added composable YAML predicates: `all`, `any`, `not`, `capability`, `capability_in`, `user`, `user_in`, `trusted_user`, `risk`, `risk_in`, `metadata_exists`, `metadata_equals`, and `service_healthy`.
- Added stronger declarative `tests[]` support in generated policy tests, including rule ID, timeout, retry, breaker, trace, and explain expectations.
- Added golden explain tests and fuzz smoke targets for policy checking, explain, predicates, and diagnostic paths.
- Added indexed capability routing for generated/builder policies with large route sets.
- Added a predicate-composition example for compound machine-authored rules.
- Added agent workflow documentation for declarative policy changes.
- Added examples documentation for choosing and validating example policies.
- Added release documentation with the `v0.1.0` checklist and tag workflow.
- Initial independent CervoRules Go library for deterministic policy decisions.
- Runtime decision flow engine with compile-time validation and reusable compiled flows.
- Policy builder helpers for routing, trusted routes, denies, audit, and limits.
- Vocabulary validation helpers for caller-owned capabilities, services, and providers.
- Vocabulary and policy code generators for agent-maintained YAML/JSON policy files.
- Example declarative policies for routing, trusted users, fallbacks, and sensitive denies.
