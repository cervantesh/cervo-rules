# Adapter Patterns

CervoRules evaluates caller-owned facts. It does not own transports, queues,
schedulers, device protocols, HTTP frameworks, or command runners.

The stable boundary is:

```text
input envelope -> adapter -> cervorules.Request -> DecisionFlow.Decide
```

Adapters should stay in the consuming application unless they are generic enough
to be useful across multiple projects.

CervoRules provides a tiny generic interface for reusable adapters:

```go
type FactsAdapter[T any] interface {
    FactsFrom(context.Context, T) (cervorules.RequestFacts, error)
}
```

Use `FactsAdapterFunc[T]` when a function is enough:

```go
adapter := cervorules.FactsAdapterFunc[Event](func(event Event) (cervorules.RequestFacts, error) {
    return cervorules.RequestFacts{
        ID:             event.ID,
        Channel:        "queue",
        CapabilityHint: cervorules.Capability(event.Type),
    }, nil
})

facts, err := adapter.FactsFrom(ctx, event)
```

## Queue Or Event Message

Use the message id as `Request.ID`, message type as `Capability`, tenant or
priority data as `Metadata`, and producer or actor as `User` when available.

```go
req := cervorules.Request{
    ID:         event.ID,
    Channel:    "queue",
    User:       event.Actor,
    Capability: cervorules.Capability(event.Type),
    Risk:       cervorules.Risk(event.Risk),
    Metadata: map[string]string{
        "tenant": event.Tenant,
        "source": event.Source,
    },
}
```

## Scheduled Job

Use the job name or operation as `Capability`. Keep schedule timestamps and
retry counters in metadata only when the policy needs them.

```go
req := cervorules.Request{
    ID:         job.RunID,
    Channel:    "scheduler",
    Capability: cervorules.Capability(job.Name),
    Metadata: map[string]string{
        "window": job.Window,
    },
}
```

## CLI Or Internal Command

Use command names as capabilities. Use the operator or service account as
`User`, and keep argument-derived values sanitized before adding metadata.

```go
req := cervorules.Request{
    ID:         command.ID,
    Channel:    "cli",
    User:       command.Operator,
    Capability: cervorules.Capability(command.Name),
    Metadata: map[string]string{
        "environment": command.Environment,
    },
}
```

## Device Telemetry Or Command

Use telemetry or command class as `Capability`. Put bounded device class,
region, or fleet information in metadata. Avoid raw device ids in metrics.

```go
req := cervorules.Request{
    ID:         message.ID,
    Channel:    "device",
    Capability: cervorules.Capability(message.Kind),
    Risk:       cervorules.Risk(message.Risk),
    Metadata: map[string]string{
        "fleet":  message.Fleet,
        "region": message.Region,
    },
}
```

## gRPC Metadata

Keep gRPC-specific parsing in the service edge. Map method or operation names to
capabilities, bounded metadata values to `RequestFacts.Metadata`, and caller
identity to `User` only if the policy needs it.

```go
facts := cervorules.RequestFacts{
    ID:             requestID,
    Channel:        "grpc",
    User:           principal,
    CapabilityHint: cervorules.Capability(methodName),
    Metadata: map[string]string{
        "region": region,
        "tier":   tenantTier,
    },
}
```

## HTTP

HTTP support is an optional adapter, not the identity of CervoRules. Consumers
that front HTTP traffic can use `FactsFromHTTPRequest` for simple extraction or
`NewClassifier` for production classifiers that precompile regex path
rules.

```go
classifier, err := httpadapter.NewClassifier(httpadapter.HTTPClassificationOptions{
    CapabilityHeaders: []string{"X-Policy-Capability"},
    UserHeaders:       []string{"X-Policy-User"},
    PathCapabilities: []cervorules.PathCapability{
        {Prefix: "/v1/messages", Methods: []string{"POST"}, Capability: "message_create"},
    },
})
if err != nil {
    return err
}

requestFacts := classifier.FactsFromHTTPRequest(r)
req := requestFacts.Request()
```

## Adapter Rules

- Keep domain vocabulary in the consuming project.
- Normalize high-cardinality values before they reach metrics labels.
- Prefer bounded metadata values such as region, class, tenant tier, or command
  group.
- Keep transport-specific parsing outside core policy code.
- Add a core adapter only when at least two domains need the same shape.
- Prefer `FactsAdapter[T]` for shared adapters and plain local functions for
  one-off mappings.
