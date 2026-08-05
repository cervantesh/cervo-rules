# Agent Workflow

This workflow is the path agents should follow when changing declarative
CervoRules policy or generated consumers.

1. Read `README.md`, `AGENTS.md`, this file, `docs/policy-authoring.md`,
   `examples/README.md`, and the schema files under `schemas/`.
2. Choose the closest neutral example or conformance fixture.
3. If input is not HTTP, read `docs/adapter-patterns.md` and keep event, job,
   command, or device mapping in the consumer adapter.
4. Edit `policy-vocabulary.yaml` first. Add every operation, target, and
   executor that policy rules may reference. Include `description` and `go_name`
   where generated Go names need explicit control.
5. Edit `policy-rules.yaml` second. Choose a stable `name`, then update
   trusted users, defaults, routes, denies, disabled routes, fallbacks, and tests
   as needed.
6. Run `cervorules-policygen check` to validate both files, catch unknown
   vocabulary, reject unsupported fields, and validate declarative tests without
   writing files.
7. If behavior needs richer matching than operation routing, keep that logic in
   consumer code or open a v3 generator issue before adding unsupported DSL
   fields.
8. Review generated output if the check succeeds. Confirm every route, deny,
   disabled route, fallback, and runtime default matches the requested policy
   change.
9. Generate vocabulary constants with `cervorules-vocabgen`.
10. Generate policy code and generated tests with `cervorules-policygen`.
11. Add or update one hand-written `testkit` contract beside the generated
    package.
12. Run the consumer test suite, including generated tests and the hand-written
    contract test.
13. For behavior review, run generated tests and attach the policy diff.
14. For v3 consumers, build through `runtime.PolicyFactory` and validate config
    before startup.
15. Update the consumer changelog or release notes with policy-impacting
    behavior changes.

## Documentation Refresh Workflow

For repo documentation changes, create or update a issue with estimate,
start time, actual time, deviation, touched docs, touched wiki pages, tests run,
and PR link. Review repository docs first, then wiki pages. Treat historical
reports as historical snapshots instead of rewriting their original context.
Run focused docs drift tests before opening the PR.

Agents must not add application-specific vocabulary to CervoRules itself.
Operations, targets, executors, users, risk labels, and audit meanings belong to
the consuming service.

## V3 Consumer Contract Shape

Generated v3 consumers should certify their factory with `testkit`:

```go
func TestGeneratedPolicyContract(t *testing.T) {
    testkit.MustAssertConsumerConformance(t, testkit.ConsumerConformanceContract{
        Name:           "billing",
        PolicyPath:     "policy-rules.yaml",
        VocabularyPath: "policy-vocabulary.yaml",
        RuntimeCases: []testkit.RuntimeCase{{
            Name:    "invoice read",
            Factory: policyrules.NewPolicyFactory(),
            Config:  policyrules.NewPolicyFactory().DefaultConfig(),
            Request: core.Request{
                Operation: core.Operation("invoice.read"),
            },
            WantDecision: core.Decision{
                Allow:  true,
                Target: core.Target("billing-reader"),
            },
        }},
    })
}
```

For non-HTTP consumers, keep transport parsing in the application and expose a
small adapter that returns `core.Request` or consumer-owned facts. Adapter
patterns for queues, scheduled jobs, CLI commands, device telemetry, and gRPC
metadata are documented in `docs/adapter-patterns.md`.
