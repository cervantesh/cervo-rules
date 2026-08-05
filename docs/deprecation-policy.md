# Deprecation Policy

CervoRules v2 can deprecate APIs, generated functions, schema fields, or CLI
flags, but deprecation must be visible, tested, documented, and reversible for
consumers during the deprecation window.

## Deprecation Window

- Public Go APIs: keep deprecated wrappers for at least one minor release unless
  security or correctness requires faster removal.
- Generated code wrappers: keep for one generator release cycle after migration
  docs and generated tests are available.
- CLI flags/subcommands: keep aliases for one minor release when practical.
- Schema fields: keep validation support and warnings before removal unless the
  field is unsafe.

## Migration Requirements

- Add a `Deprecated:` Go doc comment for Go APIs.
- Add before/after snippets to `docs/migration-v1-to-v2.md` or a versioned
  migration note.
- Add or update tests for old and new paths while the old path exists.
- Include issue, PR, and release-note references.
- State whether generated consumers must regenerate code.

## Removal Criteria

- The deprecation window has elapsed.
- Migration docs exist and point to the replacement.
- Tests cover the replacement path.
- Release notes identify the removal.
- API audit confirms the removal is intentional.
- Rollback path is documented if package consumers report breakage.

## Exceptions

Immediate removal is allowed only for security, data loss, or severe correctness
risk. The PR must explain why a normal deprecation window is unsafe.

## Current Deprecation Notes

- `RoutingPhase(rules...)` is deprecated because the name hides linear
  evaluation cost. Use `CapabilityRoutingPhase`, `OperationRoutingPhase`, or
  `PolicyBuilder` for normal route tables. Use `LinearRoutingPhase` only when
  global rule order is intentional and the rule count is small.
