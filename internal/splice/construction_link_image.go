package splice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

// ConstructionLinkImageExpectation is construction-only semantic proof input
// for one direct inline link or image.
type ConstructionLinkImageExpectation struct {
	Kind        Kind
	SyntaxRange Range
	LabelRange  Range
	Destination string
	Title       string
	HasTitle    bool
}

// ValidateConstructionLinkImages proves direct link/image destination and title
// semantics independently from typed-inline child hierarchy proof.
func ValidateConstructionLinkImages(source []byte, expected []ConstructionLinkImageExpectation) error {
	converted := make([]parser.ConstructionLinkImageExpectation, len(expected))
	for index, want := range expected {
		kind, ok := constructionReferenceParserKind(want.Kind)
		if !ok {
			return fmt.Errorf("unsupported construction link/image kind %d", want.Kind)
		}
		converted[index] = parser.ConstructionLinkImageExpectation{
			Kind:        kind,
			SyntaxRange: parser.Range{Start: want.SyntaxRange.Start, End: want.SyntaxRange.End},
			LabelRange:  parser.Range{Start: want.LabelRange.Start, End: want.LabelRange.End},
			Destination: want.Destination,
			Title:       want.Title,
			HasTitle:    want.HasTitle,
		}
	}
	if err := newParserBackend().ValidateConstructionLinkImages(source, converted); err != nil {
		return fmt.Errorf("validate construction link/image semantics: %w", err)
	}
	return nil
}
