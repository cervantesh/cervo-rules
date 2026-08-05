# Policy Authoring

Use this guide when editing `policy-vocabulary.yaml` and `policy-rules.yaml`
for a generated CervoRules policy package.

## Authoring Workflow

1. Update vocabulary first: operations, targets, and executors.
2. Update policy second: `version`, `name`, defaults, routes, disabled routes,
   denies, fallbacks, and declarative tests.
3. Run `cervorules-policygen check` before generating code.
4. Fix errors first, then review warnings.
5. Generate policy and generated tests.
6. Run consumer contract tests with `testkit`.
7. Run `cervorules-policygen check -format json` and attach a manual policy
   diff before adopting the candidate in a consumer service.

## Conflict Semantics

- A route must use `operation`, `target`, and `executor`.
- A deny must include an `operation`; `reason` is recommended.
- `disabled_by_default` removes the route from defaults and emits a deny reason
  for that operation.
- Unknown vocabulary references are errors.
- Duplicate fallback executor entries should be cleaned up during review.

## Lint And Warnings

Warnings are future authoring feedback. Current v3-native `check` returns
success or a validation error.

## Check Before Generate

Run:

```bash
cervorules-policygen check \
  -vocab policy-vocabulary.yaml \
  -policy policy-rules.yaml \
  -format text
```

Use `-format json` when CI needs machine-readable diagnostics.

## Compatibility Before Adoption

Before replacing a generated policy in a consumer project, validate the
candidate and compare it with the current baseline:

```bash
cervorules-policygen check \
  -vocab policy-vocabulary.yaml \
  -policy policy-rules.candidate.yaml \
  -format json

git diff --no-index -- policy-rules.current.yaml policy-rules.candidate.yaml
```
