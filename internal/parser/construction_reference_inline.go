package parser

// ConstructionReferenceInlineForm identifies the source form of one
// construction-only reference link or image.
type ConstructionReferenceInlineForm uint8

const (
	ConstructionReferenceInlineFull ConstructionReferenceInlineForm = iota
	ConstructionReferenceInlineCollapsed
	ConstructionReferenceInlineShortcut
)

// ConstructionReferenceDefinition is one construction-only reference target.
type ConstructionReferenceDefinition struct {
	Label       string
	Destination string
	Title       string
	HasTitle    bool
}

// ConstructionReferenceInlineExpectation describes one construction-only
// reference link or image without widening ordinary parser observations.
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
