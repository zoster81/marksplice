package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicThematicBreakPromotesExactPhysicalLine(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n  * * *  \r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var found marksplice.Node
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindThematicBreak {
			found = node
			break
		}
	}
	if found.ID() == (marksplice.NodeID{}) {
		t.Fatal("thematic break was not promoted")
	}
	detail, ok := doc.ThematicBreak(found.ID())
	if !ok {
		t.Fatal("ThematicBreak() ok = false")
	}
	got, ok := doc.SourceRange(detail.Range())
	if !ok || !bytes.Equal(got, []byte("  * * *  \r\n")) {
		t.Fatalf("thematic break source = %q/%v, want exact physical line", got, ok)
	}
	if detail.ID() != found.ID() {
		t.Fatalf("detail ID = %v, want %v", detail.ID(), found.ID())
	}
}

func TestPublicThematicBreakSupportsMarkerFamiliesAndEOF(t *testing.T) {
	t.Parallel()

	for _, source := range [][]byte{
		[]byte("***\n"),
		[]byte("- - -\n"),
		[]byte("___"),
	} {
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", source, err)
		}
		var count int
		for _, node := range doc.Nodes() {
			if node.Kind() != marksplice.KindThematicBreak {
				continue
			}
			count++
			detail, ok := doc.ThematicBreak(node.ID())
			if !ok {
				t.Fatalf("ThematicBreak(%q) ok = false", source)
			}
			got, ok := doc.SourceRange(detail.Range())
			if !ok || !bytes.Equal(got, source) {
				t.Fatalf("SourceRange(%q) = %q/%v", source, got, ok)
			}
		}
		if count != 1 {
			t.Fatalf("thematic break count for %q = %d, want 1", source, count)
		}
	}
}

func TestPrepareRemoveThematicBreakRemovesExactPhysicalLine(t *testing.T) {
	t.Parallel()

	source := []byte("before\r\n\r\n  * * *  \r\n\r\nafter\r\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindThematicBreak {
			id = node.ID()
			break
		}
	}
	change, err := doc.PrepareRemoveThematicBreak(id)
	if err != nil {
		t.Fatalf("PrepareRemoveThematicBreak() error = %v", err)
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

func TestPrepareRemoveThematicBreakFailsClosedWhenJoinChangesParagraphStructure(t *testing.T) {
	t.Parallel()

	source := []byte("before\n***\nafter\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindThematicBreak {
			id = node.ID()
			break
		}
	}
	if _, err := doc.PrepareRemoveThematicBreak(id); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("PrepareRemoveThematicBreak() error = %v, want ErrInvalidReplacement", err)
	}
}

func TestPrepareRemoveThematicBreakIsSourceBound(t *testing.T) {
	t.Parallel()

	source := []byte("---\n\nTail\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var id marksplice.NodeID
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindThematicBreak {
			id = node.ID()
			break
		}
	}
	change, err := doc.PrepareRemoveThematicBreak(id)
	if err != nil {
		t.Fatalf("PrepareRemoveThematicBreak() error = %v", err)
	}
	stale := append([]byte(nil), source...)
	stale[len(stale)-2] = 'x'
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func TestPublicThematicBreakDoesNotPromoteNestedBreak(t *testing.T) {
	t.Parallel()

	doc, err := marksplice.Parse([]byte("> ---\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindThematicBreak {
			t.Fatalf("nested thematic break unexpectedly promoted: %v", node.ID())
		}
	}
}
