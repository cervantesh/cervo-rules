package ontology

import "github.com/cervantesh/cervo-rules/v3/core"

// Constraint is a consumer-owned integrity rule.
//
// The built-in constraint families are deliberately few. Rather than wait for
// this package to grow cardinality, property hierarchies or equality, a
// consumer implements Constraint and gets the same treatment as a built-in:
// the same structured core.Errors, the same build-time gate through Validate,
// and the same closed-world Snapshot.
//
// Implementations must be side-effect free and deterministic. A Snapshot is
// already normalized when Check receives it.
type Constraint interface {
	// Name identifies the constraint in diagnostics.
	Name() string
	// Validate checks the constraint's own declaration against the model. It runs
	// at build time, with no snapshot, so authoring mistakes fail in CI.
	Validate(model Ontology) core.Errors
	// Check evaluates the constraint against a snapshot at decision time.
	Check(model Ontology, snapshot Snapshot) core.Errors
}

// ConstraintFunc adapts a plain snapshot check to the Constraint interface for
// consumers that need no build-time validation.
type ConstraintFunc struct {
	ConstraintName string
	Fn             func(model Ontology, snapshot Snapshot) core.Errors
}

// Name implements Constraint.
func (c ConstraintFunc) Name() string {
	if c.ConstraintName == "" {
		return "custom"
	}
	return c.ConstraintName
}

// Validate implements Constraint.
func (c ConstraintFunc) Validate(Ontology) core.Errors { return nil }

// Check implements Constraint.
func (c ConstraintFunc) Check(model Ontology, snapshot Snapshot) core.Errors {
	if c.Fn == nil {
		return nil
	}
	return c.Fn(model, snapshot)
}
