package publictest

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicDocumentBuilderWritesCanonicalNestedBlockquoteParagraph(t *testing.T) {
	t.Parallel()

	builder := marksplice.NewDocumentBuilder()
	if err := builder.SetTOMLFrontMatter(marksplice.FrontMatterFieldInput{Key: "title", Value: "Nested"}); err != nil {
		t.Fatalf("SetTOMLFrontMatter() error = %v", err)
	}
	if err := builder.AppendHeading(1, "Before"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.AppendNestedBlockquote(2, "first *line*\nsecond [link](https://example.test) with Unicode π"); err != nil {
		t.Fatalf("AppendNestedBlockquote() error = %v", err)
	}
	if err := builder.AppendParagraph("After."); err != nil {
		t.Fatalf("AppendParagraph() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("+++\ntitle = \"Nested\"\n+++\n\n# Before\n\n> > first *line*\n> > second [link](https://example.test) with Unicode π\n\nAfter.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}

	doc, err := marksplice.Parse(got)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v", err)
	}
	for _, node := range doc.Nodes() {
		if node.Kind() == marksplice.KindBlockquote {
			t.Fatal("nested constructed blockquote unexpectedly entered the existing-source public blockquote subset")
		}
	}
}

func TestPublicDocumentBuilderNestedBlockquoteDepthBoundaries(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendNestedBlockquote(64, "deep"); err != nil {
		t.Fatalf("AppendNestedBlockquote(depth=64) error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte(strings.Repeat("> ", 64) + "deep\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want depth-64 canonical prefix", got)
	}

	for _, depth := range []int{-1, 0, 1, 65} {
		var invalid marksplice.DocumentBuilder
		if err := invalid.AppendNestedBlockquote(depth, "text"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendNestedBlockquote(depth=%d) error = %v, want ErrInvalidConstruction", depth, err)
		}
		if output, err := invalid.Markdown(); err != nil || len(output) != 0 {
			t.Fatalf("builder after rejected depth %d Markdown() = %q, %v; want empty, nil", depth, output, err)
		}
	}
}

func TestPublicDocumentBuilderRejectsNestedBlockquoteShapeEscapes(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",
		"line one\r\nline two",
		"first\n\nsecond",
		"# heading",
		"- item",
		"> extra depth",
		"first\n> extra depth",
		"contains\x00nul",
		string([]byte{0xff}),
	}
	for _, content := range invalid {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendNestedBlockquote(2, content); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendNestedBlockquote(%q) error = %v, want ErrInvalidConstruction", content, err)
		}
		if got, err := builder.Markdown(); err != nil || len(got) != 0 {
			t.Fatalf("builder after rejected nested blockquote Markdown() = %q, %v; want empty, nil", got, err)
		}
	}
}

func TestPublicDocumentBuilderRejectedNestedBlockquoteDoesNotMutatePriorState(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendHeading(1, "Before"); err != nil {
		t.Fatalf("AppendHeading() error = %v", err)
	}
	if err := builder.AppendNestedBlockquote(2, "> extra depth"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendNestedBlockquote(invalid) error = %v, want ErrInvalidConstruction", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("# Before\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown() after rejected nested blockquote = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderWritesTypedNestedBlockquote(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendNestedBlockquoteContent(
		3,
		marksplice.TextInline("> literal "),
		marksplice.EmphasisInline(marksplice.TextInline("text")),
	); err != nil {
		t.Fatalf("AppendNestedBlockquoteContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	if want := []byte("> > > \\> literal *text*\n"); !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicDocumentBuilderNestedBlockquoteNilAndTypedErrors(t *testing.T) {
	t.Parallel()

	var nilBuilder *marksplice.DocumentBuilder
	if err := nilBuilder.AppendNestedBlockquote(2, "text"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendNestedBlockquote() error = %v, want ErrInvalidConstruction", err)
	}
	if err := nilBuilder.AppendNestedBlockquoteContent(2, marksplice.TextInline("text")); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("nil AppendNestedBlockquoteContent() error = %v, want ErrInvalidConstruction", err)
	}

	var builder marksplice.DocumentBuilder
	if err := builder.AppendNestedBlockquoteContent(1, marksplice.TextInline("text")); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendNestedBlockquoteContent(depth=1) error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.AppendNestedBlockquoteContent(2, marksplice.TextInline("line\nbreak")); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendNestedBlockquoteContent(multiline text) error = %v, want ErrInvalidConstruction", err)
	}
	if got, err := builder.Markdown(); err != nil || len(got) != 0 {
		t.Fatalf("builder after rejected typed nested blockquotes Markdown() = %q, %v; want empty, nil", got, err)
	}
}
