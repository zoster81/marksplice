package native_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/zoster81/marksplice/internal/parser"
	"github.com/zoster81/marksplice/internal/parser/native"
	"github.com/zoster81/marksplice/internal/testutil/commonmarkspec"
	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

type publishedContractFixture struct {
	Schema        int                     `json:"schema"`
	Specification string                  `json:"specification"`
	SpecSHA256    string                  `json:"spec_sha256"`
	CaseCount     int                     `json:"case_count"`
	Cases         []publishedContractCase `json:"cases"`
}

type publishedContractCase struct {
	Number         int                         `json:"number"`
	Section        string                      `json:"section,omitempty"`
	Extensions     []string                    `json:"extensions,omitempty"`
	MarkdownSHA256 string                      `json:"markdown_sha256"`
	Observations   parser.DocumentObservations `json:"observations"`
}

func TestM115NativeMatchesPublishedCommonMark0312Contract(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_COMMONMARK_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_COMMONMARK_SPEC_HTML is not set")
	}
	cases, err := commonmarkspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published CommonMark spec: %v", err)
	}
	fixture := loadPublishedContractFixture(t, "testdata/commonmark-0.31.2-observations.json")
	if fixture.Schema != 1 || fixture.SpecSHA256 != commonmarkspec.PublishedSHA256 || fixture.CaseCount != 652 || len(fixture.Cases) != 652 {
		t.Fatalf("unexpected CommonMark contract metadata: schema=%d spec=%q count=%d/%d", fixture.Schema, fixture.SpecSHA256, fixture.CaseCount, len(fixture.Cases))
	}
	if len(cases) != len(fixture.Cases) {
		t.Fatalf("published CommonMark cases = %d, contract cases = %d", len(cases), len(fixture.Cases))
	}

	backend := native.New()
	for index, tc := range cases {
		want := fixture.Cases[index]
		if want.Number != tc.Number || want.Section != tc.Section || want.MarkdownSHA256 != publishedMarkdownHash(tc.Markdown) {
			t.Fatalf("CommonMark example %d contract identity mismatch: fixture=%+v section=%q", tc.Number, want, tc.Section)
		}
		assertPublishedContractCase(t, backend, tc.Number, tc.Markdown, want.Observations)
	}
}

func TestM115NativeMatchesPublishedGFM029Contract(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	stats := gfmspec.Summarize(cases)
	wantStats := gfmspec.Stats{Total: 677, Core: 649, Table: 8, TaskList: 2, Strikethrough: 3, Autolink: 14, TagFilter: 1}
	if stats != wantStats {
		t.Fatalf("unexpected GFM corpus shape: got %+v want %+v", stats, wantStats)
	}
	fixture := loadPublishedContractFixture(t, "testdata/gfm-0.29-observations.json")
	if fixture.Schema != 1 || fixture.SpecSHA256 != gfmspec.PublishedSHA256 || fixture.CaseCount != 676 || len(fixture.Cases) != 676 {
		t.Fatalf("unexpected GFM contract metadata: schema=%d spec=%q count=%d/%d", fixture.Schema, fixture.SpecSHA256, fixture.CaseCount, len(fixture.Cases))
	}

	backend := native.New()
	fixtureIndex := 0
	for _, tc := range cases {
		if slices.Contains(tc.Extensions, "tagfilter") {
			continue
		}
		if fixtureIndex >= len(fixture.Cases) {
			t.Fatalf("GFM fixture exhausted before example %d", tc.Number)
		}
		want := fixture.Cases[fixtureIndex]
		if want.Number != tc.Number || !slices.Equal(want.Extensions, tc.Extensions) || want.MarkdownSHA256 != publishedMarkdownHash(tc.Markdown) {
			t.Fatalf("GFM example %d contract identity mismatch: fixture=%+v extensions=%v", tc.Number, want, tc.Extensions)
		}
		assertPublishedContractCase(t, backend, tc.Number, tc.Markdown, want.Observations)
		fixtureIndex++
	}
	if fixtureIndex != len(fixture.Cases) {
		t.Fatalf("compared GFM contract cases = %d, want %d", fixtureIndex, len(fixture.Cases))
	}
}

func loadPublishedContractFixture(t *testing.T, path string) publishedContractFixture {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read published parser contract %q: %v", path, err)
	}
	var fixture publishedContractFixture
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("decode published parser contract %q: %v", path, err)
	}
	return fixture
}

func assertPublishedContractCase(t *testing.T, backend parser.Backend, number int, markdown string, want parser.DocumentObservations) {
	t.Helper()
	source := []byte(markdown)
	before := bytes.Clone(source)
	got, err := backend.ParseDocument(source)
	if err != nil {
		t.Fatalf("example %d ParseDocument() error = %v", number, err)
	}
	if !bytes.Equal(source, before) {
		t.Fatalf("example %d ParseDocument() mutated source", number)
	}
	normalizePublishedObservations(&got)
	normalizePublishedObservations(&want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("example %d parser-neutral observations changed\ngot:  %#v\nwant: %#v", number, got, want)
	}
}

func normalizePublishedObservations(observed *parser.DocumentObservations) {
	if observed.Nodes == nil {
		observed.Nodes = []parser.Node{}
	}
	if observed.LinkUsages == nil {
		observed.LinkUsages = []parser.LinkUsage{}
	}
	if observed.UnresolvedReferenceUsages == nil {
		observed.UnresolvedReferenceUsages = []parser.UnresolvedReferenceUsage{}
	}
	if observed.FootnoteDefinitions == nil {
		observed.FootnoteDefinitions = []parser.FootnoteDefinitionObservation{}
	}
	if observed.FootnoteReferences == nil {
		observed.FootnoteReferences = []parser.FootnoteReferenceObservation{}
	}
	if observed.MathExpressions == nil {
		observed.MathExpressions = []parser.MathExpressionObservation{}
	}
	for index := range observed.Nodes {
		node := &observed.Nodes[index]
		if node.BlockquoteSemanticRanges == nil {
			node.BlockquoteSemanticRanges = []parser.Range{}
		}
		if node.FencedCodeContentRanges == nil {
			node.FencedCodeContentRanges = []parser.Range{}
		}
		if node.TableAlignments == nil {
			node.TableAlignments = []parser.TableAlignment{}
		}
	}
	for index := range observed.FootnoteDefinitions {
		if observed.FootnoteDefinitions[index].BodyRanges == nil {
			observed.FootnoteDefinitions[index].BodyRanges = []parser.Range{}
		}
	}
}

func publishedMarkdownHash(source string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(source)))
}
