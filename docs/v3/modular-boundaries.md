# CervoRules v3 Modular Boundaries

Issues: #314, #315, #316, #317, #318, #319.

v3 makes modular imports the normal path. The root package is marker-only,
intentionally small, and must not become a compatibility facade. New consumers
should import concrete packages such as
`github.com/cervantesh/cervo-rules/v3/core` and
`github.com/cervantesh/cervo-rules/v3/runtime`.

## Package Ownership

| Package | Owns | Does not own |
| --- | --- | --- |
| root package | Module marker and high-level package docs. | Runtime compatibility aliases, generated policy helpers, adapters. |
| `core` | core package owns decision runtime contracts. | Runtime config, HTTP parsing, observability sinks, test helpers. |
| `runtime` | runtime package owns runtime config. | Decision evaluation and transport parsing. |
| `limits` | Requested usage and policy limit checks. | JSON/body parsing and HTTP status mapping. |
| `facts` | Optional logic/facts engine. | Core policy decisions and transport adapters. |
| `httpadapter` | Optional HTTP-to-request adapter. | Core runtime behavior. |
| `observe` | Observation and report contracts. | Logger, metrics, or tracing client dependencies. |
| `testkit` | Consumer conformance helpers. | Runtime production dependencies. |
| `decisioncache` | Optional caller-keyed cache wrapper. | Global caching in core. |

## Dependency Scope

Each v3 runtime package should avoid importing the root package. The root
package may describe the module, but ownership stays in subpackages.

`core` has the strictest dependency scope:

- no root package imports;
- no HTTP adapter imports;
- no observability package imports;
- no testkit imports;
- no `net/http` dependency.

facts, httpadapter, observe, testkit stay optional so consumers can import only
the parts they need.

## Verification

The focused hygiene script enforces the physical split:

- root module is `github.com/cervantesh/cervo-rules/v3`;
- no nested `v3/` module exists;
- v2 tooling directories are not reintroduced into the active repository;
- Go code does not import `github.com/cervantesh/cervo-rules/v2`.

Run:

```bash
scripts/ci/repo-hygiene.sh
go test -count=1 ./...
go vet ./...
go mod verify
```
