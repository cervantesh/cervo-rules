package metamorphic

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/internal/policygen"
	"github.com/cervantesh/cervo-rules/v3/internal/vocabgen"
)

// familyCount is how many independent random policies each run exercises. Each
// costs four generated packages, and they are all compiled together in one
// module so the go build cost is paid once rather than per policy.
const familyCount = 6

// The properties below hold for every policy, so they need no oracle written by
// hand and no expected output to keep current. That is what makes them worth
// more than the 193 example-based tests: those check the cases somebody thought
// of, and this checks a law across a space nobody enumerated.
//
// The requests are not sampled. For each policy the facts are reduced to their
// equivalence classes -- below the minimum, at it, between, at the maximum,
// above it, non-finite, unparseable, blank, absent -- and every combination of
// those classes across every operation is enumerated. Within that lattice the
// result is exhaustive, not statistical.
//
//	P1 determinism      the same request twice gives the same answer
//	P2 fail-closed      an unusable fact yields a structured error, never a decision
//	P3 monotonicity     appending a deny can remove an allow and never create one,
//	                    and an earlier matching deny keeps its reason
//	P4 inertness        a rule that cannot hold changes nothing
//	P5 De Morgan        the De Morgan dual of a rule decides exactly as the rule
//
// P3 and P5 are the ones that bite the compiler: they compare two policies that
// must agree, so a mistranslation of rule order or of `not` shows up as a
// disagreement rather than as a wrong answer nobody wrote down.
func TestMetamorphicPropertiesHoldOverRandomPolicies(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles a module of generated policies")
	}

	seed := int64(1)
	if raw := os.Getenv("CERVORULES_METAMORPHIC_SEED"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			t.Fatalf("CERVORULES_METAMORPHIC_SEED is not a number: %v", err)
		}
		seed = parsed
	}
	rng := rand.New(rand.NewSource(seed))

	dir := t.TempDir()
	root := repoRoot(t)
	writeFile(t, filepath.Join(dir, "go.mod"), fmt.Sprintf(
		"module example.test/generated\n\ngo %s\n\nrequire github.com/cervantesh/cervo-rules/v3 v3.0.0\n\nreplace github.com/cervantesh/cervo-rules/v3 => %s\n",
		repoGoVersion(t, root), filepath.ToSlash(root)))

	var families []familySource
	for i := 0; i < familyCount; i++ {
		families = append(families, emitFamily(t, dir, rng, i))
	}
	writeFile(t, filepath.Join(dir, "driver", "driver_test.go"), driverSource(families))

	command := exec.Command("go", "test", "-count=1", "-v", "./driver/...")
	command.Dir = dir
	output, err := command.CombinedOutput()
	t.Logf("seed %d, %d families\n%s", seed, len(families), output)
	if err != nil {
		t.Fatalf("metamorphic properties failed (seed %d): %v\n\n"+
			"Reproduce with CERVORULES_METAMORPHIC_SEED=%d.\n"+
			"A failure here is a disagreement between two policies that must agree,\n"+
			"so it is a generator defect unless the property itself is wrong.",
			seed, err, seed)
	}
}

type familySource struct {
	Package    string // import path suffix shared by the family
	Operations []string
	Facts      []factClasses
}

type factClasses struct {
	Name    string
	Classes []string
}

// absentClass marks a request that omits the metadata key entirely, which is a
// different thing from sending an empty value.
const absentClass = "\x00absent"

func emitFamily(t *testing.T, dir string, rng *rand.Rand, index int) familySource {
	t.Helper()

	policy := newRandomPolicy(rng, index)
	vocabYAML := policy.vocabularyYAML()
	vocabPackage := fmt.Sprintf("v%d", index)

	vocabGo, err := vocabgen.Generate(vocabgen.Options{
		PackageName: vocabPackage,
		ImportPath:  "github.com/cervantesh/cervo-rules/v3/core",
		Reader:      strings.NewReader(vocabYAML),
	})
	if err != nil {
		t.Fatalf("family %d: vocabulary: %v", index, err)
	}
	writeFile(t, filepath.Join(dir, vocabPackage, "vocab.go"), vocabGo)

	for _, variant := range []string{variantBase, variantExtraDeny, variantInert, variantDeMorgan} {
		name := fmt.Sprintf("p%d%s", index, variant)
		out, err := policygen.Generate(policygen.Options{
			PackageName:       name,
			VocabularyPackage: vocabPackage,
			VocabularyImport:  "example.test/generated/" + vocabPackage,
			VocabularyReader:  strings.NewReader(vocabYAML),
			PolicyReader:      strings.NewReader(policy.variantYAML(variant, rng)),
		})
		if err != nil {
			t.Fatalf("family %d variant %s: %v", index, variant, err)
		}
		writeFile(t, filepath.Join(dir, name, "generated_policy.go"), out.Source)
	}

	source := familySource{Package: fmt.Sprintf("p%d", index), Operations: policy.Operations}
	for _, fact := range policy.Facts {
		source.Facts = append(source.Facts, factClasses{Name: fact.Name, Classes: equivalenceClasses(fact)})
	}
	return source
}

// equivalenceClasses reduces a fact's infinite input space to the values at
// which behaviour can change. Any two inputs in the same class are treated
// identically by every predicate the DSL can express over that fact.
func equivalenceClasses(fact randomFact) []string {
	switch fact.Kind {
	case "bool":
		return []string{absentClass, "", "true", "false", "TRUE", "1", "0", "sometimes"}
	case "enum":
		classes := []string{absentClass, "", "not-a-member"}
		classes = append(classes, fact.Values...)
		return append(classes, strings.ToUpper(fact.Values[0]))
	case "integer":
		return []string{
			absentClass, "",
			strconv.FormatInt(int64(fact.Min)-1, 10),
			strconv.FormatInt(int64(fact.Min), 10),
			strconv.FormatInt(int64((fact.Min+fact.Max)/2), 10),
			strconv.FormatInt(int64(fact.Max), 10),
			strconv.FormatInt(int64(fact.Max)+1, 10),
			"not-a-number",
		}
	default:
		return []string{
			absentClass, "",
			strconv.FormatFloat(fact.Min-1, 'f', 1, 64),
			strconv.FormatFloat(fact.Min, 'f', 1, 64),
			strconv.FormatFloat((fact.Min+fact.Max)/2, 'f', 1, 64),
			strconv.FormatFloat(fact.Max, 'f', 1, 64),
			strconv.FormatFloat(fact.Max+1, 'f', 1, 64),
			"NaN",
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func repoGoVersion(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if version, ok := strings.CutPrefix(strings.TrimSpace(line), "go "); ok {
			return strings.TrimSpace(version)
		}
	}
	t.Fatal("go.mod has no go directive")
	return ""
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("no go.mod at %s: %v", root, err)
	}
	return root
}
