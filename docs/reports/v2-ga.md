# CervoRules v2 GA Reports

This report is the repository-side checkpoint for the v2 GA hardening reports.
The wiki remains the long-form reporting surface, but this file keeps release
evidence next to the code and is covered by tests.

## Postmortem

No production runtime incident has been attributed to the v2 API line. The
release/package smoke postmortem remains the main operational learning: package
verification must prove generic artifacts, OCI artifacts, checksums, schemas,
tool identity, and runner network reachability separately.

For v2 GA, every near miss should record:

- affected consumer or generated policy shape;
- detection path;
- corrective test or gate;
- linked issue and PR;
- architecture or process change if the fix changes public behavior.

## Maturity

Current maturity direction is positive but still beta until the remaining GA
hardening issues close. Every maturity claim in this report must cite at least one issue, PR, test, script, ADR, or release command. Recent evidence:

| Evidence | Source | Maturity impact |
| --- | --- | --- |
| Performance gates and extended release command | #90, `scripts/ci/quality-gates.sh extended` | Release confidence and regression detection |
| Performance hardening batch | #251-#260, PR #261-#268, `docs/performance/hardening-2026-05-23.md` | Documents core, HTTP, generated-policy, facts, parallelism, and advisory CI performance evidence |
| Large indexed routing stress fixture | #90, `TestLargePolicyStressFixtureIndexedRouting` | Scale confidence for generated policies |
| Decision observability report and audit schema | #91, `PolicyEvaluationReport` | Operational maturity and consumer diagnostics |
| Redaction policy | #91, `PrivacySafeRedactionPolicy` | Privacy/cardinality maturity |
| Optional facts module foundation | PR #184 | Cross-domain derived facts without core coupling |
| Facts schema, constraints, ordering, and strata | PR #185, PR #186 | Safer authoring and deterministic derivation |
| Facts serialization, redaction, freshness, linting, and explanations | PR #187, PR #189, PR #190, PR #191 | Operational diagnostics and auditability |
| Changed-predicate agenda optimization | PR #192 | Runtime efficiency while preserving fixed-point semantics |
| Declarative facts DSL and optional policygen helpers | #197, #198 | Generated policies can carry optional facts rules without making core depend on facts |
| Stratified negation, controlled aggregates, partial evaluation, and incremental snapshots | #199, #200, #201, #202 | Advanced Datalog-inspired runtime semantics moved from ADR-only to tested runtime APIs |
| Policygen generated facts contract hardening | #203 | AST contracts and test ownership reduce drift after facts integration |
| Post-facts release smoke | #204, `scripts/release/check.sh v2.0.1-smoke.20260522 dist-release-check-v2.0.1-smoke.20260522` | Standard, extended, race, fuzz, benchmark, govulncheck, and artifact build gates passed |
| v2 GA maturity closure epic | #193 | Current post-facts closure plan |
| v2.1.0-rc.1 docs drift gate | #221, `TestV21RCRoadmapUsesIntentionalIssueStatuses`, `TestV2GAReportWikiChecklistMatchesRepositoryInventory` | Prevents roadmap, report, wiki, snippet, changelog, and maturity-evidence drift before #222 release work |
| Agent usability and agnosticism hardening | #231 through #238, `AGENTS.md`, `.cervorules/agent-manifest.json`, `docs/agent-quickstart.md`, `TestAgentEntrypointDocumentsRequiredWorkflow`, `TestAgentManifestIsValidAndCurrent` | Gives AI agents a canonical entrypoint, machine-readable repo map, and tested current-version guidance |
| Routing and facts performance refresh | PR #247, `BenchmarkDecisionFlowScaleIndexed1000`, `BenchmarkReachabilityAgenda`, `TestLargePolicyStressFixtureIndexedRouting` | Documents the routing performance split between indexed route tables and explicit linear routing, plus facts predicate and term index evidence |
| Full documentation refresh | #249, `docs/documentation-refresh-2026-05-23.md`, `TestDocsRefreshCoversPerformanceAndRunnerOperations` | Ties repo docs and wiki report refresh to explicit drift tests after performance, operations, SonarQube, packages, and RC evidence changes |

## Agnosticism

Core remains stack-neutral: no logging, metrics, tracing, audit-store, HTTP
framework, OpenTelemetry, or Prometheus dependency is required by the runtime.
Adapters stay optional and consumer-owned. The v2 report/audit contract uses
bounded fields and diagnostic codes so CervoProxy-specific metadata does not
shape the core API.

The agent usability hardening line adds a machine-facing agnosticism contract:
`AGENTS.md` names core boundaries, `.cervorules/agent-manifest.json` lists
machine entrypoints and package ownership, and `docs/agent-quickstart.md`
routes agents toward neutral examples and conformance fixtures before adding
domain-specific vocabulary.

## Dependencies

Runtime dependency posture remains intentionally small. Release dependency
review should keep using:

- `go mod verify`;
- `go list -m all`;
- dependency audit in CI;
- package manifest verification;
- `scripts/ci/quality-gates.sh extended` before release.

Generator and schema dependencies remain acceptable only in generator/check
workflows unless a future ADR approves broader runtime use.

## Smells

Current accepted smell pressure:

| Smell | Status | Guardrail |
| --- | --- | --- |
| Performance regressions from routing changes | Guarded | Benchmarks, stress fixture, extended quality gate |
| Observability leaking sensitive fields | Guarded | Metric label tests and redaction policy |
| Report/wiki drift | Guarded | This tested repository report plus wiki health dashboard |
| Large generated or generator files | Guarded | File size tests and generator contract tests |
| Release artifact tests sharing temp dirs | Closed | #195 uses unique per-test directories |
| Facts/policygen integration drift | Guarded | #197, #198, and #203 provide DSL, optional generated helpers, and AST contracts |
| Roadmap/report drift after facts PRs | Guarded | #196 plus #204 refresh repo reports with post-facts smoke evidence |
| Agent onboarding drift | Guarded | #231 through #238 add AGENTS.md, agent manifest, quickstart, and drift tests for current RC and agnosticism guidance |
| Runner hook freezing CI | Guarded | issue #248 bounds CI workspace permission repair and rejects global `chown -R` hooks |
| Performance guidance drifting after routing/facts changes | Guarded | PR #247 evidence is recorded in `docs/performance/baselines.md` and covered by docs drift tests |

## Limitations

- v2 remains Beta / Early Adoption until all GA hardening issues are closed.
- Facts runtime is mature enough for optional advanced use, including
  declarative DSL, optional policygen helpers, stratified negation, controlled
  aggregates, partial evaluation, and incremental snapshot evaluation.
- `PolicyEvaluationReport` exports bounded trace summaries; full trace sampling
  is consumer-owned unless a later issue proves the need for a core API.
- OTel and Prometheus examples remain adapter patterns, not runtime
  dependencies.
- Reports must cite implementation evidence; score-only changes are not enough
  to raise maturity.
- Published package verification was intentionally skipped for
  `v2.0.1-smoke.20260522` because no real tag/package was published. The local
  release artifact build produced tool archives, schemas, checksums, metadata,
  dependency manifest, provenance, and SBOM module manifest.

## Repository Report Inventory

| Repository report | Wiki page | Drift rule |
| --- | --- | --- |
| `docs/reports/v2-ga.md` | `Reports` | Refresh wiki checklist when repository report evidence changes. |
| `docs/reports/v2-ga.md` | `Reports-Maturity-2026-05` | Keep maturity claims tied to issues, PRs, tests, scripts, ADRs, or release commands. |
| `docs/reports/v2-ga.md` | `Reports-Agnosticism-2026-05` | Keep stack-neutral dependency and adapter claims aligned with repo docs. |
| `docs/reports/v2-ga.md` | `Reports-Dependencies-2026-05` | Keep dependency review commands and package manifest evidence current. |
| `docs/reports/v2-ga.md` | `Reports-Smells-2026-05` | Keep accepted smells paired with guardrails. |
| `docs/reports/v2-ga.md` | `Postmortems` | Record near misses with detection path, corrective test, linked issue, and PR. |
| `docs/v2-ga-roadmap.md` | `Roadmap-and-Issues` | Keep #214-#222 status and next action aligned with issue/PR state. |
| `docs/documentation-refresh-2026-05-23.md` | `Reports` | Keep issue #249 repo/wiki refresh evidence discoverable from the wiki dashboard. |

## Wiki Refresh Checklist

When this report changes, refresh these wiki pages:

- `Reports`;
- `Reports-Maturity-2026-05`;
- `Reports-Agnosticism-2026-05`;
- `Reports-Dependencies-2026-05`;
- `Reports-Smells-2026-05`;
- `Postmortems`;
- `Roadmap-and-Issues`.
