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
	case *ast.HTMLBlock:
		return observeHTMLBlock(source, typed)
	case *ast.ListItem:
		return observeListItem(source, typed)
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
	case *extensionast.Strikethrough:
		return observeStrikethrough(source, typed)
	case *extensionast.TableCell:
		return observeTableCell(source, typed)
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
	return parser.Node{Kind: parser.KindParagraph, Range: range_}, true, nil
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
	return parser.Node{Kind: parser.KindHeading, Range: range_, Level: heading.Level}, true, nil
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
	if lines.Len() != 1 {
		return parser.Node{}, false, nil
	}
	line := lines.At(0)
	range_ := parser.Range{Start: line.Start, End: fencedCodeContentEnd(source, line.Start)}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindFencedCode, Range: range_}, true, nil
}

func observeHTMLBlock(source []byte, block *ast.HTMLBlock) (parser.Node, bool, error) {
	range_, ok := htmlBlockSourceRange(source, block)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindHTMLBlock, Range: range_}, true, nil
}

func observeListItem(source []byte, item *ast.ListItem) (parser.Node, bool, error) {
	list, ok := item.Parent().(*ast.List)
	if !ok {
		return parser.Node{}, false, nil
	}
	range_, ok := singleLineListItemContentRange(source, item)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindListItem, Range: range_, Ordered: list.IsOrdered(), Marker: list.Marker}, true, nil
}

func observeAutoLink(source []byte, link *ast.AutoLink) (parser.Node, bool, error) {
	anchor, range_, value, ok := autoLinkSourceRange(source, link)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:          parser.KindAutoLink,
		Range:         range_,
		Anchor:        anchor,
		Value:         value,
		AutoLinkEmail: link.AutoLinkType == ast.AutoLinkEmail,
	}, true, nil
}

func observeCodeSpan(source []byte, span *ast.CodeSpan) (parser.Node, bool, error) {
	range_, ok := simplePlainTextInlineRange(source, span)
	if !ok || span.Pos() < 0 {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindCodeSpan, Range: range_, Anchor: span.Pos()}, true, nil
}

func observeEmphasis(source []byte, emphasis *ast.Emphasis) (parser.Node, bool, error) {
	if emphasis.Level != 1 && emphasis.Level != 2 {
		return parser.Node{}, false, nil
	}
	range_, ok := simplePlainTextInlineRange(source, emphasis)
	if !ok || emphasis.Pos() < 0 {
		return parser.Node{}, false, nil
	}
	kind := parser.KindEmphasis
	if emphasis.Level == 2 {
		kind = parser.KindStrong
	}
	return parser.Node{Kind: kind, Range: range_, Anchor: emphasis.Pos(), Level: emphasis.Level}, true, nil
}

func observeRawHTML(source []byte, raw *ast.RawHTML) (parser.Node, bool, error) {
	range_, ok := singleLineRawHTMLRange(source, raw)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindRawHTML, Range: range_}, true, nil
}

func observeInlineLink(source []byte, link *ast.Link) (parser.Node, bool, error) {
	if link.Reference != nil {
		return parser.Node{}, false, nil
	}
	range_, ok := simpleInlineLinkLabelRange(source, link)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:        parser.KindInlineLink,
		Range:       range_,
		Anchor:      link.Pos(),
		Destination: string(link.Destination),
		Title:       string(link.Title),
		HasTitle:    link.Title != nil,
	}, true, nil
}

func observeStrikethrough(source []byte, strike *extensionast.Strikethrough) (parser.Node, bool, error) {
	range_, ok := simpleStrikethroughContentRange(source, strike)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindStrikethrough, Range: range_}, true, nil
}

func observeTableCell(source []byte, cell *extensionast.TableCell) (parser.Node, bool, error) {
	parent := cell.Parent()
	if parent == nil || parent.Kind() != extensionast.KindTableHeader && parent.Kind() != extensionast.KindTableRow {
		return parser.Node{}, false, nil
	}
	lines := cell.Lines()
	if lines.Len() != 1 {
		return parser.Node{}, false, nil
	}
	segment := lines.At(0)
	range_ := parser.Range{Start: segment.Start, End: segment.Stop}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Node{}, false, nil
	}
	return parser.Node{
		Kind:        parser.KindTableCell,
		Range:       range_,
		TableHeader: parent.Kind() == extensionast.KindTableHeader,
		TableColumn: tableCellColumn(cell),
	}, true, nil
}

func observeTask(source []byte, checkbox *extensionast.TaskCheckBox) (parser.Node, bool, error) {
	parent, ok := checkbox.Parent().(*ast.TextBlock)
	if !ok || parent.Parent() == nil || parent.Parent().Kind() != ast.KindListItem {
		return parser.Node{}, false, nil
	}
	lines := parent.Lines()
	if lines.Len() == 0 {
		return parser.Node{}, false, nil
	}
	first := lines.At(0)
	range_ := parser.Range{Start: first.Start, End: first.Stop}
	if !range_.Valid(len(source)) {
		return parser.Node{}, false, fmt.Errorf("goldmark task anchor range [%d,%d) is outside source length %d", range_.Start, range_.End, len(source))
	}
	return parser.Node{Kind: parser.KindTask, Range: range_, Checked: checkbox.IsChecked}, true, nil
}
