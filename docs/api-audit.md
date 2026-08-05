# API Audit

CervoRules v3 treats public API shape as a product surface. Any change to the
runtime packages, testkit package, generated policy API, generated
vocabulary API, CLI contract, schemas, or release artifacts must pass this
audit before merge.

## Public API Inventory

- Root package: marker-only module package.
- Modular packages: exported types, functions, interfaces, methods, constants.
- `testkit`: exported contracts and assertion/check helpers.
- Generated policy package: factory, metadata, runtime config helpers,
  generated tests.
- Generated vocabulary package: constants, static vocabulary helper, validation
  contracts.
- CLI: subcommands, flags, output fields, exit codes.
- Schemas: accepted fields, required fields, version identifiers.
- Release artifacts: package file names, manifests, checksums, image names.

## Cross-Domain Review

- Does the change assume CervoProxy vocabulary, transport, provider, or payload
  semantics?
- Can the same API serve at least two neutral examples?
- Is the change adapter-specific? If yes, keep it outside the core identity of
  the library.
- Does ADR 0001 require a new ADR before this public field or method is added?

## Policy Review Tooling

- Run `cervorules-policygen check -format json` before accepting regenerated
  policy packages into a consumer.
- Attach a manual policy diff while native v3 `diff` / `compat` tooling is
  deferred.
- Include generated test output in issue or PR evidence when policy behavior
  changes.

## Audit Outcome

- Compatible:
- Breaking:
- Deferred:
- Follow-up issue:
