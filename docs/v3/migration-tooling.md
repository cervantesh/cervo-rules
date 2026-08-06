# CervoRules v3 Migration Tooling

Status: deferred for native v3.

The physical split moved v2 maintenance and historical migration tooling to:

```text
cervantesh/cervo-rules-v2
```

Native v3 currently ships `cervorules-vocabgen` and `cervorules-policygen`
`check` / `generate`. It does not ship `migrate-v3`, `compat`, `diff`, or
`inspect-api`.

## Future Native Scope

A future v3 migration command should produce a schema-versioned report that can
detect:

- v2 primitive names: `Capability`, `Service`, `Provider`;
- generated `BuildPolicy` usage;
- deprecated routing helper usage;
- YAML fields `capability`, `service`, and `provider`;
- old package imports that point at the v2 maintenance module.

## Manual Replacement Table

| v2 | v3 |
| --- | --- |
| `Capability` | `Operation` |
| `Service` | `Target` |
| `Provider` | `Executor` |
| `BuildPolicy(overrides...)` | `NewPolicyFactory().Build(ctx, cfg)` |
| `capability:` | `operation:` |
| `service:` | `target:` |
| `provider:` | `executor:` |

Automated rewrite remains deferred until a concrete consumer migration needs it.
