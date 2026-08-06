# Agent Commands

These commands are self-contained templates for automation. Replace placeholder
paths before running them in a consumer repository.

## Generate policy

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

```powershell
go run ./cmd/cervorules-policygen check `
  -vocab policy-vocabulary.yaml `
  -policy policy-rules.yaml `
  -format json
```

## Review policy replacement

```powershell
go run ./cmd/cervorules-policygen check `
  -vocab policy-vocabulary.yaml `
  -policy policy-rules.candidate.yaml `
  -format json

git diff --no-index -- policy-rules.current.yaml policy-rules.candidate.yaml
```

Native v3 `compat`, `inspect-api`, `diff`, `migrate-v3`, and policy `explain`
commands are intentionally not claimed here until they are redesigned for the
v3 API.

## Publish packages

```powershell
bash scripts/release/verify-github-release.sh v3.0.0-rc.5
bash scripts/release/verify-oci-tools.sh v3.0.0-rc.5
```

Use the actual release tag for future releases.

## Run conformance

```powershell
go test -run TestConsumerConformance -count=1 ./...
go test -run TestGeneratedRuntimePolicy -count=1 ./...
```

## Required local checks

```text
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
```

## Performance-sensitive checks

```powershell
go test -race ./core ./facts ./runtime ./httpadapter ./limits
bash scripts/performance/report.sh
bash scripts/performance/report.sh matrix
bash scripts/performance/profile.sh
```
