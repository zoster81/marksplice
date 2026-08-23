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
	if err := ValidateConstructionInlineHierarchy(source, expected); err != nil {
		t.Fatalf("ValidateConstructionInlineHierarchy() error = %v", err)
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
	if err := ValidateConstructionInlineHierarchy(source, changedParent); err == nil {
		t.Fatal("ValidateConstructionInlineHierarchy(changed parent) error = nil, want rejection")
	}

	changedDelimiter := append([]markparser.ConstructionInlineExpectation(nil), expected...)
	changedDelimiter[0].Marker = '_'
	if err := ValidateConstructionInlineHierarchy(source, changedDelimiter); err == nil {
		t.Fatal("ValidateConstructionInlineHierarchy(changed delimiter) error = nil, want rejection")
	}

	if err := ValidateConstructionInlineHierarchy(source, expected[:1]); err == nil {
		t.Fatal("ValidateConstructionInlineHierarchy(missing code child) error = nil, want rejection")
	}
}

func TestValidateConstructionInlineHierarchyIgnoresUnrelatedTopLevelInlineKinds(t *testing.T) {
	t.Parallel()

	source := []byte("*em* [docs](<target>)")
	expected := []markparser.ConstructionInlineExpectation{
		constructionInlineExpectation(source, "*em*", markparser.KindEmphasis, '*', 1, -1),
	}
	if err := ValidateConstructionInlineHierarchy(source, expected); err != nil {
		t.Fatalf("ValidateConstructionInlineHierarchy() error = %v", err)
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
