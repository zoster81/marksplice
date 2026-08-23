package publictest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/zoster81/marksplice"
)

func TestPublicDocumentBuilderTypedReferenceLinkAndImageConstruction(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendReferenceDefinitionWithTitle("doc", "https://example.test/docs", "Guide v1"); err != nil {
		t.Fatalf("AppendReferenceDefinitionWithTitle() error = %v", err)
	}
	if err := builder.AppendReferenceDefinition("logo", "images/logo.png"); err != nil {
		t.Fatalf("AppendReferenceDefinition() error = %v", err)
	}
	if err := builder.AppendParagraphContent(
		marksplice.TextInline("See "),
		marksplice.ReferenceLinkInline("doc", marksplice.TextInline("docs [v1]")),
		marksplice.TextInline(" and "),
		marksplice.ReferenceImageInline("logo", marksplice.TextInline("logo *literal*")),
		marksplice.TextInline("."),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("[doc]: <https://example.test/docs> \"Guide v1\"\n\n[logo]: <images/logo.png>\n\nSee [docs \\[v1\\]][doc] and ![logo \\*literal\\*][logo]\\.\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicTypedReferenceInlineRequiresExistingExactDefinition(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendParagraphContent(
		marksplice.ReferenceLinkInline("doc", marksplice.TextInline("docs")),
	); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendParagraphContent(missing definition) error = %v, want ErrInvalidConstruction", err)
	}
	if got, err := builder.Markdown(); err != nil || len(got) != 0 {
		t.Fatalf("Markdown() after missing definition = %q, %v; want empty, nil", got, err)
	}

	if err := builder.AppendReferenceDefinition("doc", "target"); err != nil {
		t.Fatalf("AppendReferenceDefinition() error = %v", err)
	}
	if err := builder.AppendParagraphContent(
		marksplice.ReferenceLinkInline("DOC", marksplice.TextInline("case mismatch")),
	); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendParagraphContent(case-mismatched reference) error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.AppendParagraphContent(
		marksplice.ReferenceLinkInline("doc", marksplice.TextInline("docs")),
	); err != nil {
		t.Fatalf("AppendParagraphContent(exact reference) error = %v", err)
	}
}

func TestPublicTypedReferenceInlineRejectsDuplicateExactDefinition(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendReferenceDefinition("doc", "first"); err != nil {
		t.Fatalf("AppendReferenceDefinition(first) error = %v", err)
	}
	if err := builder.AppendReferenceDefinition("doc", "second"); err != nil {
		t.Fatalf("AppendReferenceDefinition(second) error = %v", err)
	}
	if err := builder.AppendParagraphContent(
		marksplice.ReferenceLinkInline("doc", marksplice.TextInline("docs")),
	); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendParagraphContent(duplicate definition) error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicTypedReferenceInlineWorksAcrossTypedBlockEntrypoints(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendReferenceDefinition("doc", "target"); err != nil {
		t.Fatalf("AppendReferenceDefinition() error = %v", err)
	}
	if err := builder.AppendHeadingContent(2, marksplice.ReferenceLinkInline("doc", marksplice.TextInline("heading"))); err != nil {
		t.Fatalf("AppendHeadingContent() error = %v", err)
	}
	if err := builder.AppendBlockquoteContent(marksplice.ReferenceLinkInline("doc", marksplice.TextInline("quote"))); err != nil {
		t.Fatalf("AppendBlockquoteContent() error = %v", err)
	}
	if err := builder.AppendNestedBlockquoteContent(2, marksplice.ReferenceImageInline("doc", marksplice.TextInline("image"))); err != nil {
		t.Fatalf("AppendNestedBlockquoteContent() error = %v", err)
	}

	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("[doc]: <target>\n\n## [heading][doc]\n\n> [quote][doc]\n\n> > ![image][doc]\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicTypedReferenceInlineRejectsUnsafeReferenceAndStructuredLabel(t *testing.T) {
	t.Parallel()

	invalidReferences := []string{
		"",
		"line\nbreak",
		"contains\x00nul",
		string([]byte{0xff}),
		"left[bracket",
		"right]bracket",
		`escape\\label`,
		"entity&amp;label",
	}
	for _, reference := range invalidReferences {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendReferenceDefinition("doc", "target"); err != nil {
			t.Fatalf("AppendReferenceDefinition() error = %v", err)
		}
		for _, inline := range []marksplice.Inline{
			marksplice.ReferenceLinkInline(reference, marksplice.TextInline("label")),
			marksplice.ReferenceImageInline(reference, marksplice.TextInline("alt")),
		} {
			if err := builder.AppendParagraphContent(inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
				t.Fatalf("AppendParagraphContent(unsafe reference %q) error = %v, want ErrInvalidConstruction", reference, err)
			}
		}
	}

	for _, inline := range []marksplice.Inline{
		marksplice.ReferenceLinkInline("doc", marksplice.CodeInline("label")),
		marksplice.ReferenceImageInline("doc", marksplice.EmphasisInline(marksplice.TextInline("alt"))),
	} {
		var builder marksplice.DocumentBuilder
		if err := builder.AppendReferenceDefinition("doc", "target"); err != nil {
			t.Fatalf("AppendReferenceDefinition() error = %v", err)
		}
		if err := builder.AppendParagraphContent(inline); !errors.Is(err, marksplice.ErrInvalidConstruction) {
			t.Fatalf("AppendParagraphContent(structured reference label) error = %v, want ErrInvalidConstruction", err)
		}
	}
}
