# Materialized Facts Example

This example shows the caller-owned workflow for stable facts.

Use this pattern when a project has catalog, permission, hierarchy, route, or
inventory facts that are stable at startup or for a refresh interval.

## Workflow

1. Load stable facts and attach a stable input version from the source system.
2. Build an engine once.
3. Call `Prepare` at startup or on each refresh interval.
4. Keep request facts small.
5. Evaluate request facts with explicit budgets.
6. Include the stable input version, rule version, request facts hash, and
   option profile in any caller-owned cache key.

`Prepare` validates and materializes the stable set. When there are no request
facts, `PreparedEngine.Evaluate` returns the materialized result directly. When
request facts are present, it recombines original static facts and request facts
to preserve full-evaluation semantics for aggregates, negation, and
request-dependent derivations.

