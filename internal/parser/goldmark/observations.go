package goldmark

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	extensionast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/util"

	"github.com/zoster81/marksplice/internal/parser"
)

func observeNode(source []byte, node ast.Node) (parser.Node, bool, error) {
	if observed, matched, err := observeBlockNode(source, node); matched || err != nil {
		return observed, matched, err
	}
	return observeInlineNode(source, node)
}

func observeBlockNode(source []byte, node ast.Node) (parser.Node, bool, error) {
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
	case *extensionast.Table:
		return observeTable(source, typed)
	case *extensionast.TableRow:
		return observeTableRow(source, typed)
	default:
		return parser.Node{}, false, nil
	}
}

func observeInlineNode(source []byte, node ast.Node) (parser.Node, bool, error) {
	switch typed := node.(type) {
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
	text, err := headingSemanticText(source, heading)
	if err != nil {
		return parser.Node{}, false, fmt.Errorf("goldmark heading semantic text: %w", err)
	}
	return parser.Node{Kind: parser.KindHeading, Range: range_, Level: heading.Level, HeadingText: text, TopLevel: true}, true, nil
}

func headingSemanticText(source []byte, heading *ast.Heading) (string, error) {
	var text strings.Builder
	err := ast.Walk(heading, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node == heading {
			return ast.WalkContinue, nil
		}
		switch typed := node.(type) {
		case *ast.Text:
			value := typed.Value(source)
			if !typed.IsRaw() && (typed.Parent() == nil || typed.Parent().Kind() != ast.KindCodeSpan) {
				value = resolveHeadingText(value)
			}
			_, _ = text.Write(value)
		case *ast.String:
			value := typed.Value
			if !typed.IsRaw() && !typed.IsCode() {
				value = resolveHeadingText(value)
			}
			_, _ = text.Write(value)
		}
		return ast.WalkContinue, nil
	})
	return text.String(), err
}

func resolveHeadingText(value []byte) []byte {
	value = util.UnescapePunctuations(value)
	value = util.ResolveNumericReferences(value)
	return util.ResolveEntityNames(value)
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
	anchor := block.Pos()
	if anchor < 0 || anchor >= len(source) {
		return parser.Node{}, false, nil
	}
	lines := block.Lines()
	contentRanges := make([]parser.Range, lines.Len())
	for index := 0; index < lines.Len(); index++ {
		segment := lines.At(index)
		range_ := parser.Range{Start: segment.Start, End: fencedCodeContentEnd(source, segment.Start)}
		if !range_.Valid(len(source)) {
			return parser.Node{}, false, fmt.Errorf("goldmark fenced-code content range [%d,%d) is outside source length %d", range_.Start, range_.End, len(source))
		}
		contentRanges[index] = range_
	}
	range_ := parser.Range{Start: anchor, End: anchor}
	if len(contentRanges) != 0 {
		range_ = parser.Range{Start: contentRanges[0].Start, End: contentRanges[len(contentRanges)-1].End}
	}
	info := ""
	if block.Info != nil {
		info = string(block.Info.Segment.Value(source))
	}
	return parser.Node{
		Kind:                    parser.KindFencedCode,
		Range:                   range_,
		Anchor:                  anchor,
		FencedCodeContentRanges: contentRanges,
		FencedCodeInfo:          info,
		FencedCodeLanguage:      string(block.Language(source)),
		TopLevel:                block.Parent() != nil && block.Parent().Kind() == ast.KindDocument,
	}, true, nil
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
	if block.Parent() == nil || block.Parent().Kind() != ast.KindDocument {
		return parser.Node{}, false, nil
	}
	start := block.Pos()
	if start < 0 || start >= len(source) {
		return parser.Node{}, false, nil
	}
	range_ := parser.Range{Start: start, End: paragraphContentEnd(source, start)}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Node{}, false, nil
	}
	semanticRanges, err := blockquoteSemanticRanges(source, block)
	if err != nil {
		return parser.Node{}, false, err
	}
	return parser.Node{
		Kind:                     parser.KindBlockquote,
		Range:                    range_,
		BlockquoteContentRange:   simpleBlockquoteContentRange(source, block),
		BlockquoteSemanticRanges: semanticRanges,
		TopLevel:                 true,
	}, true, nil
}

func simpleBlockquoteContentRange(source []byte, block *ast.Blockquote) parser.Range {
	if block.ChildCount() != 1 {
		return parser.Range{}
	}
	paragraph, ok := block.FirstChild().(*ast.Paragraph)
	if !ok || paragraph.Lines().Len() != 1 {
		return parser.Range{}
	}
	segment := paragraph.Lines().At(0)
	range_ := parser.Range{Start: segment.Start, End: paragraphContentEnd(source, segment.Stop)}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Range{}
	}
	return range_
}

func blockquoteSemanticRanges(source []byte, block *ast.Blockquote) ([]parser.Range, error) {
	ranges := make([]parser.Range, 0)
	err := ast.Walk(block, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node == block || node.Type() != ast.TypeBlock {
			return ast.WalkContinue, nil
		}
		lines := node.Lines()
		if lines == nil {
			return ast.WalkContinue, nil
		}
		for index := 0; index < lines.Len(); index++ {
			segment := lines.At(index)
			range_ := parser.Range{Start: segment.Start, End: paragraphContentEnd(source, segment.Stop)}
			if !range_.Valid(len(source)) {
				return ast.WalkStop, fmt.Errorf("goldmark blockquote semantic range [%d,%d) is outside source length %d", range_.Start, range_.End, len(source))
			}
			if range_.Start != range_.End {
				ranges = append(ranges, range_)
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	return ranges, nil
}

func observeHTMLBlock(source []byte, block *ast.HTMLBlock) (parser.Node, bool, error) {
	range_, ok := htmlBlockSourceRange(source, block)
	if !ok {
		return parser.Node{}, false, nil
	}
	return parser.Node{Kind: parser.KindHTMLBlock, Range: range_}, true, nil
}
