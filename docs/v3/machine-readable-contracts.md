# CervoRules v3 Machine-Readable Contracts

Tracks epic #370 and child issues #371, #372, #373, #374, and #375.

## Goal

Agents and tooling should inspect v3 capabilities without parsing prose first.
v3 therefore versions the agent manifest schema, generated policy metadata,
the policy evaluation report, and the public API inventory.

## Schemas

Machine-readable v3 schemas:

- `.cervorules/schemas/agent-manifest.schema.json`;
- `.cervorules/schemas/task-recipe.schema.json`;
- `schemas/v3/generated-policy-metadata.schema.json`;
- `schemas/v3/policy-evaluation-report.schema.json`;
- `schemas/v3/policy-rules.schema.json`;
- `schemas/v3/policy-vocabulary.schema.json`.

## Public API Inventory

The v3 public API inventory is tracked in:

```text
docs/v3/public-api-inventory.json
```

This artifact is not a replacement for Go documentation. It is a machine-readable
index of packages, major public types, and intended ownership boundaries.

## Deferred Compatibility Reports

Native v3 does not currently ship `compat` or `inspect-api`. The historical
schema remains useful as a future target, but current machine workflows should
rely on `check -format json`, generated metadata, generated tests, and manual
policy diffs.
