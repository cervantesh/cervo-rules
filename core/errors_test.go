package core

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestStructuredErrorCarriesStableFieldsAndCause(t *testing.T) {
	cause := errors.New("source")
	err := Error{
		Code:   ErrorCodeUnknownOperation,
		Field:  "operation",
		Value:  "missing",
		Reason: "operation is not in vocabulary",
		Cause:  cause,
	}

	if !strings.Contains(err.Error(), "unknown_operation") || !strings.Contains(err.Error(), "operation") {
		t.Fatalf("error string should include code and field: %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatalf("structured error should unwrap cause")
	}
}

func TestErrorsCollectionSupportsCodeLookup(t *testing.T) {
	errs := Errors{
		{Code: ErrorCodeUnknownTarget, Field: "target"},
		{Code: ErrorCodeUnknownExecutor, Field: "executor"},
	}

	if !errs.Has(ErrorCodeUnknownTarget) {
		t.Fatalf("expected target error in collection: %v", errs)
	}
	if errs.Has(ErrorCodeUnknownOperation) {
		t.Fatalf("did not expect operation error in collection: %v", errs)
	}
	if !strings.Contains(errs.Error(), "unknown_target") {
		t.Fatalf("collection error should expose first code: %q", errs.Error())
	}
}

func TestErrorsCollectionHandlesEmptyAndSingleCollections(t *testing.T) {
	if got := (Errors{}).Error(); got != "no validation errors" {
		t.Fatalf("unexpected empty collection string: %q", got)
	}

	errs := Errors{{Code: ErrorCodeInvalidRule, Reason: "bad rule"}}
	if got := errs.Error(); !strings.Contains(got, "invalid_rule") || !strings.Contains(got, "bad rule") {
		t.Fatalf("unexpected single collection string: %q", got)
	}
}

func TestStructuredErrorHasFallbackMessage(t *testing.T) {
	if got := (Error{}).Error(); got != "cervorules error" {
		t.Fatalf("unexpected fallback string: %q", got)
	}
}

func TestStructuredErrorJSONContractHidesCause(t *testing.T) {
	err := Error{
		Code:       ErrorCodeMaxBindingsExceeded,
		Severity:   SeverityFatal,
		Component:  "facts",
		Field:      "max_bindings",
		Rule:       "reachability",
		Value:      "1001",
		Limit:      1000,
		Observed:   1001,
		Reason:     "binding budget exceeded",
		Suggestion: "reduce fanout or raise the budget",
		Cause:      errors.New("internal cause should not serialize"),
	}

	raw, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("marshal structured error: %v", marshalErr)
	}
	encoded := string(raw)
	if strings.Contains(encoded, "Cause") || strings.Contains(encoded, "internal cause") {
		t.Fatalf("cause leaked through JSON: %s", encoded)
	}

	var decoded map[string]any
	if unmarshalErr := json.Unmarshal(raw, &decoded); unmarshalErr != nil {
		t.Fatalf("unmarshal structured error JSON: %v", unmarshalErr)
	}
	assertJSONField(t, decoded, "code", string(ErrorCodeMaxBindingsExceeded))
	assertJSONField(t, decoded, "severity", string(SeverityFatal))
	assertJSONField(t, decoded, "component", "facts")
	assertJSONField(t, decoded, "field", "max_bindings")
	assertJSONField(t, decoded, "rule", "reachability")
	assertJSONField(t, decoded, "value", "1001")
	assertJSONField(t, decoded, "limit", float64(1000))
	assertJSONField(t, decoded, "observed", float64(1001))
	assertJSONField(t, decoded, "reason", "binding budget exceeded")
	assertJSONField(t, decoded, "suggestion", "reduce fanout or raise the budget")
}

func TestStructuredErrorJSONRedactsSensitiveValuesByDefault(t *testing.T) {
	for _, field := range []string{"token", "auth_header", "request_body", "prompt", "content", "memory"} {
		t.Run(field, func(t *testing.T) {
			raw, err := json.Marshal(Error{
				Code:   ErrorCodeInvalidConfig,
				Field:  field,
				Value:  "secret-value",
				Reason: "invalid sensitive value",
			})
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}
			encoded := string(raw)
			if strings.Contains(encoded, "secret-value") {
				t.Fatalf("sensitive value leaked for field %q: %s", field, encoded)
			}
			if !strings.Contains(encoded, `"value":"[REDACTED]"`) {
				t.Fatalf("redacted marker missing for field %q: %s", field, encoded)
			}
		})
	}
}

func TestStructuredErrorRedactedPreservesNonSensitiveValues(t *testing.T) {
	err := Error{Code: ErrorCodeInvalidConfig, Field: "operation", Value: "read", Reason: "bad operation"}
	if got := err.Redacted().Value; got != "read" {
		t.Fatalf("non-sensitive value should be preserved, got %q", got)
	}
	err.Field = "authorization"
	if got := err.Redacted().Value; got != "[REDACTED]" {
		t.Fatalf("sensitive value should be redacted, got %q", got)
	}

	errs := Errors{
		{Code: ErrorCodeInvalidConfig, Field: "token", Value: "secret"},
		{Code: ErrorCodeInvalidConfig, Field: "operation", Value: "read"},
	}.Redacted()
	if errs[0].Value != "[REDACTED]" || errs[1].Value != "read" {
		t.Fatalf("unexpected redacted collection: %#v", errs)
	}
}

func TestErrorCodeRegistryIncludesStableCrossComponentCodes(t *testing.T) {
	required := []ErrorCode{
		ErrorCodeInvalidConfig,
		ErrorCodeInvalidRuntimeConfig,
		ErrorCodeInvalidPolicySchema,
		ErrorCodeSchemaValidationFailed,
		ErrorCodeUnsupportedFeature,
		ErrorCodeDeprecatedField,
		ErrorCodeInternalInvariantFailed,
		ErrorCodeEvaluationFailed,
		ErrorCodeContextCanceled,
		ErrorCodeContextDeadlineExceeded,
		ErrorCodeBudgetExceeded,
		ErrorCodeUnsafeRule,
		ErrorCodeUnsafeNegation,
		ErrorCodeMaxFactsExceeded,
		ErrorCodeMaxBindingsExceeded,
		ErrorCodeMaxIterationsExceeded,
		ErrorCodeExpensiveRule,
		ErrorCodeRuleDisabled,
		ErrorCodePolicyBuildFailed,
		ErrorCodeGeneratedPolicyInvalid,
		ErrorCodeRenamedPrimitive,
		ErrorCodeDeprecatedGeneratedAPI,
		ErrorCodeCompatBreakingChange,
		ErrorCodeBodyBytesExceeded,
		ErrorCodeMaxTokensExceeded,
		ErrorCodeStreamNotAllowed,
		ErrorCodeToolsNotAllowed,
		ErrorCodeImagesNotAllowed,
	}

	seen := map[ErrorCode]bool{}
	for _, code := range required {
		if code == "" {
			t.Fatalf("registry contains empty code")
		}
		if seen[code] {
			t.Fatalf("registry contains duplicate code: %s", code)
		}
		seen[code] = true
	}
}

func TestErrorsCollectionFiltersFatalAndSortsStable(t *testing.T) {
	errs := Errors{
		{Code: ErrorCodeUnknownExecutor, Field: "executor", Reason: "z"},
		{Code: ErrorCodeUnknownOperation, Field: "operation", Severity: SeverityFatal, Reason: "fatal"},
		{Code: ErrorCodeUnknownTarget, Field: "target", Reason: "target"},
		{Code: ErrorCodeEmptyOperation, Field: "operation", Reason: "empty"},
	}

	if got := errs.ByCode(ErrorCodeUnknownOperation); len(got) != 1 || got[0].Reason != "fatal" {
		t.Fatalf("unexpected ByCode result: %#v", got)
	}
	if got := errs.ByField("operation"); len(got) != 2 {
		t.Fatalf("unexpected ByField result: %#v", got)
	}
	if got := errs.Fatal(); len(got) != 1 || got[0].Code != ErrorCodeUnknownOperation {
		t.Fatalf("unexpected Fatal result: %#v", got)
	}

	sorted := errs.SortStable()
	gotCodes := []ErrorCode{sorted[0].Code, sorted[1].Code, sorted[2].Code, sorted[3].Code}
	wantCodes := []ErrorCode{
		ErrorCodeUnknownExecutor,
		ErrorCodeEmptyOperation,
		ErrorCodeUnknownOperation,
		ErrorCodeUnknownTarget,
	}
	if !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Fatalf("unexpected stable sort order: got %v want %v", gotCodes, wantCodes)
	}

	originalCodes := []ErrorCode{errs[0].Code, errs[1].Code, errs[2].Code, errs[3].Code}
	if reflect.DeepEqual(gotCodes, originalCodes) {
		t.Fatalf("SortStable should return sorted copy, got original ordering")
	}
}

func TestErrorsAsAndAppendValidationFallbacks(t *testing.T) {
	var empty Errors
	var first Error
	if empty.As(&first) {
		t.Fatalf("empty collection should not expose first error")
	}
	var copied Errors
	if !(Errors{{Code: ErrorCodeInvalidConfig}}).As(&copied) || !copied.Has(ErrorCodeInvalidConfig) {
		t.Fatalf("expected As to expose collection: %#v", copied)
	}
	var text string
	if (Errors{{Code: ErrorCodeInvalidConfig}}).As(&text) {
		t.Fatalf("unexpected As match for string")
	}

	errs := appendValidationError(nil, errors.New("plain failure"), "plain")
	if len(errs) != 1 || errs[0].Code != ErrorCodeInvalidConfig || errs[0].Field != "plain" {
		t.Fatalf("unexpected plain error fallback: %#v", errs)
	}
	errs = appendValidationError(nil, Errors{{Code: ErrorCodeUnknownTarget}}, "target")
	if len(errs) != 1 || errs[0].Field != "target" {
		t.Fatalf("unexpected multi-error fallback: %#v", errs)
	}
}

func assertJSONField(t *testing.T, decoded map[string]any, field string, want any) {
	t.Helper()
	got, ok := decoded[field]
	if !ok {
		t.Fatalf("missing JSON field %q in %#v", field, decoded)
	}
	if got != want {
		t.Fatalf("unexpected JSON field %q: got %#v want %#v", field, got, want)
	}
}

func TestVocabularyValidationReturnsStructuredErrors(t *testing.T) {
	vocab := NewVocabulary(AllowedOperations("read"))
	err := vocab.ValidateOperation("write")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	var structured Error
	if !errors.As(err, &structured) {
		t.Fatalf("expected structured error, got %T %v", err, err)
	}
	if structured.Code != ErrorCodeUnknownOperation || structured.Field != "operation" || structured.Value != "write" {
		t.Fatalf("unexpected structured error: %#v", structured)
	}
}

func TestVocabularyValidationCoversEmptyAndFallbackFields(t *testing.T) {
	vocab := NewVocabulary(
		AllowedOperations("read"),
		AllowedTargets("queue"),
		AllowedExecutors("worker"),
	)

	cases := []struct {
		name string
		err  error
		code ErrorCode
	}{
		{name: "empty operation", err: vocab.ValidateOperation(" "), code: ErrorCodeEmptyOperation},
		{name: "empty target", err: vocab.ValidateTarget(" "), code: ErrorCodeEmptyTarget},
		{name: "empty executor", err: vocab.ValidateExecutor(" "), code: ErrorCodeEmptyExecutor},
		{name: "unknown target fallback", err: vocab.ValidateDecision(Decision{FallbackTargets: []Target{"missing"}}), code: ErrorCodeUnknownTarget},
		{name: "unknown executor fallback", err: vocab.ValidateDecision(Decision{FallbackExecutors: []Executor{"missing"}}), code: ErrorCodeUnknownExecutor},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var structured Error
			if !errors.As(tc.err, &structured) {
				t.Fatalf("expected structured error, got %T %v", tc.err, tc.err)
			}
			if structured.Code != tc.code {
				t.Fatalf("unexpected code: got %s want %s", structured.Code, tc.code)
			}
		})
	}
}

func TestValidateRequestAllowsMissingOperation(t *testing.T) {
	vocab := NewVocabulary(AllowedOperations("read"))
	if err := vocab.ValidateRequest(Request{}); err != nil {
		t.Fatalf("missing request operation should remain optional: %v", err)
	}
}

func TestValidateDecisionAcceptsKnownFields(t *testing.T) {
	vocab := NewVocabulary(
		AllowedTargets("queue", "archive"),
		AllowedExecutors("worker", "fallback"),
	)
	decision := Decision{
		Target:            "queue",
		Executor:          "worker",
		FallbackTargets:   []Target{"archive"},
		FallbackExecutors: []Executor{"fallback"},
	}
	if err := vocab.ValidateDecision(decision); err != nil {
		t.Fatalf("known decision fields should validate: %v", err)
	}
}

func TestValidateDecisionAggregatesVocabularyErrors(t *testing.T) {
	vocab := NewVocabulary(
		AllowedTargets("queue"),
		AllowedExecutors("worker"),
	)
	err := vocab.ValidateDecision(Decision{
		Target:            "missing-target",
		Executor:          "missing-executor",
		FallbackTargets:   []Target{"missing-fallback-target"},
		FallbackExecutors: []Executor{"missing-fallback-executor"},
	})
	if err == nil {
		t.Fatalf("expected aggregated validation errors")
	}
	var errs Errors
	if !errors.As(err, &errs) {
		t.Fatalf("expected core.Errors, got %T %v", err, err)
	}
	if len(errs) != 4 {
		t.Fatalf("expected four validation errors, got %#v", errs)
	}
	for _, field := range []string{"target", "executor", "fallback_targets[0]", "fallback_executors[0]"} {
		if got := errs.ByField(field); len(got) != 1 {
			t.Fatalf("expected one error for field %q, got %#v", field, got)
		}
	}
}
