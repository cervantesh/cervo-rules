package testkit

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
	"github.com/cervantesh/cervo-rules/v3/facts"
	"github.com/cervantesh/cervo-rules/v3/runtime"
)

// ConsumerConformanceContract describes repo-local checks a v3 consumer fixture
// must satisfy before it can be treated as CervoRules-compatible.
type ConsumerConformanceContract struct {
	Name           string
	PolicyPath     string
	VocabularyPath string
	RuntimeCases   []RuntimeCase
	FactsCases     []FactsCase
	PackageSmoke   PackageSmoke
}

// RuntimeCase verifies that a generated policy factory can validate config,
// build an engine, and return the expected decision.
type RuntimeCase struct {
	Name         string
	Factory      runtime.PolicyFactory
	Config       runtime.PolicyRuntimeConfig
	Request      core.Request
	Options      core.DecisionOptions
	WantDecision core.Decision
}

// FactsEvaluator is the minimal v3 facts execution shape used by conformance
// fixtures. Real consumers can adapt their facts engine without importing
// testing.TB.
type FactsEvaluator interface {
	Evaluate(context.Context, facts.EvalOptions) (facts.Result, error)
}

// FactsCase verifies that a consumer facts fixture is bounded and produces
// expected fact output.
type FactsCase struct {
	Name        string
	Evaluator   FactsEvaluator
	Options     facts.EvalOptions
	ExpectFacts []facts.Fact
}

// PackageSmoke verifies that release/package fixtures mention the machine
// artifacts a consumer needs.
type PackageSmoke struct {
	Name             string
	ModulePath       string
	RequiredPackages []string
	RequiredSchemas  []string
}

// CheckConsumerConformance verifies a consumer contract without depending on
// testing.TB. It is suitable for CLIs, release smoke tests, and agent workflows.
func CheckConsumerConformance(contract ConsumerConformanceContract) error {
	return checkConsumerConformance(contract)
}

// MustAssertConsumerConformance verifies a consumer contract and fails at the
// test boundary.
func MustAssertConsumerConformance(t testing.TB, contract ConsumerConformanceContract) {
	t.Helper()
	if err := checkConsumerConformance(contract); err != nil {
		t.Fatal(err)
	}
}

func checkConsumerConformance(contract ConsumerConformanceContract) error {
	name := strings.TrimSpace(contract.Name)
	if name == "" {
		return fmt.Errorf("consumer conformance contract requires Name")
	}
	if err := requireFixtureFile(contract.PolicyPath); err != nil {
		return fmt.Errorf("%s: policy path: %w", name, err)
	}
	if err := requireFixtureFile(contract.VocabularyPath); err != nil {
		return fmt.Errorf("%s: vocabulary path: %w", name, err)
	}
	if len(contract.RuntimeCases) == 0 && len(contract.FactsCases) == 0 && contract.PackageSmoke.Name == "" {
		return fmt.Errorf("%s: requires runtime, facts, or package smoke coverage", name)
	}
	for _, runtimeCase := range contract.RuntimeCases {
		if err := checkRuntimeCase(name, runtimeCase); err != nil {
			return err
		}
	}
	for _, factsCase := range contract.FactsCases {
		if err := checkFactsCase(name, factsCase); err != nil {
			return err
		}
	}
	if contract.PackageSmoke.Name != "" {
		if err := checkPackageSmoke(name, contract.PackageSmoke); err != nil {
			return err
		}
	}
	return nil
}

func requireFixtureFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory", path)
	}
	if info.Size() == 0 {
		return fmt.Errorf("%q is empty", path)
	}
	return nil
}

func checkRuntimeCase(contractName string, runtimeCase RuntimeCase) error {
	name := strings.TrimSpace(runtimeCase.Name)
	if name == "" {
		return fmt.Errorf("%s: runtime case requires Name", contractName)
	}
	if runtimeCase.Factory == nil {
		return fmt.Errorf("%s: runtime case %s requires Factory", contractName, name)
	}
	metadata := runtimeCase.Factory.Metadata()
	if strings.TrimSpace(metadata.Name) == "" || strings.TrimSpace(metadata.DSLVersion) == "" {
		return fmt.Errorf("%s: runtime case %s requires factory metadata name and DSL version", contractName, name)
	}
	cfg := runtimeCase.Config.Clone()
	if err := runtimeCase.Factory.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("%s: runtime case %s validate config: %w", contractName, name, err)
	}
	engine, err := runtimeCase.Factory.Build(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("%s: runtime case %s build: %w", contractName, name, err)
	}
	result, err := engine.DecideWithOptions(context.Background(), runtimeCase.Request, runtimeCase.Options)
	if err != nil {
		return fmt.Errorf("%s: runtime case %s decide: %w", contractName, name, err)
	}
	if !reflect.DeepEqual(result.Decision, runtimeCase.WantDecision) {
		return fmt.Errorf("%s: runtime case %s decision mismatch: got %+v want %+v", contractName, name, result.Decision, runtimeCase.WantDecision)
	}
	return nil
}

func checkFactsCase(contractName string, factsCase FactsCase) error {
	name := strings.TrimSpace(factsCase.Name)
	if name == "" {
		return fmt.Errorf("%s: facts case requires Name", contractName)
	}
	if factsCase.Evaluator == nil {
		return fmt.Errorf("%s: facts case %s requires Evaluator", contractName, name)
	}
	options := factsCase.Options.Normalize()
	if options.MaxIterations <= 0 || options.MaxFacts <= 0 {
		return fmt.Errorf("%s: facts case %s requires positive MaxIterations and MaxFacts", contractName, name)
	}
	result, err := factsCase.Evaluator.Evaluate(context.Background(), options)
	if err != nil {
		return fmt.Errorf("%s: facts case %s evaluate: %w", contractName, name, err)
	}
	for _, expected := range factsCase.ExpectFacts {
		if !hasFact(result.Facts, expected) {
			return fmt.Errorf("%s: facts case %s missing fact %s", contractName, name, expected.Predicate)
		}
	}
	return nil
}

func hasFact(candidates []facts.Fact, expected facts.Fact) bool {
	for _, candidate := range candidates {
		if reflect.DeepEqual(candidate, expected) {
			return true
		}
	}
	return false
}

func checkPackageSmoke(contractName string, smoke PackageSmoke) error {
	name := strings.TrimSpace(smoke.Name)
	if name == "" {
		return fmt.Errorf("%s: package smoke requires Name", contractName)
	}
	if strings.TrimSpace(smoke.ModulePath) == "" {
		return fmt.Errorf("%s: package smoke %s requires ModulePath", contractName, name)
	}
	if len(smoke.RequiredPackages) == 0 {
		return fmt.Errorf("%s: package smoke %s requires packages", contractName, name)
	}
	if len(smoke.RequiredSchemas) == 0 {
		return fmt.Errorf("%s: package smoke %s requires schemas", contractName, name)
	}
	return nil
}
