package native

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestParseBlocksLeafStructures(t *testing.T) {
	t.Parallel()

	fenced := []byte("  ~~~~ go test  \n  one\n    two\n  ~~~~\n")
	fencedBodyStart := bytes.Index(fenced, []byte("one"))
	fencedSecondLineStart := bytes.Index(fenced, []byte("    two")) + 2
	fencedSecondLineEnd := bytes.Index(fenced, []byte("\n  ~~~~"))

	tests := []struct {
		name   string
		source []byte
		want   []parser.Node
	}{
		{name: "empty", want: []parser.Node{}},
		{
			name:   "paragraphs preserve crlf offsets",
			source: []byte("first\r\nsecond\r\n\r\nthird\r\n"),
			want: []parser.Node{
				{Kind: parser.KindParagraph, Range: parser.Range{Start: 0, End: 13}, TopLevel: true},
				{Kind: parser.KindParagraph, Range: parser.Range{Start: 17, End: 22}, TopLevel: true},
			},
		},
		{
			name:   "atx heading",
			source: []byte("# Title\n"),
			want:   []parser.Node{{Kind: parser.KindHeading, Range: parser.Range{Start: 2, End: 7}, Level: 1, TopLevel: true}},
		},
		{
			name:   "empty atx headings are recognized but not promoted",
			source: []byte("## \n#\n### ###\n"),
			want:   []parser.Node{},
		},
		{
			name:   "setext heading",
			source: []byte("Title\n=====\n"),
			want:   []parser.Node{{Kind: parser.KindHeading, Range: parser.Range{Start: 0, End: 5}, Level: 1, TopLevel: true}},
		},
		{
			name:   "setext multiline range excludes trailing horizontal trivia",
			source: []byte("  Foo *bar\nbaz*\t\n====\n"),
			want:   []parser.Node{{Kind: parser.KindHeading, Range: parser.Range{Start: 2, End: 15}, Level: 1, TopLevel: true}},
		},
		{
			name:   "paragraph interrupted by atx heading",
			source: []byte("text\n# head\n"),
			want: []parser.Node{
				{Kind: parser.KindParagraph, Range: parser.Range{Start: 0, End: 4}, TopLevel: true},
				{Kind: parser.KindHeading, Range: parser.Range{Start: 7, End: 11}, Level: 1, TopLevel: true},
			},
		},
		{
			name:   "thematic break",
			source: []byte("---\n"),
			want:   []parser.Node{{Kind: parser.KindThematicBreak, Range: parser.Range{Start: 0, End: 3}, TopLevel: true}},
		},
		{
			name:   "fenced block",
			source: fenced,
			want: []parser.Node{{
				Kind:                    parser.KindFencedCode,
				Range:                   parser.Range{Start: fencedBodyStart, End: fencedSecondLineEnd},
				Anchor:                  2,
				FencedCodeContentRanges: []parser.Range{{Start: fencedBodyStart, End: fencedBodyStart + len("one")}, {Start: fencedSecondLineStart, End: fencedSecondLineEnd}},
				FencedCodeInfo:          "go test",
				FencedCodeLanguage:      "go",
				TopLevel:                true,
			}},
		},
		{
			name:   "empty fenced block",
			source: []byte("```\n```\n"),
			want: []parser.Node{{
				Kind:                    parser.KindFencedCode,
				Range:                   parser.Range{Start: 0, End: 0},
				Anchor:                  0,
				FencedCodeContentRanges: []parser.Range{},
				TopLevel:                true,
			}},
		},
		{
			name:   "indented code is not promoted as paragraph",
			source: []byte("    code\n\nparagraph\n"),
			want:   []parser.Node{{Kind: parser.KindParagraph, Range: parser.Range{Start: 10, End: 19}, TopLevel: true}},
		},
		{
			name:   "spaces plus tab reach indented code column",
			source: []byte("  \tcode\n\nparagraph\n"),
			want:   []parser.Node{{Kind: parser.KindParagraph, Range: parser.Range{Start: 9, End: 18}, TopLevel: true}},
		},
		{
			name:   "ordinary paragraph indentation is excluded from range start",
			source: []byte("   text\n"),
			want:   []parser.Node{{Kind: parser.KindParagraph, Range: parser.Range{Start: 3, End: 7}, TopLevel: true}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			before := bytes.Clone(tt.source)
			got, err := ParseBlocks(tt.source)
			if err != nil {
				t.Fatalf("ParseBlocks() error = %v", err)
			}
			if !bytes.Equal(tt.source, before) {
				t.Fatalf("ParseBlocks() mutated source: got %q want %q", tt.source, before)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseBlocks() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseBlocksRejectsBacktickInfoContainingBacktick(t *testing.T) {
	t.Parallel()

	source := []byte("``` go`bad\nbody\n```\n")
	got, err := ParseBlocks(source)
	if err != nil {
		t.Fatalf("ParseBlocks() error = %v", err)
	}
	if len(got) == 0 || got[0].Kind != parser.KindParagraph || !strings.Contains(string(source[got[0].Range.Start:got[0].Range.End]), "go`bad") {
		t.Fatalf("ParseBlocks() = %#v, want first line owned by paragraph semantics", got)
	}
	for _, node := range got {
		if node.Kind == parser.KindFencedCode && node.Anchor == 0 {
			t.Fatalf("ParseBlocks() accepted invalid backtick info as opening fence: %#v", got)
		}
	}
}
