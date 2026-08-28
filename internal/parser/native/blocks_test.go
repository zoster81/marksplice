package native

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestM114CommonMark0312BlockquoteTabStructures(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte(">\t#"),
		[]byte("   >  \t#"),
	} {
		nodes, err := ParseBlocks(source)
		if err != nil {
			t.Fatalf("ParseBlocks(%q) error = %v", source, err)
		}
		for _, node := range nodes {
			if node.Kind == parser.KindParagraph {
				t.Fatalf("ParseBlocks(%q) = %#v, empty ATX heading must not become paragraph content", source, nodes)
			}
		}
	}

	for _, tt := range []struct {
		source []byte
		fence  []byte
	}{
		{source: []byte(">\t```"), fence: []byte("```")},
		{source: []byte("   >\t~~~"), fence: []byte("~~~")},
	} {
		nodes, err := ParseBlocks(tt.source)
		if err != nil {
			t.Fatalf("ParseBlocks(%q) error = %v", tt.source, err)
		}
		wantAnchor := bytes.Index(tt.source, tt.fence)
		found := false
		for _, node := range nodes {
			if node.Kind != parser.KindFencedCode {
				continue
			}
			found = true
			if node.Anchor != wantAnchor {
				t.Fatalf("ParseBlocks(%q) fenced anchor = %d, want physical source byte %d", tt.source, node.Anchor, wantAnchor)
			}
		}
		if !found {
			t.Fatalf("ParseBlocks(%q) = %#v, want fenced-code observation", tt.source, nodes)
		}
	}
}

func TestM115GFMTableSplitsTrailingLineFromOpenParagraph(t *testing.T) {
	t.Parallel()

	source := []byte("Intro line.\n| A | B |\n| --- | --- |\n| x | y |\n")
	parsed := parseBlockLines(source, physicalLines(source), true)
	nodes := parsed.nodes

	tableAnchor := bytes.Index(source, []byte("| A | B |"))
	var table parser.Node
	found := false
	for _, node := range nodes {
		if node.Kind != parser.KindTable {
			continue
		}
		table = node
		found = true
		break
	}
	if !found {
		t.Fatalf("ParseBlocks() = %#v, want table split from trailing paragraph line", nodes)
	}
	if table.Range.Start != tableAnchor || len(parsed.tableDetails) != 1 || parsed.tableDetails[0].Anchor != tableAnchor {
		t.Fatalf("table range/detail = %v/%#v, want anchor %d", table.Range, parsed.tableDetails, tableAnchor)
	}
}

func TestM114CommonMark0312EmptyListItemMayOwnIndentedNestedList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		source    []byte
		wantRoots int
	}{
		{name: "empty plus item owns nested star list", source: []byte("+\n  *     0\n  0"), wantRoots: 1},
		{name: "preceding incompatible list remains separate", source: []byte("*\n\n+\r  *     0\n  0"), wantRoots: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBlockLines(tt.source, physicalLines(tt.source), true)
			if len(result.roots) != tt.wantRoots {
				t.Fatalf("top-level roots = %#v, want %d; nodes = %#v", result.roots, tt.wantRoots, result.nodes)
			}
			for _, root := range result.roots {
				if root.kind != rootBlockList {
					t.Fatalf("top-level root = %#v, want list", root)
				}
			}
		})
	}
}

func TestM114CommonMark0312LazyBlockquoteContinuationRequiresParagraphLeaf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		source       []byte
		wantTopLevel bool
	}{
		{name: "complete HTML leaf stops lazy continuation", source: []byte("><A>\n0"), wantTopLevel: true},
		{name: "paragraph leaf permits lazy continuation", source: []byte(">p\n0"), wantTopLevel: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := ParseBlocks(tt.source)
			if err != nil {
				t.Fatalf("ParseBlocks() error = %v", err)
			}
			offset := len(tt.source) - 1
			found := false
			for _, node := range nodes {
				if node.Kind != parser.KindParagraph || offset < node.Range.Start || offset >= node.Range.End {
					continue
				}
				found = true
				if node.TopLevel != tt.wantTopLevel {
					t.Fatalf("ParseBlocks(%q) trailing paragraph TopLevel = %v, want %v; node = %#v", tt.source, node.TopLevel, tt.wantTopLevel, node)
				}
			}
			if !found {
				t.Fatalf("ParseBlocks(%q) has no paragraph owning trailing byte; nodes = %#v", tt.source, nodes)
			}
		})
	}
}

func TestM114CommonMark0312HTMLBlockGrammar(t *testing.T) {
	t.Parallel()

	t.Run("textarea is type one", func(t *testing.T) {
		source := []byte("<textarea>\n\n*x*\n\n</textarea>\n")
		nodes, err := ParseBlocks(source)
		if err != nil {
			t.Fatalf("ParseBlocks() error = %v", err)
		}
		if len(nodes) != 1 || nodes[0].Kind != parser.KindHTMLBlock {
			t.Fatalf("ParseBlocks() = %#v, want one HTML block", nodes)
		}
	})

	t.Run("type six tag list follows CommonMark 0.31.2", func(t *testing.T) {
		for _, tt := range []struct {
			name      string
			source    []byte
			wantHTML  bool
			paragraph bool
		}{
			{name: "search interrupts paragraph", source: []byte("x\n<search>\ny\n"), wantHTML: true},
			{name: "source no longer interrupts paragraph", source: []byte("x\n<source>\ny\n"), wantHTML: false, paragraph: true},
		} {
			t.Run(tt.name, func(t *testing.T) {
				nodes, err := ParseBlocks(tt.source)
				if err != nil {
					t.Fatalf("ParseBlocks() error = %v", err)
				}
				gotHTML := false
				gotParagraph := false
				for _, node := range nodes {
					gotHTML = gotHTML || node.Kind == parser.KindHTMLBlock
					gotParagraph = gotParagraph || node.Kind == parser.KindParagraph
				}
				if gotHTML != tt.wantHTML || tt.paragraph && !gotParagraph {
					t.Fatalf("ParseBlocks() = %#v, want HTML=%v paragraph=%v", nodes, tt.wantHTML, tt.paragraph)
				}
			})
		}
	})

	t.Run("lowercase declaration starts HTML block", func(t *testing.T) {
		nodes, err := ParseBlocks([]byte("<!a0>\n"))
		if err != nil {
			t.Fatalf("ParseBlocks() error = %v", err)
		}
		if len(nodes) != 1 || nodes[0].Kind != parser.KindHTMLBlock {
			t.Fatalf("ParseBlocks() = %#v, want declaration HTML block", nodes)
		}
	})
}

func TestParseBlocksLeafStructures(t *testing.T) {
	t.Parallel()

	fenced := []byte("  ~~~~ go test  \n  one\n    two\n  ~~~~\n")
	fencedBodyStart := bytes.Index(fenced, []byte("one"))
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
				Kind:     parser.KindFencedCode,
				Range:    parser.Range{Start: fencedBodyStart, End: fencedSecondLineEnd},
				Anchor:   2,
				TopLevel: true,
			}},
		},
		{
			name:   "empty fenced block",
			source: []byte("```\n```\n"),
			want: []parser.Node{{
				Kind:     parser.KindFencedCode,
				Range:    parser.Range{Start: 0, End: 0},
				Anchor:   0,
				TopLevel: true,
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

func TestParseBlockLinesCollectsSparseBlockDetails(t *testing.T) {
	t.Parallel()

	source := []byte("> quote\n\n```go test\none\ntwo\n```\n")
	parsed := parseBlockLines(source, physicalLines(source), true)
	if len(parsed.blockquoteDetails) != 1 {
		t.Fatalf("blockquote details = %#v, want one", parsed.blockquoteDetails)
	}
	quote := parsed.blockquoteDetails[0]
	if quote.Anchor != 0 || quote.ContentRange != (parser.Range{Start: 2, End: 7}) || len(quote.SemanticRanges) != 1 || quote.SemanticRanges[0] != (parser.Range{Start: 2, End: 7}) {
		t.Fatalf("blockquote detail = %#v, want exact sparse facts", quote)
	}
	if len(parsed.fencedCodeDetails) != 1 {
		t.Fatalf("fenced details = %#v, want one", parsed.fencedCodeDetails)
	}
	fence := parsed.fencedCodeDetails[0]
	if fence.Anchor != 9 || fence.Info != "go test" || fence.Language != "go" || len(fence.ContentRanges) != 2 {
		t.Fatalf("fenced detail = %#v, want exact sparse facts", fence)
	}
	if fence.ContentRanges[0] != (parser.Range{Start: 20, End: 23}) || fence.ContentRanges[1] != (parser.Range{Start: 24, End: 27}) {
		t.Fatalf("fenced content ranges = %#v, want exact body lines", fence.ContentRanges)
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
