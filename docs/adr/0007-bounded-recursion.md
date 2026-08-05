# ADR 0007: Bounded Recursion

Date: 2026-05-22

## Status

Proposed

## Issue

Tracked by issue #162.

## Context

Some policy relationships are naturally transitive. Examples include inherited
capabilities, fallback chains, delegated provider groups, organizational
containment, and dependency reachability. Without recursion, consumers must
precompute transitive closure outside CervoRules or duplicate each depth level in
policy source.

Datalog-style positive recursion can express transitive closure with fixed-point
evaluation over finite relations. That model can fit CervoRules only if it is
bounded, deterministic, and observable. CervoRules must not add unbounded term
construction, recursive function calls, backtracking, or Prolog-style goal
search.

## Decision

CervoRules may add bounded positive recursion as a future advanced rule feature.
Recursive rules are allowed only for finite predicates whose domains and limits
are known before evaluation.

Required semantics:

1. Recursive evaluation computes a least fixed point over finite sets.
2. Only positive recursion is allowed in the recursive component.
3. Recursive predicates may not depend on negation in the same dependency
   component.
4. The evaluator must deduplicate derived facts by stable keys.
5. Evaluation must terminate when an iteration produces no new facts or when a
   configured bound is reached.
6. Hitting a configured bound is a deterministic policy evaluation failure, not
   a partial success.
7. Trace output must expose the rule, iteration, and source facts responsible
   for derived recursive facts within configured trace limits.

Recursion is intended for finite closure over existing facts. It is not a
general-purpose loop construct and does not add recursive user functions.

## Rejected Alternatives

- Unbounded recursion: rejected because policy evaluation must have predictable
  cost and termination.
- Prolog-style recursive goals with backtracking: rejected because CervoRules is
  not a query search engine.
- Fixed depth expansion only: rejected because it forces rule authors to encode
  arbitrary depth limits into policy text rather than runtime bounds.
- Consumer-side closure only: rejected as the only option because it fragments
  audit traces and duplicates generic closure logic across domains.
- Partial results when bounds are hit: rejected because fail-open or silently
  incomplete decisions are unsafe for policy evaluation.

## Operational Bounds

- Every recursive predicate must declare or inherit finite key domains.
- The runtime must enforce maximum derived facts, maximum recursive iterations,
  maximum recursive component count, and maximum trace entries.
- Recursive evaluation must use stable ordering for diagnostics and trace output
  without making result correctness depend on that order.
- Recursive rules must not create new symbols outside the declared bounded
  domains.
- Recursive evaluation must not perform I/O, call dynamic predicates, read
  wall-clock time, or use random values.
- A recursive component that reaches its bound must return a structured error
  identifying the rule component and bound.
- Stratified negation, if present, may read completed recursive predicates from
  lower strata, but negation must not be part of a recursive cycle.

## TDD Expectations For Future Implementation

Future implementation should be test-first and must include red-green coverage
for these behaviors before production code lands:

- A transitive closure policy derives all reachable facts for a finite chain.
- A branching graph produces deterministic derived fact sets and byte-stable
  decisions across repeated runs.
- A cyclic input graph terminates by deduplication and derives the expected
  finite closure.
- A recursive rule that exceeds maximum iterations fails closed with a
  structured limit diagnostic.
- A recursive rule that exceeds maximum derived facts fails closed with a
  structured limit diagnostic.
- A recursive rule with negation in the same dependency component is rejected at
  compile time.
- Trace output identifies representative recursive derivations while honoring
  configured trace limits.
- Generated-policy examples compile and evaluate recursive policies through the
  same public API path as non-recursive policies.

Tests should include randomized input fact order and repeated evaluation runs to
prove deterministic outcomes.

## Dependencies

- ADR 0001 cross-domain public API guardrails.
- ADR 0002 v2 public API change controls if public evaluator, DSL, trace, or
  diagnostic contracts change.
- ADR 0006 stratified negation if recursive predicates are later combined with
  negated reads from lower strata.
- A finite fact representation with stable identity keys.
- Limit configuration and structured diagnostics that can represent recursive
  iteration and derived fact bound failures.
