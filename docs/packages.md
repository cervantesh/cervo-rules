# Packages

This project publishes machine-consumable release artifacts as GitHub Release
assets, plus an optional tools image on GitHub Container Registry. The release
page carries both the human notes and the artifacts automation pins.

Releases:

```text
https://github.com/cervantesh/cervo-rules/releases
```

## Package Catalog

| Artifact | Where | Description |
|---|---|---|
| Release assets | GitHub Release for the tag | v3-native `cervorules-policygen` and `cervorules-vocabgen` binaries per platform, policy/vocabulary JSON schemas, dependency/build metadata, artifact manifest, checksums, README, and changelog. |
| `cervorules-tools` | `ghcr.io/<owner>` | v3-native generator tools plus schemas for automation and release smoke checks. |

## Release Assets

The `Publish Packages` workflow attaches every artifact to the GitHub Release
for the tag. Nothing needs a secret to be created: the workflow's `GITHUB_TOKEN`
already carries `contents: write` and `packages: write`.

Published files:

- `cervorules-schemas-<version>.tar.gz`
- `build-metadata.txt`
- `dependencies.txt`
- `sbom-modules.json`
- `provenance.json`
- `artifact-manifest.json`
- `checksums.txt`

Download a pinned schema bundle:

```powershell
$version = "v3.0.0-rc.5"
gh release download $version --repo cervantesh/cervo-rules `
  --pattern "cervorules-schemas-$version.tar.gz" --pattern "checksums.txt"
```

Verify the checksum before extracting:

```powershell
Get-FileHash "cervorules-schemas-$version.tar.gz" -Algorithm SHA256
Get-Content checksums.txt
```

On Linux, verify directly with `sha256sum`:

```bash
version=v3.0.0-rc.5
sha256sum -c checksums.txt --ignore-missing
tar -xzf "cervorules-schemas-${version}.tar.gz"
```

For a complete release assets proof, use the repository verification script:

```bash
scripts/release/verify-github-release.sh v0.1.0
```

The script downloads `checksums.txt`, `artifact-manifest.json`, build metadata,
dependency manifest, and schemas. It verifies checksums, checks schema presence,
and confirms the package metadata belongs to the expected release module line.
If a release also contains a tool archive, the script verifies CLI versions.
For v2 packages, the release module path is
`github.com/cervantesh/cervo-rules/v2`; for v3 packages, it is
`github.com/cervantesh/cervo-rules/v3`.

For v3 pre-releases after the physical split, the GitHub Release is
runtime/schema-first:

```bash
scripts/release/verify-github-release.sh v3.0.0-rc.5
```

The verifier expects `release_module=github.com/cervantesh/cervo-rules/v3`,
`release-dependencies.txt`, root schemas, and `schemas/v3/*.schema.json`.
The verifier also checks v3-native tool archives when present and verifies
`cervorules-policygen -version` plus `cervorules-vocabgen -version`.

## v3.0.0-rc.1 Release/Package Verification Matrix

| Check | Consumer command or file | Expected proof |
| --- | --- | --- |
| Release assets download | `scripts/release/verify-github-release.sh v3.0.0-rc.5` | Downloads all generic files, verifies `checksums.txt`, and checks root plus v3 schemas. |
| v3 release module | `build-metadata.txt`, `artifact-manifest.json`, `sbom-modules.json`, `sbom-spdx.json`, `provenance.json` | Each artifact records `release_module=github.com/cervantesh/cervo-rules/v3` or `"release_module": "github.com/cervantesh/cervo-rules/v3"`. |
| v3 dependency manifest | `release-dependencies.txt` | Captures `go list -m all` for the v3 root module. |
| SBOM/provenance validation | `sbom-modules.json`, `sbom-spdx.json`, `provenance.json`, `artifact-manifest.json` | Verification confirms release version, v3 release module, commit presence, Go version, artifact names, SPDX marker, SLSA predicate type, source material, and builder provenance. |
| Signed checksums | `checksums.txt.minisig` | Optional minisign signature is published when release signing secrets are configured. Consumers enable strict verification with `CERVORULES_VERIFY_SIGNATURES=1`. |
| OCI pull/run verification | `scripts/release/verify-oci-tools.sh v3.0.0-rc.5` | Pulls the tools image, verifies both CLI `-version` commands, and verifies root plus v3 schemas at `/opt/cervorules/schemas`. |

## v2.1.0-rc.1 Release/Package Verification Matrix

| Check | Consumer command or file | Expected proof |
| --- | --- | --- |
| Release assets download | `scripts/release/verify-github-release.sh v2.1.0-rc.1` | Downloads all generic files, verifies `checksums.txt`, extracts a tool archive, and checks schemas. |
| SBOM/provenance validation | `sbom-modules.json`, `sbom-spdx.json`, `provenance.json`, `artifact-manifest.json` | The verification script confirms release version, module path, commit presence, Go version, artifact names, SPDX document marker, SLSA predicate type, source material, and `scripts/release/build-artifacts.sh` builder provenance. |
| Signed checksums RC decision | `checksums.txt.minisig` | Minisign signs `checksums.txt` when release signing is configured. Consumers enable strict verification with `CERVORULES_VERIFY_SIGNATURES=1` and `CERVORULES_MINISIGN_PUBLIC_KEY`. |
| Clean consumer package verification | Run `scripts/release/verify-github-release.sh v2.1.0-rc.1` outside the release `dist` directory. | Proves a consumer-style download, checksum verification, archive extraction, metadata inspection, and CLI `-version` checks. |
| OCI pull/run verification | `scripts/release/verify-oci-tools.sh v2.1.0-rc.1` | Pulls the image, verifies schema files, and runs `cervorules-policygen -version` plus `cervorules-vocabgen -version`. |

Before consumers pin a v2 package version, confirm the release commit also
passed the consumer-shaped checks:

```powershell
go test -run TestExamplesGenerateAndPassConsumerTests -count=1 .
go test -run TestGatewayShapedV2FixtureUsesGeneratedFactoryHTTPAndLimits -count=1 .
```

Those tests prove that packaged v2 tools can generate code for temp consumers
and that gateway-style adoption uses the generated factory, HTTP classifier,
runtime options, `DecisionResult`, and `CheckLimits` together.

A private repository needs no extra configuration here: `gh` reuses the login
from `gh auth login`, and CI uses the workflow's own `GITHUB_TOKEN`.

Strict signed-checksum verification:

```powershell
$env:CERVORULES_VERIFY_SIGNATURES = "1"
$env:CERVORULES_MINISIGN_PUBLIC_KEY = "<public-key>"
bash scripts/release/verify-github-release.sh v2.1.0-rc.1
```

The selected release assets signing tool is `minisign`. `GPG` is not used for
automated package verification, and `cosign` is reserved for future OCI image
signing/attestation work.

The extracted tool archive contains:

- `cervorules-policygen`
- `cervorules-vocabgen`
- `schemas/*.schema.json`
- `README.md`
- `CHANGELOG.md`
- `build-metadata.txt`
- `dependencies.txt`

Inspect release metadata:

```powershell
gh release download $version --repo cervantesh/cervo-rules `
  --pattern "build-metadata.txt" --pattern "dependencies.txt" --pattern "artifact-manifest.json"
Get-Content build-metadata.txt
Get-Content dependencies.txt
Get-Content artifact-manifest.json
```

`build-metadata.txt` records the package version, commit SHA, Go toolchain, main
module, and build timestamp. When the workflow can derive `SOURCE_DATE_EPOCH`,
the timestamp is based on that source date and the epoch value is recorded too.
`dependencies.txt` is generated with `go list -m all`, so it captures the module
dependency graph used for the release build. `artifact-manifest.json` records
the release version, commit SHA, module path, Go version, and every generated
package artifact with its SHA-256 digest and byte size. The manifest intentionally
describes the other package files; `checksums.txt` remains the checksum file used
for direct `sha256sum` verification.

## OCI Tools Image

When repository variable `CERVORULES_PUBLISH_OCI=true` is set, the same workflow
also publishes a `cervorules-tools` OCI image. The image repository uses a
lowercase owner because Docker registries reject uppercase repository paths.

Default image name:

```text
ghcr.io/cervantesh/cervorules-tools:<version>
```

Pull and inspect a pinned image:

```powershell
$version = "v3.0.0-rc.5"
docker pull "ghcr.io/cervantesh/cervorules-tools:$version"
docker run --rm "ghcr.io/cervantesh/cervorules-tools:$version" -c "cervorules-policygen -version"
docker run --rm "ghcr.io/cervantesh/cervorules-tools:$version" -c "cervorules-vocabgen -version"
```

Use `latest` only for local smoke checks. Consumers should pin release tags.

The image entrypoint is `/bin/sh`, so pass commands with `-c` or override the
entrypoint. Schemas are available inside the image at `/opt/cervorules/schemas`.

## Smoke Packages

Smoke packages are temporary package versions created to verify package
publishing without declaring a real release. Use an explicit non-release version
such as `v0.0.0-smoke-YYYYMMDD-HHMM` and never reuse a real release tag.

Safe smoke workflow:

1. Trigger `Publish Packages` with `workflow_dispatch` and the smoke version.
2. Confirm the workflow `Test` step passed.
3. If OCI publishing is enabled, confirm the OCI registry preflight passed before artifact upload.
4. Run `scripts/release/verify-github-release.sh <smoke-version>` against the GitHub Release.
5. Optionally repeat one manual authenticated download with a consumer-style read token.
6. If OCI publishing is enabled, pull the matching image tag and run both version commands.
7. Delete only the smoke package version and smoke OCI tag after verification.

Do not delete real release package versions as part of smoke cleanup. If a smoke
run accidentally used a real release version, stop and record the incident in
the release notes before touching packages.

If OCI publishing is enabled and the preflight fails, the workflow should stop
before new generic artifacts are uploaded. Treat that as an infrastructure
incident and fix registry connectivity before re-running the smoke version.

## Manual Cleanup

Manual cleanup is safe only for package versions that are clearly non-release
smoke versions and are not referenced by consumers.

Before deleting a package version:

- Confirm the version contains `smoke`, `test`, or another agreed non-release marker.
- Confirm there is no matching Git release tag.
- Confirm the release checklist does not reference the package as a published release.
- Prefer deleting only the package version or OCI tag, not the whole package namespace.

Recommended cleanup targets:

- Release: a prerelease tagged `<smoke-version>` on `cervantesh/cervo-rules`
- OCI image tag:
  `ghcr.io/cervantesh/cervorules-tools:<smoke-version>`

Keep `latest` aligned with the latest real release. If a smoke OCI run moved
`latest`, retag and push the latest real release image after deleting the smoke
tag.
