# Changelog

All notable changes to CervoRules will be documented in this file.

## Unreleased

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
