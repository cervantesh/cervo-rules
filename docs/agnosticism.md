# Agnosticism Guide

CervoRules is shared policy infrastructure. It should support gateway routing,
billing, document processing, device orchestration, scheduled jobs, queues,
internal commands, and future CervoSoft projects without making one consumer the
shape of the core API.

Agents and automation should use `AGENTS.md`, `.cervorules/agent-manifest.json`,
and `docs/agent-quickstart.md` as the machine-facing entrypoints for this
boundary.

## Dynamic Vocabulary

Use `Vocabulary` when valid names are not known at code authoring time.
Generated policies usually expose a static vocabulary, but applications can also
build validators dynamically:

```go
vocab := cervorules.NewVocabulary(
    cervorules.AllowedCapabilities("invoice_lookup", "payment_capture"),
    cervorules.AllowedServices("ledger", "archive"),
    cervorules.AllowedProviders("primary", "backup"),
)

if err := vocab.ValidateRequest(req); err != nil {
    return err
}
```

Empty allowed sets are intentionally open. This lets a consumer centrally own
only the dimensions it can validate and keep other dimensions flexible.

## Naming Review

Before adding or renaming a public primitive, ask:

- Does the name work in at least two domains?
- Does the name imply HTTP, gateway, AI, or one provider ecosystem?
- Could this be adapter-specific instead of core runtime API?
- Does `docs/adr/0001-cross-domain-public-api.md` require a new ADR?

## Operation, Target, Executor

New code should use these neutral primitive names:

- `Operation`: what the request wants to do;
- `Target`: the selected destination;
- `Executor`: the caller-owned execution choice.

The older `Capability`, `Service`, and `Provider` names remain as deprecated
aliases for v2 source compatibility. Generated policies and existing consumers
can migrate gradually.

## Provider, Service, and Capability Compatibility

`Provider`, `Service`, and `Capability` remain available in v2. They are not
domain-owned by CervoProxy, but they are now compatibility aliases.

Adapters and generated vocabulary can choose domain-specific constants while
the core keeps the preferred primitives stable.

## Action Metadata Boundary

Generic action metadata is deferred. The current extension points are:

- custom `Action` implementations;
- generated policy vocabulary;
- `Decision` fields that already model routing, fallback, audit, retry, breaker,
  and limits;
- consumer-owned metadata outside CervoRules.

Do not add a public `map[string]any` or generic action payload field to core
types just to satisfy one consumer. If two or more domains need the same
metadata shape, open an ADR that defines the bounded fields, privacy behavior,
compatibility impact, and generated policy semantics.

## Adapter Boundary

Transport-derived behavior belongs in adapters. HTTP, queue, scheduled job,
CLI/internal command, device telemetry, and gRPC inputs should map into
`RequestFacts` before policy evaluation. Use `FactsAdapter[T]` when a reusable
adapter shape is needed.

See `docs/adapter-patterns.md` for queue/event, scheduled job, CLI/internal
command, device telemetry, gRPC, and HTTP adapter examples.

## Neutral Examples And Conformance

The repository keeps neutral examples and conformance fixtures so agents can
copy policy shapes without inheriting a gateway vocabulary:

- `examples/billing`;
- `examples/document-processing`;
- `examples/device-routing`;
- `examples/conformance/billing-routing`;
- `examples/conformance/document-processing`;
- `examples/conformance/device-routing`;
- `examples/conformance/queue-event-routing`;
- `examples/conformance/scheduled-job-routing`;
- `examples/conformance/cli-command-routing`.

Use these examples before creating new gateway-shaped examples. If a new public
field or helper is only useful for one consumer, keep it in the consuming
project or write an ADR before changing CervoRules.

## Logic-Inspired Facts Boundary

The optional `facts` package can derive cross-domain relationships before a
policy decision, for example tenant plans, user roles, device capabilities, or
document classifications. The package intentionally uses caller-owned predicate
names and flat terms instead of CervoRules-owned domain vocabulary.

Keep derived facts agnostic:

- predicate names should be owned by the consumer or generated vocabulary;
- facts must not imply HTTP, gateway, AI, or provider-specific payloads;
- core policy evaluation must remain usable without importing `facts`;
- advanced logic features such as negation, recursion, and aggregates require
  ADRs before public implementation.
