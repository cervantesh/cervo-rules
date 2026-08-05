# ADR 0009: Closed-World Fact Semantics

Status: Proposed

## Issue

Tracked by issue #168.

## Context

The optional facts package uses finite input facts, bounded rules, and
deterministic fixed-point evaluation. Consumers need a clear meaning for facts
that are not present or not derivable, especially before future stratified
negation, materialized views, partial evaluation, or policy diffs are added.

This ADR builds on:

- ADR 0005: logic-inspired facts;
- ADR 0006: stratified negation;
- ADR 0007: bounded recursion;
- ADR 0008: controlled aggregates.

## Decision

CervoRules facts use closed-world semantics inside a single evaluation snapshot:
if a fact is absent from the input set and cannot be derived by the configured
rules within the configured bounds, it is not derivable for that evaluation.

`not derivable` is not the same as database `NULL`, unknown remote state, or a
transport parsing failure. Adapters must decide what facts to provide. The facts
engine only reasons over the finite set it receives.

Future negation must be stratified and must only observe predicates whose lower
strata have completed. Absence checks must never trigger network calls, dynamic
lookups, or side effects.

## Consequences

- Evaluation remains deterministic and bounded.
- Missing adapter facts can change results, so adapters must be tested as part
  of consumer contracts.
- Future diffs can distinguish `removed input fact`, `no longer derived`, and
  `outside snapshot` instead of inventing an `unknown` truth value.

## Non-Goals

- No tri-valued logic.
- No open-world inference.
- No automatic database or service lookup for missing facts.
- No public negation API in this ADR.

## Rejected Alternatives

- Open-world semantics were rejected because they would make request-time
  decisions depend on unavailable external knowledge.
- A third `unknown` truth value was rejected for this layer because it would
  complicate generated policy contracts before a concrete consumer proves the
  need.

## Verification Expectations

Future implementation work should add tests proving:

- absence is evaluated within one immutable input snapshot;
- stratified negation cannot inspect same-stratum or higher-stratum predicates;
- diagnostics distinguish invalid inputs from simply non-derivable facts;
- materialized views document their snapshot boundary.
