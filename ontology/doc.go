// Package ontology is the optional symbolic guard layer for CervoRules.
//
// A decision engine answers "which route handles this operation?". An ontology
// answers a different question: "is the world this request describes even
// coherent?". The two are deliberately separate. Routing stays on the hot path;
// ontology checks run where the caller decides they are affordable.
//
// The package expresses the small subset of description-logic constraints that
// earn their keep in business policy, without adopting an RDF/OWL runtime:
//
//   - entity types with predicate domain/range signatures;
//   - disjoint type sets, so one individual cannot be two things at once;
//   - functional properties, so a relation maps a subject to at most one value;
//   - lifecycles, so an entity only moves through legal state transitions.
//
// Nothing here imports an RDF parser, a triple store, or a reasoner. Ontologies
// are plain Go values that a consumer can build by hand, load from the
// CervoRules DSL, or compile from an external OWL file at build time.
//
// All checks are side-effect free and return core.Errors, so an agent loop can
// validate a proposed action before executing anything.
package ontology
