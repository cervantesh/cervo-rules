# Materialized Fact Views

Status: Operational pattern. There is no built-in cache or view manager, but
the facts runtime now has `Prepare`, `Snapshot`, and `EvaluateDelta` APIs that
consumers can use when implementing their own materialized view lifecycle.

Tracked by issue #167.

## Purpose

A materialized fact view is a caller-owned snapshot of input or derived facts
that can be reused across evaluations. It remains caller-owned; CervoRules does
not own cache storage, refresh schedules, or invalidation policy.

Materialized views are useful when facts are expensive to assemble but stable
for a known period, for example:

- tenant plan facts loaded at startup;
- device capability catalogs refreshed periodically;
- document classification facts produced by an offline pipeline;
- ownership maps generated from a deployment inventory.

## Semantics

Views inherit ADR 0009 closed-world semantics. A view represents what is known
inside its snapshot boundary. If a fact is missing from the view, the engine
treats it as not present unless another input or rule derives it.

Views should use ADR 0010 namespaced predicates so consumers can understand
ownership, arity, and compatibility.

## Operational Pattern

1. Build or load a bounded `facts.Set`.
2. Validate it with `PredicateSchema` and constraints.
3. Record source, build time, and freshness metadata with `FactRecord` when
   those diagnostics matter to startup or request validation.
4. Combine the view with request-specific facts.
5. Evaluate rules with explicit bounds.

`Engine.Prepare` materializes static input and static derived facts once.
`PreparedEngine.Evaluate` returns that materialized result directly when there
are no request facts. When request facts are provided, it recombines the
original static facts with request facts and evaluates normally so aggregates,
negation, and request-dependent derivations remain equivalent to a full
evaluation. This is the preferred shape for startup/interval validation of
catalogs, permissions, hierarchies, routes, and other relationships that are
stable at startup or over a refresh interval.

## Caller-Owned Cache Keys

No global facts cache exists in CervoRules. Consumers may cache a prepared
engine, snapshot, or result only when they own a stable key and invalidation
policy.

Useful key parts include:

- stable input version, such as catalog revision, tenant ACL revision, device
  inventory version, or route table generation;
- rule/policy version;
- request fact hash when repeated request-shaped facts are expected;
- evaluation option profile, especially budgets and trace mode.

Do not reuse cached facts when decisions depend on volatile inputs such as live
health, wall-clock windows, request identity that is not in the key, external
lookups, or mutable trust state. If any of those inputs affect derivation, they
must either be encoded into the key or bypass the cache.

## Non-Goals

- No cache implementation.
- No global facts cache.
- No invalidation engine.
- No persistence format beyond the stable JSON already provided by facts.
- No coupling to databases, queues, or HTTP.

## Rejected Alternatives

- A built-in cache was rejected because consumers own freshness, storage, and
  invalidation boundaries.
- A database-specific view adapter was rejected because CervoRules must remain
  storage-agnostic.

## Verification Expectations

Consumer tests should prove:

- stale views are rejected or refreshed by the adapter;
- `ValidateFreshness` is called with a caller-provided `now` value when TTL is
  part of the adapter contract;
- view facts pass schema and constraint checks;
- combining a view with request facts is deterministic;
- redaction is applied before view diagnostics are shared outside trusted
  channels.

Docs-only verification for this issue should confirm this file links #167,
ADR 0009, and ADR 0010, and does not claim a cache API exists.
