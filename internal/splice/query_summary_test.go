package splice

import "testing"

func TestNodeSelectionRangeCoversEveryPromotedKind(t *testing.T) {
	t.Parallel()

	content := Range{Start: 10, End: 20}
	paragraph := Range{Start: 30, End: 40}
	row := Range{Start: 50, End: 60}
	table := Range{Start: 70, End: 80}
	thematicBreak := Range{Start: 90, End: 100}
	blockquote := Range{Start: 110, End: 120}

	for kind := KindUnknown; kind <= KindTable; kind++ {
		node := Node{Kind: kind, Range: paragraph, ContentRange: content}
		node.TableRowSource.LineRange = row
		node.TableSource.Range = table
		node.ThematicBreakSource.LineRange = thematicBreak
		node.BlockquoteSource.LineRange = blockquote

		got, ok := nodeSelectionRange(node)
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
