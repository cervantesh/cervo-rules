# Performance And Quality Gates

Use `scripts/ci/quality-gates.sh` to run the repeatable local verification set for
CervoRules v2 hardening.

For the performance hardening change-management report, see
[Performance Hardening Report 2026-05-23](performance-hardening-2026-05-23.md).

## Standard Gate

The standard gate is intended for every PR:

```bash
scripts/ci/quality-gates.sh
```

It runs:

- `go test -count=1 ./...`;
- `go test -cover ./...`;
- `go vet ./...`;
- `go mod verify`.

## Extended Gate

The extended gate is intended before releases, performance-sensitive changes,
and runner validation:

```bash
scripts/ci/quality-gates.sh extended
```

It adds:

- `go test -race ./...`;
- `go test -run Fuzz -fuzz=FuzzPolicyCheck -fuzztime=10s ./internal/policygen`;
- `go test -bench=DecisionFlow -benchmem -run '^$' ./...`.

Set `CERVORULES_FUZZTIME` to tune fuzz smoke duration.

## Benchmark Baseline

Record benchmark output in issue or release evidence when routing performance is
part of the change:

```bash
go test -bench=DecisionFlow -benchmem -run '^$' ./...
```

For the focused advisory performance report, run:

```bash
scripts/performance/report.sh
```

The `Performance Report` workflow runs the same script for performance-sensitive
PR paths and pushes to `main`. It is advisory: Required Build remains the merge
gate, while benchmark output is kept in logs. Artifact upload is best-effort
because not every CI runner/action stack supports GitHub artifact APIs.

Compare `BenchmarkDecisionFlowScale`, `BenchmarkDecisionFlowScaleIndexed1000`,
and `BenchmarkDecisionFlowConcurrent` before and after behavior changes.

For routing or facts evaluator changes, cite the PR #247 baseline unless a newer
benchmark artifact supersedes it. The critical comparison is linear
`LinearRoutingPhase` behavior versus indexed `CapabilityRoutingPhase`/
`OperationRoutingPhase`, plus facts agenda behavior before and after predicate
and term index changes.
