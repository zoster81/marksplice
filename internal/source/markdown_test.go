package source

import (
	"errors"
	"testing"
)

func TestMapSingleLineListItemPreservesMarkerBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		content     Range
		ordered     bool
		marker      byte
		wantRange   Range
		wantLine    Range
		wantContent Range
	}{
		{
			name:        "unordered with indentation and CRLF",
			source:      []byte("  *   item  \r\n"),
			content:     Range{Start: 6, End: 12},
			ordered:     false,
			marker:      '*',
			wantRange:   Range{Start: 2, End: 12},
			wantLine:    Range{Start: 0, End: 14},
			wantContent: Range{Start: 6, End: 12},
		},
		{
			name:        "ordered after container prefix",
			source:      []byte("> 12)  item\n"),
			content:     Range{Start: 7, End: 11},
			ordered:     true,
			marker:      ')',
			wantRange:   Range{Start: 2, End: 11},
			wantLine:    Range{Start: 0, End: 12},
			wantContent: Range{Start: 7, End: 11},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MapSingleLineListItem(tt.source, tt.content, tt.ordered, tt.marker)
			if err != nil {
				t.Fatalf("MapSingleLineListItem() error = %v", err)
			}
			if got.Range != tt.wantRange || got.LineRange != tt.wantLine || got.ContentRange != tt.wantContent || got.Ordered != tt.ordered || got.Marker != tt.marker {
				t.Fatalf("mapping = %+v, want range %v line %v content %v ordered %v marker %q", got, tt.wantRange, tt.wantLine, tt.wantContent, tt.ordered, tt.marker)
			}
		})
	}
}

func TestMapTableRowMapsAllCellsWithOneRowBoundary(t *testing.T) {
	t.Parallel()

	source := []byte("before\n| alpha | beta  |\r\nafter\n")
	anchor := len("before\n")
	row, err := MapTableRow(source, anchor)
	if err != nil {
		t.Fatalf("MapTableRow() error = %v", err)
	}
	if row.Range != (Range{Start: anchor, End: anchor + len("| alpha | beta  |")}) {
		t.Fatalf("row range = %v, want physical table row", row.Range)
	}
	if row.LineRange != (Range{Start: anchor, End: anchor + len("| alpha | beta  |\r\n")}) {
		t.Fatalf("row line range = %v, want complete CRLF-owned physical row", row.LineRange)
	}
	if len(row.Cells) != 2 {
		t.Fatalf("row cell count = %d, want 2", len(row.Cells))
	}
	want := []TableCellMapping{
		{Range: Range{Start: anchor + 1, End: anchor + 8}, ContentRange: Range{Start: anchor + 2, End: anchor + 7}, Column: 0},
		{Range: Range{Start: anchor + 9, End: anchor + 16}, ContentRange: Range{Start: anchor + 10, End: anchor + 14}, Column: 1},
	}
	for i := range want {
		if row.Cells[i] != want[i] {
			t.Fatalf("row cell %d = %+v, want %+v", i, row.Cells[i], want[i])
		}
	}
}

func TestMapTableRowLineRangeAtEOFHasNoSyntheticTerminator(t *testing.T) {
	t.Parallel()

	source := []byte("| alpha | beta |")
	row, err := MapTableRow(source, 0)
	if err != nil {
		t.Fatalf("MapTableRow() error = %v", err)
	}
	want := Range{Start: 0, End: len(source)}
	if row.Range != want || row.LineRange != want {
		t.Fatalf("row ranges = %v/%v, want %v/%v", row.Range, row.LineRange, want, want)
	}
}

func TestMapTableCellPreservesRawCellBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		content     Range
		column      int
		wantRange   Range
		wantContent Range
	}{
		{
			name:        "outer pipes and CRLF",
			source:      []byte("| alpha | old value  |\r\n"),
			content:     Range{Start: 10, End: 19},
			column:      1,
			wantRange:   Range{Start: 9, End: 21},
			wantContent: Range{Start: 10, End: 19},
		},
		{
			name:        "no outer pipes",
			source:      []byte("Name   | Value  \n"),
			content:     Range{Start: 9, End: 14},
			column:      1,
			wantRange:   Range{Start: 8, End: 16},
			wantContent: Range{Start: 9, End: 14},
		},
		{
			name:        "escaped pipe is not delimiter",
			source:      []byte("| x | old \\| value |\n"),
			content:     Range{Start: 6, End: 18},
			column:      1,
			wantRange:   Range{Start: 5, End: 19},
			wantContent: Range{Start: 6, End: 18},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MapTableCell(tt.source, tt.content, tt.column)
			if err != nil {
				t.Fatalf("MapTableCell() error = %v", err)
			}
			if got.Range != tt.wantRange || got.ContentRange != tt.wantContent || got.Column != tt.column {
				t.Fatalf("mapping = %+v, want range %v content %v column %d", got, tt.wantRange, tt.wantContent, tt.column)
			}
		})
	}
}

func TestMapTableCellRejectsUnprovenShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		content Range
		column  int
	}{
		{
			name:    "semantic content does not match mapped column",
			source:  []byte("| a | b |\n"),
			content: Range{Start: 2, End: 3},
			column:  1,
		},
		{
			name:    "column outside row",
			source:  []byte("| a | b |\n"),
			content: Range{Start: 6, End: 7},
			column:  2,
		},
		{
			name:    "content crosses line ending",
			source:  []byte("| a | b |\n| c | d |\n"),
			content: Range{Start: 2, End: 15},
			column:  0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := MapTableCell(tt.source, tt.content, tt.column)
			if !errors.Is(err, ErrUnsupportedTableCellShape) {
				t.Fatalf("MapTableCell() error = %v, want ErrUnsupportedTableCellShape", err)
			}
		})
	}
}

func TestMapFencedCodePreservesFenceBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		source            []byte
		content           Range
		wantRange         Range
		wantInfo          Range
		wantChar          byte
		wantFenceLength   int
		wantClosingLength int
		wantOpeningIndent int
		wantClosingIndent int
	}{
		{
			name:              "backtick CRLF with info and longer closing fence",
			source:            []byte("  ````go meta  \r\n  old()\r\n   `````\t\r\n"),
			content:           Range{Start: 19, End: 24},
			wantRange:         Range{Start: 0, End: 35},
			wantInfo:          Range{Start: 6, End: 13},
			wantChar:          '`',
			wantFenceLength:   4,
			wantClosingLength: 5,
			wantOpeningIndent: 2,
			wantClosingIndent: 3,
		},
		{
			name:              "tilde LF with different closing indentation",
			source:            []byte("   ~~~~ rust\n   old\n  ~~~~~\n"),
			content:           Range{Start: 16, End: 19},
			wantRange:         Range{Start: 0, End: 27},
			wantInfo:          Range{Start: 8, End: 12},
			wantChar:          '~',
			wantFenceLength:   4,
			wantClosingLength: 5,
			wantOpeningIndent: 3,
			wantClosingIndent: 2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MapFencedCode(tt.source, tt.content)
			if err != nil {
				t.Fatalf("MapFencedCode() error = %v", err)
			}
			if got.Range != tt.wantRange || got.ContentRange != tt.content || got.InfoRange != tt.wantInfo ||
				got.FenceChar != tt.wantChar || got.FenceLength != tt.wantFenceLength || got.ClosingFenceLength != tt.wantClosingLength ||
				got.OpeningIndent != tt.wantOpeningIndent || got.ClosingIndent != tt.wantClosingIndent {
				t.Fatalf("mapping = %+v, want range %v content %v info %v char %q lengths %d/%d indents %d/%d", got, tt.wantRange, tt.content, tt.wantInfo, tt.wantChar, tt.wantFenceLength, tt.wantClosingLength, tt.wantOpeningIndent, tt.wantClosingIndent)
			}
		})
	}
}

func TestMapFencedCodePreservesMultilineBoundaries(t *testing.T) {
	t.Parallel()

	source := []byte("````go\r\nfirst\r\nsecond\r\n  `````\t\r\n")
	content := Range{Start: len("````go\r\n"), End: len("````go\r\nfirst\r\nsecond")}
	got, err := MapFencedCode(source, content)
	if err != nil {
		t.Fatalf("MapFencedCode(multiline) error = %v", err)
	}
	wantRange := Range{Start: 0, End: len(source) - len("\r\n")}
	wantInfo := Range{Start: 4, End: 6}
	if got.Range != wantRange || got.ContentRange != content || got.InfoRange != wantInfo ||
		got.FenceChar != '`' || got.FenceLength != 4 || got.ClosingFenceLength != 5 ||
		got.OpeningIndent != 0 || got.ClosingIndent != 2 {
		t.Fatalf("multiline mapping = %+v, want range %v content %v info %v char ` lengths 4/5 indents 0/2", got, wantRange, content, wantInfo)
	}
}

func TestMapFencedCodeRejectsUnprovenShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		content Range
	}{
		{
			name:    "unclosed fence",
			source:  []byte("```\nold\n"),
			content: Range{Start: 4, End: 7},
		},
		{
			name:    "indented multiline fence has non-contiguous semantic indentation",
			source:  []byte("  ```\n  one\n  two\n  ```\n"),
			content: Range{Start: 8, End: 17},
		},
		{
			name:    "semantic range stops before physical line end",
			source:  []byte("```\nold extra\n```\n"),
			content: Range{Start: 4, End: 7},
		},
		{
			name:    "closing fence uses different delimiter",
			source:  []byte("```\nold\n~~~\n"),
			content: Range{Start: 4, End: 7},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := MapFencedCode(tt.source, tt.content)
			if !errors.Is(err, ErrUnsupportedFencedCodeShape) {
				t.Fatalf("MapFencedCode() error = %v, want ErrUnsupportedFencedCodeShape", err)
			}
		})
	}
}

func TestMapSingleLineListItemRejectsUnprovenShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		content Range
		ordered bool
		marker  byte
	}{
		{
			name:    "content crosses line ending",
			source:  []byte("- first\n  second\n"),
			content: Range{Start: 2, End: 16},
			marker:  '-',
		},
		{
			name:    "semantic marker disagrees with source",
			source:  []byte("* item\n"),
			content: Range{Start: 2, End: 6},
			marker:  '-',
		},
		{
			name:    "ordered marker has too many digits",
			source:  []byte("1234567890. item\n"),
			content: Range{Start: 12, End: 16},
			ordered: true,
			marker:  '.',
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := MapSingleLineListItem(tt.source, tt.content, tt.ordered, tt.marker)
			if !errors.Is(err, ErrUnsupportedListItemShape) {
				t.Fatalf("MapSingleLineListItem() error = %v, want ErrUnsupportedListItemShape", err)
			}
		})
	}
}
