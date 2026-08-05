package core

import (
	"strings"
	"testing"
)

func TestNeutralPrimitiveConstructorsNormalizeValues(t *testing.T) {
	if got := NewOperation("  Read.Item  "); got != Operation("read.item") {
		t.Fatalf("unexpected operation: %q", got)
	}
	if got := NewTarget("  Billing.Worker  "); got != Target("billing.worker") {
		t.Fatalf("unexpected target: %q", got)
	}
	if got := NewExecutor("  Primary.Runner  "); got != Executor("primary.runner") {
		t.Fatalf("unexpected executor: %q", got)
	}
}

func TestVocabularyUsesOperationTargetExecutorNames(t *testing.T) {
	vocab := NewVocabulary(
		AllowedOperations("read.item"),
		AllowedTargets("billing.worker"),
		AllowedExecutors("primary.runner"),
	)

	if err := vocab.ValidateOperation("read.item"); err != nil {
		t.Fatalf("operation should be valid: %v", err)
	}
	if err := vocab.ValidateTarget("billing.worker"); err != nil {
		t.Fatalf("target should be valid: %v", err)
	}
	if err := vocab.ValidateExecutor("primary.runner"); err != nil {
		t.Fatalf("executor should be valid: %v", err)
	}
	if err := vocab.ValidateRequest(Request{Operation: "missing"}); err == nil || !strings.Contains(err.Error(), "unknown operation") {
		t.Fatalf("expected unknown operation error, got %v", err)
	}
	if err := vocab.ValidateDecision(Decision{Target: "missing"}); err == nil || !strings.Contains(err.Error(), "unknown target") {
		t.Fatalf("expected unknown target error, got %v", err)
	}
	if err := vocab.ValidateDecision(Decision{Executor: "missing"}); err == nil || !strings.Contains(err.Error(), "unknown executor") {
		t.Fatalf("expected unknown executor error, got %v", err)
	}
}
