package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/cervantesh/cervo-rules/v3/core"
	"github.com/cervantesh/cervo-rules/v3/internal/vocabgen"
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
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("cervorules-vocabgen", flag.ContinueOnError)
	flags.SetOutput(stderr)
	in := flags.String("in", "", "input policy-vocabulary.yaml")
	out := flags.String("out", "", "output Go file")
	packageName := flags.String("package", "", "generated Go package name")
	importPath := flags.String("import", "", "cervo-rules core import path")
	format := flags.String("format", "text", "output format for errors: text or json")
	showVersion := flags.Bool("version", false, "print version")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintf(stderr, "cervorules-vocabgen %s\n", version)
		return 0
	}
	if *in == "" || *out == "" || *packageName == "" {
		fmt.Fprintln(stderr, "-in, -out and -package are required")
		return 2
	}
	input, err := os.Open(*in)
	if err != nil {
		fmt.Fprintf(stderr, "open input: %v\n", err)
		return 1
	}
	defer input.Close()

	source, err := vocabgen.Generate(vocabgen.Options{
		PackageName: *packageName,
		ImportPath:  *importPath,
		Reader:      input,
	})
	if err != nil {
		return writeCLIError(stderr, *format, exitValidation, core.Error{
			Code:      core.ErrorCodeInvalidConfig,
			Severity:  core.SeverityFatal,
			Component: "vocabgen",
			Field:     "vocabulary",
			Reason:    err.Error(),
		}, "generate vocabulary")
	}
	if err := os.WriteFile(*out, []byte(source), 0o600); err != nil {
		fmt.Fprintf(stderr, "write output: %v\n", err)
		return 1
	}
	return 0
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
