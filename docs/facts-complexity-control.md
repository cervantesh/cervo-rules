# Facts Complexity Control

Related epic: #292.

The `facts` package is intentionally optional and bounded. It is appropriate
for deterministic derivation, explainable joins, stratified negation, and
aggregates, but it should be operated with explicit budgets when used near a
request path.

## Operational Limits

Use `EvalOptions` budgets for every production evaluation:

- `MaxIterations`: stops recursive derivation loops.
- `MaxFacts`: caps total materialized facts.
- `MaxBindings`: caps intermediate join bindings.
- `MaxRuleEvaluations`: caps total rule evaluation attempts across iterations.
- `MaxDerivedFactsPerRule`: caps facts produced by a single rule evaluation.
- `MaxFactsPerPredicate`: caps materialized facts for specific predicates.
- `MaxAggregateGroups`, `MaxAggregateInputsPerGroup`, and
  `MaxAggregateOutputFacts`: cap aggregate fanout.

`MaxRuleEvaluations` and `MaxDerivedFactsPerRule` are intended for operational
fail-closed behavior. They produce stable diagnostics:

- `max_rule_evaluations`;
- `max_rule_derived_facts`.
- `max_bindings`.
- `max_facts_per_predicate`.

## Debug Stats And Expensive Rules

Set `Debug: true` when investigating evaluator behavior. The result stats then
include per-rule counters:

- rule name;
- stratum;
- evaluation count;
- derived fact count;
- observed maximum intermediate bindings.

Set `ExpensiveRuleDerivedFactThreshold` to emit a non-fatal `expensive_rule`
diagnostic when one rule derives more facts than expected. This is useful in
CI, staging, or profiling runs where the evaluation should still succeed but
the caller wants a clear signal that one rule is too broad.

Set `ExpensiveRuleBindingThreshold` to emit a non-fatal
`expensive_rule_bindings` diagnostic when a rule creates too many intermediate
join bindings. This catches the common performance failure mode where a rule
does a broad join, burns CPU and memory, but may not derive many final facts.
Use this together with `Debug: true` while tuning rules so `Stats.Rules` records
the observed `MaxBindings` per rule. The diagnostic message includes the rule,
the observed binding count, the declared pattern index, and the predicate that
produced the largest intermediate binding set.

Do not enable debug stats on the hottest path unless you need them. Normal
evaluation keeps the extra per-rule stats empty.

## Runtime Mode And Explain Mode

Use a clear split between runtime evaluation and explain/debug evaluation.

Runtime Mode should use `TraceDisabled` and `Debug: false`. This keeps
derivation steps, source fact copies, and per-rule debug stats out of the hot
path while preserving the same derived facts.

Explain Mode should opt in deliberately with `TraceEnabled` or default tracing,
and `Debug: true` only when per-rule counters are needed. Explain runs may use
more memory because they materialize derivation steps, sources, and diagnostic
details that are useful for humans and agents.

## Plan(input)

Use `Engine.Plan(input)` to inspect how rules will be evaluated for a concrete
fact set. The plan reports:

- rule name and stratum;
- whether the body order was reordered;
- planned pattern order;
- declared pattern index;
- predicate;
- negation flag;
- constant count;
- candidate count from current indexes.
- selectivity reason:
  - `constant_index`: a constant term lets the planner use the term index;
  - `predicate_arity`: the planner can narrow by predicate and arity;
  - `full_scan`: the pattern has no predicate and scans the whole set.
- constant positions when the planner can identify which term positions are
  selective.

This gives humans and agents a cheap way to explain why a rule is expensive
before running profiles. Candidate counts also help identify missing constants
or overly broad joins.

## Planner Behavior

The planner may reorder positive pattern segments by connectivity and
selectivity. It prefers patterns that bind variables used by the rule head,
then patterns that connect to variables already bound by earlier patterns, then
lower candidate counts and higher constant counts. This avoids putting broad
or independent guard patterns before the join path that actually derives the
head fact.

Negated patterns keep their declared position so closed-world semantics and
safety diagnostics remain stable. Positive segments before or after a negation
can be reordered independently.

The planner falls back to declared order when there are fewer than three
patterns in a segment because the overhead is unlikely to help.

## Binding Profiles

Use `NewBindingLayout(rule, patterns...)` to document the compact slot layout a
future evaluator or adapter should use for a rule. The layout assigns variables
to first-seen slot indexes, so runtime evaluation can store temporary bindings
in slices and materialize the public `Binding` map only at API/debug
boundaries.

This is the v3-native answer to the old binding-allocation debt: the active
repository exposes the compact representation contract without reintroducing a
large facts evaluator into the hot path.

## Compound Index Recommendations

Use `RecommendCompoundIndexes(plan, stats, options)` after collecting plan and
cost evidence. It recommends indexes only for positive patterns where candidate
counts or rule binding fanout exceed thresholds. Negated patterns are skipped
because indexing them can change the cost profile without addressing the join
that derives new facts.

The recommendation output is machine-readable and includes rule name,
predicate, declared pattern index, constant positions, candidate count, observed
max bindings, and reason.

Treat the output as advisory. Add compound indexes only when the benchmark
matrix shows the lookup wins more than the extra memory/index maintenance costs.

## Large Workload Profiling

Use the large join benchmark when changing indexes, planning, bindings, or
facts limits:

```bash
go test -run '^$' -bench BenchmarkFactsLargeJoinWorkload -benchmem ./facts
```

For pprof evidence:

```bash
scripts/performance/profile.sh
```

The profile script records CPU and memory profiles for representative facts
workloads. Generated `.pprof` files are local artifacts and should not be
committed.

## Production Checklist

- Compile or construct the engine once when possible.
- Use `TraceDisabled` when derivation explanations are not required.
- Keep `MaxFacts`, `MaxBindings`, and `MaxRuleEvaluations` finite.
- Add `MaxFactsPerPredicate` for predicates that can dominate memory, such as
  recursive closure outputs, eligibility sets, or broad catalog projections.
- Use `Plan(input)` during rollout to inspect candidate counts.
- Add `ExpensiveRuleDerivedFactThreshold` in staging to detect broad rules.
- Add `ExpensiveRuleBindingThreshold` when joins combine multiple broad
  predicates or user-controlled facts.
- Record benchmark output before changing planner or binding internals.
