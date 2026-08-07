package fuzzpolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/internal/policygen"
	"github.com/cervantesh/cervo-rules/v3/internal/vocabgen"
)

const (
	fuzzSubjectExample     = "predicate-composition"
	fuzzSubjectVocabImport = "github.com/cervantesh/cervo-rules/v3/internal/fuzzpolicy/vocab"
)

// The fuzz subject is committed so a fuzzer can run without a codegen step per
// execution. That makes it a copy, and a copy goes stale silently — the exact
// failure this repository has now found five times in documents.
//
// So it is not trusted: this regenerates both files from the same example the
// CLIs use and compares bytes. A generator change that alters the emitted
// engine fails here until the subject is regenerated, which is the point.
func TestFuzzSubjectMatchesGenerator(t *testing.T) {
	vocabYAML, policyYAML := readExample(t)

	wantVocab, err := vocabgen.Generate(vocabgen.Options{
		PackageName: "vocab",
		ImportPath:  "github.com/cervantesh/cervo-rules/v3/core",
		Reader:      strings.NewReader(vocabYAML),
	})
	if err != nil {
		t.Fatalf("generate vocabulary: %v", err)
	}
	assertCommitted(t, filepath.Join("vocab", "generated.go"), wantVocab)

	out, err := policygen.Generate(policygen.Options{
		PackageName:       "fuzzpolicy",
		VocabularyPackage: "vocab",
		VocabularyImport:  fuzzSubjectVocabImport,
		VocabularyReader:  strings.NewReader(vocabYAML),
		PolicyReader:      strings.NewReader(policyYAML),
	})
	if err != nil {
		t.Fatalf("generate policy: %v", err)
	}
	assertCommitted(t, "generated_policy.go", out.Source)
}

// Generation must be a function of its inputs. The whole PolicyHash argument
// rests on it: if the same YAML can produce different bytes, the hash stops
// identifying what runs. It has already failed once — generated test metadata
// was emitted in Go map-iteration order — and Go randomizes map order per
// process, so one pass proves nothing.
func TestGenerationIsByteStableAcrossRuns(t *testing.T) {
	vocabYAML, policyYAML := readExample(t)

	var firstSource, firstTests string
	for i := 0; i < 8; i++ {
		out, err := policygen.Generate(policygen.Options{
			PackageName:       "fuzzpolicy",
			VocabularyPackage: "vocab",
			VocabularyImport:  fuzzSubjectVocabImport,
			VocabularyReader:  strings.NewReader(vocabYAML),
			PolicyReader:      strings.NewReader(policyYAML),
			GeneratedTests:    true,
		})
		if err != nil {
			t.Fatalf("run %d: generate: %v", i+1, err)
		}
		if i == 0 {
			firstSource, firstTests = out.Source, out.TestSource
			continue
		}
		if out.Source != firstSource {
			t.Fatalf("run %d produced different policy source", i+1)
		}
		if out.TestSource != firstTests {
			t.Fatalf("run %d produced different generated tests", i+1)
		}
	}
}

func readExample(t *testing.T) (vocabYAML, policyYAML string) {
	t.Helper()
	dir := filepath.Join("..", "..", "examples", fuzzSubjectExample)
	vocab, err := os.ReadFile(filepath.Join(dir, "policy-vocabulary.yaml"))
	if err != nil {
		t.Fatalf("read vocabulary: %v", err)
	}
	policy, err := os.ReadFile(filepath.Join(dir, "policy-rules.yaml"))
	if err != nil {
		t.Fatalf("read policy: %v", err)
	}
	return string(vocab), string(policy)
}

func assertCommitted(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	// Committed files carry the checkout's line endings; compare content.
	got := strings.ReplaceAll(string(data), "\r\n", "\n")
	want = strings.ReplaceAll(want, "\r\n", "\n")
	if got == want {
		return
	}
	t.Fatalf("%s is stale.\nRegenerate it:\n\n"+
		"  go run ./cmd/cervorules-vocabgen -in examples/%s/policy-vocabulary.yaml \\\n"+
		"    -out internal/fuzzpolicy/vocab/generated.go -package vocab \\\n"+
		"    -import github.com/cervantesh/cervo-rules/v3/core\n\n"+
		"  go run ./cmd/cervorules-policygen generate \\\n"+
		"    -vocab examples/%s/policy-vocabulary.yaml \\\n"+
		"    -policy examples/%s/policy-rules.yaml \\\n"+
		"    -out internal/fuzzpolicy/generated_policy.go \\\n"+
		"    -package fuzzpolicy -vocab-package vocab -vocab-import %s\n",
		path, fuzzSubjectExample, fuzzSubjectExample, fuzzSubjectExample, fuzzSubjectVocabImport)
}
