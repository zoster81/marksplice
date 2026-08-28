package splice

import (
	"errors"
	"testing"
)

func TestIndexNodesBuildsDeterministicLookup(t *testing.T) {
	t.Parallel()

	nodes := []Node{
		{ID: NodeID("first"), Kind: KindParagraph},
		{ID: NodeID("second"), Kind: KindHeading},
	}
	index, err := indexNodes(nodes)
	if err != nil {
		t.Fatalf("indexNodes() error = %v", err)
	}
	if len(index) != len(nodes) || index[nodes[0].ID] != 0 || index[nodes[1].ID] != 1 {
		t.Fatalf("index = %#v, want first->0 and second->1", index)
	}
}

func TestIndexNodesRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	_, err := indexNodes([]Node{
		{ID: NodeID("duplicate"), Kind: KindParagraph},
		{ID: NodeID("duplicate"), Kind: KindHeading},
	})
	if !errors.Is(err, errDuplicateNodeID) {
		t.Fatalf("indexNodes() error = %v, want errDuplicateNodeID", err)
	}
}

func TestMergeSourceOrderedNodesReusesCapacityAndPreservesTieOrder(t *testing.T) {
	t.Parallel()

	nodes := make([]Node, 2, 5)
	nodes[0] = Node{ID: "base-10", Range: Range{Start: 10, End: 11}}
	nodes[1] = Node{ID: "base-30", Range: Range{Start: 30, End: 31}}
	first := &nodes[0]
	additions := []Node{
		{ID: "addition-10", Range: Range{Start: 10, End: 12}},
		{ID: "addition-20", Range: Range{Start: 20, End: 21}},
		{ID: "addition-40", Range: Range{Start: 40, End: 41}},
	}

	got := mergeSourceOrderedNodes(nodes, additions)
	if &got[0] != first {
		t.Fatal("mergeSourceOrderedNodes() replaced reusable backing storage")
	}
	want := []NodeID{"base-10", "addition-10", "addition-20", "base-30", "addition-40"}
	if len(got) != len(want) {
		t.Fatalf("merged node count = %d, want %d", len(got), len(want))
	}
	for index, id := range want {
		if got[index].ID != id {
			t.Fatalf("merged node %d ID = %q, want %q", index, got[index].ID, id)
		}
	}
}

func TestMergeSourceOrderedNodesGrowsWhenCapacityIsInsufficient(t *testing.T) {
	t.Parallel()

	nodes := []Node{{ID: "base-20", Range: Range{Start: 20, End: 21}}}
	got := mergeSourceOrderedNodes(nodes, []Node{
		{ID: "addition-10", Range: Range{Start: 10, End: 11}},
		{ID: "addition-30", Range: Range{Start: 30, End: 31}},
	})
	want := []NodeID{"addition-10", "base-20", "addition-30"}
	for index, id := range want {
		if got[index].ID != id {
			t.Fatalf("merged node %d ID = %q, want %q", index, got[index].ID, id)
		}
	}
}

func TestListItemHierarchyAccessFailsClosedOnCorruptAdjacency(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("- parent\n  - child\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	items := nodesOfKind(doc.nodes, KindListItem)
	if len(items) != 2 || items[0].ListChildCount != 1 {
		t.Fatalf("list-item model = %+v, want parent plus one child", items)
	}
	parent := items[0]
	doc.listChildIDs[parent.ListChildStart] = NodeID("missing-child")
	if ids, ok := doc.ListItemChildIDs(parent.ID); ok || ids != nil {
		t.Fatalf("ListItemChildIDs(corrupt) = %v, %v; want nil, false", ids, ok)
	}
	if _, err := doc.ownedListItemSubtree(parent); err == nil {
		t.Fatal("ownedListItemSubtree(corrupt) error = nil, want fail-closed error")
	}
}

func TestCompactMutationIndexesFailClosedWhenCorrupt(t *testing.T) {
	t.Parallel()

	listDoc, err := Parse([]byte("- item\n"))
	if err != nil {
		t.Fatalf("Parse(list) error = %v", err)
	}
	if len(listDoc.listItemIndexes) != 1 {
		t.Fatalf("list-item index count = %d, want 1", len(listDoc.listItemIndexes))
	}
	lineStart := listDoc.nodes[listDoc.listItemIndexes[0]].ListItemLineRange.Start
	listDoc.listItemIndexes[0] = len(listDoc.nodes)
	if _, ok := listDoc.listItemNodeAtLineStart(lineStart); ok {
		t.Fatal("listItemNodeAtLineStart(corrupt index) ok = true, want false")
	}
	if _, err := listItemCandidateMappingsFromDocument(listDoc); !errors.Is(err, ErrInvalidReplacement) {
		t.Fatalf("listItemCandidateMappingsFromDocument(corrupt index) error = %v, want ErrInvalidReplacement", err)
	}

	tableDoc, err := Parse([]byte("| h |\n| --- |\n| v |\n"))
	if err != nil {
		t.Fatalf("Parse(table) error = %v", err)
	}
	if len(tableDoc.tableRowIndexes) != 1 || len(tableDoc.tableCellIndexes) != 2 {
		t.Fatalf("table compact indexes = rows %d cells %d, want 1/2", len(tableDoc.tableRowIndexes), len(tableDoc.tableCellIndexes))
	}
	tableDoc.tableRowIndexes[0] = len(tableDoc.nodes)
	if _, ok := indexTableMutationModel(tableDoc); ok {
		t.Fatal("indexTableMutationModel(corrupt row index) ok = true, want false")
	}

	tableDoc, err = Parse([]byte("| h |\n| --- |\n| v |\n"))
	if err != nil {
		t.Fatalf("Parse(table cell corruption) error = %v", err)
	}
	tableDoc.tableCellIndexes[0] = len(tableDoc.nodes)
	if _, ok := indexTableMutationModel(tableDoc); ok {
		t.Fatal("indexTableMutationModel(corrupt cell index) ok = true, want false")
	}
}

func TestParseBuildsSourceOrderedListItemHierarchyAndSubtreeMetadata(t *testing.T) {
	t.Parallel()

	doc, err := Parse([]byte("- parent\n  - child one\n  - child two\n- leaf\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	items := nodesOfKind(doc.nodes, KindListItem)
	if len(items) != 4 {
		t.Fatalf("list-item count = %d, want 4", len(items))
	}
	parent := items[0]
	if parent.ListParentID != "" || parent.ListChildCount != 2 || !parent.ListSubtreeComplete {
		t.Fatalf("parent metadata = %+v, want root with two supported children and complete subtree", parent)
	}
	if parent.ListChildStart < 0 || parent.ListChildStart+parent.ListChildCount > len(doc.listChildIDs) {
		t.Fatalf("parent child adjacency = start %d count %d over %d IDs", parent.ListChildStart, parent.ListChildCount, len(doc.listChildIDs))
	}
	children := doc.listChildIDs[parent.ListChildStart : parent.ListChildStart+parent.ListChildCount]
	if children[0] != items[1].ID || children[1] != items[2].ID {
		t.Fatalf("parent child IDs = %v, want %q/%q", children, items[1].ID, items[2].ID)
	}
	for _, child := range items[1:3] {
		if child.ListParentID != parent.ID || !child.ListSubtreeComplete {
			t.Fatalf("child metadata = %+v, want parent %q and complete subtree", child, parent.ID)
		}
	}
	if parent.ListSubtreeEnd != items[2].ListItemLineRange.End {
		t.Fatalf("parent subtree end = %d, want %d", parent.ListSubtreeEnd, items[2].ListItemLineRange.End)
	}
	if items[3].ListParentID != "" || !items[3].ListSubtreeComplete {
		t.Fatalf("leaf metadata = %+v, want independent complete root", items[3])
	}
}
