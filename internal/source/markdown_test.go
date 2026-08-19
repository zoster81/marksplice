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
		wantContent Range
	}{
		{
			name:        "unordered with indentation and CRLF",
			source:      []byte("  *   item  \r\n"),
			content:     Range{Start: 6, End: 12},
			ordered:     false,
			marker:      '*',
			wantRange:   Range{Start: 2, End: 12},
			wantContent: Range{Start: 6, End: 12},
		},
		{
			name:        "ordered after container prefix",
			source:      []byte("> 12)  item\n"),
			content:     Range{Start: 7, End: 11},
			ordered:     true,
			marker:      ')',
			wantRange:   Range{Start: 2, End: 11},
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
			if got.Range != tt.wantRange || got.ContentRange != tt.wantContent || got.Ordered != tt.ordered || got.Marker != tt.marker {
				t.Fatalf("mapping = %+v, want range %v content %v ordered %v marker %q", got, tt.wantRange, tt.wantContent, tt.ordered, tt.marker)
			}
		})
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

func TestMapSingleLineFencedCodePreservesFenceBoundaries(t *testing.T) {
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

			got, err := MapSingleLineFencedCode(tt.source, tt.content)
			if err != nil {
				t.Fatalf("MapSingleLineFencedCode() error = %v", err)
			}
			if got.Range != tt.wantRange || got.ContentRange != tt.content || got.InfoRange != tt.wantInfo ||
				got.FenceChar != tt.wantChar || got.FenceLength != tt.wantFenceLength || got.ClosingFenceLength != tt.wantClosingLength ||
				got.OpeningIndent != tt.wantOpeningIndent || got.ClosingIndent != tt.wantClosingIndent {
				t.Fatalf("mapping = %+v, want range %v content %v info %v char %q lengths %d/%d indents %d/%d", got, tt.wantRange, tt.content, tt.wantInfo, tt.wantChar, tt.wantFenceLength, tt.wantClosingLength, tt.wantOpeningIndent, tt.wantClosingIndent)
			}
		})
	}
}

func TestMapSingleLineFencedCodeRejectsUnprovenShape(t *testing.T) {
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
			name:    "semantic range crosses line ending",
			source:  []byte("```\none\ntwo\n```\n"),
			content: Range{Start: 4, End: 11},
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

			_, err := MapSingleLineFencedCode(tt.source, tt.content)
			if !errors.Is(err, ErrUnsupportedFencedCodeShape) {
				t.Fatalf("MapSingleLineFencedCode() error = %v, want ErrUnsupportedFencedCodeShape", err)
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
