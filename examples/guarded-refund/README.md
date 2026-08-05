# Guarded Refund

The failure this example prevents is the one routing cannot see: a caller asks
to refund an order that was already refunded. From a routing point of view the
second request is identical to the first — same operation, same shape, same
target. Nothing about it is wrong until you consult the world.

## The three parts

**The policy names a guard.** [`policy-rules.yaml`](policy-rules.yaml) declares
a condition and requires it:

```yaml
conditions:
  refund_transition_legal:
    kind: transition_allowed
    lifecycle: order
    subject_key: order_id
    to: refunded

routes:
  - id: refund_route
    operation: refund
    target: ledger
    executor: primary
    requires: [refund_transition_legal]
```

The policy references the guard by name only. What the name means is bound in
Go, so the same policy text runs against an ontology guard, a plain
`core.ConditionSet`, or a fake in tests.

**The ontology answers it.** The lifecycle declares that `refunded` has no
outgoing transition. That single fact is the guard: a second refund is
structurally impossible rather than something a reviewer has to remember to
check.

```go
Lifecycles: []ontology.Lifecycle{{
    Name:    "order",
    States:  []ontology.State{"paid", "shipped", "refunded"},
    Initial: "paid",
    Transitions: []ontology.Transition{
        {From: "paid", To: []ontology.State{"shipped", "refunded"}},
        {From: "shipped", To: []ontology.State{"refunded"}},
        {From: "refunded", To: nil}, // terminal
    },
}},
```

**The decision stays pure.** Nothing executes. The engine returns a
`DecisionResult`, and the caller decides what to do with it. That ordering is
the whole point: the guard runs before any side effect exists to undo.

## Reading the refusal

`Holds` answers yes or no because that is what the `core.Conditions` contract
needs. When you have to tell whoever proposed the action *why* it was refused,
use `Explain`:

```go
holds, violations, err := guard.Explain(ctx, "refund_transition_legal", req)
// violations → illegal_transition: individual is already in state refunded
```

The two return values are different questions. `violations` says why the world
is illegal; `err` says the guard could not answer at all.

## Fail closed

A request missing `order_id` is unanswerable, and an unanswerable guard returns
an error rather than `false`. A `false` would read as "the guard checked and
found nothing wrong" — the exact silent failure this layer exists to prevent.

The same rule runs through the whole path: an unknown condition, an absent
resolver, or a policy built without an evaluator all fail loudly. The generator
goes further and rejects a rule requiring an undeclared condition at build time,
so an unwired guard breaks CI instead of a production decision.

## Cost

Install one request scope per decision:

```go
ctx = ontology.WithRequestScope(ctx)
```

Without it, each consulted condition resolves state again — three conditions
means three round-trips for one decision. With it, one resolution per request,
and every condition sees the same world. Evaluating one request against two
different snapshots is a coherence bug, not a feature.

## Run it

```bash
go test ./examples/guarded-refund/
```
