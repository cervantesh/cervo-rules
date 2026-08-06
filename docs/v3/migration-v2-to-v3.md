# Migration Guide: v2 To v3

Issues: #398, #399, #400, #401, #402, #403.

v3 is a breaking API line. It removes compatibility names and wrappers that v2
kept for adoption safety. The migration target is explicit, modular, and easier
for machines to inspect.

## Status

- v2 final split release: `v2.1.0`.
- v3 current release candidate: `v3.0.0-rc.5`.
- v3 module path: `github.com/cervantesh/cervo-rules/v3`.
- v3 is repo-ready for RC package verification, but not GA final.

## Mechanical Replacements

| v2 name | v3 name | Notes |
| --- | --- | --- |
| `Capability` | `Operation` | What the request is trying to do. |
| `Service` | `Target` | Logical destination selected by policy. |
| `Provider` | `Executor` | Caller-owned execution choice. |
| `BuildPolicy(...)` | `NewPolicyFactory().Build(ctx, cfg)` | v3 generated policies use only the factory. |
| `RoutingPhase(...)` | `NewIndexedRoutingPlan(...)` or `NewLinearRoutingPlan(...)` | v3 makes O(n) routing explicit. |
| YAML `capability` | YAML `operation` | v3 DSL schema field. |
| YAML `service` | YAML `target` | v3 DSL schema field. |
| YAML `provider` | YAML `executor` | v3 DSL schema field. |

Native v3 migration reporting is deferred. Use the manual replacement table
below, or use the v2 maintenance repository for historical migration tooling.

## Before And After: Primitives

v2:

```go
var capRead cervorules.Capability = "invoice.read"
var svcLedger cervorules.Service = "billing-ledger"
var providerPrimary cervorules.Provider = "ledger-primary"
```

v3:

```go
var opRead core.Operation = "invoice.read"
var targetLedger core.Target = "billing-ledger"
var executorPrimary core.Executor = "ledger-primary"
```

## Before And After: Generated Policy

v2:

```go
engine, err := policyrules.BuildPolicy(overrides...)
```

v3:

```go
factory := policyrules.NewPolicyFactory()
cfg := factory.DefaultConfig()
if err := factory.ValidateConfig(cfg); err != nil {
    return err
}
engine, err := factory.Build(ctx, cfg)
```

## Before And After: Decision Runtime

v3 keeps decision output in `core.DecisionResult` and makes operational detail
explicit:

```go
result, err := engine.DecideWithOptions(ctx, req, core.WithTrace(), core.WithObservation())
if err != nil {
    return err
}
decision := result.Decision
```

Fast paths can omit trace and observation:

```go
result, err := engine.Decide(ctx, req)
```

## Before And After: Facts

v3 facts are optional contracts. Per-request use must set budgets:

```go
options := facts.EvalOptions{
    MaxIterations: 4,
    MaxFacts:      128,
    MaxBindings:   512,
}
```

Trace is opt-in:

```go
options.Trace = facts.TraceEnabled
```

## YAML Migration

v2:

```yaml
routes:
  - capability: invoice.read
    service: billing-ledger
    provider: ledger-primary
```

v3:

```yaml
routes:
  - operation: invoice.read
    target: billing-ledger
    executor: ledger-primary
```

Validate v3 schema files before generating code:

```bash
cervorules-policygen check -policy policy-rules.yaml -vocab policy-vocabulary.yaml
```

Native v3 parser/codegen is available through `cmd/cervorules-policygen` and
`cmd/cervorules-vocabgen`.

## Conformance

After migrating a generated consumer, add a testkit contract:

```go
testkit.MustAssertConsumerConformance(t, testkit.ConsumerConformanceContract{
    Name:           "billing-routing",
    PolicyPath:     "policy-rules.yaml",
    VocabularyPath: "policy-vocabulary.yaml",
    RuntimeCases: []testkit.RuntimeCase{{
        Name:         "factory-decision",
        Factory:      policyrules.NewPolicyFactory(),
        Config:       policyrules.NewPolicyFactory().DefaultConfig(),
        Request:      core.Request{Operation: "invoice.read"},
        WantDecision: core.Decision{Allow: true, Target: "billing-ledger"},
    }},
})
```

## Release Verification

For v3 release candidates, release packages must include:

- `release_module=github.com/cervantesh/cervo-rules/v3`;
- `release-dependencies.txt`;
- root schemas and `schemas/v3/*.schema.json`;
- `artifact-manifest.json`;
- `sbom-modules.json`;
- `sbom-spdx.json`;
- `provenance.json`;
- `checksums.txt`;
- optional `checksums.txt.minisig`;
- v3-native `cervorules-policygen` and `cervorules-vocabgen` tool archives.

Verification commands:

```bash
scripts/release/check.sh v3.0.0-rc.5 dist-release-check-v3.0.0-rc.5
scripts/release/verify-generic-package.sh v3.0.0-rc.5
scripts/release/verify-oci-tools.sh v3.0.0-rc.5
```
