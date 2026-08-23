package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicDocumentBuilderWritesCanonicalMultilineBlockquoteParagraph(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.SetYAMLFrontMatter(marksplice.FrontMatterFieldInput{Key: "title", Value: "Quoted"}); err != nil {
		t.Fatalf("SetYAMLFrontMatter() error = %v", err)
	}
	if err := builder.AppendHeading(1, "Title"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.AppendBlockquote("first *line*\nsecond [link](https://example.test) with Unicode π"); err != nil {
		t.Fatalf("AppendBlockquote(multiline) error = %v", err)
	}
	if err := builder.AppendParagraph("Tail."); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("---\ntitle: \"Quoted\"\n---\n\n# Title\n\n> first *line*\n> second [link](https://example.test) with Unicode π\n\nTail.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			t.Fatal("multiline constructed blockquote unexpectedly entered the existing-source public blockquote subset")
		}
	}
}

func TestPublicDocumentBuilderRejectsInvalidMultilineBlockquoteParagraphs(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"line one\r\nline two",
		"line one\rline two",
		"first\n\nsecond",
		"# heading\nsecond",
		"first\n- item",
		"first\n> nested",
		"---\nsecond",
		"first\ncontains\x00nul",
		"first\n" + string([]byte{0xff}),
	}
	for _, content := range invalid {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendBlockquote(content); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendBlockquote(%q) error = %v, want ErrInvalidConstruction", content, err)
		}
		if got, err := builder.Markdown(); err != nil || len(got) != 0 {
			t.Fatalf("builder after rejected multiline blockquote Markdown() = %q, %v; want empty, nil", got, err)
		}
	}
}

func TestPublicDocumentBuilderRejectedMultilineBlockquoteDoesNotMutatePriorState(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendHeading(1, "Before"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.AppendBlockquote("first\n- item"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendBlockquote(invalid multiline) error = %v, want ErrInvalidConstruction", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("# Before\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown() after rejected blockquote = %q, want %q", got, want)
	}
}
