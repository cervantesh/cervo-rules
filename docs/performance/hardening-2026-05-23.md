# Performance Hardening Report 2026-05-23

This report records the performance hardening batch for CervoRules v2.1.0-rc.1.
It is the change-management companion for issues #251-#260 and PRs #261-#268.

## Scope

The batch focused on runtime behavior under load:

- `core.Decide` hot path allocations;
- concurrent reuse of compiled policies;
- generated-policy-shaped benchmarks;
- HTTP classifier reuse under parallel request handling;
- `facts` recursion cost and memory use;
- operational benchmark reporting in CI.

No public API was removed. `RoutingPhase` remains available but deprecated in
favor of explicit indexed or linear routing choices.

## Change Summary

| PR | Issue | Area | Change | Result |
| --- | --- | --- | --- | --- |
| #261 | #252 | Core runtime | Added `DecisionOptions` and `DecideWithOptions`. | Fast path can skip trace and observation envelope work. |
| #262 | #254 | HTTP adapter | Pre-normalized classifier matchers in `NewHTTPClassifier`. | Reusable classifier avoids per-request regex/method/path normalization. |
| #263 | #253 | Benchmarks | Added generated-policy-shaped benchmarks. | Keeps generated policy performance visible. |
| #264 | #255 | Facts | Cloned public `Set` once, then used mutable internal working state. | Avoids full set/index rebuilds per derived fact. |
| #265 | #256 | Facts | Added `TraceMode` with `TraceDisabled`. | Callers can skip derivation trace storage when explanations are not needed. |
| #266 | #257 | Facts | Added safe semi-naive recursive evaluation. | Recursive reachability now evaluates delta facts for safe positive rules. |
| #267 | #258 | Docs/examples | Added hot-path guidance and example. | Consumers have a copyable low-latency shape. |
| #268 | #260 | CI | Added advisory `Performance Report` workflow. | Benchmark evidence is produced in logs for sensitive paths. |

## Benchmark Evidence

Benchmarks were collected on a local development workstation and on the CI
runner. Values are advisory, not hard thresholds.

| Benchmark | Before | After | Notes |
| --- | ---: | ---: | --- |
| `BenchmarkDecisionFlowScaleIndexed1000` | `~1.7-1.8 us/op`, `928 B/op`, `9 allocs/op` | unchanged default | Existing behavior preserved. |
| `BenchmarkDecisionFlowScaleIndexed1000FastOptions` | not available | `~1.4-1.5 us/op`, `592 B/op`, `6 allocs/op` | Trace and observation disabled. |
| `BenchmarkDecisionFlowConcurrent` | `~1.4 KB/op`, `11 allocs/op` | unchanged default | Existing envelope preserved. |
| `BenchmarkDecisionFlowConcurrentFastOptions` | not available | `~632 B/op`, `8 allocs/op` | Better allocation profile under parallel load. |
| `BenchmarkHTTPClassifierPrecompiled` | regex helper path `~8-19 us/op`, `~8 KB/op`, `100 allocs/op` | precompiled `~0.6-1.6 us/op`, `344 B/op`, `3 allocs/op` | `NewHTTPClassifier` is the production path. |
| `BenchmarkGeneratedPolicyFastDecision` | not available | `~2.3-3.3 us/op`, `~784 B/op`, `18 allocs/op` | Generated-policy-shaped route table stays microsecond-scale. |
| `BenchmarkReachabilityAgenda` after predicate indexes | `~7-12 ms/op`, `~7.3-9.1 MB/op` | `~1.1-1.3 ms/op`, `~1.18 MB/op` | Semi-naive recursive evaluation is the main improvement. |
| `BenchmarkReachabilityAgendaTraceDisabled` | not available | `~1.1 ms/op`, `~1.08 MB/op` | Trace disabled reduces allocations further. |

## Operational Guidance

Use this shape for latency-sensitive consumers:

1. Compile policies once at startup.
2. Reuse `*core.CompiledDecisionFlow` across goroutines.
3. Build `httpadapter.HTTPClassifier` once with `NewHTTPClassifier`.
4. Use `DecideWithOptions` with trace and observation disabled when the caller
   does not need explanations or operational envelopes.
5. Keep normal route tables indexed through generated policies,
   `PolicyBuilder`, `CapabilityRoutingPhase`, or `OperationRoutingPhase`.
6. Use `facts` on request paths only with explicit budgets:
   `MaxFacts`, `MaxIterations`, and `MaxBindings`.
7. Disable facts trace with `TraceDisabled` when derivation explanations are not
   required.
8. Prefer prepared/static facts or snapshots when the same stable facts are
   reused across many requests.

The copyable example is `examples/performance-hot-path`.

## Parallelism Findings

- `CompiledDecisionFlow` remains immutable after compile and safe for
  goroutine reuse.
- `HTTPClassifier` stores immutable precompiled matchers and is safe to reuse.
- `facts.Engine` remains immutable. Each evaluation creates request-local state.
- The advisory benchmark workflow uses `-cpu 1,4,8,16` so changes show behavior
  under increasing Go scheduler parallelism.

## Risk And Guardrails

Accepted risks:

- Benchmark numbers vary by runner hardware and current load.
- `Performance Report` is advisory because shared runners can be noisy.
- Artifact upload is best-effort because the current CI action stack does not
  fully support all GitHub artifact APIs.
- `facts` is improved but still should not be used unbounded in every request.

Guardrails:

- Required Build remains the blocking merge gate.
- `Dependency Audit` and SonarQube continue to run on PRs.
- `scripts/performance/report.sh` records focused benchmark evidence.
- `docs/performance/hot-path.md` documents the production shape.
- `file_size_test.go` keeps Go source files under 500 lines; the semi-naive
  implementation was split into `facts/semi_naive.go`.

## sync.Pool Decision

Issue #259 remains open by design.

The batch reduced allocations through safer changes first:

- fewer core runtime envelope allocations;
- precompiled classifier matchers;
- mutable internal facts working state;
- optional facts trace;
- semi-naive recursive evaluation.

Current evidence does not justify adding `sync.Pool`. Pooling should only be
considered after pprof/allocation profiles from realistic consumer-shaped
workloads identify one concrete internal type whose pooling benefit exceeds the
complexity and race-safety cost.

## Verification

Local verification run across the batch:

```bash
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
go test -race ./core ./runtime ./httpadapter ./limits
go test -race ./facts
```

Focused benchmark commands:

```bash
go test -run '^$' -bench 'BenchmarkDecisionFlowScaleIndexed1000|BenchmarkDecisionFlowConcurrent' -benchmem -cpu 1,4,8,16 ./core
go test -run '^$' -bench 'BenchmarkHTTPClassifier' -benchmem -cpu 1,4,8,16 ./httpadapter
go test -run '^$' -bench 'BenchmarkGeneratedPolicy' -benchmem -cpu 1,4,8,16 ./core
go test -run '^$' -bench 'BenchmarkReachabilityAgenda' -benchmem ./facts
```

CI verification:

- Required Build passed on merged PRs.
- Dependency Audit passed on merged PRs.
- SonarQube passed on merged PRs.
- Performance Report passed after artifact upload was made best-effort.

## Follow-Up

- Keep #259 open for allocation-profile based `sync.Pool` research.
- Refresh wiki performance/maturity reports when the next RC report is updated.
- Use the advisory workflow output as release evidence for performance-sensitive
  changes.
