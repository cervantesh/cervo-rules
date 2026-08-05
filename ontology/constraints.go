package ontology

import (
	"sort"
	"strings"

	"github.com/cervantesh/cervo-rules/v3/core"
)

const componentName = "ontology"

// DisjointSet declares that an individual may hold at most one of these types.
//
// This is the OWL disjointness idea. The canonical guard is "a customer is not
// a support agent", which stops an agent from routing a refund to the person
// who requested it.
type DisjointSet struct {
	Name  string       `json:"name"`
	Types []EntityType `json:"types"`
}

// Normalize returns a copy with normalized, de-duplicated, sorted types.
func (d DisjointSet) Normalize() DisjointSet {
	seen := map[EntityType]struct{}{}
	out := make([]EntityType, 0, len(d.Types))
	for _, entityType := range d.Types {
		normalized := NewEntityType(string(entityType))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	d.Name = normalize(d.Name)
	d.Types = out
	return d
}

// check reports individuals that hold more than one type from this set.
func (d DisjointSet) check(snapshot Snapshot) core.Errors {
	set := d.Normalize()
	if len(set.Types) < 2 {
		return nil
	}
	member := map[EntityType]struct{}{}
	for _, entityType := range set.Types {
		member[entityType] = struct{}{}
	}
	held := map[Individual][]EntityType{}
	order := make([]Individual, 0, len(snapshot.Types))
	for _, assertion := range snapshot.Types {
		individual := NewIndividual(string(assertion.Individual))
		entityType := NewEntityType(string(assertion.Type))
		if _, ok := member[entityType]; !ok {
			continue
		}
		if _, seen := held[individual]; !seen {
			order = append(order, individual)
		}
		if !containsType(held[individual], entityType) {
			held[individual] = append(held[individual], entityType)
		}
	}
	var errs core.Errors
	for _, individual := range order {
		types := held[individual]
		if len(types) < 2 {
			continue
		}
		sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
		names := make([]string, 0, len(types))
		for _, entityType := range types {
			names = append(names, string(entityType))
		}
		errs = append(errs, core.Error{
			Code:      ErrorCodeDisjointViolation,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     "constraints.disjoint." + set.Name,
			Value:     string(individual),
			Reason: "individual holds disjoint types " + strings.Join(names, ", ") +
				" from set " + set.Name,
			Suggestion: "remove one of the conflicting type assertions",
		})
	}
	return errs
}

// FunctionalProperty declares that a relation maps a subject to at most one
// object.
//
// This is the OWL functional-property idea. The canonical guard is "an order
// has exactly one parent", which stops an agent from silently reparenting a
// record it should have rejected.
type FunctionalProperty struct {
	Relation Relation `json:"relation"`

	// Required makes the relation total: every individual of the relation's
	// declared domain must have exactly one object, not zero.
	//
	// Totality needs something to quantify over, so Required is only meaningful
	// when the relation has a PredicateSignature with a Domain. Ontology.Validate
	// rejects the combination when it does not.
	Required bool `json:"required,omitempty"`
}

// Normalize returns a copy with a normalized relation name.
func (f FunctionalProperty) Normalize() FunctionalProperty {
	f.Relation = NewRelation(string(f.Relation))
	return f
}

// check reports subjects bound to more than one object, and — when the property
// is Required — domain individuals bound to none.
//
// domain is the entity type declared by the relation's signature, or empty when
// the relation has no signature. Required is inert without it, which
// Ontology.Validate reports as an authoring error rather than silently ignoring.
func (f FunctionalProperty) check(snapshot Snapshot, domain EntityType) core.Errors {
	property := f.Normalize()
	if property.Relation == "" {
		return nil
	}
	objects := map[Individual][]Individual{}
	order := make([]Individual, 0, len(snapshot.Relations))
	for _, assertion := range snapshot.Relations {
		if NewRelation(string(assertion.Relation)) != property.Relation {
			continue
		}
		subject := NewIndividual(string(assertion.Subject))
		object := NewIndividual(string(assertion.Object))
		if _, seen := objects[subject]; !seen {
			order = append(order, subject)
		}
		if !containsIndividual(objects[subject], object) {
			objects[subject] = append(objects[subject], object)
		}
	}
	var errs core.Errors
	for _, subject := range order {
		bound := objects[subject]
		if len(bound) < 2 {
			continue
		}
		sort.Slice(bound, func(i, j int) bool { return bound[i] < bound[j] })
		names := make([]string, 0, len(bound))
		for _, object := range bound {
			names = append(names, string(object))
		}
		errs = append(errs, core.Error{
			Code:      ErrorCodeFunctionalViolation,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     "constraints.functional." + string(property.Relation),
			Value:     string(subject),
			Reason: "functional relation " + string(property.Relation) +
				" bound to multiple objects: " + strings.Join(names, ", "),
			Suggestion: "keep a single object for this relation, or drop the functional constraint",
		})
	}
	if !property.Required || domain == "" {
		return errs
	}
	for _, individual := range domainIndividuals(snapshot, domain) {
		if len(objects[individual]) > 0 {
			continue
		}
		errs = append(errs, core.Error{
			Code:      ErrorCodeFunctionalViolation,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     "constraints.functional." + string(property.Relation),
			Value:     string(individual),
			Reason: "required functional relation " + string(property.Relation) +
				" has no object for this " + string(domain),
			Suggestion: "assert exactly one object for this relation",
		})
	}
	return errs
}

// domainIndividuals returns the individuals typed as domain, in stable order.
func domainIndividuals(snapshot Snapshot, domain EntityType) []Individual {
	var out []Individual
	seen := map[Individual]struct{}{}
	for _, assertion := range snapshot.Types {
		if NewEntityType(string(assertion.Type)) != domain {
			continue
		}
		individual := NewIndividual(string(assertion.Individual))
		if _, ok := seen[individual]; ok {
			continue
		}
		seen[individual] = struct{}{}
		out = append(out, individual)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func containsType(values []EntityType, want EntityType) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsIndividual(values []Individual, want Individual) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
