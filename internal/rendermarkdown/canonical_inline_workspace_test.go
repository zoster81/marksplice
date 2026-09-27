package rendermarkdown

import (
	"reflect"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestCanonicalProofWorkspaceRetainsIndependentCandidates(t *testing.T) {
	var workspace canonicalInlineProofWorkspace
	var retained, expected []canonicalInlineCandidate
	for _, depth := range []int{1, 8, 0, 3, 16, 1, 0} {
		var events []parser.SemanticEvent
		for range depth {
			events = append(events, parser.SemanticEvent{Kind: parser.SemanticEmphasis, Phase: parser.SemanticEnter})
		}
		events = append(events, parser.SemanticEvent{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "a & b"})
		for range depth {
			events = append(events, parser.SemanticEvent{Kind: parser.SemanticEmphasis, Phase: parser.SemanticExit})
		}
		ast, err := buildCanonicalInlineAST(events)
		if err != nil {
			t.Fatal(err)
		}
		normalization, err := normalizeCanonicalInlineAST(ast)
		if err != nil {
			t.Fatal(err)
		}
		plan := newCanonicalInlinePlan(ast)
		for index := 1; index < len(ast.nodes); index++ {
			plan.nodes[index].marker = '*'
			if index%2 == 0 {
				plan.nodes[index].marker = '_'
			}
		}
		got, err := workspace.candidateForPlan(ast, normalization, plan, canonicalInlineEmitContext{}, false)
		if err != nil {
			t.Fatal(err)
		}
		want, err := canonicalInlineCandidateForPlan(ast, normalization, plan, canonicalInlineEmitContext{}, false)
		if err != nil {
			t.Fatal(err)
		}
		retained = append(retained, got)
		expected = append(expected, want)
		if !reflect.DeepEqual(retained, expected) {
			t.Fatalf("workspace reuse at depth %d changed a current or retained candidate", depth)
		}
	}
}
