# ADR 0005: Add Datalog-Inspired Facts Without Becoming Prolog

## Status

Proposed

## Date

2026-05-22

## Context

CervoRules needs a way to represent derived cross-domain relationships without
coupling core APIs to any gateway, HTTP transport, provider registry, AI
payload, or other consumer vocabulary.

Prolog is powerful, but a full Prolog runtime would make production evaluation
harder to bound, test, package, and explain. The useful subset for CervoRules is
closer to Datalog: flat facts, rules that derive more facts, deterministic
evaluation, explicit bounds, and explainable provenance.

## Decision

Add an optional `facts` module inspired by Datalog:

- flat facts;
- constants and variables in patterns;
- deterministic rule evaluation;
- bounded fixed-point derivation;
- derivation traces;
- query helpers.

Do not implement full Prolog semantics.

The initial boundary is an optional public package:

```text
github.com/cervantesh/cervo-rules/v2/facts
```

Core policy evaluation remains deterministic and dependency-light. Adapters and
consumers may evaluate facts before calling core and then bridge selected facts
into existing request or metadata surfaces. Generated policy and core predicate
bridges require follow-up design before implementation.

## Non-Goals

- No Prolog parser.
- No `cut`.
- No side-effecting predicates.
- No dynamic `assert` or `retract`.
- No unbounded recursion.
- No automatic procedural backtracking.
- No full Prolog unification of nested terms.
- No consumer-specific vocabulary in core or the facts module.
- No required dependency from core policy evaluation to the facts module.

## TDD And Verification

Implementation work must start with focused failing tests and move in narrow
steps:

- constructors, stable string output, and pattern variables;
- immutable set behavior and variable binding;
- bounded evaluator behavior and diagnostics;
- derivation trace provenance;
- optional adapter and generated-policy integration only after the base package
  is stable.

Verification for implementation changes must include focused package tests, the
full test suite, coverage, vet, module verification, dependency scope checks,
and changed-doc unresolved-marker scans.

## Consequences

Positive:

- more expressive neutral policies;
- better agnosticism because domain relationships remain caller-owned data;
- reusable explain trees for derived conclusions;
- cleaner adapters for HTTP and non-HTTP inputs;
- optional adoption for consumers that need derived facts.

Tradeoffs:

- new public API surface;
- generated policy integration requires careful sequencing;
- users may expect Prolog features that are intentionally unsupported;
- traces and diagnostics need redaction and deterministic ordering before broad
  production use.

## Guardrail

Any future addition that introduces recursion, negation, aggregation,
arithmetic, parser syntax, priority conflict resolution, temporal logic,
incremental evaluation, or policygen-owned derived facts requires a new ADR or
explicit checkpoint.

## Issue Links

- #151: umbrella facts epic.
- #152: immutable fact set and pattern query API.
- #153: bounded derived fact evaluator.
- #154: derivation trace and explain output.
- #155: validation guardrails.
- #156: document logic-inspired facts boundary.
- #157: neutral derived facts example.
