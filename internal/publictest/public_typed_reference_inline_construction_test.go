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

func TestPublicTypedReferenceInlineStructuredLabelAndAltConstruction(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendReferenceDefinitionWithTitle("doc", "target", "Guide"); err != nil {
		t.Fatalf("AppendReferenceDefinitionWithTitle() error = %v", err)
	}
	if err := builder.AppendReferenceDefinition("logo", "image.png"); err != nil {
		t.Fatalf("AppendReferenceDefinition() error = %v", err)
	}
	if err := builder.AppendParagraphContent(
		marksplice.ReferenceLinkInline(
			"doc",
			marksplice.StrongInline(marksplice.TextInline("docs")),
			marksplice.TextInline(" "),
			marksplice.CodeInline("v1"),
		),
		marksplice.TextInline(" and "),
		marksplice.ReferenceImageInline(
			"logo",
			marksplice.EmphasisInline(marksplice.TextInline("logo")),
			marksplice.TextInline(" "),
			marksplice.StrikethroughInline(marksplice.TextInline("old")),
		),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("[doc]: <target> \"Guide\"\n\n[logo]: <image.png>\n\n[**docs** `v1`][doc] and ![*logo* ~~old~~][logo]\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicTypedCollapsedAndShortcutReferenceConstruction(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendReferenceDefinition("Doc  Ref", "target"); err != nil {
		t.Fatalf("AppendReferenceDefinition() error = %v", err)
	}
	if err := builder.AppendReferenceDefinition("**strong**", "strong-target"); err != nil {
		t.Fatalf("AppendReferenceDefinition(strong) error = %v", err)
	}
	if err := builder.AppendParagraphContent(
		marksplice.CollapsedReferenceLinkInline(marksplice.TextInline(" doc ref ")),
		marksplice.TextInline(" "),
		marksplice.ShortcutReferenceImageInline(marksplice.StrongInline(marksplice.TextInline("strong"))),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("[Doc  Ref]: <target>\n\n[**strong**]: <strong-target>\n\n[ doc ref ][] ![**strong**]\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicTypedCollapsedReferenceRejectsNormalizedAmbiguity(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.AppendReferenceDefinition("doc", "first"); err != nil {
		t.Fatalf("AppendReferenceDefinition(first) error = %v", err)
	}
	if err := builder.AppendReferenceDefinition("DOC", "second"); err != nil {
		t.Fatalf("AppendReferenceDefinition(second) error = %v", err)
	}
	if err := builder.AppendParagraphContent(marksplice.CollapsedReferenceLinkInline(marksplice.TextInline("doc"))); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendParagraphContent(ambiguous collapsed reference) error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.AppendParagraphContent(marksplice.ReferenceLinkInline("doc", marksplice.TextInline("full"))); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendParagraphContent(normalized-ambiguous full reference) error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicTypedForwardReferenceConstructionUsesDeferredDefinitions(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.DeferReferenceDefinitionWithTitle("doc", "target", "Guide"); err != nil {
		t.Fatalf("DeferReferenceDefinitionWithTitle() error = %v", err)
	}
	if err := builder.DeferReferenceDefinition("logo", "image.png"); err != nil {
		t.Fatalf("DeferReferenceDefinition() error = %v", err)
	}
	if err := builder.AppendParagraphContent(
		marksplice.ForwardReferenceLinkInline("doc", marksplice.TextInline("docs")),
		marksplice.TextInline(" and "),
		marksplice.CollapsedReferenceImageInline(marksplice.TextInline("logo")),
	); err != nil {
		t.Fatalf("AppendParagraphContent() error = %v", err)
	}
	got, err := builder.Markdown()
	if err != nil {
		t.Fatalf("Markdown() error = %v", err)
	}
	want := []byte("[docs][doc] and ![logo][]\n\n[doc]: <target> \"Guide\"\n\n[logo]: <image.png>\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("Markdown() = %q, want %q", got, want)
	}
}

func TestPublicTypedForwardReferencePreservesM89AndRejectsDefinitionCollisions(t *testing.T) {
	t.Parallel()

	var builder marksplice.DocumentBuilder
	if err := builder.DeferReferenceDefinition("doc", "future"); err != nil {
		t.Fatalf("DeferReferenceDefinition() error = %v", err)
	}
	if err := builder.AppendParagraphContent(marksplice.ReferenceLinkInline("doc", marksplice.TextInline("old contract"))); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("ReferenceLinkInline(deferred only) error = %v, want ErrInvalidConstruction", err)
	}
	if err := builder.AppendReferenceDefinition("DOC", "shadow"); !errors.Is(err, marksplice.ErrInvalidConstruction) {
		t.Fatalf("AppendReferenceDefinition(normalized deferred collision) error = %v, want ErrInvalidConstruction", err)
	}
}

func TestPublicTypedReferenceInlineRejectsUnsafeReference(t *testing.T) {
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
}
