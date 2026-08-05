package runtime

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
)

func TestPolicyBuildErrorCarriesMetadataAndStructuredErrors(t *testing.T) {
	meta := PolicyMetadata{Name: "billing.v3", DSLVersion: "cervorules.policy.v3", PolicyHash: "abc123"}
	source := core.Errors{{Code: core.ErrorCodeUnknownExecutor, Field: "defaults.executor", Reason: "unknown executor"}}

	err := NewPolicyBuildError(meta, source)
	if err == nil {
		t.Fatalf("expected policy build error")
	}
	if err.Metadata.Name != "billing.v3" {
		t.Fatalf("metadata was not preserved: %#v", err.Metadata)
	}
	if !strings.Contains(err.Error(), "billing.v3") || !strings.Contains(err.Error(), "unknown_executor") {
		t.Fatalf("error string should include policy and source: %q", err.Error())
	}

	var errs core.Errors
	if !errors.As(err, &errs) {
		t.Fatalf("expected errors.As to expose core.Errors")
	}
	if !errs.Has(core.ErrorCodeUnknownExecutor) {
		t.Fatalf("missing source error code: %#v", errs)
	}
}

func TestPolicyBuildErrorJSONDoesNotLeakCause(t *testing.T) {
	err := NewPolicyBuildError(
		PolicyMetadata{Name: "billing.v3"},
		errors.New("internal filesystem detail"),
	)

	raw, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("marshal policy build error: %v", marshalErr)
	}
	encoded := string(raw)
	if strings.Contains(encoded, "Cause") || strings.Contains(encoded, "internal filesystem detail") {
		t.Fatalf("cause leaked through JSON: %s", encoded)
	}
	for _, want := range []string{`"metadata"`, `"errors"`, `"code":"policy_build_failed"`} {
		if !strings.Contains(encoded, want) {
			t.Fatalf("JSON missing %q: %s", want, encoded)
		}
	}
}

func TestPolicyBuildErrorHandlesNilAndSingleStructuredError(t *testing.T) {
	if got := NewPolicyBuildError(PolicyMetadata{Name: "billing.v3"}, nil); got != nil {
		t.Fatalf("nil cause should not create error: %#v", got)
	}

	source := core.Error{Code: core.ErrorCodeInvalidRuntimeConfig, Field: "defaults.executor", Reason: "invalid config"}
	err := NewPolicyBuildError(PolicyMetadata{Name: "billing.v3"}, source)
	if err == nil {
		t.Fatalf("expected build error")
	}
	if got := err.Unwrap(); got == nil {
		t.Fatalf("expected source error to unwrap")
	}
	var first core.Error
	if !errors.As(err, &first) || first.Code != core.ErrorCodeInvalidRuntimeConfig {
		t.Fatalf("expected errors.As to expose first structured error, got %#v", first)
	}
}

func TestNilPolicyBuildErrorFallbacks(t *testing.T) {
	var err *PolicyBuildError
	if got := err.Error(); got != "policy build error" {
		t.Fatalf("unexpected nil error string: %q", got)
	}
	if got := err.Unwrap(); got != nil {
		t.Fatalf("nil build error should not unwrap: %#v", got)
	}
	var errs core.Errors
	if err.As(&errs) {
		t.Fatalf("nil build error should not match errors.As")
	}
}
