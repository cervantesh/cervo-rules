# CervoRules v3 Structured Errors

Tracks epic #320 and child issues #321, #322, #323, #324, and #325.

## Goal

v3 uses structured errors so consumers and agents can react to stable codes instead of parsing prose. This applies to runtime config validation, limits violations, facts diagnostics, generated policy validation, and vocabulary validation.

## Public Contract

The common contract starts in `v3/core`:

- `ErrorCode`: stable machine-readable identifier.
- `Error`: single structured error with `Code`, `Field`, `Value`, `Reason`, and optional `Cause`.
- `Errors`: multi-error collection with stable code lookup.

Every code is listed below. The list is enforced: `core` has a test that pins
each code's wire string and fails when a code is declared without appearing
here, because a code that reaches a consumer's audit record and is documented
nowhere is not a contract.

### Vocabulary and rules

| Code | Meaning |
| --- | --- |
| `empty_operation` | An operation was required but normalized to empty. |
| `unknown_operation` | An operation is not present in the active vocabulary. |
| `empty_target` | A target was required but normalized to empty. |
| `unknown_target` | A target is not present in the active vocabulary. |
| `empty_executor` | An executor was required but normalized to empty. |
| `unknown_executor` | An executor is not present in the active vocabulary. |
| `invalid_rule` | A rule-level validation error. |
| `invalid_config` | A configuration value is not usable. |
| `renamed_primitive` | A v2 primitive name was used where a v3 name is required. |
| `unsupported_feature` | The requested feature is not part of this generator's scope. |
| `deprecated_field` | A field kept only for migration was used. |

### Runtime config

| Code | Meaning |
| --- | --- |
| `invalid_runtime_config` | A `PolicyRuntimeConfig` value failed validation. |

### Generation and compatibility

| Code | Meaning |
| --- | --- |
| `invalid_policy_schema` | Policy YAML does not match the supported shape. |
| `schema_validation_failed` | A document failed validation against a published JSON Schema. |
| `policy_build_failed` | A generated `PolicyFactory` refused to build; wraps the cause with policy metadata. |
| `generated_policy_invalid` | Generated policy source is not usable. |
| `deprecated_generated_api` | Generated code exposes an API that v3 removed. |
| `compat_breaking_change` | A compatibility comparison found a breaking change. |
| `internal_invariant_failed` | The generator reached a state it believes impossible; report it. |

### Decision evaluation

| Code | Meaning |
| --- | --- |
| `evaluation_failed` | A decision could not be produced. |
| `context_canceled` | The caller's context was cancelled mid-decision. |
| `context_deadline_exceeded` | The caller's deadline elapsed mid-decision. |

### Facts engine

These come from the optional `facts` package and its budgets.

| Code | Meaning |
| --- | --- |
| `budget_exceeded` | An evaluation budget was exhausted. |
| `unsafe_rule` | A rule is not range-restricted. |
| `unsafe_negation` | Negation was used outside a safe stratum. |
| `max_facts_exceeded` | Materialization passed `MaxFacts`. |
| `max_bindings_exceeded` | Binding expansion passed `MaxBindings`. |
| `max_iterations_exceeded` | Fixpoint iteration passed `MaxIterations`. |
| `expensive_rule` | A rule crossed the expensive-rule binding threshold. |
| `rule_disabled` | A rule was skipped because it is disabled. |

### Conditions and facts a policy reads

The seam a policy consults before allowing an operation. Every one of these is
a refusal to answer, never a "the guard ran and found nothing wrong".

| Code | Meaning | What a consumer should do |
| --- | --- | --- |
| `unknown_condition` | A policy required a condition the evaluator does not register. | Fail the request. The guard was never wired up. |
| `condition_failed` | A condition evaluator returned an error. | Fail the request. |
| `missing_conditions` | The policy declares conditions but was built without an evaluator. | Fix the build; this is a wiring error, not a request error. |
| `missing_fact` | A fact a predicate reads is absent from `Request.Metadata` and the policy declares no default. | Treat as a malformed request, not as a denial. |
| `invalid_fact` | A fact is unparseable, non-finite, or outside its declared bounds. | Treat as a malformed request. `Error.Value` carries the observed value and is marked `Sensitive`. |

### Limits

From the optional `limits` package, applied after a route is chosen.

| Code | Meaning |
| --- | --- |
| `body_bytes_exceeded` | Request body passed `MaxBodyBytes`. |
| `max_tokens_exceeded` | Requested tokens passed `MaxTokens`. |
| `stream_not_allowed` | Streaming was requested where the budget forbids it. |
| `tools_not_allowed` | Tool use was requested where the budget forbids it. |
| `images_not_allowed` | Images were requested where the budget forbids it. |

## Adoption Plan

Runtime config validation must return `Error` or `Errors` with stable codes instead of formatted strings. The first v3 runtime config implementation should map invalid operations, targets, executors, timeouts, fallbacks, retry, breaker, and limits values to documented codes.

Limits violations must use the same structured shape. Consumers remain responsible for mapping a limits violation to HTTP status codes or other transport behavior.

Facts diagnostics should keep rich diagnostic data, but any error surfaced to callers should expose stable error codes. Expensive-rule diagnostics and complexity budget failures should be machine-readable.

## Compatibility Notes

v2 error strings are unchanged. v3 consumers should use `errors.As` for `core.Error` or `core.Errors.Has` instead of comparing strings.
