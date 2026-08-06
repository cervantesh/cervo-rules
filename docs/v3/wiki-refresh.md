# v3 Wiki And Report Refresh Plan

Issues: #398, #402.

The wiki should mirror the repo docs after each v3 RC PR batch. This document is
the repo-side source of truth for the next wiki update.

## Pages To Refresh

| Wiki page | Repo source |
| --- | --- |
| Reports v3.0.0-rc.2 | `docs/reports/v3-rc2.md` |
| Architecture | `docs/v3/api-reference.md`, `docs/v3/modular-boundaries.md`, `docs/v3/module-layout.md` |
| Migration v2 to v3 | `docs/v3/migration-v2-to-v3.md`, `docs/v3/migration-tooling.md` |
| Packages | `docs/packages.md`, `docs/release.md` |
| Reports Maturity | `docs/change-management/v3-checkpoint-3.md`, `docs/v3/consumer-conformance.md` |
| Reports Agnosticism | `docs/v3/primitives.md`, `docs/v3/consumer-conformance.md`, `docs/agnosticism.md` |
| Reports Dependencies | `docs/dependencies.md`, `docs/packages.md` |
| Reports Smells | `docs/change-management/v3-breaking-api.md`, `docs/v3/public-api-inventory.json` |

## Required Wiki Evidence

The wiki refresh should mention:

- `v3.0.0-rc.2` as the clean post-fix v3 release candidate;
- `scripts/release/verify-github-release.sh v3.0.0-rc.2` and `scripts/release/verify-oci-tools.sh v3.0.0-rc.2` as release evidence;
- `scripts/performance/report.sh` as performance evidence;
- `github.com/cervantesh/cervo-rules/v3` as the v3 module path;
- `Operation`, `Target`, and `Executor` as v3 public primitive names;
- `runtime.PolicyFactory` as the only generated runtime entrypoint;
- manual v2-to-v3 migration guidance, with native migration reporting deferred;
- `v3/testkit.ConsumerConformanceContract` as the repo-local consumer evidence;
- `release_module=github.com/cervantesh/cervo-rules/v3` as the package verification marker;
- `schemas/v3/*.schema.json` as required release artifacts.

## Post-Merge Wiki Step

After this PR lands on `main`, update the project wiki from these repo docs and
record the wiki commit or page update in issue #402. If the wiki cannot be
updated in the same session, leave #402 open with the blocker and do not claim
wiki parity.
