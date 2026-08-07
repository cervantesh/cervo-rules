# Changelog

All notable changes to CervoRules will be documented in this file.

## v3.0.0-rc.7 - 2026-08-07

### Removed

- Retired three published schemas that no tool produced and no document
  satisfied: `schemas/v3/policy-inspection.schema.json`,
  `schemas/v3/compatibility-report.schema.json`, and
  `schemas/v3/agent-manifest.schema.json`. The first two described the output of
  CLI commands that were never built. The third was a rival shape for the agent
  manifest; every reference in the repository resolves to
  `.cervorules/agent-manifest.json`, which validates against the schema beside
  it and never matched the v3 one. Consumers extracting
  `cervorules-schemas-<version>.tar.gz` will no longer find these three. They
  remain in git history if the commands are ever built.

### Fixed

- `ontology`: an individual recorded in two lifecycle states let a second
  terminal transition through, and which one won depended on input order.
  Recording `paid` first permitted a refund that recording `refunded` first
  refused. `CheckTransition` now rejects the incoherent world before reading
  state, so `transition_allowed` alone is safe.
- `ontology`: two relations declaring the same `PredicateSignature` were
  accepted, after which `Check` took the last and `domainOf` took the first.
  `Validate` now rejects the duplicate.
- `ontology`: `Normalize` sorted states by a partial key, so equivalent
  ontologies could normalize to different orders.
- `httpadapter`: a classifier rule whose regex contained an uppercase letter
  could never fire, because the input was lowercased and the pattern was not.
- `decisioncache` was published and empty while the API inventory recorded it as
  owning contracts.
- Added `.gitattributes` pinning this repository to LF. The `PolicyHash` is
  sha256 over a policy file's raw bytes, so a checkout that converted line
  endings changed a policy's identity without changing the policy, and
  generation was not reproducible between a Windows machine and CI. This fixes
  the repository; the same hazard for consumers is recorded as an open gap in
  `docs/v3/known-gaps.md`, because normalizing inside the hash would invalidate
  every hash ever recorded.

### Changed

- The public API inventory now covers all ten packages, not four.

### Added

Claims this project makes in prose are now checked by the build. Each check was
verified against a mutant before being committed, because a check that cannot
fail is the defect these are meant to catch, and this repository has found
three of those.

- Fail-closed is fuzzed. `internal/fuzzpolicy` commits a generated engine as a
  subject; the assertion is that if a decision came back, every fact the policy
  read was parseable and in range. 73.7M executions with no failure.
- Generation is pinned byte-stable across runs, because Go randomizes map order
  per process and the `PolicyHash` argument rests on generation being a function
  of its inputs.
- The standard-library-only boundary is checked against the build graph with
  `go list -deps`, so an indirect dependency cannot hide from a source scan.
- The agnosticism rule is enforced: no name declared in any example vocabulary
  may appear as a string literal in hand-written library source. Generated code
  is exempt, because carrying the vocabulary is its job.
- No schema may be published without a producer. `validate-schemas.py` requires
  every schema to be backed by documents in the repository or by a named Go
  test holding its producer to it, and checks that the named test exists.
- `.cervorules` JSON is validated in CI alongside the policy YAML. The agent
  manifest had never validated against either of its rival schemas.
- `observe.PolicyEvaluationReport` is held to
  `schemas/v3/policy-evaluation-report.schema.json`; the schema sets
  `additionalProperties: false`, so a new struct field would have broken every
  consumer validating against it.
- Task recipes are checked for what they claim, not only for their shape: every
  `-run` pattern must select a real test, every script and CLI subcommand must
  exist, every flag must be declared, and a pinned version must match
  `current_version` in the manifest.
- Error-code wire strings are pinned, and every code must be documented.
- `CodegenSnapshot` gained `policy_facts` and `predicates`, so a changed fact
  bound or an added `when:` moves a snapshot field.

## v3.0.0-rc.6 - 2026-08-06

### Security

- Raised the minimum Go toolchain to 1.25.11. `govulncheck` reports GO-2026-5037
  in `crypto/x509` for go1.25.8, reachable from `limits.Check` through
  `fmt.Sprintf` to `x509.HostnameError.Error`. The trace is generic -- any
  `fmt.Sprintf` over an error can reach it -- but the fix is a toolchain bump,
  so there is nothing to weigh. The `go` directive is the consumer-visible half:
  a consumer still building with 1.25.8 links the affected standard library.

### Fixed

- The temp consumer module used by generated-policy tests now reads the `go`
  directive from this module instead of hardcoding it, so a toolchain bump no
  longer leaves the tests failing with "updates to go.mod needed".

## v3.0.0-rc.5 - 2026-08-06

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

### Changed (release tooling)

- Publishing now targets GitHub. Release artifacts are attached to the GitHub
  Release for the tag and the tools image goes to `ghcr.io/<owner>`. The
  workflow previously uploaded to a generic package API that github.com does
  not have, so the step returned 404 and could never succeed -- which is why
  v3.0.0-rc.4 shipped a release with zero assets.
- No secret has to be created for a release. `GITHUB_TOKEN` carries
  `contents: write` and `packages: write`, and Actions injects it. The
  `PACKAGE_TOKEN` secret and the `CERVORULES_OCI_REGISTRY` variable are gone.
- Re-running the workflow for a version that already has a release replaces its
  assets instead of failing, so a partial upload is repaired by re-running it.
- Replaced `scripts/release/verify-generic-package.sh` with
  `scripts/release/verify-github-release.sh`, which downloads the published
  release with `gh` and runs the same checksum, module-marker, schema and CLI
  version checks.

### Security

- Fixed a code-injection path in the generator. A predicate description embeds
  the policy's own string literals and was written into a Go line comment
  unescaped, so a string-fact literal containing a newline terminated the
  comment and everything after it was parsed as top-level Go. A policy could
  inject arbitrary declarations — including a `func init()` that runs on package
  import — into the generated engine; `check` reported the policy ok, `generate`
  exited 0, and gofmt accepted the result because the injected text was valid
  Go. The YAML is the artifact a reviewer reads, so this inverted the review
  boundary the DSL exists to create. Comments are now sanitised, and the
  benign form of the same bug — a literal newline making generation fail with
  an opaque Go parser error — is gone with it.

### Fixed

- Fixed `check` not being a function of its input. A policy naming one fact
  under two keys that normalize alike (`risk_pct:` and `RISK_PCT:`) resolved to
  a single fact, but which key supplied the bounds and which supplied the
  default came down to Go map iteration order: the same file passed `check` on
  some runs and failed on others, and when it passed it emitted a default the
  other entry's bounds forbid. Colliding keys are now rejected, and the fact
  loops iterate in sorted order so a failure and the names it quotes are
  identical on every run.
- Fixed a fact `default` outside its own declared `min`/`max` being accepted.
  The generated parser returns the default before it reaches the bounds checks,
  so absence yielded exactly the value the same policy refuses when present.
- Fixed integer fact bounds being carried as `float64` and converted with an
  unchecked `int64(...)`. A bound at or above 2^63 became `MinInt64`, silently
  dropping the declared minimum; bounds are now required to be exactly
  representable.
- Fixed two routes naming the same executor with `fallback_executors` emitting a
  Go map literal with a duplicate constant key, which only failed in the
  consumer's build. Routes that disagree on an executor's fallbacks are now
  rejected at generation time rather than one intent being silently dropped.
- Fixed `generate -test-out` being nondeterministic: generated test metadata was
  emitted in Go map-iteration order, so a checked-in generated test churned on
  every regeneration.
- Fixed a string-fact literal with surrounding whitespace, or an empty one,
  compiling to a rule that could never match. The runtime trims every metadata
  value and treats a blank one as absent, so such a literal is dead by
  construction and is now rejected.
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
