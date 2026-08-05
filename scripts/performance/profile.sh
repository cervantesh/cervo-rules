#!/usr/bin/env bash
set -euo pipefail

out_dir="${1:-${CERVORULES_PROFILE_OUT:-dist-performance-profiles}}"
benchtime="${CERVORULES_BENCHTIME:-1s}"
mkdir -p "${out_dir}"

echo "CervoRules performance profiling"
echo "date_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)" | tee "${out_dir}/manifest.txt"
echo "go_version=$(go version)" | tee -a "${out_dir}/manifest.txt"
echo "benchtime=${benchtime}" | tee -a "${out_dir}/manifest.txt"

run_profile() {
	local package="$1"
	local name="$2"
	local bench="$3"
	local cpu_profile="${out_dir}/${name}.cpu.pprof"
	local mem_profile="${out_dir}/${name}.mem.pprof"
	local log_file="${out_dir}/${name}.bench.txt"

	echo "profile=${name} package=${package} bench=${bench}" | tee -a "${out_dir}/manifest.txt"
	go test "${package}" \
		-run '^$' \
		-bench "${bench}" \
		-benchmem \
		-benchtime "${benchtime}" \
		-cpuprofile "${cpu_profile}" \
		-memprofile "${mem_profile}" \
		2>&1 | tee "${log_file}"
}

run_profile ./core core-decision 'BenchmarkDecisionFlowScaleIndexed1000|BenchmarkDecisionFlowScaleIndexed1000FastOptions|BenchmarkDecisionFlowConcurrent'
run_profile ./core generated-policy 'BenchmarkGeneratedPolicy'
run_profile ./httpadapter http-classifier 'BenchmarkHTTPClassifier'
run_profile ./facts facts-reachability 'BenchmarkReachabilityAgenda|BenchmarkReachabilityAgendaTraceDisabled'
run_profile ./facts facts-large-join 'BenchmarkFactsLargeJoinWorkload'
