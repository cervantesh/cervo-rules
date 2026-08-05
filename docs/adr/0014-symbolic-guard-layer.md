# 0014 - Symbolic Guard Layer

## Status

Accepted.

## Context

A decision engine answers "which route handles this operation?". It does not
answer "is the world this request describes even coherent?". Consumers that let
a probabilistic caller propose actions need the second question answered before
anything executes, and today they have to answer it by hand.

CervoRules already owns most of the machinery. `facts` is a bounded logic engine
with stratified negation, bounded recursion and closed-world semantics
(ADRs 0005-0012). `core` returns a pure `DecisionResult` and executes nothing.
What is missing is the small set of constraints that business policy actually
needs, and a seam that lets a policy consult them.

Three constraint families cover the recurring failures:

- an individual holding two roles that must never coincide;
- a relation silently acquiring a second value when it should hold one;
- an entity moving to a state its lifecycle does not permit, most commonly
  repeating an already-applied terminal action.

The description-logic tradition (RDFS domain/range, OWL disjointness and
functional properties) has good names and semantics for all three.

## Decision

Add an optional `ontology` package that expresses those constraints as plain Go
values, and add a `Condition` seam to `core` so a policy can consult a guard.

`ontology` provides:

- `EntityType`, `Individual`, `Relation`, `State` and a closed-world `Snapshot`;
- `PredicateSignature` for domain/range;
- `DisjointSet` and `FunctionalProperty`;
- `Lifecycle` with declarative transitions, where a state with no outgoing
  transition is terminal;
- `Guard`, which implements `core.Conditions` over an ontology plus a
  caller-supplied `SnapshotResolver`.

`core` gains `Condition`, `Conditions`, `ConditionFunc` and `ConditionSet`.

Adopt the vocabulary and the semantics of RDFS/OWL. Do not adopt the stack. No
RDF parser, triple store or reasoner enters the module graph. Any interop with
external ontology formats belongs in a separate module whose dependencies never
reach the root `go.mod`.

Closed world is retained from `facts`: an absent assertion is false, not
unknown. Guard questions such as "has this already been applied?" need a
decidable answer, and OWL's open world does not provide one.

Every check is side-effect free and returns `core.Errors`, so a caller can
validate a proposed action and still choose not to perform it.

## Consequences

### Positive

- Constraints that previously lived in ad-hoc consumer code become declarative
  and testable.
- Failures are structured `core.Errors` with codes, so a caller can feed the
  reason back to whatever proposed the action.
- `ontology` is optional and importing it is the only way to pay for it. The
  decision hot path is untouched.
- `Validate` separates authoring mistakes from runtime conditions, so an
  unreachable state or an undeclared entity type fails in CI.

### Negative, and what mitigates each

**`core` gains public primitives.** `Condition` and `Conditions` pass the
agnosticism review in `docs/agnosticism.md`: a named boolean guard implies no
transport, no provider and no single domain. The real cost is not the surface,
it is that the surface is currently *unwired* — nothing in `core.Engine`
consults it. An unwired public primitive is worse than either possible home for
it, because there is no proof the shape is right. Mitigation: wire
`runtime.PolicyRuntimeConfig.Conditions` and have generated engines consult it,
which converts speculative surface into justified surface. Until that lands,
treat the placement as provisional.

**A snapshot must be resolved per request.** The first implementation made this
worse than stated: `Guard.Holds` resolved on *every* condition, so a policy
consulting three conditions cost three resolver round-trips for one decision.
Mitigated by `WithRequestScope`, which memoizes one resolution — and one
normalization — for the lifetime of a request, including a failed one. Opting
out preserves the previous behavior. Nothing is retained beyond the context, so
CervoRules still holds no live state across requests. Scoped fetching, where a
resolver is told which slice of the world a check needs, is deferred until a
consumer has a snapshot too large to fetch whole; it is additive to this.

**Constraint coverage is intentionally partial.** Mitigated by making it an
extension point rather than a fixed limit. `Constraint` mirrors the shape the
built-in families already use through unexported methods — they are not
refactored onto the interface, since `Custom` is the extension point and the
built-ins stay declarative. `Ontology.Custom` fans out to consumer
implementations from both `Validate` and `Check`. A consumer needing
max-cardinality, symmetry or a domain-specific rule writes it against `Snapshot`
and gets the same structured errors and the same CI gate, without waiting for
upstream. `Custom` is `json:"-"` on purpose: the declarative fields stay
portable to JSON and to a future DSL, while extensions stay code. Property
hierarchies and equality remain out of scope as *reasoning*; both are reachable
by materialization in the resolver, which is how `facts` already thinks.

### Error code ownership

`core` owns the `Error` shape, the `Severity` scale and the open `ErrorCode`
type. Each package owns the code constants it is the sole producer of, declared
locally as `core.ErrorCode` values so wire strings and `Errors.Has` / `ByCode`
are unaffected. Codes belonging to a core-owned contract stay in `core`:
`unknown_condition`, `condition_failed` and `missing_conditions` are part of the
`Conditions` seam, not of this package's model.

This rule exists because the alternative has no stopping point. Without it
`core/errors.go` grows a term every time an optional package invents a check,
until a reader of `core` needs a leaf package's concepts to understand core's
vocabulary. Codes already released under v3 are not retro-migrated.

### Fail-closed rule

An unknown condition, an absent resolver or a request missing the metadata a
check needs returns a structured error, never `false`. A `false` would read as
"the guard ran and found nothing wrong", which is the exact silent failure this
layer exists to prevent.

## Alternatives Considered

**Embed an OWL reasoner.** Rejected. It contradicts the zero-dependency
posture, and open-world semantics answer "unknown" to the questions guards ask.

**Extend `facts` instead of adding a package.** Rejected. `facts` owns derivation
of new facts; this layer owns refutation of an incoherent world. Merging them
would put constraint checking on the derivation path.

**Put constraints in the policy DSL only.** Rejected as a starting point. The Go
API has to exist first; DSL surface for it is a separate decision.
