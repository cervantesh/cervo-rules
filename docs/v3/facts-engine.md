# CervoRules v3 Facts Engine

Tracks epic #348 and child issues #349, #350, #351, #352, and #353.

## Goal

Facts are an explicit optional logic engine in v3. The decision runtime does not
depend on facts, and consumers must choose when facts belong outside the hot
path.

## Runtime Boundary

`v3/facts` owns:

- rule and fact evaluation;
- stable `EvaluationPlan` JSON;
- complexity diagnostics;
- derivation trace when explicitly enabled;
- operational budget guidance.

`v3/core` owns deterministic decisions. Facts-derived information should be
adapted into `core.Request` or a generated policy input only when a consumer
chooses that architecture.

## Plan JSON

`EvaluationPlan` must include a schema version:

```json
{
  "schema_version": "cervorules.v3.facts.plan.v1",
  "rules": []
}
```

The stable fields are rule name, stratum, reordered flag, pattern declared
index, predicate, negation flag, constants count, candidate count, and
selectivity reason. Selectivity reason is intentionally machine-readable so an
agent can distinguish a constant-index lookup from a predicate/arity lookup or
a full scan without parsing prose.

## Complexity Diagnostics

Facts diagnostics use stable fields:

- `code`;
- `rule`;
- `field`;
- `limit`;
- `observed`;
- `reason`.

Initial operational codes include `max_iterations`, `max_facts`,
`max_bindings`, `max_rule_evaluations`, `max_rule_derived_facts`, and
`expensive_rule`.

`expensive_rule_bindings` is the cost diagnostic to watch first for joins. It
means a rule created too many intermediate bindings, even if it did not derive
many final facts. That is the common shape of expensive Datalog-style rules.

## Trace Defaults

Trace is opt-in by default in v3. Consumers enable trace only for explain/debug
workflows. Production hot paths should set explicit budgets and keep trace
disabled unless the request is being sampled or investigated.

## Operational Budgets

Every production use should set:

- `MaxIterations`;
- `MaxFacts`;
- `MaxBindings` when joins are non-trivial.
- `ExpensiveRuleBindingThreshold` in staging or sampled diagnostics when joins
  combine broad predicates.

Large workloads should inspect `EvaluationPlan` before rollout and record
complexity diagnostics in logs or metrics with low-cardinality labels.
