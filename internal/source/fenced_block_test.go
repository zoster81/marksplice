package source

import (
	"errors"
	"testing"
)

func TestMapFencedBlockPreservesCompleteIndentedContainer(t *testing.T) {
	t.Parallel()

	source := []byte("  ~~~~  mermaid diagram  \n  graph TD\n  A-->B\n ~~~~~~   \n")
	firstStart := len("  ~~~~  mermaid diagram  \n  ")
	secondStart := len("  ~~~~  mermaid diagram  \n  graph TD\n  ")
	content := []Range{
		{Start: firstStart, End: firstStart + len("graph TD")},
		{Start: secondStart, End: secondStart + len("A-->B")},
	}

	got, err := MapFencedBlock(source, 2, content, "mermaid diagram")
	if err != nil {
		t.Fatalf("MapFencedBlock() error = %v", err)
	}
	wantOpening := Range{Start: 2, End: 6}
	closingStart := len("  ~~~~  mermaid diagram  \n  graph TD\n  A-->B\n ")
	wantClosing := Range{Start: closingStart, End: closingStart + 6}
	if got.Range != (Range{Start: 0, End: len(source)}) || got.OpeningFenceRange != wantOpening ||
		got.ClosingFenceRange != wantClosing || got.FenceChar != '~' || got.OpeningFenceLength != 4 ||
		got.ClosingFenceLength != 6 || got.OpeningIndent != 2 || got.ClosingIndent != 1 || !got.Closed {
		t.Fatalf("mapping = %+v", got)
	}
	if got.InfoRange != (Range{Start: 8, End: 23}) || string(source[got.InfoRange.Start:got.InfoRange.End]) != "mermaid diagram" {
		t.Fatalf("info range = %v source = %q", got.InfoRange, source[got.InfoRange.Start:got.InfoRange.End])
	}
	if len(got.ContentRanges) != len(content) || got.ContentRanges[0] != content[0] || got.ContentRanges[1] != content[1] {
		t.Fatalf("content ranges = %v, want %v", got.ContentRanges, content)
	}
}

func TestMapFencedBlockPreservesEmptyAndUnclosedContainers(t *testing.T) {
	t.Parallel()

	empty := []byte("```math\n```\n")
	mappedEmpty, err := MapFencedBlock(empty, 0, nil, "math")
	if err != nil {
		t.Fatalf("MapFencedBlock(empty) error = %v", err)
	}
	if !mappedEmpty.Closed || mappedEmpty.Range != (Range{Start: 0, End: len(empty)}) || len(mappedEmpty.ContentRanges) != 0 ||
		mappedEmpty.OpeningFenceRange != (Range{Start: 0, End: 3}) || mappedEmpty.ClosingFenceRange != (Range{Start: 8, End: 11}) {
		t.Fatalf("empty mapping = %+v", mappedEmpty)
	}

	unclosed := []byte("~~~ geojson\n{\"type\":\"Point\"}")
	payloadStart := len("~~~ geojson\n")
	mappedUnclosed, err := MapFencedBlock(unclosed, 0, []Range{{Start: payloadStart, End: len(unclosed)}}, "geojson")
	if err != nil {
		t.Fatalf("MapFencedBlock(unclosed) error = %v", err)
	}
	if mappedUnclosed.Closed || mappedUnclosed.Range != (Range{Start: 0, End: len(unclosed)}) || mappedUnclosed.ClosingFenceRange != (Range{}) ||
		mappedUnclosed.ClosingFenceLength != 0 || mappedUnclosed.ClosingIndent != 0 {
		t.Fatalf("unclosed mapping = %+v", mappedUnclosed)
	}
}

func TestMapFencedBlockRejectsSemanticSourceDisagreement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		anchor  int
		content []Range
		info    string
	}{
		{
			name:    "info mismatch",
			source:  []byte("```go\nbody\n```\n"),
			anchor:  0,
			content: []Range{{Start: len("```go\n"), End: len("```go\nbody")}},
			info:    "rust",
		},
		{
			name:    "payload line stops before physical end",
			source:  []byte("```\nbody\n```\n"),
			anchor:  0,
			content: []Range{{Start: len("```\n"), End: len("```\nbody") - 1}},
		},
		{
			name:    "anchor does not identify delimiter run",
			source:  []byte("  ```\nbody\n  ```\n"),
			anchor:  0,
			content: []Range{{Start: len("  ```\n"), End: len("  ```\nbody")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := MapFencedBlock(tt.source, tt.anchor, tt.content, tt.info)
			if !errors.Is(err, ErrUnsupportedFencedCodeShape) {
				t.Fatalf("MapFencedBlock() error = %v, want ErrUnsupportedFencedCodeShape", err)
			}
		})
	}
}
