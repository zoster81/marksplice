package publictest

import (
	"slices"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicTableRowAlignments(t *testing.T) {
	t.Parallel()

	source := []byte("| A | B | C | D |\n| :--- | ---: | :---: | --- |\n| a | b | c | d |\n| e | f | g | h |\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	var rowIDs []marksplice.NodeID
	var nonRowID marksplice.NodeID
	for _, node := range doc.Nodes() {
		switch node.Kind() {
		case marksplice.KindTableRow:
			rowIDs = append(rowIDs, node.ID())
		case marksplice.KindTableCell:
			if nonRowID == (marksplice.NodeID{}) {
				nonRowID = node.ID()
			}
		}
	}
	if len(rowIDs) != 2 {
		t.Fatalf("table row count = %d, want 2", len(rowIDs))
	}

	want := []marksplice.TableAlignment{
		marksplice.TableAlignmentLeft,
		marksplice.TableAlignmentRight,
		marksplice.TableAlignmentCenter,
		marksplice.TableAlignmentDefault,
	}
	for _, rowID := range rowIDs {
		alignments, ok := doc.TableRowAlignments(rowID)
		if !ok {
			t.Fatalf("TableRowAlignments(%v) = false, want true", rowID)
		}
		if !slices.Equal(alignments, want) {
			t.Fatalf("TableRowAlignments(%v) = %v, want %v", rowID, alignments, want)
		}
	}

	first, ok := doc.TableRowAlignments(rowIDs[0])
	if !ok {
		t.Fatal("TableRowAlignments(first) = false, want true")
	}
	first[0] = marksplice.TableAlignmentDefault
	again, ok := doc.TableRowAlignments(rowIDs[0])
	if !ok || !slices.Equal(again, want) {
		t.Fatalf("TableRowAlignments(first) after caller mutation = %v/%v, want %v/true", again, ok, want)
	}

	if got, ok := doc.TableRowAlignments(nonRowID); ok || got != nil {
		t.Fatalf("TableRowAlignments(non-row) = %v/%v, want nil/false", got, ok)
	}
	if got, ok := doc.TableRowAlignments(marksplice.NodeID{}); ok || got != nil {
		t.Fatalf("TableRowAlignments(zero ID) = %v/%v, want nil/false", got, ok)
	}

	_ = map[marksplice.TableRow]struct{}{}
}

func TestPublicTableRowAlignmentsDefaultColumns(t *testing.T) {
	t.Parallel()

	doc, err := marksplice.Parse([]byte("| A | B |\n| --- | --- |\n| x | y |\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindTableRow {
			continue
		}
		alignments, ok := doc.TableRowAlignments(node.ID())
		want := []marksplice.TableAlignment{marksplice.TableAlignmentDefault, marksplice.TableAlignmentDefault}
		if !ok || !slices.Equal(alignments, want) {
			t.Fatalf("TableRowAlignments(default table) = %v/%v, want %v/true", alignments, ok, want)
		}
		return
	}
	t.Fatal("table row not found")
}
