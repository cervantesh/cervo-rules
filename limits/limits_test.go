package limits

import "testing"

func TestCheckAllowsEmptyRequestAgainstEmptyBudget(t *testing.T) {
	if violations := Check(Budget{}, Requested{}); len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestCheckReturnsAllLimitViolationsWithStableFields(t *testing.T) {
	violations := Check(Budget{
		MaxBodyBytes: 10,
		MaxTokens:    5,
	}, Requested{
		BodyBytes: 11,
		MaxTokens: 6,
		Stream:    true,
		Tools:     true,
		Images:    true,
	})

	for _, field := range []string{"body_bytes", "max_tokens", "stream", "tools", "images"} {
		if !violations.Has(field) {
			t.Fatalf("expected violation for %q, got %#v", field, violations)
		}
	}
	if violations.Error() == "" {
		t.Fatalf("expected non-empty error")
	}
}

func TestCheckAllowsOptionalFeaturesWhenBudgetAllowsThem(t *testing.T) {
	violations := Check(Budget{
		MaxBodyBytes: 10,
		MaxTokens:    5,
		AllowStream:  true,
		AllowTools:   true,
		AllowImages:  true,
	}, Requested{
		BodyBytes: 10,
		MaxTokens: 5,
		Stream:    true,
		Tools:     true,
		Images:    true,
	})
	if len(violations) != 0 {
		t.Fatalf("expected optional features to be allowed, got %#v", violations)
	}
}
