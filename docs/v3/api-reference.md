# v3 API Reference

Issues: #398, #401.

This is the human companion to `docs/v3/public-api-inventory.json`.

## Module

```text
github.com/cervantesh/cervo-rules/v3
```

Prefer modular imports. The root module exists so packages share one major Go
module line; consumers should import the specific package they need.

## `core`

Package path:

```text
github.com/cervantesh/cervo-rules/v3/core
```

Primary contracts:

- `Operation`, `Target`, `Executor`;
- `Request`;
- `Decision`;
- `DecisionResult`;
- `DecisionOptions`;
- `Engine`;
- `Error`, `Errors`;
- `RoutingPlan`.

Use `core` for deterministic decisions and routing contracts. Do not put
transport parsing, model/profile selection, or consumer vocabulary ownership in
`core`.

## `runtime`

Package path:

```text
github.com/cervantesh/cervo-rules/v3/runtime
```

Primary contracts:

- `PolicyFactory`;
- `PolicyMetadata`;
- `PolicyRuntimeConfig`.

Generated v3 policies should expose a factory compatible with
`runtime.PolicyFactory`. `BuildPolicy` is intentionally not part of v3.

## `facts`

Package path:

```text
github.com/cervantesh/cervo-rules/v3/facts
```

Primary contracts:

- `EvalOptions`;
- `Result`;
- `EvaluationPlan`;
- `ComplexityDiagnostic`;
- `Fact`;
- `Pattern`.

Facts are optional. Consumers must set budgets for per-request use and should
keep trace disabled unless explanation is needed.

## `observe`

Package path:

```text
github.com/cervantesh/cervo-rules/v3/observe
```

Primary contracts:

- `PolicyEvaluationReport`;
- `MetricLabels`;
- `LogFields`.

Observation output is versioned and low-cardinality by default. User and metadata
values are not emitted by default.

## `testkit`

Package path:

```text
github.com/cervantesh/cervo-rules/v3/testkit
```

Primary contracts:

- `ConsumerConformanceContract`;
- `RuntimeCase`;
- `FactsCase`;
- `PackageSmoke`;
- `CheckConsumerConformance`;
- `MustAssertConsumerConformance`.

Use `testkit` to certify generated-policy consumers, neutral fixtures, facts
adapters, and package smoke expectations.

## Schemas

v3 machine-readable schemas live under `schemas/v3`:

- `policy-rules.schema.json`;
- `policy-vocabulary.schema.json`;
- `policy-evaluation-report.schema.json`;
- `generated-policy-metadata.schema.json`.

The agent manifest schema is `.cervorules/schemas/agent-manifest.schema.json`,
next to the manifest it describes.

Release artifacts must include these schemas in the GitHub Release and OCI
tools image.
