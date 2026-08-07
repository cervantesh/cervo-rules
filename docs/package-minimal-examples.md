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
}, core.NewDecisionOptions(
    core.WithTrace(false),
    core.WithObservation(false),
))
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
violations := limits.Check(limits.Budget{MaxTokens: 2048}, limits.Requested{
    MaxTokens: 4096,
})
if violations.Has("max_tokens") {
    return violations
}
```

## facts

Use `facts` for bounded derived facts, not as unbounded request-path storage.

`facts` ships contracts, not an engine: `Fact`, `Pattern`, `Term`, `Result`,
`EvalOptions` and the planning and complexity types. The consumer brings the
evaluator and implements the shape `testkit.FactsEvaluator` describes.

Budgets are required rather than defaulted, so an unbounded evaluation is not
something you can reach by forgetting a field:

```go
options := facts.EvalOptions{
    MaxIterations: 20,
    MaxFacts:      1000,
    MaxBindings:   10000,
    Trace:         facts.TraceDisabled,
}.Normalize()

result, err := evaluator.Evaluate(ctx, options)
```

This section previously showed a constructor and a rule type that the package
has never exported.

## httpadapter

Use `httpadapter` only at transport boundaries.

```go
classifier, err := httpadapter.NewClassifier(httpadapter.HTTPClassificationOptions{
    Headers: httpadapter.HeaderOptions{
        RequestID: []string{"X-Request-ID"},
        User:      []string{"X-User"},
    },
})
if err != nil {
    return err
}
facts := classifier.FactsFromHTTPRequest(req)
```

## observe

Use `observe` for stack-neutral operational reporting.

```go
report := observe.NewPolicyEvaluationReport(result)
fields := report.LogFields()
labels := report.MetricLabels()
```

## testkit

Use `testkit` to certify generated policies or consumer-shaped fixtures.

```go
func TestGeneratedPolicyContract(t *testing.T) {
    testkit.MustAssertConsumerConformance(t, contract)
}
```

## decisioncache

Reserved. The package exists as a module marker and exports no cache contracts
yet, so there is nothing to import beyond `ModulePath`.

Caching a decision is safe in principle — `Decide` is pure and executes
nothing — but the key, the invalidation rule and the ownership of a distributed
cache are all caller decisions, and none of them has been designed here. Cache
in the consumer until this package has an API.

This section previously showed a constructor with a memory store and a
request-key function. None of that existed.
