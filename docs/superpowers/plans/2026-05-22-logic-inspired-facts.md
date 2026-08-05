# Logic-Inspired Facts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an optional Datalog-inspired facts layer to CervoRules so policies can use derived, explainable facts without turning CervoRules into Prolog.

**Architecture:** Keep the existing deterministic policy runtime intact. Add a new optional public module for relational facts, derived fact rules, bounded evaluation, and derivation traces, then bridge it into core policies through explicit adapters and generated policy support. Avoid Prolog features that make production behavior hard to bound: free recursion, cut, general term unification, procedural predicates, and automatic multi-solution backtracking.

**Tech Stack:** Go 1.25.8, CervoRules v2 modules, table-driven TDD, generated policy examples, issue/PR tracking, existing release/package checks.

---

## Context

CervoRules currently behaves as an operational policy engine:

- callers provide `Request`, `RequestFacts`, metadata, runtime config, service health, and vocabulary;
- policies evaluate predicates and actions deterministically;
- `DecisionResult` returns `Decision`, `DecisionTrace`, `DecisionObservation`, and diagnostics;
- optional adapters such as `httpadapter` map transports into neutral facts.

Prolog offers useful ideas, but a full Prolog engine is not the right target for CervoRules. The useful subset is closer to Datalog:

- facts as typed tuples;
- rules that derive more facts from existing facts;
- bounded, deterministic evaluation;
- explainable derivation trees;
- query helpers for tests, adapters, and policies.

The design must preserve these CervoRules principles:

- core runtime remains dependency-light;
- new functionality is optional and modular;
- no consumer-specific vocabulary enters core;
- generated policies remain reproducible;
- production behavior is bounded and observable.

## Non-Goals

- Do not implement a Prolog parser.
- Do not support general unification of nested terms.
- Do not support `cut`, assert/retract, dynamic procedural predicates, or side effects inside logic rules.
- Do not support infinite or unbounded recursion.
- Do not make the core engine depend on the facts module unless the integration is a small interface.
- Do not replace existing predicates/actions in v2.0.x.

## Proposed Public Shape

The initial module should be optional:

```text
github.com/cervantesh/cervo-rules/v2/facts
```

The package should expose a small API:

```go
type Atom string

type Term struct {
    Kind  TermKind
    Value string
}

type Fact struct {
    Predicate Atom
    Terms     []Term
}

type Set struct {
    // immutable or copy-on-write public behavior
}

type Variable string

type Pattern struct {
    Predicate Atom
    Terms     []Term
}

type Rule struct {
    Name string
    Head Pattern
    Body []Pattern
}

type Engine struct {
    // bounded evaluator
}

type EvalOptions struct {
    MaxIterations int
    MaxFacts      int
}

type Result struct {
    Facts       Set
    Derivations DerivationTrace
    Diagnostics []Diagnostic
}
```

Convenience constructors:

```go
func Const(value string) Term
func Var(name string) Term
func NewFact(predicate Atom, terms ...Term) Fact
func NewSet(facts ...Fact) Set
func NewRule(name string, head Pattern, body ...Pattern) Rule
func NewEngine(rules ...Rule) Engine
func (e Engine) Evaluate(ctx context.Context, input Set, options EvalOptions) (Result, error)
func (s Set) Has(predicate Atom, values ...string) bool
func (s Set) Query(pattern Pattern) []Binding
```

The API should be intentionally simple. More expressive operators can come later only after an ADR.

## File Structure

Create:

- `facts/doc.go`: package purpose, non-goals, and production constraints.
- `facts/term.go`: `Atom`, `Term`, `TermKind`, `Const`, `Var`.
- `facts/fact.go`: `Fact`, `Pattern`, normalization, stable string formatting.
- `facts/set.go`: immutable/copying fact set with `Add`, `Has`, `Query`.
- `facts/binding.go`: variable binding and pattern matching.
- `facts/rule.go`: `Rule`, validation, constructor.
- `facts/engine.go`: bounded fixed-point evaluator.
- `facts/trace.go`: derivation trace and explanation structures.
- `facts/errors.go`: structured diagnostics and evaluation errors.
- `facts/*_test.go`: TDD coverage for each unit.
- `docs/logic-inspired-facts.md`: user-facing design and examples.
- `docs/adr/0005-logic-inspired-facts.md`: decision record.
- `examples/logic-facts/`: neutral generated or manual example.

Modify:

- `README.md`: add optional facts module section.
- `docs/modular-architecture.md`: add `facts` module ownership and non-ownership.
- `docs/agnosticism.md`: explain why this improves agnosticism without core domain leakage.
- `docs/observability.md`: describe derivation trace relationship to `DecisionTrace`.
- `docs/release.md`: add release checklist item for facts examples.
- `docs/change-management/v2-public-api.md`: add this as v2.x optional feature work, not a v2.0 blocker.

Do not modify in the first PR:

- `core/decision_flow.go`
- `internal/policygen/*`
- `cmd/cervorules-policygen/*`
- CervoProxy

## Issue Breakdown

Create one umbrella issue:

```text
epic(facts): add Datalog-inspired derived facts without becoming Prolog
```

Create child issues:

1. `facts: add immutable fact set and pattern query API`
2. `facts: add bounded derived fact evaluator`
3. `facts: add derivation trace and explain output`
4. `docs: document logic-inspired facts and Prolog boundary`
5. `examples: add neutral derived facts example`
6. `policygen: design generated policy integration for derived facts`
7. `core: design optional predicate bridge for derived facts`

The first five can ship without changing core policy behavior. Issues 6 and 7 require a second ADR/checkpoint after the base facts API is proven.

## Phase 1: Facts Data Model

### Task 1: Create Package Skeleton

**Files:**
- Create: `facts/doc.go`
- Create: `facts/term.go`
- Create: `facts/fact.go`
- Create: `facts/fact_test.go`

- [ ] **Step 1: Write failing constructor and formatting tests**

Add `facts/fact_test.go`:

```go
package facts

import "testing"

func TestFactConstructorsNormalizeTerms(t *testing.T) {
    fact := NewFact("user_role", Const("alice"), Const("admin"))

    if fact.Predicate != Atom("user_role") {
        t.Fatalf("predicate mismatch: %q", fact.Predicate)
    }
    if len(fact.Terms) != 2 {
        t.Fatalf("terms length mismatch: %d", len(fact.Terms))
    }
    if fact.Terms[0].Value != "alice" || fact.Terms[1].Value != "admin" {
        t.Fatalf("terms mismatch: %+v", fact.Terms)
    }
}

func TestFactStringIsStable(t *testing.T) {
    fact := NewFact("tenant_plan", Const("acme"), Const("pro"))
    if got, want := fact.String(), `tenant_plan("acme","pro")`; got != want {
        t.Fatalf("String() = %q, want %q", got, want)
    }
}

func TestPatternAllowsVariables(t *testing.T) {
    pattern := NewPattern("user_role", Var("user"), Const("admin"))

    if pattern.Terms[0].Kind != TermVariable {
        t.Fatalf("first term kind = %v, want variable", pattern.Terms[0].Kind)
    }
    if pattern.Terms[1].Kind != TermConstant {
        t.Fatalf("second term kind = %v, want constant", pattern.Terms[1].Kind)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./facts -run TestFact -count=1
```

Expected: compile fails because `facts` package symbols do not exist.

- [ ] **Step 3: Implement minimal data model**

Add `facts/doc.go`:

```go
// Package facts provides a small Datalog-inspired fact and derivation layer.
//
// The package is intentionally not a Prolog runtime. It supports flat facts,
// constants, variables in query patterns, bounded rule evaluation, and
// derivation traces for production policy decisions.
package facts
```

Add `facts/term.go`:

```go
package facts

type Atom string

type TermKind string

const (
    TermConstant TermKind = "constant"
    TermVariable TermKind = "variable"
)

type Term struct {
    Kind  TermKind
    Value string
}

func Const(value string) Term {
    return Term{Kind: TermConstant, Value: value}
}

func Var(name string) Term {
    return Term{Kind: TermVariable, Value: name}
}
```

Add `facts/fact.go`:

```go
package facts

import (
    "strconv"
    "strings"
)

type Fact struct {
    Predicate Atom
    Terms     []Term
}

type Pattern struct {
    Predicate Atom
    Terms     []Term
}

func NewFact(predicate Atom, terms ...Term) Fact {
    copied := append([]Term(nil), terms...)
    return Fact{Predicate: predicate, Terms: copied}
}

func NewPattern(predicate Atom, terms ...Term) Pattern {
    copied := append([]Term(nil), terms...)
    return Pattern{Predicate: predicate, Terms: copied}
}

func (f Fact) String() string {
    parts := make([]string, len(f.Terms))
    for i, term := range f.Terms {
        parts[i] = strconv.Quote(term.Value)
    }
    return string(f.Predicate) + "(" + strings.Join(parts, ",") + ")"
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./facts -run TestFact -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add facts/doc.go facts/term.go facts/fact.go facts/fact_test.go
git commit -m "feat(facts): add fact and pattern primitives"
```

## Phase 2: Immutable Fact Set And Queries

### Task 2: Add Set, Binding, And Query

**Files:**
- Create: `facts/set.go`
- Create: `facts/binding.go`
- Create: `facts/set_test.go`

- [ ] **Step 1: Write failing set/query tests**

Add `facts/set_test.go`:

```go
package facts

import "testing"

func TestSetHasFact(t *testing.T) {
    set := NewSet(
        NewFact("user_role", Const("alice"), Const("admin")),
        NewFact("user_role", Const("bob"), Const("reader")),
    )

    if !set.Has("user_role", "alice", "admin") {
        t.Fatalf("expected alice admin fact")
    }
    if set.Has("user_role", "alice", "reader") {
        t.Fatalf("did not expect alice reader fact")
    }
}

func TestSetQueryBindsVariables(t *testing.T) {
    set := NewSet(
        NewFact("user_role", Const("alice"), Const("admin")),
        NewFact("user_role", Const("bob"), Const("reader")),
    )

    bindings := set.Query(NewPattern("user_role", Var("user"), Const("admin")))
    if len(bindings) != 1 {
        t.Fatalf("bindings length = %d, want 1", len(bindings))
    }
    if got := bindings[0].Get("user"); got != "alice" {
        t.Fatalf("binding user = %q, want alice", got)
    }
}

func TestSetAddDoesNotMutateOriginal(t *testing.T) {
    original := NewSet(NewFact("tenant_plan", Const("acme"), Const("free")))
    next := original.Add(NewFact("tenant_plan", Const("globex"), Const("pro")))

    if original.Has("tenant_plan", "globex", "pro") {
        t.Fatalf("original set was mutated")
    }
    if !next.Has("tenant_plan", "globex", "pro") {
        t.Fatalf("next set missing added fact")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./facts -run TestSet -count=1
```

Expected: compile fails because `Set`, `Binding`, `NewSet`, `Has`, and `Query` do not exist.

- [ ] **Step 3: Implement set and binding**

Add `facts/binding.go`:

```go
package facts

type Binding map[string]string

func (b Binding) Get(name string) string {
    return b[name]
}

func (b Binding) clone() Binding {
    copied := make(Binding, len(b))
    for key, value := range b {
        copied[key] = value
    }
    return copied
}
```

Add `facts/set.go`:

```go
package facts

type Set struct {
    facts []Fact
}

func NewSet(facts ...Fact) Set {
    copied := append([]Fact(nil), facts...)
    return Set{facts: copied}
}

func (s Set) Add(fact Fact) Set {
    copied := append([]Fact(nil), s.facts...)
    copied = append(copied, fact)
    return Set{facts: copied}
}

func (s Set) Facts() []Fact {
    return append([]Fact(nil), s.facts...)
}

func (s Set) Has(predicate Atom, values ...string) bool {
    for _, fact := range s.facts {
        if fact.Predicate != predicate || len(fact.Terms) != len(values) {
            continue
        }
        matched := true
        for i, value := range values {
            if fact.Terms[i].Kind != TermConstant || fact.Terms[i].Value != value {
                matched = false
                break
            }
        }
        if matched {
            return true
        }
    }
    return false
}

func (s Set) Query(pattern Pattern) []Binding {
    var out []Binding
    for _, fact := range s.facts {
        if binding, ok := match(pattern, fact, nil); ok {
            out = append(out, binding)
        }
    }
    return out
}

func match(pattern Pattern, fact Fact, base Binding) (Binding, bool) {
    if pattern.Predicate != fact.Predicate || len(pattern.Terms) != len(fact.Terms) {
        return nil, false
    }
    binding := Binding{}
    if base != nil {
        binding = base.clone()
    }
    for i, term := range pattern.Terms {
        value := fact.Terms[i].Value
        switch term.Kind {
        case TermConstant:
            if term.Value != value {
                return nil, false
            }
        case TermVariable:
            if existing, ok := binding[term.Value]; ok && existing != value {
                return nil, false
            }
            binding[term.Value] = value
        default:
            return nil, false
        }
    }
    return binding, true
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```powershell
go test ./facts -run TestSet -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```powershell
git add facts/set.go facts/binding.go facts/set_test.go
git commit -m "feat(facts): add immutable fact set queries"
```

## Phase 3: Bounded Derived Fact Evaluator

### Task 3: Add Rule Evaluation

**Files:**
- Create: `facts/rule.go`
- Create: `facts/engine.go`
- Create: `facts/errors.go`
- Create: `facts/engine_test.go`

- [ ] **Step 1: Write failing evaluator tests**

Add `facts/engine_test.go`:

```go
package facts

import (
    "context"
    "testing"
)

func TestEngineDerivesFact(t *testing.T) {
    input := NewSet(NewFact("user_role", Const("alice"), Const("admin")))
    engine := NewEngine(NewRule(
        "admins_are_trusted",
        NewPattern("trusted_user", Var("user")),
        NewPattern("user_role", Var("user"), Const("admin")),
    ))

    result, err := engine.Evaluate(context.Background(), input, EvalOptions{MaxIterations: 4, MaxFacts: 16})
    if err != nil {
        t.Fatalf("Evaluate returned error: %v", err)
    }
    if !result.Facts.Has("trusted_user", "alice") {
        t.Fatalf("expected trusted_user alice to be derived")
    }
}

func TestEngineDerivesThroughMultipleBodyPatterns(t *testing.T) {
    input := NewSet(
        NewFact("tenant_plan", Const("acme"), Const("pro")),
        NewFact("request_tenant", Const("req-1"), Const("acme")),
    )
    engine := NewEngine(NewRule(
        "pro_tenant_request",
        NewPattern("premium_request", Var("request")),
        NewPattern("request_tenant", Var("request"), Var("tenant")),
        NewPattern("tenant_plan", Var("tenant"), Const("pro")),
    ))

    result, err := engine.Evaluate(context.Background(), input, EvalOptions{MaxIterations: 4, MaxFacts: 16})
    if err != nil {
        t.Fatalf("Evaluate returned error: %v", err)
    }
    if !result.Facts.Has("premium_request", "req-1") {
        t.Fatalf("expected premium_request req-1 to be derived")
    }
}

func TestEngineStopsAtMaxFacts(t *testing.T) {
    input := NewSet(NewFact("seed", Const("a")))
    engine := NewEngine(
        NewRule("derive_a", NewPattern("a", Const("1")), NewPattern("seed", Const("a"))),
        NewRule("derive_b", NewPattern("b", Const("1")), NewPattern("seed", Const("a"))),
    )

    _, err := engine.Evaluate(context.Background(), input, EvalOptions{MaxIterations: 4, MaxFacts: 2})
    if err == nil {
        t.Fatalf("expected max facts error")
    }
}

func TestEngineRespectsContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel()

    engine := NewEngine(NewRule("noop", NewPattern("x", Const("1")), NewPattern("y", Const("1"))))
    _, err := engine.Evaluate(ctx, NewSet(), EvalOptions{MaxIterations: 4, MaxFacts: 16})
    if err == nil {
        t.Fatalf("expected context cancellation error")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./facts -run TestEngine -count=1
```

Expected: compile fails because `Rule`, `Engine`, and `Evaluate` do not exist.

- [ ] **Step 3: Implement bounded evaluator**

Add `facts/errors.go`:

```go
package facts

import "fmt"

type Diagnostic struct {
    Code   string
    Rule   string
    Reason string
}

func (d Diagnostic) Error() string {
    if d.Rule == "" {
        return d.Code + ": " + d.Reason
    }
    return fmt.Sprintf("%s: rule %s: %s", d.Code, d.Rule, d.Reason)
}
```

Add `facts/rule.go`:

```go
package facts

type Rule struct {
    Name string
    Head Pattern
    Body []Pattern
}

func NewRule(name string, head Pattern, body ...Pattern) Rule {
    copied := append([]Pattern(nil), body...)
    return Rule{Name: name, Head: head, Body: copied}
}
```

Add `facts/engine.go`:

```go
package facts

import (
    "context"
    "errors"
)

type EvalOptions struct {
    MaxIterations int
    MaxFacts      int
}

type Engine struct {
    rules []Rule
}

type Result struct {
    Facts       Set
    Derivations DerivationTrace
    Diagnostics []Diagnostic
}

func NewEngine(rules ...Rule) Engine {
    copied := append([]Rule(nil), rules...)
    return Engine{rules: copied}
}

func (e Engine) Evaluate(ctx context.Context, input Set, options EvalOptions) (Result, error) {
    if err := ctx.Err(); err != nil {
        return Result{}, err
    }
    if options.MaxIterations <= 0 {
        options.MaxIterations = 8
    }
    if options.MaxFacts <= 0 {
        options.MaxFacts = 1024
    }

    current := input
    seen := map[string]struct{}{}
    for _, fact := range current.Facts() {
        seen[fact.String()] = struct{}{}
    }

    for iteration := 0; iteration < options.MaxIterations; iteration++ {
        if err := ctx.Err(); err != nil {
            return Result{}, err
        }

        changed := false
        for _, rule := range e.rules {
            bindings := satisfyBody(current, rule.Body)
            for _, binding := range bindings {
                fact, ok := instantiate(rule.Head, binding)
                if !ok {
                    continue
                }
                key := fact.String()
                if _, exists := seen[key]; exists {
                    continue
                }
                if len(seen)+1 > options.MaxFacts {
                    return Result{}, Diagnostic{Code: "facts.max_facts", Rule: rule.Name, Reason: "derived fact limit exceeded"}
                }
                current = current.Add(fact)
                seen[key] = struct{}{}
                changed = true
            }
        }
        if !changed {
            return Result{Facts: current}, nil
        }
    }

    return Result{}, errors.New("facts.max_iterations: fixed point was not reached")
}

func satisfyBody(set Set, body []Pattern) []Binding {
    bindings := []Binding{{}}
    for _, pattern := range body {
        var next []Binding
        for _, existing := range bindings {
            for _, fact := range set.Facts() {
                if binding, ok := match(pattern, fact, existing); ok {
                    next = append(next, binding)
                }
            }
        }
        bindings = next
    }
    return bindings
}

func instantiate(pattern Pattern, binding Binding) (Fact, bool) {
    terms := make([]Term, len(pattern.Terms))
    for i, term := range pattern.Terms {
        switch term.Kind {
        case TermConstant:
            terms[i] = term
        case TermVariable:
            value, ok := binding[term.Value]
            if !ok {
                return Fact{}, false
            }
            terms[i] = Const(value)
        default:
            return Fact{}, false
        }
    }
    return NewFact(pattern.Predicate, terms...), true
}
```

- [ ] **Step 4: Add temporary trace stub**

Add `facts/trace.go`:

```go
package facts

type DerivationTrace struct {
    Entries []DerivationEntry
}

type DerivationEntry struct {
    Rule string
    Fact Fact
}
```

- [ ] **Step 5: Run test to verify it passes**

Run:

```powershell
go test ./facts -run TestEngine -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add facts/rule.go facts/engine.go facts/errors.go facts/trace.go facts/engine_test.go
git commit -m "feat(facts): add bounded derived fact evaluator"
```

## Phase 4: Derivation Trace And Explain Tree

### Task 4: Record Why Facts Exist

**Files:**
- Modify: `facts/trace.go`
- Modify: `facts/engine.go`
- Create: `facts/trace_test.go`

- [ ] **Step 1: Write failing trace tests**

Add `facts/trace_test.go`:

```go
package facts

import (
    "context"
    "testing"
)

func TestDerivationTraceRecordsRuleAndInputs(t *testing.T) {
    inputFact := NewFact("user_role", Const("alice"), Const("admin"))
    engine := NewEngine(NewRule(
        "admins_are_trusted",
        NewPattern("trusted_user", Var("user")),
        NewPattern("user_role", Var("user"), Const("admin")),
    ))

    result, err := engine.Evaluate(context.Background(), NewSet(inputFact), EvalOptions{MaxIterations: 4, MaxFacts: 16})
    if err != nil {
        t.Fatalf("Evaluate returned error: %v", err)
    }

    entry, ok := result.Derivations.Find(NewFact("trusted_user", Const("alice")))
    if !ok {
        t.Fatalf("missing trace entry for trusted_user alice")
    }
    if entry.Rule != "admins_are_trusted" {
        t.Fatalf("rule = %q, want admins_are_trusted", entry.Rule)
    }
    if len(entry.Inputs) != 1 || entry.Inputs[0].String() != inputFact.String() {
        t.Fatalf("inputs = %+v, want %s", entry.Inputs, inputFact.String())
    }
}

func TestDerivationTraceExplainIsStable(t *testing.T) {
    trace := DerivationTrace{Entries: []DerivationEntry{{
        Rule:   "admins_are_trusted",
        Fact:   NewFact("trusted_user", Const("alice")),
        Inputs: []Fact{NewFact("user_role", Const("alice"), Const("admin"))},
    }}}

    got := trace.Explain(NewFact("trusted_user", Const("alice")))
    want := `trusted_user("alice") <= admins_are_trusted <= user_role("alice","admin")`
    if got != want {
        t.Fatalf("Explain() = %q, want %q", got, want)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./facts -run TestDerivationTrace -count=1
```

Expected: fails because `Find`, `Explain`, and trace inputs are not implemented.

- [ ] **Step 3: Implement derivation tracing**

Update `facts/trace.go`:

```go
package facts

import "strings"

type DerivationTrace struct {
    Entries []DerivationEntry
}

type DerivationEntry struct {
    Rule   string
    Fact   Fact
    Inputs []Fact
}

func (t DerivationTrace) Add(entry DerivationEntry) DerivationTrace {
    copied := append([]DerivationEntry(nil), t.Entries...)
    copied = append(copied, entry)
    return DerivationTrace{Entries: copied}
}

func (t DerivationTrace) Find(fact Fact) (DerivationEntry, bool) {
    key := fact.String()
    for _, entry := range t.Entries {
        if entry.Fact.String() == key {
            return entry, true
        }
    }
    return DerivationEntry{}, false
}

func (t DerivationTrace) Explain(fact Fact) string {
    entry, ok := t.Find(fact)
    if !ok {
        return fact.String()
    }
    inputs := make([]string, len(entry.Inputs))
    for i, input := range entry.Inputs {
        inputs[i] = input.String()
    }
    return fact.String() + " <= " + entry.Rule + " <= " + strings.Join(inputs, ", ")
}
```

Update `facts/engine.go` so `Evaluate` records matched input facts. Replace `satisfyBody` with a binding candidate type:

```go
type candidate struct {
    Binding Binding
    Inputs  []Fact
}

func satisfyBody(set Set, body []Pattern) []candidate {
    candidates := []candidate{{Binding: Binding{}}}
    for _, pattern := range body {
        var next []candidate
        for _, existing := range candidates {
            for _, fact := range set.Facts() {
                if binding, ok := match(pattern, fact, existing.Binding); ok {
                    inputs := append([]Fact(nil), existing.Inputs...)
                    inputs = append(inputs, fact)
                    next = append(next, candidate{Binding: binding, Inputs: inputs})
                }
            }
        }
        candidates = next
    }
    return candidates
}
```

Update the loop in `Evaluate`:

```go
trace := DerivationTrace{}
```

and when adding a fact:

```go
trace = trace.Add(DerivationEntry{Rule: rule.Name, Fact: fact, Inputs: candidate.Inputs})
```

Return:

```go
return Result{Facts: current, Derivations: trace}, nil
```

- [ ] **Step 4: Run trace tests**

Run:

```powershell
go test ./facts -run TestDerivationTrace -count=1
```

Expected: PASS.

- [ ] **Step 5: Run full facts tests**

Run:

```powershell
go test ./facts -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add facts/trace.go facts/engine.go facts/trace_test.go
git commit -m "feat(facts): trace derived fact explanations"
```

## Phase 5: Operational Guardrails

### Task 5: Validate Rules And Bound Evaluation

**Files:**
- Modify: `facts/rule.go`
- Modify: `facts/engine.go`
- Create: `facts/validation_test.go`

- [ ] **Step 1: Write failing validation tests**

Add `facts/validation_test.go`:

```go
package facts

import (
    "context"
    "testing"
)

func TestRuleRejectsUnboundHeadVariable(t *testing.T) {
    engine := NewEngine(NewRule(
        "bad_rule",
        NewPattern("trusted_user", Var("user")),
        NewPattern("tenant_plan", Var("tenant"), Const("pro")),
    ))

    _, err := engine.Evaluate(context.Background(), NewSet(NewFact("tenant_plan", Const("acme"), Const("pro"))), EvalOptions{})
    if err == nil {
        t.Fatalf("expected validation error")
    }
}

func TestRuleRejectsEmptyName(t *testing.T) {
    engine := NewEngine(NewRule(
        "",
        NewPattern("x", Const("1")),
        NewPattern("y", Const("1")),
    ))

    _, err := engine.Evaluate(context.Background(), NewSet(), EvalOptions{})
    if err == nil {
        t.Fatalf("expected validation error")
    }
}

func TestRuleRejectsEmptyPredicate(t *testing.T) {
    engine := NewEngine(NewRule(
        "empty_predicate",
        NewPattern("", Const("1")),
        NewPattern("y", Const("1")),
    ))

    _, err := engine.Evaluate(context.Background(), NewSet(), EvalOptions{})
    if err == nil {
        t.Fatalf("expected validation error")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
go test ./facts -run TestRuleRejects -count=1
```

Expected: tests fail because validation is missing.

- [ ] **Step 3: Implement validation**

Add to `facts/rule.go`:

```go
func (r Rule) Validate() error {
    if r.Name == "" {
        return Diagnostic{Code: "facts.rule.name_required", Reason: "rule name is required"}
    }
    if r.Head.Predicate == "" {
        return Diagnostic{Code: "facts.rule.head_predicate_required", Rule: r.Name, Reason: "head predicate is required"}
    }
    bodyVars := map[string]struct{}{}
    for _, pattern := range r.Body {
        if pattern.Predicate == "" {
            return Diagnostic{Code: "facts.rule.body_predicate_required", Rule: r.Name, Reason: "body predicate is required"}
        }
        for _, term := range pattern.Terms {
            if term.Kind == TermVariable {
                bodyVars[term.Value] = struct{}{}
            }
        }
    }
    for _, term := range r.Head.Terms {
        if term.Kind != TermVariable {
            continue
        }
        if _, ok := bodyVars[term.Value]; !ok {
            return Diagnostic{Code: "facts.rule.unbound_head_variable", Rule: r.Name, Reason: "head variable must appear in the rule body"}
        }
    }
    return nil
}
```

At the start of `Engine.Evaluate`, validate all rules:

```go
for _, rule := range e.rules {
    if err := rule.Validate(); err != nil {
        return Result{}, err
    }
}
```

- [ ] **Step 4: Run validation tests**

Run:

```powershell
go test ./facts -run TestRuleRejects -count=1
```

Expected: PASS.

- [ ] **Step 5: Run full tests and coverage**

Run:

```powershell
go test -count=1 ./facts
go test -cover ./facts
```

Expected: PASS and coverage >= 90%.

- [ ] **Step 6: Commit**

```powershell
git add facts/rule.go facts/engine.go facts/validation_test.go
git commit -m "feat(facts): validate derived fact rules"
```

## Phase 6: Documentation And ADR

### Task 6: Document Prolog Boundary And Datalog-Inspired Design

**Files:**
- Create: `docs/logic-inspired-facts.md`
- Create: `docs/adr/0005-logic-inspired-facts.md`
- Modify: `README.md`
- Modify: `docs/modular-architecture.md`
- Modify: `docs/agnosticism.md`

- [ ] **Step 1: Add design document**

Create `docs/logic-inspired-facts.md`:

```markdown
# Logic-Inspired Facts

CervoRules uses an optional Datalog-inspired facts layer for derived, explainable operational facts.

It is not Prolog. The goal is deterministic policy support, not general logic programming.

## Why

Some domains need relationships that should not become rigid `Request` fields:

- user-role relationships;
- tenant-plan relationships;
- device-capability relationships;
- document-classification relationships;
- service ownership relationships.

Derived facts let consumers declare these relationships once and let policies consume the result.

## Model

The evaluation pipeline is:

```text
input facts -> derived facts -> policy predicates -> decision -> observation
```

## Example

```go
input := facts.NewSet(
    facts.NewFact("user_role", facts.Const("alice"), facts.Const("admin")),
)

engine := facts.NewEngine(facts.NewRule(
    "admins_are_trusted",
    facts.NewPattern("trusted_user", facts.Var("user")),
    facts.NewPattern("user_role", facts.Var("user"), facts.Const("admin")),
))

result, err := engine.Evaluate(ctx, input, facts.EvalOptions{
    MaxIterations: 4,
    MaxFacts:      128,
})
```

## What This Borrows From Prolog

- facts;
- rules;
- variables in patterns;
- explanation of derived conclusions.

## What This Does Not Borrow From Prolog

- unbounded backtracking;
- general nested-term unification;
- cut;
- side-effecting predicates;
- dynamic assert/retract;
- a Prolog parser.

## Operational Guardrails

- rule evaluation is bounded by `MaxIterations` and `MaxFacts`;
- rules validate before evaluation;
- derived facts are traceable;
- integrations stay optional and adapter-owned.
```

- [ ] **Step 2: Add ADR**

Create `docs/adr/0005-logic-inspired-facts.md`:

```markdown
# ADR 0005: Add Datalog-Inspired Facts Without Becoming Prolog

## Status

Proposed

## Context

CervoRules needs more expressive cross-domain relationships without coupling core APIs to CervoProxy, HTTP, providers, or AI payloads. Prolog is powerful, but a full Prolog runtime would make evaluation harder to bound, test, package, and explain in production.

## Decision

Add an optional `facts` module inspired by Datalog:

- flat facts;
- constants and variables;
- deterministic rule evaluation;
- bounded fixed-point derivation;
- derivation traces;
- query helpers.

Do not implement full Prolog semantics.

## Consequences

Positive:

- more expressive neutral policies;
- better agnosticism;
- reusable explain trees;
- cleaner adapters for non-HTTP inputs.

Negative:

- new public API surface;
- generated policy integration requires careful sequencing;
- users may expect Prolog features that are intentionally unsupported.

## Guardrail

Any future addition that introduces recursion, negation, aggregation, arithmetic, or parser syntax requires a new ADR.
```

- [ ] **Step 3: Update existing docs**

Add a short section to `README.md`:

```markdown
### Optional Logic-Inspired Facts

CervoRules can use an optional Datalog-inspired facts layer for cross-domain relationships and derived facts. This is not a Prolog runtime; it is a bounded, deterministic layer for policy inputs and explanations. See `docs/logic-inspired-facts.md`.
```

Add to `docs/modular-architecture.md` module table:

```markdown
| `facts` | Optional Datalog-inspired facts, bounded derivation, and explanation traces. | Prolog parser, side-effecting predicates, unbounded recursion, transport ownership. |
```

Add to `docs/agnosticism.md`:

```markdown
Derived facts improve agnosticism by keeping domain relationships in caller-owned data instead of adding new public fields for each project. The facts module is optional and cross-domain by design.
```

- [ ] **Step 4: Run docs grep**

Run:

```powershell
rg -n "Logic-Inspired Facts|Datalog|Prolog|facts module" README.md docs
```

Expected: references exist in README and docs.

- [ ] **Step 5: Commit**

```powershell
git add README.md docs/logic-inspired-facts.md docs/adr/0005-logic-inspired-facts.md docs/modular-architecture.md docs/agnosticism.md
git commit -m "docs: plan logic-inspired facts boundary"
```

## Phase 7: Neutral Example

### Task 7: Add Manual Neutral Example

**Files:**
- Create: `examples/logic-facts/README.md`
- Create: `examples/logic-facts/main_test.go`
- Modify: `examples/README.md`

- [ ] **Step 1: Write failing example test**

Create `examples/logic-facts/main_test.go`:

```go
package logicfacts_test

import (
    "context"
    "testing"

    "github.com/cervantesh/cervo-rules/v2/facts"
)

func TestBillingDerivedFacts(t *testing.T) {
    input := facts.NewSet(
        facts.NewFact("tenant_plan", facts.Const("acme"), facts.Const("pro")),
        facts.NewFact("request_tenant", facts.Const("req-123"), facts.Const("acme")),
    )

    engine := facts.NewEngine(facts.NewRule(
        "pro_tenants_make_premium_requests",
        facts.NewPattern("premium_request", facts.Var("request")),
        facts.NewPattern("request_tenant", facts.Var("request"), facts.Var("tenant")),
        facts.NewPattern("tenant_plan", facts.Var("tenant"), facts.Const("pro")),
    ))

    result, err := engine.Evaluate(context.Background(), input, facts.EvalOptions{MaxIterations: 4, MaxFacts: 32})
    if err != nil {
        t.Fatalf("Evaluate returned error: %v", err)
    }
    if !result.Facts.Has("premium_request", "req-123") {
        t.Fatalf("expected premium_request fact")
    }
}
```

- [ ] **Step 2: Run test to verify it passes after facts module exists**

Run:

```powershell
go test ./examples/logic-facts -count=1
```

Expected: PASS after Phases 1-5 are merged.

- [ ] **Step 3: Add example README**

Create `examples/logic-facts/README.md`:

```markdown
# Logic Facts Example

This example shows how a consumer can derive neutral facts before a policy decision.

It intentionally avoids CervoProxy, HTTP, providers, and AI vocabulary.

The example derives:

```text
premium_request(Request) <= request_tenant(Request, Tenant), tenant_plan(Tenant, "pro")
```

This pattern is useful when policies need relationships but the core `Request` type should remain stable.
```

- [ ] **Step 4: Link example**

Add to `examples/README.md`:

```markdown
| `logic-facts` | Demonstrates optional derived facts using neutral billing-style relationships. |
```

- [ ] **Step 5: Run example validation**

Run:

```powershell
go test -count=1 ./examples/logic-facts
go test -count=1 ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add examples/logic-facts examples/README.md
git commit -m "example: add neutral derived facts workflow"
```

## Phase 8: Core Bridge Design Checkpoint

### Task 8: Design Optional Predicate Bridge

**Files:**
- Create: `docs/facts-core-bridge.md`
- Do not modify core runtime yet.

- [ ] **Step 1: Document bridge options**

Create `docs/facts-core-bridge.md`:

```markdown
# Facts To Core Bridge

This document evaluates how derived facts should become policy inputs.

## Option A: Metadata Adapter

Consumers evaluate facts before calling CervoRules and copy selected derived facts into `Request.Metadata`.

Pros:

- no core API change;
- works today;
- keeps facts optional.

Cons:

- stringly typed metadata;
- less direct explain integration.

## Option B: Predicate Adapter

Expose a small predicate helper:

```go
func HasDerivedFact(set facts.Set, predicate facts.Atom, values ...string) core.Predicate
```

Pros:

- explicit policy use;
- still optional.

Cons:

- core may need an interface to avoid importing `facts`.

## Option C: DecisionResult Diagnostics Integration

Attach derivation explanations to `DecisionResult.Diagnostics` or a future `DecisionResult.Explanations`.

Pros:

- richer explain output.

Cons:

- public API change;
- requires observability review.

## Recommendation

Ship Option A first. Design Option B after real examples prove the shape. Defer Option C to a separate ADR.
```

- [ ] **Step 2: Review with ADR 0001**

Run:

```powershell
rg -n "cross-domain|adapter-specific|public API" docs/adr/0001-cross-domain-public-api.md docs/facts-core-bridge.md
```

Expected: bridge doc aligns with cross-domain guardrails.

- [ ] **Step 3: Commit**

```powershell
git add docs/facts-core-bridge.md
git commit -m "docs: design optional facts core bridge"
```

## Phase 9: Generated Policy Integration Design

### Task 9: Define Generator Integration Without Implementing It

**Files:**
- Create: `docs/facts-policygen-integration.md`
- Modify: `docs/generator-architecture.md`

- [ ] **Step 1: Document future YAML shape**

Create `docs/facts-policygen-integration.md`:

```markdown
# Facts Policygen Integration

Generated policy support should come after the base `facts` module is stable.

## Candidate YAML

```yaml
derived_facts:
  - name: pro_tenants_make_premium_requests
    head:
      predicate: premium_request
      terms:
        - var: request
    body:
      - predicate: request_tenant
        terms:
          - var: request
          - var: tenant
      - predicate: tenant_plan
        terms:
          - var: tenant
          - const: pro
```

## Generator Requirements

- validate predicates against generated vocabulary or an explicit facts vocabulary;
- reject unbound head variables;
- reject empty rule names;
- emit deterministic Go;
- emit tests that prove derived facts change decisions only when expected;
- include derivation trace in generated test assertions.

## Not In First Generator Pass

- recursive rules;
- negation;
- aggregation;
- arithmetic;
- custom functions.
```

- [ ] **Step 2: Link from generator architecture**

Add to `docs/generator-architecture.md`:

```markdown
Future generated derived facts are tracked in `docs/facts-policygen-integration.md`. They must remain optional and must not make generated policies depend on the facts module unless the YAML declares derived facts.
```

- [ ] **Step 3: Commit**

```powershell
git add docs/facts-policygen-integration.md docs/generator-architecture.md
git commit -m "docs: design derived facts policygen integration"
```

## Phase 10: Extended Datalog-Inspired Roadmap

### Task 10: Document Advanced Features As Separate Follow-Up Issues

**Files:**
- Modify: `docs/logic-inspired-facts.md`
- Modify: `docs/adr/0005-logic-inspired-facts.md`
- Modify: `docs/superpowers/plans/2026-05-22-logic-inspired-facts.md`
- Create later, when implementation starts: focused issues for each feature below.

This phase is intentionally documentation and issue-shaping first. These features are valuable, but they should not all ship in the first `facts` PR because they have different risk profiles and touch different parts of the evaluator.

- [ ] **Step 1: Add Datalog feature matrix to docs**

Add this section to `docs/logic-inspired-facts.md` after the first bounded evaluator documentation exists:

```markdown
## Datalog Features We May Adopt

| Feature | CervoRules Use | First Allowed Scope | Requires New ADR |
|---|---|---|---|
| Safety / range restriction | Prevent variables from appearing out of nowhere. | Required in v1 of `facts`. | No |
| EDB / IDB separation | Separate caller facts from derived facts. | Required in v1 of `facts`. | No |
| Provenance | Explain why each derived fact exists. | Required in v1 of `facts`. | No |
| Constraints | Detect invalid or ambiguous fact states. | v2 of `facts`. | No, if non-mutating |
| Stratified rule groups | Evaluate normalize -> identity -> eligibility -> decision facts in layers. | v2 of `facts`. | No, if acyclic |
| Stratified negation | Support `not suspended(User)` safely. | v3 of `facts`. | Yes |
| Bounded recursion | Support role inheritance and service dependency ancestry. | v3 of `facts`. | Yes |
| Controlled aggregates | Count/sum/min/max over bounded facts. | v4 of `facts`. | Yes |
| Semi-naive evaluation | Avoid recomputing unchanged derivations. | Internal optimization. | No |
| Incremental evaluation | Recompute only changed facts. | Internal/advanced API. | Yes if public |
| Magic sets | Optimize query-specific derivation. | Internal optimization after benchmarks. | No if private |
| Explain plans | Show rule cost and binding cardinality. | Observability extension. | Yes if public API changes |
| Materialized views | Cache derived route/eligibility facts. | Consumer-side pattern first. | Yes if runtime-owned |
| Closed world assumption | Treat facts that cannot be derived as false. | Required documentation. | No |
```

- [ ] **Step 2: Add constraints design**

Add this design section:

```markdown
## Constraints

Constraints detect states that should not exist. They do not derive policy decisions directly.

Example:

```text
violation("conflicting_routes", Request) :-
  route_candidate(Request, ServiceA),
  route_candidate(Request, ServiceB),
  ServiceA != ServiceB.
```

Initial supported constraint operators should be limited to:

- equality of bound variables;
- inequality of bound variables;
- fact presence.

No arithmetic or custom functions in the first pass.

Constraint output should be structured diagnostics:

```go
type ConstraintViolation struct {
    Constraint string
    Fact       facts.Fact
    Reason     string
}
```
```

- [ ] **Step 3: Add stratified rule groups design**

Add:

```markdown
## Stratified Rule Groups

Strata let policies organize derived facts into deterministic phases:

```text
stratum 1: normalize input facts
stratum 2: derive identity and tenant facts
stratum 3: derive eligibility facts
stratum 4: derive route candidates
stratum 5: derive decision facts
```

Rules in a later stratum may depend on facts from earlier strata. Rules must not depend on future strata.

Proposed API:

```go
type Stratum struct {
    Name  string
    Rules []facts.Rule
}

func NewStratifiedEngine(strata ...facts.Stratum) facts.Engine
```

This is useful for explain output because the trace can show which layer produced each fact.
```

- [ ] **Step 4: Add stratified negation design**

Add:

```markdown
## Stratified Negation

Negation is useful for deny lists and disabled states:

```text
eligible_user(User) :-
  user(User),
  not suspended(User).
```

Guardrails:

- negated patterns must be fully bound by earlier positive patterns;
- negation cannot participate in cycles;
- negation can only reference facts from the same or earlier stratum;
- closed-world assumption must be explicit in docs.

Candidate Go shape:

```go
facts.Not(facts.NewPattern("suspended", facts.Var("user")))
```

This requires an ADR before implementation.
```

- [ ] **Step 5: Add bounded recursion design**

Add:

```markdown
## Bounded Recursion

Recursion is useful for inheritance and ancestry:

```text
inherits_role(User, Role) :-
  direct_role(User, Role).

inherits_role(User, Parent) :-
  inherits_role(User, Role),
  role_inherits(Role, Parent).
```

Guardrails:

- recursion must still respect `MaxIterations` and `MaxFacts`;
- cycle detection should emit diagnostics;
- recursive predicates must be declared explicitly;
- no recursive negation.

Candidate use cases:

- role inheritance;
- tenant hierarchy;
- service dependency ancestry;
- fallback route chains.

This requires an ADR before implementation.
```

- [ ] **Step 6: Add controlled aggregates design**

Add:

```markdown
## Controlled Aggregates

Aggregates support bounded counts and summaries:

```text
tool_count(Request, count<Tool>) :-
  request_tool(Request, Tool).
```

Initial aggregate set:

- `count`;
- `sum` for integer values;
- `min` and `max` for integer values.

Guardrails:

- aggregate inputs must be finite and bounded by `MaxFacts`;
- aggregate output cardinality must be explicit;
- aggregate labels must be low-cardinality when exposed to observability;
- no floating point aggregates in the first pass.

Candidate use cases:

- requested tool count;
- failed dependency count;
- available fallback count;
- estimated cost bucket;
- rate-like policy summaries.

This requires an ADR before implementation.
```

- [ ] **Step 7: Add evaluation optimization design**

Add:

```markdown
## Evaluation Optimizations

### Semi-Naive Evaluation

The first evaluator can be simple fixed-point evaluation. If benchmarks show repeated work, add semi-naive evaluation internally:

```text
new facts from iteration N become the delta for iteration N+1
```

This should not change public behavior.

### Incremental Evaluation

Incremental evaluation is useful when service health, device state, or tenant facts change frequently.

Possible future API:

```go
type Snapshot struct {
    Facts facts.Set
}

func (e Engine) EvaluateDelta(ctx context.Context, previous Snapshot, changes ChangeSet, options EvalOptions) (Result, error)
```

This is a public API and requires an ADR.

### Magic Sets

Magic sets should remain an internal optimization for query-specific derivation. Example: if a policy only asks for `can_route(req-123, Service)`, the evaluator should avoid deriving unrelated route facts.

No public API should expose magic sets until benchmarks prove need.
```

- [ ] **Step 8: Add explain plans and provenance design**

Add:

```markdown
## Provenance And Explain Plans

Provenance explains why a fact exists:

```text
premium_request("req-123")
because:
  request_tenant("req-123", "acme")
  tenant_plan("acme", "pro")
```

Explain plans explain evaluation cost:

```text
rule pro_tenants_make_premium_requests
  scanned request_tenant: 1
  scanned tenant_plan: 12
  produced bindings: 1
  derived facts: 1
```

The first `facts` module should include provenance. Explain plans should come later as an observability extension.
```

- [ ] **Step 9: Add materialized views and closed-world assumption notes**

Add:

```markdown
## Materialized Views

Derived facts are effectively materialized views over input facts. Consumers can cache common views such as:

- `route_allowed(Request, Service)`;
- `premium_request(Request)`;
- `service_available(Service)`;
- `trusted_user(User)`.

CervoRules should not own cache invalidation in the first pass. Consumers can cache `facts.Result` if their input facts have stable versioning.

## Closed World Assumption

If a fact cannot be found or derived, it is false for the current evaluation.

This fits policy engines:

- no `allowed` fact means do not allow;
- no `route_candidate` fact means do not route;
- no `trusted_user` fact means do not trust.

Docs and examples must make this explicit.
```

- [ ] **Step 10: Create follow-up issues**

Create issues with this dependency sequence:

```text
facts: add constraints and constraint diagnostics
  depends on base facts evaluator and derivation trace

facts: add stratified rule groups
  depends on constraints or can run in parallel after base evaluator

facts: design stratified negation ADR
  depends on stratified rule groups

facts: design bounded recursion ADR
  depends on stratified rule groups

facts: design controlled aggregates ADR
  depends on base evaluator and constraints

facts: optimize evaluator with semi-naive derivation
  depends on benchmark evidence after base evaluator

facts: design incremental evaluation ADR
  depends on semi-naive evaluator and real consumer need

facts: add explain plan observability
  depends on provenance and benchmark counters

facts: document materialized views and cache patterns
  depends on base evaluator and neutral examples

facts: document closed-world policy semantics
  depends on base docs
```

Each issue should include:

- problem statement;
- non-goals;
- DoD;
- TDD test plan;
- dependency notes;
- expected blast radius;
- estimated time;
- actual time and deviation fields.

## Phase 11: Operational Hardening Roadmap

### Task 11: Add Production Guardrails And Tooling To The Plan

**Files:**
- Modify: `docs/logic-inspired-facts.md`
- Modify: `docs/adr/0005-logic-inspired-facts.md`
- Modify: `docs/observability.md`
- Modify: `docs/generator-architecture.md`
- Modify: `docs/release.md`
- Create later, when implementation starts: focused issues for schema, serialization, linting, redaction, and tooling.

This phase captures the operational work needed before the facts layer is used broadly across CervoProxy and other future projects. These items should not block the minimal facts MVP, but they should block calling the feature production-mature.

- [ ] **Step 1: Add predicate schema design**

Add this section to `docs/logic-inspired-facts.md`:

```markdown
## Predicate Schema

Predicate schemas define arity, term types, and optional enum constraints.

Example:

```yaml
predicates:
  user_role:
    terms:
      - name: user
        type: string
      - name: role
        type: enum
        values: [admin, operator, reader]
  request_cost:
    terms:
      - name: request
        type: string
      - name: cents
        type: int
```

Initial supported types:

- `string`;
- `int`;
- `bool`;
- `enum`;
- `time`;
- `duration`.

The first implementation may store values as strings internally, but schema validation must produce typed diagnostics before evaluation.

Schema validation should catch:

- unknown predicate;
- wrong arity;
- invalid term type;
- invalid enum value;
- variable used where a constant is required by a constraint or aggregate.
```

- [ ] **Step 2: Add fact source, trust, and freshness design**

Add:

```markdown
## Fact Source, Trust, And Freshness

Facts should be able to carry optional metadata without changing logical equality by default.

Candidate API:

```go
type SourceKind string

const (
    SourceRequest SourceKind = "request"
    SourceConfig  SourceKind = "config"
    SourceAdapter SourceKind = "adapter"
    SourceCache   SourceKind = "cache"
    SourceLookup  SourceKind = "external_lookup"
    SourceDerived SourceKind = "derived"
)

type FactMetadata struct {
    Source     SourceKind
    TrustLevel string
    ObservedAt time.Time
    ExpiresAt  time.Time
}
```

Use cases:

- request facts from adapters;
- service health facts from runtime health checks;
- tenant facts from cache;
- derived facts from rules;
- external lookup facts from consumer-owned adapters.

Guardrails:

- expired facts should produce diagnostics before use;
- freshness should be explicit in traces;
- trust level is advisory unless a policy rule consumes it.
```

- [ ] **Step 3: Add temporal facts and TTL design**

Add:

```markdown
## TTL And Temporal Facts

Some facts become unsafe after time passes:

- service health;
- device telemetry;
- tenant entitlement;
- rate/cost counters;
- incident flags.

Initial support should be freshness validation, not temporal logic.

Candidate checks:

```go
func ValidateFreshness(set facts.Set, now time.Time) []facts.Diagnostic
```

Future temporal facts may support windows:

```text
request_count("tenant-a", "1m", 42)
service_down("media", since="2026-05-22T10:00:00Z")
```

Temporal windows require an ADR before public API implementation.
```

- [ ] **Step 4: Add conflict resolution design**

Add:

```markdown
## Conflict Resolution

Derived facts can conflict:

```text
allow("req-1")
deny("req-1")
route_candidate("req-1", "service-a")
route_candidate("req-1", "service-b")
```

CervoRules should not hide conflicts. The preferred default is:

1. constraints detect conflicts;
2. diagnostics report them;
3. policy consumers decide whether conflict means deny, error, or priority selection.

If a future decision layer adds priority, it must be explicit:

```go
type Priority int
```

Priority and negation interaction requires a separate ADR.
```

- [ ] **Step 5: Add fact cardinality budgets**

Add:

```markdown
## Fact Cardinality Budgets

`MaxFacts` limits total growth. Predicate-level budgets limit local explosions:

```yaml
budgets:
  user_role: 100
  request_tool: 32
  route_candidate: 16
```

Diagnostics should include:

- predicate;
- configured limit;
- observed count;
- source if available.

This protects policies from high-cardinality metadata and accidental fan-out.
```

- [ ] **Step 6: Add deterministic ordering requirements**

Add:

```markdown
## Deterministic Ordering

Facts, bindings, derivation entries, diagnostics, serialized output, and explain strings must have stable ordering.

Required ordering:

1. predicate name;
2. arity;
3. term values;
4. derivation rule name;
5. insertion sequence only as final tie-breaker.

This is required for:

- golden tests;
- policy diffs;
- generated code review;
- package reproducibility.
```

- [ ] **Step 7: Add serialization format design**

Add:

```markdown
## Serialization

Facts, rules, schemas, traces, and diagnostics need stable JSON/YAML formats for fixtures and tooling.

Initial files:

```text
facts.json
rules.yaml
predicate-schema.yaml
derivation-trace.json
diagnostics.json
```

Rules:

- JSON output must be deterministic;
- YAML input must reject unknown fields;
- schema version must be explicit;
- serialized traces must support redaction.

Candidate schema header:

```yaml
schema_version: cervorules.facts/v1
```
```

- [ ] **Step 8: Add security and privacy redaction design**

Add this section to `docs/observability.md` and `docs/logic-inspired-facts.md`:

```markdown
## Redaction

Fact values can contain user IDs, tenant IDs, request IDs, device IDs, or payload-derived values. Derivation traces and explain plans must support redaction.

Candidate API:

```go
type Redactor interface {
    RedactFact(facts.Fact) facts.Fact
    RedactDiagnostic(facts.Diagnostic) facts.Diagnostic
}
```

Default behavior:

- metric labels must never include raw fact term values;
- logs may include redacted values only;
- diagnostics should include predicate and field names, not sensitive raw values, unless explicitly enabled by the caller.
```

- [ ] **Step 9: Add rule linting design**

Add:

```markdown
## Rule Linting

Linting catches problems before runtime.

Initial lint checks:

- unknown predicate;
- wrong arity;
- unused variable;
- variable appears only once in multi-pattern body;
- duplicate rule name;
- duplicate rule body/head;
- rule can never match based on enum schema;
- unbound head variable;
- unsafe negation;
- cyclic dependency where recursion is not declared;
- aggregate over high-cardinality predicate;
- constraint that can never produce a violation.

Candidate API:

```go
func Lint(rules []facts.Rule, schema facts.PredicateSchema) []facts.Diagnostic
```
```

- [ ] **Step 10: Add golden explanations and policy diff design**

Add:

```markdown
## Golden Explanations

Golden tests should cover:

- derived facts;
- derivation trace;
- explain strings;
- diagnostics;
- redacted observability output.

This protects audit quality, not just allow/deny behavior.

## Policy Diff

Policy diff should compare derived facts between two rule sets:

```text
+ premium_request("req-123")
- route_candidate("req-123", "legacy-service")
~ explanation changed for trusted_user("alice")
```

Diff output should be stable and redaction-aware.
```

- [ ] **Step 11: Add partial evaluation and materialized view design**

Add:

```markdown
## Partial Evaluation

Some facts are static across requests:

- tenant plans;
- service capability mappings;
- role inheritance;
- service dependency graph.

Partial evaluation precomputes static derived facts and leaves request-specific facts for runtime.

Candidate API:

```go
func (e Engine) PartiallyEvaluate(ctx context.Context, static facts.Set, options EvalOptions) (PreparedEngine, error)
func (p PreparedEngine) EvaluateRequest(ctx context.Context, requestFacts facts.Set) (Result, error)
```

This should come after base evaluator benchmarks.
```

- [ ] **Step 12: Add REPL/explain CLI design**

Add:

```markdown
## Explain CLI

The facts layer should eventually have a debugging CLI, not a Prolog REPL.

Candidate command:

```bash
cervorules-facts explain \
  --schema predicate-schema.yaml \
  --facts facts.json \
  --rules rules.yaml \
  --query 'premium_request("req-123")'
```

Required output:

- whether the queried fact exists;
- derivation tree;
- diagnostics;
- redacted mode by default;
- optional JSON output for automation.
```

- [ ] **Step 13: Add fact namespacing and versioned vocabulary design**

Add:

```markdown
## Namespacing

Predicate names should support namespaces:

```text
identity.user_role
billing.tenant_plan
routing.route_candidate
device.telemetry_status
```

Namespaces avoid collisions across projects and generated vocabularies.

## Versioned Fact Vocabulary

Generated policies should be able to publish a fact vocabulary:

```yaml
schema_version: cervorules.fact-vocabulary/v1
namespace: billing
predicates:
  billing.tenant_plan:
    terms:
      - name: tenant
        type: string
      - name: plan
        type: enum
        values: [free, pro, enterprise]
```

Fact vocabularies should be packaged with generated policy artifacts when policygen adopts derived facts.
```

- [ ] **Step 14: Create operational hardening issues**

Create issues in this order:

```text
facts: add predicate schema validation
  depends on base facts model

facts: add deterministic ordering guarantees
  depends on base fact set and trace

facts: add stable serialization formats
  depends on predicate schema

facts: add fact source and freshness metadata
  depends on serialization format

facts: add fact TTL validation
  depends on freshness metadata

facts: add privacy redaction for facts traces and diagnostics
  depends on serialization and observability docs

facts: add rule linting
  depends on predicate schema and constraints

facts: add conflict diagnostics pattern
  depends on constraints

facts: add predicate-level cardinality budgets
  depends on evaluator diagnostics

facts: add golden explanation contracts
  depends on derivation trace and deterministic ordering

facts: design policy diff for derived facts
  depends on golden explanations and serialization

facts: design partial evaluation ADR
  depends on benchmark evidence

facts: design explain CLI
  depends on serialization, redaction, and trace

facts: design namespaced versioned fact vocabulary
  depends on predicate schema and policygen integration design
```

Each issue should include:

- DoD;
- TDD tests;
- risk class;
- API compatibility notes;
- expected docs updates;
- issue dependency links;
- estimate, actual time, and deviation.

## Phase 12: Verification And PR

### Task 12: Final Verification

**Files:**
- All files changed by this plan.

- [ ] **Step 1: Run package tests**

Run:

```powershell
go test -count=1 ./facts
```

Expected: PASS.

- [ ] **Step 2: Run full suite**

Run:

```powershell
go test -count=1 ./...
```

Expected: PASS.

- [ ] **Step 3: Run coverage**

Run:

```powershell
go test -cover ./...
```

Expected: all statement packages remain >= 90%, including `facts`.

- [ ] **Step 4: Run vet and module verification**

Run:

```powershell
go vet ./...
go mod verify
```

Expected: PASS.

- [ ] **Step 5: Check dependency scope**

Run:

```powershell
go test -run TestDependencyScope -count=1 .
```

Expected: PASS. The facts module must not add runtime external dependencies.

- [ ] **Step 6: Open PR**

Use title:

```text
feat(facts): add Datalog-inspired derived facts
```

PR body must include:

```markdown
## Summary

- adds optional `facts` module for flat facts, queries, bounded derivation, and derivation traces;
- documents the boundary with Prolog;
- adds neutral derived facts example;
- does not change core policy runtime semantics.

## Public API

New optional module: `github.com/cervantesh/cervo-rules/v2/facts`.

## Not Prolog

No parser, no cut, no general unification, no side effects, no unbounded recursion.

## Tests

- `go test -count=1 ./facts`
- `go test -count=1 ./...`
- `go test -cover ./...`
- `go vet ./...`
- `go mod verify`
```

## Sequencing Recommendation

Merge sequence:

1. Docs-only plan and ADR proposal.
2. Base `facts` module data model and queries.
3. Bounded evaluator and trace.
4. Neutral example and docs.
5. Core bridge design.
6. Generated policygen design.
7. Constraints and closed-world docs.
8. Stratified rule groups.
9. ADRs for stratified negation, bounded recursion, controlled aggregates, and incremental evaluation.
10. Semi-naive optimization only after benchmark evidence.
11. Explain plan observability.
12. Predicate schema and minimal type system.
13. Deterministic ordering guarantees.
14. Stable serialization formats.
15. Privacy redaction for traces, diagnostics, and explain output.
16. Fact source, trust, freshness metadata, and TTL diagnostics.
17. Predicate-level cardinality budgets.
18. Rule linting and conflict diagnostics.
19. Golden explanations and policy diff.
20. Partial evaluation only after benchmark evidence.
21. Explain CLI and versioned fact vocabulary.
22. Only then implement optional core/policygen integration.

This sequence avoids merge conflicts because early phases do not touch `core` or `internal/policygen`.

## Risk Register

| Risk | Mitigation |
|---|---|
| Users assume this is Prolog | Name docs and ADR clearly: Datalog-inspired, not Prolog. |
| Evaluation becomes unbounded | Require `MaxIterations` and `MaxFacts`; default bounded values. |
| Core becomes coupled to facts | Keep facts optional; bridge through adapters first. |
| API becomes too generic too early | Ship minimal flat facts only; require ADR for recursion, negation, aggregation, arithmetic, or parser syntax. |
| Generated YAML becomes complex | Defer policygen until the Go API is stable and examples prove value. |
| Negation creates surprising decisions | Allow only stratified negation with fully-bound variables and explicit closed-world docs. |
| Recursion creates cycles or performance issues | Require explicit recursive predicates, cycle diagnostics, `MaxIterations`, and `MaxFacts`. |
| Aggregates produce high-cardinality observability | Restrict initial aggregates and document low-cardinality output expectations. |
| Optimizations change semantics | Golden tests must compare naive and optimized evaluator outputs. |
| Explain plans leak sensitive facts | Reuse observability privacy guardrails and redact caller-owned values where required. |
| Predicate schemas drift from generated vocabularies | Version fact vocabularies and require schema compatibility checks in generated artifacts. |
| TTL and freshness behavior becomes clock-dependent | Accept caller-provided `now` values in validation and tests; document stale fact diagnostics. |
| Priority hides conflicts | Keep conflict diagnostics as the default; require an ADR before priority selects outcomes. |
| Ordering instability makes diffs noisy | Define canonical ordering for facts, bindings, diagnostics, traces, and serialized output. |
| CLI tooling leaks sensitive values | Make redacted output the default and require explicit opt-in for raw values. |
| Serialization format changes break consumers | Include explicit schema versions and release migration notes for every format change. |
| Partial evaluation changes runtime decisions | Require golden equivalence tests between prepared and non-prepared evaluation. |

## Success Criteria

- A consumer can represent cross-domain relationships as facts.
- A consumer can derive facts deterministically.
- A consumer can explain why a derived fact exists.
- CervoRules remains usable without importing the facts module.
- No CervoProxy-specific names enter the module.
- All tests, coverage, vet, and module verification pass.
- Constraints can report ambiguous or invalid fact states without changing decisions.
- Stratified rules can organize derivations into predictable phases.
- Negation, recursion, aggregates, incremental evaluation, and explain plans each have ADRs before public implementation.
- Naive and optimized evaluators produce identical derived facts for the same bounded input.
- Predicate schema validation catches unknown predicates, wrong arity, invalid enum values, and unsafe variable usage before evaluation.
- Deterministic ordering makes golden explanations, policy diffs, and serialized fixtures stable.
- Serialization formats include explicit schema versions and reject unknown YAML fields.
- Redaction protects raw fact values in metrics, logs, diagnostics, traces, and explain CLI output by default.
- Freshness and TTL diagnostics identify stale facts without embedding wall-clock globals in the evaluator.
- Rule linting catches unused variables, unsafe negation, duplicate rules, undeclared recursion, and impossible matches.
- Policy diff can compare derived facts and explanations across rule versions.
- Partial evaluation has benchmark evidence and equivalence tests before public API release.
- Namespaced, versioned fact vocabularies are documented before policygen emits facts.

## Self-Review

- Spec coverage: facts, derived facts, explain tree, query subset, Datalog boundary, adapter/core separation, policygen future design, operational hardening, and TDD are covered.
- Placeholder scan: no implementation task uses TBD/TODO/fill-later language.
- Type consistency: `Atom`, `Term`, `Fact`, `Pattern`, `Set`, `Rule`, `Engine`, `Result`, and `DerivationTrace` are introduced before use.
- Merge safety: first PR can be docs-only; implementation phases avoid `core` and `policygen` until later checkpoints.
