# v3 Known Gaps

Verified, unfixed defects. Each entry records how it was checked so a later
reader can confirm it still holds instead of trusting this file.

Last verified against `e55d5aa` (v3.0.0-rc.6) on 2026-08-06.

This is a living list of open items. It is not a review record: the rc.1 review
is in [post-rc-review.md](post-rc-review.md), and per-change decisions live in
`docs/change-management/`.

## Checks that pass without checking anything

**The conformance recipe runs a test that does not exist.**
`.cervorules/recipes/run-conformance.json` invokes
`go test -run TestGeneratedRuntimePolicy`, and no test in the repository has
that name. `go test -run` with a pattern matching nothing exits 0, so the step
is green by construction and has always been.

```bash
grep -o '\-run [A-Za-z]*' .cervorules/recipes/run-conformance.json
grep -rn "func TestGeneratedRuntimePolicy" --include="*_test.go" .   # no hits
```

Fix: point it at `TestConsumerConformanceSuiteCertifiesRepoLocalConsumers`, or
delete the step. The same recipe's other step, `-run TestConsumerConformance`,
does match.

**`scripts/ci/repo-hygiene.sh` is invoked by nothing.**
It is the only enforcement of the Operation/Target/Executor naming contract, and
no workflow and no other script calls it.

```bash
grep -rn "repo-hygiene" .github/ scripts/    # only the script itself
```

Fix: add it to `required-build.yml` or to `scripts/ci/quality-gates.sh`.

## Machine-readable output that no machine can read

**`check -format json` writes to stderr.**
`cmd/cervorules-policygen/main.go` encodes the JSON result to stderr, so a
caller redirecting stdout gets nothing. The recipes that consume it
(`validate-policy.json`, `compare-policy.json`) therefore read an empty
document.

```bash
go run ./cmd/cervorules-policygen check \
  -vocab examples/routing-basic/policy-vocabulary.yaml \
  -policy examples/routing-basic/policy-rules.yaml -format json 2>/dev/null | wc -c
# 0
```

Fix: write the JSON payload to stdout and keep diagnostics on stderr. Note the
payload also carries untagged `Source` and `TestSource` fields, which are empty
for `check` but are part of the encoded struct.

**`CodegenSnapshot.Facts` counts the vocabulary while every sibling counts the
policy.** `generator.go:559` sets `Facts: len(vocab.spec.Facts)`, so retuning a
policy's `facts:` bounds, or adding a `when:` predicate, leaves every count in
the snapshot byte-identical. Only `policy_hash` moves. No field counts
predicates at all.

Fix: count the facts the policy references (`policyPlan.referenced`), and
consider adding a predicate count.

## Contracts that contradict the code

**`schemas/v3/generated-policy-metadata.schema.json` describes a document
nothing produces.** It requires a `schema_version` field and `^sha256:`-prefixed
hashes. `runtime.PolicyMetadata` has no `schema_version`, the generator emits
bare hex, and nothing in the repository writes such a document.

```bash
grep -n "schema_version\|sha256:" schemas/v3/generated-policy-metadata.schema.json
grep -rn "generated-policy-metadata" --include="*.go" .    # no producer
```

Fix: either produce the document, or delete the schema. A published contract
for an artifact that does not exist is worse than no contract.

**`.cervorules/agent-manifest.json` violates its own schema.** The manifest
carries keys that `.cervorules/schemas/agent-manifest.schema.json` rejects under
`additionalProperties: false` — `legacy_commands_repository`, `legacy_v1_schemas`
and others. There is also a second, competing schema at
`schemas/v3/agent-manifest.schema.json` with no stated arbiter.

```python
import json
from jsonschema import Draft202012Validator
d = json.load(open('.cervorules/agent-manifest.json'))
s = json.load(open('.cervorules/schemas/agent-manifest.schema.json'))
list(Draft202012Validator(s).iter_errors(d))   # 1 error
```

Fix: extend the schema to the keys the manifest actually uses, and pick which of
the two schemas is authoritative.

## Untested contract surface

**Nothing pins the wire strings of the error codes.** Tests compare the Go
constants to each other, so renaming `ErrorCodeMissingFact`'s value from
`missing_fact` to anything else passes. The strings are documented as stable and
appear in consumers' audit records.

```bash
grep -rn '"missing_fact"\|"invalid_fact"' --include="*_test.go" .   # no hits
```

Fix: one table test asserting the literal string of each exported `ErrorCode`.

## Tracked elsewhere

`docs/v3/public-api-inventory.json` omits several public packages and
`docs/v3/structured-errors.md` documents a minority of the error codes. Both are
pre-existing and belong to the "Final API audit" GA blocker already recorded in
[post-rc-review.md](post-rc-review.md).

## Deliberately out of scope

Not gaps. Recorded so nobody reopens them as oversights:

- **v2 `limits` in the DSL.** Not deferred: decided against. `limits.Budget` is
  `MaxTokens`, `AllowStream`, `AllowTools`, `AllowImages`, `MaxBodyBytes` —
  four of five are LLM gateway vocabulary, and `AGENTS.md` forbids adding "AI,
  provider-specific payload, or tenant concepts to core APIs". `limits/` as an
  optional leaf package is fine, because importing it is the only way to pay for
  it; the DSL is the shared surface and is not. The general case is already
  covered domain-neutrally by declaring a fact and writing a `when:` rule, so
  the consumer names the number and CervoRules never learns what a token is.
- **Generated derived facts.** Per [ADR 0014](../adr/0014-symbolic-guard-layer.md),
  `facts` owns derivation and this layer owns refutation; merging them would put
  guard evaluation on the derivation path, and a decision guard is on the hot
  path while `facts` is a bounded datalog engine with iteration budgets. A
  consumer needing derived facts derives them first and puts the results in
  `Request.Metadata`.

Both are answered in [compound-predicates.md](compound-predicates.md) §8.
