package core

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The error code strings are the wire contract. They appear in consumers' audit
// records and in the JSON of every structured error, and the docs call them
// stable. Nothing pinned them: every test compared the Go constants to each
// other, so renaming a value was invisible.
//
// This test asserts the literal string of every exported code, and fails when a
// new code is added without being listed here. That second half is the point:
// adding a code is a contract change and should be a deliberate edit.
func TestErrorCodeWireStrings(t *testing.T) {
	want := map[ErrorCode]string{
		ErrorCodeInvalidRule:        "invalid_rule",
		ErrorCodeInvalidConfig:      "invalid_config",
		ErrorCodeEmptyOperation:     "empty_operation",
		ErrorCodeUnknownOperation:   "unknown_operation",
		ErrorCodeEmptyTarget:        "empty_target",
		ErrorCodeUnknownTarget:      "unknown_target",
		ErrorCodeEmptyExecutor:      "empty_executor",
		ErrorCodeUnknownExecutor:    "unknown_executor",
		ErrorCodeRenamedPrimitive:   "renamed_primitive",
		ErrorCodeUnsupportedFeature: "unsupported_feature",
		ErrorCodeDeprecatedField:    "deprecated_field",

		ErrorCodeInvalidRuntimeConfig: "invalid_runtime_config",

		ErrorCodeInvalidPolicySchema:     "invalid_policy_schema",
		ErrorCodeSchemaValidationFailed:  "schema_validation_failed",
		ErrorCodePolicyBuildFailed:       "policy_build_failed",
		ErrorCodeGeneratedPolicyInvalid:  "generated_policy_invalid",
		ErrorCodeDeprecatedGeneratedAPI:  "deprecated_generated_api",
		ErrorCodeCompatBreakingChange:    "compat_breaking_change",
		ErrorCodeInternalInvariantFailed: "internal_invariant_failed",

		ErrorCodeEvaluationFailed:        "evaluation_failed",
		ErrorCodeContextCanceled:         "context_canceled",
		ErrorCodeContextDeadlineExceeded: "context_deadline_exceeded",
		ErrorCodeBudgetExceeded:          "budget_exceeded",
		ErrorCodeUnsafeRule:              "unsafe_rule",
		ErrorCodeUnsafeNegation:          "unsafe_negation",
		ErrorCodeMaxFactsExceeded:        "max_facts_exceeded",
		ErrorCodeMaxBindingsExceeded:     "max_bindings_exceeded",
		ErrorCodeMaxIterationsExceeded:   "max_iterations_exceeded",
		ErrorCodeExpensiveRule:           "expensive_rule",
		ErrorCodeRuleDisabled:            "rule_disabled",

		ErrorCodeUnknownCondition:  "unknown_condition",
		ErrorCodeConditionFailed:   "condition_failed",
		ErrorCodeMissingConditions: "missing_conditions",

		ErrorCodeMissingFact: "missing_fact",
		ErrorCodeInvalidFact: "invalid_fact",

		ErrorCodeBodyBytesExceeded: "body_bytes_exceeded",
		ErrorCodeMaxTokensExceeded: "max_tokens_exceeded",
		ErrorCodeStreamNotAllowed:  "stream_not_allowed",
		ErrorCodeToolsNotAllowed:   "tools_not_allowed",
		ErrorCodeImagesNotAllowed:  "images_not_allowed",
	}

	for code, text := range want {
		if string(code) != text {
			t.Errorf("wire string changed: got %q want %q", string(code), text)
		}
	}

	declared := declaredErrorCodeConstants(t)
	if len(declared) != len(want) {
		var missing []string
		listed := map[string]struct{}{}
		for code := range want {
			listed[string(code)] = struct{}{}
		}
		for _, value := range declared {
			if _, ok := listed[value]; !ok {
				missing = append(missing, value)
			}
		}
		sort.Strings(missing)
		t.Fatalf("core declares %d error codes but %d are pinned; unpinned: %v",
			len(declared), len(want), missing)
	}
}

var errorCodeDecl = regexp.MustCompile(`ErrorCode\w+\s+ErrorCode\s*=\s*"([a-z_]+)"`)

func declaredErrorCodeConstants(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("errors.go"))
	if err != nil {
		t.Fatalf("read errors.go: %v", err)
	}
	var out []string
	for _, match := range errorCodeDecl.FindAllStringSubmatch(string(data), -1) {
		out = append(out, match[1])
	}
	if len(out) == 0 {
		t.Fatal("found no error code declarations; the pattern is stale")
	}
	return out
}

// Every code must be documented. docs/v3/structured-errors.md is what a
// consumer reads to decide how to react to a refusal, and it listed a quarter
// of them.
func TestEveryErrorCodeIsDocumented(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "docs", "v3", "structured-errors.md"))
	if err != nil {
		t.Fatalf("read structured-errors.md: %v", err)
	}
	doc := string(data)

	var undocumented []string
	for _, code := range declaredErrorCodeConstants(t) {
		if !strings.Contains(doc, "`"+code+"`") {
			undocumented = append(undocumented, code)
		}
	}
	sort.Strings(undocumented)
	if len(undocumented) > 0 {
		t.Fatalf("codes missing from docs/v3/structured-errors.md: %v", undocumented)
	}
}
