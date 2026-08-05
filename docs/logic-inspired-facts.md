# Logic-Inspired Facts

CervoRules includes an optional Datalog-inspired facts layer for derived,
explainable operational facts.

This is not Prolog. The goal is deterministic policy support, not general logic
programming.

## Why

Some consumers need cross-domain relationships that should not become rigid
`Request` fields:

- user-role relationships;
- tenant-plan relationships;
- device-capability relationships;
- document-classification relationships;
- service ownership relationships.

Derived facts let consumers declare these relationships once and let policies
consume the result without adding project-specific vocabulary to CervoRules
core.

## Model

The intended pipeline is:

```text
input facts -> derived facts -> policy predicates -> decision -> observation
```

The facts layer owns flat facts, query patterns, bounded derivation, and
derivation traces. It does not own request parsing, transport mapping, policy
decisions, logging sinks, metrics sinks, or consumer vocabulary.

## Optional Module Boundary

The package boundary is:

```text
github.com/cervantesh/cervo-rules/v2/facts
```

The module remains optional. Consumers that do not need derived facts
should continue to use `core`, `runtime`, `limits`, `httpadapter`, `observe`,
and `testkit` without importing `facts`.

The root package may expose compatibility aliases only after the owning package
is stable. Core policy evaluation must not depend on the facts package except
through a small, explicit interface if a later ADR approves it.

## Adapter And Core Separation

Adapters create caller-owned input facts. The facts engine derives additional
facts. Core policy predicates consume selected facts through an explicit bridge.

Integration happens outside core unless a generated policy explicitly declares
optional `derived_facts` helpers:

1. A consumer or adapter maps transport data into neutral facts.
2. The optional facts engine evaluates rules with bounded options.
3. The consumer copies selected derived facts into `Request`, `RequestFacts`, or
   metadata supported by existing policy predicates.
4. Core evaluates the policy deterministically as it does today.

Generated policy integration is still an explicit helper boundary. It keeps the
facts layer optional and does not make core own transport or consumer
vocabulary.

## Example Shape

```go
role, err := facts.NewFact("user_role", facts.Const("alice"), facts.Const("admin"))
if err != nil {
    return err
}
input := facts.NewSet(role)

engine := facts.NewEngine(facts.NewRule(
    "admins_are_trusted",
    facts.NewPattern("trusted_user", facts.Var("user")),
    facts.NewPattern("user_role", facts.Var("user"), facts.Const("admin")),
))

result, err := engine.Evaluate(ctx, input, facts.EvalOptions{
    MaxIterations: 4,
    MaxFacts:      128,
    MaxBindings:   128,
})
if err != nil {
    return err
}
```

Negation is explicit and stratified:

```go
engine := facts.NewEngine(
    facts.NewRule("blocked",
        facts.NewPattern("blocked_user", facts.Var("user")),
        facts.NewPattern("deny_user", facts.Var("user")),
    ).InStratum(0),
    facts.NewRule("allowed",
        facts.NewPattern("allowed_user", facts.Var("user")),
        facts.NewPattern("known_user", facts.Var("user")),
        facts.Not(facts.NewPattern("blocked_user", facts.Var("user"))),
    ).InStratum(1),
)
```

Negated body patterns must use variables bound by earlier positive body
patterns. Negated predicates produced by rules must come from lower strata;
negative cycles fail validation with `unstratified_negation`.

Controlled aggregates summarize completed input or lower-stratum facts:

```go
engine := facts.NewEngine(
    facts.CountAggregate("approval-count",
        "approval",
        "approval_count",
        []int{0},
    ).InStratum(1),
)
```

The derived fact shape appends the aggregate value after the configured group
terms, for example `approval_count("doc-1", "2")`. `min`, `max`, and `sum`
operate on integer term values and fail closed with diagnostics for invalid
values, overflow, or aggregate budget limits.

Stable facts can be prepared once and combined with request-specific facts
later:

```go
prepared, err := engine.Prepare(ctx, staticFacts, options)
if err != nil {
    return err
}
result, err := prepared.Evaluate(ctx, requestFacts, options)
```

Prepared evaluation does not produce partial decisions and does not bypass
normal validation. It is an optimization-oriented contract whose tests prove
equivalence with full evaluation over `staticFacts.Add(requestFacts...)`.

Incremental evaluation uses explicit snapshots and change sets:

```go
snapshot, err := engine.Snapshot(ctx, inputFacts, options)
if err != nil {
    return err
}
result, err := engine.EvaluateDelta(ctx, snapshot, facts.ChangeSet{
    Add:    []facts.Fact{added},
    Remove: []facts.Fact{removed},
}, options)
```

The current implementation applies the change set to the previous input
snapshot and runs normal bounded evaluation. This keeps semantics identical to
full evaluation while establishing a safe public contract for future dependency
indexes.

## Declarative Rule Spec

`RuleSetSpec` is the domain-neutral declarative form for facts rules. It is
JSON/YAML-friendly, but the `facts` package does not import a YAML parser.
Policygen or consumers can decode YAML with their own tooling and call
`CompileRuleSetSpec`.

```go
admin := "admin"
compiled, diagnostics := facts.CompileRuleSetSpec(facts.RuleSetSpec{
    Version: facts.SupportedRuleSetVersion,
    Rules: []facts.RuleSpec{{
        Name: "admins_are_trusted",
        Head: facts.PatternSpec{
            Predicate: "trusted_user",
            Terms: []facts.TermSpec{{Var: "user"}},
        },
        Body: []facts.PatternSpec{{
            Predicate: "user_role",
            Terms: []facts.TermSpec{{Var: "user"}, {Const: &admin}},
        }},
    }},
})
if len(diagnostics) > 0 {
    return diagnostics
}
result, err := compiled.Engine().Evaluate(ctx, input, compiled.EvalOptions)
```

The declarative spec can include optional predicate schema, constraints, strata,
and evaluation bounds.
Body patterns can set `not: true` for stratified negation.
The spec can also include `aggregates` entries with `count`, `min`, `max`,
`sum`, or `exists`.

## Generated Policy Integration

`policy-rules.yaml` can declare an optional `derived_facts` block using the
same `RuleSetSpec` shape. `cervorules-policygen` validates that block with
`facts.CompileRuleSetSpec` and only imports
`github.com/cervantesh/cervo-rules/v2/facts` when the block is present.

Generated policies expose explicit helpers:

```go
factory := policyrules.NewPolicyFactory()
compiled, diagnostics := factory.DerivedFacts()
if len(diagnostics) > 0 {
    return diagnostics
}
result, err := factory.DeriveFacts(ctx, inputFacts, compiled.EvalOptions)
if err != nil {
    return err
}
```

This is intentionally not an implicit policy bridge. Consumers still own how
transport data becomes input facts and how selected derived facts become
`Request` fields or metadata before core policy evaluation. The generator
provides a stable, policy-versioned facts contract without making `core` depend
on `facts`.

## What This Borrows From Prolog

- facts;
- rules;
- variables in flat query patterns;
- explanations for derived conclusions.

## What This Does Not Borrow From Prolog

- a Prolog parser;
- `cut`;
- side-effecting predicates;
- dynamic `assert` or `retract`;
- unbounded backtracking;
- unbounded or free recursion;
- full Prolog unification of nested terms;
- automatic multi-solution procedural control.

Rules should be closer to Datalog: flat facts, variables over flat terms,
deterministic fixed-point evaluation, and explicit bounds.

## Operational Guardrails

- Evaluation is bounded by `MaxIterations` and `MaxFacts`.
- Intermediate joins are bounded by `MaxBindings`.
- Optional `PredicateSchema` validates predicate names, arity, minimal term
  types, and enum values before evaluation.
- Optional constraints such as max-facts-per-key and mutually-exclusive
  predicates report ambiguous fact states without deciding policy outcomes.
- Rules can declare a non-negative stratum with `InStratum`; lower strata run
  before higher strata, and rules in the same stratum run by stable rule name.
- `facts.Not(pattern)` and declarative `not: true` support stratified negation
  against lower strata or base input facts. Unsafe negation variables and
  negative cycles fail validation.
- Controlled aggregates derive bounded summary facts from base or lower-stratum
  predicates. Aggregate groups, per-group inputs, outputs, integer parsing, and
  sum overflow fail closed with structured diagnostics.
- `Engine.Prepare` and `PreparedEngine.Evaluate` provide partial evaluation for
  stable facts while preserving full-evaluation equivalence.
- `Engine.Snapshot` and `Engine.EvaluateDelta` provide explicit incremental
  evaluation semantics over immutable fact sets and caller-owned change sets.
- Rules validate before evaluation.
- Derived facts are traceable.
- Diagnostics report bound failures and invalid rules.
- Validation and budget failures return an error with diagnostics so callers can
  fail startup or request handling clearly.
- Fact and trace ordering must be deterministic for golden tests and reviews.
- Facts, fact sets, derivation steps, and derivation traces expose stable JSON
  for machine fixtures, audits, and policy review. Serialized sets and traces
  preserve the canonical fact ordering used by queries and explanations.
- Redaction is caller-configured by predicate and term position. The facts
  package does not guess which terms are sensitive and does not encode
  consumer-specific privacy vocabulary.
- Optional `FactRecord` metadata can carry source, trust level, observed time,
  and expiration time without changing logical fact equality.
- `ValidateFreshness` reports expired facts and facts observed after the
  caller-provided validation time. It does not add temporal logic or a global
  clock to the evaluator.
- `LintRules` reports authoring smells such as empty rule bodies, duplicate
  body patterns, and singleton body variables without changing evaluation
  behavior.
- `DerivationTrace.Summary` and `GoldenExplanation` provide deterministic,
  redaction-aware contracts for observability and golden test fixtures.
- Evaluation uses a conservative changed-predicate agenda after the first
  iteration. Rules whose body predicates did not change are skipped, preserving
  full fixed-point semantics while reducing redundant derivation work.
- `RuleSetSpec` and `CompileRuleSetSpec` provide a declarative facts DSL
  contract without adding YAML or policygen dependencies to the `facts` package.
- Policygen accepts optional `derived_facts`, validates it, and emits
  `PolicyFactory.DerivedFacts` / `PolicyFactory.DeriveFacts` helpers only when
  that block is present.
- Integrations stay optional and adapter-owned.

`Fact` fields are intentionally private even though the original design sketch
showed a public struct. This keeps sets, engines, and traces immutable after
construction and avoids caller mutations during concurrent evaluation.

## TDD And Verification Expectations

Implementation issues should use table-driven tests and verify red/green
behavior for each public contract:

- fact constructors and stable formatting;
- immutable fact sets and query binding;
- bounded evaluator behavior, including max-fact and max-iteration failures;
- derivation traces and explain output;
- stable JSON serialization for facts, sets, and derivation steps;
- redaction of facts, sets, and traces without mutating originals;
- fact metadata and freshness validation with caller-provided clocks;
- rule lint diagnostics as authoring feedback separate from runtime errors;
- trace summaries and golden explanations for reviewable explain output;
- agenda optimization tests proving unchanged rules are skipped without losing
  recursive derivations;
- declarative `RuleSetSpec` compilation and diagnostics;
- optional policygen integration without importing facts when unused;
- optional adapter integration without changing core behavior.

Before merging implementation work, run the focused package tests, the full Go
test suite, coverage, vet, module verification, dependency scope tests, and an
unresolved-marker scan for changed docs.

Docs-only changes for this boundary should at minimum run an unresolved-marker
scan over the changed Markdown files and a grep proving the Datalog/Prolog
boundary is discoverable.

## Issue Map

- #151 tracks the umbrella facts epic.
- #152 tracks immutable fact set and pattern query API.
- #153 tracks the bounded derived fact evaluator.
- #154 tracks derivation trace and explain output.
- #155 tracks validation guardrails.
- #156 tracks this documentation boundary.
- #157 tracks neutral derived facts examples.
- #172 tracks stable serialization for facts, sets, derivation steps, and
  derivation traces.
- #175 tracks privacy redaction for facts and derivation traces.
- #173 tracks fact source and freshness metadata.
- #174 tracks fact TTL validation.
- #176 tracks rule linting.
- #166 tracks explain plan observability.
- #179 tracks golden explanation contracts.
- #182 tracks explain CLI design; see `docs/facts-explain-cli.md`.
- #164 tracks semi-naive-style evaluator optimization.
- #197 tracks declarative facts DSL parser and validator.
- #198 tracks optional policygen derived facts integration.
- #199 tracks stratified negation runtime.
- #200 tracks controlled aggregates runtime.
- #201 tracks partial evaluation API.
- #202 tracks incremental evaluation API.
- #165 tracks incremental evaluation design; see
  `docs/adr/0012-incremental-fact-evaluation.md`.
- #167 tracks materialized fact view patterns; see
  `docs/materialized-fact-views.md`.
- #168 tracks closed-world semantics; see
  `docs/adr/0009-closed-world-fact-semantics.md`.
- #180 tracks policy diff design; see `docs/policy-diff-design.md`.
- #181 tracks partial evaluation design; see
  `docs/adr/0011-partial-evaluation.md`.
- #183 tracks namespaced fact vocabulary; see
  `docs/adr/0010-namespaced-fact-vocabulary.md`.
- #161 tracks stratified negation; see
  `docs/adr/0006-stratified-negation.md`.
- #162 tracks bounded recursion; see
  `docs/adr/0007-bounded-recursion.md`.
- #163 tracks controlled aggregates; see
  `docs/adr/0008-controlled-aggregates.md`.

Issues #152 through #157 can ship without changing core policy behavior. Future
generated policy or core bridge work requires a second ADR or checkpoint after
the base facts API is proven.
