package ontology

import (
	"sort"

	"github.com/cervantesh/cervo-rules/v3/core"
)

// Ontology is the caller-owned symbolic model checked around a decision.
//
// It is a plain value: build it in Go, load it from the CervoRules DSL, or
// compile it from an external source at build time. Nothing in this type
// requires a reasoner or a triple store.
type Ontology struct {
	Entities   []EntityType         `json:"entities,omitempty"`
	Signatures []PredicateSignature `json:"signatures,omitempty"`
	Disjoint   []DisjointSet        `json:"disjoint,omitempty"`
	Functional []FunctionalProperty `json:"functional,omitempty"`
	Lifecycles []Lifecycle          `json:"lifecycles,omitempty"`

	// Custom carries consumer-owned constraints. It is Go-only and deliberately
	// excluded from serialization: the declarative fields above stay portable to
	// JSON and to a future DSL, while extensions are code.
	Custom []Constraint `json:"-"`

	// normalized marks a model that Normalize already processed, so repeated
	// checks do not re-sort the same declaration. Mutating an Ontology after
	// normalizing it invalidates this; build the value, then normalize.
	normalized bool
}

// Normalize returns a copy with normalized names and stable ordering.
func (o Ontology) Normalize() Ontology {
	if o.normalized {
		return o
	}
	out := Ontology{normalized: true}
	seen := map[EntityType]struct{}{}
	for _, entity := range o.Entities {
		normalized := NewEntityType(string(entity))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out.Entities = append(out.Entities, normalized)
	}
	sort.Slice(out.Entities, func(i, j int) bool { return out.Entities[i] < out.Entities[j] })
	for _, signature := range o.Signatures {
		out.Signatures = append(out.Signatures, signature.Normalize())
	}
	sort.SliceStable(out.Signatures, func(i, j int) bool {
		return out.Signatures[i].Relation < out.Signatures[j].Relation
	})
	for _, set := range o.Disjoint {
		out.Disjoint = append(out.Disjoint, set.Normalize())
	}
	sort.SliceStable(out.Disjoint, func(i, j int) bool { return out.Disjoint[i].Name < out.Disjoint[j].Name })
	for _, property := range o.Functional {
		out.Functional = append(out.Functional, property.Normalize())
	}
	sort.SliceStable(out.Functional, func(i, j int) bool {
		return out.Functional[i].Relation < out.Functional[j].Relation
	})
	for _, lifecycle := range o.Lifecycles {
		out.Lifecycles = append(out.Lifecycles, lifecycle.Normalize())
	}
	sort.SliceStable(out.Lifecycles, func(i, j int) bool { return out.Lifecycles[i].Name < out.Lifecycles[j].Name })
	out.Custom = append([]Constraint(nil), o.Custom...)
	return out
}

// domainOf returns the declared domain of a relation, or empty when the
// relation has no signature.
func (o Ontology) domainOf(relation Relation) EntityType {
	for _, signature := range o.Signatures {
		if signature.Normalize().Relation == NewRelation(string(relation)) {
			return signature.Normalize().Domain
		}
	}
	return ""
}

// Lifecycle returns a lifecycle by name.
func (o Ontology) Lifecycle(name string) (Lifecycle, bool) {
	want := normalize(name)
	for _, lifecycle := range o.Normalize().Lifecycles {
		if lifecycle.Name == want {
			return lifecycle, true
		}
	}
	return Lifecycle{}, false
}

// Validate checks the ontology definition itself, with no snapshot involved.
//
// This is the build-time gate: run it in CI so that an authoring mistake fails
// the pipeline instead of surfacing as a runtime surprise.
func (o Ontology) Validate() core.Errors {
	model := o.Normalize()
	declared := map[EntityType]struct{}{}
	for _, entity := range model.Entities {
		declared[entity] = struct{}{}
	}
	var errs core.Errors
	// Two signatures for one relation are an authoring mistake with no right
	// answer, and the two readers disagreed about which one wins: Check builds
	// a map, so the last declaration replaced the first, while domainOf loops
	// and returns the first. A declared constraint could therefore stop being
	// enforced, or be enforced against the wrong types, with nothing reported.
	seenSignature := map[Relation]struct{}{}
	for _, signature := range model.Signatures {
		if _, ok := seenSignature[signature.Relation]; ok {
			errs = append(errs, core.Error{
				Code:       ErrorCodeFunctionalViolation,
				Severity:   core.SeverityFatal,
				Component:  componentName,
				Field:      "signatures",
				Value:      string(signature.Relation),
				Reason:     "relation has more than one predicate signature",
				Suggestion: "declare one signature per relation",
			})
			continue
		}
		seenSignature[signature.Relation] = struct{}{}
	}
	unknownEntity := func(field string, entity EntityType) {
		if entity == "" {
			return
		}
		if _, ok := declared[entity]; ok {
			return
		}
		errs = append(errs, core.Error{
			Code:       ErrorCodeUnknownEntityType,
			Severity:   core.SeverityFatal,
			Component:  componentName,
			Field:      field,
			Value:      string(entity),
			Reason:     "entity type is not declared in entities",
			Suggestion: "declare the entity type before referencing it",
		})
	}
	for _, signature := range model.Signatures {
		field := "signatures." + string(signature.Relation)
		unknownEntity(field+".domain", signature.Domain)
		unknownEntity(field+".range", signature.Range)
	}
	for _, set := range model.Disjoint {
		for _, entity := range set.Types {
			unknownEntity("constraints.disjoint."+set.Name, entity)
		}
		if len(set.Types) < 2 {
			errs = append(errs, core.Error{
				Code:       core.ErrorCodeInvalidConfig,
				Severity:   core.SeverityWarning,
				Component:  componentName,
				Field:      "constraints.disjoint." + set.Name,
				Reason:     "disjoint set needs at least two types to constrain anything",
				Suggestion: "add another type or remove the set",
			})
		}
	}
	knownRelation := map[Relation]struct{}{}
	for _, signature := range model.Signatures {
		knownRelation[signature.Relation] = struct{}{}
	}
	for _, property := range model.Functional {
		if property.Relation == "" {
			errs = append(errs, core.Error{
				Code:      core.ErrorCodeInvalidConfig,
				Severity:  core.SeverityFatal,
				Component: componentName,
				Field:     "constraints.functional",
				Reason:    "functional constraint requires a relation name",
			})
			continue
		}
		if _, ok := knownRelation[property.Relation]; !ok {
			errs = append(errs, core.Error{
				Code:       core.ErrorCodeInvalidConfig,
				Severity:   core.SeverityWarning,
				Component:  componentName,
				Field:      "constraints.functional." + string(property.Relation),
				Value:      string(property.Relation),
				Reason:     "functional relation has no declared signature",
				Suggestion: "declare a signature so domain and range are checked too",
			})
		}
		// Required quantifies over the relation's domain. Without one there is
		// nothing to require it of, so the flag would silently do nothing.
		if property.Required && model.domainOf(property.Relation) == "" {
			errs = append(errs, core.Error{
				Code:      core.ErrorCodeInvalidConfig,
				Severity:  core.SeverityFatal,
				Component: componentName,
				Field:     "constraints.functional." + string(property.Relation) + ".required",
				Value:     string(property.Relation),
				Reason:    "required functional relation has no declared domain to quantify over",
				Suggestion: "declare a signature with a domain for this relation, " +
					"or drop required",
			})
		}
	}
	for _, constraint := range model.Custom {
		if constraint == nil {
			continue
		}
		errs = append(errs, constraint.Validate(model)...)
	}
	for _, lifecycle := range model.Lifecycles {
		unknownEntity("lifecycles."+lifecycle.Name+".entity", lifecycle.Entity)
		errs = append(errs, lifecycle.Validate()...)
	}
	return errs.SortStable()
}

// Check validates a snapshot against the ontology.
//
// It reports every violation it finds rather than stopping at the first, so an
// agent gets one actionable list instead of a game of whack-a-mole.
func (o Ontology) Check(snapshot Snapshot) core.Errors {
	model := o.Normalize()
	normalized := snapshot.Normalize()
	var errs core.Errors

	declared := map[EntityType]struct{}{}
	for _, entity := range model.Entities {
		declared[entity] = struct{}{}
	}
	if len(declared) > 0 {
		for _, assertion := range normalized.Types {
			if _, ok := declared[assertion.Type]; ok {
				continue
			}
			errs = append(errs, core.Error{
				Code:       ErrorCodeUnknownEntityType,
				Severity:   core.SeverityFatal,
				Component:  componentName,
				Field:      "types",
				Value:      string(assertion.Type),
				Reason:     "asserted entity type is not declared by the ontology",
				Suggestion: "declare the type or correct the assertion",
			})
		}
	}

	signatures := map[Relation]PredicateSignature{}
	for _, signature := range model.Signatures {
		signatures[signature.Relation] = signature
	}
	for _, assertion := range normalized.Relations {
		signature, ok := signatures[assertion.Relation]
		if !ok {
			continue
		}
		errs = append(errs, signature.checkAssertion(normalized, assertion)...)
	}

	for _, set := range model.Disjoint {
		errs = append(errs, set.check(normalized)...)
	}
	for _, property := range model.Functional {
		errs = append(errs, property.check(normalized, model.domainOf(property.Relation))...)
	}
	for _, lifecycle := range model.Lifecycles {
		errs = append(errs, lifecycle.checkSnapshot(normalized)...)
	}
	for _, constraint := range model.Custom {
		if constraint == nil {
			continue
		}
		errs = append(errs, constraint.Check(model, normalized)...)
	}
	return errs.SortStable()
}

// CheckTransition validates a proposed lifecycle transition for an individual.
func (o Ontology) CheckTransition(snapshot Snapshot, lifecycle string, individual Individual, to State) core.Errors {
	definition, ok := o.Lifecycle(lifecycle)
	if !ok {
		return core.Errors{{
			Code:       ErrorCodeUnknownLifecycle,
			Severity:   core.SeverityFatal,
			Component:  componentName,
			Field:      "lifecycles",
			Value:      lifecycle,
			Reason:     "lifecycle is not declared by the ontology",
			Suggestion: "declare the lifecycle before checking transitions against it",
		}}
	}
	return definition.CheckTransition(snapshot, individual, to)
}
