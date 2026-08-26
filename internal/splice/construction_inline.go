package splice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

// ConstructionInlineExpectation is one construction-only semantic inline proof
// fact. Parent is -1 for a paragraph child or the index of another expectation.
type ConstructionInlineExpectation struct {
	Kind            Kind
	SyntaxRange     Range
	ContentRange    Range
	Marker          byte
	DelimiterLength int
	Parent          int
}

// ValidateConstructionInlineHierarchy proves generated typed inline nesting
// through the isolated Goldmark construction validator without widening ordinary
// parsed-source promotion.
func ValidateConstructionInlineHierarchy(source []byte, expected []ConstructionInlineExpectation, references []ConstructionReferenceInlineExpectation) error {
	parserExpected := make([]parser.ConstructionInlineExpectation, len(expected))
	for index, want := range expected {
		kind, ok := parserConstructionInlineKind(want.Kind)
		if !ok {
			return fmt.Errorf("unsupported construction inline kind %d", want.Kind)
		}
		parserExpected[index] = parser.ConstructionInlineExpectation{
			Kind:            kind,
			SyntaxRange:     parser.Range{Start: want.SyntaxRange.Start, End: want.SyntaxRange.End},
			ContentRange:    parser.Range{Start: want.ContentRange.Start, End: want.ContentRange.End},
			Marker:          want.Marker,
			DelimiterLength: want.DelimiterLength,
			Parent:          want.Parent,
		}
	}
	parserReferences, err := parserConstructionReferenceInlines(references)
	if err != nil {
		return err
	}
	if err := newParserBackend().ValidateConstructionInlineHierarchy(source, parserExpected, parserReferences); err != nil {
		return fmt.Errorf("validate semantic typed inline hierarchy: %w", err)
	}
	return nil
}

func parserConstructionInlineKind(kind Kind) (parser.Kind, bool) {
	switch kind {
	case KindCodeSpan:
		return parser.KindCodeSpan, true
	case KindEmphasis:
		return parser.KindEmphasis, true
	case KindStrong:
		return parser.KindStrong, true
	case KindStrikethrough:
		return parser.KindStrikethrough, true
	case KindInlineLink:
		return parser.KindInlineLink, true
	case KindImage:
		return parser.KindImage, true
	default:
		return parser.KindUnknown, false
	}
}
