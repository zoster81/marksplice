package rendermarkdown

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestBuildCanonicalInlineASTOwnsStructureFromSemanticEvents(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "a"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticStrong},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "b"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrong},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "c"},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	if len(ast.nodes) != 6 {
		t.Fatalf("nodes = %d, want 6", len(ast.nodes))
	}
	outer := ast.nodes[1]
	if outer.parent != ast.root || outer.firstChild != 2 || outer.nextSibling != 5 || outer.exitIndex != 5 {
		t.Fatalf("outer = %#v", outer)
	}
	textA := ast.nodes[2]
	if textA.parent != 1 || textA.nextSibling != 3 {
		t.Fatalf("text a = %#v", textA)
	}
	strong := ast.nodes[3]
	if strong.parent != 1 || strong.firstChild != 4 || strong.exitIndex != 4 {
		t.Fatalf("strong = %#v", strong)
	}
	if got := ast.events[ast.nodes[5].eventIndex].Value; got != "c" {
		t.Fatalf("root sibling value = %q", got)
	}
}

func TestBuildCanonicalInlineASTRejectsMismatchedClose(t *testing.T) {
	_, err := buildCanonicalInlineAST([]parser.SemanticEvent{
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrong},
	})
	if err == nil {
		t.Fatal("expected mismatched close error")
	}
}
