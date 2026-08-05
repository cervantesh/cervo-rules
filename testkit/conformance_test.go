package testkit_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
	"github.com/cervantesh/cervo-rules/v3/facts"
	"github.com/cervantesh/cervo-rules/v3/runtime"
	"github.com/cervantesh/cervo-rules/v3/testkit"
)

func TestConsumerConformanceSuiteCertifiesRepoLocalConsumers(t *testing.T) {
	for _, contract := range syntheticConsumerConformanceContracts() {
		t.Run(contract.Name, func(t *testing.T) {
			testkit.MustAssertConsumerConformance(t, contract)
			if err := testkit.CheckConsumerConformance(contract); err != nil {
				t.Fatalf("expected conformance to pass: %v", err)
			}
		})
	}
}

func TestCheckConsumerConformanceReportsClearErrors(t *testing.T) {
	tests := []struct {
		name     string
		contract testkit.ConsumerConformanceContract
		want     string
	}{
		{name: "missing name", want: "requires Name"},
		{
			name: "missing policy",
			contract: testkit.ConsumerConformanceContract{
				Name:           "broken",
				PolicyPath:     "testdata/conformance/missing/policy-rules.yaml",
				VocabularyPath: fixturePath("billing-routing", "policy-vocabulary.yaml"),
			},
			want: "policy path",
		},
		{
			name: "missing vocabulary",
			contract: testkit.ConsumerConformanceContract{
				Name:           "broken",
				PolicyPath:     fixturePath("billing-routing", "policy-rules.yaml"),
				VocabularyPath: "testdata/conformance/missing/policy-vocabulary.yaml",
			},
			want: "vocabulary path",
		},
		{
			name: "empty coverage",
			contract: testkit.ConsumerConformanceContract{
				Name:           "empty",
				PolicyPath:     fixturePath("billing-routing", "policy-rules.yaml"),
				VocabularyPath: fixturePath("billing-routing", "policy-vocabulary.yaml"),
			},
			want: "requires runtime, facts, or package smoke coverage",
		},
		{
			name:     "nameless runtime",
			contract: baseContract("runtime-broken", []testkit.RuntimeCase{{}}, nil),
			want:     "runtime case requires Name",
		},
		{
			name: "nil runtime factory",
			contract: baseContract("runtime-broken", []testkit.RuntimeCase{{
				Name: "nil-factory",
			}}, nil),
			want: "requires Factory",
		},
		{
			name: "missing factory metadata",
			contract: baseContract("runtime-broken", []testkit.RuntimeCase{{
				Name:    "metadata",
				Factory: badFactory{noMetadata: true},
			}}, nil),
			want: "requires factory metadata",
		},
		{
			name: "invalid runtime config",
			contract: baseContract("runtime-broken", []testkit.RuntimeCase{{
				Name:    "invalid-config",
				Factory: badFactory{validateErr: fmt.Errorf("config rejected")},
			}}, nil),
			want: "validate config",
		},
		{
			name: "runtime build error",
			contract: baseContract("runtime-broken", []testkit.RuntimeCase{{
				Name:    "build",
				Factory: badFactory{buildErr: fmt.Errorf("build failed")},
			}}, nil),
			want: "build",
		},
		{
			name: "runtime decide error",
			contract: baseContract("runtime-broken", []testkit.RuntimeCase{{
				Name:    "decide",
				Factory: badFactory{engine: errorEngine{}},
			}}, nil),
			want: "decide",
		},
		{
			name: "runtime mismatch",
			contract: baseContract("runtime-mismatch", []testkit.RuntimeCase{{
				Name:         "decision",
				Factory:      routeFactory("runtime-mismatch", "read", "reader", "primary"),
				Config:       runtime.PolicyRuntimeConfig{DefaultExecutor: "primary"},
				Request:      core.Request{Operation: "read"},
				WantDecision: core.Decision{Allow: false},
			}}, nil),
			want: "decision mismatch",
		},
		{
			name:     "nameless facts",
			contract: baseContract("facts-broken", nil, []testkit.FactsCase{{}}),
			want:     "facts case requires Name",
		},
		{
			name: "facts missing budget",
			contract: baseContract("facts-budget", nil, []testkit.FactsCase{{
				Name:      "missing-budget",
				Evaluator: staticFactsEvaluator{},
			}}),
			want: "positive MaxIterations and MaxFacts",
		},
		{
			name: "facts evaluate error",
			contract: baseContract("facts-error", nil, []testkit.FactsCase{{
				Name:      "evaluate",
				Evaluator: errorFactsEvaluator{},
				Options:   facts.EvalOptions{MaxIterations: 1, MaxFacts: 1},
			}}),
			want: "evaluate",
		},
		{
			name: "facts missing expected fact",
			contract: baseContract("facts-missing", nil, []testkit.FactsCase{{
				Name:        "missing-fact",
				Evaluator:   staticFactsEvaluator{},
				Options:     facts.EvalOptions{MaxIterations: 1, MaxFacts: 1},
				ExpectFacts: []facts.Fact{constFact("expected")},
			}}),
			want: "missing fact",
		},
		{
			name: "package smoke missing module",
			contract: testkit.ConsumerConformanceContract{
				Name:           "package-smoke",
				PolicyPath:     fixturePath("billing-routing", "policy-rules.yaml"),
				VocabularyPath: fixturePath("billing-routing", "policy-vocabulary.yaml"),
				PackageSmoke: testkit.PackageSmoke{
					Name:             "missing-module",
					RequiredPackages: []string{"core"},
					RequiredSchemas:  []string{"schemas/v3/policy-rules.schema.json"},
				},
			},
			want: "requires ModulePath",
		},
		{
			name: "package smoke missing packages",
			contract: testkit.ConsumerConformanceContract{
				Name:           "package-smoke",
				PolicyPath:     fixturePath("billing-routing", "policy-rules.yaml"),
				VocabularyPath: fixturePath("billing-routing", "policy-vocabulary.yaml"),
				PackageSmoke: testkit.PackageSmoke{
					Name:            "missing-packages",
					ModulePath:      "github.com/cervantesh/cervo-rules/v3",
					RequiredSchemas: []string{"schemas/v3/policy-rules.schema.json"},
				},
			},
			want: "requires packages",
		},
		{
			name: "package smoke missing schemas",
			contract: testkit.ConsumerConformanceContract{
				Name:           "package-smoke",
				PolicyPath:     fixturePath("billing-routing", "policy-rules.yaml"),
				VocabularyPath: fixturePath("billing-routing", "policy-vocabulary.yaml"),
				PackageSmoke: testkit.PackageSmoke{
					Name:             "missing-schemas",
					ModulePath:       "github.com/cervantesh/cervo-rules/v3",
					RequiredPackages: []string{"core"},
				},
			},
			want: "requires schemas",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckConsumerConformance(tt.contract)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func syntheticConsumerConformanceContracts() []testkit.ConsumerConformanceContract {
	return []testkit.ConsumerConformanceContract{
		consumerContract("billing-routing", "invoice.read", "billing-ledger", "ledger-primary"),
		consumerContract("document-processing", "document.process", "document-worker", "worker-primary"),
		consumerContract("device-routing", "device.command", "device-controller", "controller-primary"),
		consumerContract("queue-event-routing", "event.consume", "queue-worker", "queue-primary"),
		consumerContract("scheduled-job-routing", "job.run", "scheduler", "scheduler-primary"),
		consumerContract("cli-command-routing", "command.execute", "command-runner", "runner-primary"),
		proxyShapedContract(),
	}
}

func consumerContract(name string, operation core.Operation, target core.Target, executor core.Executor) testkit.ConsumerConformanceContract {
	factory := routeFactory(name, operation, target, executor)
	return testkit.ConsumerConformanceContract{
		Name:           name,
		PolicyPath:     fixturePath(name, "policy-rules.yaml"),
		VocabularyPath: fixturePath(name, "policy-vocabulary.yaml"),
		RuntimeCases: []testkit.RuntimeCase{{
			Name:         "factory-decision",
			Factory:      factory,
			Config:       factory.DefaultConfig(),
			Request:      core.Request{Operation: operation},
			WantDecision: core.Decision{Allow: true, Target: target, Executor: executor},
		}},
		FactsCases: []testkit.FactsCase{{
			Name:      "facts-fixture",
			Evaluator: staticFactsEvaluator{facts: []facts.Fact{constFact("certified_consumer", name)}},
			Options:   facts.EvalOptions{MaxIterations: 2, MaxFacts: 8},
			ExpectFacts: []facts.Fact{
				constFact("certified_consumer", name),
			},
		}},
		PackageSmoke: testkit.PackageSmoke{
			Name:             "v3-package-smoke",
			ModulePath:       "github.com/cervantesh/cervo-rules/v3",
			RequiredPackages: []string{"core", "runtime", "facts", "testkit"},
			RequiredSchemas:  []string{"schemas/v3/policy-rules.schema.json", "schemas/v3/policy-vocabulary.schema.json"},
		},
	}
}

func proxyShapedContract() testkit.ConsumerConformanceContract {
	name := "edge-request-routing"
	operation := core.Operation("request.route")
	target := core.Target("request-router")
	executor := core.Executor("primary-route")
	contract := consumerContract(name, operation, target, executor)
	contract.RuntimeCases[0].Request.Metadata = map[string]string{
		"profile":         "interactive",
		"requested_model": "neutral-model",
	}
	return contract
}

func baseContract(name string, runtimeCases []testkit.RuntimeCase, factsCases []testkit.FactsCase) testkit.ConsumerConformanceContract {
	return testkit.ConsumerConformanceContract{
		Name:           name,
		PolicyPath:     fixturePath("billing-routing", "policy-rules.yaml"),
		VocabularyPath: fixturePath("billing-routing", "policy-vocabulary.yaml"),
		RuntimeCases:   runtimeCases,
		FactsCases:     factsCases,
	}
}

func routeFactory(name string, operation core.Operation, target core.Target, executor core.Executor) fakeFactory {
	return fakeFactory{name: name, operation: operation, target: target, executor: executor}
}

type fakeFactory struct {
	name      string
	operation core.Operation
	target    core.Target
	executor  core.Executor
}

func (f fakeFactory) DefaultConfig() runtime.PolicyRuntimeConfig {
	return runtime.PolicyRuntimeConfig{
		DefaultExecutor: f.executor,
		OperationTargets: map[core.Operation]core.Target{
			f.operation: f.target,
		},
	}
}

func (f fakeFactory) ValidateConfig(cfg runtime.PolicyRuntimeConfig) error {
	if cfg.DefaultExecutor == "" {
		return fmt.Errorf("default executor is required")
	}
	return nil
}

func (f fakeFactory) Build(context.Context, runtime.PolicyRuntimeConfig) (core.Engine, error) {
	return fakeEngine{operation: f.operation, target: f.target, executor: f.executor}, nil
}

func (f fakeFactory) Metadata() runtime.PolicyMetadata {
	return runtime.PolicyMetadata{
		Name:           f.name,
		DSLVersion:     "v3",
		GeneratedWith:  "testkit",
		VocabularyHash: "test-vocab",
		PolicyHash:     "test-policy",
	}
}

type fakeEngine struct {
	operation core.Operation
	target    core.Target
	executor  core.Executor
}

func (e fakeEngine) Decide(ctx context.Context, req core.Request) (core.DecisionResult, error) {
	return e.DecideWithOptions(ctx, req, core.DecisionOptions{})
}

func (e fakeEngine) DecideWithOptions(ctx context.Context, req core.Request, _ core.DecisionOptions) (core.DecisionResult, error) {
	if err := ctx.Err(); err != nil {
		return core.DecisionResult{}, err
	}
	if req.Operation != e.operation {
		return core.DecisionResult{Decision: core.Decision{Allow: false, Reason: "operation not routed"}}, nil
	}
	return core.DecisionResult{Decision: core.Decision{Allow: true, Target: e.target, Executor: e.executor}}, nil
}

type staticFactsEvaluator struct {
	facts []facts.Fact
}

func (e staticFactsEvaluator) Evaluate(_ context.Context, _ facts.EvalOptions) (facts.Result, error) {
	return facts.Result{Facts: append([]facts.Fact(nil), e.facts...)}, nil
}

type errorFactsEvaluator struct{}

func (errorFactsEvaluator) Evaluate(context.Context, facts.EvalOptions) (facts.Result, error) {
	return facts.Result{}, fmt.Errorf("facts failed")
}

type badFactory struct {
	metadata    runtime.PolicyMetadata
	noMetadata  bool
	validateErr error
	buildErr    error
	engine      core.Engine
}

func (f badFactory) DefaultConfig() runtime.PolicyRuntimeConfig {
	return runtime.PolicyRuntimeConfig{DefaultExecutor: "bad"}
}

func (f badFactory) ValidateConfig(runtime.PolicyRuntimeConfig) error {
	return f.validateErr
}

func (f badFactory) Build(context.Context, runtime.PolicyRuntimeConfig) (core.Engine, error) {
	if f.buildErr != nil {
		return nil, f.buildErr
	}
	if f.engine != nil {
		return f.engine, nil
	}
	return fakeEngine{operation: "bad", target: "bad", executor: "bad"}, nil
}

func (f badFactory) Metadata() runtime.PolicyMetadata {
	if f.noMetadata {
		return runtime.PolicyMetadata{}
	}
	if f.metadata.Name != "" || f.metadata.DSLVersion != "" {
		return f.metadata
	}
	return runtime.PolicyMetadata{Name: "bad", DSLVersion: "v3"}
}

type errorEngine struct{}

func (errorEngine) Decide(context.Context, core.Request) (core.DecisionResult, error) {
	return core.DecisionResult{}, fmt.Errorf("decide failed")
}

func (errorEngine) DecideWithOptions(context.Context, core.Request, core.DecisionOptions) (core.DecisionResult, error) {
	return core.DecisionResult{}, fmt.Errorf("decide failed")
}

func constFact(predicate facts.Atom, values ...string) facts.Fact {
	terms := make([]facts.Term, 0, len(values))
	for _, value := range values {
		terms = append(terms, facts.Term{Kind: facts.TermConst, Value: value})
	}
	return facts.Fact{Predicate: predicate, Terms: terms}
}

func fixturePath(consumer string, file string) string {
	return filepath.Join("testdata", "conformance", consumer, file)
}
