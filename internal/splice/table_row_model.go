package splice

import "fmt"

type tableRowModel struct {
	cellIDs  []NodeID
	rowCount int
}

func resolveTableRowCells(nodes []Node) (tableRowModel, error) {
	rowIndexes := make([]int, 0)
	ordinalByAnchor := make(map[int]int)
	lastRowStart := -1
	for index := range nodes {
		row := &nodes[index]
		if row.Kind != KindTableRow || !row.Editable {
			continue
		}
		anchor := row.TableRowAnchor
		if anchor < 0 || anchor != row.TableRowSource.LineRange.Start || anchor <= lastRowStart {
			return tableRowModel{}, fmt.Errorf("invalid promoted table-row anchor %d after %d", anchor, lastRowStart)
		}
		if previous, exists := ordinalByAnchor[anchor]; exists {
			return tableRowModel{}, fmt.Errorf("duplicate promoted table-row anchor %d for ordinals %d and %d", anchor, previous, len(rowIndexes))
		}
		ordinalByAnchor[anchor] = len(rowIndexes)
		rowIndexes = append(rowIndexes, index)
		lastRowStart = anchor
	}

	counts := make([]int, len(rowIndexes))
	lastColumns := make([]int, len(rowIndexes))
	for index := range lastColumns {
		lastColumns[index] = -1
	}
	for index := range nodes {
		cell := &nodes[index]
		if cell.Kind != KindTableCell || !cell.Editable || cell.TableHeader {
			continue
		}
		cell.TableRowID = ""
		ordinal, ok := ordinalByAnchor[cell.TableRowAnchor]
		if !ok {
			continue
		}
		row := &nodes[rowIndexes[ordinal]]
		if cell.TableColumn < 0 || cell.TableColumn >= row.TableColumnCount || cell.TableColumn <= lastColumns[ordinal] {
			return tableRowModel{}, fmt.Errorf("invalid promoted table-cell column %d for row %q", cell.TableColumn, row.ID)
		}
		if cell.TableCellSource.Range.Start < row.TableRowSource.Range.Start || cell.TableCellSource.Range.End > row.TableRowSource.Range.End {
			return tableRowModel{}, fmt.Errorf("promoted table cell %q escapes row %q", cell.ID, row.ID)
		}
		cell.TableRowID = row.ID
		counts[ordinal]++
		lastColumns[ordinal] = cell.TableColumn
	}

	total := 0
	starts := make([]int, len(rowIndexes))
	for ordinal, rowIndex := range rowIndexes {
		starts[ordinal] = total
		row := &nodes[rowIndex]
		row.TableRowCellStart = total
		row.TableRowCellCount = counts[ordinal]
		total += counts[ordinal]
	}

	cellIDs := make([]NodeID, total)
	cursors := append([]int(nil), starts...)
	for index := range nodes {
		cell := &nodes[index]
		if cell.Kind != KindTableCell || !cell.Editable || cell.TableHeader || cell.TableRowID == "" {
			continue
		}
		ordinal, ok := ordinalByAnchor[cell.TableRowAnchor]
		if !ok || cursors[ordinal] >= starts[ordinal]+counts[ordinal] {
			return tableRowModel{}, fmt.Errorf("inconsistent table-row cell adjacency for %q", cell.ID)
		}
		cellIDs[cursors[ordinal]] = cell.ID
		cursors[ordinal]++
	}
	for ordinal := range cursors {
		if cursors[ordinal] != starts[ordinal]+counts[ordinal] {
			return tableRowModel{}, fmt.Errorf("incomplete table-row cell adjacency for row ordinal %d", ordinal)
		}
	}
	return tableRowModel{cellIDs: cellIDs, rowCount: len(rowIndexes)}, nil
}

// TableRowCellIDs returns the promoted non-empty body-cell identities owned by one promoted table row in source order.
func (d *Document) TableRowCellIDs(id NodeID) ([]NodeID, bool) {
	if d == nil {
		return nil, false
	}
	row, ok := d.nodeByID(id)
	if !ok || row.Kind != KindTableRow || !row.Editable || row.TableRowCellStart < 0 || row.TableRowCellCount < 0 {
		return nil, false
	}
	start := row.TableRowCellStart
	count := row.TableRowCellCount
	if start > len(d.tableCellIDs) || count > len(d.tableCellIDs)-start {
		return nil, false
	}
	ids := d.tableCellIDs[start : start+count]
	previousColumn := -1
	for _, cellID := range ids {
		cell, ok := d.nodeByID(cellID)
		if !ok || cell.Kind != KindTableCell || !cell.Editable || cell.TableHeader || cell.TableRowID != row.ID || cell.TableColumn <= previousColumn {
			return nil, false
		}
		previousColumn = cell.TableColumn
	}
	return append([]NodeID(nil), ids...), true
}
