# Policy Diff Design

Status: historical v2 design plus future v3 target. The physical split moved
the implemented v2 diff/compat tooling to `cervantesh/cervo-rules-v2`. Native v3
currently supports `check` and `generate`; v3 diff/compat/inspect commands need
new design before they are claimed in active workflows.

Tracked by issue #180.

## Purpose

Policy diffs should help reviewers understand behavior changes between
versions of facts, vocabularies, rules, generated policies, and runtime config.
This document remains the design target for broader cross-artifact diffs; the
historical v2.1.0-rc.1 CLI contract covered policy semantic drift and
policy-declared API inspection.

## Inputs

The historical v2 diff/compatibility commands compare:

- policy YAML;
- route removals, route disabling, and added denies;
- top-level and inline limit tightening for streaming, tools, images, tokens,
  and body bytes;
- derived fact predicate removals/additions where declared in policy
  `derived_facts`.

The historical v2 `inspect-api` command reports stable JSON for:

- policy name and DSL version;
- routes, denies, limits, and derived fact predicates/rules/aggregates.

Future diff inputs may compare:

- vocabulary YAML;
- generated policy metadata and generated source signatures;
- namespaced fact schemas;
- materialized fact view manifests;
- golden explanation JSON;
- release package metadata.

## Output Model

Diff output should separate:

- textual changes: fields or files changed;
- compatibility changes: predicate arity, term order, capability/provider
  vocabulary, or runtime config contract changed;
- semantic changes: facts newly derivable, no longer derivable, or derived by a
  different rule;
- operational changes: limits, timeouts, fallbacks, package metadata, or
  dependency changes.

Outputs must be deterministic and redaction-aware.

## Relationship To ADRs

- ADR 0009 defines absence and `not derivable`;
- ADR 0010 gives predicates stable names;
- ADR 0011 and ADR 0012 define future optimization boundaries that must not
  change semantic diff results.

## Coverage Limits

- Historical v2 compatibility reports included stable warnings for vocabulary
  and generated API drift because the compare command accepted policy files only.
- Derived facts are compared by declared predicates, rule heads, and aggregate
  outputs. The report warns that full semantic equivalence is not proven.
- Historical v2 `inspect-api` is policy-only; a native v3 replacement should
  decide whether generated artifacts are part of the report.

## Non-Goals

- No visual UI requirement.
- No assertion that textual diffs are sufficient for behavior review.

## Rejected Alternatives

- Text-only diffs were rejected as the only review mechanism because generated
  policy behavior can change without obvious line-level meaning.
- Redaction-optional diff output was rejected because diffs may contain fact
  terms, request examples, or derived explanations.

## Verification Expectations

Future implementation should add:

- golden diff fixtures;
- redacted output tests;
- compatibility-risk classification tests;
- fixtures that compare full evaluation against partial or incremental
  evaluation outputs.

Docs-only verification for this issue should confirm this file links #180,
ADR 0009, ADR 0010, ADR 0011, and ADR 0012, and does not claim a diff command
exists.
