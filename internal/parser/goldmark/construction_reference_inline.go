package goldmark

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/zoster81/marksplice/internal/parser"
)

// ValidateConstructionReferenceInlines proves construction-only full reference
// links/images against their already-resolved definition semantics. It does not
// widen Adapter.Parse observations.
func ValidateConstructionReferenceInlines(source []byte, expected []parser.ConstructionReferenceInlineExpectation) error {
	if len(expected) == 0 {
		return nil
	}
	for index, want := range expected {
		if err := validateConstructionReferenceInlineInput(source, want); err != nil {
			return fmt.Errorf("reference inline %d: %w", index, err)
		}
	}
	proofSource, err := constructionReferenceProofSource(source, expected)
	if err != nil {
		return err
	}
	root := newMarkdown().Parser().Parse(text.NewReader(proofSource))
	paragraph, ok := root.FirstChild().(*ast.Paragraph)
	if !ok || paragraph == nil {
		return fmt.Errorf("reference inline proof paragraph changed")
	}
	lines := paragraph.Lines()
	if lines.Len() != 1 {
		return fmt.Errorf("reference inline proof paragraph line count changed")
	}
	line := lines.At(0)
	if line.Start != 0 || paragraphContentEnd(proofSource, line.Stop) != len(source) {
		return fmt.Errorf("reference inline proof paragraph range changed")
	}

	actual := constructionReferenceInlineNodes(paragraph)
	if len(actual) != len(expected) {
		return fmt.Errorf("reference inline semantic count changed")
	}
	for index, want := range expected {
		node := findConstructionReferenceInline(actual, want)
		if node == nil {
			return fmt.Errorf("reference inline %d semantic node changed", index)
		}
		if err := validateConstructionReferenceInlineNode(proofSource, want, node); err != nil {
			return fmt.Errorf("reference inline %d: %w", index, err)
		}
	}
	return nil
}

func validateConstructionReferenceInlineInput(source []byte, want parser.ConstructionReferenceInlineExpectation) error {
	prefixLength, err := constructionReferencePrefixLength(source, want)
	if err != nil {
		return err
	}
	if err := validateConstructionReferenceInlineRanges(source, want, prefixLength); err != nil {
		return err
	}
	return validateConstructionReferenceInlineSyntax(source, want)
}

func constructionReferencePrefixLength(source []byte, want parser.ConstructionReferenceInlineExpectation) (int, error) {
	switch want.Kind {
	case parser.KindInlineLink:
		if want.SyntaxRange.Start >= len(source) || source[want.SyntaxRange.Start] != '[' {
			return 0, fmt.Errorf("link reference prefix changed")
		}
		return 1, nil
	case parser.KindImage:
		if want.SyntaxRange.Start+1 >= len(source) || source[want.SyntaxRange.Start] != '!' || source[want.SyntaxRange.Start+1] != '[' {
			return 0, fmt.Errorf("image reference prefix changed")
		}
		return 2, nil
	default:
		return 0, fmt.Errorf("unsupported reference inline kind %d", want.Kind)
	}
}

func validateConstructionReferenceInlineRanges(source []byte, want parser.ConstructionReferenceInlineExpectation, prefixLength int) error {
	if !constructionReferenceRangesValid(source, want) {
		return fmt.Errorf("invalid reference inline ranges")
	}
	if want.LabelRange.Start != want.SyntaxRange.Start+prefixLength || want.ReferenceRange.Start != want.LabelRange.End+2 || want.SyntaxRange.End != want.ReferenceRange.End+1 {
		return fmt.Errorf("reference inline boundary changed")
	}
	return nil
}

func constructionReferenceRangesValid(source []byte, want parser.ConstructionReferenceInlineExpectation) bool {
	return want.SyntaxRange.Valid(len(source)) && want.LabelRange.Valid(len(source)) && want.ReferenceRange.Valid(len(source)) &&
		want.SyntaxRange.Start != want.SyntaxRange.End && want.LabelRange.Start != want.LabelRange.End &&
		want.ReferenceRange.Start != want.ReferenceRange.End && want.Reference != ""
}

func validateConstructionReferenceInlineSyntax(source []byte, want parser.ConstructionReferenceInlineExpectation) error {
	if source[want.LabelRange.End] != ']' || source[want.LabelRange.End+1] != '[' || source[want.ReferenceRange.End] != ']' ||
		string(source[want.ReferenceRange.Start:want.ReferenceRange.End]) != want.Reference {
		return fmt.Errorf("reference inline syntax changed")
	}
	return nil
}

func constructionReferenceProofSource(source []byte, expected []parser.ConstructionReferenceInlineExpectation) ([]byte, error) {
	proof := make([]byte, 0, len(source)+len(expected)*32)
	proof = append(proof, source...)
	proof = append(proof, '\n', '\n')
	definitions := make(map[string]parser.ConstructionReferenceInlineExpectation, len(expected))
	order := make([]string, 0, len(expected))
	for _, want := range expected {
		if previous, ok := definitions[want.Reference]; ok {
			if previous.Destination != want.Destination || previous.Title != want.Title || previous.HasTitle != want.HasTitle {
				return nil, fmt.Errorf("reference %q resolves inconsistently", want.Reference)
			}
			continue
		}
		definitions[want.Reference] = want
		order = append(order, want.Reference)
	}
	for _, reference := range order {
		want := definitions[reference]
		proof = append(proof, '[')
		proof = append(proof, reference...)
		proof = append(proof, ']', ':', ' ', '<')
		proof = append(proof, want.Destination...)
		proof = append(proof, '>')
		if want.HasTitle {
			proof = append(proof, ' ', '"')
			proof = append(proof, want.Title...)
			proof = append(proof, '"')
		}
		proof = append(proof, '\n')
	}
	return proof, nil
}

func constructionReferenceInlineNodes(paragraph *ast.Paragraph) []ast.Node {
	result := make([]ast.Node, 0, 2)
	_ = ast.Walk(paragraph, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node == paragraph {
			return ast.WalkContinue, nil
		}
		switch typed := node.(type) {
		case *ast.Link:
			if typed.Reference != nil {
				result = append(result, typed)
			}
		case *ast.Image:
			if typed.Reference != nil {
				result = append(result, typed)
			}
		}
		return ast.WalkContinue, nil
	})
	return result
}

func findConstructionReferenceInline(actual []ast.Node, want parser.ConstructionReferenceInlineExpectation) ast.Node {
	for _, node := range actual {
		if node.Pos() != want.SyntaxRange.Start {
			continue
		}
		switch want.Kind {
		case parser.KindInlineLink:
			if _, ok := node.(*ast.Link); ok {
				return node
			}
		case parser.KindImage:
			if _, ok := node.(*ast.Image); ok {
				return node
			}
		}
	}
	return nil
}

func validateConstructionReferenceInlineNode(source []byte, want parser.ConstructionReferenceInlineExpectation, node ast.Node) error {
	var reference *ast.ReferenceLink
	var destination, title []byte
	switch typed := node.(type) {
	case *ast.Link:
		reference = typed.Reference
		destination = typed.Destination
		title = typed.Title
	case *ast.Image:
		reference = typed.Reference
		destination = typed.Destination
		title = typed.Title
	default:
		return fmt.Errorf("reference inline kind changed")
	}
	if reference == nil || reference.Type != ast.ReferenceLinkFull || string(reference.Value) != want.Reference {
		return fmt.Errorf("reference type or value changed")
	}
	if string(destination) != want.Destination || string(title) != want.Title || (title != nil) != want.HasTitle {
		return fmt.Errorf("resolved reference semantics changed")
	}
	label, ok := simplePlainTextInlineRange(source, node)
	if !ok || label != want.LabelRange {
		return fmt.Errorf("reference label range changed")
	}
	if !bytes.Equal(source[want.ReferenceRange.Start:want.ReferenceRange.End], []byte(want.Reference)) {
		return fmt.Errorf("reference source value changed")
	}
	return nil
}
