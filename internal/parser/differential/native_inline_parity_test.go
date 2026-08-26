package differential

import (
	"os"
	"slices"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	goldmarkparser "github.com/zoster81/marksplice/internal/parser/goldmark"
	nativeparser "github.com/zoster81/marksplice/internal/parser/native"
	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

func TestNativeInlineParserMatchesGoldmarkCodeSpan(t *testing.T) {
	t.Parallel()

	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	if err := harness.CompareKinds([]byte("`foo`\n"), parser.KindCodeSpan); err != nil {
		t.Fatalf("InlineHarness.CompareKinds() error = %v", err)
	}
}

func TestNativeInlineParserMatchesPublishedGFMCodeSpanSection(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	for number := 338; number <= 359; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindCodeSpan); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeInlineParserCodeSpanBlockBoundaries(t *testing.T) {
	t.Parallel()

	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	tests := []struct {
		name   string
		source string
	}{
		{name: "tight list", source: "- `code`\n"},
		{name: "blockquote", source: "> `code`\n"},
		{name: "table cell", source: "| `code` |\n| --- |\n"},
		{name: "fenced code excluded", source: "```\n`not code span`\n```\n"},
		{name: "indented code excluded", source: "    `not code span`\n"},
		{name: "html block excluded", source: "<div>\n`not code span`\n</div>\n"},
		{name: "reference definition excluded", source: "[ref]: /target \"`not code span`\"\n"},
		{name: "multiline blockquote span is not promoted", source: "> ``\n> body\n> ``\n"},
		{name: "multiline list span is not promoted", source: "- ``\n  body\n  ``\n"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := harness.CompareKinds([]byte(tt.source), parser.KindCodeSpan); err != nil {
				t.Fatalf("InlineHarness.CompareKinds() error = %v", err)
			}
		})
	}
}

func TestNativeInlineParserMatchesPublishedGFMEmphasisSection(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	for number := 360; number <= 490; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindEmphasis, parser.KindStrong); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeInlineParserMatchesPublishedGFMStrikethroughSection(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	for number := 491; number <= 493; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindStrikethrough); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeReferenceLabelKeyMatchesGoldmark(t *testing.T) {
	t.Parallel()

	for _, label := range []string{"Ref Label", " ref   label ", "ẞ", "SS", "Straße", "STRASSE", "A\\*B", "A\u00a0B"} {
		want := goldmarkparser.ReferenceLabelKey(label)
		if got := nativeparser.ReferenceLabelKey(label); got != want {
			t.Fatalf("ReferenceLabelKey(%q) = %q, want %q", label, got, want)
		}
	}
}

func TestNativeInlineRelationshipProjection(t *testing.T) {
	t.Parallel()

	harness := InlineRelationshipHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlineObservations}
	for _, source := range [][]byte{
		[]byte("[docs]: <target> \"Guide\"\n\n[full][docs] [docs][] [docs] ![docs]\n"),
		[]byte("[full][missing] [collapsed][] ![image][missing-image] [shortcut] \\[escaped][missing] !\\[escaped-image][missing] `[code][missing]` [resolved][ok]\n\n[ok]: /target\n"),
	} {
		if err := harness.CompareRelationships(source); err != nil {
			t.Fatalf("InlineRelationshipHarness.CompareRelationships() error = %v", err)
		}
	}
}

func TestNativeInlineParserMatchesPublishedGFMCombinedProjection(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	kinds := nativeInlineKinds()
	for number := 328; number <= 656; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), kinds...); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d, combined) error = %v", number, err)
		}
	}
}

func TestNativeInlineParserMatchesPublishedGFMInlineProjection(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	kinds := nativeInlineKinds()
	compared := 0
	for _, case_ := range cases {
		if slices.Contains(case_.Extensions, "tagfilter") {
			continue
		}
		if err := harness.CompareKinds([]byte(case_.Markdown), kinds...); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d, full inline) error = %v", case_.Number, err)
		}
		compared++
	}
	if compared != 676 {
		t.Fatalf("compared published GFM examples = %d, want 676", compared)
	}
}

func TestNativeInlineRelationshipMatchesPublishedGFMAllParserExamples(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineRelationshipHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlineObservations}
	compared := 0
	for _, case_ := range cases {
		if slices.Contains(case_.Extensions, "tagfilter") {
			continue
		}
		if err := harness.CompareRelationships([]byte(case_.Markdown)); err != nil {
			t.Fatalf("InlineRelationshipHarness.CompareRelationships(example %d, full corpus) error = %v", case_.Number, err)
		}
		compared++
	}
	if compared != 676 {
		t.Fatalf("compared published GFM examples = %d, want 676", compared)
	}
}

func nativeInlineKinds() []parser.Kind {
	return []parser.Kind{
		parser.KindCodeSpan,
		parser.KindEmphasis,
		parser.KindStrong,
		parser.KindStrikethrough,
		parser.KindInlineLink,
		parser.KindImage,
		parser.KindAutoLink,
		parser.KindRawHTML,
	}
}

func TestNativeInlineRelationshipMatchesPublishedGFMLinkImageAutolinkSections(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineRelationshipHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlineObservations}
	for number := 494; number <= 635; number++ {
		case_ := cases[number-1]
		if err := harness.CompareRelationships([]byte(case_.Markdown)); err != nil {
			t.Fatalf("InlineRelationshipHarness.CompareRelationships(example %d) error = %v", number, err)
		}
	}
}

func TestNativeInlineParserMatchesPublishedGFMLinkSection(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	for number := 494; number <= 580; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindInlineLink); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeInlineParserMatchesPublishedGFMImageSection(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	for number := 581; number <= 602; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindImage); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeInlineParserMatchesPublishedGFMAutolinkSection(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	for number := 603; number <= 621; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindAutoLink); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeInlineParserMatchesPublishedGFMExtendedAutolinkSection(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	for number := 622; number <= 635; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindAutoLink); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeInlineParserMatchesPublishedGFMRawHTMLSection(t *testing.T) {
	cases := loadPublishedInlineCases(t)
	harness := InlineHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseInlines}
	for number := 636; number <= 656; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindRawHTML); err != nil {
			t.Fatalf("InlineHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func loadPublishedInlineCases(t *testing.T) []gfmspec.Case {
	t.Helper()
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	return cases
}
