package splice

import (
	"bytes"
	"errors"
	"testing"

	markparser "github.com/zoster81/marksplice/internal/parser"
)

func TestReplaceParagraphPreservesUnchangedBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{
			name:   "LF",
			source: []byte("# Title\n\nOriginal paragraph.\n\n- keep\n- this\n"),
		},
		{
			name:   "LF with trailing spaces",
			source: []byte("# Title\n\nOriginal paragraph.  \n\n- keep\n- this\n"),
		},
		{
			name:   "CRLF",
			source: []byte("# Title\r\n\r\nOriginal paragraph.\r\n\r\n- keep\r\n- this\r\n"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
			if len(paragraphs) != 1 {
				t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
			}
			target := paragraphs[0]

			marker := []byte("Original paragraph.")
			expectedStart := bytes.Index(tt.source, marker)
			if expectedStart < 0 {
				t.Fatal("test fixture paragraph marker not found")
			}
			expectedEnd := expectedStart + len(marker)
			for expectedEnd < len(tt.source) && (tt.source[expectedEnd] == ' ' || tt.source[expectedEnd] == '\t') {
				expectedEnd++
			}
			if target.Range.Start != expectedStart || target.Range.End != expectedEnd {
				t.Fatalf("paragraph range = [%d,%d), want [%d,%d)", target.Range.Start, target.Range.End, expectedStart, expectedEnd)
			}

			prefix := append([]byte(nil), tt.source[:expectedStart]...)
			suffix := append([]byte(nil), tt.source[expectedEnd:]...)
			replacement := []byte("Replacement paragraph with **formatting**.")

			change, err := doc.PrepareReplace(target.ID, replacement)
			if err != nil {
				t.Fatalf("PrepareReplace() error = %v", err)
			}

			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}

			want := make([]byte, 0, len(prefix)+len(replacement)+len(suffix))
			want = append(want, prefix...)
			want = append(want, replacement...)
			want = append(want, suffix...)
			if !bytes.Equal(got, want) {
				t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) {
				t.Fatal("bytes before changed span were modified")
			}
			if !bytes.Equal(got[len(prefix)+len(replacement):], suffix) {
				t.Fatal("bytes after changed span were modified")
			}
		})
	}
}

func TestTableCellSourceMappingCacheReusesOneRowMapping(t *testing.T) {
	t.Parallel()

	snapshot := []byte("| alpha | beta |\n")
	cache := make(map[int]tableRowSourceResult)
	firstObservation := markparser.Node{Kind: markparser.KindTableCell, TableColumn: 0, TableRowAnchor: 0}
	secondObservation := markparser.Node{Kind: markparser.KindTableCell, TableColumn: 1, TableRowAnchor: 0}

	first, editable, err := mapTableCellSource(snapshot, firstObservation, Range{Start: 2, End: 7}, cache)
	if err != nil || !editable {
		t.Fatalf("first map = %+v, editable %v, error %v; want mapped/true/nil", first, editable, err)
	}
	second, editable, err := mapTableCellSource(snapshot, secondObservation, Range{Start: 10, End: 14}, cache)
	if err != nil || !editable {
		t.Fatalf("second map = %+v, editable %v, error %v; want mapped/true/nil", second, editable, err)
	}
	if len(cache) != 1 {
		t.Fatalf("table row mapping cache size = %d, want 1 for two cells in one row", len(cache))
	}
	if first.Column != 0 || second.Column != 1 {
		t.Fatalf("mapped columns = %d/%d, want 0/1", first.Column, second.Column)
	}
}

func TestDocumentNodeCopiesDoNotAliasTableAlignments(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("| A | B |\n| :--- | ---: |\n| one | two |\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	rows := nodesOfKind(doc.Nodes(), KindTableRow)
	if len(rows) != 1 || len(rows[0].TableAlignments) != 2 || len(rows[0].TableRowSource.Cells) != 2 {
		t.Fatalf("table rows = %+v, want one row with two alignments/cells", rows)
	}
	rowID := rows[0].ID
	rows[0].TableAlignments[0] = TableAlignmentCenter
	rows[0].TableRowSource.Cells[0].Column = 99

	again := nodesOfKind(doc.Nodes(), KindTableRow)
	if len(again) != 1 || again[0].TableAlignments[0] != TableAlignmentLeft || again[0].TableAlignments[1] != TableAlignmentRight || again[0].TableRowSource.Cells[0].Column != 0 {
		t.Fatalf("Nodes() table copy = alignments %v cells %+v, want [left right] and original cells", again[0].TableAlignments, again[0].TableRowSource.Cells)
	}
	node, ok := doc.Node(rowID)
	if !ok {
		t.Fatalf("Node(%q) ok = false", rowID)
	}
	node.TableAlignments[1] = TableAlignmentCenter
	node.TableRowSource.Cells[1].Column = 99
	nodeAgain, ok := doc.Node(rowID)
	if !ok || nodeAgain.TableAlignments[0] != TableAlignmentLeft || nodeAgain.TableAlignments[1] != TableAlignmentRight || nodeAgain.TableRowSource.Cells[1].Column != 1 {
		t.Fatalf("Node() table copy = alignments %v cells %+v, %v; want [left right], original cells, true", nodeAgain.TableAlignments, nodeAgain.TableRowSource.Cells, ok)
	}
}

func TestPreparedChangeRejectsStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("# Title\n\nOriginal paragraph.\n")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}

	change, err := doc.PrepareReplace(paragraphs[0].ID, []byte("Replacement."))
	if err != nil {
		t.Fatalf("PrepareReplace() error = %v", err)
	}

	stale := append([]byte(nil), source...)
	stale[0] = '*'
	_, err = change.Apply(stale)
	if !errors.Is(err, ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func TestDuplicateParagraphsHaveDistinctDeterministicIDs(t *testing.T) {
	t.Parallel()

	source := []byte("same\n\nsame\n")
	first, err := Parse(source)
	if err != nil {
		t.Fatalf("first Parse() error = %v", err)
	}
	second, err := Parse(source)
	if err != nil {
		t.Fatalf("second Parse() error = %v", err)
	}

	firstParagraphs := nodesOfKind(first.Nodes(), KindParagraph)
	secondParagraphs := nodesOfKind(second.Nodes(), KindParagraph)
	if len(firstParagraphs) != 2 || len(secondParagraphs) != 2 {
		t.Fatalf("paragraph counts = %d and %d, want 2 and 2", len(firstParagraphs), len(secondParagraphs))
	}
	if firstParagraphs[0].ID == firstParagraphs[1].ID {
		t.Fatal("duplicate paragraph content produced duplicate node IDs")
	}
	for i := range firstParagraphs {
		if firstParagraphs[i].ID != secondParagraphs[i].ID {
			t.Fatalf("node %d ID is not deterministic: %q != %q", i, firstParagraphs[i].ID, secondParagraphs[i].ID)
		}
	}
}

func TestPrepareReplaceRejectsUnknownNode(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("paragraph\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	_, err = doc.PrepareReplace(NodeID("missing"), []byte("replacement"))
	if !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("PrepareReplace() error = %v, want ErrNodeNotFound", err)
	}
}

func TestPrepareReplaceRejectsDifferentBlockKind(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("paragraph\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}

	_, err = doc.PrepareReplace(paragraphs[0].ID, []byte("# heading"))
	if !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("PrepareReplace() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareReplaceAllowsNestedInlineObservationsInsideParagraph(t *testing.T) {
	t.Parallel()

	source := []byte("old paragraph\n")
	replacement := []byte("new [link](dest), https://example.com, `code`, *em*, **strong**, and ~~strike~~")
	doc, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	paragraphs := nodesOfKind(doc.Nodes(), KindParagraph)
	if len(paragraphs) != 1 {
		t.Fatalf("paragraph count = %d, want 1", len(paragraphs))
	}
	change, err := doc.PrepareReplace(paragraphs[0].ID, replacement)
	if err != nil {
		t.Fatalf("PrepareReplace() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, append(replacement, '\n')) {
		t.Fatalf("result = %q, want %q", got, append(replacement, '\n'))
	}
}

func nodesOfKind(nodes []Node, kind Kind) []Node {
	var result []Node
	for _, node := range nodes {
		if node.Kind == kind {
			result = append(result, node)
		}
	}
	return result
}
