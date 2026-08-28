package native

import (
	"reflect"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
)

func TestMergeDocumentNodesPreservesStableDocumentOrder(t *testing.T) {
	orderedBlocks := []parser.Node{
		{Kind: parser.KindParagraph, Range: parser.Range{Start: 0, End: 10}, Label: "block-0"},
		{Kind: parser.KindHeading, Range: parser.Range{Start: 20, End: 25}, Label: "block-20"},
	}
	orderedInlines := make([]parser.Node, 4, 8)
	orderedInlines[0] = parser.Node{Kind: parser.KindEmphasis, Range: parser.Range{Start: 1, End: 9}, Label: "inline-1"}
	orderedInlines[1] = parser.Node{Kind: parser.KindInlineLink, Range: parser.Range{Start: 21, End: 24}, Anchor: 20, Label: "inline-20"}
	orderedInlines[2] = parser.Node{Kind: parser.KindStrong, Range: parser.Range{Start: 31, End: 40}, Anchor: 30, Label: "inline-30-long"}
	orderedInlines[3] = parser.Node{Kind: parser.KindEmphasis, Range: parser.Range{Start: 31, End: 35}, Anchor: 30, Label: "inline-30-short"}

	want := []string{"block-0", "inline-1", "block-20", "inline-20", "inline-30-long", "inline-30-short"}
	fallbackInlines := append([]parser.Node(nil), orderedInlines...)
	assertMergedNodeLabels(t, mergeDocumentNodes(orderedBlocks, orderedInlines), want)

	unsortedBlocks := []parser.Node{orderedBlocks[1], orderedBlocks[0]}
	assertMergedNodeLabels(t, mergeDocumentNodes(unsortedBlocks, fallbackInlines), want)
}

func TestAttachNodeDetailsCompactsRemovedSparseFacts(t *testing.T) {
	t.Parallel()

	nodes := []parser.Node{
		{Kind: parser.KindBlockquote, Range: parser.Range{Start: 10, End: 15}, TopLevel: true},
		{Kind: parser.KindFencedCode, Range: parser.Range{Start: 34, End: 35}, Anchor: 30, TopLevel: true},
		{Kind: parser.KindTable, Range: parser.Range{Start: 50, End: 55}},
		{Kind: parser.KindTableCell, Range: parser.Range{Start: 52, End: 53}},
		{Kind: parser.KindTableRow, Range: parser.Range{Start: 70, End: 80}},
		{Kind: parser.KindTableCell, Range: parser.Range{Start: 72, End: 73}},
	}
	blockquotes := []parser.BlockquoteDetail{{Anchor: 0}, {Anchor: 10, SemanticRanges: []parser.Range{{Start: 12, End: 15}}}}
	fenced := []parser.FencedCodeDetail{{Anchor: 20}, {Anchor: 30, ContentRanges: []parser.Range{{Start: 34, End: 35}}}}
	tables := []parser.TableDetail{{Anchor: 40}, {Anchor: 50}}
	rows := []parser.TableRowDetail{{RowAnchor: 60}, {RowAnchor: 70}}
	cells := []parser.TableCellDetail{
		{Range: parser.Range{Start: 42, End: 43}},
		{Range: parser.Range{Start: 52, End: 53}},
		{Range: parser.Range{Start: 72, End: 73}},
	}

	got, err := attachNodeDetails(nodes, sparseNodeDetails{
		blockquotes: blockquotes,
		fencedCode:  fenced,
		tables:      tables,
		tableRows:   rows,
		tableCells:  cells,
	})
	if err != nil {
		t.Fatalf("attachNodeDetails() error = %v", err)
	}
	if len(got.blockquotes) != 1 || got.blockquotes[0].Anchor != 10 || nodes[0].DetailIndex != 1 {
		t.Fatalf("compacted blockquote details/nodes = %#v / %#v", got.blockquotes, nodes)
	}
	if len(got.fencedCode) != 1 || got.fencedCode[0].Anchor != 30 || nodes[1].DetailIndex != 1 {
		t.Fatalf("compacted fenced details/nodes = %#v / %#v", got.fencedCode, nodes)
	}
	if len(got.tables) != 1 || got.tables[0].Anchor != 50 || nodes[2].DetailIndex != 1 {
		t.Fatalf("compacted table details/nodes = %#v / %#v", got.tables, nodes)
	}
	if len(got.tableRows) != 1 || got.tableRows[0].RowAnchor != 70 || nodes[4].DetailIndex != 1 {
		t.Fatalf("compacted table-row details/nodes = %#v / %#v", got.tableRows, nodes)
	}
	if len(got.tableCells) != 2 || got.tableCells[0].Range != (parser.Range{Start: 52, End: 53}) || got.tableCells[1].Range != (parser.Range{Start: 72, End: 73}) || nodes[3].DetailIndex != 1 || nodes[5].DetailIndex != 2 {
		t.Fatalf("compacted table-cell details/nodes = %#v / %#v", got.tableCells, nodes)
	}
}

func TestReferenceDefinitionIndexPreservesFirstNormalizedDefinitionAcrossBlocks(t *testing.T) {
	source := []byte("[Straße]: <one>\n[STRASSE]: <two>\n\n[first][strasse]\n\n[second][STRASSE]\n")
	assertReferenceDestinations(t, source, 2, "one")
}

func TestReferenceDefinitionIndexIsReusedInsideFootnotePasses(t *testing.T) {
	source := []byte("[Straße]: <one>\n[STRASSE]: <two>\n\n[first][strasse]\n\n[^note]: [inside][STRASSE]\n\n[second][STRASSE] [^note]\n")
	assertReferenceDestinations(t, source, 3, "one")
}

func assertReferenceDestinations(t *testing.T, source []byte, wantCount int, wantDestination string) {
	t.Helper()
	observations, err := New().ParseDocument(source)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}
	var references []parser.LinkUsage
	for _, usage := range observations.LinkUsages {
		if usage.Form != parser.LinkUsageDirect {
			references = append(references, usage)
		}
	}
	if len(references) != wantCount {
		t.Fatalf("reference usages = %#v, want %d", references, wantCount)
	}
	for index, usage := range references {
		if usage.Destination != wantDestination {
			t.Fatalf("reference %d destination = %q, want first normalized definition %q", index, usage.Destination, wantDestination)
		}
	}
}

func assertMergedNodeLabels(t *testing.T, nodes []parser.Node, want []string) {
	t.Helper()
	got := make([]string, len(nodes))
	for index, node := range nodes {
		got[index] = node.Label
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merged node order = %v, want %v", got, want)
	}
}
