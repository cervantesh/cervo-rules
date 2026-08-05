package ontology

import (
	"context"
	"sort"

	"github.com/cervantesh/cervo-rules/v3/core"
)

// SnapshotResolver produces the closed-world snapshot for one request.
//
// The resolver is the only place a caller reaches outside the decision: it may
// read a database, a cache, or an in-memory projection. It must not mutate
// anything, because the whole point is that no side effect happens before the
// decision is made.
type SnapshotResolver func(ctx context.Context, req core.Request) (Snapshot, error)

// CheckKind names the kind of question a guarded condition asks.
type CheckKind string

const (
	// CheckTransitionAllowed holds when a proposed lifecycle transition is legal.
	CheckTransitionAllowed CheckKind = "transition_allowed"
	// CheckInState holds when the subject currently sits in one of States.
	CheckInState CheckKind = "in_state"
	// CheckHasType holds when the subject holds EntityType.
	CheckHasType CheckKind = "has_type"
	// CheckIntegrity holds when the whole snapshot satisfies the ontology.
	CheckIntegrity CheckKind = "integrity"
)

// Check binds one policy condition name to an ontology question.
//
// SubjectKey names the Request.Metadata entry carrying the individual the
// question is about. That is how an agent's tool-call arguments reach the
// symbolic layer without core having to understand them.
type Check struct {
	Kind       CheckKind  `json:"kind"`
	Lifecycle  string     `json:"lifecycle,omitempty"`
	SubjectKey string     `json:"subject_key,omitempty"`
	To         State      `json:"to,omitempty"`
	States     []State    `json:"states,omitempty"`
	EntityType EntityType `json:"entity_type,omitempty"`
}

// Guard answers named policy conditions from an ontology.
//
// It implements core.Conditions, so an engine can consult it by name without
// importing this package. Binding a condition name to a check is Go-side
// configuration; the DSL keyword that references those names is not designed
// yet, and it will not be `when`, which already means route predicates in the
// legacy schema.
type Guard struct {
	Ontology Ontology
	Resolve  SnapshotResolver
	Checks   map[core.Condition]Check
}

// NewGuard builds a Guard and validates the ontology it will enforce.
//
// Validation is eager and its result is returned rather than dropped, because
// an unvalidated guard fails open: a constraint whose declaration is incomplete
// silently enforces nothing, which is the failure mode this layer exists to
// prevent. Callers must treat a non-empty result as fatal.
//
// The ontology is also normalized once here rather than on every check, so a
// guard consulted many times does not re-sort the same model.
func NewGuard(model Ontology, resolve SnapshotResolver, checks map[core.Condition]Check) (Guard, core.Errors) {
	normalized := make(map[core.Condition]Check, len(checks))
	for condition, check := range checks {
		normalized[core.NewCondition(string(condition))] = check
	}
	guard := Guard{Ontology: model.Normalize(), Resolve: resolve, Checks: normalized}
	return guard, model.Validate().Fatal()
}

// Conditions returns the condition names this guard answers, in stable order.
func (g Guard) Conditions() []core.Condition {
	out := make([]core.Condition, 0, len(g.Checks))
	for condition := range g.Checks {
		out = append(out, condition)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Holds implements core.Conditions.
//
// An unknown condition, a missing resolver, or a missing subject fails closed:
// the caller gets a structured error rather than a false that would read as
// "the guard checked and found nothing wrong".
func (g Guard) Holds(ctx context.Context, condition core.Condition, req core.Request) (bool, error) {
	holds, _, err := g.Explain(ctx, condition, req)
	return holds, err
}

// Explain answers a condition and returns the violations behind a false.
//
// Holds collapses the answer to a boolean because that is what the core
// Conditions contract needs. That loses exactly what makes a symbolic guard
// worth having: the reason. A caller that has to tell whoever proposed the
// action why it was refused should use Explain instead.
//
// The returned errors are empty when the condition holds. They describe why the
// world is illegal, and are distinct from the returned error, which reports that
// the guard could not answer at all.
func (g Guard) Explain(ctx context.Context, condition core.Condition, req core.Request) (bool, core.Errors, error) {
	check, ok := g.Checks[core.NewCondition(string(condition))]
	if !ok {
		return false, nil, core.Error{
			Code:       core.ErrorCodeUnknownCondition,
			Severity:   core.SeverityFatal,
			Component:  componentName,
			Field:      "conditions",
			Value:      string(condition),
			Reason:     "condition is not bound to an ontology check",
			Suggestion: "bind the condition on the guard before building the policy",
		}
	}
	if g.Resolve == nil {
		return false, nil, core.Error{
			Code:      core.ErrorCodeMissingConditions,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     "conditions." + string(condition),
			Reason:    "guard has no snapshot resolver",
		}
	}
	// Resolve the subject before the snapshot. A request missing the metadata a
	// check needs is already unanswerable, and paying for a resolver round-trip
	// to discover that would be waste.
	var subject Individual
	if check.Kind != CheckIntegrity {
		resolved, err := subjectFor(condition, check, req)
		if err != nil {
			return false, nil, err
		}
		subject = resolved
	}

	snapshot, err := resolveOnce(ctx, g.Resolve, req)
	if err != nil {
		return false, nil, core.Error{
			Code:      core.ErrorCodeConditionFailed,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     "conditions." + string(condition),
			Reason:    "snapshot resolver failed",
			Cause:     err,
		}
	}

	if check.Kind == CheckIntegrity {
		violations := g.Ontology.Check(snapshot)
		return len(violations) == 0, violations, nil
	}

	switch check.Kind {
	case CheckTransitionAllowed:
		violations := g.Ontology.CheckTransition(snapshot, check.Lifecycle, subject, check.To)
		return len(violations) == 0, violations, nil
	case CheckInState:
		current, ok := snapshot.StateOf(check.Lifecycle, subject)
		if !ok {
			return false, nil, nil
		}
		for _, state := range check.States {
			if NewState(string(state)) == current {
				return true, nil, nil
			}
		}
		return false, nil, nil
	case CheckHasType:
		return hasType(snapshot, subject, NewEntityType(string(check.EntityType))), nil, nil
	default:
		return false, nil, core.Error{
			Code:       core.ErrorCodeUnsupportedFeature,
			Severity:   core.SeverityFatal,
			Component:  componentName,
			Field:      "conditions." + string(condition),
			Value:      string(check.Kind),
			Reason:     "unsupported ontology check kind",
			Suggestion: "use transition_allowed, in_state, has_type, or integrity",
		}
	}
}

func subjectFor(condition core.Condition, check Check, req core.Request) (Individual, error) {
	if check.SubjectKey == "" {
		return "", core.Error{
			Code:       core.ErrorCodeInvalidConfig,
			Severity:   core.SeverityFatal,
			Component:  componentName,
			Field:      "conditions." + string(condition) + ".subject_key",
			Reason:     "check requires a subject_key naming a request metadata entry",
			Suggestion: "set subject_key to the metadata key carrying the entity id",
		}
	}
	value := NewIndividual(req.Metadata[check.SubjectKey])
	if value == "" {
		return "", core.Error{
			Code:      core.ErrorCodeConditionFailed,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     "metadata." + check.SubjectKey,
			Reason:    "request metadata is missing the subject required by condition " + string(condition),
			Suggestion: "the caller must supply " + check.SubjectKey +
				" before this operation can be evaluated",
		}
	}
	return value, nil
}
