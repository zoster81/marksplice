package goldmark

import (
	"strings"
	"testing"

	markparser "github.com/zoster81/marksplice/internal/parser"
)

func TestValidateConstructionLinkImagesAcceptsStructuredLinkAndImageSemantics(t *testing.T) {
	t.Parallel()

	source := []byte("[**docs** `v1`](<target> \"Guide\") ![*logo*](<image.png>)")
	expected := []markparser.ConstructionLinkImageExpectation{
		constructionLinkImageExpectation(source, "[**docs** `v1`](<target> \"Guide\")", "**docs** `v1`", markparser.KindInlineLink, "target", "Guide", true),
		constructionLinkImageExpectation(source, "![*logo*](<image.png>)", "*logo*", markparser.KindImage, "image.png", "", false),
	}
	if err := ValidateConstructionLinkImages(source, expected); err != nil {
		t.Fatalf("ValidateConstructionLinkImages() error = %v", err)
	}
}

func TestValidateConstructionLinkImagesRejectsChangedDestinationOrTitle(t *testing.T) {
	t.Parallel()

	source := []byte("[**docs**](<target> \"Guide\")")
	expected := constructionLinkImageExpectation(source, string(source), "**docs**", markparser.KindInlineLink, "target", "Guide", true)

	changedDestination := expected
	changedDestination.Destination = "other"
	if err := ValidateConstructionLinkImages(source, []markparser.ConstructionLinkImageExpectation{changedDestination}); err == nil {
		t.Fatal("ValidateConstructionLinkImages(changed destination) error = nil, want rejection")
	}

	changedTitle := expected
	changedTitle.Title = "Other"
	if err := ValidateConstructionLinkImages(source, []markparser.ConstructionLinkImageExpectation{changedTitle}); err == nil {
		t.Fatal("ValidateConstructionLinkImages(changed title) error = nil, want rejection")
	}
}

func constructionLinkImageExpectation(source []byte, token, label string, kind markparser.Kind, destination, title string, hasTitle bool) markparser.ConstructionLinkImageExpectation {
	start := strings.Index(string(source), token)
	prefixLength := 1
	if kind == markparser.KindImage {
		prefixLength = 2
	}
	labelStart := start + prefixLength
	return markparser.ConstructionLinkImageExpectation{
		Kind:        kind,
		SyntaxRange: markparser.Range{Start: start, End: start + len(token)},
		LabelRange:  markparser.Range{Start: labelStart, End: labelStart + len(label)},
		Destination: destination,
		Title:       title,
		HasTitle:    hasTitle,
	}
}
