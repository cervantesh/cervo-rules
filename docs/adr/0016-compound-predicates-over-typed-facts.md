# ADR 0016: Compound Predicates Over Typed Facts

Date: 2026-08-05

## Status

Accepted.

## Context

[ADR 0014](0014-symbolic-guard-layer.md) added the `Condition` seam and closed
with: "Put constraints in the policy DSL only. Rejected as a starting point. The
Go API has to exist first; DSL surface for it is a separate decision." This is
that decision.

A v3 policy could say which operation goes to which target. It could not say
under what numeric conditions. `conditions` + `requires` put the seam in the
right place but left a condition's meaning opaque: the name was in the policy,
the threshold was in the consumer's Go. A reviewer reading
`requires: [risk_within_limits]` could not tell whether the limit was 1.5 or
15.0, and changing one to the other did not move the `PolicyHash`.

The motivating consumer authorizes trading orders. Seven of its eight rules are
predicates over facts, and two of the seven are disjunctions
(`risk_pct > 1.5 OR exposure_pct > 5.0`). `requires` only conjoins, so the whole
set lived in hand-written Go outside the policy, outside the hash, and outside
the audit trail.

Three defects surfaced while verifying the ground truth, all pre-existing and
all invisible to the test suite:

- denies were emitted as a Go map keyed by operation, so two denies on one
  operation produced a duplicate constant key — `check` and `generate` both
  passed and only the consumer's build failed;
- a deny naming an operation absent from the vocabulary was accepted and never
  fired, so a typo was a silent fail-open;
- the generated `mergeRuntimeConfig` dropped `PolicyRuntimeConfig.Conditions`,
  and since `DefaultConfig` cannot supply an evaluator, `ValidateConfig` then
  rejected every condition-gated build. The rc.4 feature was unusable end to
  end.

## Decision

Add declarative predicates over typed facts to the v3 DSL, and populate the
decision trace so a compound refusal can be audited.

**Facts are declared in two places, and the split is the rule.** The vocabulary
declares identity and type (`facts:` with `type: number|integer|bool|enum|
string`, `values` for enum, `go_name`). The policy declares the numbers that
constitute a decision (`min`, `max`, `default`). Any value that, if edited,
changes which requests are denied lives in the policy file, because the
`PolicyHash` is `sha256` over that file's raw bytes. `DefaultConfig` gains no
security value.

**A predicate is a closed tagged form, not an expression language.**
Composition is `all` / `any` / `not`, nestable; a leaf compares one declared
fact against a literal, against another fact of the same declared type, or
consults a named condition. The YAML decoder is the parser and the JSON Schema
is the published contract; the generator validates arity and types.

**Predicates carry no runtime evaluator.** Each one compiles to a Go boolean
expression the compiler checks, emitted as one matcher function per rule. There
is nothing to sandbox because there is nothing to interpret, which is how the
feature adds no third-party dependency.

**Facts travel in `core.Request.Metadata`.** `core.Request` is unchanged. A
generated frame parses every fact the policy references, once per decision,
before any rule runs, so a decision is a pure function of a validated frame and
does not depend on which rule happens to be first.

**Denies become an ordered list.** First match wins, `operation` is optional
(absent applies the rule to every operation), `id` is required, and `reason`
falls back to `id`.

**`core` gains `ErrorCodeMissingFact` and `ErrorCodeInvalidFact`.** Per ADR
0014's ownership rule, codes belonging to a core-owned contract stay in `core`,
and the frame reads `core.Request.Metadata`, which core owns.

**Generated engines populate `DecisionTrace.Steps`**, one per evaluated rule,
carrying the rule id, whether it matched, and the leaf that decided it. Trace
stays opt-in per decision.

## Consequences

### Positive

- A policy states the whole of what it authorizes. The thresholds are in the
  file the hash covers, so changing a risk limit is an auditable change.
- Fail-closed becomes structural rather than remembered. `strconv.ParseFloat`
  accepts `"NaN"`, and a non-finite value passes every comparison without
  matching any — it would slip past every deny rule and land on the allow path.
  The generated parser rejects it, along with an absent fact with no declared
  default, a wrong type, and an out-of-domain value. Each returns a structured
  error; none returns `false`.
- A compound denial is auditable: the trace names the rule and which disjunct
  fired, instead of reporting only that the policy refused.
- Three latent defects are closed, one of which made the previous release's
  headline feature unusable.
- `vocabgen` emits a `Fact*` constant per declared fact, so a misspelled
  metadata key in a consumer is a compile error rather than a missing fact at
  runtime.

### Negative, and what mitigates each

**The DSL grows a second declaration site for facts.** A reader now looks in two
files to know what a fact is and what bounds apply. Mitigated by the rule being
stateable in one sentence — types in the vocabulary, decision-bearing numbers in
the policy — and by the alternative being worse: bounds in the vocabulary would
sit under `VocabularyHash`, which consumers commonly do not record.

**The grammar will attract requests to grow.** Arithmetic, string functions,
regular expressions, and user-supplied comparators are all one plausible ticket
away, and each one moves the feature toward the expression language this ADR
declined. Mitigation: the leaf set is closed and enumerated in the schema, and
any addition is a change to this ADR, not a change to a table. The test that a
proposed leaf must pass is whether it can be evaluated without interpretation —
if it needs a runtime evaluator, it does not belong here.

**Eager frame construction parses facts a short-circuiting evaluation would
skip.** A policy referencing many facts pays for all of them on every decision,
including ones no matching rule would consult. Accepted deliberately: the
alternative makes the outcome depend on rule order, so a malformed fact could
hide behind an earlier deny. Cost is bounded by the referenced set, not by the
declared set — a fact no rule reads never enters the frame.

**Every referenced fact must be supplied on every request, regardless of
operation.** The frame is policy-wide, so a fact read only by a rule for
operation A is still required when deciding operation B, unless it declares a
default. This is a real authoring constraint and the reason `default:` exists.

**Trace steps carry predicate text.** A step names the fact and the bound that
decided a rule. Predicate text is authored policy, not request data, and
observed values stay in `Error.Value` marked `Sensitive`. Consumers that
serialize a trace should still treat it as internal.

**Three deliberate breaks.** Deny operations are now validated against the
vocabulary, duplicate route operations are rejected at `check`, and `deny.id` is
now required. Each closes a hole where a policy that generated cleanly could not
be audited or could not be built. Each is recorded in the CHANGELOG under
Changed, and none affects a policy that was already correct.

## Alternatives Considered

**An expression language, or an embedded expression engine.** Rejected. A
general evaluator inside a security policy is attack surface, and a third-party
one contradicts the zero-dependency posture. The closed tagged form is
deliberately too small to express anything but a guard.

**A string mini-syntax such as `risk_pct > 1.5`.** Rejected. It needs a lexer
and a parser, it diffs worse, and it makes authoring mistakes into runtime
surprises rather than schema failures.

**Extend `facts` instead.** Rejected, for the reason ADR 0014 gave for
`ontology`: `facts` owns derivation of new facts, this layer owns refutation of
a proposed action. `facts` is a bounded datalog engine with iteration budgets
and string-typed terms; a decision guard is on the hot path and needs typed
scalars in constant time. A consumer that genuinely needs derived facts derives
them first and puts the results in `Metadata`.

**A separate typed channel on `core.Request`.** Rejected. It adds public surface
and forces consumers to serialize twice. The request arrived as strings anyway;
what matters is that exactly one place converts them, that the conversion is
generated from a declaration, and that every failure mode is an error.

**Keep `conditions` only, with thresholds in Go.** Rejected: it is the status
quo the work exists to remove. It gives a policy traceability without
expressiveness, and leaves the numbers outside the hash.

**Phase 2 without `any`.** Considered and shipped as an intermediate, then
superseded. It works — a disjunction splits into two denies sharing an id — but
it duplicates rules and degrades the explanation, and it is only viable at all
once denies are ordered.

## Related

- [ADR 0014](0014-symbolic-guard-layer.md), which deferred this decision and
  whose Condition seam this makes usable.
- [ADR 0015](0015-caller-owned-error-redaction.md), for the `Sensitive` marker
  used on observed fact values.
- `docs/v3/compound-predicates.md` for the long-form design record, the
  worked trading-policy example, and the verification evidence.
- `docs/v3/policygen-dsl.md` for the authoring reference.
