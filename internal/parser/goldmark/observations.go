package goldmark

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"

	"github.com/zoster81/marksplice/internal/parser"
)

func observeNode(source []byte, node ast.Node) (parser.Node, bool, error) {
	switch typed := node.(type) {
	case *ast.Paragraph:
		return observeParagraph(source, typed)
	case *ast.Heading:
		return observeHeading(source, typed)
	case *ast.LinkReferenceDefinition:
		return observeReferenceDefinition(source, typed)
	case *ast.FencedCodeBlock:
		return observeFencedCode(source, typed)
	case *ast.ThematicBreak:
		return observeThematicBreak(source, typed)
	case *ast.Blockquote:
		return observeBlockquote(source, typed)
	case *ast.HTMLBlock:
		return observeHTMLBlock(source, typed)
	case *ast.ListItem:
		return observeListItem(source, typed)
	case *extensionast.TableRow:
		return observeTableRow(source, typed)
	case *ast.AutoLink:
		return observeAutoLink(source, typed)
	case *ast.CodeSpan:
		return observeCodeSpan(source, typed)
	case *ast.Emphasis:
		return observeEmphasis(source, typed)
	case *ast.RawHTML:
		return observeRawHTML(source, typed)
	case *ast.Link:
		return observeInlineLink(source, typed)
	case *ast.Image:
		return observeImage(source, typed)
	case *extensionast.Strikethrough:
		return observeStrikethrough(source, typed)
	case *extensionast.TaskCheckBox:
		return observeTask(source, typed)
	default:
		return parser.Node{}, false, nil
	}
}

func observeParagraph(source []byte, paragraph *ast.Paragraph) (parser.Node, bool, error) {
	lines := paragraph.Lines()
	if lines.Len() == 0 {
		return parser.Node{}, false, nil
	}
	first := lines.At(0)
	last := lines.At(lines.Len() - 1)
	range_ := parser.Range{Start: first.Start, End: paragraphContentEnd(source, last.Stop)}
	if !range_.Valid(len(source)) {
		return parser.Node{}, false, fmt.Errorf("goldmark paragraph range [%d,%d) is outside source length %d", range_.Start, range_.End, len(source))
	}
	return parser.Node{
		Kind:     parser.KindParagraph,
		Range:    range_,
		TopLevel: paragraph.Parent() != nil && paragraph.Parent().Kind() == ast.KindDocument,
	}, true, nil
}

func observeHeading(source []byte, heading *ast.Heading) (parser.Node, bool, error) {
	if heading.Parent() == nil || heading.Parent().Kind() != ast.KindDocument {
		return parser.Node{}, false, nil
	}
	lines := heading.Lines()
	if lines.Len() == 0 {
		return parser.Node{}, false, nil
	}
	first := lines.At(0)
	last := lines.At(lines.Len() - 1)
	range_ := parser.Range{Start: first.Start, End: last.Stop}
	if !range_.Valid(len(source)) {
		return parser.Node{}, false, fmt.Errorf("goldmark heading content range [%d,%d) is outside source length %d", range_.Start, range_.End, len(source))
	}
	return parser.Node{Kind: parser.KindHeading, Range: range_, Level: heading.Level, TopLevel: true}, true, nil
}

func observeReferenceDefinition(source []byte, definition *ast.LinkReferenceDefinition) (parser.Node, bool, error) {
	lines := definition.Lines()
	if lines.Len() != 1 || len(definition.Destination) == 0 {
		return parser.Node{}, false, nil
	}
	segment := lines.At(0)
	range_ := parser.Range{Start: segment.Start, End: paragraphContentEnd(source, segment.Stop)}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:        parser.KindReferenceDefinition,
		Range:       range_,
		Destination: string(definition.Destination),
		Label:       string(definition.Label),
		Title:       string(definition.Title),
		HasTitle:    definition.Title != nil,
	}, true, nil
}

func observeFencedCode(source []byte, block *ast.FencedCodeBlock) (parser.Node, bool, error) {
	lines := block.Lines()
	if lines.Len() == 0 {
		return parser.Node{}, false, nil
	}
	first := lines.At(0)
	last := lines.At(lines.Len() - 1)
	range_ := parser.Range{Start: first.Start, End: fencedCodeContentEnd(source, last.Start)}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindFencedCode, Range: range_}, true, nil
}

func observeThematicBreak(source []byte, thematic *ast.ThematicBreak) (parser.Node, bool, error) {
	start := thematic.Pos()
	if start < 0 || start >= len(source) {
		return parser.Node{}, false, nil
	}
	range_ := parser.Range{Start: start, End: paragraphContentEnd(source, start)}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:     parser.KindThematicBreak,
		Range:    range_,
		TopLevel: thematic.Parent() != nil && thematic.Parent().Kind() == ast.KindDocument,
	}, true, nil
}

func observeBlockquote(source []byte, block *ast.Blockquote) (parser.Node, bool, error) {
	if block.Parent() == nil || block.Parent().Kind() != ast.KindDocument || block.ChildCount() != 1 {
		return parser.Node{}, false, nil
	}
	paragraph, ok := block.FirstChild().(*ast.Paragraph)
	if !ok {
		return parser.Node{}, false, nil
	}
	lines := paragraph.Lines()
	if lines.Len() != 1 {
		return parser.Node{}, false, nil
	}
	segment := lines.At(0)
	contentRange := parser.Range{Start: segment.Start, End: paragraphContentEnd(source, segment.Stop)}
	start := block.Pos()
	range_ := parser.Range{Start: start, End: contentRange.End}
	if start < 0 || !range_.Valid(len(source)) || !contentRange.Valid(len(source)) || range_.Start >= contentRange.Start || contentRange.Start == contentRange.End {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:                   parser.KindBlockquote,
		Range:                  range_,
		BlockquoteContentRange: contentRange,
		TopLevel:               true,
	}, true, nil
}

func observeHTMLBlock(source []byte, block *ast.HTMLBlock) (parser.Node, bool, error) {
	range_, ok := htmlBlockSourceRange(source, block)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindHTMLBlock, Range: range_}, true, nil
}
