# CervoRules v3 Final Reevaluation

Issues: #416, #417, #418, #419

Scope: final checkpoint after workstreams #392, #398, #404 and #410.

Status: checkpoint complete; not ready for GA.

## Release And Package Verification

The final release block proved that the v3 release path is close, but still
needs one clean post-fix RC before GA:

- #392 established package and supply-chain metadata for v3 artifacts.
- #398 refreshed migration, API, wiki, release and agent-facing docs.
- #404 created `v3.0.0-rc.1` and verified generic packages, OCI image, clean
  consumer import and release evidence.
- #410 reviewed post-RC feedback and recorded GA blockers.

The important release finding is the Manual OCI recovery. The original
`v3.0.0-rc.1` tag workflow failed while building the tools image because
`build/tools/Dockerfile` did not copy root facade files required by `policygen`.
PR #439 fixed that. The generic package was already published and verified; the
OCI image was rebuilt from fixed `main`, pushed and verified manually.

This is acceptable for an RC record, but it does not satisfy GA evidence because
GA should be reproducible from a clean tag workflow.

## GA Recommendation

Recommendation: do not tag `v3.0.0` GA yet.

Recommended next release: `v3.0.0-rc.2`.

Reasoning:

- v3 API shape is coherent and documented.
- Migration tooling and conformance tests exist.
- Package verification works after the Dockerfile fix.
- The first RC required manual recovery, so the release automation itself still
  needs a clean post-fix proof.

GA should wait until `v3.0.0-rc.2` publishes generic and OCI artifacts from CI
without manual repair, and the same verification commands pass against those
artifacts.

## Keep

Keep these v3 decisions:

- module path `github.com/cervantesh/cervo-rules/v3`;
- `Operation`, `Target`, `Executor` as public primitives;
- generated `PolicyFactory` as the canonical generated runtime entrypoint;
- modular packages and a small root facade;
- trace and observation opt-in for production defaults;
- optional bounded `facts` engine;
- machine-readable API inventory, policy inspection and compatibility reports;
- Generic packages and OCI as release artifacts.

## Drop

Do not bring these back into v3:

- v2 primitive aliases in public API;
- generated `BuildPolicy` as a canonical entrypoint;
- implicit `RoutingPhase(rules...)` style APIs that hide linear cost;
- consumer-specific vocabulary in core;
- global runtime caches in core.

## Defer

Defer until after GA unless real consumer migration proves otherwise:

- automated source rewrites for migration;
- stronger facts planner heuristics;
- hard performance thresholds in CI;
- signed checksums;
- project-specific adapters beyond neutral examples and documented patterns.

## v3.1 Follow-Up Candidates

Candidates for v3.1:

- `cervorules-policygen migrate-v3 --rewrite` if consumers need automated
  migration patches;
- richer facts explain visualizations and cost reports;
- benchmark-history baselines once enough runner data exists;
- optional checksum signing with documented key ownership;
- consumer adapter packages that remain outside core, such as queue or job
  adapters, if multiple projects need them.

## Final Checkpoint Decision

Proceed with another RC, not GA:

1. Tag `v3.0.0-rc.2` from fixed `main`.
2. Require `Publish Packages` success without manual intervention.
3. Verify generic package and OCI image.
4. Refresh release evidence and wiki reports.
5. Re-run public API inventory review.
6. Decide GA only after the clean post-fix release proof exists.
