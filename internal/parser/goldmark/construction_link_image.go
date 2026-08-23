package goldmark

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/zoster81/marksplice/internal/parser"
)

// ValidateConstructionLinkImages proves direct construction-only link/image
// destination and title semantics independently from child hierarchy proof.
func ValidateConstructionLinkImages(source []byte, expected []parser.ConstructionLinkImageExpectation) error {
	if len(expected) == 0 {
		return nil
	}
	for index, want := range expected {
		if err := validateConstructionLinkImageInput(source, want, index); err != nil {
			return err
		}
	}
	root := newMarkdown().Parser().Parse(text.NewReader(normalizeIsolatedCR(source)))
	paragraph, err := constructionInlineProofParagraph(source, len(source), root)
	if err != nil {
		return err
	}
	actual, err := matchConstructionLinkImageNodes(paragraph, expected)
	if err != nil {
		return err
	}
	for index, want := range expected {
		if err := validateConstructionLinkImageNode(want, actual[index]); err != nil {
			return fmt.Errorf("construction link/image %d: %w", index, err)
		}
	}
	return nil
}

func validateConstructionLinkImageInput(source []byte, want parser.ConstructionLinkImageExpectation, index int) error {
	prefixLength, ok := constructionLinkImagePrefixLength(source, want)
	if !ok || !want.SyntaxRange.Valid(len(source)) || !want.LabelRange.Valid(len(source)) ||
		want.SyntaxRange.Start == want.SyntaxRange.End || want.LabelRange.Start == want.LabelRange.End || want.Destination == "" ||
		want.LabelRange.Start != want.SyntaxRange.Start+prefixLength || want.HasTitle != (want.Title != "") {
		return fmt.Errorf("invalid construction link/image expectation %d", index)
	}
	suffix := constructionLinkImageSuffix(want)
	if want.LabelRange.End+len(suffix) != want.SyntaxRange.End || !bytes.Equal(source[want.LabelRange.End:want.SyntaxRange.End], suffix) {
		return fmt.Errorf("construction link/image syntax %d changed", index)
	}
	return nil
}

func constructionLinkImagePrefixLength(source []byte, want parser.ConstructionLinkImageExpectation) (int, bool) {
	if want.SyntaxRange.Start < 0 || want.SyntaxRange.Start >= len(source) {
		return 0, false
	}
	switch want.Kind {
	case parser.KindInlineLink:
		return 1, source[want.SyntaxRange.Start] == '['
	case parser.KindImage:
		return 2, want.SyntaxRange.Start+1 < len(source) && source[want.SyntaxRange.Start] == '!' && source[want.SyntaxRange.Start+1] == '['
	default:
		return 0, false
	}
}

func constructionLinkImageSuffix(want parser.ConstructionLinkImageExpectation) []byte {
	suffix := "](<" + want.Destination + ">"
	if want.HasTitle {
		suffix += " \"" + want.Title + "\""
	}
	return []byte(suffix + ")")
}

func matchConstructionLinkImageNodes(paragraph *ast.Paragraph, expected []parser.ConstructionLinkImageExpectation) ([]ast.Node, error) {
	byStart := make(map[int][]int, len(expected))
	for index, want := range expected {
		byStart[want.SyntaxRange.Start] = append(byStart[want.SyntaxRange.Start], index)
	}
	actual := make([]ast.Node, len(expected))
	if err := ast.Walk(paragraph, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node == paragraph {
			return ast.WalkContinue, nil
		}
		for _, index := range byStart[node.Pos()] {
			if actual[index] == nil && constructionInlineNodeMatchesKind(node, expected[index].Kind) {
				actual[index] = node
				break
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return nil, fmt.Errorf("walk construction link/image semantics: %w", err)
	}
	for index, node := range actual {
		if node == nil {
			return nil, fmt.Errorf("construction link/image %d changed", index)
		}
	}
	return actual, nil
}

func validateConstructionLinkImageNode(want parser.ConstructionLinkImageExpectation, node ast.Node) error {
	destination, title, reference, ok := constructionLinkImageNodeSemantics(node)
	if !ok || reference != nil {
		return fmt.Errorf("direct link/image kind changed")
	}
	if string(destination) != want.Destination || string(title) != want.Title || (title != nil) != want.HasTitle {
		return fmt.Errorf("destination or title changed")
	}
	return nil
}

func constructionLinkImageNodeSemantics(node ast.Node) (destination, title []byte, reference *ast.ReferenceLink, ok bool) {
	switch typed := node.(type) {
	case *ast.Link:
		return typed.Destination, typed.Title, typed.Reference, true
	case *ast.Image:
		return typed.Destination, typed.Title, typed.Reference, true
	default:
		return nil, nil, nil, false
	}
}
