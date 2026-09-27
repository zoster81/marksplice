package rendermarkdown

import (
	"errors"
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
		assertCanonicalEmitStackCleared(t, &workspace.emitter)
	}
}

func TestCanonicalEmitWorkspaceReleasesFramesAfterFailure(t *testing.T) {
	ast, err := buildCanonicalInlineAST([]parser.SemanticEvent{
		{Kind: parser.SemanticLink, Phase: parser.SemanticEnter, Label: "ref", Destination: "/target"},
		{Kind: parser.SemanticText, Phase: parser.SemanticLeaf, Value: "label"},
		{Kind: parser.SemanticLink, Phase: parser.SemanticExit},
	})
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	plan := newCanonicalInlinePlan(ast)
	var workspace canonicalInlineEmitWorkspace
	if _, err := workspace.emit(ast, normalization, plan, canonicalInlineEmitContext{}, false, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("missing reference authority: %v", err)
	}
	assertCanonicalEmitStackCleared(t, &workspace)
	ast.events[0].Label = ""
	output, err := workspace.emit(ast, normalization, plan, canonicalInlineEmitContext{}, false, nil)
	if err != nil || string(output) != "[label](</target>)" {
		t.Fatalf("emission after failure: %q, %v", output, err)
	}
	assertCanonicalEmitStackCleared(t, &workspace)
}

func assertCanonicalEmitStackCleared(t *testing.T, workspace *canonicalInlineEmitWorkspace) {
	t.Helper()
	if len(workspace.stack) != 0 {
		t.Fatal("emission retained active frames")
	}
	for index, frame := range workspace.stack[:cap(workspace.stack)] {
		if !reflect.DeepEqual(frame, canonicalInlineASTEmitFrame{}) {
			t.Fatalf("emission retained frame references at %d", index)
		}
	}
}
