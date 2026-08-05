# Agent Task Recipes

Machine-readable recipes live under `.cervorules/recipes`. Each recipe uses
`.cervorules/schemas/task-recipe.schema.json`.

| Recipe | Purpose |
| --- | --- |
| `.cervorules/recipes/generate-policy.json` | Generate policy code and tests from vocabulary and policy YAML. |
| `.cervorules/recipes/validate-policy.json` | Validate policy YAML without mutating generated files. |
| `.cervorules/recipes/compare-policy.json` | Compare baseline and candidate policies for compatibility risk. |
| `.cervorules/recipes/publish-packages.json` | Verify generic and OCI packages for a release tag. |
| `.cervorules/recipes/run-conformance.json` | Run generated-policy or consumer conformance contracts. |

Agents should load `.cervorules/agent-manifest.json` first, then follow the
recipe matching the task. The recipe command list is intentionally explicit so
an agent can execute it without relying on chat history.

## Failure Handling

- If a recipe command fails, stop and attach the command output to the issue.
- If a recipe reports breaking compatibility, do not generate or publish until
  the issue and PR explicitly accept the break.
- If release package verification fails, do not mark the tag as consumable.
- If conformance fails, treat the generated policy or consumer contract as
  incorrect until proven otherwise.
