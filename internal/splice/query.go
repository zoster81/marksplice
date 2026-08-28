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
	selectionRange, ok := d.nodeSelectionRange(node)
	if ok && node.Kind == KindThematicBreak {
		mapping, mapped := remapThematicBreakSource(d.source, node)
		if !mapped {
			return NodeSummary{}, Range{}, false
		}
		selectionRange = mapping.LineRange
	}
	if !ok || !selectionRange.Valid(len(d.source)) {
		return NodeSummary{}, Range{}, false
	}
	return summarizeNode(node), selectionRange, true
}

func (d *Document) nodeSelectionRange(node Node) (Range, bool) {
	switch node.Kind {
	case KindParagraph:
		return node.Range, true
	case KindHeading, KindListItem, KindTask, KindTableCell, KindFencedCode, KindStrikethrough,
		KindCodeSpan, KindEmphasis, KindStrong, KindInlineLink, KindImage, KindReferenceDefinition,
		KindAutoLink, KindYAMLFrontMatterField, KindTOMLFrontMatterField, KindHTMLComment, KindHTMLAnchor:
		return node.ContentRange, true
	case KindTableRow:
		return node.Range, true
	case KindTable, KindThematicBreak:
		return node.Range, true
	case KindBlockquote:
		mapping, ok := d.blockquoteSource(node)
		return mapping.LineRange, ok
	case KindFootnoteDefinition:
		mapping, ok := d.footnoteSource(node)
		return mapping.Range, ok
	case KindMathExpression:
		return node.Range, true
	default:
		return Range{}, false
	}
}
