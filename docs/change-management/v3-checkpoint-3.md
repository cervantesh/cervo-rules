# CervoRules v3 Checkpoint 3

Epic: #302

Checkpoint issue: #388

Child issues: #389, #390, #391

Date: 2026-05-24

## Scope Reviewed

This checkpoint covers workstreams 12-15:

| Workstream | Issue | Result |
| --- | --- | --- |
| Observability schema versioning | #364 | Keep. v3 has a versioned operational report contract, stable low-cardinality labels, and a schema that excludes user/metadata by default. |
| Machine-readable contracts | #370 | Keep. v3 has schemas for agent manifest, policy inspection, compatibility reports, generated metadata, and public API inventory. |
| Migration tooling | #376 | Reclassified after the physical split. Historical v2 migration tooling lives in `CervoSoft/cervo-rules-v2`; native v3 migration reporting is deferred. |
| Consumer conformance and compatibility suite | #382 | Keep. v3 has repo-local conformance contracts, neutral fixtures, a proxy-shaped neutral fixture, facts fixture validation, and package smoke shape. |

## Release Readiness Scorecard

| Area | Score | Evidence | Release action |
| --- | --- | --- | --- |
| Machine usability | 4.4 / 5 | Versioned schemas, public API inventory, migration report command, and conformance docs. | Keep schemas stable through `v3.0.0-rc.1`; avoid prose-only contracts. |
| Schema stability | 4.2 / 5 | v3 policy, vocabulary, observation, inspection, compatibility, manifest, and generated metadata schemas exist. | Add package verification in #392 and tag verification in #404. |
| Migration sufficiency | 4.0 / 5 | Manual v2-to-v3 replacements are documented. | Raise after native v3 migration reporting is redesigned or a real consumer migration is recorded. |
| Consumer confidence | 4.1 / 5 | `v3/testkit.ConsumerConformanceContract` covers generated factory, facts, package smoke, and neutral fixtures. | Keep CervoProxy-shaped fixture neutral until real consumer migration happens outside this repo. |
| Public API freeze confidence | 4.1 / 5 | Checkpoints 1-3 keep `Operation`, `Target`, `Executor`, modular packages, factory, optional facts, explicit routing, and versioned reports. | Freeze these names for the RC unless release work exposes a concrete blocker. |

## Decisions

### Keep

- Keep `Operation`, `Target`, and `Executor` as the v3 public primitive names.
- Keep `runtime.PolicyFactory` as the only generated runtime entrypoint.
- Keep `BuildPolicy` out of v3 generated APIs and migration tooling only.
- Keep `RoutingPhase` removed from v3 and preserve explicit indexed/linear routing plans.
- Keep facts optional, budgeted, and trace opt-in by default.
- Keep observability reports versioned and low-cardinality by default.
- Keep machine-readable schemas as release artifacts.
- Keep consumer conformance in `v3/testkit`, not in core.

### Drop

- Drop any compatibility alias that reintroduces `Capability`, `Service`, or `Provider` into v3 public API.
- Drop generic root-facade expansion for convenience; modular imports remain the intended path.
- Drop any attempt to make CervoProxy-specific names part of v3 fixtures.
- Drop automatic source rewriting from the first migration tooling pass. Reporting is enough for RC.

### Defer

- Defer full v3 policy parser/codegen implementation until release/package work proves the artifact path and schemas.
- Defer real CervoProxy migration until after v3 RC package verification.
- Defer automated v2-to-v3 rewrites to v3.1 unless a real consumer migration proves they are necessary.
- Defer hard performance gates for v3 until benchmark history spans more runner executions.

## Answers To Required Questions

### Can a machine understand and use v3 better than v2?

Yes. v3 now has explicit module boundaries, versioned schemas, a public API
inventory, migration reporting, and conformance fixtures. An agent can start
from `AGENTS.md`, inspect schemas under `schemas/v3`, use migration docs, and use
`v3/testkit` contracts without relying only on prose.

### Are schemas stable enough for release candidates?

Stable enough for `v3.0.0-rc.1`, with one condition: #392 must package and
verify the v3 schemas as release artifacts before tag work begins.

### Is migration tooling sufficient before publishing?

Sufficient for an RC through documentation, not yet through native tooling.
Manual guidance covers the high-risk v2 usages: `Capability`, `Service`,
`Provider`, `BuildPolicy`, `RoutingPhase`, and old YAML field names. GA should
either add native v3 migration reporting, prove a real consumer migration, or
document why manual migration remains acceptable.

### Which v3 APIs are still too broad?

`PolicyRuntimeConfig` may still be broader than some non-routing consumers need,
but it is isolated in `runtime` and no longer pollutes `core`. Keep it for RC and
revisit only after real generated policies exist.

### Should any v3 feature move to v3.1 instead of v3.0.0?

Yes:

- automated source rewrites;
- full facts evaluator implementation in v3;
- hard benchmark regression gates;
- real consumer migrations such as CervoProxy.

## Public API Freeze

Freeze the following through `v3.0.0-rc.1` unless #392 release validation exposes
a concrete blocker:

- `core.Operation`, `core.Target`, `core.Executor`;
- `core.DecisionResult`, `core.DecisionOptions`, `core.Engine`;
- `runtime.PolicyFactory`, `runtime.PolicyMetadata`, `runtime.PolicyRuntimeConfig`;
- `facts.EvalOptions`, `facts.Result`, `facts.EvaluationPlan`;
- `observe.PolicyEvaluationReport`;
- `testkit.ConsumerConformanceContract`.

## Sequence Adjustment

No issue order change is required. Continue to #392 release, packages, and supply
chain after this checkpoint is merged and #388-#391 are closed.

## Next Sequence

Proceed to #392 after this checkpoint is merged and #388-#391 are closed.
