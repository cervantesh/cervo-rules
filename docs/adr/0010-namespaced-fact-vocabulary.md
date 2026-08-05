# ADR 0010: Namespaced Fact Vocabulary

Status: Proposed

## Issue

Tracked by issue #183.

## Context

Facts are intentionally consumer-owned. Without a naming convention, examples,
materialized views, policy diffs, and generated fixtures can drift into
ambiguous predicate names such as `status`, `enabled`, or `user`.

This ADR builds on ADR 0005 and ADR 0009.

## Decision

Predicate names should be stable, lowercase, dot-separated namespaced atoms:

- `auth.user_role`;
- `tenant.plan`;
- `document.classification`;
- `device.capability`;
- `billing.account_tier`.

The namespace prefix identifies the owning domain or adapter. Predicate arity is
part of the compatibility contract. Changing term order, term meaning, enum
values, or arity is a vocabulary migration and should be visible in policy
diffs and release notes.

Reserved prefixes:

- `cervorules.*` is reserved for future library-owned diagnostics;
- `core.*` is discouraged unless a consumer owns that namespace explicitly;
- project names such as `cervoproxy.*` must stay in consumer vocabularies and
  generated policies, not in CervoRules core docs or examples.

## Consequences

- Diffs and golden explanations can compare predicates consistently.
- Multiple projects can share the facts package without leaking vocabulary.
- Adapters have a clear place to document ownership and compatibility.

## Non-Goals

- No registry service.
- No mandatory namespace validator in this ADR.
- No rename of existing `Atom` or `Predicate` terminology.

## Rejected Alternatives

- Unqualified predicate names were rejected for shared examples and future
  tooling because they make diffs and ownership ambiguous.
- A central vocabulary registry was rejected for now because static docs and
  optional linting are enough until multiple consumers need shared governance.

## Verification Expectations

Future implementation work should add:

- optional linting for namespaced predicates;
- tests that neutral examples avoid consumer-specific namespaces;
- policy diff checks that flag arity or term-order changes as compatibility
  risks.
