# CervoRules v3 Routing

Tracks epic #354 and child issues #355, #356, #357, #358, and #359.

## Goal

v3 removes implicit or ambiguous linear routing. Routing cost must be visible in
the API and generated policies should use indexed operation routing by default.

## Decisions

- RoutingPhase is not part of the v3 public API.
- `IndexedRoutingStrategy` is the default generated-policy routing path.
- `LinearRoutingStrategy` remains available only through explicit names.
- Linear routing is O(n) over route count and should be reserved for small,
  ordered, custom, or non-indexable rule sets.

## Migration

v2 code that used implicit `RoutingPhase(rules...)` should move to one of:

- generated policy factory routing, which should use indexed routing;
- `NewIndexedRoutingPlan(...)` for operation-keyed routing;
- `NewLinearRoutingPlan(...)` only when global route order is required.

Migration tooling should flag `RoutingPhase(` as deprecated v2 usage and suggest
an explicit indexed or linear replacement.

## Performance Note

Indexed routing keeps operation lookup independent of total route count in the
normal generated-policy path. Linear routing evaluates routes in order and has
cost proportional to route count.
