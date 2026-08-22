package source

import "fmt"

// TableCellMapping binds one GFM table cell to its raw cell span.
type TableCellMapping struct {
	Range        Range
	ContentRange Range
	Column       int
}

// TableRowMapping binds one physical GFM table row to all of its lossless cell spans.
type TableRowMapping struct {
	Range     Range
	LineRange Range
	Cells     []TableCellMapping
}

// MapTableRow maps all cells in one physical GFM table row with a single row scan.
func MapTableRow(input []byte, anchor int) (TableRowMapping, error) {
	if anchor < 0 || anchor >= len(input) {
		return TableRowMapping{}, fmt.Errorf("%w: row anchor %d is outside source length %d", ErrUnsupportedTableCellShape, anchor, len(input))
	}
	lineStart := physicalLineStart(input, anchor)
	lineEnd := physicalLineEnd(input, anchor)
	if lineStart == lineEnd {
		return TableRowMapping{}, fmt.Errorf("%w: empty physical row", ErrUnsupportedTableCellShape)
	}

	line := input[lineStart:lineEnd]
	spans := tableCellSpans(line)
	if len(spans) == 0 {
		return TableRowMapping{}, fmt.Errorf("%w: physical row contains no table-cell delimiters", ErrUnsupportedTableCellShape)
	}
	cells := make([]TableCellMapping, len(spans))
	for column, raw := range spans {
		trimmed := trimHorizontalSpaceRange(line, raw)
		cells[column] = TableCellMapping{
			Range:        Range{Start: lineStart + raw.Start, End: lineStart + raw.End},
			ContentRange: Range{Start: lineStart + trimmed.Start, End: lineStart + trimmed.End},
			Column:       column,
		}
	}
	lineRangeEnd := lineEnd
	if next, ok := nextPhysicalLineStart(input, lineEnd); ok {
		lineRangeEnd = next
	}
	return TableRowMapping{
		Range:     Range{Start: lineStart, End: lineEnd},
		LineRange: Range{Start: lineStart, End: lineRangeEnd},
		Cells:     cells,
	}, nil
}

// MapTableCell verifies a non-empty semantic table-cell range against one physical GFM table row.
func MapTableCell(input []byte, content Range, column int) (TableCellMapping, error) {
	if !content.Valid(len(input)) || content.Start == content.End || column < 0 {
		return TableCellMapping{}, fmt.Errorf("%w: invalid content range or column", ErrUnsupportedTableCellShape)
	}
	if containsLineBreak(input[content.Start:content.End]) {
		return TableCellMapping{}, fmt.Errorf("%w: content crosses a physical line", ErrUnsupportedTableCellShape)
	}

	row, err := MapTableRow(input, content.Start)
	if err != nil {
		return TableCellMapping{}, err
	}
	if column >= len(row.Cells) {
		return TableCellMapping{}, fmt.Errorf("%w: column %d is outside %d mapped cells", ErrUnsupportedTableCellShape, column, len(row.Cells))
	}
	mapping := row.Cells[column]
	if mapping.ContentRange != content {
		return TableCellMapping{}, fmt.Errorf("%w: semantic content [%d,%d) does not match mapped content [%d,%d)", ErrUnsupportedTableCellShape, content.Start, content.End, mapping.ContentRange.Start, mapping.ContentRange.End)
	}
	return mapping, nil
}

func tableCellSpans(line []byte) []Range {
	var delimiters []int
	for i, b := range line {
		if b != '|' || i > 0 && line[i-1] == '\\' {
			continue
		}
		delimiters = append(delimiters, i)
	}
	if len(delimiters) == 0 {
		return nil
	}

	start := 0
	firstDelimiter := 0
	if allHorizontalSpace(line[:delimiters[0]]) {
		start = delimiters[0] + 1
		firstDelimiter = 1
	}
	trailingDelimiter := allHorizontalSpace(line[delimiters[len(delimiters)-1]+1:])

	spans := make([]Range, 0, len(delimiters)+1)
	for i := firstDelimiter; i < len(delimiters); i++ {
		delimiter := delimiters[i]
		spans = append(spans, Range{Start: start, End: delimiter})
		start = delimiter + 1
		if trailingDelimiter && i == len(delimiters)-1 {
			return spans
		}
	}
	if !trailingDelimiter {
		spans = append(spans, Range{Start: start, End: len(line)})
	}
	return spans
}
