# CervoRules v2 GA Roadmap

`v2.0.0` is the first CervoRules v2 module release and is marked in the release tracker as
Beta / Early Adoption. The tag remains valid and machine-consumable, but GA
confidence is tracked through the `CervoRules v2 GA Hardening` milestone.

Parent epic: #83

## Why v2.0.0 Is Beta / Early Adoption

The v2 line has the core breaking API shape in place:

- Go module path `github.com/cervantesh/cervo-rules/v2`;
- structured validation errors and multi-violation limit checks;
- `DecisionResult`;
- context/error-aware predicates and actions;
- generated `PolicyFactory`;
- error-first testkit helpers;
- `Vocabulary` interface and `StaticVocabulary`;
- runtime config options;
- verified CervoProxy migration;
- verified generic package and OCI release artifacts.

The remaining work is not about proving the core can function. It is about
making the v2 line easier to operate, audit, evolve, and adopt across more
projects without drift.

The facts workstream completed after the original v2.0.0 beta checkpoint. PRs
#184 through #192 added the optional facts module foundation, operational
hardening, semantic ADRs, explainability contracts, and evaluator optimization.
The follow-up maturity closure work is tracked by epic #193.

The next checkpoint is `v2.1.0-rc.1`, an RC aggressive / strong beta line for
repo-only maturity hardening. It is tracked by epic #214 and remains separate
from GA final because external consumer validation is deferred.

## GA Criteria

The v2 line reaches GA confidence when these criteria are met or explicitly
deferred by ADR:

1. Generated policy metadata includes real policy and vocabulary hashes.
2. Release packages include machine-auditable manifests.
3. Policy diff and compatibility checks protect consumers from accidental
   breaking policy changes.
4. DSL/schema authoring has lint, warnings, versioned schemas, and clearer
   diagnostics.
5. Generator tests rely more on structured AST/contract checks and less on
   string fragments.
6. Benchmarks, stress fixtures, race/fuzz/static analysis gates cover the main
   runtime and generator risks.
7. Observability, audit, privacy, and reporting contracts are documented and
   tested.
8. Non-HTTP adapters and fixtures demonstrate cross-domain use beyond gateway
   traffic.
9. Runtime config, limits, errors, and CLI UX remain cross-domain and
   versioned.
10. Testkit and conformance suites support downstream consumers.
11. Supply-chain and package verification cover all release artifacts and
    supported platforms.
12. Maturity, agnosticism, dependencies, smells, and v2 postmortem reports are
    refreshed with evidence.

## Workstreams

| Issue | Workstream | Purpose |
|---|---|---|
| #84 | GA status and reports | Define beta/GA status, roadmap, and report workflow. |
| #85 | Metadata and manifests | Add policy manifests, hashes, generated metadata, and artifact manifests. |
| #86 | Compatibility tooling | Add policy diff, compatibility checker, registry, and validation release flow. |
| #87 | Governance | Add templates, CODEOWNERS, API audit, deprecation policy, and roadmap governance. |
| #88 | DSL authoring | Add schema versioning, lint, warnings, and authoring UX improvements. |
| #89 | Generator quality | Add AST contracts, structured snapshots, freshness checks, and formatting guards. |
| #90 | Performance and CI gates | Add benchmarks, stress fixtures, race/fuzz/static analysis gates. |
| #91 | Observability and audit | Improve trace, reports, privacy, audit schema, and diagnostics usage. |
| #92 | Reports | Produce postmortem and refresh maturity/agnosticism/dependency/smell reports. |
| #93 | Adapters | Add non-HTTP adapters and agnostic consumer fixtures. |
| #94 | Agnosticism | Improve vocabulary docs, naming review, and primitive/action metadata ADRs. |
| #95 | Runtime API | Improve config builder, options, errors, custom limits, explain/inspect/graph. |
| #96 | Testkit and conformance | Add conformance suite, consumer contracts, and readiness scorecards. |
| #97 | Release automation | Add executable release checks and package verification matrix. |
| #98 | Supply chain | Add dependency review, SBOM, provenance, signed checksums, and runner docs. |
| #193 | v2 GA maturity closure | Close post-facts gaps before a stronger beta/RC checkpoint. |
| #214 | v2.1 RC hardening | Coordinate repo-only RC maturity work. |
| #215 | RC readiness | Change-management plan and readiness scorecard. |
| #216 | Conformance | Repo-local consumer certification suite. |
| #217 | Compatibility drift | Semantic policy and generated API drift checks. |
| #218 | Facts stress | Advanced cross-feature facts fixtures. |
| #219 | Policygen contracts | Generator contract hardening for RC. |
| #220 | Observability schema | Version operational report contracts. |
| #221 | Docs drift | Automated roadmap, wiki, and report drift detection. |
| #222 | Release and supply chain | Publish and verify v2.1.0-rc.1 packages and supply chain. |

## Recommended Sequence

1. Finish #193 foundation work (#194, #195, #196) so roadmap, reports, and
   release-test guardrails match the current facts evidence.
2. Finish #84 so every contributor understands that `v2.0.0` is beta/early
   adoption and knows what GA means.
3. #197 is complete: declarative facts DSL is stable enough for generated
   integration.
4. Do #198 before #203: policygen complexity hardening should respond to the
   real integration shape.
5. #199 is complete: stratified negation is implemented in the optional facts
   runtime and declarative facts DSL.
6. #200 is complete: controlled aggregates are implemented in the optional
   facts runtime and declarative facts DSL.
7. #201 is complete: partial evaluation preparation is implemented with
   equivalence tests.
8. #202 is complete: incremental snapshot/change-set evaluation is implemented
   with equivalence tests.
9. Do #85 and #89 together where possible: generated metadata and generator
   contract tests are tightly related.
10. Do #86 after metadata hashes exist, because compatibility reports need
   stable policy identity.
11. Do #88 after the compatibility shape is clear, so lint warnings and schema
   versions align with compatibility semantics.
12. Do #96 before asking more consumers to adopt v2.
13. Do #91 and #92 after the core reports/observability contracts stabilize.
14. Do #93 and #94 to prove agnosticism beyond HTTP/gateway use cases.
15. #204 is complete: post-facts local release smoke passed for
    `v2.0.1-smoke.20260522`.
16. Do #90, #97, and #98 as remaining release hardening before declaring GA
    confidence.
17. Do #214 through #222 as the `v2.1.0-rc.1` RC aggressive / strong beta
    hardening line before any GA final decision.

## Tracking Rules

- Each issue records estimate, actual, and deviation.
- Each PR references the issue it advances and the parent epic #83.
- Each issue updates this roadmap if the sequence or GA criteria changes.
- Any v3-only candidate must be deferred with rationale instead of forced into
  v2.
- Reports must cite implementation evidence, not only desired state.

## v2.1.0-rc.1 Issue Status

These rows are intentionally per issue so closed, active, or pending work does
not drift behind a stale aggregate status sentence.

| Issue | Status | Current evidence | Next action |
| --- | --- | --- | --- |
| #214 | Active epic | Child issues #215 through #222 define the repo-only RC hardening line. | Keep child issue and PR links current until the RC tag decision. |
| #215 | Repo evidence complete | `docs/change-management/v2-1-rc.md` defines the RC scorecard and merge strategy. | Keep scorecard evidence aligned as child issues land. |
| #216 | Repo evidence complete | `testkit.ConsumerConformanceContract` and `examples/conformance/` cover six synthetic consumers. | Keep consumer certification evidence cited in the RC change-management doc. |
| #217 | Repo evidence complete | `cervorules-policygen compat`, `diff`, and `inspect-api` cover semantic policy drift and stable API reports. | Keep known limits explicit until broader vocabulary/source drift comparison exists. |
| #218 | Repo evidence complete | `facts/advanced_stress_test.go` covers full, partial, incremental, aggregate, negation, explanation, and redaction invariants. | Keep facts stress coverage tied to RC readiness evidence. |
| #219 | Repo evidence complete | Generator contracts cover AST shape, inspect-api JSON, freshness, imports, metadata, and optional facts helpers. | Keep generated public surface changes paired with contract tests. |
| #220 | Repo evidence complete | `PolicyEvaluationReportContract()` exposes versioned allowed fields, privacy exclusions, metric labels, and neutral examples. | Keep report-contract changes reflected in maturity evidence. |
| #221 | In progress | Docs drift tests cover roadmap status, repo/wiki report sync, snippet policy, changelog references, and maturity evidence. | Commit docs drift gate and keep #222 release notes linked to issues/PRs. |
| #222 | Pending release work | Release and supply-chain evidence remains the final RC child after docs drift gates. | Run RC package, checksum, SBOM, provenance, and clean-consumer checks. |

## Current Status

- `v2.0.0`: Beta / Early Adoption.
- CervoProxy migration: complete.
- Generic package verification: complete for `v2.0.0`.
- OCI verification: complete for `v2.0.0`.
- Facts workstream: complete through PR #192.
- Declarative facts DSL: complete through #197.
- Optional policygen derived facts integration: complete through #198.
- Stratified negation runtime: complete through #199.
- Controlled aggregates runtime: complete through #200.
- Partial evaluation API: complete through #201.
- Incremental evaluation API: complete through #202.
- Policygen complexity hardening after facts integration: complete through #203.
- Post-facts release smoke: complete through #204 for
  `v2.0.1-smoke.20260522`.
- Post-facts maturity closure: #194 through #204 complete.
- v2.1.0-rc.1 repo-only RC hardening: tracked per issue in the status table
  above.
- Post-PR #247 performance/routing/facts docs and issue #248 runner operations
  docs are being refreshed through #249 before the next RC/release confidence
  decision.
- GA hardening milestone: open.
