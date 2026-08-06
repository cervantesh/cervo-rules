# Generator Architecture

`cmd/` and `internal/` are now v3-native. They exist in this repository only for
tooling that targets `github.com/cervantesh/cervo-rules/v3`.

## Responsibilities

| Area | Files | Responsibility |
| --- | --- | --- |
| Policy CLI | `cmd/cervorules-policygen` | Parse `check` and `generate` commands, open files, and call `internal/policygen`. |
| Vocabulary CLI | `cmd/cervorules-vocabgen` | Generate v3 `Operation`, `Target`, and `Executor` constants. |
| Policy generator core | `internal/policygen/generator.go` | Load YAML, validate v3 vocabulary references, build metadata, and expose `Check` / `Generate`. |
| Fact and predicate model | `internal/policygen/facts.go` | Resolve declared facts against policy bounds, and validate predicate trees against fact types and declared conditions. |
| Policy rendering | `internal/policygen/render.go` | Emit `PolicyFactory`, default config, config validation, runtime merge, and generated engine code. |
| Predicate rendering | `internal/policygen/render_facts.go` | Compile predicates to Go boolean expressions, and emit the fact frame and its fail-closed parsers. |
| Generated tests | `internal/policygen/test_render.go` | Emit generated test cases from declarative policy tests. |
| Vocabulary generator core | `internal/vocabgen/generator.go` | Load v3 vocabulary YAML and emit typed constants, fact-name constants, plus `Vocabulary()`. |
| Contracts | `internal/*/*_test.go`, `cmd/*/*_test.go` | Keep CLI behavior, generated output, examples, and temp consumer compilation stable. |

## Current v3 Generator Scope

The native v3 policy generator supports:

- `version: cervorules.policy.v3`;
- neutral vocabulary maps: `operations`, `targets`, `executors`;
- operation routes to target/executor;
- executor fallbacks;
- disabled-by-default routes;
- ordered denies, evaluated in authored order, optionally operation-wide;
- named conditions and rule `requires`;
- vocabulary `facts` with declared types, and policy `facts` with bounds and
  defaults;
- compound `when` predicates over those facts (`all` / `any` / `not`), compiled
  to Go boolean expressions;
- populated `DecisionTrace` steps naming the rule and the deciding leaf;
- trusted users in runtime config;
- generated `PolicyFactory`;
- generated metadata, default config, config validation, and tests.

The native v3 generator does not currently claim historical v2 commands or DSL
features such as `compat`, `inspect-api`, `diff`, `migrate-v3`, limits codegen,
or derived facts codegen. Those need separate v3 design and issues before being
reintroduced.

Predicates are a closed tagged form, not an expression language, and they carry
no runtime evaluator: each one compiles to a Go boolean expression the compiler
checks. That is deliberate — an expression language inside a security policy is
attack surface, not convenience — and it is why the feature added no dependency.

## Review Rules

- Generator changes that alter generated public API must update README,
  `AGENTS.md`, this document, schemas, and change-management notes.
- New DSL fields need schema coverage, parser validation, generated output
  tests, and at least one example or explicit negative test.
- Generated imports are part of the contract and must remain v3 modular imports.
- Generated policies must not emit `BuildPolicy` wrappers or v2 primitive names.
- Example vocabulary must stay domain-neutral unless a document is explicitly
  describing a consumer integration.

## Drift Checks

Before merging generator changes, run:

```powershell
go test -count=1 ./...
go run ./cmd/cervorules-policygen check `
  -vocab examples/routing-basic/policy-vocabulary.yaml `
  -policy examples/routing-basic/policy-rules.yaml `
  -format json
```

When release artifacts are affected, also run the package smoke command in
`docs/release.md`.
