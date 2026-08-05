# Script Layout

Scripts are grouped by operational domain. Keep new automation in the narrowest
matching directory and update repoquality tests when a script becomes part of a
release or CI contract.

| Directory | Purpose |
| --- | --- |
| `scripts/ci/` | Local and workflow quality gates. |
| `scripts/release/` | Release artifact build and package verification. |
| `scripts/performance/` | Benchmark reports, profiling, and benchmark history checks. |
| `scripts/sonar/` | SonarQube scan and local reset helpers. |

Root-level scripts are intentionally avoided so operational intent stays clear.
