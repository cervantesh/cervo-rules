package runtime

import (
	"context"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
)

func TestPolicyFactoryContractRequiresCanonicalMethods(t *testing.T) {
	factory := fakePolicyFactory{}
	var contract PolicyFactory = factory

	cfg := contract.DefaultConfig()
	if err := contract.ValidateConfig(cfg); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}

	engine, err := contract.Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	result, err := engine.Decide(context.Background(), core.Request{ID: "req-1", Operation: "read.item"})
	if err != nil {
		t.Fatalf("decide failed: %v", err)
	}
	if !result.Decision.Allow {
		t.Fatalf("expected allow decision: %#v", result.Decision)
	}

	metadata := contract.Metadata()
	if metadata.Name == "" || metadata.DSLVersion == "" || metadata.GeneratedWith == "" || metadata.VocabularyHash == "" {
		t.Fatalf("metadata must be complete: %#v", metadata)
	}
}

func TestPolicyRuntimeConfigCopiesMutableFields(t *testing.T) {
	cfg := PolicyRuntimeConfig{
		TrustedUsers: []string{"ada"},
		OperationTargets: map[core.Operation]core.Target{
			"read.item": "queue",
		},
		ExecutorFallbacks: map[core.Executor][]core.Executor{
			"primary": {"fallback"},
		},
	}

	copied := cfg.Clone()
	copied.TrustedUsers[0] = "mutated"
	copied.OperationTargets["read.item"] = "archive"
	copied.ExecutorFallbacks["primary"][0] = "mutated"

	if cfg.TrustedUsers[0] != "ada" {
		t.Fatalf("trusted users should be copied defensively: %#v", cfg.TrustedUsers)
	}
	if cfg.OperationTargets["read.item"] != "queue" {
		t.Fatalf("operation targets should be copied defensively: %#v", cfg.OperationTargets)
	}
	if cfg.ExecutorFallbacks["primary"][0] != "fallback" {
		t.Fatalf("executor fallbacks should be copied defensively: %#v", cfg.ExecutorFallbacks)
	}
}

type fakePolicyFactory struct{}

func (fakePolicyFactory) DefaultConfig() PolicyRuntimeConfig {
	return PolicyRuntimeConfig{
		TrustedUsers:    []string{"ada"},
		DefaultExecutor: "primary",
		OperationTargets: map[core.Operation]core.Target{
			"read.item": "queue",
		},
	}
}

func (fakePolicyFactory) ValidateConfig(PolicyRuntimeConfig) error {
	return nil
}

func (fakePolicyFactory) Build(context.Context, PolicyRuntimeConfig) (core.Engine, error) {
	return fakeEngine{}, nil
}

func (fakePolicyFactory) Metadata() PolicyMetadata {
	return PolicyMetadata{
		Name:           "test-policy",
		DSLVersion:     "v3",
		GeneratedWith:  "cervorules-policygen",
		VocabularyHash: "sha256:test",
		PolicyHash:     "sha256:policy",
	}
}

type fakeEngine struct{}

func (fakeEngine) Decide(ctx context.Context, req core.Request) (core.DecisionResult, error) {
	return fakeEngine{}.DecideWithOptions(ctx, req, core.DecisionOptions{})
}

func (fakeEngine) DecideWithOptions(_ context.Context, req core.Request, _ core.DecisionOptions) (core.DecisionResult, error) {
	return core.NewDecisionResult(req, core.Decision{Allow: true, Target: "queue", Executor: "primary"}), nil
}
