#!/usr/bin/env bash
set -euo pipefail

go_version="${GO_VERSION:-1.25.11}"
tool_cache="${RUNNER_TOOL_CACHE:-/var/lib/ci-runner/work/tool_cache}"
go_root="${tool_cache}/go/${go_version}/x64"
go_bin="${go_root}/bin"

if [[ ! -x "${go_bin}/go" ]]; then
  tmp_dir="$(mktemp -d)"
  archive="${tmp_dir}/go.tar.gz"
  url="https://go.dev/dl/go${go_version}.linux-amd64.tar.gz"

  mkdir -p "${go_root}"
  curl --fail --location --silent --show-error "${url}" --output "${archive}"
  tar xzf "${archive}" --strip-components=1 -C "${go_root}"
  rm -rf "${tmp_dir}"
fi

"${go_bin}/go" version
echo "${go_bin}" >> "${GITHUB_PATH}"
