package ontology

import (
	"sort"
	"strings"
)

// EntityType names a caller-owned class of individuals, such as "order" or
// "support_agent".
type EntityType string

// Individual identifies one concrete entity instance, such as an order id.
type Individual string

// Relation names a caller-owned binary property between individuals, such as
// "order.parent" or "assigned_to".
type Relation string

// State names one lifecycle state, such as "paid" or "refunded".
type State string

// NewEntityType normalizes a caller-owned entity type name.
func NewEntityType(value string) EntityType { return EntityType(normalize(value)) }

// NewIndividual normalizes a caller-owned individual name.
//
// Individuals keep their original case because they are frequently opaque ids
// where case is significant. Only surrounding whitespace is trimmed.
func NewIndividual(value string) Individual { return Individual(strings.TrimSpace(value)) }

// NewRelation normalizes a caller-owned relation name.
func NewRelation(value string) Relation { return Relation(normalize(value)) }

// NewState normalizes a caller-owned state name.
func NewState(value string) State { return State(normalize(value)) }

// TypeAssertion states that an individual belongs to an entity type.
type TypeAssertion struct {
	Individual Individual `json:"individual"`
	Type       EntityType `json:"type"`
}

// RelationAssertion states that a subject relates to an object.
type RelationAssertion struct {
	Relation Relation   `json:"relation"`
	Subject  Individual `json:"subject"`
	Object   Individual `json:"object"`
}

// StateAssertion states that an individual currently sits in a lifecycle state.
type StateAssertion struct {
	Lifecycle  string     `json:"lifecycle"`
	Individual Individual `json:"individual"`
	State      State      `json:"state"`
}

// Snapshot is the closed-world set of assertions checked against an Ontology.
//
// Closed world is deliberate and matches the facts engine: an assertion that is
// absent is treated as false, not as unknown. For guard questions such as "has
// this order already been refunded?" that is the semantics a caller wants.
type Snapshot struct {
	Types     []TypeAssertion     `json:"types,omitempty"`
	Relations []RelationAssertion `json:"relations,omitempty"`
	States    []StateAssertion    `json:"states,omitempty"`

	// normalized marks a snapshot Normalize already processed. A snapshot
	// arriving from a request scope is already normalized, so checks must not
	// re-sort the whole world on every condition.
	normalized bool
}

// Normalize returns a copy with normalized names and stable ordering, so that
// checks and their structured errors are deterministic.
func (s Snapshot) Normalize() Snapshot {
	if s.normalized {
		return s
	}
	out := Snapshot{
		normalized: true,
		Types:      make([]TypeAssertion, 0, len(s.Types)),
		Relations:  make([]RelationAssertion, 0, len(s.Relations)),
		States:     make([]StateAssertion, 0, len(s.States)),
	}
	for _, assertion := range s.Types {
		out.Types = append(out.Types, TypeAssertion{
			Individual: NewIndividual(string(assertion.Individual)),
			Type:       NewEntityType(string(assertion.Type)),
		})
	}
	for _, assertion := range s.Relations {
		out.Relations = append(out.Relations, RelationAssertion{
			Relation: NewRelation(string(assertion.Relation)),
			Subject:  NewIndividual(string(assertion.Subject)),
			Object:   NewIndividual(string(assertion.Object)),
		})
	}
	for _, assertion := range s.States {
		out.States = append(out.States, StateAssertion{
			Lifecycle:  normalize(assertion.Lifecycle),
			Individual: NewIndividual(string(assertion.Individual)),
			State:      NewState(string(assertion.State)),
		})
	}
	sort.SliceStable(out.Types, func(i, j int) bool {
		if out.Types[i].Individual != out.Types[j].Individual {
			return out.Types[i].Individual < out.Types[j].Individual
		}
		return out.Types[i].Type < out.Types[j].Type
	})
	sort.SliceStable(out.Relations, func(i, j int) bool {
		if out.Relations[i].Relation != out.Relations[j].Relation {
			return out.Relations[i].Relation < out.Relations[j].Relation
		}
		if out.Relations[i].Subject != out.Relations[j].Subject {
			return out.Relations[i].Subject < out.Relations[j].Subject
		}
		return out.Relations[i].Object < out.Relations[j].Object
	})
	// State is part of the key, unlike the sibling sorts which already order by
	// their full tuple. Without it, two contradictory assertions for one
	// individual kept whatever order the resolver produced, and StateOf returns
	// the first match -- so the same world answered a guard differently
	// depending on how the caller happened to build the slice.
	sort.SliceStable(out.States, func(i, j int) bool {
		if out.States[i].Lifecycle != out.States[j].Lifecycle {
			return out.States[i].Lifecycle < out.States[j].Lifecycle
		}
		if out.States[i].Individual != out.States[j].Individual {
			return out.States[i].Individual < out.States[j].Individual
		}
		return out.States[i].State < out.States[j].State
	})
	return out
}

// TypesOf returns the entity types asserted for an individual, in stable order.
func (s Snapshot) TypesOf(individual Individual) []EntityType {
	individual = NewIndividual(string(individual))
	var out []EntityType
	for _, assertion := range s.Types {
		if NewIndividual(string(assertion.Individual)) == individual {
			out = append(out, NewEntityType(string(assertion.Type)))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// StateOf returns the current state asserted for an individual in a lifecycle.
func (s Snapshot) StateOf(lifecycle string, individual Individual) (State, bool) {
	lifecycle = normalize(lifecycle)
	individual = NewIndividual(string(individual))
	for _, assertion := range s.States {
		if normalize(assertion.Lifecycle) != lifecycle {
			continue
		}
		if NewIndividual(string(assertion.Individual)) == individual {
			return NewState(string(assertion.State)), true
		}
	}
	return "", false
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
