package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicInsertCompleteListItemSubtreeBeforeAndAfter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   []byte
		anchor   string
		fragment []byte
		after    bool
		root     string
		child    string
		want     []byte
	}{
		{
			name:     "before root leaf anchor",
			source:   []byte("- anchor\n- tail\n"),
			anchor:   "anchor",
			fragment: []byte("- parent\n  - child\n    - grandchild\n"),
			root:     "parent",
			child:    "child",
			want:     []byte("- parent\n  - child\n    - grandchild\n- anchor\n- tail\n"),
		},
		{
			name:     "after complete parent anchor",
			source:   []byte("- anchor\n  - anchor child\n- tail\n"),
			anchor:   "anchor",
			fragment: []byte("- parent\n  - child\n"),
			after:    true,
			root:     "parent",
			child:    "child",
			want:     []byte("- anchor\n  - anchor child\n- parent\n  - child\n- tail\n"),
		},
		{
			name:     "nested subtree keeps destination parent",
			source:   []byte("- root\n  - anchor\n  - tail\n"),
			anchor:   "anchor",
			fragment: []byte("  - parent\n    - child π\n"),
			after:    true,
			root:     "parent",
			child:    "child π",
			want:     []byte("- root\n  - anchor\n  - parent\n    - child π\n  - tail\n"),
		},
		{
			name:     "ordered CRLF Unicode preserves caller numbering",
			source:   []byte("10. anchor π\r\n11. tail\r\n"),
			anchor:   "anchor π",
			fragment: []byte("42. parent π\r\n    - child\r\n"),
			after:    true,
			root:     "parent π",
			child:    "child",
			want:     []byte("10. anchor π\r\n42. parent π\r\n    - child\r\n11. tail\r\n"),
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
			insertedRoot := publicListItemByContent(t, updated, got, tt.root)
			insertedChild := publicListItemByContent(t, updated, got, tt.child)
			if !insertedRoot.HasChildren() {
				t.Fatal("inserted root HasChildren() = false, want true")
			}
			if parentID, ok := insertedChild.ParentID(); !ok || parentID != insertedRoot.ID() {
				t.Fatalf("inserted child ParentID() = (%v,%v), want root %v", parentID, ok, insertedRoot.ID())
			}
			updatedAnchor := publicListItemByContent(t, updated, got, tt.anchor)
			rootParent, rootHasParent := insertedRoot.ParentID()
			anchorParent, anchorHasParent := updatedAnchor.ParentID()
			if rootHasParent != anchorHasParent || rootParent != anchorParent {
				t.Fatalf("inserted root ParentID() = (%v,%v), anchor = (%v,%v)", rootParent, rootHasParent, anchorParent, anchorHasParent)
			}
		})
	}
}

func TestPublicInsertCompleteListItemSubtreeRejectsInvalidStandaloneFragment(t *testing.T) {
	t.Parallel()

	source := []byte("- anchor\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchor := publicListItemByContent(t, doc, source, "anchor")

	tests := []struct {
		name     string
		fragment []byte
	}{
		{name: "multiple root siblings", fragment: []byte("- one\n- two\n")},
		{name: "trailing blank line outside subtree", fragment: []byte("- parent\n  - child\n\n")},
		{name: "incomplete unsupported descendant", fragment: []byte("- parent\n  - complex\n\n    second paragraph\n")},
		{name: "wrong sibling marker", fragment: []byte("* parent\n  * child\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := doc.PrepareInsertListItemAfter(anchor.ID(), tt.fragment); !errors.Is(err, marksplice.ErrInvalidReplacement) {
				t.Fatalf("PrepareInsertListItemAfter() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}
}

func TestPublicInsertCompleteListItemSubtreeRejectsStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("- anchor\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchor := publicListItemByContent(t, doc, source, "anchor")
	change, err := doc.PrepareInsertListItemAfter(anchor.ID(), []byte("- parent\n  - child\n"))
	if err != nil {
		t.Fatalf("PrepareInsertListItemAfter() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '*'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
