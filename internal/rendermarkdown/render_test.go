package rendermarkdown

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
	"github.com/zoster81/marksplice/internal/testutil/commonmarkspec"
	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

type semanticFact struct {
	phase       parser.SemanticPhase
	kind        parser.SemanticKind
	value       string
	level       int
	destination string
	title       string
	hasTitle    bool
	label       string
	email       bool
	ordered     bool
	start       int
	tight       bool
	checked     bool
	header      bool
	column      int
	columns     int
	alignment   parser.TableAlignment
	language    string
	info        string
	alert       parser.SemanticAlertKind
	frontMatter parser.SemanticFrontMatterFormat
	math        parser.MathExpressionStyle
}

func TestCanonicalMarkdownSemanticRoundTripComplexFamilies(t *testing.T) {
	t.Parallel()

	source := []byte("---\r\ntitle: demo\r\n---\r\n\r\nHeading\r\n=======\r\n\r\nParagraph with *em **strong** ~~strike~~*, [direct](target.md 'a \\\"title\\\"'), [reference][ref], ![image *alt*](image.png), <https://example.test/a>, www.example.test, `code`, and $x$ plus $`y`$.  \r\nnext line\r\n\r\n[ref]: /reference 'reference title'\r\n\r\n> quote\r\n>\r\n> 3. outer\r\n>    - inner\r\n\r\n> [!NOTE]\r\n> alert **body**\r\n>\r\n> - child\r\n\r\n- [x] task\r\n- plain\r\n\r\n3. loose one\r\n\r\n4. loose two\r\n\r\n| Left | Right |\r\n| :--- | ---: |\r\n| `a\\|b` | ~~cell~~ |\r\n\r\n~~~go`x\r\n```\r\nbody\r\n~~~\r\n\r\n    indented\r\n    code\r\n\r\n<div data-x=\"a|b\">\r\nraw\r\n</div>\r\n\r\nuse[^note]\r\n\r\n[^note]:\r\n    first *footnote* block\r\n\r\n    - nested item\r\n\r\n$$z^2$$\r\n")
	backend := native.New()
	firstFacts := collectSemanticFacts(t, backend, source)

	var canonical bytes.Buffer
	if err := Render(&canonical, source, backend); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if bytes.Contains(canonical.Bytes(), []byte("\r")) {
		t.Fatalf("canonical output contains CR bytes: %q", canonical.Bytes())
	}
	secondFacts := collectSemanticFacts(t, backend, canonical.Bytes())
	if !reflect.DeepEqual(secondFacts, firstFacts) {
		t.Fatalf("semantic round trip changed: %s\noutput:\n%s", semanticFactsDifference(firstFacts, secondFacts), canonical.Bytes())
	}

	var second bytes.Buffer
	if err := Render(&second, canonical.Bytes(), backend); err != nil {
		t.Fatalf("second Render() error = %v", err)
	}
	if !bytes.Equal(second.Bytes(), canonical.Bytes()) {
		t.Fatalf("canonical output is not idempotent\nfirst:\n%s\nsecond:\n%s", canonical.Bytes(), second.Bytes())
	}
}

func TestPublishedCommonMarkCanonicalSemanticRoundTrip(t *testing.T) {
	path := os.Getenv("MARKSPLICE_COMMONMARK_SPEC_HTML")
	if path == "" {
		t.Skip("MARKSPLICE_COMMONMARK_SPEC_HTML is not set")
	}
	cases, err := commonmarkspec.LoadPublished(path)
	if err != nil {
		t.Fatalf("load CommonMark spec: %v", err)
	}
	if len(cases) != 652 {
		t.Fatalf("CommonMark case count = %d, want 652", len(cases))
	}
	backend := native.New()
	for _, case_ := range cases {
		case_ := case_
		t.Run(strconv.Itoa(case_.Number), func(t *testing.T) {
			assertCanonicalSemanticRoundTrip(t, backend, []byte(case_.Markdown))
		})
	}
}

func TestPublishedGFMCanonicalSemanticRoundTrip(t *testing.T) {
	path := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if path == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(path)
	if err != nil {
		t.Fatalf("load GFM spec: %v", err)
	}
	if len(cases) != 677 {
		t.Fatalf("GFM case count = %d, want 677", len(cases))
	}
	backend := native.New()
	for _, case_ := range cases {
		case_ := case_
		t.Run(strconv.Itoa(case_.Number), func(t *testing.T) {
			assertCanonicalSemanticRoundTrip(t, backend, []byte(case_.Markdown))
		})
	}
}

func TestCanonicalMarkdownEdgeSyntaxRoundTrips(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		check  func(*testing.T, string)
	}{
		{
			name:   "code span containing backtick",
			source: "`` ` ``\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "`` ` ``\n" {
					t.Fatalf("canonical code span = %q", canonical)
				}
			},
		},
		{
			name:   "code span containing only spaces",
			source: "`  `\n",
		},
		{
			name:   "title quote canonicalization",
			source: "[label](target(a).md 'he said \"hi\"')\n\n[ref]: /target 'he said \"again\"'\n\n[use][ref]\n",
		},
		{
			name:   "standalone reference definition is retained",
			source: "[unused]: /target 'title'\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "[unused]: </target> \"title\"\n" {
					t.Fatalf("canonical standalone reference definition = %q", canonical)
				}
			},
		},
		{
			name:   "multiline reference definition is synthesized canonically",
			source: "[foo]:\n/url\n\n[foo]\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "[foo][foo]\n\n[foo]: </url>\n") {
					t.Fatalf("canonical multiline reference definition = %q", canonical)
				}
			},
		},
		{
			name:   "ordered list crosses digit width",
			source: "9. nine\n10. ten\n11. eleven\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "9. nine\n10. ten\n11. eleven\n") {
					t.Fatalf("ordered markers not sequential: %q", canonical)
				}
			},
		},
		{
			name:   "nested strikethrough",
			source: "~~outer ~inner~ end~~\n",
		},
		{
			name:   "empty heading and thematic break",
			source: "#\n\n***\n",
		},
		{
			name:   "terminal indented code preserves EOF payload",
			source: "    code",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "    code" {
					t.Fatalf("terminal indented code = %q", canonical)
				}
			},
		},
		{
			name:   "terminal fenced code preserves EOF payload",
			source: "```go\ncode",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if canonical != "```go\ncode" {
					t.Fatalf("terminal fenced code = %q", canonical)
				}
			},
		},
		{
			name:   "synthetic reference precedes terminal code",
			source: "[foo]:\n/url\n\n[foo]\n\n    code",
		},
		{
			name:   "footnote-shaped reference link renders inline",
			source: "[CVE][^8] and note[^8]\n\n[^8]: https://example.test/advisory\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "[CVE](<https://example.test/advisory>)") {
					t.Fatalf("footnote-shaped reference remained ambiguous: %q", canonical)
				}
			},
		},
		{
			name:   "nested list keeps following opaque sibling at parent depth",
			source: "- outer\n    - inner\n    <!--\n    sibling\n    -->\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "- outer\n    - inner\n    <!--\n") {
					t.Fatalf("nested list and sibling indentation = %q", canonical)
				}
			},
		},
		{
			name:   "linked reference image uses stable shortcut form",
			source: "[![Build Status]][actions]\n\n[Build Status]: https://example.test/badge.svg\n[actions]: https://example.test/actions\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "[![Build Status]][actions]") {
					t.Fatalf("linked reference image did not use shortcut form: %q", canonical)
				}
			},
		},
		{
			name:   "shortcut image keeps reference punctuation spelling",
			source: "[![serde_derive msrv]][Rust 1.71]\n\n[serde_derive msrv]: https://example.test/badge.svg\n[Rust 1.71]: https://example.test/rust\n",
			check: func(t *testing.T, canonical string) {
				t.Helper()
				if !strings.Contains(canonical, "[![serde_derive msrv]][Rust 1.71]") {
					t.Fatalf("shortcut image label spelling = %q", canonical)
				}
			},
		},
	}

	backend := native.New()
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			source := []byte(test.source)
			before := collectSemanticFacts(t, backend, source)
			var output bytes.Buffer
			if err := Render(&output, source, backend); err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			after := collectSemanticFacts(t, backend, output.Bytes())
			if !reflect.DeepEqual(after, before) {
				t.Fatalf("semantic facts changed\nbefore: %#v\nafter:  %#v\ncanonical: %q", before, after, output.String())
			}
			var second bytes.Buffer
			if err := Render(&second, output.Bytes(), backend); err != nil {
				t.Fatalf("second Render() error = %v", err)
			}
			if second.String() != output.String() {
				t.Fatalf("not idempotent: first=%q second=%q", output.String(), second.String())
			}
			if test.check != nil {
				test.check(t, output.String())
			}
		})
	}
}

func assertCanonicalSemanticRoundTrip(t *testing.T, backend Backend, source []byte) {
	t.Helper()
	before := collectSemanticFacts(t, backend, source)
	var output bytes.Buffer
	if err := Render(&output, source, backend); err != nil {
		t.Fatalf("Render() error = %v\nsource=%q", err, source)
	}
	after := collectSemanticFacts(t, backend, output.Bytes())
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("semantic facts changed: %s\nsource: %q\ncanonical: %q", semanticFactsDifference(before, after), source, output.Bytes())
	}
	var second bytes.Buffer
	if err := Render(&second, output.Bytes(), backend); err != nil {
		t.Fatalf("second Render() error = %v\ncanonical=%q", err, output.Bytes())
	}
	if !bytes.Equal(second.Bytes(), output.Bytes()) {
		t.Fatalf("canonical output is not idempotent\nfirst:  %q\nsecond: %q", output.Bytes(), second.Bytes())
	}
}

func semanticFactsDifference(before, after []semanticFact) string {
	limit := len(before)
	if len(after) < limit {
		limit = len(after)
	}
	for index := 0; index < limit; index++ {
		if before[index] != after[index] {
			return fmt.Sprintf("index=%d before=%#v after=%#v", index, before[index], after[index])
		}
	}
	return fmt.Sprintf("length before=%d after=%d", len(before), len(after))
}

func collectSemanticFacts(t *testing.T, backend Backend, source []byte) []semanticFact {
	t.Helper()
	facts := make([]semanticFact, 0, 32)
	footnotes := make([][]semanticFact, 0, 4)
	var currentFootnote []semanticFact
	if err := backend.WalkSemantic(source, func(event parser.SemanticEvent) error {
		// Reference-definition leaf projection depends on source shape: Native can
		// resolve a multiline definition for link semantics without projecting a
		// SemanticReferenceDefinition leaf. Canonical rendering intentionally turns
		// such targets into a single-line definition, so definition-leaf presence is
		// not part of the semantic-equivalence oracle. Dedicated tests above still
		// require standalone definitions to be retained and missing projected
		// definitions to be synthesized deterministically.
		if event.Kind == parser.SemanticReferenceDefinition {
			return nil
		}
		fact := semanticFactForEvent(event)
		if currentFootnote != nil {
			currentFootnote = append(currentFootnote, fact)
			if event.Kind == parser.SemanticFootnoteDefinition && event.Phase == parser.SemanticExit {
				footnotes = append(footnotes, currentFootnote)
				currentFootnote = nil
			}
			return nil
		}
		if event.Kind == parser.SemanticFootnoteDefinition && event.Phase == parser.SemanticEnter {
			currentFootnote = []semanticFact{fact}
			return nil
		}
		facts = append(facts, fact)
		return nil
	}); err != nil {
		t.Fatalf("WalkSemantic() error = %v", err)
	}
	if currentFootnote != nil {
		t.Fatal("WalkSemantic() ended with an open footnote definition")
	}
	// Native may replace a captured paragraph with a footnote definition or add
	// the same definition as a top-level semantic overlay. Canonical source shape
	// can therefore move an otherwise identical definition subtree within the
	// event stream. Compare every definition subtree intact, but independently
	// from flow position and in deterministic order.
	slices.SortFunc(footnotes, func(left, right []semanticFact) int {
		return strings.Compare(fmt.Sprintf("%#v", left), fmt.Sprintf("%#v", right))
	})
	for _, footnote := range footnotes {
		facts = append(facts, footnote...)
	}
	return facts
}

func semanticFactForEvent(event parser.SemanticEvent) semanticFact {
	value := event.Value
	switch event.Kind {
	case parser.SemanticFrontMatter, parser.SemanticRawHTML, parser.SemanticCodeBlock:
		value = normalizeLineEndings(value)
	case parser.SemanticHTMLBlock:
		value = strings.TrimSuffix(normalizeLineEndings(value), "\n")
	}
	label := event.Label
	if event.Kind == parser.SemanticLink || event.Kind == parser.SemanticImage {
		// Reference-style versus inline links are source representations of the
		// same rendered relationship. Destination, title, and inline content carry
		// the semantic contract; the parser-owned label spelling does not.
		label = ""
	}
	return semanticFact{
		phase:       event.Phase,
		kind:        event.Kind,
		value:       value,
		level:       event.Level,
		destination: event.Destination,
		title:       decodeMarkdownSyntaxString(event.Title),
		hasTitle:    event.HasTitle,
		label:       label,
		email:       event.AutoLinkEmail,
		ordered:     event.Ordered,
		start:       event.Start,
		tight:       event.Tight,
		checked:     event.Checked,
		header:      event.Header,
		column:      event.Column,
		columns:     event.Columns,
		alignment:   event.Alignment,
		language:    event.Language,
		info:        event.Info,
		alert:       event.AlertKind,
		frontMatter: event.FrontMatterFormat,
		math:        event.MathStyle,
	}
}

func decodeMarkdownSyntaxString(value string) string {
	var output strings.Builder
	for position := 0; position < len(value); {
		if value[position] == '\\' && position+1 < len(value) && isASCIIPunctuation(value[position+1]) {
			output.WriteByte(value[position+1])
			position += 2
			continue
		}
		output.WriteByte(value[position])
		position++
	}
	return output.String()
}
