# ADR 0008: Controlled Aggregates

Date: 2026-05-22

## Status

Accepted / Implemented for bounded facts runtime

## Issue

Tracked by issue #163 and implementation issue #200.

## Context

Policies often need bounded summaries: count eligible providers, choose the
minimum configured priority, require at least two independent approvals, or
derive the maximum score among already-eligible facts. Today these summaries
must be precomputed by consumers or modeled with bespoke predicates.

Datalog-inspired aggregates can express these summaries, but uncontrolled
aggregation can make policies non-monotonic, order-sensitive, expensive, or
ambiguous when combined with recursion and negation. CervoRules must preserve
deterministic, bounded evaluation and remain a policy runtime, not an analytical
query engine or Prolog system.

## Decision

CervoRules supports controlled aggregates in the optional `facts` runtime.
Aggregates are allowed only over finite, fully materialized predicate inputs and
only through an approved set of deterministic aggregate functions.

Approved aggregate families:

- `count` over finite fact groups.
- `min` and `max` over ordered scalar values with stable comparison semantics.
- `sum` over bounded integer or fixed-precision numeric values with overflow
  checks.
- `exists` as a named aggregate form when it improves readability over count.

Required semantics:

1. Aggregate inputs must be complete before the aggregate is evaluated.
2. Aggregate grouping keys must be finite and derived from bounded domains.
3. Aggregate output must use stable keys and stable value ordering.
4. Empty input behavior must be explicit per aggregate.
5. Numeric overflow, unsupported value types, and exceeded group limits fail
   closed with structured diagnostics.
6. Aggregates used with recursion must follow monotonicity restrictions or be
   rejected by the compiler.
7. Aggregates used with negation must respect stratum boundaries: a higher
   stratum may aggregate completed lower-stratum predicates.

Implementation notes:

- Go API: `CountAggregate`, `MinAggregate`, `MaxAggregate`, `SumAggregate`, and
  `ExistsAggregate`.
- Declarative DSL: `AggregateSpec` under `RuleSetSpec.Aggregates`.
- Aggregate inputs must be base facts or facts produced in lower strata.
- Aggregate output facts are emitted in stable group-key order.
- Sum uses `int64` and fails closed on overflow.

Aggregates are policy summaries for bounded fact sets. They are not window
functions, streaming analytics, arbitrary reducers, or user-defined code hooks.

## Rejected Alternatives

- Arbitrary user-defined aggregate functions: rejected because they can hide
  nondeterminism, I/O, and unbounded cost inside policy evaluation.
- Floating-point-first numeric aggregation: rejected because rounding behavior
  can vary and complicate auditability.
- SQL-style broad aggregation semantics: rejected because CervoRules does not
  need a general analytical query language.
- Aggregates evaluated during recursive fixed-point loops without restrictions:
  rejected because non-monotonic aggregates can make results order-dependent.
- Consumer-side summaries only: rejected as the sole path because it fragments
  traceability and duplicates common bounded summaries across domains.

## Operational Bounds

- The runtime must enforce maximum aggregate groups, maximum input facts per
  group, maximum aggregate output facts, and numeric range limits.
- Aggregate functions must be pure, deterministic, and independent of map
  iteration order.
- Aggregate evaluation must not perform I/O, call dynamic predicates, inspect
  wall-clock time, or invoke consumer callbacks.
- Unsupported aggregate input types must fail during compile or validation when
  statically knowable, otherwise at evaluation with a structured diagnostic.
- Aggregates over recursive predicates may run only after the recursive
  component reaches a fixed point, unless a future ADR defines a narrower
  monotonic-recursive aggregate model.
- Aggregate traces must identify the source predicate, grouping key, aggregate
  function, input count, and resulting value within configured trace limits.

## TDD Evidence

The implementation includes red-green coverage for these behaviors:

- `count` derives deterministic group counts from finite input facts.
- `min` and `max` choose stable values independent of input fact ordering.
- `sum` detects overflow and fails closed with a structured diagnostic.
- Aggregate group and input-size limits fail closed with structured diagnostics.
- Aggregates over completed lower-stratum predicates work with stratified
  negation semantics.
- Aggregates inside same-stratum producer components are rejected during
  validation.
- Derived aggregate facts are recorded in derivation trace steps using the
  aggregate name and stratum.
- Generated-policy examples compile and evaluate aggregate policies through the
  same public API path as non-aggregate policies.

Stable set ordering and sorted aggregate group keys keep repeated evaluations
deterministic.

## Dependencies

- ADR 0001 cross-domain public API guardrails.
- ADR 0002 v2 public API change controls if public DSL, evaluator, trace, or
  diagnostic contracts change.
- ADR 0006 stratified negation for aggregate reads across strata.
- ADR 0007 bounded recursion for aggregate restrictions around recursive
  predicates.
- A typed finite fact representation with stable scalar comparison semantics.
- Limit configuration and structured diagnostics that can represent aggregate
  group, input-size, type, and overflow failures.
