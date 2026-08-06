package policygen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The JSON Schemas are the published contract, and nothing in this repository
// used to read them: a schema could go malformed, grow a dangling $ref, or
// drift from the generator's own vocabulary and ship anyway.
//
// Validating YAML documents against the schemas needs a JSON Schema
// implementation, which would be a new dependency and is therefore a CI step
// against an external CLI (scripts/ci/validate-schemas.py). What can be checked
// here without a dependency is the schema documents themselves, and the two
// places where they enumerate something the generator also enumerates.

func schemaFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	root := filepath.Join("..", "..", "schemas")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk schemas: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no schema files found")
	}
	return files
}

func loadSchema(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("%s is not valid JSON: %v", path, err)
	}
	return doc
}

func TestSchemasAreWellFormed(t *testing.T) {
	for _, path := range schemaFiles(t) {
		t.Run(filepath.ToSlash(strings.TrimPrefix(path, filepath.Join("..", "..")+string(filepath.Separator))), func(t *testing.T) {
			doc := loadSchema(t, path)
			for _, key := range []string{"$schema", "$id", "title"} {
				if _, ok := doc[key]; !ok {
					t.Errorf("schema is missing %q", key)
				}
			}

			defs := map[string]struct{}{}
			if raw, ok := doc["$defs"].(map[string]any); ok {
				for name := range raw {
					defs[name] = struct{}{}
				}
			}
			refs := map[string]struct{}{}
			collectRefs(doc, refs)

			// A dangling $ref is silently permissive in most validators: the
			// subschema simply never constrains anything.
			var dangling []string
			for ref := range refs {
				local, ok := strings.CutPrefix(ref, "#/$defs/")
				if !ok {
					continue
				}
				if _, ok := defs[local]; !ok {
					dangling = append(dangling, ref)
				}
			}
			sort.Strings(dangling)
			if len(dangling) > 0 {
				t.Errorf("dangling $ref: %v", dangling)
			}

			var unused []string
			for name := range defs {
				if _, ok := refs["#/$defs/"+name]; !ok {
					unused = append(unused, name)
				}
			}
			sort.Strings(unused)
			if len(unused) > 0 {
				t.Errorf("unreferenced $defs: %v", unused)
			}
		})
	}
}

func collectRefs(node any, out map[string]struct{}) {
	switch typed := node.(type) {
	case map[string]any:
		for key, value := range typed {
			if key == "$ref" {
				if ref, ok := value.(string); ok {
					out[ref] = struct{}{}
				}
				continue
			}
			collectRefs(value, out)
		}
	case []any:
		for _, item := range typed {
			collectRefs(item, out)
		}
	}
}

// TestPredicateOperatorsMatchGenerator ties the published operator list to the
// one the generator accepts. Adding an operator to opsByKind without adding it
// to the schema ships a policy shape the contract says is invalid, and the
// reverse ships a contract promising something the generator rejects.
func TestPredicateOperatorsMatchGenerator(t *testing.T) {
	doc := loadSchema(t, filepath.Join("..", "..", "schemas", "v3", "policy-rules.schema.json"))
	schemaOps := enumAt(t, doc, "$defs", "predicate", "properties", "op")

	generatorOps := map[string]struct{}{}
	for _, ops := range opsByKind {
		for op := range ops {
			generatorOps[op] = struct{}{}
		}
	}
	assertSameSet(t, "predicate operators", schemaOps, generatorOps)
}

// TestFactTypesMatchGenerator does the same for the declared fact kinds.
func TestFactTypesMatchGenerator(t *testing.T) {
	doc := loadSchema(t, filepath.Join("..", "..", "schemas", "v3", "policy-vocabulary.schema.json"))
	schemaTypes := enumAt(t, doc, "$defs", "fact", "properties", "type")

	generatorTypes := map[string]struct{}{}
	for kind := range opsByKind {
		generatorTypes[string(kind)] = struct{}{}
	}
	assertSameSet(t, "fact types", schemaTypes, generatorTypes)
}

func enumAt(t *testing.T, doc map[string]any, path ...string) map[string]struct{} {
	t.Helper()
	node := any(doc)
	for _, step := range path {
		object, ok := node.(map[string]any)
		if !ok {
			t.Fatalf("path %v: %q is not an object", path, step)
		}
		node, ok = object[step]
		if !ok {
			t.Fatalf("path %v: %q is missing", path, step)
		}
	}
	object, ok := node.(map[string]any)
	if !ok {
		t.Fatalf("path %v does not resolve to an object", path)
	}
	raw, ok := object["enum"].([]any)
	if !ok {
		t.Fatalf("path %v has no enum", path)
	}
	out := map[string]struct{}{}
	for _, value := range raw {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("path %v has a non-string enum member %v", path, value)
		}
		out[text] = struct{}{}
	}
	return out
}

func assertSameSet(t *testing.T, what string, schema map[string]struct{}, generator map[string]struct{}) {
	t.Helper()
	var onlySchema, onlyGenerator []string
	for value := range schema {
		if _, ok := generator[value]; !ok {
			onlySchema = append(onlySchema, value)
		}
	}
	for value := range generator {
		if _, ok := schema[value]; !ok {
			onlyGenerator = append(onlyGenerator, value)
		}
	}
	sort.Strings(onlySchema)
	sort.Strings(onlyGenerator)
	if len(onlySchema) > 0 {
		t.Errorf("%s promised by the schema but rejected by the generator: %v", what, onlySchema)
	}
	if len(onlyGenerator) > 0 {
		t.Errorf("%s accepted by the generator but absent from the schema: %v", what, onlyGenerator)
	}
}
