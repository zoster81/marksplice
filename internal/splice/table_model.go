package splice

import "fmt"

type tableOwnerModel struct {
	rowIDs        []NodeID
	headerCellIDs []NodeID
}

type tableOwnerModelBuilder struct {
	nodes                []Node
	tableIndexes         []int
	tableOrdinalByAnchor map[int]int
	semanticRowCounts    []int
	lastSemanticRow      []int
	promotedRowCounts    []int
	headerCellCounts     []int
}

func resolveTables(nodes []Node) (tableOwnerModel, error) {
	builder := tableOwnerModelBuilder{
		nodes:                nodes,
		tableOrdinalByAnchor: make(map[int]int),
	}
	if err := builder.collectTables(); err != nil {
		return tableOwnerModel{}, err
	}
	builder.semanticRowCounts = make([]int, len(builder.tableIndexes))
	builder.lastSemanticRow = make([]int, len(builder.tableIndexes))
	builder.promotedRowCounts = make([]int, len(builder.tableIndexes))
	builder.headerCellCounts = make([]int, len(builder.tableIndexes))
	if err := builder.resolveOwnership(); err != nil {
		return tableOwnerModel{}, err
	}
	if err := builder.validateSemanticRows(); err != nil {
		return tableOwnerModel{}, err
	}
	rowTotal, headerTotal := builder.assignAdjacencyRanges()
	rowIDs, headerCellIDs, err := builder.collectAdjacency(rowTotal, headerTotal)
	if err != nil {
		return tableOwnerModel{}, err
	}
	return tableOwnerModel{rowIDs: rowIDs, headerCellIDs: headerCellIDs}, nil
}

func (b *tableOwnerModelBuilder) collectTables() error {
	lastAnchor := -1
	for index := range b.nodes {
		table := &b.nodes[index]
		if table.Kind != KindTable || !table.Editable {
			continue
		}
		anchor := table.TableAnchor
		if table.ID == "" || anchor < 0 || anchor != table.TableSource.Range.Start || anchor <= lastAnchor || table.TableColumnCount <= 0 || len(table.TableAlignments) != table.TableColumnCount || table.TableBodyRowCount < 0 {
			return fmt.Errorf("invalid promoted table at anchor %d", anchor)
		}
		if _, exists := b.tableOrdinalByAnchor[anchor]; exists {
			return fmt.Errorf("duplicate promoted table anchor %d", anchor)
		}
		table.TablePromotedRowStart = 0
		table.TablePromotedRowCount = 0
		table.TableOwnedHeaderCellStart = 0
		table.TableOwnedHeaderCellCount = 0
		b.tableOrdinalByAnchor[anchor] = len(b.tableIndexes)
		b.tableIndexes = append(b.tableIndexes, index)
		lastAnchor = anchor
	}
	return nil
}

func (b *tableOwnerModelBuilder) resolveOwnership() error {
	for index := range b.nodes {
		node := &b.nodes[index]
		switch node.Kind {
		case KindTableRow:
			node.TableID = ""
			ordinal, ok := b.tableOrdinalByAnchor[node.TableAnchor]
			if !ok {
				continue
			}
			table := &b.nodes[b.tableIndexes[ordinal]]
			if node.TableRowAnchor <= table.TableSource.Delimiter.Range.Start || node.TableRowAnchor >= table.TableSource.Range.End {
				return fmt.Errorf("table row anchor %d escapes table %q", node.TableRowAnchor, table.ID)
			}
			b.semanticRowCounts[ordinal]++
			b.lastSemanticRow[ordinal] = node.TableRowAnchor
			if node.Editable {
				node.TableID = table.ID
				b.promotedRowCounts[ordinal]++
			}
		case KindTableCell:
			node.TableID = ""
			if !node.Editable {
				continue
			}
			ordinal, ok := b.tableOrdinalByAnchor[node.TableAnchor]
			if !ok {
				continue
			}
			table := &b.nodes[b.tableIndexes[ordinal]]
			if node.TableColumn < 0 || node.TableColumn >= table.TableColumnCount {
				return fmt.Errorf("table cell %q column %d escapes table %q", node.ID, node.TableColumn, table.ID)
			}
			node.TableID = table.ID
			if node.TableHeader {
				b.headerCellCounts[ordinal]++
			}
		}
	}
	return nil
}

func (b *tableOwnerModelBuilder) validateSemanticRows() error {
	for ordinal, tableIndex := range b.tableIndexes {
		table := &b.nodes[tableIndex]
		if b.semanticRowCounts[ordinal] != table.TableBodyRowCount {
			return fmt.Errorf("table %q semantic body-row count %d disagrees with observed %d", table.ID, table.TableBodyRowCount, b.semanticRowCounts[ordinal])
		}
		if table.TableBodyRowCount == 0 {
			if table.TableLastBodyRowAnchor != 0 {
				return fmt.Errorf("table %q has unexpected last body-row anchor %d", table.ID, table.TableLastBodyRowAnchor)
			}
			continue
		}
		if b.lastSemanticRow[ordinal] != table.TableLastBodyRowAnchor {
			return fmt.Errorf("table %q last body-row anchor %d disagrees with observed %d", table.ID, table.TableLastBodyRowAnchor, b.lastSemanticRow[ordinal])
		}
	}
	return nil
}

func (b *tableOwnerModelBuilder) assignAdjacencyRanges() (int, int) {
	rowTotal := 0
	headerTotal := 0
	for ordinal, tableIndex := range b.tableIndexes {
		table := &b.nodes[tableIndex]
		table.TablePromotedRowStart = rowTotal
		table.TablePromotedRowCount = b.promotedRowCounts[ordinal]
		rowTotal += b.promotedRowCounts[ordinal]
		table.TableOwnedHeaderCellStart = headerTotal
		table.TableOwnedHeaderCellCount = b.headerCellCounts[ordinal]
		headerTotal += b.headerCellCounts[ordinal]
	}
	return rowTotal, headerTotal
}

func (b *tableOwnerModelBuilder) collectAdjacency(rowTotal, headerTotal int) ([]NodeID, []NodeID, error) {
	rowIDs := make([]NodeID, rowTotal)
	headerIDs := make([]NodeID, headerTotal)
	rowCursors, headerCursors := b.tableAdjacencyCursors()
	for index := range b.nodes {
		if err := b.collectNodeAdjacency(&b.nodes[index], rowIDs, headerIDs, rowCursors, headerCursors); err != nil {
			return nil, nil, err
		}
	}
	if err := b.validateAdjacencyCursors(rowCursors, headerCursors); err != nil {
		return nil, nil, err
	}
	return rowIDs, headerIDs, nil
}

func (b *tableOwnerModelBuilder) tableAdjacencyCursors() ([]int, []int) {
	rowCursors := make([]int, len(b.tableIndexes))
	headerCursors := make([]int, len(b.tableIndexes))
	for ordinal, tableIndex := range b.tableIndexes {
		table := &b.nodes[tableIndex]
		rowCursors[ordinal] = table.TablePromotedRowStart
		headerCursors[ordinal] = table.TableOwnedHeaderCellStart
	}
	return rowCursors, headerCursors
}

func (b *tableOwnerModelBuilder) collectNodeAdjacency(node *Node, rowIDs, headerIDs []NodeID, rowCursors, headerCursors []int) error {
	ordinal, ok := b.tableOrdinalByAnchor[node.TableAnchor]
	if !ok {
		return nil
	}
	table := &b.nodes[b.tableIndexes[ordinal]]
	if node.Kind == KindTableRow && node.Editable && node.TableID == table.ID {
		limit := table.TablePromotedRowStart + table.TablePromotedRowCount
		if rowCursors[ordinal] >= limit {
			return fmt.Errorf("inconsistent table-row adjacency for %q", node.ID)
		}
		rowIDs[rowCursors[ordinal]] = node.ID
		rowCursors[ordinal]++
		return nil
	}
	if node.Kind == KindTableCell && node.Editable && node.TableHeader && node.TableID == table.ID {
		limit := table.TableOwnedHeaderCellStart + table.TableOwnedHeaderCellCount
		if headerCursors[ordinal] >= limit {
			return fmt.Errorf("inconsistent table-header adjacency for %q", node.ID)
		}
		headerIDs[headerCursors[ordinal]] = node.ID
		headerCursors[ordinal]++
	}
	return nil
}

func (b *tableOwnerModelBuilder) validateAdjacencyCursors(rowCursors, headerCursors []int) error {
	for ordinal, tableIndex := range b.tableIndexes {
		table := &b.nodes[tableIndex]
		if rowCursors[ordinal] != table.TablePromotedRowStart+table.TablePromotedRowCount ||
			headerCursors[ordinal] != table.TableOwnedHeaderCellStart+table.TableOwnedHeaderCellCount {
			return fmt.Errorf("incomplete table adjacency for %q", table.ID)
		}
	}
	return nil
}

// TableRowIDs returns the promoted body-row identities owned by one promoted table in source order.
func (d *Document) TableRowIDs(id NodeID) ([]NodeID, bool) {
	table, ids, ok := d.tableOwnedAdjacency(id, false)
	if !ok {
		return nil, false
	}
	previousStart := -1
	for _, rowID := range ids {
		row, ok := d.nodeByID(rowID)
		if !ok || row.Kind != KindTableRow || !row.Editable || row.TableID != table.ID || row.TableAnchor != table.TableAnchor || row.TableRowSource.LineRange.Start <= previousStart {
			return nil, false
		}
		previousStart = row.TableRowSource.LineRange.Start
	}
	return append([]NodeID(nil), ids...), true
}

// TableHeaderCellIDs returns the promoted non-empty header-cell identities owned by one promoted table in source order.
func (d *Document) TableHeaderCellIDs(id NodeID) ([]NodeID, bool) {
	table, ids, ok := d.tableOwnedAdjacency(id, true)
	if !ok {
		return nil, false
	}
	previousColumn := -1
	for _, cellID := range ids {
		cell, ok := d.nodeByID(cellID)
		if !ok || cell.Kind != KindTableCell || !cell.Editable || !cell.TableHeader || cell.TableID != table.ID || cell.TableAnchor != table.TableAnchor || cell.TableColumn <= previousColumn || cell.TableColumn >= table.TableColumnCount {
			return nil, false
		}
		previousColumn = cell.TableColumn
	}
	return append([]NodeID(nil), ids...), true
}

func (d *Document) tableOwnedAdjacency(id NodeID, header bool) (Node, []NodeID, bool) {
	if d == nil {
		return Node{}, nil, false
	}
	table, ok := d.nodeByID(id)
	if !ok || table.Kind != KindTable || !table.Editable {
		return Node{}, nil, false
	}
	start, count := table.TablePromotedRowStart, table.TablePromotedRowCount
	ids := d.tableRowIDs
	if header {
		start, count = table.TableOwnedHeaderCellStart, table.TableOwnedHeaderCellCount
		ids = d.tableOwnedHeaderCellIDs
	}
	if start < 0 || count < 0 || start > len(ids) || count > len(ids)-start {
		return Node{}, nil, false
	}
	return table, ids[start : start+count], true
}
