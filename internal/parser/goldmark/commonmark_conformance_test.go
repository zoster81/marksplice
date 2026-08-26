package goldmark

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	goldmarklib "github.com/yuin/goldmark"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"

	"github.com/zoster81/marksplice/internal/testutil/commonmarkspec"
)

type commonMarkMismatch struct {
	example  int
	section  string
	markdown string
	got      string
	want     string
}

func TestCommonMark0312PublishedSpecificationAudit(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_COMMONMARK_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_COMMONMARK_SPEC_HTML is not set")
	}
	cases, err := commonmarkspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published CommonMark spec: %v", err)
	}
	if len(cases) != 652 {
		t.Fatalf("published CommonMark example count = %d, want 652", len(cases))
	}

	markdown := goldmarklib.New(goldmarklib.WithRendererOptions(goldmarkhtml.WithXHTML(), goldmarkhtml.WithUnsafe()))
	mismatches := make([]commonMarkMismatch, 0)
	for _, tc := range cases {
		var output bytes.Buffer
		if err := markdown.Convert([]byte(tc.Markdown), &output); err != nil {
			t.Fatalf("example %d (%s): Convert() error: %v", tc.Number, tc.Section, err)
		}
		if got := output.String(); got != tc.HTML {
			mismatches = append(mismatches, commonMarkMismatch{
				example:  tc.Number,
				section:  tc.Section,
				markdown: tc.Markdown,
				got:      got,
				want:     tc.HTML,
			})
		}
	}
	if len(mismatches) != 0 {
		t.Fatalf("%d of %d CommonMark 0.31.2 examples differ; first mismatches:\n%s", len(mismatches), len(cases), formatCommonMarkMismatches(mismatches, 40))
	}
}

func formatCommonMarkMismatches(mismatches []commonMarkMismatch, limit int) string {
	if limit > len(mismatches) {
		limit = len(mismatches)
	}
	formatted := make([]string, 0, limit)
	for _, mismatch := range mismatches[:limit] {
		formatted = append(formatted, fmt.Sprintf("example %d (%s)\nmarkdown:\n%s\n got:\n%s\nwant:\n%s", mismatch.example, mismatch.section, mismatch.markdown, mismatch.got, mismatch.want))
	}
	return strings.Join(formatted, "\n---\n")
}
