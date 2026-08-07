package moduleguard

import (
	"encoding/json"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// updateInventory rewrites docs/v3/public-api-inventory.json from the source
// instead of asserting against it:
//
//	go test ./internal/policygen -run TestPublicAPIInventoryMatchesSource -update-inventory
//
// The ownership statements are hand-written and are preserved; only the name
// lists are regenerated. Regenerating is deliberate, so an API change still has
// to be acknowledged rather than absorbed silently.
var updateInventory = flag.Bool("update-inventory", false, "rewrite the public API inventory from source")

// docs/v3/public-api-inventory.json is the machine-readable statement of what
// this module exposes, and the GA checklist calls for it to be reviewed after
// the final RC. It listed four of the ten public packages, so a reviewer
// reading it would not have seen limits, ontology, httpadapter, testkit,
// decisioncache or the root marker at all.
//
// A document that has to be updated by hand drifts. This parses the packages
// with go/ast -- standard library, no dependency -- and fails when the
// inventory and the source disagree in either direction.

var publicPackages = []string{
	"", "core", "decisioncache", "facts", "httpadapter",
	"limits", "observe", "ontology", "runtime", "testkit",
}

type inventory struct {
	SchemaVersion string             `json:"schema_version"`
	Module        string             `json:"module"`
	Packages      []inventoryPackage `json:"packages"`
}

type inventoryPackage struct {
	Path   string   `json:"path"`
	Owns   []string `json:"owns"`
	Types  []string `json:"public_types"`
	Funcs  []string `json:"public_funcs"`
	Consts []string `json:"public_consts"`
}

func TestPublicAPIInventoryMatchesSource(t *testing.T) {
	root := filepath.Join("..", "..")

	data, err := os.ReadFile(filepath.Join(root, "docs", "v3", "public-api-inventory.json"))
	if err != nil {
		t.Fatalf("read inventory: %v", err)
	}
	var recorded inventory
	if err := json.Unmarshal(data, &recorded); err != nil {
		t.Fatalf("inventory is not valid JSON: %v", err)
	}

	byPath := map[string]inventoryPackage{}
	for _, entry := range recorded.Packages {
		byPath[entry.Path] = entry
	}

	if *updateInventory {
		rewriteInventory(t, root, recorded, byPath)
		return
	}

	for _, name := range publicPackages {
		path := recorded.Module
		dir := root
		if name != "" {
			path += "/" + name
			dir = filepath.Join(root, name)
		}
		t.Run(name+"/", func(t *testing.T) {
			entry, ok := byPath[path]
			if !ok {
				t.Fatalf("package %s is public but absent from the inventory", path)
			}
			if len(entry.Owns) == 0 {
				t.Errorf("package %s records no ownership statement", path)
			}
			types, funcs, consts := exportedNames(t, dir)
			assertSameNames(t, path, "public_types", entry.Types, types)
			assertSameNames(t, path, "public_funcs", entry.Funcs, funcs)
			assertSameNames(t, path, "public_consts", entry.Consts, consts)
		})
	}

	// The reverse direction: an entry for a package that no longer exists, or
	// one nobody listed above, is drift too.
	known := map[string]struct{}{}
	for _, name := range publicPackages {
		path := recorded.Module
		if name != "" {
			path += "/" + name
		}
		known[path] = struct{}{}
	}
	for _, entry := range recorded.Packages {
		if _, ok := known[entry.Path]; !ok {
			t.Errorf("inventory lists %s, which is not a public package", entry.Path)
		}
	}
}

func rewriteInventory(t *testing.T, root string, recorded inventory, byPath map[string]inventoryPackage) {
	t.Helper()
	out := inventory{
		SchemaVersion: recorded.SchemaVersion,
		Module:        recorded.Module,
	}
	for _, name := range publicPackages {
		path := recorded.Module
		dir := root
		if name != "" {
			path += "/" + name
			dir = filepath.Join(root, name)
		}
		types, funcs, consts := exportedNames(t, dir)
		entry := inventoryPackage{Path: path, Owns: byPath[path].Owns}
		if len(entry.Owns) == 0 {
			t.Fatalf("package %s needs a hand-written ownership statement before it can be regenerated", path)
		}
		entry.Types, entry.Funcs, entry.Consts = types, funcs, consts
		out.Packages = append(out.Packages, entry)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		t.Fatalf("encode inventory: %v", err)
	}
	target := filepath.Join(root, "docs", "v3", "public-api-inventory.json")
	if err := os.WriteFile(target, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write inventory: %v", err)
	}
	t.Logf("rewrote %s", target)
}

// exportedNames returns the exported top-level declarations of a package,
// ignoring test files and methods.
func exportedNames(t *testing.T, dir string) (types, funcs, consts []string) {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", dir, err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch typed := decl.(type) {
				case *ast.FuncDecl:
					// Methods belong to their receiver, not to the package.
					if typed.Recv == nil && typed.Name.IsExported() {
						funcs = append(funcs, typed.Name.Name)
					}
				case *ast.GenDecl:
					for _, spec := range typed.Specs {
						switch s := spec.(type) {
						case *ast.TypeSpec:
							if s.Name.IsExported() {
								types = append(types, s.Name.Name)
							}
						case *ast.ValueSpec:
							if typed.Tok != token.CONST {
								continue
							}
							for _, name := range s.Names {
								if name.IsExported() {
									consts = append(consts, name.Name)
								}
							}
						}
					}
				}
			}
		}
	}
	sort.Strings(types)
	sort.Strings(funcs)
	sort.Strings(consts)
	return types, funcs, consts
}

func assertSameNames(t *testing.T, path, field string, recorded, actual []string) {
	t.Helper()
	inRecorded := map[string]struct{}{}
	for _, name := range recorded {
		inRecorded[name] = struct{}{}
	}
	inActual := map[string]struct{}{}
	for _, name := range actual {
		inActual[name] = struct{}{}
	}

	var missing, extra []string
	for _, name := range actual {
		if _, ok := inRecorded[name]; !ok {
			missing = append(missing, name)
		}
	}
	for _, name := range recorded {
		if _, ok := inActual[name]; !ok {
			extra = append(extra, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 {
		t.Errorf("%s %s: exported by the source but absent from the inventory: %v", path, field, missing)
	}
	if len(extra) > 0 {
		t.Errorf("%s %s: listed in the inventory but not exported: %v", path, field, extra)
	}
}
