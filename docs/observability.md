# Observability

CervoRules exposes stack-neutral decision observations. It does not import a
logger, metrics client, tracing SDK, OpenTelemetry, or Prometheus package.
Applications adapt the observation contract to their own operational stack.

## Contract

Create one observation or report after each decision:

```go
result, err := engine.Decide(ctx, req)
if err != nil {
    return err
}

obs := cervorules.NewDecisionObservation(result)
report := cervorules.NewPolicyEvaluationReport(result)
schema := cervorules.PolicyEvaluationReportContract()
```

`PolicyEvaluationReportContract()` returns a machine-readable descriptor with
version `cervorules.policy_evaluation_report.v1`. The descriptor lists the
allowed top-level report fields, observation fields, audit fields, and metric
labels so adapters can assert compatibility without importing logging, OTel, or
Prometheus packages.

`MetricLabels()` returns stable, low-cardinality dimensions:

| Label | Source |
| --- | --- |
| `outcome` | allow or deny |
| `capability` | caller-owned request capability |
| `risk` | caller-owned request risk |
| `service` | selected service |
| `provider` | selected provider |
| `audit_class` | final audit class |
| `rule_id` | final rule id |
| `decision_flow` | compiled decision flow name |

`LogFields()` returns diagnostic fields:

| Field | Source |
| --- | --- |
| `request_id`, `channel`, `capability`, `risk` | request facts |
| `outcome`, `reason`, `rule_id`, `service`, `provider` | decision result |
| `audit_class`, `requires_audit` | audit result |
| `decision_flow`, `ruleset_version` | policy identity |
| `timeout_ms` | effective timeout |
| `phase_count`, `rule_count`, `matched_rule_count` | trace summary |
| `required_rule_count`, `action_count` | trace summary |
| `fallback_service_count`, `fallback_provider_count` | fallback summary |

## Privacy And Cardinality

`DecisionObservation` intentionally omits `Request.User` and
`Request.Metadata`. User identifiers, tenants, document ids, prompts, and other
application metadata often have privacy and high-cardinality risk. If an
application needs those values, it should add sanitized fields in its own
adapter.

Do not add these values to `MetricLabels()`:

- user ids;
- request ids;
- raw reasons;
- metadata values;
- tenant ids unless the caller has a bounded tenant set and accepts the metric
  cardinality.

Use `PolicyEvaluationReport.Redacted(cervorules.PrivacySafeRedactionPolicy())`
before sending reports to shared logs, external audit sinks, or metric events
where request ids, raw reasons, or diagnostics may leak sensitive details.
`OperationalRedactionPolicy()` is intended for trusted internal channels only.

Optional logic-inspired fact traces need the same discipline. Derived fact
explanations may contain tenant ids, user ids, document ids, or device ids in
term values. `facts.RedactionPolicy` can redact full predicates or selected
term positions before a fact set or `facts.DerivationTrace` is serialized,
logged, or attached to an explain artifact. Applications should still avoid
using raw fact terms as metric labels.

## Evaluation Reports

`PolicyEvaluationReport` combines:

| Field | Purpose |
| --- | --- |
| `Observation` | flattened request/decision facts for metrics and logs |
| `Trace` | bounded `TraceSummary` counts |
| `Audit` | stable audit event schema |
| `Diagnostics` | structured validation or runtime diagnostics |

The audit event schema is versioned with
`cervorules.AuditEventSchemaVersion`. Audit events deliberately use bounded
fields and diagnostic codes instead of raw metadata, users, tenants, prompts, or
payload values.

The report contract schema is versioned separately with
`cervorules.PolicyEvaluationReportSchemaVersion`. Increment this version only
when the report descriptor changes in a way that consumers must handle
explicitly.

## Structured Logging Adapter

```go
obs := cervorules.NewDecisionObservation(result)
logger.Info("policy decision", obs.LogFields())
```

If the logger requires typed arguments, adapt the map at the edge:

```go
for key, value := range obs.LogFields() {
    logger = logger.With(key, value)
}
logger.Info("policy decision")
```

## Prometheus-Style Adapter

Keep the dependency in the application, not in CervoRules:

```go
obs := cervorules.NewDecisionObservation(result)
labels := obs.MetricLabels()

policyDecisionTotal.WithLabelValues(
    labels["outcome"],
    labels["capability"],
    labels["risk"],
    labels["service"],
    labels["provider"],
    labels["audit_class"],
    labels["rule_id"],
    labels["decision_flow"],
).Inc()
```

## Audit Adapter

```go
report := cervorules.NewPolicyEvaluationReport(result).
    Redacted(cervorules.PrivacySafeRedactionPolicy())

auditSink.Write(ctx, report.Audit)
```

Keep persistence, transport, retries, signing, and schema registry concerns in
the consuming application. CervoRules owns only the stable in-process event
shape.

Recommended metric names:

| Metric | Type | Labels |
| --- | --- | --- |
| `cervorules_decision_total` | counter | all labels from `MetricLabels()` |
| `cervorules_decision_trace_rules` | histogram or summary | `decision_flow`, `outcome` |
| `cervorules_decision_timeout_ms` | histogram | `decision_flow`, `service`, `provider` |

## Release Expectations

Before release:

- run `go test -run TestDecisionObservation -count=1 .`;
- confirm no new labels or log fields were added without updating this file;
- confirm audit schema changes update `AuditEventSchemaVersion` only with a
  migration note;
- confirm no observability dependency was added to the root runtime package;
- update the maturity report if the operational contract changes.
