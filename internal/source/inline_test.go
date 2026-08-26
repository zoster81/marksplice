package source

import (
	"errors"
	"testing"
)

func TestMapSimpleCodeSpanPreservesFenceBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    []byte
		anchor    int
		content   Range
		wantRange Range
		wantFence int
	}{
		{name: "single backtick", source: []byte("before `old` after\r\n"), anchor: 7, content: Range{Start: 8, End: 11}, wantRange: Range{Start: 7, End: 12}, wantFence: 1},
		{name: "double backtick with inner backtick", source: []byte("before ``old`code`` after\n"), anchor: 7, content: Range{Start: 9, End: 17}, wantRange: Range{Start: 7, End: 19}, wantFence: 2},
		{name: "line start after LF", source: []byte("prev`\n`old`\n"), anchor: 6, content: Range{Start: 7, End: 10}, wantRange: Range{Start: 6, End: 11}, wantFence: 1},
		{name: "line start after CRLF", source: []byte("prev`\r\n`old`\r\n"), anchor: 7, content: Range{Start: 8, End: 11}, wantRange: Range{Start: 7, End: 12}, wantFence: 1},
		{name: "line start after CR", source: []byte("prev`\r`old`\r"), anchor: 6, content: Range{Start: 7, End: 10}, wantRange: Range{Start: 6, End: 11}, wantFence: 1},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := MapSimpleCodeSpan(tt.source, tt.anchor, tt.content)
			if err != nil {
				t.Fatalf("MapSimpleCodeSpan() error = %v", err)
			}
			if got.Range != tt.wantRange || got.ContentRange != tt.content || got.FenceLength != tt.wantFence {
				t.Fatalf("mapping = %+v, want range %v content %v fence %d", got, tt.wantRange, tt.content, tt.wantFence)
			}
		})
	}
}

func TestMapSimpleCodeSpanRejectsNormalizedOrMismatchedShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		anchor  int
		content Range
	}{
		{name: "Goldmark-normalized surrounding spaces", source: []byte("` old `\n"), anchor: 0, content: Range{Start: 2, End: 5}},
		{name: "mismatched closing run", source: []byte("``old`\n"), anchor: 0, content: Range{Start: 2, End: 5}},
		{name: "content crosses line", source: []byte("`one\ntwo`\n"), anchor: 0, content: Range{Start: 1, End: 8}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := MapSimpleCodeSpan(tt.source, tt.anchor, tt.content)
			if !errors.Is(err, ErrUnsupportedCodeSpanShape) {
				t.Fatalf("MapSimpleCodeSpan() error = %v, want ErrUnsupportedCodeSpanShape", err)
			}
		})
	}
}

func TestMapSimpleEmphasisPreservesMarkerAndLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     []byte
		anchor     int
		content    Range
		level      int
		wantRange  Range
		wantMarker byte
	}{
		{name: "asterisk emphasis", source: []byte("before *old* after\r\n"), anchor: 7, content: Range{Start: 8, End: 11}, level: 1, wantRange: Range{Start: 7, End: 12}, wantMarker: '*'},
		{name: "underscore strong", source: []byte("before __old__ after\n"), anchor: 7, content: Range{Start: 9, End: 12}, level: 2, wantRange: Range{Start: 7, End: 14}, wantMarker: '_'},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := MapSimpleEmphasis(tt.source, tt.anchor, tt.content, tt.level)
			if err != nil {
				t.Fatalf("MapSimpleEmphasis() error = %v", err)
			}
			if got.Range != tt.wantRange || got.ContentRange != tt.content || got.Marker != tt.wantMarker || got.Level != tt.level {
				t.Fatalf("mapping = %+v, want range %v content %v marker %q level %d", got, tt.wantRange, tt.content, tt.wantMarker, tt.level)
			}
		})
	}
}

func TestMapSimpleEmphasisRejectsCompoundOrAsymmetricRuns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		anchor  int
		content Range
		level   int
	}{
		{name: "triple run is compound", source: []byte("***old***\n"), anchor: 0, content: Range{Start: 3, End: 6}, level: 2},
		{name: "asymmetric closing run", source: []byte("*old**\n"), anchor: 0, content: Range{Start: 1, End: 4}, level: 1},
		{name: "content crosses line", source: []byte("*one\ntwo*\n"), anchor: 0, content: Range{Start: 1, End: 8}, level: 1},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := MapSimpleEmphasis(tt.source, tt.anchor, tt.content, tt.level)
			if !errors.Is(err, ErrUnsupportedEmphasisShape) {
				t.Fatalf("MapSimpleEmphasis() error = %v, want ErrUnsupportedEmphasisShape", err)
			}
		})
	}
}

func TestMapSimpleStrikethroughPreservesDelimiterBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		source        []byte
		content       Range
		wantRange     Range
		wantDelimiter int
	}{
		{
			name:          "single tilde",
			source:        []byte("before ~old~ after\r\n"),
			content:       Range{Start: 8, End: 11},
			wantRange:     Range{Start: 7, End: 12},
			wantDelimiter: 1,
		},
		{
			name:          "double tilde",
			source:        []byte("xx ~~old~~ yy\n"),
			content:       Range{Start: 5, End: 8},
			wantRange:     Range{Start: 3, End: 10},
			wantDelimiter: 2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MapSimpleStrikethrough(tt.source, tt.content)
			if err != nil {
				t.Fatalf("MapSimpleStrikethrough() error = %v", err)
			}
			if got.Range != tt.wantRange || got.ContentRange != tt.content || got.DelimiterLength != tt.wantDelimiter {
				t.Fatalf("mapping = %+v, want range %v content %v delimiter length %d", got, tt.wantRange, tt.content, tt.wantDelimiter)
			}
		})
	}
}

func TestMapSimpleStrikethroughRejectsUnprovenShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		content Range
	}{
		{
			name:    "three-tilde run",
			source:  []byte("~~~old~~~\n"),
			content: Range{Start: 3, End: 6},
		},
		{
			name:    "asymmetric delimiters",
			source:  []byte("~~old~\n"),
			content: Range{Start: 2, End: 5},
		},
		{
			name:    "content crosses physical line",
			source:  []byte("~~one\ntwo~~\n"),
			content: Range{Start: 2, End: 9},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := MapSimpleStrikethrough(tt.source, tt.content)
			if !errors.Is(err, ErrUnsupportedStrikethroughShape) {
				t.Fatalf("MapSimpleStrikethrough() error = %v, want ErrUnsupportedStrikethroughShape", err)
			}
		})
	}
}
