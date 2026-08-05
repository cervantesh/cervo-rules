#!/usr/bin/env bash
set -euo pipefail

version="${1:?usage: scripts/release/build-artifacts.sh <version> [dist-dir]}"
dist_dir="${2:-dist}"
commit_sha="${GITHUB_SHA:-$(git rev-parse HEAD 2>/dev/null || printf 'unknown')}"
module_path="$(go list -m)"
release_module="${module_path}"
if [[ -n "${SOURCE_DATE_EPOCH:-}" ]]; then
  build_time="$(date -u -d "@${SOURCE_DATE_EPOCH}" +%Y-%m-%dT%H:%M:%SZ)"
else
  build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
fi

rm -rf "${dist_dir}"
mkdir -p "${dist_dir}"

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

create_tar_gz() {
  local base_dir="$1"
  local output="$2"
  local entry="$3"
  local tar_flags=()
  if [[ -n "${SOURCE_DATE_EPOCH:-}" ]]; then
    tar_flags+=(--sort=name --mtime="@${SOURCE_DATE_EPOCH}" --owner=0 --group=0 --numeric-owner)
  fi
  GZIP=-n tar "${tar_flags[@]}" -C "${base_dir}" -czf "${output}" "${entry}"
}

cat >"${dist_dir}/build-metadata.txt" <<EOF
version=${version}
commit=${commit_sha}
build_time=${build_time}
source_date_epoch=${SOURCE_DATE_EPOCH:-}
go_version=$(go version)
go_env_goversion=$(go env GOVERSION)
module=${module_path}
release_module=${release_module}
EOF

go list -m all >"${dist_dir}/dependencies.txt"
cp "${dist_dir}/dependencies.txt" "${dist_dir}/release-dependencies.txt"

{
  printf '{\n'
  printf '  "version": "%s",\n' "$(json_escape "${version}")"
  printf '  "module": "%s",\n' "$(json_escape "${module_path}")"
  printf '  "release_module": "%s",\n' "$(json_escape "${release_module}")"
  printf '  "modules": [\n'
  first=1
  while IFS= read -r module_line; do
    if [[ "${first}" -eq 0 ]]; then
      printf ',\n'
    fi
    first=0
    printf '    "%s"' "$(json_escape "${module_line}")"
  done <"${dist_dir}/dependencies.txt"
  printf '\n  ]\n'
  printf '}\n'
} >"${dist_dir}/sbom-modules.json"

{
  printf '{\n'
  printf '  "spdxVersion": "SPDX-2.3",\n'
  printf '  "dataLicense": "CC0-1.0",\n'
  printf '  "SPDXID": "SPDXRef-DOCUMENT",\n'
  printf '  "name": "cervo-rules-%s",\n' "$(json_escape "${version}")"
  printf '  "documentNamespace": "https://github.com/cervantesh/cervo-rules/releases/%s/sbom-spdx.json",\n' "$(json_escape "${version}")"
  printf '  "creationInfo": {\n'
  printf '    "created": "%s",\n' "$(json_escape "${build_time}")"
  printf '    "creators": ["Tool: scripts/release/build-artifacts.sh"]\n'
  printf '  },\n'
  printf '  "version": "%s",\n' "$(json_escape "${version}")"
  printf '  "module": "%s",\n' "$(json_escape "${module_path}")"
  printf '  "release_module": "%s",\n' "$(json_escape "${release_module}")"
  printf '  "packages": [\n'
  first=1
  while IFS= read -r module_line; do
    module_name="$(printf '%s' "${module_line}" | awk '{print $1}')"
    module_version="$(printf '%s' "${module_line}" | awk '{print $2}')"
    if [[ -z "${module_version}" ]]; then
      module_version="NOASSERTION"
    fi
    package_id="SPDXRef-Package-$(printf '%s' "${module_name}" | sed 's/[^A-Za-z0-9.-]/-/g')"
    if [[ "${first}" -eq 0 ]]; then
      printf ',\n'
    fi
    first=0
    printf '    {"SPDXID": "%s", "name": "%s", "versionInfo": "%s", "downloadLocation": "NOASSERTION", "filesAnalyzed": false, "licenseConcluded": "NOASSERTION", "licenseDeclared": "NOASSERTION", "copyrightText": "NOASSERTION"}' \
      "$(json_escape "${package_id}")" \
      "$(json_escape "${module_name}")" \
      "$(json_escape "${module_version}")"
  done <"${dist_dir}/dependencies.txt"
  printf '\n  ]\n'
  printf '}\n'
} >"${dist_dir}/sbom-spdx.json"

cat >"${dist_dir}/provenance.json" <<EOF
{
  "version": "$(json_escape "${version}")",
  "commit": "$(json_escape "${commit_sha}")",
  "build_time": "$(json_escape "${build_time}")",
  "source_date_epoch": "$(json_escape "${SOURCE_DATE_EPOCH:-}")",
  "module": "$(json_escape "${module_path}")",
  "release_module": "$(json_escape "${release_module}")",
  "go_version": "$(json_escape "$(go version)")",
  "builder": "scripts/release/build-artifacts.sh",
  "workflow": "$(json_escape "${GITHUB_WORKFLOW:-local}")",
  "run_id": "$(json_escape "${GITHUB_RUN_ID:-}")",
  "predicateType": "https://slsa.dev/provenance/v1",
  "buildType": "https://github.com/cervantesh/cervo-rules/build-release-artifacts",
  "materials": [
    {
      "uri": "git+https://github.com/cervantesh/cervo-rules",
      "digest": {
        "sha1": "$(json_escape "${commit_sha}")"
      }
    }
  ],
  "reproducible_when": "SOURCE_DATE_EPOCH is set and the same Go toolchain is used"
}
EOF

platforms=(
  "linux amd64"
  "linux arm64"
  "windows amd64"
)

commands=()
if [[ -d cmd/cervorules-policygen && -d cmd/cervorules-vocabgen ]]; then
  commands=(
    "cervorules-policygen ./cmd/cervorules-policygen"
    "cervorules-vocabgen ./cmd/cervorules-vocabgen"
  )
fi

if [[ "${#commands[@]}" -gt 0 ]]; then
  for platform in "${platforms[@]}"; do
    read -r goos goarch <<<"${platform}"
    platform_dir="${dist_dir}/cervorules-tools-${version}-${goos}-${goarch}"
    mkdir -p "${platform_dir}/schemas"
    cp schemas/*.schema.json "${platform_dir}/schemas/"
    if [[ -d schemas/v3 ]]; then
      mkdir -p "${platform_dir}/schemas/v3"
      cp schemas/v3/*.schema.json "${platform_dir}/schemas/v3/"
    fi
    cp README.md CHANGELOG.md "${platform_dir}/"
    cp "${dist_dir}/build-metadata.txt" "${dist_dir}/dependencies.txt" "${dist_dir}/release-dependencies.txt" "${platform_dir}/"

    for command_spec in "${commands[@]}"; do
      read -r name package_path <<<"${command_spec}"
      output="${platform_dir}/${name}"
      if [[ "${goos}" == "windows" ]]; then
        output="${output}.exe"
      fi
      GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build \
        -trimpath \
        -ldflags "-s -w -X main.version=${version}" \
        -o "${output}" \
        "${package_path}"
    done

    archive_base="${dist_dir}/cervorules-tools-${version}-${goos}-${goarch}"
    create_tar_gz "${dist_dir}" "${archive_base}.tar.gz" "$(basename "${platform_dir}")"
    rm -rf "${platform_dir}"
  done
else
  echo "No cmd tools found; building metadata and schema release artifacts only." >&2
fi

create_tar_gz schemas "${dist_dir}/cervorules-schemas-${version}.tar.gz" .
(cd "${dist_dir}" && sha256sum * > checksums.txt)

minisign_key_file="${CERVORULES_MINISIGN_SECRET_KEY_FILE:-}"
if [[ -n "${minisign_key_file}" ]]; then
  if ! command -v minisign >/dev/null 2>&1; then
    echo "CERVORULES_MINISIGN_SECRET_KEY_FILE was set but minisign was not found" >&2
    exit 1
  fi
  minisign_args=(-S -m "${dist_dir}/checksums.txt" -s "${minisign_key_file}" -x "${dist_dir}/checksums.txt.minisig")
  if [[ "${CERVORULES_MINISIGN_UNENCRYPTED:-0}" == "1" ]]; then
    minisign_args+=(-W)
  fi
  minisign "${minisign_args[@]}"
fi

go_version="$(go version)"
manifest_path="${dist_dir}/artifact-manifest.json"
{
  printf '{\n'
  printf '  "version": "%s",\n' "$(json_escape "${version}")"
  printf '  "commit": "%s",\n' "$(json_escape "${commit_sha}")"
  printf '  "build_time": "%s",\n' "$(json_escape "${build_time}")"
  printf '  "module": "%s",\n' "$(json_escape "${module_path}")"
  printf '  "release_module": "%s",\n' "$(json_escape "${release_module}")"
  printf '  "go_version": "%s",\n' "$(json_escape "${go_version}")"
  printf '  "artifacts": [\n'
  first=1
  while IFS= read -r artifact_path; do
    artifact_name="$(basename "${artifact_path}")"
    artifact_sha="$(sha256sum "${artifact_path}" | awk '{print $1}')"
    artifact_bytes="$(wc -c <"${artifact_path}" | tr -d '[:space:]')"
    if [[ "${first}" -eq 0 ]]; then
      printf ',\n'
    fi
    first=0
    printf '    {"name": "%s", "sha256": "%s", "bytes": %s}' \
      "$(json_escape "${artifact_name}")" \
      "$(json_escape "${artifact_sha}")" \
      "${artifact_bytes}"
  done < <(find "${dist_dir}" -maxdepth 1 -type f ! -name artifact-manifest.json | sort)
  printf '\n'
  printf '  ]\n'
  printf '}\n'
} >"${manifest_path}"
