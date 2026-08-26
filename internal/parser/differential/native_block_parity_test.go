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

func TestNativeBlockParserMatchesPublishedGFMBlockProjection(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	compared := 0
	for _, case_ := range cases {
		if slices.Contains(case_.Extensions, "tagfilter") {
			continue
		}
		if err := harness.Compare([]byte(case_.Markdown)); err != nil {
			t.Fatalf("BlockHarness.Compare(example %d) error = %v", case_.Number, err)
		}
		compared++
	}
	if compared != 676 {
		t.Fatalf("compared published GFM examples = %d, want 676", compared)
	}
}

func TestNativeBlockParserMatchesPublishedGFMLeafBlockSections(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	sections := []struct {
		name        string
		first, last int
		kind        parser.Kind
	}{
		{name: "thematic breaks", first: 13, last: 31, kind: parser.KindThematicBreak},
		{name: "atx headings", first: 32, last: 49, kind: parser.KindHeading},
		{name: "setext headings", first: 50, last: 76, kind: parser.KindHeading},
		{name: "fenced code blocks", first: 89, last: 117, kind: parser.KindFencedCode},
		{name: "paragraphs", first: 189, last: 196, kind: parser.KindParagraph},
	}
	for _, section := range sections {
		section := section
		t.Run(section.name, func(t *testing.T) {
			for number := section.first; number <= section.last; number++ {
				case_ := cases[number-1]
				if err := harness.CompareKinds([]byte(case_.Markdown), section.kind); err != nil {
					t.Fatalf("BlockHarness.CompareKinds(example %d) error = %v", number, err)
				}
			}
		})
	}
	if err := harness.Compare([]byte(cases[196].Markdown)); err != nil {
		t.Fatalf("BlockHarness.Compare(blank-line example 197) error = %v", err)
	}
}

func TestNativeBlockParserMatchesPublishedGFMHTMLBlockSection(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	for number := 118; number <= 160; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindHTMLBlock); err != nil {
			t.Fatalf("BlockHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeBlockParserMatchesPublishedGFMReferenceDefinitionSection(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	for number := 161; number <= 188; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindReferenceDefinition); err != nil {
			t.Fatalf("BlockHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeBlockParserMatchesPublishedGFMTableSection(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	for number := 198; number <= 205; number++ {
		case_ := cases[number-1]
		if err := harness.Compare([]byte(case_.Markdown)); err != nil {
			t.Fatalf("BlockHarness.Compare(example %d) error = %v", number, err)
		}
	}
}

func TestNativeBlockParserMatchesPublishedGFMBlockquoteSection(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	for number := 206; number <= 230; number++ {
		case_ := cases[number-1]
		if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindBlockquote); err != nil {
			t.Fatalf("BlockHarness.CompareKinds(example %d) error = %v", number, err)
		}
	}
}

func TestNativeBlockParserMatchesPublishedGFMListSections(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	sections := []struct {
		name        string
		first, last int
	}{
		{name: "list items", first: 231, last: 278},
		{name: "task list items", first: 279, last: 280},
		{name: "lists", first: 281, last: 307},
	}
	for _, section := range sections {
		section := section
		t.Run(section.name, func(t *testing.T) {
			for number := section.first; number <= section.last; number++ {
				case_ := cases[number-1]
				if err := harness.CompareKinds([]byte(case_.Markdown), parser.KindListItem, parser.KindTask); err != nil {
					t.Fatalf("BlockHarness.CompareKinds(example %d) error = %v", number, err)
				}
			}
		})
	}
}

func TestNativeBlockParserMatchesPublishedGFMFullListProjection(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	for number := 231; number <= 307; number++ {
		case_ := cases[number-1]
		if err := harness.Compare([]byte(case_.Markdown)); err != nil {
			t.Fatalf("BlockHarness.Compare(example %d) error = %v", number, err)
		}
	}
}

func TestNativeBlockParserMatchesGoldmarkBlockquoteProjection(t *testing.T) {
	t.Parallel()

	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	for _, source := range []string{
		"> quoted *text*\n",
		"> first\nsecond\n",
		"> foo\nbar\n===\n",
		">\t\tfoo\n",
	} {
		if err := harness.Compare([]byte(source)); err != nil {
			t.Fatalf("BlockHarness.Compare(%q) error = %v", source, err)
		}
	}
}

func TestNativeBlockParserMatchesGoldmarkListProjection(t *testing.T) {
	t.Parallel()

	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	tests := []struct {
		name   string
		source string
	}{
		{name: "root item", source: "- root\n"},
		{name: "nested siblings", source: "- root\n  - child\n  - sibling\n"},
		{name: "nested list without line-bearing parent child", source: "- - foo\n"},
		{name: "mixed ordered nesting", source: "1. root\n   1) parent\n      + child\n"},
		{name: "tab leaves virtual indentation for deeper child", source: " - foo\n   - bar\n\t - baz\n"},
		{name: "blank line keeps container", source: "- one\n\n- two\n"},
		{name: "blank opener with trailing spaces", source: "-   \n  foo\n"},
		{name: "blank opener separated from content", source: "-\n\n  foo\n"},
		{name: "paragraph separates containers", source: "- one\n\nParagraph.\n\n- two\n"},
		{name: "ordered delimiter change starts new container", source: "1. foo\n2. bar\n3) baz\n"},
		{name: "simple parent promotion only", source: "- parent\n  - child\n- leaf\n\n- complex\n\n  second paragraph\n"},
		{name: "tasks", source: "- [x] done\n- [ ] open\n"},
		{name: "nested thematic precedence", source: "- Foo\n- * * *\n"},
		{name: "list beats setext candidate", source: "- Foo\n---\n"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := harness.Compare([]byte(tt.source)); err != nil {
				t.Fatalf("BlockHarness.Compare() error = %v", err)
			}
		})
	}
}

func TestNativeBlockParserMatchesGoldmarkLeafProjection(t *testing.T) {
	t.Parallel()

	harness := BlockHarness{Oracle: goldmarkparser.New(), Candidate: nativeparser.ParseBlocks}
	tests := []struct {
		name   string
		source string
	}{
		{name: "empty", source: ""},
		{name: "paragraph lf", source: "first\nsecond\n\nthird\n"},
		{name: "paragraph crlf", source: "first\r\nsecond\r\n\r\nthird\r\n"},
		{name: "atx headings", source: "# one\n\n###### six\n"},
		{name: "setext headings", source: "one\n===\n\ntwo\n---\n"},
		{name: "paragraph interrupted by heading", source: "text\n# head\n"},
		{name: "thematic breaks", source: "***\n\n- - -\n\n___\n"},
		{name: "empty fence", source: "```\n```\n"},
		{name: "tilde fence", source: "  ~~~~ go test  \n  one\n    two\n  ~~~~\n"},
		{name: "unclosed fence", source: "```text\nbody\n"},
		{name: "indented code remains unobserved", source: "    code\n\nparagraph\n"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := harness.Compare([]byte(tt.source)); err != nil {
				t.Fatalf("BlockHarness.Compare() error = %v", err)
			}
		})
	}
}
