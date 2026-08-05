# Change Request Template

Use this template for CervoRules v3 changes that affect runtime behavior,
generated code, CLI behavior, schemas, release artifacts, or consumer adoption.

## Summary

- Issue:
- Parent epic:
- Branch:
- Owner:
- Start time:
- Estimate:
- Actual:
- Deviation:

## Compatibility

- Public API changed: yes/no.
- Generated code changed: yes/no.
- Schema changed: yes/no.
- CLI contract changed: yes/no.
- Consumer migration required: yes/no.
- Compatibility review:
  - `cervorules-policygen check -vocab <vocab> -policy <candidate> -format json`
  - `git diff --no-index -- <baseline> <candidate>`
- Policy registry impact:
  - baseline recorded: yes/no.
  - registry file:

## TDD Evidence

- Red test command:
- Red failure:
- Green implementation:
- Green test command:
- Refactor notes:

## Verification

- `go test -count=1 ./...`
- `go test -cover ./...`
- `go vet ./...`
- `go mod verify`
- Additional commands:
- Required Build:
- Dependency Audit:
- Docs/Wiki updated:
  - repository docs:
  - wiki pages:
  - historical snapshot wording needed: yes/no.

## Rollback

- Revert PR:
- Corrective patch:
- Package/tag action:
- Consumer action:
- Registry baseline action:

## Time Tracking

- Estimate:
- Actual:
- Deviation:
- Reason for deviation:
