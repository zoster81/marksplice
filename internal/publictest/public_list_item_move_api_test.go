package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicMoveListItemBeforeAndAfterPreserveExactLeafLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  []byte
		moved   string
		anchor  string
		prepare func(*marksplice.Document, marksplice.NodeID, marksplice.NodeID) (marksplice.ChangeSet, error)
		want    []byte
	}{
		{
			name:   "forward unordered after",
			source: []byte("- alpha\n- move\n- beta\n"),
			moved:  "move",
			anchor: "beta",
			prepare: func(doc *marksplice.Document, moved, anchor marksplice.NodeID) (marksplice.ChangeSet, error) {
				return doc.PrepareMoveListItemAfter(moved, anchor)
			},
			want: []byte("- alpha\n- beta\n- move\n"),
		},
		{
			name:   "backward ordered before preserves number",
			source: []byte("7) alpha\n8) beta\n9) move\n"),
			moved:  "move",
			anchor: "beta",
			prepare: func(doc *marksplice.Document, moved, anchor marksplice.NodeID) (marksplice.ChangeSet, error) {
				return doc.PrepareMoveListItemBefore(moved, anchor)
			},
			want: []byte("7) alpha\n9) move\n8) beta\n"),
		},
		{
			name:   "nested CRLF Unicode reparent across ordered parents",
			source: []byte("1. first\r\n   - move π\r\n2. second\r\n   - anchor\r\n3. tail\r\n"),
			moved:  "move π",
			anchor: "anchor",
			prepare: func(doc *marksplice.Document, moved, anchor marksplice.NodeID) (marksplice.ChangeSet, error) {
				return doc.PrepareMoveListItemAfter(moved, anchor)
			},
			want: []byte("1. first\r\n2. second\r\n   - anchor\r\n   - move π\r\n3. tail\r\n"),
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
			change, err := tt.prepare(doc, moved.ID(), anchor.ID())
			if err != nil {
				t.Fatalf("prepare move error = %v", err)
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
			if publicListItemByContent(t, updated, got, tt.moved).ID().String() == "" {
				t.Fatal("moved item is not a promoted leaf in result")
			}
		})
	}
}

func TestPublicMoveListItemAdjacentNoOpRemainsSnapshotBound(t *testing.T) {
	t.Parallel()

	source := []byte("- alpha\n- beta\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	alpha := publicListItemByContent(t, doc, source, "alpha")
	beta := publicListItemByContent(t, doc, source, "beta")

	for _, prepare := range []func() (marksplice.ChangeSet, error){
		func() (marksplice.ChangeSet, error) { return doc.PrepareMoveListItemBefore(alpha.ID(), beta.ID()) },
		func() (marksplice.ChangeSet, error) { return doc.PrepareMoveListItemAfter(beta.ID(), alpha.ID()) },
	} {
		change, err := prepare()
		if err != nil {
			t.Fatalf("prepare no-op move error = %v", err)
		}
		got, err := change.Apply(source)
		if err != nil {
			t.Fatalf("Apply(no-op) error = %v", err)
		}
		if !bytes.Equal(got, source) {
			t.Fatalf("no-op move changed source: %q", got)
		}
		stale := append([]byte(nil), source...)
		stale[0] = '*'
		if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
			t.Fatalf("Apply(stale no-op) error = %v, want ErrSourceConflict", err)
		}
	}
}

func TestPublicMoveListItemRejectsSelfShapeMismatchAndInvalidTargets(t *testing.T) {
	t.Parallel()

	source := []byte("- root\n\n1. parent\n   - nested\n2. tail\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	root := publicListItemByContent(t, doc, source, "root")
	nested := publicListItemByContent(t, doc, source, "nested")

	if _, err := doc.PrepareMoveListItemBefore(root.ID(), root.ID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("self move error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareMoveListItemAfter(root.ID(), nested.ID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("prefix mismatch error = %v, want ErrInvalidReplacement", err)
	}

	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}
	if _, err := doc.PrepareMoveListItemBefore(marksplice.NodeID{}, root.ID()); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("missing source error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareMoveListItemBefore(root.ID(), marksplice.NodeID{}); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("missing anchor error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareMoveListItemAfter(root.ID(), paragraph.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("wrong anchor error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestPublicMoveListItemFailsClosedForUnterminatedMovedLine(t *testing.T) {
	t.Parallel()

	source := []byte("- anchor\n- move")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchor := publicListItemByContent(t, doc, source, "anchor")
	moved := publicListItemByContent(t, doc, source, "move")
	if _, err := doc.PrepareMoveListItemBefore(moved.ID(), anchor.ID()); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("unterminated move error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPublicMoveListItemPreparedChangeRejectsStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("- alpha\n- move\n- beta\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	moved := publicListItemByContent(t, doc, source, "move")
	beta := publicListItemByContent(t, doc, source, "beta")
	change, err := doc.PrepareMoveListItemAfter(moved.ID(), beta.ID())
	if err != nil {
		t.Fatalf("PrepareMoveListItemAfter() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '*'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
