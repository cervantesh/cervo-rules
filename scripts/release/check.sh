#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: scripts/release/check.sh <version> [dist-dir]" >&2
  exit 2
fi

version="$1"
dist_dir="${2:-dist-release-check}"

case "${version}" in
  v*) ;;
  *)
    echo "release version must start with v: ${version}" >&2
    exit 2
    ;;
esac

echo "==> standard quality gate"
scripts/ci/quality-gates.sh standard

echo "==> extended quality gate"
scripts/ci/quality-gates.sh extended

echo "==> module list"
go list -m all > "${dist_dir}.modules.tmp"
rm -f "${dist_dir}.modules.tmp"

if command -v govulncheck >/dev/null 2>&1; then
  echo "==> govulncheck"
  govulncheck ./...
else
  echo "==> govulncheck skipped: command not found" >&2
fi

echo "==> build release artifacts"
rm -rf "${dist_dir}"
scripts/release/build-artifacts.sh "${version}" "${dist_dir}"

if [[ "${CERVORULES_VERIFY_PACKAGE:-0}" == "1" ]]; then
  echo "==> verify published GitHub Release"
  scripts/release/verify-github-release.sh "${version}" "${dist_dir}-package"
else
  echo "==> release verification skipped; set CERVORULES_VERIFY_PACKAGE=1 after publishing"
fi

echo "release check completed for ${version}"
