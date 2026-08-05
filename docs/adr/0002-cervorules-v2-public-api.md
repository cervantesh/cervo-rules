# ADR 0002: CervoRules v2 Public API Redesign

Date: 2026-05-21

## Status

Proposed

## Issue

Tracked by issue #60:
https://github.com/cervantesh/cervo-rules/issues/60

## Context

CervoRules has matured from a small deterministic rules package into a shared
policy runtime with generated policies, runtime configuration validation, HTTP
classification adapters, package publishing, observability contracts, and
consumer contract helpers.

The current API has several successful v1-era choices that now limit long-term
evolution:

- `Engine.Decide` returns only `Decision`, while trace, observation, diagnostics,
  and future evaluation metadata are operationally separate concerns.
- `Predicate.Evaluate` cannot return an error, which limits dynamic predicates
  and makes failure reporting less explicit.
- `Action.Apply` receives only a mutator, so custom actions cannot see
  `context.Context` or immutable evaluation facts.
- `CheckLimits` returns only the first violation as `error`, even when multiple
  requested limits are invalid.
- `testkit.AssertGeneratedRuntimePolicy` is assertion-first rather than
  error-first, limiting reuse outside `testing.TB`.
- Generated policies expose free functions rather than a cohesive factory.
- `PolicyRuntimeConfig` is a public struct whose fields become permanent
  compile-time contract.
- `Vocabulary` is a concrete struct, which makes dynamic or generated
  vocabulary implementations harder to introduce cleanly.
- Validation errors are mostly textual rather than structured.

These changes are compile-time breaking for consumers and generated policies.
They should be treated as a v2 line rather than a patch or minor release.

## Decision

CervoRules v2 will be designed as an intentional public API break with
change-management controls before implementation.

The v2 line should prioritize:

1. Explicit result envelopes over overloaded structs.
2. Error-returning contracts for predicates, generated policy checks, validation,
   and limits.
3. Context-aware actions and evaluation.
4. Structured validation and diagnostics.
5. Generated policy factories instead of loose package functions.
6. Compatibility shims only where they reduce migration risk without hiding v2
   semantics.
7. Cross-domain naming review for every new public surface.

The following changes are approved for detailed implementation planning:

- Introduce `DecisionResult` and change `Engine.Decide` to return it.
- Change `Predicate.Evaluate` to return `(PredicateResult, error)`.
- Change `Action.Apply` to receive `context.Context` and `EvalContext`.
- Change `CheckLimits` to return `LimitViolations`.
- Add error-first testkit APIs and keep fatal helpers as explicit `Must...`
  helpers.
- Generate a `PolicyFactory` API for generated policies.
- Evaluate moving `PolicyRuntimeConfig` toward options or immutable builder
  construction.
- Evaluate changing `Vocabulary` from concrete struct to interface plus default
  implementation.
- Add structured validation errors and diagnostics.

The following changes are not approved without a separate ADR:

- Renaming `Provider`, `Service`, or `Capability`.
- Moving HTTP classification into the identity of the core runtime.
- Adding CervoProxy-specific fields to core runtime structs.
- Introducing external runtime dependencies into the core package.

## Consequences

- CervoRules v2 will require consumer code changes.
- Generated policies must be regenerated.
- CervoProxy migration must be sequenced after generated policy v2 stabilizes.
- Documentation must include a v1-to-v2 migration guide.
- Package releases must clearly separate v1 and v2 artifacts.
- The v2 branch must maintain high coverage and consumer-shaped examples before
  merge to main.

## Compatibility Policy

The v2 implementation may include compatibility helpers, but compatibility
helpers must be explicit and temporary. Examples:

- `MustAssertGeneratedRuntimePolicy` may wrap `CheckGeneratedRuntimePolicy`.
- A v1-style generated `BuildPolicy` wrapper may exist for one migration cycle
  only if it delegates to `PolicyFactory.Build`.
- Limit helper wrappers may exist if they preserve v2 multi-violation data.

Compatibility helpers must be documented with deprecation text and a removal
target.

## Verification Requirements

Before any v2 public API PR is merged:

- `go test -count=1 ./...`
- `go test -cover ./...` with package-level coverage at or above 90 percent
- `go vet ./...`
- `go mod verify`
- generated examples compile in consumer-shaped temp modules
- CervoProxy migration branch compiles against the candidate v2 API
- docs and migration guide are updated in the same PR or a prerequisite PR

## Related Documents

- `docs/adr/0001-cross-domain-public-api.md`
- `docs/change-management/v2-public-api.md`
- `docs/release.md`
