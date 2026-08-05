package limits

import (
	"encoding/json"
	"testing"
)

func TestBudgetJSONTagsMatchRuntimeConfigNames(t *testing.T) {
	var budget Budget
	if err := json.Unmarshal([]byte(`{"max_body_bytes":10,"max_tokens":5,"allow_stream":true,"allow_tools":true,"allow_images":true}`), &budget); err != nil {
		t.Fatalf("unmarshal budget: %v", err)
	}
	if budget.MaxBodyBytes != 10 || budget.MaxTokens != 5 || !budget.AllowStream || !budget.AllowTools || !budget.AllowImages {
		t.Fatalf("unexpected budget: %#v", budget)
	}
}
