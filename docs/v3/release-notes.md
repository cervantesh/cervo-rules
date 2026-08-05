# v3.0.0-rc.3 Release Notes

Issues: #398, #403, #472, #473, #474, #478, #480, #481, #487, #489.

`v3.0.0-rc.3` is a CervoRules v3 release candidate with repository
organization, package-script layout, and v3 root-boundary hardening. It remains
a release candidate, not GA final.

## RC3 Changes

- v3 docs now live behind topic indexes in `docs/v3`, `docs/performance`,
  `docs/change-management`, `docs/operations`, and `docs/reports`.
- Operational scripts are grouped by domain under `scripts/ci`,
  `scripts/release`, `scripts/performance`, and `scripts/sonar`.
- v3 now lives at the repository root after the physical v2/v3 split.
- v2 maintenance moved to `CervoSoft/cervo-rules-v2` with final cut `v2.1.0`.
- Future breaking work will not carry v2 compatibility shims forward.

## Breaking Changes

- Public primitive names are `Operation`, `Target`, and `Executor`.
- v3 removes public `Capability`, `Service`, and `Provider` aliases.
- Generated policies use `runtime.PolicyFactory`; `BuildPolicy` is not part of
  the v3 generated runtime API.
- v3 DSL field names are `operation`, `target`, and `executor`.
- `RoutingPhase` is removed from v3; indexed and linear routing are explicit.
- Facts are optional and budgeted; trace is opt-in.
- Observability reports are versioned and low-cardinality by default.

## Machine-Readable Contracts

- v3 schema bundle under `schemas/v3`.
- Public API inventory in `docs/v3/public-api-inventory.json`.
- Legacy migration report command remains available from `CervoSoft/cervo-rules-v2`.
- Consumer conformance contract in `testkit`.
- Package release marker: `release_module=github.com/cervantesh/cervo-rules/v3`.

## Verification Before Tag

```bash
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
scripts/release/check.sh v3.0.0-rc.3 dist-release-check-v3.0.0-rc.3
```

Windows PowerShell operators can run the same check through:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/release/check.ps1 v3.0.0-rc.3 dist-release-check-v3.0.0-rc.3
```

## Verification After Package Publish

```bash
scripts/release/verify-generic-package.sh v3.0.0-rc.3
scripts/release/verify-oci-tools.sh v3.0.0-rc.3
```
