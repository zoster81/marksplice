package splice

// SourceLen returns the immutable source snapshot length without copying source bytes.
func (d *Document) SourceLen() int {
	if d == nil {
		return 0
	}
	return len(d.source)
}

// NodeSelectionAt returns lightweight public-promotion facts plus the operation-oriented
// source span used by structural queries, without cloning the full internal node.
func (d *Document) NodeSelectionAt(index int) (NodeSummary, Range, bool) {
	if d == nil || index < 0 || index >= len(d.nodes) {
		return NodeSummary{}, Range{}, false
	}
	node := d.nodes[index]
	selectionRange, ok := nodeSelectionRange(node)
	if !ok || !selectionRange.Valid(len(d.source)) {
		return NodeSummary{}, Range{}, false
	}
	return summarizeNode(node), selectionRange, true
}

func nodeSelectionRange(node Node) (Range, bool) {
	switch node.Kind {
	case KindParagraph:
		return node.Range, true
	case KindHeading, KindListItem, KindTask, KindTableCell, KindFencedCode, KindStrikethrough,
		KindCodeSpan, KindEmphasis, KindStrong, KindInlineLink, KindImage, KindReferenceDefinition,
		KindAutoLink, KindYAMLFrontMatterField, KindTOMLFrontMatterField, KindHTMLComment, KindHTMLAnchor:
		return node.ContentRange, true
	case KindTableRow:
		return node.TableRowSource.LineRange, true
	case KindTable:
		return node.TableSource.Range, true
	case KindThematicBreak:
		return node.ThematicBreakSource.LineRange, true
	case KindBlockquote:
		return node.BlockquoteSource.LineRange, true
	case KindFootnoteDefinition:
		return node.FootnoteSource.Range, true
	case KindMathExpression:
		return node.MathSource.Range, true
	default:
		return Range{}, false
	}
}
