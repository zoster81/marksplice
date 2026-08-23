package marksplice_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicBlockquotePromotesSimpleExactPhysicalLine(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n  > quoted *text*  \r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var found marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			found = node
			break
		}
	}
	if found.ID() == (marksplice.NodeID{}) {
		t.Fatal("simple blockquote was not promoted")
	}
	detail, ok := doc.Blockquote(found.ID())
	if !ok {
		t.Fatal("Blockquote() ok = false")
	}
	line, ok := doc.SourceRange(detail.Range())
	if !ok || !bytes.Equal(line, []byte("  > quoted *text*  \r\n")) {
		t.Fatalf("blockquote line = %q/%v, want exact physical line", line, ok)
	}
	content, ok := doc.SourceRange(detail.ContentRange())
	if !ok || !bytes.Equal(content, []byte("quoted *text*  ")) {
		t.Fatalf("blockquote content = %q/%v, want exact inner paragraph source", content, ok)
	}
	if detail.ID() != found.ID() {
		t.Fatalf("detail ID = %v, want %v", detail.ID(), found.ID())
	}
}

func TestPublicBlockquoteSupportsMarkerWithoutFollowingSpace(t *testing.T) {
	t.Parallel()

	source := []byte(">quoted\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindBlockquote {
			continue
		}
		detail, ok := doc.Blockquote(node.ID())
		if !ok {
			t.Fatal("Blockquote() ok = false")
		}
		content, ok := doc.SourceRange(detail.ContentRange())
		if !ok || !bytes.Equal(content, []byte("quoted")) {
			t.Fatalf("content = %q/%v, want quoted", content, ok)
		}
		return
	}
	t.Fatal("block quote without following marker space was not promoted")
}

func TestPublicBlockquoteRejectsBroaderContainers(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("> one\n> two\n"),
		[]byte("> - item\n"),
	} {
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", source, err)
		}
		for _, node := range doc.Nodes() {
			if node.Kind() == marksplice.KindBlockquote {
				t.Fatalf("unsupported blockquote %q unexpectedly promoted", source)
			}
		}
	}
}

func TestPrepareRemoveBlockquoteRemovesExactPhysicalLine(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n> quoted\r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			id = node.ID()
			break
		}
	}
	change, err := doc.PrepareRemoveBlockquote(id)
	if err != nil {
		t.Fatalf("PrepareRemoveBlockquote() error = %v", err)
	}
	got, err := change.Apply(source)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	want := []byte("before\r\n\r\n\r\nafter\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("result = %q, want %q", got, want)
	}
}

func TestPrepareRemoveBlockquoteFailsClosedWhenJoinChangesParagraphStructure(t *testing.T) {
	t.Parallel()

	source := []byte("before\n> quoted\n---\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			id = node.ID()
			break
		}
	}
	if _, err := doc.PrepareRemoveBlockquote(id); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareRemoveBlockquote() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareRemoveBlockquoteIsSourceBound(t *testing.T) {
	t.Parallel()

	source := []byte("> quoted\n\nTail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			id = node.ID()
			break
		}
	}
	change, err := doc.PrepareRemoveBlockquote(id)
	if err != nil {
		t.Fatalf("PrepareRemoveBlockquote() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[len(stale)-2] = 'x'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func TestKindBlockquoteAppendsAfterThematicBreak(t *testing.T) {
	t.Parallel()
	if marksplice.KindBlockquote != marksplice.KindThematicBreak+1 {
		t.Fatalf("KindBlockquote = %d, want KindThematicBreak+1 = %d", marksplice.KindBlockquote, marksplice.KindThematicBreak+1)
	}
}
