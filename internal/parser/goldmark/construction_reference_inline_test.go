package goldmark

import (
	"strings"
	"testing"

	markparser "github.com/zoster81/marksplice/internal/parser"
)

func TestValidateConstructionReferenceInlinesAcceptsExactFullLinkAndImage(t *testing.T) {
	t.Parallel()

	source := []byte("[docs][ref] and ![logo][img]")
	expected := []markparser.ConstructionReferenceInlineExpectation{
		constructionReferenceInlineExpectation(source, "[docs][ref]", "docs", "ref", false, "https://example.test", "Guide", true),
		constructionReferenceInlineExpectation(source, "![logo][img]", "logo", "img", true, "images/logo.png", "", false),
	}
	if err := ValidateConstructionReferenceInlines(source, expected); err != nil {
		t.Fatalf("ValidateConstructionReferenceInlines() error = %v", err)
	}
}

func TestValidateConstructionReferenceInlinesRejectsNonFullOrChangedReferenceSource(t *testing.T) {
	t.Parallel()

	collapsed := []byte("[docs][]")
	collapsedExpectation := markparser.ConstructionReferenceInlineExpectation{
		Kind:           markparser.KindInlineLink,
		SyntaxRange:    markparser.Range{Start: 0, End: len(collapsed)},
		LabelRange:     markparser.Range{Start: 1, End: 5},
		ReferenceRange: markparser.Range{Start: 7, End: 7},
		Reference:      "docs",
		Destination:    "target",
	}
	if err := ValidateConstructionReferenceInlines(collapsed, []markparser.ConstructionReferenceInlineExpectation{collapsedExpectation}); err == nil {
		t.Fatal("ValidateConstructionReferenceInlines(collapsed) error = nil, want rejection")
	}

	source := []byte("[docs][ref]")
	changed := constructionReferenceInlineExpectation(source, "[docs][ref]", "docs", "ref", false, "target", "", false)
	changed.Reference = "REF"
	if err := ValidateConstructionReferenceInlines(source, []markparser.ConstructionReferenceInlineExpectation{changed}); err == nil {
		t.Fatal("ValidateConstructionReferenceInlines(changed reference) error = nil, want rejection")
	}
}

func TestValidateConstructionReferenceInlinesRejectsConflictingDefinitionSemantics(t *testing.T) {
	t.Parallel()

	source := []byte("[a][ref] [b][ref]")
	first := constructionReferenceInlineExpectation(source, "[a][ref]", "a", "ref", false, "first", "", false)
	second := constructionReferenceInlineExpectation(source, "[b][ref]", "b", "ref", false, "second", "", false)
	if err := ValidateConstructionReferenceInlines(source, []markparser.ConstructionReferenceInlineExpectation{first, second}); err == nil {
		t.Fatal("ValidateConstructionReferenceInlines(conflicting definition) error = nil, want rejection")
	}
}

func constructionReferenceInlineExpectation(source []byte, token, label, reference string, image bool, destination, title string, hasTitle bool) markparser.ConstructionReferenceInlineExpectation {
	start := strings.Index(string(source), token)
	prefixLength := 1
	kind := markparser.KindInlineLink
	if image {
		prefixLength = 2
		kind = markparser.KindImage
	}
	labelStart := start + prefixLength
	labelEnd := labelStart + len(label)
	referenceStart := labelEnd + 2
	return markparser.ConstructionReferenceInlineExpectation{
		Kind:           kind,
		SyntaxRange:    markparser.Range{Start: start, End: start + len(token)},
		LabelRange:     markparser.Range{Start: labelStart, End: labelEnd},
		ReferenceRange: markparser.Range{Start: referenceStart, End: referenceStart + len(reference)},
		Reference:      reference,
		Destination:    destination,
		Title:          title,
		HasTitle:       hasTitle,
	}
}
