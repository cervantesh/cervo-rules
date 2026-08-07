package ontology

import (
	"sort"
	"strings"

	"github.com/cervantesh/cervo-rules/v3/core"
)

// Transition declares the states reachable from one state.
//
// A state with no outgoing transition is terminal. That is the whole mechanism
// behind the duplicate-refund guard: once an order reaches "refunded" there is
// nowhere legal left to go, so a second refund is rejected structurally rather
// than by remembering to write a rule for it.
type Transition struct {
	From State   `json:"from"`
	To   []State `json:"to,omitempty"`
}

// Lifecycle is a declarative state machine for one entity type.
type Lifecycle struct {
	Name        string       `json:"name"`
	Entity      EntityType   `json:"entity,omitempty"`
	States      []State      `json:"states"`
	Initial     State        `json:"initial,omitempty"`
	Transitions []Transition `json:"transitions,omitempty"`

	// normalized marks a lifecycle Normalize already processed. HasState, Next
	// and IsTerminal each normalize defensively, so without this a single
	// CheckTransition re-sorted the same lifecycle several times.
	normalized bool
}

// Normalize returns a copy with normalized names and stable ordering.
func (l Lifecycle) Normalize() Lifecycle {
	if l.normalized {
		return l
	}
	l.normalized = true
	l.Name = normalize(l.Name)
	l.Entity = NewEntityType(string(l.Entity))
	l.Initial = NewState(string(l.Initial))
	states := make([]State, 0, len(l.States))
	seen := map[State]struct{}{}
	for _, state := range l.States {
		normalized := NewState(string(state))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		states = append(states, normalized)
	}
	l.States = states
	transitions := make([]Transition, 0, len(l.Transitions))
	for _, transition := range l.Transitions {
		normalized := Transition{From: NewState(string(transition.From))}
		targetSeen := map[State]struct{}{}
		for _, to := range transition.To {
			target := NewState(string(to))
			if target == "" {
				continue
			}
			if _, ok := targetSeen[target]; ok {
				continue
			}
			targetSeen[target] = struct{}{}
			normalized.To = append(normalized.To, target)
		}
		sort.Slice(normalized.To, func(i, j int) bool { return normalized.To[i] < normalized.To[j] })
		transitions = append(transitions, normalized)
	}
	sort.SliceStable(transitions, func(i, j int) bool { return transitions[i].From < transitions[j].From })
	l.Transitions = transitions
	return l
}

// HasState reports whether a state is declared by this lifecycle.
func (l Lifecycle) HasState(state State) bool {
	want := NewState(string(state))
	for _, declared := range l.Normalize().States {
		if declared == want {
			return true
		}
	}
	return false
}

// Next returns the states legally reachable from a state, in stable order.
func (l Lifecycle) Next(from State) []State {
	want := NewState(string(from))
	for _, transition := range l.Normalize().Transitions {
		if transition.From == want {
			return append([]State(nil), transition.To...)
		}
	}
	return nil
}

// IsTerminal reports whether a declared state has no outgoing transition.
func (l Lifecycle) IsTerminal(state State) bool {
	return len(l.Next(state)) == 0
}

// Validate checks the lifecycle definition itself, independent of any snapshot.
//
// This runs at build time: an unreachable state or a transition to an
// undeclared state is a policy authoring bug, not a runtime condition.
func (l Lifecycle) Validate() core.Errors {
	lifecycle := l.Normalize()
	var errs core.Errors
	field := "lifecycles." + lifecycle.Name
	if lifecycle.Name == "" {
		errs = append(errs, core.Error{
			Code:      ErrorCodeUnknownLifecycle,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     "lifecycles",
			Reason:    "lifecycle name is required",
		})
	}
	if len(lifecycle.States) == 0 {
		errs = append(errs, core.Error{
			Code:      ErrorCodeUnknownState,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     field + ".states",
			Reason:    "lifecycle declares no states",
		})
		return errs.SortStable()
	}
	declared := map[State]struct{}{}
	for _, state := range lifecycle.States {
		declared[state] = struct{}{}
	}
	if lifecycle.Initial != "" {
		if _, ok := declared[lifecycle.Initial]; !ok {
			errs = append(errs, core.Error{
				Code:      ErrorCodeUnknownState,
				Severity:  core.SeverityFatal,
				Component: componentName,
				Field:     field + ".initial",
				Value:     string(lifecycle.Initial),
				Reason:    "initial state is not declared in states",
			})
		}
	}
	for _, transition := range lifecycle.Transitions {
		if _, ok := declared[transition.From]; !ok {
			errs = append(errs, core.Error{
				Code:      ErrorCodeUnknownState,
				Severity:  core.SeverityFatal,
				Component: componentName,
				Field:     field + ".transitions.from",
				Value:     string(transition.From),
				Reason:    "transition source state is not declared in states",
			})
		}
		for _, to := range transition.To {
			if _, ok := declared[to]; !ok {
				errs = append(errs, core.Error{
					Code:      ErrorCodeUnknownState,
					Severity:  core.SeverityFatal,
					Component: componentName,
					Field:     field + ".transitions.to",
					Value:     string(to),
					Reason:    "transition target state is not declared in states",
				})
			}
		}
	}
	if lifecycle.Initial != "" {
		for _, state := range lifecycle.unreachableFrom(lifecycle.Initial) {
			errs = append(errs, core.Error{
				Code:       ErrorCodeUnreachableState,
				Severity:   core.SeverityWarning,
				Component:  componentName,
				Field:      field + ".states",
				Value:      string(state),
				Reason:     "state is not reachable from the initial state",
				Suggestion: "add a transition into this state or remove it",
			})
		}
	}
	return errs.SortStable()
}

// unreachableFrom returns declared states not reachable from a start state.
func (l Lifecycle) unreachableFrom(start State) []State {
	reachable := map[State]struct{}{start: {}}
	queue := []State{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range l.Next(current) {
			if _, ok := reachable[next]; ok {
				continue
			}
			reachable[next] = struct{}{}
			queue = append(queue, next)
		}
	}
	var out []State
	for _, state := range l.States {
		if _, ok := reachable[state]; !ok {
			out = append(out, state)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// CheckTransition reports whether moving an individual from its current state
// to a proposed state is legal.
//
// The current state comes from the snapshot, so a caller never has to trust the
// agent's claim about where the entity already is.
func (l Lifecycle) CheckTransition(snapshot Snapshot, individual Individual, to State) core.Errors {
	lifecycle := l.Normalize()
	target := NewState(string(to))
	field := "lifecycles." + lifecycle.Name
	if !lifecycle.HasState(target) {
		return core.Errors{{
			Code:       ErrorCodeUnknownState,
			Severity:   core.SeverityFatal,
			Component:  componentName,
			Field:      field + ".to",
			Value:      string(target),
			Reason:     "target state is not declared by lifecycle " + lifecycle.Name,
			Suggestion: "use one of: " + joinStates(lifecycle.States),
		}}
	}
	// Fail closed before reading the current state. StateOf returns the first
	// match, so a snapshot recording an individual in two states of this
	// lifecycle would silently answer with one of them: an order recorded as
	// paid *and* refunded read as "paid", and a second refund was permitted.
	// The honest answer is that the current state cannot be determined, and
	// this check must not be reachable only through CheckIntegrity -- a policy
	// requiring transition_allowed alone has to be safe too.
	if errs := lifecycle.checkSingleState(snapshot, individual); len(errs) > 0 {
		return errs
	}
	current, ok := snapshot.Normalize().StateOf(lifecycle.Name, individual)
	if !ok {
		if lifecycle.Initial == "" || target == lifecycle.Initial {
			return nil
		}
		return core.Errors{{
			Code:      ErrorCodeIllegalTransition,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     field,
			Value:     string(individual),
			Reason: "individual has no recorded state; lifecycle " + lifecycle.Name +
				" must start at " + string(lifecycle.Initial),
			Suggestion: "record the current state before proposing a transition",
		}}
	}
	if current == target {
		return core.Errors{{
			Code:       ErrorCodeIllegalTransition,
			Severity:   core.SeverityFatal,
			Component:  componentName,
			Field:      field,
			Value:      string(individual),
			Reason:     "individual is already in state " + string(target),
			Suggestion: "treat this action as already applied instead of repeating it",
		}}
	}
	if lifecycle.IsTerminal(current) {
		return core.Errors{{
			Code:      ErrorCodeTerminalState,
			Severity:  core.SeverityFatal,
			Component: componentName,
			Field:     field,
			Value:     string(individual),
			Reason: "state " + string(current) + " is terminal in lifecycle " +
				lifecycle.Name + "; no transition to " + string(target) + " is legal",
			Suggestion: "reject the action; the entity has reached a final state",
		}}
	}
	for _, allowed := range lifecycle.Next(current) {
		if allowed == target {
			return nil
		}
	}
	return core.Errors{{
		Code:      ErrorCodeIllegalTransition,
		Severity:  core.SeverityFatal,
		Component: componentName,
		Field:     field,
		Value:     string(individual),
		Reason: "transition " + string(current) + " -> " + string(target) +
			" is not declared in lifecycle " + lifecycle.Name,
		Suggestion: "legal next states: " + joinStates(lifecycle.Next(current)),
	}}
}

// checkSnapshot validates that recorded states belong to the lifecycle.
// checkSingleState refutes a snapshot recording one individual in two states of
// this lifecycle. A lifecycle state is functional by nature: an individual is
// in one state, not two. Pass an empty individual to check every one.
//
// Nothing refuted this before, and StateOf returns the first match, so an order
// recorded as both paid and refunded read as "paid" and a second refund was
// permitted — defeating the terminal-state guarantee this type exists to give.
func (l Lifecycle) checkSingleState(snapshot Snapshot, only Individual) core.Errors {
	lifecycle := l.Normalize()
	wanted := NewIndividual(string(only))
	seen := map[Individual]State{}
	var errs core.Errors
	for _, assertion := range snapshot.Normalize().States {
		if assertion.Lifecycle != lifecycle.Name {
			continue
		}
		individual := NewIndividual(string(assertion.Individual))
		if only != "" && individual != wanted {
			continue
		}
		previous, ok := seen[individual]
		if !ok {
			seen[individual] = assertion.State
			continue
		}
		if previous == assertion.State {
			continue
		}
		errs = append(errs, core.Error{
			Code:       ErrorCodeFunctionalViolation,
			Severity:   core.SeverityFatal,
			Component:  componentName,
			Field:      "lifecycles." + lifecycle.Name + ".states",
			Value:      string(individual),
			Reason:     "individual is recorded in two states of lifecycle " + lifecycle.Name + ": " + string(previous) + " and " + string(assertion.State),
			Suggestion: "record one current state per individual; a history belongs outside the snapshot",
		})
	}
	return errs
}

func (l Lifecycle) checkSnapshot(snapshot Snapshot) core.Errors {
	lifecycle := l.Normalize()
	var errs core.Errors
	errs = append(errs, lifecycle.checkSingleState(snapshot, "")...)
	for _, assertion := range snapshot.Normalize().States {
		if assertion.Lifecycle != lifecycle.Name {
			continue
		}
		if lifecycle.HasState(assertion.State) {
			continue
		}
		errs = append(errs, core.Error{
			Code:       ErrorCodeUnknownState,
			Severity:   core.SeverityFatal,
			Component:  componentName,
			Field:      "lifecycles." + lifecycle.Name + ".states",
			Value:      string(assertion.State),
			Reason:     "recorded state is not declared by lifecycle " + lifecycle.Name,
			Suggestion: "use one of: " + joinStates(lifecycle.States),
		})
	}
	return errs
}

func joinStates(states []State) string {
	names := make([]string, 0, len(states))
	for _, state := range states {
		names = append(names, string(state))
	}
	if len(names) == 0 {
		return "(none)"
	}
	return strings.Join(names, ", ")
}
