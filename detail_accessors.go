package marksplice

import "github.com/zoster81/marksplice/internal/splice"

func (d *Document) Paragraph(id NodeID) (Paragraph, bool) {
	node, err := d.promotedNode(id, splice.KindParagraph, true)
	if err != nil {
		return Paragraph{}, false
	}
	return Paragraph{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.Range.Start, End: node.Range.End},
	}, true
}

// Heading returns typed detail for one promoted top-level heading.
func (d *Document) Heading(id NodeID) (Heading, bool) {
	node, err := d.promotedNode(id, splice.KindHeading, true)
	if err != nil {
		return Heading{}, false
	}
	style, ok := publicHeadingStyle(node.HeadingStyle)
	if !ok {
		return Heading{}, false
	}
	return Heading{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		level:       node.Level,
		style:       style,
	}, true
}

// ListItem returns typed detail for one promoted single-line list item.
func (d *Document) ListItem(id NodeID) (ListItem, bool) {
	node, err := d.promotedNode(id, splice.KindListItem, false)
	if err != nil {
		return ListItem{}, false
	}
	internalChildIDs, ok := d.document.ListItemChildIDs(node.ID)
	if !ok {
		return ListItem{}, false
	}
	childIDs := publicNodeIDs(internalChildIDs)
	item := ListItem{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		ordered:     node.ListOrdered,
		marker:      node.ListMarker,
		parentID:    publicNodeID(node.ListParentID),
		childIDs:    childIDs,
		hasChildren: node.ListHasChildren,
	}
	if node.ListSubtreeComplete {
		item.subtreeRange = Range{Start: node.ListItemSource.LineRange.Start, End: node.ListSubtreeEnd}
		item.hasSubtreeRange = true
	}
	return item, true
}

// Task returns typed detail for one promoted GFM task marker.
func (d *Document) Task(id NodeID) (Task, bool) {
	node, err := d.promotedNode(id, splice.KindTask, false)
	if err != nil {
		return Task{}, false
	}
	return Task{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		checked:     node.Checked,
	}, true
}

// TableCell returns typed detail for one promoted non-empty GFM table cell.
func (d *Document) TableCell(id NodeID) (TableCell, bool) {
	node, err := d.promotedNode(id, splice.KindTableCell, false)
	if err != nil {
		return TableCell{}, false
	}
	return TableCell{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		rowID:       publicNodeID(node.TableRowID),
		header:      node.TableHeader,
		column:      node.TableColumn,
	}, true
}

// TableRowCellIDs returns the promoted non-empty cells owned by one promoted body row in source order.
// Empty cells are omitted because M5 does not assign them public cell identities.
func (d *Document) TableRowCellIDs(rowID NodeID) ([]NodeID, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	internalIDs, ok := d.document.TableRowCellIDs(internalNodeID(rowID))
	if !ok {
		return nil, false
	}
	return publicNodeIDs(internalIDs), true
}

// TableRowHeaderCellIDs returns the promoted non-empty header cells for the table that owns one promoted body row.
// Empty header cells are omitted because M5 does not assign them public cell identities.
func (d *Document) TableRowHeaderCellIDs(rowID NodeID) ([]NodeID, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	internalIDs, ok := d.document.TableRowHeaderCellIDs(internalNodeID(rowID))
	if !ok {
		return nil, false
	}
	return publicNodeIDs(internalIDs), true
}

// TableRow returns typed detail for one promoted GFM table body row.
func (d *Document) TableRow(id NodeID) (TableRow, bool) {
	node, err := d.promotedNode(id, splice.KindTableRow, false)
	if err != nil {
		return TableRow{}, false
	}
	previousID, nextID, ok := d.document.TableRowNeighborIDs(node.ID)
	if !ok {
		return TableRow{}, false
	}
	return TableRow{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.TableRowSource.LineRange.Start, End: node.TableRowSource.LineRange.End},
		columnCount: node.TableColumnCount,
		previousID:  publicNodeID(previousID),
		nextID:      publicNodeID(nextID),
	}, true
}

// FencedCode returns typed detail for one promoted supported fenced code block.
func (d *Document) FencedCode(id NodeID) (FencedCode, bool) {
	node, err := d.promotedNode(id, splice.KindFencedCode, false)
	if err != nil {
		return FencedCode{}, false
	}
	return FencedCode{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
	}, true
}

// Strikethrough returns typed detail for one promoted simple GFM strikethrough.
func (d *Document) Strikethrough(id NodeID) (Strikethrough, bool) {
	node, err := d.promotedNode(id, splice.KindStrikethrough, false)
	if err != nil {
		return Strikethrough{}, false
	}
	return Strikethrough{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// CodeSpan returns typed detail for one promoted simple single-line code span.
func (d *Document) CodeSpan(id NodeID) (CodeSpan, bool) {
	node, err := d.promotedNode(id, splice.KindCodeSpan, false)
	if err != nil {
		return CodeSpan{}, false
	}
	return CodeSpan{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// Emphasis returns typed detail for one promoted simple emphasis span.
func (d *Document) Emphasis(id NodeID) (Emphasis, bool) {
	node, err := d.promotedNode(id, splice.KindEmphasis, false)
	if err != nil {
		return Emphasis{}, false
	}
	return Emphasis{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// Strong returns typed detail for one promoted simple strong-emphasis span.
func (d *Document) Strong(id NodeID) (Strong, bool) {
	node, err := d.promotedNode(id, splice.KindStrong, false)
	if err != nil {
		return Strong{}, false
	}
	return Strong{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// InlineLink returns typed detail for one promoted simple inline link.
func (d *Document) InlineLink(id NodeID) (InlineLink, bool) {
	node, err := d.promotedNode(id, splice.KindInlineLink, false)
	if err != nil {
		return InlineLink{}, false
	}
	return InlineLink{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// Image returns typed detail for one promoted simple inline image.
func (d *Document) Image(id NodeID) (Image, bool) {
	node, err := d.promotedNode(id, splice.KindImage, false)
	if err != nil {
		return Image{}, false
	}
	return Image{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// ReferenceDefinition returns typed detail for one promoted single-line reference definition.
func (d *Document) ReferenceDefinition(id NodeID) (ReferenceDefinition, bool) {
	node, err := d.promotedNode(id, splice.KindReferenceDefinition, false)
	if err != nil {
		return ReferenceDefinition{}, false
	}
	return ReferenceDefinition{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// AutoLink returns typed detail for one promoted single-line GFM autolink.
func (d *Document) AutoLink(id NodeID) (AutoLink, bool) {
	node, err := d.promotedNode(id, splice.KindAutoLink, false)
	if err != nil {
		return AutoLink{}, false
	}
	return AutoLink{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// FrontMatterField returns typed detail for one promoted simple leading YAML/TOML scalar field.
func (d *Document) FrontMatterField(id NodeID) (FrontMatterField, bool) {
	node, err := d.promotedNodeKinds(id, false, splice.KindYAMLFrontMatterField, splice.KindTOMLFrontMatterField)
	if err != nil {
		return FrontMatterField{}, false
	}
	format, ok := publicFrontMatterFormat(node.FrontMatterFormat)
	if !ok {
		return FrontMatterField{}, false
	}
	return FrontMatterField{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		key:         node.Key,
		format:      format,
	}, true
}

// HTMLComment returns typed detail for one promoted single-line HTML comment.
func (d *Document) HTMLComment(id NodeID) (HTMLComment, bool) {
	node, err := d.promotedNode(id, splice.KindHTMLComment, false)
	if err != nil {
		return HTMLComment{}, false
	}
	return HTMLComment{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// HTMLAnchor returns typed detail for one promoted simple quoted id/name attribute on an <a> tag.
func (d *Document) HTMLAnchor(id NodeID) (HTMLAnchor, bool) {
	node, err := d.promotedNode(id, splice.KindHTMLAnchor, false)
	if err != nil {
		return HTMLAnchor{}, false
	}
	attribute, ok := publicHTMLAnchorAttribute(node.HTMLAttribute)
	if !ok {
		return HTMLAnchor{}, false
	}
	return HTMLAnchor{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		attribute:   attribute,
	}, true
}

func publicHeadingStyle(style splice.HeadingStyle) (HeadingStyle, bool) {
	switch style {
	case splice.HeadingStyleATX:
		return HeadingStyleATX, true
	case splice.HeadingStyleSetext:
		return HeadingStyleSetext, true
	default:
		return HeadingStyleUnknown, false
	}
}

func publicFrontMatterFormat(format splice.FrontMatterFormat) (FrontMatterFormat, bool) {
	switch format {
	case splice.FrontMatterFormatYAML:
		return FrontMatterFormatYAML, true
	case splice.FrontMatterFormatTOML:
		return FrontMatterFormatTOML, true
	default:
		return FrontMatterFormatUnknown, false
	}
}

func publicHTMLAnchorAttribute(attribute string) (HTMLAnchorAttribute, bool) {
	switch attribute {
	case "id":
		return HTMLAnchorAttributeID, true
	case "name":
		return HTMLAnchorAttributeName, true
	default:
		return HTMLAnchorAttributeUnknown, false
	}
}
