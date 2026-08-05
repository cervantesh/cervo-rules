# CervoRules v3 Primitives

Issues: #308, #309, #310, #311, #312, #313.

v3 uses neutral primitive names only:

| v3 primitive | Meaning |
| --- | --- |
| `Operation` | What a request wants to do. |
| `Target` | The logical destination selected by a decision. |
| `Executor` | The caller-owned execution choice selected by a decision. |

The v3 public API does not expose the v2 compatibility aliases for these
concepts.

## Vocabulary

v3 vocabulary validation uses operation, target, and executor terminology:

- `AllowedOperations`;
- `AllowedTargets`;
- `AllowedExecutors`;
- `ValidateOperation`;
- `ValidateTarget`;
- `ValidateExecutor`.

The generated vocabulary packages for v3 must emit constants and `Vocabulary()`
helpers using the same names. The generator implementation is completed in the
policygen/schema workstream because it also owns DSL major-version changes.

## Migration Notes

When moving handwritten v2 code to v3:

- replace request capability fields with operation fields;
- replace selected service fields with target fields;
- replace provider fields with executor fields;
- replace allowed capability/service/provider vocabulary options with the v3
  options listed above.

Do not add v2 compatibility aliases to v3 packages. Compatibility belongs in
the migration tooling, not in the v3 runtime API.
