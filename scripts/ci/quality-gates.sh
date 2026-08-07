#!/usr/bin/env bash
set -euo pipefail

mode="${1:-standard}"

run() {
  echo "+ $*"
  "$@"
}

run go test -count=1 ./...
run go test -cover ./...
run go vet ./...
run go mod verify

# The only enforcement of the Operation/Target/Executor naming contract. It was
# invoked by no workflow and no script.
run bash scripts/ci/repo-hygiene.sh

# The published JSON Schemas are not readable by any Go check without a new
# dependency, so the documents are validated against an external
# implementation. Missing tooling skips unless
# CERVORULES_REQUIRE_SCHEMA_VALIDATION=1, which CI sets.
if command -v python >/dev/null 2>&1; then
  run python scripts/ci/validate-schemas.py
elif command -v python3 >/dev/null 2>&1; then
  run python3 scripts/ci/validate-schemas.py
else
  echo "schema validation skipped: no python interpreter found"
  if [[ "${CERVORULES_REQUIRE_SCHEMA_VALIDATION:-0}" == "1" ]]; then
    exit 1
  fi
fi

case "${mode}" in
  standard)
    ;;
  extended)
    run go test -race ./...
    run go test -bench=. -benchmem -run '^$' ./...
    ;;
  *)
    echo "usage: scripts/ci/quality-gates.sh [standard|extended]" >&2
    exit 2
    ;;
esac
