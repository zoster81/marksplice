package parser

// ConstructionReferenceInlineExpectation describes one construction-only full
// reference link or image without widening ordinary parser observations.
type ConstructionReferenceInlineExpectation struct {
	Kind           Kind
	SyntaxRange    Range
	LabelRange     Range
	ReferenceRange Range
	Reference      string
	Destination    string
	Title          string
	HasTitle       bool
}
