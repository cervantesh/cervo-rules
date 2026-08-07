package observe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
)

// schemas/v3/policy-evaluation-report.schema.json is published as a contract:
// operators build dashboards and alerts against it, and it is shipped in the
// release artifacts. Nothing checked that the report this package produces
// still matches it.
//
// That is the same shape of gap the generated-policy-metadata schema had, and
// this follows the same pattern: read the schema's own required keys, consts
// and types, and hold the produced document to them. A published contract with
// a producer and no check between them drifts on the first field rename.
func TestPolicyEvaluationReportMatchesItsSchema(t *testing.T) {
	schema := readReportSchema(t)

	cases := map[string]core.DecisionResult{
		// The zero decision: `omitempty` strips almost everything, so this is
		// the document at its smallest. It must still carry what is required.
		"empty": {},
		"full": {
			Decision: core.Decision{
				Allow:    true,
				Target:   core.Target("some_target"),
				Executor: core.Executor("some_executor"),
				Reason:   "matched a rule",
			},
			Diagnostics: core.Errors{{}, {}},
			Observation: &core.DecisionObservation{
				RequestID: "req-1",
				Operation: core.Operation("some_operation"),
			},
		},
	}

	for name, result := range cases {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(NewPolicyEvaluationReport(result))
			if err != nil {
				t.Fatalf("marshal report: %v", err)
			}
			var document map[string]any
			if err := json.Unmarshal(raw, &document); err != nil {
				t.Fatalf("unmarshal report: %v", err)
			}

			for _, key := range schema.Required {
				if _, ok := document[key]; !ok {
					t.Errorf("required key %q is missing from the produced report", key)
				}
			}
			for key, value := range document {
				rule, ok := schema.Properties[key]
				if !ok {
					// The schema sets additionalProperties: false, so a new
					// field on the struct breaks every consumer validating
					// against it.
					t.Errorf("produced key %q is not allowed by the schema", key)
					continue
				}
				if rule.Const != nil && value != rule.Const {
					t.Errorf("%s: got %v want const %v", key, value, rule.Const)
				}
				if rule.Type != "" && !matchesJSONType(value, rule.Type) {
					t.Errorf("%s: %v (%T) is not a JSON %s", key, value, value, rule.Type)
				}
			}
		})
	}
}

type reportSchema struct {
	Required   []string `json:"required"`
	Properties map[string]struct {
		Const any    `json:"const"`
		Type  string `json:"type"`
	} `json:"properties"`
}

func readReportSchema(t *testing.T) reportSchema {
	t.Helper()

	path := filepath.Join("..", "schemas", "v3", "policy-evaluation-report.schema.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema reportSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	if len(schema.Required) == 0 || len(schema.Properties) == 0 {
		t.Fatalf("schema at %s declares no required keys or no properties; "+
			"this test would then assert nothing", path)
	}
	return schema
}

// matchesJSONType reports whether a value decoded from JSON has the type the
// schema declares. Numbers decode as float64, so "integer" checks the value is
// whole rather than checking the Go type.
func matchesJSONType(value any, want string) bool {
	switch want {
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == float64(int64(number))
	case "number":
		_, ok := value.(float64)
		return ok
	default:
		return true
	}
}
