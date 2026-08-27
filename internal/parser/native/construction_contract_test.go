package native_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
)

func TestM115NativeConstructionProofAcceptsFrozenContract(t *testing.T) {
	t.Parallel()
	backend := native.New()

	t.Run("blockquote paragraph", func(t *testing.T) {
		source := []byte("# Title\n\n> > first *line*\n> > second π\n\nTail.\n")
		start := bytes.Index(source, []byte("> > first *line*"))
		firstStart := start + 4
		firstEnd := firstStart + len("first *line*")
		secondStart := firstEnd + 1 + 4
		secondEnd := secondStart + len("second π")
		outer := parser.Range{Start: start, End: secondEnd}
		lines := []parser.Range{{Start: firstStart, End: firstEnd}, {Start: secondStart, End: secondEnd}}
		if err := backend.ValidateNestedBlockquoteParagraph(source, outer, lines, 2); err != nil {
			t.Fatalf("ValidateNestedBlockquoteParagraph() error = %v", err)
		}
	})

	t.Run("blockquote blocks", func(t *testing.T) {
		inner := []byte("## Head\n\n- parent\n  - child\n\n```go\nx\n```\n")
		source := []byte("> > ## Head\n> > \n> > - parent\n> >   - child\n> > \n> > ```go\n> > x\n> > ```\n")
		outer := parser.Range{Start: 0, End: len(source) - 1}
		if err := backend.ValidateNestedBlockquoteBlocks(source, outer, inner, 2); err != nil {
			t.Fatalf("ValidateNestedBlockquoteBlocks() error = %v", err)
		}
	})

	t.Run("inline hierarchy", func(t *testing.T) {
		source := []byte("*before ``a`b`` after* **_inside_** *~~gone~~*")
		expected := []parser.ConstructionInlineExpectation{
			m115InlineExpectation(source, "*before ``a`b`` after*", parser.KindEmphasis, '*', 1, -1),
			m115InlineExpectation(source, "``a`b``", parser.KindCodeSpan, '`', 2, 0),
			m115InlineExpectation(source, "**_inside_**", parser.KindStrong, '*', 2, -1),
			m115InlineExpectation(source, "_inside_", parser.KindEmphasis, '_', 1, 2),
			m115InlineExpectation(source, "*~~gone~~*", parser.KindEmphasis, '*', 1, -1),
			m115InlineExpectation(source, "~~gone~~", parser.KindStrikethrough, '~', 2, 4),
		}
		if err := backend.ValidateConstructionInlineHierarchy(source, expected, nil); err != nil {
			t.Fatalf("ValidateConstructionInlineHierarchy() error = %v", err)
		}
	})

	t.Run("unmatched backtick emphasis", func(t *testing.T) {
		source := []byte("*a`b*")
		expected := []parser.ConstructionInlineExpectation{
			m115InlineExpectation(source, string(source), parser.KindEmphasis, '*', 1, -1),
		}
		if err := backend.ValidateConstructionInlineHierarchy(source, expected, nil); err != nil {
			t.Fatalf("ValidateConstructionInlineHierarchy() error = %v", err)
		}
	})

	t.Run("direct link image", func(t *testing.T) {
		source := []byte("[**docs**](<target> \"Guide\") ![*logo*](<image.png>)")
		expected := []parser.ConstructionLinkImageExpectation{
			m115LinkImageExpectation(source, "[**docs**](<target> \"Guide\")", "**docs**", parser.KindInlineLink, "target", "Guide", true),
			m115LinkImageExpectation(source, "![*logo*](<image.png>)", "*logo*", parser.KindImage, "image.png", "", false),
		}
		if err := backend.ValidateConstructionLinkImages(source, expected); err != nil {
			t.Fatalf("ValidateConstructionLinkImages() error = %v", err)
		}
	})

	t.Run("reference inline", func(t *testing.T) {
		source := []byte("[docs][ref] and ![logo][img]")
		expected := []parser.ConstructionReferenceInlineExpectation{
			m115ReferenceExpectation(source, "[docs][ref]", "docs", "ref", parser.KindInlineLink, "https://example.test", "Guide", true),
			m115ReferenceExpectation(source, "![logo][img]", "logo", "img", parser.KindImage, "images/logo.png", "", false),
		}
		if err := backend.ValidateConstructionReferenceInlines(source, expected); err != nil {
			t.Fatalf("ValidateConstructionReferenceInlines() error = %v", err)
		}
	})

	for _, source := range [][]byte{
		[]byte("[**docs** `v1`][ref]"),
		[]byte("[docs][ref]"),
	} {
		label := string(source[1:bytes.IndexByte(source, ']')])
		expected := m115ReferenceExpectation(source, string(source), label, "ref", parser.KindInlineLink, "target", "Guide", true)
		expected.StructuredLabel = true
		if err := backend.ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{expected}); err != nil {
			t.Fatalf("structured reference %q error = %v", source, err)
		}
	}
}

func TestM115NativeConstructionProofRejectsFrozenContractChanges(t *testing.T) {
	t.Parallel()
	backend := native.New()

	t.Run("blockquote depth mismatch", func(t *testing.T) {
		source := []byte("> > paragraph\n")
		outer := parser.Range{Start: 0, End: len(source) - 1}
		contentStart := len("> > ")
		content := []parser.Range{{Start: contentStart, End: len(source) - 1}}
		if err := backend.ValidateNestedBlockquoteParagraph(source, outer, content, 1); err == nil {
			t.Fatal("ValidateNestedBlockquoteParagraph() error = nil, want rejection")
		}
	})

	t.Run("inline parent mismatch", func(t *testing.T) {
		source := []byte("*outer `code`*")
		expected := []parser.ConstructionInlineExpectation{
			m115InlineExpectation(source, "*outer `code`*", parser.KindEmphasis, '*', 1, -1),
			m115InlineExpectation(source, "`code`", parser.KindCodeSpan, '`', 1, -1),
		}
		if err := backend.ValidateConstructionInlineHierarchy(source, expected, nil); err == nil {
			t.Fatal("ValidateConstructionInlineHierarchy() error = nil, want rejection")
		}
	})

	t.Run("direct destination mismatch", func(t *testing.T) {
		source := []byte("[docs](<target>)")
		expected := []parser.ConstructionLinkImageExpectation{
			m115LinkImageExpectation(source, string(source), "docs", parser.KindInlineLink, "other", "", false),
		}
		if err := backend.ValidateConstructionLinkImages(source, expected); err == nil {
			t.Fatal("ValidateConstructionLinkImages() error = nil, want rejection")
		}
	})

	t.Run("structured reference declared plain", func(t *testing.T) {
		source := []byte("[**docs** `v1`][ref]")
		expected := m115ReferenceExpectation(source, string(source), "**docs** `v1`", "ref", parser.KindInlineLink, "target", "Guide", true)
		if err := backend.ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{expected}); err == nil {
			t.Fatal("ValidateConstructionReferenceInlines() error = nil, want rejection")
		}
	})

	t.Run("conflicting reference semantics", func(t *testing.T) {
		source := []byte("[a][ref] [b][ref]")
		first := m115ReferenceExpectation(source, "[a][ref]", "a", "ref", parser.KindInlineLink, "first", "", false)
		second := m115ReferenceExpectation(source, "[b][ref]", "b", "ref", parser.KindInlineLink, "second", "", false)
		if err := backend.ValidateConstructionReferenceInlines(source, []parser.ConstructionReferenceInlineExpectation{first, second}); err == nil {
			t.Fatal("ValidateConstructionReferenceInlines() error = nil, want rejection")
		}
	})
}

func TestM115NativeReferenceOperationsFrozenContract(t *testing.T) {
	t.Parallel()
	backend := native.New()

	labels := map[string]string{
		"  A  B\tC ": "a b c",
		"Straße":     "strasse",
		"STRASSE":    "strasse",
		"İ":          "i̇",
		"\u212a":     "k",
		"a\nb":       "a b",
	}
	for label, want := range labels {
		if got := backend.ReferenceLabelKey(label); got != want {
			t.Fatalf("ReferenceLabelKey(%q) = %q, want %q", label, got, want)
		}
	}

	definitions := []parser.ConstructionReferenceDefinition{
		{Label: "Straße", Destination: "one"},
		{Label: "other", Destination: "two"},
	}
	got, err := backend.ResolveConstructionReference("STRASSE", definitions)
	if err != nil {
		t.Fatalf("ResolveConstructionReference() error = %v", err)
	}
	if want := definitions[0]; !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveConstructionReference() = %#v, want %#v", got, want)
	}

	definitions = append(definitions, parser.ConstructionReferenceDefinition{Label: "STRASSE", Destination: "duplicate"})
	if _, err := backend.ResolveConstructionReference("straße", definitions); err == nil {
		t.Fatal("ResolveConstructionReference() duplicate normalized definition error = nil")
	}
}

func m115InlineExpectation(source []byte, token string, kind parser.Kind, marker byte, delimiterLength, parent int) parser.ConstructionInlineExpectation {
	start := strings.Index(string(source), token)
	end := start + len(token)
	return parser.ConstructionInlineExpectation{
		Kind: kind, SyntaxRange: parser.Range{Start: start, End: end},
		ContentRange: parser.Range{Start: start + delimiterLength, End: end - delimiterLength},
		Marker:       marker, DelimiterLength: delimiterLength, Parent: parent,
	}
}

func m115LinkImageExpectation(source []byte, token, label string, kind parser.Kind, destination, title string, hasTitle bool) parser.ConstructionLinkImageExpectation {
	start := strings.Index(string(source), token)
	prefix := 1
	if kind == parser.KindImage {
		prefix = 2
	}
	labelStart := start + prefix
	return parser.ConstructionLinkImageExpectation{
		Kind: kind, SyntaxRange: parser.Range{Start: start, End: start + len(token)},
		LabelRange:  parser.Range{Start: labelStart, End: labelStart + len(label)},
		Destination: destination, Title: title, HasTitle: hasTitle,
	}
}

func m115ReferenceExpectation(source []byte, token, label, reference string, kind parser.Kind, destination, title string, hasTitle bool) parser.ConstructionReferenceInlineExpectation {
	start := strings.Index(string(source), token)
	prefix := 1
	if kind == parser.KindImage {
		prefix = 2
	}
	labelStart := start + prefix
	labelEnd := labelStart + len(label)
	referenceStart := labelEnd + 2
	return parser.ConstructionReferenceInlineExpectation{
		Kind: kind, Form: parser.ConstructionReferenceInlineFull,
		SyntaxRange:    parser.Range{Start: start, End: start + len(token)},
		LabelRange:     parser.Range{Start: labelStart, End: labelEnd},
		ReferenceRange: parser.Range{Start: referenceStart, End: referenceStart + len(reference)},
		Reference:      reference, Destination: destination, Title: title, HasTitle: hasTitle,
	}
}
