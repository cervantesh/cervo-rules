// Package moduleguard holds the contracts that belong to the module as a whole
// rather than to any one package: what the public API surface is, whether the
// current-facing documents describe an API that exists, and what a consumer
// pulls into their build by importing us.
//
// These checks used to live in internal/policygen because that is where the
// first one was written. Nothing in them concerns the generator.
//
// The package has no non-test source on purpose. It is a place for assertions,
// not for code that ships.
package moduleguard
