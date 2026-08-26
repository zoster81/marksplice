package goldmark

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	goldmarklib "github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"

	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

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

func TestVerifyPublishedGFMSpecSnapshotRejectsUnexpectedBytes(t *testing.T) {
	t.Parallel()

	if err := gfmspec.VerifyPublishedSnapshot([]byte("Version 0.29-gfm\nchanged")); err == nil {
		t.Fatal("gfmspec.VerifyPublishedSnapshot() error = nil, want snapshot mismatch")
	}
}

func TestGFM029PublishedExtensionConformance(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}

	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	stats := gfmspec.Summarize(cases)
	if stats.Total != 677 || stats.Core != 649 || stats.Table != 8 || stats.TaskList != 2 || stats.Strikethrough != 3 || stats.Autolink != 14 || stats.TagFilter != 1 {
		t.Fatalf("unexpected published GFM corpus shape: %+v", stats)
	}

	result, err := evaluateGFMExtensions(cases)
	if err != nil {
		t.Fatalf("evaluate published GFM extensions: %v", err)
	}
	if result.validated != 27 {
		t.Fatalf("validated extension examples = %d, want 27", result.validated)
	}
	if len(result.mismatches) != 0 {
		t.Fatalf("%d of %d validated GFM extension examples differ; first mismatches:\n%s", len(result.mismatches), result.validated, formatGFMMismatches(result.mismatches, 20))
	}
}

func evaluateGFMExtensions(cases []gfmspec.Case) (gfmEvaluation, error) {
	result := gfmEvaluation{mismatches: make([]gfmMismatch, 0)}
	for _, tc := range cases {
		if len(tc.Extensions) == 0 || slices.Contains(tc.Extensions, "tagfilter") {
			continue
		}

		markdown, err := markdownForGFMSpecCase(tc.Extensions)
		if err != nil {
			return gfmEvaluation{}, fmt.Errorf("example %d: %w", tc.Number, err)
		}
		var output bytes.Buffer
		if err := markdown.Convert([]byte(tc.Markdown), &output); err != nil {
			return gfmEvaluation{}, fmt.Errorf("example %d (%s): Convert() error: %w", tc.Number, strings.Join(tc.Extensions, ","), err)
		}
		if got := output.String(); got != tc.HTML {
			result.mismatches = append(result.mismatches, gfmMismatch{
				example:    tc.Number,
				extensions: append([]string(nil), tc.Extensions...),
				markdown:   tc.Markdown,
				got:        got,
				want:       tc.HTML,
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
