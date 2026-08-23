package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicInsertListItemSiblingAroundCompleteParentSubtree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   []byte
		anchor   string
		fragment []byte
		after    bool
		inserted string
		want     []byte
	}{
		{
			name:     "before root parent",
			source:   []byte("- parent\n  - child\n- tail\n"),
			anchor:   "parent",
			fragment: []byte("- before\n"),
			inserted: "before",
			want:     []byte("- before\n- parent\n  - child\n- tail\n"),
		},
		{
			name:     "after root parent subtree",
			source:   []byte("- parent\n  - child\n    - grandchild\n- tail\n"),
			anchor:   "parent",
			fragment: []byte("- after\n"),
			after:    true,
			inserted: "after",
			want:     []byte("- parent\n  - child\n    - grandchild\n- after\n- tail\n"),
		},
		{
			name:     "nested sibling keeps same immediate parent",
			source:   []byte("- root\n  - parent\n    - child\n  - tail\n"),
			anchor:   "parent",
			fragment: []byte("  - after\n"),
			after:    true,
			inserted: "after",
			want:     []byte("- root\n  - parent\n    - child\n  - after\n  - tail\n"),
		},
		{
			name:     "ordered CRLF Unicode",
			source:   []byte("10. parent π\r\n    - child\r\n11. tail\r\n"),
			anchor:   "parent π",
			fragment: []byte("12. sibling π\r\n"),
			after:    true,
			inserted: "sibling π",
			want:     []byte("10. parent π\r\n    - child\r\n12. sibling π\r\n11. tail\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			anchor := publicListItemByContent(t, doc, tt.source, tt.anchor)
			if !anchor.HasChildren() {
				t.Fatal("anchor HasChildren() = false, want true")
			}
			var change marksplice.ChangeSet
			if tt.after {
				change, err = doc.PrepareInsertListItemAfter(anchor.ID(), tt.fragment)
			} else {
				change, err = doc.PrepareInsertListItemBefore(anchor.ID(), tt.fragment)
			}
			if err != nil {
				t.Fatalf("PrepareInsertListItem...() error = %v", err)
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
			inserted := publicListItemByContent(t, updated, got, tt.inserted)
			updatedAnchor := publicListItemByContent(t, updated, got, tt.anchor)
			insertedParent, insertedHasParent := inserted.ParentID()
			anchorParent, anchorHasParent := updatedAnchor.ParentID()
			if insertedHasParent != anchorHasParent || insertedParent != anchorParent {
				t.Fatalf("inserted ParentID() = (%v,%v), anchor = (%v,%v)", insertedParent, insertedHasParent, anchorParent, anchorHasParent)
			}
		})
	}
}

func TestPublicInsertListItemSiblingRejectsIncompleteParentAnchor(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n  - complex\n\n    second paragraph\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	if _, err := doc.PrepareInsertListItemBefore(parent.ID(), []byte("- before\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareInsertListItemBefore() error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareInsertListItemAfter(parent.ID(), []byte("- after\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareInsertListItemAfter() error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestPublicInsertListItemSiblingAroundParentRejectsStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n  - child\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	change, err := doc.PrepareInsertListItemAfter(parent.ID(), []byte("- sibling\n"))
	if err != nil {
		t.Fatalf("PrepareInsertListItemAfter() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '*'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
