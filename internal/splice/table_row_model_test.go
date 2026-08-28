package splice

import "testing"

func TestTableRowCellIdentityModelBuildsSupportedAdjacency(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("| A | B | C |\n| - | - | - |\n| one | | three |\n| four | five | six |\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := internalTableRows(doc)
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2", len(rows))
	}
	ids, ok := doc.TableRowCellIDs(rows[0].ID)
	if !ok || len(ids) != 2 {
		t.Fatalf("TableRowCellIDs(first) = %v, %v; want 2 supported cells, true", ids, ok)
	}
	columns := []int{0, 2}
	for index, id := range ids {
		cell, ok := doc.nodeByID(id)
		if !ok || cell.TableHeader || cell.TableRowID != rows[0].ID || cell.TableColumn != columns[index] {
			t.Fatalf("cell %d = %+v, %v; want row %q column %d", index, cell, ok, rows[0].ID, columns[index])
		}
	}
	for _, node := range doc.nodes {
		if node.Kind == KindTableCell && node.Editable && node.TableHeader && node.TableRowID != "" {
			t.Fatalf("header cell %q unexpectedly has row ID %q", node.ID, node.TableRowID)
		}
	}
}

func TestTableRowCellIdentityModelAllowsContainerRelativeRowAnchor(t *testing.T) {
	t.Parallel()

	source := []byte("- JDK\n\n    | Version | JDK |\n    |---------|-----|\n    | master | 21 |\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := internalTableRows(doc)
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.TableRowAnchor <= row.Range.Start || row.TableRowAnchor >= row.ContentRange.End {
		t.Fatalf("semantic row anchor = %d, physical row = %v; want anchor inside indented physical row", row.TableRowAnchor, row.Range)
	}
	ids, ok := doc.TableRowCellIDs(row.ID)
	if !ok || len(ids) != 2 {
		t.Fatalf("TableRowCellIDs(indented row) = %v, %v; want 2 cells, true", ids, ok)
	}
}

func TestTableRowCellIdentityModelAllowsNoPromotedCells(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("| A | B |\n| - | - |\n| | |\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := internalTableRows(doc)
	if len(rows) != 1 || rows[0].TableColumnCount != 2 {
		t.Fatalf("rows = %+v, want one two-column body row", rows)
	}
	ids, ok := doc.TableRowCellIDs(rows[0].ID)
	if !ok || len(ids) != 0 {
		t.Fatalf("TableRowCellIDs(empty cells) = %v, %v; want empty, true", ids, ok)
	}
}

func TestResolveTableRowCellsLeavesUnpromotedParentUnresolved(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("| A | B |\n| - | - |\n| one | two |\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	nodes := append([]Node(nil), doc.nodes...)
	cellIndex := firstBodyTableCellIndex(nodes)
	if cellIndex < 0 {
		t.Fatal("body table cell not found")
	}
	nodes[cellIndex].TableRowAnchor = len(doc.source) + 10
	nodes[cellIndex].TableRowID = "stale"
	if _, err := resolveTableRowCells(nodes); err != nil {
		t.Fatalf("resolveTableRowCells(unpromoted parent) error = %v", err)
	}
	if nodes[cellIndex].TableRowID != "" {
		t.Fatalf("unresolved body cell row ID = %q, want empty", nodes[cellIndex].TableRowID)
	}
}

func TestResolveTableRowCellsRejectsCorruptSourceRelations(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n")
	tests := []struct {
		name   string
		mutate func([]Node)
	}{
		{
			name: "row anchor mismatches physical line",
			mutate: func(nodes []Node) {
				index := firstTableRowIndex(nodes)
				nodes[index].TableRowAnchor++
			},
		},
		{
			name: "cell column exceeds row width",
			mutate: func(nodes []Node) {
				rowIndex := firstTableRowIndex(nodes)
				cellIndex := firstBodyTableCellIndex(nodes)
				nodes[cellIndex].TableColumn = nodes[rowIndex].TableColumnCount
			},
		},
		{
			name: "cell source escapes row",
			mutate: func(nodes []Node) {
				rowIndex := firstTableRowIndex(nodes)
				cellIndex := firstBodyTableCellIndex(nodes)
				nodes[cellIndex].TableCellRange.Start = nodes[rowIndex].ContentRange.Start - 1
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := Parse(source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			nodes := append([]Node(nil), doc.nodes...)
			tt.mutate(nodes)
			if _, err := resolveTableRowCells(nodes); err == nil {
				t.Fatal("resolveTableRowCells(corrupt) error = nil, want fail-closed error")
			}
		})
	}
}

func TestTableRowCellIDsFailsClosedOnCorruptAdjacency(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n")
	tests := []struct {
		name   string
		mutate func(*Document, Node)
	}{
		{
			name: "range out of bounds",
			mutate: func(doc *Document, row Node) {
				index := doc.nodeIndex[row.ID]
				doc.nodes[index].TableRowCellStart = len(doc.tableCellIDs) + 1
			},
		},
		{
			name: "missing cell identity",
			mutate: func(doc *Document, row Node) {
				doc.tableCellIDs[row.TableRowCellStart] = "missing"
			},
		},
		{
			name: "cell belongs to another row",
			mutate: func(doc *Document, row Node) {
				rows := internalTableRows(doc)
				otherIDs, ok := doc.TableRowCellIDs(rows[1].ID)
				if !ok || len(otherIDs) == 0 {
					panic("second row cells unavailable")
				}
				doc.tableCellIDs[row.TableRowCellStart] = otherIDs[0]
			},
		},
		{
			name: "cell source order reversed",
			mutate: func(doc *Document, row Node) {
				start := row.TableRowCellStart
				doc.tableCellIDs[start], doc.tableCellIDs[start+1] = doc.tableCellIDs[start+1], doc.tableCellIDs[start]
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := Parse(source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			row := internalTableRows(doc)[0]
			if ids, ok := doc.TableRowCellIDs(row.ID); !ok || len(ids) != 2 {
				t.Fatalf("valid TableRowCellIDs() = %v, %v", ids, ok)
			}
			tt.mutate(doc, row)
			if ids, ok := doc.TableRowCellIDs(row.ID); ok || ids != nil {
				t.Fatalf("TableRowCellIDs(corrupt) = %v, %v; want nil, false", ids, ok)
			}
		})
	}
}

func TestTableRowModelBuildsSameTableNeighborsAndHeaderAdjacency(t *testing.T) {
	t.Parallel()

	source := []byte("| A | | C |\n| - | - | - |\n| one | two | three |\n| four | five | six |\n\n| X | Y |\n| - | - |\n| other | table |\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := internalTableRows(doc)
	if len(rows) != 3 {
		t.Fatalf("row count = %d, want 3", len(rows))
	}
	previous, next, ok := doc.TableRowNeighborIDs(rows[0].ID)
	if !ok || previous != "" || next != rows[1].ID {
		t.Fatalf("first-row neighbors = %q/%q, %v; want empty/%q, true", previous, next, ok, rows[1].ID)
	}
	previous, next, ok = doc.TableRowNeighborIDs(rows[1].ID)
	if !ok || previous != rows[0].ID || next != "" {
		t.Fatalf("second-row neighbors = %q/%q, %v; want %q/empty, true", previous, next, ok, rows[0].ID)
	}
	previous, next, ok = doc.TableRowNeighborIDs(rows[2].ID)
	if !ok || previous != "" || next != "" {
		t.Fatalf("other-table neighbors = %q/%q, %v; want empty/empty, true", previous, next, ok)
	}

	headerIDs, ok := doc.TableRowHeaderCellIDs(rows[0].ID)
	if !ok || len(headerIDs) != 2 {
		t.Fatalf("first-table header IDs = %v, %v; want 2, true", headerIDs, ok)
	}
	for index, column := range []int{0, 2} {
		cell, ok := doc.nodeByID(headerIDs[index])
		if !ok || !cell.TableHeader || cell.TableAnchor != rows[0].TableAnchor || cell.TableColumn != column {
			t.Fatalf("header cell %d = %+v, %v; want table anchor %d column %d", index, cell, ok, rows[0].TableAnchor, column)
		}
	}
	sameHeaderIDs, ok := doc.TableRowHeaderCellIDs(rows[1].ID)
	if !ok || len(sameHeaderIDs) != len(headerIDs) || sameHeaderIDs[0] != headerIDs[0] || sameHeaderIDs[1] != headerIDs[1] {
		t.Fatalf("same-table header IDs = %v, %v; want %v, true", sameHeaderIDs, ok, headerIDs)
	}
	otherHeaderIDs, ok := doc.TableRowHeaderCellIDs(rows[2].ID)
	if !ok || len(otherHeaderIDs) != 2 || otherHeaderIDs[0] == headerIDs[0] {
		t.Fatalf("other-table header IDs = %v, %v; want distinct 2-cell adjacency", otherHeaderIDs, ok)
	}
}

func TestResolveTableRowCellsRejectsMismatchedTableMembership(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("| A | B |\n| - | - |\n| one | two |\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	nodes := append([]Node(nil), doc.nodes...)
	cellIndex := firstBodyTableCellIndex(nodes)
	if cellIndex < 0 {
		t.Fatal("body table cell not found")
	}
	nodes[cellIndex].TableAnchor++
	if _, err := resolveTableRowCells(nodes); err == nil {
		t.Fatal("resolveTableRowCells(mismatched table anchor) error = nil, want fail-closed error")
	}
}

func TestTableRowNavigationFailsClosedOnCorruptModel(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n\n| X | Y |\n| - | - |\n| other | table |\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := internalTableRows(doc)
	if len(rows) != 3 {
		t.Fatalf("row count = %d, want 3", len(rows))
	}

	firstIndex := doc.nodeIndex[rows[0].ID]
	doc.nodes[firstIndex].TableNextRowID = rows[2].ID
	if previous, next, ok := doc.TableRowNeighborIDs(rows[0].ID); ok || previous != "" || next != "" {
		t.Fatalf("corrupt neighbors = %q/%q, %v; want empty/empty, false", previous, next, ok)
	}

	doc, err = Parse(source)
	if err != nil {
		t.Fatalf("Parse(second) error = %v", err)
	}
	rows = internalTableRows(doc)
	first := rows[0]
	if first.TableHeaderCellCount < 2 {
		t.Fatalf("header cell count = %d, want at least 2", first.TableHeaderCellCount)
	}
	start := first.TableHeaderCellStart
	doc.tableHeaderCellIDs[start], doc.tableHeaderCellIDs[start+1] = doc.tableHeaderCellIDs[start+1], doc.tableHeaderCellIDs[start]
	if ids, ok := doc.TableRowHeaderCellIDs(first.ID); ok || ids != nil {
		t.Fatalf("corrupt header adjacency = %v, %v; want nil, false", ids, ok)
	}
}

func firstTableRowIndex(nodes []Node) int {
	for index, node := range nodes {
		if node.Kind == KindTableRow && node.Editable {
			return index
		}
	}
	return -1
}

func firstBodyTableCellIndex(nodes []Node) int {
	for index, node := range nodes {
		if node.Kind == KindTableCell && node.Editable && !node.TableHeader {
			return index
		}
	}
	return -1
}
