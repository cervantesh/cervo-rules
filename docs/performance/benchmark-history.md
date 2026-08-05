# Performance Benchmark History

Status: advisory for v2.1 RC.

Issue: #278.

## Purpose

The performance workflow records repeatable benchmark logs before hard
thresholds exist. Shared runners can be noisy, so the first gate is advisory:
collect standard and matrix reports, keep them as artifacts, and compare
`ns/op`, `B/op`, and `allocs/op` during performance-sensitive PR review.

## Workflow Outputs

The `Performance Report` workflow uploads:

- `performance-report.txt`;
- `performance-report-matrix.txt`.

The artifact name includes the CI run id so multiple runs can be compared.

## Soft Regression Check

Use:

```bash
scripts/performance/benchmark-history-check.sh performance-report.txt
```

With a local baseline:

```bash
scripts/performance/benchmark-history-check.sh performance-report.txt previous-performance-report.txt
```

The script is non-blocking by default. Set `CERVORULES_BENCH_STRICT=1` only
after enough runner history exists to define stable thresholds.

## Review Guidance

For every performance-sensitive PR, record:

- command;
- runner or laptop context;
- affected benchmark names;
- before/after `ns/op`;
- before/after `B/op`;
- before/after `allocs/op`;
- whether the change affects concurrent use.

Hard gates should wait until the project has several comparable runs from the
same runner class.
