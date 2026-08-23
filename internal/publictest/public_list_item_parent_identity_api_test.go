package publictest

import (
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicListItemParentIDResolvesImmediateSupportedParent(t *testing.T) {
	t.Parallel()

	source := []byte("- root\n  + child\n    1. grandchild\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	root := publicListItemByContent(t, doc, source, "root")
	child := publicListItemByContent(t, doc, source, "child")
	grandchild := publicListItemByContent(t, doc, source, "grandchild")

	if parentID, ok := root.ParentID(); ok || parentID != (marksplice.NodeID{}) {
		t.Fatalf("root ParentID() = (%v, %v), want zero,false", parentID, ok)
	}
	if parentID, ok := child.ParentID(); !ok || parentID != root.ID() {
		t.Fatalf("child ParentID() = (%v, %v), want (%v,true)", parentID, ok, root.ID())
	}
	if parentID, ok := grandchild.ParentID(); !ok || parentID != child.ID() {
		t.Fatalf("grandchild ParentID() = (%v, %v), want (%v,true)", parentID, ok, child.ID())
	}
}

func TestPublicListItemParentIDRequiresPromotedImmediateParent(t *testing.T) {
	t.Parallel()

	source := []byte("- complex\n\n  second paragraph\n\n  - child\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	child := publicListItemByContent(t, doc, source, "child")
	if parentID, ok := child.ParentID(); ok || parentID != (marksplice.NodeID{}) {
		t.Fatalf("child ParentID() = (%v, %v), want zero,false for unsupported immediate parent", parentID, ok)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindListItem {
			continue
		}
		item, ok := doc.ListItem(node.ID())
		if ok && string(source[item.Range().Start:item.Range().End]) == "complex" {
			t.Fatal("complex multi-block parent was unexpectedly promoted")
		}
	}
}

func TestPublicListItemParentIDIsContainerAware(t *testing.T) {
	t.Parallel()

	source := []byte("> 10.  parent\r\n>      - child π\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	child := publicListItemByContent(t, doc, source, "child π")
	if parentID, ok := child.ParentID(); !ok || parentID != parent.ID() {
		t.Fatalf("child ParentID() = (%v, %v), want (%v,true)", parentID, ok, parent.ID())
	}
}

func TestPublicListItemParentIDTracksM21ResultSnapshot(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	change, err := doc.PrepareAppendListItemChild(parent.ID(), []byte("  - child\n"))
	if err != nil {
		t.Fatalf("PrepareAppendListItemChild() error = %v", err)
	}
	updatedSource, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	updated, err := marksplice.Parse(updatedSource)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	updatedParent := publicListItemByContent(t, updated, updatedSource, "parent")
	child := publicListItemByContent(t, updated, updatedSource, "child")
	if parentID, ok := child.ParentID(); !ok || parentID != updatedParent.ID() {
		t.Fatalf("child ParentID() = (%v, %v), want (%v,true)", parentID, ok, updatedParent.ID())
	}
	if updatedParent.ID() == parent.ID() {
		t.Fatal("snapshot-scoped parent ID did not change after source mutation")
	}
}
