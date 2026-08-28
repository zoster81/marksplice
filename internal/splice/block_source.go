package splice

import "github.com/zoster81/marksplice/internal/source"

func remapListItemSource(input []byte, node Node) (source.ListItemMapping, bool) {
	if node.Kind != KindListItem || !node.Editable {
		return source.ListItemMapping{}, false
	}
	mapping, err := source.MapSingleLineListItem(input, node.ContentRange, node.ListOrdered, node.ListMarker)
	if err != nil || mapping.Range != node.Range || mapping.LineRange != node.ListItemLineRange {
		return source.ListItemMapping{}, false
	}
	return mapping, true
}

func remapTableCellSource(input []byte, node Node) (source.TableCellMapping, bool) {
	if node.Kind != KindTableCell || !node.Editable {
		return source.TableCellMapping{}, false
	}
	mapping, err := source.MapTableCell(input, node.ContentRange, node.TableColumn)
	if err != nil || mapping.Range != node.TableCellRange || mapping.Column != node.TableColumn {
		return source.TableCellMapping{}, false
	}
	return mapping, true
}

func remapTableRowSource(input []byte, node Node) (source.TableRowMapping, bool) {
	if node.Kind != KindTableRow || !node.Editable {
		return source.TableRowMapping{}, false
	}
	mapping, err := source.MapTableRow(input, node.TableRowAnchor)
	if err != nil || node.TableRowAnchor != node.TableRowSourceAnchor || mapping.Anchor != node.TableRowSourceAnchor || mapping.Range != node.ContentRange || mapping.LineRange != node.Range ||
		len(mapping.Cells) != node.TableColumnCount {
		return source.TableRowMapping{}, false
	}
	return mapping, true
}

func remapTableSource(input []byte, node Node) (source.TableMapping, bool) {
	if node.Kind != KindTable || !node.Editable {
		return source.TableMapping{}, false
	}
	mapping, err := source.MapTable(input, node.TableAnchor, node.TableBodyRowCount, node.TableLastBodyRowAnchor)
	if err != nil || mapping.Range != node.Range || len(mapping.Header.Cells) != node.TableColumnCount ||
		len(mapping.Delimiter.Cells) != node.TableColumnCount || len(mapping.DelimiterAlignments) != node.TableColumnCount {
		return source.TableMapping{}, false
	}
	return mapping, true
}

func remapThematicBreakSource(input []byte, node Node) (source.ThematicBreakMapping, bool) {
	if node.Kind != KindThematicBreak || !node.Editable || !node.TopLevel {
		return source.ThematicBreakMapping{}, false
	}
	mapping, err := source.MapTopLevelThematicBreak(input, node.Range)
	if err != nil || mapping.Range != node.Range {
		return source.ThematicBreakMapping{}, false
	}
	return mapping, true
}

func remapMathSource(input []byte, node Node) (source.MathExpressionMapping, bool) {
	if node.Kind != KindMathExpression || !node.Editable || node.MathStyle == MathExpressionUnknown {
		return source.MathExpressionMapping{}, false
	}
	syntax := node.Range
	if node.MathStyle == MathExpressionBlockDollar {
		syntax.End = trimOwnedLineEnding(input, syntax)
	}
	mapping, err := source.MapMathExpression(input, node.MathStyle, syntax, node.ContentRange)
	if err != nil || mapping.Range != node.Range || mapping.PayloadRange != node.ContentRange {
		return source.MathExpressionMapping{}, false
	}
	return mapping, true
}

func trimOwnedLineEnding(input []byte, range_ Range) int {
	end := range_.End
	if !range_.Valid(len(input)) || end <= range_.Start {
		return end
	}
	if input[end-1] == '\n' {
		end--
		if end > range_.Start && input[end-1] == '\r' {
			end--
		}
		return end
	}
	if input[end-1] == '\r' {
		return end - 1
	}
	return end
}

func (d *Document) blockquoteSource(node Node) (source.BlockquoteMapping, bool) {
	if d == nil || node.Kind != KindBlockquote || !node.Editable || !node.TopLevel {
		return source.BlockquoteMapping{}, false
	}
	index, ok := sourceDetailIndex(node.SourceDetailIndex, len(d.blockquoteSources))
	if !ok {
		return source.BlockquoteMapping{}, false
	}
	mapping := d.blockquoteSources[index]
	if !validBlockquoteSourceMapping(d.source, node, mapping) {
		return source.BlockquoteMapping{}, false
	}
	return mapping, true
}

func validBlockquoteSourceMapping(input []byte, node Node, mapping source.BlockquoteMapping) bool {
	if mapping.Range != node.Range || mapping.ContentRange != node.ContentRange || !mapping.LineRange.Valid(len(input)) || mapping.LineRange.Start >= mapping.LineRange.End {
		return false
	}
	if !mapping.MarkerRange.Valid(len(input)) || len(mapping.ContentRanges) == 0 {
		return false
	}
	return sourceRangesWithin(mapping.ContentRanges, mapping.LineRange, len(input))
}

// BlockquoteSource returns a caller-owned source mapping for one promoted top-level blockquote.
func (d *Document) BlockquoteSource(id NodeID) (source.BlockquoteMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.BlockquoteMapping{}, false
	}
	mapping, ok := d.blockquoteSource(node)
	if !ok {
		return source.BlockquoteMapping{}, false
	}
	mapping.ContentRanges = append([]source.Range(nil), mapping.ContentRanges...)
	return mapping, true
}

// ListItemSource remaps the exact source capability for one editable list item.
func (d *Document) ListItemSource(id NodeID) (source.ListItemMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.ListItemMapping{}, false
	}
	return remapListItemSource(d.source, node)
}

// TableCellSource remaps the exact source capability for one editable GFM table cell.
func (d *Document) TableCellSource(id NodeID) (source.TableCellMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.TableCellMapping{}, false
	}
	return remapTableCellSource(d.source, node)
}

// TableRowSource remaps the exact source capability for one editable GFM table row.
func (d *Document) TableRowSource(id NodeID) (source.TableRowMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.TableRowMapping{}, false
	}
	return remapTableRowSource(d.source, node)
}

// TableSource remaps the exact source capability for one editable GFM table.
func (d *Document) TableSource(id NodeID) (source.TableMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.TableMapping{}, false
	}
	return remapTableSource(d.source, node)
}

// ThematicBreakSource remaps the exact source capability for one editable thematic break.
func (d *Document) ThematicBreakSource(id NodeID) (source.ThematicBreakMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.ThematicBreakMapping{}, false
	}
	return remapThematicBreakSource(d.source, node)
}

// MathSource remaps the exact source capability for one editable math expression.
func (d *Document) MathSource(id NodeID) (source.MathExpressionMapping, bool) {
	node, ok := d.nodeByID(id)
	if !ok {
		return source.MathExpressionMapping{}, false
	}
	return remapMathSource(d.source, node)
}
