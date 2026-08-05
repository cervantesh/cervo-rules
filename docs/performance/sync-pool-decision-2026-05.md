# sync.Pool Performance Decision 2026-05

Status: accepted for v2.1 RC.

Related issue: #279.

## Decision

Do not implement `sync.Pool` in the v2.1 RC runtime.

The current performance evidence points to algorithmic and API-shape wins as
the high-value work:

- `DecisionOptions` avoids trace and observation work on hot paths.
- HTTP no-copy mode removes header map allocation when callers do not need
  copied headers.
- Compact facts bindings reduce map churn inside recursive evaluation.
- Selective facts planning improves join order without public API changes.
- Generated fast helpers make the trace-disabled path easy for generated
  policies.

Those improvements remove work before it is allocated. `sync.Pool` would only
hide some remaining allocations, and the current profile workflow has not
identified a single small, frequently allocated, safely resettable internal
object that dominates CPU or memory.

## Why Not Pool Now

`sync.Pool` is useful when a profile shows a clear allocation hotspot and the
object has a reliable reset protocol. CervoRules does not yet meet that bar for
v2.1 RC.

The main risks are:

- retained memory under bursty workloads;
- stale state leaking across decisions or fact evaluations;
- harder race analysis for a runtime intended to be shared across goroutines;
- less predictable memory behavior on small consumers;
- extra implementation complexity before pprof evidence justifies it.

## Revisit Criteria

These are the revisit criteria for changing the decision.

Reopen pooling research only when `scripts/performance/profile.sh` or runner
profiles show all of the following:

- one internal allocation type is a top contributor in `pprof` memory output;
- the type is not part of public API output;
- the type can be reset completely with a documented reset protocol;
- race tests pass with pooling enabled;
- benchmark evidence shows meaningful improvement in both `B/op` and
  `allocs/op`;
- the implementation does not make normal non-pooled code paths harder to
  audit.

Acceptable future candidates must be narrow and internal, for example temporary
trace builders or facts evaluator scratch buffers. Public `Request`, `Decision`,
`DecisionResult`, `Binding`, and `Set` values should not be pooled by the core.

## Required Evidence For A Future Change

A future pooling PR must include:

- before/after pprof CPU and memory artifacts from local or runner execution;
- before/after benchmark lines for the affected benchmark;
- race detector output for the affected packages;
- a reset protocol documented beside the pooled type;
- a rollback note explaining how to remove the pool if it regresses memory.

Until that evidence exists, the preferred performance strategy remains:

- remove unnecessary work;
- precompute immutable structures;
- use compact internal representations;
- keep caches caller-owned;
- keep pooling out of the core runtime.
