# ADR 0001: Cross-Domain Public API Guardrail

Date: 2026-05-21

## Status

Accepted

## Context

CervoRules is shared infrastructure. CervoProxy is an important consumer, but
its gateway needs must not silently define the core API for every future
project. New public fields and DSL features can create long-lived coupling even
when the implementation is small.

## Decision

Every new public API, public struct field, generated policy helper, or DSL field
must pass cross-domain review before merge.

The change must satisfy one of these paths:

1. It is useful in at least two non-identical domains, such as gateway routing
   and document processing, or billing and device orchestration.
2. It is explicitly adapter-specific and named/documented as an adapter rather
   than a core concept.
3. It remains private to the consumer or generated consumer vocabulary.

Needs from CervoProxy must stay in CervoProxy or generated CervoProxy policy
vocabulary unless they can be expressed as a cross-domain concept.

Fields on `PolicyRuntimeConfig`, `Decision`, `Request`, public action types, and
the policy DSL require this review. Transport-derived behavior should live in
adapters such as HTTP classification or consumer-owned mapping code.

## Consequences

- CervoRules may reject convenient one-consumer shortcuts.
- New generic metadata surfaces are deferred until real cross-domain use exists.
- ADRs are required when a consumer need would otherwise shape core semantics.
- Release review must check this ADR when public API or DSL changes.
