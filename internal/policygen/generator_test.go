package policygen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const sampleVocab = `
operations:
  invoice_read: {}
targets:
  ledger_api: {}
executors:
  ledger_primary: {}
  ledger_backup: {}
`

const samplePolicy = `
version: cervorules.policy.v3
name: billing.v3
trusted_users:
  - finance-admin
defaults:
  executor: ledger_primary
routes:
  - id: invoice.read
    operation: invoice_read
    target: ledger_api
    executor: ledger_primary
    fallback_executors:
      - ledger_backup
denies:
  - id: deny.unknown
    operation: unknown_operation
    reason: unsupported operation
tests:
  - name: invoice read routes
    request:
      operation: invoice_read
    expect:
      allow: true
      target: ledger_api
      executor: ledger_primary
      fallback_executors:
        - ledger_backup
  - name: unknown denied
    request:
      operation: unknown_operation
    expect:
      allow: false
      reason_contains: unsupported
`

func TestCheckV3Policy(t *testing.T) {
	out, err := Check(Options{
		VocabularyReader: strings.NewReader(sampleVocab),
		PolicyReader:     strings.NewReader(samplePolicy),
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if out.Metadata.Name != "billing.v3" || out.Metadata.DSLVersion != "cervorules.policy.v3" {
		t.Fatalf("unexpected metadata: %#v", out.Metadata)
	}
	if out.Snapshot.Routes != 1 || out.Snapshot.Denies != 1 || out.Snapshot.Tests != 2 {
		t.Fatalf("unexpected snapshot: %#v", out.Snapshot)
	}
}

func TestCheckRejectsV2FieldsAndUnknownVocabulary(t *testing.T) {
	tests := []struct {
		name   string
		vocab  string
		policy string
		want   string
	}{
		{
			name:   "v2 policy version",
			vocab:  sampleVocab,
			policy: strings.Replace(samplePolicy, "cervorules.policy.v3", "cervorules.policy.v1", 1),
			want:   "unsupported policy version",
		},
		{
			name:  "unknown target",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
name: bad.v3
routes:
  - operation: invoice_read
    target: missing
    executor: ledger_primary
`,
			want: "unknown target",
		},
		{
			name:  "unknown operation",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
name: bad.v3
routes:
  - operation: missing
    target: ledger_api
    executor: ledger_primary
`,
			want: "unknown operation",
		},
		{
			name:  "unknown executor",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
name: bad.v3
routes:
  - operation: invoice_read
    target: ledger_api
    executor: missing
`,
			want: "unknown executor",
		},
		{
			name:  "unknown default executor",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
name: bad.v3
defaults:
  executor: missing
`,
			want: "unknown executor",
		},
		{
			name:  "unknown fallback executor",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
name: bad.v3
routes:
  - operation: invoice_read
    target: ledger_api
    executor: ledger_primary
    fallback_executors:
      - missing
`,
			want: "unknown executor",
		},
		{
			name:  "missing route operation",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
name: bad.v3
routes:
  - target: ledger_api
    executor: ledger_primary
`,
			want: "missing operation",
		},
		{
			name:  "missing deny operation",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
name: bad.v3
denies:
  - reason: no operation
`,
			want: "missing operation",
		},
		{
			name:  "missing test operation",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
name: bad.v3
tests:
  - name: missing request operation
    request: {}
    expect:
      allow: false
`,
			want: "missing request operation",
		},
		{
			name:  "missing policy name",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
routes: []
`,
			want: "policy name is required",
		},
		{
			name:  "v2 field rejected",
			vocab: sampleVocab,
			policy: `
version: cervorules.policy.v3
name: bad.v3
routes:
  - capability: invoice_read
    target: ledger_api
    executor: ledger_primary
`,
			want: "decode policy",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Check(Options{
				VocabularyReader: strings.NewReader(tt.vocab),
				PolicyReader:     strings.NewReader(tt.policy),
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func TestCheckRejectsBadReaders(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{name: "missing vocab reader", opts: Options{PolicyReader: strings.NewReader(samplePolicy)}, want: "missing vocabulary reader"},
		{name: "missing policy reader", opts: Options{VocabularyReader: strings.NewReader(sampleVocab)}, want: "missing policy reader"},
		{name: "bad vocab yaml", opts: Options{VocabularyReader: strings.NewReader("operations: ["), PolicyReader: strings.NewReader(samplePolicy)}, want: "decode vocabulary"},
		{name: "bad policy yaml", opts: Options{VocabularyReader: strings.NewReader(sampleVocab), PolicyReader: strings.NewReader("version: [")}, want: "decode policy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Check(tt.opts)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}

func TestGenerateV3PolicyFactoryCompilesAndRoutes(t *testing.T) {
	out, err := Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		CervoRulesImport:  "github.com/cervantesh/cervo-rules/v3",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(sampleVocab),
		PolicyReader:      strings.NewReader(samplePolicy),
		GeneratedTests:    true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	forbidden := []string{"BuildPolicy", "Capability", "Service", "Provider"}
	for _, value := range forbidden {
		if strings.Contains(out.Source, value) {
			t.Fatalf("generated v3 policy must not include %q:\n%s", value, out.Source)
		}
	}
	required := []string{
		"type PolicyFactory struct{}",
		"func NewPolicyFactory() PolicyFactory",
		"func (PolicyFactory) Build(ctx context.Context, cfg cervoruntime.PolicyRuntimeConfig) (cervorules.Engine, error)",
		"func (PolicyFactory) Metadata() cervoruntime.PolicyMetadata",
	}
	for _, want := range required {
		if !strings.Contains(out.Source, want) {
			t.Fatalf("generated source missing %q:\n%s", want, out.Source)
		}
	}
	compileGeneratedPolicy(t, out.Source, out.TestSource)
}

func TestGeneratedPolicyValidateConfigAggregatesRuntimeErrors(t *testing.T) {
	out, err := Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(sampleVocab),
		PolicyReader:      strings.NewReader(samplePolicy),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	compileGeneratedPolicy(t, out.Source, `package policyrules

import (
	"errors"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
)

func TestValidateConfigAggregatesRuntimeErrors(t *testing.T) {
	err := NewPolicyFactory().ValidateConfig(cervoruntime.PolicyRuntimeConfig{
		DefaultExecutor: cervorules.Executor("missing-default"),
		OperationTargets: map[cervorules.Operation]cervorules.Target{
			cervorules.Operation("missing-operation"): cervorules.Target("missing-target"),
		},
		ExecutorFallbacks: map[cervorules.Executor][]cervorules.Executor{
			cervorules.Executor("missing-executor"): {cervorules.Executor("missing-fallback")},
		},
	})
	if err == nil {
		t.Fatalf("expected validation errors")
	}
	var errs cervorules.Errors
	if !errors.As(err, &errs) {
		t.Fatalf("expected core.Errors, got %T %v", err, err)
	}
	if len(errs) != 5 {
		t.Fatalf("expected five validation errors, got %#v", errs)
	}
	for _, field := range []string{
		"defaults.executor",
		"operation_targets[missing-operation]",
		"operation_targets[missing-operation].target",
		"executor_fallbacks[missing-executor]",
		"executor_fallbacks[missing-executor][0]",
	} {
		if got := errs.ByField(field); len(got) != 1 {
			t.Fatalf("expected one error for field %q, got %#v", field, got)
		}
	}
}
`)
}

func TestGeneratedPolicyBuildWrapsValidationErrorsWithMetadata(t *testing.T) {
	out, err := Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(sampleVocab),
		PolicyReader:      strings.NewReader(samplePolicy),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	compileGeneratedPolicy(t, out.Source, `package policyrules

import (
	"context"
	"errors"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
)

func TestBuildWrapsValidationErrorsWithMetadata(t *testing.T) {
	_, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{
		DefaultExecutor: cervorules.Executor("missing-default"),
	})
	if err == nil {
		t.Fatalf("expected build error")
	}
	var buildErr *cervoruntime.PolicyBuildError
	if !errors.As(err, &buildErr) {
		t.Fatalf("expected PolicyBuildError, got %T %v", err, err)
	}
	if buildErr.Metadata.Name != "billing.v3" || buildErr.Metadata.PolicyHash == "" {
		t.Fatalf("expected generated metadata, got %#v", buildErr.Metadata)
	}
	var errs cervorules.Errors
	if !errors.As(err, &errs) || !errs.Has(cervorules.ErrorCodeUnknownExecutor) {
		t.Fatalf("expected structured source errors, got %T %v", err, err)
	}
}
`)
}

func TestGenerateV3PolicyTestsOmitStringsWhenNoReasonContains(t *testing.T) {
	policy := strings.Replace(samplePolicy, "reason_contains: unsupported", "reason: unsupported operation", 1)
	out, err := Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		CervoRulesImport:  "github.com/cervantesh/cervo-rules/v3",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(sampleVocab),
		PolicyReader:      strings.NewReader(policy),
		GeneratedTests:    true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if strings.Contains(out.TestSource, `"strings"`) {
		t.Fatalf("generated test imported strings without reason_contains:\n%s", out.TestSource)
	}
	compileGeneratedPolicy(t, out.Source, out.TestSource)
}

func TestGenerateV3PolicyWithoutTestsAndDisabledRoute(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: disabled.v3
routes:
  - operation: invoice_read
    target: ledger_api
    executor: ledger_primary
    disabled_by_default: true
    disabled_reason: disabled until configured
`
	out, err := Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(sampleVocab),
		PolicyReader:      strings.NewReader(policy),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if out.TestSource != "" {
		t.Fatalf("unexpected test source: %s", out.TestSource)
	}
	if strings.Contains(out.Source, `routes := map[cervorules.Operation]generatedRoute{
cervorules.Operation("invoice_read")`) {
		t.Fatalf("disabled route was emitted as active route:\n%s", out.Source)
	}
	if !strings.Contains(out.Source, "disabled until configured") {
		t.Fatalf("disabled reason missing:\n%s", out.Source)
	}
}

func TestGenerateV3PolicyEnablesDisabledRouteWithRuntimeTarget(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: disabled.v3
defaults:
  executor: ledger_primary
routes:
  - operation: invoice_read
    target: ledger_api
    executor: ledger_primary
    disabled_by_default: true
    disabled_reason: disabled until configured
`
	out, err := Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(sampleVocab),
		PolicyReader:      strings.NewReader(policy),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	compileGeneratedPolicy(t, out.Source, `package policyrules

import (
	"context"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
	"example.test/generated/policyvocab"
)

func TestRuntimeTargetEnablesDisabledRoute(t *testing.T) {
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil { t.Fatalf("build default: %v", err) }
	denied, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationInvoiceRead})
	if err != nil { t.Fatalf("decide default: %v", err) }
	if denied.Decision.Allow || denied.Decision.Reason != "disabled until configured" {
		t.Fatalf("expected disabled route deny, got %#v", denied.Decision)
	}

	engine, err = NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{
		DefaultExecutor: policyvocab.ExecutorLedgerBackup,
		OperationTargets: map[cervorules.Operation]cervorules.Target{policyvocab.OperationInvoiceRead: policyvocab.TargetLedgerAPI},
		ExecutorFallbacks: map[cervorules.Executor][]cervorules.Executor{policyvocab.ExecutorLedgerBackup: {policyvocab.ExecutorLedgerPrimary}},
	})
	if err != nil { t.Fatalf("build override: %v", err) }
	allowed, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationInvoiceRead})
	if err != nil { t.Fatalf("decide override: %v", err) }
	if !allowed.Decision.Allow || allowed.Decision.Target != policyvocab.TargetLedgerAPI {
		t.Fatalf("expected runtime override to enable route, got %#v", allowed.Decision)
	}
	if allowed.Decision.Executor != policyvocab.ExecutorLedgerBackup {
		t.Fatalf("expected disabled route to use runtime default executor, got %#v", allowed.Decision)
	}
	if len(allowed.Decision.FallbackExecutors) != 1 || allowed.Decision.FallbackExecutors[0] != policyvocab.ExecutorLedgerPrimary {
		t.Fatalf("expected runtime fallback executors, got %#v", allowed.Decision)
	}
}
`)
}

func TestGenerateV3PolicyAppliesRuntimeDefaultExecutorToDefaultRoutes(t *testing.T) {
	out, err := Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(sampleVocab),
		PolicyReader:      strings.NewReader(samplePolicy),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	compileGeneratedPolicy(t, out.Source, `package policyrules

import (
	"context"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
	"example.test/generated/policyvocab"
)

func TestRuntimeDefaultExecutorAppliesToDefaultRoute(t *testing.T) {
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{
		DefaultExecutor: policyvocab.ExecutorLedgerBackup,
		ExecutorFallbacks: map[cervorules.Executor][]cervorules.Executor{policyvocab.ExecutorLedgerBackup: {policyvocab.ExecutorLedgerPrimary}},
	})
	if err != nil { t.Fatalf("build override: %v", err) }
	result, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationInvoiceRead})
	if err != nil { t.Fatalf("decide: %v", err) }
	if !result.Decision.Allow {
		t.Fatalf("expected allow, got %#v", result.Decision)
	}
	if result.Decision.Executor != policyvocab.ExecutorLedgerBackup {
		t.Fatalf("expected runtime default executor, got %#v", result.Decision)
	}
	if len(result.Decision.FallbackExecutors) != 1 || result.Decision.FallbackExecutors[0] != policyvocab.ExecutorLedgerPrimary {
		t.Fatalf("expected runtime fallback executors, got %#v", result.Decision)
	}
}
`)
}

func TestGenerateV3PolicyEnforcesTrustedRoute(t *testing.T) {
	policy := `
version: cervorules.policy.v3
name: trusted.v3
trusted_users:
  - finance-admin
routes:
  - operation: invoice_read
    target: ledger_api
    executor: ledger_primary
    requires_trusted_user: true
`
	out, err := Generate(Options{
		PackageName:       "policyrules",
		VocabularyPackage: "policyvocab",
		VocabularyImport:  "example.test/generated/policyvocab",
		VocabularyReader:  strings.NewReader(sampleVocab),
		PolicyReader:      strings.NewReader(policy),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	compileGeneratedPolicy(t, out.Source, `package policyrules

import (
	"context"
	"strings"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"
	"example.test/generated/policyvocab"
)

func TestTrustedRouteRequiresTrustedUser(t *testing.T) {
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil { t.Fatalf("build: %v", err) }
	denied, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationInvoiceRead, User: "guest"})
	if err != nil { t.Fatalf("decide denied: %v", err) }
	if denied.Decision.Allow || !strings.Contains(denied.Decision.Reason, "trusted user") {
		t.Fatalf("expected trusted deny, got %#v", denied.Decision)
	}
	allowed, err := engine.Decide(context.Background(), cervorules.Request{Operation: policyvocab.OperationInvoiceRead, User: "finance-admin"})
	if err != nil { t.Fatalf("decide allowed: %v", err) }
	if !allowed.Decision.Allow {
		t.Fatalf("expected trusted user to pass, got %#v", allowed.Decision)
	}
}
`)
}

func TestGenerateRejectsBadOptions(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{name: "bad package", opts: Options{PackageName: "bad-name", VocabularyPackage: "policyvocab", VocabularyImport: "example.test/policyvocab", VocabularyReader: strings.NewReader(sampleVocab), PolicyReader: strings.NewReader(samplePolicy)}, want: "invalid package name"},
		{name: "missing vocab package", opts: Options{PackageName: "policyrules", VocabularyImport: "example.test/policyvocab", VocabularyReader: strings.NewReader(sampleVocab), PolicyReader: strings.NewReader(samplePolicy)}, want: "missing vocabulary package"},
		{name: "missing vocab import", opts: Options{PackageName: "policyrules", VocabularyPackage: "policyvocab", VocabularyReader: strings.NewReader(sampleVocab), PolicyReader: strings.NewReader(samplePolicy)}, want: "missing vocabulary import"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Generate(tt.opts)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}

func compileGeneratedPolicy(t *testing.T, source string, testSource string) {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.test/generated\n\ngo 1.25.8\n\nrequire github.com/cervantesh/cervo-rules/v3 v3.0.0\n\nreplace github.com/cervantesh/cervo-rules/v3 => "+filepath.ToSlash(repoRoot(t))+"\n")
	writeFile(t, filepath.Join(dir, "policyvocab", "vocab.go"), `package policyvocab

import "github.com/cervantesh/cervo-rules/v3/core"

const (
	OperationInvoiceRead core.Operation = "invoice_read"
	OperationUnknownOperation core.Operation = "unknown_operation"
	TargetLedgerAPI core.Target = "ledger_api"
	ExecutorLedgerPrimary core.Executor = "ledger_primary"
	ExecutorLedgerBackup core.Executor = "ledger_backup"
)

func Vocabulary() core.Vocabulary {
	return core.NewVocabulary(
		core.AllowedOperations(OperationInvoiceRead, OperationUnknownOperation),
		core.AllowedTargets(TargetLedgerAPI),
		core.AllowedExecutors(ExecutorLedgerPrimary, ExecutorLedgerBackup),
	)
}
`)
	writeFile(t, filepath.Join(dir, "policyrules", "generated_policy.go"), source)
	writeFile(t, filepath.Join(dir, "policyrules", "generated_policy_test.go"), testSource)
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated policy did not compile: %v\n%s", err, output)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found")
		}
		dir = parent
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
