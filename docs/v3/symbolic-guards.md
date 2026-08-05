# Symbolic Guards

A decision engine answers "which route handles this operation?". It does not
answer "is the world this request describes even coherent?". The `ontology`
package answers the second question, and the `Condition` seam lets a policy
consult it.

See [ADR 0014](../adr/0014-symbolic-guard-layer.md) for the reasoning, and
[examples/guarded-refund](../../examples/guarded-refund) for a runnable case.

## What the package expresses

| Constraint | Declares | Catches |
| --- | --- | --- |
| `PredicateSignature` | domain and range of a relation | a refund approved by the customer who requested it |
| `DisjointSet` | types one individual may not hold together | a user who is both customer and support agent |
| `FunctionalProperty` | a relation with at most one object | an order silently acquiring a second parent |
| `Lifecycle` | legal state transitions | a second refund, or any action after a terminal state |

`Lifecycle` needs no special "already applied" rule. A state with no outgoing
transition is terminal, so repeating a terminal action is refused by the shape of
the declaration rather than by a rule someone remembered to write.

## Declaring in the DSL

A policy names the guards it requires. `requires` gates whether a rule applies,
and reads as a sibling of the existing `requires_trusted_user`:

```yaml
conditions:
  refund_transition_legal:
    kind: transition_allowed   # or in_state, has_type, integrity
    lifecycle: order
    subject_key: order_id      # which Request.Metadata entry carries the id
    to: refunded

routes:
  - operation: refund
    target: ledger
    executor: primary
    requires: [refund_transition_legal]

denies:
  - operation: reparent
    reason: order already has a parent
    requires: [order_has_parent]
```

Semantics are uniform across both rule kinds:

- every listed condition holds → the rule applies;
- a condition returns false → the rule does not apply;
- a condition **cannot be answered** → the whole decision fails.

That last line is the important one. Falling through past a guard that could not
answer is precisely the silent failure the layer exists to prevent.

The policy references names only. What a name means is bound in Go, so the same
policy text runs against an ontology guard, a plain `core.ConditionSet`, or a
fake in tests.

## Wiring

```go
guard, errs := ontology.NewGuard(model, resolveSnapshot, checks)
if len(errs) > 0 {
    return errs // an invalid ontology enforces less than it claims
}

engine, err := factory.Build(ctx, runtime.PolicyRuntimeConfig{
    Conditions: guard,
})
```

`NewGuard` validates eagerly and returns the result rather than dropping it. A
constraint whose declaration is incomplete silently enforces nothing, so the
result must be treated as fatal.

A policy that declares conditions and is built without an evaluator is rejected
by `ValidateConfig`. It would otherwise allow exactly what the guards exist to
stop.

## Cost

```go
ctx = ontology.WithRequestScope(ctx)
result, err := engine.Decide(ctx, req)
```

One resolution per request instead of one per condition, and every condition
observes the same world. Reusing a scope across requests is refused: answering a
second request against the first one's snapshot is a wrong answer with nothing to
notice it.

Nothing is retained beyond the context. CervoRules still holds no live state.

## Explaining a refusal

```go
holds, violations, err := guard.Explain(ctx, condition, req)
```

`violations` says why the world is illegal. `err` says the guard could not
answer. `Holds` returns the same answer without the reason, because that is what
the `core.Conditions` contract needs.

## Extending

The built-in families are deliberately few. Rather than wait for this package to
grow cardinality, hierarchies or equality, implement `Constraint`:

```go
type Constraint interface {
    Name() string
    Validate(model Ontology) core.Errors            // build time
    Check(model Ontology, snapshot Snapshot) core.Errors // decision time
}

model.Custom = []ontology.Constraint{maxCardinality{relation: "tag", max: 3}}
```

Custom constraints get the same structured errors and the same CI gate as the
built-ins. `Custom` is `json:"-"` on purpose: the declarative fields stay
portable to JSON and to the DSL, while extensions stay code.

Property hierarchies and equality are out of scope as *reasoning*, but both are
reachable by materialization inside your `SnapshotResolver` — expand a
sub-relation onto its super-relation, or map merged identifiers to a canonical
one, before returning the snapshot. That is how the `facts` package already
thinks, and it keeps a reasoner out of the module.

## What this is not

No RDF parser, triple store or reasoner enters the module graph. The vocabulary
and semantics come from RDFS and OWL; the stack does not. Closed-world semantics
are kept from `facts`: an absent assertion is false, not unknown, because "has
this already been applied?" needs a decidable answer and open-world logic
answers "unknown".
