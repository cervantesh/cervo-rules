#!/usr/bin/env bash
set -euo pipefail

if [[ "${CERVORULES_RESET_SONARQUBE:-}" != "1" ]]; then
  cat >&2 <<'MSG'
Refusing to delete local SonarQube volumes.

This command is only for disposable CervoRules proof-of-install data.
Set CERVORULES_RESET_SONARQUBE=1 when you intentionally want to remove the
repo-local SonarQube containers and volumes.
MSG
  exit 2
fi

docker compose -f ops/sonar/docker-compose.yml down -v
