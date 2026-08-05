package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
)

func TestRunGenerateVocabularyFile(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "policy-vocabulary.yaml")
	out := filepath.Join(dir, "generated.go")
	if err := os.WriteFile(in, []byte(`
operations:
  read_item: {}
targets:
  queue: {}
executors:
  worker: {}
`), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stderr strings.Builder
	code := run([]string{"-in", in, "-out", out, "-package", "policyvocab"}, &stderr)
	if code != 0 {
		t.Fatalf("run failed code=%d stderr=%s", code, stderr.String())
	}
	generated, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(generated), "OperationReadItem") {
		t.Fatalf("unexpected generated output:\n%s", generated)
	}
}

func TestRunVersion(t *testing.T) {
	var stderr strings.Builder
	code := run([]string{"-version"}, &stderr)
	if code != 0 {
		t.Fatalf("version failed: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "cervorules-vocabgen") {
		t.Fatalf("version output missing tool name: %q", stderr.String())
	}
}

func TestRunErrors(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "policy-vocabulary.yaml")
	if err := os.WriteFile(in, []byte(`
operations:
  read_item: {}
targets:
  queue: {}
executors:
  worker: {}
`), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	tests := []struct {
		name string
		args []string
		code int
		want string
	}{
		{name: "bad flag", args: []string{"-bad"}, code: 2, want: "flag provided"},
		{name: "missing required", args: nil, code: 2, want: "required"},
		{name: "open error", args: []string{"-in", filepath.Join(dir, "missing.yaml"), "-out", filepath.Join(dir, "out.go"), "-package", "policyvocab"}, code: 1, want: "open input"},
		{name: "generate error", args: []string{"-in", in, "-out", filepath.Join(dir, "out.go"), "-package", "bad-name"}, code: 2, want: "generate vocabulary"},
		{name: "write error", args: []string{"-in", in, "-out", dir, "-package", "policyvocab"}, code: 1, want: "write output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr strings.Builder
			if code := run(tt.args, &stderr); code != tt.code {
				t.Fatalf("code=%d want=%d stderr=%s", code, tt.code, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr missing %q: %s", tt.want, stderr.String())
			}
		})
	}
}

func TestRunJSONValidationErrorReport(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "policy-vocabulary.yaml")
	if err := os.WriteFile(in, []byte(`
operations:
  read_item: {}
`), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stderr strings.Builder
	code := run([]string{"-in", in, "-out", filepath.Join(dir, "out.go"), "-package", "bad-name", "-format", "json"}, &stderr)
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
	if report.ExitCode != 2 || len(report.Errors) != 1 || report.Errors[0].Code != core.ErrorCodeInvalidConfig {
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
