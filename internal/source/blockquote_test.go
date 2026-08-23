package source

import (
	"bytes"
	"errors"
	"testing"
)

func TestMapSimpleTopLevelBlockquoteOwnsExactIndentedCRLFLine(t *testing.T) {
	t.Parallel()

	input := []byte("  > quoted  \r\nnext\r\n")
	mapping, err := MapSimpleTopLevelBlockquote(input, Range{Start: 2, End: 12}, Range{Start: 4, End: 12})
	if err != nil {
		t.Fatalf("MapSimpleTopLevelBlockquote() error = %v", err)
	}
	if mapping.Range != (Range{Start: 2, End: 12}) || mapping.ContentRange != (Range{Start: 4, End: 12}) ||
		mapping.MarkerRange != (Range{Start: 2, End: 3}) || mapping.LineRange != (Range{Start: 0, End: 14}) {
		t.Fatalf("mapping = %+v, want exact observed/content/marker/line ranges", mapping)
	}
	if got := input[mapping.LineRange.Start:mapping.LineRange.End]; !bytes.Equal(got, []byte("  > quoted  \r\n")) {
		t.Fatalf("line = %q", got)
	}
}

func TestMapSimpleTopLevelBlockquoteSupportsNoMarkerSpace(t *testing.T) {
	t.Parallel()

	input := []byte(">quoted\n")
	mapping, err := MapSimpleTopLevelBlockquote(input, Range{Start: 0, End: 7}, Range{Start: 1, End: 7})
	if err != nil {
		t.Fatalf("MapSimpleTopLevelBlockquote() error = %v", err)
	}
	if mapping.MarkerRange != (Range{Start: 0, End: 1}) || mapping.ContentRange != (Range{Start: 1, End: 7}) || mapping.LineRange != (Range{Start: 0, End: 8}) {
		t.Fatalf("mapping = %+v", mapping)
	}
}

func TestValidateCanonicalBlockquoteParagraphAcceptsMultilineSource(t *testing.T) {
	t.Parallel()

	input := []byte("> first\n> second π\n")
	first := Range{Start: 2, End: 7}
	second := Range{Start: 10, End: 19}
	outer := Range{Start: 0, End: 19}
	if err := ValidateCanonicalBlockquoteParagraph(input, outer, []Range{first, second}); err != nil {
		t.Fatalf("ValidateCanonicalBlockquoteParagraph() error = %v", err)
	}
}

func TestValidateCanonicalNestedBlockquoteParagraphAcceptsExactDepth(t *testing.T) {
	t.Parallel()

	input := []byte("> > first\n> > second π\n")
	first := Range{Start: 4, End: 9}
	second := Range{Start: 14, End: 23}
	outer := Range{Start: 0, End: 23}
	if err := ValidateCanonicalNestedBlockquoteParagraph(input, outer, []Range{first, second}, 2); err != nil {
		t.Fatalf("ValidateCanonicalNestedBlockquoteParagraph() error = %v", err)
	}
}

func TestValidateCanonicalNestedBlockquoteParagraphRejectsDepthMismatch(t *testing.T) {
	t.Parallel()

	input := []byte("> > > text\n")
	outer := Range{Start: 0, End: 10}
	content := []Range{{Start: 6, End: 10}}
	if err := ValidateCanonicalNestedBlockquoteParagraph(input, outer, content, 2); !errors.Is(err, ErrUnsupportedBlockquoteShape) {
		t.Fatalf("error = %v, want ErrUnsupportedBlockquoteShape", err)
	}
}

func TestValidateCanonicalNestedBlockquoteBlocksAcceptsExactSource(t *testing.T) {
	t.Parallel()

	inner := []byte("## Head\n\nfirst\nsecond π\n\n---\n\n```go\nx\n```\n")
	input := []byte("> > ## Head\n> > \n> > first\n> > second π\n> > \n> > ---\n> > \n> > ```go\n> > x\n> > ```\n")
	outer := Range{Start: 0, End: len(input) - 1}
	if err := ValidateCanonicalNestedBlockquoteBlocks(input, outer, inner, 2); err != nil {
		t.Fatalf("ValidateCanonicalNestedBlockquoteBlocks() error = %v", err)
	}
}

func TestValidateCanonicalNestedBlockquoteBlocksRejectsChangedSourceAndPathologicalDepth(t *testing.T) {
	t.Parallel()

	inner := []byte("first\n\nsecond\n")
	changed := []byte("> > first\n> > \n> > changed\n")
	outer := Range{Start: 0, End: len(changed) - 1}
	if err := ValidateCanonicalNestedBlockquoteBlocks(changed, outer, inner, 2); !errors.Is(err, ErrUnsupportedBlockquoteShape) {
		t.Fatalf("changed source error = %v, want ErrUnsupportedBlockquoteShape", err)
	}
	if err := ValidateCanonicalNestedBlockquoteBlocks([]byte("> x\n"), Range{Start: 0, End: 3}, []byte("x\n"), int(^uint(0)>>1)); !errors.Is(err, ErrUnsupportedBlockquoteShape) {
		t.Fatalf("pathological depth error = %v, want ErrUnsupportedBlockquoteShape", err)
	}
}

func TestValidateCanonicalBlockquoteParagraphRejectsChangedCanonicalLayout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		outer Range
		lines []Range
	}{
		{name: "missing second marker space", input: []byte("> first\n>second\n"), outer: Range{Start: 0, End: 15}, lines: []Range{{Start: 2, End: 7}, {Start: 9, End: 15}}},
		{name: "missing interline LF", input: []byte("> first> second\n"), outer: Range{Start: 0, End: 15}, lines: []Range{{Start: 2, End: 7}, {Start: 9, End: 15}}},
		{name: "empty content line", input: []byte("> first\n> \n"), outer: Range{Start: 0, End: 10}, lines: []Range{{Start: 2, End: 7}, {Start: 10, End: 10}}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateCanonicalBlockquoteParagraph(tt.input, tt.outer, tt.lines); !errors.Is(err, ErrUnsupportedBlockquoteShape) {
				t.Fatalf("error = %v, want ErrUnsupportedBlockquoteShape", err)
			}
		})
	}
}

func TestMapSimpleTopLevelBlockquoteRejectsUnsupportedPrefixes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []byte
		observed Range
		content  Range
	}{
		{name: "four leading spaces", input: []byte("    > quote\n"), observed: Range{Start: 4, End: 11}, content: Range{Start: 6, End: 11}},
		{name: "tab indentation", input: []byte("\t> quote\n"), observed: Range{Start: 1, End: 8}, content: Range{Start: 3, End: 8}},
		{name: "two marker spaces excluded", input: []byte(">  quote\n"), observed: Range{Start: 0, End: 8}, content: Range{Start: 3, End: 8}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := MapSimpleTopLevelBlockquote(tt.input, tt.observed, tt.content); !errors.Is(err, ErrUnsupportedBlockquoteShape) {
				t.Fatalf("error = %v, want ErrUnsupportedBlockquoteShape", err)
			}
		})
	}
}
