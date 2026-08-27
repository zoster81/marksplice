package splice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/source"
)

// ValidateConstructionBlockquoteParagraph combines canonical source proof with
// the semantic GFM proof used only by depth-1 new-document construction.
func ValidateConstructionBlockquoteParagraph(input []byte, outer Range, contentLines []Range) error {
	return ValidateConstructionNestedBlockquoteParagraph(input, outer, contentLines, 1)
}

// ValidateConstructionNestedBlockquoteBlocks combines exact canonical quoting
// proof with the construction-only parser child-sequence proof.
func ValidateConstructionNestedBlockquoteBlocks(input []byte, outer Range, innerSource []byte, depth int) error {
	if err := source.ValidateCanonicalNestedBlockquoteBlocks(input, outer, innerSource, depth); err != nil {
		return fmt.Errorf("validate canonical blockquote blocks: %w", err)
	}
	parserOuter := parser.Range{Start: outer.Start, End: outer.End}
	if err := newParserBackend().ValidateNestedBlockquoteBlocks(input, parserOuter, innerSource, depth); err != nil {
		return fmt.Errorf("validate semantic blockquote blocks: %w", err)
	}
	return nil
}

// ValidateConstructionNestedBlockquoteParagraph combines canonical source proof
// with the construction-only semantic proof for explicit blockquote depth.
func ValidateConstructionNestedBlockquoteParagraph(input []byte, outer Range, contentLines []Range, depth int) error {
	if err := source.ValidateCanonicalNestedBlockquoteParagraph(input, outer, contentLines, depth); err != nil {
		return fmt.Errorf("validate canonical blockquote paragraph: %w", err)
	}

	parserLines := make([]parser.Range, len(contentLines))
	for index, line := range contentLines {
		parserLines[index] = parser.Range{Start: line.Start, End: line.End}
	}
	parserOuter := parser.Range{Start: outer.Start, End: outer.End}
	if err := newParserBackend().ValidateNestedBlockquoteParagraph(input, parserOuter, parserLines, depth); err != nil {
		return fmt.Errorf("validate semantic blockquote paragraph: %w", err)
	}
	return nil
}
