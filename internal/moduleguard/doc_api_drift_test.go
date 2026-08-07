package moduleguard

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// docs/package-minimal-examples.md is listed in the agent manifest as a machine
// entrypoint, and it documented five symbols that do not exist:
// limits.CheckLimits, limits.RequestedLimits, httpadapter.NewHTTPClassifier,
// observe.PolicyEvaluationReport used as a function, and
// testkit.MustAssertGeneratedRuntimePolicy. A whole decisioncache section
// showed an API the package never had.
//
// The same phantom name reached .cervorules/recipes/run-conformance.json as
// `go test -run TestGeneratedRuntimePolicy`, which matched nothing and passed
// silently. Copyable examples that do not compile are worse than absent ones:
// an agent following them writes code that cannot build, and a reader concludes
// the library is broken.
//
// This checks package-qualified references in current-facing docs against the
// exported surface. Historical and migration docs are excluded by name, because
// they are supposed to describe an API that no longer exists.

var docSymbolRef = regexp.MustCompile(`\b(core|runtime|facts|limits|httpadapter|observe|ontology|testkit|decisioncache)\.([A-Z]\w*)`)

// currentFacingDocs are the documents AGENTS.md requires to use v3 names.
var currentFacingDocs = []string{
	"README.md",
	"AGENTS.md",
	"docs/package-minimal-examples.md",
	"docs/adapter-patterns.md",
	"docs/modular-architecture.md",
	"docs/agent-quickstart.md",
	"docs/agent-commands.md",
	"docs/v3/api-reference.md",
	"docs/v3/primitives.md",
	"docs/v3/routing.md",
	"docs/v3/decision-result.md",
	"docs/v3/observability.md",
	"docs/v3/symbolic-guards.md",
	"docs/v3/structured-errors.md",
	"docs/v3/consumer-conformance.md",
	"docs/v3/generated-policy-factory.md",
	"docs/v3/policygen-dsl.md",
	"docs/v3/compound-predicates.md",
	"docs/v3/known-gaps.md",
}

func TestCurrentFacingDocsReferenceRealAPI(t *testing.T) {
	root := filepath.Join("..", "..")

	exported := map[string]map[string]struct{}{}
	for _, name := range publicPackages {
		if name == "" {
			continue
		}
		types, funcs, consts := exportedNames(t, filepath.Join(root, name))
		set := map[string]struct{}{}
		for _, group := range [][]string{types, funcs, consts} {
			for _, symbol := range group {
				set[symbol] = struct{}{}
			}
		}
		exported[name] = set
	}

	for _, doc := range currentFacingDocs {
		path := filepath.Join(root, filepath.FromSlash(doc))
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				t.Errorf("%s is listed as current-facing but does not exist", doc)
				continue
			}
			t.Fatalf("read %s: %v", doc, err)
		}

		var unknown []string
		seen := map[string]struct{}{}
		for _, match := range docSymbolRef.FindAllStringSubmatch(string(data), -1) {
			pkg, symbol := match[1], match[2]
			set, ok := exported[pkg]
			if !ok {
				continue
			}
			if _, ok := set[symbol]; ok {
				continue
			}
			ref := pkg + "." + symbol
			if _, ok := seen[ref]; ok {
				continue
			}
			seen[ref] = struct{}{}
			unknown = append(unknown, ref)
		}
		sort.Strings(unknown)
		if len(unknown) > 0 {
			t.Errorf("%s references symbols no package exports: %s", doc, strings.Join(unknown, ", "))
		}
	}
}
