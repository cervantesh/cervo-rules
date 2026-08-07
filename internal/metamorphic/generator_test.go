package metamorphic

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// A fact as the generator invented it, carrying enough to build both the YAML
// that declares it and the request values that probe it.
type randomFact struct {
	Name   string
	Kind   string // number | integer | bool | enum
	Min    float64
	Max    float64
	Values []string // enum members
}

type randomPolicy struct {
	Name       string
	Operations []string
	Targets    []string
	Executor   string
	Facts      []randomFact
	Denies     []randomDeny
	// bounded is a numeric fact with both a minimum and a maximum, which the
	// inert variant needs to build a predicate that cannot hold.
	bounded randomFact
}

type randomDeny struct {
	ID        string
	Operation string // empty means every operation
	Reason    string
	When      predNode
}

// predNode is the closed tagged form the DSL accepts: composition is all / any
// / not, and a leaf compares one fact against a literal.
type predNode struct {
	Kind  string // all | any | not | leaf
	Kids  []predNode
	Fact  string
	Op    string
	Value string
}

func newRandomPolicy(rng *rand.Rand, index int) randomPolicy {
	policy := randomPolicy{
		Name:       fmt.Sprintf("random-%d.v3", index),
		Operations: []string{"op_alpha", "op_beta"},
		Targets:    []string{"target_one", "target_two"},
		Executor:   "executor_main",
	}

	// One bounded numeric fact is mandatory: the inert variant is built from it.
	policy.bounded = randomFact{Name: "bounded_pct", Kind: "number", Min: 0, Max: 100}
	policy.Facts = append(policy.Facts, policy.bounded)

	kinds := []string{"number", "integer", "bool", "enum"}
	for i := 0; i < 1+rng.Intn(3); i++ {
		kind := kinds[rng.Intn(len(kinds))]
		fact := randomFact{Name: fmt.Sprintf("fact_%d", i), Kind: kind}
		switch kind {
		case "number":
			fact.Min, fact.Max = 0, float64(10*(1+rng.Intn(10)))
		case "integer":
			fact.Min, fact.Max = 0, float64(10*(1+rng.Intn(10)))
		case "enum":
			fact.Values = []string{"red", "green", "blue"}[:2+rng.Intn(2)]
		}
		policy.Facts = append(policy.Facts, fact)
	}

	for i := 0; i < 1+rng.Intn(3); i++ {
		when := randomCompositePredicate(rng, policy.Facts)
		if i == 0 {
			// The first rule must read the bounded fact. Its bounds are what
			// make the inert variant's predicate unsatisfiable, and a bound is
			// only enforced -- and only accepted by the generator -- for a fact
			// some rule reads.
			when.Kids[0] = predNode{Kind: "leaf", Fact: policy.bounded.Name, Op: "gte",
				Value: strconv.FormatFloat(policy.bounded.Min, 'f', 1, 64)}
		}
		deny := randomDeny{
			ID:     fmt.Sprintf("deny-%d", i),
			Reason: fmt.Sprintf("random rule %d refused it", i),
			When:   when,
		}
		// Half the rules are scoped to one operation; the rest apply to all.
		if rng.Intn(2) == 0 {
			deny.Operation = policy.Operations[rng.Intn(len(policy.Operations))]
		}
		policy.Denies = append(policy.Denies, deny)
	}
	return policy
}

func randomPredicate(rng *rand.Rand, facts []randomFact, depth int) predNode {
	if depth <= 0 || rng.Intn(3) == 0 {
		return randomLeaf(rng, facts)
	}
	switch rng.Intn(3) {
	case 0:
		return predNode{Kind: "not", Kids: []predNode{randomPredicate(rng, facts, depth-1)}}
	case 1:
		return predNode{Kind: "all", Kids: []predNode{
			randomPredicate(rng, facts, depth-1),
			randomPredicate(rng, facts, depth-1),
		}}
	default:
		return predNode{Kind: "any", Kids: []predNode{
			randomPredicate(rng, facts, depth-1),
			randomPredicate(rng, facts, depth-1),
		}}
	}
}

func randomLeaf(rng *rand.Rand, facts []randomFact) predNode {
	fact := facts[rng.Intn(len(facts))]
	leaf := predNode{Kind: "leaf", Fact: fact.Name}
	switch fact.Kind {
	case "bool":
		leaf.Op = []string{"is_true", "is_false"}[rng.Intn(2)]
	case "enum":
		leaf.Op = []string{"eq", "ne"}[rng.Intn(2)]
		leaf.Value = fact.Values[rng.Intn(len(fact.Values))]
	default:
		leaf.Op = []string{"eq", "ne", "gt", "gte", "lt", "lte"}[rng.Intn(6)]
		// Chosen inside the declared bounds so the rule can actually fire; a
		// literal outside them makes the leaf constant and tests nothing.
		span := fact.Max - fact.Min
		value := fact.Min + span*float64(rng.Intn(11))/10
		if fact.Kind == "integer" {
			leaf.Value = strconv.FormatInt(int64(value), 10)
		} else {
			leaf.Value = strconv.FormatFloat(value, 'f', 1, 64)
		}
	}
	return leaf
}

// renderYAML writes a predicate at the given indentation.
func (n predNode) renderYAML(indent string) string {
	switch n.Kind {
	case "leaf":
		if n.Value == "" {
			return fmt.Sprintf("%s{ fact: %s, op: %s }", indent, n.Fact, n.Op)
		}
		return fmt.Sprintf("%s{ fact: %s, op: %s, value: %s }", indent, n.Fact, n.Op, n.Value)
	case "not":
		return fmt.Sprintf("%snot:\n%s", indent, n.Kids[0].renderYAML(indent+"  "))
	default:
		var b strings.Builder
		fmt.Fprintf(&b, "%s%s:\n", indent, n.Kind)
		for _, kid := range n.Kids {
			// Render the child one level deeper, then move the dash onto its
			// first line. Deeper lines keep the indentation they were given.
			rendered := kid.renderYAML(indent + "    ")
			b.WriteString(indent + "  - " + strings.TrimPrefix(rendered, indent+"    ") + "\n")
		}
		return strings.TrimRight(b.String(), "\n")
	}
}

// deMorgan rewrites a predicate into its De Morgan dual, which must decide
// identically for every request.
//
// The obvious transformation is not(not(p)), but it is a blind oracle: a
// compiler that rendered `not` as the identity would render both sides
// identically wrong and the property would never fire. De Morgan is not blind,
// because it moves the negation across a change of connective -- a broken
// `not`, or `all` and `any` swapped, makes the two sides disagree.
func deMorgan(n predNode) predNode {
	switch n.Kind {
	case "all":
		return negate(predNode{Kind: "any", Kids: negateEach(n.Kids)})
	case "any":
		return negate(predNode{Kind: "all", Kids: negateEach(n.Kids)})
	default:
		return negate(negate(n))
	}
}

func negate(n predNode) predNode {
	return predNode{Kind: "not", Kids: []predNode{n}}
}

func negateEach(kids []predNode) []predNode {
	out := make([]predNode, 0, len(kids))
	for _, kid := range kids {
		out = append(out, negate(kid))
	}
	return out
}

// randomCompositePredicate always returns all or any at the root, so the De
// Morgan variant is a real change of connective rather than a double negation.
func randomCompositePredicate(rng *rand.Rand, facts []randomFact) predNode {
	kind := "all"
	if rng.Intn(2) == 0 {
		kind = "any"
	}
	return predNode{Kind: kind, Kids: []predNode{
		randomPredicate(rng, facts, 1),
		randomPredicate(rng, facts, 1),
	}}
}

// unsatisfiable returns a predicate that cannot hold, given that a fact outside
// its declared bounds fails the decision before any rule runs.
func (p randomPolicy) unsatisfiable() predNode {
	return predNode{Kind: "all", Kids: []predNode{
		{Kind: "leaf", Fact: p.bounded.Name, Op: "lt", Value: strconv.FormatFloat(p.bounded.Min, 'f', 1, 64)},
		{Kind: "leaf", Fact: p.bounded.Name, Op: "gt", Value: strconv.FormatFloat(p.bounded.Max, 'f', 1, 64)},
	}}
}

func (p randomPolicy) vocabularyYAML() string {
	var b strings.Builder
	b.WriteString("operations:\n")
	for _, op := range p.Operations {
		fmt.Fprintf(&b, "  %s: {}\n", op)
	}
	b.WriteString("targets:\n")
	for _, target := range p.Targets {
		fmt.Fprintf(&b, "  %s: {}\n", target)
	}
	fmt.Fprintf(&b, "executors:\n  %s: {}\n", p.Executor)
	b.WriteString("facts:\n")
	for _, fact := range p.Facts {
		fmt.Fprintf(&b, "  %s:\n    type: %s\n", fact.Name, fact.Kind)
		if fact.Kind == "enum" {
			fmt.Fprintf(&b, "    values: [%s]\n", strings.Join(fact.Values, ", "))
		}
	}
	return b.String()
}

// policyYAML renders a variant. transform rewrites each deny's predicate, and
// extra is appended as a further deny when non-nil.
func (p randomPolicy) policyYAML(variant string, transform func(predNode) predNode, extra *randomDeny) string {
	var b strings.Builder
	fmt.Fprintf(&b, "version: cervorules.policy.v3\nname: %s-%s\n\n", strings.TrimSuffix(p.Name, ".v3"), variant)
	fmt.Fprintf(&b, "defaults:\n  executor: %s\n\n", p.Executor)

	// Bounds are declared only for facts a rule reads: the generator refuses a
	// bound that would protect nothing, which is the behaviour these policies
	// are meant to exercise rather than sidestep.
	read := p.readFacts()
	b.WriteString("facts:\n")
	for _, fact := range p.Facts {
		if !read[fact.Name] {
			continue
		}
		switch fact.Kind {
		case "number":
			fmt.Fprintf(&b, "  %s: { min: %s, max: %s }\n", fact.Name,
				strconv.FormatFloat(fact.Min, 'f', 1, 64), strconv.FormatFloat(fact.Max, 'f', 1, 64))
		case "integer":
			fmt.Fprintf(&b, "  %s: { min: %d, max: %d }\n", fact.Name, int64(fact.Min), int64(fact.Max))
		}
	}

	denies := append([]randomDeny{}, p.Denies...)
	if extra != nil {
		denies = append(denies, *extra)
	}
	b.WriteString("\ndenies:\n")
	for _, deny := range denies {
		fmt.Fprintf(&b, "  - id: %s\n", deny.ID)
		if deny.Operation != "" {
			fmt.Fprintf(&b, "    operation: %s\n", deny.Operation)
		}
		fmt.Fprintf(&b, "    reason: %s\n", deny.Reason)
		when := deny.When
		if transform != nil {
			when = transform(when)
		}
		fmt.Fprintf(&b, "    when:\n%s\n", when.renderYAML("      "))
	}

	b.WriteString("\nroutes:\n")
	for i, op := range p.Operations {
		fmt.Fprintf(&b, "  - id: route_%s\n    operation: %s\n    target: %s\n    executor: %s\n",
			op, op, p.Targets[i%len(p.Targets)], p.Executor)
	}
	return b.String()
}

// readFacts returns the facts some deny predicate mentions. Only these may
// carry bounds: the generator rejects a bound no rule reads.
func (p randomPolicy) readFacts() map[string]bool {
	read := map[string]bool{}
	var walk func(predNode)
	walk = func(n predNode) {
		if n.Kind == "leaf" {
			read[n.Fact] = true
			return
		}
		for _, kid := range n.Kids {
			walk(kid)
		}
	}
	for _, deny := range p.Denies {
		walk(deny.When)
	}
	return read
}
