package source

import (
	"bytes"
	"errors"
	"slices"
	"testing"
)

func TestMapTopLevelFootnoteDefinitionOwnsCompletePhysicalSource(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n[^note]: first line\r\n\r\n    second paragraph\r\n\r\nafter\r\n")
	anchor := bytes.Index(source, []byte("[^note]:"))
	first := bytes.Index(source, []byte("first line"))
	second := bytes.Index(source, []byte("second paragraph"))
	semantic := []Range{
		{Start: first, End: first + len("first line")},
		{Start: second, End: second + len("second paragraph")},
	}

	got, err := MapTopLevelFootnoteDefinition(source, anchor, "note", semantic)
	if err != nil {
		t.Fatalf("MapTopLevelFootnoteDefinition() error = %v", err)
	}
	wantEnd := bytes.Index(source, []byte("after"))
	if got.Range != (Range{Start: anchor, End: wantEnd}) {
		t.Fatalf("Range = %v, want [%d,%d)", got.Range, anchor, wantEnd)
	}
	if got.LabelRange != (Range{Start: anchor + 2, End: anchor + 6}) {
		t.Fatalf("LabelRange = %v", got.LabelRange)
	}
	if got.BodyRange != (Range{}) {
		t.Fatalf("BodyRange = %v, want zero for segmented multiline body", got.BodyRange)
	}
	if !slices.Equal(got.BodyRanges, semantic) {
		t.Fatalf("BodyRanges = %v, want %v", got.BodyRanges, semantic)
	}
}

func TestMapTopLevelFootnoteDefinitionExposesSimpleBodyRange(t *testing.T) {
	t.Parallel()

	source := []byte("  [^x]: note text\nnext\n")
	anchor := bytes.Index(source, []byte("[^x]:"))
	bodyStart := bytes.Index(source, []byte("note text"))
	body := Range{Start: bodyStart, End: bodyStart + len("note text")}
	got, err := MapTopLevelFootnoteDefinition(source, anchor, "x", []Range{body})
	if err != nil {
		t.Fatalf("MapTopLevelFootnoteDefinition() error = %v", err)
	}
	if got.Range != (Range{Start: 0, End: len("  [^x]: note text\n")}) || got.BodyRange != body {
		t.Fatalf("mapping = %+v", got)
	}
}

func TestMapTopLevelFootnoteDefinitionFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   []byte
		anchor   int
		label    string
		semantic []Range
	}{
		{name: "wrong label", source: []byte("[^a]: note\n"), anchor: 0, label: "b"},
		{name: "container prefix", source: []byte("> [^a]: note\n"), anchor: 2, label: "a"},
		{name: "semantic outside owner", source: []byte("[^a]: note\nafter\n"), anchor: 0, label: "a", semantic: []Range{{Start: len("[^a]: note\n"), End: len("[^a]: note\nafter")}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := MapTopLevelFootnoteDefinition(tt.source, tt.anchor, tt.label, tt.semantic)
			if !errors.Is(err, ErrUnsupportedFootnoteShape) {
				t.Fatalf("error = %v, want ErrUnsupportedFootnoteShape", err)
			}
		})
	}
}
