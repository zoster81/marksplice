package source

import (
	"errors"
	"slices"
	"testing"
)

func TestMapTableOwnsCompleteCRLFSpanAndDelimiterAlignments(t *testing.T) {
	t.Parallel()

	input := []byte(" | A | B |\r\n | :--- | ---: |\r\n | only one |\r\nTail\r\n")
	lastBodyAnchor := len(" | A | B |\r\n | :--- | ---: |\r\n")
	mapping, err := MapTable(input, 0, 1, lastBodyAnchor)
	if err != nil {
		t.Fatalf("MapTable() error = %v", err)
	}
	wantEnd := len(" | A | B |\r\n | :--- | ---: |\r\n | only one |\r\n")
	if mapping.Range != (Range{Start: 0, End: wantEnd}) {
		t.Fatalf("Range = %+v, want [0,%d)", mapping.Range, wantEnd)
	}
	wantAlignments := []TableDelimiterAlignment{TableDelimiterAlignmentLeft, TableDelimiterAlignmentRight}
	if !slices.Equal(mapping.DelimiterAlignments, wantAlignments) {
		t.Fatalf("delimiter alignments = %v, want %v", mapping.DelimiterAlignments, wantAlignments)
	}
	if len(mapping.Header.Cells) != 2 || len(mapping.Delimiter.Cells) != 2 {
		t.Fatalf("header/delimiter cell counts = %d/%d, want 2/2", len(mapping.Header.Cells), len(mapping.Delimiter.Cells))
	}
}

func TestMapTableWithoutBodyStopsAfterDelimiter(t *testing.T) {
	t.Parallel()

	input := []byte("| A |\n| --- |\nTail\n")
	mapping, err := MapTable(input, 0, 0, 0)
	if err != nil {
		t.Fatalf("MapTable() error = %v", err)
	}
	wantEnd := len("| A |\n| --- |\n")
	if mapping.Range != (Range{Start: 0, End: wantEnd}) {
		t.Fatalf("Range = %+v, want [0,%d)", mapping.Range, wantEnd)
	}
}

func TestTableDelimiterAlignmentReplacementPreservesDashRun(t *testing.T) {
	t.Parallel()

	input := []byte(":-----:")
	content := Range{Start: 0, End: len(input)}
	tests := []struct {
		alignment TableDelimiterAlignment
		want      string
	}{
		{TableDelimiterAlignmentDefault, "-----"},
		{TableDelimiterAlignmentLeft, ":-----"},
		{TableDelimiterAlignmentRight, "-----:"},
		{TableDelimiterAlignmentCenter, ":-----:"},
	}
	for _, test := range tests {
		got, err := TableDelimiterAlignmentReplacement(input, content, test.alignment)
		if err != nil {
			t.Fatalf("TableDelimiterAlignmentReplacement(%d) error = %v", test.alignment, err)
		}
		if string(got) != test.want {
			t.Fatalf("TableDelimiterAlignmentReplacement(%d) = %q, want %q", test.alignment, got, test.want)
		}
	}
	if _, err := TableDelimiterAlignmentReplacement(input, content, TableDelimiterAlignment(255)); !errors.Is(err, ErrUnsupportedTableShape) {
		t.Fatalf("invalid alignment error = %v, want ErrUnsupportedTableShape", err)
	}
}

func TestMapTableRejectsInvalidDelimiterAndBodyAnchor(t *testing.T) {
	t.Parallel()

	if _, err := MapTable([]byte("| A |\n| :x |\n"), 0, 0, 0); !errors.Is(err, ErrUnsupportedTableShape) {
		t.Fatalf("invalid delimiter error = %v, want ErrUnsupportedTableShape", err)
	}
	input := []byte("| A |\n| --- |\n| body |\n")
	if _, err := MapTable(input, 0, 1, len("| A |\n| --- |\n")+1); !errors.Is(err, ErrUnsupportedTableShape) {
		t.Fatalf("mid-line body anchor error = %v, want ErrUnsupportedTableShape", err)
	}
}
