package goldmark

import "github.com/zoster81/marksplice/internal/parser"

// ValidateNestedBlockquoteBlocks implements the parser-independent Backend contract.
func (a *Adapter) ValidateNestedBlockquoteBlocks(source []byte, outer parser.Range, innerSource []byte, depth int) error {
	return ValidateNestedBlockquoteBlocks(source, outer, innerSource, depth)
}

// ValidateNestedBlockquoteParagraph implements the parser-independent Backend contract.
func (a *Adapter) ValidateNestedBlockquoteParagraph(source []byte, outer parser.Range, contentLines []parser.Range, depth int) error {
	return ValidateNestedBlockquoteParagraph(source, outer, contentLines, depth)
}

// ValidateConstructionInlineHierarchy implements the parser-independent Backend contract.
func (a *Adapter) ValidateConstructionInlineHierarchy(source []byte, expected []parser.ConstructionInlineExpectation, references []parser.ConstructionReferenceInlineExpectation) error {
	return ValidateConstructionInlineHierarchy(source, expected, references)
}

// ValidateConstructionLinkImages implements the parser-independent Backend contract.
func (a *Adapter) ValidateConstructionLinkImages(source []byte, expected []parser.ConstructionLinkImageExpectation) error {
	return ValidateConstructionLinkImages(source, expected)
}

// ValidateConstructionReferenceInlines implements the parser-independent Backend contract.
func (a *Adapter) ValidateConstructionReferenceInlines(source []byte, expected []parser.ConstructionReferenceInlineExpectation) error {
	return ValidateConstructionReferenceInlines(source, expected)
}

// ResolveConstructionReference implements the parser-independent Backend contract.
func (a *Adapter) ResolveConstructionReference(label string, definitions []parser.ConstructionReferenceDefinition) (parser.ConstructionReferenceDefinition, error) {
	return ResolveConstructionReference(label, definitions)
}

// ReferenceLabelKey implements the parser-independent Backend contract.
func (a *Adapter) ReferenceLabelKey(label string) string {
	return ReferenceLabelKey(label)
}

var _ parser.Backend = (*Adapter)(nil)
