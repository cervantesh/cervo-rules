# CervoRules v3 DecisionResult And Runtime Options

Tracks epic #326 and child issues #327, #328, #329, #330, and #331.

## Goal

v3 makes `DecisionResult` the only runtime envelope and removes ambiguous zero-value observation semantics. Runtime decisions stay deterministic, while explanation and operational reporting are explicit opt-ins.

## Contract

`DecisionResult` contains:

- `Decision`: required policy outcome.
- `Trace`: optional `*DecisionTrace`.
- `Observation`: optional `*DecisionObservation`.
- `Diagnostics`: structured `Errors`.
- `Stats`: cheap counters that do not require trace storage.

`DecisionOptions` is configured with functional options:

- `WithTrace(true)` enables trace materialization.
- `WithObservation(true)` enables operational observation materialization.

By default, trace and observation are opt-in. This is the production-oriented default for v3 hot paths.
Diagnostics are runtime output. Engines attach structured `Errors` to `DecisionResult.Diagnostics`; callers do not request diagnostics with decision options.

## Why This Is Breaking

v2 kept value-shaped trace and observation fields for compatibility. v3 uses pointers so missing trace or observation is unambiguous:

- `nil` means not requested or not materialized;
- non-nil means the caller explicitly asked for the data or the engine chose to emit it.

## Engine Contract

The v3 `Engine` interface returns `DecisionResult`:

```go
type Engine interface {
    Decide(context.Context, Request) (DecisionResult, error)
    DecideWithOptions(context.Context, Request, DecisionOptions) (DecisionResult, error)
}
```

Generated policy factories and hand-written runtimes should keep `Decide` cheap and use `DecideWithOptions` when callers need trace or observation.

## Issue Mapping

| Issue | Outcome |
| --- | --- |
| #327 | Trace and observation are optional values. |
| #328 | Zero-value observation semantics are replaced by nil/non-nil pointers. |
| #329 | Defaults are production-oriented and low-allocation. |
| #330 | Runtime behavior is configured with functional decision options. |
| #331 | Core examples and docs use `DecisionResult` and `DecisionOptions`. |
