package goldmark

import (
	"strings"
	"testing"

	markparser "github.com/zoster81/marksplice/internal/parser"
)

func TestValidateConstructionInlineHierarchyAcceptsExactNestedStructure(t *testing.T) {
	t.Parallel()

	source := []byte("*before ``a`b`` after* **_inside_** *~~gone~~*")
	expected := []markparser.ConstructionInlineExpectation{
		constructionInlineExpectation(source, "*before ``a`b`` after*", markparser.KindEmphasis, '*', 1, -1),
		constructionInlineExpectation(source, "``a`b``", markparser.KindCodeSpan, '`', 2, 0),
		constructionInlineExpectation(source, "**_inside_**", markparser.KindStrong, '*', 2, -1),
		constructionInlineExpectation(source, "_inside_", markparser.KindEmphasis, '_', 1, 2),
		constructionInlineExpectation(source, "*~~gone~~*", markparser.KindEmphasis, '*', 1, -1),
		constructionInlineExpectation(source, "~~gone~~", markparser.KindStrikethrough, '~', 2, 4),
	}
	if err := ValidateConstructionInlineHierarchy(source, expected, nil); err != nil {
		t.Fatalf("ValidateConstructionInlineHierarchy() error = %v", err)
	}
}

func TestValidateConstructionInlineHierarchyAcceptsStructuredLinkImageAndReferenceOwners(t *testing.T) {
	t.Parallel()

	inlineSource := []byte("[**docs** `v1`](<target> \"Guide\") ![*logo*](<image.png>)")
	inlineExpected := []markparser.ConstructionInlineExpectation{
		constructionInlineOwnerExpectation(inlineSource, "[**docs** `v1`](<target> \"Guide\")", "**docs** `v1`", markparser.KindInlineLink),
		constructionInlineExpectation(inlineSource, "**docs**", markparser.KindStrong, '*', 2, 0),
		constructionInlineExpectation(inlineSource, "`v1`", markparser.KindCodeSpan, '`', 1, 0),
		constructionInlineOwnerExpectation(inlineSource, "![*logo*](<image.png>)", "*logo*", markparser.KindImage),
		constructionInlineExpectation(inlineSource, "*logo*", markparser.KindEmphasis, '*', 1, 3),
	}
	if err := ValidateConstructionInlineHierarchy(inlineSource, inlineExpected, nil); err != nil {
		t.Fatalf("ValidateConstructionInlineHierarchy(inline owners) error = %v", err)
	}

	referenceSource := []byte("[**docs**][ref]")
	referenceExpected := []markparser.ConstructionInlineExpectation{
		constructionInlineOwnerExpectation(referenceSource, "[**docs**][ref]", "**docs**", markparser.KindInlineLink),
		constructionInlineExpectation(referenceSource, "**docs**", markparser.KindStrong, '*', 2, 0),
	}
	referenceDefinitions := []markparser.ConstructionReferenceInlineExpectation{{Reference: "ref", Destination: "target"}}
	if err := ValidateConstructionInlineHierarchy(referenceSource, referenceExpected, referenceDefinitions); err != nil {
		t.Fatalf("ValidateConstructionInlineHierarchy(reference owner) error = %v", err)
	}
}

func TestValidateConstructionInlineHierarchyRejectsChangedOwnerChildParent(t *testing.T) {
	t.Parallel()

	source := []byte("[**docs**](<target>)")
	expected := []markparser.ConstructionInlineExpectation{
		constructionInlineOwnerExpectation(source, "[**docs**](<target>)", "**docs**", markparser.KindInlineLink),
		constructionInlineExpectation(source, "**docs**", markparser.KindStrong, '*', 2, 0),
	}

	changedParent := append([]markparser.ConstructionInlineExpectation(nil), expected...)
	changedParent[1].Parent = -1
	if err := ValidateConstructionInlineHierarchy(source, changedParent, nil); err == nil {
		t.Fatal("ValidateConstructionInlineHierarchy(changed owner child parent) error = nil, want rejection")
	}
}

func TestValidateConstructionInlineHierarchyRejectsChangedParentDelimiterOrChildSet(t *testing.T) {
	t.Parallel()

	source := []byte("*before ``code`` after*")
	expected := []markparser.ConstructionInlineExpectation{
		constructionInlineExpectation(source, "*before ``code`` after*", markparser.KindEmphasis, '*', 1, -1),
		constructionInlineExpectation(source, "``code``", markparser.KindCodeSpan, '`', 2, 0),
	}

	changedParent := append([]markparser.ConstructionInlineExpectation(nil), expected...)
	changedParent[1].Parent = -1
	if err := ValidateConstructionInlineHierarchy(source, changedParent, nil); err == nil {
		t.Fatal("ValidateConstructionInlineHierarchy(changed parent) error = nil, want rejection")
	}

	changedDelimiter := append([]markparser.ConstructionInlineExpectation(nil), expected...)
	changedDelimiter[0].Marker = '_'
	if err := ValidateConstructionInlineHierarchy(source, changedDelimiter, nil); err == nil {
		t.Fatal("ValidateConstructionInlineHierarchy(changed delimiter) error = nil, want rejection")
	}

	if err := ValidateConstructionInlineHierarchy(source, expected[:1], nil); err == nil {
		t.Fatal("ValidateConstructionInlineHierarchy(missing code child) error = nil, want rejection")
	}
}

func TestValidateConstructionInlineHierarchyIgnoresUnrelatedTopLevelInlineKinds(t *testing.T) {
	t.Parallel()

	source := []byte("*em* [docs](<target>)")
	expected := []markparser.ConstructionInlineExpectation{
		constructionInlineExpectation(source, "*em*", markparser.KindEmphasis, '*', 1, -1),
	}
	if err := ValidateConstructionInlineHierarchy(source, expected, nil); err != nil {
		t.Fatalf("ValidateConstructionInlineHierarchy() error = %v", err)
	}
}

func constructionInlineOwnerExpectation(source []byte, token, label string, kind markparser.Kind) markparser.ConstructionInlineExpectation {
	start := strings.Index(string(source), token)
	prefixLength := 1
	if kind == markparser.KindImage {
		prefixLength = 2
	}
	labelStart := start + prefixLength
	return markparser.ConstructionInlineExpectation{
		Kind:         kind,
		SyntaxRange:  markparser.Range{Start: start, End: start + len(token)},
		ContentRange: markparser.Range{Start: labelStart, End: labelStart + len(label)},
		Parent:       -1,
	}
}

func constructionInlineExpectation(source []byte, token string, kind markparser.Kind, marker byte, delimiterLength, parent int) markparser.ConstructionInlineExpectation {
	start := strings.Index(string(source), token)
	end := start + len(token)
	return markparser.ConstructionInlineExpectation{
		Kind:            kind,
		SyntaxRange:     markparser.Range{Start: start, End: end},
		ContentRange:    markparser.Range{Start: start + delimiterLength, End: end - delimiterLength},
		Marker:          marker,
		DelimiterLength: delimiterLength,
		Parent:          parent,
	}
}
