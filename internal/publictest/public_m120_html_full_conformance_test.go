package publictest

import (
	"os"
	"slices"
	"testing"

	"github.com/zoster81/marksplice"
	"github.com/zoster81/marksplice/internal/testutil/commonmarkspec"
	"github.com/zoster81/marksplice/internal/testutil/gfmspec"
)

var m120CommonMarkProfileDivergences = map[int]string{
	98:  "empty YAML front matter takes precedence over CommonMark thematic breaks",
	608: "Marksplice always enables the reviewed extended-autolink profile",
	611: "Marksplice always enables the reviewed extended-autolink profile",
	612: "Marksplice always enables the reviewed extended-autolink profile",
	625: "Marksplice uses the reviewed GFM HTML-comment grammar",
	626: "Marksplice uses the reviewed GFM HTML-comment grammar",
}

var m120GFMProfileDivergences = map[int]string{
	68:  "empty YAML front matter takes precedence over core thematic breaks",
	617: "Marksplice always enables the reviewed extended-autolink profile",
	620: "Marksplice always enables the reviewed extended-autolink profile",
	621: "Marksplice always enables the reviewed extended-autolink profile",
}

func TestM120PublishedCommonMarkHTMLFullProfileContract(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_COMMONMARK_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_COMMONMARK_SPEC_HTML is not set")
	}
	cases, err := commonmarkspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published CommonMark spec: %v", err)
	}
	seen := make(map[int]bool, len(m120CommonMarkProfileDivergences))
	for _, case_ := range cases {
		if _, divergent := m120CommonMarkProfileDivergences[case_.Number]; divergent {
			seen[case_.Number] = true
			continue
		}
		document, err := marksplice.Parse([]byte(case_.Markdown))
		if err != nil {
			t.Fatalf("CommonMark example %d parse error: %v", case_.Number, err)
		}
		got, err := document.HTML(marksplice.HTMLRenderOptions{TagFilter: marksplice.HTMLTagFilterDisabled})
		if err != nil {
			t.Fatalf("CommonMark example %d render error: %v", case_.Number, err)
		}
		if string(got) != case_.HTML {
			t.Fatalf("CommonMark example %d section %q mismatch\ngot:  %q\nwant: %q", case_.Number, case_.Section, got, case_.HTML)
		}
	}
	assertM120ProfileDivergencesPresent(t, "CommonMark", m120CommonMarkProfileDivergences, seen)
}

func TestM120PublishedGFMHTMLFullProfileContract(t *testing.T) {
	specPath := os.Getenv("MARKSPLICE_GFM_SPEC_HTML")
	if specPath == "" {
		t.Skip("MARKSPLICE_GFM_SPEC_HTML is not set")
	}
	cases, err := gfmspec.LoadPublished(specPath)
	if err != nil {
		t.Fatalf("load published GFM spec: %v", err)
	}
	seen := make(map[int]bool, len(m120GFMProfileDivergences))
	for _, case_ := range cases {
		if _, divergent := m120GFMProfileDivergences[case_.Number]; divergent {
			seen[case_.Number] = true
			continue
		}
		document, err := marksplice.Parse([]byte(case_.Markdown))
		if err != nil {
			t.Fatalf("GFM example %d parse error: %v", case_.Number, err)
		}
		options := marksplice.HTMLRenderOptions{TagFilter: marksplice.HTMLTagFilterDisabled}
		if slices.Contains(case_.Extensions, "tagfilter") {
			options.TagFilter = marksplice.HTMLTagFilterEnabled
		}
		got, err := document.HTML(options)
		if err != nil {
			t.Fatalf("GFM example %d render error: %v", case_.Number, err)
		}
		if string(got) != case_.HTML {
			t.Fatalf("GFM example %d extensions %v mismatch\ngot:  %q\nwant: %q", case_.Number, case_.Extensions, got, case_.HTML)
		}
	}
	assertM120ProfileDivergencesPresent(t, "GFM", m120GFMProfileDivergences, seen)
}

func assertM120ProfileDivergencesPresent(t *testing.T, profile string, divergences map[int]string, seen map[int]bool) {
	t.Helper()
	for number, reason := range divergences {
		if !seen[number] {
			t.Fatalf("%s documented profile divergence example %d is missing: %s", profile, number, reason)
		}
	}
}
