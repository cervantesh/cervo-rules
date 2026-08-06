# Agent Quickstart

This guide is the copyable workflow for AI agents and automation working on
CervoRules consumers or examples.

Current version: `v3.0.0-rc.6`.

The active repository is v3-root and ships v3-native generator commands.

## Task Routing Matrix

| Task | Start with |
| --- | --- |
| Create vocabulary | `cervorules-vocabgen` and `policy-vocabulary.yaml` |
| Generate policy | `cervorules-policygen generate` |
| Validate policy | `cervorules-policygen check` |
| Legacy migration/diff | `cervantesh/cervo-rules-v2` tooling; native v3 compat tooling is future work |
| Run conformance | `testkit.MustAssertConsumerConformance` |
| Consume packages | `scripts/release/verify-github-release.sh` and `scripts/release/verify-oci-tools.sh` |
| Pick package import | `docs/package-minimal-examples.md` |
| Execute machine recipe | `.cervorules/recipes/*.json` |

Additional machine-oriented references:

- `docs/agent-commands.md` has self-contained command templates.
- `docs/agent-task-recipes.md` explains recipe selection and failure handling.
- `.cervorules/agent-manifest.json` indexes schemas, recipes, examples,
  commands, checks, and boundaries.

## Create vocabulary

Edit `policy-vocabulary.yaml`, then generate constants:

```powershell
go run ./cmd/cervorules-vocabgen `
  -in policy-vocabulary.yaml `
  -out internal/policyvocab/generated.go `
  -package policyvocab
```

## Generate policy

Edit `policy-rules.yaml`, then generate policy code and tests:

```powershell
go run ./cmd/cervorules-policygen generate `
  -vocab policy-vocabulary.yaml `
  -policy policy-rules.yaml `
  -out internal/policyrules/generated_policy.go `
  -test-out internal/policyrules/generated_policy_test.go `
  -package policyrules `
  -vocab-package policyvocab `
  -vocab-import your/module/internal/policyvocab
```

## Validate policy

Run a non-mutating validation before generation:

```powershell
go run ./cmd/cervorules-policygen check `
  -vocab policy-vocabulary.yaml `
  -policy policy-rules.yaml `
  -format json
```

## Review compatibility

Native v3 currently provides policy validation and generated tests. For a policy
replacement, validate the candidate and attach a manual diff:

```powershell
go run ./cmd/cervorules-policygen check `
  -vocab policy-vocabulary.yaml `
  -policy policy-rules.candidate.yaml `
  -format json

git diff --no-index -- policy-rules.current.yaml policy-rules.candidate.yaml
```

The old `compat`, `inspect-api`, `diff`, `migrate-v3`, and policy `explain`
commands live only in the v2 maintenance repository until they are redesigned
as v3-native commands.

## Run conformance

Generated consumers should add a contract test:

```go
func TestConsumerConformance(t *testing.T) {
    testkit.MustAssertConsumerConformance(t, contract)
}
```

Use `testkit.CheckConsumerConformance` from smoke tools when `testing.TB` is not
available.

## Consume packages

Verify the published GitHub Release:

```powershell
bash scripts/release/verify-github-release.sh v3.0.0-rc.6
```

Verify the OCI image when enabled:

```powershell
bash scripts/release/verify-oci-tools.sh v3.0.0-rc.6
```

OCI verification checks schema files plus `cervorules-policygen -version` and
`cervorules-vocabgen -version`.

## Required checks

Run before PR:

```text
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
```

## Documentation refresh

For documentation-only work, use the same change-management discipline:

```text
create or update issue
record estimate, start time, actual, and deviation
update repository docs first
update wiki summary pages second
run focused docs drift tests
link PR and wiki pages in the issue
```

Historical report pages should be marked as historical snapshots when their
original context matters.
