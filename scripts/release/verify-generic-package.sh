#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
usage: scripts/release/verify-generic-package.sh <version> [work-dir]

Verifies the generic package for a CervoRules release or smoke version.

Environment:
  PACKAGE_BASE_URL  Required. Generic package base URL, for example
                          https://<registry-host>/api/packages/<owner>/generic/cervo-rules
  PACKAGE_REGISTRY_USER              Optional package read user.
  PACKAGE_REGISTRY_TOKEN             Optional package read token.
  CERVORULES_PACKAGE_OS   Tool archive OS. Defaults from uname.
  CERVORULES_PACKAGE_ARCH Tool archive architecture. Defaults from uname.
  CERVORULES_VERIFY_SIGNATURES
                          Set to 1 to require and verify checksums.txt.minisig.
  CERVORULES_MINISIGN_PUBLIC_KEY
                          Minisign public key used when signature verification is enabled.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || $# -lt 1 ]]; then
  usage
  exit 2
fi

version="$1"
work_dir="${2:-$(mktemp -d)}"
if [[ -z "${PACKAGE_BASE_URL:-}" ]]; then
  echo "PACKAGE_BASE_URL is required; see --help" >&2
  exit 1
fi
base_url="${PACKAGE_BASE_URL}"
package_url="${base_url%/}/${version}"

expected_module="github.com/cervantesh/cervo-rules"
case "${version}" in
  v2.*) expected_module="github.com/cervantesh/cervo-rules/v2" ;;
  v3.*) expected_module="github.com/cervantesh/cervo-rules/v3" ;;
esac

detect_os() {
  case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
    mingw*|msys*|cygwin*) printf 'windows' ;;
    darwin*) printf 'darwin' ;;
    linux*) printf 'linux' ;;
    *) printf 'linux' ;;
  esac
}

detect_arch() {
  case "$(uname -m | tr '[:upper:]' '[:lower:]')" in
    x86_64|amd64) printf 'amd64' ;;
    aarch64|arm64) printf 'arm64' ;;
    *) printf 'amd64' ;;
  esac
}

package_os="${CERVORULES_PACKAGE_OS:-$(detect_os)}"
package_arch="${CERVORULES_PACKAGE_ARCH:-$(detect_arch)}"
tool_archive="cervorules-tools-${version}-${package_os}-${package_arch}.tar.gz"
schema_archive="cervorules-schemas-${version}.tar.gz"

case "${version}" in
  v*) ;;
  *)
    echo "version must start with v, got ${version}" >&2
    exit 1
    ;;
esac

mkdir -p "${work_dir}"

curl_args=(--fail --show-error --silent --location)
if [[ -n "${PACKAGE_REGISTRY_USER:-}" || -n "${PACKAGE_REGISTRY_TOKEN:-}" ]]; then
  if [[ -z "${PACKAGE_REGISTRY_USER:-}" || -z "${PACKAGE_REGISTRY_TOKEN:-}" ]]; then
    echo "PACKAGE_REGISTRY_USER and PACKAGE_REGISTRY_TOKEN must be set together" >&2
    exit 1
  fi
  curl_args+=(--user "${PACKAGE_REGISTRY_USER}:${PACKAGE_REGISTRY_TOKEN}")
fi

files=(
  "checksums.txt"
  "artifact-manifest.json"
  "build-metadata.txt"
  "dependencies.txt"
  "release-dependencies.txt"
  "sbom-modules.json"
  "sbom-spdx.json"
  "provenance.json"
  "${schema_archive}"
)

if [[ "${CERVORULES_VERIFY_SIGNATURES:-0}" == "1" ]]; then
  if [[ -z "${CERVORULES_MINISIGN_PUBLIC_KEY:-}" ]]; then
    echo "CERVORULES_MINISIGN_PUBLIC_KEY is required when CERVORULES_VERIFY_SIGNATURES=1" >&2
    exit 1
  fi
  if ! command -v minisign >/dev/null 2>&1; then
    echo "minisign is required when CERVORULES_VERIFY_SIGNATURES=1" >&2
    exit 1
  fi
  files+=("checksums.txt.minisig")
fi

for file in "${files[@]}"; do
  curl "${curl_args[@]}" "${package_url}/${file}" --output "${work_dir}/${file}"
done
if curl "${curl_args[@]}" "${package_url}/${tool_archive}" --output "${work_dir}/${tool_archive}"; then
  has_tool_archive=1
else
  has_tool_archive=0
  rm -f "${work_dir}/${tool_archive}"
  echo "tool archive not present; verifying metadata and schema package"
fi

if [[ "${CERVORULES_VERIFY_SIGNATURES:-0}" == "1" ]]; then
  minisign -V -m "${work_dir}/checksums.txt" -x "${work_dir}/checksums.txt.minisig" -P "${CERVORULES_MINISIGN_PUBLIC_KEY}"
fi

(
  cd "${work_dir}"
  sha256sum -c checksums.txt --ignore-missing
)

extract_dir="${work_dir}/extract"
rm -rf "${extract_dir}"
mkdir -p "${extract_dir}"
schema_dir="${extract_dir}/schemas"
mkdir -p "${schema_dir}"
tar -xzf "${work_dir}/${schema_archive}" -C "${schema_dir}"

if [[ "${has_tool_archive}" == "1" ]]; then
  tar -xzf "${work_dir}/${tool_archive}" -C "${extract_dir}"
  tool_dir="${extract_dir}/cervorules-tools-${version}-${package_os}-${package_arch}"
  exe_suffix=""
  if [[ "${package_os}" == "windows" ]]; then
    exe_suffix=".exe"
  fi

  policy_version="$("${tool_dir}/cervorules-policygen${exe_suffix}" -version)"
  vocab_version="$("${tool_dir}/cervorules-vocabgen${exe_suffix}" -version)"

  expected_policy="cervorules-policygen ${version}"
  expected_vocab="cervorules-vocabgen ${version}"

  if [[ "${policy_version}" != "${expected_policy}" ]]; then
    echo "unexpected policygen version: ${policy_version}" >&2
    exit 1
  fi
  if [[ "${vocab_version}" != "${expected_vocab}" ]]; then
    echo "unexpected vocabgen version: ${vocab_version}" >&2
    exit 1
  fi

  test -s "${tool_dir}/schemas/policy-rules.schema.json"
  test -s "${tool_dir}/schemas/policy-vocabulary.schema.json"
  if [[ "${version}" == v3.* ]]; then
    test -s "${tool_dir}/schemas/v3/policy-rules.schema.json"
    test -s "${tool_dir}/schemas/v3/policy-vocabulary.schema.json"
  fi
  test -s "${tool_dir}/build-metadata.txt"
  test -s "${tool_dir}/dependencies.txt"
  test -s "${tool_dir}/release-dependencies.txt"
fi
test -s "${schema_dir}/policy-rules.schema.json"
test -s "${schema_dir}/policy-vocabulary.schema.json"
if [[ "${version}" == v3.* ]]; then
  test -s "${schema_dir}/v3/policy-rules.schema.json"
  test -s "${schema_dir}/v3/policy-vocabulary.schema.json"
fi
test -s "${work_dir}/artifact-manifest.json"
grep -qx "version=${version}" "${work_dir}/build-metadata.txt"
grep -qx "release_module=${expected_module}" "${work_dir}/build-metadata.txt"
grep -qx "${expected_module}" "${work_dir}/release-dependencies.txt"
grep -q "\"version\": \"${version}\"" "${work_dir}/artifact-manifest.json"
grep -q "\"release_module\": \"${expected_module}\"" "${work_dir}/artifact-manifest.json"
if [[ "${has_tool_archive}" == "1" ]]; then
  grep -q "\"name\": \"${tool_archive}\"" "${work_dir}/artifact-manifest.json"
fi
grep -q "\"name\": \"${schema_archive}\"" "${work_dir}/artifact-manifest.json"
grep -q "\"version\": \"${version}\"" "${work_dir}/sbom-modules.json"
grep -q "\"release_module\": \"${expected_module}\"" "${work_dir}/sbom-modules.json"
grep -q "${expected_module}" "${work_dir}/release-dependencies.txt"
grep -q '"SPDXID": "SPDXRef-DOCUMENT"' "${work_dir}/sbom-spdx.json"
grep -q "\"version\": \"${version}\"" "${work_dir}/sbom-spdx.json"
grep -q "\"release_module\": \"${expected_module}\"" "${work_dir}/sbom-spdx.json"
grep -q "\"version\": \"${version}\"" "${work_dir}/provenance.json"
grep -q "\"release_module\": \"${expected_module}\"" "${work_dir}/provenance.json"
grep -q '"commit":' "${work_dir}/provenance.json"
grep -q '"go_version":' "${work_dir}/provenance.json"
grep -q "\"builder\": \"scripts/release/build-artifacts.sh\"" "${work_dir}/provenance.json"
grep -q "\"predicateType\": \"https://slsa.dev/provenance/v1\"" "${work_dir}/provenance.json"
grep -q '"materials":' "${work_dir}/provenance.json"

cat <<EOF
Verified generic package ${version}
work_dir=${work_dir}
tool_archive_present=${has_tool_archive}
module=${expected_module}
artifact_manifest=artifact-manifest.json
sbom=sbom-modules.json
sbom_spdx=sbom-spdx.json
provenance=provenance.json
policygen=${policy_version}
vocabgen=${vocab_version}
EOF
