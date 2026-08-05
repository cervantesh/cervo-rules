# SonarQube

CervoRules can be scanned with a local SonarQube Community Build instance or an
existing SonarQube server. This setup is intentionally optional: Required Build
and Dependency Audit stay authoritative until SonarQube credentials are added.

## Local Server

Start SonarQube Community Build and PostgreSQL:

```bash
docker compose -f ops/sonar/docker-compose.yml up -d
```

The compose file defaults to `sonarqube:community`, the active Community Build
tag. Do not use `sonarqube:lts-community`; it currently resolves to an inactive
9.9.x image and SonarQube will show an upgrade warning.

If an existing local volume was created from `9.9.x`, a real instance must be
upgraded through the intermediate Community Build requested by SonarQube before
returning to the default image:

```bash
SONARQUBE_IMAGE=sonarqube:25.12.0.117093-community docker compose -f ops/sonar/docker-compose.yml up -d --force-recreate sonarqube
docker compose -f ops/sonar/docker-compose.yml up -d --force-recreate sonarqube
```

Do not start the latest image against old shared data before the intermediate
upgrade. Once the latest image touches the Elasticsearch data, the intermediate
image cannot safely start against that same volume.

For a disposable local proof-of-install with no projects, tokens, or scan
history to preserve, reset only this repo-local compose stack and start fresh:

```bash
CERVORULES_RESET_SONARQUBE=1 scripts/sonar/reset-local.sh
docker compose -f ops/sonar/docker-compose.yml up -d
```

Open <http://localhost:9000>. The first local login is usually `admin` /
`admin`; SonarQube will ask for a new password.

Create a token:

1. Go to `My Account` -> `Security`.
2. Generate a project or user token.
3. Export it as `SONAR_TOKEN`.

Run a scan:

```bash
SONAR_TOKEN=<token> scripts/sonar/scan.sh
```

The script runs:

```bash
go test -coverprofile=coverage.out -covermode=atomic ./...
sonar-scanner
```

If `sonar-scanner` is not installed locally, the script uses the
`sonarsource/sonar-scanner-cli` container.

## CI

Add these repository secrets before expecting the workflow to scan:

- `SONAR_HOST_URL`: for example `http://sonarqube.internal.example:9000` or a stable
  SonarQube URL.
- `SONAR_TOKEN`: token created in SonarQube.

Workflow: `.github/workflows/sonarqube.yml`.

If either secret is missing, the workflow exits successfully after recording
that SonarQube is not configured. This avoids blocking PRs while the server is
being installed or moved.

## Project Configuration

`sonar-project.properties` defines:

- project key: `cervosoft_cervo-rules`;
- Go coverage report: `coverage.out`;
- source/test layout for the single Go module;
- exclusions for generated artifacts, docs, schemas, scripts, and temporary
  outputs.

## Operational Notes

- Do not run `docker compose -f ops/sonar/docker-compose.yml down -v` unless you
  intentionally want to delete SonarQube and PostgreSQL data.
- For this repo-local proof-of-install, deleting the local volumes is acceptable
  if you need to replace an inactive test image before any projects or tokens are
  stored. Prefer `scripts/sonar/reset-local.sh` because it requires an
  explicit confirmation variable. For shared SonarQube instances, follow the
  official upgrade path and take a database backup first.
- On Linux hosts, SonarQube may require `vm.max_map_count=262144`.
- Prefer a long-lived host or VM for shared use; local Docker is fine for
  validation but not a durable team service.
- Keep Required Build, Dependency Audit, `govulncheck`, and package checks even
  after SonarQube is active. SonarQube is additive, not a replacement.

## Current Documentation Status

As of the 2026-05-23 documentation refresh, SonarQube is documented as an
optional/additive quality signal for CervoRules. Repository gates still come
from Required Build, Dependency Audit, release checks, package verification,
and the local Go verification commands. If a SonarQube issue is fixed, record
the issue/PR in the changelog or tracking issue instead of treating the browser
state as the only evidence.

