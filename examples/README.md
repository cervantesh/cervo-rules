# Examples

Use examples as policy shapes, not as domain vocabulary to copy blindly.

## Which Example To Copy

- `billing`: payment or ledger style routing.
- `document-processing`: file pipelines with OCR, classification, and
  redaction.
- `device-routing`: fleet or device command routing.
- `logic-facts`: optional derived facts before policy evaluation.
- `performance-hot-path`: low-latency request handling with compiled
  singletons and bounded request facts.
- `conformance/*`: tiny fixtures for billing, document processing, device
  routing, queue events, scheduled jobs, CLI commands, and edge requests.

## What These Examples Do Not Cover

- They do not run traffic, retries, circuit breakers, health checks, or upstream
  calls.
- They do not choose models or profile chains.
- They do not define application-owned vocabulary outside local policy files.
- They do not replace service-specific integration tests.

## Performance Notes

For normal route tables, prefer generated policies or explicit indexed routing
plans. Use linear routing only when small ordered rule lists are intentional.

The optional `facts` package is for bounded derivation and explanations. Keep
large facts evaluation out of per-request hot paths unless benchmarks prove the
shape is acceptable, or use prepared/static facts and incremental snapshots to
move repeated work out of request handling.

See `docs/performance/hot-path.md` for the production checklist.

## Policy Shape

The v3-native generator currently supports deterministic operation routing:
`operation`, `target`, `executor`, optional `fallback_executors`, optional
`disabled_by_default`, explicit denies, and generated test cases. Compound
predicate DSL support is future v3 work, not part of the current generator
contract.

## Commands

The active repository is v3-root. These command examples use the v3-native
tooling in this repository.

Check policy files without writing generated code:

```powershell
go run ./cmd/cervorules-policygen check `
  -vocab examples/billing/policy-vocabulary.yaml `
  -policy examples/billing/policy-rules.yaml
```

Generate vocabulary constants:

```powershell
go run ./cmd/cervorules-vocabgen `
  -in examples/billing/policy-vocabulary.yaml `
  -out internal/policyvocab/generated.go `
  -package policyvocab
```

Generate policy code and tests:

```powershell
go run ./cmd/cervorules-policygen generate `
  -vocab examples/billing/policy-vocabulary.yaml `
  -policy examples/billing/policy-rules.yaml `
  -out internal/policyrules/generated_policy.go `
  -test-out internal/policyrules/generated_policy_test.go `
  -package policyrules `
  -vocab-package policyvocab `
  -vocab-import your/module/internal/policyvocab
```

## Consumer Contract Test

Generated v3 consumers should keep one hand-written conformance test beside
generated policy code. The contract should prove default generation, runtime
config, decisions, and package smoke are still valid after CervoRules upgrades.

CI or smoke tooling can use `testkit.CheckConsumerConformance` when it needs an
error-returning API instead of `testing.TB`.
