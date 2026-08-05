# v3 Native Generators

Date: 2026-05-24.

Issues: #492, #493, #494, #495.

## Objective

Restore CervoRules generator tooling after the physical v3 split without
carrying v2 compatibility shims forward.

## Decisions

- `cervorules-vocabgen` is rebuilt against v3 vocabulary primitives:
  `Operation`, `Target`, and `Executor`.
- `cervorules-policygen` is rebuilt as a v3-native routing generator.
- Generated policies expose `PolicyFactory`; no generated `BuildPolicy`
  compatibility wrapper is emitted.
- The active v3 generator subset is intentionally smaller than the historical
  v2 generator: operation routes, target/executor selection, executor
  fallbacks, disabled-by-default routes, explicit denies, default config,
  config validation, metadata, and generated tests.
- Historical `compat`, `inspect-api`, `diff`, `migrate-v3`, and policy
  `explain` workflows remain future v3 work unless redesigned.

## Contract

The v3-root repository may contain `cmd/` and `internal/` only for v3-native
tooling. Hygiene checks reject v2 imports and v2 primitive names in those
directories.

Release packages should include:

- `cervorules-policygen`;
- `cervorules-vocabgen`;
- root and v3 schemas;
- checksums;
- build metadata;
- dependency/module manifests;
- SBOM/provenance artifacts when generated.

## Verification

Required checks for this change:

- `go test -count=1 ./...`;
- `go test -cover ./...`;
- `go vet ./...`;
- `go mod verify`;
- `scripts/ci/repo-hygiene.sh`;
- `scripts/release/build-artifacts.sh <version> <dist-dir>`.
