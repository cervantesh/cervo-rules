# Hot Path Performance Guidance

CervoRules is safe to use on request paths when expensive work is moved to
startup and shared runtime objects are reused.

## Startup

- Compile policies once with `Compile()` or a generated `PolicyFactory`.
- Reuse `*core.CompiledDecisionFlow` across goroutines.
- Build one `httpadapter.HTTPClassifier` with `NewHTTPClassifier`; do not call
  the package-level helper for high-volume routes with regex matchers.
- Set `HTTPClassificationOptions.OmitHeaders` when the caller only needs
  selected classification headers and does not need the full header map in
  `RequestFacts.Headers`.
- Precompute stable `facts` derivations with `Prepare` or snapshots when
  request-time derivation is not required.

## Request Path

- Use `DecideWithOptions` with trace and observation disabled for low-latency
  decisions where callers do not need per-request explanations:

```go
result, err := engine.DecideWithOptions(ctx, req, core.DecisionOptions{
    Trace:       core.DecisionOptionDisabled,
    Observation: core.DecisionOptionDisabled,
})
```

- Keep route tables indexed with generated policies, `PolicyBuilder`,
  `CapabilityRoutingPhase`, or `OperationRoutingPhase`.
- Use `LinearRoutingPhase` only for small, intentionally ordered rule lists.
- Use `decisioncache` only when the caller can build a stable key and knows the
  decision does not depend on volatile health, rollout, trust, or request-body
  state that is absent from the key.
- If `facts` must run per request, set `MaxFacts`, `MaxIterations`, and
  `MaxBindings`; disable trace when explanations are not needed.

## Concurrency

`CompiledDecisionFlow` and `HTTPClassifier` are immutable after construction and
can be shared by many goroutines. `facts.Engine` is also immutable; evaluation
creates request-local working state internally.

## Example

See `examples/performance-hot-path` for a small HTTP-shaped example with:

- compiled policy singleton;
- precompiled HTTP classifier singleton;
- fast decision options;
- prepared stable facts outside the request path.
