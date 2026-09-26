package splice

import (
	"fmt"

	"github.com/zoster81/marksplice/internal/source"
)

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
	tableSources         map[int]source.TableMapping
}

func resolveTables(nodes []Node, tableSources map[int]source.TableMapping) (tableOwnerModel, error) {
	builder := tableOwnerModelBuilder{
		nodes:                nodes,
		tableIndexes:         make([]int, 0, len(tableSources)),
		tableOrdinalByAnchor: make(map[int]int, len(tableSources)),
		tableSources:         tableSources,
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
		anchor := table.tableData().TableAnchor
		mapping, mapped := b.tableSources[anchor]
		if err := validatePromotedTable(table, mapping, mapped, lastAnchor); err != nil {
			return err
		}
		if _, exists := b.tableOrdinalByAnchor[anchor]; exists {
			return fmt.Errorf("duplicate promoted table anchor %d", anchor)
		}
		resetTableAdjacency(table)
		b.tableOrdinalByAnchor[anchor] = len(b.tableIndexes)
		b.tableIndexes = append(b.tableIndexes, index)
		lastAnchor = anchor
	}
	return nil
}

func validatePromotedTable(table *Node, mapping source.TableMapping, mapped bool, lastAnchor int) error {
	tableTable := table.tableData()
	anchor := tableTable.TableAnchor
	if table.ID == "" || anchor < 0 || !mapped || anchor != table.Range.Start || mapping.Range != table.Range ||
		anchor <= lastAnchor || tableTable.TableColumnCount <= 0 || len(tableTable.TableAlignments) != tableTable.TableColumnCount || tableTable.TableBodyRowCount < 0 ||
		len(mapping.Delimiter.Cells) != tableTable.TableColumnCount {
		return fmt.Errorf("invalid promoted table at anchor %d", anchor)
	}
	return nil
}

func resetTableAdjacency(table *Node) {
	table.table.TablePromotedRowStart = 0
	table.table.TablePromotedRowCount = 0
	table.table.TableOwnedHeaderCellStart = 0
	table.table.TableOwnedHeaderCellCount = 0
}

func (b *tableOwnerModelBuilder) resolveOwnership() error {
	for index := range b.nodes {
		node := &b.nodes[index]
		switch node.Kind {
		case KindTableRow:
			node.table.TableID = ""
			ordinal, ok := b.tableOrdinalByAnchor[node.tableData().TableAnchor]
			if !ok {
				continue
			}
			table := &b.nodes[b.tableIndexes[ordinal]]
			mapping, ok := b.tableSources[table.tableData().TableAnchor]
			if !ok || node.tableData().TableRowAnchor <= mapping.Delimiter.Range.Start || node.tableData().TableRowAnchor >= table.Range.End {
				return fmt.Errorf("table row anchor %d escapes table %q", node.tableData().TableRowAnchor, table.ID)
			}
			b.semanticRowCounts[ordinal]++
			b.lastSemanticRow[ordinal] = node.tableData().TableRowAnchor
			if node.Editable {
				node.table.TableID = table.ID
				b.promotedRowCounts[ordinal]++
			}
		case KindTableCell:
			node.table.TableID = ""
			if !node.Editable {
				continue
			}
			ordinal, ok := b.tableOrdinalByAnchor[node.tableData().TableAnchor]
			if !ok {
				continue
			}
			table := &b.nodes[b.tableIndexes[ordinal]]
			if node.tableData().TableColumn < 0 || node.tableData().TableColumn >= table.tableData().TableColumnCount {
				return fmt.Errorf("table cell %q column %d escapes table %q", node.ID, node.tableData().TableColumn, table.ID)
			}
			node.table.TableID = table.ID
			if node.tableData().TableHeader {
				b.headerCellCounts[ordinal]++
			}
		}
	}
	return nil
}

func (b *tableOwnerModelBuilder) validateSemanticRows() error {
	for ordinal, tableIndex := range b.tableIndexes {
		table := &b.nodes[tableIndex]
		if b.semanticRowCounts[ordinal] != table.tableData().TableBodyRowCount {
			return fmt.Errorf("table %q semantic body-row count %d disagrees with observed %d", table.ID, table.tableData().TableBodyRowCount, b.semanticRowCounts[ordinal])
		}
		if table.tableData().TableBodyRowCount == 0 {
			if table.tableData().TableLastBodyRowAnchor != 0 {
				return fmt.Errorf("table %q has unexpected last body-row anchor %d", table.ID, table.tableData().TableLastBodyRowAnchor)
			}
			continue
		}
		if b.lastSemanticRow[ordinal] != table.tableData().TableLastBodyRowAnchor {
			return fmt.Errorf("table %q last body-row anchor %d disagrees with observed %d", table.ID, table.tableData().TableLastBodyRowAnchor, b.lastSemanticRow[ordinal])
		}
	}
	return nil
}

func (b *tableOwnerModelBuilder) assignAdjacencyRanges() (int, int) {
	rowTotal := 0
	headerTotal := 0
	for ordinal, tableIndex := range b.tableIndexes {
		table := &b.nodes[tableIndex]
		table.table.TablePromotedRowStart = rowTotal
		table.table.TablePromotedRowCount = b.promotedRowCounts[ordinal]
		rowTotal += b.promotedRowCounts[ordinal]
		table.table.TableOwnedHeaderCellStart = headerTotal
		table.table.TableOwnedHeaderCellCount = b.headerCellCounts[ordinal]
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
		rowCursors[ordinal] = table.tableData().TablePromotedRowStart
		headerCursors[ordinal] = table.tableData().TableOwnedHeaderCellStart
	}
	return rowCursors, headerCursors
}

func (b *tableOwnerModelBuilder) collectNodeAdjacency(node *Node, rowIDs, headerIDs []NodeID, rowCursors, headerCursors []int) error {
	ordinal, ok := b.tableOrdinalByAnchor[node.tableData().TableAnchor]
	if !ok {
		return nil
	}
	table := &b.nodes[b.tableIndexes[ordinal]]
	if node.Kind == KindTableRow && node.Editable && node.tableData().TableID == table.ID {
		limit := table.tableData().TablePromotedRowStart + table.tableData().TablePromotedRowCount
		if rowCursors[ordinal] >= limit {
			return fmt.Errorf("inconsistent table-row adjacency for %q", node.ID)
		}
		rowIDs[rowCursors[ordinal]] = node.ID
		rowCursors[ordinal]++
		return nil
	}
	if node.Kind == KindTableCell && node.Editable && node.tableData().TableHeader && node.tableData().TableID == table.ID {
		limit := table.tableData().TableOwnedHeaderCellStart + table.tableData().TableOwnedHeaderCellCount
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
		if rowCursors[ordinal] != table.tableData().TablePromotedRowStart+table.tableData().TablePromotedRowCount ||
			headerCursors[ordinal] != table.tableData().TableOwnedHeaderCellStart+table.tableData().TableOwnedHeaderCellCount {
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
		if !ok || row.Kind != KindTableRow || !row.Editable || row.tableData().TableID != table.ID || row.tableData().TableAnchor != table.tableData().TableAnchor || row.Range.Start <= previousStart {
			return nil, false
		}
		previousStart = row.Range.Start
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
		if !ok || cell.Kind != KindTableCell || !cell.Editable || !cell.tableData().TableHeader || cell.tableData().TableID != table.ID || cell.tableData().TableAnchor != table.tableData().TableAnchor || cell.tableData().TableColumn <= previousColumn || cell.tableData().TableColumn >= table.tableData().TableColumnCount {
			return nil, false
		}
		previousColumn = cell.tableData().TableColumn
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
	start, count := table.tableData().TablePromotedRowStart, table.tableData().TablePromotedRowCount
	ids := d.tableRowIDs
	if header {
		start, count = table.tableData().TableOwnedHeaderCellStart, table.tableData().TableOwnedHeaderCellCount
		ids = d.tableOwnedHeaderCellIDs
	}
	if start < 0 || count < 0 || start > len(ids) || count > len(ids)-start {
		return Node{}, nil, false
	}
	return table, ids[start : start+count], true
}
