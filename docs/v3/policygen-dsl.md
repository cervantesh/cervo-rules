# CervoRules v3 Policygen DSL

Tracks epic #342 and child issues #343, #344, #345, #346, and #347.

## Goal

v3 policy authoring uses neutral names in YAML and schema contracts:

- `operation` replaces `capability`;
- `target` replaces `service`;
- `executor` replaces `provider`;
- `operation_in` replaces `capability_in`;
- `target_healthy` replaces `service_healthy`;
- `fallback_executors` replaces `fallback_providers`.

The DSL major version is `cervorules.policy.v3`.

## Schema Files

The v3 schema contracts live under `schemas/v3/`:

- `schemas/v3/policy-rules.schema.json`;
- `schemas/v3/policy-vocabulary.schema.json`.

These schemas reject deprecated v2 field names. v2 schemas remain available at
the existing paths and continue to describe `cervorules.policy.v1`.

## Compatibility / Diff Expectations

Native v3 `compat` and `inspect-api` are deferred. Until they are redesigned,
review policy changes with `check -format json`, generated tests, and manual
diffs. Future migration tooling should report replacements such as:

- `capability -> operation`;
- `service -> target`;
- `provider -> executor`;
- `BuildPolicy -> PolicyFactory.Build`.

## Implemented Generator Scope

The current v3 parser and generator cover the routing subset: operations,
targets, executors, fallbacks, disabled routes, denies, default config,
metadata, config validation, and generated tests. Compound predicates, generated
limits, generated derived facts, and policy explain need separate v3 issues.
