# CervoRules Agent Guide

Current release candidate: `v3.0.0-rc.5`.

Primary module: `github.com/cervantesh/cervo-rules/v3`.

CervoRules is a deterministic policy decision library. It is not a proxy,
controller, scheduler, model selector, data plane, logger, metrics backend, or
transport parser.

## Start Here

1. Read this file first.
2. Read `README.md` for the human overview.
3. Read `docs/agent-quickstart.md` for copyable machine workflows.
4. Read `.cervorules/agent-manifest.json` for machine-readable entrypoints,
   packages, commands, checks, and boundaries.
5. Read `docs/v3/api-reference.md` before changing v3 contracts.
6. Read `docs/v3/migration-v2-to-v3.md` only when touching legacy migration work.

## Primary V3 Surface

Use these names in new work:

- `Operation`
- `Target`
- `Executor`
- `PolicyFactory`

Generated v3 policy packages should expose `runtime.PolicyFactory` and should
not expose old generated construction wrappers.

## Package Routing

| Need | Package |
| --- | --- |
| Build or evaluate policy decisions | `core` |
| Use generated policy factories and runtime config | `runtime` |
| Check selected policy limits | `limits` |
| Derive bounded facts | `facts` |
| Classify HTTP inputs at the adapter boundary | `httpadapter` |
| Produce reports and audit envelopes | `observe` |
| Certify generated consumers | `testkit` |

Prefer modular imports. The v3 root package is marker-only and is not a
compatibility facade. New v3 code should import concrete subpackages directly,
for example:

```go
import (
	"github.com/cervantesh/cervo-rules/v3/core"
	"github.com/cervantesh/cervo-rules/v3/runtime"
)
```

## Required Checks Before PR

Run these before opening or updating a PR:

```text
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
```

For the v3 module, also run:

```text
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
```

For release, facts, generator, or performance changes also run the relevant
release, race, fuzz, benchmark, or package checks documented in `docs/release.md`.

After SonarQube is configured, run `scripts/sonar/scan.sh` for static analysis;
see `docs/operations/sonarqube.md`.

## Policy And Vocabulary Workflow

Use `cervorules-vocabgen` for v3 vocabulary constants.

Use `cervorules-policygen check` before changing generated code.
Use `cervorules-policygen generate` to produce v3 generated policy code and tests.
Use v2 maintenance tooling from `cervantesh/cervo-rules-v2` only for legacy
`migrate-v3`, `compat`, `diff`, or `inspect-api` workflows until those commands
are intentionally redesigned for v3.

Generated consumers should keep `testkit` contract tests beside generated
policy packages.

## Machine-Readable Artifacts

Schemas:

- `.cervorules/schemas/agent-manifest.schema.json`;
- `.cervorules/schemas/task-recipe.schema.json`;
- `schemas/policy-rules.schema.json`;
- `schemas/policy-vocabulary.schema.json`;
- `schemas/v3/*.schema.json`.

Task recipes:

- `.cervorules/recipes/generate-policy.json`;
- `.cervorules/recipes/validate-policy.json`;
- `.cervorules/recipes/compare-policy.json`;
- `.cervorules/recipes/publish-packages.json`;
- `.cervorules/recipes/run-conformance.json`.

## Agnosticism Rules

- Keep domain vocabulary in the consuming project.
- Keep transport parsing outside `core`.
- Keep HTTP-specific code in `httpadapter`.
- Keep model/profile selection outside CervoRules.
- Keep logs, metrics, tracing, and audit storage outside CervoRules.
- Add shared adapters only after at least two domains need the same shape.
- Do not add CervoProxy, gateway, AI, provider-specific payload, or tenant
  concepts to core APIs.

## No-Touch Boundaries Without ADR

Do not change public Go APIs without ADR and change management.
Do not change the module path `github.com/cervantesh/cervo-rules/v3`.
Do not add new v2 compatibility shims, root-facade aliases, or generated
compatibility wrappers to v3 or later breaking work; see
`docs/change-management/vnext-no-v2-compatibility.md`.
Do not add runtime dependencies for logging, metrics, tracing, package
publishing, schema validation, or YAML parsing.
Do not weaken release, compatibility, freshness, conformance, or drift checks.

## Deprecated API Hygiene

Current-facing docs and examples must use the v3 names and factory shape.
Deprecated or removed APIs belong only in explicit migration, history, or
deprecation-policy docs. Do not add them to README, AGENTS, quickstarts, or
copyable examples.

## TDD And Change Management

Use TDD for behavior changes:

1. Add the failing test.
2. Run it and confirm the expected failure.
3. Implement the minimal change.
4. Run the targeted test.
5. Run the required checks.

Use issues for change management. Each issue should record estimate,
start time, actual time, deviation, linked PR, tests run, and maturity impact.
Documentation-only work must still update the issue with touched repository
docs, touched wiki pages, drift tests, and any historical snapshot wording.

