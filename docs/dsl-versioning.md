# DSL Versioning

CervoRules v3 currently supports the native declarative policy DSL version:
`cervorules.policy.v3`.

The Go module version and the policy DSL version are intentionally separate.
The module can release v3.x improvements while the declarative policy schema
remains `cervorules.policy.v3`.

## Supported Versions

| DSL version | Policy schema | Vocabulary schema | Status |
|---|---|---|---|
| `cervorules.policy.v3` | `schemas/v3/policy-rules.schema.json` | `schemas/v3/policy-vocabulary.schema.json` | supported |
| `cervorules.policy.v1` | `schemas/policy-rules.schema.json` | `schemas/policy-vocabulary.schema.json` | legacy schema artifact |

The canonical v3 schema `$id` values include `/schemas/v3/` so tools can cache
and compare schema identity by DSL major version.

## Schema Compatibility Matrix

| Change type | Compatible in `cervorules.policy.v3` | Requires new DSL version |
|---|---:|---:|
| New optional field with safe default | yes | no |
| New lint warning | yes | no |
| New generated metadata field | yes | no |
| New required field | no | yes |
| Existing field removed or renamed | no | yes |
| Existing field meaning changed for the same input | no | yes |
| Validation becomes stricter for previously accepted safe input | case by case | likely |

Use `cervorules-policygen check -format json`, generated tests, and manual
policy diffs when a schema or DSL change can alter policy behavior for
consumers. Native v3 diff/compat tooling is deferred.

## Warning Semantics

Warnings are authoring feedback. They are emitted by `check`, included in JSON
diagnostics, and printed in text output, but they do not fail `check`.

Warnings may be added in a v3.x release without a new DSL version when they do
not reject existing valid policies. A warning may become an error only after a
documented deprecation window or a security/correctness exception.

## Version Introduction Checklist

Before introducing a future `cervorules.policy.v4`:

- add versioned schema `$id` values;
- document migration from `cervorules.policy.v1`;
- update `docs/api-audit.md`;
- update generated policy tests and examples;
- run validation and manual compatibility review for representative consumers;
- update release notes and package verification.
