# Release

Machine-consumable package consumption is documented in
[docs/packages.md](packages.md).
Dependency acceptance and release review rules are documented in
[docs/dependencies.md](dependencies.md).
Use this document for release execution.

## Release Checklist

Complete this checklist before publishing a release tag. v2 maintenance now
lives in `CervoSoft/cervo-rules-v2`; the active repository is v3-root.

### Before Tagging

- Confirm the public Go API intended for the target release is documented in `README.md`.
- For v2 maintenance releases, use the `CervoSoft/cervo-rules-v2` repository.
- Confirm `schemas/policy-vocabulary.schema.json` and `schemas/policy-rules.schema.json` describe the supported v1 DSL fields.
- Confirm `examples/README.md` identifies the default copy example and current command workflow.
- Confirm `docs/change-management/physical-repo-split.md` still matches the repository layout.
- Confirm public API or DSL changes were reviewed against `docs/adr/0001-cross-domain-public-api.md`.
- Confirm primitive naming and action metadata changes were reviewed against `docs/adr/0003-v2-naming-and-action-metadata.md`.
- For breaking public API or generated policy changes, confirm the v2 change-management gate in `docs/change-management/v2-public-api.md` has maintainer approval and linked implementation issues.
- Complete `docs/change-request-template.md` for runtime, generator, schema, CLI, or release-artifact behavior changes.
- Run `docs/api-audit.md` when public API, generated API, schemas, or CLI contracts change.
- Apply `docs/deprecation-policy.md` before removing public APIs, generated wrappers, schema fields, or CLI flags.
- Confirm `docs/dependencies.md` matches `go.mod`, `go.sum`, and the runtime/generator dependency split.
- Confirm every new or materially expanded dependency has recorded review fields: purpose, standard library alternative, runtime/generator scope, license, update policy, and consumer impact.
- Confirm generated policy packages do not import generator-only dependencies.
- Confirm runtime config merge semantics in `docs/runtime-env-mapping.md` match implementation and tests.
- Confirm production HTTP classifier docs prefer `NewHTTPClassifier` for regex validation.
- When the optional `facts` package changes, confirm the logic-inspired facts
  docs, ADRs, examples, and package coverage remain current.
- Confirm generated-policy consumers can use `testkit.AssertGeneratedRuntimePolicy` instead of local compatibility checks.
- Confirm `docs/observability.md` still matches `DecisionObservation`, `MetricLabels`, and `LogFields`.
- Review dependency policy in `docs/dependencies.md` for any new or updated modules.
- Confirm docs/wiki refresh is complete for the release scope. At minimum,
  review `README.md`, `AGENTS.md`, `CHANGELOG.md`, `.cervorules/agent-manifest.json`,
  `docs/v2-ga-roadmap.md`, `docs/reports/v2-ga.md`, `docs/packages.md`,
  `docs/performance/baselines.md`, and the relevant wiki reports.
  and do not rely on a global workspace `chown -R` repair.
- Review the latest wiki smell report and update the status when release work closes a smell.
- Run `go list -m all` and compare the module list against `docs/dependencies.md`.
- Run `go mod verify`.
- Run `go test ./...` from the repository root.
- Run `go vet ./...`.
- Run `scripts/ci/quality-gates.sh` for the standard verification gate.
- Run `govulncheck ./...`.
- Run `golangci-lint run ./...`.
- Run `scripts/ci/quality-gates.sh extended` for race, fuzz smoke, and benchmark smoke.
- Run `go test -bench=DecisionFlow -benchmem -run '^$' ./...` when a separate benchmark artifact is needed.
- Run `scripts/release/check.sh <version>` before tagging to execute standard
  gates, extended gates, dependency listing, vulnerability scan when available,
  and local release artifact generation. On Windows PowerShell, prefer
  `pwsh scripts/release/check.ps1 <version>` or
  `powershell -ExecutionPolicy Bypass -File scripts/release/check.ps1 <version>`;
  the wrapper prefers Git Bash and avoids the Windows WSL launcher when it is
  not backed by `/bin/bash`.
- For v3-root releases, confirm examples compile with `go test ./...`.
- For v3 generated-policy checks, use the native `cmd/cervorules-policygen`
  and `cmd/cervorules-vocabgen` commands in this repository.
- Review `CHANGELOG.md` and move completed items from `Unreleased` into the target release section.
- Merge the release branch to `main`.
- Pull the updated `main` branch locally and verify the working tree is clean.

### Tag And Actions

- Create the annotated release tag from the updated `main` branch.
- Push the tag to origin.
- Confirm the `Publish Packages` workflow starts for the pushed tag.
- Confirm the workflow `Test` step completed `go test -count=1 ./...`.
- If `CERVORULES_PUBLISH_OCI=true`, confirm the OCI preflight step can log in to the registry before generic package upload.
- Confirm the workflow uploaded all generic package files listed below.
- Confirm the `Dependency Audit` workflow is passing on the release commit.
- If `CERVORULES_PUBLISH_OCI=true`, confirm the OCI publish step pushed both `<version>` and `latest`.
- Create or update the release notes from `CHANGELOG.md` after package verification passes.

### Package Verification

- Download `checksums.txt` from the generic package version.
- Download `artifact-manifest.json` and confirm it records the release version,
  commit, module path, Go version, artifact names, SHA-256 digests, and byte
  sizes.
- Download `build-metadata.txt` and confirm the version and commit match the release.
- For v2 packages, confirm `build-metadata.txt` records module `github.com/cervantesh/cervo-rules/v2`.
- For v2 packages, confirm the release commit included the gateway-shaped adoption fixture and generated example verification.
- Download `dependencies.txt` and confirm the module list is expected.
- If the release contains platform tool archives, download at least one archive,
  verify its checksum, extract it, and run CLI `-version` checks.
- Download `cervorules-schemas-<version>.tar.gz` and confirm both schema files are present.
- Run `scripts/release/verify-generic-package.sh <version>` from a clean checkout or release operator shell.
- If the package is private, repeat one download using a consumer-style read token.

### OCI Verification

- Pull `$CERVORULES_OCI_REGISTRY/cervosoft/cervorules-tools:<version>`.
- Confirm `/opt/cervorules/schemas/policy-vocabulary.schema.json` exists.
- Pull `$CERVORULES_OCI_REGISTRY/cervosoft/cervorules-tools:latest` and confirm it resolves to the same real release after the final release is published.

### Rollback Notes

- If tests fail before upload, leave the tag in place only if no package artifacts were published; otherwise treat it as a partial release.
- If generic packages were published with bad artifacts, publish a corrected patch version instead of overwriting the release version.
- If the OCI `latest` tag points to a bad image, retag the latest known-good release image as `latest` and push it.
- If a tag was created from the wrong commit before packages published, delete the local and remote tag, recreate it from `main`, and push it again.
- Record any partial release or rollback action in the release notes and `CHANGELOG.md`.

## Tagging

Create the release tag only after the release changes are merged to `main`.

```powershell
git checkout main
git pull --ff-only
git status --short
git tag -a v2.0.0 -m "CervoRules v2.0.0"
git push origin v2.0.0
```

Do not tag a feature branch or unmerged release branch. If a tag is created before the main merge, delete it locally and remotely, merge first, and recreate the tag from `main`.

## Packages

Releases are for human release notes, tags, changelog, and source archives.
Packages are for artifacts consumed by automation and downstream projects.
Consumer download commands, OCI usage, smoke package policy, and cleanup notes
live in [docs/packages.md](packages.md).

## v3.0.0-rc.1 Release/Package Verification Matrix

| Check | Evidence command or artifact | RC decision |
| --- | --- | --- |
| Local release artifact build | `scripts/release/check.sh v3.0.0-rc.3 dist-release-check-v3.0.0-rc.3` | Required before tagging; proves v3 root plus v3 release metadata can be built from a clean dist directory. |
| Generic package download | `scripts/release/verify-generic-package.sh v3.0.0-rc.3` | Required after package publication; confirms `release_module=github.com/cervantesh/cervo-rules/v3`. |
| v3 schemas | `schemas/v3/*.schema.json` in generic archives and OCI image | Required; release consumers must receive v3 policy, vocabulary, observation, inspection, compatibility, manifest, and metadata schemas. |
| v3 dependency manifest | `release-dependencies.txt` | Required; generated with `go list -m all` for the v3 root module. |
| SBOM/provenance validation | `sbom-modules.json`, `sbom-spdx.json`, `provenance.json`, and `artifact-manifest.json` are checked by `scripts/release/verify-generic-package.sh v3.0.0-rc.3`. | Required; each machine-readable artifact must contain `"release_module": "github.com/cervantesh/cervo-rules/v3"`. |
| Signed checksums | `checksums.txt.minisig` | Published when minisign release signing is configured; strict consumers verify with `CERVORULES_VERIFY_SIGNATURES=1`. |
| OCI pull/run verification | `scripts/release/verify-oci-tools.sh v3.0.0-rc.3` | Required when OCI publishing is enabled; verifies both CLI `-version` commands plus root and v3 schemas in `/opt/cervorules/schemas`. |

## v2.1.0-rc.1 Release/Package Verification Matrix

| Check | Evidence command or artifact | RC decision |
| --- | --- | --- |
| Local release artifact build | `scripts/release/check.sh v2.1.0-rc.1` | Required before tagging; do not publish from a failing tree. |
| Generic package download | `scripts/release/verify-generic-package.sh v2.1.0-rc.1` | Required after the package workflow publishes. |
| SBOM/provenance validation | `sbom-modules.json`, `sbom-spdx.json`, `provenance.json`, and `artifact-manifest.json` are downloaded and checked by `scripts/release/verify-generic-package.sh v2.1.0-rc.1`. | Required; provenance must name `scripts/release/build-artifacts.sh`, commit, module, Go version, release version, SLSA predicate type, and source material. SPDX SBOM must contain `SPDXRef-DOCUMENT`. |
| Signed checksums RC decision | `checksums.txt.minisig` signs `checksums.txt` with minisign when `CERVORULES_MINISIGN_SECRET_KEY_FILE` is configured. | Minisign is selected for generic package checksum signatures. GPG is not selected for automated package verification; cosign is deferred to OCI image signing. |
| Clean consumer package verification | Run from a clean checkout or release operator shell: `scripts/release/verify-generic-package.sh v2.1.0-rc.1`. | Required to prove download, checksum, extract, schema, metadata, and CLI `-version` checks without relying on the release build directory. |
| OCI pull/run verification | `scripts/release/verify-oci-tools.sh v2.1.0-rc.1` | Required when OCI publishing is enabled; it pulls the image and runs `cervorules-policygen -version` and `cervorules-vocabgen -version`. |

The `Publish Packages` workflow runs for tags matching `v*` and publishes a
generic package named `cervo-rules` under the `CervoSoft` owner. The package
contains:

- `cervorules-tools-<version>-linux-amd64.tar.gz`;
- `cervorules-tools-<version>-linux-arm64.tar.gz`;
- `cervorules-tools-<version>-windows-amd64.tar.gz`;
- `cervorules-schemas-<version>.tar.gz`;
- `build-metadata.txt`;
- `dependencies.txt`;
- `sbom-modules.json`;
- `sbom-spdx.json`;
- `provenance.json`;
- `artifact-manifest.json`;
- `checksums.txt`;
- `checksums.txt.minisig` when minisign signing is configured;

The workflow requires a repository secret named `PACKAGE_TOKEN` with package
write permissions. If the container registry is enabled, set repository variable
`CERVORULES_PUBLISH_OCI=true` to also publish the `cervorules-tools` OCI image
from `build/tools/Dockerfile`. If the Actions-internal `github.server_url` is not
resolvable by Docker, set `CERVORULES_OCI_REGISTRY` to the public registry host,
for example `registry.example.com`. OCI image repository paths are pushed
with a lowercase owner because Docker rejects uppercase repository names.

For signed generic packages, configure a minisign secret key file for the release
operator or runner and pass it as `CERVORULES_MINISIGN_SECRET_KEY_FILE`.
Consumers verify signatures with:

```bash
CERVORULES_VERIFY_SIGNATURES=1 \
CERVORULES_MINISIGN_PUBLIC_KEY=<public-key> \
scripts/release/verify-generic-package.sh v2.1.0-rc.1
```

Local artifact smoke test:

```powershell
bash scripts/release/build-artifacts.sh v0.0.0-local dist
Get-ChildItem dist
```

Published generic package smoke test:

```bash
scripts/release/verify-generic-package.sh v0.0.0-smoke-YYYYMMDD-HHMM
```

Full release operator check:

```bash
scripts/release/check.sh v2.0.1
CERVORULES_VERIFY_PACKAGE=1 scripts/release/check.sh v2.0.1
```

The package workflow intentionally runs `go test -count=1 ./...` before
publishing so package artifacts are not uploaded from a failing tag. When OCI
publishing is enabled, it also performs a Docker registry login preflight before
generic artifact upload. That keeps known-bad registry routing from creating a
partial package where generic artifacts exist but the OCI image cannot be
published.

Use `workflow_dispatch` with a clearly non-release version such as
`v0.0.0-smoke-YYYYMMDD-HHMM` for package smoke tests. Delete only the smoke
package version and smoke OCI tag after verification. Never delete a real
release package version as routine cleanup.
