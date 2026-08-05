# Physical v2/v3 Repository Split

Date: 2026-05-24

## Decision

CervoRules now uses a physical split between the v2 maintenance line and the
active v3/vNext development line.

- v2 maintenance repository:
  `https://github.com/cervantesh/cervo-rules-v2`
- active v3 repository:
  `https://github.com/cervantesh/cervo-rules`
- final v2 maintenance cut:
  `v2.1.0`
- current v3 release candidate:
  `v3.0.0-rc.3`

## Rationale

The nested v3 module worked for compatibility, but it kept two mental models in
one repository:

- root module as v2 compatibility facade;
- nested `v3/` module as future runtime.

The split makes the active repository v3-root and removes the default pressure
to carry v2 shims into future breaking work.

## What Moved

The v2 root runtime, v2 generator tooling, v2 repo-quality tests, and v2 command
packages were preserved in `cervo-rules-v2`.

The active repository now keeps:

- v3 root module `github.com/cervantesh/cervo-rules/v3`;
- v3 packages: `core`, `runtime`, `facts`, `limits`, `httpadapter`, `observe`,
  `testkit`, and `decisioncache`;
- v3 schemas and docs;
- release/package scripts adjusted for v3-root artifacts, with v3-native
  generator tooling restored in a follow-up change.

## Operational Consequences

- Do not add v2 aliases, v2 root facades, or v2 generated wrappers to the active
  repository.
- Use `cervo-rules-v2` for v2 maintenance fixes and legacy generator commands.
- Port v3 generator tooling intentionally in a new issue instead of copying v2
  internals back into the active repository.
- Treat `cmd/` and `internal/` as v3-native only. They must not import v2 or
  reintroduce v2 primitive names.

## Verification

The v2 cut was verified before publishing:

```text
go test -count=1 ./...
go vet ./...
go mod verify
```

The v3-root transition must pass:

```text
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
scripts/ci/repo-hygiene.sh
```

## Follow-Up

- Port or redesign v3-native policy/vocabulary tooling.
- Refresh release/package docs after the first v3-root package smoke.
- Decide whether the OCI image should be renamed from `cervorules-tools` to a
  runtime/schema-specific image.
