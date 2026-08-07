# CervoRules v3 Policygen DSL

Tracks epic #342 and child issues #343, #344, #345, #346, and #347.

## Goal

v3 policy authoring uses neutral names in YAML and schema contracts:

- `operation` replaces `capability`;
- `target` replaces `service`;
- `executor` replaces `provider`;
- `operation_in` replaces `capability_in`;
- `target_healthy` replaces `service_healthy`;
- `fallback_executors` replaces `fallback_providers`.

The DSL major version is `cervorules.policy.v3`.

## Schema Files

The v3 schema contracts live under `schemas/v3/`:

- `schemas/v3/policy-rules.schema.json`;
- `schemas/v3/policy-vocabulary.schema.json`.

These schemas reject deprecated v2 field names. v2 schemas remain available at
the existing paths and continue to describe `cervorules.policy.v1`.

## Compatibility / Diff Expectations

Native v3 `compat` and `inspect-api` are deferred. Until they are redesigned,
review policy changes with `check -format json`, generated tests, and manual
diffs. Future migration tooling should report replacements such as:

- `capability -> operation`;
- `service -> target`;
- `provider -> executor`;
- `BuildPolicy -> PolicyFactory.Build`.

## Implemented Generator Scope

The current v3 parser and generator cover the routing subset: operations,
targets, executors, fallbacks, disabled routes, denies, default config,
metadata, config validation, and generated tests; plus compound predicates over
typed facts, ordered denies, and populated decision traces. Generated limits and
generated derived facts need separate v3 issues.

## Facts and Predicates

The vocabulary declares what a fact is:

```yaml
facts:
  account_mode: { type: enum, values: [demo, live] }
  risk_pct: { type: number }
  orders_last_hour: { type: integer }
  operable: { type: bool }
```

`type` is one of `number`, `integer`, `bool`, `enum` or `string`. A `go_name`
makes `vocabgen` emit a `Fact*` constant, so a misspelled metadata key in the
consumer is a compile error rather than a fact the policy cannot find.

The policy declares what it does with a fact:

```yaml
facts:
  risk_pct: { min: 0 }
  min_score: { min: 0, default: 50 }
```

The split is deliberate. `min`, `max` and `default` are decision-bearing
numbers, and the `PolicyHash` is taken over the raw bytes of the policy file, so
putting them there means editing a threshold moves the hash.

**A fact you declare bounds for must be read by some rule.** The generator emits
a parser only for the facts a predicate mentions, so a bound on a fact nothing
reads would never be applied: a request carrying `999`, or `NaN`, or nonsense
for it would be decided as though the key were absent. Rather than let a
declaration look like validation without being one, `check` and `generate`
refuse the policy and name the fact. Reference it in a `when:`, or delete the
declaration.

A fact counts as read whether it is the subject of a leaf, the right-hand side
of a `fact_value` comparison, or read by a route's predicate rather than a
deny's.

A route or deny can carry a `when` predicate:

```yaml
denies:
  - id: deny-risk-or-exposure
    reason: risk_pct or exposure_pct exceeds its limit
    when:
      any:
        - { fact: risk_pct, op: gt, value: 1.5 }
        - { fact: exposure_pct, op: gt, value: 5.0 }
```

Composition is `all`, `any` and `not`, nestable. A leaf is one of:

| Form | Meaning |
| --- | --- |
| `{ fact: F, op: eq\|ne, value: V }` | scalar equality, typed by `F` |
| `{ fact: F, op: gt\|gte\|lt\|lte, value: N }` | numeric comparison against a literal |
| `{ fact: F, op: gt\|gte\|lt\|lte, fact_value: G }` | comparison against another fact of the same type |
| `{ fact: F, op: is_true\|is_false }` | boolean fact |
| `{ fact: F, op: in, values: [...] }` | membership |
| `{ condition: NAME }` | a named condition from the `conditions` block |

This is a closed tagged form, not an expression language. Predicates compile to
Go boolean expressions in the generated policy, so there is no evaluator at
runtime and no third-party dependency. `requires: [a, b]` remains exactly
equivalent to `when: {all: [{condition: a}, {condition: b}]}`.

Facts are read from `Request.Metadata`, parsed once per decision, before any
rule runs. A fact that is absent with no declared default, unparseable,
non-finite, or outside its declared bounds fails the decision with a structured
`missing_fact` or `invalid_fact` error. It is never reported as a predicate that
did not hold.

### Denies

`denies` is an ordered list; the first match wins. A deny with no `operation`
applies to every operation in the vocabulary. `reason` is free text for humans
and defaults to the rule's `id`.

### Explain

With `DecisionOptions` trace enabled, the result carries one
`DecisionTraceStep` per evaluated rule: `Name` is the rule id, `Matched` says
whether it fired, and `Reason` names the leaf that decided it — which of the
disjuncts, for a compound rule. Trace stays opt-in per decision; an untraced
decision allocates nothing for explanation.

See [compound-predicates.md](compound-predicates.md) for the design record and
the worked trading-policy example.
