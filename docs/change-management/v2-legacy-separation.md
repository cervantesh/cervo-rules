# v2 Legacy Separation

Issues: #485.

## Decision

The repository keeps v2 available as a legacy compatibility line while v3/vNext
continues without v2 compatibility shims.

Operational pointer:

```text
v2-legacy
```

The `v2-legacy` branch points at the verified `v3.0.0-rc.3` era repository
state where v2 compatibility still exists in the root module. It is a
maintenance pointer, not the default development line.

## Rules

- New v3/vNext work must follow `docs/change-management/vnext-no-v2-compatibility.md`.
- Do not add new v2 compatibility shims to v3 or vNext packages.
- Keep v2 compatibility isolated in the root v2 module while that module exists.
- Use tags for released v2 artifacts; use the `v2-legacy` branch only when a
  maintenance branch is needed.

## Physical Repo Split

A physical repository split is not required today. If it becomes necessary, use
this sequence:

1. Freeze root v2 compatibility changes.
2. Cut a final v2 maintenance tag.
3. Create a dedicated v2 repository or long-lived maintenance branch.
4. Move v3/vNext to the default development repository layout.
5. Update package docs and agent manifests to remove v2 root-facade references.

Until that split happens, the hygiene contract is:

- root v2 facade remains small;
- v3 root remains marker-only;
- future breaking work carries no v2 compatibility shims.
