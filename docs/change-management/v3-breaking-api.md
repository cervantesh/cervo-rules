# CervoRules v3 Breaking API Change Management

Epic: #302

Status: planned.

## Objective

CervoRules v3 is the next breaking line. The goal is to remove compatibility
surfaces that v2 intentionally kept, reduce ambiguity in public names, make
modular imports the normal path, and strengthen machine-readable contracts for
agents and consumers.

v3 should be implemented as a sequence of small PRs. Do not combine unrelated
API, generator, facts, docs, release, and supply-chain work in one PR.

## Defaults

- Module path: `github.com/cervantesh/cervo-rules/v3`.
- v2 remains available during v3 RC work.
- v3 removes deprecated public aliases instead of carrying them forward.
- New public names are `Operation`, `Target`, and `Executor`.
- `facts` remains optional and separate from hot-path core.
- HTTP remains an adapter.
- Production defaults prefer low-allocation behavior; trace and observation are
  opt-in unless a workstream checkpoint changes that decision.

## Workstream Sequence

| Sequence | Issue | Workstream |
| --- | --- | --- |
| 1 | #303 | v3 module path and repository layout |
| 2 | #308 | Rename primitives to Operation, Target, Executor |
| 3 | #314 | Shrink root facade and enforce modular imports |
| 4 | #320 | Structured error model |
| 5 | #326 | DecisionResult and runtime options cleanup |
| 6 | #332 | Checkpoint: reevaluate workstreams 1-5 |
| 7 | #336 | Generated policy factory as only generated runtime API |
| 8 | #342 | Policygen schema and DSL breaking cleanup |
| 9 | #348 | Facts as explicit optional logic engine |
| 10 | #354 | Remove implicit or ambiguous linear routing |
| 11 | #360 | Checkpoint: reevaluate workstreams 7-10 |
| 12 | #364 | Observability schema versioning |
| 13 | #370 | Machine-readable contracts |
| 14 | #376 | Migration tooling |
| 15 | #382 | Consumer conformance and compatibility suite |
| 16 | #388 | Checkpoint: reevaluate workstreams 12-15 |
| 17 | #392 | Release, packages and supply chain |
| 18 | #398 | Final docs and wiki refresh |
| 19 | #404 | Release candidate tag |
| 20 | #410 | Post-RC review and GA decision |
| 21 | #416 | Checkpoint: reevaluate workstreams 17-20 |

## Checkpoint Rule

Every five workstreams must be followed by a checkpoint before the next group
starts. A checkpoint must decide:

- what to keep;
- what to remove;
- what to defer;
- what new opportunity is worth adding;
- whether any public API decision needs an ADR.

Checkpoint outputs:

- `docs/change-management/v3-checkpoint-1.md` after #303-#326;
- `docs/change-management/v3-checkpoint-2.md` after #336-#354;
- `docs/change-management/v3-checkpoint-3.md` after #364-#382;
- `docs/change-management/v3-final-reevaluation.md` after #392-#410.

Checkpoint 1 is complete in `docs/change-management/v3-checkpoint-1.md`.
Checkpoint 2 is complete in `docs/change-management/v3-checkpoint-2.md`.
Checkpoint 3 is complete in `docs/change-management/v3-checkpoint-3.md`.
Post-RC review is complete in `docs/v3/post-rc-review.md`.
Final reevaluation is complete in `docs/change-management/v3-final-reevaluation.md`.

## Implementation Rules

- Each child issue must record estimate, start time, actual time, deviation,
  linked PR, tests and release/maturity impact.
- TDD is required for behavior changes.
- Generated code changes must include generated tests and freshness checks.
- Public API removals must be listed in the migration guide.
- No CervoProxy-specific vocabulary may enter core.
- Any deferred item must state target version and rationale.

## Required Verification

Every implementation PR:

```bash
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
```

API, schema and generator PRs:

```bash
go test -count=1 ./internal/policygen ./cmd/cervorules-policygen ./testkit
```

Facts or performance-sensitive PRs:

```bash
go test -race ./core ./facts ./runtime ./httpadapter ./limits
scripts/performance/report.sh
```

Before `v3.0.0-rc.1`:

- clean temp consumer import;
- generated examples compile;
- v2-to-v3 migration report against fixtures;
- generic package verification;
- release/package smoke;
- OCI verification if enabled;
- release notes and wiki report updated.
