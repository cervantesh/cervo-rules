#!/usr/bin/env bash
set -euo pipefail

mode="${1:-standard}"
benchtime="${CERVORULES_BENCHTIME:-1s}"
matrix_benchtime="${CERVORULES_BENCHTIME:-1x}"
cpu_list="${CERVORULES_BENCH_CPU:-1,4,8,16}"
bench_regex='BenchmarkDecisionFlow|BenchmarkGeneratedPolicy|BenchmarkHTTPClassifier|BenchmarkReachabilityAgenda'
matrix_regex='BenchmarkDecisionFlowScaleIndexedMatrix|BenchmarkGeneratedPolicy|BenchmarkHTTPClassifier|BenchmarkReachabilityMatrix|BenchmarkSelectivePatternPlanning|BenchmarkFactsLargeJoinWorkload'
packages=(./core ./facts ./httpadapter)

echo "CervoRules performance report"
echo "date_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "go_version=$(go version)"
echo "mode=${mode}"
echo "benchtime=${benchtime}"
echo "matrix_benchtime=${matrix_benchtime}"
echo

set -x
case "${mode}" in
	standard)
		echo "bench_regex=${bench_regex}"
		go test -run '^$' -bench "${bench_regex}" -benchmem -cpu "${cpu_list}" "${packages[@]}"
		;;
	matrix)
		echo "bench_regex=${matrix_regex}"
		go test -run '^$' -bench "${matrix_regex}" -benchmem -benchtime "${matrix_benchtime}" -cpu 1,4,8,16,32 "${packages[@]}"
		;;
	*)
		echo "usage: $0 [standard|matrix]" >&2
		exit 2
		;;
esac
