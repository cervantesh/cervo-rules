# CervoRules v3 Generated Policy Factory

Tracks epic #336 and child issues #337, #338, #339, #340, and #341.

## Goal

v3 generated policies use one canonical runtime entrypoint: `PolicyFactory`.
BuildPolicy is not part of the v3 generated API.

## Contract

Generated packages must expose a factory compatible with `v3/runtime.PolicyFactory`:

```go
type PolicyFactory interface {
    DefaultConfig() PolicyRuntimeConfig
    ValidateConfig(PolicyRuntimeConfig) error
    Build(context.Context, PolicyRuntimeConfig) (core.Engine, error)
    Metadata() PolicyMetadata
}
```

The required generated methods are:

- `Metadata`: machine-readable policy name, DSL version, generator identity, vocabulary hash, and policy hash.
- `DefaultConfig`: the generated default runtime config.
- `ValidateConfig`: structured validation before build.
- `Build`: context-aware construction of a `core.Engine`.

## Decisions

- `BuildPolicy` wrappers are removed from the v3 generated runtime API.
- `PolicyFactory` is the canonical generated entrypoint.
- `PolicyMetadata` is required for every generated policy.
- `PolicyRuntimeConfig` remains in `v3/runtime`, not `v3/core`.
- Schema and YAML field renames are deferred to #342.

## Migration Note

v2 consumers may still use generated `BuildPolicy` during the v2 support window.
v3 consumers should instantiate the generated factory, validate config, then call
`Build(ctx, cfg)`.
