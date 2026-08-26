package splice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
)

// ConstructionReferenceInlineForm identifies one construction-only reference source form.
type ConstructionReferenceInlineForm uint8

const (
	ConstructionReferenceInlineFull ConstructionReferenceInlineForm = iota
	ConstructionReferenceInlineCollapsed
	ConstructionReferenceInlineShortcut
)

// ConstructionReferenceInlineExpectation is construction-only proof input for
// one reference link or image.
type ConstructionReferenceInlineExpectation struct {
	Kind            Kind
	Form            ConstructionReferenceInlineForm
	SyntaxRange     Range
	LabelRange      Range
	ReferenceRange  Range
	Reference       string
	Destination     string
	Title           string
	HasTitle        bool
	StructuredLabel bool
}

// ValidateConstructionReferenceInlines proves reference link/image semantics
// without promoting ordinary parsed reference inlines.
func ValidateConstructionReferenceInlines(source []byte, expected []ConstructionReferenceInlineExpectation) error {
	converted, err := parserConstructionReferenceInlines(expected)
	if err != nil {
		return err
	}
	return newParserBackend().ValidateConstructionReferenceInlines(source, converted)
}

func parserConstructionReferenceInlines(expected []ConstructionReferenceInlineExpectation) ([]parser.ConstructionReferenceInlineExpectation, error) {
	converted := make([]parser.ConstructionReferenceInlineExpectation, len(expected))
	for index, want := range expected {
		kind, ok := constructionReferenceParserKind(want.Kind)
		if !ok {
			return nil, fmt.Errorf("unsupported construction reference inline kind %d", want.Kind)
		}
		form, ok := constructionReferenceParserForm(want.Form)
		if !ok {
			return nil, fmt.Errorf("unsupported construction reference inline form %d", want.Form)
		}
		converted[index] = parser.ConstructionReferenceInlineExpectation{
			Kind:            kind,
			Form:            form,
			SyntaxRange:     parser.Range{Start: want.SyntaxRange.Start, End: want.SyntaxRange.End},
			LabelRange:      parser.Range{Start: want.LabelRange.Start, End: want.LabelRange.End},
			ReferenceRange:  parser.Range{Start: want.ReferenceRange.Start, End: want.ReferenceRange.End},
			Reference:       want.Reference,
			Destination:     want.Destination,
			Title:           want.Title,
			HasTitle:        want.HasTitle,
			StructuredLabel: want.StructuredLabel,
		}
	}
	return converted, nil
}

func constructionReferenceParserForm(form ConstructionReferenceInlineForm) (parser.ConstructionReferenceInlineForm, bool) {
	switch form {
	case ConstructionReferenceInlineFull:
		return parser.ConstructionReferenceInlineFull, true
	case ConstructionReferenceInlineCollapsed:
		return parser.ConstructionReferenceInlineCollapsed, true
	case ConstructionReferenceInlineShortcut:
		return parser.ConstructionReferenceInlineShortcut, true
	default:
		return parser.ConstructionReferenceInlineFull, false
	}
}

func constructionReferenceParserKind(kind Kind) (parser.Kind, bool) {
	switch kind {
	case KindInlineLink:
		return parser.KindInlineLink, true
	case KindImage:
		return parser.KindImage, true
	default:
		return parser.KindUnknown, false
	}
}
