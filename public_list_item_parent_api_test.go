package marksplice_test

import (
	"bytes"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicListItemParentDetailAndContentReplacementPreserveChildren(t *testing.T) {
	t.Parallel()

	source := []byte("1. parent\r\n   - child one\r\n   - child two π\r\n2. tail\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	if !parent.HasChildren() {
		t.Fatal("parent HasChildren() = false, want true")
	}
	if !parent.Ordered() || parent.Marker() != '.' {
		t.Fatalf("parent shape = ordered %v marker %q, want true '.'", parent.Ordered(), parent.Marker())
	}
	child := publicListItemByContent(t, doc, source, "child one")
	if child.HasChildren() {
		t.Fatal("leaf child HasChildren() = true, want false")
	}

	childSource := append([]byte(nil), source[parent.Range().End:]...)
	change, err := doc.PrepareReplaceListItem(parent.ID(), []byte("renamed parent"))
	if err != nil {
		t.Fatalf("PrepareReplaceListItem(parent) error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("1. renamed parent\r\n   - child one\r\n   - child two π\r\n2. tail\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
	if !bytes.Equal(got[parent.Range().Start+len("renamed parent"):], childSource) {
		t.Fatal("replacement changed bytes after parent content")
	}
	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	updatedParent := publicListItemByContent(t, updated, got, "renamed parent")
	if !updatedParent.HasChildren() {
		t.Fatal("renamed parent lost child-list state")
	}
	_ = publicListItemByContent(t, updated, got, "child one")
	_ = publicListItemByContent(t, updated, got, "child two π")
}

func TestPublicListItemParentRejectsUnsupportedMultiBlockShape(t *testing.T) {
	t.Parallel()

	source := []byte("- supported\n  - child\n\n- complex\n\n  second paragraph\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !publicListItemByContent(t, doc, source, "supported").HasChildren() {
		t.Fatal("supported parent missing")
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindListItem {
			continue
		}
		item, ok := doc.ListItem(node.ID())
		if ok && string(source[item.Range().Start:item.Range().End]) == "complex" {
			t.Fatal("unsupported multi-block list item was promoted")
		}
	}
}

func TestPublicAppendListItemChildResultKeepsParentPublic(t *testing.T) {
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
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	updatedParent := publicListItemByContent(t, updated, got, "parent")
	if !updatedParent.HasChildren() {
		t.Fatal("parent HasChildren() = false after M21 child append")
	}
	if publicListItemByContent(t, updated, got, "child").HasChildren() {
		t.Fatal("new child HasChildren() = true, want false")
	}
}
