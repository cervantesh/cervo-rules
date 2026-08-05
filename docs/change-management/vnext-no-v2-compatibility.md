# vNext No v2 Compatibility

Issue: #481.

## Decision

The next breaking line after the current v3 release-candidate work will not
carry v2 compatibility shims, v2 aliases, or root-facade behavior forward.

v2 remains available through its existing module path and tags:

```text
github.com/cervantesh/cervo-rules/v2
```

Future major-version design work should optimize for the new public contracts
instead of preserving v2 source compatibility.

## Rules

- Do not add new v2 compatibility aliases to v3 or future major modules.
- Do not add generated `BuildPolicy` compatibility wrappers to future generated
  policy APIs.
- Do not make the v3 root package a facade.
- Do not preserve `Capability`, `Service`, or `Provider` as public aliases in
  future major lines.

## Rationale

The repository now keeps v2 compatibility in the v2 module and keeps v3
marker-only at the root. Carrying v2 compatibility forward would reintroduce the
same surface-area and ambiguity that v3 was created to remove.

## Release Impact

`v3.0.0-rc.3` still includes the v2 root facade because the repository root is
the v2 module. This decision applies to future breaking work and prevents new
compatibility shims from entering v3 or later major modules.
