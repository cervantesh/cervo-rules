# Runtime Environment Mapping

This guide describes the recommended pattern for mapping deployment
environment variables into `PolicyRuntimeConfig` without putting
consumer-specific names into CervoRules.

## Goals

- Keep environment variable names in the consumer.
- Convert values into typed CervoRules runtime config at startup.
- Validate once before serving traffic.
- Build a stable `CompiledDecisionFlow`.
- Avoid hand-written compatibility checks around generated policy internals.

## Startup Pattern

```go
func buildPolicyFromEnv() (*cervorules.CompiledDecisionFlow, error) {
    cfg := cervorules.PolicyRuntimeConfig{
        DefaultProvider: cervorules.Provider(os.Getenv("PROXY_DEFAULT_PROVIDER")),
        CapabilityRoutes: map[cervorules.Capability]cervorules.Service{},
    }

    if planning := os.Getenv("PROXY_PLANNING_SERVICE"); planning != "" {
        cfg.CapabilityRoutes["planning"] = cervorules.Service(planning)
    }
    if media := os.Getenv("PROXY_MEDIA_SERVICE"); media != "" {
        cfg.CapabilityRoutes["media_request"] = cervorules.Service(media)
    }

    cfg = cervorules.MergePolicyRuntimeConfig(policyrules.DefaultRuntimeConfig(), cfg)
    if err := policyrules.ValidateRuntimeConfig(cfg); err != nil {
        return nil, err
    }
    return policyrules.BuildPolicy(cfg)
}
```

## Recommended Mapping Table

| Environment input | Runtime field | Notes |
| --- | --- | --- |
| Default provider | `DefaultProvider` | Empty means generated default remains active. |
| Trusted users list | `TrustedUsers` | Preserve empty vs nil semantics: nil keeps defaults, empty slice clears. |
| Planning backend | `CapabilityRoutes["planning"]` | Capability name belongs to generated vocabulary. |
| Media backend | `CapabilityRoutes["media_request"]` | Enables optional disabled route when configured. |
| Fallback JSON | `ProviderFallbacks` | Validate all providers with generated vocabulary. |
| Limits JSON | `Limits` | Negative values fail startup validation. |
| Retry JSON | `RetryPolicy` | HTTP statuses must be `100..599`; backoffs must be non-negative. |
| Breaker JSON | `BreakerPolicy` | Threshold and cooldown must be non-negative. |

## Error Handling

Return validation errors directly from startup. CervoRules includes field paths
in runtime config errors, for example:

- `policy runtime config DefaultProvider: invalid rule: unknown provider "missing"`
- `policy runtime config RetryPolicy.BackoffMax: must be >= RetryPolicy.BackoffMin`
- `policy runtime config ProviderFallbacks["standard_provider"][0]: invalid rule: unknown provider "missing"`

Consumers should add environment-variable context when useful, but should not
reimplement vocabulary validation.

## Merge Semantics

`MergePolicyRuntimeConfig` overlays configs in order:

- later `TrustedUsers` replaces earlier trusted users when non-nil;
- later `DefaultTimeout` replaces earlier timeout when positive;
- later `DefaultProvider` replaces earlier provider when non-empty;
- later pointer fields replace earlier pointer fields when non-nil;
- `CapabilityRoutes` merges key-by-key;
- `ProviderFallbacks` merges key-by-key and defensively copies slices.

This lets consumers compose generated defaults, file config, and environment
overrides predictably.

| Field shape | Runtime meaning |
| --- | --- |
| `TrustedUsers == nil` | Preserve earlier trusted users. |
| `TrustedUsers == []string{}` | Clear earlier trusted users. |
| Empty scalar string | Preserve earlier scalar value. |
| Non-empty scalar string | Replace earlier scalar value. |
| Nil pointer field | Preserve earlier pointer config. |
| Non-nil pointer field | Replace earlier pointer config, then validate operational values. |
| Nil map | Preserve earlier map entries. |
| Non-nil map | Merge by key and replace matching keys. |

## CervoProxy Notes

CervoProxy should keep names such as `PROXY_MEDIA_SERVICE` local. The generated
policy should only receive typed runtime config. This keeps CervoRules generic
and makes gateway behavior testable through generated policy contracts.
