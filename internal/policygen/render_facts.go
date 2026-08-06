package policygen

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

var goComparisons = map[string]string{
	"eq":  "==",
	"ne":  "!=",
	"gt":  ">",
	"gte": ">=",
	"lt":  "<",
	"lte": "<=",
}

// buildRenderModel compiles every `when` predicate into named Go functions.
//
// There is no evaluator: a predicate becomes a boolean expression the Go
// compiler checks, which is why this feature needs no expression engine and no
// third-party dependency.
func buildRenderModel(policy policySpec, plan policyPlan) (renderModel, error) {
	model := renderModel{
		plan:       plan,
		denyMatch:  map[int]string{},
		routeMatch: map[string]string{},
	}
	builder := &matcherBuilder{facts: plan.facts}
	// One counter across denies and routes keeps every generated matcher name
	// unique even when two rules share an id, which is what a disjunction split
	// across two denies looks like.
	rule := 0
	for i, deny := range policy.Denies {
		if deny.When == nil {
			continue
		}
		name := builder.rootName(rule, deny.ID)
		builder.build(name, *deny.When)
		model.denyMatch[i] = name
		rule++
	}
	for _, route := range sortedRoutes(policy.Routes) {
		if route.When == nil {
			continue
		}
		name := builder.rootName(rule, route.ID)
		builder.build(name, *route.When)
		model.routeMatch[normalize(route.Operation)] = name
		rule++
	}
	if builder.err != nil {
		return renderModel{}, builder.err
	}
	model.matchers = builder.out
	for _, binding := range plan.bindings() {
		switch binding.Kind {
		case kindNumber:
			model.needsStrconv = true
			model.needsMath = true
		case kindInteger, kindBool:
			model.needsStrconv = true
		}
	}
	return model, nil
}

// matcherBuilder emits one function per predicate node that needs its own
// scope. Fact leaves stay inline as expressions, which is what keeps a simple
// rule readable as a single `if`.
type matcherBuilder struct {
	facts factIndex
	base  string
	next  int
	out   []renderedMatcher
	err   error
}

func (m *matcherBuilder) rootName(index int, id string) string {
	m.base = fmt.Sprintf("matchRule%d%s", index, exportIdent(id))
	m.next = 0
	return m.base
}

func (m *matcherBuilder) allocate(node predicateSpec) string {
	m.next++
	name := fmt.Sprintf("%sNode%d", m.base, m.next)
	m.build(name, node)
	return name
}

func (m *matcherBuilder) build(name string, node predicateSpec) {
	var b bytes.Buffer
	m.writeBody(&b, node)
	m.out = append(m.out, renderedMatcher{name: name, comment: describePredicate(node), body: b.String()})
}

func (m *matcherBuilder) writeBody(b *bytes.Buffer, node predicateSpec) {
	switch {
	case len(node.All) > 0:
		m.writeAll(b, node)
	case len(node.Any) > 0:
		m.writeAny(b, node)
	case node.Not != nil:
		m.writeNot(b, node)
	case node.Condition != "":
		m.writeCondition(b, node)
	default:
		m.writeLeaf(b, node)
	}
}

func (m *matcherBuilder) writeLeaf(b *bytes.Buffer, node predicateSpec) {
	fmt.Fprintf(b, "if %s {\nreturn true, %q, nil\n}\n", m.expr(node), describePredicate(node))
	fmt.Fprintln(b, `return false, "", nil`)
}

func (m *matcherBuilder) writeCondition(b *bytes.Buffer, node predicateSpec) {
	name := normalize(node.Condition)
	fmt.Fprintf(b, "holds, err := e.conditionsHold(ctx, []cervorules.Condition{cervorules.Condition(%q)}, req)\n", name)
	fmt.Fprintln(b, `if err != nil { return false, "", err }`)
	fmt.Fprintf(b, "if holds { return true, %q, nil }\n", "condition "+name)
	fmt.Fprintln(b, `return false, "", nil`)
}

func (m *matcherBuilder) writeAny(b *bytes.Buffer, node predicateSpec) {
	for i, child := range node.Any {
		if isFactLeaf(child) {
			fmt.Fprintf(b, "if %s {\nreturn true, %q, nil\n}\n", m.expr(child), describePredicate(child))
			continue
		}
		sub := m.allocate(child)
		fmt.Fprintf(b, "ok%d, detail%d, err%d := %s(ctx, e, req, f)\n", i, i, i, sub)
		fmt.Fprintf(b, "if err%d != nil { return false, \"\", err%d }\n", i, i)
		fmt.Fprintf(b, "if ok%d { return true, detail%d, nil }\n", i, i)
	}
	fmt.Fprintln(b, `return false, "", nil`)
}

func (m *matcherBuilder) writeAll(b *bytes.Buffer, node predicateSpec) {
	for i, child := range node.All {
		if isFactLeaf(child) {
			fmt.Fprintf(b, "if !(%s) { return false, \"\", nil }\n", m.expr(child))
			continue
		}
		sub := m.allocate(child)
		fmt.Fprintf(b, "ok%d, _, err%d := %s(ctx, e, req, f)\n", i, i, sub)
		fmt.Fprintf(b, "if err%d != nil { return false, \"\", err%d }\n", i, i)
		fmt.Fprintf(b, "if !ok%d { return false, \"\", nil }\n", i)
	}
	fmt.Fprintf(b, "return true, %q, nil\n", describePredicate(node))
}

func (m *matcherBuilder) writeNot(b *bytes.Buffer, node predicateSpec) {
	child := *node.Not
	if isFactLeaf(child) {
		fmt.Fprintf(b, "if %s { return false, \"\", nil }\n", m.expr(child))
	} else {
		sub := m.allocate(child)
		fmt.Fprintf(b, "ok, _, err := %s(ctx, e, req, f)\n", sub)
		fmt.Fprintln(b, `if err != nil { return false, "", err }`)
		fmt.Fprintln(b, `if ok { return false, "", nil }`)
	}
	fmt.Fprintf(b, "return true, %q, nil\n", describePredicate(node))
}

// expr renders a fact leaf as a Go boolean expression.
func (m *matcherBuilder) expr(node predicateSpec) string {
	binding, ok := m.facts[normalize(node.Fact)]
	if !ok {
		m.fail(fmt.Errorf("predicate reads undeclared fact %q", node.Fact))
		return "false"
	}
	field := "f." + binding.Field
	switch normalize(node.Op) {
	case "is_true":
		return field
	case "is_false":
		return "!" + field
	case "in":
		parts := make([]string, 0, len(node.Values))
		for _, value := range node.Values {
			parts = append(parts, field+" == "+m.literal(binding, value))
		}
		return "(" + strings.Join(parts, " || ") + ")"
	}
	comparison, ok := goComparisons[normalize(node.Op)]
	if !ok {
		m.fail(fmt.Errorf("predicate on fact %q uses unsupported operator %q", node.Fact, node.Op))
		return "false"
	}
	if node.FactValue != "" {
		other, ok := m.facts[normalize(node.FactValue)]
		if !ok {
			m.fail(fmt.Errorf("predicate compares against undeclared fact %q", node.FactValue))
			return "false"
		}
		return fmt.Sprintf("%s %s f.%s", field, comparison, other.Field)
	}
	return fmt.Sprintf("%s %s %s", field, comparison, m.literal(binding, node.Value))
}

func (m *matcherBuilder) literal(binding factBinding, raw any) string {
	value, err := coerceFactValue(binding, raw)
	if err != nil {
		m.fail(fmt.Errorf("fact %q: %w", binding.Name, err))
		return "0"
	}
	switch typed := value.(type) {
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64)
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		return strconv.FormatBool(typed)
	case string:
		return strconv.Quote(typed)
	}
	m.fail(fmt.Errorf("fact %q has an unrenderable value", binding.Name))
	return "0"
}

func (m *matcherBuilder) fail(err error) {
	if m.err == nil {
		m.err = err
	}
}

func isFactLeaf(node predicateSpec) bool {
	return node.Fact != "" && len(node.All) == 0 && len(node.Any) == 0 && node.Not == nil && node.Condition == ""
}

func writeFactFrame(b *bytes.Buffer, model renderModel) {
	if !model.hasMatchers() {
		return
	}
	bindings := model.plan.bindings()
	fmt.Fprintln(b, "// factFrame holds the facts this policy reads, parsed once per decision.")
	fmt.Fprintln(b, "type factFrame struct {")
	for _, binding := range bindings {
		fmt.Fprintf(b, "%s %s\n", binding.Field, goTypeFor(binding.Kind))
	}
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "// newFactFrame parses every fact this policy reads before any rule runs.")
	fmt.Fprintln(b, "// A fact that is absent, unparseable, non-finite or outside its declared")
	fmt.Fprintln(b, "// bounds fails the decision with a structured error. It never reports a")
	fmt.Fprintln(b, "// predicate as unsatisfied, which would read as \"the guard ran and found")
	fmt.Fprintln(b, "// nothing wrong\".")
	fmt.Fprintln(b, "func newFactFrame(req cervorules.Request) (factFrame, error) {")
	fmt.Fprintln(b, "var frame factFrame")
	if len(bindings) > 0 {
		fmt.Fprintln(b, "var err error")
	}
	for _, binding := range bindings {
		fmt.Fprintf(b, "if frame.%s, err = %s; err != nil { return factFrame{}, err }\n", binding.Field, factCall(binding))
	}
	fmt.Fprintln(b, "return frame, nil")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
	writeFactHelpers(b, bindings)
	fmt.Fprintln(b, "// ruleMatcher is a compiled predicate. It reports whether the rule matched")
	fmt.Fprintln(b, "// and which leaf decided it, so a trace step can name the reason.")
	fmt.Fprintln(b, "type ruleMatcher func(ctx context.Context, e generatedEngine, req cervorules.Request, f factFrame) (bool, string, error)")
	fmt.Fprintln(b)
}

func writeMatchers(b *bytes.Buffer, model renderModel) {
	for _, matcher := range model.matchers {
		fmt.Fprintf(b, "// %s\n", commentSafe(matcher.comment))
		fmt.Fprintf(b, "func %s(ctx context.Context, e generatedEngine, req cervorules.Request, f factFrame) (bool, string, error) {\n", matcher.name)
		fmt.Fprint(b, matcher.body)
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
	}
}

func goTypeFor(kind factKind) string {
	switch kind {
	case kindNumber:
		return "float64"
	case kindInteger:
		return "int64"
	case kindBool:
		return "bool"
	}
	return "string"
}

func factCall(binding factBinding) string {
	switch binding.Kind {
	case kindNumber:
		return fmt.Sprintf("factNumberValue(req, %q, %s)", binding.Name, numberBoundsLiteral(binding))
	case kindInteger:
		return fmt.Sprintf("factIntegerValue(req, %q, %s)", binding.Name, integerBoundsLiteral(binding))
	case kindBool:
		return fmt.Sprintf("factBoolValue(req, %q, %s)", binding.Name, boolBoundsLiteral(binding))
	case kindEnum:
		return fmt.Sprintf("factEnumValue(req, %q, %s, %s)", binding.Name, stringSliceLiteral(binding.Values), stringBoundsLiteral(binding))
	}
	return fmt.Sprintf("factStringValue(req, %q, %s)", binding.Name, stringBoundsLiteral(binding))
}

func numberBoundsLiteral(binding factBinding) string {
	fields := []string{}
	if binding.HasDefault {
		fields = append(fields, "hasDefault: true", "def: "+strconv.FormatFloat(binding.Default.(float64), 'g', -1, 64))
	}
	if binding.Min != nil {
		fields = append(fields, "hasMin: true", "min: "+strconv.FormatFloat(*binding.Min, 'g', -1, 64))
	}
	if binding.Max != nil {
		fields = append(fields, "hasMax: true", "max: "+strconv.FormatFloat(*binding.Max, 'g', -1, 64))
	}
	return "factNumberBounds{" + strings.Join(fields, ", ") + "}"
}

func integerBoundsLiteral(binding factBinding) string {
	fields := []string{}
	if binding.HasDefault {
		fields = append(fields, "hasDefault: true", "def: "+strconv.FormatInt(binding.Default.(int64), 10))
	}
	if binding.Min != nil {
		fields = append(fields, "hasMin: true", "min: "+strconv.FormatInt(int64(*binding.Min), 10))
	}
	if binding.Max != nil {
		fields = append(fields, "hasMax: true", "max: "+strconv.FormatInt(int64(*binding.Max), 10))
	}
	return "factIntegerBounds{" + strings.Join(fields, ", ") + "}"
}

func boolBoundsLiteral(binding factBinding) string {
	if !binding.HasDefault {
		return "factBoolBounds{}"
	}
	return "factBoolBounds{hasDefault: true, def: " + strconv.FormatBool(binding.Default.(bool)) + "}"
}

func stringBoundsLiteral(binding factBinding) string {
	if !binding.HasDefault {
		return "factStringBounds{}"
	}
	return "factStringBounds{hasDefault: true, def: " + strconv.Quote(binding.Default.(string)) + "}"
}

func stringSliceLiteral(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Quote(value))
	}
	return "[]string{" + strings.Join(parts, ", ") + "}"
}

// writeFactHelpers emits only the parsers the policy actually uses.
func writeFactHelpers(b *bytes.Buffer, bindings []factBinding) {
	if len(bindings) == 0 {
		return
	}
	kinds := map[factKind]bool{}
	for _, binding := range bindings {
		kinds[binding.Kind] = true
	}

	fmt.Fprintln(b, "// factRawValue reports the trimmed metadata value for a fact. A blank value")
	fmt.Fprintln(b, "// counts as absent: an empty string is not an answer.")
	fmt.Fprintln(b, "func factRawValue(req cervorules.Request, name string) (string, bool) {")
	fmt.Fprintln(b, "raw, ok := req.Metadata[name]")
	fmt.Fprintln(b, `if !ok { return "", false }`)
	fmt.Fprintln(b, "raw = strings.TrimSpace(raw)")
	fmt.Fprintln(b, `if raw == "" { return "", false }`)
	fmt.Fprintln(b, "return raw, true")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "func missingFactError(name string) error {")
	fmt.Fprintln(b, "return cervorules.Error{")
	fmt.Fprintln(b, "Code: cervorules.ErrorCodeMissingFact,")
	fmt.Fprintln(b, "Severity: cervorules.SeverityFatal,")
	fmt.Fprintln(b, "Component: \"policy\",")
	fmt.Fprintln(b, `Field: "facts." + name,`)
	fmt.Fprintln(b, "Reason: \"fact is absent from request metadata and the policy declares no default\",")
	fmt.Fprintln(b, "Suggestion: \"supply the fact in Request.Metadata, or declare a default for it in the policy\",")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)
	fmt.Fprintln(b, "// invalidFactError marks the observed value sensitive: only the caller knows")
	fmt.Fprintln(b, "// whether a fact carries secrets, so it is withheld from serialization by")
	fmt.Fprintln(b, "// default and still available in process.")
	fmt.Fprintln(b, "func invalidFactError(name string, value string, reason string) error {")
	fmt.Fprintln(b, "return cervorules.Error{")
	fmt.Fprintln(b, "Code: cervorules.ErrorCodeInvalidFact,")
	fmt.Fprintln(b, "Severity: cervorules.SeverityFatal,")
	fmt.Fprintln(b, "Component: \"policy\",")
	fmt.Fprintln(b, `Field: "facts." + name,`)
	fmt.Fprintln(b, "Value: value,")
	fmt.Fprintln(b, "Reason: reason,")
	fmt.Fprintln(b, "Sensitive: true,")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b, "}")
	fmt.Fprintln(b)

	if kinds[kindNumber] {
		fmt.Fprintln(b, "type factNumberBounds struct {")
		fmt.Fprintln(b, "hasDefault bool")
		fmt.Fprintln(b, "def float64")
		fmt.Fprintln(b, "hasMin bool")
		fmt.Fprintln(b, "min float64")
		fmt.Fprintln(b, "hasMax bool")
		fmt.Fprintln(b, "max float64")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
		fmt.Fprintln(b, "func factNumberValue(req cervorules.Request, name string, bounds factNumberBounds) (float64, error) {")
		fmt.Fprintln(b, "raw, ok := factRawValue(req, name)")
		fmt.Fprintln(b, "if !ok {")
		fmt.Fprintln(b, "if bounds.hasDefault { return bounds.def, nil }")
		fmt.Fprintln(b, "return 0, missingFactError(name)")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b, "value, err := strconv.ParseFloat(raw, 64)")
		fmt.Fprintln(b, "if err != nil { return 0, invalidFactError(name, raw, \"fact is not a number\") }")
		fmt.Fprintln(b, "// ParseFloat accepts \"NaN\" and \"Inf\". A non-finite value passes every")
		fmt.Fprintln(b, "// comparison in this policy without matching any of them, so it would")
		fmt.Fprintln(b, "// slip through the deny rules and land on the allow path.")
		fmt.Fprintln(b, "if math.IsNaN(value) || math.IsInf(value, 0) { return 0, invalidFactError(name, raw, \"fact is not a finite number\") }")
		fmt.Fprintln(b, "if bounds.hasMin && value < bounds.min { return 0, invalidFactError(name, raw, \"fact is below its declared minimum\") }")
		fmt.Fprintln(b, "if bounds.hasMax && value > bounds.max { return 0, invalidFactError(name, raw, \"fact is above its declared maximum\") }")
		fmt.Fprintln(b, "return value, nil")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
	}
	if kinds[kindInteger] {
		fmt.Fprintln(b, "type factIntegerBounds struct {")
		fmt.Fprintln(b, "hasDefault bool")
		fmt.Fprintln(b, "def int64")
		fmt.Fprintln(b, "hasMin bool")
		fmt.Fprintln(b, "min int64")
		fmt.Fprintln(b, "hasMax bool")
		fmt.Fprintln(b, "max int64")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
		fmt.Fprintln(b, "func factIntegerValue(req cervorules.Request, name string, bounds factIntegerBounds) (int64, error) {")
		fmt.Fprintln(b, "raw, ok := factRawValue(req, name)")
		fmt.Fprintln(b, "if !ok {")
		fmt.Fprintln(b, "if bounds.hasDefault { return bounds.def, nil }")
		fmt.Fprintln(b, "return 0, missingFactError(name)")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b, "value, err := strconv.ParseInt(raw, 10, 64)")
		fmt.Fprintln(b, "if err != nil { return 0, invalidFactError(name, raw, \"fact is not an integer\") }")
		fmt.Fprintln(b, "if bounds.hasMin && value < bounds.min { return 0, invalidFactError(name, raw, \"fact is below its declared minimum\") }")
		fmt.Fprintln(b, "if bounds.hasMax && value > bounds.max { return 0, invalidFactError(name, raw, \"fact is above its declared maximum\") }")
		fmt.Fprintln(b, "return value, nil")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
	}
	if kinds[kindBool] {
		fmt.Fprintln(b, "type factBoolBounds struct {")
		fmt.Fprintln(b, "hasDefault bool")
		fmt.Fprintln(b, "def bool")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
		fmt.Fprintln(b, "func factBoolValue(req cervorules.Request, name string, bounds factBoolBounds) (bool, error) {")
		fmt.Fprintln(b, "raw, ok := factRawValue(req, name)")
		fmt.Fprintln(b, "if !ok {")
		fmt.Fprintln(b, "if bounds.hasDefault { return bounds.def, nil }")
		fmt.Fprintln(b, "return false, missingFactError(name)")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b, "value, err := strconv.ParseBool(raw)")
		fmt.Fprintln(b, "if err != nil { return false, invalidFactError(name, raw, \"fact is not a bool\") }")
		fmt.Fprintln(b, "return value, nil")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
	}
	if kinds[kindEnum] || kinds[kindString] {
		fmt.Fprintln(b, "type factStringBounds struct {")
		fmt.Fprintln(b, "hasDefault bool")
		fmt.Fprintln(b, "def string")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
	}
	if kinds[kindEnum] {
		fmt.Fprintln(b, "func factEnumValue(req cervorules.Request, name string, allowed []string, bounds factStringBounds) (string, error) {")
		fmt.Fprintln(b, "raw, ok := factRawValue(req, name)")
		fmt.Fprintln(b, "if !ok {")
		fmt.Fprintln(b, "if bounds.hasDefault { return bounds.def, nil }")
		fmt.Fprintln(b, `return "", missingFactError(name)`)
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b, "value := strings.ToLower(raw)")
		fmt.Fprintln(b, "for _, candidate := range allowed {")
		fmt.Fprintln(b, "if value == candidate { return value, nil }")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b, `return "", invalidFactError(name, raw, "fact is outside its declared domain")`)
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
	}
	if kinds[kindString] {
		fmt.Fprintln(b, "func factStringValue(req cervorules.Request, name string, bounds factStringBounds) (string, error) {")
		fmt.Fprintln(b, "raw, ok := factRawValue(req, name)")
		fmt.Fprintln(b, "if !ok {")
		fmt.Fprintln(b, "if bounds.hasDefault { return bounds.def, nil }")
		fmt.Fprintln(b, `return "", missingFactError(name)`)
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b, "return raw, nil")
		fmt.Fprintln(b, "}")
		fmt.Fprintln(b)
	}
}

// commentSafe makes author-controlled text safe to put in a Go line comment.
//
// A predicate description embeds the policy's own string literals. A literal
// containing a newline used to terminate the comment, so everything after it
// was parsed as top-level Go: a policy could inject arbitrary declarations —
// including a func init() that runs on package import — into the generated
// engine. `check` reported the policy ok, `generate` exited 0, and gofmt
// accepted the result, because the injected text was valid Go.
//
// That inverts what the DSL is for. The YAML is the artifact a reviewer reads;
// the generated Go is what ships.
func commentSafe(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteRune(' ')
		case r < 0x20 || r == 0x7f:
			// Other control characters have no business in a comment and can
			// hide text from a reader of the generated file.
			continue
		case r == '\uFEFF':
			// A byte order mark is not a control character by codepoint, but
			// go/scanner rejects it anywhere past offset 0 — including inside
			// a comment. Left in, it turns a policy `check` accepts into a
			// generation failure with an opaque parser error.
			continue
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// exportIdent turns a rule id into a Go identifier fragment.
func exportIdent(value string) string {
	var b strings.Builder
	upperNext := true
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if upperNext {
				b.WriteRune(unicode.ToUpper(r))
				upperNext = false
				continue
			}
			b.WriteRune(r)
			continue
		}
		upperNext = true
	}
	return b.String()
}
