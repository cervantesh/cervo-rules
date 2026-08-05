# CervoRules v3 Structured Errors

Tracks epic #320 and child issues #321, #322, #323, #324, and #325.

## Goal

v3 uses structured errors so consumers and agents can react to stable codes instead of parsing prose. This applies to runtime config validation, limits violations, facts diagnostics, generated policy validation, and vocabulary validation.

## Public Contract

The common contract starts in `v3/core`:

- `ErrorCode`: stable machine-readable identifier.
- `Error`: single structured error with `Code`, `Field`, `Value`, `Reason`, and optional `Cause`.
- `Errors`: multi-error collection with stable code lookup.

Initial vocabulary codes:

| Code | Field | Meaning |
| --- | --- | --- |
| `empty_operation` | `operation` | An operation was required but normalized to empty. |
| `unknown_operation` | `operation` | An operation is not present in the active vocabulary. |
| `empty_target` | `target` | A target was required but normalized to empty. |
| `unknown_target` | `target` | A target is not present in the active vocabulary. |
| `empty_executor` | `executor` | An executor was required but normalized to empty. |
| `unknown_executor` | `executor` | An executor is not present in the active vocabulary. |
| `invalid_rule` | `rule` | A rule-level validation error. |

## Adoption Plan

Runtime config validation must return `Error` or `Errors` with stable codes instead of formatted strings. The first v3 runtime config implementation should map invalid operations, targets, executors, timeouts, fallbacks, retry, breaker, and limits values to documented codes.

Limits violations must use the same structured shape. Consumers remain responsible for mapping a limits violation to HTTP status codes or other transport behavior.

Facts diagnostics should keep rich diagnostic data, but any error surfaced to callers should expose stable error codes. Expensive-rule diagnostics and complexity budget failures should be machine-readable.

## Compatibility Notes

v2 error strings are unchanged. v3 consumers should use `errors.As` for `core.Error` or `core.Errors.Has` instead of comparing strings.
