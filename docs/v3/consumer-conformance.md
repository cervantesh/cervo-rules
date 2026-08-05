# v3 Consumer Conformance

Issues: #382, #383, #384, #385, #386, #387.

The v3 conformance suite is repo-local evidence that generated policy shapes,
neutral consumer fixtures, facts adapters, and package smoke expectations remain
usable before a v3 release candidate is published.

## Contract

Use `v3/testkit.ConsumerConformanceContract` for generated-policy or
consumer-shaped tests:

```go
testkit.MustAssertConsumerConformance(t, testkit.ConsumerConformanceContract{
    Name:           "billing-routing",
    PolicyPath:     "policy-rules.yaml",
    VocabularyPath: "policy-vocabulary.yaml",
    RuntimeCases: []testkit.RuntimeCase{{
        Name:         "factory-decision",
        Factory:      generated.NewPolicyFactory(),
        Config:       generated.NewPolicyFactory().DefaultConfig(),
        Request:      core.Request{Operation: "invoice.read"},
        WantDecision: core.Decision{Allow: true, Target: "billing-ledger"},
    }},
})
```

`CheckConsumerConformance` returns an error and is meant for CLI or package smoke
checks. `MustAssertConsumerConformance` is the only testing-boundary wrapper.

## Coverage

The fixture matrix covers:

| Fixture | Purpose |
| --- | --- |
| `billing-routing` | business operation routing without gateway vocabulary |
| `document-processing` | background document work |
| `device-routing` | device command routing |
| `queue-event-routing` | queue/event adapter shape |
| `scheduled-job-routing` | scheduled job adapter shape |
| `cli-command-routing` | internal command adapter shape |
| `edge-request-routing` | CervoProxy-shaped routing without CervoProxy-specific names |

## Generated Policy Fixture

Runtime cases require a `runtime.PolicyFactory`. This proves v3 consumers use the
canonical generated entrypoint instead of the removed v2 `BuildPolicy` wrapper.
The factory must provide metadata, validate config, build an engine, and return
the expected `core.Decision`.

## Facts Fixture

Facts cases use the minimal `FactsEvaluator` interface:

```go
type FactsEvaluator interface {
    Evaluate(context.Context, facts.EvalOptions) (facts.Result, error)
}
```

This keeps `facts` optional. Consumers can adapt a generated facts engine, a
precomputed materialized view, or a test double. Conformance requires positive
`MaxIterations` and `MaxFacts` so per-request facts work is never unbounded.

## Package Smoke Fixture

`PackageSmoke` records the modules and schemas a clean consumer needs:

- module path: `github.com/cervantesh/cervo-rules/v3`;
- packages such as `core`, `runtime`, `facts`, and `testkit`;
- schemas such as `schemas/v3/policy-rules.schema.json` and
  `schemas/v3/policy-vocabulary.schema.json`.

Release workstreams should reuse this shape before publishing `v3.0.0-rc.1`.

## Agnosticism Guardrail

The conformance fixtures intentionally avoid CervoProxy vocabulary in core
contracts. `edge-request-routing` is proxy-shaped only in the sense that it has
request metadata and routing behavior; names remain neutral:

- operation: `request.route`;
- target: `request-router`;
- executor: `primary-route`.
