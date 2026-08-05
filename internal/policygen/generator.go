package policygen

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/format"
	"io"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	supportedPolicyVersion = "cervorules.policy.v3"
	defaultCervoRulesPath  = "github.com/cervantesh/cervo-rules/v3"
)

var packagePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Options controls v3 policy generation.
type Options struct {
	PackageName       string
	VocabularyPackage string
	CervoRulesImport  string
	VocabularyImport  string
	VocabularyReader  io.Reader
	PolicyReader      io.Reader
	GeneratedTests    bool
}

// Output carries generated source and machine-readable metadata.
type Output struct {
	Source     string
	TestSource string
	Metadata   PolicyMetadata
	Snapshot   CodegenSnapshot
}

// PolicyMetadata mirrors runtime.PolicyMetadata with generator identity.
type PolicyMetadata struct {
	Name           string `json:"name"`
	DSLVersion     string `json:"dsl_version"`
	GeneratedWith  string `json:"generated_with"`
	VocabularyHash string `json:"vocabulary_hash"`
	PolicyHash     string `json:"policy_hash"`
}

// CodegenSnapshot records stable generation counts.
type CodegenSnapshot struct {
	PolicyName string `json:"policy_name"`
	DSLVersion string `json:"dsl_version"`
	Operations int    `json:"operations"`
	Targets    int    `json:"targets"`
	Executors  int    `json:"executors"`
	Routes     int    `json:"routes"`
	Denies     int    `json:"denies"`
	Tests      int    `json:"tests"`
	Conditions int    `json:"conditions"`
}

type vocabularySpec struct {
	Operations map[string]vocabularyEntry `yaml:"operations"`
	Targets    map[string]vocabularyEntry `yaml:"targets"`
	Executors  map[string]vocabularyEntry `yaml:"executors"`
}

type vocabularyEntry struct {
	Description string `yaml:"description"`
	GoName      string `yaml:"go_name"`
}

type policySpec struct {
	Version      string       `yaml:"version"`
	Name         string       `yaml:"name"`
	TrustedUsers []string     `yaml:"trusted_users"`
	Defaults     defaultsSpec `yaml:"defaults"`
	Routes       []routeSpec  `yaml:"routes"`
	Denies       []denySpec   `yaml:"denies"`
	Tests        []testSpec   `yaml:"tests"`

	Conditions map[string]conditionSpec `yaml:"conditions"`
}

// conditionSpec declares a named guard the policy can require. It mirrors the
// JSON shape of ontology.Check, but the generator never imports ontology: a
// condition is only a name plus the data a consumer needs to bind an evaluator.
type conditionSpec struct {
	Kind        string   `yaml:"kind"`
	Lifecycle   string   `yaml:"lifecycle"`
	SubjectKey  string   `yaml:"subject_key"`
	To          string   `yaml:"to"`
	States      []string `yaml:"states"`
	EntityType  string   `yaml:"entity_type"`
	Description string   `yaml:"description"`
}

type defaultsSpec struct {
	Executor string `yaml:"executor"`
}

type routeSpec struct {
	ID                  string   `yaml:"id"`
	Operation           string   `yaml:"operation"`
	Target              string   `yaml:"target"`
	Executor            string   `yaml:"executor"`
	FallbackExecutors   []string `yaml:"fallback_executors"`
	DisabledByDefault   bool     `yaml:"disabled_by_default"`
	DisabledReason      string   `yaml:"disabled_reason"`
	RequiresTrustedUser bool     `yaml:"requires_trusted_user"`
	Requires            []string `yaml:"requires"`
}

type denySpec struct {
	ID        string   `yaml:"id"`
	Operation string   `yaml:"operation"`
	Reason    string   `yaml:"reason"`
	Requires  []string `yaml:"requires"`
}

type testSpec struct {
	Name    string          `yaml:"name"`
	Request requestSpec     `yaml:"request"`
	Expect  expectationSpec `yaml:"expect"`
}

type requestSpec struct {
	ID        string            `yaml:"id"`
	User      string            `yaml:"user"`
	Channel   string            `yaml:"channel"`
	Operation string            `yaml:"operation"`
	Metadata  map[string]string `yaml:"metadata"`
}

type expectationSpec struct {
	Allow             *bool    `yaml:"allow"`
	Target            string   `yaml:"target"`
	Executor          string   `yaml:"executor"`
	FallbackExecutors []string `yaml:"fallback_executors"`
	Reason            string   `yaml:"reason"`
	ReasonContains    string   `yaml:"reason_contains"`
}

type loadedVocabulary struct {
	spec vocabularySpec
	data []byte
}

type loadedPolicy struct {
	spec policySpec
	data []byte
}

type vocabularyIndex struct {
	operations map[string]struct{}
	targets    map[string]struct{}
	executors  map[string]struct{}
}

// Check validates v3 vocabulary and policy YAML without generating code.
func Check(options Options) (Output, error) {
	vocab, policy, err := load(options)
	if err != nil {
		return Output{}, err
	}
	if err := validate(vocab.spec, policy.spec); err != nil {
		return Output{}, err
	}
	return Output{
		Metadata: metadataFor(policy, vocab),
		Snapshot: snapshotFor(policy, vocab),
	}, nil
}

// Generate validates v3 YAML and returns formatted generated policy source.
func Generate(options Options) (Output, error) {
	if err := validateGenerateOptions(options); err != nil {
		return Output{}, err
	}
	vocab, policy, err := load(options)
	if err != nil {
		return Output{}, err
	}
	if err := validate(vocab.spec, policy.spec); err != nil {
		return Output{}, err
	}
	importRoot := strings.TrimSpace(options.CervoRulesImport)
	if importRoot == "" {
		importRoot = defaultCervoRulesPath
	}
	source, err := renderPolicySource(options, importRoot, policy, vocab)
	if err != nil {
		return Output{}, err
	}
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return Output{}, fmt.Errorf("format generated policy: %w\n%s", err, source)
	}
	var testSource string
	if options.GeneratedTests {
		rawTest, err := renderTestSource(options, importRoot, policy)
		if err != nil {
			return Output{}, err
		}
		formattedTest, err := format.Source([]byte(rawTest))
		if err != nil {
			return Output{}, fmt.Errorf("format generated policy tests: %w\n%s", err, rawTest)
		}
		testSource = string(formattedTest)
	}
	return Output{
		Source:     string(formatted),
		TestSource: testSource,
		Metadata:   metadataFor(policy, vocab),
		Snapshot:   snapshotFor(policy, vocab),
	}, nil
}

func validateGenerateOptions(options Options) error {
	if !packagePattern.MatchString(options.PackageName) {
		return fmt.Errorf("invalid package name %q", options.PackageName)
	}
	if strings.TrimSpace(options.VocabularyPackage) == "" {
		return errors.New("missing vocabulary package")
	}
	if strings.TrimSpace(options.VocabularyImport) == "" {
		return errors.New("missing vocabulary import")
	}
	return nil
}

func load(options Options) (loadedVocabulary, loadedPolicy, error) {
	if options.VocabularyReader == nil {
		return loadedVocabulary{}, loadedPolicy{}, errors.New("missing vocabulary reader")
	}
	if options.PolicyReader == nil {
		return loadedVocabulary{}, loadedPolicy{}, errors.New("missing policy reader")
	}
	vocabData, err := io.ReadAll(options.VocabularyReader)
	if err != nil {
		return loadedVocabulary{}, loadedPolicy{}, fmt.Errorf("read vocabulary: %w", err)
	}
	policyData, err := io.ReadAll(options.PolicyReader)
	if err != nil {
		return loadedVocabulary{}, loadedPolicy{}, fmt.Errorf("read policy: %w", err)
	}
	var vocab vocabularySpec
	vocabDecoder := yaml.NewDecoder(bytes.NewReader(vocabData))
	vocabDecoder.KnownFields(true)
	if err := vocabDecoder.Decode(&vocab); err != nil {
		return loadedVocabulary{}, loadedPolicy{}, fmt.Errorf("decode vocabulary: %w", err)
	}
	var policy policySpec
	policyDecoder := yaml.NewDecoder(bytes.NewReader(policyData))
	policyDecoder.KnownFields(true)
	if err := policyDecoder.Decode(&policy); err != nil {
		return loadedVocabulary{}, loadedPolicy{}, fmt.Errorf("decode policy: %w", err)
	}
	normalizeVocabulary(&vocab)
	return loadedVocabulary{spec: vocab, data: vocabData}, loadedPolicy{spec: policy, data: policyData}, nil
}

func normalizeVocabulary(vocab *vocabularySpec) {
	if vocab.Operations == nil {
		vocab.Operations = map[string]vocabularyEntry{}
	}
	if vocab.Targets == nil {
		vocab.Targets = map[string]vocabularyEntry{}
	}
	if vocab.Executors == nil {
		vocab.Executors = map[string]vocabularyEntry{}
	}
}

func validate(vocab vocabularySpec, policy policySpec) error {
	if policy.Version != supportedPolicyVersion {
		return fmt.Errorf("unsupported policy version %q", policy.Version)
	}
	if strings.TrimSpace(policy.Name) == "" {
		return errors.New("policy name is required")
	}
	index := newVocabularyIndex(vocab)
	if policy.Defaults.Executor != "" {
		if !index.hasExecutor(policy.Defaults.Executor) {
			return fmt.Errorf("unknown executor %q in defaults", policy.Defaults.Executor)
		}
	}
	for _, route := range policy.Routes {
		if strings.TrimSpace(route.Operation) == "" {
			return fmt.Errorf("route %q missing operation", route.ID)
		}
		if !index.hasOperation(route.Operation) {
			return fmt.Errorf("unknown operation %q in route %q", route.Operation, route.ID)
		}
		if !index.hasTarget(route.Target) {
			return fmt.Errorf("unknown target %q in route %q", route.Target, route.ID)
		}
		if !index.hasExecutor(route.Executor) {
			return fmt.Errorf("unknown executor %q in route %q", route.Executor, route.ID)
		}
		for _, executor := range route.FallbackExecutors {
			if !index.hasExecutor(executor) {
				return fmt.Errorf("unknown executor %q in route %q fallback", executor, route.ID)
			}
		}
	}
	declaredConditions, err := validateConditions(policy)
	if err != nil {
		return err
	}
	for _, route := range policy.Routes {
		if err := validateRequires(declaredConditions, route.Requires, "route", route.ID); err != nil {
			return err
		}
	}
	for _, deny := range policy.Denies {
		if strings.TrimSpace(deny.Operation) == "" {
			return fmt.Errorf("deny %q missing operation", deny.ID)
		}
		if err := validateRequires(declaredConditions, deny.Requires, "deny", deny.ID); err != nil {
			return err
		}
	}
	for _, test := range policy.Tests {
		if test.Request.Operation == "" {
			return fmt.Errorf("test %q missing request operation", test.Name)
		}
	}
	return nil
}

// validateConditions checks each declared condition carries the fields its kind
// needs, and returns the set of declared names.
func validateConditions(policy policySpec) (map[string]struct{}, error) {
	declared := make(map[string]struct{}, len(policy.Conditions))
	for name, condition := range policy.Conditions {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New("condition name is required")
		}
		normalized := normalize(name)
		switch condition.Kind {
		case "transition_allowed":
			if condition.Lifecycle == "" || condition.To == "" || condition.SubjectKey == "" {
				return nil, fmt.Errorf("condition %q of kind transition_allowed needs lifecycle, to, and subject_key", name)
			}
		case "in_state":
			if condition.Lifecycle == "" || condition.SubjectKey == "" || len(condition.States) == 0 {
				return nil, fmt.Errorf("condition %q of kind in_state needs lifecycle, subject_key, and states", name)
			}
		case "has_type":
			if condition.EntityType == "" || condition.SubjectKey == "" {
				return nil, fmt.Errorf("condition %q of kind has_type needs entity_type and subject_key", name)
			}
		case "integrity":
			// Integrity asks about the whole snapshot, so it takes no subject.
		case "":
			return nil, fmt.Errorf("condition %q missing kind", name)
		default:
			return nil, fmt.Errorf("condition %q has unknown kind %q", name, condition.Kind)
		}
		declared[normalized] = struct{}{}
	}
	return declared, nil
}

// validateRequires rejects a rule that requires a condition the policy never
// declares. Catching it here means an unwired guard fails the build instead of
// failing a decision in production.
func validateRequires(declared map[string]struct{}, requires []string, kind, id string) error {
	seen := make(map[string]struct{}, len(requires))
	for _, name := range requires {
		normalized := normalize(name)
		if normalized == "" {
			return fmt.Errorf("%s %q requires an empty condition name", kind, id)
		}
		if _, ok := declared[normalized]; !ok {
			return fmt.Errorf("%s %q requires undeclared condition %q", kind, id, name)
		}
		if _, ok := seen[normalized]; ok {
			return fmt.Errorf("%s %q requires condition %q more than once", kind, id, name)
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

// sortedConditionNames returns declared condition names in stable order.
func sortedConditionNames(policy policySpec) []string {
	out := make([]string, 0, len(policy.Conditions))
	for name := range policy.Conditions {
		out = append(out, normalize(name))
	}
	sort.Strings(out)
	return out
}

func newVocabularyIndex(vocab vocabularySpec) vocabularyIndex {
	index := vocabularyIndex{
		operations: map[string]struct{}{},
		targets:    map[string]struct{}{},
		executors:  map[string]struct{}{},
	}
	for value := range vocab.Operations {
		index.operations[normalize(value)] = struct{}{}
	}
	for value := range vocab.Targets {
		index.targets[normalize(value)] = struct{}{}
	}
	for value := range vocab.Executors {
		index.executors[normalize(value)] = struct{}{}
	}
	return index
}

func (i vocabularyIndex) hasOperation(value string) bool {
	_, ok := i.operations[normalize(value)]
	return ok
}

func (i vocabularyIndex) hasTarget(value string) bool {
	_, ok := i.targets[normalize(value)]
	return ok
}

func (i vocabularyIndex) hasExecutor(value string) bool {
	_, ok := i.executors[normalize(value)]
	return ok
}

func metadataFor(policy loadedPolicy, vocab loadedVocabulary) PolicyMetadata {
	return PolicyMetadata{
		Name:           policy.spec.Name,
		DSLVersion:     policy.spec.Version,
		GeneratedWith:  "cervorules-policygen/v3",
		VocabularyHash: hash(vocab.data),
		PolicyHash:     hash(policy.data),
	}
}

func snapshotFor(policy loadedPolicy, vocab loadedVocabulary) CodegenSnapshot {
	return CodegenSnapshot{
		PolicyName: policy.spec.Name,
		DSLVersion: policy.spec.Version,
		Operations: len(vocab.spec.Operations),
		Targets:    len(vocab.spec.Targets),
		Executors:  len(vocab.spec.Executors),
		Routes:     len(policy.spec.Routes),
		Denies:     len(policy.spec.Denies),
		Tests:      len(policy.spec.Tests),
		Conditions: len(policy.spec.Conditions),
	}
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sortedRoutes(routes []routeSpec) []routeSpec {
	out := append([]routeSpec(nil), routes...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Operation < out[j].Operation
	})
	return out
}
