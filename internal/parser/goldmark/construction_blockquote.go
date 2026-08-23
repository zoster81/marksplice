package goldmark

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"

	"github.com/zoster81/marksplice/internal/parser"
)

// ValidateTopLevelBlockquoteParagraph proves the depth-1 construction shape.
func ValidateTopLevelBlockquoteParagraph(source []byte, outer parser.Range, contentLines []parser.Range) error {
	return ValidateNestedBlockquoteParagraph(source, outer, contentLines, 1)
}

// ValidateNestedBlockquoteBlocks proves an exact construction-only blockquote
// hierarchy whose innermost container has the same reviewed child-block sequence
// as standalone canonical innerSource.
func ValidateNestedBlockquoteBlocks(source []byte, outer parser.Range, innerSource []byte, depth int) error {
	if depth < 1 || !outer.Valid(len(source)) || outer.Start == outer.End || len(innerSource) == 0 || innerSource[len(innerSource)-1] != '\n' {
		return fmt.Errorf("invalid blockquote block construction input")
	}

	root := newMarkdown().Parser().Parse(text.NewReader(normalizeIsolatedCR(source)))
	block := topLevelBlockquoteAt(root, outer.Start)
	container, ok := nestedBlockquoteAtDepth(block, depth)
	if !ok {
		return fmt.Errorf("expected exact nested blockquote hierarchy")
	}
	innerRoot := newMarkdown().Parser().Parse(text.NewReader(innerSource))
	if !sameConstructionBlockSequence(innerRoot, innerSource, container, source) {
		return fmt.Errorf("blockquote child sequence changed")
	}
	return nil
}

// ValidateNestedBlockquoteParagraph proves that one exact source range parses
// as exactly depth nested blockquotes containing one paragraph with the requested
// physical content-line ranges. It is a construction-only semantic proof and
// does not widen the observations returned by Adapter.Parse.
func ValidateNestedBlockquoteParagraph(source []byte, outer parser.Range, contentLines []parser.Range, depth int) error {
	if depth < 1 || !outer.Valid(len(source)) || outer.Start == outer.End || len(contentLines) == 0 {
		return fmt.Errorf("invalid blockquote construction ranges")
	}

	root := newMarkdown().Parser().Parse(text.NewReader(normalizeIsolatedCR(source)))
	block := topLevelBlockquoteAt(root, outer.Start)
	paragraph, ok := nestedBlockquoteParagraph(block, depth)
	if !ok {
		return fmt.Errorf("expected exact nested blockquote paragraph hierarchy")
	}
	lines := paragraph.Lines()
	if lines.Len() != len(contentLines) {
		return fmt.Errorf("blockquote paragraph line count changed")
	}
	for index, want := range contentLines {
		segment := lines.At(index)
		if segment.Start != want.Start {
			return fmt.Errorf("blockquote paragraph line %d start = %d, want %d", index, segment.Start, want.Start)
		}
		if index == len(contentLines)-1 && paragraphContentEnd(source, segment.Stop) != want.End {
			return fmt.Errorf("blockquote paragraph final line end changed")
		}
	}
	if contentLines[len(contentLines)-1].End != outer.End {
		return fmt.Errorf("blockquote paragraph outer end changed")
	}
	return nil
}

func nestedBlockquoteParagraph(block *ast.Blockquote, depth int) (*ast.Paragraph, bool) {
	container, ok := nestedBlockquoteAtDepth(block, depth)
	if !ok || container.ChildCount() != 1 {
		return nil, false
	}
	paragraph, ok := container.FirstChild().(*ast.Paragraph)
	return paragraph, ok
}

func nestedBlockquoteAtDepth(block *ast.Blockquote, depth int) (*ast.Blockquote, bool) {
	if block == nil || depth < 1 {
		return nil, false
	}
	current := block
	for level := 1; level < depth; level++ {
		if current.ChildCount() != 1 {
			return nil, false
		}
		nested, ok := current.FirstChild().(*ast.Blockquote)
		if !ok {
			return nil, false
		}
		current = nested
	}
	return current, true
}

type constructionBlockPair struct {
	expected ast.Node
	actual   ast.Node
}

func sameConstructionBlockSequence(expectedRoot ast.Node, expectedSource []byte, actual *ast.Blockquote, actualSource []byte) bool {
	stack := []constructionBlockPair{{expected: expectedRoot.FirstChild(), actual: actual.FirstChild()}}
	for len(stack) != 0 {
		pair := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		expected := pair.expected
		got := pair.actual
		for expected != nil && got != nil {
			if !sameConstructionBlock(expected, expectedSource, got, actualSource) {
				return false
			}
			if constructionBlockHasChildren(expected) {
				stack = append(stack, constructionBlockPair{expected: expected.FirstChild(), actual: got.FirstChild()})
			}
			expected = expected.NextSibling()
			got = got.NextSibling()
		}
		if expected != nil || got != nil {
			return false
		}
	}
	return true
}

func constructionBlockHasChildren(node ast.Node) bool {
	switch node.(type) {
	case *ast.Blockquote, *ast.List, *ast.ListItem, *extensionast.Table, *extensionast.TableHeader, *extensionast.TableRow:
		return true
	default:
		return false
	}
}

func sameConstructionBlock(expected ast.Node, expectedSource []byte, actual ast.Node, actualSource []byte) bool {
	switch want := expected.(type) {
	case *ast.Paragraph:
		got, ok := actual.(*ast.Paragraph)
		return ok && sameConstructionLines(want.Lines(), expectedSource, got.Lines(), actualSource)
	case *ast.TextBlock:
		got, ok := actual.(*ast.TextBlock)
		return ok && sameConstructionLines(want.Lines(), expectedSource, got.Lines(), actualSource)
	case *ast.Heading:
		got, ok := actual.(*ast.Heading)
		return ok && want.Level == got.Level && sameConstructionLines(want.Lines(), expectedSource, got.Lines(), actualSource)
	case *ast.ThematicBreak:
		_, ok := actual.(*ast.ThematicBreak)
		return ok
	case *ast.FencedCodeBlock:
		got, ok := actual.(*ast.FencedCodeBlock)
		return ok && sameConstructionFencedCode(want, expectedSource, got, actualSource)
	case *ast.Blockquote:
		_, ok := actual.(*ast.Blockquote)
		return ok
	case *ast.List, *ast.ListItem:
		return sameConstructionListBlock(expected, actual)
	case *ast.LinkReferenceDefinition, *extensionast.Table, *extensionast.TableHeader, *extensionast.TableRow, *extensionast.TableCell:
		return sameConstructionReferenceOrTableBlock(expected, actual)
	default:
		return false
	}
}

func sameConstructionListBlock(expected, actual ast.Node) bool {
	switch want := expected.(type) {
	case *ast.List:
		got, ok := actual.(*ast.List)
		return ok && want.Marker == got.Marker && want.Start == got.Start && want.IsTight == got.IsTight
	case *ast.ListItem:
		_, ok := actual.(*ast.ListItem)
		return ok
	default:
		return false
	}
}

func sameConstructionReferenceOrTableBlock(expected, actual ast.Node) bool {
	switch want := expected.(type) {
	case *ast.LinkReferenceDefinition:
		got, ok := actual.(*ast.LinkReferenceDefinition)
		return ok && bytes.Equal(want.Label, got.Label) && bytes.Equal(want.Destination, got.Destination) && bytes.Equal(want.Title, got.Title)
	case *extensionast.Table:
		got, ok := actual.(*extensionast.Table)
		return ok && slices.Equal(want.Alignments, got.Alignments)
	case *extensionast.TableHeader:
		got, ok := actual.(*extensionast.TableHeader)
		return ok && slices.Equal(want.Alignments, got.Alignments)
	case *extensionast.TableRow:
		got, ok := actual.(*extensionast.TableRow)
		return ok && slices.Equal(want.Alignments, got.Alignments)
	case *extensionast.TableCell:
		got, ok := actual.(*extensionast.TableCell)
		return ok && want.Alignment == got.Alignment
	default:
		return false
	}
}

func sameConstructionFencedCode(expected *ast.FencedCodeBlock, expectedSource []byte, actual *ast.FencedCodeBlock, actualSource []byte) bool {
	if !sameConstructionLines(expected.Lines(), expectedSource, actual.Lines(), actualSource) {
		return false
	}
	if expected.Info == nil || actual.Info == nil {
		return expected.Info == nil && actual.Info == nil
	}
	expectedInfo := expected.Info.Segment
	actualInfo := actual.Info.Segment
	return bytes.Equal(expectedInfo.Value(expectedSource), actualInfo.Value(actualSource))
}

func sameConstructionLines(expected *text.Segments, expectedSource []byte, actual *text.Segments, actualSource []byte) bool {
	if expected.Len() != actual.Len() {
		return false
	}
	for index := 0; index < expected.Len(); index++ {
		expectedSegment := expected.At(index)
		actualSegment := actual.At(index)
		if !bytes.Equal(expectedSegment.Value(expectedSource), actualSegment.Value(actualSource)) {
			return false
		}
	}
	return true
}

func topLevelBlockquoteAt(root ast.Node, start int) *ast.Blockquote {
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		block, ok := child.(*ast.Blockquote)
		if ok && block.Pos() == start {
			return block
		}
	}
	return nil
}
