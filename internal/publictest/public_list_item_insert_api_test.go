package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicInsertListItemBeforeAndAfterPreserveSiblingSourceShape(t *testing.T) {
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
			name:     "before unordered sibling",
			source:   []byte("- alpha\n- beta\n"),
			anchor:   "beta",
			fragment: []byte("- inserted\n"),
			prepare: func(doc *marksplice.Document, id marksplice.NodeID, fragment []byte) (marksplice.ChangeSet, error) {
				return doc.PrepareInsertListItemBefore(id, fragment)
			},
			want: []byte("- alpha\n- inserted\n- beta\n"),
		},
		{
			name:     "after ordered sibling keeps caller number",
			source:   []byte("7) alpha\n8) beta\n"),
			anchor:   "alpha",
			fragment: []byte("42) inserted\n"),
			prepare: func(doc *marksplice.Document, id marksplice.NodeID, fragment []byte) (marksplice.ChangeSet, error) {
				return doc.PrepareInsertListItemAfter(id, fragment)
			},
			want: []byte("7) alpha\n42) inserted\n8) beta\n"),
		},
		{
			name:     "before nested CRLF Unicode sibling",
			source:   []byte("1. parent\r\n   - alpha\r\n   - beta\r\n2. tail\r\n"),
			anchor:   "beta",
			fragment: []byte("   - inserted π\r\n"),
			prepare: func(doc *marksplice.Document, id marksplice.NodeID, fragment []byte) (marksplice.ChangeSet, error) {
				return doc.PrepareInsertListItemBefore(id, fragment)
			},
			want: []byte("1. parent\r\n   - alpha\r\n   - inserted π\r\n   - beta\r\n2. tail\r\n"),
		},
		{
			name:     "after blockquote sibling",
			source:   []byte("> - alpha\n> - beta\n"),
			anchor:   "alpha",
			fragment: []byte("> - inserted\n"),
			prepare: func(doc *marksplice.Document, id marksplice.NodeID, fragment []byte) (marksplice.ChangeSet, error) {
				return doc.PrepareInsertListItemAfter(id, fragment)
			},
			want: []byte("> - alpha\n> - inserted\n> - beta\n"),
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
			change, err := tt.prepare(doc, anchor.ID(), tt.fragment)
			if err != nil {
				t.Fatalf("prepare insertion error = %v", err)
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
			if publicListItemByContent(t, updated, got, string(listItemFragmentContent(t, tt.fragment))).ID().String() == "" {
				t.Fatal("inserted sibling was not promoted after candidate parse")
			}
		})
	}
}

func TestPublicInsertListItemSupportsTaskFragment(t *testing.T) {
	t.Parallel()

	source := []byte("- keep\n- tail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchor := publicListItemByContent(t, doc, source, "tail")
	change, err := doc.PrepareInsertListItemBefore(anchor.ID(), []byte("- [ ] inserted task\n"))
	if err != nil {
		t.Fatalf("PrepareInsertListItemBefore() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, []byte("- keep\n- [ ] inserted task\n- tail\n")) {
		t.Fatalf("result = %q", got)
	}
}

func TestPublicInsertListItemRejectsWrongFragmentShapeAndUnsafeBoundaries(t *testing.T) {
	t.Parallel()

	nestedSource := []byte("1. parent\n   - alpha\n   - beta\n2. tail\n")
	doc, err := marksplice.Parse(nestedSource)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	anchor := publicListItemByContent(t, doc, nestedSource, "beta")
	invalid := []struct {
		name     string
		fragment []byte
	}{
		{name: "empty", fragment: nil},
		{name: "wrong indentation", fragment: []byte("- inserted\n")},
		{name: "different marker", fragment: []byte("   * inserted\n")},
		{name: "different list kind", fragment: []byte("   1. inserted\n")},
		{name: "multiple items", fragment: []byte("   - one\n   - two\n")},
		{name: "plain text", fragment: []byte("inserted\n")},
		{name: "unterminated before fragment", fragment: []byte("   - inserted")},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := doc.PrepareInsertListItemBefore(anchor.ID(), tt.fragment); !errors.Is(err, marksplice.ErrInvalidReplacement) {
				t.Fatalf("PrepareInsertListItemBefore() error = %v, want ErrInvalidReplacement", err)
			}
		})
	}

	unsafeEOF := []byte("- alpha\n- beta")
	unsafeDoc, err := marksplice.Parse(unsafeEOF)
	if err != nil {
		t.Fatalf("Parse(unsafe EOF) error = %v", err)
	}
	beta := publicListItemByContent(t, unsafeDoc, unsafeEOF, "beta")
	if _, err := unsafeDoc.PrepareInsertListItemAfter(beta.ID(), []byte("- inserted\n")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareInsertListItemAfter(unsafe EOF) error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPublicInsertListItemRejectsInvalidTargetAndStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("- alpha\n- beta\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	alpha := publicListItemByContent(t, doc, source, "alpha")
	var paragraph marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindParagraph {
			paragraph = node
			break
		}
	}
	if _, err := doc.PrepareInsertListItemBefore(marksplice.NodeID{}, []byte("- new\n")); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("missing anchor error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareInsertListItemAfter(paragraph.ID(), []byte("- new\n")); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("wrong anchor error = %v, want ErrInvalidTargetKind", err)
	}
	change, err := doc.PrepareInsertListItemAfter(alpha.ID(), []byte("- inserted\n"))
	if err != nil {
		t.Fatalf("PrepareInsertListItemAfter() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '*'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func listItemFragmentContent(t *testing.T, fragment []byte) []byte {
	t.Helper()
	doc, err := marksplice.Parse(fragment)
	if err != nil {
		t.Fatalf("Parse(fragment) error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindListItem {
			continue
		}
		item, ok := doc.ListItem(node.ID())
		if ok {
			return fragment[item.Range().Start:item.Range().End]
		}
	}
	t.Fatal("fragment list item not found")
	return nil
}
