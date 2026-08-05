#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

: "${SONAR_HOST_URL:=http://localhost:9000}"

if [[ -z "${SONAR_TOKEN:-}" && -z "${SONAR_LOGIN:-}" ]]; then
  cat >&2 <<'MSG'
Missing SONAR_TOKEN.

Create a token in SonarQube, then run:
  SONAR_TOKEN=<token> scripts/sonar/scan.sh
MSG
  exit 2
fi

echo "+ go test -coverprofile=coverage.out -covermode=atomic ./..."
go test -coverprofile=coverage.out -covermode=atomic ./...

scanner_args=(
  "-Dsonar.host.url=${SONAR_HOST_URL}"
)

if [[ -n "${SONAR_TOKEN:-}" ]]; then
  scanner_args+=("-Dsonar.token=${SONAR_TOKEN}")
else
  scanner_args+=("-Dsonar.login=${SONAR_LOGIN}")
fi

if command -v sonar-scanner >/dev/null 2>&1; then
  echo "+ sonar-scanner"
  sonar-scanner "${scanner_args[@]}"
  exit 0
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "sonar-scanner is not installed and docker is unavailable" >&2
  exit 2
fi

echo "+ docker run sonarsource/sonar-scanner-cli"
docker run --rm \
  --network host \
  -e SONAR_HOST_URL="${SONAR_HOST_URL}" \
  -e SONAR_TOKEN="${SONAR_TOKEN:-}" \
  -e SONAR_LOGIN="${SONAR_LOGIN:-}" \
  -v "${repo_root}:/usr/src" \
  sonarsource/sonar-scanner-cli:latest \
  "${scanner_args[@]}"

