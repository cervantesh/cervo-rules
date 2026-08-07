package moduleguard

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const modulePath = "github.com/cervantesh/cervo-rules/v3"

// The claim, made in the README, in ADR 0016 and in the compound-predicate
// design: importing CervoRules adds nothing to a consumer's build but the Go
// standard library. It is the reason predicates compile to Go expressions
// instead of running through an expression evaluator — "a complete expression
// language inside a security policy is an attack surface, not a convenience".
//
// A promise about a dependency graph is the kind that breaks by accident. One
// import added inside a public package, and every consumer of this module
// inherits a third-party module, its transitive graph and its advisories,
// without anyone deciding to.
//
// So it is checked against the build graph rather than against the import lines
// in source: `go list -deps` resolves the whole transitive closure, which is
// where an indirect dependency would hide.
func TestPublicPackagesDependOnNothingButTheStandardLibrary(t *testing.T) {
	root := repoRoot(t)

	for _, name := range publicPackages {
		pkg := modulePath
		if name != "" {
			pkg += "/" + name
		}
		t.Run(packageLabel(name), func(t *testing.T) {
			external := externalModules(t, root, pkg)
			if len(external) == 0 {
				return
			}
			t.Errorf("public package %s pulls in %d external module(s): %s\n\n"+
				"A consumer inherits these by importing us. If one is genuinely\n"+
				"needed, that is a decision to record in an ADR, not a test to edit.",
				pkg, len(external), strings.Join(external, ", "))
		})
	}
}

// A check that cannot fail is worth nothing, and this repository has now found
// three of them. If `go list` were invoked wrongly — a bad format string, a flag
// that stopped resolving the transitive graph — the test above would report a
// clean boundary for a dirty one, silently and forever.
//
// internal/policygen is the positive control: it parses YAML, so it must show
// gopkg.in/yaml.v3. That dependency is also the reason the boundary exists at
// all — YAML is read at generation time, by a tool the operator runs, never by
// the engine serving decisions.
func TestTheDependencyCheckDetectsARealExternalDependency(t *testing.T) {
	root := repoRoot(t)
	pkg := modulePath + "/internal/policygen"

	external := externalModules(t, root, pkg)
	if len(external) == 0 {
		t.Fatalf("%s reported no external modules.\n"+
			"It parses YAML, so it has at least one. The query is not resolving\n"+
			"the dependency graph, which means the boundary check above proves\n"+
			"nothing.", pkg)
	}
	if !contains(external, "gopkg.in/yaml.v3") {
		t.Errorf("expected %s to depend on gopkg.in/yaml.v3, got %v", pkg, external)
	}
}

// externalModules returns the modules other than this one that pkg needs to
// build, transitively. Standard library packages carry no module and drop out.
func externalModules(t *testing.T, root, pkg string) []string {
	t.Helper()

	cmd := exec.Command("go", "list", "-deps", "-f", "{{if .Module}}{{.Module.Path}}{{end}}", pkg)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps %s: %v\n%s", pkg, err, output)
	}

	seen := map[string]bool{}
	for _, line := range strings.Split(string(output), "\n") {
		path := strings.TrimSpace(line)
		if path == "" || path == modulePath {
			continue
		}
		seen[path] = true
	}

	modules := make([]string, 0, len(seen))
	for path := range seen {
		modules = append(modules, path)
	}
	sort.Strings(modules)
	return modules
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

func packageLabel(name string) string {
	if name == "" {
		return "root"
	}
	return name
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
