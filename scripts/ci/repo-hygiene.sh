#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

root_go_files="$(find . -maxdepth 1 -type f -name '*.go' | wc -l | tr -d ' ')"
root_files="$(find . -maxdepth 1 -type f | wc -l | tr -d ' ')"

echo "==> root budget"
echo "root files: ${root_files}"
echo "root Go files: ${root_go_files}"

if [[ "${root_go_files}" -gt 1 ]]; then
  echo "root Go files exceed v3 marker-only hygiene budget: ${root_go_files} > 1" >&2
  exit 1
fi

echo "==> v3 root contract"
if [[ "$(go list -m)" != "github.com/cervantesh/cervo-rules/v3" ]]; then
  echo "root module must be github.com/cervantesh/cervo-rules/v3" >&2
  exit 1
fi
if [[ -d v3 ]]; then
  echo "nested v3 module must not exist after physical split" >&2
  exit 1
fi
if grep -R --include='*.go' -n 'github.com/cervantesh/cervo-rules/v2' . >/tmp/cervorules-v2-go-refs.txt; then
  cat /tmp/cervorules-v2-go-refs.txt >&2
  echo "v3 root Go code must not import v2" >&2
  exit 1
fi
if [[ -d cmd || -d internal ]]; then
  echo "==> v3 native tooling contract"
  if grep -R --include='*.go' -n 'github.com/cervantesh/cervo-rules/v2' cmd internal >/tmp/cervorules-v2-tooling-refs.txt; then
    cat /tmp/cervorules-v2-tooling-refs.txt >&2
    echo "v3 native tooling must not import v2" >&2
    exit 1
  fi
  if find cmd internal -type f -name '*.go' ! -name '*_test.go' -print0 | xargs -0 grep -nE '\b(Capability|Service|Provider)\b' >/tmp/cervorules-v2-name-refs.txt; then
    cat /tmp/cervorules-v2-name-refs.txt >&2
    echo "v3 native tooling must use Operation, Target, and Executor names" >&2
    exit 1
  fi
fi

echo "repo hygiene check completed"
