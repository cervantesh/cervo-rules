# CervoRules v2.1.0-rc.1 Change Management

`v2.1.0-rc.1` is an RC aggressive / strong beta checkpoint. It is a serious
release candidate for repo-local maturity, but it is not GA final because
external consumer validation is intentionally deferred.

Parent epic: #214.

## Intent

This program pushes CervoRules as far as it can go as an individual project:
conformance, compatibility, facts stress, policygen contracts, observability,
docs drift, release verification, and supply-chain evidence. It does not modify
CervoProxy or any other consumer.

## Repo-Only Readiness Scorecard

| Area | Issue | RC evidence required |
| --- | --- | --- |
| Change management | #215 | This document, explicit RC criteria, and roadmap tracking. |
| Conformance | #216 | Repo-local consumer certification suite covers runtime config and facts checks for six synthetic consumers. |
| Compatibility | #217 | Policy and generated API drift checks produce stable reports. |
| Facts stress | #218 | Advanced facts scenarios pass for full, partial, and incremental evaluation. |
| Policygen contracts | #219 | Generator contracts cover metadata, facts helpers, imports, and freshness. |
| Observability schema | #220 | `PolicyEvaluationReportContract()` exposes version `cervorules.policy_evaluation_report.v1`; allowed fields, privacy exclusions, low-cardinality metric labels, and neutral examples are tested. |
| Docs drift | #221 | Automated checks detect stale roadmap/report/wiki references. |
| Release and supply chain | #222 | RC packages, checksums, SBOM, provenance, and clean-consumer checks verify. |

### #216 Conformance Evidence

Wave 1 adds `testkit.ConsumerConformanceContract` with
`CheckConsumerConformance` and `MustAssertConsumerConformance` so repo-local
consumer fixtures can certify policy artifacts, runtime override behavior, and
facts derivation without importing generator or runtime internals.

Synthetic fixtures under `examples/conformance/` cover billing routing,
document processing, device routing, queue event routing, scheduled job
routing, and CLI command routing. Each contract checks policy/vocabulary files,
primary and backup runtime decisions, invalid runtime config rejection, and a
derived `certified_consumer` fact.

### #218 Facts Stress Evidence

Wave 1 adds `facts/advanced_stress_test.go` as a cross-feature fixture for the
optional facts layer. The fixture combines stratified negation, count
aggregates, prepared partial evaluation, incremental add/remove evaluation,
golden explanations, redaction, and aggregate cardinality bounds.

Required invariants:

- full evaluation and prepared partial evaluation return identical fact sets for
  mixed rules and aggregates;
- full evaluation and incremental evaluation return identical fact sets after
  add/remove changes;
- unsafe negation still fails validation with `unsafe_negation_variable`;
- aggregate input limits fail closed without deriving downstream eligibility;
- golden explanation JSON remains deterministic and redacts request email terms.

### #217 Compatibility Evidence

Wave 2 adds semantic policy drift checks to `cervorules-policygen diff` and
`cervorules-policygen compatibility`, plus the requested
`cervorules-policygen compat --old old.yaml --new new.yaml --format json`
alias. Breaking changes return a non-zero exit code.

Covered breaking surfaces:

- route removal and route disabling;
- newly added denies;
- limit tightening for streaming, tools, images, token ceilings, and body byte
  ceilings;
- derived fact removals by declared predicate, rule head, or aggregate output.

Wave 2 also adds `cervorules-policygen inspect-api --policy policy-rules.yaml
--format json` for stable policy-declared API inspection. The JSON includes
policy identity, routes, denies, limits, derived fact predicates/rules/
aggregates, and warnings.

Known coverage limits are explicit in the JSON warnings: vocabulary drift and
generated source/API signature drift are not compared by `compat` because the
command accepts policy files only. Derived facts are compared by declared
surface, not by full semantic equivalence proof.

### #219 Policygen Contract Evidence

Wave 2 hardens generator contracts around the RC public surface:

- AST contracts cover generated factory types/functions, metadata helpers,
  runtime config helpers, optional derived facts helpers, exact import sets, and
  generated test-source helpers.
- Structured `inspect-api` tests unmarshal CLI JSON into `APIInspection`
  instead of relying on string fragments only.
- Freshness checks compare generated policy source and generated test source
  when `check -generated-tests` is requested.
- The `facts` import is asserted absent unless `derived_facts` is declared and
  present when derived facts helpers are generated.

### #221 Docs Drift Evidence

Wave 3 adds automated repository checks for the RC documentation surfaces.
Roadmap status rows are intentional per issue instead of a stale aggregate open/pending sentence.
Repo report and wiki checklist references are synchronized by tests.

Docs drift rules:

- Critical Go snippets must either live in compilable tests/examples or be explicitly labeled pseudocode.
- Shell snippets must be copy-paste runnable for the documented platform or marked illustrative.
- Release notes and changelog entries must cite the issue and PR that changed RC behavior or release evidence.
- Maturity and readiness claims must cite concrete evidence such as issues,
  PRs, tests, scripts, ADRs, or release commands.

### #222 Release And Supply Chain Evidence

## v2.1.0-rc.1 Release/Package Verification Matrix

| Check | Required evidence for #222 |
| --- | --- |
| Local release artifacts | `scripts/release/check.sh v2.1.0-rc.1` runs before any tag or publish action. |
| Generic package | `scripts/release/verify-generic-package.sh v2.1.0-rc.1` downloads packages from a clean consumer path and verifies checksums, metadata, schemas, and CLI `-version` output. |
| SBOM/provenance validation | The generic package verifier checks `sbom-modules.json`, `sbom-spdx.json`, `provenance.json`, and `artifact-manifest.json` for version, module, commit, Go version, SPDX marker, SLSA predicate type, source material, and `scripts/release/build-artifacts.sh` builder evidence. |
| Signed checksums RC decision | `minisign` is selected for generic package checksum signatures. Release builds publish `checksums.txt.minisig` when `CERVORULES_MINISIGN_SECRET_KEY_FILE` is configured, and consumers verify with `CERVORULES_VERIFY_SIGNATURES=1` plus `CERVORULES_MINISIGN_PUBLIC_KEY`. |
| Clean consumer package verification | The clean path is `scripts/release/verify-generic-package.sh v2.1.0-rc.1` from a clean checkout or release operator shell after package publication. |
| OCI pull/run | `scripts/release/verify-oci-tools.sh v2.1.0-rc.1` pulls the image, checks schemas, and runs `cervorules-policygen -version` and `cervorules-vocabgen -version` when OCI publishing is enabled. |

## RC Criteria

- Every child issue records estimate, actual time, deviation, linked PR, tests,
  and maturity impact.
- Each PR starts with failing tests when it changes behavior.
- Required Build and Dependency Audit pass before merge.
- `go test -count=1 ./...`, `go test -cover ./...`, `go vet ./...`, and
  `go mod verify` pass for every PR.
- Release candidates additionally run race, vulnerability, extended quality,
  release, package, and OCI checks where applicable.

## Not GA Final

This RC can be consumed for serious validation, but GA final still requires a
separate decision after package verification and at least one real downstream
consumer validation cycle. Issue #123 remains infrastructure debt for moving the
runner off the laptop; it does not block this repo-only RC.

## Merge Strategy

1. Merge #215 first because it defines the shared scorecard.
2. Merge #216, #218, and #220 in parallel-safe branches.
3. Merge #217 before #219 so generator contracts can consume compatibility
   report shape.
4. Merge #221 before #222 so release evidence is checked against current docs.
5. Tag `v2.1.0-rc.1` only after all child issues close.

## Issue Checklist

Each RC issue must contain:

- objective;
- Definition of Done;
- estimate;
- start time;
- actual time;
- deviation;
- linked PR;
- tests run;
- release or maturity impact.
