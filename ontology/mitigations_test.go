package ontology

import (
	"context"
	"errors"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
)

// Negative 2: without a request scope the guard resolved once per condition.
// With one, a decision consulting many conditions resolves exactly once.
func TestRequestScopeResolvesSnapshotOnce(t *testing.T) {
	calls := 0
	guard, guardErrs := NewGuard(orderModel(), func(context.Context, core.Request) (Snapshot, error) {
		calls++
		return Snapshot{States: []StateAssertion{
			{Lifecycle: "order", Individual: "o-1", State: "paid"},
		}}, nil
	}, map[core.Condition]Check{
		"refund_allowed": {Kind: CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "refunded"},
		"ship_allowed":   {Kind: CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "shipped"},
		"is_paid":        {Kind: CheckInState, Lifecycle: "order", SubjectKey: "order_id", States: []State{"paid"}},
	})
	requireValidOntology(t, guardErrs)

	req := core.Request{Operation: "refund", Metadata: map[string]string{"order_id": "o-1"}}
	ctx := WithRequestScope(context.Background())
	for _, condition := range []core.Condition{"refund_allowed", "ship_allowed", "is_paid"} {
		if _, err := guard.Holds(ctx, condition, req); err != nil {
			t.Fatalf("unexpected error for %s: %v", condition, err)
		}
	}
	if calls != 1 {
		t.Fatalf("expected 1 resolver call inside a request scope, got %d", calls)
	}
}

func TestWithoutRequestScopeBehaviorIsUnchanged(t *testing.T) {
	calls := 0
	guard, guardErrs := NewGuard(orderModel(), func(context.Context, core.Request) (Snapshot, error) {
		calls++
		return Snapshot{States: []StateAssertion{
			{Lifecycle: "order", Individual: "o-1", State: "paid"},
		}}, nil
	}, map[core.Condition]Check{
		"refund_allowed": {Kind: CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "refunded"},
		"ship_allowed":   {Kind: CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "shipped"},
	})
	requireValidOntology(t, guardErrs)
	req := core.Request{Metadata: map[string]string{"order_id": "o-1"}}
	for _, condition := range []core.Condition{"refund_allowed", "ship_allowed"} {
		if _, err := guard.Holds(context.Background(), condition, req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if calls != 2 {
		t.Fatalf("opting out must keep prior behavior, got %d calls", calls)
	}
}

func TestRequestScopeMemoizesResolverFailure(t *testing.T) {
	calls := 0
	guard, guardErrs := NewGuard(orderModel(), func(context.Context, core.Request) (Snapshot, error) {
		calls++
		return Snapshot{}, context.DeadlineExceeded
	}, map[core.Condition]Check{
		"a": {Kind: CheckIntegrity},
		"b": {Kind: CheckIntegrity},
	})
	requireValidOntology(t, guardErrs)
	ctx := WithRequestScope(context.Background())
	for _, condition := range []core.Condition{"a", "b"} {
		if _, err := guard.Holds(ctx, condition, core.Request{}); err == nil {
			t.Fatalf("expected failure for %s", condition)
		}
	}
	if calls != 1 {
		t.Fatalf("a failing resolver must not be retried per condition, got %d", calls)
	}
}

// A missing subject is unanswerable, so it must not cost a resolver round-trip.
func TestMissingSubjectShortCircuitsBeforeResolving(t *testing.T) {
	calls := 0
	guard, guardErrs := NewGuard(orderModel(), func(context.Context, core.Request) (Snapshot, error) {
		calls++
		return Snapshot{}, nil
	}, map[core.Condition]Check{
		"refund_allowed": {Kind: CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "refunded"},
	})
	requireValidOntology(t, guardErrs)
	if _, err := guard.Holds(context.Background(), "refund_allowed", core.Request{}); err == nil {
		t.Fatal("expected missing subject to fail")
	}
	if calls != 0 {
		t.Fatalf("missing subject must not resolve a snapshot, got %d calls", calls)
	}
}

// Negative 3: a consumer adds a constraint family the package does not ship.
type maxCardinality struct {
	relation Relation
	max      int
}

func (m maxCardinality) Name() string { return "max_cardinality." + string(m.relation) }

func (m maxCardinality) Validate(model Ontology) core.Errors {
	if m.max >= 1 {
		return nil
	}
	return core.Errors{{
		Code:      core.ErrorCodeInvalidConfig,
		Component: "consumer",
		Field:     m.Name(),
		Reason:    "max must be at least 1",
	}}
}

func (m maxCardinality) Check(_ Ontology, snapshot Snapshot) core.Errors {
	counts := map[Individual]int{}
	for _, assertion := range snapshot.Relations {
		if assertion.Relation == m.relation {
			counts[assertion.Subject]++
		}
	}
	var errs core.Errors
	for _, assertion := range snapshot.Relations {
		if counts[assertion.Subject] <= m.max {
			continue
		}
		if counts[assertion.Subject] == -1 {
			continue
		}
		errs = append(errs, core.Error{
			Code:      core.ErrorCodeInvalidConfig,
			Component: "consumer",
			Field:     m.Name(),
			Value:     string(assertion.Subject),
			Reason:    "too many objects for relation",
		})
		counts[assertion.Subject] = -1
	}
	return errs
}

func TestCustomConstraintExtendsCoverage(t *testing.T) {
	model := orderModel()
	model.Signatures = append(model.Signatures, PredicateSignature{
		Relation: "tagged_with", Domain: "order", Range: "order",
	})
	model.Custom = []Constraint{maxCardinality{relation: "tagged_with", max: 2}}

	if errs := model.Validate(); len(errs) > 0 {
		t.Fatalf("expected valid model, got %v", errs)
	}

	snapshot := Snapshot{
		Types: []TypeAssertion{
			{Individual: "o-1", Type: "order"},
			{Individual: "o-2", Type: "order"},
			{Individual: "o-3", Type: "order"},
			{Individual: "o-4", Type: "order"},
		},
		Relations: []RelationAssertion{
			{Relation: "tagged_with", Subject: "o-1", Object: "o-2"},
			{Relation: "tagged_with", Subject: "o-1", Object: "o-3"},
			{Relation: "tagged_with", Subject: "o-1", Object: "o-4"},
		},
	}
	errs := model.Check(snapshot)
	if len(errs) == 0 {
		t.Fatal("expected the custom constraint to fire")
	}
	found := false
	for _, err := range errs {
		if err.Field == "max_cardinality.tagged_with" {
			found = true
		}
	}
	if !found {
		t.Fatalf("custom constraint diagnostics missing: %v", errs)
	}
}

func TestCustomConstraintValidationReachesBuildTimeGate(t *testing.T) {
	model := orderModel()
	model.Custom = []Constraint{maxCardinality{relation: "tagged_with", max: 0}}
	if errs := model.Validate(); len(errs) == 0 {
		t.Fatal("expected the custom constraint's own validation to fail the build gate")
	}
}

func TestNilCustomConstraintIsIgnored(t *testing.T) {
	model := orderModel()
	model.Custom = []Constraint{nil}
	if errs := model.Validate(); len(errs) > 0 {
		t.Fatalf("nil constraint must be skipped, got %v", errs)
	}
	if errs := model.Check(Snapshot{}); len(errs) > 0 {
		t.Fatalf("nil constraint must be skipped, got %v", errs)
	}
}

// The Required flag used to be documented but never enforced.
func TestRequiredFunctionalPropertyIsEnforced(t *testing.T) {
	model := orderModel()
	for i := range model.Functional {
		if model.Functional[i].Relation == "order.parent" {
			model.Functional[i].Required = true
		}
	}
	snapshot := Snapshot{
		Types: []TypeAssertion{
			{Individual: "o-1", Type: "order"},
			{Individual: "o-2", Type: "order"},
		},
		Relations: []RelationAssertion{
			{Relation: "order.parent", Subject: "o-1", Object: "o-2"},
		},
	}
	errs := model.Check(snapshot)
	if !hasCode(errs, ErrorCodeFunctionalViolation) {
		t.Fatalf("expected o-2 to violate the required relation, got %v", errs)
	}
}

func TestRequiredWithoutDomainFailsValidation(t *testing.T) {
	model := Ontology{
		Entities:   []EntityType{"order"},
		Functional: []FunctionalProperty{{Relation: "orphan_relation", Required: true}},
	}
	errs := model.Validate()
	if !hasCode(errs, core.ErrorCodeInvalidConfig) {
		t.Fatalf("required without a domain must fail validation, got %v", errs)
	}
}

// requireValidOntology fails the test when NewGuard reports an invalid model.
// A guard built from an invalid ontology enforces less than it claims, so tests
// must never ignore this result.
func requireValidOntology(t *testing.T, errs core.Errors) {
	t.Helper()
	if len(errs) > 0 {
		t.Fatalf("ontology failed validation: %v", errs)
	}
}

// Reusing one scope across requests would answer the second against the first
// one's world. That is a wrong answer with no error, so it must be refused.
func TestRequestScopeRefusesReuseAcrossRequests(t *testing.T) {
	guard, guardErrs := NewGuard(orderModel(), func(_ context.Context, req core.Request) (Snapshot, error) {
		return Snapshot{States: []StateAssertion{
			{Lifecycle: "order", Individual: Individual(req.Metadata["order_id"]), State: "paid"},
		}}, nil
	}, map[core.Condition]Check{
		"refund_allowed": {Kind: CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "refunded"},
	})
	requireValidOntology(t, guardErrs)

	ctx := WithRequestScope(context.Background())
	first := core.Request{ID: "req-1", Metadata: map[string]string{"order_id": "o-1"}}
	if _, err := guard.Holds(ctx, "refund_allowed", first); err != nil {
		t.Fatalf("unexpected error on first request: %v", err)
	}

	second := core.Request{ID: "req-2", Metadata: map[string]string{"order_id": "o-2"}}
	holds, err := guard.Holds(ctx, "refund_allowed", second)
	if holds {
		t.Fatal("a reused scope must not answer a second request")
	}
	var structured core.Error
	if !errors.As(err, &structured) || structured.Code != core.ErrorCodeConditionFailed {
		t.Fatalf("expected condition_failed on scope reuse, got %v", err)
	}
}

// An ontology whose declaration is incomplete enforces less than it claims, so
// NewGuard must surface that rather than hand back a quietly weakened guard.
func TestNewGuardReportsInvalidOntology(t *testing.T) {
	broken := Ontology{
		Entities:   []EntityType{"order"},
		Functional: []FunctionalProperty{{Relation: "orphan", Required: true}},
	}
	_, errs := NewGuard(broken, func(context.Context, core.Request) (Snapshot, error) {
		return Snapshot{}, nil
	}, nil)
	if len(errs) == 0 {
		t.Fatal("NewGuard must report a model that cannot enforce what it declares")
	}
}

// Holds answers yes or no. Explain answers why, which is what a caller needs to
// tell whoever proposed the action what was wrong.
func TestExplainReturnsViolationsBehindAFalse(t *testing.T) {
	guard, guardErrs := NewGuard(orderModel(), func(context.Context, core.Request) (Snapshot, error) {
		return Snapshot{States: []StateAssertion{
			{Lifecycle: "order", Individual: "o-1", State: "refunded"},
		}}, nil
	}, map[core.Condition]Check{
		"refund_allowed": {Kind: CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "refunded"},
	})
	requireValidOntology(t, guardErrs)

	req := core.Request{ID: "r-1", Metadata: map[string]string{"order_id": "o-1"}}
	holds, violations, err := guard.Explain(context.Background(), "refund_allowed", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if holds {
		t.Fatal("a duplicate refund must not hold")
	}
	if len(violations) == 0 {
		t.Fatal("Explain must report why the condition failed")
	}
	if !violations.Has(ErrorCodeIllegalTransition) {
		t.Fatalf("expected illegal_transition among violations, got %v", violations)
	}
	// Holds must agree with Explain, just without the reason.
	sameAnswer, err := guard.Holds(context.Background(), "refund_allowed", req)
	if err != nil || sameAnswer != holds {
		t.Fatalf("Holds and Explain disagree: %v %v", sameAnswer, err)
	}
}

func TestExplainReturnsNoViolationsWhenConditionHolds(t *testing.T) {
	guard, guardErrs := NewGuard(orderModel(), func(context.Context, core.Request) (Snapshot, error) {
		return Snapshot{States: []StateAssertion{
			{Lifecycle: "order", Individual: "o-1", State: "paid"},
		}}, nil
	}, map[core.Condition]Check{
		"refund_allowed": {Kind: CheckTransitionAllowed, Lifecycle: "order", SubjectKey: "order_id", To: "refunded"},
	})
	requireValidOntology(t, guardErrs)

	req := core.Request{ID: "r-1", Metadata: map[string]string{"order_id": "o-1"}}
	holds, violations, err := guard.Explain(context.Background(), "refund_allowed", req)
	if err != nil || !holds || len(violations) != 0 {
		t.Fatalf("expected a clean pass, got holds=%v violations=%v err=%v", holds, violations, err)
	}
}

// Normalization became idempotent to stop the re-sorting storm. It must not
// change what normalization produces.
func TestNormalizeIsIdempotent(t *testing.T) {
	model := orderModel()
	once := model.Normalize()
	twice := once.Normalize()
	if len(once.Lifecycles) != len(twice.Lifecycles) || len(once.Entities) != len(twice.Entities) {
		t.Fatal("re-normalizing changed the model")
	}
	for i := range once.Entities {
		if once.Entities[i] != twice.Entities[i] {
			t.Fatalf("entity order changed at %d", i)
		}
	}
	snapshot := Snapshot{Types: []TypeAssertion{
		{Individual: "u-2", Type: "customer"},
		{Individual: "u-1", Type: "customer"},
	}}
	a := snapshot.Normalize()
	b := a.Normalize()
	for i := range a.Types {
		if a.Types[i] != b.Types[i] {
			t.Fatalf("snapshot order changed at %d", i)
		}
	}
	// An unnormalized copy must still sort the same way as the normalized one.
	fresh := Snapshot{Types: []TypeAssertion{
		{Individual: "u-2", Type: "customer"},
		{Individual: "u-1", Type: "customer"},
	}}.Normalize()
	for i := range fresh.Types {
		if fresh.Types[i] != a.Types[i] {
			t.Fatalf("normalization is not deterministic at %d", i)
		}
	}
}
