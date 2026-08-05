# Package Minimal Examples

These examples are intentionally small. They help agents choose the narrowest
package import without reading the whole repository.

## core

Use `core` to build and evaluate deterministic decisions.

```go
plan := core.NewIndexedRoutingPlan(
    core.RoutingRule{
        Operation: core.Operation("invoice.read"),
        Target:    core.Target("billing-read"),
        Executor:  core.Executor("standard"),
    },
)
_ = plan

result, err := compiled.DecideWithOptions(ctx, core.Request{
    Operation: core.Operation("invoice.read"),
}, core.DecisionOptions{
    Trace:       core.DecisionOptionDisabled,
    Observation: core.DecisionOptionDisabled,
})
```

## runtime

Use `runtime` to consume generated policy factories and startup configuration.

```go
cfg := runtime.PolicyRuntimeConfig{
    DefaultExecutor: core.Executor("standard"),
    OperationTargets: map[core.Operation]core.Target{
        core.Operation("invoice.read"): core.Target("billing-read"),
    },
}
factory := policyrules.NewPolicyFactory()
engine, err := factory.Build(ctx, cfg)
```

## limits

Use `limits` when the consumer has already extracted requested usage.

```go
violations := limits.CheckLimits(limits.Limits{MaxTokens: 2048}, limits.RequestedLimits{
    MaxTokens: 4096,
})
if violations.Has("max_tokens") {
    return violations
}
```

## facts

Use `facts` for bounded derived facts, not as unbounded request-path storage.

```go
engine, err := facts.NewEngine([]facts.Rule{rule}, facts.EngineOptions{
    MaxFacts:      1000,
    MaxIterations: 20,
    MaxBindings:   10000,
})
if err != nil {
    return err
}
result, err := engine.Evaluate(ctx, baseFacts, facts.EvalOptions{
    TraceMode: facts.TraceDisabled,
})
```

## httpadapter

Use `httpadapter` only at transport boundaries.

```go
classifier, err := httpadapter.NewHTTPClassifier(httpadapter.HTTPClassificationOptions{
    RequestIDHeaders: []string{"X-Request-ID"},
    UserHeaders:      []string{"X-User"},
    OmitHeaders:      true,
})
if err != nil {
    return err
}
facts := classifier.FactsFromHTTPRequest(req)
```

## observe

Use `observe` for stack-neutral operational reporting.

```go
report := observe.PolicyEvaluationReport(result.Observation)
fields := report.LogFields()
labels := report.MetricLabels()
```

## testkit

Use `testkit` to certify generated policies or consumer-shaped fixtures.

```go
func TestGeneratedPolicyContract(t *testing.T) {
    testkit.MustAssertGeneratedRuntimePolicy(t, contract)
}
```

## decisioncache

Use `decisioncache` only when the caller owns the key and knows the decision is
pure for that key.

```go
cached := decisioncache.New(engine, decisioncache.NewMemoryStore(time.Minute), decisioncache.RequestKey)
result, err := cached.Decide(ctx, req)
```
