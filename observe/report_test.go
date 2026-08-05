package observe

import (
	"encoding/json"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
)

func TestPolicyEvaluationReportIsVersionedAndLowCardinality(t *testing.T) {
	result := core.NewDecisionResult(
		core.Request{ID: "req-1", User: "sensitive", Operation: "read.item", Metadata: map[string]string{"tenant": "acme"}},
		core.Decision{Allow: true, Target: "queue", Executor: "primary"},
		core.WithObservation(true),
	)

	report := NewPolicyEvaluationReport(result)
	if report.SchemaVersion != PolicyEvaluationReportSchemaVersion {
		t.Fatalf("unexpected schema version: %#v", report)
	}

	labels := report.MetricLabels()
	for _, forbidden := range []string{"user", "tenant", "metadata", "request_id"} {
		if _, ok := labels[forbidden]; ok {
			t.Fatalf("metric labels must not include %q: %#v", forbidden, labels)
		}
	}
	if labels["outcome"] != "allow" || labels["operation"] != "read.item" {
		t.Fatalf("unexpected labels: %#v", labels)
	}
}

func TestPolicyEvaluationReportLogFieldsExcludeSensitiveMetadata(t *testing.T) {
	result := core.NewDecisionResult(
		core.Request{ID: "req-2", User: "sensitive", Operation: "write.item", Metadata: map[string]string{"tenant": "acme"}},
		core.Decision{Allow: false, Target: "archive", Executor: "primary", Reason: "blocked"},
		core.WithObservation(true),
	)

	fields := NewPolicyEvaluationReport(result).LogFields()
	for _, forbidden := range []string{"user", "metadata", "tenant"} {
		if _, ok := fields[forbidden]; ok {
			t.Fatalf("log fields must not include %q: %#v", forbidden, fields)
		}
	}
	if fields["request_id"] != "req-2" || fields["reason"] != "blocked" {
		t.Fatalf("unexpected log fields: %#v", fields)
	}
}

func TestPolicyEvaluationReportJSONIsMachineReadable(t *testing.T) {
	report := PolicyEvaluationReport{
		SchemaVersion: PolicyEvaluationReportSchemaVersion,
		RequestID:     "req-3",
		Operation:     "read.item",
		Allow:         true,
		Target:        "queue",
		Executor:      "primary",
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if decoded["schema_version"] != PolicyEvaluationReportSchemaVersion {
		t.Fatalf("schema version missing from JSON: %s", data)
	}
}
