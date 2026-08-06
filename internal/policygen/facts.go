package policygen

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// factNamePattern is the identifier shape a predicate uses to reference a fact.
// A vocabulary that declared a name outside it could not be referenced by any
// schema-valid policy.
var factNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

// maxExactIntegerBound is the largest magnitude a float64 represents exactly.
// Policy bounds are decoded as float64, so a larger integer bound could not be
// written down faithfully, let alone compared against.
const maxExactIntegerBound = 1 << 53

// factKind is the declared type of a fact.
//
// The vocabulary declares the kind; the policy declares the bounds and the
// default. That split puts every decision-bearing *number* in the file the
// PolicyHash covers, so retuning a threshold is always a hash change.
//
// It is not the whole story: a kind selects the parser and an enum's domain is
// baked into the generated frame, so both are decision-bearing too. Those live
// in the vocabulary and move VocabularyHash instead. A consumer recording only
// PolicyHash sees every threshold change and no type change.
type factKind string

const (
	kindNumber  factKind = "number"
	kindInteger factKind = "integer"
	kindBool    factKind = "bool"
	kindEnum    factKind = "enum"
	kindString  factKind = "string"
)

// factDeclSpec is a vocabulary fact declaration.
type factDeclSpec struct {
	Type        string   `yaml:"type"`
	Values      []string `yaml:"values"`
	Description string   `yaml:"description"`
	GoName      string   `yaml:"go_name"`
}

// factBoundsSpec is the per-policy bound and default for a declared fact.
//
// Default is a yaml.Node so an absent key can be told apart from one written
// with no value. Decoding into `any` collapsed both to nil, and a bare
// `default:` then silently made the fact required — the opposite of what the
// author wrote.
type factBoundsSpec struct {
	Min     *float64  `yaml:"min"`
	Max     *float64  `yaml:"max"`
	Default yaml.Node `yaml:"default"`
}

// predicateSpec is one node of a policy predicate.
//
// The form is closed and tagged rather than an expression string: the YAML
// decoder is the parser, and there is no evaluator at runtime because every
// node compiles to a Go boolean expression.
//
// Closure comes from two places. The decoder runs with KnownFields(true), so an
// unknown key is a decode error; validatePredicate below checks a node's arity
// and its operator against the fact's declared type. The JSON Schema states the
// same shape for editors and third-party tooling, but nothing in this repo runs
// it, so it is the published contract rather than the enforcement.
type predicateSpec struct {
	All       []predicateSpec `yaml:"all"`
	Any       []predicateSpec `yaml:"any"`
	Not       *predicateSpec  `yaml:"not"`
	Fact      string          `yaml:"fact"`
	Op        string          `yaml:"op"`
	Value     any             `yaml:"value"`
	FactValue string          `yaml:"fact_value"`
	Values    []any           `yaml:"values"`
	Condition string          `yaml:"condition"`
}

// factBinding is a declared fact resolved against its policy bounds.
type factBinding struct {
	Name       string
	Kind       factKind
	Values     []string
	Field      string
	Min        *float64
	Max        *float64
	Default    any
	HasDefault bool
}

// factIndex holds every declared fact by normalized name.
type factIndex map[string]factBinding

func (i factIndex) names() []string {
	out := make([]string, 0, len(i))
	for name := range i {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// buildFactIndex resolves vocabulary declarations against policy bounds.
func buildFactIndex(vocab vocabularySpec, policy policySpec) (factIndex, error) {
	index := factIndex{}
	fields := map[string]string{}
	for name, decl := range vocab.Facts {
		normalized := normalize(name)
		if normalized == "" {
			return nil, fmt.Errorf("fact name is required")
		}
		if !factNamePattern.MatchString(normalized) {
			return nil, fmt.Errorf("fact name %q is not a valid identifier: no policy predicate could reference it", name)
		}
		kind, err := parseFactKind(normalized, decl)
		if err != nil {
			return nil, err
		}
		field := factFieldName(normalized)
		if previous, ok := fields[field]; ok {
			return nil, fmt.Errorf("facts %q and %q both generate field %q", previous, name, field)
		}
		fields[field] = name
		// Values are compared after normalization, so two that differ only in
		// case or surrounding space would collapse into one and silently
		// shrink the declared domain.
		values := make([]string, 0, len(decl.Values))
		seenValues := map[string]string{}
		for _, value := range decl.Values {
			normalizedValue := normalize(value)
			if previous, ok := seenValues[normalizedValue]; ok {
				return nil, fmt.Errorf("fact %q declares values %q and %q, which are the same value", name, previous, value)
			}
			seenValues[normalizedValue] = value
			values = append(values, normalizedValue)
		}
		index[normalized] = factBinding{Name: normalized, Kind: kind, Values: values, Field: field}
	}
	for name, bounds := range policy.Facts {
		normalized := normalize(name)
		binding, ok := index[normalized]
		if !ok {
			return nil, fmt.Errorf("policy declares bounds for undeclared fact %q", name)
		}
		if err := applyFactBounds(&binding, bounds); err != nil {
			return nil, err
		}
		index[normalized] = binding
	}
	return index, nil
}

func parseFactKind(name string, decl factDeclSpec) (factKind, error) {
	kind := factKind(normalize(decl.Type))
	switch kind {
	case kindNumber, kindInteger, kindBool, kindString:
		if len(decl.Values) > 0 {
			return "", fmt.Errorf("fact %q of type %s must not declare values", name, kind)
		}
	case kindEnum:
		if len(decl.Values) == 0 {
			return "", fmt.Errorf("fact %q of type enum needs values", name)
		}
	case "":
		return "", fmt.Errorf("fact %q missing type", name)
	default:
		return "", fmt.Errorf("fact %q has unknown type %q", name, decl.Type)
	}
	return kind, nil
}

func applyFactBounds(binding *factBinding, bounds factBoundsSpec) error {
	if bounds.Min != nil || bounds.Max != nil {
		if binding.Kind != kindNumber && binding.Kind != kindInteger {
			return fmt.Errorf("fact %q of type %s cannot declare min or max", binding.Name, binding.Kind)
		}
		if err := checkFinite(binding.Name, bounds.Min); err != nil {
			return err
		}
		if err := checkFinite(binding.Name, bounds.Max); err != nil {
			return err
		}
		if bounds.Min != nil && bounds.Max != nil && *bounds.Min > *bounds.Max {
			return fmt.Errorf("fact %q declares min greater than max", binding.Name)
		}
		// An integer fact compares as int64, so a fractional bound would be
		// silently truncated into a different threshold than the one authored.
		// Beyond 2^53 a float64 cannot represent every integer, and the
		// generator's int64 conversion of an out-of-range value is
		// implementation-defined: a bound above 2^63 became MinInt64, which
		// silently drops the bound instead of enforcing it.
		if binding.Kind == kindInteger {
			for _, bound := range []*float64{bounds.Min, bounds.Max} {
				if bound == nil {
					continue
				}
				if *bound != math.Trunc(*bound) {
					return fmt.Errorf("fact %q is an integer and cannot declare a fractional bound", binding.Name)
				}
				if math.Abs(*bound) > maxExactIntegerBound {
					return fmt.Errorf("fact %q declares the integer bound %v, which is too large to represent exactly", binding.Name, *bound)
				}
			}
		}
		binding.Min = bounds.Min
		binding.Max = bounds.Max
	}
	// Kind 0 means the key was never written. A key written with no value is a
	// different thing and almost never what the author meant: it used to make
	// the fact required, which is the opposite of declaring a default.
	if bounds.Default.Kind == 0 {
		return nil
	}
	if bounds.Default.Tag == "!!null" {
		return fmt.Errorf("fact %q declares an empty default: write a value, or remove the key to make the fact required", binding.Name)
	}
	var raw any
	if err := bounds.Default.Decode(&raw); err != nil {
		return fmt.Errorf("fact %q default: %w", binding.Name, err)
	}
	value, err := coerceFactValue(*binding, raw)
	if err != nil {
		return fmt.Errorf("fact %q default: %w", binding.Name, err)
	}
	// The generated parser returns the default before it reaches the bounds
	// checks, so a default outside min/max is the one value the policy accepts
	// while declaring it invalid: absence would allow exactly what presence
	// refuses.
	if err := checkDefaultWithinBounds(*binding, value); err != nil {
		return err
	}
	binding.Default = value
	binding.HasDefault = true
	return nil
}

func checkDefaultWithinBounds(binding factBinding, value any) error {
	var numeric float64
	switch typed := value.(type) {
	case float64:
		numeric = typed
	case int64:
		numeric = float64(typed)
	default:
		return nil
	}
	if binding.Min != nil && numeric < *binding.Min {
		return fmt.Errorf("fact %q declares the default %v, which is below its own minimum %v", binding.Name, numeric, *binding.Min)
	}
	if binding.Max != nil && numeric > *binding.Max {
		return fmt.Errorf("fact %q declares the default %v, which is above its own maximum %v", binding.Name, numeric, *binding.Max)
	}
	return nil
}

func checkFinite(name string, value *float64) error {
	if value == nil {
		return nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) {
		return fmt.Errorf("fact %q declares a non-finite bound", name)
	}
	return nil
}

// coerceFactValue converts a YAML scalar to the fact's declared Go type.
func coerceFactValue(binding factBinding, raw any) (any, error) {
	switch binding.Kind {
	case kindNumber:
		value, ok := toFloat(raw)
		if !ok {
			return nil, fmt.Errorf("expected a number, got %v", raw)
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("value is not finite")
		}
		return value, nil
	case kindInteger:
		switch typed := raw.(type) {
		case int:
			return int64(typed), nil
		case int64:
			return typed, nil
		default:
			return nil, fmt.Errorf("expected an integer, got %v", raw)
		}
	case kindBool:
		value, ok := raw.(bool)
		if !ok {
			return nil, fmt.Errorf("expected a bool, got %v", raw)
		}
		return value, nil
	case kindEnum:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string, got %v", raw)
		}
		normalized := normalize(value)
		if !containsString(binding.Values, normalized) {
			return nil, fmt.Errorf("value %q is outside the declared domain %v", value, binding.Values)
		}
		return normalized, nil
	case kindString:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string, got %v", raw)
		}
		// The generated reader trims every metadata value and treats a blank
		// one as absent, so a literal with surrounding space or an empty
		// literal can never match. Accepting it would compile a rule that is
		// dead by construction.
		if value == "" {
			return nil, fmt.Errorf("value is empty, and a blank fact counts as absent")
		}
		if strings.TrimSpace(value) != value {
			return nil, fmt.Errorf("value %q has surrounding whitespace, which the runtime trims, so it could never match", value)
		}
		return value, nil
	}
	return nil, fmt.Errorf("unknown fact type %q", binding.Kind)
}

func toFloat(raw any) (float64, bool) {
	switch typed := raw.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	}
	return 0, false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// factFieldName derives the unexported frame field for a fact.
func factFieldName(name string) string {
	var b strings.Builder
	b.WriteString("fact")
	upperNext := true
	for _, r := range name {
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

// numericOps, and the tables below, are what keeps the grammar closed. An
// operator that is not listed for a fact's type fails generation.
var opsByKind = map[factKind]map[string]struct{}{
	kindNumber:  {"eq": {}, "ne": {}, "gt": {}, "gte": {}, "lt": {}, "lte": {}, "in": {}},
	kindInteger: {"eq": {}, "ne": {}, "gt": {}, "gte": {}, "lt": {}, "lte": {}, "in": {}},
	kindBool:    {"eq": {}, "ne": {}, "is_true": {}, "is_false": {}},
	kindEnum:    {"eq": {}, "ne": {}, "in": {}},
	kindString:  {"eq": {}, "ne": {}, "in": {}},
}

// validatePredicate checks one predicate tree and records the facts it reads.
func validatePredicate(node predicateSpec, facts factIndex, conditions map[string]struct{}, referenced map[string]struct{}, path string) error {
	forms := 0
	if len(node.All) > 0 {
		forms++
	}
	if len(node.Any) > 0 {
		forms++
	}
	if node.Not != nil {
		forms++
	}
	if node.Fact != "" {
		forms++
	}
	if node.Condition != "" {
		forms++
	}
	if forms != 1 {
		return fmt.Errorf("predicate at %s must be exactly one of all, any, not, a fact comparison, or a condition", path)
	}

	switch {
	case len(node.All) > 0:
		for i, child := range node.All {
			if err := validatePredicate(child, facts, conditions, referenced, fmt.Sprintf("%s.all[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	case len(node.Any) > 0:
		for i, child := range node.Any {
			if err := validatePredicate(child, facts, conditions, referenced, fmt.Sprintf("%s.any[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	case node.Not != nil:
		return validatePredicate(*node.Not, facts, conditions, referenced, path+".not")
	case node.Condition != "":
		normalized := normalize(node.Condition)
		if _, ok := conditions[normalized]; !ok {
			return fmt.Errorf("predicate at %s requires undeclared condition %q", path, node.Condition)
		}
		return nil
	}
	return validateFactLeaf(node, facts, referenced, path)
}

func validateFactLeaf(node predicateSpec, facts factIndex, referenced map[string]struct{}, path string) error {
	name := normalize(node.Fact)
	binding, ok := facts[name]
	if !ok {
		return fmt.Errorf("predicate at %s reads undeclared fact %q", path, node.Fact)
	}
	op := normalize(node.Op)
	allowed, ok := opsByKind[binding.Kind]
	if !ok {
		return fmt.Errorf("predicate at %s reads fact %q of unknown type", path, node.Fact)
	}
	if _, ok := allowed[op]; !ok {
		return fmt.Errorf("predicate at %s uses operator %q, which is not valid for a %s fact", path, node.Op, binding.Kind)
	}
	referenced[name] = struct{}{}

	switch op {
	case "is_true", "is_false":
		if node.Value != nil || node.FactValue != "" || len(node.Values) > 0 {
			return fmt.Errorf("predicate at %s uses %s, which takes no value", path, op)
		}
		return nil
	case "in":
		if len(node.Values) == 0 {
			return fmt.Errorf("predicate at %s uses in and needs values", path)
		}
		if node.Value != nil || node.FactValue != "" {
			return fmt.Errorf("predicate at %s uses in and must not also set value or fact_value", path)
		}
		for i, raw := range node.Values {
			if _, err := coerceFactValue(binding, raw); err != nil {
				return fmt.Errorf("predicate at %s values[%d]: %w", path, i, err)
			}
		}
		return nil
	}

	if len(node.Values) > 0 {
		return fmt.Errorf("predicate at %s uses %s and must not set values", path, op)
	}
	if node.FactValue != "" {
		if node.Value != nil {
			return fmt.Errorf("predicate at %s sets both value and fact_value", path)
		}
		return validateFactComparison(binding, node, facts, referenced, path)
	}
	if node.Value == nil {
		return fmt.Errorf("predicate at %s uses %s and needs value or fact_value", path, op)
	}
	if _, err := coerceFactValue(binding, node.Value); err != nil {
		return fmt.Errorf("predicate at %s value: %w", path, err)
	}
	return nil
}

// validateFactComparison keeps fact-to-fact comparison strict: both sides must
// carry the same declared type, so no implicit conversion happens where a
// threshold is being decided.
func validateFactComparison(binding factBinding, node predicateSpec, facts factIndex, referenced map[string]struct{}, path string) error {
	otherName := normalize(node.FactValue)
	other, ok := facts[otherName]
	if !ok {
		return fmt.Errorf("predicate at %s compares against undeclared fact %q", path, node.FactValue)
	}
	if other.Kind != binding.Kind {
		return fmt.Errorf("predicate at %s compares %s fact %q against %s fact %q", path, binding.Kind, binding.Name, other.Kind, other.Name)
	}
	if other.Name == binding.Name {
		return fmt.Errorf("predicate at %s compares fact %q against itself", path, binding.Name)
	}
	referenced[otherName] = struct{}{}
	return nil
}

// describePredicate renders a predicate as the static string a trace step
// reports when the node matched.
func describePredicate(node predicateSpec) string {
	switch {
	case len(node.All) > 0:
		return "all(" + describeChildren(node.All) + ")"
	case len(node.Any) > 0:
		return "any(" + describeChildren(node.Any) + ")"
	case node.Not != nil:
		return "not(" + describePredicate(*node.Not) + ")"
	case node.Condition != "":
		return "condition " + normalize(node.Condition)
	}
	op := normalize(node.Op)
	switch op {
	case "is_true", "is_false":
		return fmt.Sprintf("%s %s", normalize(node.Fact), op)
	case "in":
		parts := make([]string, 0, len(node.Values))
		for _, value := range node.Values {
			parts = append(parts, describeScalar(value))
		}
		return fmt.Sprintf("%s in [%s]", normalize(node.Fact), strings.Join(parts, ", "))
	}
	if node.FactValue != "" {
		return fmt.Sprintf("%s %s %s", normalize(node.Fact), op, normalize(node.FactValue))
	}
	return fmt.Sprintf("%s %s %s", normalize(node.Fact), op, describeScalar(node.Value))
}

func describeChildren(children []predicateSpec) string {
	parts := make([]string, 0, len(children))
	for _, child := range children {
		parts = append(parts, describePredicate(child))
	}
	return strings.Join(parts, ", ")
}

func describeScalar(raw any) string {
	switch typed := raw.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64)
	}
	return fmt.Sprintf("%v", raw)
}
