# CervoRules v3 Observability

Tracks epic #364 and child issues #365, #366, #367, #368, and #369.

## Goal

v3 operational reports are versioned, machine-readable, low-cardinality, and
safe by default.

## Report Schema

The report schema version is:

```text
cervorules.v3.policy_evaluation_report.v1
```

The machine-readable JSON schema lives at:

```text
schemas/v3/policy-evaluation-report.schema.json
```

## Metric Labels

Default metric labels are intentionally low-cardinality:

- `schema_version`;
- `outcome`;
- `operation`;
- `target`;
- `executor`.

Metric labels must not include user ids, request ids, metadata keys, tenant ids,
document ids, prompts, payload fields, or other high-cardinality values.

## Structured Logs

Default log fields include:

- `schema_version`;
- `request_id`;
- `operation`;
- `allow`;
- `target`;
- `executor`;
- `reason`;
- `diagnostics`.

Logs must not include sensitive metadata by default. Consumers that need richer
debug fields must opt in outside the default report contract.
