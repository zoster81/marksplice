package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicReplaceCompleteListItemSubtree(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		target      string
		replacement []byte
		root        string
		child       string
		want        []byte
	}{
		{
			name:        "root subtree",
			source:      []byte("- target\n  - old child\n- tail\n"),
			target:      "target",
			replacement: []byte("- replaced\n  + child two\n    - grandchild\n"),
			root:        "replaced",
			child:       "child two",
			want:        []byte("- replaced\n  + child two\n    - grandchild\n- tail\n"),
		},
		{
			name:        "nested subtree preserves external parent",
			source:      []byte("- outer\n  - target\n    - old child\n  - tail\n"),
			target:      "target",
			replacement: []byte("  - replaced \u03c0\n    - child\n      - grandchild\n"),
			root:        "replaced \u03c0",
			child:       "child",
			want:        []byte("- outer\n  - replaced \u03c0\n    - child\n      - grandchild\n  - tail\n"),
		},
		{
			name:        "ordered CRLF preserves delimiter while caller changes numbering",
			source:      []byte("10. target\r\n    - old\r\n11. tail\r\n"),
			target:      "target",
			replacement: []byte("42. replaced \u03c0\r\n    + child\r\n"),
			root:        "replaced \u03c0",
			child:       "child",
			want:        []byte("42. replaced \u03c0\r\n    + child\r\n11. tail\r\n"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			target := publicListItemByContent(t, doc, tt.source, tt.target)
			change, err := doc.PrepareReplaceListItemSubtree(target.ID(), tt.replacement)
			if err != nil {
				t.Fatalf("PrepareReplaceListItemSubtree() error = %v", err)
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
			if parentID, ok := insertedChild.ParentID(); !ok || parentID != insertedRoot.ID() {
				t.Fatalf("child ParentID() = (%v,%v), want (%v,true)", parentID, ok, insertedRoot.ID())
			}
			if tt.name == "nested subtree preserves external parent" {
				outer := publicListItemByContent(t, updated, got, "outer")
				if parentID, ok := insertedRoot.ParentID(); !ok || parentID != outer.ID() {
					t.Fatalf("replacement root ParentID() = (%v,%v), want (%v,true)", parentID, ok, outer.ID())
				}
			}
		})
	}
}

func TestPublicReplaceListItemSubtreeRejectsAmbiguousOrUnsafeReplacement(t *testing.T) {
	t.Parallel()

	source := []byte("- target\n  - old child\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	target := publicListItemByContent(t, doc, source, "target")

	tests := []struct {
		name        string
		replacement []byte
	}{
		{name: "empty belongs to removal", replacement: nil},
		{name: "multiple sibling roots", replacement: []byte("- one\n- two\n")},
		{name: "different root marker", replacement: []byte("* replaced\n  - child\n")},
		{name: "trailing bytes outside subtree", replacement: []byte("- replaced\n  - child\n\n")},
		{name: "unsupported descendant", replacement: []byte("- replaced\n  - complex\n\n    second paragraph\n")},
		{name: "unterminated replacement consumes following source", replacement: []byte("- replaced")},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := doc.PrepareReplaceListItemSubtree(target.ID(), tt.replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
				t.Fatalf("PrepareReplaceListItemSubtree() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}

	nestedSource := []byte("- outer\n  - target\n  - tail\n")
	nestedDoc, err := marksplice.Parse(nestedSource)
	if err != nil {
		t.Fatalf("Parse(nested) error = %v", err)
	}
	nestedTarget := publicListItemByContent(t, nestedDoc, nestedSource, "target")
	if _, err := nestedDoc.PrepareReplaceListItemSubtree(nestedTarget.ID(), []byte("- replaced\n")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("wrong-parent replacement error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPublicReplaceListItemSubtreeRejectsInvalidTargetAndStaleSource(t *testing.T) {
	t.Parallel()

	incompleteSource := []byte("- target\n  - complex\n\n    second paragraph\n- tail\n")
	incompleteDoc, err := marksplice.Parse(incompleteSource)
	if err != nil {
		t.Fatalf("Parse(incomplete) error = %v", err)
	}
	incompleteTarget := publicListItemByContent(t, incompleteDoc, incompleteSource, "target")
	if _, err := incompleteDoc.PrepareReplaceListItemSubtree(incompleteTarget.ID(), []byte("- replacement\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("incomplete target error = %v, want ErrInvalidTargetKind", err)
	}

	source := []byte("- target\n- tail\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	target := publicListItemByContent(t, doc, source, "target")
	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}
	if _, err := doc.PrepareReplaceListItemSubtree(marksplice.NodeID{}, []byte("- replacement\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("missing target error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareReplaceListItemSubtree(paragraph.ID(), []byte("- replacement\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("wrong target error = %v, want ErrInvalidTargetKind", err)
	}

	change, err := doc.PrepareReplaceListItemSubtree(target.ID(), []byte("- replacement\n"))
	if err != nil {
		t.Fatalf("PrepareReplaceListItemSubtree() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '*'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
