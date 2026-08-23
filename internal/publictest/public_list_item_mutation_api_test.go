package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicRemoveListItemDeletesExactPhysicalLeafLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source []byte
		target string
		want   []byte
	}{
		{
			name:   "middle CRLF Unicode item",
			source: []byte("- keep\r\n- remove π\r\n- tail\r\n"),
			target: "remove π",
			want:   []byte("- keep\r\n- tail\r\n"),
		},
		{
			name:   "nested leaf owns indentation and CRLF",
			source: []byte("1. parent\r\n   - keep\r\n   - remove π\r\n2. tail\r\n"),
			target: "remove π",
			want:   []byte("1. parent\r\n   - keep\r\n2. tail\r\n"),
		},
		{
			name:   "final EOF item has no synthetic newline",
			source: []byte("- keep\n- remove"),
			target: "remove",
			want:   []byte("- keep\n"),
		},
		{
			name:   "last nested child removal may promote its parent leaf",
			source: []byte("- parent\n  - child\n- tail\n"),
			target: "child",
			want:   []byte("- parent\n- tail\n"),
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
		})
	}
}

func TestPublicRemoveListItemPreservesSurroundingBlankLines(t *testing.T) {
	t.Parallel()

	source := []byte("before\n\n- remove\n\nafter\n")
	want := []byte("before\n\n\nafter\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	item := publicListItemByContent(t, doc, source, "remove")
	change, err := doc.PrepareRemoveListItem(item.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveListItem() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPublicRemoveListItemSupportsTaskItems(t *testing.T) {
	t.Parallel()

	source := []byte("- [ ] remove\n- [x] keep\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	item := publicListItemByContent(t, doc, source, "[ ] remove")
	change, err := doc.PrepareRemoveListItem(item.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveListItem() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if !bytes.Equal(got, []byte("- [x] keep\n")) {
		t.Fatalf("result = %q, want remaining task item", got)
	}
}

func TestPublicRemoveListItemRejectsInvalidTargetsAndStaleSource(t *testing.T) {
	t.Parallel()

	source := []byte("- remove\n\nParagraph.\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	item := publicListItemByContent(t, doc, source, "remove")

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
	if _, err := doc.PrepareRemoveListItem(marksplice.NodeID{}); !errors.Is(err, marksplice.ErrNodeNotFound) {
		t.Fatalf("PrepareRemoveListItem(zero) error = %v, want ErrNodeNotFound", err)
	}
	if _, err := doc.PrepareRemoveListItem(paragraph.ID()); !errors.Is(err, marksplice.ErrInvalidTargetKind) {
		t.Fatalf("PrepareRemoveListItem(paragraph) error = %v, want ErrInvalidTargetKind", err)
	}

	change, err := doc.PrepareRemoveListItem(item.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveListItem() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[0] = '*'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func publicListItemByContent(t *testing.T, doc *marksplice.Document, source []byte, content string) marksplice.ListItem {
	t.Helper()
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindListItem {
			continue
		}
		item, ok := doc.ListItem(node.ID())
		if !ok {
			continue
		}
		if string(source[item.Range().Start:item.Range().End]) == content {
			return item
		}
	}
	t.Fatalf("list item %q not found", content)
	return marksplice.ListItem{}
}
