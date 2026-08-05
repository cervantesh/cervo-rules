package ontology

import "github.com/cervantesh/cervo-rules/v3/core"

// Error codes produced by this package.
//
// `core.ErrorCode` is an open string type, so a leaf package declares the codes
// it is the sole producer of instead of growing core's catalog every time a
// constraint family is added. The wire values are unchanged by living here, and
// core.Errors.Has / ByCode work exactly as before.
//
// The condition-seam codes — unknown_condition, condition_failed and
// missing_conditions — deliberately stay in core: they belong to the
// core-owned Conditions contract, not to this package's model.
const (
	ErrorCodeUnknownEntityType   core.ErrorCode = "unknown_entity_type"
	ErrorCodeDomainViolation     core.ErrorCode = "domain_violation"
	ErrorCodeRangeViolation      core.ErrorCode = "range_violation"
	ErrorCodeDisjointViolation   core.ErrorCode = "disjoint_violation"
	ErrorCodeFunctionalViolation core.ErrorCode = "functional_violation"
	ErrorCodeUnknownLifecycle    core.ErrorCode = "unknown_lifecycle"
	ErrorCodeUnknownState        core.ErrorCode = "unknown_state"
	ErrorCodeIllegalTransition   core.ErrorCode = "illegal_transition"
	ErrorCodeTerminalState       core.ErrorCode = "terminal_state"
	ErrorCodeUnreachableState    core.ErrorCode = "unreachable_state"
)
