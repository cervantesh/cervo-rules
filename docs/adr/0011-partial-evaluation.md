# ADR 0011: Partial Evaluation

Status: Accepted / Implemented for facts runtime preparation

## Issue

Tracked by issue #181 and implementation issue #201.

## Context

Some facts are stable across many requests while others are request-specific.
Future optimizations may precompute parts of evaluation over stable facts.

This ADR depends on ADR 0009 closed-world semantics, ADR 0010 namespaced fact
vocabulary, and `docs/materialized-fact-views.md`.

## Decision

Partial evaluation precomputes derived facts over static inputs and rules,
then leaves request-specific facts for runtime. It must be observationally
equivalent to evaluating all facts and rules together in a single bounded run.

Partial evaluation must not produce partial decisions. It only produces facts,
residual rule state, diagnostics, or review artifacts that are later consumed by
normal policy evaluation.

Implemented API:

```go
prepared, err := engine.Prepare(ctx, staticFacts, options)
result, err := prepared.Evaluate(ctx, requestFacts, options)
```

`Prepare` stores the static evaluation result and an immutable copy of the
engine. `PreparedEngine.Evaluate` combines static derived facts with
request-specific facts and runs normal evaluation, preserving the same
diagnostic and bound behavior as `Engine.Evaluate`.

## Consequences

- Startup or release workflows can move work out of request paths.
- Consumers must track which facts were static and which were request-specific.
- Trace output needs to show precomputed sources distinctly from request-time
  sources if this becomes runtime API.

## Non-Goals

- No compiled binary rule format.
- No bypass of normal validation, bounds, or redaction.

## Rejected Alternatives

- Producing partial policy decisions was rejected because decisions must remain
  request-complete and auditable.
- Skipping full-equivalence tests was rejected because partial evaluation is an
  optimization, not a new semantic mode.

## Verification Expectations

Implementation tests prove:

- full evaluation and partial-plus-residual evaluation produce identical final
  fact sets for the same snapshot;
- diagnostics preserve source phase;
- normal validation and bounds still apply.

Redaction of prepared traces and compatibility tests for changed static fact
vocabularies remain release-hardening concerns rather than separate evaluation
semantics.
