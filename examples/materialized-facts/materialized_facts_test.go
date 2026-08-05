package materializedfacts

import (
	"testing"

	"github.com/cervantesh/cervo-rules/v3/facts"
)

func TestMaterializedFactsWorkflowContract(t *testing.T) {
	staticFact := facts.Fact{
		Predicate: "tenant_plan",
		Terms: []facts.Term{
			{Kind: facts.TermConst, Value: "acme"},
			{Kind: facts.TermConst, Value: "pro"},
		},
	}
	requestFact := facts.Fact{
		Predicate: "request_tenant",
		Terms: []facts.Term{
			{Kind: facts.TermConst, Value: "req-1"},
			{Kind: facts.TermConst, Value: "acme"},
		},
	}
	options := facts.EvalOptions{
		MaxIterations: 4,
		MaxFacts:      32,
		MaxBindings:   32,
		Trace:         facts.TraceDisabled,
	}.Normalize()

	if options.TraceEnabled() {
		t.Fatalf("runtime materialization should keep trace disabled by default")
	}
	if staticFact.Predicate == requestFact.Predicate {
		t.Fatalf("stable and request facts should stay separated")
	}
}
