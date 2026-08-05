# Documentation Refresh 2026-05-23

Issue: #249

Scope: repository docs plus project wiki.

This refresh is documentation-only. It reconciles public docs after PR #247
performance/routing/facts work, issue #248 runner hook hardening, SonarQube
setup, modular imports, packages, v2.1.0-rc.1 release planning, and report/wiki
drift checks.

## Reviewed Surfaces

| Surface | Review result |
| --- | --- |
| README / AGENTS / CHANGELOG / manifest | Current RC, modular imports, SonarQube, performance, and issue references reviewed. |
| Main docs | Performance, release, packages, runner operations, agent workflow, facts, and roadmap docs reviewed. |
| ADRs | ADRs reviewed for factual drift. Facts ADRs remain historical design records unless amended; current implemented state is documented in runtime docs. |
| examples | Neutral examples and logic facts example reviewed for current optional-facts and performance guidance. |
| reports / roadmap | Repo reports now cite PR #247, issue #248, and this docs issue #249 as evidence. |
| wiki pages | Wiki pages are refreshed as summary surfaces. Historical report pages are marked or treated as historical snapshot material when appropriate. |
| tests / drift checks | Documentation drift tests cover performance evidence, runner operations, release docs, and report references. |

## Current Documentation Decisions

- `RoutingPhase` remains documented only as deprecated compatibility.
- `LinearRoutingPhase` is the explicit name for small ordered rule lists.
- `CapabilityRoutingPhase`, `OperationRoutingPhase`, and `PolicyBuilder` remain
  the preferred route-table path.
- Facts are optional and implemented for bounded Datalog-inspired derivation,
  stratified negation, controlled aggregates, partial preparation, incremental
  snapshots, and generated-policy optional helpers.
- Materialized fact views and a facts explain CLI remain documented patterns or
  designs, not runtime features.
- Runner permission repair must stay bounded to a job workspace; global
  `chown -R` repairs are an operational smell.

## Wiki Sync Notes

The wiki is the human-readable reporting surface. Repository docs remain the
source of truth for tests, scripts, and versioned change evidence. Wiki updates
for this issue should summarize:

- performance baseline from PR #247;
- runner hook hardening from issue #248;
- SonarQube as additive quality analysis;
- package and RC verification expectations;
- this documentation refresh issue #249.
