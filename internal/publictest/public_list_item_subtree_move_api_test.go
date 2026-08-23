package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicMoveCompleteListItemSubtreeBeforeAndAfter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		moved  string
		anchor string
		after  bool
		want   []byte
	}{
		{
			name:   "forward root parent after leaf anchor",
			source: []byte("- parent\n  - child\n    - grandchild\n- anchor\n- tail\n"),
			moved:  "parent",
			anchor: "anchor",
			after:  true,
			want:   []byte("- anchor\n- parent\n  - child\n    - grandchild\n- tail\n"),
		},
		{
			name:   "backward root parent before parent anchor",
			source: []byte("- anchor\n  - anchor child\n- parent\n  - child\n- tail\n"),
			moved:  "parent",
			anchor: "anchor",
			want:   []byte("- parent\n  - child\n- anchor\n  - anchor child\n- tail\n"),
		},
		{
			name:   "nested subtree reparents with descendants intact",
			source: []byte("1. first\n   - parent\n     - child π\n2. second\n   - anchor\n3. tail\n"),
			moved:  "parent",
			anchor: "anchor",
			after:  true,
			want:   []byte("1. first\n2. second\n   - anchor\n   - parent\n     - child π\n3. tail\n"),
		},
		{
			name:   "nested subtree reorder under same parent",
			source: []byte("- root\n  - parent\n    - child\n  - anchor\n  - tail\n"),
			moved:  "parent",
			anchor: "anchor",
			after:  true,
			want:   []byte("- root\n  - anchor\n  - parent\n    - child\n  - tail\n"),
		},
		{
			name:   "ordered CRLF Unicode preserves root number",
			source: []byte("10. parent π\r\n    - child\r\n11. anchor\r\n12. tail\r\n"),
			moved:  "parent π",
			anchor: "anchor",
			after:  true,
			want:   []byte("11. anchor\r\n10. parent π\r\n    - child\r\n12. tail\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			moved := publicListItemByContent(t, doc, tt.source, tt.moved)
			anchor := publicListItemByContent(t, doc, tt.source, tt.anchor)
			if !moved.HasChildren() {
				t.Fatal("moved HasChildren() = false, want parent subtree")
			}

			var change marksplice.ChangeSet
			if tt.after {
				change, err = doc.PrepareMoveListItemAfter(moved.ID(), anchor.ID())
			} else {
				change, err = doc.PrepareMoveListItemBefore(moved.ID(), anchor.ID())
			}
			if err != nil {
				t.Fatalf("PrepareMoveListItem...() error = %v", err)
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
			updatedMoved := publicListItemByContent(t, updated, got, tt.moved)
			updatedAnchor := publicListItemByContent(t, updated, got, tt.anchor)
			if !updatedMoved.HasChildren() {
				t.Fatal("moved subtree root lost children")
			}
			movedParent, movedHasParent := updatedMoved.ParentID()
			anchorParent, anchorHasParent := updatedAnchor.ParentID()
			if movedHasParent != anchorHasParent || movedParent != anchorParent {
				t.Fatalf("moved ParentID() = (%v,%v), anchor = (%v,%v)", movedParent, movedHasParent, anchorParent, anchorHasParent)
			}
			if tt.name == "nested subtree reparents with descendants intact" {
				child := publicListItemByContent(t, updated, got, "child π")
				if parentID, ok := child.ParentID(); !ok || parentID != updatedMoved.ID() {
					t.Fatalf("child ParentID() = (%v,%v), want moved root %v", parentID, ok, updatedMoved.ID())
				}
			}
		})
	}
}

func TestPublicMoveCompleteListItemSubtreeAdjacentNoOpRemainsSnapshotBound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		moved  string
		anchor string
		after  bool
	}{
		{
			name:   "subtree already before anchor",
			source: []byte("- parent\n  - child\n- anchor\n- tail\n"),
			moved:  "parent",
			anchor: "anchor",
		},
		{
			name:   "subtree already after anchor",
			source: []byte("- anchor\n- parent\n  - child\n- tail\n"),
			moved:  "parent",
			anchor: "anchor",
			after:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			moved := publicListItemByContent(t, doc, tt.source, tt.moved)
			anchor := publicListItemByContent(t, doc, tt.source, tt.anchor)
			var change marksplice.ChangeSet
			if tt.after {
				change, err = doc.PrepareMoveListItemAfter(moved.ID(), anchor.ID())
			} else {
				change, err = doc.PrepareMoveListItemBefore(moved.ID(), anchor.ID())
			}
			if err != nil {
				t.Fatalf("prepare no-op move error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply(no-op) error = %v", err)
			}
			if !bytes.Equal(got, tt.source) {
				t.Fatalf("no-op changed source: %q", got)
			}
			stale := append([]byte(nil), tt.source...)
			stale[0] = '*'
			if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
				t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
			}
		})
	}
}

func TestPublicMoveCompleteListItemSubtreeRejectsIncompleteSourceOrAnchor(t *testing.T) {
	t.Parallel()

	t.Run("incomplete source", func(t *testing.T) {
		source := []byte("- parent\n  - complex\n\n    second paragraph\n- anchor\n")
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		parent := publicListItemByContent(t, doc, source, "parent")
		anchor := publicListItemByContent(t, doc, source, "anchor")
		if _, err := doc.PrepareMoveListItemAfter(parent.ID(), anchor.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
			t.Fatalf("PrepareMoveListItemAfter() error = %v, want ErrInvalidTargetKind", err)
		}
	})

	t.Run("incomplete anchor", func(t *testing.T) {
		source := []byte("- parent\n  - child\n- anchor\n  - complex\n\n    second paragraph\n")
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		parent := publicListItemByContent(t, doc, source, "parent")
		anchor := publicListItemByContent(t, doc, source, "anchor")
		if _, err := doc.PrepareMoveListItemBefore(parent.ID(), anchor.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
			t.Fatalf("PrepareMoveListItemBefore() error = %v, want ErrInvalidTargetKind", err)
		}
	})
}

func TestPublicMoveCompleteListItemSubtreeRejectsOverlappingSourceAndAnchor(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n  - child\n    - grandchild\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	child := publicListItemByContent(t, doc, source, "child")

	if _, err := doc.PrepareMoveListItemBefore(parent.ID(), child.ID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("move ancestor before descendant error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareMoveListItemAfter(child.ID(), parent.ID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("move descendant after ancestor error = %v, want ErrInvalidReplacement", err)
	}
}
