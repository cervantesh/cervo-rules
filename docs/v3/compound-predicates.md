# Compound Predicates over Typed Facts

Design and implementation notes. All three phases in [Phases](#phases) have
landed; this document is the record of what was decided and why.

Line references to unchanged code are against `771a247`, the revision the
design was verified against.

## 1. Problem

A v3 policy can say which operation goes to which target. It cannot say under
what numeric conditions. Every predicate over facts lives in hand-written Go in
the consumer, outside the policy, outside the `PolicyHash`, and outside the
audit trail.

`conditions` + `requires` (added in rc.4) put the seam in the right place but
left the meaning of a condition opaque: the name is in the policy, the threshold
is in Go. A reviewer reading `requires: [risk_within_limits]` cannot tell
whether the limit is 1.5 or 15.0, and changing it from one to the other does not
move the `PolicyHash`.

## 2. Three blocking defects found while verifying this

All three were found by generating and compiling, not by reading. All three are
independent of the feature; §2.1 and §2.2 are fixed by Phase A, §2.3 by a
one-line change to the generated merge.

### 2.1 Two denies on the same operation emit Go that does not compile

`internal/policygen/render.go:218-222` emits denies as a Go map literal keyed by
operation:

```go
denies := map[cervorules.Operation]generatedDeny{
	cervorules.Operation("order.place"): {reason: "first", ...},
	cervorules.Operation("order.place"): {reason: "second", ...},
}
```

`check` reports `ok ... denies=2`. `generate` writes the file and exits 0 —
`go/format` parses and formats, it does not type-check. The failure surfaces in
the *consumer's* build:

```
duplicate key "order.place" in map literal
```

The same applies to `routes` (`render.go:170-181`).

This matters for this design specifically: the brief describes "split each
disjunction into two denies" as a viable-but-ugly workaround for the absence of
`any`. It is not viable. It does not build today.

### 2.2 A deny naming an operation that does not exist is accepted silently

`validate` (`generator.go:320-327`) checks only that `deny.operation` is
non-empty. It never consults the vocabulary index, although routes are checked
(`generator.go:296`). A policy with `operation: order.plcae` passes `check`,
generates, compiles, and the deny never fires. A typo in a deny rule is a
silent fail-open.

### 2.3 No condition-gated policy could be built at all

The generated `mergeRuntimeConfig` copied trusted users, the default executor,
operation targets and executor fallbacks from the caller's config — but not
`Conditions`. Since `DefaultConfig` cannot supply an evaluator (it is a
caller-owned collaborator, not config data), the merged config always had
`Conditions == nil`, and the `ValidateConfig` check added in the same release
then rejected the build:

```
policy build failed: missing_conditions: policy declares 1 condition(s)
but no evaluator is configured
```

The `conditions` + `requires` feature shipped in rc.4 was therefore unusable end
to end. The existing tests only covered the nil-evaluator case and the generated
source text, so nothing exercised a successful build with a real evaluator.

## 3. State before this change (verified at `771a247`)

The table below describes the world the design was written against. Everything
in the "State" column has since changed except the fail-closed behaviours, which
were the parts worth keeping.

| Piece | Where | State |
| --- | --- | --- |
| `Condition`, `Conditions`, `ConditionFunc`, `ConditionSet` | `core/conditions.go` | Fail-closed on unknown/unwired |
| `conditions:` map, `requires: []string` | `schemas/v3/policy-rules.schema.json:14,43` | AND only |
| `validateConditions`, `validateRequires` | `generator.go:338-389` | Rejects undeclared conditions |
| `conditionsHold` in the generated engine | `render.go:312-323` | Unanswerable condition fails the decision |
| `PolicyRuntimeConfig.Conditions` | `runtime/factory.go:41` | Carried by reference through `Clone` |
| `DecisionTrace{Steps []DecisionTraceStep}` | `core/decision_result.go:19-29` | Envelope allocated, **never populated** |

Three limits, all of them removed by this change:

1. `requires` was AND only. No `any`, no `not`, at any layer.
2. `conditions[].kind` covered `transition_allowed | in_state | has_type |
   integrity` — lifecycle and ontology. No comparison over facts.
3. The vocabulary declared `operations`, `targets`, `executors` only. There was
   nowhere to declare a fact, its type, or its domain.

## 4. Design

### 4.1 Facts are declared in two places, and the split is the whole rule

**Vocabulary owns identity and type.** What kind of thing this noun is.

**Policy owns every number that constitutes a decision.** Threshold, bound,
default.

```
Any *number* that, if edited, changes which requests are denied lives in the
policy file, because the policy file is what the PolicyHash covers.
```

The word "number" is load-bearing. A fact's `type` selects the parser and an
enum's `values` are baked into the generated frame, so both are decision-bearing
too — and both live in the vocabulary, under `VocabularyHash`. A consumer
recording only `PolicyHash` therefore sees every threshold change and no type
change. That is the intended trade: types change rarely and loudly (a wrong type
fails generation or fails every parse closed), thresholds change quietly and
often. A consumer that wants one identifier for "the ruleset I ran" should
record both hashes; `PolicyMetadata` exposes them separately.

`hash()` is `sha256` over the raw policy bytes (`generator.go:434-461`), so this
is automatic: put the number in `policy-rules.yaml` and it is in the
`PolicyHash`. Nothing in `DefaultConfig` gains a security value; runtime
override stays limited to what it overrides today (trusted users, default
executor, operation targets, executor fallbacks — `render.go:230-243`).

`VocabularyHash` is also in `PolicyMetadata` but consumers frequently record
only `PolicyHash` (gold does, `gate.go:104`). Keeping type in the vocabulary and
values in the policy means a consumer that records one hash still records every
threshold.

#### Vocabulary — new `facts:` block

```yaml
facts:
  account_mode:
    type: enum
    values: [demo, live]
    description: Whether the account is a demo or a funded account.
    go_name: AccountMode
  risk_pct:
    type: number
    go_name: RiskPct
  operable:
    type: bool
    go_name: Operable
  orders_last_hour:
    type: integer
    go_name: OrdersLastHour
```

`type` is one of `number | integer | bool | enum | string`. `values` is
required for `enum` and forbidden otherwise. `go_name` feeds `vocabgen`, which
emits `FactRiskPct = "risk_pct"` so a consumer building `Request.Metadata` gets
a compile error on a misspelled key instead of a missing fact at runtime.

The block is optional; a vocabulary without it behaves exactly as today.

#### Policy — new `facts:` block

```yaml
facts:
  risk_pct:      { min: 0 }
  min_score:     { min: 0, default: 50 }
```

`min`, `max`, `default`. All optional. A fact not listed here is required to be
present in `Request.Metadata` and is only range-checked against its declared
type. Declaring a fact here that the vocabulary does not declare is an error.

`default` is deliberately not a convenience. An absent fact with a default is a
policy statement that the absence means that value, and it is hashed as such.

### 4.2 Predicates: a closed tagged form, not an expression language

No expression engine, no parser. The YAML decoder is the parser and the leaf set
is closed:

```yaml
when:
  any:
    - { fact: risk_pct, op: gt, value: 1.5 }
    - { fact: exposure_pct, op: gt, value: 5.0 }
```

**Leaf forms**

| Form | Meaning |
| --- | --- |
| `{ fact: F, op: eq\|ne, value: V }` | scalar equality; `V` typed by `F` |
| `{ fact: F, op: gt\|gte\|lt\|lte, value: N }` | numeric comparison against a literal |
| `{ fact: F, op: gt\|gte\|lt\|lte, fact_value: G }` | numeric comparison against another fact |
| `{ fact: F, op: is_true\|is_false }` | boolean fact |
| `{ fact: F, op: in, values: [...] }` | membership |
| `{ condition: NAME }` | bridge to the existing named-condition seam |

**Composition:** `all: [...]`, `any: [...]`, `not: {...}`, nestable. Exactly
one of `all`/`any`/`not`/leaf per node — enforced by JSON Schema `oneOf`, so a
malformed predicate is a schema failure, not a runtime surprise.

No arithmetic. No string functions. No regex. No user-supplied comparators. The
grammar is deliberately too small to express anything but a guard, which is the
security argument for not reaching for an expression engine: there is nothing to
sandbox because there is nothing to evaluate.

`{ condition: NAME }` is what keeps this additive. Ontology guards, process
state, and anything else a consumer wants to answer in Go stay reachable, and
`requires: [a, b]` remains exactly equivalent to `when: {all: [{condition: a},
{condition: b}]}`.

### 4.3 Denies become ordered, and may be operation-wide

Two schema changes to `deny`:

- `operation` becomes optional. Absent means every operation in the vocabulary.
  Safe only in combination with the §2.2 fix: with deny operations validated
  against the vocabulary, an absent operation is a deliberate authoring choice
  and a typo is a build failure.
- `when` is added.

And one generator change, forced by §2.1: **`denies` is emitted as an ordered
slice, evaluated in authored order, first match wins.** The `id` is what the
audit needs; `reason` stays free text for humans. When `reason` is absent it
defaults to `id`, matching what routes already do (`render.go:175-178`).

Routes stay a map keyed by operation, but duplicate route operations are now
rejected at `check` time instead of emitting Go that will not compile.

### 4.4 What the generator emits

Four pieces, all plain readable Go, no interpreter at runtime.

**A fact frame**, built once per decision, over exactly the facts the policy
references:

```go
type factFrame struct {
	accountMode    string
	riskPct        float64
	exposurePct    float64
	operable       bool
	ordersLastHour int64
}

func newFactFrame(req cervorules.Request) (factFrame, error) {
	var f factFrame
	var err error
	if f.accountMode, err = factEnum(req, "account_mode", []string{"demo", "live"}); err != nil {
		return factFrame{}, err
	}
	if f.riskPct, err = factNumber(req, "risk_pct", factBound{min: 0, hasMin: true}); err != nil {
		return factFrame{}, err
	}
	// ...
	return f, nil
}
```

Eager, not lazy: every referenced fact is parsed before any rule is evaluated,
so the decision is a pure function of a validated frame and does not depend on
which rule happens to be first. The cost is parsing facts a short-circuiting
evaluation would have skipped; the benefit is that a malformed fact cannot hide
behind an earlier rule.

**A fail-closed parse helper**, emitted once:

```go
func factNumber(req cervorules.Request, name string, b factBound) (float64, error) {
	raw, ok := req.Metadata[name]
	if !ok || strings.TrimSpace(raw) == "" {
		if b.hasDefault {
			return b.def, nil
		}
		return 0, cervorules.Error{Code: cervorules.ErrorCodeMissingFact, ...}
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, cervorules.Error{Code: cervorules.ErrorCodeInvalidFact, ...}
	}
	// ParseFloat accepts "NaN" and "Inf". A non-finite value passes every
	// comparison below without matching any of them, which is exactly the
	// silent allow this layer exists to prevent.
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, cervorules.Error{Code: cervorules.ErrorCodeInvalidFact, ...}
	}
	if b.hasMin && v < b.min { ... }
	if b.hasMax && v > b.max { ... }
	return v, nil
}
```

Missing, unparseable, non-finite, out of domain, or unknown — every one returns
a structured error, never `false`. This is `conditionsHold`'s existing contract
(`render.go:309-315`) extended to facts.

**One matcher per rule**, returning the leaf that decided it:

```go
// deny-risk-or-exposure
func (f factFrame) matchDenyRiskOrExposure() (bool, string) {
	if f.riskPct > 1.5 {
		return true, "risk_pct gt 1.5"
	}
	if f.exposurePct > 5.0 {
		return true, "exposure_pct gt 5.0"
	}
	return false, ""
}
```

`any` becomes sequential early-return-true, `all` sequential
early-return-false, `not` an inversion, nesting a nested call. The returned
string is the explain payload and costs nothing when the rule does not match.

**An ordered deny loop** in `DecideWithOptions`, replacing the map lookup at
`render.go:281-287`.

### 4.5 Explain

`DecisionTrace` exists and is opt-in per decision through `DecisionOptions`;
today the generated engine allocates the envelope and appends nothing. This
design populates it: one `DecisionTraceStep` per evaluated rule, with `Name` =
rule id, `Matched`, and `Reason` = the deciding leaf from §4.4.

`Decision.Reason` keeps its current meaning — authored text, or the id when no
text was authored. A caller that needs to know *which* predicate denied reads
the trace. That is a per-decision opt-in, which is the right cost model: one
trade decision per order can afford it, a million routing decisions per second
cannot.

Structured beats formatted here. Today gold produces
`"risk_pct 3.0000 or exposure_pct 2.0000 exceeds limits (1.5 / 5.0)"` — one
string a human reads and no machine can query. A trace step naming the rule, the
fact and the bound is queryable.

### 4.6 New error codes

`ErrorCodeMissingFact = "missing_fact"` and `ErrorCodeInvalidFact =
"invalid_fact"`, in `core/errors.go` beside the conditions-seam codes. Per ADR
0014's ownership rule, codes belonging to a core-owned contract stay in core,
and the fact frame reads `core.Request.Metadata`, which core owns.

## 5. The gold-executor policy, written out in full

The acceptance case. Seven fact rules including two disjunctions, plus the
ALLOW.

### 5.1 `policy-vocabulary.yaml` (additions)

```yaml
facts:
  account_mode:     { type: enum, values: [demo, live], go_name: AccountMode }
  risk_pct:         { type: number,  go_name: RiskPct }
  exposure_pct:     { type: number,  go_name: ExposurePct }
  daily_loss_pct:   { type: number,  go_name: DailyLossPct }
  score:            { type: number,  go_name: Score }
  min_score:        { type: number,  go_name: MinScore }
  operable:         { type: bool,    go_name: Operable }
  event_window:     { type: bool,    go_name: EventWindow }
  orders_last_hour: { type: integer, go_name: OrdersLastHour }
  killswitch:       { type: bool,    go_name: Killswitch }
```

`killswitch` is a fact here, but `gate.Facts` still has no killswitch field — the
daemon injects it into `Metadata` (`gate.go:157,287`). The property that a
client cannot turn it off by sending JSON is preserved: there is still nowhere
to deserialize it into.

`style` is not declared. No rule reads it, and an undeclared fact in `Metadata`
is ignored, not rejected.

### 5.2 `policy-rules.yaml`

```yaml
version: cervorules.policy.v3
name: gold-executor.v1

defaults:
  executor: manual-confirm

facts:
  risk_pct:         { min: 0 }
  exposure_pct:     { min: 0 }
  daily_loss_pct:   { min: 0 }
  score:            { min: 0 }
  min_score:        { min: 0, default: 50 }
  orders_last_hour: { min: 0 }

# Ordered. First match wins. No `operation` means every operation, which is
# what the current gate does: rules 1-7 run before the route lookup for
# order.* and for killswitch.check alike.
denies:
  - id: deny-non-demo
    reason: account_mode must be demo (live blocked until sign-off)
    when: { fact: account_mode, op: ne, value: demo }

  - id: deny-killswitch
    reason: killswitch is active
    when: { fact: killswitch, op: is_true }

  - id: deny-daily-loss
    reason: daily_loss_pct exceeds its limit
    when: { fact: daily_loss_pct, op: gt, value: 2.0 }

  - id: deny-risk-or-exposure
    reason: risk_pct or exposure_pct exceeds its limit
    when:
      any:
        - { fact: risk_pct,     op: gt, value: 1.5 }
        - { fact: exposure_pct, op: gt, value: 5.0 }

  - id: deny-event-window
    reason: event_window is active (EIA/NFP)
    when: { fact: event_window, op: is_true }

  - id: deny-not-operable-or-score
    reason: instrument not operable, or score below its profile minimum
    when:
      any:
        - { fact: operable, op: is_false }
        - { fact: score, op: lt, fact_value: min_score }

  - id: deny-rate-limit
    reason: orders_last_hour reached its limit
    when: { fact: orders_last_hour, op: gte, value: 6 }

routes:
  - id: allow-order-place-manual
    operation: order.place
    target: mt4_demo
    executor: manual-confirm
  - id: allow-order-close-manual
    operation: order.close
    target: mt4_demo
    executor: manual-confirm
  - id: allow-order-modify-manual
    operation: order.modify
    target: mt4_demo
    executor: manual-confirm
  - id: allow-killswitch-check
    operation: killswitch.check
    target: mt4_demo
    executor: manual-confirm
```

Every threshold in the table below is now inside the bytes the `PolicyHash`
covers. Editing `1.5` to `15.0` moves the hash.

| Rule | Predicate | Boundary fixed by gold's tests |
| --- | --- | --- |
| deny-daily-loss | `daily_loss_pct > 2.0` | `2.0` exactly is allowed |
| deny-risk-or-exposure | `risk_pct > 1.5 OR exposure_pct > 5.0` | `1.5` exactly is allowed |
| deny-rate-limit | `orders_last_hour >= 6` | `6` exactly is denied |

### 5.3 What is left in `internal/gate/gate.go`

Input sanitisation and error mapping.

Not a shorter file — 293 lines against 305. `evaluateDenies` and `validateFacts`
came out, and `classifyPolicyError` and `deniedRule` went in to translate a
structured policy error and a trace step into the audit shape. What changed is
not the size but the content: the file no longer holds a single threshold or a
single comparison. The five constants `MaxDailyLossPct`, `MaxRiskPct`,
`MaxExposurePct`, `MaxOrdersPerHour` and `DefaultMinScore` are gone from Go and
exist only in YAML, where the hash covers them.

- reject an empty operation (`deny-malformed`);
- `vocab.ValidateOperation` (`deny-unknown-operation`);
- inject `killswitch` into `Metadata` from daemon state;
- serialise facts to `Metadata` using the generated `policyvocab.Fact*`
  constants, omitting `min_score` when it is `<= 0` so the policy's
  `default: 50` applies — that is where gold's "0 means unset" coercion goes;
- call `DecideWithOptions` and map the outcome:
  - `ErrorCodeMissingFact` / `ErrorCodeInvalidFact` → `deny-invalid-facts`;
  - any other error → `deny-malformed`;
  - `Allow == false` → rule id from the trace (or `Decision.Reason`);
  - `Executor == auto` → `deny-malformed` (the existing belt-and-braces check).

`evaluateDenies` and `validateFacts` are deleted. The five constants
`MaxDailyLossPct`, `MaxRiskPct`, `MaxExposurePct`, `MaxOrdersPerHour`,
`DefaultMinScore` are deleted from Go and exist only in YAML.

### 5.4 Test impact

43 assertions across five packages. The rule-id contract has three consumers,
not one: `internal/gate/gate_test.go`, `internal/gate/facts_test.go`, and
`internal/server/server_test.go:69,203`.

Expectations that must hold unchanged:

- the nine `RuleDeny*` identifiers asserted in `TestDenyCases`;
- `RuleDenyInvalidFacts` for NaN, +Inf and negative values
  (`facts_test.go:32-61`) — now produced by `factNumber`, not by hand;
- the three exact boundaries in `TestLimitesExactosSonLosDelPlan`;
- `RuleDenyKillswitch` when the daemon flag is set and the client did not send
  it (`facts_test.go:87-98`);
- `server_test.go` asserting `RuleDenyRiskOrExposure` and `RuleDenyKillswitch`
  end-to-end over HTTP.

The one thing that visibly changes is the `reason` *text* in the HTTP response
and the audit record. It used to interpolate the observed values
(`"risk_pct 3.0000 ... exceeds limits (1.5 / 5.0)"`); it now carries the
authored reason plus the deciding leaf from the trace
(`"risk_pct or exposure_pct exceeds its limit (exposure_pct gt 5)"`). No test
asserts that text — verified by grep across all five test packages — but it is a
wire-visible change and belongs in gold's changelog.

Outcome: 47 assertions pass, none fail. Per package: authn 4, gate 18,
policyrules 9, server 13, store 3. `gate`, `server`, `authn` and `store` are
unchanged in count; `policyrules` grew from 5 to 9 because the declarative
`tests:` block in the policy now covers the thresholds and both disjunctions.

Two test edits were needed, neither of which changes an expectation:

- `facts_test.go` compared against `gate.MaxRiskPct` and friends, which were
  constants in the same package. That was a tautology — it could not fail
  whatever the constant held. With the thresholds in YAML the test now writes
  `1.5`, `6` and `2.0` literally, which pins the plan's boundary for real.
- `SaneDemoFacts` used `DefaultMinScore`; it now writes `50`.

## 6. Phases

Each phase is independently shippable. **Phase B alone passes the gold
acceptance test**; C is what makes the result pleasant to read and audit. All
three landed together.

### Phase A — ordered denies (no new predicate syntax)

Fixes §2.1 and §2.2. `denies` becomes an ordered slice in the generated engine,
`operation` becomes optional, deny operations are validated against the
vocabulary, duplicate route operations are rejected at `check`, `reason`
defaults to `id`.

Nothing new in the DSL beyond an optional field. Valuable on its own: it turns
a miscompile and a fail-open into build failures. Prerequisite for everything
else — without it, "split the disjunction into two denies" does not build.

### Phase B — typed facts and leaf predicates

Vocabulary `facts:`, policy `facts:`, `when:` restricted to a single leaf, the
generated frame, the fail-closed parse helpers, the two new error codes,
`vocabgen` fact constants.

With A, gold's rules 1, 3, 5, 7 map one-to-one, and 4 and 6 each split into two
denies sharing an `id` — nine denies, same nine rule identifiers, every
threshold in the `PolicyHash`, `gate.go` reduced to sanitisation. **This is the
minimum that satisfies the acceptance criterion.**

### Phase C — composition and explain

`all` / `any` / `not`, nestable. Populated `DecisionTrace` steps carrying rule
id and deciding leaf.

Collapses gold's nine denies back to the seven rules the business actually has,
and makes a compound denial auditable: which of the two disjuncts fired, and
what the observed value was.

With C in place, gold's audit record for a compound denial reads:

```json
{"allowed":false,"rule":"deny-risk-or-exposure",
 "reason":"risk_pct or exposure_pct exceeds its limit (exposure_pct gt 5)"}
```

The rule id and the deciding disjunct both come from the trace. Before, the same
denial said only that the policy refused.

## 7. What breaks

Additive, no change required:

- Vocabulary `facts:` — new optional block.
- Policy `facts:`, `when:` — new optional fields.
- `conditions` + `requires` — unchanged semantics. `requires: [a, b]` is exactly
  `when: {all: [{condition: a}, {condition: b}]}`.
- `core.Request` — unchanged. Facts travel in `Metadata` as today.
- `core.Conditions`, `PolicyRuntimeConfig` — unchanged.
- Public generated API (`PolicyFactory`, `Metadata`, `DefaultConfig`,
  `ValidateConfig`, `Build`) — unchanged. The deny map→slice change is in
  unexported generated internals.

Deliberate breaks, each closing a fail-open:

| Change | Who it breaks | Why |
| --- | --- | --- |
| Deny operations validated against the vocabulary | A policy with a typo'd deny operation, which previously generated and never fired | §2.2 |
| Duplicate route operations rejected at `check` | A policy that previously produced Go the consumer could not compile | §2.1 |
| `deny.reason` defaults to `id` | A deny that omits `reason`, which previously yielded `Decision.Reason == ""` | Empty audit reason |
| `mergeRuntimeConfig` carries `Conditions` | Nobody: no condition-gated policy could be built before | §2.3 |

The first three need a CHANGELOG entry under Changed, not Added. The fourth is a
fix.

The repository's own fixtures were affected by the second and third rows:
`sampleVocab` in `internal/policygen/generator_test.go` denied
`unknown_operation` without declaring it, which is precisely the shape §2.2 now
rejects.

Not addressed, and needing its own decision: `PolicyMetadata` exposes
`VocabularyHash` and `PolicyHash` separately and consumers commonly record only
the latter. This design keeps every decision-bearing number under `PolicyHash`
so that is sufficient, but a consumer wanting one identifier for "the ruleset I
ran" has to combine them itself.

## 8. Answers to the open questions

**Predicates over `Request.Metadata`, or a separate typed channel?**
Over `Metadata`, with the type declared in the vocabulary and the parse
generated. A second channel means a second serialisation for the consumer and a
new public field on `core.Request`; the string boundary is honest, because the
request arrived over HTTP as strings anyway. What matters is not where the
string lives but that exactly one place converts it, that the conversion is
generated from a declaration, and that every failure mode is an error rather
than `false`.

**Is `facts/` the natural base?**
No. ADR 0014 already answered the analogous question for `ontology`: "`facts`
owns derivation of new facts; this layer owns refutation of an incoherent
world. Merging them would put constraint checking on the derivation path." The
same reasoning applies with more force here, because a decision guard is on the
hot path and `facts` is a bounded datalog engine with iteration budgets,
string-typed `Term`s and an `EvaluationPlan`. A consumer that genuinely needs
derived facts derives them first and puts the results in `Metadata`; the seam
already exists and needs nothing new.

**Do the v2 `limits` come back in the same work?**
No, and on reflection the answer is stronger than "separate issue".

`limits.Budget` is `MaxTokens`, `AllowStream`, `AllowTools`, `AllowImages`,
`MaxBodyBytes`. Four of those five are LLM gateway vocabulary, and
`AGENTS.md` states the rule directly: "Do not add CervoProxy, gateway, AI,
provider-specific payload, or tenant concepts to core APIs", and "Keep
model/profile selection outside CervoRules."

The placement distinction is what matters. `limits/` as an optional leaf package
is defensible on the same grounds as `ontology`: importing it is the only way to
pay for it. Putting `max_tokens` in the **DSL** is not, because the DSL is the
shared surface — every policy author sees it, the published schema carries it,
and `policygen` has to know about it. That is precisely "making one consumer the
shape of the core API" from `docs/agnosticism.md`.

And the general case is already covered, domain-neutrally, by this change. A
consumer that wants to refuse an oversized request declares the fact and writes
the rule:

```yaml
facts:
  requested_tokens: { type: integer }
```
```yaml
denies:
  - id: deny-token-budget
    when: { fact: requested_tokens, op: gt, value: 4000 }
```

No AI vocabulary enters the shared contract; the consumer names the fact and
picks the number. The one thing this does not reproduce is `limits.Violations`
reporting several breaches at once after a route is chosen, where a deny is
first-match. That is a difference in output shape, resolvable in the consumer.

So this is not deferred work waiting for a slot. It is a decision that the v2
shape should not return to the DSL.

**Is `explain` part of this or later?**
Phase C of this, not a separate issue. It is the difference between "the policy
denied" and "rule 4 denied because exposure_pct was 6.0 against a limit of 5.0",
and a compound predicate without it is harder to audit than the hand-written Go
it replaces. Phase A's `reason` defaults to `id` so audit is not blocked while C
is in flight.

## 9. Non-negotiables, and how each is met

| Constraint | How |
| --- | --- |
| Additive — current v3 policies keep working untouched | Every new field is optional; `requires` keeps its list form and its meaning; public API unchanged. The three breaks in §7 each close a fail-open and are listed explicitly. |
| Fail-closed with no exceptions | Missing fact, wrong type, non-finite value, out-of-domain value, unknown condition, absent evaluator — all return a structured `core.Error`. `false` is only ever returned by a predicate that genuinely evaluated. |
| Security thresholds inside the `PolicyHash` | Every threshold, bound and default is in `policy-rules.yaml`, over whose raw bytes the hash is taken. `DefaultConfig` gains no security value. |
| No third-party dependency for predicate evaluation | There is no evaluator. Predicates compile to Go boolean expressions in the generated file. The grammar is a closed tagged form validated by JSON Schema — too small to express anything but a guard, which is the point. |
| Generator keeps emitting readable Go with generated tests | Frame, helpers and one matcher per rule are ordinary readable Go. Declarative `tests:` gain `metadata:` coverage for fact values, so a threshold change is caught by a generated test. |
