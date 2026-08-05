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
