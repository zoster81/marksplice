package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicRemoveListItemRemovesCompleteSupportedSubtree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		target string
		want   []byte
	}{
		{
			name:   "root parent with deep descendants",
			source: []byte("- parent\n  - child\n    - grandchild\n- tail\n"),
			target: "parent",
			want:   []byte("- tail\n"),
		},
		{
			name:   "nested parent preserves sibling under outer parent",
			source: []byte("- root\n  - parent\n    - child\n  - sibling\n- tail\n"),
			target: "parent",
			want:   []byte("- root\n  - sibling\n- tail\n"),
		},
		{
			name:   "ordered CRLF Unicode source remains exact",
			source: []byte("10. parent π\r\n    - child\r\n11. tail\r\n"),
			target: "parent π",
			want:   []byte("11. tail\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			target := publicListItemByContent(t, doc, tt.source, tt.target)
			if !target.HasChildren() {
				t.Fatal("target HasChildren() = false, want true")
			}
			change, err := doc.PrepareRemoveListItem(target.ID())
			if err != nil {
				t.Fatalf("PrepareRemoveListItem() error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
			if _, err := marksplice.Parse(got); err != nil {
				t.Fatalf("Parse(result) error = %v", err)
			}
		})
	}
}

func TestPublicRemoveListItemSubtreeUpdatesOuterParentState(t *testing.T) {
	t.Parallel()

	source := []byte("- root\n  - parent\n    - child\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	target := publicListItemByContent(t, doc, source, "parent")
	change, err := doc.PrepareRemoveListItem(target.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveListItem() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("- root\n- tail\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
	updated, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(result) error = %v", err)
	}
	root := publicListItemByContent(t, updated, got, "root")
	if root.HasChildren() {
		t.Fatal("outer parent HasChildren() = true after removing its only child subtree")
	}
}

func TestPublicRemoveListItemRejectsIncompleteSupportedSubtree(t *testing.T) {
	t.Parallel()

	tests := [][]byte{
		[]byte("- parent\n  - complex\n\n    second paragraph\n- tail\n"),
		[]byte("- parent\n  - child\n    - complex\n\n      second paragraph\n- tail\n"),
	}
	for _, source := range tests {
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", source, err)
		}
		parent := publicListItemByContent(t, doc, source, "parent")
		if _, err := doc.PrepareRemoveListItem(parent.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
			t.Fatalf("PrepareRemoveListItem() error = %v, want ErrInvalidTargetKind", err)
		}
	}
}

func TestPublicRemoveListItemSubtreeRejectsStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n  - child\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	change, err := doc.PrepareRemoveListItem(parent.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveListItem() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '*'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
