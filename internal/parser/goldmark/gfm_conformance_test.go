package goldmark

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	htmlstd "html"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	goldmarklib "github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

const publishedGFMSpecSHA256 = "b153d814fdfc8624bb6da7449162c1cd707a637f7d1c1b636eb44b9cf63fa220"

type gfmSpecCase struct {
	number     int
	extensions []string
	markdown   string
	html       string
}

type gfmCorpusStats struct {
	total         int
	core          int
	table         int
	tasklist      int
	strikethrough int
	autolink      int
	tagfilter     int
}

type gfmMismatch struct {
	example    int
	extensions []string
	markdown   string
	got        string
	want       string
}

type gfmEvaluation struct {
	validated  int
	mismatches []gfmMismatch
}

type gfmSection struct {
	start int
	id    string
}

var (
	gfmH2Pattern      = regexp.MustCompile(`(?s)<h2 id="([^"]+)"[^>]*>`)
	gfmExamplePattern = regexp.MustCompile(
		`(?s)<div class="example" id="example-([0-9]+)">.*?<pre><code class="language-markdown">(.*?)</code></pre>.*?<pre><code class="language-html">(.*?)</code></pre>`,
	)
	gfmHTMLTagPattern = regexp.MustCompile(`<[^>]+>`)
)

var gfmExtensionBySection = map[string]string{
	"tables-extension-":              "table",
	"task-list-items-extension-":     "tasklist",
	"strikethrough-extension-":       "strikethrough",
	"autolinks-extension-":           "autolink",
	"disallowed-raw-html-extension-": "tagfilter",
}

func TestVerifyPublishedGFMSpecSnapshotRejectsUnexpectedBytes(t *testing.T) {
	t.Parallel()

	if err := verifyPublishedGFMSpecSnapshot([]byte("Version 0.29-gfm\nchanged")); err == nil {
		t.Fatal("verifyPublishedGFMSpecSnapshot() error = nil, want snapshot mismatch")
	}
}

func TestGFM029PublishedSpecificationConformance(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}

	cases, err := loadGFMPublishedSpecCases(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	stats := summarizeGFMSpec(cases)
	if stats.total != 677 || stats.core != 649 || stats.table != 8 || stats.tasklist != 2 || stats.strikethrough != 3 || stats.autolink != 14 || stats.tagfilter != 1 {
		t.Fatalf("unexpected published GFM corpus shape: %+v", stats)
	}

	result, err := evaluateGFMSpec(cases)
	if err != nil {
		t.Fatalf("evaluate published GFM spec: %v", err)
	}
	if result.validated != 676 {
		t.Fatalf("validated examples = %d, want 676", result.validated)
	}
	if len(result.mismatches) != 0 {
		t.Fatalf("%d of %d validated GFM examples differ; first mismatches:\n%s", len(result.mismatches), result.validated, formatGFMMismatches(result.mismatches, 20))
	}
}

func summarizeGFMSpec(cases []gfmSpecCase) gfmCorpusStats {
	stats := gfmCorpusStats{total: len(cases)}
	for _, tc := range cases {
		if len(tc.extensions) == 0 {
			stats.core++
		}
		if slices.Contains(tc.extensions, "table") {
			stats.table++
		}
		if slices.Contains(tc.extensions, "tasklist") {
			stats.tasklist++
		}
		if slices.Contains(tc.extensions, "strikethrough") {
			stats.strikethrough++
		}
		if slices.Contains(tc.extensions, "autolink") {
			stats.autolink++
		}
		if slices.Contains(tc.extensions, "tagfilter") {
			stats.tagfilter++
		}
	}
	return stats
}

func evaluateGFMSpec(cases []gfmSpecCase) (gfmEvaluation, error) {
	result := gfmEvaluation{mismatches: make([]gfmMismatch, 0)}
	for _, tc := range cases {
		if slices.Contains(tc.extensions, "tagfilter") {
			continue
		}

		markdown, err := markdownForGFMSpecCase(tc.extensions)
		if err != nil {
			return gfmEvaluation{}, fmt.Errorf("example %d: %w", tc.number, err)
		}
		var output bytes.Buffer
		if err := markdown.Convert([]byte(tc.markdown), &output); err != nil {
			return gfmEvaluation{}, fmt.Errorf("example %d (%s): Convert() error: %w", tc.number, strings.Join(tc.extensions, ","), err)
		}
		if got := output.String(); got != tc.html {
			result.mismatches = append(result.mismatches, gfmMismatch{
				example:    tc.number,
				extensions: append([]string(nil), tc.extensions...),
				markdown:   tc.markdown,
				got:        got,
				want:       tc.html,
			})
		}
		result.validated++
	}
	return result, nil
}

func formatGFMMismatches(mismatches []gfmMismatch, limit int) string {
	if limit > len(mismatches) {
		limit = len(mismatches)
	}
	formatted := make([]string, 0, limit)
	for _, mismatch := range mismatches[:limit] {
		formatted = append(formatted, fmt.Sprintf("example %d (%s)\nmarkdown:\n%s\n got:\n%s\nwant:\n%s", mismatch.example, strings.Join(mismatch.extensions, ","), mismatch.markdown, mismatch.got, mismatch.want))
	}
	return strings.Join(formatted, "\n---\n")
}

func markdownForGFMSpecCase(extensionNames []string) (goldmarklib.Markdown, error) {
	extenders := make([]goldmarklib.Extender, 0, len(extensionNames))
	for _, name := range extensionNames {
		switch name {
		case "table":
			extenders = append(extenders, extension.Table)
		case "tasklist":
			extenders = append(extenders, extension.TaskList)
		case "strikethrough":
			extenders = append(extenders, extension.Strikethrough)
		case "autolink":
			extenders = append(extenders, extension.Linkify)
		case "tagfilter":
			return nil, fmt.Errorf("tagfilter is an HTML-rendering extension and is not implemented by the Marksplice parser adapter")
		default:
			return nil, fmt.Errorf("unsupported GFM spec extension %q", name)
		}
	}

	var options []goldmarklib.Option
	if slices.Contains(extensionNames, "tasklist") {
		options = []goldmarklib.Option{
			goldmarklib.WithRendererOptions(goldmarkhtml.WithUnsafe()),
		}
	} else {
		options = []goldmarklib.Option{
			goldmarklib.WithRendererOptions(
				goldmarkhtml.WithXHTML(),
				goldmarkhtml.WithUnsafe(),
			),
		}
	}
	return newMarkdownWithExtensions(extenders, options...), nil
}

func loadGFMPublishedSpecCases(path string) ([]gfmSpecCase, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := verifyPublishedGFMSpecSnapshot(source); err != nil {
		return nil, err
	}
	if !bytes.Contains(source, []byte("Version 0.29-gfm")) {
		return nil, fmt.Errorf("input is not the published GFM 0.29 specification")
	}

	sections := extractGFMSpecSections(source)
	matches := gfmExamplePattern.FindAllSubmatchIndex(source, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no GFM examples found in published specification")
	}

	cases := make([]gfmSpecCase, 0, len(matches))
	for index, match := range matches {
		number, err := strconv.Atoi(string(source[match[2]:match[3]]))
		if err != nil {
			return nil, fmt.Errorf("parse example number: %w", err)
		}
		if number != index+1 {
			return nil, fmt.Errorf("non-contiguous GFM example numbering at index %d: got %d", index, number)
		}

		case_ := gfmSpecCase{
			number:   number,
			markdown: decodeGFMSpecCode(source[match[4]:match[5]]),
			html:     decodeGFMSpecCode(source[match[6]:match[7]]),
		}
		if extensionName := gfmExtensionForPosition(sections, match[0]); extensionName != "" {
			case_.extensions = []string{extensionName}
		}
		cases = append(cases, case_)
	}
	return cases, nil
}

func verifyPublishedGFMSpecSnapshot(source []byte) error {
	actual := fmt.Sprintf("%x", sha256.Sum256(source))
	if actual != publishedGFMSpecSHA256 {
		return fmt.Errorf("published GFM specification snapshot SHA-256 = %s, want %s", actual, publishedGFMSpecSHA256)
	}
	return nil
}

func extractGFMSpecSections(source []byte) []gfmSection {
	matches := gfmH2Pattern.FindAllSubmatchIndex(source, -1)
	sections := make([]gfmSection, 0, len(matches))
	for _, match := range matches {
		sections = append(sections, gfmSection{
			start: match[0],
			id:    string(source[match[2]:match[3]]),
		})
	}
	return sections
}

func gfmExtensionForPosition(sections []gfmSection, position int) string {
	index := sort.Search(len(sections), func(i int) bool {
		return sections[i].start > position
	}) - 1
	if index < 0 {
		return ""
	}
	return gfmExtensionBySection[sections[index].id]
}

func decodeGFMSpecCode(encoded []byte) string {
	withoutMarkup := gfmHTMLTagPattern.ReplaceAll(encoded, nil)
	decoded := htmlstd.UnescapeString(string(withoutMarkup))
	return strings.ReplaceAll(decoded, "→", "\t")
}
