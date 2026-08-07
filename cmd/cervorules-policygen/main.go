package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cervantesh/cervo-rules/v3/core"
	"github.com/cervantesh/cervo-rules/v3/internal/policygen"
)

var version = "dev"

const (
	exitSuccess       = 0
	exitGeneric       = 1
	exitValidation    = 2
	exitCompatibility = 3
	exitBudget        = 4
	exitInternal      = 5
)

type errorReport struct {
	ExitCode int         `json:"exit_code"`
	Errors   core.Errors `json:"errors"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: cervorules-policygen <check|generate> [flags]")
		return 2
	}
	if args[0] == "-version" || args[0] == "--version" || args[0] == "version" {
		fmt.Fprintf(stderr, "cervorules-policygen %s\n", version)
		return 0
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "generate":
		return runGenerate(args[1:], stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

// runCheck writes its machine-readable result to stdout and every diagnostic
// to stderr. It used to encode the JSON to stderr, so a caller redirecting
// stdout -- which is what the task recipes do -- read an empty document.
func runCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	vocabPath := flags.String("vocab", "", "policy-vocabulary.yaml")
	policyPath := flags.String("policy", "", "policy-rules.yaml")
	format := flags.String("format", "text", "output format: text or json")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *vocabPath == "" || *policyPath == "" {
		fmt.Fprintln(stderr, "-vocab and -policy are required")
		return 2
	}
	vocab, policy, closeFiles, err := openInputs(*vocabPath, *policyPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer closeFiles()
	out, err := policygen.Check(policygen.Options{VocabularyReader: vocab, PolicyReader: policy})
	if err != nil {
		return writeCLIError(stderr, *format, exitValidation, core.Error{
			Code:      core.ErrorCodeInvalidPolicySchema,
			Severity:  core.SeverityFatal,
			Component: "policygen",
			Field:     "policy",
			Reason:    err.Error(),
		}, "check policy")
	}
	switch *format {
	case "json":
		if err := json.NewEncoder(stdout).Encode(out); err != nil {
			fmt.Fprintf(stderr, "encode check result: %v\n", err)
			return 1
		}
		return 0
	case "text", "":
	default:
		fmt.Fprintf(stderr, "unsupported -format %q\n", *format)
		return 2
	}
	fmt.Fprintf(stderr, "ok policy=%s routes=%d denies=%d predicates=%d conditions=%d facts=%d policy_facts=%d tests=%d\n",
		out.Metadata.Name, out.Snapshot.Routes, out.Snapshot.Denies, out.Snapshot.Predicates,
		out.Snapshot.Conditions, out.Snapshot.Facts, out.Snapshot.PolicyFacts, out.Snapshot.Tests)
	return 0
}

func runGenerate(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	vocabPath := flags.String("vocab", "", "policy-vocabulary.yaml")
	policyPath := flags.String("policy", "", "policy-rules.yaml")
	outPath := flags.String("out", "", "generated policy Go file")
	testOutPath := flags.String("test-out", "", "generated policy test Go file")
	packageName := flags.String("package", "", "generated package name")
	vocabPackage := flags.String("vocab-package", "", "vocabulary package alias")
	vocabImport := flags.String("vocab-import", "", "vocabulary package import path")
	cervoImport := flags.String("cervorules-import", "github.com/cervantesh/cervo-rules/v3", "cervo-rules v3 module import path")
	metadataOutPath := flags.String("metadata-out", "", "generated policy metadata JSON, matching schemas/v3/generated-policy-metadata.schema.json")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *vocabPath == "" || *policyPath == "" || *outPath == "" || *packageName == "" || *vocabPackage == "" || *vocabImport == "" {
		fmt.Fprintln(stderr, "-vocab, -policy, -out, -package, -vocab-package and -vocab-import are required")
		return 2
	}
	vocab, policy, closeFiles, err := openInputs(*vocabPath, *policyPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer closeFiles()
	out, err := policygen.Generate(policygen.Options{
		PackageName:       *packageName,
		VocabularyPackage: *vocabPackage,
		VocabularyImport:  *vocabImport,
		CervoRulesImport:  *cervoImport,
		VocabularyReader:  vocab,
		PolicyReader:      policy,
		GeneratedTests:    *testOutPath != "",
	})
	if err != nil {
		fmt.Fprintf(stderr, "generate policy: %v\n", err)
		return exitValidation
	}
	if err := os.WriteFile(*outPath, []byte(out.Source), 0o600); err != nil {
		fmt.Fprintf(stderr, "write policy: %v\n", err)
		return 1
	}
	if *testOutPath != "" {
		if err := os.WriteFile(*testOutPath, []byte(out.TestSource), 0o600); err != nil {
			fmt.Fprintf(stderr, "write policy tests: %v\n", err)
			return 1
		}
	}
	if *metadataOutPath != "" {
		document, err := json.MarshalIndent(generatedPolicyMetadata(out.Metadata), "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "encode policy metadata: %v\n", err)
			return 1
		}
		if err := os.WriteFile(*metadataOutPath, append(document, '\n'), 0o600); err != nil {
			fmt.Fprintf(stderr, "write policy metadata: %v\n", err)
			return 1
		}
	}
	return 0
}

// generatedPolicyMetadata renders the document described by
// schemas/v3/generated-policy-metadata.schema.json.
//
// The schema was published without a producer, so it described a document that
// did not exist. The hashes carry an explicit sha256: prefix here while the
// generated Go embeds them bare: the prefix names the algorithm for a reader of
// the document, and putting it in PolicyMetadata would be a public API change
// for no gain.
func generatedPolicyMetadata(meta policygen.PolicyMetadata) map[string]string {
	return map[string]string{
		"schema_version":  "cervorules.v3.generated_policy_metadata.v1",
		"name":            meta.Name,
		"dsl_version":     meta.DSLVersion,
		"generated_with":  meta.GeneratedWith,
		"vocabulary_hash": "sha256:" + meta.VocabularyHash,
		"policy_hash":     "sha256:" + meta.PolicyHash,
	}
}

func writeCLIError(stderr io.Writer, format string, exitCode int, err core.Error, prefix string) int {
	switch format {
	case "json":
		if encodeErr := json.NewEncoder(stderr).Encode(errorReport{ExitCode: exitCode, Errors: core.Errors{err}}); encodeErr != nil {
			fmt.Fprintf(stderr, "encode error report: %v\n", encodeErr)
			return exitInternal
		}
	default:
		if prefix != "" {
			fmt.Fprintf(stderr, "%s: %v\n", prefix, err.Reason)
		} else {
			fmt.Fprintln(stderr, err.Error())
		}
	}
	return exitCode
}

func openInputs(vocabPath, policyPath string) (io.ReadCloser, io.ReadCloser, func(), error) {
	vocab, err := os.Open(vocabPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open vocabulary: %w", err)
	}
	policy, err := os.Open(policyPath)
	if err != nil {
		vocab.Close()
		return nil, nil, nil, fmt.Errorf("open policy: %w", err)
	}
	closeFiles := func() {
		vocab.Close()
		policy.Close()
	}
	return vocab, policy, closeFiles, nil
}
