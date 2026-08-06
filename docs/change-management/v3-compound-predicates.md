# v3 Compound Predicates — Change Management Record

Release: `v3.0.0-rc.5`. Date: 2026-08-05.
Decision record: [ADR 0016](../adr/0016-compound-predicates-over-typed-facts.md).
Design and verification: [docs/v3/compound-predicates.md](../v3/compound-predicates.md).

This record answers `docs/api-audit.md` for a change that touches public Go API,
generated API, both v3 schemas, and the CLI contract.

## Surfaces Changed

| Surface | Change | Compatibility |
| --- | --- | --- |
| Public Go API | `core.ErrorCodeMissingFact`, `core.ErrorCodeInvalidFact` | Additive constants on an open `ErrorCode` type. |
| Public Go API | `testkit.RuntimeCase.WantErrorCode`, `.WantTraceStepNames` | Additive struct fields; zero values preserve previous behaviour. |
| Generated API | `denies` becomes an ordered slice; `generatedRoute` gains `id`; predicate matchers and the fact frame are new | All unexported inside the generated file. `PolicyFactory`, `Metadata`, `DefaultConfig`, `ValidateConfig` and `Build` are unchanged. |
| Generated API | `DecisionTrace.Steps` is populated | The envelope already existed and was always empty. Trace remains opt-in per decision. |
| Generated API | `vocabgen` emits `Fact*` constants | Additive; only for vocabularies declaring `facts:`. |
| Vocabulary schema | `facts:` block | Additive optional property. |
| Rules schema | `facts:`, `when:`, `deny.operation` optional, `deny.id` required, predicate arity constraints | Additive except `deny.id`; see Breaking below. |
| CLI | `check` text output gains `conditions=` and `facts=`; the JSON snapshot gains `facts` | Additive fields. |

## Breaking Review

Three rejections of input previously accepted. `docs/dsl-versioning.md:30`
requires a case-by-case call when validation becomes stricter; the exception at
`:44` covers warnings becoming errors and does not apply, so the call is
recorded here.

| Rejection | Previously | Call |
| --- | --- | --- |
| Deny operation not in the vocabulary | Generated, compiled, and never fired | Correctness fix. The accepted input produced a silent fail-open. |
| Two routes on one operation | Generated Go with duplicate constant map keys; only the consumer's build failed | Correctness fix. The accepted input produced a policy that could not be built. |
| `deny` without `id` | Denied every operation with an empty reason and an unnamed trace step | Correctness fix. The accepted input produced a decision that could not be audited. |

No policy that was already correct is affected: every deny in this repository
and in the reference consumer already carried an `id`, and no shipped policy
declared a duplicate route operation or an unknown deny operation.

`deny.reason` now defaults to `id` when absent. This changes generated output
for a policy that omitted `reason`, at an unchanged `PolicyHash` — the policy
file is not edited. It replaces an empty audit reason with the rule name.

## Cross-Domain Review (ADR 0001)

- **Does the change assume CervoProxy vocabulary, transport, provider, or
  payload semantics?** No. A fact is a caller-declared name with a scalar type;
  the vocabulary ships no fact names of its own. `Request.Metadata` is unchanged.
- **Can the same API serve at least two neutral examples?**
  Yes: `examples/predicate-composition` (queue depth, error budget, change
  freeze) and the reference consumer's trading policy (risk, exposure, score).
  Neither shares a fact name with the other.
- **Is the change adapter-specific?** No. Nothing in the predicate grammar names
  a transport, a provider, or a domain.
- **Does ADR 0001 require a new ADR?** Yes, and
  [ADR 0016](../adr/0016-compound-predicates-over-typed-facts.md) records it.
  ADR 0014 had explicitly deferred this decision.

## Dependency Review

None added. Predicates compile to Go boolean expressions, so there is no
evaluator to depend on. JSON Schema validation of shipped documents runs in CI
against an external implementation (`scripts/ci/validate-schemas.py`) rather
than importing one, which keeps `go.mod` unchanged — `docs/dependencies.md`
needs no edit.

## Verification

- `go test ./...`, `go vet ./...`, `go mod verify` across 16 packages.
- Generated policies are compiled and executed in a temporary consumer module,
  including every example's declarative `tests:` block.
- Every shipped policy document validates against the v3 JSON Schemas; the
  validator was confirmed to fail on a deliberately malformed leaf.
- Schema and generator agree in both directions on the nine shapes previously
  found to diverge.
- The reference consumer (`gold-executor`) expresses all seven of its fact rules
  in YAML, including two disjunctions, with its gate reduced to input
  sanitisation. 48 assertions pass, none fail.
