package policygen

import (
	"strings"
	"testing"
)

const predicateVocab = `
operations:
  trade.place: {}
  trade.close: {}
targets:
  desk: {}
executors:
  manual: {}
facts:
  account_mode: { type: enum, values: [demo, live] }
  risk_pct: { type: number }
  exposure_pct: { type: number }
  score: { type: number }
  min_score: { type: number }
  operable: { type: bool }
  orders_last_hour: { type: integer }
  style: { type: string }
`

const predicateVocabGo = `package policyvocab

import "github.com/cervantesh/cervo-rules/v3/core"

const (
	OperationTradePlace core.Operation = "trade.place"
	OperationTradeClose core.Operation = "trade.close"
	TargetDesk          core.Target    = "desk"
	ExecutorManual      core.Executor  = "manual"
)

func Vocabulary() core.Vocabulary {
	return core.NewVocabulary(
		core.AllowedOperations(OperationTradePlace, OperationTradeClose),
		core.AllowedTargets(TargetDesk),
		core.AllowedExecutors(ExecutorManual),
	)
}
`

// predicatePolicy is the shape the gold-executor acceptance case needs: ordered
// denies that apply to every operation, two of them disjunctions, and every
// threshold inside the policy file.
const predicatePolicy = `
version: cervorules.policy.v3
name: trading.v3
defaults:
  executor: manual
facts:
  risk_pct: { min: 0 }
  exposure_pct: { min: 0 }
  score: { min: 0 }
  min_score: { min: 0, default: 50 }
  orders_last_hour: { min: 0 }
denies:
  - id: deny-non-demo
    when: { fact: account_mode, op: ne, value: demo }
  - id: deny-risk-or-exposure
    reason: risk or exposure over limit
    when:
      any:
        - { fact: risk_pct, op: gt, value: 1.5 }
        - { fact: exposure_pct, op: gt, value: 5.0 }
  - id: deny-not-operable-or-score
    when:
      any:
        - { fact: operable, op: is_false }
        - { fact: score, op: lt, fact_value: min_score }
  - id: deny-rate-limit
    when: { fact: orders_last_hour, op: gte, value: 6 }
routes:
  - id: allow-trade
    operation: trade.place
    target: desk
    executor: manual
  - id: allow-close
    operation: trade.close
    target: desk
    executor: manual
`

func generatePredicatePolicy(t *testing.T, vocab string, policy string) (Output, error) {
	t.Helper()
	return Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(vocab),
		PolicyReader:      strings.NewReader(policy),
	})
}

// The predicate compiles to a Go boolean expression: no evaluator, no
// expression engine, nothing to sandbox.
func TestCompoundPredicateCompilesToBooleanExpressions(t *testing.T) {
	out, err := generatePredicatePolicy(t, predicateVocab, predicatePolicy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	flat := strings.Join(strings.Fields(out.Source), " ")
	for _, want := range []string{
		`if f.factRiskPct > 1.5 { return true, "risk_pct gt 1.5", nil }`,
		`if f.factExposurePct > 5 { return true, "exposure_pct gt 5", nil }`,
		`if !f.factOperable { return true, "operable is_false", nil }`,
		`if f.factScore < f.factMinScore { return true, "score lt min_score", nil }`,
		`if f.factOrdersLastHour >= 6 { return true, "orders_last_hour gte 6", nil }`,
		`if f.factAccountMode != "demo" { return true, "account_mode ne demo", nil }`,
	} {
		if !strings.Contains(flat, strings.Join(strings.Fields(want), " ")) {
			t.Fatalf("generated source missing %q:\n%s", want, out.Source)
		}
	}
	// A fact no rule reads must not enter the frame: a policy pays only for the
	// facts it consults.
	if strings.Contains(out.Source, "factStyle") {
		t.Fatalf("unreferenced fact entered the frame:\n%s", out.Source)
	}
	if out.Snapshot.Facts != 8 {
		t.Fatalf("snapshot should record 8 declared facts, got %d", out.Snapshot.Facts)
	}
}

func TestCompoundPredicateDecidesAndFailsClosed(t *testing.T) {
	out, err := generatePredicatePolicy(t, predicateVocab, predicatePolicy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	compileGeneratedPolicyWithVocab(t, predicateVocabGo, out.Source, `package policyrules

import (
	"context"
	"errors"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
	"example.test/generated/policyvocab"
)

func sane() map[string]string {
	return map[string]string{
		"account_mode":     "demo",
		"risk_pct":         "1.0",
		"exposure_pct":     "2.0",
		"score":            "72",
		"min_score":        "50",
		"operable":         "true",
		"orders_last_hour": "1",
	}
}

func decide(t *testing.T, facts map[string]string) cervorules.DecisionResult {
	t.Helper()
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	result, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationTradePlace, Metadata: facts})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	return result
}

func TestSaneFactsAllow(t *testing.T) {
	result := decide(t, sane())
	if !result.Decision.Allow || result.Decision.Executor != policyvocab.ExecutorManual {
		t.Fatalf("expected allow, got %#v", result.Decision)
	}
}

func TestEachDisjunctDeniesOnItsOwn(t *testing.T) {
	for name, mutation := range map[string]map[string]string{
		"risk":     {"risk_pct": "3.0"},
		"exposure": {"exposure_pct": "6.0"},
	} {
		t.Run(name, func(t *testing.T) {
			facts := sane()
			for key, value := range mutation {
				facts[key] = value
			}
			result := decide(t, facts)
			if result.Decision.Allow || result.Decision.Reason != "risk or exposure over limit" {
				t.Fatalf("expected the disjunction to deny, got %#v", result.Decision)
			}
		})
	}
}

// A deny that authored no reason reports its id, so an audit record is never
// left with an empty reason.
func TestDenyReasonFallsBackToID(t *testing.T) {
	facts := sane()
	facts["account_mode"] = "live"
	result := decide(t, facts)
	if result.Decision.Allow || result.Decision.Reason != "deny-non-demo" {
		t.Fatalf("expected the rule id as reason, got %#v", result.Decision)
	}
}

func TestOrderedDeniesReportTheFirstMatch(t *testing.T) {
	facts := sane()
	facts["account_mode"] = "live"
	facts["risk_pct"] = "9.0"
	result := decide(t, facts)
	if result.Decision.Reason != "deny-non-demo" {
		t.Fatalf("first authored match must win, got %#v", result.Decision)
	}
}

// The boundaries the thresholds actually encode: gt allows the limit itself,
// gte denies it.
func TestThresholdBoundaries(t *testing.T) {
	facts := sane()
	facts["risk_pct"] = "1.5"
	if result := decide(t, facts); !result.Decision.Allow {
		t.Fatalf("risk_pct at the limit must be allowed, got %#v", result.Decision)
	}
	facts = sane()
	facts["orders_last_hour"] = "6"
	if result := decide(t, facts); result.Decision.Allow {
		t.Fatalf("orders_last_hour at the limit must be denied, got %#v", result.Decision)
	}
	facts = sane()
	facts["orders_last_hour"] = "5"
	if result := decide(t, facts); !result.Decision.Allow {
		t.Fatalf("orders_last_hour below the limit must be allowed, got %#v", result.Decision)
	}
}

// A non-finite value passes every comparison in the policy without matching any
// of them. It must fail the decision, not reach the allow path.
func TestNonFiniteFactsFailClosed(t *testing.T) {
	for _, raw := range []string{"NaN", "Inf", "+Inf", "-Inf"} {
		facts := sane()
		facts["risk_pct"] = raw
		engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		_, err = engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationTradePlace, Metadata: facts})
		if err == nil {
			t.Fatalf("risk_pct=%q reached a decision", raw)
		}
		var errs cervorules.Error
		if !errors.As(err, &errs) || errs.Code != cervorules.ErrorCodeInvalidFact {
			t.Fatalf("risk_pct=%q: expected invalid_fact, got %v", raw, err)
		}
	}
}

func TestUnusableFactsFailClosed(t *testing.T) {
	cases := map[string]struct {
		mutate func(map[string]string)
		code   cervorules.ErrorCode
	}{
		"negative below declared minimum": {func(f map[string]string) { f["risk_pct"] = "-1" }, cervorules.ErrorCodeInvalidFact},
		"not a number":                    {func(f map[string]string) { f["risk_pct"] = "quite high" }, cervorules.ErrorCodeInvalidFact},
		"not an integer":                  {func(f map[string]string) { f["orders_last_hour"] = "1.5" }, cervorules.ErrorCodeInvalidFact},
		"not a bool":                      {func(f map[string]string) { f["operable"] = "maybe" }, cervorules.ErrorCodeInvalidFact},
		"outside the enum domain":         {func(f map[string]string) { f["account_mode"] = "paper" }, cervorules.ErrorCodeInvalidFact},
		"absent with no default":          {func(f map[string]string) { delete(f, "risk_pct") }, cervorules.ErrorCodeMissingFact},
		"blank with no default":           {func(f map[string]string) { f["risk_pct"] = "  " }, cervorules.ErrorCodeMissingFact},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			facts := sane()
			tc.mutate(facts)
			engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			_, err = engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationTradePlace, Metadata: facts})
			if err == nil {
				t.Fatal("expected the decision to fail")
			}
			var structured cervorules.Error
			if !errors.As(err, &structured) || structured.Code != tc.code {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
			if !structured.Sensitive && structured.Value != "" {
				t.Fatalf("an observed fact value must be marked sensitive: %#v", structured)
			}
		})
	}
}

// A malformed fact fails the decision even when an earlier rule would have
// denied anyway: the frame is built before any rule runs.
func TestMalformedFactBeatsAnEarlierDeny(t *testing.T) {
	facts := sane()
	facts["account_mode"] = "live"
	facts["risk_pct"] = "NaN"
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationTradePlace, Metadata: facts}); err == nil {
		t.Fatal("a malformed fact must not hide behind an earlier deny")
	}
}

// min_score declares a default, so an absent value is the policy's stated
// meaning of absence rather than a missing fact.
func TestDeclaredDefaultAppliesToAnAbsentFact(t *testing.T) {
	facts := sane()
	delete(facts, "min_score")
	facts["score"] = "40"
	result := decide(t, facts)
	if result.Decision.Allow {
		t.Fatalf("score below the defaulted minimum must deny, got %#v", result.Decision)
	}
	facts["score"] = "60"
	if result := decide(t, facts); !result.Decision.Allow {
		t.Fatalf("score above the defaulted minimum must allow, got %#v", result.Decision)
	}
}

// A deny with no operation applies to every operation.
func TestOperationWideDenyCoversEveryOperation(t *testing.T) {
	facts := sane()
	facts["account_mode"] = "live"
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, operation := range []cervorules.Operation{policyvocab.OperationTradePlace, policyvocab.OperationTradeClose} {
		result, err := engine.Decide(context.Background(), cervorules.Request{Operation: operation, Metadata: facts})
		if err != nil {
			t.Fatalf("decide %s: %v", operation, err)
		}
		if result.Decision.Allow {
			t.Fatalf("%s should be denied by the operation-wide rule: %#v", operation, result.Decision)
		}
	}
}

// The trace names the rule that denied and the leaf that decided it. Without
// it, a compound denial says only that the policy refused.
func TestTraceNamesTheDecidingLeaf(t *testing.T) {
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	facts := sane()
	facts["exposure_pct"] = "6.0"
	result, err := engine.DecideWithOptions(context.Background(), cervorules.Request{
		Operation: policyvocab.OperationTradePlace,
		Metadata:  facts,
	}, cervorules.NewDecisionOptions(cervorules.WithTrace(true)))
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if result.Trace == nil || len(result.Trace.Steps) == 0 {
		t.Fatalf("expected trace steps, got %#v", result.Trace)
	}
	last := result.Trace.Steps[len(result.Trace.Steps)-1]
	if last.Name != "deny-risk-or-exposure" || !last.Matched {
		t.Fatalf("expected the matching rule last, got %#v", result.Trace.Steps)
	}
	if last.Reason != "exposure_pct gt 5" {
		t.Fatalf("expected the deciding leaf, got %q", last.Reason)
	}
	for _, step := range result.Trace.Steps[:len(result.Trace.Steps)-1] {
		if step.Matched {
			t.Fatalf("only the last step may match: %#v", result.Trace.Steps)
		}
	}
}

// Trace is opt-in and stays that way.
func TestUntracedDecisionCarriesNoSteps(t *testing.T) {
	result := decide(t, sane())
	if result.Trace != nil {
		t.Fatalf("trace must stay opt-in, got %#v", result.Trace)
	}
}
`)
}

// Two denies on one operation used to emit a Go map literal with duplicate
// constant keys: check passed, generate passed, and only the consumer's build
// failed. Splitting a disjunction into two denies is the phase-B idiom, so this
// has to work.
func TestTwoDeniesOnOneOperationCompile(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: split.v3
defaults:
  executor: manual
facts:
  risk_pct: { min: 0 }
  exposure_pct: { min: 0 }
denies:
  - id: deny-risk-or-exposure
    operation: trade.place
    reason: risk or exposure over limit
    when: { fact: risk_pct, op: gt, value: 1.5 }
  - id: deny-risk-or-exposure
    operation: trade.place
    reason: risk or exposure over limit
    when: { fact: exposure_pct, op: gt, value: 5.0 }
routes:
  - id: allow-trade
    operation: trade.place
    target: desk
    executor: manual
`
	out, err := generatePredicatePolicy(t, predicateVocab, policy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	compileGeneratedPolicyWithVocab(t, predicateVocabGo, out.Source, `package policyrules

import (
	"context"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
	"example.test/generated/policyvocab"
)

func TestBothSplitDeniesFire(t *testing.T) {
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil { t.Fatalf("build: %v", err) }
	for name, facts := range map[string]map[string]string{
		"risk":     {"risk_pct": "9", "exposure_pct": "1"},
		"exposure": {"risk_pct": "1", "exposure_pct": "9"},
	} {
		result, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationTradePlace, Metadata: facts})
		if err != nil { t.Fatalf("%s: decide: %v", name, err) }
		if result.Decision.Allow || result.Decision.Reason != "risk or exposure over limit" {
			t.Fatalf("%s: expected deny, got %#v", name, result.Decision)
		}
	}
}
`)
}

func TestNestedCompositionAndNegation(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: nested.v3
defaults:
  executor: manual
denies:
  - id: deny-nested
    operation: trade.place
    when:
      all:
        - { fact: account_mode, op: eq, value: live }
        - not:
            any:
              - { fact: operable, op: is_true }
              - { fact: risk_pct, op: lt, value: 0.5 }
routes:
  - id: allow-trade
    operation: trade.place
    target: desk
    executor: manual
`
	out, err := generatePredicatePolicy(t, predicateVocab, policy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	compileGeneratedPolicyWithVocab(t, predicateVocabGo, out.Source, `package policyrules

import (
	"context"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
	"example.test/generated/policyvocab"
)

func TestNested(t *testing.T) {
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil { t.Fatalf("build: %v", err) }
	cases := map[string]struct{
		facts map[string]string
		allow bool
	}{
		"live, not operable, risky": {map[string]string{"account_mode": "live", "operable": "false", "risk_pct": "2"}, false},
		"live but operable":         {map[string]string{"account_mode": "live", "operable": "true", "risk_pct": "2"}, true},
		"live, not operable, tiny":  {map[string]string{"account_mode": "live", "operable": "false", "risk_pct": "0.1"}, true},
		"demo":                      {map[string]string{"account_mode": "demo", "operable": "false", "risk_pct": "2"}, true},
	}
	for name, tc := range cases {
		result, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationTradePlace, Metadata: tc.facts})
		if err != nil { t.Fatalf("%s: decide: %v", name, err) }
		if result.Decision.Allow != tc.allow {
			t.Fatalf("%s: allow=%v, want %v (%s)", name, result.Decision.Allow, tc.allow, result.Decision.Reason)
		}
	}
}
`)
}

// A predicate can consult a named condition, so ontology guards and process
// state stay reachable from the same expression as a fact comparison.
func TestPredicateCanConsultNamedCondition(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: mixed.v3
defaults:
  executor: manual
conditions:
  world_is_coherent:
    kind: integrity
denies:
  - id: deny-mixed
    operation: trade.place
    when:
      any:
        - { fact: risk_pct, op: gt, value: 1.5 }
        - not: { condition: world_is_coherent }
routes:
  - id: allow-trade
    operation: trade.place
    target: desk
    executor: manual
`
	out, err := generatePredicatePolicy(t, predicateVocab, policy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(out.Source, `cervorules.Condition("world_is_coherent")`) {
		t.Fatalf("predicate should consult the named condition:\n%s", out.Source)
	}
	compileGeneratedPolicyWithVocab(t, predicateVocabGo, out.Source, `package policyrules

import (
	"context"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
	"example.test/generated/policyvocab"
)

func TestConditionLeaf(t *testing.T) {
	build := func(coherent bool) cervorules.Engine {
		engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{
			Conditions: cervorules.ConditionSet{
				"world_is_coherent": func(context.Context, cervorules.Request) (bool, error) { return coherent, nil },
			},
		})
		if err != nil { t.Fatalf("build: %v", err) }
		return engine
	}
	facts := map[string]string{"risk_pct": "1.0"}
	if result, err := build(true).Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationTradePlace, Metadata: facts}); err != nil || !result.Decision.Allow {
		t.Fatalf("a coherent world with low risk must allow: %#v %v", result.Decision, err)
	}
	if result, err := build(false).Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationTradePlace, Metadata: facts}); err != nil || result.Decision.Allow {
		t.Fatalf("an incoherent world must deny: %#v %v", result.Decision, err)
	}
}

// An unanswerable condition inside a predicate fails the decision rather than
// counting as a non-match.
func TestUnansweredConditionFailsClosed(t *testing.T) {
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{
		Conditions: cervorules.ConditionSet{},
	})
	if err != nil { t.Fatalf("build: %v", err) }
	if _, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationTradePlace, Metadata: map[string]string{"risk_pct": "1.0"}}); err == nil {
		t.Fatal("an unregistered condition must fail the decision")
	}
}
`)
}

func TestPredicateAuthoringErrorsFailGeneration(t *testing.T) {
	cases := map[string]struct {
		rule string
		want string
	}{
		"undeclared fact": {
			`when: { fact: nonexistent, op: gt, value: 1 }`,
			`reads undeclared fact "nonexistent"`,
		},
		"operator wrong for type": {
			`when: { fact: account_mode, op: gt, value: demo }`,
			`is not valid for a enum fact`,
		},
		"enum value outside domain": {
			`when: { fact: account_mode, op: eq, value: paper }`,
			"outside the declared domain",
		},
		"numeric value on a bool fact": {
			`when: { fact: operable, op: eq, value: 3 }`,
			"expected a bool",
		},
		"fact-to-fact type mismatch": {
			`when: { fact: risk_pct, op: lt, fact_value: orders_last_hour }`,
			"compares number fact",
		},
		"fact compared against itself": {
			`when: { fact: risk_pct, op: lt, fact_value: risk_pct }`,
			"against itself",
		},
		"value and fact_value together": {
			`when: { fact: risk_pct, op: lt, value: 1, fact_value: score }`,
			"sets both value and fact_value",
		},
		"comparison with no value": {
			`when: { fact: risk_pct, op: lt }`,
			"needs value or fact_value",
		},
		"is_true with a value": {
			`when: { fact: operable, op: is_true, value: true }`,
			"takes no value",
		},
		"in with no values": {
			`when: { fact: account_mode, op: in }`,
			"needs values",
		},
		"two forms at once": {
			"when:\n      all: [{ fact: operable, op: is_true }]\n      any: [{ fact: operable, op: is_false }]",
			"must be exactly one of",
		},
		"empty predicate": {
			"when: {}",
			"must be exactly one of",
		},
		"undeclared condition": {
			`when: { condition: never_declared }`,
			`requires undeclared condition "never_declared"`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			policy := `
version: cervorules.policy.v3
name: broken.v3
defaults:
  executor: manual
denies:
  - id: broken
    operation: trade.place
    ` + tc.rule + "\n"
			_, err := generatePredicatePolicy(t, predicateVocab, policy)
			if err == nil {
				t.Fatal("expected generation to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestFactDeclarationErrorsFailGeneration(t *testing.T) {
	cases := map[string]struct {
		vocab  string
		policy string
		want   string
	}{
		"unknown fact type": {
			"operations:\n  trade.place: {}\nfacts:\n  weird: { type: colour }\n",
			"version: cervorules.policy.v3\nname: bad.v3\n",
			`unknown type "colour"`,
		},
		"enum without values": {
			"operations:\n  trade.place: {}\nfacts:\n  mode: { type: enum }\n",
			"version: cervorules.policy.v3\nname: bad.v3\n",
			"type enum needs values",
		},
		"values on a non-enum": {
			"operations:\n  trade.place: {}\nfacts:\n  risk_pct: { type: number, values: [a] }\n",
			"version: cervorules.policy.v3\nname: bad.v3\n",
			"must not declare values",
		},
		"missing type": {
			"operations:\n  trade.place: {}\nfacts:\n  risk_pct: {}\n",
			"version: cervorules.policy.v3\nname: bad.v3\n",
			"missing type",
		},
		"bounds on an undeclared fact": {
			predicateVocab,
			"version: cervorules.policy.v3\nname: bad.v3\nfacts:\n  nonexistent: { min: 0 }\n",
			"bounds for undeclared fact",
		},
		"bounds on a non-numeric fact": {
			predicateVocab,
			"version: cervorules.policy.v3\nname: bad.v3\nfacts:\n  operable: { min: 0 }\n",
			"cannot declare min or max",
		},
		"min above max": {
			predicateVocab,
			"version: cervorules.policy.v3\nname: bad.v3\nfacts:\n  risk_pct: { min: 5, max: 1 }\n",
			"min greater than max",
		},
		"fractional bound on an integer fact": {
			predicateVocab,
			"version: cervorules.policy.v3\nname: bad.v3\nfacts:\n  orders_last_hour: { min: 0.5 }\n",
			"fractional bound",
		},
		"default outside the enum domain": {
			predicateVocab,
			"version: cervorules.policy.v3\nname: bad.v3\nfacts:\n  account_mode: { default: paper }\n",
			"outside the declared domain",
		},
		"default of the wrong type": {
			predicateVocab,
			"version: cervorules.policy.v3\nname: bad.v3\nfacts:\n  risk_pct: { default: high }\n",
			"expected a number",
		},
		// A key written with no value used to be indistinguishable from an
		// absent one, so `default:` silently made the fact required — the
		// opposite of what it says.
		"default written with no value": {
			predicateVocab,
			"version: cervorules.policy.v3\nname: bad.v3\nfacts:\n  risk_pct:\n    default:\n",
			"declares an empty default",
		},
		// Values are compared after normalization, so two that differ only in
		// case would collapse and silently shrink the declared domain.
		"enum values colliding after normalization": {
			"operations:\n  trade.place: {}\nfacts:\n  mode: { type: enum, values: [Demo, demo] }\n",
			"version: cervorules.policy.v3\nname: bad.v3\n",
			"which are the same value",
		},
		// A fact the vocabulary can declare but no predicate could reference,
		// because `fact:` is constrained to the identifier shape.
		"fact name that is not an identifier": {
			"operations:\n  trade.place: {}\nfacts:\n  2fa_enabled: { type: bool }\n",
			"version: cervorules.policy.v3\nname: bad.v3\n",
			"is not a valid identifier",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Check(Options{
				VocabularyReader: strings.NewReader(tc.vocab),
				PolicyReader:     strings.NewReader(tc.policy),
			})
			if err == nil {
				t.Fatal("expected validation to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

// Every trace step must name something a reader can find in the policy file.
// The route step used to carry `reason`, which falls back to the literal
// "route matched" when a route authors no id — a name that names nothing.
func TestTraceStepsNameRulesNotFallbackText(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: named.v3
defaults:
  executor: manual
denies:
  - id: deny-risky
    operation: trade.place
    when: { fact: risk_pct, op: gt, value: 1.5 }
routes:
  - operation: trade.place
    target: desk
    executor: manual
`
	out, err := generatePredicatePolicy(t, predicateVocab, policy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	compileGeneratedPolicyWithVocab(t, predicateVocabGo, out.Source, `package policyrules

import (
	"context"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
	"example.test/generated/policyvocab"
)

func TestRouteStepIsNamedWithoutAnAuthoredID(t *testing.T) {
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil { t.Fatalf("build: %v", err) }
	result, err := engine.DecideWithOptions(context.Background(), cervorules.Request{
		Operation: policyvocab.OperationTradePlace,
		Metadata:  map[string]string{"risk_pct": "1.0"},
	}, cervorules.NewDecisionOptions(cervorules.WithTrace(true)))
	if err != nil { t.Fatalf("decide: %v", err) }
	if len(result.Trace.Steps) != 2 {
		t.Fatalf("expected a deny step and a route step, got %#v", result.Trace.Steps)
	}
	if result.Trace.Steps[0].Name != "deny-risky" {
		t.Fatalf("deny step must carry its id, got %q", result.Trace.Steps[0].Name)
	}
	// The route authored no id, so the operation stands in. What must never
	// appear is the "route matched" fallback text.
	if result.Trace.Steps[1].Name != "trade.place" {
		t.Fatalf("route step must be named, got %q", result.Trace.Steps[1].Name)
	}
	for _, step := range result.Trace.Steps {
		if step.Name == "" || step.Name == "route matched" || step.Name == "runtime override" {
			t.Fatalf("trace step is not identifiable: %#v", result.Trace.Steps)
		}
	}
}
`)
}

// A policy that declares no predicates must not pay for the machinery.
func TestPolicyWithoutPredicatesEmitsNoFactMachinery(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: plain.v3
defaults:
  executor: manual
routes:
  - id: allow-trade
    operation: trade.place
    target: desk
    executor: manual
`
	out, err := generatePredicatePolicy(t, predicateVocab, policy)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, forbidden := range []string{"factFrame", "newFactFrame", "ruleMatcher", `"strconv"`, `"math"`} {
		if strings.Contains(out.Source, forbidden) {
			t.Fatalf("a policy without predicates must not emit %q:\n%s", forbidden, out.Source)
		}
	}
	compileGeneratedPolicyWithVocab(t, predicateVocabGo, out.Source, minimalTestSource)
}
