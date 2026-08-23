package publictest

import (
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicListItemChildIDsAreImmediateAndSourceOrdered(t *testing.T) {
	t.Parallel()

	source := []byte("- root\n  - child one\n    - grandchild\n  + child two\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicListItemByContent(t, doc, source, "root")
	childOne := publicListItemByContent(t, doc, source, "child one")
	grandchild := publicListItemByContent(t, doc, source, "grandchild")
	childTwo := publicListItemByContent(t, doc, source, "child two")

	rootChildren := root.ChildIDs()
	if len(rootChildren) != 2 || rootChildren[0] != childOne.ID() || rootChildren[1] != childTwo.ID() {
		t.Fatalf("root ChildIDs() = %v, want [%v %v]", rootChildren, childOne.ID(), childTwo.ID())
	}
	childOneChildren := childOne.ChildIDs()
	if len(childOneChildren) != 1 || childOneChildren[0] != grandchild.ID() {
		t.Fatalf("child one ChildIDs() = %v, want [%v]", childOneChildren, grandchild.ID())
	}
	if got := grandchild.ChildIDs(); len(got) != 0 {
		t.Fatalf("grandchild ChildIDs() = %v, want empty", got)
	}
	if got := childTwo.ChildIDs(); len(got) != 0 {
		t.Fatalf("child two ChildIDs() = %v, want empty", got)
	}

	if parentID, ok := childOne.ParentID(); !ok || parentID != root.ID() {
		t.Fatalf("child one ParentID() = (%v,%v), want (%v,true)", parentID, ok, root.ID())
	}
	if parentID, ok := childTwo.ParentID(); !ok || parentID != root.ID() {
		t.Fatalf("child two ParentID() = (%v,%v), want (%v,true)", parentID, ok, root.ID())
	}
}

func TestPublicListItemChildIDsExposeOnlySupportedDirectChildren(t *testing.T) {
	t.Parallel()

	source := []byte("- root\n  - supported\n  - complex\n\n    second paragraph\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicListItemByContent(t, doc, source, "root")
	supported := publicListItemByContent(t, doc, source, "supported")
	if !root.HasChildren() {
		t.Fatal("root HasChildren() = false, want semantic children")
	}
	children := root.ChildIDs()
	if len(children) != 1 || children[0] != supported.ID() {
		t.Fatalf("root ChildIDs() = %v, want only supported child %v", children, supported.ID())
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindListItem {
			continue
		}
		item, ok := doc.ListItem(node.ID())
		if ok && string(source[item.Range().Start:item.Range().End]) == "complex" {
			t.Fatal("unsupported complex child was unexpectedly promoted")
		}
	}
}

func TestPublicListItemChildIDsCanBeEmptyWhenAllSemanticChildrenAreUnsupported(t *testing.T) {
	t.Parallel()

	source := []byte("- root\n  - complex\n\n    second paragraph\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicListItemByContent(t, doc, source, "root")
	if !root.HasChildren() {
		t.Fatal("root HasChildren() = false, want semantic child")
	}
	if got := root.ChildIDs(); len(got) != 0 {
		t.Fatalf("root ChildIDs() = %v, want empty because direct child is unsupported", got)
	}
}

func TestPublicListItemChildIDsReturnIndependentCopies(t *testing.T) {
	t.Parallel()

	source := []byte("- root\n  - child one\n  - child two\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicListItemByContent(t, doc, source, "root")
	childOne := publicListItemByContent(t, doc, source, "child one")

	first := root.ChildIDs()
	if len(first) != 2 {
		t.Fatalf("first ChildIDs() len = %d, want 2", len(first))
	}
	first[0] = marksplice.NodeID{}
	second := root.ChildIDs()
	if len(second) != 2 || second[0] != childOne.ID() {
		t.Fatalf("second ChildIDs() = %v, want independent original copy", second)
	}
}

func TestPublicListItemChildIDsTrackReparsedAppendSnapshot(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	if got := parent.ChildIDs(); len(got) != 0 {
		t.Fatalf("initial ChildIDs() = %v, want empty", got)
	}

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
		t.Fatalf("Parse(updated) error = %v", err)
	}
	updatedParent := publicListItemByContent(t, updated, updatedSource, "parent")
	child := publicListItemByContent(t, updated, updatedSource, "child")
	children := updatedParent.ChildIDs()
	if len(children) != 1 || children[0] != child.ID() {
		t.Fatalf("updated ChildIDs() = %v, want [%v]", children, child.ID())
	}
	if updatedParent.ID() == parent.ID() {
		t.Fatal("snapshot-scoped parent ID did not change after append")
	}
	if got := parent.ChildIDs(); len(got) != 0 {
		t.Fatalf("original snapshot ChildIDs() = %v, want unchanged empty", got)
	}
}
