package policygen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamplesUseSupportedV3PolicyShape(t *testing.T) {
	root := filepath.Join("..", "..", "examples")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "policy-rules.yaml" {
			return nil
		}
		dir := filepath.Dir(path)
		vocabPath := filepath.Join(dir, "policy-vocabulary.yaml")
		if _, err := os.Stat(vocabPath); err != nil {
			return nil
		}
		t.Run(strings.TrimPrefix(dir, root+string(filepath.Separator)), func(t *testing.T) {
			vocab, err := os.Open(vocabPath)
			if err != nil {
				t.Fatalf("open vocab: %v", err)
			}
			defer vocab.Close()
			policy, err := os.Open(path)
			if err != nil {
				t.Fatalf("open policy: %v", err)
			}
			defer policy.Close()
			if _, err := Check(Options{VocabularyReader: vocab, PolicyReader: policy}); err != nil {
				t.Fatalf("check example: %v", err)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk examples: %v", err)
	}
}
