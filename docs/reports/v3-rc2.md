# CervoRules v3.0.0-rc.2 Reports

Issue: #445

Status: current release-candidate report.

Current Overall Maturity: 4.5 / 5

This report consolidates maturity, agnosticism, dependency, smell/hygiene,
performance, release, and machine-usability evidence for `v3.0.0-rc.2`.
Historical v2 reports remain useful context, but this file is the current repo
report for the v3 breaking line.

## Evidence Snapshot

| Evidence | Result |
| --- | --- |
| Required tests | `go test -count=1 ./...` passed before this report batch. |
| Coverage gate | `go test -cover ./...` passed; statement packages remain above the repo 90% target. |
| Static checks | `go vet ./...` passed. |
| Module integrity | `go mod verify` passed in the root module and the `v3` module. |
| v3 module integrity | `cd v3 && go test -count=1 ./...`, `go test -cover ./...`, `go vet ./...`, and `go mod verify` passed before the report batch. |
| Package workflow | `Publish Packages / package-tools` passed for `v3.0.0-rc.2` in CI. |
| Generic package | `scripts/release/verify-generic-package.sh v3.0.0-rc.2 verify-v3-rc2-report` passed. |
| OCI tools image | `scripts/release/verify-oci-tools.sh v3.0.0-rc.2` passed. |
| Performance report | `scripts/performance/report.sh` passed on Windows/amd64 with Go `1.25.8`. |

## Maturity

Score: 4.5 / 5

| Capability | Score | Evidence | Remaining Gap |
| --- | ---: | --- | --- |
| Core API design | 4.6 | v3 uses `Operation`, `Target`, `Executor`, explicit `DecisionResult`, and modular packages. | Real consumer migration feedback before GA. |
| Generated policy model | 4.4 | `runtime.PolicyFactory`, generated metadata, conformance contracts, and migration docs. | Run a real generated consumer migration before GA. |
| Runtime config | 4.4 | v3 config is smaller and uses `DefaultExecutor`, `OperationTargets`, and `ExecutorFallbacks`. | More complex real-world override cases may need v3.1 options. |
| Facts | 4.3 | Bounded engine, optional trace, complexity controls, binding-cost diagnostics, selectivity plan evidence, and performance report evidence. | Still the costliest subsystem; richer explain tooling can improve v3.1. |
| Observability | 4.5 | Versioned report contracts and low-cardinality defaults. | External observability adapter examples can mature after consumer adoption. |
| Release management | 4.6 | Clean `v3.0.0-rc.2` package workflow, generic package verification, OCI verification, SBOM/provenance. | GA requires repeating the same clean release path for `v3.0.0`. |
| Machine usability | 4.7 | `AGENTS.md`, agent manifest, schemas, recipes, conformance, API inventory, and migration report command. | Automated source rewrite remains deferred. |

Assessment: v3 is strong enough for RC adoption and consumer migration trials,
but GA should wait for at least one clean migration exercise and one final
release verification tag.

## Agnosticism

Score: 4.6 / 5

Strong evidence:

- v3 public primitives are domain-neutral: `Operation`, `Target`, `Executor`.
- HTTP remains optional adapter code, not core identity.
- `facts` is optional and transport-neutral.
- Consumer vocabulary stays outside core.
- Neutral conformance fixtures cover billing, document processing, device
  routing, queue events, scheduled jobs, CLI commands, and edge requests.
- PR #444 sanitized current-facing docs so deprecated v2 names no longer appear
  as normal usage in README, AGENTS, quick paths, or copyable examples.

Remaining gaps:

- Some root/v2 docs intentionally preserve historical vocabulary for migration.
- Real non-gateway consumers have not yet validated the v3 names in production.
- Shared adapters beyond HTTP should wait for at least two projects with the
  same shape.

Recommendation: keep v3 core closed to consumer-specific vocabulary. Add new
adapter packages only through ADR and with cross-domain evidence.

## Dependencies

Score: 4.5 / 5

Root module dependency snapshot from `go list -m all`:

| Module | Role |
| --- | --- |
| `github.com/dlclark/regexp2 v1.11.0` | Regex support where needed outside hot core. |
| `github.com/santhosh-tekuri/jsonschema/v6 v6.0.2` | Schema validation for generator/check tooling. |
| `gopkg.in/yaml.v3 v3.0.1` | YAML DSL parsing for tooling. |
| `golang.org/x/mod`, `x/sys`, `x/text`, `x/tools` | Toolchain/build-list support. |
| `gopkg.in/check.v1` | Transitive test/build-list dependency. |

v3 module dependency snapshot:

```text
github.com/cervantesh/cervo-rules/v3
```

Assessment:

- v3 runtime is dependency-light and currently has no external module
  dependencies in `go list -m all`.
- YAML/schema dependencies stay in root tooling and do not define the v3 runtime
  contract.
- `go mod verify` passed in both modules.
- Release packages include dependency manifests and SBOM/provenance.

Recommendation: keep schema/YAML dependencies in generator/check workflows and
avoid adding runtime dependencies to v3 packages.

## Smells And Hygiene

Score: 4.4 / 5

Closed or guarded:

- Current-facing deprecated API drift is guarded by `current_docs_hygiene_test.go`
  from PR #444.
- v3 forbids legacy primitive names in `v3/core`.
- v3 generated factory docs/tests forbid generated `BuildPolicy` wrappers.
- v3 routing docs/tests forbid implicit `RoutingPhase` in v3.
- File-size and dependency-scope tests continue to guard maintainability.
- Release package metadata now includes v3 schemas, module metadata,
  dependencies, SBOM, and provenance.

Accepted residual smells:

- Root/v2 code intentionally retains deprecated aliases for compatibility.
- Some historical reports mention deprecated APIs as migration evidence.
- `internal/policygen` remains a sensitive area because it owns schema parsing
  and generation.

Recommendation: do not remove historical migration references, but keep them out
of current entrypoints. Continue requiring tests before generator refactors.

## Performance

Score: 4.3 / 5

Latest local `scripts/performance/report.sh` evidence:

| Area | Observed Result | Interpretation |
| --- | --- | --- |
| Indexed 1000-route decision | about `2.2 us/op`, `928 B/op`, `9 allocs/op` | Scales well for normal route tables. |
| Indexed 1000-route fast options | about `1.4 us/op`, `592 B/op`, `6 allocs/op` | Recommended for hot paths that do not need trace/observation. |
| Generated policy fast decision | about `2.8 us/op`, `785 B/op`, `18 allocs/op` | Good generated-policy baseline. |
| Generated policy parallel decision | improves to sub-microsecond under higher CPU counts | Compiled policies behave well under parallel load. |
| HTTP classifier precompiled no headers | about `430 ns/op`, `8 B/op`, `1 alloc/op` | Recommended HTTP adapter path. |
| HTTP classifier simple helper | about `6-7 us/op`, about `8 KB/op` | Acceptable helper, not production hot path default. |
| Reachability facts benchmark | about `0.7-1.1 ms/op`, about `1.0 MB/op` | Improved but still the costliest subsystem. |

Recommendation:

- Compile/build policies once at startup.
- Reuse compiled engines and HTTP classifiers.
- Use fast decision options where trace/observation are not needed.
- Keep facts bounded and avoid unbounded per-request facts evaluation.
- Use `ExpensiveRuleBindingThreshold` and `Plan(input)` selectivity reasons to
  catch broad joins before they become production latency problems.
- Keep performance CI advisory until enough runner history exists for stable
  thresholds.

## Release And Supply Chain

Score: 4.6 / 5

`v3.0.0-rc.2` supersedes the `v3.0.0-rc.1` package-workflow gap:

- `v3.0.0-rc.1` had a historical failed OCI publish because the Dockerfile did
  not copy root facade files.
- PR #439 fixed `build/tools/Dockerfile`.
- `v3.0.0-rc.2` published cleanly from current `main`.
- Generic package verification passed with checksums, schemas, metadata,
  dependency manifests, SBOM, provenance, and CLI `-version` checks.
- OCI verification passed and confirmed both CLI versions plus root/v3 schemas.

GA gate:

- repeat this same clean workflow for `v3.0.0`;
- avoid manual artifact repair;
- refresh release notes and wiki reports after the final tag.

## Machine Usability

Score: 4.7 / 5

Strong evidence:

- `AGENTS.md` is the current canonical agent entrypoint.
- `.cervorules/agent-manifest.json` points to `v3.0.0-rc.2` and the v3 module.
- JSON schemas and recipes index machine workflows.
- Native v3 `check -format json`, generated metadata, generated tests, schemas,
  and recipes provide the current machine-readable review surfaces. Native v3
  migration/diff/compat tooling is deferred after the physical split.
- `v3/testkit` includes consumer conformance contracts.

Remaining gap:

- Automated migration rewrite is intentionally deferred; machines can report
  migration work but do not yet patch consumers automatically.

## Postmortem Notes

Incident: `v3.0.0-rc.1` OCI publish failed.

Cause: `build/tools/Dockerfile` did not copy root `.go` facade files, so
`policygen` could not resolve `github.com/cervantesh/cervo-rules/v2`.

Detection: `Publish Packages / package-tools` failed on the
`Publish OCI tools image` step.

Correction:

- PR #439 copied root `.go` files into the Docker build context.
- `v3.0.0-rc.2` proved the corrected package workflow.

Preventive action:

- keep Dockerfile coverage tests;
- treat `v3.0.0-rc.2` as the valid release evidence for v3 RC package health;
- require clean tag workflow before GA.

## Recommendation

Proceed with v3 consumer migration trials from `v3.0.0-rc.2`.

Do not cut `v3.0.0` GA until:

1. a representative consumer migration has been reviewed;
2. release docs and wiki reports cite that migration;
3. the final GA tag repeats the clean package workflow;
4. generic and OCI verification pass without manual repair.
