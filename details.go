package marksplice

import "github.com/zoster81/marksplice/internal/splice"

// HeadingStyle identifies the source syntax of a promoted heading.
type HeadingStyle uint8

const (
	HeadingStyleUnknown HeadingStyle = iota
	HeadingStyleATX
	HeadingStyleSetext
)

// Paragraph is immutable typed detail for one promoted top-level paragraph.
type Paragraph struct {
	id          NodeID
	sourceRange Range
}

// ID returns the paragraph's snapshot-scoped node identity.
func (p Paragraph) ID() NodeID {
	return p.id
}

// Range returns the exact paragraph byte span replaced by PrepareReplaceParagraph.
// A line ending immediately following the paragraph is outside this range.
func (p Paragraph) Range() Range {
	return p.sourceRange
}

// Heading is immutable typed detail for one promoted top-level heading.
type Heading struct {
	id          NodeID
	sourceRange Range
	level       int
	style       HeadingStyle
}

// ID returns the heading's snapshot-scoped node identity.
func (h Heading) ID() NodeID {
	return h.id
}

// Range returns the exact heading-content byte span replaced by PrepareRenameHeading.
// ATX markers, optional closing markers, Setext underlines, and line endings are outside this range.
func (h Heading) Range() Range {
	return h.sourceRange
}

// Level returns the GFM heading level from 1 through 6.
func (h Heading) Level() int {
	return h.level
}

// Style returns whether the heading uses ATX or Setext source syntax.
func (h Heading) Style() HeadingStyle {
	return h.style
}

// ListItem is immutable typed detail for one promoted single-line list item.
type ListItem struct {
	id          NodeID
	sourceRange Range
	ordered     bool
	marker      byte
}

// ID returns the list item's snapshot-scoped node identity.
func (i ListItem) ID() NodeID {
	return i.id
}

// Range returns the exact list-item content span replaced by PrepareReplaceListItem.
// Indentation, list numbering, marker/delimiter bytes, post-marker spacing, and line endings are outside this range.
func (i ListItem) Range() Range {
	return i.sourceRange
}

// Ordered reports whether the item belongs to an ordered list.
func (i ListItem) Ordered() bool {
	return i.ordered
}

// Marker returns the source marker/delimiter byte.
// Unordered items use '-', '*', or '+'; ordered items use '.' or ')'.
func (i ListItem) Marker() byte {
	return i.marker
}

// Task is immutable typed detail for one promoted GFM task marker.
type Task struct {
	id          NodeID
	sourceRange Range
	checked     bool
}

// ID returns the task's snapshot-scoped node identity.
func (t Task) ID() NodeID {
	return t.id
}

// Range returns the exact one-byte task state span changed by PrepareSetTaskChecked.
// The surrounding brackets and list-item source are outside this range.
func (t Task) Range() Range {
	return t.sourceRange
}

// Checked reports the semantic task state.
func (t Task) Checked() bool {
	return t.checked
}

// TableCell is immutable typed detail for one promoted non-empty GFM table cell.
type TableCell struct {
	id          NodeID
	sourceRange Range
	header      bool
	column      int
}

// ID returns the table cell's snapshot-scoped node identity.
func (c TableCell) ID() NodeID {
	return c.id
}

// Range returns the exact table-cell content span replaced by PrepareReplaceTableCell.
// Pipes, cell padding, alignment syntax, neighboring cells, and line endings are outside this range.
func (c TableCell) Range() Range {
	return c.sourceRange
}

// Header reports whether the cell belongs to the table header row.
func (c TableCell) Header() bool {
	return c.header
}

// Column returns the zero-based column index within the mapped table row.
func (c TableCell) Column() int {
	return c.column
}

// FencedCode is immutable typed detail for one promoted single-line fenced code block.
type FencedCode struct {
	id          NodeID
	sourceRange Range
}

// ID returns the fenced code block's snapshot-scoped node identity.
func (f FencedCode) ID() NodeID {
	return f.id
}

// Range returns the exact fenced-code content span replaced by PrepareReplaceFencedCode.
// Fence delimiters, indentation, info-string source, and line endings are outside this range.
func (f FencedCode) Range() Range {
	return f.sourceRange
}

// Strikethrough is immutable typed detail for one promoted simple GFM strikethrough.
type Strikethrough struct {
	id          NodeID
	sourceRange Range
}

// ID returns the strikethrough's snapshot-scoped node identity.
func (s Strikethrough) ID() NodeID { return s.id }

// Range returns the exact strikethrough content span replaced by PrepareReplaceStrikethrough.
func (s Strikethrough) Range() Range { return s.sourceRange }

// CodeSpan is immutable typed detail for one promoted simple single-line code span.
type CodeSpan struct {
	id          NodeID
	sourceRange Range
}

// ID returns the code span's snapshot-scoped node identity.
func (c CodeSpan) ID() NodeID { return c.id }

// Range returns the exact code-span content span replaced by PrepareReplaceCodeSpan.
func (c CodeSpan) Range() Range { return c.sourceRange }

// Emphasis is immutable typed detail for one promoted simple emphasis span.
type Emphasis struct {
	id          NodeID
	sourceRange Range
}

// ID returns the emphasis span's snapshot-scoped node identity.
func (e Emphasis) ID() NodeID { return e.id }

// Range returns the exact emphasis content span replaced by PrepareReplaceEmphasis.
func (e Emphasis) Range() Range { return e.sourceRange }

// Strong is immutable typed detail for one promoted simple strong-emphasis span.
type Strong struct {
	id          NodeID
	sourceRange Range
}

// ID returns the strong span's snapshot-scoped node identity.
func (s Strong) ID() NodeID { return s.id }

// Range returns the exact strong content span replaced by PrepareReplaceStrong.
func (s Strong) Range() Range { return s.sourceRange }

// Paragraph returns typed detail for one promoted top-level paragraph.
func (d *Document) Paragraph(id NodeID) (Paragraph, bool) {
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindParagraph || !node.Editable || !node.TopLevel {
		return Paragraph{}, false
	}
	return Paragraph{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.Range.Start, End: node.Range.End},
	}, true
}

// Heading returns typed detail for one promoted top-level heading.
func (d *Document) Heading(id NodeID) (Heading, bool) {
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindHeading || !node.Editable || !node.TopLevel {
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
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindListItem || !node.Editable {
		return ListItem{}, false
	}
	return ListItem{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		ordered:     node.ListOrdered,
		marker:      node.ListMarker,
	}, true
}

// Task returns typed detail for one promoted GFM task marker.
func (d *Document) Task(id NodeID) (Task, bool) {
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindTask || !node.Editable {
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
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindTableCell || !node.Editable {
		return TableCell{}, false
	}
	return TableCell{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
		header:      node.TableHeader,
		column:      node.TableColumn,
	}, true
}

// FencedCode returns typed detail for one promoted single-line fenced code block.
func (d *Document) FencedCode(id NodeID) (FencedCode, bool) {
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindFencedCode || !node.Editable {
		return FencedCode{}, false
	}
	return FencedCode{
		id:          publicNodeID(node.ID),
		sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End},
	}, true
}

// Strikethrough returns typed detail for one promoted simple GFM strikethrough.
func (d *Document) Strikethrough(id NodeID) (Strikethrough, bool) {
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindStrikethrough || !node.Editable {
		return Strikethrough{}, false
	}
	return Strikethrough{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// CodeSpan returns typed detail for one promoted simple single-line code span.
func (d *Document) CodeSpan(id NodeID) (CodeSpan, bool) {
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindCodeSpan || !node.Editable {
		return CodeSpan{}, false
	}
	return CodeSpan{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// Emphasis returns typed detail for one promoted simple emphasis span.
func (d *Document) Emphasis(id NodeID) (Emphasis, bool) {
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindEmphasis || !node.Editable {
		return Emphasis{}, false
	}
	return Emphasis{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
}

// Strong returns typed detail for one promoted simple strong-emphasis span.
func (d *Document) Strong(id NodeID) (Strong, bool) {
	node, ok := d.internalNode(id)
	if !ok || node.Kind != splice.KindStrong || !node.Editable {
		return Strong{}, false
	}
	return Strong{id: publicNodeID(node.ID), sourceRange: Range{Start: node.ContentRange.Start, End: node.ContentRange.End}}, true
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
