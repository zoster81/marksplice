package source

import "fmt"

// MapCompleteTableRows maps header, delimiter, and every semantic body row only when all rows have the same source-proven column count.
func MapCompleteTableRows(input []byte, table TableMapping, columnCount, bodyRowCount int) ([]TableRowMapping, error) {
	if columnCount <= 0 || bodyRowCount < 0 || !table.Range.Valid(len(input)) ||
		len(table.Header.Cells) != columnCount || len(table.Delimiter.Cells) != columnCount {
		return nil, fmt.Errorf("%w: incomplete table row model", ErrUnsupportedTableShape)
	}
	rows := make([]TableRowMapping, 0, bodyRowCount+2)
	rows = append(rows, table.Header, table.Delimiter)
	cursor := table.Delimiter.LineRange.End
	for index := 0; index < bodyRowCount; index++ {
		if cursor >= table.Range.End {
			return nil, fmt.Errorf("%w: missing body row %d", ErrUnsupportedTableShape, index)
		}
		row, err := MapTableRow(input, cursor)
		if err != nil || row.LineRange.Start != cursor || row.LineRange.End > table.Range.End || len(row.Cells) != columnCount {
			return nil, wrapUnsupportedTableShape(fmt.Sprintf("map complete body row %d", index), err)
		}
		rows = append(rows, row)
		cursor = row.LineRange.End
	}
	if cursor != table.Range.End {
		return nil, fmt.Errorf("%w: complete body rows end at %d, table ends at %d", ErrUnsupportedTableShape, cursor, table.Range.End)
	}
	return rows, nil
}

// TableColumnInsertion returns one zero-width patch that inserts a cell by cloning
// the nearest destination slot's horizontal padding and one existing row pipe.
func TableColumnInsertion(input []byte, row TableRowMapping, column int, content []byte) (Range, []byte, error) {
	if !validTableColumnInsertion(input, row, column, content) {
		return Range{}, nil, fmt.Errorf("%w: invalid table-column insertion", ErrUnsupportedTableShape)
	}

	templateIndex := column
	if templateIndex == len(row.Cells) {
		templateIndex--
	}
	template := row.Cells[templateIndex]
	separator, ok := tableColumnSeparator(input, row)
	if !ok {
		return Range{}, nil, fmt.Errorf("%w: row has no reusable column separator", ErrUnsupportedTableShape)
	}

	raw := make([]byte, 0, template.Range.End-template.Range.Start+len(content))
	raw = append(raw, input[template.Range.Start:template.ContentRange.Start]...)
	raw = append(raw, content...)
	raw = append(raw, input[template.ContentRange.End:template.Range.End]...)

	if column < len(row.Cells) {
		insertAt := row.Cells[column].Range.Start
		replacement := make([]byte, 0, len(raw)+len(separator))
		replacement = append(replacement, raw...)
		replacement = append(replacement, separator...)
		return Range{Start: insertAt, End: insertAt}, replacement, nil
	}

	insertAt := row.Cells[len(row.Cells)-1].Range.End
	replacement := make([]byte, 0, len(separator)+len(raw))
	replacement = append(replacement, separator...)
	replacement = append(replacement, raw...)
	return Range{Start: insertAt, End: insertAt}, replacement, nil
}

func validTableColumnInsertion(input []byte, row TableRowMapping, column int, content []byte) bool {
	return column >= 0 && column <= len(row.Cells) && !containsLineBreak(content) && validTableColumnEditRow(input, row)
}

func validTableColumnEditRow(input []byte, row TableRowMapping) bool {
	if !row.Range.Valid(len(input)) || len(row.Cells) == 0 {
		return false
	}
	for index, cell := range row.Cells {
		if !validTableColumnEditCell(input, row, cell) {
			return false
		}
		if index > 0 && row.Cells[index-1].Range.End >= cell.Range.Start {
			return false
		}
	}
	return true
}

func validTableColumnEditCell(input []byte, row TableRowMapping, cell TableCellMapping) bool {
	return cell.Range.Valid(len(input)) && cell.ContentRange.Valid(len(input)) &&
		cell.Range.Start >= row.Range.Start && cell.Range.End <= row.Range.End &&
		cell.ContentRange.Start >= cell.Range.Start && cell.ContentRange.End <= cell.Range.End
}

func tableColumnSeparator(input []byte, row TableRowMapping) ([]byte, bool) {
	for index := 0; index+1 < len(row.Cells); index++ {
		start := row.Cells[index].Range.End
		end := row.Cells[index+1].Range.Start
		if start >= 0 && end == start+1 && end <= len(input) && input[start] == '|' {
			return input[start:end], true
		}
	}
	cell := row.Cells[0]
	if cell.Range.Start > row.Range.Start && cell.Range.Start <= len(input) && input[cell.Range.Start-1] == '|' {
		return input[cell.Range.Start-1 : cell.Range.Start], true
	}
	if cell.Range.End < row.Range.End && cell.Range.End < len(input) && input[cell.Range.End] == '|' {
		return input[cell.Range.End : cell.Range.End+1], true
	}
	return nil, false
}

// TableColumnRemovalRange returns the exact source span whose deletion removes one column while retaining the row's outer-pipe style.
func TableColumnRemovalRange(row TableRowMapping, column int) (Range, error) {
	if len(row.Cells) <= 1 || column < 0 || column >= len(row.Cells) {
		return Range{}, fmt.Errorf("%w: invalid removal column %d", ErrUnsupportedTableShape, column)
	}
	if column < len(row.Cells)-1 {
		range_ := Range{Start: row.Cells[column].Range.Start, End: row.Cells[column+1].Range.Start}
		if range_.Start >= range_.End {
			return Range{}, fmt.Errorf("%w: invalid forward removal span", ErrUnsupportedTableShape)
		}
		return range_, nil
	}
	range_ := Range{Start: row.Cells[column-1].Range.End, End: row.Cells[column].Range.End}
	if range_.Start >= range_.End {
		return Range{}, fmt.Errorf("%w: invalid trailing removal span", ErrUnsupportedTableShape)
	}
	return range_, nil
}

// ReorderTableRowColumns returns one row with exact cell-content bytes reordered while preserving each destination slot's whitespace, separators, outer-pipe style, and line ending outside Row.Range.
func ReorderTableRowColumns(input []byte, row TableRowMapping, order []int) ([]byte, error) {
	if len(order) != len(row.Cells) || !validTableColumnEditRow(input, row) {
		return nil, fmt.Errorf("%w: invalid row or column order", ErrUnsupportedTableShape)
	}
	if !validTableColumnPermutation(order) {
		return nil, fmt.Errorf("%w: invalid column permutation", ErrUnsupportedTableShape)
	}

	result := make([]byte, 0, row.Range.End-row.Range.Start)
	result = append(result, input[row.Range.Start:row.Cells[0].Range.Start]...)
	for slot, column := range order {
		destination := row.Cells[slot]
		moved := row.Cells[column]
		result = append(result, input[destination.Range.Start:destination.ContentRange.Start]...)
		result = append(result, input[moved.ContentRange.Start:moved.ContentRange.End]...)
		result = append(result, input[destination.ContentRange.End:destination.Range.End]...)
		if slot < len(row.Cells)-1 {
			left := row.Cells[slot]
			right := row.Cells[slot+1]
			result = append(result, input[left.Range.End:right.Range.Start]...)
		}
	}
	result = append(result, input[row.Cells[len(row.Cells)-1].Range.End:row.Range.End]...)
	if len(result) != row.Range.End-row.Range.Start {
		return nil, fmt.Errorf("%w: reordered row length changed", ErrUnsupportedTableShape)
	}
	return result, nil
}

func validTableColumnPermutation(order []int) bool {
	seen := make([]bool, len(order))
	for _, column := range order {
		if column < 0 || column >= len(order) || seen[column] {
			return false
		}
		seen[column] = true
	}
	return true
}
