package ontology

import (
	"context"
	"sync"

	"github.com/cervantesh/cervo-rules/v3/core"
)

type requestScopeKey struct{}

// requestScope memoizes one snapshot resolution for the lifetime of a request.
//
// owner records which request the memoized snapshot belongs to, so that reusing
// one scope across two different requests is refused instead of silently
// answering the second request against the first request's world.
type requestScope struct {
	once     sync.Once
	owner    string
	bound    bool
	snapshot Snapshot
	err      error
}

// WithRequestScope returns a context in which a Guard resolves its snapshot at
// most once, no matter how many conditions a policy consults.
//
// Without it a policy that consults three conditions triggers three resolver
// calls, which for a database-backed resolver means three round-trips for one
// decision. Install it once per request:
//
//	ctx = ontology.WithRequestScope(ctx)
//	result, err := engine.Decide(ctx, req)
//
// The scope also makes every condition in one decision observe the same world.
// Evaluating a single request against two different snapshots is a coherence
// bug, not a feature.
//
// Nothing is cached beyond the context, so CervoRules still never holds live
// state across requests.
func WithRequestScope(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestScopeKey{}, &requestScope{})
}

// resolveOnce returns the request-scoped snapshot, resolving it if needed.
//
// When no scope is installed it falls back to resolving directly, so callers
// that never opt in keep working unchanged.
func resolveOnce(ctx context.Context, resolve SnapshotResolver, req core.Request) (Snapshot, error) {
	scope, ok := ctx.Value(requestScopeKey{}).(*requestScope)
	if !ok || scope == nil {
		snapshot, err := resolve(ctx, req)
		if err != nil {
			return Snapshot{}, err
		}
		return snapshot.Normalize(), nil
	}
	scope.once.Do(func() {
		scope.owner = req.ID
		scope.bound = true
		snapshot, err := resolve(ctx, req)
		if err != nil {
			scope.err = err
			return
		}
		scope.snapshot = snapshot.Normalize()
	})
	// A scope belongs to one request. Reusing it across a batch would answer the
	// second request against the first one's world, with no error to notice.
	// Request IDs are optional, so this catches the detectable case and the doc
	// comment carries the rest.
	if scope.bound && req.ID != "" && scope.owner != "" && req.ID != scope.owner {
		return Snapshot{}, core.Error{
			Code:      core.ErrorCodeConditionFailed,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     "request.id",
			Value:     req.ID,
			Reason: "request scope was created for request " + scope.owner +
				" and cannot be reused for another request",
			Suggestion: "install ontology.WithRequestScope once per request, not per batch",
		}
	}
	return scope.snapshot, scope.err
}
