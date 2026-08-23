package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicTableRowsExposeCompleteBodyLineRanges(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\r\n| - | - |\r\n| one | two |\r\n| three | four |\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := publicTableRows(t, doc)
	if len(rows) != 2 {
		t.Fatalf("table row count = %d, want 2", len(rows))
	}
	want := []string{"| one | two |\r\n", "| three | four |\r\n"}
	for i, row := range rows {
		if row.ColumnCount() != 2 {
			t.Fatalf("row %d column count = %d, want 2", i, row.ColumnCount())
		}
		got, ok := doc.SourceRange(row.Range())
		if !ok || string(got) != want[i] {
			t.Fatalf("row %d bytes = %q, %v; want %q, true", i, got, ok, want[i])
		}
	}
	if _, ok := doc.TableRow(marksplice.NodeID{}); ok {
		t.Fatal("TableRow(zero ID) ok = true, want false")
	}
}

func TestPublicTableRowsSupportEmptyCellsUnicodeAndUnterminatedEOF(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| | β |")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := publicTableRows(t, doc)
	if len(rows) != 1 {
		t.Fatalf("table row count = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ColumnCount() != 2 {
		t.Fatalf("ColumnCount() = %d, want 2", row.ColumnCount())
	}
	got, ok := doc.SourceRange(row.Range())
	if !ok || !bytes.Equal(got, []byte("| | β |")) {
		t.Fatalf("unterminated row bytes = %q, %v; want exact EOF row, true", got, ok)
	}
	if row.Range().End != len(source) {
		t.Fatalf("unterminated row end = %d, want EOF %d", row.Range().End, len(source))
	}
	if marksplice.KindTableRow != marksplice.KindImage+1 {
		t.Fatalf("KindTableRow = %d, want append-only kind after KindImage %d", marksplice.KindTableRow, marksplice.KindImage)
	}
}

func TestPublicRemoveTableRowPreservesSourceAndAllowsFinalBodyRemoval(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\r\n| - | - |\r\n| one | two |\r\n| three | four |\r\nTail\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := publicTableRows(t, doc)
	change, err := doc.PrepareRemoveTableRow(rows[0].ID())
	if err != nil {
		t.Fatalf("PrepareRemoveTableRow() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("| A | B |\r\n| - | - |\r\n| three | four |\r\nTail\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}

	oneRowSource := []byte("| A | B |\n| - | - |\n| one | two |\nTail\n")
	oneRowDoc, err := marksplice.Parse(oneRowSource)
	if err != nil {
		t.Fatalf("Parse(one row) error = %v", err)
	}
	oneRow := publicTableRows(t, oneRowDoc)[0]
	removeLast, err := oneRowDoc.PrepareRemoveTableRow(oneRow.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveTableRow(last) error = %v", err)
	}
	withoutRows, err := removeLast.Apply(oneRowSource)
	if err != nil {
		t.Fatalf("Apply(last) error = %v", err)
	}
	reparsed, err := marksplice.Parse(withoutRows)
	if err != nil {
		t.Fatalf("Parse(without rows) error = %v", err)
	}
	if rows := publicTableRows(t, reparsed); len(rows) != 0 {
		t.Fatalf("rows after final removal = %d, want 0", len(rows))
	}
	headerCells := 0
	for _, node := range reparsed.Nodes() {
		if node.Kind() != marksplice.KindTableCell {
			continue
		}
		cell, ok := reparsed.TableCell(node.ID())
		if ok && cell.Header() {
			headerCells++
		}
	}
	if headerCells != 2 {
		t.Fatalf("header cells after final row removal = %d, want 2", headerCells)
	}
}

func TestPublicInsertTableRowBeforeAfterRequiresOneCompatibleOwnedRow(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := publicTableRows(t, doc)

	before, err := doc.PrepareInsertTableRowBefore(rows[1].ID(), []byte("| middle | value |\n"))
	if err != nil {
		t.Fatalf("PrepareInsertTableRowBefore() error = %v", err)
	}
	got, err := before.Apply(source)
	if err != nil {
		t.Fatalf("Apply(before) error = %v", err)
	}
	wantBefore := []byte("| A | B |\n| - | - |\n| one | two |\n| middle | value |\n| three | four |\n")
	if !bytes.Equal(got, wantBefore) {
		t.Fatalf("before result = %q, want %q", got, wantBefore)
	}

	after, err := doc.PrepareInsertTableRowAfter(rows[1].ID(), []byte("| tail | value |"))
	if err != nil {
		t.Fatalf("PrepareInsertTableRowAfter(EOF) error = %v", err)
	}
	got, err = after.Apply(source)
	if err != nil {
		t.Fatalf("Apply(after) error = %v", err)
	}
	wantAfter := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n| tail | value |")
	if !bytes.Equal(got, wantAfter) {
		t.Fatalf("after result = %q, want %q", got, wantAfter)
	}
	afterDoc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(after result) error = %v", err)
	}
	afterRows := publicTableRows(t, afterDoc)
	if len(afterRows) != 3 {
		t.Fatalf("after row count = %d, want 3", len(afterRows))
	}
	lastBytes, ok := afterDoc.SourceRange(afterRows[2].Range())
	if !ok || string(lastBytes) != "| tail | value |" {
		t.Fatalf("inserted EOF row bytes = %q, %v; want exact fragment, true", lastBytes, ok)
	}

	for _, fragment := range [][]byte{
		[]byte("| merged | row |"),
		[]byte("| one | two |\n| extra | row |\n"),
		[]byte("| only one |\n"),
	} {
		if _, err := doc.PrepareInsertTableRowBefore(rows[0].ID(), fragment); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareInsertTableRowBefore(%q) error = %v, want ErrInvalidReplacement", fragment, err)
		}
	}
}

func TestPublicMoveTableRowWithinTablePreservesExactBytesAndRejectsCrossTable(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n| five | six |\n\n| X | Y |\n| - | - |\n| other | table |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := publicTableRows(t, doc)
	if len(rows) != 4 {
		t.Fatalf("table row count = %d, want 4", len(rows))
	}
	change, err := doc.PrepareMoveTableRowBefore(rows[2].ID(), rows[0].ID())
	if err != nil {
		t.Fatalf("PrepareMoveTableRowBefore() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(move) error = %v", err)
	}
	want := []byte("| A | B |\n| - | - |\n| five | six |\n| one | two |\n| three | four |\n\n| X | Y |\n| - | - |\n| other | table |\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("move result = %q, want %q", got, want)
	}
	if _, err := doc.PrepareMoveTableRowAfter(rows[0].ID(), rows[3].ID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("cross-table move error = %v, want ErrInvalidReplacement", err)
	}

	noOp, err := doc.PrepareMoveTableRowBefore(rows[1].ID(), rows[2].ID())
	if err != nil {
		t.Fatalf("PrepareMoveTableRowBefore(no-op) error = %v", err)
	}
	unchanged, err := noOp.Apply(source)
	if err != nil {
		t.Fatalf("Apply(no-op) error = %v", err)
	}
	if !bytes.Equal(unchanged, source) {
		t.Fatalf("no-op result = %q, want original", unchanged)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := noOp.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale no-op) error = %v, want ErrSourceConflict", err)
	}
}

func TestPublicTableRowRejectsInvalidTargetsAndPreparedChangesStaySourceBound(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	row := publicTableRows(t, doc)[0]
	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}
	if paragraph.ID().String() == "" {
		t.Fatal("paragraph not found")
	}
	if _, ok := doc.TableRow(paragraph.ID()); ok {
		t.Fatal("TableRow(paragraph) ok = true, want false")
	}
	if _, err := doc.PrepareRemoveTableRow(marksplice.NodeID{}); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareRemoveTableRow(zero) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareRemoveTableRow(paragraph.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareRemoveTableRow(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareReplaceTableRow(marksplice.NodeID{}, []byte("| x | y |\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareReplaceTableRow(zero) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareReplaceTableRow(paragraph.ID(), []byte("| x | y |\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceTableRow(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareInsertTableRowBefore(paragraph.ID(), []byte("| x | y |\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareInsertTableRowBefore(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareInsertTableRowAfter(marksplice.NodeID{}, []byte("| x | y |\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareInsertTableRowAfter(zero) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareMoveTableRowBefore(paragraph.ID(), row.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareMoveTableRowBefore(paragraph source) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareMoveTableRowBefore(row.ID(), paragraph.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareMoveTableRowBefore(paragraph anchor) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareMoveTableRowAfter(paragraph.ID(), row.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareMoveTableRowAfter(paragraph source) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareMoveTableRowAfter(row.ID(), paragraph.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareMoveTableRowAfter(paragraph anchor) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareMoveTableRowBefore(row.ID(), row.ID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareMoveTableRowBefore(self) error = %v, want ErrInvalidReplacement", err)
	}

	change, err := doc.PrepareRemoveTableRow(row.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveTableRow() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale removal) error = %v, want ErrSourceConflict", err)
	}
}

func TestPublicTableCellRowIdentityAndRowCellIDs(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B | C |\n| - | - | - |\n| one | | three |\n| four | five | six |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := publicTableRows(t, doc)
	if len(rows) != 2 {
		t.Fatalf("table row count = %d, want 2", len(rows))
	}
	if rows[0].ColumnCount() != 3 {
		t.Fatalf("first row ColumnCount() = %d, want 3", rows[0].ColumnCount())
	}

	firstBodyCells, headerCount := publicBodyCellsForRow(t, doc, rows[0].ID())
	if headerCount != 3 {
		t.Fatalf("header cell count = %d, want 3", headerCount)
	}
	if len(firstBodyCells) != 2 || firstBodyCells[0].Column() != 0 || firstBodyCells[1].Column() != 2 {
		t.Fatalf("first body promoted cells = %+v, want columns 0 and 2 only", firstBodyCells)
	}

	ids, ok := doc.TableRowCellIDs(rows[0].ID())
	if !ok {
		t.Fatal("TableRowCellIDs(first row) ok = false")
	}
	if len(ids) != 2 || ids[0] != firstBodyCells[0].ID() || ids[1] != firstBodyCells[1].ID() {
		t.Fatalf("TableRowCellIDs(first row) = %v, want promoted columns 0 and 2 in source order", ids)
	}
	ids[0] = marksplice.NodeID{}
	again, ok := doc.TableRowCellIDs(rows[0].ID())
	if !ok || len(again) != 2 || again[0] != firstBodyCells[0].ID() {
		t.Fatalf("mutating returned row cell IDs changed document: %v, %v", again, ok)
	}
	if ids, ok := doc.TableRowCellIDs(marksplice.NodeID{}); ok || ids != nil {
		t.Fatalf("TableRowCellIDs(zero) = %v, %v; want nil, false", ids, ok)
	}
	var paragraphID marksplice.NodeID
	paragraphDoc, err := marksplice.Parse([]byte("Paragraph.\n"))
	if err != nil {
		t.Fatalf("Parse(paragraph) error = %v", err)
	}
	for _, node := range paragraphDoc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraphID = node.ID()
			break
		}
	}
	if ids, ok := paragraphDoc.TableRowCellIDs(paragraphID); ok || ids != nil {
		t.Fatalf("TableRowCellIDs(paragraph) = %v, %v; want nil, false", ids, ok)
	}
	var nilDoc *marksplice.Document
	if ids, ok := nilDoc.TableRowCellIDs(rows[0].ID()); ok || ids != nil {
		t.Fatalf("nil Document.TableRowCellIDs() = %v, %v; want nil, false", ids, ok)
	}
}

func TestPublicTableRowNeighborIDsStayWithinSameTable(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n| five | six |\n\n| X | Y |\n| - | - |\n| other | table |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := publicTableRows(t, doc)
	if len(rows) != 4 {
		t.Fatalf("table row count = %d, want 4", len(rows))
	}
	seen := map[marksplice.TableRow]struct{}{rows[0]: {}}
	if _, ok := seen[rows[0]]; !ok {
		t.Fatal("TableRow must remain comparable")
	}

	if previous, ok := rows[0].PreviousID(); ok || previous.String() != "" {
		t.Fatalf("first row PreviousID() = %v, %v; want zero, false", previous, ok)
	}
	if next, ok := rows[0].NextID(); !ok || next != rows[1].ID() {
		t.Fatalf("first row NextID() = %v, %v; want %v, true", next, ok, rows[1].ID())
	}
	if previous, ok := rows[1].PreviousID(); !ok || previous != rows[0].ID() {
		t.Fatalf("middle row PreviousID() = %v, %v; want %v, true", previous, ok, rows[0].ID())
	}
	if next, ok := rows[1].NextID(); !ok || next != rows[2].ID() {
		t.Fatalf("middle row NextID() = %v, %v; want %v, true", next, ok, rows[2].ID())
	}
	if next, ok := rows[2].NextID(); ok || next.String() != "" {
		t.Fatalf("last first-table row NextID() = %v, %v; want zero, false", next, ok)
	}
	if previous, ok := rows[3].PreviousID(); ok || previous.String() != "" {
		t.Fatalf("only second-table row PreviousID() = %v, %v; want zero, false", previous, ok)
	}
	if next, ok := rows[3].NextID(); ok || next.String() != "" {
		t.Fatalf("only second-table row NextID() = %v, %v; want zero, false", next, ok)
	}

	var zero marksplice.TableRow
	if previous, ok := zero.PreviousID(); ok || previous.String() != "" {
		t.Fatalf("zero TableRow PreviousID() = %v, %v; want zero, false", previous, ok)
	}
	if next, ok := zero.NextID(); ok || next.String() != "" {
		t.Fatalf("zero TableRow NextID() = %v, %v; want zero, false", next, ok)
	}
}

func TestPublicTableRowHeaderCellIDsStayTableScopedAndSourceOrdered(t *testing.T) {
	t.Parallel()

	source := []byte("| A | | C |\n| - | - | - |\n| one | two | three |\n\n| X | Y |\n| - | - |\n| other | table |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := publicTableRows(t, doc)
	if len(rows) != 2 {
		t.Fatalf("table row count = %d, want 2", len(rows))
	}

	ids, ok := doc.TableRowHeaderCellIDs(rows[0].ID())
	if !ok || len(ids) != 2 {
		t.Fatalf("TableRowHeaderCellIDs(first table) = %v, %v; want 2 promoted header cells, true", ids, ok)
	}
	wantColumns := []int{0, 2}
	for index, id := range ids {
		cell, ok := doc.TableCell(id)
		if !ok || !cell.Header() || cell.Column() != wantColumns[index] {
			t.Fatalf("header cell %d = %+v, %v; want header column %d", index, cell, ok, wantColumns[index])
		}
		if rowID, ok := cell.RowID(); ok || rowID.String() != "" {
			t.Fatalf("header cell RowID() = %v, %v; want zero, false", rowID, ok)
		}
	}
	ids[0] = marksplice.NodeID{}
	again, ok := doc.TableRowHeaderCellIDs(rows[0].ID())
	if !ok || len(again) != 2 || again[0].String() == "" {
		t.Fatalf("mutating returned header IDs changed document: %v, %v", again, ok)
	}

	secondIDs, ok := doc.TableRowHeaderCellIDs(rows[1].ID())
	if !ok || len(secondIDs) != 2 {
		t.Fatalf("TableRowHeaderCellIDs(second table) = %v, %v; want 2, true", secondIDs, ok)
	}
	for _, id := range secondIDs {
		for _, firstID := range again {
			if id == firstID {
				t.Fatalf("header cell %v leaked across tables", id)
			}
		}
	}

	if ids, ok := doc.TableRowHeaderCellIDs(marksplice.NodeID{}); ok || ids != nil {
		t.Fatalf("TableRowHeaderCellIDs(zero) = %v, %v; want nil, false", ids, ok)
	}
	var nilDoc *marksplice.Document
	if ids, ok := nilDoc.TableRowHeaderCellIDs(rows[0].ID()); ok || ids != nil {
		t.Fatalf("nil Document.TableRowHeaderCellIDs() = %v, %v; want nil, false", ids, ok)
	}

	emptyHeaderDoc, err := marksplice.Parse([]byte("| | |\n| - | - |\n| one | two |\n"))
	if err != nil {
		t.Fatalf("Parse(all-empty header) error = %v", err)
	}
	emptyHeaderRows := publicTableRows(t, emptyHeaderDoc)
	if len(emptyHeaderRows) != 1 {
		t.Fatalf("all-empty-header row count = %d, want 1", len(emptyHeaderRows))
	}
	if ids, ok := emptyHeaderDoc.TableRowHeaderCellIDs(emptyHeaderRows[0].ID()); !ok || len(ids) != 0 {
		t.Fatalf("all-empty-header TableRowHeaderCellIDs() = %v, %v; want empty, true", ids, ok)
	}
}

func publicBodyCellsForRow(t *testing.T, doc *marksplice.Document, rowID marksplice.NodeID) ([]marksplice.TableCell, int) {
	t.Helper()

	var bodyCells []marksplice.TableCell
	headerCount := 0
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTableCell {
			continue
		}
		cell, ok := doc.TableCell(node.ID())
		if !ok {
			t.Fatalf("TableCell(%v) ok = false", node.ID())
		}
		if cell.Header() {
			headerCount++
			if headerRowID, ok := cell.RowID(); ok || headerRowID.String() != "" {
				t.Fatalf("header RowID() = %v, %v; want zero, false", headerRowID, ok)
			}
			continue
		}
		cellRowID, ok := cell.RowID()
		if !ok {
			t.Fatalf("body cell column %d RowID() ok = false", cell.Column())
		}
		if cellRowID == rowID {
			bodyCells = append(bodyCells, cell)
		}
	}
	return bodyCells, headerCount
}

func TestPublicReplaceTableRowPreservesExactHostStructure(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\r\n| - | - |\r\n| one | two |\r\n| three | four |\r\nTail\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := publicTableRows(t, doc)
	change, err := doc.PrepareReplaceTableRow(rows[0].ID(), []byte("| changed | value |\r\n"))
	if err != nil {
		t.Fatalf("PrepareReplaceTableRow() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("| A | B |\r\n| - | - |\r\n| changed | value |\r\n| three | four |\r\nTail\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("replacement result = %q, want %q", got, want)
	}
	reparsed, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(replacement) error = %v", err)
	}
	gotRows := publicTableRows(t, reparsed)
	if len(gotRows) != 2 {
		t.Fatalf("replacement row count = %d, want 2", len(gotRows))
	}
	firstBytes, ok := reparsed.SourceRange(gotRows[0].Range())
	if !ok || !bytes.Equal(firstBytes, []byte("| changed | value |\r\n")) {
		t.Fatalf("replacement row bytes = %q, %v", firstBytes, ok)
	}

	for _, replacement := range [][]byte{
		nil,
		[]byte("| only one |\r\n"),
		[]byte("| merged | row |"),
		[]byte("| one | two |\r\n| extra | row |\r\n"),
	} {
		if _, err := doc.PrepareReplaceTableRow(rows[0].ID(), replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceTableRow(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}

	eofSource := []byte("| A | B |\n| - | - |\n| one | two |")
	eofDoc, err := marksplice.Parse(eofSource)
	if err != nil {
		t.Fatalf("Parse(EOF table) error = %v", err)
	}
	eofRow := publicTableRows(t, eofDoc)[0]
	eofChange, err := eofDoc.PrepareReplaceTableRow(eofRow.ID(), []byte("| tail | value |"))
	if err != nil {
		t.Fatalf("PrepareReplaceTableRow(EOF) error = %v", err)
	}
	eofGot, err := eofChange.Apply(eofSource)
	if err != nil {
		t.Fatalf("Apply(EOF replacement) error = %v", err)
	}
	if wantEOF := []byte("| A | B |\n| - | - |\n| tail | value |"); !bytes.Equal(eofGot, wantEOF) {
		t.Fatalf("EOF replacement = %q, want %q", eofGot, wantEOF)
	}

	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale replacement) error = %v, want ErrSourceConflict", err)
	}
}

func publicTableRows(t *testing.T, doc *marksplice.Document) []marksplice.TableRow {
	t.Helper()
	var rows []marksplice.TableRow
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTableRow {
			continue
		}
		row, ok := doc.TableRow(node.ID())
		if !ok {
			t.Fatalf("TableRow(%v) ok = false", node.ID())
		}
		rows = append(rows, row)
	}
	return rows
}
