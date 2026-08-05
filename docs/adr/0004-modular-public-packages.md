# ADR 0004: Modular Public Packages

Status: Accepted

Date: 2026-05-21

## Context

CervoRules v2 grew into a broad public surface: core decision evaluation,
runtime config validation, limit checks, HTTP classification, observability, and
test helpers. Not every consumer needs every part. Importing the root package for
small use cases makes dependency and ownership boundaries harder to see.

## Decision

Expose stable public subpackages and make the root package a compatibility
facade.

Public package ownership:

- `core`: decision engine primitives and policy evaluation.
- `runtime`: startup/runtime policy config construction, merge, and validation.
- `limits`: generic limit request checks.
- `httpadapter`: optional HTTP and facts adapters.
- `observe`: stack-neutral observations, reports, and audit envelopes.
- `testkit`: consumer-side contract checks.

Generated policies must import `core`, `runtime`, and `limits` directly instead
of importing the root facade.

## Consequences

Positive:

- Selective imports are possible for consumers that only need limits, runtime
  config, HTTP classification, or test helpers.
- Generated policies no longer depend on the broad root facade.
- Package ownership is clearer during code review and change management.

Tradeoffs:

- Documentation must teach both the root facade and direct subpackage imports.
- Generated policy examples and migration snippets need to use modular aliases.
- Public API changes must choose an owning subpackage before adding root facade
  aliases.

## Root Facade

The Root facade remains for migration convenience and compatibility. It must
only expose aliases or delegating functions to owning packages. It should not
contain copied runtime logic.

## Selective Imports

Consumers should import the narrowest package that matches their use case:

- `limits` for limit checks without policy evaluation;
- `runtime` for config validation without request handling;
- `httpadapter` for HTTP classification without generated policy code;
- `core` for full decision-flow evaluation;
- `testkit` only in tests.

## Follow-Up

- Keep dependency scope tests active.
- Keep generated policy import tests active.
- Refresh README, migration, dependency, and wiki report links whenever package
  boundaries change.
