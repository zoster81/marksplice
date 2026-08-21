package splice

import "fmt"

type tableRowModel struct {
	cellIDs       []NodeID
	headerCellIDs []NodeID
	rowCount      int
}

type tableRowModelBuilder struct {
	nodes                []Node
	rowIndexes           []int
	rowOrdinalByAnchor   map[int]int
	firstRowIndexByTable map[int]int
	bodyCellCounts       []int
	bodyCellStarts       []int
	headerCellCounts     map[int]int
	headerCellStarts     map[int]int
}

func resolveTableRowCells(nodes []Node) (tableRowModel, error) {
	builder := tableRowModelBuilder{
		nodes:                nodes,
		rowOrdinalByAnchor:   make(map[int]int),
		firstRowIndexByTable: make(map[int]int),
	}
	if err := builder.collectRows(); err != nil {
		return tableRowModel{}, err
	}
	builder.bodyCellCounts = make([]int, len(builder.rowIndexes))
	builder.bodyCellStarts = make([]int, len(builder.rowIndexes))
	builder.headerCellCounts = make(map[int]int, len(builder.firstRowIndexByTable))
	builder.headerCellStarts = make(map[int]int, len(builder.firstRowIndexByTable))
	if err := builder.resolveCellMembership(); err != nil {
		return tableRowModel{}, err
	}
	bodyTotal, headerTotal := builder.assignAdjacencyRanges()
	cellIDs, headerCellIDs, err := builder.collectAdjacency(bodyTotal, headerTotal)
	if err != nil {
		return tableRowModel{}, err
	}
	return tableRowModel{cellIDs: cellIDs, headerCellIDs: headerCellIDs, rowCount: len(builder.rowIndexes)}, nil
}

func (b *tableRowModelBuilder) collectRows() error {
	lastRowIndexByTable := make(map[int]int)
	lastRowStart := -1
	lastTableAnchor := -1
	for index := range b.nodes {
		row := &b.nodes[index]
		if row.Kind != KindTableRow || !row.Editable {
			continue
		}
		anchor := row.TableRowAnchor
		tableAnchor := row.TableAnchor
		if anchor < 0 || anchor != row.TableRowSource.LineRange.Start || anchor <= lastRowStart {
			return fmt.Errorf("invalid promoted table-row anchor %d after %d", anchor, lastRowStart)
		}
		if tableAnchor < 0 || tableAnchor > anchor || tableAnchor < lastTableAnchor {
			return fmt.Errorf("invalid promoted table anchor %d for row %q", tableAnchor, row.ID)
		}
		if previous, exists := b.rowOrdinalByAnchor[anchor]; exists {
			return fmt.Errorf("duplicate promoted table-row anchor %d for ordinals %d and %d", anchor, previous, len(b.rowIndexes))
		}
		row.TablePreviousRowID = ""
		row.TableNextRowID = ""
		if previousIndex, exists := lastRowIndexByTable[tableAnchor]; exists {
			previous := &b.nodes[previousIndex]
			row.TablePreviousRowID = previous.ID
			previous.TableNextRowID = row.ID
		} else {
			b.firstRowIndexByTable[tableAnchor] = index
		}
		lastRowIndexByTable[tableAnchor] = index
		b.rowOrdinalByAnchor[anchor] = len(b.rowIndexes)
		b.rowIndexes = append(b.rowIndexes, index)
		lastRowStart = anchor
		lastTableAnchor = tableAnchor
	}
	return nil
}

func (b *tableRowModelBuilder) resolveCellMembership() error {
	lastBodyColumns := make([]int, len(b.rowIndexes))
	for index := range lastBodyColumns {
		lastBodyColumns[index] = -1
	}
	lastHeaderColumns := make(map[int]int, len(b.firstRowIndexByTable))
	for index := range b.nodes {
		cell := &b.nodes[index]
		if cell.Kind != KindTableCell || !cell.Editable {
			continue
		}
		cell.TableRowID = ""
		if cell.TableHeader {
			if err := b.resolveHeaderCell(cell, lastHeaderColumns); err != nil {
				return err
			}
			continue
		}
		if err := b.resolveBodyCell(cell, lastBodyColumns); err != nil {
			return err
		}
	}
	return nil
}

func (b *tableRowModelBuilder) resolveHeaderCell(cell *Node, lastColumns map[int]int) error {
	rowIndex, hasPromotedRows := b.firstRowIndexByTable[cell.TableAnchor]
	if !hasPromotedRows {
		return nil
	}
	previousColumn, hasPrevious := lastColumns[cell.TableAnchor]
	if !hasPrevious {
		previousColumn = -1
	}
	row := &b.nodes[rowIndex]
	if cell.TableColumn < 0 || cell.TableColumn >= row.TableColumnCount || cell.TableColumn <= previousColumn {
		return fmt.Errorf("invalid promoted table-header column %d for table anchor %d", cell.TableColumn, cell.TableAnchor)
	}
	b.headerCellCounts[cell.TableAnchor]++
	lastColumns[cell.TableAnchor] = cell.TableColumn
	return nil
}

func (b *tableRowModelBuilder) resolveBodyCell(cell *Node, lastColumns []int) error {
	ordinal, ok := b.rowOrdinalByAnchor[cell.TableRowAnchor]
	if !ok {
		return nil
	}
	row := &b.nodes[b.rowIndexes[ordinal]]
	if cell.TableAnchor != row.TableAnchor {
		return fmt.Errorf("promoted table cell %q belongs to table anchor %d, want %d", cell.ID, cell.TableAnchor, row.TableAnchor)
	}
	if cell.TableColumn < 0 || cell.TableColumn >= row.TableColumnCount || cell.TableColumn <= lastColumns[ordinal] {
		return fmt.Errorf("invalid promoted table-cell column %d for row %q", cell.TableColumn, row.ID)
	}
	if cell.TableCellSource.Range.Start < row.TableRowSource.Range.Start || cell.TableCellSource.Range.End > row.TableRowSource.Range.End {
		return fmt.Errorf("promoted table cell %q escapes row %q", cell.ID, row.ID)
	}
	cell.TableRowID = row.ID
	b.bodyCellCounts[ordinal]++
	lastColumns[ordinal] = cell.TableColumn
	return nil
}

func (b *tableRowModelBuilder) assignAdjacencyRanges() (int, int) {
	bodyTotal := 0
	headerTotal := 0
	for ordinal, rowIndex := range b.rowIndexes {
		row := &b.nodes[rowIndex]
		b.bodyCellStarts[ordinal] = bodyTotal
		row.TableRowCellStart = bodyTotal
		row.TableRowCellCount = b.bodyCellCounts[ordinal]
		bodyTotal += b.bodyCellCounts[ordinal]

		headerStart, exists := b.headerCellStarts[row.TableAnchor]
		if !exists {
			headerStart = headerTotal
			b.headerCellStarts[row.TableAnchor] = headerStart
			headerTotal += b.headerCellCounts[row.TableAnchor]
		}
		row.TableHeaderCellStart = headerStart
		row.TableHeaderCellCount = b.headerCellCounts[row.TableAnchor]
	}
	return bodyTotal, headerTotal
}

func (b *tableRowModelBuilder) collectAdjacency(bodyTotal, headerTotal int) ([]NodeID, []NodeID, error) {
	bodyIDs := make([]NodeID, bodyTotal)
	headerIDs := make([]NodeID, headerTotal)
	bodyCursors := append([]int(nil), b.bodyCellStarts...)
	headerCursors := make(map[int]int, len(b.headerCellStarts))
	for tableAnchor, start := range b.headerCellStarts {
		headerCursors[tableAnchor] = start
	}
	for index := range b.nodes {
		cell := &b.nodes[index]
		if cell.Kind != KindTableCell || !cell.Editable {
			continue
		}
		if cell.TableHeader {
			if err := b.appendHeaderCellID(headerIDs, headerCursors, cell); err != nil {
				return nil, nil, err
			}
			continue
		}
		if err := b.appendBodyCellID(bodyIDs, bodyCursors, cell); err != nil {
			return nil, nil, err
		}
	}
	if err := b.validateAdjacencyCursors(bodyCursors, headerCursors); err != nil {
		return nil, nil, err
	}
	return bodyIDs, headerIDs, nil
}

func (b *tableRowModelBuilder) appendHeaderCellID(ids []NodeID, cursors map[int]int, cell *Node) error {
	start, ok := b.headerCellStarts[cell.TableAnchor]
	if !ok {
		return nil
	}
	cursor := cursors[cell.TableAnchor]
	if cursor >= start+b.headerCellCounts[cell.TableAnchor] {
		return fmt.Errorf("inconsistent table-header cell adjacency for %q", cell.ID)
	}
	ids[cursor] = cell.ID
	cursors[cell.TableAnchor] = cursor + 1
	return nil
}

func (b *tableRowModelBuilder) appendBodyCellID(ids []NodeID, cursors []int, cell *Node) error {
	if cell.TableRowID == "" {
		return nil
	}
	ordinal, ok := b.rowOrdinalByAnchor[cell.TableRowAnchor]
	if !ok || cursors[ordinal] >= b.bodyCellStarts[ordinal]+b.bodyCellCounts[ordinal] {
		return fmt.Errorf("inconsistent table-row cell adjacency for %q", cell.ID)
	}
	ids[cursors[ordinal]] = cell.ID
	cursors[ordinal]++
	return nil
}

func (b *tableRowModelBuilder) validateAdjacencyCursors(bodyCursors []int, headerCursors map[int]int) error {
	for ordinal, cursor := range bodyCursors {
		if cursor != b.bodyCellStarts[ordinal]+b.bodyCellCounts[ordinal] {
			return fmt.Errorf("incomplete table-row cell adjacency for row ordinal %d", ordinal)
		}
	}
	for tableAnchor, cursor := range headerCursors {
		if cursor != b.headerCellStarts[tableAnchor]+b.headerCellCounts[tableAnchor] {
			return fmt.Errorf("incomplete table-header cell adjacency for table anchor %d", tableAnchor)
		}
	}
	return nil
}

// TableRowCellIDs returns the promoted non-empty body-cell identities owned by one promoted table row in source order.
func (d *Document) TableRowCellIDs(id NodeID) ([]NodeID, bool) {
	row, ids, ok := d.tableRowCellAdjacency(id, false)
	if !ok {
		return nil, false
	}
	previousColumn := -1
	for _, cellID := range ids {
		cell, ok := d.nodeByID(cellID)
		if !ok || !validBodyTableRowCell(row, cell, previousColumn) {
			return nil, false
		}
		previousColumn = cell.TableColumn
	}
	return append([]NodeID(nil), ids...), true
}

// TableRowNeighborIDs returns the nearest promoted body-row identities before and after one row within the same table.
func (d *Document) TableRowNeighborIDs(id NodeID) (NodeID, NodeID, bool) {
	if d == nil {
		return "", "", false
	}
	row, ok := d.nodeByID(id)
	if !ok || row.Kind != KindTableRow || !row.Editable {
		return "", "", false
	}
	if row.TablePreviousRowID != "" && !d.validTableRowNeighbor(row, row.TablePreviousRowID, true) {
		return "", "", false
	}
	if row.TableNextRowID != "" && !d.validTableRowNeighbor(row, row.TableNextRowID, false) {
		return "", "", false
	}
	return row.TablePreviousRowID, row.TableNextRowID, true
}

func (d *Document) validTableRowNeighbor(row Node, neighborID NodeID, previous bool) bool {
	neighbor, ok := d.nodeByID(neighborID)
	if !ok || neighbor.Kind != KindTableRow || !neighbor.Editable || neighbor.TableAnchor != row.TableAnchor || neighbor.ID == row.ID {
		return false
	}
	if previous {
		return neighbor.TableRowSource.LineRange.Start < row.TableRowSource.LineRange.Start && neighbor.TableNextRowID == row.ID
	}
	return neighbor.TableRowSource.LineRange.Start > row.TableRowSource.LineRange.Start && neighbor.TablePreviousRowID == row.ID
}

// TableRowHeaderCellIDs returns the promoted non-empty header-cell identities for the table that owns one promoted body row.
func (d *Document) TableRowHeaderCellIDs(id NodeID) ([]NodeID, bool) {
	row, ids, ok := d.tableRowCellAdjacency(id, true)
	if !ok {
		return nil, false
	}
	previousColumn := -1
	for _, cellID := range ids {
		cell, ok := d.nodeByID(cellID)
		if !ok || !validHeaderTableRowCell(row, cell, previousColumn) {
			return nil, false
		}
		previousColumn = cell.TableColumn
	}
	return append([]NodeID(nil), ids...), true
}

func (d *Document) tableRowCellAdjacency(id NodeID, header bool) (Node, []NodeID, bool) {
	if d == nil {
		return Node{}, nil, false
	}
	row, ok := d.nodeByID(id)
	if !ok || row.Kind != KindTableRow || !row.Editable {
		return Node{}, nil, false
	}
	start, count := row.TableRowCellStart, row.TableRowCellCount
	ids := d.tableCellIDs
	if header {
		start, count = row.TableHeaderCellStart, row.TableHeaderCellCount
		ids = d.tableHeaderCellIDs
	}
	if start < 0 || count < 0 || start > len(ids) || count > len(ids)-start {
		return Node{}, nil, false
	}
	return row, ids[start : start+count], true
}

func validBodyTableRowCell(row, cell Node, previousColumn int) bool {
	return cell.Kind == KindTableCell && cell.Editable && !cell.TableHeader && cell.TableRowID == row.ID && cell.TableAnchor == row.TableAnchor && cell.TableColumn > previousColumn
}

func validHeaderTableRowCell(row, cell Node, previousColumn int) bool {
	return cell.Kind == KindTableCell && cell.Editable && cell.TableHeader && cell.TableRowID == "" && cell.TableAnchor == row.TableAnchor && cell.TableColumn >= 0 && cell.TableColumn < row.TableColumnCount && cell.TableColumn > previousColumn
}
