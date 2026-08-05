# ADR 0013: Release Signing And SBOM

Status: accepted for the v2.1 release-candidate line.

## Context

CervoRules already publishes machine-consumable release artifacts with
`checksums.txt`, `artifact-manifest.json`, `provenance.json`,
`sbom-modules.json`, and CLI archives. The remaining supply-chain gap is
authenticity: a consumer can verify bytes against `checksums.txt`, but cannot
verify who produced the checksum file.

The project considered three signing tools:

- GPG;
- minisign;
- cosign.

## Decision

Use minisign for generic package checksum signatures.

Release builds may publish:

- `checksums.txt`;
- `checksums.txt.minisig`;
- `sbom-modules.json`;
- `sbom-spdx.json`;
- `provenance.json`;
- `artifact-manifest.json`.

The signed object is `checksums.txt`, not every artifact individually. Consumers
verify the signature first, then run SHA-256 verification for all artifacts.

The release artifact builder signs checksums only when
`CERVORULES_MINISIGN_SECRET_KEY_FILE` is configured. Local unsigned builds stay
supported so contributors can run release checks without private keys.

Package verification supports strict signature mode through:

```bash
CERVORULES_VERIFY_SIGNATURES=1 \
CERVORULES_MINISIGN_PUBLIC_KEY=<public-key> \
scripts/release/verify-generic-package.sh v2.1.0-rc.1
```

## Why Minisign

Minisign is small, detached-signature oriented, easy to use from shell scripts,
and fits generic package downloads well. It does not require a Git identity,
commit signing setup, or OIDC identity plumbing.

## Rejected Alternatives

GPG is acceptable for humans, but it adds more keyring and trust-model
complexity for automated package verification.

Cosign is the right candidate for future OCI image signing and SLSA-style
attestations. It is not selected for generic package checksum signing in this
step because CervoRules needs a simple file-signature path that also works
outside container registries.

## Consequences

- Release operators need a minisign secret key file in CI or their operator
  shell.
- Consumers need the corresponding minisign public key.
- Unsigned local builds remain valid for development but must not be described
  as signed releases.
- `sbom-spdx.json` becomes the standard SBOM artifact; `sbom-modules.json`
  remains as a compact Go-module manifest for simple tooling.

## Future Work

- Add cosign signing for `cervorules-tools` OCI images after registry identity
  and keyless/keyed signing policy are selected.
- Add stricter SLSA provenance if CI runner identity can provide
  stable builder identity.
