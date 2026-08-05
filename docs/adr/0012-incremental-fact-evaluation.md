# ADR 0012: Incremental Fact Evaluation

Status: Accepted / Implemented for explicit snapshots

## Issue

Tracked by issue #165 and implementation issue #202.

## Context

Incremental evaluation can avoid recomputing all derived facts when a small
subset of input facts changes. This is valuable for large materialized views or
long-lived processes, but it increases complexity and cache invalidation risk.

This ADR depends on ADR 0009, ADR 0010, `docs/materialized-fact-views.md`, and
ADR 0011.

## Decision

Incremental evaluation is allowed only as an optimization over the same
semantics as full bounded evaluation. For the same input snapshot and rule set,
incremental and full evaluation must produce the same facts, diagnostics, and
observable trace summaries.

The first implementation should prefer explicit invalidation boundaries and
small dependency indexes over implicit mutable global state.

Implemented API:

```go
snapshot, err := engine.Snapshot(ctx, inputFacts, options)
result, err := engine.EvaluateDelta(ctx, snapshot, facts.ChangeSet{
    Add:    []facts.Fact{newFact},
    Remove: []facts.Fact{oldFact},
}, options)
```

The first implementation is intentionally conservative: it applies the explicit
change set to the previous input snapshot and runs normal bounded evaluation.
This establishes public snapshot/change-set semantics and equivalence tests
without introducing mutable global state or hidden background refresh.

## Consequences

- Performance can improve for large stable fact sets.
- Tests must compare incremental results against full evaluation.
- Trace and diagnostics must remain understandable when facts are reused.

## Non-Goals

- No mutable global fact database.
- No streaming event processor in the facts package.
- No hidden background refresh.

## Rejected Alternatives

- Mutable global state was rejected because it would make policy results
  history-dependent and harder to test.
- Event-stream ownership in the facts package was rejected because transports
  and delivery semantics belong in adapters.

## Verification Expectations

Implementation tests include:

- added-fact and removed-fact comparisons between incremental and full
  evaluation;
- diagnostic preservation through `EvaluateDelta`.

Dependency indexes, trace reuse, and redaction-aware reuse remain future
optimizations over the same snapshot/change-set contract.
