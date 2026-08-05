#!/usr/bin/env bash
set -euo pipefail

current="${1:-performance-report.txt}"
baseline="${2:-}"
strict="${CERVORULES_BENCH_STRICT:-0}"

echo "CervoRules benchmark history check"
echo "current=${current}"
echo "baseline=${baseline:-none}"
echo "mode=$([[ "${strict}" == "1" ]] && echo strict || echo advisory)"

if [[ ! -f "${current}" ]]; then
	echo "soft regression check skipped: current benchmark report not found" >&2
	[[ "${strict}" == "1" ]] && exit 1 || exit 0
fi

if [[ -z "${baseline}" || ! -f "${baseline}" ]]; then
	echo "soft regression check skipped: no baseline report provided"
	echo "Tracked fields: ns/op, B/op, allocs/op"
	exit 0
fi

current_count="$(grep -E 'Benchmark.*[[:space:]][0-9]+[[:space:]]+[0-9.]+ ns/op' "${current}" | wc -l | tr -d ' ')"
baseline_count="$(grep -E 'Benchmark.*[[:space:]][0-9]+[[:space:]]+[0-9.]+ ns/op' "${baseline}" | wc -l | tr -d ' ')"

echo "current_benchmarks=${current_count}"
echo "baseline_benchmarks=${baseline_count}"
echo "Tracked fields: ns/op, B/op, allocs/op"

if [[ "${current_count}" == "0" ]]; then
	echo "soft regression check warning: current report has no parseable benchmark lines" >&2
	[[ "${strict}" == "1" ]] && exit 1 || exit 0
fi

echo "soft regression check complete: compare reports manually until stable runner history exists"
