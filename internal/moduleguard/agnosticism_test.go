package moduleguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// vocabularySections are the parts of a vocabulary document whose keys are
// names the operator invents. Everything under them is domain, by definition.
var vocabularySections = []string{"operations", "targets", "executors", "facts", "conditions"}

// The engine treats vocabulary as data, never as knowledge. An operation, a
// target, an executor or a fact is a name the operator invents in YAML; the
// library must be able to run a policy about refunds, firmware or radiology
// without a line of it mentioning any of them.
//
// The check: no identifier declared in any example vocabulary may appear as a
// string literal in hand-written library source. It reads literals through the
// Go parser rather than by scanning text, so a word inside a comment or an
// identifier is not mistaken for a hardcoded name.
//
// Generated code is excluded, and that exclusion is the rule stated exactly
// rather than a hole in it: domain vocabulary is supposed to land in generated
// code. That is the whole architecture. What must stay clean is the engine.
//
// The denylist is derived from the examples rather than hand-written, so it
// grows on its own as examples are added and cannot go stale. An earlier draft
// of this test compared word fragments of exported names instead — it flagged
// nine terms of which seven were false positives (Error, Executor, Runtime and
// the like), which is the sort of test that gets a baseline on day one and is
// edited rather than obeyed from then on.
func TestLibrarySourceHardcodesNoDomainVocabulary(t *testing.T) {
	root := repoRoot(t)
	vocabulary := exampleVocabulary(t, root)
	if len(vocabulary) == 0 {
		t.Fatal("no vocabulary identifiers found in examples; the check has nothing to look for")
	}

	for _, file := range libraryGoFiles(t, root, false) {
		for _, literal := range stringLiterals(t, file) {
			if !vocabulary[literal.value] {
				continue
			}
			rel, _ := filepath.Rel(root, file)
			t.Errorf("%s:%d hardcodes the domain name %q\n\n"+
				"Operations, targets, executors and facts are named by the operator\n"+
				"in YAML. The engine must not know any of them by name.",
				filepath.ToSlash(rel), literal.line, literal.value)
		}
	}
}

// The check above passes trivially if the scanner finds no literals at all — a
// wrong file filter, a parser that silently returned nothing. Generated code is
// the positive control: it exists precisely to carry domain names, so it must
// be full of them. If it is not, the scanner is broken and the test above is
// asserting nothing.
func TestTheAgnosticismScannerSeesDomainNamesWhereTheyBelong(t *testing.T) {
	root := repoRoot(t)
	vocabulary := exampleVocabulary(t, root)

	found := map[string]bool{}
	for _, file := range libraryGoFiles(t, root, true) {
		if !isGenerated(t, file) {
			continue
		}
		for _, literal := range stringLiterals(t, file) {
			if vocabulary[literal.value] {
				found[literal.value] = true
			}
		}
	}
	if len(found) == 0 {
		t.Fatal("no domain names found in generated code.\n" +
			"Generated code carries the vocabulary by design, so finding none\n" +
			"means the scanner is not reading literals and the agnosticism\n" +
			"check above proves nothing.")
	}
}

// exampleVocabulary returns every operation, target, executor, fact and
// condition name declared across the example vocabularies.
func exampleVocabulary(t *testing.T, root string) map[string]bool {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(root, "examples", "*", "policy-vocabulary.yaml"))
	if err != nil {
		t.Fatalf("list example vocabularies: %v", err)
	}

	names := map[string]bool{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var document map[string]yaml.Node
		if err := yaml.Unmarshal(data, &document); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, section := range vocabularySections {
			node, ok := document[section]
			if !ok || node.Kind != yaml.MappingNode {
				continue
			}
			// A mapping node stores key and value alternating in Content.
			for i := 0; i < len(node.Content); i += 2 {
				names[node.Content[i].Value] = true
			}
		}
	}
	return names
}

// libraryGoFiles returns the module's own Go source. Tests are excluded because
// a test may legitimately name a domain, and examples because they are the
// domain. Generated files are excluded unless asked for.
func libraryGoFiles(t *testing.T, root string, includeGenerated bool) []string {
	t.Helper()

	skipDirs := map[string]bool{".git": true, "examples": true, "testdata": true, "docs": true, "scripts": true}

	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skipDirs[entry.Name()] || strings.HasPrefix(entry.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		if !includeGenerated && isGenerated(t, path) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(files)
	return files
}

// isGenerated reports the convention Go tooling agrees on: a line matching
// "// Code generated ... DO NOT EDIT." before the package clause.
func isGenerated(t *testing.T, path string) bool {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			return false
		}
		if strings.HasPrefix(line, "// Code generated ") && strings.HasSuffix(line, " DO NOT EDIT.") {
			return true
		}
	}
	return false
}

type sourceLiteral struct {
	value string
	line  int
}

// stringLiterals returns the string constants in a file, read through the Go
// parser. Scanning text would also match the word inside a comment or an
// identifier, and neither of those hardcodes anything.
func stringLiterals(t *testing.T, path string) []sourceLiteral {
	t.Helper()

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var literals []sourceLiteral
	ast.Inspect(parsed, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		literals = append(literals, sourceLiteral{value: value, line: fileSet.Position(literal.Pos()).Line})
		return true
	})
	return literals
}
