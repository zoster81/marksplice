package source

import (
	"bytes"
	"errors"
	"testing"
)

func TestTableColumnInsertionClonesLocalSlotTrivia(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		column  int
		content string
		want    string
	}{
		{name: "middle uneven padding", input: "|A  | B |", column: 1, content: "X", want: "|A  | X | B |"},
		{name: "append single column", input: "| A |", column: 1, content: "B", want: "| A | B |"},
		{name: "prepend", input: "| A | B |", column: 0, content: "Z", want: "| Z | A | B |"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			input := []byte(test.input)
			row, err := MapTableRow(input, 0)
			if err != nil {
				t.Fatalf("MapTableRow() error = %v", err)
			}
			range_, replacement, err := TableColumnInsertion(input, row, test.column, []byte(test.content))
			if err != nil {
				t.Fatalf("TableColumnInsertion() error = %v", err)
			}
			got := append([]byte(nil), input[:range_.Start]...)
			got = append(got, replacement...)
			got = append(got, input[range_.End:]...)
			if string(got) != test.want {
				t.Fatalf("inserted row = %q, want %q", got, test.want)
			}
		})
	}

	input := []byte("| A | B |")
	row, err := MapTableRow(input, 0)
	if err != nil {
		t.Fatalf("MapTableRow() error = %v", err)
	}
	for _, invalid := range []struct {
		column  int
		content []byte
	}{
		{column: -1, content: []byte("X")},
		{column: 3, content: []byte("X")},
		{column: 1, content: []byte("bad\nline")},
	} {
		if _, _, err := TableColumnInsertion(input, row, invalid.column, invalid.content); !errors.Is(err, ErrUnsupportedTableShape) {
			t.Fatalf("TableColumnInsertion(%d, %q) error = %v, want ErrUnsupportedTableShape", invalid.column, invalid.content, err)
		}
	}
}

func TestTableColumnRemovalRangePreservesOuterPipeStyle(t *testing.T) {
	t.Parallel()

	input := []byte("| A | B | C |")
	row, err := MapTableRow(input, 0)
	if err != nil {
		t.Fatalf("MapTableRow() error = %v", err)
	}
	want := []string{"| B | C |", "| A | C |", "| A | B |"}
	for column := range want {
		range_, err := TableColumnRemovalRange(row, column)
		if err != nil {
			t.Fatalf("TableColumnRemovalRange(%d) error = %v", column, err)
		}
		got := append([]byte(nil), input[:range_.Start]...)
		got = append(got, input[range_.End:]...)
		if string(got) != want[column] {
			t.Fatalf("column %d removal = %q, want %q", column, got, want[column])
		}
	}
}

func TestReorderTableRowColumnsKeepsDestinationSlotTrivia(t *testing.T) {
	t.Parallel()

	input := []byte("|A  | B |   C|")
	row, err := MapTableRow(input, 0)
	if err != nil {
		t.Fatalf("MapTableRow() error = %v", err)
	}
	got, err := ReorderTableRowColumns(input, row, []int{2, 0, 1})
	if err != nil {
		t.Fatalf("ReorderTableRowColumns() error = %v", err)
	}
	want := "|C  | A |   B|"
	if string(got) != want {
		t.Fatalf("reordered row = %q, want %q", got, want)
	}
	if _, err := ReorderTableRowColumns(input, row, []int{0, 0, 2}); !errors.Is(err, ErrUnsupportedTableShape) {
		t.Fatalf("duplicate permutation error = %v, want ErrUnsupportedTableShape", err)
	}
}

func TestMapCompleteTableRowsRequiresEverySemanticBodyRowToBeSourceMapped(t *testing.T) {
	t.Parallel()

	complete := []byte("| A | B |\n| --- | --- |\n| one | two |\n| three | four |\n")
	lastAnchor := bytes.Index(complete, []byte("| three"))
	table, err := MapTable(complete, 0, 2, lastAnchor)
	if err != nil {
		t.Fatalf("MapTable(complete) error = %v", err)
	}
	rows, err := MapCompleteTableRows(complete, table, 2, 2)
	if err != nil {
		t.Fatalf("MapCompleteTableRows(complete) error = %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("complete row count = %d, want 4", len(rows))
	}

	incomplete := []byte("| A | B |\n| --- | --- |\nTail\n")
	bodyAnchor := bytes.Index(incomplete, []byte("Tail"))
	incompleteTable, err := MapTable(incomplete, 0, 1, bodyAnchor)
	if err != nil {
		t.Fatalf("MapTable(incomplete) error = %v", err)
	}
	if _, err := MapCompleteTableRows(incomplete, incompleteTable, 2, 1); !errors.Is(err, ErrUnsupportedTableShape) {
		t.Fatalf("MapCompleteTableRows(incomplete) error = %v, want ErrUnsupportedTableShape", err)
	}
}
