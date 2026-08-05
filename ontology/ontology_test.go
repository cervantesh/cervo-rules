package ontology

import (
	"context"
	"errors"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
)

// orderModel is the running example from the neuro-symbolic guardrail case:
// an order lifecycle plus the roles that must never collapse into one another.
func orderModel() Ontology {
	return Ontology{
		Entities: []EntityType{"order", "customer", "support_agent"},
		Signatures: []PredicateSignature{
			{Relation: "order.parent", Domain: "order", Range: "order"},
			{Relation: "requested_by", Domain: "order", Range: "customer"},
			{Relation: "refunded_by", Domain: "order", Range: "support_agent"},
		},
		Disjoint: []DisjointSet{
			{Name: "roles", Types: []EntityType{"customer", "support_agent"}},
		},
		Functional: []FunctionalProperty{
			{Relation: "order.parent"},
		},
		Lifecycles: []Lifecycle{{
			Name:    "order",
			Entity:  "order",
			States:  []State{"paid", "shipped", "refunded"},
			Initial: "paid",
			Transitions: []Transition{
				{From: "paid", To: []State{"shipped", "refunded"}},
				{From: "shipped", To: []State{"refunded"}},
				{From: "refunded", To: nil},
			},
		}},
	}
}

func TestValidateAcceptsWellFormedOntology(t *testing.T) {
	if errs := orderModel().Validate(); len(errs) > 0 {
		t.Fatalf("expected valid ontology, got %v", errs)
	}
}

func TestValidateRejectsUndeclaredEntityType(t *testing.T) {
	model := orderModel()
	model.Signatures = append(model.Signatures, PredicateSignature{
		Relation: "handled_by",
		Domain:   "order",
		Range:    "robot",
	})
	errs := model.Validate()
	if !hasCode(errs, ErrorCodeUnknownEntityType) {
		t.Fatalf("expected unknown_entity_type, got %v", errs)
	}
}

func TestValidateRejectsTransitionToUndeclaredState(t *testing.T) {
	model := orderModel()
	model.Lifecycles[0].Transitions = append(model.Lifecycles[0].Transitions, Transition{
		From: "paid",
		To:   []State{"vaporized"},
	})
	if errs := model.Validate(); !hasCode(errs, ErrorCodeUnknownState) {
		t.Fatalf("expected unknown_state, got %v", errs)
	}
}

func TestValidateFlagsUnreachableState(t *testing.T) {
	model := orderModel()
	model.Lifecycles[0].States = append(model.Lifecycles[0].States, "archived")
	if errs := model.Validate(); !hasCode(errs, ErrorCodeUnreachableState) {
		t.Fatalf("expected unreachable_state, got %v", errs)
	}
}

// Feature 1: disjointness. The agent must not treat one individual as both the
// customer who asked for the refund and the agent who approves it.
func TestDisjointTypesAreRejected(t *testing.T) {
	snapshot := Snapshot{Types: []TypeAssertion{
		{Individual: "u-1", Type: "customer"},
		{Individual: "u-1", Type: "support_agent"},
	}}
	errs := orderModel().Check(snapshot)
	if !hasCode(errs, ErrorCodeDisjointViolation) {
		t.Fatalf("expected disjoint_violation, got %v", errs)
	}
}

func TestDisjointAllowsSingleType(t *testing.T) {
	snapshot := Snapshot{Types: []TypeAssertion{
		{Individual: "u-1", Type: "customer"},
		{Individual: "u-2", Type: "support_agent"},
	}}
	if errs := orderModel().Check(snapshot); len(errs) > 0 {
		t.Fatalf("expected no violations, got %v", errs)
	}
}

// Feature 1: functional properties. An order has at most one parent.
func TestFunctionalPropertyRejectsSecondObject(t *testing.T) {
	snapshot := Snapshot{
		Types: []TypeAssertion{
			{Individual: "o-1", Type: "order"},
			{Individual: "o-2", Type: "order"},
			{Individual: "o-3", Type: "order"},
		},
		Relations: []RelationAssertion{
			{Relation: "order.parent", Subject: "o-1", Object: "o-2"},
			{Relation: "order.parent", Subject: "o-1", Object: "o-3"},
		},
	}
	errs := orderModel().Check(snapshot)
	if !hasCode(errs, ErrorCodeFunctionalViolation) {
		t.Fatalf("expected functional_violation, got %v", errs)
	}
}

func TestFunctionalPropertyIgnoresRepeatedIdenticalAssertion(t *testing.T) {
	snapshot := Snapshot{
		Types: []TypeAssertion{
			{Individual: "o-1", Type: "order"},
			{Individual: "o-2", Type: "order"},
		},
		Relations: []RelationAssertion{
			{Relation: "order.parent", Subject: "o-1", Object: "o-2"},
			{Relation: "order.parent", Subject: "o-1", Object: "o-2"},
		},
	}
	if errs := orderModel().Check(snapshot); len(errs) > 0 {
		t.Fatalf("expected idempotent assertion to pass, got %v", errs)
	}
}

// Feature 4: domain and range. A refund approved by a customer is a type error,
// not a plausible string.
func TestRangeViolationIsCaught(t *testing.T) {
	snapshot := Snapshot{
		Types: []TypeAssertion{
			{Individual: "o-1", Type: "order"},
			{Individual: "u-1", Type: "customer"},
		},
		Relations: []RelationAssertion{
			{Relation: "refunded_by", Subject: "o-1", Object: "u-1"},
		},
	}
	errs := orderModel().Check(snapshot)
	if !hasCode(errs, ErrorCodeRangeViolation) {
		t.Fatalf("expected range_violation, got %v", errs)
	}
}

func TestDomainViolationIsCaught(t *testing.T) {
	snapshot := Snapshot{
		Types: []TypeAssertion{
			{Individual: "u-1", Type: "customer"},
			{Individual: "a-1", Type: "support_agent"},
		},
		Relations: []RelationAssertion{
			{Relation: "refunded_by", Subject: "u-1", Object: "a-1"},
		},
	}
	errs := orderModel().Check(snapshot)
	if !hasCode(errs, ErrorCodeDomainViolation) {
		t.Fatalf("expected domain_violation, got %v", errs)
	}
}

// Feature 2: lifecycles. This is the duplicate-refund guard.
func TestSecondRefundHitsTerminalState(t *testing.T) {
	snapshot := Snapshot{States: []StateAssertion{
		{Lifecycle: "order", Individual: "o-1", State: "refunded"},
	}}
	errs := orderModel().CheckTransition(snapshot, "order", "o-1", "refunded")
	if !hasCode(errs, ErrorCodeIllegalTransition) {
		t.Fatalf("expected illegal_transition for repeated state, got %v", errs)
	}
}

func TestTransitionOutOfTerminalStateIsRejected(t *testing.T) {
	snapshot := Snapshot{States: []StateAssertion{
		{Lifecycle: "order", Individual: "o-1", State: "refunded"},
	}}
	errs := orderModel().CheckTransition(snapshot, "order", "o-1", "shipped")
	if !hasCode(errs, ErrorCodeTerminalState) {
		t.Fatalf("expected terminal_state, got %v", errs)
	}
}

func TestLegalTransitionIsAccepted(t *testing.T) {
	snapshot := Snapshot{States: []StateAssertion{
		{Lifecycle: "order", Individual: "o-1", State: "paid"},
	}}
	if errs := orderModel().CheckTransition(snapshot, "order", "o-1", "refunded"); len(errs) > 0 {
		t.Fatalf("expected paid -> refunded to be legal, got %v", errs)
	}
}

func TestUndeclaredTargetStateIsRejected(t *testing.T) {
	snapshot := Snapshot{States: []StateAssertion{
		{Lifecycle: "order", Individual: "o-1", State: "paid"},
	}}
	errs := orderModel().CheckTransition(snapshot, "order", "o-1", "incinerated")
	if !hasCode(errs, ErrorCodeUnknownState) {
		t.Fatalf("expected unknown_state, got %v", errs)
	}
}

func TestUnknownLifecycleIsRejected(t *testing.T) {
	errs := orderModel().CheckTransition(Snapshot{}, "invoice", "i-1", "paid")
	if !hasCode(errs, ErrorCodeUnknownLifecycle) {
		t.Fatalf("expected unknown_lifecycle, got %v", errs)
	}
}

// Feature 3: the guard answers named policy conditions.
func TestGuardAllowsFirstRefundAndBlocksSecond(t *testing.T) {
	states := map[string]State{"o-1": "paid", "o-2": "refunded"}
	guard, guardErrs := NewGuard(
		orderModel(),
		func(_ context.Context, req core.Request) (Snapshot, error) {
			id := req.Metadata["order_id"]
			return Snapshot{States: []StateAssertion{
				{Lifecycle: "order", Individual: Individual(id), State: states[id]},
			}}, nil
		},
		map[core.Condition]Check{
			"refund_allowed": {
				Kind:       CheckTransitionAllowed,
				Lifecycle:  "order",
				SubjectKey: "order_id",
				To:         "refunded",
			},
		},
	)
	requireValidOntology(t, guardErrs)

	first := core.Request{Operation: "refund", Metadata: map[string]string{"order_id": "o-1"}}
	holds, err := guard.Holds(context.Background(), "refund_allowed", first)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !holds {
		t.Fatal("expected first refund to be allowed")
	}

	second := core.Request{Operation: "refund", Metadata: map[string]string{"order_id": "o-2"}}
	holds, err = guard.Holds(context.Background(), "refund_allowed", second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if holds {
		t.Fatal("expected duplicate refund to be blocked")
	}
}

func TestGuardFailsClosedOnUnknownCondition(t *testing.T) {
	guard, guardErrs := NewGuard(orderModel(), func(context.Context, core.Request) (Snapshot, error) {
		return Snapshot{}, nil
	}, nil)
	requireValidOntology(t, guardErrs)
	holds, err := guard.Holds(context.Background(), "never_registered", core.Request{})
	if holds {
		t.Fatal("unknown condition must not report true")
	}
	var structured core.Error
	if !errors.As(err, &structured) || structured.Code != core.ErrorCodeUnknownCondition {
		t.Fatalf("expected unknown_condition error, got %v", err)
	}
}

func TestGuardFailsClosedOnMissingSubject(t *testing.T) {
	guard, guardErrs := NewGuard(orderModel(), func(context.Context, core.Request) (Snapshot, error) {
		return Snapshot{}, nil
	}, map[core.Condition]Check{
		"refund_allowed": {
			Kind:       CheckTransitionAllowed,
			Lifecycle:  "order",
			SubjectKey: "order_id",
			To:         "refunded",
		},
	})
	requireValidOntology(t, guardErrs)
	holds, err := guard.Holds(context.Background(), "refund_allowed", core.Request{Operation: "refund"})
	if holds {
		t.Fatal("missing subject must not report true")
	}
	var structured core.Error
	if !errors.As(err, &structured) || structured.Code != core.ErrorCodeConditionFailed {
		t.Fatalf("expected condition_failed error, got %v", err)
	}
}

func TestGuardPropagatesResolverFailure(t *testing.T) {
	sentinel := errors.New("database unavailable")
	guard, guardErrs := NewGuard(orderModel(), func(context.Context, core.Request) (Snapshot, error) {
		return Snapshot{}, sentinel
	}, map[core.Condition]Check{
		"refund_allowed": {Kind: CheckIntegrity},
	})
	requireValidOntology(t, guardErrs)
	_, err := guard.Holds(context.Background(), "refund_allowed", core.Request{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected resolver cause to be preserved, got %v", err)
	}
}

func TestCoreConditionSetFailsClosed(t *testing.T) {
	set := core.ConditionSet{}
	holds, err := set.Holds(context.Background(), "missing", core.Request{})
	if holds {
		t.Fatal("empty condition set must not report true")
	}
	var structured core.Error
	if !errors.As(err, &structured) || structured.Code != core.ErrorCodeUnknownCondition {
		t.Fatalf("expected unknown_condition, got %v", err)
	}
}

// Determinism matters: the same snapshot must always produce the same ordered
// diagnostics, otherwise generated reports churn between runs.
func TestCheckIsDeterministic(t *testing.T) {
	snapshot := Snapshot{Types: []TypeAssertion{
		{Individual: "u-2", Type: "support_agent"},
		{Individual: "u-1", Type: "support_agent"},
		{Individual: "u-1", Type: "customer"},
		{Individual: "u-2", Type: "customer"},
	}}
	first := orderModel().Check(snapshot)
	for i := 0; i < 20; i++ {
		next := orderModel().Check(snapshot)
		if len(next) != len(first) {
			t.Fatalf("unstable diagnostic count: %d vs %d", len(next), len(first))
		}
		for j := range first {
			if next[j].Value != first[j].Value || next[j].Code != first[j].Code {
				t.Fatalf("unstable diagnostic order at %d", j)
			}
		}
	}
}

func hasCode(errs core.Errors, code core.ErrorCode) bool {
	for _, err := range errs {
		if err.Code == code {
			return true
		}
	}
	return false
}
