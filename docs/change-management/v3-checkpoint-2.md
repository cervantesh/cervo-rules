# CervoRules v3 Checkpoint 2

Epic: #302

Checkpoint issue: #360

Child issues: #361, #362, #363

Date: 2026-05-24

## Scope Reviewed

This checkpoint covers workstreams 7-10:

| Workstream | Issue | Result |
| --- | --- | --- |
| Generated policy factory as only generated runtime API | #336 | Keep. v3 has a canonical `runtime.PolicyFactory` contract and forbids generated `BuildPolicy`. |
| Policygen schema and DSL breaking cleanup | #342 | Keep as schema contract. Full v3 parser/codegen implementation remains deferred. |
| Facts as explicit optional logic engine | #348 | Keep. v3 facts has separate contracts, versioned plan JSON, complexity diagnostics, and trace opt-in defaults. |
| Remove implicit or ambiguous linear routing | #354 | Keep. v3 routing names make indexed and linear cost explicit. |

## Decisions

### Keep

- Keep `PolicyFactory` as the only v3 generated runtime entrypoint.
- Keep `BuildPolicy` out of the v3 module and generated API.
- Keep `cervorules.policy.v3` as the v3 DSL major version.
- Keep v3 schema fields as `operation`, `target`, `executor`, and `fallback_executors`.
- Keep facts optional and separate from core decisions.
- Keep facts trace disabled by default.
- Keep indexed routing as the normal generated-policy path.
- Keep explicit linear routing for small ordered or non-indexable policies.

### Drop

- Drop free generated runtime helpers from the v3 generated API unless a later migration fixture proves they are needed.
- Drop any attempt to make facts part of the mandatory decision runtime.
- Drop implicit routing names that hide O(n) behavior.

### Defer

- Defer full v3 policy parser/codegen implementation until after the next machine-readable contract and migration tooling workstreams clarify output expectations.
- Defer actual v3 facts evaluator implementation; current v3 facts work is a stable contract shell.
- Defer v2-to-v3 automated rewrite tooling to #376.
- Defer package/release validation for v3 artifacts to #392 and #404.

## Answers To Required Questions

### Is the generated factory API complete enough?

Yes for a contract checkpoint. It defines required config, validation, build, and metadata surfaces. It is not yet a full generated implementation.

### Did schema renames improve clarity enough to justify breakage?

Yes. `operation`, `target`, and `executor` are more reusable outside gateway/provider domains. The breakage is justified for v3, but migration tooling must provide precise replacements.

### Is `facts` too prominent or too hidden?

The current position is correct: facts are visible as an optional package with explicit plan and diagnostic contracts, but not required by core.

### Should any removed routing helper return as compatibility tooling?

No runtime helper should return under the old ambiguous name. Migration tooling should detect `RoutingPhase(` and suggest explicit indexed or linear replacements.

### What new consumer pain appeared in generated examples?

The largest pain is not the factory contract; it is that a complete v3 generator must either implement a small v3 engine or introduce a v3 policy builder. That should be resolved before v3 RC.

## Sequence Adjustment

No issue order change is required. Continue to #364 observability schema versioning, then #370 machine-readable contracts, #376 migration tooling, and #382 consumer conformance.

## Next Sequence

Proceed to #364 after this checkpoint is merged and #360-#363 are closed.
