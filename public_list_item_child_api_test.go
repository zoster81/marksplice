package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicAppendListItemChildUsesCandidateParentSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   []byte
		parent   string
		child    string
		fragment []byte
		want     []byte
	}{
		{
			name:     "unordered parent accepts unordered child",
			source:   []byte("- parent\n- tail\n"),
			parent:   "parent",
			child:    "child",
			fragment: []byte("  + child\n"),
			want:     []byte("- parent\n  + child\n- tail\n"),
		},
		{
			name:     "ordered marker width and spacing drive child indentation",
			source:   []byte("10.  parent\n11.  tail\n"),
			parent:   "parent",
			child:    "child π",
			fragment: []byte("     - child π\r\n"),
			want:     []byte("10.  parent\n     - child π\r\n11.  tail\n"),
		},
		{
			name:     "nested leaf becomes parent",
			source:   []byte("1. outer\n   - parent\n   - tail\n2. end\n"),
			parent:   "parent",
			child:    "child",
			fragment: []byte("     1) child\n"),
			want:     []byte("1. outer\n   - parent\n     1) child\n   - tail\n2. end\n"),
		},
		{
			name:     "blockquote parent relation is container-aware",
			source:   []byte("> - parent\n> - tail\n"),
			parent:   "parent",
			child:    "child",
			fragment: []byte(">   + child\n"),
			want:     []byte("> - parent\n>   + child\n> - tail\n"),
		},
		{
			name:     "task child is allowed",
			source:   []byte("- parent\n- tail\n"),
			parent:   "parent",
			child:    "[ ] child task",
			fragment: []byte("  - [ ] child task\n"),
			want:     []byte("- parent\n  - [ ] child task\n- tail\n"),
		},
		{
			name:     "final child may omit terminator when parent is terminated",
			source:   []byte("- parent\n"),
			parent:   "parent",
			child:    "child",
			fragment: []byte("  - child"),
			want:     []byte("- parent\n  - child"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			parent := publicListItemByContent(t, doc, tt.source, tt.parent)
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
			updatedParent := publicListItemByContent(t, updated, got, tt.parent)
			if !updatedParent.HasChildren() {
				t.Fatal("parent HasChildren() = false after child insertion")
			}
			updatedChild := publicListItemByContent(t, updated, got, tt.child)
			if updatedChild.HasChildren() {
				t.Fatal("inserted child HasChildren() = true, want false")
			}
		})
	}
}

func TestPublicAppendListItemChildRejectsNonChildAndAmbiguousFragments(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")

	invalid := []struct {
		name     string
		fragment []byte
	}{
		{name: "empty", fragment: nil},
		{name: "same-level sibling", fragment: []byte("- child\n")},
		{name: "ordered child cannot interrupt parent with non-one start", fragment: []byte("  7) child\n")},
		{name: "plain text continuation", fragment: []byte("  child\n")},
		{name: "multiple child items", fragment: []byte("  - one\n  - two\n")},
		{name: "preamble newline", fragment: []byte("\n  - child\n")},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := doc.PrepareAppendListItemChild(parent.ID(), tt.fragment); !errors.Is(err, marksplice.ErrInvalidReplacement) {
				t.Fatalf("PrepareAppendListItemChild() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}

	unterminated := []byte("- parent")
	unterminatedDoc, err := marksplice.Parse(unterminated)
	if err != nil {
		t.Fatalf("Parse(unterminated) error = %v", err)
	}
	unterminatedParent := publicListItemByContent(t, unterminatedDoc, unterminated, "parent")
	if _, err := unterminatedDoc.PrepareAppendListItemChild(unterminatedParent.ID(), []byte("  - child\n")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("unterminated parent error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPublicAppendListItemChildRejectsInvalidTargetAndStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("- parent\n- tail\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	parent := publicListItemByContent(t, doc, source, "parent")
	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}

	if _, err := doc.PrepareAppendListItemChild(marksplice.NodeID{}, []byte("  - child\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("missing parent error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareAppendListItemChild(paragraph.ID(), []byte("  - child\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("wrong parent error = %v, want ErrInvalidTargetKind", err)
	}
	change, err := doc.PrepareAppendListItemChild(parent.ID(), []byte("  - child\n"))
	if err != nil {
		t.Fatalf("PrepareAppendListItemChild() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '*'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
