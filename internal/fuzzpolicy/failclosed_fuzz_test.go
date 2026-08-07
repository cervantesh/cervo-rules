package fuzzpolicy

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	cervorules "github.com/cervantesh/cervo-rules/v3/core"
	cervoruntime "github.com/cervantesh/cervo-rules/v3/runtime"

	"github.com/cervantesh/cervo-rules/v3/internal/fuzzpolicy/vocab"
)

// The invariant: a fact this policy cannot evaluate must fail the decision with
// a structured error. It must never be reported as a predicate that did not
// hold, because that reads as "the guard ran and found nothing wrong" and lands
// the request on the allow path.
//
// This class has failed twice. In a consumer, a `> 0` guard dropped both -99
// and NaN and let a declared default stand in for them, turning a refusal into
// an allow. In ontology, an individual recorded in two lifecycle states let a
// second terminal transition through. Both were found by someone constructing
// the case by hand, which is exactly the work a fuzzer does better.
//
// The assertion is not "it does not panic". It is: if a decision came back,
// then every fact the policy reads was genuinely parseable and in range. That
// makes the target hunt the specific shape that bit, rather than crashes.
//
// `go test` runs the seed corpus. To search, run:
//
//	go test ./internal/fuzzpolicy -run FuzzDecide -fuzz FuzzDecideNeverAllowsUnusableFacts
func FuzzDecideNeverAllowsUnusableFacts(f *testing.F) {
	engine, err := NewPolicyFactory().Build(context.Background(), cervoruntime.PolicyRuntimeConfig{})
	if err != nil {
		f.Fatalf("build policy: %v", err)
	}

	// Seeds: the healthy shape, then one per failure mode already known to be
	// interesting. "NaN" and "Inf" are here because ParseFloat accepts them and
	// a non-finite value passes every comparison without matching any.
	f.Add("dev", "12", "0.1", "1", "false")
	f.Add("production", "0", "0.0", "1", "true")
	f.Add("dev", "12", "NaN", "1", "false")
	f.Add("dev", "12", "Inf", "1", "false")
	f.Add("dev", "12", "-0", "1", "false")
	f.Add("dev", "9007199254740993", "0.1", "1", "false")
	f.Add("DEV", "12", "0.1", "1", "TRUE")
	f.Add("", "", "", "", "")

	f.Fuzz(func(t *testing.T, environment, queueDepth, errorRate, errorBudget, changeFreeze string) {
		metadata := map[string]string{
			vocab.FactEnvironment:    environment,
			vocab.FactQueueDepth:     queueDepth,
			vocab.FactErrorRatePct:   errorRate,
			vocab.FactErrorBudgetPct: errorBudget,
			vocab.FactChangeFreeze:   changeFreeze,
		}
		result, err := engine.Decide(context.Background(), cervorules.Request{
			Operation: vocab.OperationMessageCreate,
			Metadata:  metadata,
		})

		if err != nil {
			// A refusal is fine, but it has to be answerable: a caller decides
			// what to do from the code, not from the prose.
			var structured cervorules.Error
			if !errors.As(err, &structured) {
				t.Fatalf("decision failed with an unstructured error %T: %v", err, err)
			}
			switch structured.Code {
			case cervorules.ErrorCodeMissingFact, cervorules.ErrorCodeInvalidFact:
			default:
				t.Fatalf("unexpected error code %q for metadata %#v", structured.Code, metadata)
			}
			return
		}

		// It decided. Every fact it reads therefore had to be usable.
		if _, ok := parseEnum(environment, "dev", "staging", "production"); !ok {
			t.Fatalf("decided with an environment outside its declared domain: %q (allow=%v)",
				environment, result.Decision.Allow)
		}
		requireInteger(t, "queue_depth", queueDepth, 0, math.MaxInt64)
		requireNumber(t, "error_rate_pct", errorRate, 0, 100)
		// error_budget_pct declares a default, so absence is a policy statement
		// and only a present-but-unusable value is a defect.
		if strings.TrimSpace(errorBudget) != "" {
			requireNumber(t, "error_budget_pct", errorBudget, 0, 100)
		}
		requireBool(t, "change_freeze", changeFreeze)
	})
}

func parseEnum(raw string, allowed ...string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	for _, candidate := range allowed {
		if value == candidate {
			return value, true
		}
	}
	return "", false
}

func requireNumber(t *testing.T, name, raw string, min, max float64) {
	t.Helper()
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		t.Fatalf("decided while %s was blank; a blank fact counts as absent", name)
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		t.Fatalf("decided while %s was unparseable: %q", name, raw)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		t.Fatalf("decided while %s was non-finite: %q", name, raw)
	}
	if value < min || value > max {
		t.Fatalf("decided while %s was outside [%v,%v]: %q", name, min, max, raw)
	}
}

func requireInteger(t *testing.T, name, raw string, min, max int64) {
	t.Helper()
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		t.Fatalf("decided while %s was blank; a blank fact counts as absent", name)
	}
	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		t.Fatalf("decided while %s was unparseable as an integer: %q", name, raw)
	}
	if value < min || value > max {
		t.Fatalf("decided while %s was outside [%d,%d]: %q", name, min, max, raw)
	}
}

func requireBool(t *testing.T, name, raw string) {
	t.Helper()
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		t.Fatalf("decided while %s was blank; a blank fact counts as absent", name)
	}
	if _, err := strconv.ParseBool(trimmed); err != nil {
		t.Fatalf("decided while %s was unparseable as a bool: %q", name, raw)
	}
}
