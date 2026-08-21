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
				nodes[cellIndex].TableCellSource.Range.Start = nodes[rowIndex].TableRowSource.Range.Start - 1
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
