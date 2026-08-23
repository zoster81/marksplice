package publictest

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPrepareSetTableColumnAlignmentPreservesDelimiterTrivia(t *testing.T) {
	t.Parallel()

	source := []byte("Before\r\n\r\n | A | B | C | \r\n | :----- | ---: | :----: | \r\n | x | y | z | \r\nAfter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tables := publicTables(t, doc)
	if len(tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(tables))
	}

	change, err := doc.PrepareSetTableColumnAlignment(tables[0].ID(), 0, marksplice.TableAlignmentRight)
	if err != nil {
		t.Fatalf("PrepareSetTableColumnAlignment() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("Before\r\n\r\n | A | B | C | \r\n | -----: | ---: | :----: | \r\n | x | y | z | \r\nAfter\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}

	reparsed, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updated := publicTables(t, reparsed)
	alignments, ok := reparsed.TableAlignments(updated[0].ID())
	wantAlignments := []marksplice.TableAlignment{marksplice.TableAlignmentRight, marksplice.TableAlignmentRight, marksplice.TableAlignmentCenter}
	if !ok || !slices.Equal(alignments, wantAlignments) {
		t.Fatalf("TableAlignments(updated) = %v/%v, want %v/true", alignments, ok, wantAlignments)
	}
}

func TestPrepareSetTableAlignmentsChangesMultipleColumnsAtomically(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B | C |\r\n| :----- | ---: | :----: |\r\n| x | y | z |\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	table := publicTables(t, doc)[0]
	requested := []marksplice.TableAlignment{
		marksplice.TableAlignmentCenter,
		marksplice.TableAlignmentDefault,
		marksplice.TableAlignmentLeft,
	}
	change, err := doc.PrepareSetTableAlignments(table.ID(), requested)
	if err != nil {
		t.Fatalf("PrepareSetTableAlignments() error = %v", err)
	}
	requested[0] = marksplice.TableAlignmentDefault
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("| A | B | C |\r\n| :-----: | --- | :---- |\r\n| x | y | z |\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}

	reparsed, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updated := publicTables(t, reparsed)[0]
	alignments, ok := reparsed.TableAlignments(updated.ID())
	wantAlignments := []marksplice.TableAlignment{
		marksplice.TableAlignmentCenter,
		marksplice.TableAlignmentDefault,
		marksplice.TableAlignmentLeft,
	}
	if !ok || !slices.Equal(alignments, wantAlignments) {
		t.Fatalf("TableAlignments(updated) = %v/%v, want %v/true", alignments, ok, wantAlignments)
	}

	if _, err := doc.PrepareSetTableAlignments(table.ID(), requested[:2]); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("short alignment vector error = %v, want ErrInvalidReplacement", err)
	}
	invalid := []marksplice.TableAlignment{marksplice.TableAlignmentLeft, marksplice.TableAlignment(255), marksplice.TableAlignmentRight}
	if _, err := doc.PrepareSetTableAlignments(table.ID(), invalid); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("invalid alignment vector error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareSetTableColumnAlignmentSupportsHeaderOnlyAndSourceBoundNoOp(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| --- | :----: |\nTail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	table := publicTables(t, doc)[0]

	change, err := doc.PrepareSetTableColumnAlignment(table.ID(), 1, marksplice.TableAlignmentCenter)
	if err != nil {
		t.Fatalf("PrepareSetTableColumnAlignment(no-op) error = %v", err)
	}
	unchanged, err := change.Apply(source)
	if err != nil || !bytes.Equal(unchanged, source) {
		t.Fatalf("Apply(no-op) = %q, %v; want original, nil", unchanged, err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale no-op) error = %v, want ErrSourceConflict", err)
	}

	if _, err := doc.PrepareSetTableColumnAlignment(table.ID(), -1, marksplice.TableAlignmentLeft); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("negative column error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareSetTableColumnAlignment(table.ID(), table.ColumnCount(), marksplice.TableAlignmentLeft); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("out-of-range column error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareSetTableColumnAlignment(table.ID(), 0, marksplice.TableAlignment(255)); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("invalid alignment error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareAppendTableRowSupportsHeaderOnlyAndUnpromotedExistingRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   []byte
		fragment []byte
		want     []byte
	}{
		{
			name:     "header only CRLF",
			source:   []byte("| A | B |\r\n| --- | --- |\r\n\r\nTail\r\n"),
			fragment: []byte("| one | two |\r\n"),
			want:     []byte("| A | B |\r\n| --- | --- |\r\n| one | two |\r\n\r\nTail\r\n"),
		},
		{
			name:     "existing semantic unpromoted row",
			source:   []byte("| A | B |\n| --- | --- |\n| only one |\n\nTail\n"),
			fragment: []byte("| two | columns |\n"),
			want:     []byte("| A | B |\n| --- | --- |\n| only one |\n| two | columns |\n\nTail\n"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			table := publicTables(t, doc)[0]
			change, err := doc.PrepareAppendTableRow(table.ID(), tt.fragment)
			if err != nil {
				t.Fatalf("PrepareAppendTableRow() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
			reparsed, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(updated) error = %v", err)
			}
			updated := publicTables(t, reparsed)[0]
			if updated.BodyRowCount() != table.BodyRowCount()+1 {
				t.Fatalf("BodyRowCount(updated) = %d, want %d", updated.BodyRowCount(), table.BodyRowCount()+1)
			}
		})
	}
}

func TestPrepareAppendTableRowAcceptsEOFRowAfterExistingLineBoundary(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| --- | --- |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	table := publicTables(t, doc)[0]
	change, err := doc.PrepareAppendTableRow(table.ID(), []byte("| one | two |"))
	if err != nil {
		t.Fatalf("PrepareAppendTableRow(EOF) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(EOF) error = %v", err)
	}
	want := []byte("| A | B |\n| --- | --- |\n| one | two |")
	if !bytes.Equal(got, want) {
		t.Fatalf("EOF append = %q, want %q", got, want)
	}
}

func TestTableMutationTargetsFailClosed(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| --- | --- |\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var paragraph marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node.ID()
			break
		}
	}
	if paragraph == (marksplice.NodeID{}) {
		t.Fatal("paragraph ID not found")
	}
	if _, err := doc.PrepareAppendTableRow(marksplice.NodeID{}, []byte("| x | y |\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("append zero ID error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareAppendTableRow(paragraph, []byte("| x | y |\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("append paragraph error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareSetTableColumnAlignment(paragraph, 0, marksplice.TableAlignmentLeft); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("alignment paragraph error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareSetTableAlignments(paragraph, []marksplice.TableAlignment{marksplice.TableAlignmentLeft}); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("alignment vector paragraph error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareInsertTableColumn(paragraph, 0, []byte("x"), marksplice.TableAlignmentDefault, nil); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("insert column paragraph error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareRemoveTableColumn(paragraph, 0); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("remove column paragraph error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareMoveTableColumn(paragraph, 0, 0); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("move column paragraph error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestPrepareInsertTableColumnPreservesRowStyleAndSemanticAlignment(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B | C |\r\n| :---- | -----: | :---: |\r\n| one | two | three |\r\n| alpha | beta | gamma |\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	table := publicTables(t, doc)[0]
	header := []byte("X")
	body := [][]byte{[]byte("new"), nil}
	change, err := doc.PrepareInsertTableColumn(table.ID(), 1, header, marksplice.TableAlignmentCenter, body)
	if err != nil {
		t.Fatalf("PrepareInsertTableColumn() error = %v", err)
	}
	header[0] = 'Y'
	body[0][0] = 'N'
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("| A | X | B | C |\r\n| :---- | :-----: | -----: | :---: |\r\n| one | new | two | three |\r\n| alpha |  | beta | gamma |\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}

	reparsed, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updated := publicTables(t, reparsed)[0]
	if updated.ColumnCount() != 4 || updated.BodyRowCount() != 2 {
		t.Fatalf("updated table counts = columns %d rows %d, want 4/2", updated.ColumnCount(), updated.BodyRowCount())
	}
	alignments, ok := reparsed.TableAlignments(updated.ID())
	wantAlignments := []marksplice.TableAlignment{
		marksplice.TableAlignmentLeft,
		marksplice.TableAlignmentCenter,
		marksplice.TableAlignmentRight,
		marksplice.TableAlignmentCenter,
	}
	if !ok || !slices.Equal(alignments, wantAlignments) {
		t.Fatalf("updated alignments = %v/%v, want %v/true", alignments, ok, wantAlignments)
	}
}

func TestPrepareInsertTableColumnSupportsSingleColumnAndHeaderOnlyTables(t *testing.T) {
	t.Parallel()

	source := []byte("| A |\n| :---- |\n| one |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse(single) error = %v", err)
	}
	table := publicTables(t, doc)[0]
	change, err := doc.PrepareInsertTableColumn(table.ID(), 1, []byte("B"), marksplice.TableAlignmentRight, [][]byte{[]byte("two")})
	if err != nil {
		t.Fatalf("PrepareInsertTableColumn(single) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply(single) error = %v", err)
	}
	want := []byte("| A | B |\n| :---- | ----: |\n| one | two |\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("single-column insertion = %q, want %q", got, want)
	}

	headerOnly := []byte("| A | B |\n| --- | --- |\n\nTail\n")
	headerDoc, err := marksplice.Parse(headerOnly)
	if err != nil {
		t.Fatalf("Parse(header-only) error = %v", err)
	}
	headerTable := publicTables(t, headerDoc)[0]
	change, err = headerDoc.PrepareInsertTableColumn(headerTable.ID(), 0, []byte("Z"), marksplice.TableAlignmentDefault, nil)
	if err != nil {
		t.Fatalf("PrepareInsertTableColumn(header-only) error = %v", err)
	}
	got, err = change.Apply(headerOnly)
	if err != nil {
		t.Fatalf("Apply(header-only) error = %v", err)
	}
	want = []byte("| Z | A | B |\n| --- | --- | --- |\n\nTail\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("header-only insertion = %q, want %q", got, want)
	}
}

func TestPrepareInsertTableColumnPreservesFollowingTableExactly(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| --- | --- |\n| one | two |\n\n| X | Y |\n| :---- | ---: |\n| left | right |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tables := publicTables(t, doc)
	if len(tables) != 2 {
		t.Fatalf("table count = %d, want 2", len(tables))
	}
	secondBefore, ok := doc.SourceRange(tables[1].Range())
	if !ok {
		t.Fatal("SourceRange(second table) ok = false")
	}
	change, err := doc.PrepareInsertTableColumn(tables[0].ID(), 1, []byte("M"), marksplice.TableAlignmentLeft, [][]byte{[]byte("middle")})
	if err != nil {
		t.Fatalf("PrepareInsertTableColumn() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	reparsed, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updatedTables := publicTables(t, reparsed)
	if len(updatedTables) != 2 {
		t.Fatalf("updated table count = %d, want 2", len(updatedTables))
	}
	secondAfter, ok := reparsed.SourceRange(updatedTables[1].Range())
	if !ok || !bytes.Equal(secondAfter, secondBefore) {
		t.Fatalf("following table = %q/%v, want exact %q/true", secondAfter, ok, secondBefore)
	}
}

func TestPrepareInsertTableColumnFailsClosedForInvalidInputOrIncompleteRows(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| --- | --- |\n| one | two |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	table := publicTables(t, doc)[0]
	tests := []struct {
		column    int
		header    []byte
		alignment marksplice.TableAlignment
		body      [][]byte
	}{
		{-1, []byte("X"), marksplice.TableAlignmentDefault, [][]byte{[]byte("x")}},
		{3, []byte("X"), marksplice.TableAlignmentDefault, [][]byte{[]byte("x")}},
		{1, []byte("X"), marksplice.TableAlignmentDefault, nil},
		{1, []byte("X"), marksplice.TableAlignment(255), [][]byte{[]byte("x")}},
		{1, []byte("bad | split"), marksplice.TableAlignmentDefault, [][]byte{[]byte("x")}},
		{1, []byte("X"), marksplice.TableAlignmentDefault, [][]byte{[]byte("bad\nline")}},
	}
	for _, test := range tests {
		if _, err := doc.PrepareInsertTableColumn(table.ID(), test.column, test.header, test.alignment, test.body); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareInsertTableColumn(%d, %q, %d, %q) error = %v, want ErrInvalidReplacement", test.column, test.header, test.alignment, test.body, err)
		}
	}

	incomplete := []byte("| A | B |\n| --- | --- |\nTail\n\nAfter\n")
	incompleteDoc, err := marksplice.Parse(incomplete)
	if err != nil {
		t.Fatalf("Parse(incomplete) error = %v", err)
	}
	incompleteTable := publicTables(t, incompleteDoc)[0]
	if _, err := incompleteDoc.PrepareInsertTableColumn(incompleteTable.ID(), 1, []byte("X"), marksplice.TableAlignmentLeft, [][]byte{[]byte("x")}); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("incomplete table insertion error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareRemoveTableColumnPreservesRemainingSourceTrivia(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B | C |\r\n| :----- | ---: | :----: |\r\n| one | two | three |\r\n| | middle | |\r\n\r\nTail\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	table := publicTables(t, doc)[0]
	change, err := doc.PrepareRemoveTableColumn(table.ID(), 1)
	if err != nil {
		t.Fatalf("PrepareRemoveTableColumn() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("| A | C |\r\n| :----- | :----: |\r\n| one | three |\r\n| | |\r\n\r\nTail\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}

	reparsed, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updated := publicTables(t, reparsed)[0]
	if updated.ColumnCount() != 2 || updated.BodyRowCount() != 2 {
		t.Fatalf("updated table counts = columns %d rows %d, want 2/2", updated.ColumnCount(), updated.BodyRowCount())
	}
	alignments, ok := reparsed.TableAlignments(updated.ID())
	wantAlignments := []marksplice.TableAlignment{marksplice.TableAlignmentLeft, marksplice.TableAlignmentCenter}
	if !ok || !slices.Equal(alignments, wantAlignments) {
		t.Fatalf("updated alignments = %v/%v, want %v/true", alignments, ok, wantAlignments)
	}
}

func TestPrepareRemoveTableColumnPreservesFollowingTableExactly(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B | C |\n| --- | --- | --- |\n| one | two | three |\n\n| X | Y |\n| :---- | ---: |\n| left | right |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	tables := publicTables(t, doc)
	if len(tables) != 2 {
		t.Fatalf("table count = %d, want 2", len(tables))
	}
	secondBefore, ok := doc.SourceRange(tables[1].Range())
	if !ok {
		t.Fatal("SourceRange(second table) ok = false")
	}
	change, err := doc.PrepareRemoveTableColumn(tables[0].ID(), 1)
	if err != nil {
		t.Fatalf("PrepareRemoveTableColumn() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	reparsed, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updatedTables := publicTables(t, reparsed)
	if len(updatedTables) != 2 {
		t.Fatalf("updated table count = %d, want 2", len(updatedTables))
	}
	secondAfter, ok := reparsed.SourceRange(updatedTables[1].Range())
	if !ok || !bytes.Equal(secondAfter, secondBefore) {
		t.Fatalf("following table = %q/%v, want exact %q/true", secondAfter, ok, secondBefore)
	}
}

func TestPrepareRemoveTableColumnFailsClosedForIncompleteOrSingleColumnTable(t *testing.T) {
	t.Parallel()

	incomplete := []byte("| A | B |\n| --- | --- |\nTail\n\nAfter\n")
	doc, err := marksplice.Parse(incomplete)
	if err != nil {
		t.Fatalf("Parse(incomplete) error = %v", err)
	}
	table := publicTables(t, doc)[0]
	if _, err := doc.PrepareRemoveTableColumn(table.ID(), 0); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("incomplete table removal error = %v, want ErrInvalidReplacement", err)
	}

	single := []byte("| A |\n| --- |\n| one |\n")
	singleDoc, err := marksplice.Parse(single)
	if err != nil {
		t.Fatalf("Parse(single) error = %v", err)
	}
	singleTable := publicTables(t, singleDoc)[0]
	if _, err := singleDoc.PrepareRemoveTableColumn(singleTable.ID(), 0); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("single-column removal error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := singleDoc.PrepareRemoveTableColumn(singleTable.ID(), 1); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("out-of-range removal error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareMoveTableColumnReordersExactCellSource(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B | C |\n| :----- | ---: | :----: |\n| one | two | three |\n| alpha | beta | gamma |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	table := publicTables(t, doc)[0]
	change, err := doc.PrepareMoveTableColumn(table.ID(), 2, 0)
	if err != nil {
		t.Fatalf("PrepareMoveTableColumn() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("| C | A | B |\n| :----: | :----- | ---: |\n| three | one | two |\n| gamma | alpha | beta |\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}

	reparsed, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updated := publicTables(t, reparsed)[0]
	alignments, ok := reparsed.TableAlignments(updated.ID())
	wantAlignments := []marksplice.TableAlignment{marksplice.TableAlignmentCenter, marksplice.TableAlignmentLeft, marksplice.TableAlignmentRight}
	if !ok || !slices.Equal(alignments, wantAlignments) {
		t.Fatalf("updated alignments = %v/%v, want %v/true", alignments, ok, wantAlignments)
	}
}

func TestPrepareMoveTableColumnNoOpAndIncompleteTableFailClosed(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B |\n| --- | --- |\n| one | two |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	table := publicTables(t, doc)[0]
	change, err := doc.PrepareMoveTableColumn(table.ID(), 1, 1)
	if err != nil {
		t.Fatalf("PrepareMoveTableColumn(no-op) error = %v", err)
	}
	unchanged, err := change.Apply(source)
	if err != nil || !bytes.Equal(unchanged, source) {
		t.Fatalf("Apply(no-op) = %q, %v; want original, nil", unchanged, err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale no-op) error = %v, want ErrSourceConflict", err)
	}
	for _, indexes := range [][2]int{{-1, 0}, {0, -1}, {2, 0}, {0, 2}} {
		if _, err := doc.PrepareMoveTableColumn(table.ID(), indexes[0], indexes[1]); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("move %v error = %v, want ErrInvalidReplacement", indexes, err)
		}
	}

	incomplete := []byte("| A | B |\n| --- | --- |\nTail\n\nAfter\n")
	incompleteDoc, err := marksplice.Parse(incomplete)
	if err != nil {
		t.Fatalf("Parse(incomplete) error = %v", err)
	}
	incompleteTable := publicTables(t, incompleteDoc)[0]
	if _, err := incompleteDoc.PrepareMoveTableColumn(incompleteTable.ID(), 0, 1); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("incomplete table move error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareAppendTableRowFailsClosedWithoutExistingLineBoundaryOrCompatibleShape(t *testing.T) {
	t.Parallel()

	eofSource := []byte("| A | B |\n| --- | --- |")
	eofDoc, err := marksplice.Parse(eofSource)
	if err != nil {
		t.Fatalf("Parse(EOF) error = %v", err)
	}
	eofTable := publicTables(t, eofDoc)[0]
	if _, err := eofDoc.PrepareAppendTableRow(eofTable.ID(), []byte("| one | two |")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("append without line boundary error = %v, want ErrInvalidReplacement", err)
	}

	source := []byte("| A | B |\n| --- | --- |\n\nTail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	table := publicTables(t, doc)[0]
	for _, fragment := range [][]byte{nil, []byte("| only one |\n"), []byte("| one | two |\n| extra | row |\n")} {
		if _, err := doc.PrepareAppendTableRow(table.ID(), fragment); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("PrepareAppendTableRow(%q) error = %v, want ErrInvalidReplacement", fragment, err)
		}
	}
}
