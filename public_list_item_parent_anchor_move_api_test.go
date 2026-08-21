package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicMoveLeafListItemAroundCompleteParentAnchor(t *testing.T) {
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
			name:   "forward after root parent subtree",
			source: []byte("- move\n- parent\n  - child\n    - grandchild\n- tail\n"),
			moved:  "move",
			anchor: "parent",
			after:  true,
			want:   []byte("- parent\n  - child\n    - grandchild\n- move\n- tail\n"),
		},
		{
			name:   "backward before root parent subtree",
			source: []byte("- parent\n  - child\n- move\n- tail\n"),
			moved:  "move",
			anchor: "parent",
			want:   []byte("- move\n- parent\n  - child\n- tail\n"),
		},
		{
			name:   "nested reparent after complete parent",
			source: []byte("1. first\n   - move π\n2. second\n   - parent\n     - child\n3. tail\n"),
			moved:  "move π",
			anchor: "parent",
			after:  true,
			want:   []byte("1. first\n2. second\n   - parent\n     - child\n   - move π\n3. tail\n"),
		},
		{
			name:   "ordered CRLF preserves source number",
			source: []byte("10. move π\r\n11. parent\r\n    - child\r\n12. tail\r\n"),
			moved:  "move π",
			anchor: "parent",
			after:  true,
			want:   []byte("11. parent\r\n    - child\r\n10. move π\r\n12. tail\r\n"),
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
			if moved.HasChildren() {
				t.Fatal("moved item HasChildren() = true, want leaf")
			}
			if !anchor.HasChildren() {
				t.Fatal("anchor HasChildren() = false, want parent")
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
			movedParent, movedHasParent := updatedMoved.ParentID()
			anchorParent, anchorHasParent := updatedAnchor.ParentID()
			if movedHasParent != anchorHasParent || movedParent != anchorParent {
				t.Fatalf("moved ParentID() = (%v,%v), anchor = (%v,%v)", movedParent, movedHasParent, anchorParent, anchorHasParent)
			}
		})
	}
}

func TestPublicMoveLeafListItemAroundParentAdjacentNoOpRemainsSnapshotBound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		moved  string
		anchor string
		after  bool
	}{
		{
			name:   "already before parent",
			source: []byte("- move\n- parent\n  - child\n- tail\n"),
			moved:  "move",
			anchor: "parent",
		},
		{
			name:   "already after parent subtree",
			source: []byte("- parent\n  - child\n- move\n- tail\n"),
			moved:  "move",
			anchor: "parent",
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

func TestPublicMoveLeafListItemRejectsIncompleteParentAnchor(t *testing.T) {
	t.Parallel()

	source := []byte("- move\n- parent\n  - complex\n\n    second paragraph\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	moved := publicListItemByContent(t, doc, source, "move")
	parent := publicListItemByContent(t, doc, source, "parent")
	if _, err := doc.PrepareMoveListItemBefore(moved.ID(), parent.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareMoveListItemBefore() error = %v, want ErrInvalidTargetKind", err)
	}
	if _, err := doc.PrepareMoveListItemAfter(moved.ID(), parent.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareMoveListItemAfter() error = %v, want ErrInvalidTargetKind", err)
	}
}
