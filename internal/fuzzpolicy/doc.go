// Package fuzzpolicy holds a generated policy used as a fuzzing subject.
//
// The fact-parsing logic a policy runs is emitted into generated code, not
// carried by any library package, so there is nothing in this module to fuzz
// directly. This package is the subject: a real generated engine, committed so
// a fuzzer can run against it without a codegen step per execution.
//
// It is generated from examples/predicate-composition, which is the one example
// declaring a fact of every kind — enum, integer, number and bool — with a
// minimum, a maximum and a default between them.
//
// generated_policy.go and vocab/generated.go are produced by the CLIs and must
// not be edited. TestFuzzSubjectMatchesGenerator regenerates both and compares
// bytes, so this package cannot drift away from what the generator emits today.
package fuzzpolicy
