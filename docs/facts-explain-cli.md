# Facts Explain CLI Design

Status: Proposed design; no facts explain CLI command is implemented by this
document.

Tracked by issue #182.

## Purpose

A future explain CLI should make fact derivations reviewable by humans and
machine-auditable by release tooling. It should use the public facts contracts
instead of inspecting private evaluator state.

## Inputs

The CLI should accept:

- input facts as stable JSON;
- rules from a future declarative rule source or generated fixture;
- optional predicate schema;
- optional constraints;
- optional redaction policy;
- a target fact to explain.

## Outputs

The primary machine output should be the same shape as
`facts.GoldenExplanation`:

- `target`;
- ordered derivation `steps`;
- `summary` with step, input, derived, rule, and stratum counts.

Human output may render `DerivationTrace.ExplainString`, but JSON must remain
the source of truth for tests and package artifacts.

## Guardrails

- The CLI must not add a new facts language without an ADR.
- Redaction must be available before writing explanations to shared logs or
  release artifacts.
- Exit codes should distinguish invalid input, evaluation diagnostics, and
  infrastructure failures.
- Large explain outputs should remain bounded by evaluation options.

## Non-Goals

- No implementation in this document.
- No HTTP server.
- No policy decision explain UI.
- No dependency on CervoProxy vocabulary.

## Rejected Alternatives

- A human-only text output was rejected because release checks and golden tests
  need machine-stable JSON.
- A CLI that reads private trace internals was rejected because it would make
  future trace refactors breaking.

## Verification Expectations

Future implementation should add:

- golden JSON fixtures using `facts.GoldenExplanation`;
- redaction tests;
- invalid-input exit-code tests;
- package smoke tests proving the CLI is included in release artifacts only
  when intentionally added.
