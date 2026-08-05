// Package guardedrefund shows the full loop a symbolic guard closes: a policy
// names a condition, an ontology answers it from recorded state, and the engine
// refuses the action before anything executes.
//
// The failure this prevents is the canonical one: a caller proposes refunding
// an order that was already refunded. Routing alone cannot see it, because from
// a routing point of view the second request is identical to the first.
package guardedrefund

import (
	"context"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
	"github.com/cervantesh/cervo-rules/v3/ontology"
)

// orderOntology declares the lifecycle. "refunded" has no outgoing transition,
// which is what makes it terminal — the duplicate-refund guard is a property of
// the declaration, not a rule someone remembered to write.
func orderOntology() ontology.Ontology {
	return ontology.Ontology{
		Entities: []ontology.EntityType{"order"},
		Lifecycles: []ontology.Lifecycle{{
			Name:    "order",
			Entity:  "order",
			States:  []ontology.State{"paid", "shipped", "refunded"},
			Initial: "paid",
			Transitions: []ontology.Transition{
				{From: "paid", To: []ontology.State{"shipped", "refunded"}},
				{From: "shipped", To: []ontology.State{"refunded"}},
				{From: "refunded", To: nil},
			},
		}},
	}
}

// store stands in for whatever records order state in a real system.
type store map[string]ontology.State

func (s store) resolve(_ context.Context, req core.Request) (ontology.Snapshot, error) {
	id := req.Metadata["order_id"]
	state, ok := s[id]
	if !ok {
		return ontology.Snapshot{}, nil
	}
	return ontology.Snapshot{States: []ontology.StateAssertion{
		{Lifecycle: "order", Individual: ontology.Individual(id), State: state},
	}}, nil
}

func newGuard(t *testing.T, orders store) ontology.Guard {
	t.Helper()
	guard, errs := ontology.NewGuard(orderOntology(), orders.resolve, map[core.Condition]ontology.Check{
		"refund_transition_legal": {
			Kind:       ontology.CheckTransitionAllowed,
			Lifecycle:  "order",
			SubjectKey: "order_id",
			To:         "refunded",
		},
		"ship_transition_legal": {
			Kind:       ontology.CheckTransitionAllowed,
			Lifecycle:  "order",
			SubjectKey: "order_id",
			To:         "shipped",
		},
	})
	if len(errs) > 0 {
		t.Fatalf("ontology is not valid: %v", errs)
	}
	return guard
}

func TestGuardAllowsAFirstRefundAndRefusesTheSecond(t *testing.T) {
	orders := store{"o-1": "paid", "o-2": "refunded"}
	guard := newGuard(t, orders)

	ctx := ontology.WithRequestScope(context.Background())
	first := core.Request{
		ID:        "req-1",
		Operation: "refund",
		Metadata:  map[string]string{"order_id": "o-1"},
	}
	holds, err := guard.Holds(ctx, "refund_transition_legal", first)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !holds {
		t.Fatal("refunding a paid order must be allowed")
	}

	// A fresh scope, because this is a different request.
	ctx = ontology.WithRequestScope(context.Background())
	second := core.Request{
		ID:        "req-2",
		Operation: "refund",
		Metadata:  map[string]string{"order_id": "o-2"},
	}
	holds, violations, err := guard.Explain(ctx, "refund_transition_legal", second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if holds {
		t.Fatal("refunding an already-refunded order must be refused")
	}
	if !violations.Has(ontology.ErrorCodeIllegalTransition) {
		t.Fatalf("expected an illegal_transition explanation, got %v", violations)
	}
	// The reason is machine-readable, so a caller can hand it back to whatever
	// proposed the action instead of just saying no.
	if violations[0].Reason == "" {
		t.Fatal("a refusal must carry a reason")
	}
}

// Shipping an order that is already refunded is refused for a different reason:
// the current state is terminal, so no transition out of it is legal.
func TestTerminalStateRefusesAnyFurtherAction(t *testing.T) {
	guard := newGuard(t, store{"o-2": "refunded"})
	ctx := ontology.WithRequestScope(context.Background())
	req := core.Request{ID: "req-3", Operation: "ship", Metadata: map[string]string{"order_id": "o-2"}}

	holds, violations, err := guard.Explain(ctx, "ship_transition_legal", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if holds {
		t.Fatal("shipping a refunded order must be refused")
	}
	if !violations.Has(ontology.ErrorCodeTerminalState) {
		t.Fatalf("expected terminal_state, got %v", violations)
	}
}

// A request that omits the id the guard needs is unanswerable. It must fail
// closed with an error rather than report false, which would read as "checked,
// nothing wrong".
func TestMissingOrderIDFailsClosed(t *testing.T) {
	guard := newGuard(t, store{})
	req := core.Request{ID: "req-4", Operation: "refund"}
	holds, err := guard.Holds(context.Background(), "refund_transition_legal", req)
	if holds {
		t.Fatal("an unanswerable guard must not report true")
	}
	if err == nil {
		t.Fatal("an unanswerable guard must report an error, not a bare false")
	}
}

// One decision consulting several conditions must resolve state once.
func TestOneRequestResolvesStateOnce(t *testing.T) {
	calls := 0
	orders := store{"o-1": "paid"}
	counted := func(ctx context.Context, req core.Request) (ontology.Snapshot, error) {
		calls++
		return orders.resolve(ctx, req)
	}
	guard, errs := ontology.NewGuard(orderOntology(), counted, map[core.Condition]ontology.Check{
		"refund_transition_legal": {Kind: ontology.CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "refunded"},
		"ship_transition_legal":   {Kind: ontology.CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "shipped"},
	})
	if len(errs) > 0 {
		t.Fatalf("ontology is not valid: %v", errs)
	}

	ctx := ontology.WithRequestScope(context.Background())
	req := core.Request{ID: "req-5", Operation: "refund", Metadata: map[string]string{"order_id": "o-1"}}
	for _, condition := range []core.Condition{"refund_transition_legal", "ship_transition_legal"} {
		if _, err := guard.Holds(ctx, condition, req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("two conditions on one request must resolve state once, got %d", calls)
	}
}
