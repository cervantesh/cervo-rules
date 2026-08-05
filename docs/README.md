# Documentation Map

CervoRules documentation is grouped by operational purpose so v3 work does not
turn the top-level `docs/` directory into a flat archive.

| Area | Path | Purpose |
| --- | --- | --- |
| v3 contracts | [docs/v3/README.md](v3/README.md) | v3 API, migration, generated policy, facts, observability, and release notes. |
| Change management | [docs/change-management/README.md](change-management/README.md) | Version plans, checkpoints, reevaluations, and breaking-change governance. |
| Performance | [docs/performance/README.md](performance/README.md) | Baselines, benchmark history, gates, hot-path guidance, and profiling decisions. |
| Operations | [docs/operations/README.md](operations/README.md) | Runner and local service operation notes. |
| Reports | [docs/reports/README.md](reports/README.md) | Version-level maturity, risk, and release evidence reports. |
| ADRs | `docs/adr/` | Architecture decisions that constrain public API and release direction. |

Keep short, stable entrypoints in the repository root (`README.md`,
`AGENTS.md`, `CHANGELOG.md`) and move longer topic documents into one of these
directories.

Operational scripts are indexed separately in
[`scripts/README.md`](../scripts/README.md).
