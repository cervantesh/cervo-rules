package moduleguard

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// A task recipe in .cervorules/recipes is an instruction an agent follows
// literally: it names commands to run, flags to pass and tests to select. The
// JSON Schema checks the shape of those fields, which means a recipe can be
// perfectly valid and still tell an agent to run a test that does not exist.
//
// That is not hypothetical. "The conformance recipe pointing at a test name
// nothing matches" was one of the seven gaps opened during the rc.5 work. It
// was fixed by hand, and nothing stopped it coming back.
//
// So the claims are checked rather than the shape: every -run pattern selects
// a real test, every script exists, every CLI subcommand and flag is declared,
// and every pinned version matches the manifest. Values are deliberately not
// checked -- a recipe names files like policy-rules.yaml that live in the
// consumer's repository, not ours.
func TestRecipeCommandsReferenceThingsThatExist(t *testing.T) {
	root := repoRoot(t)

	tests := declaredTestNames(t, root)
	flags := declaredCLIFlags(t, root)
	subcommands := declaredCLISubcommands(t, root)
	version := manifestVersion(t, root)

	versionToken := regexp.MustCompile(`^v\d+\.\d+\.\d+`)

	for _, recipe := range recipeFiles(t, root) {
		name := filepath.Base(recipe)
		for _, command := range recipeCommands(t, recipe) {
			fields := strings.Fields(command)
			for i, field := range fields {
				next := ""
				if i+1 < len(fields) {
					next = fields[i+1]
				}

				switch {
				case field == "-run":
					if !selectsATest(t, next, tests) {
						t.Errorf("%s: `-run %s` selects no test in this repository\n  in: %s", name, next, command)
					}

				case field == "bash" && next != "":
					if _, err := os.Stat(filepath.Join(root, next)); err != nil {
						t.Errorf("%s: script %s does not exist\n  in: %s", name, next, command)
					}

				case strings.HasPrefix(field, "./cmd/"):
					if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(field))); err != nil {
						t.Errorf("%s: command %s does not exist\n  in: %s", name, field, command)
					}
					// The token right after the binary is its subcommand.
					if next != "" && !strings.HasPrefix(next, "-") && !subcommands[next] {
						t.Errorf("%s: %s has no subcommand %q\n  in: %s", name, field, next, command)
					}

				case strings.HasPrefix(field, "-"):
					flag := strings.TrimLeft(field, "-")
					if index := strings.IndexByte(flag, '='); index >= 0 {
						flag = flag[:index]
					}
					if flag == "" {
						continue // "-" and "--" are separators, not flags
					}
					if flag == "run" || flag == "count" || flag == "no-index" {
						continue // go test and git flags, not ours
					}
					if !flags[flag] {
						t.Errorf("%s: no CLI declares the flag -%s\n  in: %s", name, flag, command)
					}

				case versionToken.MatchString(field):
					// A recipe that pins a version goes stale at the next tag,
					// silently, and an agent then verifies the wrong release.
					if field != version {
						t.Errorf("%s: pins %s but the manifest says %s\n  in: %s\n\n"+
							"Update the recipe with the release, or stop pinning a version in it.",
							name, field, version, command)
					}
				}
			}
		}
	}
}

// The check above passes trivially if nothing was collected. Each input is
// therefore required to be non-empty: an empty test set makes every -run look
// valid, and an empty flag set makes every flag look declared.
func TestTheRecipeCheckHasSomethingToCheckAgainst(t *testing.T) {
	root := repoRoot(t)

	for name, size := range map[string]int{
		"tests in the repository": len(declaredTestNames(t, root)),
		"CLI flags":               len(declaredCLIFlags(t, root)),
		"CLI subcommands":         len(declaredCLISubcommands(t, root)),
		"recipes":                 len(recipeFiles(t, root)),
	} {
		if size == 0 {
			t.Errorf("found no %s; the recipe check would assert nothing", name)
		}
	}
	if manifestVersion(t, root) == "" {
		t.Error("the manifest declares no current_version; pinned versions would go unchecked")
	}
}

func selectsATest(t *testing.T, pattern string, tests map[string]bool) bool {
	t.Helper()

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		t.Errorf("-run %s is not a valid regular expression: %v", pattern, err)
		return true // already reported; do not report it twice
	}
	for name := range tests {
		if compiled.MatchString(name) {
			return true
		}
	}
	return false
}

func recipeFiles(t *testing.T, root string) []string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(root, ".cervorules", "recipes", "*.json"))
	if err != nil {
		t.Fatalf("list recipes: %v", err)
	}
	sort.Strings(paths)
	return paths
}

func recipeCommands(t *testing.T, path string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var recipe struct {
		Commands []string `json:"commands"`
		Checks   []string `json:"checks"`
	}
	if err := json.Unmarshal(raw, &recipe); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return append(recipe.Commands, recipe.Checks...)
}

var testDeclaration = regexp.MustCompile(`func ((?:Test|Fuzz|Benchmark)\w*)\(`)

func declaredTestNames(t *testing.T, root string) map[string]bool {
	t.Helper()

	names := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range testDeclaration.FindAllStringSubmatch(string(source), -1) {
			names[match[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan tests: %v", err)
	}
	return names
}

var (
	flagDeclaration       = regexp.MustCompile(`(?:flags|flag|fs)\.\w+(?:Var)?\(\s*(?:&\w+\s*,\s*)?"([\w-]+)"`)
	subcommandDeclaration = regexp.MustCompile(`case "([\w-]+)"`)
)

func declaredCLIFlags(t *testing.T, root string) map[string]bool {
	return scanCLISource(t, root, flagDeclaration)
}

func declaredCLISubcommands(t *testing.T, root string) map[string]bool {
	return scanCLISource(t, root, subcommandDeclaration)
}

func scanCLISource(t *testing.T, root string, pattern *regexp.Regexp) map[string]bool {
	t.Helper()

	found := map[string]bool{}
	paths, err := filepath.Glob(filepath.Join(root, "cmd", "*", "*.go"))
	if err != nil {
		t.Fatalf("list CLI source: %v", err)
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
			found[match[1]] = true
		}
	}
	return found
}

func manifestVersion(t *testing.T, root string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(root, ".cervorules", "agent-manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest struct {
		CurrentVersion string `json:"current_version"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	return manifest.CurrentVersion
}
