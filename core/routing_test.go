package core

import "testing"

func TestIndexedRoutingPlanIsDefaultExplicitStrategy(t *testing.T) {
	plan := NewIndexedRoutingPlan(RoutingRule{Operation: "read.item", Target: "queue", Executor: "primary"})

	if plan.Strategy != IndexedRoutingStrategy {
		t.Fatalf("unexpected strategy: %#v", plan)
	}
	if plan.IsLinear() {
		t.Fatalf("indexed plan should not report linear cost")
	}
	if plan.Rules[0].Operation != "read.item" || plan.Rules[0].Target != "queue" {
		t.Fatalf("unexpected route: %#v", plan.Rules[0])
	}
}

func TestLinearRoutingPlanMakesONTradeoffExplicit(t *testing.T) {
	plan := NewLinearRoutingPlan(RoutingRule{Operation: "fallback", Target: "manual"})

	if plan.Strategy != LinearRoutingStrategy {
		t.Fatalf("unexpected strategy: %#v", plan)
	}
	if !plan.IsLinear() {
		t.Fatalf("linear plan should report linear cost")
	}
}
