# ADR 0003: v2 Naming And Action Metadata Boundaries

Date: 2026-05-21

## Status

Accepted, amended on 2026-05-23.

## Context

The v2 planning work identified two recurring agnosticism questions:

- should `Provider`, `Service`, or `Capability` be renamed to more generic
  terms;
- should the core expose generic action metadata so consumers can attach
  arbitrary domain payloads to decisions.

Both changes would affect public API, generated policies, migration docs, and
consumer code.

## Decision

Original decision: `Provider`, `Service`, and `Capability` stayed as the v2
primitive names.

Amended decision for v2.1.x: CervoRules introduces more neutral preferred names
while keeping the original names as deprecated aliases:

| Deprecated compatibility alias | Preferred name | Rationale |
| --- | --- | --- |
| `Capability` | `Operation` | Describes what the request wants to do without assuming gateway vocabulary. |
| `Service` | `Target` | Describes the selected destination without implying a service topology. |
| `Provider` | `Executor` | Describes the caller-owned execution choice without implying AI/provider or backend-specific semantics. |

The aliases remain source-compatible for v2 consumers and generated policies.
Generated code may continue to emit the compatibility aliases until a separate
generator migration is planned.

Generic action metadata is Deferred. CervoRules will not add a public
`map[string]any`, arbitrary payload, or unbounded action metadata field to
`Decision`, `Action`, `Request`, or generated policy output in v2.0.

Consumers should use one of these existing extension points:

- custom actions for domain-specific behavior;
- generated vocabulary constants for domain names;
- bounded first-class fields when the concept is already cross-domain;
- consumer-owned adapter or report metadata outside the core runtime.

## Rationale

The original names are understandable in routing and gateway-heavy domains, but
they read less naturally for document processing, device orchestration,
scheduled jobs, and CLI commands. Preferred neutral names improve future
project onboarding without forcing immediate consumer rewrites.

Unbounded generic action metadata would make the core look flexible but would
also weaken compatibility, privacy review, generated policy contracts, and
agnosticism. Future metadata should be added only when at least two domains
need the same bounded shape.

## Consequences

- v2 migration remains source-compatible because old names are type aliases.
- New code should prefer `Operation`, `Target`, and `Executor`.
- Existing domains may continue compiling with `Capability`, `Service`, and
  `Provider` while migrating gradually.
- Public metadata expansion requires a new ADR with compatibility and privacy
  analysis.
- CervoProxy-specific metadata remains in CervoProxy or generated CervoProxy
  vocabulary.

## Deferred

Potential future work:

- migrate generated policy output to preferred names in a separate PR;
- add bounded action metadata only if two or more domains need the same fields;
- document migration tooling if a future major version removes aliases.
