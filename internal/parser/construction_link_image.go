package parser

// ConstructionLinkImageExpectation describes one construction-only direct link
// or image semantic proof without widening ordinary parser observations.
type ConstructionLinkImageExpectation struct {
	Kind        Kind
	SyntaxRange Range
	LabelRange  Range
	Destination string
	Title       string
	HasTitle    bool
}
