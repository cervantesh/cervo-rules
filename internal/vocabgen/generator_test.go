package vocabgen

import (
	"strings"
	"testing"
)

const sampleVocabulary = `
operations:
  invoice_read:
    go_name: InvoiceRead
targets:
  ledger_api:
    go_name: LedgerAPI
executors:
  ledger_primary:
    go_name: LedgerPrimary
`

func TestGenerateV3VocabularyConstants(t *testing.T) {
	source, err := Generate(Options{
		PackageName: "policyvocab",
		Reader:      strings.NewReader(sampleVocabulary),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	required := []string{
		`import cervorules "github.com/cervantesh/cervo-rules/v3/core"`,
		`OperationInvoiceRead cervorules.Operation = "invoice_read"`,
		`TargetLedgerAPI cervorules.Target = "ledger_api"`,
		`ExecutorLedgerPrimary cervorules.Executor = "ledger_primary"`,
		`func Vocabulary() cervorules.Vocabulary`,
		`cervorules.AllowedOperations(`,
		`cervorules.AllowedTargets(`,
		`cervorules.AllowedExecutors(`,
	}
	for _, want := range required {
		if !strings.Contains(source, want) {
			t.Fatalf("generated source missing %q:\n%s", want, source)
		}
	}
	forbidden := []string{"Capability", "Service", "Provider"}
	for _, value := range forbidden {
		if strings.Contains(source, value) {
			t.Fatalf("generated v3 source must not include v2 primitive %q:\n%s", value, source)
		}
	}
}

func TestGenerateV3VocabularyCustomImport(t *testing.T) {
	source, err := Generate(Options{
		PackageName: "policyvocab",
		ImportPath:  "example.test/cervorules/core",
		Reader:      strings.NewReader(sampleVocabulary),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(source, `import cervorules "example.test/cervorules/core"`) {
		t.Fatalf("custom import not used:\n%s", source)
	}
}

func TestGenerateV3VocabularyDefaultNamesAndEmptySections(t *testing.T) {
	source, err := Generate(Options{
		PackageName: "policyvocab",
		Reader: strings.NewReader(`
operations:
  read_item: {}
targets: {}
executors: {}
`),
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(source, `OperationReadItem cervorules.Operation = "read_item"`) {
		t.Fatalf("default generated name missing:\n%s", source)
	}
	if strings.Contains(source, "const (\n)\n") {
		t.Fatalf("empty const block emitted:\n%s", source)
	}
}

func TestGenerateV3VocabularyRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want string
	}{
		{
			name: "invalid package",
			opts: Options{PackageName: "bad-name", Reader: strings.NewReader(sampleVocabulary)},
			want: "invalid package",
		},
		{
			name: "missing reader",
			opts: Options{PackageName: "policyvocab"},
			want: "missing input reader",
		},
		{
			name: "duplicate go names",
			opts: Options{PackageName: "policyvocab", Reader: strings.NewReader(`
operations:
  read_one:
    go_name: Read
  read_two:
    go_name: Read
targets: {}
executors: {}
`)},
			want: "duplicate generated constant",
		},
		{
			name: "v2 field rejected",
			opts: Options{PackageName: "policyvocab", Reader: strings.NewReader(`
capabilities:
  read: {}
targets: {}
executors: {}
`)},
			want: "decode vocabulary spec",
		},
		{
			name: "malformed yaml",
			opts: Options{PackageName: "policyvocab", Reader: strings.NewReader("operations: [")},
			want: "decode vocabulary spec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Generate(tt.opts)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}
