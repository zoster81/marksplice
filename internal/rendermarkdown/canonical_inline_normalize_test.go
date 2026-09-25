package rendermarkdown

import (
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestNormalizeCanonicalInlineASTExposesTrailingTabBoundaryAlternative(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "!\t"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticEmphasis},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticEmphasis},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	if got := normalization.nodes[1].trailingAlternative; got != canonicalInlineRawTabBoundary {
		t.Fatalf("trailing alternative = %d", got)
	}
}

func TestNormalizeCanonicalInlineASTDoesNotInventTabAlternativeWithoutDelimiterBoundary(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "!\t"},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	if got := normalization.nodes[1].trailingAlternative; got != canonicalInlineNoBoundaryAlternative {
		t.Fatalf("trailing alternative = %d", got)
	}
}

func TestNormalizeCanonicalInlineASTTreatsStrikethroughAsDelimiterBoundary(t *testing.T) {
	events := []parser.SemanticEvent{
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "!\t"},
		{Phase: parser.SemanticEnter, Kind: parser.SemanticStrikethrough},
		{Phase: parser.SemanticLeaf, Kind: parser.SemanticText, Value: "x"},
		{Phase: parser.SemanticExit, Kind: parser.SemanticStrikethrough},
	}
	ast, err := buildCanonicalInlineAST(events)
	if err != nil {
		t.Fatal(err)
	}
	normalization, err := normalizeCanonicalInlineAST(ast)
	if err != nil {
		t.Fatal(err)
	}
	if got := normalization.nodes[1].trailingAlternative; got != canonicalInlineRawTabBoundary {
		t.Fatalf("trailing alternative = %d", got)
	}
}
