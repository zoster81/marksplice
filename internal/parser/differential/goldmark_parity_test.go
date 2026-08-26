package differential

import (
	"os"
	"slices"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	goldmarkparser "github.com/zoster81/marksplice/internal/parser/goldmark"
	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

func TestGoldmarkBackendsMatchPublishedGFMDifferentialCorpus(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}

	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	stats := gfmspec.Summarize(cases)
	if stats != (gfmspec.Stats{Total: 677, Core: 649, Table: 8, TaskList: 2, Strikethrough: 3, Autolink: 14, TagFilter: 1}) {
		t.Fatalf("unexpected published GFM corpus shape: %+v", stats)
	}

	harness := Harness{Oracle: goldmarkparser.New(), Candidate: goldmarkparser.New()}
	compared := 0
	for _, case_ := range cases {
		if slices.Contains(case_.Extensions, "tagfilter") {
			continue
		}
		if err := harness.CompareDocument([]byte(case_.Markdown)); err != nil {
			t.Fatalf("CompareDocument(example %d) error = %v", case_.Number, err)
		}
		compared++
	}
	if compared != 676 {
		t.Fatalf("compared published GFM examples = %d, want 676", compared)
	}
}

func TestGoldmarkBackendsMatchFocusedSemanticSourceRegressions(t *testing.T) {
	t.Parallel()

	harness := Harness{Oracle: goldmarkparser.New(), Candidate: goldmarkparser.New()}
	cases := []struct {
		name   string
		source string
	}{
		{name: "gfm profile guards", source: "~strike~ ~~double~~ mailto:foo@bar.baz xmpp:foo@bar.baz/txt@bin.com ftp://example.com\n"},
		{name: "html comment compatibility", source: "foo <!-- valid comment -->\n\nfoo <!-- not a comment -- two hyphens -->\n"},
		{name: "resolved reference forms", source: "[docs]: <target> \"Guide\"\n\n[full][docs] [docs][] [docs] ![docs]\n"},
		{name: "unresolved explicit references", source: "[full][missing] [collapsed][] ![image][missing-image] [shortcut] \\[escaped][missing] `[code][missing]` [resolved][ok]\n\n[ok]: /target\n"},
		{name: "thematic break", source: "before\n\n---\n\nafter\n"},
		{name: "simple blockquote", source: "before\n\n> quoted *text*\n\nafter\n"},
		{name: "lazy blockquote continuation", source: "> first\nsecond\n"},
		{name: "table ownership", source: "before\n\n| A | B |\n| - | - |\n| x | y |\n"},
		{name: "table alignments", source: "| Left | Right | Center | Default |\n| :--- | ---: | :---: | --- |\n| l | r | c | d |\n"},
		{name: "table empty cell", source: "| A | B |\n| - | - |\n|   | value |\n"},
		{name: "empty fenced body", source: "before\n\n```math\n```\n"},
		{name: "indented fenced body", source: "  ~~~~  mermaid diagram  \n  graph TD\n  A-->B\n ~~~~~~   \n"},
		{name: "raw html crlf", source: "before <!-- old --> <a id=\"anchor\">text</a>\n\n<div data-x=\"1\">\r\n*not markdown*\r\n</div>\r\n\r\nafter\r\n"},
		{name: "simple image", source: "before ![alt](old/path \"title\") after\n"},
		{name: "compound image alt", source: "![**alt**](old/path)\n"},
		{name: "nested list anchors", source: "1. root\n   1) parent\n      + child\n"},
		{name: "separate list containers", source: "- one\n\nParagraph.\n\n- two\n"},
		{name: "supported list parent", source: "- parent\n  - child\n- leaf\n\n- complex\n\n  second paragraph\n"},
		{name: "unsupported list trailing paragraph", source: "- parent\n\n  second paragraph\n"},
		{name: "footnote source ordering", source: "before[^b] and again[^a] and again[^a]\n\n[^a]: first line\n\n    second paragraph\n\n[^unused]: unused\n[^b]: bee\n"},
		{name: "footnote reference conflict", source: "foot[^n] [normal][^n] [ok][docs]\n\n[^n]: note\n\n[docs]: /target\n"},
		{name: "links inside footnotes", source: "[outside](#out)\n\n[^n]: [inside](b.md#part) <https://example.com>\n"},
		{name: "math ownership exclusions", source: "plain $x+1$ and $`a*b`$\n\n$$x^2$$\r\n\n`$code$` [link $x$](https://example.com)\n\n> $$quoted$$\n\n```text\n$fenced$\n```\n"},
		{name: "math conflict suppression", source: "$`a*b`$\n\n$$x^2$$\n\n`ordinary`\n\nparagraph\n"},
		{name: "setext and crlf", source: "Title\r\n=====\r\n\r\nparagraph  \r\n"},
	}
	for _, case_ := range cases {
		if err := harness.CompareDocument([]byte(case_.source)); err != nil {
			t.Fatalf("CompareDocument(%s) error = %v", case_.name, err)
		}
	}
}

func TestGoldmarkBackendsMatchCompleteM111Contract(t *testing.T) {
	t.Parallel()

	harness := Harness{Oracle: goldmarkparser.New(), Candidate: goldmarkparser.New()}
	documents := [][]byte{
		[]byte("# Heading\n\nparagraph *em* [link](target.md#part)\n"),
		[]byte("- [x] task\n\n| A | B |\n| :--- | ---: |\n| x | y |\n"),
		[]byte("Text[^n] and $x+1$.\n\n[^n]: note [inside](other.md)\n"),
		[]byte("Title\r\n=====\r\n\r\n> quoted\r\n\r\n```go\r\ncode\r\n```\r\n"),
		[]byte("[missing][ref]\n\n<a id=\"anchor\"></a>\n"),
	}
	for index, source := range documents {
		if err := harness.CompareDocument(source); err != nil {
			t.Fatalf("CompareDocument(case %d) error = %v", index, err)
		}
	}

	blockquote := []byte("> > text\n")
	if err := harness.CompareNestedBlockquoteParagraph(
		blockquote,
		parser.Range{Start: 0, End: len(blockquote) - 1},
		[]parser.Range{{Start: 4, End: 8}},
		2,
	); err != nil {
		t.Fatalf("CompareNestedBlockquoteParagraph() error = %v", err)
	}
	if err := harness.CompareNestedBlockquoteParagraph(
		[]byte("> > > text\n"),
		parser.Range{Start: 0, End: len("> > > text")},
		[]parser.Range{{Start: 6, End: 10}},
		2,
	); err != nil {
		t.Fatalf("CompareNestedBlockquoteParagraph(rejected depth) error = %v", err)
	}

	inner := []byte("## Head\n\ntext\n")
	blocks := []byte("> > ## Head\n> > \n> > text\n")
	if err := harness.CompareNestedBlockquoteBlocks(blocks, parser.Range{Start: 0, End: len(blocks) - 1}, inner, 2); err != nil {
		t.Fatalf("CompareNestedBlockquoteBlocks() error = %v", err)
	}
	changedBlocks := []byte("> > ### Head\n> > \n> > text\n")
	if err := harness.CompareNestedBlockquoteBlocks(changedBlocks, parser.Range{Start: 0, End: len(changedBlocks) - 1}, inner, 2); err != nil {
		t.Fatalf("CompareNestedBlockquoteBlocks(rejected heading) error = %v", err)
	}

	inline := []byte("*x*")
	inlineExpected := []parser.ConstructionInlineExpectation{{
		Kind:            parser.KindEmphasis,
		SyntaxRange:     parser.Range{Start: 0, End: 3},
		ContentRange:    parser.Range{Start: 1, End: 2},
		Marker:          '*',
		DelimiterLength: 1,
		Parent:          -1,
	}}
	if err := harness.CompareConstructionInlineHierarchy(inline, inlineExpected, nil); err != nil {
		t.Fatalf("CompareConstructionInlineHierarchy() error = %v", err)
	}
	changedInline := slices.Clone(inlineExpected)
	changedInline[0].Marker = '_'
	if err := harness.CompareConstructionInlineHierarchy(inline, changedInline, nil); err != nil {
		t.Fatalf("CompareConstructionInlineHierarchy(rejected marker) error = %v", err)
	}

	direct := []byte("[x](<y>)")
	directExpected := []parser.ConstructionLinkImageExpectation{{
		Kind:        parser.KindInlineLink,
		SyntaxRange: parser.Range{Start: 0, End: len(direct)},
		LabelRange:  parser.Range{Start: 1, End: 2},
		Destination: "y",
	}}
	if err := harness.CompareConstructionLinkImages(direct, directExpected); err != nil {
		t.Fatalf("CompareConstructionLinkImages() error = %v", err)
	}
	changedDirect := slices.Clone(directExpected)
	changedDirect[0].Destination = "other"
	if err := harness.CompareConstructionLinkImages(direct, changedDirect); err != nil {
		t.Fatalf("CompareConstructionLinkImages(rejected destination) error = %v", err)
	}

	reference := []byte("[x][ref]")
	referenceExpected := []parser.ConstructionReferenceInlineExpectation{{
		Kind:           parser.KindInlineLink,
		Form:           parser.ConstructionReferenceInlineFull,
		SyntaxRange:    parser.Range{Start: 0, End: len(reference)},
		LabelRange:     parser.Range{Start: 1, End: 2},
		ReferenceRange: parser.Range{Start: 4, End: 7},
		Reference:      "ref",
		Destination:    "/target",
	}}
	if err := harness.CompareConstructionReferenceInlines(reference, referenceExpected); err != nil {
		t.Fatalf("CompareConstructionReferenceInlines() error = %v", err)
	}
	changedReference := slices.Clone(referenceExpected)
	changedReference[0].Reference = "other"
	if err := harness.CompareConstructionReferenceInlines(reference, changedReference); err != nil {
		t.Fatalf("CompareConstructionReferenceInlines(rejected reference) error = %v", err)
	}

	definitions := []parser.ConstructionReferenceDefinition{{Label: "Ref Label", Destination: "/target", Title: "Guide", HasTitle: true}}
	if err := harness.CompareConstructionReferenceResolution("ref label", definitions); err != nil {
		t.Fatalf("CompareConstructionReferenceResolution() error = %v", err)
	}
	ambiguousDefinitions := append(slices.Clone(definitions), parser.ConstructionReferenceDefinition{Label: " ref   label ", Destination: "/other"})
	if err := harness.CompareConstructionReferenceResolution("ref label", ambiguousDefinitions); err != nil {
		t.Fatalf("CompareConstructionReferenceResolution(ambiguous) error = %v", err)
	}
	for _, label := range []string{"Ref Label", " ref   label ", "π", "A\\*B"} {
		if err := harness.CompareReferenceLabelKey(label); err != nil {
			t.Fatalf("CompareReferenceLabelKey(%q) error = %v", label, err)
		}
	}
}
