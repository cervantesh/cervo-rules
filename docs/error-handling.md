# Structured Error Handling

CervoRules v3 uses structured errors for validation, generation, facts
complexity and generated policy build failures.

## Contract

- `Decision.Allow == false` is a policy decision, not an error.
- `err != nil` means validation, generation, build or evaluation was not
  trustworthy enough to produce a normal result.
- Consumers should fail closed when `err != nil`.
- CervoRules does not assign HTTP status codes. HTTP, gRPC and queue adapters
  map errors at the application boundary.

## Core Shape

```go
type Error struct {
    Code       ErrorCode
    Severity   Severity
    Component  string
    Field      string
    Rule       string
    Value      string
    Limit      int
    Observed   int
    Reason     string
    Suggestion string
    Cause      error
}
```

`Cause` is available for `errors.Is` and `errors.As`, but it is never serialized
to JSON. `Value` is redacted by default in JSON when the field name indicates
sensitive material such as token, auth, header, body, prompt, content or memory.

Use `core.Errors` when multiple failures can be reported together:

```go
var errs core.Errors
if errors.As(err, &errs) {
    if errs.Has(core.ErrorCodeUnknownExecutor) {
        // map at the consumer boundary
    }
}
```

## Stable Codes

Important code groups:

| Area | Codes |
| --- | --- |
| Vocabulary/config | `invalid_config`, `invalid_runtime_config`, `empty_operation`, `unknown_operation`, `empty_target`, `unknown_target`, `empty_executor`, `unknown_executor` |
| Schema/policygen | `invalid_policy_schema`, `schema_validation_failed`, `policy_build_failed`, `generated_policy_invalid`, `compat_breaking_change` |
| Runtime/facts | `evaluation_failed`, `context_canceled`, `context_deadline_exceeded`, `budget_exceeded`, `unsafe_rule`, `unsafe_negation`, `max_facts_exceeded`, `max_bindings_exceeded`, `max_iterations_exceeded` |
| Limits | `body_bytes_exceeded`, `max_tokens_exceeded`, `stream_not_allowed`, `tools_not_allowed`, `images_not_allowed` |

## Consumer Mapping

Recommended boundary behavior:

| Error family | HTTP-style mapping |
| --- | --- |
| invalid config/schema/runtime policy | fail startup or deployment check |
| request body budget exceeded | `413` |
| token/tools/images/stream disallowed | `403` |
| context canceled | caller-specific cancellation handling |
| context deadline exceeded | timeout mapping such as `504` |
| facts budget or unsafe rule | fail closed and alert |

These are recommendations only. CervoRules keeps transport semantics outside
core packages.

## Generated Policy Build Errors

Generated policies wrap build-time validation failures with
`runtime.PolicyBuildError`:

```go
engine, err := factory.Build(ctx, cfg)
if err != nil {
    var buildErr *runtime.PolicyBuildError
    if errors.As(err, &buildErr) {
        _ = buildErr.Metadata.PolicyHash
    }
    return err
}
```

`PolicyBuildError` exposes generated `PolicyMetadata` plus `core.Errors`, which
lets CI and agents identify the failing policy artifact.

## Facts Diagnostics

`facts.ComplexityDiagnostic` remains the warning/debug shape. Fatal complexity
or safety diagnostics can be converted to `core.Errors`:

```go
errs := facts.StructuredErrorsFromDiagnostics(result.Diagnostics)
if len(errs) > 0 {
    return errs
}
```

`expensive_rule` is intentionally non-fatal. It should tune rules and budgets,
not fail production traffic by itself.

## CLI JSON

For agent/CI workflows:

```powershell
go run ./cmd/cervorules-policygen check `
  -vocab policy-vocabulary.yaml `
  -policy policy-rules.yaml `
  -format json

go run ./cmd/cervorules-vocabgen `
  -in policy-vocabulary.yaml `
  -out internal/policyvocab/generated.go `
  -package policyvocab `
  -format json
```

Exit codes:

| Code | Meaning |
| --- | --- |
| `0` | success |
| `1` | generic failure |
| `2` | validation or usage failure |
| `3` | compatibility-breaking change |
| `4` | budget or complexity failure |
| `5` | internal tooling failure |
