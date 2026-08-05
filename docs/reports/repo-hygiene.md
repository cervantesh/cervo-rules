# Repository Hygiene

Updated: 2026-05-24.

This dashboard tracks repository structure, not runtime maturity. Use it before
large refactors, release candidates, and vNext work.

## Current Status

| Area | Status | Evidence |
| --- | --- | --- |
| Root budget | Healthy | Active root has one Go marker file and no nested `v3/` module after the physical split. |
| v2 legacy | Split | v2 maintenance now lives in `CervoSoft/cervo-rules-v2` with final cut `v2.1.0`. |
| v3 root | Enforced | `scripts/ci/repo-hygiene.sh` verifies root module `github.com/cervantesh/cervo-rules/v3` and rejects v2 Go imports. |
| Docs layout | Healthy | Topic indexes exist under `docs/v3`, `docs/change-management`, `docs/performance`, `docs/operations`, and `docs/reports`. |
| Native tooling layout | Healthy | `cmd/` and `internal/` are allowed only for v3-native tooling and must not import v2 or use v2 primitive names. |
| Scripts layout | Healthy | Script domains are `scripts/ci`, `scripts/release`, `scripts/performance`, and `scripts/sonar`. |
| Release wrapper | Healthy | `scripts/release/check.ps1` avoids the Windows WSL launcher and prefers Git Bash. |

## Focused Check

Run the focused hygiene gate when changing repository structure:

```bash
scripts/ci/repo-hygiene.sh
```

The script reports the root budget and checks the active v3-root contract:

- root module must be `github.com/cervantesh/cervo-rules/v3`;
- nested `v3/` must not exist;
- `cmd/` and `internal/` may exist only as v3-native tooling;
- v3-native tooling must use `Operation`, `Target`, and `Executor`;
- Go code must not import `github.com/cervantesh/cervo-rules/v2`.

## Boundaries

The active repository is v3-root. Future v3/vNext work must not add v2
compatibility shims. Use `CervoSoft/cervo-rules-v2` for v2 maintenance. See
`docs/change-management/physical-repo-split.md` and
`docs/change-management/vnext-no-v2-compatibility.md`.
