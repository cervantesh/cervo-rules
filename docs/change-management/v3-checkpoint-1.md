# CervoRules v3 Checkpoint 1

Epic: #302

Checkpoint issue: #332

Child issues: #333, #334, #335

Date: 2026-05-24

## Scope Reviewed

This checkpoint covers the first five v3 workstreams:

| Workstream | Issue | Result |
| --- | --- | --- |
| Module path and repository layout | #303 | Keep. Nested `v3/` module with `github.com/cervantesh/cervo-rules/v3` is working. |
| Primitive rename | #308 | Keep. v3 public core uses `Operation`, `Target`, and `Executor`. |
| Modular imports | #314 | Keep. v3 root remains marker-only and packages own their boundaries. |
| Structured errors | #320 | Keep. `ErrorCode`, `Error`, and `Errors` are the common v3 shape. |
| DecisionResult and options cleanup | #326 | Keep. Runtime envelope and production-oriented options are explicit. |

## Decisions

### Keep

- Keep `Operation`, `Target`, and `Executor` as the v3 primitive names.
- Keep the nested `v3/` module and test it separately from the root v2 module.
- Keep the v3 root package minimal. Consumers should import `v3/core`, `v3/runtime`, `v3/facts`, or other owned packages directly.
- Keep structured errors as the common machine-readable contract.
- Keep trace and observation opt-in by default for v3. This matches the production hot-path goal.
- Keep diagnostics as runtime output on `DecisionResult`, not as caller-selected `DecisionOptions`.

### Drop

- Drop v2 alias names from v3 public API. v3 should not expose `Capability`, `Service`, or `Provider`.
- Drop generated `BuildPolicy` wrappers from the eventual v3 generated runtime API unless a later checkpoint finds a migration blocker.
- Drop value-shaped observation semantics in v3. `nil` means not materialized.

### Defer

- Defer generated vocabulary output using `Operation`, `Target`, and `Executor` to #342.
- Defer v3 runtime config and policy factory implementation to #336 and #342.
- Defer facts runtime/API decisions to #348.
- Defer release tag shape for nested module artifacts to #392.

## Answers To Required Questions

### Are `Operation`, `Target`, and `Executor` still the right names?

Yes. They are more domain-neutral than the v2 names and are broad enough for routing, authorization, workflow dispatch, device orchestration, document processing, and billing examples.

### Did modular imports make the API easier or just more fragmented?

They make the API easier for machines and consumers because ownership is explicit. The risk is discoverability, so `AGENTS.md`, package docs, and generated examples must point to the correct package instead of relying on a large root facade.

### Did production defaults create migration pain?

Yes, intentionally. v2 defaulted toward rich envelope data for compatibility, while v3 defaults toward low-allocation runtime behavior. Migration docs must call out that trace and observation are opt-in.

### Which removed v2 APIs need migration helpers outside runtime?

- Primitive names: migration tooling should report `Capability -> Operation`, `Service -> Target`, and `Provider -> Executor`.
- Generated policy entrypoint: migration tooling should report `BuildPolicy` and suggest `PolicyFactory`.
- Routing helpers: migration tooling should report implicit `RoutingPhase` usage and suggest explicit indexed or linear routing.

### What should be dropped before continuing?

No completed v3 workstream should be reverted. The only cleanup is to keep diagnostics out of `DecisionOptions`; that was already applied in #326.

## ADR Need

No new ADR is required from checkpoint 1. The current decisions stay aligned with `docs/change-management/v3-breaking-api.md`. If the nested module release tag shape changes from the current package plan, that should be documented during #392.

## Next Sequence

Proceed to #336: generated policy factory as the only generated runtime API.

Blockers before proceeding:

- #332-#335 must be closed.
- This checkpoint document must be linked from `docs/change-management/v3-breaking-api.md`.
