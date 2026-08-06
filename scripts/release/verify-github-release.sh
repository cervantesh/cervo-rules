#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'EOF'
usage: scripts/release/verify-github-release.sh <version> [work-dir]

Downloads a published GitHub Release and verifies it the way a consumer would:
checksums, the release-module marker in every machine-readable artifact, the
schema bundle, and the CLI -version of the tool archive for this platform.

Replaces verify-generic-package.sh, which read a generic package API that
github.com does not have, so it could never verify anything published here.

Environment:
  GH_REPO                 Repository to read, as owner/name. Defaults to
                          GITHUB_REPOSITORY, then to the git remote.
  GH_TOKEN                Optional. Only needed for a private repository; gh
                          uses its own login otherwise.
  CERVORULES_PACKAGE_OS   Tool archive OS. Defaults from uname.
  CERVORULES_PACKAGE_ARCH Tool archive architecture. Defaults from uname.
  CERVORULES_VERIFY_SIGNATURES
                          Set to 1 to require and verify checksums.txt.minisig.
  CERVORULES_MINISIGN_PUBLIC_KEY
                          Minisign public key, required when signature
                          verification is enabled.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 2
fi

version="$1"
work_dir="${2:-$(mktemp -d)}"

case "${version}" in
  v*) ;;
  *)
    echo "version must start with v, got ${version}" >&2
    exit 1
    ;;
esac

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required to download a GitHub Release" >&2
  exit 1
fi

repo="${GH_REPO:-${GITHUB_REPOSITORY:-}}"
if [[ -z "${repo}" ]]; then
  origin="$(git remote get-url origin 2>/dev/null || true)"
  origin="${origin%.git}"
  repo="${origin#*github.com[:/]}"
fi
if [[ -z "${repo}" ]]; then
  echo "cannot determine the repository; set GH_REPO to owner/name" >&2
  exit 1
fi

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

if [[ "${CERVORULES_VERIFY_SIGNATURES:-0}" == "1" ]]; then
  if [[ -z "${CERVORULES_MINISIGN_PUBLIC_KEY:-}" ]]; then
    echo "CERVORULES_MINISIGN_PUBLIC_KEY is required when CERVORULES_VERIFY_SIGNATURES=1" >&2
    exit 1
  fi
  if ! command -v minisign >/dev/null 2>&1; then
    echo "minisign is required when CERVORULES_VERIFY_SIGNATURES=1" >&2
    exit 1
  fi
fi

mkdir -p "${work_dir}"
echo "Downloading release ${version} from ${repo}"
gh release download "${version}" --repo "${repo}" --dir "${work_dir}" --clobber

required=(
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
for file in "${required[@]}"; do
  if [[ ! -s "${work_dir}/${file}" ]]; then
    echo "release ${version} is missing ${file}" >&2
    exit 1
  fi
done

if [[ "${CERVORULES_VERIFY_SIGNATURES:-0}" == "1" ]]; then
  if [[ ! -s "${work_dir}/checksums.txt.minisig" ]]; then
    echo "release ${version} has no checksums.txt.minisig" >&2
    exit 1
  fi
  minisign -V -m "${work_dir}/checksums.txt" -x "${work_dir}/checksums.txt.minisig" -P "${CERVORULES_MINISIGN_PUBLIC_KEY}"
fi

(
  cd "${work_dir}"
  sha256sum -c checksums.txt --ignore-missing
)

has_tool_archive=0
if [[ -s "${work_dir}/${tool_archive}" ]]; then
  has_tool_archive=1
else
  echo "no tool archive for ${package_os}/${package_arch}; verifying metadata and schema bundle"
fi

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

  policy_version="$("${tool_dir}/cervorules-policygen${exe_suffix}" -version 2>&1)"
  vocab_version="$("${tool_dir}/cervorules-vocabgen${exe_suffix}" -version 2>&1)"

  if [[ "${policy_version}" != "cervorules-policygen ${version}" ]]; then
    echo "unexpected policygen version: ${policy_version}" >&2
    exit 1
  fi
  if [[ "${vocab_version}" != "cervorules-vocabgen ${version}" ]]; then
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
Verified GitHub Release ${version}
  repository: ${repo}
  module:     ${expected_module}
  work dir:   ${work_dir}
  tool archive verified: $([[ "${has_tool_archive}" == "1" ]] && echo "yes (${package_os}/${package_arch})" || echo "no")
EOF
