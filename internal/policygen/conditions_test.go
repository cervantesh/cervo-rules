package policygen

import (
	"strings"
	"testing"
)

const conditionVocab = `
operations:
  refund:
    description: refund an order
targets:
  ledger:
    description: ledger service
executors:
  primary:
    description: primary executor
`

const conditionPolicy = `
version: cervorules.policy.v3
name: order_guards
defaults:
  executor: primary
conditions:
  refund_transition_legal:
    kind: transition_allowed
    lifecycle: order
    subject_key: order_id
    to: refunded
    description: the order may still move to refunded
routes:
  - id: refund_route
    operation: refund
    target: ledger
    executor: primary
    requires: [refund_transition_legal]
`

// compileGeneratedPolicy always writes a test file, so a package clause is the
// minimum that still compiles when a case has nothing extra to assert.
const minimalTestSource = "package policyrules\n"

func generateConditionPolicy(t *testing.T, policy string) (Output, error) {
	t.Helper()
	return Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(conditionVocab),
		PolicyReader:      strings.NewReader(policy),
	})
}

func TestConditionGatedRouteCompilesAndCarriesRequirement(t *testing.T) {
	out, err := generateConditionPolicy(t, conditionPolicy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// gofmt aligns struct fields, so compare with whitespace collapsed.
	flat := strings.Join(strings.Fields(out.Source), " ")
	required := []string{
		`requires: []cervorules.Condition{cervorules.Condition("refund_transition_legal")}`,
		"conditions cervorules.Conditions",
		"func (e generatedEngine) conditionsHold(",
		"conditions: cfg.Conditions",
	}
	for _, want := range required {
		if !strings.Contains(flat, strings.Join(strings.Fields(want), " ")) {
			t.Fatalf("generated source missing %q:\n%s", want, out.Source)
		}
	}
	if out.Snapshot.Conditions != 1 {
		t.Fatalf("snapshot should record 1 condition, got %d", out.Snapshot.Conditions)
	}
	compileGeneratedPolicy(t, out.Source, minimalTestSource)
}

// A policy that declares guards but is built without an evaluator would allow
// exactly what the guards exist to stop, so ValidateConfig must refuse it.
func TestGeneratedPolicyRefusesMissingConditionEvaluator(t *testing.T) {
	out, err := generateConditionPolicy(t, conditionPolicy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(out.Source, "cervorules.ErrorCodeMissingConditions") {
		t.Fatalf("ValidateConfig must reject a nil evaluator:\n%s", out.Source)
	}
	compileGeneratedPolicy(t, out.Source, `package policyrules

import (
	"context"
	"strings"
	"testing"

	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
)

func TestBuildRefusesWithoutConditions(t *testing.T) {
	_, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err == nil {
		t.Fatal("a policy declaring conditions must not build without an evaluator")
	}
	if !strings.Contains(err.Error(), "missing_conditions") {
		t.Fatalf("unexpected error: %v", err)
	}
}
`)
}

// The whole point of validating at generation time: an unwired guard fails the
// build instead of failing a decision in production.
func TestUndeclaredConditionFailsGeneration(t *testing.T) {
	policy := strings.Replace(conditionPolicy, "requires: [refund_transition_legal]", "requires: [never_declared]", 1)
	if _, err := generateConditionPolicy(t, policy); err == nil {
		t.Fatal("requiring an undeclared condition must fail generation")
	} else if !strings.Contains(err.Error(), "undeclared condition") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConditionKindsAreValidated(t *testing.T) {
	cases := map[string]string{
		"unknown kind": `
conditions:
  bad:
    kind: telepathy
`,
		"transition without lifecycle": `
conditions:
  bad:
    kind: transition_allowed
    subject_key: order_id
    to: refunded
`,
		"in_state without states": `
conditions:
  bad:
    kind: in_state
    lifecycle: order
    subject_key: order_id
`,
		"has_type without entity_type": `
conditions:
  bad:
    kind: has_type
    subject_key: order_id
`,
		"missing kind": `
conditions:
  bad:
    lifecycle: order
`,
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			policy := "version: cervorules.policy.v3\nname: broken\n" + block
			if _, err := generateConditionPolicy(t, policy); err == nil {
				t.Fatal("expected generation to fail")
			}
		})
	}
}

// integrity asks about the whole snapshot, so it legitimately takes no subject.
func TestIntegrityConditionNeedsNoSubject(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: integrity_only
defaults:
  executor: primary
conditions:
  world_is_coherent:
    kind: integrity
denies:
  - id: block_incoherent
    operation: refund
    reason: the recorded world is inconsistent
    requires: [world_is_coherent]
`
	out, err := generateConditionPolicy(t, policy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	flat := strings.Join(strings.Fields(out.Source), " ")
	if !strings.Contains(flat, `requires: []cervorules.Condition{cervorules.Condition("world_is_coherent")}`) {
		t.Fatalf("deny should carry its requirement:\n%s", out.Source)
	}
	compileGeneratedPolicy(t, out.Source, minimalTestSource)
}

func TestDuplicateRequirementFailsGeneration(t *testing.T) {
	policy := strings.Replace(conditionPolicy,
		"requires: [refund_transition_legal]",
		"requires: [refund_transition_legal, refund_transition_legal]", 1)
	if _, err := generateConditionPolicy(t, policy); err == nil {
		t.Fatal("a repeated requirement must fail generation")
	}
}

// A policy with no conditions must generate exactly as before.
func TestPolicyWithoutConditionsStaysUnchanged(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: order_guards
defaults:
  executor: primary
routes:
  - id: refund_route
    operation: refund
    target: ledger
    executor: primary
`
	out, err := generateConditionPolicy(t, policy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// The conditionsHold helper is always emitted and is inert without
	// requirements. What must not appear is ValidateConfig demanding an
	// evaluator from a policy that declares no guards.
	if strings.Contains(out.Source, "if cfg.Conditions == nil") {
		t.Fatalf("a policy without conditions must not require an evaluator:\n%s", out.Source)
	}
	if out.Snapshot.Conditions != 0 {
		t.Fatalf("expected 0 conditions, got %d", out.Snapshot.Conditions)
	}
	compileGeneratedPolicy(t, out.Source, minimalTestSource)
}
