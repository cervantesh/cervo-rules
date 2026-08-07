package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
	"github.com/cervantesh/cervo-rules/v3/internal/policygen"
)

func TestRunCheckAndGenerate(t *testing.T) {
	dir := t.TempDir()
	vocab := filepath.Join(dir, "policy-vocabulary.yaml")
	policy := filepath.Join(dir, "policy-rules.yaml")
	out := filepath.Join(dir, "generated_policy.go")
	testOut := filepath.Join(dir, "generated_policy_test.go")
	writeFile(t, vocab, `
operations:
  read: {}
targets:
  queue: {}
executors:
  worker: {}
`)
	writeFile(t, policy, `
version: cervorules.policy.v3
name: cli.v3
routes:
  - operation: read
    target: queue
    executor: worker
`)

	var stdout, stderr strings.Builder
	if code := run([]string{"check", "-vocab", vocab, "-policy", policy}, &stdout, &stderr); code != 0 {
		t.Fatalf("check failed code=%d stderr=%s", code, stderr.String())
	}
	stderr.Reset()
	if code := run([]string{"check", "-vocab", vocab, "-policy", policy, "-format", "json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("json check failed code=%d stderr=%s", code, stderr.String())
	}
	// The machine-readable result goes to stdout; diagnostics stay on stderr.
	if !strings.Contains(stdout.String(), `"policy_hash"`) {
		t.Fatalf("json check output missing metadata: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if strings.Contains(stderr.String(), `"policy_hash"`) {
		t.Fatalf("the JSON result must not go to stderr: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := run([]string{
		"generate",
		"-vocab", vocab,
		"-policy", policy,
		"-out", out,
		"-test-out", testOut,
		"-package", "policyrules",
		"-vocab-package", "policyvocab",
		"-vocab-import", "example.test/policyvocab",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("generate failed code=%d stderr=%s", code, stderr.String())
	}
	generated, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read generated: %v", err)
	}
	if !strings.Contains(string(generated), "NewPolicyFactory") {
		t.Fatalf("generated policy missing factory:\n%s", generated)
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr strings.Builder
	code := run([]string{"-version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("version failed: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "cervorules-policygen") {
		t.Fatalf("version output missing tool name: %q", stderr.String())
	}
}

func TestRunErrors(t *testing.T) {
	dir := t.TempDir()
	vocab := filepath.Join(dir, "policy-vocabulary.yaml")
	policy := filepath.Join(dir, "policy-rules.yaml")
	writeFile(t, vocab, `
operations:
  read: {}
targets:
  queue: {}
executors:
  worker: {}
`)
	writeFile(t, policy, `
version: cervorules.policy.v3
name: cli.v3
routes:
  - operation: read
    target: queue
    executor: worker
`)

	tests := []struct {
		name string
		args []string
		code int
		want string
	}{
		{name: "no args", args: nil, code: 2, want: "usage"},
		{name: "unknown command", args: []string{"nope"}, code: 2, want: "unknown command"},
		{name: "check missing flags", args: []string{"check"}, code: 2, want: "required"},
		{name: "check bad flag", args: []string{"check", "-bad"}, code: 2, want: "flag provided"},
		{name: "check open error", args: []string{"check", "-vocab", filepath.Join(dir, "missing.yaml"), "-policy", policy}, code: 1, want: "open vocabulary"},
		{name: "check invalid format", args: []string{"check", "-vocab", vocab, "-policy", policy, "-format", "xml"}, code: 2, want: "unsupported"},
		{name: "generate missing flags", args: []string{"generate"}, code: 2, want: "required"},
		{name: "generate bad flag", args: []string{"generate", "-bad"}, code: 2, want: "flag provided"},
		{name: "generate open error", args: []string{"generate", "-vocab", vocab, "-policy", filepath.Join(dir, "missing.yaml"), "-out", filepath.Join(dir, "out.go"), "-package", "policyrules", "-vocab-package", "policyvocab", "-vocab-import", "example.test/policyvocab"}, code: 1, want: "open policy"},
		{name: "generate write error", args: []string{"generate", "-vocab", vocab, "-policy", policy, "-out", dir, "-package", "policyrules", "-vocab-package", "policyvocab", "-vocab-import", "example.test/policyvocab"}, code: 1, want: "write policy"},
		{name: "generate test write error", args: []string{"generate", "-vocab", vocab, "-policy", policy, "-out", filepath.Join(dir, "ok.go"), "-test-out", dir, "-package", "policyrules", "-vocab-package", "policyvocab", "-vocab-import", "example.test/policyvocab"}, code: 1, want: "write policy tests"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			if code := run(tt.args, &stdout, &stderr); code != tt.code {
				t.Fatalf("code=%d want=%d stderr=%s", code, tt.code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr missing %q: %s", tt.want, stderr.String())
			}
		})
	}
}

func TestRunCheckJSONValidationErrorReport(t *testing.T) {
	dir := t.TempDir()
	vocab := filepath.Join(dir, "policy-vocabulary.yaml")
	policy := filepath.Join(dir, "policy-rules.yaml")
	writeFile(t, vocab, `
operations:
  read: {}
targets:
  queue: {}
executors:
  worker: {}
`)
	writeFile(t, policy, `
version: cervorules.policy.v3
name: bad.v3
routes:
  - operation: missing
    target: queue
    executor: worker
`)

	var stdout, stderr strings.Builder
	code := run([]string{"check", "-vocab", vocab, "-policy", policy, "-format", "json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code=%d want=2 stderr=%s", code, stderr.String())
	}
	var report struct {
		ExitCode int          `json:"exit_code"`
		Errors   []core.Error `json:"errors"`
	}
	if err := json.Unmarshal([]byte(stderr.String()), &report); err != nil {
		t.Fatalf("expected JSON error report, got %q: %v", stderr.String(), err)
	}
	if report.ExitCode != 2 || len(report.Errors) != 1 || report.Errors[0].Code != core.ErrorCodeInvalidPolicySchema {
		t.Fatalf("unexpected JSON error report: %#v", report)
	}
}

func TestWriteCLIErrorTextAndJSON(t *testing.T) {
	err := core.Error{Code: core.ErrorCodeInvalidConfig, Reason: "bad config"}

	var text strings.Builder
	if code := writeCLIError(&text, "text", 2, err, "prefix"); code != 2 {
		t.Fatalf("text code=%d want=2", code)
	}
	if !strings.Contains(text.String(), "prefix: bad config") {
		t.Fatalf("unexpected text output: %q", text.String())
	}

	var raw strings.Builder
	if code := writeCLIError(&raw, "", 2, err, ""); code != 2 {
		t.Fatalf("default code=%d want=2", code)
	}
	if !strings.Contains(raw.String(), "invalid_config") {
		t.Fatalf("unexpected default output: %q", raw.String())
	}

	var js strings.Builder
	if code := writeCLIError(&js, "json", 2, err, "prefix"); code != 2 {
		t.Fatalf("json code=%d want=2", code)
	}
	var report struct {
		ExitCode int          `json:"exit_code"`
		Errors   []core.Error `json:"errors"`
	}
	if parseErr := json.Unmarshal([]byte(js.String()), &report); parseErr != nil {
		t.Fatalf("parse JSON output %q: %v", js.String(), parseErr)
	}
	if report.ExitCode != 2 || len(report.Errors) != 1 || report.Errors[0].Code != core.ErrorCodeInvalidConfig {
		t.Fatalf("unexpected JSON output: %#v", report)
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

// The metadata schema was published without a producer: it required a
// schema_version field and sha256:-prefixed hashes that nothing emitted. Now
// that -metadata-out writes the document, this pins it to the schema's own
// required keys, consts and patterns, so the contract and the producer cannot
// drift apart again.
func TestGeneratedPolicyMetadataMatchesItsSchema(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "schemas", "v3", "generated-policy-metadata.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Const   string `json:"const"`
			Pattern string `json:"pattern"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	document := generatedPolicyMetadata(policygen.PolicyMetadata{
		Name:           "sample.v3",
		DSLVersion:     "cervorules.policy.v3",
		GeneratedWith:  "cervorules-policygen/v3",
		VocabularyHash: "aa",
		PolicyHash:     "bb",
	})

	for _, key := range schema.Required {
		if document[key] == "" {
			t.Errorf("required key %q is missing or empty in the produced document", key)
		}
	}
	for key := range document {
		if _, ok := schema.Properties[key]; !ok {
			t.Errorf("produced key %q is not allowed by the schema", key)
		}
	}
	for key, rule := range schema.Properties {
		value := document[key]
		if rule.Const != "" && value != rule.Const {
			t.Errorf("%s: got %q want const %q", key, value, rule.Const)
		}
		if rule.Pattern != "" {
			matched, err := regexp.MatchString(rule.Pattern, value)
			if err != nil {
				t.Fatalf("%s: bad pattern in schema: %v", key, err)
			}
			if !matched {
				t.Errorf("%s: %q does not match %q", key, value, rule.Pattern)
			}
		}
	}
}
