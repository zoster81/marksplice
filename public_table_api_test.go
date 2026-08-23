package marksplice_test

import (
	"slices"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicTablesExposeIndependentIdentityOwnershipAndNavigation(t *testing.T) {
	t.Parallel()

	source := []byte("Before.\n\n| A | B |\n| :--- | ---: |\n| one | two |\n\n| X | Y |\n| --- | :---: |\n\nAfter.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	tables := publicTables(t, doc)
	if len(tables) != 2 {
		t.Fatalf("table count = %d, want 2", len(tables))
	}
	_ = map[marksplice.Table]struct{}{tables[0]: {}}
	if marksplice.KindTable != marksplice.KindTableRow+1 {
		t.Fatalf("KindTable = %d, want append-only kind after KindTableRow %d", marksplice.KindTable, marksplice.KindTableRow)
	}

	wantSource := []string{
		"| A | B |\n| :--- | ---: |\n| one | two |\n",
		"| X | Y |\n| --- | :---: |\n",
	}
	wantRows := []int{1, 0}
	wantAlignments := [][]marksplice.TableAlignment{
		{marksplice.TableAlignmentLeft, marksplice.TableAlignmentRight},
		{marksplice.TableAlignmentDefault, marksplice.TableAlignmentCenter},
	}
	for index, table := range tables {
		if table.ColumnCount() != 2 || table.BodyRowCount() != wantRows[index] {
			t.Fatalf("table %d columns/body rows = %d/%d, want 2/%d", index, table.ColumnCount(), table.BodyRowCount(), wantRows[index])
		}
		got, ok := doc.SourceRange(table.Range())
		if !ok || string(got) != wantSource[index] {
			t.Fatalf("table %d source = %q/%v, want %q/true", index, got, ok, wantSource[index])
		}
		alignments, ok := doc.TableAlignments(table.ID())
		if !ok || !slices.Equal(alignments, wantAlignments[index]) {
			t.Fatalf("table %d alignments = %v/%v, want %v/true", index, alignments, ok, wantAlignments[index])
		}
		rowIDs, ok := doc.TableRowIDs(table.ID())
		if !ok || len(rowIDs) != wantRows[index] {
			t.Fatalf("table %d row IDs = %v/%v, want %d promoted rows/true", index, rowIDs, ok, wantRows[index])
		}
		headerIDs, ok := doc.TableHeaderCellIDs(table.ID())
		if !ok || len(headerIDs) != 2 {
			t.Fatalf("table %d header IDs = %v/%v, want 2/true", index, headerIDs, ok)
		}
		for _, id := range headerIDs {
			cell, ok := doc.TableCell(id)
			if !ok || !cell.Header() {
				t.Fatalf("TableCell(%v) = %+v/%v, want promoted header", id, cell, ok)
			}
			tableID, ok := cell.TableID()
			if !ok || tableID != table.ID() {
				t.Fatalf("header cell TableID() = %v/%v, want %v/true", tableID, ok, table.ID())
			}
		}
	}

	firstRows, _ := doc.TableRowIDs(tables[0].ID())
	row, ok := doc.TableRow(firstRows[0])
	if !ok {
		t.Fatal("TableRow(first) ok = false")
	}
	if tableID, ok := row.TableID(); !ok || tableID != tables[0].ID() {
		t.Fatalf("row TableID() = %v/%v, want %v/true", tableID, ok, tables[0].ID())
	}

	if _, ok := doc.Table(marksplice.NodeID{}); ok {
		t.Fatal("Table(zero ID) ok = true, want false")
	}
	if ids, ok := doc.TableRowIDs(marksplice.NodeID{}); ok || ids != nil {
		t.Fatalf("TableRowIDs(zero) = %v/%v, want nil/false", ids, ok)
	}
	if ids, ok := doc.TableHeaderCellIDs(marksplice.NodeID{}); ok || ids != nil {
		t.Fatalf("TableHeaderCellIDs(zero) = %v/%v, want nil/false", ids, ok)
	}
	if alignments, ok := doc.TableAlignments(marksplice.NodeID{}); ok || alignments != nil {
		t.Fatalf("TableAlignments(zero) = %v/%v, want nil/false", alignments, ok)
	}
}

func TestPublicTableSurvivesWithoutPromotedBodyRows(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| --- | --- |\n| only one |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tables := publicTables(t, doc)
	if len(tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(tables))
	}
	table := tables[0]
	if table.BodyRowCount() != 1 {
		t.Fatalf("BodyRowCount() = %d, want semantic count 1", table.BodyRowCount())
	}
	rowIDs, ok := doc.TableRowIDs(table.ID())
	if !ok || len(rowIDs) != 0 {
		t.Fatalf("TableRowIDs() = %v/%v, want empty promoted subset/true", rowIDs, ok)
	}
	bodyCells := 0
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTableCell {
			continue
		}
		cell, ok := doc.TableCell(node.ID())
		if !ok || cell.Header() {
			continue
		}
		bodyCells++
		tableID, ok := cell.TableID()
		if !ok || tableID != table.ID() {
			t.Fatalf("unowned promoted body cell TableID() = %v/%v, want %v/true", tableID, ok, table.ID())
		}
		if rowID, ok := cell.RowID(); ok || rowID.String() != "" {
			t.Fatalf("unpromoted-row cell RowID() = %v/%v, want zero/false", rowID, ok)
		}
	}
	if bodyCells != 1 {
		t.Fatalf("promoted body cells = %d, want 1", bodyCells)
	}
	got, ok := doc.SourceRange(table.Range())
	if !ok || string(got) != string(source) {
		t.Fatalf("Table.Range() source = %q/%v, want complete table %q/true", got, ok, source)
	}
}

func TestPublicTableUsesHeaderAnchorWhenGoldmarkSplitsLeadingParagraph(t *testing.T) {
	t.Parallel()

	source := []byte("Intro line.\n| A | B |\n| --- | --- |\n| x | y |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tables := publicTables(t, doc)
	if len(tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(tables))
	}
	got, ok := doc.SourceRange(tables[0].Range())
	want := "| A | B |\n| --- | --- |\n| x | y |\n"
	if !ok || string(got) != want {
		t.Fatalf("split-paragraph table source = %q/%v, want %q/true", got, ok, want)
	}
	rowIDs, ok := doc.TableRowIDs(tables[0].ID())
	if !ok || len(rowIDs) != 1 {
		t.Fatalf("split-paragraph TableRowIDs() = %v/%v, want one/true", rowIDs, ok)
	}
	row, ok := doc.TableRow(rowIDs[0])
	if !ok {
		t.Fatal("TableRow(split paragraph) ok = false")
	}
	if tableID, ok := row.TableID(); !ok || tableID != tables[0].ID() {
		t.Fatalf("split-paragraph row TableID() = %v/%v, want %v/true", tableID, ok, tables[0].ID())
	}
}

func TestPublicTableSupportsAllEmptyHeaderWithoutBodyRows(t *testing.T) {
	t.Parallel()

	doc, err := marksplice.Parse([]byte("| | |\n| :--- | ---: |\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tables := publicTables(t, doc)
	if len(tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(tables))
	}
	table := tables[0]
	if table.ColumnCount() != 2 || table.BodyRowCount() != 0 {
		t.Fatalf("table columns/body rows = %d/%d, want 2/0", table.ColumnCount(), table.BodyRowCount())
	}
	if ids, ok := doc.TableHeaderCellIDs(table.ID()); !ok || len(ids) != 0 {
		t.Fatalf("TableHeaderCellIDs() = %v/%v, want empty/true", ids, ok)
	}
	if ids, ok := doc.TableRowIDs(table.ID()); !ok || len(ids) != 0 {
		t.Fatalf("TableRowIDs() = %v/%v, want empty/true", ids, ok)
	}
}

func publicTables(t *testing.T, doc *marksplice.Document) []marksplice.Table {
	t.Helper()
	var tables []marksplice.Table
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTable {
			continue
		}
		table, ok := doc.Table(node.ID())
		if !ok {
			t.Fatalf("Table(%v) ok = false", node.ID())
		}
		tables = append(tables, table)
	}
	return tables
}
