package splice

import (
	"testing"

	"github.com/zoster81/marksplice/internal/source"
)

func TestNodeSelectionRangeCoversEveryPromotedKind(t *testing.T) {
	t.Parallel()

	content := Range{Start: 10, End: 20}
	paragraph := Range{Start: 30, End: 40}
	row := Range{Start: 50, End: 60}
	table := Range{Start: 70, End: 80}
	thematicBreak := Range{Start: 90, End: 100}
	blockquote := Range{Start: 110, End: 120}

	document := &Document{source: make([]byte, 200)}
	for kind := KindUnknown; kind <= KindTable; kind++ {
		node := Node{Kind: kind, Range: paragraph, ContentRange: content}
		document.blockquoteSources = nil
		if kind == KindBlockquote {
			node.Range = Range{Start: 112, End: 119}
			node.ContentRange = Range{Start: 112, End: 119}
			node.Editable = true
			node.TopLevel = true
			node.SourceDetailIndex = 1
			document.blockquoteSources = []source.BlockquoteMapping{{
				Range:         node.Range,
				LineRange:     blockquote,
				ContentRange:  node.ContentRange,
				ContentRanges: []source.Range{node.ContentRange},
				MarkerRange:   Range{Start: 110, End: 111},
			}}
		}
		if kind == KindTableRow {
			node.Range = row
		}
		if kind == KindTable {
			node.Range = table
		}
		if kind == KindThematicBreak {
			node.Range = thematicBreak
		}

		got, ok := document.nodeSelectionRange(node)
		if kind == KindUnknown || kind == KindHTMLOpaque {
			if ok || got != (Range{}) {
				t.Fatalf("nodeSelectionRange(kind=%d) = %v/%v, want zero/false", kind, got, ok)
			}
			continue
		}

		want := content
		switch kind {
		case KindParagraph:
			want = paragraph
		case KindTableRow:
			want = row
		case KindTable:
			want = table
		case KindThematicBreak:
			want = thematicBreak
		case KindBlockquote:
			want = blockquote
		}
		if !ok || got != want {
			t.Fatalf("nodeSelectionRange(kind=%d) = %v/%v, want %v/true", kind, got, ok, want)
		}
	}
}

func TestDocumentQuerySummaryPrimitivesAreNilSafe(t *testing.T) {
	t.Parallel()

	var document *Document
	if document.NodeCount() != 0 || document.SourceLen() != 0 {
		t.Fatalf("nil document counts = nodes %d source %d, want 0/0", document.NodeCount(), document.SourceLen())
	}
	if _, ok := document.NodeSummaryAt(0); ok {
		t.Fatal("nil document NodeSummaryAt(0) ok = true")
	}
	if _, _, ok := document.NodeSelectionAt(0); ok {
		t.Fatal("nil document NodeSelectionAt(0) ok = true")
	}
}
