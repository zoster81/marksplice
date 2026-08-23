package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicAppendListItemChildAcceptsCompleteChildSubtree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		source     []byte
		fragment   []byte
		child      string
		grandchild string
		want       []byte
	}{
		{
			name:       "leaf parent receives child subtree",
			source:     []byte("- parent\n- tail\n"),
			fragment:   []byte("  - child\n    - grandchild\n"),
			child:      "child",
			grandchild: "grandchild",
			want:       []byte("- parent\n  - child\n    - grandchild\n- tail\n"),
		},
		{
			name:       "existing parent appends after current descendants",
			source:     []byte("- parent\n  - existing\n    - existing grandchild\n- tail\n"),
			fragment:   []byte("  - child\n    1. grandchild π\n"),
			child:      "child",
			grandchild: "grandchild π",
			want:       []byte("- parent\n  - existing\n    - existing grandchild\n  - child\n    1. grandchild π\n- tail\n"),
		},
		{
			name:       "wide ordered parent CRLF",
			source:     []byte("10.  parent\r\n11.  tail\r\n"),
			fragment:   []byte("     - child π\r\n       + grandchild\r\n"),
			child:      "child π",
			grandchild: "grandchild",
			want:       []byte("10.  parent\r\n     - child π\r\n       + grandchild\r\n11.  tail\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			parent := publicListItemByContent(t, doc, tt.source, "parent")
			change, err := doc.PrepareAppendListItemChild(parent.ID(), tt.fragment)
			if err != nil {
				t.Fatalf("PrepareAppendListItemChild() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}

			updated, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(result) error = %v", err)
			}
			updatedParent := publicListItemByContent(t, updated, got, "parent")
			child := publicListItemByContent(t, updated, got, tt.child)
			grandchild := publicListItemByContent(t, updated, got, tt.grandchild)
			if parentID, ok := child.ParentID(); !ok || parentID != updatedParent.ID() {
				t.Fatalf("child ParentID() = (%v,%v), want (%v,true)", parentID, ok, updatedParent.ID())
			}
			if parentID, ok := grandchild.ParentID(); !ok || parentID != child.ID() {
				t.Fatalf("grandchild ParentID() = (%v,%v), want (%v,true)", parentID, ok, child.ID())
			}
			if !child.HasChildren() || grandchild.HasChildren() {
				t.Fatalf("child/grandchild HasChildren() = %v/%v, want true/false", child.HasChildren(), grandchild.HasChildren())
			}
		})
	}
}

func TestPublicAppendListItemChildSubtreeRejectsAmbiguousOrIncompleteFragment(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")

	tests := []struct {
		name     string
		fragment []byte
	}{
		{name: "multiple direct child roots", fragment: []byte("  - one\n  - two\n")},
		{name: "trailing blank line outside subtree", fragment: []byte("  - child\n    - grandchild\n\n")},
		{name: "unsupported descendant", fragment: []byte("  - child\n    - complex\n\n      second paragraph\n")},
		{name: "root is same-level sibling", fragment: []byte("- child\n  - grandchild\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := doc.PrepareAppendListItemChild(parent.ID(), tt.fragment); !errors.Is(err, marksplice.ErrInvalidReplacement) {
				t.Fatalf("PrepareAppendListItemChild() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}
}
