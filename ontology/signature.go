package ontology

import "github.com/cervantesh/cervo-rules/v3/core"

// PredicateSignature declares the domain and range of a relation.
//
// This is the RDFS idea, kept deliberately small: a relation says which entity
// type its subject must have and which entity type its object must have. That
// alone catches a large class of agent mistakes, because a wrong-role argument
// stops being a plausible string and becomes a type error.
type PredicateSignature struct {
	Relation Relation   `json:"relation"`
	Domain   EntityType `json:"domain"`
	Range    EntityType `json:"range"`

	// Description documents intent for humans and for generated code.
	Description string `json:"description,omitempty"`
}

// Normalize returns a copy with normalized names.
func (s PredicateSignature) Normalize() PredicateSignature {
	s.Relation = NewRelation(string(s.Relation))
	s.Domain = NewEntityType(string(s.Domain))
	s.Range = NewEntityType(string(s.Range))
	return s
}

// checkAssertion validates one relation assertion against this signature.
//
// Domain and range are only enforced when the snapshot actually types the
// individual. An untyped individual is reported separately by checkTyped so a
// caller can choose how strict to be.
func (s PredicateSignature) checkAssertion(snapshot Snapshot, assertion RelationAssertion) core.Errors {
	var errs core.Errors
	signature := s.Normalize()
	if signature.Domain != "" && !hasType(snapshot, assertion.Subject, signature.Domain) {
		errs = append(errs, core.Error{
			Code:      ErrorCodeDomainViolation,
			Severity:  core.SeverityError,
			Component: componentName,
			Field:     "relations." + string(signature.Relation) + ".subject",
			Value:     string(assertion.Subject),
			Reason: "subject of " + string(signature.Relation) +
				" must be of type " + string(signature.Domain),
			Suggestion: "assert the subject type before relating it, or fix the relation",
		})
	}
	if signature.Range != "" && !hasType(snapshot, assertion.Object, signature.Range) {
		errs = append(errs, core.Error{
			Code:      ErrorCodeRangeViolation,
			Severity:  core.SeverityError,
			Component: componentName,
			Field:     "relations." + string(signature.Relation) + ".object",
			Value:     string(assertion.Object),
			Reason: "object of " + string(signature.Relation) +
				" must be of type " + string(signature.Range),
			Suggestion: "assert the object type before relating it, or fix the relation",
		})
	}
	return errs
}

func hasType(snapshot Snapshot, individual Individual, want EntityType) bool {
	for _, candidate := range snapshot.TypesOf(individual) {
		if candidate == want {
			return true
		}
	}
	return false
}
