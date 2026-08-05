# CervoRules v2 GA Maturity Closure Change Management

Parent epic: #193

## Purpose

This change program closes the maturity gaps found after the facts workstream
and prepares CervoRules v2 for a stronger beta/RC checkpoint.

The work is intentionally sequenced. Facts runtime changes, policygen DSL
changes, generator refactors, report refreshes, and release validation have
different blast radii and must not land in one large PR.

## Current Baseline

Evidence before this program:

- PRs #184 through #192 completed the optional facts layer foundation,
  serialization, redaction, freshness validation, linting, explanations,
  semantic ADRs, and changed-predicate agenda optimization.
- `go test -cover ./...` keeps all statement packages at or above 90%.
- Go files remain under the 500-line budget.
- Required Build and Dependency Audit pass in CI after runner cache issues
  are cleared.
- The only open non-PR issue before this program was #123, which tracks moving
  the runner off the laptop when a dedicated machine exists.

## Workstreams

| Issue | Workstream | Merge position |
| --- | --- | --- |
| #194 | Change-management plan | First |
| #195 | Parallel-safe release tests | First |
| #196 | Roadmap and report refresh | First |
| #197 | Declarative facts DSL parser/validator | Before policygen integration |
| #198 | Optional policygen derived facts integration | After #197 |
| #199 | Stratified negation runtime | After #197 |
| #200 | Controlled aggregates runtime | After #197 and #199 guardrails |
| #201 | Partial evaluation API | After facts DSL and explain contracts |
| #202 | Incremental evaluation API | After #201 and benchmark evidence |
| #203 | Policygen complexity hardening | After #198 |
| #204 | Post-facts beta/RC release smoke | Last |

## Merge Strategy

1. Land a foundation PR for #194, #195, and #196.
2. Land facts DSL support in the `facts` package without touching policygen.
3. Land policygen support only after the facts DSL has focused tests and docs.
4. Land runtime advanced facts features in independent PRs:
   stratified negation, aggregates, partial evaluation, incremental evaluation.
5. Refactor `internal/policygen` only after derived facts integration exposes
   the real complexity pressure.
6. Run non-release smoke validation and update reports before any beta/RC tag.

## TDD Rules

- Code changes start with failing focused tests.
- Runtime facts changes must include equivalence or diagnostic tests.
- Policygen changes must include schema, parser/check, generation, and generated
  consumer tests.
- Release test changes must demonstrate isolation from parallel test runs.
- Docs-only changes must run unresolved-marker scans over changed docs.

## Risk Matrix

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| Facts DSL couples core to facts | Medium | High | Keep DSL in `facts`; policygen import is optional when YAML declares derived facts. |
| Policygen complexity rises again | High | Medium | Add structured tests and split files under #203 if any file nears 500 lines. |
| Negation surprises consumers | Medium | High | Only implement stratified negation with validation and closed-world docs. |
| Aggregates create high-cardinality output | Medium | Medium | Start with bounded aggregate specs and diagnostics. |
| Partial/incremental evaluation changes semantics | Medium | High | Require equivalence tests against full evaluation. |
| Release tests collide under parallel runs | Medium | Medium | Unique temp dirs per test invocation; avoid fixed repo temp paths. |
| Report/wiki drift | High | Medium | Update repo reports in #196 and wiki checklist before release smoke. |
| Runner infrastructure flakiness | Medium | Medium | Keep #123 open; record CI reruns caused by checkout/cache as infrastructure. |

## Rollback

- Revert the PR that introduced the failing behavior.
- Do not rewrite published tags or package versions.
- If a release smoke fails after publishing a smoke package, publish a new smoke
  version instead of mutating artifacts.
- If policygen derived facts integration is blocked, leave the facts DSL
  package API intact and defer policygen support with an ADR update.

## Release Validation Gate

Before any new beta/RC tag:

```text
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
scripts/release/check.sh v2.0.1-smoke.<date>
```

If package verification is enabled after publishing:

```text
CERVORULES_VERIFY_PACKAGE=1 scripts/release/check.sh <version>
```

## Done Criteria

- Issues #194 through #204 are closed or explicitly deferred by ADR.
- `docs/v2-ga-roadmap.md` and `docs/reports/v2-ga.md` cite current evidence.
- Release artifact tests are safe under concurrent local invocations.
- Facts DSL and policygen integration remain optional.
- Final release smoke evidence is recorded in #204.
