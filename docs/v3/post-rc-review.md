# CervoRules v3 Post-RC Review

Issues: #410, #411, #412, #413, #414, #415

Release candidate reviewed: `v3.0.0-rc.1`

Status: no-go for immediate GA.

This review closes the post-RC workstream by recording what the first v3
release candidate proved, what remains risky, what should be deferred, and what
must be true before `v3.0.0` GA.

## RC Feedback Review

The first RC validated the repo-only v3 design:

- v3 can coexist with v2 through `github.com/cervantesh/cervo-rules/v3`.
- Generated and handwritten v3 examples compile in isolation.
- Consumer-shaped conformance exists in `v3/testkit`.
- Migration replacement guidance exists; native v3 migration reporting remains
  deferred after the physical split.
- Package artifacts are machine-verifiable.
- OCI tooling is available for consumers that prefer containerized CLI usage.

The most important operational feedback was from release packaging, not runtime
behavior. The original tag publish workflow failed in the OCI build because
`build/tools/Dockerfile` omitted the root facade files needed by `policygen`. PR #439
fixed the Dockerfile, and the `v3.0.0-rc.1` OCI image was rebuilt, pushed, and
verified manually after the generic package had already been verified.

That recovery is acceptable for an RC, but it is not acceptable evidence for GA.
GA needs one clean tag publish from the fixed workflow without manual package
repair.

## Public API Audit

The v3 public API is frozen enough for another RC, but not for GA final.

Kept for v3:

- `Operation`, `Target`, and `Executor` replace the v2
  `Capability`, `Service`, and `Provider` names.
- Modular imports remain the recommended path.
- The root facade stays small and compatibility-free.
- Generated policies use `PolicyFactory` as the canonical entrypoint.
- Trace and observation are opt-in for low-allocation defaults.
- `facts` remains optional and operationally bounded.

Still requiring pre-GA audit:

- confirm no generated public wrappers reintroduced v2 naming;
- confirm policy inspection and compatibility JSON are versioned in every CLI
  output path;
- confirm docs, schemas, and migration examples use v3 names consistently;
- confirm every intentionally deferred compatibility helper is outside the v3
  runtime surface.

## GA Blockers

The current blockers are operational and adoption-oriented:

| Blocker | Why It Blocks GA | Required Evidence |
| --- | --- | --- |
| Clean tag publish | `v3.0.0-rc.1` needed manual OCI recovery after PR #439. | A new RC tag publishes generic and OCI packages from CI without manual repair. |
| Release workflow confidence | The fixed Dockerfile has CI coverage, but the full tag workflow has not rerun successfully after the fix. | Successful `Publish Packages` run on a post-fix tag. |
| Migration confidence | Tooling exists, but no real external consumer has migrated. | At least one clean consumer fixture or real consumer migration report reviewed. |
| Final API audit | v3 API changed broadly across primitives, generated policy, facts, and observability. | Public API inventory reviewed after final RC changes. |
| Wiki/report drift | Docs were refreshed before the tag; release evidence changed afterward. | Final checkpoint and wiki/report refresh after the clean post-fix RC. |

## Deferred Work

Deferred to v3.1 unless a real migration proves otherwise:

- automated source rewriting for v2-to-v3 imports and symbol names;
- additional facts planner heuristics beyond the current bounded planner;
- hard benchmark thresholds in CI;
- checksum signing if the signing key/process is not ready;
- consumer-specific adapters beyond neutral examples and CervoProxy-shaped
  fixtures.

Deferred work must not block GA unless it is required to keep v3 safe,
understandable, or package-verifiable.

## v3.0.0 GA Checklist

Before tagging `v3.0.0`:

- cut one more RC after PR #439, preferably `v3.0.0-rc.2`;
- confirm `Publish Packages` succeeds from the tag without manual repair;
- run `scripts/release/verify-generic-package.sh <tag> <workdir>`;
- run `scripts/release/verify-oci-tools.sh <tag>`;
- run a clean temp consumer import of `github.com/cervantesh/cervo-rules/v3`;
- run v2-to-v3 manual migration review against representative fixtures;
- run generated policy conformance and package smoke fixtures;
- refresh wiki reports with package and migration evidence;
- update `CHANGELOG.md`, release notes, `AGENTS.md`, and the agent manifest;
- perform one final public API inventory review;
- close or explicitly defer all v3 RC issues.

## Decision

Proceed to the final reevaluation checkpoint (#416) with a no-go for immediate
GA. The next release action should be a post-fix RC tag, not GA final.
