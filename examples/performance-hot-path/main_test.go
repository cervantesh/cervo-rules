package performancehotpath

import (
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
	"github.com/cervantesh/cervo-rules/v3/facts"
)

func TestPerformanceHotPathShape(t *testing.T) {
	decisionOptions := core.NewDecisionOptions(
		core.WithTrace(false),
		core.WithObservation(false),
	)
	if decisionOptions.TraceEnabled() || decisionOptions.ObservationEnabled() {
		t.Fatalf("hot path should opt out of trace and observation")
	}

	routePlan := core.NewIndexedRoutingPlan(core.RoutingRule{
		Operation: core.NewOperation("invoice.write"),
		Target:    core.NewTarget("invoice_writer"),
		Executor:  core.NewExecutor("standard_executor"),
	})
	if routePlan.IsLinear() {
		t.Fatalf("hot path examples should prefer indexed routing")
	}

	factOptions := facts.EvalOptions{
		MaxIterations: 4,
		MaxFacts:      16,
		MaxBindings:   16,
		Trace:         facts.TraceDisabled,
	}.Normalize()
	if factOptions.TraceEnabled() {
		t.Fatalf("request-path facts should keep trace disabled unless explaining")
	}
}
