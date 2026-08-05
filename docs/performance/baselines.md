# Performance Baselines

CervoRules performance review focuses on deterministic routing cost, allocation
behavior, and safety under concurrent evaluation.

The implementation narrative for the 2026-05-23 hardening batch is recorded in
[Performance Hardening Report 2026-05-23](performance-hardening-2026-05-23.md).

## Benchmark Commands

Run:

```bash
go test -bench=DecisionFlow -benchmem -run '^$' ./...
```

Core benchmarks:

- `BenchmarkDecisionFlowScale`: linear routing phase at 10, 100, and 1000 rules.
- `BenchmarkDecisionFlowScaleIndexed1000`: indexed capability routing at 1000
  routes.
- `BenchmarkDecisionFlowConcurrent`: concurrent decisions over a representative
  mixed policy.
- `BenchmarkReachabilityAgenda`: facts evaluation fixture for recursive
  reachability.

Current local benchmark evidence from 2026-05-23.
PR #247 is the routing baseline: it deprecated the ambiguous linear helper,
added explicit `LinearRoutingPhase`, documented `CapabilityRoutingPhase` as the
normal route-table path, and added facts predicate and term indexes. PRs
#261-#266 are the newer hot-path evidence points for fast decision options,
HTTP classifier precompilation, generated policy benchmarks, mutable facts
working set, trace-disabled facts evaluation, and semi-naive recursive facts.

| Benchmark | Result | Allocation Notes |
| --- | ---: | ---: |
| `BenchmarkDecisionFlowScale/1000_rules` | `~339 us/op` | `~230 KB/op`, linear route scan |
| `BenchmarkDecisionFlowScaleIndexed1000` | `~1.8 us/op` | `~930 B/op`, indexed by capability |
| `BenchmarkDecisionFlowScaleIndexed1000FastOptions` | `~1.4 us/op` | `~592 B/op`, trace and observation disabled |
| `BenchmarkHTTPClassifierPrecompiled` | `~0.6 us/op` | `~344 B/op`, regex/method/path matchers precompiled |
| `BenchmarkGeneratedPolicyFastDecision` | `~3.3 us/op` | `~785 B/op`, generated-policy-shaped benchmark |
| `BenchmarkReachabilityAgenda` before facts indexes | `~24.7 ms/op` | `~23 MB/op`, broad scans and set rebuilds |
| `BenchmarkReachabilityAgenda` after facts predicate and term indexes | `~9-12 ms/op` | `~9.1 MB/op`, still not recommended for unbounded per-request hot paths |
| `BenchmarkReachabilityAgenda` after semi-naive | `~1.2 ms/op` | `~1.18 MB/op`, bounded recursive facts |
| `BenchmarkReachabilityAgendaTraceDisabled` after semi-naive | `~1.1 ms/op` | `~1.08 MB/op`, no derivation trace storage |

`RoutingPhase` is deprecated because the name hides its linear cost. Prefer
`CapabilityRoutingPhase`, `OperationRoutingPhase`, or `PolicyBuilder` for route
tables. Use `LinearRoutingPhase` only when global rule order is intentional.

## Large Policy Stress Fixture

`TestLargePolicyStressFixtureIndexedRouting` compiles an indexed policy with
2500 capability routes and verifies first, middle, and last route decisions. It
is not a microbenchmark; it is a regression fixture that proves large generated
policy shapes remain practical in normal `go test`.

## Regression Review

When a PR changes routing, predicates, actions, generated policy structure, or
runtime config merge behavior:

- run `scripts/ci/quality-gates.sh`;
- run `scripts/ci/quality-gates.sh extended` before release or when the change is
  performance-sensitive;
- paste benchmark output into the issue or PR when routing complexity changes;
- compare indexed routing against linear routing before changing default
  generated policy structure.

Treat large allocation growth, significant indexed-routing slowdown, or flaky
concurrency/race output as release blockers until explained.
