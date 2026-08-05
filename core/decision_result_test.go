package core

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestNewDecisionResultDefaultsToProductionHotPath(t *testing.T) {
	req := Request{ID: "req-1", Operation: "read.item", User: "sensitive-user"}
	decision := Decision{Allow: true, Target: "queue", Executor: "worker"}

	result := NewDecisionResult(req, decision)

	if !reflect.DeepEqual(result.Decision, decision) {
		t.Fatalf("unexpected decision: %#v", result.Decision)
	}
	if result.Trace != nil {
		t.Fatalf("trace should be opt-in by default: %#v", result.Trace)
	}
	if result.Observation != nil {
		t.Fatalf("observation should be opt-in by default: %#v", result.Observation)
	}
}

func TestNewDecisionResultCanOptIntoTraceAndObservation(t *testing.T) {
	req := Request{ID: "req-2", Operation: "read.item", User: "sensitive-user"}
	decision := Decision{Allow: false, Target: "queue", Executor: "worker", Reason: "blocked"}

	result := NewDecisionResult(req, decision, WithTrace(true), WithObservation(true))

	if result.Trace == nil {
		t.Fatalf("expected trace when enabled")
	}
	if result.Trace.SchemaVersion != DecisionTraceSchemaVersion {
		t.Fatalf("unexpected trace schema version: %q", result.Trace.SchemaVersion)
	}
	if result.Observation == nil {
		t.Fatalf("expected observation when enabled")
	}
	if result.Observation.SchemaVersion != DecisionObservationSchemaVersion {
		t.Fatalf("unexpected observation schema version: %q", result.Observation.SchemaVersion)
	}
	if result.Observation.RequestID != "req-2" || result.Observation.Operation != "read.item" {
		t.Fatalf("unexpected observation: %#v", result.Observation)
	}
	if result.Observation.User != "" {
		t.Fatalf("observation must not copy request user by default: %#v", result.Observation)
	}
}

func TestDecisionOptionsFunctionalOptionsAreExplicit(t *testing.T) {
	options := NewDecisionOptions(WithTrace(false), WithObservation(true))

	if options.TraceEnabled() {
		t.Fatalf("trace should be disabled explicitly")
	}
	if !options.ObservationEnabled() {
		t.Fatalf("observation should be enabled explicitly")
	}

	defaults := NewDecisionOptions()
	if defaults.TraceEnabled() || defaults.ObservationEnabled() {
		t.Fatalf("v3 defaults should be hot-path oriented: %#v", defaults)
	}
}

func TestEngineInterfaceReturnsDecisionResult(t *testing.T) {
	var engine Engine = engineFunc(func(context.Context, Request, DecisionOptions) (DecisionResult, error) {
		return NewDecisionResult(Request{ID: "req-3"}, Decision{Allow: true}), nil
	})

	result, err := engine.Decide(context.Background(), Request{ID: "req-3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Decision.Allow {
		t.Fatalf("expected allow decision: %#v", result.Decision)
	}
}

func TestDecisionResultDiagnosticsUseStructuredErrors(t *testing.T) {
	result := NewDecisionResult(Request{ID: "req-4"}, Decision{Allow: false})
	result.Diagnostics = Errors{{Code: ErrorCodeInvalidRule, Field: "rule"}}
	result.Stats.Diagnostics = len(result.Diagnostics)

	if !result.Diagnostics.Has(ErrorCodeInvalidRule) {
		t.Fatalf("expected structured diagnostic: %#v", result.Diagnostics)
	}
	if !strings.Contains(result.Diagnostics.Error(), "invalid_rule") {
		t.Fatalf("diagnostics should expose stable code: %q", result.Diagnostics.Error())
	}
}

type engineFunc func(context.Context, Request, DecisionOptions) (DecisionResult, error)

func (fn engineFunc) Decide(ctx context.Context, req Request) (DecisionResult, error) {
	return fn(ctx, req, DecisionOptions{})
}

func (fn engineFunc) DecideWithOptions(ctx context.Context, req Request, options DecisionOptions) (DecisionResult, error) {
	return fn(ctx, req, options)
}
