# Dependency Acceptance Policy

CervoRules keeps the runtime dependency surface intentionally small because the
runtime library is reused by downstream services. External packages are allowed
only when the accepted scope, consumer impact, and update policy are documented.

## Dependency Classes

| Class | Examples | Acceptance bar |
| --- | --- | --- |
| Runtime core | Code imported by `core`, `runtime`, `limits`, `observe`, and the Root facade at request time. | Highest. Prefer standard library. Requires explicit justification and consumer-impact review. |
| Generator/tooling | `internal/policygen`, `internal/vocabgen`, CLI commands, schema validation. | Acceptable when it removes meaningful parser/schema risk or improves generated-policy safety without leaking into generated runtime code. |
| Release tooling | Packaging scripts, package metadata, audit workflows. | Acceptable when it improves reproducibility, auditability, or supply-chain visibility. |
| Test-only | Unit, fuzz, benchmark, or contract tests. | Acceptable when it materially improves coverage without leaking into runtime packages. |

Runtime dependencies must not require background services, global state, network
access, cgo, or heavyweight transitive graphs unless a release owner accepts the
downstream impact explicitly. Generator dependencies may be accepted for YAML
parsing, schema validation, and deterministic code generation, but generated
packages must compile without importing generator-only packages.

Generated policy packages import `core`, `runtime`, and `limits` directly. They
must not import the root facade, generator internals, YAML parsers, or schema
validators. The Root facade exists for source compatibility and should remain a
thin alias/delegation layer over public subpackages.

## Review Requirements

Every new dependency proposal or material expansion of an existing dependency
must state:

| Field | Required decision |
| --- | --- |
| Purpose | What capability the package provides, who owns the decision, and which code path uses it. |
| Standard library alternative | Whether the Go standard library can solve the problem, and why it is accepted or rejected. |
| Runtime/generator scope | Whether the package is runtime, generator/tooling, release, or test-only. |
| License | License name and whether it is compatible with CervoRules distribution and downstream reuse. |
| Update policy | How patch, minor, major, and security updates are evaluated and when they are allowed into a release. |
| Consumer impact | Whether downstream modules inherit the dependency, generated code changes, binary size changes, package contents change, or operational requirements change. |

Runtime-core dependencies require a stronger reason than generator dependencies.
If the same result can be achieved with the standard library without excessive
complexity, keep the runtime dependency-free.

## Current Accepted Dependencies

| Dependency | Scope | Purpose | Acceptance notes |
| --- | --- | --- | --- |
| `gopkg.in/yaml.v3` | Generator/tooling | Parse vocabulary and policy YAML for `policygen` and `vocabgen`. | Accepted because the standard library does not include YAML parsing. Keep it out of generated runtime code. |
| `github.com/santhosh-tekuri/jsonschema/v6` | Generator/tooling | JSON Schema validation used by generator/check workflows. | Accepted for schema validation coverage. Keep it out of generated runtime code and review updates during release dependency checks. |
| `github.com/dlclark/regexp2` | Generator/tooling transitive | Regular expression support pulled by JSON Schema validation. | Accepted only through `jsonschema/v6`; recheck if it becomes direct or runtime-scoped. |
| `golang.org/x/mod` | Toolchain/test build-list transitive | Appears in `go list -m all` through the Go dependency graph, but `go mod why -m` reports the main module does not need it directly. | Accepted only as build-list metadata; investigate if it appears in `go.sum` or becomes reachable from CervoRules packages. |
| `golang.org/x/sys` | Toolchain/test build-list transitive | Appears in `go list -m all` through the Go dependency graph, but `go mod why -m` reports the main module does not need it directly. | Accepted only as build-list metadata; investigate if it appears in `go.sum` or becomes reachable from CervoRules packages. |
| `golang.org/x/text` | Generator/tooling transitive | Transitive text support through schema validation. | Accepted only as a transitive dependency. Recheck if it becomes direct or runtime-scoped. |
| `golang.org/x/tools` | Toolchain/test build-list transitive | Appears in `go list -m all` through the Go dependency graph, but `go mod why -m` reports the main module does not need it directly. | Accepted only as build-list metadata; investigate if it appears in `go.sum` or becomes reachable from CervoRules packages. |
| `gopkg.in/check.v1` | Generator/tooling transitive test dependency | Test dependency pulled by `gopkg.in/yaml.v3`. | Accepted only as an upstream test dependency; recheck if it becomes direct or runtime-scoped. |

No dependency is currently accepted as a required runtime-core dependency.

## Required Checks

Run these before release and after dependency changes:

```powershell
go list -m all
go mod verify
go test -count=1 ./...
go vet ./...
govulncheck ./...
```

The `Dependency Audit` workflow runs the same class of checks in CI.
Compare `go list -m all` and package manifest output against this file before
tagging a release.

SonarQube may add useful static-analysis findings, but it is not a dependency
acceptance gate by itself. Dependency acceptance still depends on module
inventory, runtime/generator scope, vulnerability checks, package manifests, and
release verification.

The repository also includes `TestDependencyScope`, which runs as part of
`go test ./...`. That guardrail fails the build when:

- a new direct module is added without being explicitly allowed;
- the root runtime package imports any external module package;
- generator-only imports leak into runtime, `testkit`, or other non-generator
  packages;
- a module in `go list -m all` is missing from this document.

## Release Metadata

Release packages include:

- `build-metadata.txt`: version, commit, build time, Go version, module path;
- `dependencies.txt`: `go list -m all` output;
- `sbom-modules.json`: JSON module inventory derived from `go list -m all`;
- `provenance.json`: release build provenance with version, commit, builder,
  workflow, run id, Go version, module path, and build time;
- `checksums.txt`: SHA-256 checksums for published package files.

Consumers should inspect `dependencies.txt` when pinning CervoRules tools in
another project, and should verify `checksums.txt` before extracting archives.
If signed checksum publishing is enabled for a future release, the signature
must cover `checksums.txt` and be referenced from the release notes.
