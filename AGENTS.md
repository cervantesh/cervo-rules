# CervoRules Agent Guide

Current release candidate: `v3.0.0-rc.7`.

Primary module: `github.com/cervantesh/cervo-rules/v3`.

CervoRules is a deterministic policy decision library. Given a request, it
answers whether to allow it, where to route it, and why. It executes nothing.

It is not a proxy, controller, scheduler, model selector, data plane, logger,
metrics backend, or transport parser.

## Start Here

1. Read this file first.
2. Read `README.md` for the human overview.
3. Read `docs/agent-quickstart.md` for copyable machine workflows.
4. Read `.cervorules/agent-manifest.json` for machine-readable entrypoints,
   packages, commands, checks, and boundaries.
5. Read `docs/v3/api-reference.md` before changing v3 contracts.
6. Read `docs/v3/known-gaps.md` before reporting a defect, so you do not
   re-report a verified open one.
7. Read `docs/v3/migration-v2-to-v3.md` only when touching legacy migration work.

## What A Policy Can Express Today

A policy is YAML compiled to Go by `cervorules-policygen`. The full surface:

| Construct | Meaning |
| --- | --- |
| `routes` | Operation to target/executor, with optional fallbacks and disabled-by-default |
| `denies` | Ordered, first match wins. `operation` optional (absent means every operation); `id` required |
| `conditions` + `requires` | Named guards the consumer answers in Go, conjoined |
| vocabulary `facts` | A fact's name and type: `number`, `integer`, `bool`, `enum`, `string` |
| policy `facts` | That fact's `min`, `max`, `default` |
| `when` | A predicate: `all` / `any` / `not` over leaves comparing a fact to a literal, to another fact of the same type, or consulting a condition |
| `tests` | Declarative cases compiled into a generated test |

See `docs/v3/policygen-dsl.md` for the authoring reference,
`docs/v3/compound-predicates.md` for the design record, and
[ADR 0016](docs/adr/0016-compound-predicates-over-typed-facts.md) for why the
predicate grammar is shaped the way it is.

## Invariants You Must Not Break

These are load-bearing. Each one has been violated at least once and cost real
debugging.

**Fail closed, with no exceptions.** A fact that is absent with no declared
default, unparseable, non-finite, or outside its declared bounds returns a
structured `missing_fact` / `invalid_fact` error. It never returns `false`. A
`false` reads as "the guard ran and found nothing wrong", which is the silent
allow this library exists to prevent. `strconv.ParseFloat` accepts `"NaN"`, and
a non-finite value passes every comparison without matching any.

**No author-controlled text reaches generated Go unquoted.** A predicate
description embeds the policy's own string literals. It once went into a Go
comment unescaped, so a literal containing a newline injected top-level
declarations into the generated engine while `check`, `generate` and gofmt all
passed. Anything new that emits author text goes through `%q` or `commentSafe`.

**`check` is a function of its input.** Do not iterate a Go map in any path that
can fail or emit. Two policy keys that normalize alike once made the same file
pass on some runs and fail on others.

**Every decision-bearing number lives in the policy file.** The `PolicyHash` is
sha256 over that file's bytes, so a threshold there is auditable by construction.
`DefaultConfig` carries no security value.

**No dependency for evaluating predicates.** They compile to Go boolean
expressions. The grammar is a closed tagged form, deliberately too small to
express anything but a guard. An expression language inside a security policy is
attack surface, not convenience.

## Primary V3 Surface

Use these names in new work: `Operation`, `Target`, `Executor`, `PolicyFactory`.

Generated v3 policy packages expose `runtime.PolicyFactory` and no older
construction wrapper.

## Package Routing

| Need | Package |
| --- | --- |
| Build or evaluate policy decisions | `core` |
| Use generated policy factories and runtime config | `runtime` |
| Clamp a request payload against a budget, after a route is chosen | `limits` |
| Derive bounded facts before deciding | `facts` |
| Classify HTTP inputs at the adapter boundary | `httpadapter` |
| Produce reports and audit envelopes | `observe` |
| Refute an incoherent world with ontology constraints | `ontology` |
| Certify generated consumers | `testkit` |

Prefer modular imports. The v3 root package is marker-only and is not a
compatibility facade:

```go
import (
	"github.com/cervantesh/cervo-rules/v3/core"
	"github.com/cervantesh/cervo-rules/v3/runtime"
)
```

## Required Checks Before PR

```text
go test -count=1 ./...
go test -cover ./...
go vet ./...
go mod verify
python scripts/ci/validate-schemas.py
```

`scripts/ci/quality-gates.sh standard` runs all of them; `extended` adds race
and benchmark passes.

The schema check needs `pyyaml` and `jsonschema`, and skips with a message when
they are absent. CI sets `CERVORULES_REQUIRE_SCHEMA_VALIDATION=1` so a skip is a
failure there. It exists because nothing else in the repository reads the
published JSON Schemas, so schema/parser divergence is otherwise invisible.

For release, facts, generator, or performance changes also run the relevant
release, race, fuzz, benchmark, or package checks in `docs/release.md`.

Note that `govulncheck` reports the standard library of the toolchain running
it. A local run on a newer Go than CI installs will not see what CI sees.

After SonarQube is configured, run `scripts/sonar/scan.sh`; see
`docs/operations/sonarqube.md`.

## Policy And Vocabulary Workflow

Use `cervorules-vocabgen` for vocabulary constants, including a `Fact*` constant
per declared fact so a misspelled metadata key is a compile error.

Use `cervorules-policygen check` before changing generated code, and
`cervorules-policygen generate` to produce the policy package and its tests.

Generated consumers keep `testkit` contract tests beside generated policy
packages. `testkit.RuntimeCase` can assert a refusal (`WantErrorCode`) and the
explanation (`WantTraceStepNames`), not only a successful decision.

Use v2 maintenance tooling from `cervantesh/cervo-rules-v2` only for legacy
`migrate-v3`, `compat`, `diff`, or `inspect-api` workflows.

## Releasing

Artifacts are GitHub Release assets; the optional tools image goes to
`ghcr.io/<owner>`. Nothing needs a secret to be created — `GITHUB_TOKEN` carries
`contents: write` and `packages: write`.

Pushing a `v*` tag runs `Publish Packages`, which builds, publishes, and then
verifies what it published with `scripts/release/verify-github-release.sh`.
Re-running a version replaces its assets rather than failing, so a partial
upload is repaired by re-running. Full checklist in `docs/release.md`.

## Machine-Readable Artifacts

Current schemas are `schemas/v3/*.schema.json`. The pair at `schemas/` without
the `v3/` prefix describes the legacy `cervorules.policy.v1` DSL and is kept for
migration only.

Agent tooling: `.cervorules/schemas/*.json` and `.cervorules/recipes/*.json`.
Both are checked by CI. The schemas are validated against the documents they
cover, and no schema may be published without one. A recipe is checked for what
it claims, not only for its shape: every `-run` pattern must select a real
test, every script and CLI subcommand must exist, every flag must be declared,
and a pinned version must match `current_version` in the manifest. Follow a
recipe as written — if it names something that does not exist, that is a bug in
this repository, not in your reading of it.

## Agnosticism Rules

- Keep domain vocabulary in the consuming project. This one is enforced:
  `TestLibrarySourceHardcodesNoDomainVocabulary` fails the build if a name
  declared in any example vocabulary appears as a string literal in
  hand-written library source. Generated code is exempt, because carrying the
  vocabulary is its job.
- Keep transport parsing outside `core`.
- Keep HTTP-specific code in `httpadapter`.
- Keep model/profile selection outside CervoRules.
- Keep logs, metrics, tracing, and audit storage outside CervoRules.
- Add shared adapters only after at least two domains need the same shape.
- Do not add CervoProxy, gateway, AI, provider-specific payload, or tenant
  concepts to core APIs.

The test for the DSL specifically: an optional leaf package is paid for only by
whoever imports it, but the DSL is shared, so one consumer's vocabulary in the
DSL makes every policy author carry it. That is why `limits`, whose fields are
`MaxTokens` / `AllowTools` / `AllowImages`, stays a Go package and does not
become DSL surface. A consumer that wants that guard declares a fact and writes
a `when:` rule, naming the number itself.

## No-Touch Boundaries Without ADR

Do not change public Go APIs without ADR and change management.
Do not change the module path `github.com/cervantesh/cervo-rules/v3`.
Do not add new v2 compatibility shims, root-facade aliases, or generated
compatibility wrappers; see `docs/change-management/vnext-no-v2-compatibility.md`.
Do not add runtime dependencies for logging, metrics, tracing, package
publishing, schema validation, or YAML parsing.
Do not weaken release, compatibility, freshness, conformance, or drift checks.

## Where This Is Headed

Still a release candidate, not GA. What stands between here and `v3.0.0`:

- **The final API audit.** `docs/v3/public-api-inventory.json` omits public
  packages and `docs/v3/structured-errors.md` documents a minority of the error
  codes. Recorded as a GA blocker in `docs/v3/post-rc-review.md`.
- **The open gaps in `docs/v3/known-gaps.md`.** Small, but three of them are
  checks that pass without checking anything.
- **Policy `explain` beyond the trace.** `DecisionTrace.Steps` now names the
  rule and the deciding leaf. Whether a consumer-facing explain API belongs in
  `core` is undecided.

Deliberately not coming back, so do not reopen them as oversights:

- v2 `limits` as DSL surface, for the agnosticism reason above.
- Generated derived facts. Per [ADR 0014](docs/adr/0014-symbolic-guard-layer.md),
  `facts` owns derivation and this layer owns refutation; merging them puts guard
  evaluation on the derivation path. A consumer needing derived facts derives
  them first and puts the results in `Request.Metadata`.

## Deprecated API Hygiene

Current-facing docs and examples must use the v3 names and factory shape.
Deprecated or removed APIs belong only in explicit migration, history, or
deprecation-policy docs. Do not add them to README, AGENTS, quickstarts, or
copyable examples.

## TDD And Change Management

Use TDD for behavior changes:

1. Add the failing test.
2. Run it and confirm the expected failure.
3. Implement the minimal change.
4. Run the targeted test.
5. Run the required checks.

Use issues for change management. Each issue should record estimate, start time,
actual time, deviation, linked PR, tests run, and maturity impact.
Documentation-only work must still update the issue with touched repository
docs, touched wiki pages, drift tests, and any historical snapshot wording.
