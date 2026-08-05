package logicfacts_test

import (
	"testing"

	"github.com/cervantesh/cervo-rules/v3/facts"
)

func TestLogicFactsExampleDocumentsBoundedPlan(t *testing.T) {
	options := facts.EvalOptions{
		MaxIterations: 4,
		MaxFacts:      16,
		MaxBindings:   16,
		Trace:         facts.TraceEnabled,
	}
	if !options.TraceEnabled() {
		t.Fatalf("example should enable trace for explain workflows")
	}

	plan := facts.NewEvaluationPlan(facts.RulePlan{
		Name:      "enterprise-tenants-use-priority-lane",
		Stratum:   0,
		Reordered: true,
		Patterns: []facts.PatternPlan{{
			DeclaredIndex:     0,
			Predicate:         "tenant_plan",
			Constants:         1,
			CandidateCount:    1,
			SelectivityReason: "constant terms are evaluated first",
		}},
	})

	if plan.SchemaVersion == "" {
		t.Fatalf("facts plans must be machine-readable")
	}
	if len(plan.Rules) != 1 || !plan.Rules[0].Reordered {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}
