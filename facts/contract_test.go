package facts

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
)

func TestEvalOptionsDefaultTraceIsOptIn(t *testing.T) {
	options := EvalOptions{MaxIterations: 4, MaxFacts: 128, ExpensiveRuleBindingThreshold: 32}.Normalize()

	if options.Trace != TraceDisabled {
		t.Fatalf("trace should default to disabled in v3: %#v", options)
	}
	if options.ExpensiveRuleBindingThreshold != 32 {
		t.Fatalf("normalization should preserve binding threshold: %#v", options)
	}
	if options.TraceEnabled() {
		t.Fatalf("trace should be opt-in")
	}

	options.Trace = TraceEnabled
	if !options.TraceEnabled() {
		t.Fatalf("explicit trace should be enabled")
	}
}

func TestEvaluationPlanJSONIsVersionedAndStable(t *testing.T) {
	plan := NewEvaluationPlan(RulePlan{
		Name:      "derive_access",
		Stratum:   1,
		Reordered: true,
		Patterns: []PatternPlan{
			{DeclaredIndex: 1, Predicate: "role", Constants: 1, CandidateCount: 2, SelectivityReason: "constant_index"},
			{DeclaredIndex: 2, Predicate: "blocked", Negated: true, CandidateCount: 0},
		},
	})

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	for _, want := range []string{
		`"schema_version":"cervorules.v3.facts.plan.v1"`,
		`"declared_index":1`,
		`"candidate_count":2`,
		`"selectivity_reason":"constant_index"`,
		`"negated":true`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("plan JSON missing %q: %s", want, data)
		}
	}
}

func TestComplexityDiagnosticJSONUsesStableFields(t *testing.T) {
	diagnostic := ComplexityDiagnostic{
		Code:     "max_facts",
		Rule:     "derive_access",
		Field:    "facts",
		Limit:    128,
		Observed: 129,
		Reason:   "fact budget exceeded",
	}

	data, err := json.Marshal(diagnostic)
	if err != nil {
		t.Fatalf("marshal diagnostic: %v", err)
	}
	for _, want := range []string{
		`"code":"max_facts"`,
		`"rule":"derive_access"`,
		`"limit":128`,
		`"observed":129`,
		`"reason":"fact budget exceeded"`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("diagnostic JSON missing %q: %s", want, data)
		}
	}
}

func TestResultJSONKeepsFactsPlanDiagnosticsAndStatsSeparate(t *testing.T) {
	plan := NewEvaluationPlan(RulePlan{Name: "derive_access"})
	result := Result{
		Facts: []Fact{
			{Predicate: "access", Terms: []Term{{Kind: TermConst, Value: "doc-1"}}},
		},
		Plan: &plan,
		Diagnostics: []ComplexityDiagnostic{
			{Code: "expensive_rule", Rule: "derive_access", Reason: "derived facts exceeded threshold"},
		},
		Stats: EvaluationStats{Iterations: 2, Rules: []RuleEvaluationStat{{Name: "derive_access", Evaluations: 2, DerivedFacts: 1, MaxBindings: 4}}},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	for _, want := range []string{
		`"predicate":"access"`,
		`"schema_version":"cervorules.v3.facts.plan.v1"`,
		`"code":"expensive_rule"`,
		`"iterations":2`,
		`"derived_facts":1`,
		`"max_bindings":4`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("result JSON missing %q: %s", want, data)
		}
	}
}

func TestStructuredErrorsFromComplexityDiagnosticsMapsFatalBudgets(t *testing.T) {
	diagnostics := []ComplexityDiagnostic{
		{Code: "max_bindings", Rule: "derive_access", Field: "bindings", Limit: 100, Observed: 101, Reason: "binding budget exceeded"},
		{Code: "max_facts", Rule: "derive_access", Field: "facts", Limit: 10, Observed: 11, Reason: "fact budget exceeded"},
		{Code: "max_iterations", Rule: "closure", Field: "iterations", Limit: 4, Observed: 5, Reason: "iteration budget exceeded"},
		{Code: "unsafe_negation", Rule: "deny_without_binding", Field: "rule", Reason: "negated variable is not bound"},
		{Code: "unsafe_rule", Rule: "bad_rule", Field: "rule", Reason: "rule is unsafe"},
		{Code: "expensive_rule", Rule: "broad_join", Field: "bindings", Limit: 32, Observed: 2048, Reason: "rule is expensive"},
	}

	errs := StructuredErrorsFromDiagnostics(diagnostics)
	if len(errs) != 5 {
		t.Fatalf("expected five fatal structured errors, got %#v", errs)
	}
	for _, code := range []core.ErrorCode{
		core.ErrorCodeMaxBindingsExceeded,
		core.ErrorCodeMaxFactsExceeded,
		core.ErrorCodeMaxIterationsExceeded,
		core.ErrorCodeUnsafeNegation,
		core.ErrorCodeUnsafeRule,
	} {
		if !errs.Has(code) {
			t.Fatalf("missing structured code %s in %#v", code, errs)
		}
	}
	if got := errs.ByCode(core.ErrorCodeMaxBindingsExceeded); len(got) != 1 ||
		got[0].Severity != core.SeverityFatal ||
		got[0].Component != "facts" ||
		got[0].Rule != "derive_access" ||
		got[0].Limit != 100 ||
		got[0].Observed != 101 {
		t.Fatalf("unexpected max bindings error: %#v", got)
	}
	if errs.Has(core.ErrorCodeExpensiveRule) {
		t.Fatalf("expensive_rule should remain a non-fatal diagnostic, got %#v", errs)
	}
}

func TestStructuredErrorFromDiagnosticReturnsFalseForNonFatalDiagnostic(t *testing.T) {
	_, ok := StructuredErrorFromDiagnostic(ComplexityDiagnostic{
		Code:     "expensive_rule",
		Rule:     "broad_join",
		Observed: 2048,
		Reason:   "rule exceeded tuning threshold",
	})
	if ok {
		t.Fatalf("expensive rule diagnostics should not become fatal errors")
	}
}
