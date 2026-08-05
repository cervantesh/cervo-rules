# 0015 - Caller-Owned Error Redaction

## Status

Accepted. Supersedes the redaction behavior described in ADR 0001 discussion
notes and changes observable output.

## Context

`core.Error.Redacted` decided whether to withhold `Value` by substring-matching
the `Field` name against a hardcoded list:

```go
[]string{"token", "auth", "header", "body", "prompt", "content", "memory"}
```

Two problems.

First, it is a domain commitment inside `core`. `docs/agnosticism.md` requires
asking whether a name "implies HTTP, gateway, AI, or one provider ecosystem"
before it enters the public surface. `header` and `body` imply HTTP; `prompt`,
`content` and `memory` imply an AI consumer. `core` does not otherwise know
those domains exist, and `limits` — where the AI-shaped budget vocabulary
actually lives — is a separate package that does not even import `core`.

Second, substring matching produces false positives. The field `max_tokens`
contains `token`, so a legitimate numeric limit was replaced with `[REDACTED]`,
hiding exactly the diagnostic a caller needed.

`core` cannot know which values are secret. Only the producer of an error knows
what it put in `Value`.

## Decision

Remove the built-in field vocabulary. Redaction becomes explicit and
caller-owned through two mechanisms:

1. `Error.Sensitive bool`. The producer marks the value. `Redacted` honors only
   this marker, and `MarshalJSON` calls `Redacted`.
2. `Redactor func(field string) bool` with `RedactFields(names ...string)` and
   `Error.RedactWith` / `Errors.RedactWith`. A consumer or adapter supplies its
   own field vocabulary.

`RedactFields` matches whole segments, splitting on `.`, `_`, `[` and `]`, so
`max_tokens` no longer matches `token` while `auth_header` still matches `auth`.

`Error.Error()` now also withholds a value marked sensitive. Previously the
string form printed it unredacted while JSON did not, so anything logging
`err.Error()` bypassed redaction entirely.

Transport-shaped names move to their owner: `httpadapter.Redactor()` and
`httpadapter.SensitiveFieldNames()`.

## Consequences

### Positive

- `core` contains no domain vocabulary. Verified mechanically: no HTTP, gateway
  or AI terms remain in non-test `core` sources.
- The `max_tokens` false positive is fixed.
- The `Error()` string leak is closed.
- Adapters own their own sensitive names, matching how `httpadapter` already
  owns HTTP classification.

### Negative

- **Behavior change.** A consumer that relied on automatic name-based redaction
  loses it silently on upgrade. This is the reason the ADR exists rather than a
  quiet patch.

### Migration

Restore the previous behavior in one call at the serialization boundary:

```go
redactor := core.RedactFields("token", "auth", "header", "body", "prompt", "content", "memory")
payload := errs.RedactWith(redactor)
```

For HTTP consumers:

```go
payload := errs.RedactWith(httpadapter.Redactor())
```

Prefer marking at the source instead, since it survives any serialization path:

```go
core.Error{Code: code, Field: "credential", Value: raw, Sensitive: true}
```

## Alternatives Considered

**Keep the list, fix only the substring bug.** Rejected. It leaves a domain
commitment in `core` and keeps `core` guessing at meanings it cannot know.

**Redact every value unless marked safe.** Rejected. Most values are the
diagnostic payload — an unknown operation name is the point of the error.
Inverting the default would make errors useless to preserve secrecy that the
producer can declare precisely.

**Move the list to a package-level variable consumers mutate.** Rejected.
Global mutable state makes redaction depend on init order and is not safe for
consumers embedding more than one policy.
