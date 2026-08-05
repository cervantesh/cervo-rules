package httpadapter

import (
	"net/http"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/core"
)

func TestClassifierBuildsFactsFromHeadersPathAndChannel(t *testing.T) {
	classifier, err := NewClassifier(HTTPClassificationOptions{
		Headers: HeaderOptions{
			RequestID: []string{"X-Request-Id"},
			User:      []string{"X-User"},
			Channel:   []string{"X-Channel"},
			Operation: []string{"X-Operation"},
			Risk:      []string{"X-Risk"},
		},
		PathOperations: []PathOperation{
			{Methods: []string{"POST"}, Prefix: "/v1/items", Operation: "item.create"},
			{Regex: `/(secret|vault)(/|$)`, Operation: "secret.read"},
		},
		ChannelOperations: map[string]core.Operation{
			"events": "event.route",
		},
		CopyHeaders: true,
	})
	if err != nil {
		t.Fatalf("new classifier: %v", err)
	}

	req, _ := http.NewRequest(http.MethodPost, "/v1/items/123", nil)
	req.Header.Set("X-Request-Id", "req-1")
	req.Header.Set("X-User", "ada")
	req.Header.Set("X-Channel", "api")
	req.Header.Set("X-Risk", "admin")
	facts := classifier.FactsFromHTTPRequest(req)
	if facts.ID != "req-1" || facts.User != "ada" || facts.Channel != "api" || facts.Risk != "admin" || facts.Operation != "item.create" {
		t.Fatalf("unexpected facts: %#v", facts)
	}
	if facts.Headers.Get("X-User") != "ada" {
		t.Fatalf("expected copied headers, got %#v", facts.Headers)
	}

	req, _ = http.NewRequest(http.MethodGet, "/internal/vault/read", nil)
	if got := classifier.FactsFromHTTPRequest(req).Operation; got != "secret.read" {
		t.Fatalf("expected regex operation, got %q", got)
	}

	req, _ = http.NewRequest(http.MethodGet, "/unknown", nil)
	req.Header.Set("X-Channel", "events")
	if got := classifier.FactsFromHTTPRequest(req).Operation; got != "event.route" {
		t.Fatalf("expected channel operation, got %q", got)
	}
}

func TestClassifierHeaderOperationWinsAndNoCopyFastPath(t *testing.T) {
	classifier, err := NewClassifier(HTTPClassificationOptions{
		Headers: HeaderOptions{Operation: []string{"X-Operation"}},
		PathOperations: []PathOperation{
			{Prefix: "/v1/items", Operation: "item.create"},
		},
	})
	if err != nil {
		t.Fatalf("new classifier: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, "/v1/items", nil)
	req.Header.Set("X-Operation", "item.override")
	facts := classifier.FactsFromHTTPRequest(req)
	if facts.Operation != "item.override" {
		t.Fatalf("expected header operation to win, got %q", facts.Operation)
	}
	if facts.Headers != nil {
		t.Fatalf("expected no header copy by default")
	}
}

func TestClassifierRejectsInvalidRegexAndHandlesNil(t *testing.T) {
	if _, err := NewClassifier(HTTPClassificationOptions{PathOperations: []PathOperation{{Regex: "["}}}); err == nil {
		t.Fatalf("expected invalid regex error")
	}
	var classifier *Classifier
	if facts := classifier.FactsFromHTTPRequest(nil); facts.Operation != "" {
		t.Fatalf("expected empty facts for nil classifier/request, got %#v", facts)
	}
}

func TestRequestFactsConvertsToCoreRequest(t *testing.T) {
	facts := RequestFacts{
		ID:        "req-1",
		User:      "ada",
		Channel:   "api",
		Risk:      "elevated",
		Operation: "item.create",
	}
	req := facts.Request()
	if req.ID != "req-1" || req.User != "ada" || req.Channel != "api" || req.Operation != "item.create" {
		t.Fatalf("unexpected request: %#v", req)
	}
	if req.Metadata["risk"] != "elevated" {
		t.Fatalf("expected risk metadata, got %#v", req.Metadata)
	}

	facts.Risk = ""
	req = facts.Request()
	if req.Metadata != nil {
		t.Fatalf("empty risk should not allocate metadata, got %#v", req.Metadata)
	}
}

func TestClassifierPathRulesCanMissBeforeMatch(t *testing.T) {
	classifier, err := NewClassifier(HTTPClassificationOptions{
		PathOperations: []PathOperation{
			{Methods: []string{"POST"}, Prefix: "/v1/items", Operation: "item.create"},
			{Contains: "/reports/", Operation: "report.read"},
			{Regex: `/devices/[0-9]+$`, Operation: "device.read"},
		},
	})
	if err != nil {
		t.Fatalf("new classifier: %v", err)
	}

	cases := []struct {
		name string
		req  *http.Request
		want core.Operation
	}{
		{name: "method miss then no match", req: newRequest(t, http.MethodGet, "/v1/items"), want: ""},
		{name: "contains miss then regex match", req: newRequest(t, http.MethodGet, "/devices/42"), want: "device.read"},
		{name: "contains match", req: newRequest(t, http.MethodGet, "/tenant/reports/summary"), want: "report.read"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifier.FactsFromHTTPRequest(tc.req).Operation; got != tc.want {
				t.Fatalf("operation=%q want=%q", got, tc.want)
			}
		})
	}
}

func newRequest(t *testing.T, method, path string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}
