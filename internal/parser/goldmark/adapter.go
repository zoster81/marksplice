package goldmark

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extensionast "github.com/yuin/goldmark/extension/ast"
	goldmarkparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/zoster81/marksplice/internal/parser"
)

// Adapter isolates Goldmark behind Marksplice-owned parser types.
type Adapter struct {
	markdown  goldmark.Markdown
	footnotes goldmark.Markdown
}

// New returns a Goldmark-backed semantic parser configured for Marksplice's
// normative GFM profile plus an isolated footnote semantic pass.
func New() *Adapter {
	return &Adapter{
		markdown:  newMarkdown(),
		footnotes: newFootnoteMarkdown(),
	}
}

func newMarkdown() goldmark.Markdown {
	return newMarkdownWithExtensions([]goldmark.Extender{extension.GFM})
}

func newMarkdownWithExtensions(extenders []goldmark.Extender, options ...goldmark.Option) goldmark.Markdown {
	parserOptions := []goldmarkparser.Option{
		goldmarkparser.WithBlockParsers(
			util.Prioritized(newFencedCodePositionParser(), 699),
		),
		goldmarkparser.WithInlineParsers(
			util.Prioritized(&gfmCompatibilityInlineParser{}, 399),
		),
	}
	if hasGFMExtendedAutolinks(extenders) {
		parserOptions = append(parserOptions,
			goldmarkparser.WithInlineParsers(
				util.Prioritized(&gfmProtocolAutolinkParser{}, 998),
			),
			extension.WithLinkifyAllowedProtocols([]string{"http:", "https:"}),
		)
	}
	options = append(options, goldmark.WithParserOptions(parserOptions...))
	if len(extenders) != 0 {
		options = append(options, goldmark.WithExtensions(extenders...))
	}
	return goldmark.New(options...)
}

func hasGFMExtendedAutolinks(extenders []goldmark.Extender) bool {
	for _, extender := range extenders {
		if extender == extension.GFM || extender == extension.Linkify {
			return true
		}
	}
	return false
}

// Parse returns parser-independent semantic observations tied to source byte ranges.
func (a *Adapter) Parse(source []byte) ([]parser.Node, error) {
	observed, err := a.ParseDocument(source)
	return observed.Nodes, err
}

// ParseWithLinkUsages returns ordinary structural observations, every
// parser-resolved link/image/autolink relationship, and conservative unresolved
// explicit full/collapsed reference usages in semantic source order.
func (a *Adapter) ParseWithLinkUsages(source []byte) ([]parser.Node, []parser.LinkUsage, []parser.UnresolvedReferenceUsage, error) {
	observed, err := a.ParseDocument(source)
	return observed.Nodes, observed.LinkUsages, observed.UnresolvedReferenceUsages, err
}

func (a *Adapter) parseGFM(source []byte) ([]parser.Node, []parser.LinkUsage, []parser.UnresolvedReferenceUsage, []parser.MathExpressionObservation, error) {
	parseSource := normalizeIsolatedCR(source)
	context := goldmarkparser.NewContext()
	root := a.markdown.Parser().Parse(text.NewReader(parseSource), goldmarkparser.WithContext(context))
	nodes := make([]parser.Node, 0)
	usages := make([]parser.LinkUsage, 0)
	currentTableRowAnchor := -1
	nextTableColumn := 0

	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if usage, ok := linkUsage(source, node); ok {
			usages = append(usages, usage)
		}
		var observation parser.Node
		var ok bool
		var err error
		if cell, isTableCell := node.(*extensionast.TableCell); isTableCell {
			parent := cell.Parent()
			if parent == nil || parent.Pos() < 0 {
				return ast.WalkContinue, nil
			}
			rowAnchor := parent.Pos()
			if rowAnchor != currentTableRowAnchor {
				currentTableRowAnchor = rowAnchor
				nextTableColumn = 0
			}
			column := nextTableColumn
			nextTableColumn++
			observation, ok = observeTableCell(source, cell, column)
		} else {
			observation, ok, err = observeNode(source, node)
		}
		if err != nil {
			return ast.WalkStop, err
		}
		if ok {
			nodes = append(nodes, observation)
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("walk goldmark AST: %w", err)
	}
	unresolved := unresolvedReferenceUsages(source, root, context)
	mathExpressions := mathExpressionObservations(source, root)
	nodes = removeMathGFMConflicts(nodes, mathExpressions)
	return nodes, usages, unresolved, mathExpressions, nil
}

func linkUsage(source []byte, node ast.Node) (parser.LinkUsage, bool) {
	var kind parser.Kind
	var destination, title []byte
	var reference *ast.ReferenceLink
	usage := parser.LinkUsage{Anchor: node.Pos(), Form: parser.LinkUsageDirect}
	switch typed := node.(type) {
	case *ast.Link:
		kind = parser.KindInlineLink
		destination, title, reference = typed.Destination, typed.Title, typed.Reference
	case *ast.Image:
		kind = parser.KindImage
		destination, title, reference = typed.Destination, typed.Title, typed.Reference
	case *ast.AutoLink:
		usage.Kind = parser.KindAutoLink
		usage.Destination = string(typed.URL(source))
		usage.AutoLinkEmail = typed.AutoLinkType == ast.AutoLinkEmail
		if anchor, _, _, ok := autoLinkSourceRange(source, typed); ok {
			usage.Anchor = anchor
		}
		return usage, usage.Anchor >= 0
	default:
		return parser.LinkUsage{}, false
	}
	if usage.Anchor < 0 {
		return parser.LinkUsage{}, false
	}
	usage.Kind = kind
	usage.Destination = string(destination)
	usage.Title = string(title)
	usage.HasTitle = title != nil
	if reference == nil {
		return usage, true
	}
	form, ok := referenceUsageForm(reference.Type)
	if !ok {
		return parser.LinkUsage{}, false
	}
	usage.Form = form
	usage.Reference = string(reference.Value)
	return usage, true
}

func referenceUsageForm(form ast.ReferenceLinkType) (parser.LinkUsageForm, bool) {
	switch form {
	case ast.ReferenceLinkFull:
		return parser.LinkUsageFull, true
	case ast.ReferenceLinkCollapsed:
		return parser.LinkUsageCollapsed, true
	case ast.ReferenceLinkShortcut:
		return parser.LinkUsageShortcut, true
	default:
		return parser.LinkUsageUnknown, false
	}
}
func fencedCodeContentEnd(source []byte, start int) int {
	end := start
	for end < len(source) && source[end] != '\r' && source[end] != '\n' {
		end++
	}
	return end
}

func singleLineRawHTMLRange(source []byte, raw *ast.RawHTML) (parser.Range, bool) {
	if raw.Segments == nil || raw.Segments.Len() != 1 {
		return parser.Range{}, false
	}
	segment := raw.Segments.At(0)
	range_ := parser.Range{Start: segment.Start, End: segment.Stop}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Range{}, false
	}
	for _, b := range source[range_.Start:range_.End] {
		if b == '\r' || b == '\n' {
			return parser.Range{}, false
		}
	}
	return range_, true
}

func htmlBlockSourceRange(source []byte, block *ast.HTMLBlock) (parser.Range, bool) {
	lines := block.Lines()
	if lines.Len() == 0 {
		return parser.Range{}, false
	}
	start := lines.At(0).Start
	end := lines.At(lines.Len() - 1).Stop
	if block.HasClosure() && block.ClosureLine.Stop > end {
		end = block.ClosureLine.Stop
	}
	range_ := parser.Range{Start: start, End: end}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Range{}, false
	}
	return range_, true
}

func autoLinkSourceRange(source []byte, link *ast.AutoLink) (int, parser.Range, string, bool) {
	pos := link.Pos()
	if pos < 0 || pos >= len(source) {
		return 0, parser.Range{}, "", false
	}
	value := link.Label(source)
	if len(value) == 0 {
		return 0, parser.Range{}, "", false
	}

	if source[pos] == '<' {
		start := pos + 1
		end := start + len(value)
		if end >= len(source) || source[end] != '>' || !bytes.Equal(source[start:end], value) {
			return 0, parser.Range{}, "", false
		}
		return pos, parser.Range{Start: start, End: end}, string(value), true
	}

	start := pos
	end := start + len(value)
	if end <= len(source) && bytes.Equal(source[start:end], value) {
		return start, parser.Range{Start: start, End: end}, string(value), true
	}
	if pos+1 < len(source) && isGFMExtendedAutolinkBoundary(source[pos]) {
		start = pos + 1
		end = start + len(value)
		if end <= len(source) && bytes.Equal(source[start:end], value) {
			return start, parser.Range{Start: start, End: end}, string(value), true
		}
	}
	return 0, parser.Range{}, "", false
}

func simplePlainTextInlineRange(source []byte, node ast.Node) (parser.Range, bool) {
	if node.ChildCount() != 1 {
		return parser.Range{}, false
	}
	child, ok := node.FirstChild().(*ast.Text)
	if !ok || child.SoftLineBreak() || child.HardLineBreak() {
		return parser.Range{}, false
	}
	segment := child.Segment
	range_ := parser.Range{Start: segment.Start, End: segment.Stop}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Range{}, false
	}
	for _, b := range source[range_.Start:range_.End] {
		if b == '\r' || b == '\n' {
			return parser.Range{}, false
		}
	}
	return range_, true
}

func simpleInlineLinkLabelRange(source []byte, link *ast.Link) (parser.Range, bool) {
	if link.Pos() < 0 || link.ChildCount() != 1 {
		return parser.Range{}, false
	}
	child, ok := link.FirstChild().(*ast.Text)
	if !ok || child.SoftLineBreak() || child.HardLineBreak() {
		return parser.Range{}, false
	}
	segment := child.Segment
	range_ := parser.Range{Start: segment.Start, End: segment.Stop}
	if !range_.Valid(len(source)) || range_.Start == range_.End || link.Pos() >= range_.Start {
		return parser.Range{}, false
	}
	for _, b := range source[range_.Start:range_.End] {
		if b == '\r' || b == '\n' {
			return parser.Range{}, false
		}
	}
	return range_, true
}

func simpleStrikethroughContentRange(source []byte, strike *extensionast.Strikethrough) (parser.Range, bool) {
	if strike.ChildCount() != 1 {
		return parser.Range{}, false
	}
	child, ok := strike.FirstChild().(*ast.Text)
	if !ok || child.SoftLineBreak() || child.HardLineBreak() {
		return parser.Range{}, false
	}

	segment := child.Segment
	range_ := parser.Range{Start: segment.Start, End: segment.Stop}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Range{}, false
	}
	for _, b := range source[range_.Start:range_.End] {
		if b == '\r' || b == '\n' {
			return parser.Range{}, false
		}
	}
	return range_, true
}

func simpleListItemContentRange(source []byte, item *ast.ListItem) (parser.Range, int, bool) {
	first := item.FirstChild()
	if first == nil {
		return parser.Range{}, 0, false
	}

	var lines *text.Segments
	switch child := first.(type) {
	case *ast.TextBlock:
		lines = child.Lines()
	case *ast.Paragraph:
		lines = child.Lines()
	default:
		return parser.Range{}, 0, false
	}
	if lines.Len() != 1 {
		return parser.Range{}, 0, false
	}

	directChildCount := 0
	for child := first.NextSibling(); child != nil; child = child.NextSibling() {
		list, ok := child.(*ast.List)
		if !ok {
			return parser.Range{}, 0, false
		}
		for listChild := list.FirstChild(); listChild != nil; listChild = listChild.NextSibling() {
			if _, ok := listChild.(*ast.ListItem); !ok {
				return parser.Range{}, 0, false
			}
			directChildCount++
		}
	}

	line := lines.At(0)
	range_ := parser.Range{Start: line.Start, End: paragraphContentEnd(source, line.Stop)}
	if !range_.Valid(len(source)) || range_.Start == range_.End {
		return parser.Range{}, 0, false
	}
	return range_, directChildCount, true
}

func normalizeIsolatedCR(source []byte) []byte {
	var normalized []byte
	for i, b := range source {
		if b != '\r' || (i+1 < len(source) && source[i+1] == '\n') {
			continue
		}
		if normalized == nil {
			normalized = append([]byte(nil), source...)
		}
		normalized[i] = '\n'
	}
	if normalized == nil {
		return source
	}
	return normalized
}

func paragraphContentEnd(source []byte, semanticEnd int) int {
	end := semanticEnd
	for end < len(source) && source[end] != '\n' && source[end] != '\r' {
		end++
	}
	return end
}
