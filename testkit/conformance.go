package testkit

import (
	"context"
	"errors"
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
//
// A case either expects a decision or expects the decision to fail. Failing is
// a first-class outcome, not an accident: a fact that is absent, unparseable,
// non-finite or outside its declared bounds refuses the decision rather than
// reporting a predicate as unsatisfied, and a consumer certifying against this
// contract has to be able to cover that.
type RuntimeCase struct {
	Name         string
	Factory      runtime.PolicyFactory
	Config       runtime.PolicyRuntimeConfig
	Request      core.Request
	Options      core.DecisionOptions
	WantDecision core.Decision

	// WantErrorCode expects the decision to fail with this structured code
	// instead of returning WantDecision. Typical values are
	// core.ErrorCodeMissingFact, core.ErrorCodeInvalidFact and
	// core.ErrorCodeUnknownCondition.
	WantErrorCode core.ErrorCode

	// WantTraceStepNames expects these rule identifiers, in order, in the
	// decision trace. Options must enable trace for this to be checked.
	WantTraceStepNames []string
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
	if runtimeCase.WantErrorCode != "" {
		return checkExpectedDecisionError(contractName, name, runtimeCase.WantErrorCode, err)
	}
	if err != nil {
		return fmt.Errorf("%s: runtime case %s decide: %w", contractName, name, err)
	}
	if !reflect.DeepEqual(result.Decision, runtimeCase.WantDecision) {
		return fmt.Errorf("%s: runtime case %s decision mismatch: got %+v want %+v", contractName, name, result.Decision, runtimeCase.WantDecision)
	}
	return checkTraceStepNames(contractName, name, runtimeCase.WantTraceStepNames, result)
}

// checkExpectedDecisionError requires a structured refusal. An unstructured
// error would satisfy "it failed" while telling a consumer nothing about why,
// which is the difference this contract exists to hold.
func checkExpectedDecisionError(contractName, name string, want core.ErrorCode, err error) error {
	if err == nil {
		return fmt.Errorf("%s: runtime case %s expected the decision to fail with %s", contractName, name, want)
	}
	var single core.Error
	if errors.As(err, &single) {
		if single.Code != want {
			return fmt.Errorf("%s: runtime case %s expected %s, got %s", contractName, name, want, single.Code)
		}
		return nil
	}
	var many core.Errors
	if errors.As(err, &many) {
		if !many.Has(want) {
			return fmt.Errorf("%s: runtime case %s expected %s, got %v", contractName, name, want, many)
		}
		return nil
	}
	return fmt.Errorf("%s: runtime case %s expected structured error %s, got %T: %v", contractName, name, want, err, err)
}

func checkTraceStepNames(contractName, name string, want []string, result core.DecisionResult) error {
	if len(want) == 0 {
		return nil
	}
	if result.Trace == nil {
		return fmt.Errorf("%s: runtime case %s expects trace steps but the case did not enable trace", contractName, name)
	}
	if len(result.Trace.Steps) != len(want) {
		return fmt.Errorf("%s: runtime case %s expected %d trace steps, got %d", contractName, name, len(want), len(result.Trace.Steps))
	}
	for i, expected := range want {
		if result.Trace.Steps[i].Name != expected {
			return fmt.Errorf("%s: runtime case %s trace step %d: got %q want %q", contractName, name, i, result.Trace.Steps[i].Name, expected)
		}
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
