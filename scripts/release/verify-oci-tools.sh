#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
usage: scripts/release/verify-oci-tools.sh <version> [image]

Pulls and verifies the CervoRules OCI image for a release or smoke version.
Pass the image explicitly, or let it default to GitHub Container Registry:
  ghcr.io/<owner>/cervorules-tools:<version>

The owner comes from GITHUB_REPOSITORY_OWNER, or from the git remote.

Set CERVORULES_OCI_SKIP_PULL=1 when verifying a local image tag built in the
current Docker daemon.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 2
fi

version="$1"
image="${2:-}"
if [[ -z "${image}" ]]; then
  owner="${GITHUB_REPOSITORY_OWNER:-}"
  if [[ -z "${owner}" ]]; then
    origin="$(git remote get-url origin 2>/dev/null || true)"
    origin="${origin%.git}"
    origin="${origin#*github.com[:/]}"
    owner="${origin%%/*}"
  fi
  if [[ -z "${owner}" ]]; then
    echo "pass an image argument, or set GITHUB_REPOSITORY_OWNER; see --help" >&2
    exit 1
  fi
  owner="$(printf '%s' "${owner}" | tr '[:upper:]' '[:lower:]')"
  image="ghcr.io/${owner}/cervorules-tools:${version}"
fi

case "${version}" in
  v*) ;;
  *)
    echo "version must start with v, got ${version}" >&2
    exit 1
    ;;
esac

if [[ "${CERVORULES_OCI_SKIP_PULL:-0}" != "1" ]]; then
  docker pull "${image}"
fi

docker run --rm "${image}" "cervorules-policygen -version" 2>&1 | grep -F "cervorules-policygen ${version}"
docker run --rm "${image}" "cervorules-vocabgen -version" 2>&1 | grep -F "cervorules-vocabgen ${version}"
docker run --rm "${image}" -c "test -s /opt/cervorules/schemas/policy-vocabulary.schema.json"
docker run --rm "${image}" -c "test -s /opt/cervorules/schemas/policy-rules.schema.json"
if [[ "${version}" == v3.* ]]; then
  docker run --rm "${image}" -c "test -s /opt/cervorules/schemas/v3/policy-vocabulary.schema.json"
  docker run --rm "${image}" -c "test -s /opt/cervorules/schemas/v3/policy-rules.schema.json"
fi

cat <<EOF
Verified OCI image ${image}
schemas=/opt/cervorules/schemas
EOF
