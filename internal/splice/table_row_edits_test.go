package splice

import (
	"bytes"
	"errors"
	"slices"
	"testing"
)

func TestTableRowModelAndRemovalPreserveTableStructure(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\r\n| - | - |\r\n| one | two |\r\n| three | four |\r\nTail\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := internalTableRows(doc)
	if len(rows) != 2 {
		t.Fatalf("table row count = %d, want 2", len(rows))
	}
	if rows[0].TableAnchor != rows[1].TableAnchor || rows[0].TableColumnCount != 2 || rows[1].TableColumnCount != 2 {
		t.Fatalf("row table/column facts = anchor %d/%d columns %d/%d", rows[0].TableAnchor, rows[1].TableAnchor, rows[0].TableColumnCount, rows[1].TableColumnCount)
	}
	if got := string(source[rows[0].TableRowSource.LineRange.Start:rows[0].TableRowSource.LineRange.End]); got != "| one | two |\r\n" {
		t.Fatalf("first row bytes = %q", got)
	}

	change, err := doc.PrepareRemoveTableRow(rows[0].ID)
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
	oneRowDoc, err := Parse(oneRowSource)
	if err != nil {
		t.Fatalf("Parse(one row) error = %v", err)
	}
	last := internalTableRows(oneRowDoc)[0]
	removeLast, err := oneRowDoc.PrepareRemoveTableRow(last.ID)
	if err != nil {
		t.Fatalf("PrepareRemoveTableRow(last) error = %v", err)
	}
	withoutRows, err := removeLast.Apply(oneRowSource)
	if err != nil {
		t.Fatalf("Apply(last) error = %v", err)
	}
	reparsed, err := Parse(withoutRows)
	if err != nil {
		t.Fatalf("Parse(without rows) error = %v", err)
	}
	if got := len(internalTableRows(reparsed)); got != 0 {
		t.Fatalf("body rows after final removal = %d, want 0", got)
	}
	headerCells := 0
	for _, node := range reparsed.nodes {
		if node.Kind == KindTableCell && node.Editable && node.TableHeader {
			headerCells++
		}
	}
	if headerCells != 2 {
		t.Fatalf("header cells after final row removal = %d, want 2", headerCells)
	}
}

func TestTableRowInsertionValidatesHostOwnershipAndColumns(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := internalTableRows(doc)
	change, err := doc.PrepareInsertTableRowBefore(rows[1].ID, []byte("| middle | value |\n"))
	if err != nil {
		t.Fatalf("PrepareInsertTableRowBefore() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(before) error = %v", err)
	}
	want := []byte("| A | B |\n| - | - |\n| one | two |\n| middle | value |\n| three | four |\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("before result = %q, want %q", got, want)
	}

	after, err := doc.PrepareInsertTableRowAfter(rows[1].ID, []byte("| eof | value |"))
	if err != nil {
		t.Fatalf("PrepareInsertTableRowAfter(EOF) error = %v", err)
	}
	got, err = after.Apply(source)
	if err != nil {
		t.Fatalf("Apply(after) error = %v", err)
	}
	if want := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n| eof | value |"); !bytes.Equal(got, want) {
		t.Fatalf("after result = %q, want %q", got, want)
	}

	for _, fragment := range [][]byte{nil, []byte("| unsafe | row |"), []byte("| only one |\n"), []byte("| one | two |\n| extra | row |\n")} {
		if _, err := doc.PrepareInsertTableRowBefore(rows[0].ID, fragment); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareInsertTableRowBefore(%q) error = %v, want ErrInvalidReplacement", fragment, err)
		}
	}
}

func TestTableRowMoveIsAtomicSameTableAndSourceBound(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n| three | four |\n| five | six |\n\n| X | Y |\n| - | - |\n| other | table |\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := internalTableRows(doc)
	if len(rows) != 4 {
		t.Fatalf("table row count = %d, want 4", len(rows))
	}
	change, err := doc.PrepareMoveTableRowBefore(rows[2].ID, rows[0].ID)
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
	if _, err := doc.PrepareMoveTableRowAfter(rows[0].ID, rows[3].ID); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("cross-table move error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareMoveTableRowBefore(rows[0].ID, rows[0].ID); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("self move error = %v, want ErrInvalidReplacement", err)
	}

	noOp, err := doc.PrepareMoveTableRowBefore(rows[1].ID, rows[2].ID)
	if err != nil {
		t.Fatalf("PrepareMoveTableRowBefore(no-op) error = %v", err)
	}
	unchanged, err := noOp.Apply(source)
	if err != nil || !bytes.Equal(unchanged, source) {
		t.Fatalf("Apply(no-op) = %q, %v; want original, nil", unchanged, err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := noOp.Apply(stale); !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("Apply(stale no-op) error = %v, want ErrSourceConflict", err)
	}
}

func TestTableRowReplacementValidatesOwnedBodyRow(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\r\n| - | - |\r\n| one | two |\r\n| three | four |\r\nTail\r\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	row := internalTableRows(doc)[0]
	change, err := doc.PrepareReplaceTableRow(row.ID, []byte("| changed | value |\r\n"))
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
	for _, replacement := range [][]byte{nil, []byte("| only one |\r\n"), []byte("| unsafe | row |"), []byte("| one | two |\r\n| extra | row |\r\n")} {
		if _, err := doc.PrepareReplaceTableRow(row.ID, replacement); !errors.Is(err, ErrInvalidReplacement) {
			t.Fatalf("PrepareReplaceTableRow(%q) error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("Apply(stale replacement) error = %v, want ErrSourceConflict", err)
	}

	eofSource := []byte("| A | B |\n| - | - |\n| one | two |")
	eofDoc, err := Parse(eofSource)
	if err != nil {
		t.Fatalf("Parse(EOF) error = %v", err)
	}
	eofRow := internalTableRows(eofDoc)[0]
	eofChange, err := eofDoc.PrepareReplaceTableRow(eofRow.ID, []byte("| tail | value |"))
	if err != nil {
		t.Fatalf("PrepareReplaceTableRow(EOF) error = %v", err)
	}
	eofGot, err := eofChange.Apply(eofSource)
	if err != nil {
		t.Fatalf("Apply(EOF) error = %v", err)
	}
	if wantEOF := []byte("| A | B |\n| - | - |\n| tail | value |"); !bytes.Equal(eofGot, wantEOF) {
		t.Fatalf("EOF replacement = %q, want %q", eofGot, wantEOF)
	}
}

func TestAlignedTableRowReplacementPreservesSemanticAlignments(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| :--- | ---: |\n| one | two |\n| three | four |\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := internalTableRows(doc)
	if len(rows) != 2 || !slices.Equal(rows[0].TableAlignments, []TableAlignment{TableAlignmentLeft, TableAlignmentRight}) {
		t.Fatalf("aligned rows = %+v, want left/right semantics", rows)
	}
	change, err := doc.PrepareReplaceTableRow(rows[0].ID, []byte("| changed | value |\n"))
	if err != nil {
		t.Fatalf("PrepareReplaceTableRow() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("| A | B |\n| :--- | ---: |\n| changed | value |\n| three | four |\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("replacement result = %q, want %q", got, want)
	}
	reparsed, err := Parse(got)
	if err != nil {
		t.Fatalf("Parse(replaced) error = %v", err)
	}
	for index, row := range internalTableRows(reparsed) {
		if !slices.Equal(row.TableAlignments, []TableAlignment{TableAlignmentLeft, TableAlignmentRight}) {
			t.Fatalf("row %d alignments = %v, want left/right", index, row.TableAlignments)
		}
	}
}

func TestTableRowTargetsFailClosed(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| - | - |\n| one | two |\n\nParagraph.\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var paragraph NodeID
	for _, node := range doc.nodes {
		if node.Kind == KindParagraph && node.Editable {
			paragraph = node.ID
			break
		}
	}
	if paragraph == "" {
		t.Fatal("paragraph not found")
	}
	if _, err := doc.PrepareRemoveTableRow("missing"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("PrepareRemoveTableRow(missing) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareRemoveTableRow(paragraph); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareRemoveTableRow(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareReplaceTableRow("missing", []byte("| x | y |\n")); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("PrepareReplaceTableRow(missing) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareReplaceTableRow(paragraph, []byte("| x | y |\n")); !errors.Is(err, ErrInvalidTargetKind) {
		t.Fatalf("PrepareReplaceTableRow(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func internalTableRows(doc *Document) []Node {
	rows := make([]Node, 0)
	for _, node := range doc.nodes {
		if node.Kind == KindTableRow && node.Editable {
			rows = append(rows, node)
		}
	}
	return rows
}
