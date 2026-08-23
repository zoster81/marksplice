package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicInsertSectionBeforeAndAfterPreserveSiblingHierarchyAndBytes(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nintro\n## Alpha\na\n### Deep\nd\n## Beta\nb\n# Tail\ntail\n")
	tests := []struct {
		name     string
		anchor   string
		fragment []byte
		prepare  func(*marksplice.Document, marksplice.NodeID, []byte) (marksplice.ChangeSet, error)
		want     []byte
		inserted string
		child    string
	}{
		{
			name:     "before sibling subtree",
			anchor:   "Beta",
			fragment: []byte("## Before\nbody\n### Before Child\nchild\n"),
			prepare: func(doc *marksplice.Document, id marksplice.NodeID, fragment []byte) (marksplice.ChangeSet, error) {
				return doc.PrepareInsertSectionBefore(id, fragment)
			},
			want:     []byte("# Root\nintro\n## Alpha\na\n### Deep\nd\n## Before\nbody\n### Before Child\nchild\n## Beta\nb\n# Tail\ntail\n"),
			inserted: "Before",
			child:    "Before Child",
		},
		{
			name:     "after complete sibling subtree",
			anchor:   "Alpha",
			fragment: []byte("## After\nbody\n### After Child\nchild\n"),
			prepare: func(doc *marksplice.Document, id marksplice.NodeID, fragment []byte) (marksplice.ChangeSet, error) {
				return doc.PrepareInsertSectionAfter(id, fragment)
			},
			want:     []byte("# Root\nintro\n## Alpha\na\n### Deep\nd\n## After\nbody\n### After Child\nchild\n## Beta\nb\n# Tail\ntail\n"),
			inserted: "After",
			child:    "After Child",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			sections := publicSectionsByHeadingText(t, doc, source, doc.Sections())
			anchor := sections[tt.anchor]
			root := sections["Root"]
			if anchor.HeadingID().String() == "" || root.HeadingID().String() == "" {
				t.Fatalf("anchor/root not found")
			}

			insertAt := anchor.Range().Start
			if tt.name == "after complete sibling subtree" {
				insertAt = anchor.Range().End
			}
			prefix := append([]byte(nil), source[:insertAt]...)
			suffix := append([]byte(nil), source[insertAt:]...)

			change, err := tt.prepare(doc, anchor.HeadingID(), tt.fragment)
			if err != nil {
				t.Fatalf("prepare insert error = %v", err)
			}
			got, err := change.Apply(source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result bytes differ\n got: %q\nwant: %q", got, tt.want)
			}
			if !bytes.Equal(got[:len(prefix)], prefix) || !bytes.Equal(got[len(prefix)+len(tt.fragment):], suffix) {
				t.Fatal("section insertion modified bytes outside the zero-width insertion point")
			}

			updated, err := marksplice.Parse(got)
			if err != nil {
				t.Fatalf("Parse(result) error = %v", err)
			}
			byHeading := publicSectionsByHeadingText(t, updated, got, updated.Sections())
			inserted := byHeading[tt.inserted]
			parent, ok := inserted.ParentHeadingID()
			updatedRoot := byHeading["Root"]
			if !ok || parent != updatedRoot.HeadingID() {
				t.Fatalf("inserted parent = %v, %v; want Root", parent, ok)
			}
			child := byHeading[tt.child]
			childParent, ok := child.ParentHeadingID()
			if !ok || childParent != inserted.HeadingID() {
				t.Fatalf("inserted child parent = %v, %v; want inserted root", childParent, ok)
			}
		})
	}
}

func TestPublicInsertSectionSupportsRootPreambleCRLFSetextAndEOF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   []byte
		anchor   string
		fragment []byte
		prepare  func(*marksplice.Document, marksplice.NodeID, []byte) (marksplice.ChangeSet, error)
		want     []byte
	}{
		{
			name:     "before first root keeps preamble",
			source:   []byte("preamble\n\n# One\nbody\n# Two\ntail\n"),
			anchor:   "One",
			fragment: []byte("# Before\nnew\n"),
			prepare: func(doc *marksplice.Document, id marksplice.NodeID, fragment []byte) (marksplice.ChangeSet, error) {
				return doc.PrepareInsertSectionBefore(id, fragment)
			},
			want: []byte("preamble\n\n# Before\nnew\n# One\nbody\n# Two\ntail\n"),
		},
		{
			name:     "after final root appends through EOF",
			source:   []byte("# One\nbody\n# Last\nold"),
			anchor:   "Last",
			fragment: []byte("# Final\nnew"),
			prepare: func(doc *marksplice.Document, id marksplice.NodeID, fragment []byte) (marksplice.ChangeSet, error) {
				return doc.PrepareInsertSectionAfter(id, fragment)
			},
			want: []byte("# One\nbody\n# Last\nold# Final\nnew"),
		},
		{
			name:     "before CRLF child accepts separated Setext sibling",
			source:   []byte("# Root\r\nintro\r\n\r\n## Child\r\nbody\r\n## Tail\r\ntail\r\n"),
			anchor:   "Child",
			fragment: []byte("Inserted\r\n--------\r\nπ body\r\n"),
			prepare: func(doc *marksplice.Document, id marksplice.NodeID, fragment []byte) (marksplice.ChangeSet, error) {
				return doc.PrepareInsertSectionBefore(id, fragment)
			},
			want: []byte("# Root\r\nintro\r\n\r\nInserted\r\n--------\r\nπ body\r\n## Child\r\nbody\r\n## Tail\r\ntail\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doc, err := marksplice.Parse(tt.source)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			anchor := publicSectionsByHeadingText(t, doc, tt.source, doc.Sections())[tt.anchor]
			change, err := tt.prepare(doc, anchor.HeadingID(), tt.fragment)
			if tt.name == "after final root appends through EOF" {
				if !errors.Is(err, marksplice.ErrInvalidReplacement) {
					t.Fatalf("prepare EOF unsafe join error = %v, want ErrInvalidReplacement", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("prepare insert error = %v", err)
			}
			got, err := change.Apply(tt.source)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Fatalf("result = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPublicInsertSectionAfterFinalRootSupportsSafeEOFFragment(t *testing.T) {
	t.Parallel()

	source := []byte("# One\nbody\n# Last\nold\n")
	fragment := []byte("# Final\nnew")
	want := []byte("# One\nbody\n# Last\nold\n# Final\nnew")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	last := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Last"]
	change, err := doc.PrepareInsertSectionAfter(last.HeadingID(), fragment)
	if err != nil {
		t.Fatalf("PrepareInsertSectionAfter() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPublicInsertSectionRejectsInvalidFragmentsUnsafeJoinsAndTargets(t *testing.T) {
	t.Parallel()

	source := []byte("# Root\nintro\n## Child\nbody\nSibling\n-------\ntail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	child := publicSectionsByHeadingText(t, doc, source, doc.Sections())["Child"]

	invalid := []struct {
		name     string
		fragment []byte
	}{
		{name: "empty", fragment: nil},
		{name: "wrong level", fragment: []byte("# Wrong\nbody\n")},
		{name: "preamble", fragment: []byte("text\n## New\nbody\n")},
		{name: "multiple siblings", fragment: []byte("## One\na\n## Two\nb\n")},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := doc.PrepareInsertSectionBefore(child.HeadingID(), tt.fragment); !errors.Is(err, marksplice.ErrInvalidReplacement) {
				t.Fatalf("PrepareInsertSectionBefore() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}

	if _, err := doc.PrepareInsertSectionAfter(child.HeadingID(), []byte("## New\njoined")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareInsertSectionAfter(unsafe join) error = %v, want ErrInvalidReplacement", err)
	}
	if _, err := doc.PrepareInsertSectionBefore(marksplice.NodeID{}, []byte("## New\nbody\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareInsertSectionBefore(zero) error = %v, want ErrNodeNotFound", err)
	}

	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}
	if paragraph.ID().String() == "" {
		t.Fatal("paragraph not found")
	}
	if _, err := doc.PrepareInsertSectionAfter(paragraph.ID(), []byte("## New\nbody\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareInsertSectionAfter(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}
}

func TestPublicInsertSectionPreparedChangeRejectsStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("# One\nbody\n# Two\ntail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	one := publicSectionsByHeadingText(t, doc, source, doc.Sections())["One"]
	change, err := doc.PrepareInsertSectionAfter(one.HeadingID(), []byte("# Inserted\nnew\n"))
	if err != nil {
		t.Fatalf("PrepareInsertSectionAfter() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '!'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}
