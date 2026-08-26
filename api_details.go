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
		tableID:     publicNodeID(node.TableID),
		rowID:       publicNodeID(node.TableRowID),
		header:      node.TableHeader,
		column:      node.TableColumn,
	}, true
}

// TableRowCellIDs returns the promoted non-empty cells owned by one promoted body row in source order.
// Empty cells are omitted because they do not receive public cell identities.
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
// Empty header cells are omitted because they do not receive public cell identities.
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

// Table returns typed detail for one promoted GFM table.
func (d *Document) Table(id NodeID) (Table, bool) {
	node, err := d.promotedNode(id, splice.KindTable, false)
	if err != nil || node.TableColumnCount <= 0 || node.TableBodyRowCount < 0 || len(node.TableAlignments) != node.TableColumnCount {
		return Table{}, false
	}
	return Table{
		id:           publicNodeID(node.ID),
		sourceRange:  Range{Start: node.TableSource.Range.Start, End: node.TableSource.Range.End},
		columnCount:  node.TableColumnCount,
		bodyRowCount: node.TableBodyRowCount,
	}, true
}

// TableRowIDs returns the promoted body-row identities owned by one promoted table in source order.
// The returned slice is caller-owned and can be empty even when BodyRowCount is non-zero.
func (d *Document) TableRowIDs(tableID NodeID) ([]NodeID, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	internalIDs, ok := d.document.TableRowIDs(internalNodeID(tableID))
	if !ok {
		return nil, false
	}
	return publicNodeIDs(internalIDs), true
}

// TableHeaderCellIDs returns the promoted non-empty header-cell identities owned by one promoted table in source order.
// Empty or otherwise unpromoted header cells are omitted.
func (d *Document) TableHeaderCellIDs(tableID NodeID) ([]NodeID, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	internalIDs, ok := d.document.TableHeaderCellIDs(internalNodeID(tableID))
	if !ok {
		return nil, false
	}
	return publicNodeIDs(internalIDs), true
}

// TableAlignments returns one semantic alignment per source-proven table column.
// The returned slice is caller-owned.
func (d *Document) TableAlignments(tableID NodeID) ([]TableAlignment, bool) {
	node, err := d.promotedNode(tableID, splice.KindTable, false)
	if err != nil {
		return nil, false
	}
	return publicTableAlignments(node.TableAlignments, node.TableColumnCount)
}

// TableRowAlignments returns the semantic column alignments for the table that owns one promoted body row.
// The returned slice has exactly TableRow.ColumnCount entries and is caller-owned.
func (d *Document) TableRowAlignments(rowID NodeID) ([]TableAlignment, bool) {
	node, err := d.promotedNode(rowID, splice.KindTableRow, false)
	if err != nil {
		return nil, false
	}
	return publicTableAlignments(node.TableAlignments, node.TableColumnCount)
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
		tableID:     publicNodeID(node.TableID),
		columnCount: node.TableColumnCount,
		previousID:  publicNodeID(previousID),
		nextID:      publicNodeID(nextID),
	}, true
}

// ThematicBreak returns typed detail for one promoted top-level thematic break.
func (d *Document) ThematicBreak(id NodeID) (ThematicBreak, bool) {
	node, err := d.promotedNode(id, splice.KindThematicBreak, true)
	if err != nil {
		return ThematicBreak{}, false
	}
	return ThematicBreak{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ThematicBreakSource.LineRange.Start, End: node.ThematicBreakSource.LineRange.End},
	}, true
}

// Blockquote returns typed detail for one promoted complete top-level blockquote container.
func (d *Document) Blockquote(id NodeID) (Blockquote, bool) {
	node, err := d.promotedNode(id, splice.KindBlockquote, true)
	if err != nil {
		return Blockquote{}, false
	}
	return Blockquote{
		id:           publicNodeID(node.ID),
		sourceRange:  Range{Start: node.BlockquoteSource.LineRange.Start, End: node.BlockquoteSource.LineRange.End},
		contentRange: Range{Start: node.BlockquoteSource.ContentRange.Start, End: node.BlockquoteSource.ContentRange.End},
	}, true
}

// BlockquoteContentRanges returns caller-owned inner source segments for every
// physical line owned by one promoted top-level blockquote, in source order.
// Marker-only lines are represented by valid empty ranges. Lazy continuation
// lines have no synthetic marker removal: their complete physical content is returned.
func (d *Document) BlockquoteContentRanges(id NodeID) ([]Range, bool) {
	if d == nil || d.document == nil {
		return nil, false
	}
	internalRanges, ok := d.document.BlockquoteContentRanges(internalNodeID(id))
	if !ok {
		return nil, false
	}
	return publicRanges(internalRanges), true
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
	return InlineLink{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		destination: node.Destination,
		title:       node.Title,
		hasTitle:    node.HasTitle,
	}, true
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
	return ReferenceDefinition{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		label:       node.Label,
		destination: node.Destination,
		title:       node.Title,
		hasTitle:    node.HasTitle,
	}, true
}

// AutoLink returns typed detail for one promoted single-line GFM autolink.
func (d *Document) AutoLink(id NodeID) (AutoLink, bool) {
	node, err := d.promotedNode(id, splice.KindAutoLink, false)
	if err != nil {
		return AutoLink{}, false
	}
	return AutoLink{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		value:       node.Value,
		email:       node.AutoLinkEmail,
	}, true
}

// FrontMatter returns the recognized document-leading YAML/TOML metadata envelope.
// Complex or duplicate metadata can be readable through this envelope even when no
// individual field is safe to promote for source-preserving mutation.
func (d *Document) FrontMatter() (FrontMatter, bool) {
	if d == nil || d.document == nil {
		return FrontMatter{}, false
	}
	envelope, ok := d.document.FrontMatter()
	if !ok {
		return FrontMatter{}, false
	}
	format, ok := publicFrontMatterFormat(envelope.Format)
	if !ok {
		return FrontMatter{}, false
	}
	return FrontMatter{
		format:       format,
		sourceRange:  publicRange(envelope.Range),
		openingRange: publicRange(envelope.OpeningRange),
		closingRange: publicRange(envelope.ClosingRange),
	}, true
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

func publicTableAlignments(values []splice.TableAlignment, columnCount int) ([]TableAlignment, bool) {
	if columnCount <= 0 || len(values) != columnCount {
		return nil, false
	}
	alignments := make([]TableAlignment, len(values))
	for index, alignment := range values {
		public, ok := publicTableAlignment(alignment)
		if !ok {
			return nil, false
		}
		alignments[index] = public
	}
	return alignments, true
}

func publicTableAlignment(alignment splice.TableAlignment) (TableAlignment, bool) {
	switch alignment {
	case splice.TableAlignmentDefault:
		return TableAlignmentDefault, true
	case splice.TableAlignmentLeft:
		return TableAlignmentLeft, true
	case splice.TableAlignmentRight:
		return TableAlignmentRight, true
	case splice.TableAlignmentCenter:
		return TableAlignmentCenter, true
	default:
		return TableAlignmentDefault, false
	}
}

func internalTableAlignment(alignment TableAlignment) (splice.TableAlignment, bool) {
	switch alignment {
	case TableAlignmentDefault:
		return splice.TableAlignmentDefault, true
	case TableAlignmentLeft:
		return splice.TableAlignmentLeft, true
	case TableAlignmentRight:
		return splice.TableAlignmentRight, true
	case TableAlignmentCenter:
		return splice.TableAlignmentCenter, true
	default:
		return splice.TableAlignmentDefault, false
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
