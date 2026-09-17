package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestParagraphStructuralMutationsOwnSeparatorsAndPreserveNeighbors(t *testing.T) {
	t.Parallel()

	t.Run("remove indented LF paragraph", func(t *testing.T) {
		source := []byte("before\n\n  remove me\n\nafter\n")
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		paragraph := publicParagraphContaining(t, doc, "remove me")
		change, err := doc.PrepareRemoveParagraph(paragraph.ID())
		if err != nil {
			t.Fatalf("PrepareRemoveParagraph() error = %v", err)
		}
		got, err := change.Apply(source)
		if err != nil {
			t.Fatal(err)
		}
		want := []byte("before\n\nafter\n")
		if !bytes.Equal(got, want) {
			t.Fatalf("Apply() = %q, want %q", got, want)
		}
	})

	t.Run("insert before preserves CRLF and inline semantics", func(t *testing.T) {
		source := []byte("first\r\n\r\ntarget [link](dest)\r\n")
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		target := publicParagraphContaining(t, doc, "target")
		change, err := doc.PrepareInsertParagraphBefore(target.ID(), []byte("new *paragraph*"))
		if err != nil {
			t.Fatalf("PrepareInsertParagraphBefore() error = %v", err)
		}
		got, err := change.Apply(source)
		if err != nil {
			t.Fatal(err)
		}
		want := []byte("first\r\n\r\nnew *paragraph*\r\n\r\ntarget [link](dest)\r\n")
		if !bytes.Equal(got, want) {
			t.Fatalf("Apply() = %q, want %q", got, want)
		}
	})

	t.Run("insert after preserves existing following separator", func(t *testing.T) {
		source := []byte("target\n\nafter\n")
		doc, err := marksplice.Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		target := publicParagraphContaining(t, doc, "target")
		change, err := doc.PrepareInsertParagraphAfter(target.ID(), []byte("new **paragraph**"))
		if err != nil {
			t.Fatalf("PrepareInsertParagraphAfter() error = %v", err)
		}
		got, err := change.Apply(source)
		if err != nil {
			t.Fatal(err)
		}
		want := []byte("target\n\nnew **paragraph**\n\nafter\n")
		if !bytes.Equal(got, want) {
			t.Fatalf("Apply() = %q, want %q", got, want)
		}
	})
}

func TestParagraphStructuralMutationsFailClosedAndRemainSnapshotBound(t *testing.T) {
	t.Parallel()

	source := []byte("target\n")
	doc, err := marksplice.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	target := publicParagraphContaining(t, doc, "target")
	for _, replacement := range [][]byte{nil, []byte("# heading"), []byte("one\n\ntwo")} {
		if _, err := doc.PrepareInsertParagraphBefore(target.ID(), replacement); !errors.Is(err, marksplice.ErrInvalidReplacement) {
			t.Fatalf("before content %q error = %v, want ErrInvalidReplacement", replacement, err)
		}
	}

	noEOL := []byte("target")
	noEOLDoc, err := marksplice.Parse(noEOL)
	if err != nil {
		t.Fatal(err)
	}
	noEOLTarget := publicParagraphContaining(t, noEOLDoc, "target")
	if _, err := noEOLDoc.PrepareInsertParagraphAfter(noEOLTarget.ID(), []byte("new")); !errors.Is(err, marksplice.ErrInvalidReplacement) {
		t.Fatalf("after EOF error = %v, want ErrInvalidReplacement", err)
	}

	change, err := doc.PrepareRemoveParagraph(target.ID())
	if err != nil {
		t.Fatalf("PrepareRemoveParagraph() error = %v", err)
	}
	stale := []byte("targeX\n")
	if _, err := change.Apply(stale); !errors.Is(err, marksplice.ErrSourceConflict) {
		t.Fatalf("Apply(stale) error = %v, want ErrSourceConflict", err)
	}
}

func publicParagraphContaining(t *testing.T, doc *marksplice.Document, needle string) marksplice.Paragraph {
	t.Helper()
	for _, node := range doc.Nodes() {
		if node.Kind() != marksplice.KindParagraph {
			continue
		}
		paragraph, ok := doc.Paragraph(node.ID())
		if !ok {
			continue
		}
		value, ok := doc.SourceRange(paragraph.Range())
		if ok && bytes.Contains(value, []byte(needle)) {
			return paragraph
		}
	}
	t.Fatalf("paragraph containing %q not found", needle)
	return marksplice.Paragraph{}
}
