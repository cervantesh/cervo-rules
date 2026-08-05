package facts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBindingLayoutUsesStableFirstSeenSlots(t *testing.T) {
	layout := NewBindingLayout("derive_access",
		Pattern{Predicate: "role", Terms: []Term{{Kind: TermVar, Value: "user"}, {Kind: TermConst, Value: "admin"}}},
		Pattern{Predicate: "owns", Terms: []Term{{Kind: TermVar, Value: "user"}, {Kind: TermVar, Value: "document"}}},
	)

	if len(layout.Variables) != 2 {
		t.Fatalf("unexpected layout: %#v", layout)
	}
	if slot, ok := layout.SlotFor("user"); !ok || slot != 0 {
		t.Fatalf("user slot mismatch: slot=%d ok=%v", slot, ok)
	}
	if slot, ok := layout.SlotFor("document"); !ok || slot != 1 {
		t.Fatalf("document slot mismatch: slot=%d ok=%v", slot, ok)
	}
	if _, ok := layout.SlotFor("missing"); ok {
		t.Fatalf("missing variable should not have a slot")
	}
}

func TestBindingFrameMaterializesPublicBindingOnlyAtBoundary(t *testing.T) {
	layout := NewBindingLayout("derive_access",
		Pattern{Predicate: "role", Terms: []Term{{Kind: TermVar, Value: "user"}, {Kind: TermVar, Value: "document"}}},
	)
	frame := NewBindingFrame(layout)

	if frame.Bind("missing", Term{Kind: TermConst, Value: "x"}) {
		t.Fatalf("unknown variable should not bind")
	}
	if !frame.Bind("user", Term{Kind: TermConst, Value: "ana"}) {
		t.Fatalf("user should bind")
	}
	if value, ok := frame.Lookup("user"); !ok || value.Value != "ana" {
		t.Fatalf("lookup mismatch: value=%#v ok=%v", value, ok)
	}
	if _, ok := frame.Lookup("document"); ok {
		t.Fatalf("unset document should not resolve")
	}
	public := frame.Binding()
	if len(public) != 1 || public["user"].Value != "ana" {
		t.Fatalf("public binding mismatch: %#v", public)
	}
}

func TestRecommendCompoundIndexesUsesPlanAndStatsEvidence(t *testing.T) {
	plan := NewEvaluationPlan(RulePlan{
		Name: "derive_access",
		Patterns: []PatternPlan{
			{DeclaredIndex: 0, Predicate: "role", Constants: 1, ConstantPositions: []int{1}, CandidateCount: 12},
			{DeclaredIndex: 1, Predicate: "owns", Constants: 0, CandidateCount: 900},
			{DeclaredIndex: 2, Predicate: "blocked", Negated: true, CandidateCount: 1200},
		},
	})
	stats := EvaluationStats{Rules: []RuleEvaluationStat{{Name: "derive_access", MaxBindings: 2000}}}

	recommendations := RecommendCompoundIndexes(plan, stats, CompoundIndexOptions{MinCandidateCount: 100, MinBindings: 500})
	if len(recommendations) != 2 {
		t.Fatalf("unexpected recommendations: %#v", recommendations)
	}
	if recommendations[0].Predicate != "owns" || recommendations[0].Reason != "high candidate count and binding fanout" {
		t.Fatalf("first recommendation mismatch: %#v", recommendations[0])
	}
	if recommendations[1].Predicate != "role" || recommendations[1].ConstantPositions[0] != 1 {
		t.Fatalf("constant-position recommendation mismatch: %#v", recommendations[1])
	}
	for _, recommendation := range recommendations {
		if recommendation.Predicate == "blocked" {
			t.Fatalf("negated pattern should not be recommended: %#v", recommendations)
		}
	}
}

func TestCompoundIndexRecommendationJSONIsStable(t *testing.T) {
	recommendations := RecommendCompoundIndexes(NewEvaluationPlan(RulePlan{
		Name: "derive_access",
		Patterns: []PatternPlan{
			{DeclaredIndex: 1, Predicate: "owns", ConstantPositions: []int{0, 2}, CandidateCount: 300},
		},
	}), EvaluationStats{}, CompoundIndexOptions{MinCandidateCount: 100})

	data, err := json.Marshal(recommendations)
	if err != nil {
		t.Fatalf("marshal recommendations: %v", err)
	}
	for _, want := range []string{
		`"rule":"derive_access"`,
		`"predicate":"owns"`,
		`"declared_index":1`,
		`"constant_positions":[0,2]`,
		`"reason":"high candidate count"`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("recommendation JSON missing %q: %s", want, data)
		}
	}
}
