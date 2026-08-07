# Modular Architecture

CervoRules v2 is split into public packages so consumers can import only the
surface they need. The root package remains a compatibility facade, but new code
should prefer direct subpackage imports.

## Import Matrix

| Package | Owns | Should not own |
| --- | --- | --- |
| `core` | Decision flows, compiled engines, requests, decisions, predicates, actions, vocabulary, retry, breaker, trace, inspection. | Runtime config loading, transport parsing, package publishing, generator parsing. |
| `runtime` | `PolicyRuntimeConfig`, runtime config merge, config validation, runtime option constructors. | Decision evaluation, HTTP parsing, consumer environment variable parsing. |
| `limits` | `Budget`, `Requested`, `Violations`, `Check`. | JSON payload parsing, HTTP status mapping, provider-specific payload names. |
| `httpadapter` | Optional HTTP request-to-facts classification and generic facts adapters. | Core policy semantics, non-HTTP adapter ownership. |
| `facts` | Optional flat facts, pattern queries, bounded derivation, validation diagnostics, and derivation explanations. | Core policy decisions, transport parsing, consumer vocabulary, Prolog runtime semantics. |
| `decisioncache` | Reserved: a module marker with no cache contracts yet. | Core policy semantics, cache invalidation policy, distributed cache ownership, live health freshness. |
| `observe` | Decision observations, reports, redaction, audit envelopes. | Logger, metrics, tracing, database, or audit sink dependencies. |
| `testkit` | Generated policy contract checks and readiness scorecards. | Runtime production behavior or generator implementation. |

## Root Facade

`github.com/cervantesh/cervo-rules/v2` is a compatibility facade. It exposes
aliases and delegating functions for common public symbols so existing v2 Beta
consumers do not need an immediate source rewrite.

Root facade rules:

- no copied implementation;
- no generator dependency;
- no external runtime dependency;
- no consumer-specific vocabulary;
- no new public API unless the owning subpackage is chosen first.

## Generated Policy Imports

Generated policies import modular packages directly:

```go
import (
    cervorules "github.com/cervantesh/cervo-rules/v2/core"
    cervolimits "github.com/cervantesh/cervo-rules/v2/limits"
    cervoruntime "github.com/cervantesh/cervo-rules/v2/runtime"
)
```

Generated factory methods take `runtime.PolicyRuntimeConfig` and return
`core.Engine`. This keeps generated policy packages independent from the root
facade while preserving readable generated code.

## Migration Sequence

1. Regenerate policy code with the v2 tools.
2. Change consumer runtime config code to import `runtime`.
3. Change request/decision handling to import `core` where narrow imports help.
4. Keep root facade imports temporarily only when migration churn is not worth
   the immediate change.
5. Add a `testkit.ConsumerConformanceContract` test beside the generated
   policy package.

## Change Management

Public API additions must land in the owning subpackage before they are exposed
through the root facade. If a requested API crosses package boundaries, create
an ADR or change request before implementation.

When a consumer need is transport-specific, prefer an adapter package. HTTP
classification belongs in `httpadapter`; queue, CLI, device, or scheduler
mapping should remain consumer-owned unless two or more domains prove a common
shape.

## Wiki Reports

The following project reports should be refreshed when package boundaries move:

- [Agnosticism report](https://github.com/cervantesh/cervo-rules/wiki/Reports-Agnosticism-2026-05)
- [Maturity report](https://github.com/cervantesh/cervo-rules/wiki/Reports-Maturity-2026-05)
- [Dependencies report](https://github.com/cervantesh/cervo-rules/wiki/Reports-Dependencies-2026-05)
- [Smells report](https://github.com/cervantesh/cervo-rules/wiki/Reports-Smells-2026-05)

Issue #124 tracks the modular public package work. Issues #125 through #136
track the implementation sequence and release readiness.
