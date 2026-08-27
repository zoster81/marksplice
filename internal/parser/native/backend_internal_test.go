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
