# CervoRules v3 Module Layout

Issues: #303, #304, #305, #306, #307, #487, #489.

## Module Path

The v3 breaking line uses:

```text
github.com/cervantesh/cervo-rules/v3
```

The active repository now keeps v3 at the repository root. v2 maintenance moved
to `CervoSoft/cervo-rules-v2` as part of the physical split.

The v3 root package is marker-only. It exposes module identity, not runtime
helpers or compatibility aliases. Consumers should import concrete subpackages
such as `github.com/cervantesh/cervo-rules/v3/core` and
`github.com/cervantesh/cervo-rules/v3/runtime`.

## Package Boundaries

| Package | Ownership |
| --- | --- |
| root | Minimal marker-only module identity; not a compatibility facade. |
| `core` | Decision runtime contracts. |
| `runtime` | Runtime policy configuration. |
| `limits` | Requested usage and limits checks. |
| `facts` | Optional logic/facts engine. |
| `httpadapter` | Optional HTTP adapter. |
| `observe` | Observation and report contracts. |
| `testkit` | Consumer conformance helpers. |
| `decisioncache` | Optional caller-keyed cache wrapper. |

## Coexistence

v2 remains available from the maintenance repository and module path:

```text
github.com/cervantesh/cervo-rules/v2
```

Do not mix v2 and v3 packages inside the same generated policy package.

## Package Registry Plan

v3 release artifacts should use the existing package channels with v3-specific
metadata:

- generic package: schemas, checksums, build metadata, module manifest, and
  v3-native `cervorules-policygen` / `cervorules-vocabgen` binaries;
- OCI image: v3-native tools plus schemas tagged with the v3 version;
- release notes: identify v3 as breaking and link the migration guide.

The first package proof happens during the v3 release/supply-chain workstream,
not in this module-layout PR.

## Verification Note

v3 PRs verify the root module directly:

```bash
go test -count=1 ./...
go vet ./...
go mod verify
```
