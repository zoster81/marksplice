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
	lineStart := listDoc.nodes[listDoc.listItemIndexes[0]].ListItemSource.LineRange.Start
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
	if parent.ListSubtreeEnd != items[2].ListItemSource.LineRange.End {
		t.Fatalf("parent subtree end = %d, want %d", parent.ListSubtreeEnd, items[2].ListItemSource.LineRange.End)
	}
	if items[3].ListParentID != "" || !items[3].ListSubtreeComplete {
		t.Fatalf("leaf metadata = %+v, want independent complete root", items[3])
	}
}
