# CervoRules

CervoRules is an independent Go library for deterministic policy decisions.

It is a decision engine, not a proxy, scheduler, model selector, transport
parser, logger, metrics backend, or data plane.

The current breaking-line release candidate is `v3.0.0-rc.3`. New projects
should evaluate the v3 module path:

```go
import "github.com/cervantesh/cervo-rules/v3/core"
```

v3 uses neutral public primitives:

- `Operation`: what the request wants to do;
- `Target`: the logical destination selected by policy;
- `Executor`: the caller-owned execution choice selected by policy.

Generated v3 policies should expose `runtime.PolicyFactory`. Deprecated v2
names and generated wrappers are documented only in migration/history material,
not in the current quick path.

Start with:

- [AGENTS.md](AGENTS.md) for agent workflow and repository boundaries;
- [docs/agent-quickstart.md](docs/agent-quickstart.md) for copyable commands;
- [.cervorules/agent-manifest.json](.cervorules/agent-manifest.json) for the
  machine-readable repo index;
- [docs/v3/api-reference.md](docs/v3/api-reference.md) for the v3 public API;
- [docs/error-handling.md](docs/error-handling.md) for structured errors,
  redaction, exit codes, and fail-closed guidance;
- [docs/v3/migration-v2-to-v3.md](docs/v3/migration-v2-to-v3.md) for legacy
  consumer migration;
- [docs/reports/v3-rc2.md](docs/reports/v3-rc2.md) for the current maturity,
  agnosticism, dependency, smell, performance, and release evidence report.

## Mental Model

```text
Request -> Engine -> DecisionResult -> Decision
```

- `Request` is the normalized input supplied by the consumer.
- `Engine` evaluates a compiled/generated policy.
- `DecisionResult` carries the decision plus optional trace, observation,
  diagnostics, and stats.
- `Decision` is the final structured answer that the caller executes.

## What CervoRules Does

- Evaluates deterministic policy decisions.
- Keeps domain vocabulary caller-owned.
- Provides explicit routing plans with visible cost.
- Provides structured validation errors.
- Provides optional bounded facts evaluation.
- Provides stack-neutral observability contracts.
- Provides generated-policy conformance and release/package verification hooks.

## What CervoRules Does Not Do

- It does not forward HTTP or gRPC traffic.
- It does not perform retries or store breaker state.
- It does not inspect live upstream health by itself.
- It does not choose models, profiles, tenants, or application-owned names.
- It does not own logging, tracing, metrics, audit storage, or package hosting.

## Modular Imports

Use the narrow package that owns the behavior you need:

| Package | Use when |
| --- | --- |
| `github.com/cervantesh/cervo-rules/v3/core` | Building or evaluating decision requests, results, routing plans, errors, and vocabulary contracts. |
| `github.com/cervantesh/cervo-rules/v3/runtime` | Consuming generated `PolicyFactory` values and runtime config. |
| `github.com/cervantesh/cervo-rules/v3/limits` | Checking caller-requested usage against policy limits. |
| `github.com/cervantesh/cervo-rules/v3/facts` | Running bounded derived-fact evaluation. |
| `github.com/cervantesh/cervo-rules/v3/httpadapter` | Keeping HTTP classification at the transport boundary. |
| `github.com/cervantesh/cervo-rules/v3/observe` | Producing stack-neutral operational report contracts. |
| `github.com/cervantesh/cervo-rules/v3/testkit` | Certifying generated consumers and fixtures. |

The v3 root package is a module marker, not a compatibility facade. See
[docs/v3/modular-boundaries.md](docs/v3/modular-boundaries.md).

## Minimal Runtime Shape

```go
package policy

import (
    "context"

    "github.com/cervantesh/cervo-rules/v3/core"
)

func Decide(ctx context.Context, engine core.Engine) (core.Decision, error) {
    result, err := engine.DecideWithOptions(ctx, core.Request{
        ID:        "req-1",
        User:      "operator",
        Operation: core.Operation("invoice.read"),
    }, core.NewDecisionOptions(
        core.WithTrace(false),
        core.WithObservation(false),
    ))
    if err != nil {
        return core.Decision{}, err
    }
    return result.Decision, nil
}
```

## Routing Plans

v3 makes routing cost explicit:

```go
plan := core.NewIndexedRoutingPlan(
    core.RoutingRule{
        Operation: core.Operation("invoice.read"),
        Target:    core.Target("billing-reader"),
        Executor:  core.Executor("standard"),
    },
)
```

Use indexed routing for normal route tables. Use `NewLinearRoutingPlan` only
when global rule order is intentionally part of policy behavior and the rule
count is small enough for `O(n)` evaluation.

## Generated Policy Factory

Generated v3 policies should be consumed through a factory:

```go
factory := policyrules.NewPolicyFactory()
cfg := factory.DefaultConfig()
if err := factory.ValidateConfig(cfg); err != nil {
    return err
}
engine, err := factory.Build(ctx, cfg)
```

The factory exposes metadata for machine inspection and release tooling:

```go
metadata := factory.Metadata()
_ = metadata.PolicyHash
```

## Facts

The `facts` package is optional. Use it for bounded derived facts and
explanations, not as unbounded per-request storage. Always configure budgets
for request-path use:

```go
options := facts.EvalOptions{
    MaxFacts:      1000,
    MaxIterations: 20,
    MaxBindings:   10000,
    Trace:         facts.TraceDisabled,
}
```

See [docs/facts-complexity-control.md](docs/facts-complexity-control.md).

## HTTP And Other Adapters

HTTP support is an adapter, not the identity of the library. Keep transport
parsing in consuming applications or optional adapter packages. For non-HTTP
inputs, map queues, jobs, CLI commands, or device messages into `core.Request`.

See [docs/adapter-patterns.md](docs/adapter-patterns.md).

## Tooling

The active repository is v3-root and ships v3-native generator commands:

```powershell
go run ./cmd/cervorules-vocabgen `
  -in policy-vocabulary.yaml `
  -out internal/policyvocab/generated.go `
  -package policyvocab

go run ./cmd/cervorules-policygen check `
  -vocab policy-vocabulary.yaml `
  -policy policy-rules.yaml

go run ./cmd/cervorules-policygen generate `
  -vocab policy-vocabulary.yaml `
  -policy policy-rules.yaml `
  -out internal/policyrules/generated_policy.go `
  -test-out internal/policyrules/generated_policy_test.go `
  -package policyrules `
  -vocab-package policyvocab `
  -vocab-import your/module/internal/policyvocab
```

Use the v3 tooling for:

- `check` before generation;
- `generate` after policy review;
- generated tests beside the generated factory.

Legacy v2 migration tooling remains in
[CervoSoft/cervo-rules-v2](https://github.com/cervantesh/cervo-rules-v2).

## Release And Packages

`v3.0.0-rc.3` is the clean post-fix v3 RC. It verified:

- package publishing workflow;
- generic package artifacts;
- OCI tools image;
- checksums;
- schemas;
- SBOM/provenance;
- dependency manifests.

Package consumption commands and verification steps live in
[docs/packages.md](docs/packages.md).

## Best Practices

- Prefer v3 modular imports for new work.
- Keep application vocabulary in the consuming project.
- Compile/build policy once at startup and reuse the runtime engine.
- Keep facts bounded and outside hot paths unless benchmark evidence supports
  request-path evaluation.
- Keep transport adapters outside `core`.
- Use change management and TDD for public API changes.


## License

MIT. See [LICENSE](LICENSE).
