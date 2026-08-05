# Advanced Performance Hardening 2026-05

Status: v2.1 RC workstream completed in repo code and docs. Package/release
evidence should still be collected again before tagging a new RC.

Related issues:

- #270 Epic: advanced performance hardening for v2.1 RC
- #271 perf: add pprof profiling suite and benchmark matrix
- #272 perf: add HTTP classifier no-copy headers fast path
- #273 perf: reduce core predicate trace allocations
- #274 perf: compact facts binding representation
- #275 perf: add facts rule planning and selective pattern ordering
- #276 perf: generate fast decision helpers for policies
- #277 perf: add optional consumer decision cache helper
- #278 ci: add benchmark history and soft regression checks
- #279 research: decide sync.Pool from pprof evidence
- #280 docs: refresh performance reports and operational guidance

## Why This Batch Exists

The first v2.1 RC performance batch established the main runtime guidance:
compile policies once, reuse compiled flows concurrently, prefer indexed
capability routing, use `DecideWithOptions` for hot paths, precompile HTTP
classifiers, and keep unbounded facts evaluation out of per-request critical
paths.

This advanced batch adds repeatable profiling and a benchmark matrix before
the next internal optimizations. That matters because the next candidates are
more sensitive than the first batch:

- compact facts bindings;
- selective facts rule planning;
- predicate trace allocation reduction;
- no-copy HTTP header classification;
- optional caller-owned cache wrappers through `decisioncache`;
- possible `sync.Pool` usage.

Those changes should be justified by evidence, not intuition.

## Completed PR Evidence

| PR | Issue | Area | Result | Performance impact |
| --- | --- | --- | --- | --- |
| #281 | #271 | Profiling and matrix | Added `scripts/performance/profile.sh`, matrix mode in `scripts/performance/report.sh`, ignored profile artifacts, and scale benchmarks. | Gives repeatable pprof and benchmark evidence before optimizing. |
| #282 | #272 | HTTP classifier | Added `HTTPClassificationOptions.OmitHeaders`. | Avoids full header map allocation when callers only need selected classification facts. |
| #283 | #273 | Core predicates | Added internal `predicateMatcher` fast path for built-in `All`, `Any`, `Not` when trace is disabled. | Reduces predicate child trace allocations without changing public `Predicate`. |
| #284 | #274 | Facts bindings | Added compact facts bindings and per-rule slot frames internally. | Reduces recursive facts map churn while keeping public `Binding` unchanged. |
| #285 | #276 | Generated policies | Generated `FastDecisionOptions` and `PolicyFactory.DecideFast`. | Makes trace-disabled generated-policy decisions easy for consumers. |
| #286 | #277 | Optional cache | Added opt-in `decisioncache` wrapper package. | Gives consumers caller-owned caching without adding global state to core. |
| #287 | #275 | Facts planning | Added conservative selective facts planning for larger positive rule bodies. | Improves join order while preserving negation and diagnostics safety. |
| #288 | #278 | Benchmark history | Added soft benchmark history checks and workflow artifacts. | Captures `ns/op`, `B/op`, and `allocs/op` trends without flaky hard gates. |
| #289 | #279 | sync.Pool decision | Documented the sync.Pool decision and revisit criteria. | Keeps pooling out of v2.1 RC until pprof identifies a narrow safe target. |

Wiki evidence: `Reports-Performance-2026-05` mirrors this batch for the CI
project wiki and links the operational guidance from Testing and Operations.

## Local Profiling

Use:

```bash
scripts/performance/profile.sh
```

By default the script writes CPU profiles, memory profiles and benchmark logs
to `dist-performance-profiles/`. That directory is intentionally ignored by
git.

To write elsewhere:

```bash
CERVORULES_PROFILE_OUT=/tmp/cervorules-profiles scripts/performance/profile.sh
```

To shorten exploratory runs:

```bash
CERVORULES_BENCHTIME=200ms scripts/performance/profile.sh
```

The script records focused profiles for:

- core decision flow;
- generated policy decisions;
- HTTP classifier;
- recursive facts reachability.

## Benchmark Matrix

The standard report remains:

```bash
scripts/performance/report.sh
```

The matrix report adds broader scale and CPU coverage:

```bash
scripts/performance/report.sh matrix
```

Benchmark history and soft regression workflow details live in
[performance-benchmark-history.md](performance-benchmark-history.md).

Matrix coverage includes:

- indexed routing at 100, 1000 and 10000 routes;
- generated policy decision benchmarks;
- HTTP classifier serial and parallel benchmarks;
- facts reachability at 100, 1000 and 10000 facts;
- low and high fanout facts fixtures;
- selective facts planning fixtures with three or more positive body patterns;
- CPU lists `1,4,8,16,32`.

The matrix defaults to one iteration per sub-benchmark because the largest
facts fixtures are intentionally expensive. Set `CERVORULES_BENCHTIME` when
you need a longer local run.

## Operational Guidance

For production hot paths:

- compile policies once at startup;
- reuse `*core.CompiledDecisionFlow` across goroutines;
- call generated `PolicyFactory.DecideFast` when trace and observation are not
  needed;
- use `FastDecisionOptions` or explicit `core.DecisionOptions` for manual
  flows;
- reuse `httpadapter.HTTPClassifier` and prefer `OmitHeaders` when the full
  copied header map is not needed;
- keep `facts` evaluation bounded with `MaxFacts`, `MaxIterations`, and
  `MaxBindings`;
- disable facts trace when explanations are not needed;
- use `decisioncache` only with caller-owned keys that include every volatile
  input affecting the decision;
- keep `LinearRoutingPhase` limited to small intentionally ordered rule sets.

For performance reviews:

- run `scripts/performance/report.sh` for focused evidence;
- run `scripts/performance/report.sh matrix` for release candidates or changes
  to `core`, `facts`, `httpadapter`, or `internal/policygen`;
- run `scripts/performance/profile.sh` before proposing pooling or larger facts
  evaluator changes;
- attach before/after benchmark lines to the issue and PR.

## Interpretation

These benchmarks are advisory during the RC. They are intended to:

- establish local and runner baselines;
- identify allocation hotspots before adding pools;
- compare serial and parallel behavior;
- catch obvious regressions before release notes are written.

They are not yet hard CI gates because shared runners can be noisy. Hard
thresholds should be added only after enough history exists in #278.

## sync.Pool Decision

#279 is resolved by [performance-sync-pool-decision-2026-05.md](performance-sync-pool-decision-2026-05.md).

The v2.1 RC decision is: Do not implement `sync.Pool` yet. The profile and
benchmark workflow exists, but current evidence favors removing work through
fast paths, no-copy classification, compact facts bindings, selective planning,
generated fast helpers, and optional caller-owned caching. Pooling needs pprof
evidence for a specific internal type plus a documented reset protocol before
it is accepted.

## Change Management

When a PR changes runtime performance behavior, attach:

- benchmark command;
- machine or runner context;
- before/after lines for the affected benchmark;
- whether the change affects public API;
- whether the change changes concurrency assumptions;
- whether `sync.Pool` remains unnecessary or needs investigation in #279.

Generated profile artifacts are local evidence. Do not commit `.pprof` files
or benchmark output directories.
