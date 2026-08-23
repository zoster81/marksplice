package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicAppendListItemChildSupportsExistingCompleteParentSubtree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   []byte
		child    string
		fragment []byte
		want     []byte
	}{
		{
			name:     "existing direct child",
			source:   []byte("- parent\n  - first\n- tail\n"),
			child:    "second",
			fragment: []byte("  + second\n"),
			want:     []byte("- parent\n  - first\n  + second\n- tail\n"),
		},
		{
			name:     "append after deepest descendant",
			source:   []byte("- parent\n  - first\n    - grandchild\n- tail\n"),
			child:    "second",
			fragment: []byte("  - second\n"),
			want:     []byte("- parent\n  - first\n    - grandchild\n  - second\n- tail\n"),
		},
		{
			name:     "wide ordered parent CRLF",
			source:   []byte("10.  parent\r\n     - first\r\n11.  tail\r\n"),
			child:    "second π",
			fragment: []byte("     + second π\r\n"),
			want:     []byte("10.  parent\r\n     - first\r\n     + second π\r\n11.  tail\r\n"),
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
			if !parent.HasChildren() {
				t.Fatal("parent HasChildren() = false, want true")
			}
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
			second := publicListItemByContent(t, updated, got, tt.child)
			if parentID, ok := second.ParentID(); !ok || parentID != updatedParent.ID() {
				t.Fatalf("second ParentID() = (%v,%v), want (%v,true)", parentID, ok, updatedParent.ID())
			}
		})
	}
}

func TestPublicAppendListItemChildRejectsIncompleteSupportedParentSubtree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
	}{
		{
			name:   "unsupported direct child",
			source: []byte("- parent\n  - complex\n\n    second paragraph\n- tail\n"),
		},
		{
			name:   "unsupported grandchild propagates incompleteness",
			source: []byte("- parent\n  - child\n    - complex\n\n      second paragraph\n- tail\n"),
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
			if !parent.HasChildren() {
				t.Fatal("parent HasChildren() = false, want true")
			}
			if _, err := doc.PrepareAppendListItemChild(parent.ID(), []byte("  - appended\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
				t.Fatalf("PrepareAppendListItemChild() error = %v, want ErrInvalidTargetKind", err)
			}
		})
	}
}

func TestPublicAppendListItemChildCanAppendAgainAfterM21Reparse(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	firstChange, err := doc.PrepareAppendListItemChild(parent.ID(), []byte("  - first\n"))
	if err != nil {
		t.Fatalf("first append error = %v", err)
	}
	withFirst, err := firstChange.Apply(source)
	if err != nil {
		t.Fatalf("first Apply() error = %v", err)
	}
	updated, err := marksplice.Parse(withFirst)
	if err != nil {
		t.Fatalf("Parse(first result) error = %v", err)
	}
	updatedParent := publicListItemByContent(t, updated, withFirst, "parent")
	secondChange, err := updated.PrepareAppendListItemChild(updatedParent.ID(), []byte("  - second\n"))
	if err != nil {
		t.Fatalf("second append error = %v", err)
	}
	got, err := secondChange.Apply(withFirst)
	if err != nil {
		t.Fatalf("second Apply() error = %v", err)
	}
	want := []byte("- parent\n  - first\n  - second\n- tail\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}
