# Migrating From v1 To v2

This guide covers the current v2 migration surface. It does not imply
CervoProxy has migrated.

## Import Path

Change CervoRules imports from the v1 module path:

```go
import cervorules "github.com/cervantesh/cervo-rules"
```

to the v2 module path:

```go
import cervorules "github.com/cervantesh/cervo-rules/v2"
```

The same rule applies to subpackages:

```go
import "github.com/cervantesh/cervo-rules/v2/testkit"
```

## Modular Imports

The v2 root package remains available as a compatibility facade:

```go
import cervorules "github.com/cervantesh/cervo-rules/v2"
```

New code should prefer narrower imports when it only needs one part of the
library:

```go
import (
    "github.com/cervantesh/cervo-rules/v2/core"
    "github.com/cervantesh/cervo-rules/v2/runtime"
    "github.com/cervantesh/cervo-rules/v2/limits"
    "github.com/cervantesh/cervo-rules/v2/httpadapter"
    "github.com/cervantesh/cervo-rules/v2/observe"
    "github.com/cervantesh/cervo-rules/v2/testkit"
)
```

Use `core` for decision-flow construction and evaluation, `runtime` for
`PolicyRuntimeConfig`, `limits` for requested-limit checks, `httpadapter` only at
HTTP boundaries, `observe` for reports/audit envelopes, and `testkit` in
consumer tests.

## Generated Policy Code

Regenerate v2 policy code with v2 tools so generated files import
the modular runtime packages:

- `github.com/cervantesh/cervo-rules/v2/core`;
- `github.com/cervantesh/cervo-rules/v2/runtime`;
- `github.com/cervantesh/cervo-rules/v2/limits`.

Regenerate vocabulary code with the v2 `cervorules-vocabgen` as well. Its
default runtime import is `github.com/cervantesh/cervo-rules/v2/core`, so new
generated vocabulary packages no longer need the root `/v2` facade unless a
consumer explicitly passes `-import github.com/cervantesh/cervo-rules/v2`.

```powershell
go run github.com/cervantesh/cervo-rules/v2/cmd/cervorules-policygen generate `
  -vocab policy-vocabulary.yaml `
  -policy policy-rules.yaml `
  -out internal/policyrules/generated_policy.go `
  -test-out internal/policyrules/generated_policy_test.go `
  -package policyrules `
  -vocab-package policyvocab `
  -vocab-import your/module/internal/policyvocab
```

The generated vocabulary package is still owned by the consuming application.
Only the CervoRules runtime and tool imports move to `/v2`.

## Coexistence

v1 remains available through v1 release tags and package artifacts. Existing v1
consumers can stay on those tags while v2 consumers adopt
`github.com/cervantesh/cervo-rules/v2`.

Do not mix v1 and v2 CervoRules packages in the same generated policy package.
Migrate one consumer module at a time, regenerate policy code, then run that
consumer's tests.

## Phase 2: Structured Validation Errors

Validation failures use structured errors with stable fields:

```go
type ValidationError struct {
    Code   string
    Field  string
    Value  string
    Reason string
}

type ValidationErrors []ValidationError
```

Do not parse validation error strings in v2 consumers. Switch string matching to
field checks against `Code`, `Field`, `Value`, and `Reason`.

```go
if validationErrors, ok := err.(cervorules.ValidationErrors); ok {
    for _, item := range validationErrors {
        switch item.Code {
        case "unknown_capability":
            log.Printf("invalid policy field %s=%q: %s", item.Field, item.Value, item.Reason)
        default:
            log.Printf("policy validation failed at %s: %s", item.Field, item.Reason)
        }
    }
}
```

## Phase 2: Limit Violations

`CheckLimits` now returns `LimitViolations` instead of `error`:

```go
violations := cervorules.CheckLimits(decision.Limits, cervorules.RequestedLimits{
    BodyBytes: int64(len(body)),
    MaxTokens: requestedMaxTokens,
    Stream:    streamRequested,
    Tools:     toolsRequested,
    Images:    imagesRequested,
})
if len(violations) > 0 {
    status := http.StatusForbidden
    if violations.Has("body_bytes") {
        status = http.StatusRequestEntityTooLarge
    }
    http.Error(w, violations.Error(), status)
    return
}
```

Map `body_bytes` violations to HTTP `413`. Map other limit fields to HTTP
`403`. If a request has both `body_bytes` and other violations, prefer `413`
for the response status and keep the full `LimitViolations` value for logs or
diagnostics.

## Phase 3: DecisionResult Runtime Envelope

`Engine.Decide` returns `DecisionResult` in v2:

```go
result, err := engine.Decide(ctx, req)
if err != nil {
    return err
}
decision := result.Decision
request := result.Request
trace := result.Trace
observation := result.Observation
diagnostics := result.Diagnostics
```

`DecisionResult` carries the original request facts needed by observability
adapters. It embeds `Decision`, so existing decision field reads can be migrated
incrementally, but new code should prefer `result.Decision` to make the runtime
envelope explicit.

For short compatibility migrations, use `DecideDecision`:

```go
decision, err := cervorules.DecideDecision(engine, ctx, req)
```

Context cancellation is now explicit. If the context is already canceled or its
deadline has expired, `Decide` returns that context error instead of silently
producing an allow/deny decision.

Runtime inspection helpers are available for health checks, diagnostics, and
docs tooling:

```go
summary := engine.Inspect()
graph := engine.GraphDOT()
```

`Inspect` returns bounded counts. `GraphDOT` renders phase and rule structure
only; it does not include request metadata, users, or decision reasons.

## Phase 4: Context And Error Aware Predicates And Actions

Custom predicates now receive `context.Context` and can return errors:

```go
type Predicate interface {
    Evaluate(context.Context, cervorules.EvalContext) (cervorules.PredicateResult, error)
}
```

Custom actions now receive `context.Context`, the current `EvalContext`, and the
decision mutator:

```go
type Action interface {
    Apply(context.Context, cervorules.EvalContext, cervorules.DecisionMutator) error
    Name() string
}
```

Use adapters for simple v1-style implementations:

```go
predicate := cervorules.PredicateFunc("TenantInternal", func(eval cervorules.EvalContext) cervorules.PredicateResult {
    return cervorules.PredicateResult{Matched: eval.Request.Metadata["tenant"] == "internal"}
})

action := cervorules.ActionFunc("AllowInternal", func(mutator cervorules.DecisionMutator) error {
    mutator.Allow()
    return nil
})
```

Predicate or action errors stop evaluation and are returned to the caller. If a
context is canceled during evaluation, the current loop exits with that context
error.

## Phase 5: Generated Policy Factory

Generated policy packages now expose a factory API:

```go
factory := policyrules.NewPolicyFactory()
cfg := factory.DefaultConfig()
engine, err := factory.Build(ctx, cfg)
metadata := factory.Metadata()
```

Use `factory.ValidateConfig(cfg)` before build when config is loaded from env or
files and you want a validation-only startup check.

The legacy wrapper remains for one migration cycle:

```go
// Deprecated: use NewPolicyFactory().Build(ctx, cfg).
engine, err := policyrules.BuildPolicy(overrides...)
```

Prefer the factory in new consumers because it makes default config, validation,
build context, and generated metadata explicit.

## Phase 6: Testkit Error-First API

`testkit` now exposes an error-first generated policy contract checker:

```go
err := testkit.CheckGeneratedRuntimePolicy(testkit.GeneratedRuntimePolicyContract{
    BuildPolicy:     policyrules.BuildPolicy,
    DefaultConfig:   policyrules.DefaultRuntimeConfig(),
    ValidOverride:   validOverride,
    InvalidOverride: invalidOverride,
    Request:         request,
})
```

Use the `Must...` wrapper at test boundaries:

```go
testkit.MustAssertGeneratedRuntimePolicy(t, contract)
```

The old assertion-first helper remains for one migration cycle:

```go
// Deprecated: use MustAssertGeneratedRuntimePolicy or CheckGeneratedRuntimePolicy.
testkit.AssertGeneratedRuntimePolicy(t, contract)
```

For CI or smoke tooling that needs a serializable result:

```go
scorecard := testkit.GeneratedRuntimePolicyReadiness(contract)
if !scorecard.Ready() {
    return errors.New(scorecard.Summary())
}
```

## Phase 7: Vocabulary Interface

`Vocabulary` is now an interface so consumers can provide dynamic validators:

```go
type Vocabulary interface {
    ValidateCapability(cervorules.Capability) error
    ValidateService(cervorules.Service) error
    ValidateProvider(cervorules.Provider) error
    ValidateRequest(cervorules.Request) error
    ValidateDecision(cervorules.Decision) error
}
```

The fixed-set implementation is `StaticVocabulary`. Most consumers should keep
using the constructor:

```go
vocab := cervorules.NewVocabulary(
    cervorules.AllowedCapabilities("read", "write"),
    cervorules.AllowedServices("reader"),
    cervorules.AllowedProviders("primary"),
)
```

Generated vocabulary packages still expose:

```go
func Vocabulary() cervorules.Vocabulary
```

If v1 code used a `cervorules.Vocabulary` struct literal, replace it with
`cervorules.NewVocabulary(...)` or `cervorules.StaticVocabulary{}`. Prefer the
constructor unless you intentionally need an open static vocabulary.

## Phase 8: Runtime Config Options

`PolicyRuntimeConfig` still has public fields in v2.0, but new code should
prefer option constructors for defensive copies and clearer startup wiring:

```go
cfg := cervorules.NewPolicyRuntimeConfig(
    cervorules.WithTrustedUsers("admin", "ops"),
    cervorules.WithDefaultTimeout(5*time.Second),
    cervorules.WithDefaultProvider("primary"),
    cervorules.WithRetryPolicy(cervorules.RetryPolicy{MaxAttempts: 2}),
    cervorules.WithBreakerPolicy(cervorules.BreakerPolicy{FailureThreshold: 3}),
    cervorules.WithLimits(cervorules.Limits{MaxTokens: 1000}),
    cervorules.WithCapabilityRoute("read", "reader"),
    cervorules.WithProviderFallbacks("primary", "backup"),
)
```

Options compose with existing merge behavior:

```go
cfg := cervorules.MergePolicyRuntimeConfig(
    factory.DefaultConfig(),
    cervorules.NewPolicyRuntimeConfig(
        cervorules.WithDefaultProvider("backup"),
    ),
)
```

The options copy slices, maps, and policy structs before storing them in the
config. That keeps env/file parsing code from mutating the policy startup
configuration after construction.

## Phase 9: Consumer Adoption Checklist

Before migrating a real consumer such as a gateway, prove the v2 contracts in a
consumer-shaped test module:

1. Generate vocabulary and policy code with v2 tools.
2. Build the policy through `NewPolicyFactory().Build(ctx, cfg)`.
3. Convert transport input through an adapter such as `NewHTTPClassifier`.
4. Call `engine.Decide(ctx, request)` and read `result.Decision`,
   `result.Trace`, and `result.Observation`.
5. Convert payload-specific resource usage to `RequestedLimits` and call
   `CheckLimits`.
6. Exercise disabled-by-default routes with `WithCapabilityRoute`.
7. Keep one generated-policy contract test using `testkit.CheckGeneratedRuntimePolicy`
   or `testkit.MustAssertGeneratedRuntimePolicy`.

The repository runs a gateway-shaped fixture in
`TestGatewayShapedV2FixtureUsesGeneratedFactoryHTTPAndLimits`. It intentionally
uses neutral names while covering the same migration surface a CervoProxy-style
consumer needs: HTTP classification, generated factory build, runtime options,
`DecisionResult`, disabled route enablement, and limit checks.
