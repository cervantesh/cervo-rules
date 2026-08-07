package metamorphic

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/cervantesh/cervo-rules/v3/internal/policygen"
	"github.com/cervantesh/cervo-rules/v3/internal/vocabgen"
)

// variants are the four policies each family produces. base is the subject;
// the other three are transformations whose effect on decisions is known in
// advance, which is what makes them an oracle.
const (
	variantBase      = "base"
	variantExtraDeny = "extradeny"
	variantInert     = "inert"
	variantDeMorgan  = "demorgan"
)

func (p randomPolicy) variantYAML(variant string, rng *rand.Rand) string {
	switch variant {
	case variantBase:
		return p.policyYAML(variant, nil, nil)
	case variantExtraDeny:
		return p.policyYAML(variant, nil, &randomDeny{
			ID:     "deny-appended",
			Reason: "the appended rule refused it",
			When:   randomPredicate(rng, p.Facts, 1),
		})
	case variantInert:
		return p.policyYAML(variant, nil, &randomDeny{
			ID:     "deny-unsatisfiable",
			Reason: "this rule can never hold",
			When:   p.unsatisfiable(),
		})
	case variantDeMorgan:
		return p.policyYAML(variant, deMorgan, nil)
	}
	panic("unknown variant " + variant)
}

// Before any metamorphic claim can mean anything, the generator has to produce
// policies the real toolchain accepts. A generator that silently emitted
// something policygen rejects would make every property vacuously true.
func TestRandomPoliciesAreAcceptedByTheGenerator(t *testing.T) {
	rng := rand.New(rand.NewSource(1))

	for i := 0; i < 40; i++ {
		policy := newRandomPolicy(rng, i)
		vocabYAML := policy.vocabularyYAML()

		if _, err := vocabgen.Generate(vocabgen.Options{
			PackageName: "policyvocab",
			ImportPath:  "github.com/cervantesh/cervo-rules/v3/core",
			Reader:      strings.NewReader(vocabYAML),
		}); err != nil {
			t.Fatalf("policy %d: vocabulary rejected: %v\n%s", i, err, vocabYAML)
		}

		for _, variant := range []string{variantBase, variantExtraDeny, variantInert, variantDeMorgan} {
			policyYAML := policy.variantYAML(variant, rng)
			if _, err := policygen.Generate(policygen.Options{
				PackageName:       "policyrules",
				VocabularyPackage: "policyvocab",
				VocabularyImport:  "example.test/generated/policyvocab",
				VocabularyReader:  strings.NewReader(vocabYAML),
				PolicyReader:      strings.NewReader(policyYAML),
			}); err != nil {
				t.Fatalf("policy %d variant %s rejected: %v\n%s", i, variant, err, policyYAML)
			}
		}
	}
}
