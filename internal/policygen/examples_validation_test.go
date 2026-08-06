package policygen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/internal/vocabgen"
)

func examplePolicyDirs(t *testing.T) []string {
	t.Helper()
	root := filepath.Join("..", "..", "examples")
	var dirs []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "policy-rules.yaml" {
			return nil
		}
		dir := filepath.Dir(path)
		if _, err := os.Stat(filepath.Join(dir, "policy-vocabulary.yaml")); err != nil {
			return nil
		}
		dirs = append(dirs, dir)
		return nil
	})
	if err != nil {
		t.Fatalf("walk examples: %v", err)
	}
	if len(dirs) == 0 {
		t.Fatal("no example policy pairs found")
	}
	return dirs
}

func exampleName(dir string) string {
	root := filepath.Join("..", "..", "examples")
	return filepath.ToSlash(strings.TrimPrefix(dir, root+string(filepath.Separator)))
}

func TestExamplesUseSupportedV3PolicyShape(t *testing.T) {
	for _, dir := range examplePolicyDirs(t) {
		t.Run(exampleName(dir), func(t *testing.T) {
			vocab, policy := readExamplePair(t, dir)
			if _, err := Check(Options{
				VocabularyReader: strings.NewReader(vocab),
				PolicyReader:     strings.NewReader(policy),
			}); err != nil {
				t.Fatalf("check example: %v", err)
			}
		})
	}
}

// TestExamplesGenerateCompileAndPass runs each example end to end.
//
// Check alone only proves an example parses. An example that ships declarative
// `tests:` is making a claim about what the policy decides, and until the
// generated tests are compiled and run, nothing verifies that claim — the seven
// cases in examples/predicate-composition asserted thresholds nothing executed.
func TestExamplesGenerateCompileAndPass(t *testing.T) {
	for _, dir := range examplePolicyDirs(t) {
		t.Run(exampleName(dir), func(t *testing.T) {
			t.Parallel()
			vocabYAML, policyYAML := readExamplePair(t, dir)

			vocabGo, err := vocabgen.Generate(vocabgen.Options{
				PackageName: "policyvocab",
				ImportPath:  "github.com/cervantesh/cervo-rules/v3/core",
				Reader:      strings.NewReader(vocabYAML),
			})
			if err != nil {
				t.Fatalf("generate vocabulary: %v", err)
			}

			out, err := Generate(Options{
				PackageName:       "policyrules",
				VocabularyPackage: "policyvocab",
				VocabularyImport:  "example.test/generated/policyvocab",
				VocabularyReader:  strings.NewReader(vocabYAML),
				PolicyReader:      strings.NewReader(policyYAML),
				GeneratedTests:    true,
			})
			if err != nil {
				t.Fatalf("generate policy: %v", err)
			}
			testSource := out.TestSource
			if testSource == "" {
				testSource = minimalTestSource
			}
			compileGeneratedPolicyWithVocab(t, vocabGo, out.Source, testSource)
		})
	}
}

func readExamplePair(t *testing.T, dir string) (vocab string, policy string) {
	t.Helper()
	vocabBytes, err := os.ReadFile(filepath.Join(dir, "policy-vocabulary.yaml"))
	if err != nil {
		t.Fatalf("read vocabulary: %v", err)
	}
	policyBytes, err := os.ReadFile(filepath.Join(dir, "policy-rules.yaml"))
	if err != nil {
		t.Fatalf("read policy: %v", err)
	}
	return string(vocabBytes), string(policyBytes)
}
