# ADR 0006: Stratified Negation

Date: 2026-05-22

## Status

Accepted / Implemented for bounded facts runtime

## Issue

Tracked by issue #161 and implementation issue #199.

## Context

CervoRules policies sometimes need to express absence: a provider is allowed
when it has no active deny marker, a fallback applies when no stronger match
exists, or a capability is eligible when no explicit block is present. Today
that shape must be modeled with positive facts, precomputed adapter state, or
consumer-side filtering.

Datalog-style negation can express these cases, but unrestricted negation makes
evaluation order-sensitive and can introduce ambiguous or non-terminating
programs. CervoRules must remain deterministic, bounded, auditable, and
cross-domain. It must not become a Prolog interpreter, a backtracking search
engine, or a general logic programming runtime.

## Decision

CervoRules supports stratified negation in the optional `facts` runtime.
Negation is allowed only when validation can prove that negated dependencies
point strictly to lower strata or to base input facts with no derived producer.

Required semantics:

1. Facts and derived predicates remain finite sets over bounded domains.
2. Positive dependencies may stay within the same stratum.
3. Negated dependencies must reference predicates from an earlier stratum.
4. A cycle containing negation is rejected at compile time.
5. Evaluation proceeds stratum by stratum to a fixed point before the next
   stratum can read the completed lower-stratum facts.
6. Negation means "not derivable in the completed lower stratum", not
   open-ended search and not database null semantics.
7. Trace output must show the positive derivation and the completed predicate
   consulted by any negated condition.

Implementation notes:

- Go API: `facts.Not(facts.NewPattern(...))`.
- Declarative DSL: `PatternSpec{Negated: true}` or YAML `not: true`.
- Negated patterns are allowed in rule bodies only.
- Variables used by a negated pattern must be bound by an earlier positive body
  pattern.
- Evaluation runs each stratum to a fixed point before moving to the next
  stratum.

This feature is a rule authoring convenience over bounded set evaluation. It
does not change the core decision model, request model, action model, runtime
limit model, or package ownership boundaries.

## Rejected Alternatives

- Unrestricted negation: rejected because negative cycles make results
  order-dependent or undefined.
- Prolog-style negation as failure with backtracking: rejected because it would
  change CervoRules from bounded policy evaluation into goal search.
- Runtime best-effort cycle handling: rejected because invalid rule graphs must
  fail before deployment.
- Consumer-owned absence flags only: rejected as the sole path because it pushes
  repetitive, error-prone derivation logic into adapters.
- Three-valued logic: rejected because unknown values complicate deterministic
  audit output and cross-domain policy expectations.

## Operational Bounds

- The compiler must reject every rule graph where a negated edge participates in
  a dependency cycle.
- All predicates participating in stratified negation must have finite,
  enumerable input domains known to the runtime before evaluation starts.
- No rule may allocate new unbounded symbols while evaluating negated clauses.
- The implementation must respect existing evaluation limits and add explicit
  limits for derived fact count, stratum count, and per-stratum iteration count.
- Evaluation order inside one stratum must not affect the final result.
- Diagnostics must report the dependency path that made a program
  unstratifiable.
- Negation must not perform I/O, call dynamic predicates, inspect wall-clock
  time, or depend on map iteration order.

## TDD Evidence

The implementation includes red-green coverage for these behaviors:

- A positive acyclic policy using negation against a lower stratum derives the
  same decision across repeated runs.
- A policy with `allowed` depending on `not blocked` derives only allowed facts
  whose lower-stratum blocked fact is absent.
- A direct negative cycle is rejected at compile time with a structured
  diagnostic naming the cycle.
- An indirect negative cycle through two or more predicates is rejected at
  compile time with a structured diagnostic naming the path.
- A positive recursive cycle in one stratum remains valid when no negated edge
  participates in that cycle.
- Existing fact-count and per-stratum iteration limits fail closed with
  deterministic errors.
- Existing canonical fact ordering keeps repeated runs stable.

The runtime tests verify public behavior and diagnostic shape without asserting
on incidental formatting. Generated-policy DSL support is covered by policygen
tests for `not: true` emission.

## Dependencies

- ADR 0001 cross-domain public API guardrails.
- ADR 0002 v2 public API change controls if public DSL or diagnostics change.
- ADR 0004 modular package ownership if the feature requires compiler,
  runtime, or testkit package boundaries.
- A finite fact representation for rule inputs and derived predicates.
- Structured diagnostics capable of reporting dependency graphs and limit
  violations.
