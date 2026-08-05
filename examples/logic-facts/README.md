# Logic Facts Example

This example shows the optional `facts` package without generated policy YAML.

It models a neutral billing-style relationship:

```text
tenant_plan("tenant-a", "enterprise")
  -> priority_lane("tenant-a")
```

Use this pattern when a project needs derived, explainable facts before calling
the core CervoRules policy engine. The facts package remains optional and does
not add transport or project vocabulary to core.

For request hot paths, benchmark the concrete fact set and rule shape. The
runtime has predicate and term indexes plus prepared and incremental APIs, but
large recursive derivations should still be bounded and, where possible,
precomputed outside the request handler.
